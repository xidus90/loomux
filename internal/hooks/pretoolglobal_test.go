package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A rule of the machine-wide file binds a project that has no configuration.
func TestAGlobalPathRuleRefusesAWriteInAnyProject(t *testing.T) {
	root, state := world(t, "")
	writeGlobal(t, state, "[[policy.paths.rules]]\nmatch = \"secrets/**\"\nreason = \"machine-wide secrets\"\n")
	code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "secrets", "a.txt")))
	if code != 2 || !strings.Contains(errOut, "machine-wide secrets") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go"))); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestAGlobalCommandRuleRefusesAShellCommand(t *testing.T) {
	root, state := world(t, "")
	writeGlobal(t, state, "[[policy.commands.rules]]\nregex = '(^|\\s)pip\\s+install'\nreason = \"machine-wide: uv, never pip\"\n")
	code, _, errOut := call(t, root, state, `{"tool_name":"Bash","tool_input":{"command":"pip install requests"}}`)
	if code != 2 || !strings.Contains(errOut, "machine-wide: uv, never pip") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// A global file the reader refuses refuses every call, but a tool that
// reaches the hook only for the graph hint runs through.
func TestABrokenGlobalPolicyRefusesAndNamesTheFile(t *testing.T) {
	root, state := world(t, "")
	writeGlobal(t, state, "[guard]\nmode = \"strict\"\n")
	code, _, errOut := call(t, root, state, payload("Write", "file_path", filepath.Join(root, "main.go")))
	if code != 2 || !strings.Contains(errOut, "cannot read its policy") || !strings.Contains(errOut, filepath.Join(state, "config.toml")) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if code, _, errOut := call(t, root, state, payload("Grep", "pattern", "x")); code != 0 {
		t.Fatalf("a hint-only tool must pass: code %d, err %q", code, errOut)
	}
}

// writeGlobal puts text into the machine-wide file of the state directory.
func writeGlobal(t *testing.T, state, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(state, "config.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
