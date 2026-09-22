package hooks

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/sessions"
)

// subagentWorld is two commits with a bare remote that holds master at the
// second and a branch `old` at the first, so a test can push, delete a remote
// ref and branch without building any of it itself.
const subagentWorld = `
[[commit]]
message = "base"
[commit.files]
"a.txt" = "1\n"

[[commit]]
message = "second"
[commit.files]
"a.txt" = "2\n"

[remote.push]
master = 2
old = 1
`

const (
	subagentStartPayload = `{"session_id":"s1","agent_id":"a1","hook_event_name":"SubagentStart"}`
	subagentStopPayload  = `{"session_id":"s1","agent_id":"a1","hook_event_name":"SubagentStop"}`
)

func subagentRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWorldFile(t, root, cases.GitWorldFile, subagentWorld)
	if err := cases.BuildGitWorld(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func runSubagentStart(t *testing.T, root, payload string) (int, string) {
	t.Helper()
	var errOut strings.Builder
	code := SubagentStart(strings.NewReader(payload), &errOut, root, "claude")
	return code, errOut.String()
}

func runSubagentStop(t *testing.T, root string) (int, string) {
	t.Helper()
	var errOut strings.Builder
	code := SubagentStop(strings.NewReader(subagentStopPayload), &errOut, root, "claude")
	return code, errOut.String()
}

// startedSubagent runs the start hook and fails the test when it does not
// file a snapshot, so every stop test begins from a known one.
func startedSubagent(t *testing.T, root string) {
	t.Helper()
	if code, se := runSubagentStart(t, root, subagentStartPayload); code != ExitOK {
		t.Fatalf("start: %d %q", code, se)
	}
}

func agentFile(t *testing.T, root string) (sessions.AgentFile, bool) {
	t.Helper()
	return sessions.ReadAgent(root, "s1", "a1")
}

// finding runs the stop hook and answers the lines it parked.
func finding(t *testing.T, root string) []string {
	t.Helper()
	code, se := runSubagentStop(t, root)
	if code != ExitOK {
		t.Fatalf("stop: %d %q", code, se)
	}
	file, ok := agentFile(t, root)
	if !ok {
		return nil
	}
	return file.Finding
}

// gitIn runs git in a world with an identity of its own, the way
// cases.BuildGitWorld does: the user's configuration, hooks and signing key
// have no business in a test's commit.
func gitIn(t *testing.T, root string, args ...string) string {
	t.Helper()
	config := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(config, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", append([]string{
		"-c", "core.autocrlf=false", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + os.DevNull,
	}, args...)...)
	cmd.Dir = root
	cmd.Env = append(gitenv.Environ(),
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+config,
		"GIT_AUTHOR_NAME=loomux tests", "GIT_AUTHOR_EMAIL=tests@loomux.invalid",
		"GIT_COMMITTER_NAME=loomux tests", "GIT_COMMITTER_EMAIL=tests@loomux.invalid")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out))
}

// The expectations are read through gitwork, which is what the hook reads
// them through; a short SHA spelled by any other configuration could differ.
func remoteRefsOf(t *testing.T, root string) map[string]string {
	t.Helper()
	refs, err := gitwork.LsRemote(root, "origin")
	if err != nil {
		t.Fatal(err)
	}
	return refs
}

func localHeadsOf(t *testing.T, root string) map[string]string {
	t.Helper()
	heads, _, err := gitwork.LocalBranches(root)
	if err != nil {
		t.Fatal(err)
	}
	return heads
}

func onelineOf(t *testing.T, root, from, to string) string {
	t.Helper()
	commits, err := gitwork.LogOneline(root, from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("%d commits between %s and %s", len(commits), from, to)
	}
	return commits[0]
}

func wantFinding(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("finding %q, want %q", got, want)
	}
}

func TestSubagentStartRefusesWithoutAgent(t *testing.T) {
	root := subagentRoot(t)
	code, se := runSubagentStart(t, root, `{"session_id":"s1"}`)
	if code != ExitInternal || !strings.Contains(se, "payload carries no agent_id\n") {
		t.Fatalf("%d %q", code, se)
	}
	if _, ok := agentFile(t, root); ok {
		t.Fatal("a file was written all the same")
	}
}

// Both halves of the name are needed: a snapshot filed without a session
// lands under the shared fallback name, where the next session's stop gate
// would read it as its own subagent's.
func TestSubagentStartRefusesWithoutASession(t *testing.T) {
	root := subagentRoot(t)
	code, se := runSubagentStart(t, root, `{"agent_id":"a1"}`)
	if code != ExitInternal || !strings.Contains(se, "payload carries no session_id\n") {
		t.Fatalf("%d %q", code, se)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(sessions.StateDir))); err == nil {
		t.Fatal("a file was written all the same")
	}
}

// A payload with neither id names both.
func TestSubagentStartNamesBothMissingIDs(t *testing.T) {
	code, se := runSubagentStart(t, subagentRoot(t), `{}`)
	if code != ExitInternal || !strings.Contains(se, "payload carries no session_id and agent_id\n") {
		t.Fatalf("%d %q", code, se)
	}
}

func TestSubagentStartRecordsTheSnapshot(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	file, ok := agentFile(t, root)
	if !ok || file.Snapshot == nil {
		t.Fatalf("%+v %v", file, ok)
	}
	snap := *file.Snapshot
	refs := remoteRefsOf(t, root)
	if snap.Refs["HEAD"] != refs["HEAD"] || snap.Refs["refs/heads/master"] != refs["refs/heads/master"] {
		t.Fatalf("%+v against %v", snap, refs)
	}
	if snap.Heads == nil || snap.Heads["refs/heads/master"] != headOf(t, root) {
		t.Fatalf("%+v against head %s", snap, headOf(t, root))
	}
	if snap.Head != headOf(t, root) || snap.Remote != sessions.RemoteOK {
		t.Fatalf("%+v against head %s", snap, headOf(t, root))
	}
}

// A session directory that is a file is a snapshot nobody can file, and the
// hook says so rather than letting the stop hook compare against nothing.
func TestSubagentStartReportsAFileItCannotWrite(t *testing.T) {
	root := subagentRoot(t)
	writeWorldFile(t, root, sessions.StateDir+"/s1", "")
	code, se := runSubagentStart(t, root, subagentStartPayload)
	if code != ExitInternal || !strings.Contains(se, "loomux hook subagent-start: ") {
		t.Fatalf("%d %q", code, se)
	}
}

func TestSubagentStopWithoutSnapshotIsSilent(t *testing.T) {
	root := subagentRoot(t)
	code, se := runSubagentStop(t, root)
	if code != ExitOK || se != "" {
		t.Fatalf("%d %q", code, se)
	}
	if _, ok := agentFile(t, root); ok {
		t.Fatal("a file was written all the same")
	}
}

// A file that holds a parked finding and no snapshot belongs to a subagent
// whose start this hook never saw. There is nothing to compare against, the
// finding is not this run's to touch, and the hook says nothing -- least of
// all by comparing against a snapshot that is not there.
func TestSubagentStopWithAFindingAndNoSnapshotIsSilent(t *testing.T) {
	root := subagentRoot(t)
	parked := []string{"origin refs/heads/x is new at c"}
	if err := sessions.WriteAgent(root, "s1", "a1", sessions.AgentFile{Finding: parked}); err != nil {
		t.Fatal(err)
	}

	code, se := runSubagentStop(t, root)

	if code != ExitOK || se != "" {
		t.Fatalf("%d %q", code, se)
	}
	file, ok := agentFile(t, root)
	if !ok || !slices.Equal(file.Finding, parked) {
		t.Fatalf("%+v %v", file, ok)
	}
}

func TestSubagentStopWithNothingChangedRemovesTheFile(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	if code, se := runSubagentStop(t, root); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if _, ok := agentFile(t, root); ok {
		t.Fatal("the agent file is still there")
	}
}

func TestSubagentStopSeesAPush(t *testing.T) {
	root := subagentRoot(t)
	before, beforeRefs := headOf(t, root), remoteRefsOf(t, root)
	startedSubagent(t, root)
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "third")
	gitIn(t, root, "push", "-q", "origin", "HEAD:master")
	got := finding(t, root)
	after := headOf(t, root)
	wantFinding(t, got, []string{
		fmt.Sprintf("origin HEAD moved %s -> %s", beforeRefs["HEAD"], after),
		fmt.Sprintf("origin refs/heads/master moved %s -> %s", beforeRefs["refs/heads/master"], after),
		fmt.Sprintf("branch master moved %s -> %s", before, after),
		"new commit " + onelineOf(t, root, before, after),
	})
}

func TestSubagentStopSeesANewAndAGoneRef(t *testing.T) {
	root := subagentRoot(t)
	beforeRefs := remoteRefsOf(t, root)
	startedSubagent(t, root)
	gitIn(t, root, "push", "-q", "origin", "HEAD:refs/heads/feature")
	gitIn(t, root, "push", "-q", "origin", ":refs/heads/old")
	got := finding(t, root)
	wantFinding(t, got, []string{
		"origin refs/heads/feature is new at " + headOf(t, root),
		"origin refs/heads/old is gone; it was " + beforeRefs["refs/heads/old"],
	})
}

// A commit on a detached HEAD moves no branch at all, so HEAD's own range is
// the only account of it. Without that range the commit would go unreported.
func TestSubagentStopSeesACommitOffEveryBranch(t *testing.T) {
	root := subagentRoot(t)
	before := headOf(t, root)
	startedSubagent(t, root)
	gitIn(t, root, "switch", "-q", "--detach")
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "detached")

	got := finding(t, root)

	wantFinding(t, got, []string{"new commit " + onelineOf(t, root, before, headOf(t, root))})
}

// A branch moved to a commit HEAD never stood on: the ranges of the branches
// are what reports it, and `branch ... moved` alone would name the SHA
// without saying what arrived with it.
func TestSubagentStopSeesACommitOnABranchItIsNotOn(t *testing.T) {
	root := subagentRoot(t)
	gitIn(t, root, "branch", "side")
	before := localHeadsOf(t, root)["refs/heads/side"]
	startedSubagent(t, root)
	made := gitIn(t, root, "commit-tree", "HEAD^{tree}", "-p", "HEAD", "-m", "beside")
	gitIn(t, root, "branch", "-f", "side", made)

	got := finding(t, root)

	wantFinding(t, got, []string{
		fmt.Sprintf("branch side moved %s -> %s", before, made),
		"new commit " + onelineOf(t, root, before, made),
	})
}

// A branch that is gone has no range at all. `LogOneline` spells its range
// `from..to`, so an empty `to` would read as `from..HEAD` -- and a branch
// deleted from behind HEAD would report every commit it was behind by as
// news of this subagent.
func TestSubagentStopLogsNothingForABranchThatIsGone(t *testing.T) {
	root := subagentRoot(t)
	gitIn(t, root, "branch", "side", "HEAD~1")
	before := localHeadsOf(t, root)["refs/heads/side"]
	startedSubagent(t, root)
	gitIn(t, root, "branch", "-q", "-D", "side")

	got := finding(t, root)

	wantFinding(t, got, []string{"branch side is gone; it was " + before})
}

// The commit on `work` is no line of its own: a branch that was not there at
// the start has no range to log, and HEAD is back where it stood.
func TestSubagentStopSeesALocalBranch(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	gitIn(t, root, "switch", "-q", "-c", "work")
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "fourth")
	gitIn(t, root, "switch", "-q", "master")
	got := finding(t, root)
	wantFinding(t, got, []string{"branch work is new at " + localHeadsOf(t, root)["refs/heads/work"]})
}

func TestSubagentStopCountsACommitOnce(t *testing.T) {
	root := subagentRoot(t)
	before := headOf(t, root)
	startedSubagent(t, root)
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "third")
	got := finding(t, root)
	after := headOf(t, root)
	wantFinding(t, got, []string{
		fmt.Sprintf("branch master moved %s -> %s", before, after),
		"new commit " + onelineOf(t, root, before, after),
	})
}

func TestSubagentStopWithARemoteGoneAtStop(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	gitIn(t, root, "remote", "remove", "origin")
	wantFinding(t, finding(t, root), []string{"remote could not be read at stop"})
}

func TestSubagentStopWithARemoteMissingAtStart(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	editSnapshot(t, root, func(s *sessions.Snapshot) {
		s.Refs, s.Remote = nil, sessions.RemoteUnavailable
	})
	wantFinding(t, finding(t, root), []string{"remote could not be read at start"})
}

// A snapshot the Python hooks left knows no local branches, and a branch made
// while it ran is none of its business: nothing is parked at all.
func TestSubagentStopWithATranslatedSnapshot(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	editSnapshot(t, root, func(s *sessions.Snapshot) { s.Heads = nil })
	gitIn(t, root, "branch", "work")
	if lines := finding(t, root); lines != nil {
		t.Fatalf("finding %q", lines)
	}
}

// editSnapshot rewrites the filed snapshot through the package that wrote it,
// so the file stays the shape the hook reads.
func editSnapshot(t *testing.T, root string, edit func(*sessions.Snapshot)) {
	t.Helper()
	file, ok := agentFile(t, root)
	if !ok || file.Snapshot == nil {
		t.Fatalf("%+v %v", file, ok)
	}
	edit(file.Snapshot)
	if err := sessions.WriteAgent(root, "s1", "a1", file); err != nil {
		t.Fatal(err)
	}
}

// A subagent continued under the same id starts again before the main agent
// has read what it left; the new snapshot is filed beside that finding, not
// over it.
func TestSubagentStartKeepsAParkedFinding(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "third")
	parked := finding(t, root)
	if len(parked) == 0 {
		t.Fatal("nothing was parked")
	}
	startedSubagent(t, root)
	file, ok := agentFile(t, root)
	if !ok || file.Snapshot == nil || file.Snapshot.Head != headOf(t, root) {
		t.Fatalf("%+v %v", file, ok)
	}
	wantFinding(t, file.Finding, parked)
}

// parkedFinding runs one whole round of the same subagent -- start, a commit,
// stop -- and answers what it left for the main agent.
func parkedFinding(t *testing.T, root, message string) []string {
	t.Helper()
	startedSubagent(t, root)
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", message)
	lines := finding(t, root)
	if len(lines) == 0 {
		t.Fatal("nothing was parked")
	}
	return lines
}

// A second run that changed nothing still may not take the first run's lines
// with it: the main agent has not read them.
func TestSubagentStopKeepsACarriedFindingWithNothingChanged(t *testing.T) {
	root := subagentRoot(t)
	first := parkedFinding(t, root, "third")
	startedSubagent(t, root)
	wantFinding(t, finding(t, root), first)
}

func TestSubagentStopAddsToACarriedFinding(t *testing.T) {
	root := subagentRoot(t)
	first := parkedFinding(t, root, "third")
	startedSubagent(t, root)
	before := headOf(t, root)
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "fourth")
	got := finding(t, root)
	after := headOf(t, root)
	wantFinding(t, got, append(slices.Clone(first),
		fmt.Sprintf("branch master moved %s -> %s", before, after),
		"new commit "+onelineOf(t, root, before, after)))
}

func TestSubagentHooksWithAnUnknownHost(t *testing.T) {
	root := subagentRoot(t)
	var errOut strings.Builder
	if code := SubagentStart(strings.NewReader(subagentStartPayload), &errOut, root, "nope"); code != ExitInternal {
		t.Fatalf("start: %d %q", code, errOut.String())
	}
	if code := SubagentStop(strings.NewReader(subagentStopPayload), &errOut, root, "nope"); code != ExitInternal {
		t.Fatalf("stop: %d %q", code, errOut.String())
	}
	// The host is reported where it is parsed. Read from all the same, the
	// empty host would fail at the payload and the name nobody knows would
	// go unnamed.
	if !strings.Contains(errOut.String(), `"nope"`) {
		t.Fatalf("the message does not name the host: %q", errOut.String())
	}
}

func TestSubagentStopReportsAFileItCannotWrite(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "third")
	write := writeAgent
	t.Cleanup(func() { writeAgent = write })
	writeAgent = func(string, string, string, sessions.AgentFile) error { return errors.New("boom") }
	code, se := runSubagentStop(t, root)
	if code != ExitInternal || !strings.Contains(se, "loomux hook subagent-stop: boom") {
		t.Fatalf("%d %q", code, se)
	}
}

// otherWorktree adds a worktree of root's repository on a new branch and
// answers its directory. Local branches are shared by every worktree, so a
// session working there moves a branch root's snapshot also holds.
func otherWorktree(t *testing.T, root, branch string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "other")
	gitIn(t, root, "worktree", "add", "-q", "-b", branch, dir)
	return dir
}

// A concurrent session committing in its own worktree while the subagent ran
// is not what moved here: its branch gets no line and no commit range. What
// moved in this worktree still does.
func TestSubagentStopLeavesOutTheBranchOfAnotherWorktree(t *testing.T) {
	root := subagentRoot(t)
	other := otherWorktree(t, root, "other")
	before := headOf(t, root)
	startedSubagent(t, root)
	gitIn(t, other, "commit", "--allow-empty", "-q", "-m", "elsewhere")
	gitIn(t, root, "commit", "--allow-empty", "-q", "-m", "here")

	got := finding(t, root)

	after := headOf(t, root)
	wantFinding(t, got, []string{
		fmt.Sprintf("branch master moved %s -> %s", before, after),
		"new commit " + onelineOf(t, root, before, after),
	})
}

// Checked out elsewhere at the start is enough: a worktree removed before
// the stop leaves its branch checked out nowhere, and the move it made while
// it existed is still not this worktree's.
func TestSubagentStopLeavesOutABranchCheckedOutElsewhereAtStart(t *testing.T) {
	root := subagentRoot(t)
	other := otherWorktree(t, root, "other")
	startedSubagent(t, root)
	gitIn(t, other, "commit", "--allow-empty", "-q", "-m", "elsewhere")
	gitIn(t, root, "worktree", "remove", "--force", other)

	if lines := finding(t, root); lines != nil {
		t.Fatalf("finding %q", lines)
	}
}

// And checked out elsewhere at the stop is enough too: a worktree another
// session adds while the subagent runs brings a branch that is new here only
// by the calendar.
func TestSubagentStopLeavesOutABranchCheckedOutElsewhereAtStop(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	other := otherWorktree(t, root, "other")
	gitIn(t, other, "commit", "--allow-empty", "-q", "-m", "elsewhere")

	if lines := finding(t, root); lines != nil {
		t.Fatalf("finding %q", lines)
	}
}

// A branch no worktree has checked out stays in, beside one that is:
// the filter takes out what belongs to another worktree, nothing more.
func TestSubagentStopReportsABranchCheckedOutNowhere(t *testing.T) {
	root := subagentRoot(t)
	otherWorktree(t, root, "other")
	gitIn(t, root, "branch", "side", "HEAD~1")
	before := localHeadsOf(t, root)["refs/heads/side"]
	startedSubagent(t, root)
	gitIn(t, root, "branch", "-f", "side", "master")

	got := finding(t, root)

	after := localHeadsOf(t, root)["refs/heads/side"]
	wantFinding(t, got, []string{
		fmt.Sprintf("branch side moved %s -> %s", before, after),
		"new commit " + onelineOf(t, root, before, after),
	})
}

// A repository that never had an origin has nothing a subagent could push to:
// no line, no file, no turn held for it.
func TestSubagentStopWithoutAnyOriginIsSilent(t *testing.T) {
	root := subagentRoot(t)
	gitIn(t, root, "remote", "remove", "origin")
	startedSubagent(t, root)
	if file, _ := agentFile(t, root); file.Snapshot == nil || file.Snapshot.Remote != sessions.RemoteNone {
		t.Fatalf("%+v", file.Snapshot)
	}
	wantFinding(t, finding(t, root), nil)
}

// An origin that appears while the subagent runs is news, but its refs
// cannot be compared against a start that had none.
func TestSubagentStopWithAnOriginAddedWhileItRan(t *testing.T) {
	root := subagentRoot(t)
	url := strings.TrimSpace(gitIn(t, root, "remote", "get-url", "origin"))
	gitIn(t, root, "remote", "remove", "origin")
	startedSubagent(t, root)
	gitIn(t, root, "remote", "add", "origin", url)
	wantFinding(t, finding(t, root), []string{"remote could not be read at start"})
}

// An origin that is configured and does not answer is the other case: it may
// hide a push, and the finding says the remote could not be read.
func TestSubagentStopWithAnOriginThatDoesNotAnswer(t *testing.T) {
	root := subagentRoot(t)
	startedSubagent(t, root)
	gitIn(t, root, "remote", "set-url", "origin", filepath.Join(t.TempDir(), "nowhere"))
	file := finding(t, root)
	wantFinding(t, file, []string{"remote could not be read at stop"})
}
