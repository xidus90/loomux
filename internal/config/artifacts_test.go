package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestArtifactLookupPrefersPrimary(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-loomux", "_identities.tsv")
	mustWriteArtifact(t, filepath.Join(primary, relative), "new")
	mustWriteArtifact(t, filepath.Join(fallback, relative), "old")

	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve(relative), filepath.Join(primary, relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestArtifactLookupFallsBackWhenPrimaryHasNothing(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-space", "index.md")
	mustWriteArtifact(t, filepath.Join(fallback, relative), "old")

	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve(relative), filepath.Join(fallback, relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

// Missing on both sides, the new place wins: the error the caller then shows
// names the path the file is meant to live at.
func TestArtifactLookupNamesPrimaryWhenNeitherHasIt(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve("missing.tsv"), filepath.Join(primary, "missing.tsv"); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

// A directory counts like a file: `areas/<scope>/` is the place an area
// occupies, and an empty directory there is a statement.
func TestArtifactLookupAcceptsADirectory(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-loomux")
	if err := os.MkdirAll(filepath.Join(primary, relative), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.Resolve(relative), filepath.Join(primary, relative); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

func TestArtifactLookupWritesOnlyToPrimary(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	mustWriteArtifact(t, filepath.Join(fallback, "stamp.txt"), "old")
	lookup := config.ArtifactLookup{Primary: primary, Fallback: fallback}
	if got, want := lookup.WritePath("stamp.txt"), filepath.Join(primary, "stamp.txt"); got != want {
		t.Fatalf("WritePath = %q, want %q", got, want)
	}
}

// Without a legacy directory there is nothing to fall back to -- not even the
// working directory, which an empty fallback joined onto `x` would name.
func TestArtifactLookupWithoutFallback(t *testing.T) {
	primary := t.TempDir()
	here := t.TempDir()
	if err := os.WriteFile(filepath.Join(here, "x"), []byte("not ours"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(here)
	lookup := config.ArtifactLookup{Primary: primary}
	if got, want := lookup.Resolve("x"), filepath.Join(primary, "x"); got != want {
		t.Fatalf("Resolve = %q, want %q", got, want)
	}
}

// NewArtifactLookup takes the two places the environment names.
func TestNewArtifactLookupTakesBothDirectories(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	t.Setenv(config.StateDirEnv, primary)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", fallback)

	lookup := config.NewArtifactLookup()
	if lookup.Primary != primary || lookup.Fallback != fallback {
		t.Fatalf("NewArtifactLookup = %+v, want {%q %q}", lookup, primary, fallback)
	}
}

// A writable area keeps its artefacts in its own tree; only a read-only area
// is looked up new first, legacy second.
func TestResolvedAreaDirAnswersTheAreaPathWhenItIsWritable(t *testing.T) {
	area := config.Area{Scope: "project/a", Path: filepath.Join("some", "path")}
	if got := config.ResolvedAreaDir(area, t.TempDir(), ""); got != area.Path {
		t.Fatalf("ResolvedAreaDir = %q, want %q", got, area.Path)
	}
}

func TestResolvedAreaDirPrefersTheNewStateDir(t *testing.T) {
	primary, fallback := t.TempDir(), t.TempDir()
	relative := filepath.Join("areas", "project-a")
	if err := os.MkdirAll(filepath.Join(primary, relative), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(fallback, relative), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	area := config.Area{Scope: "project/a", Path: filepath.Join("some", "path"), ReadOnly: true}
	if got, want := config.ResolvedAreaDir(area, primary, fallback), filepath.Join(primary, relative); got != want {
		t.Fatalf("ResolvedAreaDir = %q, want %q", got, want)
	}
}

func mustWriteArtifact(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
