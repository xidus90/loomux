package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A broken machine-wide rule keeps every other global key from being written,
// as a broken [model] does: the guard would refuse every call on the file.
func TestConfigGlobalRefusesAWriteBesideABrokenPolicyRule(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	if err := os.WriteFile(filepath.Join(state, "config.toml"), []byte("[[policy.commands.rules]]\nregex = \"(\"\nreason = \"x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := runConfig(t, "", "set", "model.enabled", "true", "--yes", "--global")
	if code != 1 || !strings.Contains(errOut, "does not compile") {
		t.Fatalf("%d %s", code, errOut)
	}
}
