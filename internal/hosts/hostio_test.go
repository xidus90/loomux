package hosts_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

func TestParseHost(t *testing.T) {
	for _, name := range []string{"claude", "antigravity", "codex"} {
		if _, err := hosts.ParseHost(name); err != nil {
			t.Errorf("ParseHost(%q): %v", name, err)
		}
	}
	// An unknown host is refused here rather than guessed at. The flag is
	// written by loomux init, so a value nobody knows means the two have drifted
	// apart, and answering a hook in the wrong shape is worse than refusing.
	if _, err := hosts.ParseHost("gemini-cli"); err == nil {
		t.Error("an unknown host is refused")
	}
	if _, err := hosts.ParseHost(""); err == nil {
		t.Error("an empty host is refused")
	}
}

// An Antigravity hook runs with its working directory set to the directory
// holding hooks.json -- `.agents/`, not the project root -- so the root has to
// be found by walking up. Measured on 2026-09-10 against agy 1.1.24 and
// documented in docs/.superpowers/specs/2026-09-10-antigravity-hook-messung.md,
// finding 2.
func TestFindRootWalksUpToTheConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, ".agents")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := hosts.FindRoot(deep)
	if err != nil {
		t.Fatal(err)
	}

	// EvalSymlinks on both sides: a temp directory on macOS is reached through
	// /var, which is a link to /private/var, and the comparison would fail on
	// the spelling rather than on the answer.
	want, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Fatalf("FindRoot = %q, want %q", gotResolved, want)
	}
}

// The caller passes ".", so a relative start has to be resolved before the
// walk: `filepath.Dir` would end it at "." instead of at the volume root.
func TestFindRootResolvesARelativeStart(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, ".agents")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	// os.Chdir and not t.Chdir: the latter arrived in Go 1.24 and this module
	// is on 1.22. Restored by defer, and no test here runs in parallel.
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(deep); err != nil {
		t.Fatal(err)
	}

	got, err := hosts.FindRoot(".")
	if err != nil {
		t.Fatal(err)
	}

	want, _ := filepath.EvalSymlinks(root)
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Fatalf("FindRoot = %q, want %q", gotResolved, want)
	}
}

func TestFindRootWithoutAConfig(t *testing.T) {
	_, err := hosts.FindRoot(t.TempDir())
	if !errors.Is(err, hosts.ErrNoRoot) {
		t.Fatalf("expected ErrNoRoot, got %v", err)
	}
}
