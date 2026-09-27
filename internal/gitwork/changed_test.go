package gitwork

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChangedFilesOfACleanTreeIsEmpty(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	got, err := ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("a clean tree has no changes, got %v", got)
	}
}

// -uall, so an untracked directory comes back as the files in it and not as
// one entry that is no file's path.
func TestChangedFilesListsModifiedAndUntrackedFiles(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, "a.txt"), "changed")
	writeFile(t, filepath.Join(root, "new", "deep", "b.txt"), "b")
	got, err := ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"a.txt", "new/deep/b.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A rename is two fields and only the first carries the status prefix; the
// old path is a change too, or a test moved out of the way would go unseen.
func TestChangedFilesReportsBothSidesOfARename(t *testing.T) {
	root := repo(t)
	writeFile(t, filepath.Join(root, "old.txt"), "o")
	run(t, root, "add", "old.txt")
	run(t, root, "commit", "-m", "first")
	run(t, root, "mv", "old.txt", "renamed.txt")
	got, err := ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"old.txt", "renamed.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A new path that is only intent-to-add marks the rename in the worktree
// column: git prints " R cd\0ab\0", measured on 2026-09-11 with git 2.54. Read
// from the index column alone, the bare old path would lose three bytes it does
// not have, and the whole call would panic instead of answering.
func TestChangedFilesReportsBothSidesOfAnIntentToAddRename(t *testing.T) {
	root := repo(t)
	writeFile(t, filepath.Join(root, "ab"), "a file long enough to read as renamed\n")
	run(t, root, "add", "ab")
	run(t, root, "commit", "-m", "first")
	if err := os.Rename(filepath.Join(root, "ab"), filepath.Join(root, "cd")); err != nil {
		t.Fatal(err)
	}
	run(t, root, "add", "-N", "cd")
	got, err := ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"ab", "cd"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// git answers relative to the repository root; a root below it gets its own
// spelling, and changes outside it are not its project's.
func TestChangedFilesBelowTheRepositoryRoot(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, "top.txt"), "t")
	writeFile(t, filepath.Join(root, "sub", "x.txt"), "x")
	got, err := ChangedFiles(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"x.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// A directory name may begin with a space, and git's prefix keeps it. Trimmed
// away, the prefix would match no path, and the call would answer "nothing
// changed" without an error.
func TestChangedFilesBelowADirectoryWhoseNameBeginsWithASpace(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, " sub", "x.txt"), "x")
	got, err := ChangedFiles(filepath.Join(root, " sub"))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"x.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// -z, so a non-ASCII path is not C-quoted into a string no rule matches.
func TestChangedFilesKeepsANonASCIIPathVerbatim(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, "grün.txt"), "g")
	got, err := ChangedFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"grün.txt"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestChangedFilesOutsideARepository(t *testing.T) {
	if _, err := ChangedFiles(t.TempDir()); err == nil {
		t.Fatal("a directory that is not a repository has no changes to report")
	}
}

func TestChangedFilesRefusesAnIgnoredRoot(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, ".gitignore"), "parked/\n")
	parked := filepath.Join(root, "parked")
	if err := os.MkdirAll(parked, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := ChangedFiles(parked)
	if !errors.Is(err, ErrIgnoredRoot) {
		t.Fatalf("want ErrIgnoredRoot, got %v", err)
	}
}

// Measured on 2026-09-11: with a damaged index, rev-parse --show-prefix still
// answers and status fails with exit 128.
func TestChangedFilesReportsAStatusThatFails(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	writeFile(t, filepath.Join(root, ".git", "index"), "garbage")
	_, err := ChangedFiles(root)
	if err == nil || !strings.Contains(err.Error(), "git status") {
		t.Fatalf("want the failed status named, got %v", err)
	}
}
