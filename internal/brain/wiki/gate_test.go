package wiki

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
)

// decoyRepo is a repository with one untracked file: asked about itself it
// answers loudly, which is what makes it useful as a decoy.
func decoyRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = root
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, "loud.md"), []byte("loud"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// A stop hook runs inside the host's environment, and a git hook exports
// GIT_DIR. Unscrubbed it outranks cmd.Dir, so a directory that is no
// repository answers with the exported repository's changes -- and a wiki
// looks touched when nothing touched it.
func TestGetGitChangedFilesIgnoresAnInheritedGitDir(t *testing.T) {
	decoy := decoyRepo(t)
	t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))
	t.Setenv("GIT_WORK_TREE", decoy)

	if changed := getGitChangedFiles(t.TempDir()); changed != nil {
		t.Fatalf("a directory that is no repository answered with %v", changed)
	}
}

func TestGetGitChangedFilesListsTheChangedPaths(t *testing.T) {
	changed := getGitChangedFiles(decoyRepo(t))
	if len(changed) != 1 || changed[0] != "loud.md" {
		t.Fatalf("got %v", changed)
	}
}

// A tracked file that was changed is the porcelain line the old order got
// wrong: `git status --porcelain` writes two status columns and a blank, and
// the worktree-only change leaves the first column blank -- " M name". Trimming
// the whole line before cutting the fixed three characters ate the first two of
// the name, so every such path came back corrupted.
func TestGetGitChangedFilesKeepsThePathOfAWorktreeOnlyChange(t *testing.T) {
	root := decoyRepo(t)
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{
			"-c", "user.name=loomux test", "-c", "user.email=test@example.invalid",
		}, args...)...)
		cmd.Dir = root
		cmd.Env = gitenv.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("add", "loud.md")
	git("commit", "-q", "-m", "first")
	if err := os.WriteFile(filepath.Join(root, "loud.md"), []byte("louder"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed := getGitChangedFiles(root)
	if len(changed) != 1 || changed[0] != "loud.md" {
		t.Fatalf("got %v, want the whole path", changed)
	}
}

// The wiki is a repository of its own and clean, while the project has an
// untracked file: code moved, the wiki did not.
func TestCheckWikiGateReportsDrift(t *testing.T) {
	project := decoyRepo(t)
	wikiRoot := mkdir(t, project, "docs", "wiki")
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = wikiRoot
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	violations := CheckWikiGate(project)
	if len(violations) != 1 || violations[0].Name != "wiki-drift" {
		t.Fatalf("got %v", violations)
	}
}
