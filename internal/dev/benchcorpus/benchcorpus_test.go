package benchcorpus

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

func TestBenchmarkRepo(t *testing.T) {
	mockClock := func() func() time.Time {
		currentTime := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
		return func() time.Time {
			currentTime = currentTime.Add(10 * time.Millisecond)
			return currentTime
		}
	}

	mockLookPathSuccess := func(file string) (string, error) {
		return "/bin/" + file, nil
	}

	t.Run("Invalid warm runs returns error", func(t *testing.T) {
		opts := Options{WarmRuns: 0}
		_, err := BenchmarkRepo(".", opts, nil, nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "warm_runs") {
			t.Fatalf("expected error for warm_runs < 1, got %v", err)
		}
	})

	t.Run("OpenFS failure returns error", func(t *testing.T) {
		opts := Options{WarmRuns: 2}
		openFSError := func(string) (fs.FS, error) {
			return nil, errors.New("open failed")
		}
		_, err := BenchmarkRepo(".", opts, nil, nil, openFSError, nil)
		if err == nil || !strings.Contains(err.Error(), "open failed") {
			t.Fatalf("expected openFS error, got %v", err)
		}
	})

	t.Run("Go project with all components applicable", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"go.mod":       &fstest.MapFile{Data: []byte("module example.com/test\n\ngo 1.25\n")},
			"main.go":      &fstest.MapFile{Data: []byte("package main\n")},
			"main_test.go": &fstest.MapFile{Data: []byte("package main\n")},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		var commandsRun []string
		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			commandsRun = append(commandsRun, strings.Join(argv, " "))
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 3}
		audit, err := BenchmarkRepo("/repo/go", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if audit.Dir != "/repo/go" {
			t.Errorf("expected dir /repo/go, got %s", audit.Dir)
		}
		total, _ := audit.Timing(TotalTiming)
		if len(total.WarmMS) != 3 {
			t.Fatalf("expected 3 warm runs, got %d", len(total.WarmMS))
		}
		if total.MedianMS == 0 || total.MinMS == 0 || total.MaxMS == 0 {
			t.Errorf("expected non-zero warm stats: %+v", total)
		}

		graph, ok := audit.Timing("graph build")
		if !ok {
			t.Fatalf("expected a graph build timing")
		}
		if graph.Applicable != nil {
			t.Errorf("expected graph build to be applicable for Go project")
		}
	})

	t.Run("Non-Go project with graph build non-applicable", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"package.json": &fstest.MapFile{Data: []byte(`{"name":"foo","scripts":{"test":"vitest"}}`)},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 2}
		audit, err := BenchmarkRepo("/repo/node", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		graph, _ := audit.Timing("graph build")
		if graph.Applicable == nil || *graph.Applicable {
			t.Errorf("expected graph build to NOT be applicable for non-Go project")
		}
		if graph.ColdMS != 0 || graph.WarmMS != nil || graph.ExitCodes != nil {
			t.Errorf("expected no measurement when non-applicable, got %+v", graph)
		}
	})

	t.Run("Baseline Claude hook calculation and speedup", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"go.mod":  &fstest.MapFile{Data: []byte("module foo\n")},
			".claude/settings.json": &fstest.MapFile{
				Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"python -m unittest"}]}]}}`),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 2}
		audit, err := BenchmarkRepo("/repo/claude", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if audit.ClaudeWarmMed == 0 {
			t.Errorf("expected non-zero ClaudeWarmMed")
		}
		if audit.Speedup <= 0 {
			t.Errorf("expected positive speedup, got %f", audit.Speedup)
		}
	})

	t.Run("Runner returns error during pre-tool-use", func(t *testing.T) {
		mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if slices.Contains(argv, "pre-tool-use") {
				return "", 1, false, errors.New("pre-tool-use crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		_, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err == nil || !strings.Contains(err.Error(), "pre-tool-use crashed") {
			t.Fatalf("expected pre-tool-use error, got %v", err)
		}
	})

	t.Run("Runner returns error during post-tool-use", func(t *testing.T) {
		mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if slices.Contains(argv, "post-tool-use") {
				return "", 1, false, errors.New("post-tool-use crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		_, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err == nil || !strings.Contains(err.Error(), "post-tool-use crashed") {
			t.Fatalf("expected post-tool-use error, got %v", err)
		}
	})

	t.Run("Runner returns error during graph build", func(t *testing.T) {
		mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if slices.Contains(argv, "graph") {
				return "", 1, false, errors.New("graph build crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		_, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err == nil || !strings.Contains(err.Error(), "graph build crashed") {
			t.Fatalf("expected graph build error, got %v", err)
		}
	})

	t.Run("Runner returns error during claude hook", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"main.go": &fstest.MapFile{Data: []byte("package main\n")},
			"go.mod":  &fstest.MapFile{Data: []byte("module foo\n")},
			".claude/settings.json": &fstest.MapFile{
				Data: []byte(`{"hooks":{"PostToolUse":[{"hooks":[{"type":"command","command":"sh fail.sh"}]}]}}`),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runnerErr := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			if strings.Contains(strings.Join(argv, " "), "sh fail.sh") {
				return "", 1, false, errors.New("claude hook crashed")
			}
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		audit, err := BenchmarkRepo("/repo/err", opts, runnerErr, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("a broken baseline must not fail the repository: %v", err)
		}
		_, measured := audit.Timing(BaselineTiming("claude PostToolUse"))
		if audit.BaselineError != "claude hook crashed" || audit.ClaudeWarmMed != 0 || audit.Speedup != 0 || measured {
			t.Errorf("expected an unavailable baseline, got %+v", audit)
		}
		if total, _ := audit.Timing(TotalTiming); total.MedianMS == 0 {
			t.Errorf("loomux measurement lost")
		}
	})

	t.Run("Project with pre-commit gate matches test commands", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"go.mod": &fstest.MapFile{Data: []byte("module foo\n\ngo 1.25\n")},
			".githooks/pre-commit": &fstest.MapFile{
				Data: []byte("#!/bin/sh\nset -e\ngo test ./... -count=1\n"),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }

		runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
			return "ok", 0, false, nil
		}

		opts := Options{WarmRuns: 1}
		audit, err := BenchmarkRepo("/repo/gate", opts, runner, mockClock(), openFS, mockLookPathSuccess)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var goTestFound bool
		for _, check := range audit.Audit {
			if check.Tool == "go-test" {
				goTestFound = true
				if check.Lane != "go test ./... -count=1" {
					t.Errorf("expected lane 'go test ./... -count=1', got '%s'", check.Lane)
				}
			}
		}
		if !goTestFound {
			t.Errorf("expected go-test in audit")
		}
		if audit.CoverageRate != 100.0 {
			t.Errorf("expected 100.0%% coverage rate, got %f", audit.CoverageRate)
		}
	})
}

// timings is a row's timings with total first, the order BenchmarkRepo writes.
func timings(total benchreport.Timing, parts ...benchreport.Timing) []benchreport.Timing {
	total.Name = TotalTiming
	return append([]benchreport.Timing{total}, parts...)
}

// comp is an applicable component whose every run, cold and warm, exited 0.
func comp(name string, coldMS float64, warmMS ...float64) benchreport.Timing {
	t := benchreport.Summarize(name, coldMS, warmMS)
	t.ExitCodes = make([]int, 1+len(warmMS))
	return t
}

// notApplicable is a component the repository gave nothing to measure.
func notApplicable(name string) benchreport.Timing {
	no := false
	return benchreport.Timing{Name: name, Applicable: &no}
}

// runFakeRepoWithBaseline benchmarks a repository without Go whose Claude
// settings hold one PreToolUse hook; every command takes 10 ms.
func runFakeRepoWithBaseline(t *testing.T) *RepoAudit {
	t.Helper()
	mockFS := fstest.MapFS{
		"README.md": &fstest.MapFile{Data: []byte("# foo\n")},
		".claude/settings.json": &fstest.MapFile{
			Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"python -m unittest"}]}]}}`),
		},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo("/repo/claude", Options{WarmRuns: 2}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return audit
}

func TestBenchmarkRepoNamesTotalComponentsAndBaseline(t *testing.T) {
	audit := runFakeRepoWithBaseline(t)
	var names []string
	for _, tm := range audit.Timings {
		names = append(names, tm.Name)
	}
	want := []string{"total", "pre-tool-use", "post-tool-use", "graph build", "baseline:claude PreToolUse"}
	if !slices.Equal(names, want) {
		t.Fatalf("names = %v", names)
	}
	if g, _ := audit.Timing("graph build"); g.Applicable == nil || *g.Applicable {
		t.Fatalf("graph build without Go must be applicable=false: %+v", g)
	}
}

func TestBenchmarkRepoSummarizesInMilliseconds(t *testing.T) {
	audit := runFakeRepoWithBaseline(t)
	total, _ := audit.Timing(TotalTiming)
	if total.ColdMS != 20 || !slices.Equal(total.WarmMS, []float64{20, 20}) || total.MedianMS != 20 || total.MinMS != 20 || total.MaxMS != 20 {
		t.Errorf("total = %+v", total)
	}
	pre, _ := audit.Timing("pre-tool-use")
	if pre.ColdMS != 10 || pre.MedianMS != 10 || !slices.Equal(pre.ExitCodes, []int{0, 0, 0}) || pre.TimedOut != 0 {
		t.Errorf("pre-tool-use = %+v", pre)
	}
	base, _ := audit.Timing(BaselineTiming("claude PreToolUse"))
	if base.ColdMS != 10 || base.MedianMS != 10 || len(base.ExitCodes) != 3 {
		t.Errorf("baseline = %+v", base)
	}
	if audit.HookWarmMedian != 20 || audit.ClaudeWarmMed != 10 || audit.Speedup != 0.5 {
		t.Errorf("hook %v, claude %v, speedup %v", audit.HookWarmMedian, audit.ClaudeWarmMed, audit.Speedup)
	}
}

func TestRepoAuditTimingLookups(t *testing.T) {
	a := &RepoAudit{Timings: timings(benchreport.Timing{MedianMS: 3},
		benchreport.Timing{Name: "pre-tool-use"},
		benchreport.Timing{Name: BaselineTiming("claude PreToolUse")},
		benchreport.Timing{Name: "graph build"})}
	if got, ok := a.Timing(TotalTiming); !ok || got.MedianMS != 3 {
		t.Errorf("Timing(total) = %+v, %v", got, ok)
	}
	if _, ok := a.Timing("absent"); ok {
		t.Error("Timing(absent) found something")
	}
	var names []string
	for _, c := range a.Components() {
		names = append(names, c.Name)
	}
	if !slices.Equal(names, []string{"pre-tool-use", "graph build"}) {
		t.Errorf("Components() = %v", names)
	}
	if BaselineTiming("x") != "baseline:x" {
		t.Errorf("BaselineTiming(x) = %q", BaselineTiming("x"))
	}
}

func TestBenchmarkCorpus(t *testing.T) {
	matrixMD := `# Open Source Matrix

## Python

### Python + Django

| Kategorie | Repository | Sterne | LLM-Nutzung | Beschreibung |
|---|---|---|---|---|
| **Sehr viel (≥100k / Top)** | [django/django](https://github.com/django/django) | ⭐ 91.134 | ` + "`Nein`" + ` | The Web framework |
| **Viel (10k–100k)** | [encode/django-rest-framework](https://github.com/encode/django-rest-framework) | ⭐ 30.188 | ` + "`Nein`" + ` | Web APIs |

### Python + Flask

| Kategorie | Repository | Sterne | LLM-Nutzung | Beschreibung |
|---|---|---|---|---|
| **Sehr viel (≥100k / Top)** | [pallets/flask](https://github.com/pallets/flask) | ⭐ 74.749 | ` + "`Nein`" + ` | Micro framework |
| **Sehr viel (≥100k / Top)** | [django/django](https://github.com/django/django) | ⭐ 91.134 | ` + "`Nein`" + ` | Duplicate entry test |

## JavaScript

### JavaScript + React

| Kategorie | Repository | Sterne | LLM-Nutzung | Beschreibung |
|---|---|---|---|---|
| **Sehr viel (≥100k / Top)** | [facebook/react](https://github.com/facebook/react) | ⭐ 220.000 | ` + "`Nein`" + ` | UI library |
`

	mockCloner := func(repoURL, targetDir string) (string, error) {
		return "abc1234", nil
	}

	mockBenchRepo := func(dir string, opts Options) (*RepoAudit, error) {
		return &RepoAudit{
			Dir:          dir,
			Timings:      timings(benchreport.Timing{MedianMS: 50}),
			CoverageRate: 100.0,
		}, nil
	}

	t.Run("Corpus run with languages limit and tier filter", func(t *testing.T) {
		opts := Options{
			Languages: 1, // Only Python
			Tier:      "Sehr viel",
			WarmRuns:  2,
			CacheDir:  "/tmp/cache",
		}

		report, err := BenchmarkCorpus([]byte(matrixMD), opts, mockCloner, mockBenchRepo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if report.Mode != "corpus" {
			t.Errorf("expected mode corpus, got %s", report.Mode)
		}
		if report.WarmRuns != 2 {
			t.Errorf("expected warm runs 2, got %d", report.WarmRuns)
		}
		// Expect django/django and pallets/flask (deduplicated django/django)
		if len(report.Repos) != 2 {
			t.Fatalf("expected 2 unique repos for Python Sehr viel, got %d", len(report.Repos))
		}
		if report.Repos[0].RepoURL != "https://github.com/django/django" {
			t.Errorf("expected django/django, got %s", report.Repos[0].RepoURL)
		}
		if report.Repos[0].CommitSHA != "abc1234" {
			t.Errorf("expected commit sha abc1234, got %s", report.Repos[0].CommitSHA)
		}
		if report.Repos[0].Language != "Python" {
			t.Errorf("expected language Python, got %s", report.Repos[0].Language)
		}
	})

	t.Run("Cloner failure skips the repository and continues", func(t *testing.T) {
		opts := Options{
			Languages: 1,
			Tier:      "Sehr viel",
			WarmRuns:  1,
		}
		clonerFail := func(repoURL, targetDir string) (string, error) {
			if strings.Contains(repoURL, "django") {
				return "", errors.New("git clone: exit status 128: Cloning into 'x'...\nerror: invalid path 'a|b.md'\nfatal: unable to checkout working tree")
			}
			return "abc1234", nil
		}

		report, err := BenchmarkCorpus([]byte(matrixMD), opts, clonerFail, mockBenchRepo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(report.Repos) != 1 || report.Repos[0].RepoURL != "https://github.com/pallets/flask" {
			t.Fatalf("expected only pallets/flask to be benchmarked, got %+v", report.Repos)
		}
		want := SkippedRepo{
			RepoURL:   "https://github.com/django/django",
			Language:  "Python",
			Framework: "Django",
			Reason:    "error: invalid path 'a|b.md'; fatal: unable to checkout working tree",
		}
		if len(report.Skipped) != 1 || report.Skipped[0] != want {
			t.Fatalf("expected %+v skipped, got %+v", want, report.Skipped)
		}
	})

	t.Run("Every repository failing returns an error", func(t *testing.T) {
		opts := Options{
			Languages: 1,
			Tier:      "Sehr viel",
			WarmRuns:  1,
		}
		benchFail := func(dir string, opts Options) (*RepoAudit, error) {
			return nil, errors.New("bench failed")
		}

		_, err := BenchmarkCorpus([]byte(matrixMD), opts, mockCloner, benchFail)
		if err == nil || !strings.Contains(err.Error(), "none of 2 repositories") || !strings.Contains(err.Error(), "bench failed") {
			t.Fatalf("expected an all-failed error naming the cause, got %v", err)
		}
	})

	t.Run("Corpus run with all languages and no tier filter", func(t *testing.T) {
		opts := Options{
			Languages: 0,  // All languages
			Tier:      "", // All tiers
			WarmRuns:  1,
		}
		report, err := BenchmarkCorpus([]byte(matrixMD), opts, mockCloner, mockBenchRepo)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Expect django/django, encode/django-rest-framework, pallets/flask, facebook/react
		if len(report.Repos) != 4 {
			t.Fatalf("expected 4 repos across all languages and tiers, got %d", len(report.Repos))
		}
	})
}

func fixedClock() func() time.Time {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	return func() time.Time {
		now = now.Add(10 * time.Millisecond)
		return now
	}
}

func noLookPath(string) (string, error) { return "", errors.New("absent") }

type hookPayloadDoc struct {
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		FilePath  string `json:"file_path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	} `json:"tool_input"`
	Cwd string `json:"cwd"`
}

func TestBenchmarkRepoHookPayload(t *testing.T) {
	mockFS := fstest.MapFS{
		"go.mod":            &fstest.MapFile{Data: []byte("module foo\n\ngo 1.25\n")},
		".hidden/a.go":      &fstest.MapFile{Data: []byte("package a\n")},
		"build/b.go":        &fstest.MapFile{Data: []byte("package b\n")},
		"cmd/tool/main.go":  &fstest.MapFile{Data: []byte("package main\n")},
		"vendor/x/c.go":     &fstest.MapFile{Data: []byte("package c\n")},
		"zzz_last/other.go": &fstest.MapFile{Data: []byte("package z\n")},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }

	payloads := map[string][]byte{}
	var timeouts []time.Duration
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		timeouts = append(timeouts, timeout)
		if len(argv) > 2 && argv[1] == "hook" {
			payloads[argv[2]] = stdin
			if argv[2] == "pre-tool-use" {
				return "", 2, false, nil
			}
		}
		return "", 0, false, nil
	}

	opts := Options{WarmRuns: 1, ComponentTimeout: 7 * time.Second}
	audit, err := BenchmarkRepo("/repo/go", opts, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if audit.SampleFile != "cmd/tool/main.go" {
		t.Errorf("sample file = %q, want cmd/tool/main.go", audit.SampleFile)
	}
	for _, timeout := range timeouts {
		if timeout != 7*time.Second {
			t.Errorf("runner got timeout %v, want 7s", timeout)
		}
	}
	wantAbs, _ := filepath.Abs("/repo/go")
	for hook, event := range map[string]string{"pre-tool-use": "PreToolUse", "post-tool-use": "PostToolUse"} {
		var doc hookPayloadDoc
		if err := json.Unmarshal(payloads[hook], &doc); err != nil {
			t.Fatalf("%s payload is no JSON: %v", hook, err)
		}
		if doc.SessionID != "loomux-bench" || doc.HookEventName != event || doc.ToolName != "Edit" || doc.Cwd != wantAbs {
			t.Errorf("%s payload = %+v", hook, doc)
		}
		if doc.ToolInput.FilePath != filepath.Join(wantAbs, "cmd", "tool", "main.go") {
			t.Errorf("%s file_path = %q", hook, doc.ToolInput.FilePath)
		}
	}
	pre := audit.Timings[1]
	if pre.Name != "pre-tool-use" || !slices.Equal(pre.ExitCodes, []int{2, 2}) || pre.TimedOut != 0 {
		t.Errorf("pre-tool-use timing = %+v", pre)
	}
}

func TestBenchmarkRepoRecordsTimeout(t *testing.T) {
	mockFS := fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		if slices.Contains(argv, "post-tool-use") {
			return "", -1, true, nil
		}
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo("/repo/go", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if post, _ := audit.Timing("post-tool-use"); post.TimedOut != 2 {
		t.Errorf("expected both post-tool-use runs to be recorded as timed out, got %+v", post)
	}
}

func TestBenchmarkRepoSampleFallbacks(t *testing.T) {
	var hookCalls int
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		if slices.Contains(argv, "hook") {
			hookCalls++
		}
		return "", 0, false, nil
	}

	t.Run("README when no source file matches", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"go.mod":    &fstest.MapFile{Data: []byte("module foo\n")},
			"README.md": &fstest.MapFile{Data: []byte("# foo\n")},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }
		audit, err := BenchmarkRepo("/repo/doc", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pre, _ := audit.Timing("pre-tool-use"); audit.SampleFile != "README.md" || pre.Applicable != nil {
			t.Errorf("expected README.md sample with applicable hooks, got %q %+v", audit.SampleFile, pre)
		}
	})

	t.Run("no file skips hook components", func(t *testing.T) {
		hookCalls = 0
		mockFS := fstest.MapFS{
			"package.json": &fstest.MapFile{Data: []byte(`{"name":"foo"}`)},
			".claude/settings.json": &fstest.MapFile{
				Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"true"}]}]}}`),
			},
		}
		openFS := func(string) (fs.FS, error) { return mockFS, nil }
		audit, err := BenchmarkRepo("/repo/empty", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hookCalls != 0 || audit.SampleFile != "" {
			t.Errorf("expected no hook calls and no sample, got %d calls, sample %q", hookCalls, audit.SampleFile)
		}
		for _, c := range audit.Components()[:2] {
			if c.Applicable == nil || *c.Applicable {
				t.Errorf("expected hook components to be n/a, got %+v", c)
			}
		}
		if audit.ClaudeWarmMed != 0 {
			t.Errorf("expected no baseline without a sample file, got %v", audit.ClaudeWarmMed)
		}
	})
}

func TestBenchmarkRepoBaselineHooks(t *testing.T) {
	mockFS := fstest.MapFS{
		"main.go": &fstest.MapFile{Data: []byte("package main\n")},
		"go.mod":  &fstest.MapFile{Data: []byte("module foo\n")},
		".claude/settings.json": &fstest.MapFile{Data: []byte(`{"hooks":{
			"PreToolUse":[{"matcher":"Edit","hooks":[{"type":"command","command":"pre-check"}]}],
			"PostToolUse":[{"hooks":[{"type":"command","command":"post-check"}]}],
			"Stop":[{"hooks":[{"type":"command","command":"stop-check"}]}]}}`)},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }

	var shells []string
	events := map[string]string{}
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		if argv[0] == "sh" {
			shells = append(shells, argv[2])
			var doc hookPayloadDoc
			_ = json.Unmarshal(stdin, &doc)
			events[argv[2]] = doc.HookEventName
			return "", 1, false, nil
		}
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo("/it's", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	abs, _ := filepath.Abs("/it's")
	quoted := "'" + strings.ReplaceAll(abs, "'", `'\''`) + "'"
	wantPost := "export CLAUDE_PROJECT_DIR=" + quoted + "; post-check"
	wantPre := "export CLAUDE_PROJECT_DIR=" + quoted + "; pre-check"
	if len(shells) != 4 || shells[0] != wantPost || shells[1] != wantPre {
		t.Fatalf("baseline shells = %q", shells)
	}
	if events[wantPost] != "PostToolUse" || events[wantPre] != "PreToolUse" {
		t.Errorf("baseline payload events = %v", events)
	}
	baseline := audit.Timings[len(audit.Timings)-2:]
	if baseline[0].Name != BaselineTiming("claude PostToolUse") || baseline[1].Name != BaselineTiming("claude PreToolUse") || !slices.Equal(baseline[0].ExitCodes, []int{1, 1}) {
		t.Errorf("baseline timings = %+v", baseline)
	}
	if audit.ClaudeWarmMed == 0 {
		t.Errorf("expected a baseline median")
	}
}

// readDirFailFS refuses to list one directory, as a permission error would.
type readDirFailFS struct{ fstest.MapFS }

func (f readDirFailFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "locked" {
		return nil, errors.New("permission denied")
	}
	return f.MapFS.ReadDir(name)
}

func TestPickSampleFileSkipsUnreadableDirs(t *testing.T) {
	root := readDirFailFS{fstest.MapFS{
		"locked/a.go": &fstest.MapFile{Data: []byte("package a\n")},
		"src/b.go":    &fstest.MapFile{Data: []byte("package b\n")},
		"notes.md":    &fstest.MapFile{Data: []byte("# notes\n")},
	}}
	if got := pickSampleFile(root, []string{"go", "wiki"}); got != "src/b.go" {
		t.Errorf("pickSampleFile = %q, want src/b.go", got)
	}
}

// A C++ project with shell scripts is audited for what post-edit runs on an
// edit: clang-format on the file rather than clang-tidy on the build tree,
// and shellcheck, whose lane has only a form for one file.
func TestBenchmarkRepoAuditsTheEditForms(t *testing.T) {
	mockFS := fstest.MapFS{
		"CMakeLists.txt": &fstest.MapFile{Data: []byte("project(p)\n")},
		".clang-format":  &fstest.MapFile{Data: []byte("BasedOnStyle: LLVM\n")},
		".shellcheckrc":  &fstest.MapFile{Data: []byte("\n")},
		"main.cpp":       &fstest.MapFile{Data: []byte("int main() {}\n")},
		"run.sh":         &fstest.MapFile{Data: []byte("#!/bin/sh\n")},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	runner := func(string, []string, []byte, time.Duration) (string, int, bool, error) { return "", 0, false, nil }
	audit, err := BenchmarkRepo("/repo/cpp", Options{WarmRuns: 1}, runner, fixedClock(), openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lanes := map[string]string{}
	for _, a := range audit.Audit {
		lanes[a.Tool] = a.Lane
	}
	if lanes["clang-format"] != "clang-format --dry-run --Werror {file}" || lanes["shellcheck"] != "shellcheck {file}" {
		t.Errorf("audited lanes = %q, executed = %q", lanes, audit.ExecutedLanes)
	}
}

// goRepoFS opens a Go module with one source file, whatever the directory.
func goRepoFS(string) (fs.FS, error) {
	return fstest.MapFS{"go.mod": &fstest.MapFile{Data: []byte("module foo\n")}, "main.go": &fstest.MapFile{Data: []byte("package main\n")}}, nil
}

// budgetRecorder is a runner whose every command takes 40 seconds on the
// returned clock and records the timeout it was given.
func budgetRecorder() (ProcessRunner, func() time.Time, *[]time.Duration) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	budgets := &[]time.Duration{}
	runner := func(dir string, argv []string, stdin []byte, timeout time.Duration) (string, int, bool, error) {
		*budgets = append(*budgets, timeout)
		now = now.Add(40 * time.Second)
		return "", 0, false, nil
	}
	return runner, func() time.Time { return now }, budgets
}

func TestBenchmarkRepoStopsAtTheRepositoryTimeout(t *testing.T) {
	runner, clock, budgets := budgetRecorder()
	opts := Options{WarmRuns: 3, Timeout: 100 * time.Second, ComponentTimeout: 60 * time.Second}
	_, err := BenchmarkRepo(".", opts, runner, clock, goRepoFS, noLookPath)
	if !errors.Is(err, ErrRepoTimeout) {
		t.Fatalf("err = %v, want ErrRepoTimeout", err)
	}
	want := []time.Duration{60 * time.Second, 60 * time.Second, 20 * time.Second}
	if !slices.Equal(*budgets, want) {
		t.Fatalf("budgets = %v, want %v", *budgets, want)
	}
}

func TestBenchmarkRepoWithoutTimeoutKeepsTheComponentBudget(t *testing.T) {
	// Timeout 0 means no repository deadline, as before.
	runner, clock, budgets := budgetRecorder()
	opts := Options{WarmRuns: 3, ComponentTimeout: 60 * time.Second}
	if _, err := BenchmarkRepo(".", opts, runner, clock, goRepoFS, noLookPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*budgets) != 12 {
		t.Fatalf("got %d runs, want 12", len(*budgets))
	}
	for _, budget := range *budgets {
		if budget != 60*time.Second {
			t.Fatalf("budgets = %v, want every one 60s", *budgets)
		}
	}
}

func TestBenchmarkRepoWithoutComponentTimeoutUsesTheRepositoryBudget(t *testing.T) {
	// ComponentTimeout 0 means no limit per command; the repository deadline
	// still bounds each one.
	runner, clock, budgets := budgetRecorder()
	opts := Options{WarmRuns: 3, Timeout: 100 * time.Second}
	_, err := BenchmarkRepo(".", opts, runner, clock, goRepoFS, noLookPath)
	if !errors.Is(err, ErrRepoTimeout) {
		t.Fatalf("err = %v, want ErrRepoTimeout", err)
	}
	want := []time.Duration{100 * time.Second, 60 * time.Second, 20 * time.Second}
	if !slices.Equal(*budgets, want) {
		t.Fatalf("budgets = %v, want %v", *budgets, want)
	}
}

func TestBenchmarkRepoStopsWhenTheDeadlineIsReachedExactly(t *testing.T) {
	// Three runs of 40 s spend 120 s: the fourth command finds nothing left
	// and must not start with a budget of zero.
	runner, clock, budgets := budgetRecorder()
	opts := Options{WarmRuns: 3, Timeout: 120 * time.Second, ComponentTimeout: 60 * time.Second}
	_, err := BenchmarkRepo(".", opts, runner, clock, goRepoFS, noLookPath)
	if !errors.Is(err, ErrRepoTimeout) {
		t.Fatalf("err = %v, want ErrRepoTimeout", err)
	}
	want := []time.Duration{60 * time.Second, 60 * time.Second, 40 * time.Second}
	if !slices.Equal(*budgets, want) {
		t.Fatalf("budgets = %v, want %v", *budgets, want)
	}
}

func TestBenchmarkRepoKeepsTheColdRunApartFromTheWarmOnes(t *testing.T) {
	// The cold pass (three commands) takes 30 ms per command, every later
	// command 10 ms.
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	calls := 0
	runner := func(string, []string, []byte, time.Duration) (string, int, bool, error) {
		calls++
		if calls <= 3 {
			now = now.Add(30 * time.Millisecond)
		} else {
			now = now.Add(10 * time.Millisecond)
		}
		return "", 0, false, nil
	}
	audit, err := BenchmarkRepo(".", Options{WarmRuns: 1}, runner, func() time.Time { return now }, goRepoFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pre, _ := audit.Timing("pre-tool-use")
	if pre.ColdMS != 30 || !slices.Equal(pre.WarmMS, []float64{10}) {
		t.Errorf("pre-tool-use = %+v, want cold 30 and warm [10]", pre)
	}
}

func TestBenchmarkRepoLeavesTheSpeedupOutWhenTheHooksTakeNoTime(t *testing.T) {
	// A clock that never moves makes every run 0 ms: there is no loomux time
	// to divide by, so no speedup (and no NaN, which JSON cannot carry).
	mockFS := fstest.MapFS{
		"main.go": &fstest.MapFile{Data: []byte("package main\n")},
		"go.mod":  &fstest.MapFile{Data: []byte("module foo\n")},
		".claude/settings.json": &fstest.MapFile{
			Data: []byte(`{"hooks":{"PreToolUse":[{"hooks":[{"type":"command","command":"check"}]}]}}`),
		},
	}
	openFS := func(string) (fs.FS, error) { return mockFS, nil }
	runner := func(string, []string, []byte, time.Duration) (string, int, bool, error) { return "", 0, false, nil }
	still := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	audit, err := BenchmarkRepo("/repo/still", Options{WarmRuns: 2}, runner, func() time.Time { return still }, openFS, noLookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, measured := audit.Timing(BaselineTiming("claude PreToolUse")); !measured {
		t.Fatalf("baseline not measured: %+v", audit.Timings)
	}
	if audit.HookWarmMedian != 0 || audit.Speedup != 0 {
		t.Errorf("hook median %v, speedup %v, want both 0", audit.HookWarmMedian, audit.Speedup)
	}
}

func TestPickSampleFileOnARootThatCannotBeOpened(t *testing.T) {
	// fs.WalkDir hands the callback a nil entry when the root itself fails.
	if got := pickSampleFile(errFS{}, []string{"go"}); got != "" {
		t.Errorf("pickSampleFile = %q, want nothing", got)
	}
}

func TestParseMatrixSkipsItsOwnSections(t *testing.T) {
	data := []byte(`## Python
| Top | [p/a](https://github.com/p/a) |
## Table of Contents
| Top | [p/b](https://github.com/p/b) |
## Inhaltsverzeichnis
| Top | [p/c](https://github.com/p/c) |
## Key Metrics
| Top | [p/d](https://github.com/p/d) |
## Gesamtkennzahlen
| Top | [p/e](https://github.com/p/e) |
## Star Categories
| Top | [p/f](https://github.com/p/f) |
## Sterne-Kategorien
| Top | [p/g](https://github.com/p/g) |
## Open Source Matrix
| Top | [p/h](https://github.com/p/h) |
`)
	entries := parseMatrix(data, 0, "")
	if len(entries) != 8 {
		t.Fatalf("got %d entries, want 8: %+v", len(entries), entries)
	}
	for _, e := range entries {
		if e.Language != "Python" {
			t.Errorf("%s filed under %q, want Python", e.RepoURL, e.Language)
		}
	}
}

func TestParseMatrixLanguageLimitCountsEachLanguageOnce(t *testing.T) {
	// Go comes back after Python; a repeated heading is no new language.
	data := []byte(`## Go
| Top | [g/a](https://github.com/g/a) |
## Go
| Top | [g/b](https://github.com/g/b) |
## Python
| Top | [p/c](https://github.com/p/c) |
## Go
| Top | [g/d](https://github.com/g/d) |
`)
	urls := func(entries []matrixEntry) []string {
		var out []string
		for _, e := range entries {
			out = append(out, strings.TrimPrefix(e.RepoURL, "https://github.com/"))
		}
		return out
	}
	if got := urls(parseMatrix(data, 2, "")); !slices.Equal(got, []string{"g/a", "g/b", "p/c", "g/d"}) {
		t.Errorf("two languages = %v", got)
	}
	if got := urls(parseMatrix(data, 1, "")); !slices.Equal(got, []string{"g/a", "g/b", "g/d"}) {
		t.Errorf("one language = %v", got)
	}
}

func TestParseMatrixToleratesStrayLines(t *testing.T) {
	// A framework heading before any language keeps its text; prose with a
	// link, a pipe line without one and a row with a bare URL are no entries.
	data := []byte(`### + Bare
| Top | [b/a](https://github.com/b/a) |
See https://github.com/b/prose for more.
| note
| Top | https://github.com/b/bare |
`)
	entries := parseMatrix(data, 0, "")
	if len(entries) != 1 || entries[0].Framework != "+ Bare" || entries[0].RepoURL != "https://github.com/b/a" {
		t.Errorf("entries = %+v", entries)
	}
}

// TestParseMatrixSkipsAPipeLineWithALinkButNoRepositoryColumn: a line that
// starts with a pipe and mentions a link, but has no third cell, is no row;
// reading its repository column used to end the run with a panic. A row
// without its closing pipe still has both columns, and a line that does not
// start with a pipe is no row at all, whatever cells it has.
func TestParseMatrixSkipsAPipeLineWithALinkButNoRepositoryColumn(t *testing.T) {
	data := []byte("## Go\n" +
		"| see https://github.com/b/short\n" +
		"x | Top | [b/x](https://github.com/b/x) |\n" +
		"| Top | [b/a](https://github.com/b/a) |\n" +
		"| Top | [b/c](https://github.com/b/c)\n")
	entries := parseMatrix(data, 0, "")
	var urls []string
	for _, e := range entries {
		urls = append(urls, e.RepoURL)
	}
	if !slices.Equal(urls, []string{"https://github.com/b/a", "https://github.com/b/c"}) {
		t.Errorf("entries = %q", urls)
	}
}

func TestBenchmarkCorpusNamesTheFirstFailure(t *testing.T) {
	matrix := []byte("## Go\n| Top | [a/one](https://github.com/a/one) |\n| Top | [a/two](https://github.com/a/two) |\n")
	cloner := func(string, string) (string, error) { return "sha", nil }
	benchFail := func(dir string, _ Options) (*RepoAudit, error) {
		return nil, errors.New("broken " + filepath.Base(dir))
	}
	report, err := BenchmarkCorpus(matrix, Options{WarmRuns: 1}, cloner, benchFail)
	if report != nil || err == nil || !strings.HasSuffix(err.Error(), "first: broken a_one") {
		t.Fatalf("report %+v, err %v", report, err)
	}
}

func TestBenchmarkCorpusWithoutEntriesIsAnEmptyReport(t *testing.T) {
	report, err := BenchmarkCorpus([]byte("# nothing here\n"), Options{WarmRuns: 1}, nil, nil)
	if err != nil || report == nil || len(report.Repos) != 0 || len(report.Skipped) != 0 {
		t.Fatalf("report %+v, err %v", report, err)
	}
}

func TestSummarizeReasonKeepsTheFirstLineWithoutGitErrors(t *testing.T) {
	if got := summarizeReason(errors.New("  boom  \nmore detail")); got != "boom" {
		t.Errorf("summarizeReason = %q, want boom", got)
	}
}
