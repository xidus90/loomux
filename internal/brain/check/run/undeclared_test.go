package run

import (
	"os"
	"path/filepath"
	"strings"
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
	manifest, findings := areaManifest(area, config.ArtifactLookup{Primary: t.TempDir(), Fallback: t.TempDir()})
	if manifest != nil || len(findings) != 0 {
		t.Fatalf("got %+v, %v", manifest, findings)
	}
}

// An area that still carries only an old manifest is reported as
// `manifest-unreadable` with the hint, not taken as undeclared.
func TestAreaManifestReportsAnOldManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	area := config.Area{Scope: "project/p", Path: dir}
	manifest, findings := areaManifest(area, config.ArtifactLookup{Primary: t.TempDir(), Fallback: t.TempDir()})
	if manifest != nil || len(findings) != 1 || !strings.Contains(findings[0].Message, "an old manifest lies there") {
		t.Fatalf("got %+v, %+v; want one finding with the hint", manifest, findings)
	}
}
