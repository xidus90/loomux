package gitwork

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func indexOf(t *testing.T, root string) string {
	t.Helper()
	return filepath.Join(mustGit(t, root, "rev-parse", "--absolute-git-dir"), "index")
}

func TestStagedPathsNamesWhatTheCommitChanges(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"docs/ä.md": "x", "b.go": "package b"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustGit(t, root, "add", "docs", "b.go")
	got, err := StagedPaths(root, indexOf(t, root))
	slices.Sort(got)
	if err != nil || !slices.Equal(got, []string{"b.go", "docs/ä.md"}) {
		t.Fatalf("%q %v", got, err)
	}
}

// A rename lists both ends: code that left must count.
func TestStagedPathsListsBothEndsOfARename(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "mv", "a.txt", "docs/a.txt")
	got, err := StagedPaths(root, indexOf(t, root))
	slices.Sort(got)
	if err != nil || !slices.Equal(got, []string{"a.txt", "docs/a.txt"}) {
		t.Fatalf("%q %v", got, err)
	}
}

// git names the index relative to the directory it starts the hook in. From a
// subdirectory the relative name differs from the one resolved against root.
func TestStagedPathsReadsARelativeIndexFromTheHooksDirectory(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", "b.txt")
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join(root, "sub"))
	got, err := StagedPaths(root, "../.git/index")
	if err != nil || !slices.Equal(got, []string{"b.txt"}) {
		t.Fatalf("%q %v", got, err)
	}
}

// An index of another repository is no answer: the gate runs every lane.
func TestStagedPathsRefusesAnotherRepositorysIndex(t *testing.T) {
	root, other := repoWithCommit(t), repoWithCommit(t)
	if _, err := StagedPaths(root, indexOf(t, other)); err == nil {
		t.Fatal("a foreign index must be an error")
	}
	if _, err := StagedPaths(root, filepath.Join(t.TempDir(), "index")); err == nil {
		t.Fatal("an index that is not there must be an error")
	}
	if _, err := StagedPaths(t.TempDir(), indexOf(t, root)); err == nil {
		t.Fatal("a root outside git must be an error")
	}
	dir := filepath.Dir(indexOf(t, root))
	if _, err := StagedPaths(root, filepath.Join(dir, "index.lock")); err == nil {
		t.Fatal("an index.lock that is not there must be an error")
	}
	garbage := filepath.Join(dir, "next-index-1.lock")
	if err := os.WriteFile(garbage, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(garbage)
	if _, err := StagedPaths(root, garbage); err == nil {
		t.Fatal("an index git cannot read must be an error")
	}
}

// The copies of the index a commit hands its hook: -a lays it in index.lock,
// a commit of paths in next-index-<pid>.lock.
func TestStagedPathsReadsTheIndexCopiesOfACommit(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", "b.txt")
	index := indexOf(t, root)
	bytes, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "reset", "-q")
	for _, name := range []string{"index.lock", "next-index-123.lock"} {
		copyPath := filepath.Join(filepath.Dir(index), name)
		if err := os.WriteFile(copyPath, bytes, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := StagedPaths(root, copyPath)
		os.Remove(copyPath)
		if err != nil || !slices.Equal(got, []string{"b.txt"}) {
			t.Errorf("%s: %q %v", name, got, err)
		}
	}
}

// A merge concluded by git commit hands in .git/index, with the merged
// changes staged.
func TestStagedPathsSeesAMergeBeforeItIsCommitted(t *testing.T) {
	root := repoWithCommit(t)
	main := mustGit(t, root, "branch", "--show-current")
	mustGit(t, root, "checkout", "-q", "-b", "side")
	if err := os.WriteFile(filepath.Join(root, "c.go"), []byte("package c"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", "c.go")
	mustGit(t, root, "commit", "-q", "-m", "side")
	mustGit(t, root, "checkout", "-q", main)
	if err := os.WriteFile(filepath.Join(root, "d.go"), []byte("package d"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", "d.go")
	mustGit(t, root, "commit", "-q", "-m", "own")
	mustGit(t, root, "merge", "--no-commit", "--no-ff", "-q", "side")
	got, err := StagedPaths(root, indexOf(t, root))
	if err != nil || !slices.Equal(got, []string{"c.go"}) {
		t.Fatalf("%q %v", got, err)
	}
}

func TestChangedBetweenNamesBothEndsOfARename(t *testing.T) {
	root := repoWithCommit(t)
	from := mustGit(t, root, "rev-parse", "HEAD^{tree}")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "mv", "a.txt", "docs/ä.txt")
	to := mustGit(t, root, "write-tree")
	got, err := ChangedBetween(root, from, to)
	slices.Sort(got)
	if err != nil || !slices.Equal(got, []string{"a.txt", "docs/ä.txt"}) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := ChangedBetween(root, from, "0000000000000000000000000000000000000000"); err == nil {
		t.Fatal("a tree git does not know must be an error")
	}
}

// A repository's config must not hide a staged change: with
// diff.ignoreSubmodules a bumped submodule would vanish from the list, and a
// commit of it plus docs would pass for a docs-only commit.
func TestStagedPathsNamesAGitlinkDespiteIgnoreSubmodules(t *testing.T) {
	root := repoWithCommit(t)
	mustGit(t, root, "config", "diff.ignoreSubmodules", "all")
	mustGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+mustGit(t, root, "rev-parse", "HEAD")+",sub")
	got, err := StagedPaths(root, indexOf(t, root))
	if err != nil || !slices.Equal(got, []string{"sub"}) {
		t.Fatalf("%q %v", got, err)
	}
}
