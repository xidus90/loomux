package run

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// An area whose `.loomux/config.toml` holds only policy declares nothing: no
// manifest and no finding, as for an area without the file.
func TestAreaManifestTakesAPolicyOnlyConfigAsNoDeclaration(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	area := config.Area{Scope: "project/p", Path: dir}
	manifest, findings := areaManifest(area, config.ArtifactLookup{Primary: t.TempDir()})
	if manifest != nil || len(findings) != 0 {
		t.Fatalf("got %+v, %v", manifest, findings)
	}
}
