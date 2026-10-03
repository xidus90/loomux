package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// A `.loomux/config.toml` that holds only policy declares no area: it has no
// inbox and counts as manual_cloud, the same as an area without the file.
func TestAreasTakesAPolicyOnlyConfigAsNoDeclaration(t *testing.T) {
	w := newWorld(t)
	w.area("project/p", "[verify]\n", false)
	registered, err := config.ReadRegistry(w.state)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := Areas(registered, w.state)
	if err != nil || len(entries) != 1 || entries[0].Mode != "manual_cloud" || entries[0].Manifest != nil {
		t.Fatalf("got %+v, %v", entries, err)
	}
}

// An area that still carries only an old manifest stops the run with the
// hint: taken as undeclared, a local_only area would run as manual_cloud.
func TestAreasRefusesAnAreaWithOnlyAnOldManifest(t *testing.T) {
	w := newWorld(t)
	dir := w.area("project/p", "", false)
	if err := os.WriteFile(filepath.Join(dir, ".brain.toml"), []byte("[area]\nscope = \"project/p\"\n\n[privacy]\nmode = \"local_only\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registered, err := config.ReadRegistry(w.state)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := Areas(registered, w.state)
	if err == nil || !strings.Contains(err.Error(), "an old manifest lies there") {
		t.Fatalf("got %+v, %v; want the old-manifest hint", entries, err)
	}
}
