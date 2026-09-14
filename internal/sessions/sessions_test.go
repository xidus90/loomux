package sessions

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func state(t *testing.T, root, id string, age time.Duration) string {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, id+".json")
	if err := os.WriteFile(path, []byte(`{"blocks":0,"snapshots":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOthersCountsEveryOtherLivingSession(t *testing.T) {
	root := t.TempDir()
	state(t, root, "mine", 0)
	state(t, root, "yours", time.Minute)

	got, err := Others(root, "mine", 12*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("Others = %d, want 1", got)
	}
}

// Nobody deletes these files today, so an old one must not hold a junction
// hostage for ever.
func TestOthersIgnoresAStaleFile(t *testing.T) {
	root := t.TempDir()
	state(t, root, "mine", 0)
	state(t, root, "ancient", 48*time.Hour)

	got, err := Others(root, "mine", 12*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("Others = %d, want 0", got)
	}
}

func TestOthersOfADirectoryWithoutStateIsZero(t *testing.T) {
	got, err := Others(t.TempDir(), "mine", 12*time.Hour)
	if err != nil || got != 0 {
		t.Fatalf("Others = %d, %v; want 0 and nil", got, err)
	}
}

// Only the session files are counted. The Python hooks are not the only writer
// in a project, and a directory or a file with another suffix beside them is
// nobody's session.
func TestOthersCountsOnlySessionFiles(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	if err := os.MkdirAll(filepath.Join(dir, "sub.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := Others(root, "mine", 12*time.Hour)
	if err != nil || got != 0 {
		t.Fatalf("Others = %d, %v; want 0 and nil", got, err)
	}
}

// A count that could not be taken is not a count of zero: read as zero, the
// caller would take a junction out from under whoever is still there. Absence
// is the one error that reads as zero, and this is not absence -- `?` is
// illegal in a Windows filename, so os.ReadDir answers with a bad syntax and
// os.IsNotExist is false for it. Measured on 2026-09-07.
func TestOthersReportsADirectoryItCannotRead(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the illegal-name trick this leans on is a Windows rule")
	}
	if _, err := Others(filepath.Join(t.TempDir(), "bad?name"), "mine", 12*time.Hour); err == nil {
		t.Fatal("Others = nil error, want one")
	}
}

// The id comes from outside, so it may not decide which file is read -- the
// same reasoning as ultraloom/hooks/state.py's own path builder.
func TestForgetRemovesOnlyThisSessionsFile(t *testing.T) {
	root := t.TempDir()
	mine := state(t, root, "mine", 0)
	yours := state(t, root, "yours", 0)

	if err := Forget(root, "mine"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); !os.IsNotExist(err) {
		t.Fatalf("the file survived: %v", err)
	}
	if _, err := os.Stat(yours); err != nil {
		t.Fatalf("somebody else's file was removed: %v", err)
	}
}

func TestForgetAnUnknownSessionIsNotAnError(t *testing.T) {
	if err := Forget(t.TempDir(), "nobody"); err != nil {
		t.Fatalf("Forget = %v, want nil", err)
	}
}

func TestASeparatorInTheIdCannotClimbOut(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside.json")
	if err := os.WriteFile(outside, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Forget(root, "../outside"); err != nil {
		t.Fatalf("Forget = %v, want nil", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("a file outside the state directory was removed: %v", err)
	}
}

// state.py keeps `char.isalnum()`, and that answers true for a letter outside
// ASCII: measured on 2026-09-07 with CPython 3.13, `'ä'.isalnum()` is
// True. So the Python side writes `<letter>.json` for such an id, and a Go
// rule that dropped it would look for `unnamed.json` and find nothing.
func TestAnIdOutsideAsciiIsTheSameNameOnBothSides(t *testing.T) {
	root := t.TempDir()
	mine := state(t, root, "ä٣", 0)

	if err := Forget(root, "ä٣"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); !os.IsNotExist(err) {
		t.Fatalf("the file survived: %v", err)
	}
}

// An id with nothing keepable in it lands on one shared name rather than on
// the state directory itself -- state.py's `or "unnamed"`, and the reason it
// is there.
func TestAnIdWithNothingKeepableInItBecomesUnnamed(t *testing.T) {
	root := t.TempDir()
	mine := state(t, root, "unnamed", 0)

	if err := Forget(root, "!!!"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mine); !os.IsNotExist(err) {
		t.Fatalf("the file survived: %v", err)
	}
}

// A file that is there and will not go is not the same as one that was never
// written, and reading it as "done" would leave the junction standing with
// nobody to take it out. A directory with something in it is the portable way
// to make os.Remove refuse: measured on 2026-09-07 it answers "directory not
// empty", for which os.IsNotExist is false.
func TestForgetReportsAFileItCannotRemove(t *testing.T) {
	root := t.TempDir()
	busy := filepath.Join(root, filepath.FromSlash(StateDir), "mine.json")
	if err := os.MkdirAll(busy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(busy, "inside"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Forget(root, "mine"); err == nil {
		t.Fatal("Forget = nil, want an error")
	}
}
