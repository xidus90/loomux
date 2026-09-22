package gitwork

import (
	"errors"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/child"
)

func TestLsRemoteReadsTheRemote(t *testing.T) {
	root := repoWithCommit(t)
	bare := t.TempDir()
	mustGit(t, bare, "init", "-q", "--bare")
	mustGit(t, root, "remote", "add", "origin", bare)
	mustGit(t, root, "push", "-q", "origin", "HEAD:refs/heads/master")
	refs, err := LsRemote(root, "origin")
	head := mustGit(t, root, "rev-parse", "HEAD")
	if err != nil || refs["refs/heads/master"] != head {
		t.Fatalf("refs %v, %v", refs, err)
	}
}

func TestLsRemoteWithoutRemote(t *testing.T) {
	if _, err := LsRemote(repoWithCommit(t), "origin"); err == nil {
		t.Fatal("want an error")
	}
}

// The deadline and the one variable that goes with the call are the whole
// contract with child: ten seconds, and no credential prompt. SSH is
// deliberately not touched, so the user's own setup keeps working.
func TestLsRemoteGivesUpAtItsDeadline(t *testing.T) {
	old := remoteStart
	t.Cleanup(func() { remoteStart = old })
	var seen child.Spec
	remoteStart = func(s child.Spec) child.Result { seen = s; return child.Result{Code: -1, TimedOut: true} }
	if _, err := LsRemote(t.TempDir(), "origin"); err == nil {
		t.Fatal("want an error")
	}
	if seen.Timeout != RemoteTimeout || !slices.Contains(seen.Env, "GIT_TERMINAL_PROMPT=0") {
		t.Fatalf("spec %+v", seen)
	}
}

// git that never started at all: no exit code to read, and what there is to
// say arrives through Err.
func TestLsRemoteReportsAStartThatFailed(t *testing.T) {
	old := remoteStart
	t.Cleanup(func() { remoteStart = old })
	remoteStart = func(child.Spec) child.Result {
		return child.Result{Code: -1, Err: errors.New("no git")}
	}
	if _, err := LsRemote(t.TempDir(), "origin"); err == nil {
		t.Fatal("want an error")
	}
}

func TestHasRemoteNamesAConfiguredRemote(t *testing.T) {
	root := repoWithCommit(t)
	if HasRemote(root, "origin") {
		t.Fatal("a repository without a remote has origin")
	}
	mustGit(t, root, "remote", "add", "origin", t.TempDir())
	if !HasRemote(root, "origin") {
		t.Fatal("a configured origin is missing")
	}
}
