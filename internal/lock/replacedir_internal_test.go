package lock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// failingOn refuses the nth rename and performs every other. The instant
// between the two renames of a swap is reachable no other way, and it is the
// only instant this function makes a promise about -- so the undo that
// follows has to go through, which is why one call fails and not all after it.
func failingOn(n int, err error) func(string, string) error {
	calls := 0
	return func(source, target string) error {
		calls++
		if calls == n {
			return err
		}
		return os.Rename(source, target)
	}
}

// renaming is the real calls with the rename swapped out.
func renaming(rename func(string, string) error) dirOps {
	ops := realDirs()
	ops.rename = rename
	return ops
}

// tree writes a directory with one file in it, enough to tell two stocks
// apart.
func tree(t *testing.T, dir, text string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return dir
}

// mark reads the one file a tree carries.
func mark(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return string(data)
}

// A swap killed between its two renames leaves no half: the old stock is put
// back whole, and no caller ever sees a target made of both.
func TestReplaceDirPutsTheOldStockBackWhenTheSwapFails(t *testing.T) {
	root := t.TempDir()
	target := tree(t, filepath.Join(root, "areas", "scope"), "old\n")
	staging := tree(t, filepath.Join(root, "areas", "scope.staging"), "new\n")

	refused := errors.New("refused")
	err := replaceDir(staging, target, renaming(failingOn(2, refused)))
	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
	if got := mark(t, target); got != "old\n" {
		t.Fatalf("index.md = %q, want the old stock", got)
	}
}

// Fails the move aside itself, nothing has moved: the target is untouched and
// the caller keeps its staging.
func TestReplaceDirReportsAFailedMoveAside(t *testing.T) {
	root := t.TempDir()
	target := tree(t, filepath.Join(root, "areas", "scope"), "old\n")
	staging := tree(t, filepath.Join(root, "areas", "scope.staging"), "new\n")

	refused := errors.New("refused")
	err := replaceDir(staging, target, renaming(failingOn(1, refused)))
	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
	if got := mark(t, target); got != "old\n" {
		t.Fatalf("index.md = %q, want the old stock", got)
	}
	if got := mark(t, staging); got != "new\n" {
		t.Fatalf("staging = %q, want it left to the caller", got)
	}
}

// Nothing was moved aside, so the failed swap has nothing to undo -- and the
// error still names the target.
func TestReplaceDirReportsAFailedSwapIntoAnEmptyPlace(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "areas", "scope")
	staging := tree(t, filepath.Join(root, "areas", "scope.staging"), "new\n")

	refused := errors.New("refused")
	if err := replaceDir(staging, target, renaming(failingOn(1, refused))); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target exists: %v", err)
	}
}

// The stock is swapped in but the aside will not go. The readers are served
// -- the target is whole and new -- and the caller still hears of the sibling
// left in the state directory.
func TestReplaceDirReportsAnAsideItCannotRemove(t *testing.T) {
	root := t.TempDir()
	target := tree(t, filepath.Join(root, "areas", "scope"), "old\n")
	staging := tree(t, filepath.Join(root, "areas", "scope.staging"), "new\n")

	refused := errors.New("refused")
	ops := realDirs()
	ops.remove = func(string) error { return refused }
	if err := replaceDir(staging, target, ops); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
	if got := mark(t, target); got != "new\n" {
		t.Fatalf("index.md = %q, want the new stock", got)
	}
}

// A target that is neither there nor readable is not an absence, and taking
// it for one would swap over a stock this process cannot see.
func TestMoveAwayReportsAStatThatIsNotAnAbsence(t *testing.T) {
	refused := errors.New("refused")
	ops := realDirs()
	ops.stat = func(string) (fs.FileInfo, error) { return nil, refused }
	if _, err := moveAway("scope", "aside", ops); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
}

// Recover's own two failure arms. Both end the call rather than leaving the
// caller to swap over an aside whose fate nobody knows.
func TestRecoverReportsAnAsideItCannotRemove(t *testing.T) {
	root := t.TempDir()
	target := tree(t, filepath.Join(root, "scope"), "new\n")
	tree(t, target+AsideSuffix, "old\n")

	refused := errors.New("refused")
	ops := realDirs()
	ops.remove = func(string) error { return refused }
	if err := recoverAside(target, ops); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
}

func TestRecoverReportsAnAsideItCannotMoveBack(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "scope")
	tree(t, target+AsideSuffix, "old\n")

	refused := errors.New("refused")
	if err := recoverAside(target, renaming(failingOn(1, refused))); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
}

// A swap always finishes the last one first. Could it not, it stops there
// rather than renaming the new stock over an aside whose fate is unsettled.
func TestReplaceDirStopsWhenTheLastSwapCannotBeFinished(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "areas", "scope")
	tree(t, target+AsideSuffix, "old\n")
	staging := tree(t, filepath.Join(root, "areas", "scope.staging"), "new\n")

	refused := errors.New("refused")
	if err := replaceDir(staging, target, renaming(failingOn(1, refused))); !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
	if got := mark(t, target+AsideSuffix); got != "old\n" {
		t.Fatalf("aside = %q, want the old stock still there", got)
	}
}
