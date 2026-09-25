package setup

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

func TestGatherReadsHostsHooksPathAndTheUserScope(t *testing.T) {
	root := world(t, map[string]string{".claude/": "", ".agents/skills/": "", ".git/": "", "go.mod": goMod})
	home := t.TempDir()
	writeFile(t, home, ".claude.json", `{"mcpServers": {"loomux": {"command": "loomux"}}}`)
	var asked []string
	run := func(dir string, argv ...string) (string, error) {
		if slices.Contains(argv, "config") {
			asked = append([]string{dir}, argv...)
		}
		return ".githooks\n", nil
	}
	f, err := Gather(root, home, tested, run)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.Hosts, []hosts.Host{hosts.HostClaude, hosts.HostAntigravity}) {
		t.Errorf("hosts = %v", f.Hosts)
	}
	if f.HooksPath != ".githooks" || !slices.Equal(asked, []string{root, "git", "config", "--type=path", "--get", "core.hooksPath"}) {
		t.Errorf("hooks path = %q, asked %v", f.HooksPath, asked)
	}
	if !f.UserMCP || f.Registered || f.Checkout || !f.Detect.HasGit || f.Config != "" {
		t.Errorf("facts = %+v", f)
	}
	if !slices.Contains(f.Detect.Stacks, "go") {
		t.Errorf("stacks = %v", f.Detect.Stacks)
	}
}

// Other tools keep an .agents/ too, so Antigravity is taken only from its
// hook file, its skills or its GEMINI.md; a bare directory, or one with
// something else in it, is no Antigravity.
func TestAntigravityIsTakenOnlyFromItsOwnFiles(t *testing.T) {
	claude := []hosts.Host{hosts.HostClaude}
	agy := []hosts.Host{hosts.HostAntigravity}
	for _, c := range []struct {
		files map[string]string
		want  []hosts.Host
	}{
		{map[string]string{".agents/hooks.json": "{}"}, agy},
		{map[string]string{".agents/skills/": ""}, agy},
		{map[string]string{"GEMINI.md": "# demo\n"}, agy},
		{map[string]string{".agents/": ""}, claude},
		{map[string]string{".agents/rules.md": "x", ".agents/hooks.json/": "", ".agents/skills": "x", "GEMINI.md/": ""}, claude},
	} {
		if got := gather(t, world(t, c.files), "").Hosts; !slices.Equal(got, c.want) {
			t.Errorf("%v: hosts = %v, want %v", c.files, got, c.want)
		}
	}
}

func TestTheUserScopeCountsOnlyAServerNamedLoomux(t *testing.T) {
	root := world(t, map[string]string{})
	for _, text := range []string{"", "{not json", `{"mcpServers": {"other": {}}}`} {
		home := t.TempDir()
		if text != "" {
			writeFile(t, home, ".claude.json", text)
		}
		if f, _ := Gather(root, home, tested, git("")); f.UserMCP {
			t.Errorf("%q counts as a loomux server", text)
		}
	}
}

// cmd.exe splits the unquoted %LOCALAPPDATA% path of Antigravity's entries
// at any whitespace and at its own syntax, so Gather says whether there is
// some.
func TestGatherSeesWhitespaceInLocalAppData(t *testing.T) {
	root := world(t, map[string]string{})
	if f := gather(t, root, ""); f.LocalAppDataSpaced {
		t.Errorf("%q counts as spaced", os.Getenv("LOCALAPPDATA"))
	}
	for _, local := range []string{`C:\Users\Jane Doe\AppData\Local`, "C:\\x\ty", `C:\Users\R&D\AppData\Local`, `C:\Users\a^b`} {
		t.Setenv("LOCALAPPDATA", local)
		if f := gather(t, root, ""); !f.LocalAppDataSpaced {
			t.Errorf("%q does not count as spaced", local)
		}
	}
}

func TestGatherTakesTheBinaryTheEntriesCall(t *testing.T) {
	files := fixture(t)
	root := world(t, map[string]string{".claude/settings.json": files[".claude/settings.json"]})
	if f := gather(t, root, ""); f.Checkout || f.Binary != hostfile.Checkout {
		t.Errorf("facts = %+v, want the checkout binary from the entries", f)
	}
}

func TestACheckoutIsTheModuleLineExactly(t *testing.T) {
	for text, want := range map[string]bool{
		checkoutGoMod:                              true,
		"module \"github.com/xidus90/loomux\"\n":   true,
		"module github.com/xidus90/loomux/tools\n": false,
		"// module github.com/xidus90/loomux\n":    false,
	} {
		if got := declaresModule(text, checkoutModule); got != want {
			t.Errorf("%q: %v, want %v", text, got, want)
		}
	}
}

func TestGatherFindsTheRootInTheRegistry(t *testing.T) {
	root := world(t, map[string]string{})
	state := os.Getenv("LOOMUX_STATE_DIR")
	registry := "[[area]]\nscope = \"project/other\"\npath = \"" + filepath.ToSlash(t.TempDir()) + "\"\n\n" +
		"[[area]]\nscope = \"project/demo\"\npath = \"" + strings.ToUpper(filepath.ToSlash(root)[:1]) + filepath.ToSlash(root)[1:] + "\"\n"
	writeFile(t, state, "registry.toml", registry)
	if f := gather(t, root, ""); !f.Registered {
		t.Errorf("the root is not found in\n%s", registry)
	}
	writeFile(t, state, "registry.toml", "[[area]]\nscope = \"project/other\"\npath = \""+filepath.ToSlash(t.TempDir())+"\"\n")
	if f := gather(t, root, ""); f.Registered {
		t.Errorf("another area counts as this root")
	}
	writeFile(t, state, "registry.toml", "not toml [")
	if _, err := Gather(root, t.TempDir(), tested, git("")); err == nil || !strings.Contains(err.Error(), "registry.toml") {
		t.Errorf("err = %v", err)
	}
}

func TestGatherStopsOnWhatItCannotRead(t *testing.T) {
	root := world(t, map[string]string{})
	if _, err := Gather(root, t.TempDir(), tested, func(string, ...string) (string, error) {
		return "", errors.New("no git")
	}); err == nil || !strings.Contains(err.Error(), "core.hooksPath") {
		t.Errorf("err = %v", err)
	}
	// A directory where a file should be reads as an error, not as missing.
	// Only the files init merges into are unmergeable.
	for rel, merged := range map[string]bool{configPath: true, "go.mod": false, ".claude/settings.json": true} {
		root := world(t, map[string]string{rel + "/": ""})
		if _, err := Gather(root, t.TempDir(), tested, git("")); err == nil || !strings.Contains(err.Error(), rel) || Unmergeable(err) != merged {
			t.Errorf("%s: err = %v", rel, err)
		}
	}
}

func TestGatherAsksGitForItsHookDirectory(t *testing.T) {
	root := world(t, map[string]string{".git/": "", ".git/hooks/pre-commit.sample": ""})
	answer := func(dir string, err error) detect.Runner {
		return func(_ string, argv ...string) (string, error) {
			if slices.Contains(argv, "rev-parse") {
				return dir, err
			}
			return "", nil
		}
	}
	// An older git answers relative to where it ran.
	f, err := Gather(root, t.TempDir(), tested, answer(".git/hooks\n", nil))
	if err != nil || f.GitHooksDir != filepath.Join(root, ".git", "hooks") || f.GitHooksLive {
		t.Errorf("dir = %q, live = %v, err = %v", f.GitHooksDir, f.GitHooksLive, err)
	}
	for name, run := range map[string]detect.Runner{
		"fails":      answer("", errors.New("boom")),
		"names none": answer("\n", nil),
		"unreadable": answer(root+"/bad\x00dir", nil),
	} {
		if _, err := Gather(root, t.TempDir(), tested, run); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	// With core.hooksPath set git's own directory is not asked for.
	if f := gather(t, root, ".githooks"); f.GitHooksDir != "" {
		t.Errorf("dir = %q", f.GitHooksDir)
	}
}

func TestGatherReadsTheConfiguration(t *testing.T) {
	root := world(t, map[string]string{configPath: "[area]\nscope = \"project/demo\"\n"})
	f := gather(t, root, "")
	if !hasArea(f.Config) || hasArea("not toml [") || hasArea("") {
		t.Errorf("config = %q", f.Config)
	}
}

func TestATildeInCoreHooksPathIsTheHome(t *testing.T) {
	root := world(t, map[string]string{})
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	global := filepath.Join(home, "gitconfig")
	writeFile(t, home, "gitconfig", "")
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	run(t, root, "init", "-q")
	run(t, root, "config", "core.hooksPath", "~/.githooks")
	writeFile(t, home, ".githooks/post-merge", "#!/bin/sh\n# loomux post-merge hook\n")
	f, err := Gather(root, home, tested, realGit)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(f.HooksPath, filepath.Join(home, ".githooks")) || !f.MergeHook {
		t.Errorf("hooks path = %q, merge hook = %v", f.HooksPath, f.MergeHook)
	}
	p := plan(t, f)
	for _, path := range paths(p) {
		if strings.HasPrefix(path, "~") {
			t.Errorf("planned under the root's ~: %s", path)
		}
	}
	if !hasNote(p, "lies outside the project") || slices.Contains(actions(p), "merge-hook") {
		t.Errorf("actions = %v, notes = %v", actions(p), p.Notes)
	}
}

func TestGatherFindsTheRegisteredAreaThroughALink(t *testing.T) {
	real := world(t, map[string]string{})
	root := filepath.Join(t.TempDir(), "via")
	link(t, real, root)
	state := os.Getenv("LOOMUX_STATE_DIR")
	writeFile(t, state, "registry.toml", "[[area]]\nscope = \"project/demo\"\npath = \""+filepath.ToSlash(real)+"\"\n")
	if f := gather(t, root, ""); !f.Registered {
		t.Error("the area is not found through the link")
	}
	// A registered path that is gone is compared by its spelling.
	gone := filepath.Join(t.TempDir(), "gone")
	writeFile(t, state, "registry.toml", "[[area]]\nscope = \"project/demo\"\npath = \""+filepath.ToSlash(gone)+"\"\n")
	if f := gather(t, gone, ""); !f.Registered {
		t.Error("a gone root is not found by its spelling")
	}
}
