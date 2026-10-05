package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// An area whose `.loomux/config.toml` holds only policy declares nothing: the
// sweep runs with the defaults, as for an area without the file.
func TestSweepContextTakesAPolicyOnlyConfigAsNoDeclaration(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	area := config.Area{Scope: "project/p", Path: dir, WikiPath: dir}
	ctx, err := sweepContext(area, []config.Area{area}, map[string]bool{}, config.ArtifactLookup{Primary: t.TempDir()})
	if err != nil || ctx.UntouchedDays != config.DefaultUntouchedDays || len(ctx.DeclaredTypes) != 0 {
		t.Fatalf("got %+v, %v", ctx, err)
	}
}
