package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// A state file is read and written in the one state directory, whether it
// lies there or not; a copy elsewhere is never the answer.
func TestArtifactLookupAnswersTheStateDirectoryAlone(t *testing.T) {
	primary := t.TempDir()
	lookup := config.ArtifactLookup{Primary: primary}
	relative := filepath.Join("maintenance", "last-run.txt")
	want := filepath.Join(primary, relative)
	if got := lookup.Resolve(relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
	if got := lookup.WritePath(relative); got != want {
		t.Fatalf("WritePath = %q, want %q", got, want)
	}
}

// LOOMUX_LEGACY_BRAIN_DIR names nothing any more: the lookup takes the state
// directory and asks no second place, even where only that one holds the file.
func TestNewArtifactLookupTakesTheStateDirectoryAlone(t *testing.T) {
	primary, old := t.TempDir(), t.TempDir()
	t.Setenv(config.StateDirEnv, primary)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", old)
	relative := filepath.Join("maintenance", "last-run.txt")
	if err := os.MkdirAll(filepath.Join(old, "maintenance"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, relative), []byte("2999-01-01T00:00:00+00:00\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lookup := config.NewArtifactLookup()
	if lookup.Primary != primary {
		t.Fatalf("NewArtifactLookup = %+v, want the state directory %q", lookup, primary)
	}
	if got, want := lookup.Resolve(relative), filepath.Join(primary, relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}
