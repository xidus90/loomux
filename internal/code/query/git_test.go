package query

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
)

// hookIndex stages the edit to Add in a copy of the index inside the git
// directory, the way `git commit -a` hands its hook a temporary index, and
// leaves .git/index with nothing staged. It returns the copy's absolute path.
func hookIndex(t *testing.T, root string) string {
	t.Helper()
	editAdd(t, root)
	// editAdd keeps the file's size. Landing in the clock tick of the commit,
	// the edit would leave the stat data the index recorded unchanged, and
	// the copy below, newer than the file, switches off git's racy-clean
	// check: `git add` would skip the file. A later mtime makes it look.
	src := filepath.Join(root, "calc", "calc.go")
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(src, later, later); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(root, ".git", "next-index-1.lock")
	if err := os.WriteFile(index, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", root, "add", "calc/calc.go")
	cmd.Env = append(gitenv.Environ(), "GIT_INDEX_FILE="+index)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	return index
}

// A commit hook reads the index git hands it, not .git/index: under
// `git commit -a` only the temporary one holds the change.
func TestGitReadsTheIndexACommitHookHandsIn(t *testing.T) {
	root := gitRepo(t)
	index := hookIndex(t, root)
	if ok, note := GraphReady(root); ok || note != "nothing staged" {
		t.Fatalf("without the variable: %v %q", ok, note)
	}
	t.Setenv("GIT_INDEX_FILE", index)
	if ok, note := GraphReady(root); !ok || note != "" {
		t.Fatalf("with the hook's index: %v %q", ok, note)
	}
	a, _, err := Blast(root, BlastOptions{Cached: true, NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := areaPaths(a); !slices.Equal(got, []string{"calc/calc.go"}) {
		t.Fatalf("areas %v", got)
	}
}

// An index outside the repository's git directory belongs to some other
// repository, and a relative one names no place we can check: both are the
// inherited environment gitenv keeps away.
func TestGitIgnoresAnIndexThatIsNotThisRepositorys(t *testing.T) {
	root := gitRepo(t)
	staged := stagedAdd(t)
	hookIndex(t, root)
	for name, value := range map[string]string{
		"another repository": filepath.Join(staged, ".git", "index"),
		"relative":           filepath.Join(".git", "next-index-1.lock"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GIT_INDEX_FILE", value)
			if ok, note := GraphReady(root); ok || note != "nothing staged" {
				t.Fatalf("%v %q", ok, note)
			}
		})
	}
}

func TestIndexFileForOutsideARepositoryIsNothing(t *testing.T) {
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "index"))
	if index, ok := indexFileFor(t.TempDir()); ok || index != "" {
		t.Fatalf("%q %v", index, ok)
	}
}

// GraphEnv hands a lane's child processes exactly the index indexFileFor
// accepts, and nothing when it accepts none.
func TestGraphEnvCarriesTheIndexACommitHookHandsIn(t *testing.T) {
	root := gitRepo(t)
	index := hookIndex(t, root)
	t.Setenv("GIT_INDEX_FILE", index)
	if got := GraphEnv(root); !slices.Equal(got, []string{"GIT_INDEX_FILE=" + index}) {
		t.Fatalf("got %q", got)
	}
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "index"))
	if got := GraphEnv(root); got != nil {
		t.Fatalf("a foreign index: got %q", got)
	}
}

// The git directory itself is no index, and an index under worktrees/ of
// the main git directory belongs to a linked worktree: only that worktree
// may read it.
func TestIndexFileForAcceptsOnlyThisWorktreesOwnIndex(t *testing.T) {
	root := gitRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	git(t, root, "worktree", "add", "-q", linked)
	gitDir := filepath.Join(root, ".git")
	linkedIndex := filepath.Join(gitDir, "worktrees", "linked", "index")
	for name, c := range map[string]struct {
		dir, index string
		ok         bool
	}{
		"the git directory":               {root, gitDir, false},
		"a linked worktree's, from main":  {root, linkedIndex, false},
		"upper-case worktrees, from main": {root, filepath.Join(gitDir, "Worktrees", "linked", "index"), false},
		"its own, from the linked one":    {linked, linkedIndex, true},
		"main's, from the linked one":     {linked, filepath.Join(gitDir, "index"), false},
		"main's own":                      {root, filepath.Join(gitDir, "index"), true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GIT_INDEX_FILE", c.index)
			if got, ok := indexFileFor(c.dir); ok != c.ok || (ok && got != c.index) {
				t.Fatalf("%q %v, want ok %v", got, ok, c.ok)
			}
		})
	}
}
