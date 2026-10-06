package wiki

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

	if changed, underGit, err := getGitChangedFiles(t.TempDir()); changed != nil || underGit || err != nil {
		t.Fatalf("a directory that is no repository answered with %v, %v, %v", changed, underGit, err)
	}
}

func TestGetGitChangedFilesListsTheChangedPaths(t *testing.T) {
	changed, _, _ := getGitChangedFiles(decoyRepo(t))
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

	changed, _, _ := getGitChangedFiles(root)
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

// driftOf keeps the wiki-drift violations, because a wiki page written for a
// test also draws lint violations the drift check does not care about.
func driftOf(violations []GateViolation) []GateViolation {
	var drift []GateViolation
	for _, v := range violations {
		if v.Name == "wiki-drift" {
			drift = append(drift, v)
		}
	}
	return drift
}

// The wiki is a directory of the project's own repository and nothing under it
// changed, while the project has an untracked file. Asked from the wiki
// directory, git answers for the whole repository, so a second status call saw
// the project's change as the wiki's and the drift could never be reported.
func TestCheckWikiGateReportsDriftForAWikiInTheSameRepository(t *testing.T) {
	project := decoyRepo(t)
	mkdir(t, project, "docs", "wiki")

	drift := driftOf(CheckWikiGate(project))
	if len(drift) != 1 || !strings.Contains(drift[0].Message, "(1 changed files)") {
		t.Fatalf("got %v", drift)
	}
}

// A page in a wiki directory git has never seen is a wiki change. Without
// listing every untracked file git folds the new directory into one line,
// `?? docs/`, which lies outside the wiki and would read as drift.
func TestCheckWikiGateCountsAnUntrackedWikiPageAsAWikiChange(t *testing.T) {
	project := decoyRepo(t)
	writePage(t, project, "docs/wiki/page.md", "---\ntitle: Page\ntype: concept\n---\n")

	if drift := driftOf(CheckWikiGate(project)); len(drift) != 0 {
		t.Fatalf("got %v", drift)
	}
}

func TestSplitAtWikiSeparatesTheWikiFromTheCode(t *testing.T) {
	changes := []string{"loud.md", "docs/wiki/page.md", "docs/wikipedia.md", "docs/wiki/sub/"}
	code, wikiChanges := splitAtWiki(changes, "docs/wiki")
	if strings.Join(code, ",") != "loud.md,docs/wikipedia.md" ||
		strings.Join(wikiChanges, ",") != "docs/wiki/page.md,docs/wiki/sub/" {
		t.Fatalf("code %v, wiki %v", code, wikiChanges)
	}
}

// A wiki declared at the project root holds every change of the project.
func TestSplitAtWikiGivesEverythingToAWikiAtTheRoot(t *testing.T) {
	code, wikiChanges := splitAtWiki([]string{"loud.md"}, ".")
	if len(code) != 0 || len(wikiChanges) != 1 {
		t.Fatalf("code %v, wiki %v", code, wikiChanges)
	}
}

// gitFailuresOf keeps the violations that name a git failure.
func gitFailuresOf(violations []GateViolation) []GateViolation {
	var failures []GateViolation
	for _, v := range violations {
		if v.Name == "wiki-git" {
			failures = append(failures, v)
		}
	}
	return failures
}

// `git status` that fails in a repository is no answer of "nothing changed":
// a corrupt index made the gate pass with "no drift detected". The failure is
// a violation that carries git's own message.
func TestCheckWikiGateReportsAGitStatusThatFails(t *testing.T) {
	project := decoyRepo(t)
	mkdir(t, project, "docs", "wiki")
	if err := os.WriteFile(filepath.Join(project, ".git", "index"), []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}

	failures := gitFailuresOf(CheckWikiGate(project))
	if len(failures) != 1 || !strings.Contains(failures[0].Message, "index file smaller than expected") || !strings.Contains(failures[0].Message, "in "+project+":") {
		t.Fatalf("got %v", failures)
	}
}

// The same failure in a wiki that carries a repository of its own.
func TestCheckWikiGateReportsAGitStatusThatFailsInTheWiki(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "proj")
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = mkdir(t, project)
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(project, "loud.md"), []byte("loud"), 0o644); err != nil {
		t.Fatal(err)
	}
	wiki := mkdir(t, parent, "proj_wiki")
	mkdir(t, wiki, ".git")

	failures := gitFailuresOf(CheckWikiGate(project))
	if len(failures) != 1 || !strings.Contains(failures[0].Message, "in "+wiki+":") {
		t.Fatalf("got %v", failures)
	}
}

// A wiki beside the project that is no repository has no `git status` to
// compare with the project's: it read as untouched for ever, and every change
// of the code was drift.
func TestCheckWikiGateDoesNotJudgeDriftForAWikiThatIsNoRepository(t *testing.T) {
	parent := t.TempDir()
	project := filepath.Join(parent, "proj")
	cmd := exec.Command("git", "init", "-q", "-b", "main")
	cmd.Dir = mkdir(t, project)
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(project, "loud.md"), []byte("loud"), 0o644); err != nil {
		t.Fatal(err)
	}
	mkdir(t, parent, "proj_wiki")

	violations := CheckWikiGate(project)
	if len(driftOf(violations)) != 0 || len(gitFailuresOf(violations)) != 0 {
		t.Fatalf("got %v", violations)
	}
}

// A project that is no repository has no change to judge either.
func TestCheckWikiGateDoesNotJudgeDriftForAProjectThatIsNoRepository(t *testing.T) {
	project := t.TempDir()
	mkdir(t, project, "docs", "wiki")

	violations := CheckWikiGate(project)
	if len(driftOf(violations)) != 0 || len(gitFailuresOf(violations)) != 0 {
		t.Fatalf("got %v", violations)
	}
}
