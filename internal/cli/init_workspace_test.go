package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hooks"
)

// registryText is the machine registry of the test world, "" when there is
// none.
func registryText(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(os.Getenv("LOOMUX_STATE_DIR"), "registry.toml"))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// guardWrite sends a Write of notes.md under root through the write barrier
// and answers its exit code.
func guardWrite(t *testing.T, root string) int {
	t.Helper()
	target := filepath.ToSlash(filepath.Join(root, "notes.md"))
	payload := `{"tool_name":"Write","tool_input":{"file_path":"` + target + `","content":"x"}}`
	code, _, _ := runWith(payload, "hook", "pre-tool-use", "--host", "claude", "--root", root)
	return code
}

func TestInitWithoutTheBrainRegistersAWorkspace(t *testing.T) {
	root, s := initWorld(t)
	if code := guardWrite(t, root); code != hooks.ExitDenied {
		t.Fatalf("before init: the barrier answered %d, want %d", code, hooks.ExitDenied)
	}
	code, out, errOut := run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	want := "[[area]]\nscope = \"project/demo\"\npath = \"" + filepath.ToSlash(root) + "\"\nworkspace = true\n"
	if got := registryText(t); got != want {
		t.Errorf("registry:\n%s\nwant:\n%s", got, want)
	}
	if len(s.actions) != 0 {
		t.Errorf("subcommands run: %v", s.actions)
	}
	if !strings.Contains(out, "written: workspace-add") {
		t.Errorf("report:\n%s", out)
	}
	if code := guardWrite(t, root); code != 0 {
		t.Errorf("after init: the barrier answered %d, want 0", code)
	}
	registry := registryText(t)
	code, out, errOut = run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 0 || !strings.HasPrefix(out, "nothing to change\n") {
		t.Errorf("second run: code %d: %s\n%s", code, errOut, out)
	}
	if registryText(t) != registry {
		t.Errorf("second run changed the registry:\n%s", registryText(t))
	}
}

func TestInitWithoutTheBrainKeepsAnEntryAtItsRoot(t *testing.T) {
	root, _ := initWorld(t)
	// The hand-made entry of a project set up before init could: no wiki,
	// workspace = true, a scope init would not pick.
	text := "[[area]]\nscope = \"project/own\"\npath = \"" + filepath.ToSlash(root) + "\"\nworkspace = true\n"
	writeAt(t, filepath.Join(os.Getenv("LOOMUX_STATE_DIR"), "registry.toml"), text)
	code, out, errOut := run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if registryText(t) != text || strings.Contains(out, "workspace-add") {
		t.Errorf("registry:\n%s\nstdout:\n%s", registryText(t), out)
	}
	const note = "  workspace: skipped; the registry has an area at this root already\n"
	if strings.Count(out, note) != 1 {
		t.Errorf("the note on the registered root is not named once:\n%s", out)
	}
}

func TestInitWithoutTheBrainStopsAtATakenScope(t *testing.T) {
	root, _ := initWorld(t)
	other := t.TempDir()
	registerArea(t, [2]string{"project/demo", other})
	registry := registryText(t)
	code, out, errOut := run("init", "--root", root, "--yes", "--brain=none", "--graph=none")
	if code != 1 || !strings.Contains(errOut, `scope "project/demo" is already registered`) {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if registryText(t) != registry || there(root, ".loomux/state/installed.toml") {
		t.Errorf("registry:\n%s", registryText(t))
	}
	// The refusal comes before any file: hook entries without a registered
	// tree would arm a barrier that refuses every write in the project.
	for _, rel := range []string{".claude/settings.json", ".loomux/config.toml", ".gitignore"} {
		if there(root, rel) {
			t.Errorf("%s was written before the refusal", rel)
		}
	}
}

func TestInitDryRunWithoutTheBrainNamesTheWorkspace(t *testing.T) {
	root, _ := initWorld(t)
	code, out, errOut := run("init", "--root", root, "--dry-run", "--brain=none")
	if code != 0 {
		t.Fatalf("code %d: %s\n%s", code, errOut, out)
	}
	if !strings.Contains(out, "  workspace-add: register project/demo as a workspace without wiki in the registry\n") {
		t.Errorf("stdout:\n%s", out)
	}
	if registryText(t) != "" {
		t.Errorf("dry run wrote the registry:\n%s", registryText(t))
	}
}
