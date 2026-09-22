package apply

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/vcs"
	"github.com/xidus90/loomux/internal/gitenv"
)

// The one commit of a decision, ported from `_commit` and `_report`
// (apply.py:1318-1378). Where a test carries over one of test_apply.py, its
// comment names it and its line.

// commitCall is what one call of the faked vcs.CommitPaths was handed.
type commitCall struct {
	repo, message, scratch string
	add, remove            []string
}

// fakeCommits swaps commitPaths for answers handed out one per call, the
// last one repeated, and returns the record of every call.
func fakeCommits(t *testing.T, answers ...func() (*vcs.Commit, error)) *[]commitCall {
	t.Helper()
	calls := &[]commitCall{}
	seam(t, &commitPaths, func(repo, message string, add, remove []string, scratch string) (*vcs.Commit, error) {
		*calls = append(*calls, commitCall{repo: repo, message: message, scratch: scratch, add: add, remove: remove})
		return answers[min(len(*calls), len(answers))-1]()
	})
	return calls
}

// movedRef is ErrRefMoved as vcs hands it out: wrapped, so a comparison
// with == instead of errors.Is fails here.
func movedRef() (*vcs.Commit, error) {
	return nil, fmt.Errorf("%w: refs/heads/master moved away from abc", vcs.ErrRefMoved)
}

func landed(head string) func() (*vcs.Commit, error) {
	return func() (*vcs.Commit, error) { return &vcs.Commit{Head: head, Created: true}, nil }
}

func unchanged() (*vcs.Commit, error) { return &vcs.Commit{Head: "old", Created: false}, nil }

func wantCommit(t *testing.T, sha, warning, wantSha, wantWarning string) {
	t.Helper()
	if sha != wantSha || warning != wantWarning {
		t.Fatalf("commit = (%q, %q), want (%q, %q)", sha, warning, wantSha, wantWarning)
	}
}

func TestCommitHandsTheNamedPathsToGit(t *testing.T) {
	calls := fakeCommits(t, landed("abc"))
	sha, warning := commit("vault", "c1", "written", "Land it", []string{"a.md"}, []string{"review/c1"}, "scratch")
	wantCommit(t, sha, warning, "abc", "")
	want := []commitCall{{repo: "vault", message: "Land it\n", scratch: "scratch",
		add: []string{"a.md"}, remove: []string{"review/c1"}}}
	if !slices.EqualFunc(*calls, want, sameCall) {
		t.Fatalf("calls = %+v, want %+v", *calls, want)
	}
}

func sameCall(a, b commitCall) bool {
	return a.repo == b.repo && a.message == b.message && a.scratch == b.scratch &&
		slices.Equal(a.add, b.add) && slices.Equal(a.remove, b.remove)
}

// `_commit` (apply.py:1351-1359): a moved ref is tried again, once, with the
// same arguments.
func TestCommitRetriesOnceAfterAMovedRef(t *testing.T) {
	calls := fakeCommits(t, movedRef, landed("def"))
	sha, warning := commit("vault", "c1", "written", "Land it", []string{"a.md"}, []string{"review/c1"}, "scratch")
	wantCommit(t, sha, warning, "def", "")
	if len(*calls) != 2 || !sameCall((*calls)[0], (*calls)[1]) {
		t.Fatalf("calls = %+v, want the same call twice", *calls)
	}
}

// A second moved ref is reported, not retried: the window is milliseconds
// wide, so losing it twice means something commits continuously.
func TestCommitWarnsOnASecondMovedRef(t *testing.T) {
	calls := fakeCommits(t, movedRef)
	sha, warning := commit("vault", "c1", "written", "Land it", nil, []string{"review/c1"}, "scratch")
	wantCommit(t, sha, warning, "",
		"c1: written, but not committed (the branch moved while committing: refs/heads/master moved away from abc)")
	if len(*calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(*calls))
	}
}

// test_a_rejection_that_cannot_be_committed_does_not_claim_a_write
// (test_apply.py:1028): any other git failure is a warning after one call,
// and a decision that wrote nothing says so.
func TestCommitWarnsOnAnyOtherGitError(t *testing.T) {
	calls := fakeCommits(t, func() (*vcs.Commit, error) { return nil, errors.New("kaputt") })
	sha, warning := commit("vault", "c1", "decision recorded", "Reject", nil, []string{"review/c1"}, "scratch")
	wantCommit(t, sha, warning, "", "c1: decision recorded, but not committed (kaputt)")
	if len(*calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(*calls))
	}
}

// The case id enters the warning as it stands, as in `_commit`: the warning
// goes to the terminal, not into a protocol.
func TestCommitNamesTheCaseAsItStands(t *testing.T) {
	fakeCommits(t, func() (*vcs.Commit, error) { return nil, errors.New("kaputt") })
	_, warning := commit("vault", "f-%%%", "written", "Land it", nil, nil, "scratch")
	if !strings.HasPrefix(warning, "f-%%%: ") {
		t.Fatalf("warning = %q", warning)
	}
}

func TestCommitWarnsWithoutARepository(t *testing.T) {
	requireGit(t)
	sha, warning := commit(t.TempDir(), "c1", "written", "Land it", nil, nil, t.TempDir())
	wantCommit(t, sha, warning, "", "c1: no git repository in the vault; written but not committed")
}

// test_a_rejection_without_a_repository_does_not_claim_a_write
// (test_apply.py:1045).
func TestCommitWarnsWithoutARepositoryThatNothingWasWritten(t *testing.T) {
	fakeCommits(t, func() (*vcs.Commit, error) { return nil, nil })
	sha, warning := commit("vault", "c1", "decision recorded", "Reject", nil, nil, "scratch")
	wantCommit(t, sha, warning, "", "c1: no git repository in the vault; decision recorded but not committed")
}

// `_report` (apply.py:1373-1377): an unchanged tree is no success; the case
// belongs back in the queue.
func TestCommitWarnsWhenNothingChangedOnTheFirstAttempt(t *testing.T) {
	fakeCommits(t, unchanged)
	sha, warning := commit("vault", "c1", "written", "Land it", nil, nil, "scratch")
	wantCommit(t, sha, warning, "", "c1: nothing to commit on the first attempt; the case belongs back in the queue")
}

func TestCommitWarnsWhenNothingChangedAfterTheRetry(t *testing.T) {
	fakeCommits(t, movedRef, unchanged)
	sha, warning := commit("vault", "c1", "written", "Land it", nil, nil, "scratch")
	wantCommit(t, sha, warning, "", "c1: nothing to commit after the retry; the case belongs back in the queue")
}

// The real vcs.CommitPaths behind commit, once, end to end.
func TestCommitLandsInARealRepository(t *testing.T) {
	repo := gitVault(t, map[string]string{"a.md": "alt\n", "review/c1/case.toml": "x\n"})
	writeFile(t, filepath.Join(repo, "a.md"), "neu\n")
	sha, warning := commit(repo, "c1", "written", "Land it", []string{"a.md"}, []string{"review/c1"}, t.TempDir())
	wantCommit(t, sha, warning, gitOut(t, repo, "rev-parse", "HEAD"), "")
	if got := committedPaths(t, repo); !slices.Equal(got, []string{"a.md", "review/c1/case.toml"}) {
		t.Fatalf("paths of the commit = %q", got)
	}
	if got := gitOut(t, repo, "log", "-1", "--format=%B"); got != "Land it" {
		t.Fatalf("message = %q", got)
	}
}

// requireGit skips where git cannot run: a failed spawn reads like "no
// repository", and the test would pass for the wrong reason.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

// gitVault is a repository holding files in one commit, with an identity
// of its own for the commits CommitPaths makes.
func gitVault(t *testing.T, files map[string]string) string {
	t.Helper()
	requireGit(t)
	repo := t.TempDir()
	gitRun(t, repo, "init", "-q")
	gitRun(t, repo, "config", "user.name", "Brain")
	gitRun(t, repo, "config", "user.email", "brain@example.invalid")
	gitRun(t, repo, "config", "commit.gpgsign", "false")
	for name, text := range files {
		writeFile(t, filepath.Join(repo, filepath.FromSlash(name)), text)
	}
	gitRun(t, repo, "add", "--all")
	gitRun(t, repo, "commit", "-q", "-m", "first")
	return repo
}

// gitRun and gitOut strip the git environment as vcs does: the pre-commit
// hook runs these tests with GIT_DIR and GIT_INDEX_FILE exported, and both
// outrank the working directory.
func gitRun(t *testing.T, repo string, args ...string) {
	t.Helper()
	gitOut(t, repo, args...)
}

func gitOut(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repo
	command.Env = gitenv.Environ()
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// committedPaths is the paths the head commit changed, sorted.
func committedPaths(t *testing.T, repo string) []string {
	t.Helper()
	listing := gitOut(t, repo, "diff-tree", "--root", "--no-commit-id", "-r", "-z", "--name-only", "HEAD")
	paths := []string{}
	for _, entry := range strings.Split(listing, "\x00") {
		if entry != "" {
			paths = append(paths, entry)
		}
	}
	sort.Strings(paths)
	return paths
}
