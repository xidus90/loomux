package setup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

func TestSelfUseChangesNothing(t *testing.T) {
	files := fixture(t)
	files["go.mod"] = checkoutGoMod
	files[".git/"] = ""
	root := world(t, files)
	f := gather(t, root, ".githooks")
	if !f.Checkout || f.Binary != hostfile.Checkout {
		t.Fatalf("facts = %+v, want a checkout calling its own binary", f)
	}
	p := plan(t, f)
	for _, c := range p.Changes {
		if c.Path == ".claude/settings.json" || c.Path == configPath || strings.HasPrefix(c.Path, ".githooks/") {
			t.Errorf("self-use changes %s:\n%s", c.Path, c.After)
		}
	}
	if got := actions(p); !slices.Equal(got, []string{"binary-build"}) {
		t.Errorf("actions = %v, want only binary-build", got)
	}
	if !hasNote(p, ".githooks/pre-commit: kept; it runs a gate already") {
		t.Errorf("notes = %v", p.Notes)
	}

	// A fresh clone has no core.hooksPath yet: the checked-in hooks stay and
	// only the setting is planned.
	p = plan(t, gather(t, root, ""))
	if len(p.Changes) != 0 || !slices.Equal(actions(p), []string{"binary-build", "hooks-path"}) {
		t.Errorf("fresh clone: changes = %v, actions = %v", paths(p), actions(p))
	}
	for _, name := range []string{"commit-msg", "pre-commit", "pre-push"} {
		if !hasNote(p, ".githooks/"+name+": kept") {
			t.Errorf("fresh clone: %s not named in %v", name, p.Notes)
		}
	}
}

func TestNoActionIsPlannedWhoseResultIsThere(t *testing.T) {
	const ours = "#!/bin/sh\n# loomux post-merge hook -- records\n"
	root := world(t, map[string]string{
		"go.mod": goMod, ".git/": "",
		".loomux/state/graph/wiring.json": "{}",
		".githooks/post-merge":            ours,
	})
	writeFile(t, os.Getenv("LOCALAPPDATA"), "loomux/bin/loomux.exe", "binary")
	// core.hooksPath already points at .githooks, where our hook stands.
	f := gather(t, root, ".githooks")
	if !f.BinaryThere || !f.MergeHook || !f.Graph {
		t.Fatalf("facts = %+v", f)
	}
	if got := actions(plan(t, f)); !slices.Equal(got, []string{"area-add"}) {
		t.Errorf("actions = %v, want only area-add", got)
	}
	// An absolute core.hooksPath is read as it stands.
	if f := gather(t, root, filepath.ToSlash(filepath.Join(root, ".githooks"))); !f.MergeHook {
		t.Error("absolute core.hooksPath: hook not found")
	}
	// Without core.hooksPath the plan sets it to .githooks, so our hook
	// there counts and one in git's own directory would not.
	f = gather(t, root, "")
	if f.MergeHook {
		t.Error("git's own directory has no hook")
	}
	if slices.Contains(actions(plan(t, f)), "merge-hook") {
		t.Error("merge-hook planned though .githooks has ours")
	}
	writeFile(t, root, ".githooks/post-merge", "#!/bin/sh\necho mine\n")
	if !slices.Contains(actions(plan(t, gather(t, root, ""))), "merge-hook") {
		t.Error("a foreign post-merge counts as ours")
	}
}

func TestBinaryPathNeedsLocalAppData(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	if got := BinaryPath("root", hostfile.Canonical); got != "" {
		t.Errorf("canonical = %q", got)
	}
	if got := BinaryPath("root", hostfile.Checkout); got != filepath.Join("root", "bin", "loomux.exe") {
		t.Errorf("checkout = %q", got)
	}
}

func TestAFreshRepositoryGetsEveryPart(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	if f.Checkout || f.Binary != hostfile.Canonical || !slices.Equal(f.Hosts, []hosts.Host{hosts.HostClaude}) {
		t.Fatalf("facts = %+v", f)
	}
	p := plan(t, f)
	want := []string{
		".gitignore", "AGENTS.md", ".mcp.json", ".claude/settings.json",
		".githooks/commit-msg", ".githooks/pre-commit", ".githooks/pre-push",
		".claude/skills/verify-until-green/SKILL.md",
		".claude/skills/brain-ingest/SKILL.md", ".claude/skills/brain-land/SKILL.md",
		".claude/skills/brain-research/SKILL.md", ".claude/skills/brain-review/SKILL.md",
		".claude/skills/brain-wiki-plan/SKILL.md",
	}
	if got := paths(p); !slices.Equal(got, want) {
		t.Errorf("changes = %v\nwant      %v", got, want)
	}
	wantActions := []string{"binary-install", "hooks-path", "area-add", "merge-hook", "graph-build"}
	if got := actions(p); !slices.Equal(got, wantActions) {
		t.Errorf("actions = %v, want %v", got, wantActions)
	}
	settings, _ := changeOf(p, ".claude/settings.json")
	if n := strings.Count(settings.After, hostfile.Canonical[1:len(hostfile.Canonical)-1]); n != 6 {
		t.Errorf("settings.json calls the canonical binary %d times, want 6:\n%s", n, settings.After)
	}
	mcp, _ := changeOf(p, ".mcp.json")
	wantMCP := "{\n  \"mcpServers\": {\n    \"loomux\": {\n      \"command\": \"${LOCALAPPDATA}/loomux/bin/loomux.exe\",\n" +
		"      \"args\": [\n        \"mcp\",\n        \"--channel\",\n        \"local\"\n      ]\n    }\n  }\n}\n"
	if mcp.After != wantMCP || mcp.Exists {
		t.Errorf(".mcp.json =\n%s", mcp.After)
	}
	hook, _ := changeOf(p, ".githooks/pre-commit")
	if !strings.Contains(hook.After, `exec "${LOCALAPPDATA}/loomux/bin/loomux.exe" check precommit`) {
		t.Errorf("pre-commit =\n%s", hook.After)
	}
	if len(p.Notes) != 0 {
		t.Errorf("notes = %v", p.Notes)
	}
}

func TestABrokenSettingsFileStopsThePlan(t *testing.T) {
	root := world(t, map[string]string{".claude/settings.json": "{not json"})
	_, err := Build(gather(t, root, ""), DefaultChoice(gather(t, root, ""), Answers{}), reader(root))
	if err == nil || !strings.Contains(err.Error(), ".claude/settings.json") {
		t.Fatalf("err = %v", err)
	}
}

func TestAForeignHostEntryIsKeptAndNamed(t *testing.T) {
	settings := `{"hooks": {"SessionStart": [{"hooks": [{"type": "command", "command": "ulguard start"}]}]}}`
	root := world(t, map[string]string{".claude/settings.json": settings})
	p := plan(t, gather(t, root, ""))
	c, ok := changeOf(p, ".claude/settings.json")
	if !ok || !strings.Contains(c.After, "ulguard start") || !hasNote(p, ".claude/settings.json: SessionStart/ keeps a hook of the project") {
		t.Errorf("settings =\n%s\nnotes = %v", c.After, p.Notes)
	}
}

func TestAnExistingHookIsKeptAndNamed(t *testing.T) {
	root := world(t, map[string]string{
		".git/":                "",
		".githooks/pre-commit": "#!/bin/sh\nulguard check\n",
		".githooks/pre-push":   "#!/bin/sh\necho mine\n",
	})
	p := plan(t, gather(t, root, ""))
	for _, path := range []string{".githooks/pre-commit", ".githooks/pre-push"} {
		if _, ok := changeOf(p, path); ok {
			t.Errorf("%s is rewritten", path)
		}
	}
	if !hasNote(p, ".githooks/pre-commit: kept; it runs a gate already") ||
		!hasNote(p, ".githooks/pre-push: kept; a hook of the project is already there") {
		t.Errorf("notes = %v", p.Notes)
	}
	if _, ok := changeOf(p, ".githooks/commit-msg"); !ok {
		t.Errorf("the missing commit-msg is not planned")
	}
}

func TestALiveHookInDotGitHooksKeepsItsDirectory(t *testing.T) {
	root := world(t, map[string]string{
		".git/hooks/pre-commit":        "#!/bin/sh\nsh ci/gate.sh\n",
		".git/hooks/pre-push.sample":   "#!/bin/sh\n",
		".git/hooks/commit-msg.sample": "#!/bin/sh\n",
	})
	p := plan(t, gather(t, root, ""))
	if slices.Contains(actions(p), "hooks-path") {
		t.Errorf("hooks-path would switch off .git/hooks/pre-commit: %v", actions(p))
	}
	if _, ok := changeOf(p, ".git/hooks/pre-commit"); ok || !hasNote(p, ".git/hooks/pre-commit: kept; it runs a gate already") {
		t.Errorf("changes = %v, notes = %v", paths(p), p.Notes)
	}
	for _, name := range []string{"commit-msg", "pre-push"} {
		if _, ok := changeOf(p, ".git/hooks/"+name); !ok {
			t.Errorf("%s is not planned under .git/hooks: %v", name, paths(p))
		}
	}
	if strings.Contains(strings.Join(paths(p), " "), ".githooks/") {
		t.Errorf("changes = %v", paths(p))
	}
}

func TestOnlySampleHooksChangeNothing(t *testing.T) {
	root := world(t, map[string]string{
		".git/hooks/pre-commit.sample": "#!/bin/sh\n",
		".git/hooks/pre-push.sample":   "#!/bin/sh\n",
	})
	p := plan(t, gather(t, root, ""))
	if !slices.Contains(actions(p), "hooks-path") {
		t.Errorf("actions = %v, want hooks-path", actions(p))
	}
	if _, ok := changeOf(p, ".githooks/pre-commit"); !ok {
		t.Errorf("changes = %v", paths(p))
	}
}

// linkedWorktree makes a repository with one commit and a linked worktree
// of it, and returns the worktree's root; hook, when not empty, becomes the
// main repository's .git/hooks/pre-commit.
func linkedWorktree(t *testing.T, hook string) (main, wt string) {
	t.Helper()
	world(t, map[string]string{})
	// Resolved to the long spelling git answers with: where TEMP is an 8.3
	// short path (C:\Users\RUNNER~1 on a GitHub runner), t.TempDir() hands
	// out the short form.
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	main, wt = filepath.Join(base, "main"), filepath.Join(base, "wt")
	if err := os.Mkdir(main, 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, main, "init", "-q")
	run(t, main, "commit", "-q", "--allow-empty", "-m", "init")
	run(t, main, "worktree", "add", "-q", wt)
	if hook != "" {
		writeFile(t, main, ".git/hooks/pre-commit", hook)
	}
	return main, wt
}

func TestAWorktreeLeavesTheSharedHooksAlone(t *testing.T) {
	main, wt := linkedWorktree(t, "#!/bin/sh\nsh ci/gate.sh\n")
	f, err := Gather(wt, t.TempDir(), tested, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if !f.GitHooksLive || !samePath(f.GitHooksDir, filepath.Join(main, ".git", "hooks")) {
		t.Fatalf("hook dir = %q, live = %v", f.GitHooksDir, f.GitHooksLive)
	}
	p := plan(t, f)
	if slices.Contains(actions(p), "hooks-path") {
		t.Errorf("hooks-path would write the shared config: %v", actions(p))
	}
	for _, path := range paths(p) {
		if strings.Contains(path, "hooks/") {
			t.Errorf("a hook is written: %s", path)
		}
	}
	if !hasNote(p, "git hooks live in "+filepath.ToSlash(f.GitHooksDir)+", shared with other checkouts; init leaves them") {
		t.Errorf("notes = %v", p.Notes)
	}
}

// link makes link point at target: a symlink where the system allows one,
// a junction on Windows otherwise; the test is skipped when neither works.
func link(t *testing.T, target, link string) {
	t.Helper()
	if os.Symlink(target, link) == nil {
		return
	}
	if runtime.GOOS == "windows" && exec.Command("cmd", "/c", "mklink", "/J", link, target).Run() == nil {
		return
	}
	t.Skip("neither a symlink nor a junction can be made here")
}

func TestARootReachedThroughALinkKeepsItsHooksInside(t *testing.T) {
	real := world(t, map[string]string{".git/": "", ".git/hooks/pre-commit": "#!/bin/sh\nsh ci/gate.sh\n"})
	real, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "via")
	link(t, real, root)
	// git answers with the resolved spelling, not the one init was given.
	answer := func(_ string, argv ...string) (string, error) {
		if slices.Contains(argv, "rev-parse") {
			return filepath.ToSlash(filepath.Join(real, ".git", "hooks")), nil
		}
		return "", nil
	}
	f, err := Gather(root, t.TempDir(), tested, answer)
	if err != nil {
		t.Fatal(err)
	}
	p := plan(t, f)
	if hasNote(p, "shared with other checkouts") || slices.Contains(actions(p), "hooks-path") {
		t.Errorf("a linked root reads as elsewhere: actions = %v, notes = %v", actions(p), p.Notes)
	}
	if _, ok := changeOf(p, ".git/hooks/pre-push"); !ok || !hasNote(p, ".git/hooks/pre-commit: kept") {
		t.Errorf("changes = %v, notes = %v", paths(p), p.Notes)
	}
}

func TestWithinComparesAMissingRootBySpelling(t *testing.T) {
	root := filepath.Join(t.TempDir(), "gone")
	for path, want := range map[string]string{
		"hooks":                                  "hooks",
		filepath.Join(root, "a", "b"):            "a/b",
		filepath.Join(filepath.Dir(root), "out"): "",
	} {
		if got, inside := within(root, path); got != want || inside != (want != "") {
			t.Errorf("%s: %q, %v, want %q", path, got, inside, want)
		}
	}
	existing := t.TempDir()
	if got, inside := within(existing, existing); got != "." || !inside {
		t.Errorf("root itself: %q, %v", got, inside)
	}
}

func TestAWorktreeWithoutLiveHooksIsAsBefore(t *testing.T) {
	_, wt := linkedWorktree(t, "")
	f, err := Gather(wt, t.TempDir(), tested, realGit)
	if err != nil {
		t.Fatal(err)
	}
	p := plan(t, f)
	if f.GitHooksLive || !slices.Contains(actions(p), "hooks-path") {
		t.Errorf("live = %v, actions = %v", f.GitHooksLive, actions(p))
	}
	if _, ok := changeOf(p, ".githooks/pre-commit"); !ok {
		t.Errorf("changes = %v", paths(p))
	}
}

func TestTheHookDirectoryFollowsCoreHooksPath(t *testing.T) {
	root := world(t, map[string]string{".git/": ""})
	f := gather(t, root, "tools/hooks")
	p := plan(t, f)
	if _, ok := changeOf(p, "tools/hooks/pre-commit"); !ok || slices.Contains(actions(p), "hooks-path") {
		t.Errorf("changes = %v, actions = %v", paths(p), actions(p))
	}
	f.HooksPath = root + "/abs"
	if _, ok := changeOf(plan(t, f), "abs/pre-commit"); !ok {
		t.Errorf("an absolute hooks path inside the root is not followed")
	}
	f.HooksPath = "../elsewhere"
	p = plan(t, f)
	if !hasNote(p, "lies outside the project") || strings.Contains(strings.Join(paths(p), " "), "pre-commit") {
		t.Errorf("changes = %v, notes = %v", paths(p), p.Notes)
	}
}

func TestACheckoutWithoutHooksCallsItsBinaryFromTheTree(t *testing.T) {
	root := world(t, map[string]string{".git/": "", "go.mod": checkoutGoMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	p, _ := Build(f, c, reader(root))
	hook, _ := changeOf(p, ".githooks/pre-commit")
	if !strings.Contains(hook.After, `exec "./bin/loomux.exe" check precommit`) {
		t.Errorf("pre-commit =\n%s", hook.After)
	}
}

func TestAnExistingAgentsMDIsKept(t *testing.T) {
	root := world(t, map[string]string{"AGENTS.md": "# mine\n", ".claude/skills/brain-land/SKILL.md": "mine"})
	p := plan(t, gather(t, root, ""))
	if _, ok := changeOf(p, "AGENTS.md"); ok || !hasNote(p, "AGENTS.md: kept") {
		t.Errorf("changes = %v, notes = %v", paths(p), p.Notes)
	}
	if _, ok := changeOf(p, ".claude/skills/brain-land/SKILL.md"); ok {
		t.Errorf("an existing skill is rewritten")
	}
}

func TestNoRepositoryMeansNoGitParts(t *testing.T) {
	root := world(t, map[string]string{})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	if c.Parts["git-hooks"] || c.Parts["merge-hook"] {
		t.Errorf("git parts chosen without a repository: %v", c.Parts)
	}
	c.Parts["git-hooks"], c.Parts["merge-hook"] = true, true
	p, _ := Build(f, c, reader(root))
	if slices.Contains(actions(p), "merge-hook") || slices.Contains(actions(p), "hooks-path") ||
		!hasNote(p, "git-hooks: skipped") || !hasNote(p, "merge-hook: skipped") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
}

func TestAHostWithoutAHookFileGetsANote(t *testing.T) {
	root := world(t, map[string]string{".agents/skills/": ""})
	f := gather(t, root, "")
	if !slices.Equal(f.Hosts, []hosts.Host{hosts.HostAntigravity}) {
		t.Fatalf("hosts = %v", f.Hosts)
	}
	c := DefaultChoice(f, Answers{Hosts: []string{"codex", "nobody"}})
	if !slices.Equal(c.Hosts, []hosts.Host{hosts.HostCodex}) {
		t.Fatalf("hosts = %v", c.Hosts)
	}
	p, err := Build(f, c, reader(root))
	if err != nil || !hasNote(p, "codex: init writes nothing") {
		t.Errorf("err = %v, notes = %v", err, p.Notes)
	}
	for _, path := range paths(p) {
		if strings.HasPrefix(path, ".agents/") || strings.HasPrefix(path, ".claude/") {
			t.Errorf("codex gets %s", path)
		}
	}
}

const agyTrust = "antigravity: .agents/hooks.json loads only in a folder agy trusts (trustedWorkspaces)"

// Antigravity gets its two entries, in the form cmd.exe expands, and the
// skills under .agents/skills -- in a checkout too, where Claude's entries
// call the checkout's binary.
func TestAntigravityGetsItsEntriesAndSkills(t *testing.T) {
	root := world(t, map[string]string{".agents/skills/": "", ".claude/": "", ".git/": "", "go.mod": checkoutGoMod})
	installBinary(t)
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	if !slices.Equal(c.Hosts, []hosts.Host{hosts.HostClaude, hosts.HostAntigravity}) {
		t.Fatalf("hosts = %v", c.Hosts)
	}
	// A checkout has its skills checked in; here they are asked for.
	c.Parts["verify-skill"], c.Parts["brain-skills"] = true, true
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	ch, ok := changeOf(p, ".agents/hooks.json")
	if !ok || ch.Part != "host-entries" ||
		!strings.Contains(ch.After, `"command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook session-start --host antigravity --root .."`) ||
		!strings.Contains(ch.After, `"command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root .."`) ||
		!strings.Contains(ch.After, `"command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook post-tool-use --host antigravity --root .."`) ||
		!strings.Contains(ch.After, `"command": "%LOCALAPPDATA%/loomux/bin/loomux.exe hook stop --host antigravity --root .. --budget 270s"`) {
		t.Fatalf(".agents/hooks.json = %+v", ch)
	}
	if claude, _ := changeOf(p, ".claude/settings.json"); !strings.Contains(claude.After, "${CLAUDE_PROJECT_DIR}/bin/loomux.exe") {
		t.Errorf("claude's entries do not call the checkout:\n%s", claude.After)
	}
	for _, rel := range []string{".agents/skills/verify-until-green/SKILL.md", ".agents/skills/brain-land/SKILL.md"} {
		if _, ok := changeOf(p, rel); !ok {
			t.Errorf("no %s in %v", rel, paths(p))
		}
	}
	if !hasNote(p, agyTrust) || hasNote(p, "no entries or skills yet") {
		t.Errorf("notes = %v", p.Notes)
	}
	// A hook file that does not read stops the plan, named.
	writeFile(t, root, ".agents/hooks.json", "{not json")
	if _, err := Build(f, c, reader(root)); err == nil || !strings.Contains(err.Error(), ".agents/hooks.json") {
		t.Errorf("err = %v", err)
	}
}

// Antigravity's entries call the installed binary even in a checkout, so
// they are planned only where it stands or binary-install runs.
func TestAntigravityEntriesWaitForTheInstalledBinary(t *testing.T) {
	root := world(t, map[string]string{".agents/skills/": "", ".git/": "", "go.mod": checkoutGoMod})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.Parts["verify-skill"] = true
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := changeOf(p, ".agents/hooks.json"); ok || f.CanonicalThere {
		t.Error("entries planned without the installed binary")
	}
	want := "antigravity: no entries; they call %LOCALAPPDATA%/loomux/bin/loomux.exe, which is not installed; run loomux self-update"
	if !slices.Contains(p.Notes, want) || !hasNote(p, agyTrust) {
		t.Errorf("notes = %v", p.Notes)
	}
	if _, ok := changeOf(p, ".agents/skills/verify-until-green/SKILL.md"); !ok {
		t.Errorf("no skill in %v", paths(p))
	}
	installBinary(t)
	f = gather(t, root, "")
	if p, err = Build(f, c, reader(root)); err != nil {
		t.Fatal(err)
	}
	if _, ok := changeOf(p, ".agents/hooks.json"); !ok || hasNote(p, "which is not installed") {
		t.Errorf("paths = %v, notes = %v", paths(p), p.Notes)
	}
}

// An installed binary older than this init does not know Antigravity's hooks,
// and agy aborts on one that fails; so the entries wait for one at least as
// new. A development build has nothing to compare, and a binary-install in
// the same run brings the newest release, at least a released init.
func TestAntigravityEntriesWaitForAnInstalledBinaryAsNewAsInit(t *testing.T) {
	root := world(t, map[string]string{".agents/skills/": "", "go.mod": goMod})
	const older = "antigravity: no entries; the installed loomux 2.11.1 is older than this init 2.13.0; run loomux self-update"
	const unnamed = "antigravity: no entries; the installed loomux names no version; run loomux self-update"
	const dev = "antigravity: no entries; this init is the development build 0.0.0-dev, and no installed loomux can be compared with it; run a released loomux init"
	for _, c := range []struct {
		name, init, installed string
		there                 bool
		note                  string
	}{
		{"older", "2.13.0", "2.11.1", true, older},
		{"names none", "2.13.0", "", true, unnamed},
		{"as new", "2.13.0", "2.13.0", true, ""},
		{"newer", "2.13.0", "2.14.0", true, ""},
		{"development build", selfupdate.DevVersion, "2.13.0", true, dev},
		{"installed in this run", "2.13.0", "", false, ""},
		{"development build installing", selfupdate.DevVersion, "", false, dev},
	} {
		t.Run(c.name, func(t *testing.T) {
			local := localAppData(t)
			t.Setenv("LOCALAPPDATA", local)
			if c.there {
				installBinary(t)
			}
			asked := 0
			running := Running{Version: c.init, VersionOf: func(path string) string {
				asked++
				if path != filepath.Join(local, "loomux", "bin", "loomux.exe") {
					t.Errorf("asked %s", path)
				}
				return c.installed
			}}
			f, err := Gather(root, t.TempDir(), running, git(""))
			if err != nil {
				t.Fatal(err)
			}
			if f.Version != c.init || f.Installed != c.installed || asked != map[bool]int{true: 1}[c.there] {
				t.Fatalf("version %q, installed %q, asked %d", f.Version, f.Installed, asked)
			}
			p := plan(t, f)
			_, planned := changeOf(p, ".agents/hooks.json")
			if planned != (c.note == "") || (c.note != "" && !slices.Contains(p.Notes, c.note)) {
				t.Errorf("planned %v, notes = %v", planned, p.Notes)
			}
			if !slices.Contains(actions(p), "binary-install") && !c.there {
				t.Errorf("actions = %v", actions(p))
			}
		})
	}
}

// With whitespace in LOCALAPPDATA cmd.exe would split the unquoted path, so
// Antigravity gets no entries, and its hook file is not read; the skills
// still come.
func TestASpacedLocalAppDataWritesNoAntigravityEntries(t *testing.T) {
	root := world(t, map[string]string{".agents/hooks.json": "{not json"})
	f := gather(t, root, "")
	f.LocalAppDataSpaced = true
	p, err := Build(f, DefaultChoice(f, Answers{}), reader(root))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := changeOf(p, ".agents/hooks.json"); ok {
		t.Error("entries planned under a spaced LOCALAPPDATA")
	}
	want := "antigravity: no entries; %LOCALAPPDATA% contains a space or a character cmd.exe reads as a space or as syntax, and cmd.exe would split the unquoted path"
	if !slices.Contains(p.Notes, want) || !hasNote(p, agyTrust) {
		t.Errorf("notes = %v", p.Notes)
	}
	if _, ok := changeOf(p, ".agents/skills/verify-until-green/SKILL.md"); !ok {
		t.Errorf("no skill in %v", paths(p))
	}
}

// A group of the project that runs our command already is named in the plan.
func TestAnotherAntigravityGroupRunningOurCommandIsNamed(t *testing.T) {
	pre := hostfile.Entries(hosts.HostAntigravity, hostfile.Canonical)[0].Command
	root := world(t, map[string]string{".agents/hooks.json": `{"mine":{"PreToolUse":[{"hooks":[{"type":"command","command":` +
		strconv.Quote(pre) + `}]}]}}`})
	p := plan(t, gather(t, root, ""))
	if !slices.Contains(p.Notes, ".agents/hooks.json: the group mine already runs loomux hook session-start; it now fires twice") {
		t.Errorf("notes = %v", p.Notes)
	}
}

func TestAFileThatDoesNotReadStopsThePlan(t *testing.T) {
	root := world(t, map[string]string{})
	f := gather(t, root, "")
	read := func(rel string) ([]byte, bool, error) {
		if rel == ".gitignore" {
			return nil, false, errors.New("denied")
		}
		return reader(root)(rel)
	}
	_, err := Build(f, DefaultChoice(f, Answers{}), read)
	if err == nil || err.Error() != ".gitignore: denied" {
		t.Fatalf("err = %v", err)
	}
}

func TestAnEmptyChangeIsEmpty(t *testing.T) {
	if !(Change{Before: "a", After: "a"}).Empty() || (Change{After: "a"}).Empty() {
		t.Error("Empty compares the texts")
	}
}

func TestTheMergeHookWaitsForConsent(t *testing.T) {
	const declared = "[area]\nscope = \"project/demo\"\n"
	root := world(t, map[string]string{".git/": "", configPath: declared})
	f := gather(t, root, "")
	if f.OnMerge {
		t.Fatal("consent read from a declaration without on_merge")
	}
	p := plan(t, f)
	if slices.Contains(actions(p), "merge-hook") || !hasNote(p, "merge-hook: skipped; no declaration here consents") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
	// A clone brings the consent along, but no area of this machine.
	writeFile(t, root, configPath, declared+"\n[maintenance]\non_merge = true\n")
	f = gather(t, root, "")
	p = plan(t, f)
	if !f.OnMerge || slices.Contains(actions(p), "merge-hook") || !hasNote(p, "the registry of this machine has no area here") {
		t.Errorf("consent: %v, actions = %v, notes = %v", f.OnMerge, actions(p), p.Notes)
	}
	writeFile(t, os.Getenv("LOOMUX_STATE_DIR"), "registry.toml",
		"[[area]]\nscope = \"project/demo\"\npath = \""+filepath.ToSlash(root)+"\"\n")
	if f := gather(t, root, ""); !f.HookWanted() || !slices.Contains(actions(plan(t, f)), "merge-hook") {
		t.Errorf("registered with consent: actions = %v", actions(plan(t, f)))
	}
}

func TestSameDirComparesByIdentity(t *testing.T) {
	root := t.TempDir()
	if !SameDir(root, strings.ToUpper(root[:1])+root[1:]) || !SameDir(root, filepath.ToSlash(root)+"/") {
		t.Error("another spelling of the root is not the root")
	}
	if SameDir(root, filepath.Join(root, "sub")) || SameDir(root, filepath.Dir(root)) {
		t.Error("a directory below or above counts as the root")
	}
}

func TestTheAreaActionNamesTheWikiAreaAddWillUse(t *testing.T) {
	for files, want := range map[string]string{"": "(wiki wiki)", "docs/": "(wiki docs/wiki)"} {
		layout := map[string]string{}
		if files != "" {
			layout[files] = ""
		}
		root := world(t, layout)
		var describe string
		for _, a := range plan(t, gather(t, root, "")).Actions {
			if a.ID == "area-add" {
				describe = a.Describe
			}
		}
		if !strings.HasSuffix(describe, want) {
			t.Errorf("%q: area-add = %q, want %s", files, describe, want)
		}
	}
}

func TestAreaAddOverAnExistingConfigurationBringsNoMergeHook(t *testing.T) {
	// area add keeps a configuration that stands whole, so it writes no
	// consent into one without [area].
	root := world(t, map[string]string{".git/": "", configPath: "[commit]\nlanguage = \"de\"\n"})
	f := gather(t, root, "")
	c := DefaultChoice(f, Answers{})
	c.CommitLanguage = "en"
	p, err := Build(f, c, reader(root))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(actions(p), "area-add") || slices.Contains(actions(p), "merge-hook") ||
		!hasNote(p, "merge-hook: skipped; no declaration here consents") || hasNote(p, "area-add writes [area]") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
}

func TestThePartAreaNamesTheWikiAreaAddWillUse(t *testing.T) {
	root := world(t, map[string]string{"docs/": ""})
	for _, p := range Parts(gather(t, root, "")) {
		if p.ID == "area" && !strings.HasSuffix(p.Label, "default wiki, docs/wiki") {
			t.Errorf("label = %q", p.Label)
		}
	}
}

func TestAnOwnEntryUnderAnOldMatcherGetsABlockForTheToolItLacks(t *testing.T) {
	settings := `{"hooks": {"PreToolUse": [{"matcher": "Write|Edit|NotebookEdit|Bash|PowerShell", "hooks": ` +
		`[{"type": "command", "command": "loomux hook pre-tool-use --host claude"}]}]}}`
	root := world(t, map[string]string{".claude/settings.json": settings})
	p := plan(t, gather(t, root, ""))
	c, _ := changeOf(p, ".claude/settings.json")
	if strings.Count(c.After, "hook pre-tool-use") != 2 || !strings.Contains(c.After, `"matcher": "MultiEdit"`) ||
		!hasNote(p, ".claude/settings.json: PreToolUse: kept an own entry under matcher Write|Edit|NotebookEdit|Bash|PowerShell; added one for MultiEdit") {
		t.Errorf("settings =\n%s\nnotes = %v", c.After, p.Notes)
	}
}

func TestAChangeNamesTheBinaryItsFileCalls(t *testing.T) {
	root := world(t, map[string]string{"go.mod": goMod, ".git/": ""})
	f := gather(t, root, "")
	p := plan(t, f)
	for path, want := range map[string]string{
		mcpPath:                 hostfile.Canonical,
		".claude/settings.json": f.Binary,
		".githooks/pre-commit":  f.Binary,
		"AGENTS.md":             "",
		".gitignore":            "",
	} {
		ch, ok := changeOf(p, path)
		if !ok {
			t.Errorf("%s not planned", path)
			continue
		}
		if ch.Binary != want {
			t.Errorf("%s: Binary = %q, want %q", path, ch.Binary, want)
		}
	}
}
