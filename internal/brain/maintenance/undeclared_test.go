package maintenance

import (
	"os"
	"path/filepath"
	"strings"
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
	got, err := Manifests(areas, config.ArtifactLookup{Primary: t.TempDir(), Fallback: t.TempDir()})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

// An area that still carries only an old manifest stops Manifests with the
// hint instead of being left out.
func TestManifestsRefusesAnAreaWithOnlyAnOldManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	areas := []config.Area{{Scope: "project/p", Path: dir}}
	got, err := Manifests(areas, config.ArtifactLookup{Primary: t.TempDir(), Fallback: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "an old manifest lies there") {
		t.Fatalf("got %v, %v; want the old-manifest hint", got, err)
	}
}
