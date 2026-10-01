package gitwork

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
)

// aliasOf is another path to dir: a symlink where the platform lets a test
// make one, a junction on Windows. Without either the caller skips.
func aliasOf(t *testing.T, dir string) string {
	t.Helper()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(dir, link); err == nil {
		return link
	}
	if runtime.GOOS == "windows" {
		if err := exec.Command("cmd", "/c", "mklink", "/J", link, dir).Run(); err == nil {
			return link
		}
	}
	t.Skip("neither a symlink nor a junction can be made here")
	return ""
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// What git hands a pre-commit hook, as measured: the real index, relative,
// for a plain commit; index.lock for -a and --include; next-index-<pid>.lock
// for a commit of paths. Only the first two become the index afterwards.
func TestCommitIndexTellsAWholeCommitFromAPartialOne(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	gitDir := filepath.Join(root, ".git")

	if index, whole := CommitIndex(root, ""); index != "" || !whole {
		t.Fatalf("no index handed in: %q %v", index, whole)
	}
	// Outside a repository too: a run by hand, where nothing is handed in.
	if index, whole := CommitIndex(t.TempDir(), ""); index != "" || !whole {
		t.Fatalf("no index, no repository: %q %v", index, whole)
	}

	realIndex := filepath.Join(gitDir, "index")
	if index, whole := CommitIndex(root, realIndex); index != realIndex || !whole {
		t.Fatalf("the real index: %q %v", index, whole)
	}
	// Relative, as git names it for a plain commit: relative to the directory
	// the hook runs in, and handed on as an absolute path.
	t.Chdir(root)
	index, whole := CommitIndex(root, filepath.Join(".git", "index"))
	if !whole || !filepath.IsAbs(index) || filepath.Base(index) != "index" {
		t.Fatalf("the real index, relative: %q %v", index, whole)
	}

	lock := filepath.Join(gitDir, "index.lock")
	touch(t, lock)
	if index, whole := CommitIndex(root, lock); index != lock || !whole {
		t.Fatalf("index.lock: %q %v", index, whole)
	}
	os.Remove(lock)

	partial := filepath.Join(gitDir, "next-index-4711.lock")
	touch(t, partial)
	if index, whole := CommitIndex(root, partial); index != "" || whole {
		t.Fatalf("a commit of paths: %q %v", index, whole)
	}
	// A copy of the real index is another file, whatever it holds.
	data, _ := os.ReadFile(realIndex)
	os.WriteFile(partial, data, 0o644)
	if _, whole := CommitIndex(root, partial); whole {
		t.Fatal("a copy of the index counts as the index")
	}
	if _, whole := CommitIndex(root, filepath.Join(gitDir, "gone")); whole {
		t.Fatal("an index that is not there counts as whole")
	}
	// The index of another repository is not this one's.
	other := repo(t)
	commit(t, other, "first")
	if _, whole := CommitIndex(root, filepath.Join(other, ".git", "index")); whole {
		t.Fatal("another repository's index counts as whole")
	}
	// No repository to hold it against.
	stray := filepath.Join(t.TempDir(), "index")
	touch(t, stray)
	if _, whole := CommitIndex(filepath.Dir(stray), stray); whole {
		t.Fatal("an index outside every repository counts as whole")
	}
}

// By identity, not by spelling: git answers the long name of its directory,
// the hook's variable may spell the same file through a short name, another
// case or a link.
func TestCommitIndexComparesFilesNotSpellings(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	alias := aliasOf(t, root)
	spelt := filepath.Join(alias, ".git", "index")
	if index, whole := CommitIndex(root, spelt); index != spelt || !whole {
		t.Fatalf("the index through another path: %q %v", index, whole)
	}
}

// A linked worktree has an index of its own, in its own git directory under
// the main one. The main worktree's index is another commit's.
func TestCommitIndexKnowsTheIndexOfALinkedWorktree(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	linked := filepath.Join(t.TempDir(), "linked")
	run(t, root, "worktree", "add", "-q", linked, "-b", "linked")
	own := filepath.Join(root, ".git", "worktrees", "linked", "index")
	if index, whole := CommitIndex(linked, own); index != own || !whole {
		t.Fatalf("the linked worktree's own index: %q %v", index, whole)
	}
	if _, whole := CommitIndex(linked, filepath.Join(root, ".git", "index")); whole {
		t.Fatal("the main worktree's index counts as the linked one's")
	}
	if _, whole := CommitIndex(root, own); whole {
		t.Fatal("the linked worktree's index counts as the main one's")
	}
}

func TestStagePutsAFileIntoTheIndexItIsGiven(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	staged := func(env ...string) string {
		t.Helper()
		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = root
		cmd.Env = append(gitenv.Environ(), env...)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	touch(t, filepath.Join(root, "f.txt"))
	if err := Stage(root, "", "f.txt"); err != nil || staged() != "f.txt" {
		t.Fatalf("git's own index: %v, staged %q", err, staged())
	}
	// Another index: the file lands there and the real one stays as it was.
	other := filepath.Join(root, ".git", "other-index")
	data, _ := os.ReadFile(filepath.Join(root, ".git", "index"))
	os.WriteFile(other, data, 0o644)
	touch(t, filepath.Join(root, "g.txt"))
	if err := Stage(root, other, "g.txt"); err != nil {
		t.Fatal(err)
	}
	if staged() != "f.txt" || staged("GIT_INDEX_FILE="+other) != "f.txt\ng.txt" {
		t.Fatalf("real %q, other %q", staged(), staged("GIT_INDEX_FILE="+other))
	}
	// A file git ignores is refused, and the refusal comes back.
	touch(t, filepath.Join(root, "h.txt"))
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("h.txt\n"), 0o644)
	if err := Stage(root, "", "h.txt"); err == nil {
		t.Fatal("an ignored file was staged without a word")
	}
}
