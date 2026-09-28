package setup

// The tests in this file hold what the mutation round over this package
// found untested: a part switched off, a first error, a link in a parent
// directory, and the values that only look like each other.

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/selfupdate"
)

// linkDir points link at target, by a symlink or, where the account may not
// make one, by a directory junction; both are links CheckParents refuses.
func linkDir(t *testing.T, target, link string) {
	t.Helper()
	err := os.Symlink(target, link)
	if err == nil {
		return
	}
	if runtime.GOOS != "windows" {
		t.Skipf("no symlink here: %v", err)
	}
	if out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); jerr != nil {
		t.Skipf("neither a symlink (%v) nor a junction (%v: %s) can be made here", err, jerr, out)
	}
}

func TestAPartSwitchedOffPlansNothing(t *testing.T) {
	root := world(t, map[string]string{".git/": ""})
	old := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { lookPath = old })
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.Modules[schema.Graph] = false
	for _, part := range []string{"config", "gitignore", "tools", "area"} {
		c.Parts[part] = false
	}
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{configPath, ".gitignore"} {
		if _, ok := changeOf(p, path); ok {
			t.Errorf("%s is planned with its part off", path)
		}
	}
	if slices.Contains(actions(p), "area-add") {
		t.Errorf("area-add is planned with its part off: %v", actions(p))
	}
	if hasNote(p, "is not on PATH") {
		t.Errorf("tools are checked with their part off: %v", p.Notes)
	}
}

func TestThePlanNamesItsFirstError(t *testing.T) {
	root := world(t, map[string]string{
		configPath:              "[commit]\nthreshold = \"x\"\n",
		".claude/settings.json": "not json",
	})
	f := gather(t, root, "")
	_, err := Build(f, DefaultChoice(f, Answers{}), reader(root))
	if err == nil || !strings.Contains(err.Error(), "config.toml") || strings.Contains(err.Error(), "settings.json") {
		t.Errorf("err = %v", err)
	}
}

func TestNoRepositoryAndNoGitPartIsNoNote(t *testing.T) {
	p := plan(t, gather(t, world(t, map[string]string{}), ""))
	if hasNote(p, "skipped; the project is no git repository") {
		t.Errorf("notes = %v", p.Notes)
	}
}

func TestAHooksPathInsideTheProjectIsNoNote(t *testing.T) {
	p := plan(t, gather(t, world(t, map[string]string{".git/": ""}), "tools/hooks"))
	if hasNote(p, "lies outside the project") {
		t.Errorf("notes = %v", p.Notes)
	}
}

func TestOurMergeHookInGitsOwnDirectoryIsThere(t *testing.T) {
	root := world(t, map[string]string{
		".git/hooks/pre-commit": "#!/bin/sh\nsh ci/gate.sh\n",
		".git/hooks/post-merge": maintenance.HookText(),
	})
	f := gather(t, root, "")
	if !f.MergeHook {
		t.Fatal("Facts.MergeHook is false beside our hook in .git/hooks")
	}
	p := plan(t, f)
	// binary-install stands in the plan before the merge hook is looked for.
	if !slices.Contains(actions(p), "binary-install") || slices.Contains(actions(p), "merge-hook") {
		t.Errorf("actions = %v", actions(p))
	}
}

func TestGatherOutsideARepositoryAsksGitNothing(t *testing.T) {
	f := gather(t, world(t, map[string]string{}), "")
	if f.GitHooksDir != "" {
		t.Errorf("GitHooksDir = %q outside a repository", f.GitHooksDir)
	}
}

func TestGatherCarriesGitsError(t *testing.T) {
	root := world(t, map[string]string{".git/": ""})
	boom := errors.New("boom")
	_, err := Gather(root, t.TempDir(), tested, func(_ string, argv ...string) (string, error) {
		if slices.Contains(argv, "rev-parse") {
			return "", boom
		}
		return "", nil
	})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want it to carry %v", err, boom)
	}
}

func TestAGraphWithoutAStackIsNoDefault(t *testing.T) {
	f := gather(t, world(t, map[string]string{".git/": ""}), "")
	if len(f.Detect.Stacks) != 0 {
		t.Fatalf("stacks = %v", f.Detect.Stacks)
	}
	if DefaultChoice(f, Answers{}).Parts["graph-build"] {
		t.Error("graph-build is on for a project without a stack")
	}
}

func TestMCPServersAsTheFirstKeyStaysOne(t *testing.T) {
	out, err := withMCPServer([]byte(`{"mcpServers": {"z": {}}, "other": 1}`))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), `"mcpServers"`); n != 1 || !strings.Contains(string(out), `"loomux"`) {
		t.Errorf("mcpServers %d times:\n%s", n, out)
	}
}

func TestAModuleOnAndSetToTrueKeepsItsLine(t *testing.T) {
	text := "[modules]\nbrain = true\n"
	got, err := configText(text, nil, Choice{})
	if err != nil || got != text {
		t.Errorf("text = %q, err = %v", got, err)
	}
}

func TestAnEditThatFailsIsNotOverwrittenByTheNext(t *testing.T) {
	c := Choice{Modules: map[schema.Module]bool{schema.Hooks: false, schema.Brain: false}}
	if _, err := configText("modules = { hooks = true }\n", nil, c); err == nil {
		t.Error("a failed edit of hooks was lost behind the edit of brain")
	}
}

func TestAnotherRuleInTheTableIsNotTheCatalogs(t *testing.T) {
	for _, tc := range []struct{ stack, before, want string }{
		{"uv", "[[policy.commands.rules]]\nregex = 'curl'\nreason = \"x\"\n", "regex = '" + pipRegex + "'"},
		{"django", "[[policy.paths.rules]]\nmatch = [\"x/**\"]\nreason = \"x\"\n", migrationsGlob},
	} {
		got, err := configText(tc.before, []string{tc.stack}, Choice{})
		if err != nil || !strings.Contains(got, tc.want) {
			t.Errorf("%s: text =\n%s\nerr = %v", tc.stack, got, err)
		}
	}
}

func TestARefusedConfigurationIsNamedOnce(t *testing.T) {
	_, err := configText("[commit]\nthreshold = \"x\"\n", nil, Choice{})
	if err == nil || strings.Count(err.Error(), "config.toml") != 1 {
		t.Errorf("err = %v", err)
	}
}

func TestAFailedBuildOfTheCheckoutIsAFailedBinaryStep(t *testing.T) {
	root := world(t, map[string]string{})
	p := Plan{Actions: []Action{{Part: "binary", ID: "binary-build"}}}
	run := func(Action) error { return errors.New("no toolchain") }
	r, err := Apply(root, p, Choice{}, all, run, func() bool { return false }, "1", applyTime)
	if err != nil || !slices.Contains(r.Failed, "binary-build") {
		t.Errorf("report = %+v, err = %v", r, err)
	}
}

func TestApplyWritesNothingThroughALink(t *testing.T) {
	outside := t.TempDir()
	writeFile(t, outside, "settings.json", "theirs\n")
	root := world(t, map[string]string{})
	linkDir(t, outside, filepath.Join(root, ".claude"))
	p := Plan{Changes: []Change{{Part: "host-entries", Path: ".claude/settings.json", Before: "theirs\n", After: "ours\n", Exists: true}}}
	if _, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime); err == nil {
		t.Error("a change through a linked directory is no error")
	}
	if got := read(t, outside, "settings.json"); got != "theirs\n" {
		t.Errorf("the file behind the link reads %q", got)
	}
}

func TestTheStateIsNotWrittenThroughALink(t *testing.T) {
	outside := t.TempDir()
	root := world(t, map[string]string{".loomux/": ""})
	linkDir(t, outside, filepath.Join(root, ".loomux", "state"))
	if _, err := Apply(root, Plan{}, Choice{}, all, nil, there, "1", applyTime); err == nil {
		t.Error("the state was written through a linked directory without an error")
	}
	if exists(outside, "answers.toml") {
		t.Error("answers.toml landed behind the link")
	}
}

// redone is a Redo that marks what it made, so a test sees whether Apply
// called it.
func redone(current string) (string, error) { return current + "redone\n", nil }

func TestAnUnchangedFileTakesThePlannedTextNotARedo(t *testing.T) {
	root := world(t, map[string]string{"notes.txt": "old\n"})
	p := Plan{Changes: []Change{{Part: "x", Path: "notes.txt", Before: "old\n", After: "planned\n", Exists: true, Redo: redone}}}
	if _, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime); err != nil {
		t.Fatal(err)
	}
	if got := read(t, root, "notes.txt"); got != "planned\n" {
		t.Errorf("notes.txt = %q, want the planned text", got)
	}
}

func TestAChangedFileIsRedoneNotRefused(t *testing.T) {
	root := world(t, map[string]string{"notes.txt": "theirs\n"})
	p := Plan{Changes: []Change{{Part: "x", Path: "notes.txt", Before: "old\n", After: "planned\n", Exists: true, Redo: redone}}}
	if _, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime); err != nil {
		t.Fatal(err)
	}
	if got := read(t, root, "notes.txt"); got != "theirs\nredone\n" {
		t.Errorf("notes.txt = %q, want the redo over their text", got)
	}
}

// A directory where the change expects a file cannot be read; that is an
// error, neither a missing file to create nor a changed one.
func TestAFileThatCannotBeReadStopsTheChange(t *testing.T) {
	for name, redo := range map[string]func(string) (string, error){"redo": redone, "no redo": nil} {
		t.Run(name, func(t *testing.T) {
			root := world(t, map[string]string{"notes.txt/": ""})
			p := Plan{Changes: []Change{{Part: "x", Path: "notes.txt", Before: "old\n", After: "planned\n", Exists: true, Redo: redo}}}
			_, err := Apply(root, p, Choice{}, all, nil, there, "1", applyTime)
			var pathErr *fs.PathError
			if !errors.As(err, &pathErr) {
				t.Errorf("err = %v, want the read error", err)
			}
		})
	}
}

// Outside a repository nothing about git hooks is looked at: git is not
// asked for its hook directory, and no hook is read from the working
// directory, where an empty hook directory would resolve.
func TestGatherOutsideARepositoryLooksAtNoHooks(t *testing.T) {
	root := world(t, map[string]string{})
	cwd := t.TempDir()
	writeFile(t, cwd, "pre-commit", "#!/bin/sh\n")
	writeFile(t, cwd, "post-merge", maintenance.HookText())
	t.Chdir(cwd)
	f, err := Gather(root, t.TempDir(), tested, func(_ string, argv ...string) (string, error) {
		if slices.Contains(argv, "rev-parse") {
			return "", errors.New("not a git repository")
		}
		return "", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.GitHooksLive || f.MergeHook {
		t.Errorf("GitHooksLive = %v, MergeHook = %v outside a repository", f.GitHooksLive, f.MergeHook)
	}
}

func TestADevelopmentBuildStillPlansClaudesEntries(t *testing.T) {
	f := gather(t, world(t, map[string]string{".claude/": ""}), "")
	f.Version = selfupdate.DevVersion
	p := plan(t, f)
	if _, ok := changeOf(p, ".claude/settings.json"); !ok || hasNote(p, "antigravity: no entries") {
		t.Errorf("paths = %v, notes = %v", paths(p), p.Notes)
	}
}

func TestTheModelPartOffLooksAtNoModel(t *testing.T) {
	f := gather(t, world(t, map[string]string{}), "")
	f.ModelProblem = "broken settings"
	c := DefaultChoice(f, Answers{})
	c.Parts["model"] = false
	p, err := Build(f, c, reader(f.Root))
	if err != nil {
		t.Fatal(err)
	}
	if hasNote(p, "model: skipped") {
		t.Errorf("notes = %v", p.Notes)
	}
}
