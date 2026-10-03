package index

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

// The entry `init --brain=none` writes: a workspace without wiki whose
// project declares no [area]. Reindex skips it for the missing declaration
// and gives it no collection, while an area beside it is still indexed.
func TestReindexGivesAWorkspaceWithoutDeclarationNoCollection(t *testing.T) {
	tmp := t.TempDir()
	stateDir := filepath.Join(tmp, "state")
	t.Setenv("XDG_CONFIG_HOME", tmp)

	workspace := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(filepath.Join(workspace, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".loomux", "config.toml"), []byte("[modules]\nbrain = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "notes.md"), []byte("# Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	validAreaDir := filepath.Join(tmp, "valid_area")
	setupTestArea(t, validAreaDir, "[area]\nscope = \"valid\"\n")
	if err := os.WriteFile(filepath.Join(validAreaDir, "valid.md"), []byte("# Valid\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registryContent := "[[area]]\nscope = \"project/ws\"\npath = \"" + filepath.ToSlash(workspace) + "\"\nworkspace = true\n\n" +
		"[[area]]\nscope = \"valid\"\npath = \"" + filepath.ToSlash(validAreaDir) + "\"\n"
	regPath := writeTestRegistry(t, stateDir, registryContent)

	port := search.NewFakePort()
	var stderr bytes.Buffer
	code, err := ReindexWithOutput(regPath, stateDir, port, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code %d, err %v: %s", code, err, stderr.String())
	}
	out := stderr.String()
	if !strings.Contains(out, "skipping project/ws: ") || !strings.Contains(out, "updated qmd collections: valid\n") {
		t.Errorf("stderr:\n%s", out)
	}
	qmd, err := os.ReadFile(QmdConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if name := search.CollectionName("project/ws"); strings.Contains(string(qmd), name) {
		t.Errorf("a collection %s for the workspace:\n%s", name, qmd)
	}
	if len(port.Refreshed) != 1 || len(port.Refreshed[0]) != 1 || port.Refreshed[0][0] != "valid" {
		t.Errorf("refreshed %#v", port.Refreshed)
	}
}
