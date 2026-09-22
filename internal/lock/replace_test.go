package lock_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

func TestReplaceTextWritesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.toml")
	if err := lock.ReplaceText(path, "scope = \"a\"\n"); err != nil {
		t.Fatalf("ReplaceText: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "scope = \"a\"\n" {
		t.Fatalf("content = %q", got)
	}
}

// The run must leave no .tmp file behind: a reader listing the directory
// would otherwise see two registries.
func TestReplaceTextLeavesNoTemporary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.toml")
	if err := lock.ReplaceText(path, "x\n"); err != nil {
		t.Fatalf("ReplaceText: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
}

// The case the consumers actually have: a registry that is already there.
// Windows only overwrites because os.Rename asks for MOVEFILE_REPLACE_EXISTING.
func TestReplaceTextOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.toml")
	if err := lock.ReplaceText(path, "first\n"); err != nil {
		t.Fatalf("ReplaceText first: %v", err)
	}
	if err := lock.ReplaceText(path, "second\n"); err != nil {
		t.Fatalf("ReplaceText second: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "second\n" {
		t.Fatalf("content = %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
}

// CRLF stays CRLF and LF stays LF: the registry is compared byte for byte.
func TestReplaceTextDoesNotTranslateNewlines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.txt")
	if err := lock.ReplaceText(path, "a\r\nb\n"); err != nil {
		t.Fatalf("ReplaceText: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "a\r\nb\n" {
		t.Fatalf("content = %q", got)
	}
}

// A target directory that does not exist is an error and no silent creation:
// whoever writes here has already established the place.
func TestReplaceTextRefusesMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "registry.toml")
	if err := lock.ReplaceText(path, "x"); err == nil {
		t.Fatal("ReplaceText: want error for a missing directory")
	}
}

// exhaustedFloor is how long five attempts with a 20 ms pause take at the
// least: four pauses, the last attempt without one. Written out rather than
// derived from the constants, because a bound computed from them would shrink
// along with a mistake there and hold nothing.
const exhaustedFloor = 80 * time.Millisecond

// A swap that never works gives up and takes its temporary file with it: a
// directory in the target's place refuses the rename on every platform, so
// this is the retry loop's exhausted arm. The elapsed time is the only
// evidence that the loop really ran -- without a floor, a single attempt or a
// pause of zero would pass here just as well, and the Windows window this
// function exists for would be unguarded.
func TestReplaceTextGivesUpAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "occupied")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	started := time.Now()
	if err := lock.ReplaceText(path, "x"); err == nil {
		t.Fatal("ReplaceText: want error for a directory in the way")
	}
	// A floor and never a ceiling: a loaded machine may take much longer, and
	// a test that fails for slowness would be worse than no test.
	if elapsed := time.Since(started); elapsed < exhaustedFloor {
		t.Fatalf("elapsed = %s, want at least %s -- the retries did not run", elapsed, exhaustedFloor)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1 -- the temporary file was left behind", len(entries))
	}
}
