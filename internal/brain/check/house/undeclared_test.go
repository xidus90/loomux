package house

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// A signpost whose `.loomux/config.toml` holds only policy declares no hub:
// the rule runs with an empty folder, as for a signpost without the file.
func TestHubFolderTakesAPolicyOnlyConfigAsNoDeclaration(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	signpost := config.Area{Scope: "project/p", Path: dir, Signpost: true}
	hub, ok := hubFolder(signpost, config.ArtifactLookup{Primary: t.TempDir(), Fallback: t.TempDir()})
	if hub != "" || !ok {
		t.Fatalf("got %q, %v", hub, ok)
	}
}

// A signpost that still carries only an old manifest has an unusable
// declaration: the rule does not run, as for a broken one.
func TestHubFolderRefusesAnOldManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	signpost := config.Area{Scope: "project/p", Path: dir, Signpost: true}
	hub, ok := hubFolder(signpost, config.ArtifactLookup{Primary: t.TempDir(), Fallback: t.TempDir()})
	if hub != "" || ok {
		t.Fatalf("got %q, %v; want the rule stopped", hub, ok)
	}
}
