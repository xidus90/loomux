package hooks

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

// DefaultBudget is how long post-edit may take before it lets the lanes it
// has not reached go: an edit waits for its checks, but not forever.
const DefaultBudget = 50 * time.Second

// EditEnv is what a post-edit run needs from outside: what starts a tool and
// finds it on the PATH, which binary {loomux} names, the budget, the clock,
// and whether Godot has imported a project. A nil ImportReady asks the disk.
type EditEnv struct {
	Start       func(child.Spec) child.Result
	Look        func(string) (string, error)
	Loomux      string
	Budget      time.Duration
	Now         func() time.Time
	ImportReady func(dir string) bool
}

// The seams no project can provoke: presets that fail to load, a plan that
// fails, and a binary that cannot name its own path.
var (
	editPresets    = verify.LoadPresets
	editPlan       = verify.Plan
	editExecutable = os.Executable
)

// PostToolUse checks the file an edit touched, with the real tools.
func PostToolUse(stdin io.Reader, stdout, stderr io.Writer, root string, budget time.Duration) int {
	loomux, err := editExecutable()
	if err != nil {
		loomux = "loomux"
	}
	return RunPostEdit(stdin, stdout, stderr, root, EditEnv{
		Start: child.Run, Look: exec.LookPath, Loomux: loomux, Budget: budget, Now: time.Now, ImportReady: verify.ImportReady,
	})
}

// RunPostEdit runs the lanes of the `edit` profile for the edited file's
// stack, as [verify] and the presets lay them out, or the wiki lint for a
// wiki page. A red lane blocks the edit with 2; a config it cannot read ends
// with 1, which shows the error and blocks nothing.
func RunPostEdit(stdin io.Reader, stdout, stderr io.Writer, root string, env EditEnv) int {
	raw := editedFile(stdin)
	if raw == "" {
		return ExitOK
	}
	// Tools run in their area, so every path handed to them must be absolute.
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	fail := func(err error) int {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: %v\n", err)
		return ExitInternal
	}
	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		return fail(err)
	}
	ext := strings.ToLower(filepath.Ext(raw))
	if slices.Contains(eff.Ignored, ext) {
		return ExitOK
	}
	runID := verify.NewRunID(env.Now(), os.Getpid())
	var jobs []verify.Job
	if eff.Extensions[ext] == "wiki" {
		jobs = wikiJobs(eff, facts, root, raw)
	} else if jobs, err = editJobs(eff, root, raw, runID, env); err != nil {
		return fail(err)
	}
	if err := verify.PrepareCover(root); err != nil {
		return fail(err)
	}
	outs := verify.Run(jobs, verify.RunOptions{
		Scope: verify.ScopeEdit, MaxParallel: eff.Config.MaxParallel, Timeout: eff.Config.Timeout,
		Budget: env.Budget, Start: env.Start, Look: env.Look, Now: env.Now,
	})
	aside := ""
	if eff.Extensions[ext] == "go" && !slices.ContainsFunc(outs, func(o verify.Outcome) bool { return verify.Red(o.State, verify.ScopeEdit) }) {
		if rel, ok := relInRoot(root, raw); ok {
			aside = blastAside(root, rel, func(p string) ([]byte, error) {
				return os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
			})
		}
	}
	code := ExitOK
	if verify.WriteEdit(stdout, stderr, outs, aside) != 0 {
		code = ExitDenied
	}
	// The same rule as a check: a red edit keeps its files for whoever looks
	// into it, and a file left behind costs disk, not the verdict.
	if err := verify.CleanCover(root, runID, code == ExitOK); err != nil {
		fmt.Fprintf(stderr, "loomux hook post-tool-use: cleaning coverage files: %v\n", err)
	}
	return code
}

// editedFile is the path an edit payload names, "" when it names none or
// cannot be read. A notebook edit names its file under notebook_path.
func editedFile(stdin io.Reader) string {
	var payload HookPayload
	if err := json.NewDecoder(stdin).Decode(&payload); err != nil {
		return ""
	}
	raw, _ := payload.ToolInput["file_path"].(string)
	if raw == "" {
		raw, _ = payload.ToolInput["notebook_path"].(string)
	}
	return raw
}

// editLoad lays the project's config over the presets for what root holds.
func editLoad(root string, facts detect.Facts) (verify.Effective, error) {
	cfg, err := verify.ReadConfig(root)
	if err != nil {
		return verify.Effective{}, err
	}
	presets, err := editPresets()
	if err != nil {
		return verify.Effective{}, err
	}
	return verify.Resolve(cfg, presets, facts)
}

// editJobs plans the edit profile for a file inside root; a file outside it
// belongs to no lane of this project.
func editJobs(eff verify.Effective, root, raw, runID string, env EditEnv) ([]verify.Job, error) {
	rel, ok := relInRoot(root, raw)
	if !ok {
		return nil, nil
	}
	// The defaults hold an edit profile and a config can only replace it, so
	// asking for it cannot fail.
	kinds, _ := verify.ExpandProfile(eff.Config, "edit")
	ready := env.ImportReady
	if ready == nil {
		ready = verify.ImportReady
	}
	return editPlan(eff, verify.Request{Kinds: kinds, Scope: verify.ScopeEdit, File: rel}, verify.PlanEnv{
		Root:        root,
		Loomux:      env.Loomux,
		RunID:       runID,
		HasTests:    verify.HasTests,
		ImportReady: ready,
	})
}

// wikiJobs is the one lane a wiki page gets: the lint this binary carries,
// run in this process. It runs only for a page inside a wiki the project has
// -- detected or declared through [layout] wiki -- and not when
// [verify.wiki] lint = false switched it off.
func wikiJobs(eff verify.Effective, facts detect.Facts, root, raw string) []verify.Job {
	// wikiDirFor and not detection first: the manifest's [layout] wiki
	// outranks what detection guesses.
	wikiDir := wikiDirFor(root)
	if !slices.Contains(stacksWithWiki(facts.Stacks, root), "wiki") || !isWikiPath(raw, root, wikiDir) || eff.Config.Stacks["wiki"]["lint"].Lane.Off {
		return nil
	}
	// Claude names the edited file absolutely; a relative one comes from a
	// caller that speaks from the root.
	page := raw
	if !filepath.IsAbs(page) {
		page = filepath.Join(root, page)
	}
	wikiRoot := filepath.Join(root, filepath.FromSlash(wikiDir))
	return []verify.Job{{
		Name: "lint/wiki", Kind: "lint", Stack: "wiki", Area: ".", Origin: "in-process", Dir: root, After: -1,
		Fn: func() (string, error) {
			var report strings.Builder
			if wiki.LintReport(page, wikiRoot, &report) != 0 {
				return report.String(), fmt.Errorf("wiki lint found errors in %s", raw)
			}
			return "", nil
		},
	}}
}

// wikiDirFor answers where this project's wiki bundle is, seen from its root.
//
// The answer comes from the resolver the lint itself uses, wiki.Root: the
// manifest's [layout] wiki, then docs/wiki, then wiki, then a neighbour
// bundle. What stood here read `bundle` out of ultraloom's
// .ultraloom/answers.toml line by line -- a file loomux does not write, and a
// second opinion about the wiki beside the manifest's own.
//
// Detection still answers where wiki.Root found nothing: a project that
// declares a wiki it has not created yet keeps the place detection named for
// it, and "wiki/" is the last word.
func wikiDirFor(projectRoot string) string {
	if dir := wiki.Root(projectRoot); dir != "" {
		return relativeToRoot(projectRoot, dir)
	}
	if detected := detect.Detect(os.DirFS(projectRoot)).WikiPath; detected != "" {
		return filepath.ToSlash(detected)
	}
	return "wiki/"
}

//coverage:exempt filepath.Rel fails only across volumes, and wiki.Root builds every answer from projectRoot itself
func relativeToRoot(projectRoot, dir string) string {
	rel, err := filepath.Rel(projectRoot, dir)
	if err != nil {
		return filepath.ToSlash(dir)
	}
	return filepath.ToSlash(rel)
}

// declaresWikiLayout answers whether this project says it has a wiki through
// the one key loomux has for that, [layout] wiki.
//
// detect.Detect does not read that key -- it knows [wiki], `wiki = true` and
// an okf_version in the bundle -- so without this the wiki lane never started
// in a loomux project, the pilot's own repository first among them. The
// manifest's word alone is not enough: wiki.Root has to arrive at the declared
// place, or the lane would lint against a directory nobody created.
func declaresWikiLayout(projectRoot string) bool {
	manifest, err := config.ReadManifest(projectRoot)
	if err != nil {
		return false
	}
	layout, err := manifest.WikiLayout()
	if err != nil || layout == "" {
		return false
	}
	return wiki.Root(projectRoot) == filepath.Join(projectRoot, filepath.FromSlash(layout))
}

// stacksWithWiki adds the wiki lane to what detection found, wherever the
// manifest declares a wiki that detection cannot see.
func stacksWithWiki(stacks []string, projectRoot string) []string {
	if slices.Contains(stacks, "wiki") || !declaresWikiLayout(projectRoot) {
		return stacks
	}
	return append(slices.Clone(stacks), "wiki")
}

func isWikiPath(rawPath, root, configuredWikiDir string) bool {
	wikiDir := strings.TrimSuffix(filepath.ToSlash(configuredWikiDir), "/")
	if wikiDir == "" {
		wikiDir = "wiki"
	}

	norm := strings.TrimPrefix(filepath.ToSlash(rawPath), "./")

	if strings.HasPrefix(norm, wikiDir+"/") || norm == wikiDir ||
		strings.Contains(norm, "/"+wikiDir+"/") ||
		strings.HasPrefix(norm, "docs/wiki/") || strings.HasPrefix(norm, "wiki/") {
		return true
	}

	if root != "" {
		rel, err := filepath.Rel(root, rawPath)
		if err == nil {
			relNorm := strings.TrimPrefix(filepath.ToSlash(rel), "./")
			if strings.HasPrefix(relNorm, wikiDir+"/") || relNorm == wikiDir ||
				strings.Contains(relNorm, "/"+wikiDir+"/") ||
				strings.HasPrefix(relNorm, "docs/wiki/") || strings.HasPrefix(relNorm, "wiki/") {
				return true
			}
		}
	}
	return false
}

// StackForExtension names the stack whose lanes an edit to a file with this
// extension (dot included) runs, so callers outside the hook agree with it.
func StackForExtension(ext string) (string, bool) {
	presets, err := editPresets()
	if err != nil {
		return "", false
	}
	stack, ok := presets.Extensions[ext]
	return stack, ok
}

// EditLaneCommands are the commands post-edit runs for a project with these
// facts when no config changes them: the variant a detected signal selects,
// each lane's form for one file where it has one, stack by stack in byte
// order. {file} stays as it is, since the audit reads only the tool.
func EditLaneCommands(facts detect.Facts) []string {
	presets, err := editPresets()
	if err != nil {
		return nil
	}
	// An empty document is the default config, which holds an edit profile
	// and forms no ring with the presets, so none of these steps can fail.
	cfg, _ := verify.ParseConfig("", map[string]any{})
	eff, _ := verify.Resolve(cfg, presets, facts)
	kinds, _ := verify.ExpandProfile(cfg, "edit")
	var out []string
	for _, stack := range eff.Active {
		for _, kind := range kinds {
			r := eff.Stacks[stack][kind]
			if !r.Defined {
				continue
			}
			cmds := r.Lane.OnFile
			if len(cmds) == 0 {
				cmds = r.Lane.Commands
			}
			for _, c := range cmds {
				out = append(out, strings.ReplaceAll(c, "{loomux}", "loomux"))
			}
		}
	}
	return out
}
