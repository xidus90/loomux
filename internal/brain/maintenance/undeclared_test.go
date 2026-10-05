package maintenance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// A `.loomux/config.toml` that holds only policy declares no area: Manifests
// leaves it out instead of stopping at it.
func TestManifestsLeavesOutAPolicyOnlyConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	areas := []config.Area{{Scope: "project/p", Path: dir}}
	got, err := Manifests(areas, config.ArtifactLookup{Primary: t.TempDir()})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

// A workspace that declares no [area] is left out like any undeclared area, so
// reconcile and the catch-up scan nothing of it and land no case in it.
func TestManifestsLeavesOutAWorkspaceThatDeclaresNoArea(t *testing.T) {
	code, declared := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(declared, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(declared, ".loomux", "config.toml"), []byte("[area]\nscope = \"project/declared\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	areas := []config.Area{
		{Scope: "project/code", Path: code, Workspace: true},
		{Scope: "project/declared", Path: declared, Workspace: true},
	}
	got, err := Manifests(areas, config.ArtifactLookup{Primary: t.TempDir()})
	if err != nil || len(got) != 1 || got["project/declared"] == nil {
		t.Fatalf("got %v, %v", got, err)
	}
}
