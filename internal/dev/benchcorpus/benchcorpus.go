package benchcorpus

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/hooks"
)

// ProcessRunner executes argv in dir with stdin as its input. A run that
// outlives timeout is killed and reported with timedOut set, not as an error;
// err is kept for a command that could not be started at all.
type ProcessRunner func(dir string, argv []string, stdin []byte, timeout time.Duration) (output string, exitCode int, timedOut bool, err error)

// Cloner clones a repository URL to targetDir and returns the commit SHA.
type Cloner func(repoURL, targetDir string) (commitSHA string, err error)

// benchStep is one component of a measurement pass.
type benchStep struct {
	name       string
	argv       []string
	stdin      []byte
	applicable bool
}

// BenchmarkRepo runs the benchmark and gap-audit suite on a single repository directory.
func BenchmarkRepo(dir string, opts Options, runner ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*RepoAudit, error) {
	if opts.WarmRuns < 1 {
		return nil, fmt.Errorf("warm_runs must be at least 1")
	}

	rootFS, err := openFS(dir)
	if err != nil {
		return nil, err
	}

	signals := InspectProject(rootFS)
	facts := detect.Detect(rootFS)
	lanes := hooks.TargetCommandsForStacks(facts.Stacks, "", facts.GodotDir, dir, facts.WikiPath)
	lanes = append(lanes, gateLanes(rootFS)...)
	checks, gaps, rate := AuditGaps(signals, facts.Stacks, lanes, lookPath)

	hasGo := slices.Contains(signals.Languages, "go") || slices.Contains(facts.Stacks, "go")

	// The hooks read an edit event from stdin; without one they measure only
	// their own refusal, so every hook call gets the edit of a real file.
	absDir := dir
	if abs, err := filepath.Abs(dir); err == nil {
		absDir = abs
	}
	sample := pickSampleFile(rootFS, facts.Stacks)
	payloads := map[string][]byte{}
	if sample != "" {
		absFile := filepath.Join(absDir, filepath.FromSlash(sample))
		for _, event := range []string{"PreToolUse", "PostToolUse"} {
			payloads[event] = hookPayload(event, absDir, absFile)
		}
	}

	steps := []benchStep{
		{name: "pre-tool-use", argv: []string{"loomux", "hook", "pre-tool-use", "--host", "claude", "--root", absDir}, stdin: payloads["PreToolUse"], applicable: sample != ""},
		{name: "post-tool-use", argv: []string{"loomux", "hook", "post-tool-use", "--host", "claude", "--root", absDir}, stdin: payloads["PostToolUse"], applicable: sample != ""},
		{name: "graph build", argv: []string{"loomux", "graph", "build", "--root", absDir}, applicable: hasGo},
	}
	var baseline []benchStep
	for _, h := range signals.ClaudeHooks {
		if payload, ok := payloads[h.Event]; ok {
			script := "export CLAUDE_PROJECT_DIR=" + shellQuote(absDir) + "; " + h.Command
			baseline = append(baseline, benchStep{name: "claude " + h.Event, argv: []string{"sh", "-c", script}, stdin: payload, applicable: true})
		}
	}

	measure := func(steps []benchStep) ([]ComponentTiming, time.Duration, error) {
		timings := make([]ComponentTiming, 0, len(steps))
		var total time.Duration
		for _, step := range steps {
			if !step.applicable {
				timings = append(timings, ComponentTiming{Name: step.name})
				continue
			}
			t0 := clock()
			_, code, timedOut, err := runner(absDir, step.argv, step.stdin, opts.ComponentTimeout)
			if err != nil {
				return nil, 0, err
			}
			elapsed := clock().Sub(t0)
			total += elapsed
			timings = append(timings, ComponentTiming{Name: step.name, Applicable: true, Elapsed: elapsed, ExitCode: code, TimedOut: timedOut})
		}
		return timings, total, nil
	}

	var coldRun TimingRun
	var warmRuns []TimingRun
	var warmTotals []time.Duration
	var claudeWarmTotals []time.Duration
	var hookWarmTotals []time.Duration
	var baselineErr string

	for p := 0; p < 1+opts.WarmRuns; p++ {
		components, total, err := measure(steps)
		if err != nil {
			return nil, err
		}
		// The baseline is a comparison, not the measurement: a hook that cannot
		// start drops the comparison and keeps what loomux measured.
		baseTimings, baseTotal, err := measure(baseline)
		if err != nil {
			baselineErr = err.Error()
			baseline, baseTimings, claudeWarmTotals = nil, nil, nil
			coldRun.Baseline = nil
			for i := range warmRuns {
				warmRuns[i].Baseline = nil
			}
		}
		run := TimingRun{Total: total, Components: components}
		if len(baseTimings) > 0 {
			run.Baseline = baseTimings
		}

		if p == 0 {
			coldRun = run
			continue
		}
		warmRuns = append(warmRuns, run)
		warmTotals = append(warmTotals, total)
		// The baseline replaces the edit hooks alone, so only they are its rival.
		hookWarmTotals = append(hookWarmTotals, components[0].Elapsed+components[1].Elapsed)
		if len(baseline) > 0 {
			claudeWarmTotals = append(claudeWarmTotals, baseTotal)
		}
	}

	sortedWarmTotals := slices.Clone(warmTotals)
	slices.Sort(sortedWarmTotals)

	audit := &RepoAudit{
		Dir:            dir,
		SampleFile:     sample,
		BaselineError:  baselineErr,
		DetectedStacks: facts.Stacks,
		ExecutedLanes:  lanes,
		Audit:          checks,
		MissingGaps:    gaps,
		CoverageRate:   rate,
		Cold:           coldRun,
		Warm:           warmRuns,
		WarmMedian:     calculateMedian(sortedWarmTotals),
		WarmMin:        sortedWarmTotals[0],
		WarmMax:        sortedWarmTotals[len(sortedWarmTotals)-1],
	}

	slices.Sort(hookWarmTotals)
	audit.HookWarmMedian = calculateMedian(hookWarmTotals)

	if len(claudeWarmTotals) > 0 {
		slices.Sort(claudeWarmTotals)
		audit.ClaudeWarmMed = calculateMedian(claudeWarmTotals)
		if audit.HookWarmMedian > 0 {
			audit.Speedup = float64(audit.ClaudeWarmMed) / float64(audit.HookWarmMedian)
		}
	}

	return audit, nil
}

// hookPayload is the event Claude Code sends a hook for an empty edit of file.
func hookPayload(event, cwd, file string) []byte {
	doc := struct {
		SessionID     string `json:"session_id"`
		HookEventName string `json:"hook_event_name"`
		ToolName      string `json:"tool_name"`
		ToolInput     struct {
			FilePath  string `json:"file_path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		} `json:"tool_input"`
		Cwd string `json:"cwd"`
	}{SessionID: "loomux-bench", HookEventName: event, ToolName: "Edit", Cwd: cwd}
	doc.ToolInput.FilePath = file
	// A struct of strings always marshals.
	data, _ := json.Marshal(doc)
	return data
}

// shellQuote makes s one single-quoted word for sh.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sampleSkipDirs hold vendored, generated or build output rather than the
// project's own sources.
var sampleSkipDirs = []string{".git", "vendor", "node_modules", "third_party", "build", "dist", "target"}

// auxiliaryStacks describe files around the code rather than the code itself;
// an edit to one of them says little about the language a project is in.
var auxiliaryStacks = []string{"shell", "css", "html", "sql", "wiki"}

// pickSampleFile names the file whose edit the hooks are measured on: the
// first file in lexical walk order that belongs to a detected language stack,
// else to a detected auxiliary stack, else README.md.
func pickSampleFile(root fs.FS, stacks []string) string {
	var primary, auxiliary []string
	for _, stack := range stacks {
		if slices.Contains(auxiliaryStacks, stack) {
			auxiliary = append(auxiliary, stack)
		} else {
			primary = append(primary, stack)
		}
	}
	for _, wanted := range [][]string{primary, auxiliary} {
		if sample := firstFileOfStacks(root, wanted); sample != "" {
			return sample
		}
	}
	if _, err := fs.Stat(root, "README.md"); err == nil {
		return "README.md"
	}
	return ""
}

// firstFileOfStacks walks root in lexical order, past vendored and hidden
// directories, to the first file one of stacks lints.
func firstFileOfStacks(root fs.FS, stacks []string) string {
	sample := ""
	_ = fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != "." && (strings.HasPrefix(d.Name(), ".") || slices.Contains(sampleSkipDirs, d.Name())) {
				return fs.SkipDir
			}
			return nil
		}
		if stack, ok := hooks.StackForExtension(path.Ext(p)); ok && slices.Contains(stacks, stack) {
			sample = p
			return fs.SkipAll
		}
		return nil
	})
	return sample
}

type matrixEntry struct {
	Language  string
	Framework string
	Tier      string
	RepoName  string
	RepoURL   string
}

var linkRegex = regexp.MustCompile(`\[(.*?)\]\((https?://.*?)\)`)

func parseMatrix(matrixData []byte, maxLanguages int, tierFilter string) []matrixEntry {
	var entries []matrixEntry
	seenURLs := make(map[string]bool)
	languagesSeen := make(map[string]bool)
	var languageList []string

	currentLang := ""
	currentFramework := ""

	scanner := bufio.NewScanner(bytes.NewReader(matrixData))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			lowerTitle := strings.ToLower(title)
			if lowerTitle != "table of contents" && lowerTitle != "inhaltsverzeichnis" && lowerTitle != "key metrics" && lowerTitle != "gesamtkennzahlen" && lowerTitle != "star categories" && lowerTitle != "sterne-kategorien" && !strings.Contains(lowerTitle, "matrix") {
				currentLang = title
				if !languagesSeen[currentLang] {
					languagesSeen[currentLang] = true
					languageList = append(languageList, currentLang)
				}
			}
			continue
		}

		if maxLanguages > 0 && len(languageList) > maxLanguages {
			if !slices.Contains(languageList[:maxLanguages], currentLang) {
				continue
			}
		}

		if strings.HasPrefix(line, "### ") {
			rawFw := strings.TrimPrefix(line, "### ")
			if currentLang != "" {
				rawFw = strings.TrimSpace(strings.TrimPrefix(rawFw, currentLang+"+"))
				rawFw = strings.TrimSpace(strings.TrimPrefix(rawFw, currentLang+" +"))
			}
			currentFramework = rawFw
			continue
		}

		if strings.HasPrefix(line, "|") && strings.Contains(line, "http") {
			parts := strings.Split(line, "|")
			rawTier := strings.TrimSpace(parts[1])
			cleanTier := strings.Trim(rawTier, "* ")
			if idx := strings.Index(cleanTier, "("); idx != -1 {
				cleanTier = strings.TrimSpace(cleanTier[:idx])
			}

			if tierFilter != "" {
				if !strings.EqualFold(cleanTier, tierFilter) {
					continue
				}
			}

			repoCol := parts[2]
			matches := linkRegex.FindStringSubmatch(repoCol)
			if len(matches) == 3 {
				repoName := matches[1]
				repoURL := matches[2]
				if !seenURLs[repoURL] {
					seenURLs[repoURL] = true
					entries = append(entries, matrixEntry{
						Language:  currentLang,
						Framework: currentFramework,
						Tier:      cleanTier,
						RepoName:  repoName,
						RepoURL:   repoURL,
					})
				}
			}
		}
	}

	return entries
}

func sanitizeRepoDir(name string) string {
	r := strings.NewReplacer("/", "_", "\\", "_", ":", "_")
	return r.Replace(name)
}

func calculateMedian(d []time.Duration) time.Duration {
	n := len(d)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return d[n/2]
	}
	return (d[n/2-1] + d[n/2]) / 2
}

// BenchmarkCorpus benchmarks multiple repositories defined in the open-source matrix corpus.
func BenchmarkCorpus(matrixData []byte, opts Options, cloner Cloner, benchRepo func(string, Options) (*RepoAudit, error)) (*BenchmarkReport, error) {
	entries := parseMatrix(matrixData, opts.Languages, opts.Tier)
	report := &BenchmarkReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Mode:      "corpus",
		WarmRuns:  opts.WarmRuns,
		Repos:     make([]*RepoAudit, 0, len(entries)),
	}

	// One repository the platform cannot check out (a `|` in a path on
	// Windows) must not cost the other few hundred their measurement.
	var firstErr error
	skip := func(entry matrixEntry, err error) {
		if firstErr == nil {
			firstErr = err
		}
		report.Skipped = append(report.Skipped, SkippedRepo{
			RepoURL:   entry.RepoURL,
			Language:  entry.Language,
			Framework: entry.Framework,
			Reason:    summarizeReason(err),
		})
	}

	for _, entry := range entries {
		targetDir := filepath.Join(opts.CacheDir, sanitizeRepoDir(entry.RepoName))
		commitSHA, err := cloner(entry.RepoURL, targetDir)
		if err != nil {
			skip(entry, err)
			continue
		}

		audit, err := benchRepo(targetDir, opts)
		if err != nil {
			skip(entry, err)
			continue
		}

		audit.RepoURL = entry.RepoURL
		audit.Language = entry.Language
		audit.Framework = entry.Framework
		audit.Tier = entry.Tier
		audit.CommitSHA = commitSHA
		report.Repos = append(report.Repos, audit)
	}

	if len(report.Repos) == 0 && len(report.Skipped) > 0 {
		return nil, fmt.Errorf("none of %d repositories could be benchmarked; first: %w", len(report.Skipped), firstErr)
	}
	return report, nil
}

// summarizeReason keeps git's `error:` and `fatal:` lines, which say why a
// clone failed, and drops the progress chatter around them.
func summarizeReason(err error) string {
	lines := strings.Split(err.Error(), "\n")
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "error:") || strings.HasPrefix(trimmed, "fatal:") {
			kept = append(kept, trimmed)
		}
	}
	if len(kept) == 0 {
		return strings.TrimSpace(lines[0])
	}
	return strings.Join(kept, "; ")
}
