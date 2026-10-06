package hooks

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/verify"
)

// graphWorld is gitWorld with a graph built over it.
func graphWorld(t *testing.T, decl, state string) string {
	t.Helper()
	root := gitWorld(t, decl, state)
	buildGraph(t, root)
	return root
}

// fakeWiring stands a graph file where a test needs only that one exists.
func fakeWiring(t *testing.T, root string) {
	t.Helper()
	writeWorldFile(t, root, filepath.ToSlash(mustRel(t, root, store.WiringPath(root))), "{}")
}

func mustRel(t *testing.T, root, path string) string {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		t.Fatal(err)
	}
	return rel
}

// gitRead runs git in root apart from the machine's own configuration, so no
// global hook or setting shapes a world, with env added. It fails with an
// error, not the test: lane callbacks run off the test's goroutine.
func gitRead(root string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-c", "core.hooksPath=" + os.DevNull}, args...)...)
	cmd.Dir = root
	cmd.Env = append(append(gitenv.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull), env...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitOut(t *testing.T, root string, env []string, args ...string) string {
	t.Helper()
	out, err := gitRead(root, env, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// listCopies lists the kept copies of the index in the git directory of
// root, by the name gitwork gives them.
func listCopies(root string) ([]string, error) {
	dir, err := gitRead(root, nil, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), gitwork.KeptIndexPrefix) {
			found = append(found, e.Name())
		}
	}
	return found, nil
}

func copiesIn(t *testing.T, root string) []string {
	t.Helper()
	found, err := listCopies(root)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// graphKinds is a stop profile that asks for the graph lane.
var graphKinds = []string{"lint", "graph"}

// A graph gone between the copy and the probe: the probe asks again.
func TestStopIndexWithoutAGraph(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"blocks":0}`)
	fakeWiring(t, root)
	idx, err := OpenStopIndex(root, graphKinds)
	if err != nil || idx == nil {
		t.Fatalf("%v %v", idx, err)
	}
	defer idx.Close()
	if err := os.Remove(store.WiringPath(root)); err != nil {
		t.Fatal(err)
	}
	if ok, note := idx.Ready(root); ok || note != "no graph at .loomux/state/graph/wiring.json" {
		t.Fatalf("%v %q", ok, note)
	}
}

func TestStopIndexWithoutHead(t *testing.T) {
	root := gitWorld(t, unbornWorld, `{"blocks":0}`)
	fakeWiring(t, root)
	idx, err := OpenStopIndex(root, graphKinds)
	if err != nil || idx != nil {
		t.Fatalf("%v %v", idx, err)
	}
	if ok, note := idx.Ready(root); ok || !strings.HasPrefix(note, "no HEAD to compare with") {
		t.Fatalf("%v %q", ok, note)
	}
	if env := idx.Env(root); env != nil {
		t.Fatalf("%v", env)
	}
	idx.Close()
}

// With a graph but no view there is no HEAD to judge against: OpenStopIndex
// answers nil without one.
func TestStopIndexNilPastThePrereq(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"blocks":0}`)
	if ok, note := (*StopIndex)(nil).Ready(root); ok || note != "no HEAD to compare with" {
		t.Fatalf("%v %q", ok, note)
	}
}

func TestStopIndexOutsideARepository(t *testing.T) {
	root := gitWorld(t, "", `{"blocks":0}`)
	fakeWiring(t, root)
	if idx, err := OpenStopIndex(root, graphKinds); err != nil || idx != nil {
		t.Fatalf("%v %v", idx, err)
	}
}

func TestStopIndexDuringAnOperation(t *testing.T) {
	for _, m := range []struct{ path, note string }{
		{"MERGE_HEAD", "a merge is in progress"},
		{"rebase-merge", "a rebase is in progress"},
		{"rebase-apply", "a rebase is in progress"},
		{"CHERRY_PICK_HEAD", "a cherry-pick is in progress"},
		{"REVERT_HEAD", "a revert is in progress"},
	} {
		t.Run(m.path, func(t *testing.T) {
			root := graphWorld(t, stopWorld, `{"blocks":0}`)
			writeWorldFile(t, root, ".git/"+m.path, "x\n")
			idx, err := OpenStopIndex(root, graphKinds)
			if err != nil {
				t.Fatal(err)
			}
			defer idx.Close()
			if ok, note := idx.Ready(root); ok || note != m.note {
				t.Fatalf("%v %q", ok, note)
			}
		})
	}
}

func TestStopIndexWithNothingChanged(t *testing.T) {
	root := graphWorld(t, twoCommitsOnly, `{"blocks":0}`)
	idx, err := OpenStopIndex(root, graphKinds)
	if err != nil {
		t.Fatal(err)
	}
	defer idx.Close()
	if ok, note := idx.Ready(root); ok || note != "nothing changed against HEAD" {
		t.Fatalf("%v %q", ok, note)
	}
}

func TestStopIndexWithAChange(t *testing.T) {
	root := graphWorld(t, twoCommitsOnly, `{"blocks":0}`)
	writeWorldFile(t, root, "b_test.go", "package a\n")
	idx, err := OpenStopIndex(root, graphKinds)
	if err != nil {
		t.Fatal(err)
	}
	if ok, note := idx.Ready(root); !ok || note != "" {
		t.Fatalf("%v %q", ok, note)
	}
	env := idx.Env(root)
	if !slices.Equal(env, []string{"GIT_INDEX_FILE=" + idx.Index}) {
		t.Fatalf("%v", env)
	}
	if files := gitOut(t, root, env, "ls-files"); !strings.Contains(files, "b_test.go") {
		t.Fatalf("the copy lacks the untracked test: %q", files)
	}
	idx.Close()
	if _, err := os.Stat(idx.Index); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the copy stayed: %v", err)
	}
}

func TestStopIndexReportsAGitFailure(t *testing.T) {
	kept := stopKeptTree
	t.Cleanup(func() { stopKeptTree = kept })
	stopKeptTree = func(string) (string, string, error) { return "", "", errors.New("boom") }
	root := gitWorld(t, stopWorld, `{"blocks":0}`)
	fakeWiring(t, root)
	if idx, err := OpenStopIndex(root, graphKinds); err == nil || idx != nil {
		t.Fatalf("%v %v", idx, err)
	}
}

// Only a profile with the graph kind in a project with a graph keeps a copy;
// any other turn end pays nothing for one.
func TestOpenStopIndexOnlyForAGraphLane(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"blocks":0}`)
	if idx, err := OpenStopIndex(root, graphKinds); err != nil || idx != nil {
		t.Fatalf("kept without a graph: %v %v", idx, err)
	}
	fakeWiring(t, root)
	if idx, err := OpenStopIndex(root, []string{"lint", "test"}); err != nil || idx != nil {
		t.Fatalf("kept without the graph kind: %v %v", idx, err)
	}
	idx, err := OpenStopIndex(root, graphKinds)
	if err != nil || idx == nil {
		t.Fatalf("not kept with a graph and the kind: %v %v", idx, err)
	}
	idx.Close()
}

// blastTools answers every lane 0 and blast-audit with code and out; each
// start of blast-audit is handed to seen under a lock, since lanes run in
// parallel.
func blastTools(code int, out string, seen func(child.Spec)) StopEnv {
	env := greenTools()
	var mu sync.Mutex
	inner := env.Start
	env.Start = func(s child.Spec) child.Result {
		if slices.Contains(s.Argv, "blast-audit") {
			mu.Lock()
			defer mu.Unlock()
			if seen != nil {
				seen(s)
			}
			return child.Result{Code: code, Stdout: out}
		}
		return inner(s)
	}
	return env
}

func indexOf(s child.Spec) string {
	for _, e := range s.Env {
		if v, ok := strings.CutPrefix(e, "GIT_INDEX_FILE="); ok {
			return v
		}
	}
	return ""
}

func TestStopJudgesAnUntrackedTestThroughTheCopy(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, "b_test.go", "package a\n")
	var files, index string
	var standing []string
	var failed error
	env := blastTools(0, "", func(s child.Spec) {
		index = indexOf(s)
		var err, listErr error
		files, err = gitRead(root, []string{"GIT_INDEX_FILE=" + index}, "ls-files")
		standing, listErr = listCopies(root)
		failed = errors.Join(err, listErr)
	})
	code, se := runStop(t, root, s1, env)
	if code != ExitOK || failed != nil {
		t.Fatalf("%d %q %v", code, se, failed)
	}
	if index == "" || !strings.Contains(files, "b_test.go") {
		t.Fatalf("blast-audit saw index %q holding %q", index, files)
	}
	// The leak checks below look for this name; they prove nothing unless
	// they find the copy while it stands.
	if len(standing) != 1 || standing[0] != filepath.Base(index) {
		t.Fatalf("copies %v while the lane ran on %q", standing, index)
	}
	if state := stateOf(t, root); state.Base != headOf(t, root) {
		t.Fatalf("%+v", state)
	}
	if left := copiesIn(t, root); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
}

func TestStopHoldsARedBlastAudit(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	started := false
	code, se := runStop(t, root, s1, blastTools(1, "A has 7 callers and no test\n", func(child.Spec) { started = true }))
	if code != ExitDenied || !started || !strings.Contains(se, "A has 7 callers and no test") {
		t.Fatalf("%d %v %q", code, started, se)
	}
	if state := stateOf(t, root); state.Blocks != 1 {
		t.Fatalf("%+v", state)
	}
	if left := copiesIn(t, root); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
	// The next turn end, green, leaves nothing behind either.
	if code, se := runStop(t, root, s1, blastTools(0, "", nil)); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if code, se := runStop(t, root, s1, blastTools(1, "", nil)); code != ExitOK {
		t.Fatalf("a green tree ran again: %d %q", code, se)
	}
	if left := copiesIn(t, root); len(left) != 0 {
		t.Fatalf("left behind by a green tree: %v", left)
	}
}

// A commit inside the turn leaves the content tree as it was and moves HEAD,
// and the graph lane judges against HEAD: committing only the test takes it
// out of what the lane sees beside the changed code. The tree found green
// before is no answer then.
func TestStopRunsAGreenTreeAgainOnceHeadMoved(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, "b_test.go", "package a\n")
	if code, se := runStop(t, root, s1, blastTools(0, "", nil)); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	gitOut(t, root, nil, "add", "b_test.go")
	gitOut(t, root, nil, "commit", "-q", "-m", "the test alone")
	started := false
	code, se := runStop(t, root, s1, blastTools(1, "A has 7 callers and no test\n", func(child.Spec) { started = true }))
	if code != ExitDenied || !started {
		t.Fatalf("a green tree under a new HEAD passed unrun: %d %v %q", code, started, se)
	}
}

// committedWorld holds everything gitWorld writes in its one commit, so the
// tree at a turn end is HEAD's; the config stays out of that world.
const committedWorld = `
[[commit]]
message = "base"
paths = ["go.mod", "a.go", "a_test.go"]
`

// A session without a base measures from HEAD, and HEAD cannot have moved
// away from itself: with nothing new the turn ends unrun, graph lane or not.
func TestStopWithoutABaseAndNothingNewRunsNothing(t *testing.T) {
	root := gitWorld(t, committedWorld, `{"blocks":0}`, ".loomux/config.toml")
	fakeWiring(t, root)
	jobs := planned(t)
	if code, se := runStop(t, root, s1, blastTools(1, "", nil)); code != ExitOK || *jobs != nil {
		t.Fatalf("the chain ran on nothing new: %d %q %v", code, se, *jobs)
	}
}

// Without a graph lane nothing judges against HEAD, and a green tree stays
// green under a new one.
func TestStopKeepsAGreenTreeUnderANewHeadWithoutAGraph(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, "b_test.go", "package a\n")
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	gitOut(t, root, nil, "add", "b_test.go")
	gitOut(t, root, nil, "commit", "-q", "-m", "the test alone")
	jobs := planned(t)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK || *jobs != nil {
		t.Fatalf("a green tree ran again: %d %q %v", code, se, *jobs)
	}
}

// planned wraps stopPlan so a test can read the graph job it made.
func planned(t *testing.T) *[]verify.Job {
	t.Helper()
	plan := stopPlan
	t.Cleanup(func() { stopPlan = plan })
	var jobs []verify.Job
	stopPlan = func(eff verify.Effective, req verify.Request, env verify.PlanEnv) ([]verify.Job, error) {
		out, err := plan(eff, req, env)
		jobs = out
		return out, err
	}
	return &jobs
}

func graphJob(t *testing.T, jobs []verify.Job) verify.Job {
	t.Helper()
	for _, j := range jobs {
		if j.Kind == "graph" {
			return j
		}
	}
	t.Fatalf("no graph job in %v", jobs)
	return verify.Job{}
}

func TestStopWithoutAGraphPasses(t *testing.T) {
	jobs := planned(t)
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	started := false
	code, se := runStop(t, root, s1, blastTools(1, "", func(child.Spec) { started = true }))
	if code != ExitOK || started {
		t.Fatalf("%d %v %q", code, started, se)
	}
	if j := graphJob(t, *jobs); j.Name != "graph/go" || j.Pre != verify.StateNotApplicable || !strings.Contains(j.Note, "no graph") {
		t.Fatalf("%+v", j)
	}
	if state := stateOf(t, root); state.Base != headOf(t, root) {
		t.Fatalf("%+v", state)
	}
}

// A graph in an unborn repository has no HEAD to judge against: the tree is
// written without a kept copy, and the lane says why it did not run.
func TestStopWithAGraphAndNoHead(t *testing.T) {
	jobs := planned(t)
	root := gitWorld(t, unbornWorld, `{"blocks":0}`)
	fakeWiring(t, root)
	if code, se := runStop(t, root, s1, blastTools(1, "", nil)); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if j := graphJob(t, *jobs); j.Pre != verify.StateNotApplicable || !strings.HasPrefix(j.Note, "no HEAD to compare with") {
		t.Fatalf("%+v", j)
	}
	if left := copiesIn(t, root); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
}

const pythonWorld = `
[[commit]]
message = "base"
[commit.files]
"pyproject.toml" = "[project]\nname = \"a\"\n"
"a.py" = "A = 1\n"
"tests/test_a.py" = "def test_a():\n    pass\n"

[worktree]
"a.py" = "A = 2\n"
`

// Python carries the graph lane as Go does, so a blast finding there holds
// the turn too.
func TestStopHoldsOnAPythonBlast(t *testing.T) {
	root := gitWorld(t, pythonWorld, `{"base":"{{COMMIT:1}}","blocks":0}`, "go.mod", "a.go", "a_test.go")
	fakeWiring(t, root)
	code, se := runStop(t, root, s1, blastTools(1, "A has 7 callers and no test\n", nil))
	if code != ExitDenied || !strings.Contains(se, "graph/python") || !strings.Contains(se, "A has 7 callers and no test") {
		t.Fatalf("%d %q", code, se)
	}
}

const typescriptWorld = `
[[commit]]
message = "base"
[commit.files]
"package.json" = "{\"devDependencies\": {\"vitest\": \"1\"}}\n"
"tsconfig.json" = "{}\n"
"a.ts" = "export const A = 1\n"

[worktree]
"a.ts" = "export const A = 2\n"
`

// A stack without a graph table has a graph lane with no command:
// not-applicable beside lanes that ran, so the turn ends and the base moves.
func TestStopWithoutAGraphLanePasses(t *testing.T) {
	jobs := planned(t)
	root := gitWorld(t, typescriptWorld, `{"base":"{{COMMIT:1}}","blocks":0}`, "go.mod", "a.go", "a_test.go")
	fakeWiring(t, root)
	code, se := runStop(t, root, s1, blastTools(1, "", nil))
	if code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if j := graphJob(t, *jobs); j.Pre != verify.StateNotApplicable || j.Note != "no command" {
		t.Fatalf("%+v", j)
	}
	if state := stateOf(t, root); state.Base != headOf(t, root) {
		t.Fatalf("%+v", state)
	}
}

func TestStopLeavesNoCopyAfterAPlanError(t *testing.T) {
	plan := stopPlan
	t.Cleanup(func() { stopPlan = plan })
	stopPlan = func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) {
		return nil, errors.New("boom")
	}
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitInternal {
		t.Fatalf("%d %q", code, se)
	}
	if left := copiesIn(t, root); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
}

func TestStopLeavesNoCopyAfterACoverError(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/state/cover", "")
	if code, se := runStop(t, root, s1, greenTools()); code != ExitInternal {
		t.Fatalf("%d %q", code, se)
	}
	if left := copiesIn(t, root); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
}

func TestStopHoldsAFailureToKeepTheCopy(t *testing.T) {
	kept := stopKeptTree
	t.Cleanup(func() { stopKeptTree = kept })
	stopKeptTree = func(string) (string, string, error) { return "", "", errors.New("boom") }
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	code, se := runStop(t, root, s1, greenTools())
	if code != ExitDenied || !strings.Contains(se, "boom") {
		t.Fatalf("%d %q", code, se)
	}
	if state := stateOf(t, root); state.Blocks != 1 {
		t.Fatalf("%+v", state)
	}
}

func TestStopProfileWithoutGraphKeepsNoCopy(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify.profiles]\nstop = [\"lint\"]\n")
	var mu sync.Mutex
	var seen []string
	env := greenTools()
	inner := env.Start
	var failed error
	env.Start = func(s child.Spec) child.Result {
		left, err := listCopies(root)
		mu.Lock()
		seen = append(seen, left...)
		failed = errors.Join(failed, err)
		mu.Unlock()
		return inner(s)
	}
	if code, se := runStop(t, root, s1, env); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	if failed != nil || len(seen) != 0 {
		t.Fatalf("a copy stood while the lanes ran: %v %v", seen, failed)
	}
}

func TestStopKeepsTheCopyInALinkedWorktree(t *testing.T) {
	main := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	wt := filepath.Join(t.TempDir(), "wt")
	gitOut(t, main, nil, "worktree", "add", "-q", wt)
	writeWorldFile(t, wt, ".loomux/config.toml", "[verify.go]\ncoverage = false\n")
	writeWorldFile(t, wt, "b_test.go", "package a\n")
	buildGraph(t, wt)
	var index string
	if code, se := runStop(t, wt, s1, blastTools(0, "", func(s child.Spec) { index = indexOf(s) })); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	want, err := os.Stat(filepath.Join(main, ".git", "worktrees", "wt"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.Stat(filepath.Dir(index))
	if err != nil || !os.SameFile(got, want) {
		t.Fatalf("the copy %q is not in the worktree's git directory: %v", index, err)
	}
}

func TestStopReportsABadConfigOnAGreenTree(t *testing.T) {
	root := gitWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	if code, se := runStop(t, root, s1, greenTools()); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	writeWorldFile(t, root, ".loomux/config.toml", "[verify]\nnope = 1\n")
	if code, se := runStop(t, root, s1, greenTools()); code != ExitInternal {
		t.Fatalf("%d %q", code, se)
	}
}

// A commit inside the turn changes what the graph lane judges against HEAD
// while the tree diff from the green tree shows only the docs written after
// it: the lane may not sit out on that list alone.
func TestStopDoesNotSkipTheGraphLaneForACommitInsideTheTurn(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify.go]\ncoverage = false\n[verify.go.graph]\nskip_when_only = [\"docs/**\"]\n")
	writeWorldFile(t, root, "b_test.go", "package a\n")
	if code, se := runStop(t, root, s1, blastTools(0, "", nil)); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	gitOut(t, root, nil, "add", "b_test.go")
	gitOut(t, root, nil, "commit", "-q", "-m", "the test alone")
	writeWorldFile(t, root, "docs/a.md", "x")
	started := false
	code, se := runStop(t, root, s1, blastTools(1, "A has 7 callers and no test\n", func(child.Spec) { started = true }))
	if code != ExitDenied || !started {
		t.Fatalf("the graph lane sat out although HEAD moved: %d %v %q", code, started, se)
	}
}

// What a commit took is added to what the trees differ in, not put in its
// place: docs committed beside code that is still being changed.
func TestStopKeepsWhatTheTreesShowWhenACommitMovedHead(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify.go]\ncoverage = false\n[verify.go.graph]\nskip_when_only = [\"docs/**\"]\n")
	writeWorldFile(t, root, "b_test.go", "package a\n")
	if code, se := runStop(t, root, s1, blastTools(0, "", nil)); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	writeWorldFile(t, root, "docs/b.md", "x")
	gitOut(t, root, nil, "add", "docs/b.md")
	gitOut(t, root, nil, "commit", "-q", "-m", "docs")
	writeWorldFile(t, root, "a.go", "package a\n\nfunc A() int { return 444 }\n")
	started := false
	code, se := runStop(t, root, s1, blastTools(1, "A has 7 callers and no test\n", func(child.Spec) { started = true }))
	if code != ExitDenied || !started {
		t.Fatalf("the graph lane sat out beside code changed since the green tree: %d %v %q", code, started, se)
	}
}

// A commit that takes everything leaves the tree and HEAD alike, and the docs
// written after it are all the trees differ in: what the commit took, measured
// from the base, still keeps the lane from sitting out.
func TestStopCountsEverythingACommitTookWhenHeadMoved(t *testing.T) {
	root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
	writeWorldFile(t, root, ".loomux/config.toml", "[verify.go]\ncoverage = false\n[verify.go.graph]\nskip_when_only = [\"docs/**\"]\n")
	writeWorldFile(t, root, "b_test.go", "package a\n")
	if code, se := runStop(t, root, s1, blastTools(0, "", nil)); code != ExitOK {
		t.Fatalf("%d %q", code, se)
	}
	gitOut(t, root, nil, "add", "-A")
	gitOut(t, root, nil, "commit", "-q", "-m", "everything")
	writeWorldFile(t, root, "docs/a.md", "x")
	started := false
	code, se := runStop(t, root, s1, blastTools(1, "A has 7 callers and no test\n", func(child.Spec) { started = true }))
	if code != ExitDenied || !started {
		t.Fatalf("the graph lane sat out after a commit of code: %d %v %q", code, started, se)
	}
}

// Where git cannot list one of the two ranges, there is no list, and the lane
// runs.
func TestStopRunsTheGraphLaneWhenGitCannotListAMovedHead(t *testing.T) {
	for failing := 1; failing <= 2; failing++ {
		root := graphWorld(t, stopWorld, `{"base":"{{COMMIT:1}}","blocks":0}`)
		writeWorldFile(t, root, ".loomux/config.toml", "[verify.go]\ncoverage = false\n[verify.go.graph]\nskip_when_only = [\"docs/**\"]\n")
		writeWorldFile(t, root, "b_test.go", "package a\n")
		if code, se := runStop(t, root, s1, blastTools(0, "", nil)); code != ExitOK {
			t.Fatalf("%d %q", code, se)
		}
		gitOut(t, root, nil, "add", "b_test.go")
		gitOut(t, root, nil, "commit", "-q", "-m", "the test alone")
		writeWorldFile(t, root, "docs/a.md", "x")
		inner, calls := stopChanged, 0
		stopChanged = func(root, from, to string) ([]string, error) {
			if calls++; calls == failing {
				return nil, errors.New("git failed")
			}
			return inner(root, from, to)
		}
		started := false
		code, se := runStop(t, root, s1, blastTools(1, "A has 7 callers and no test\n", func(child.Spec) { started = true }))
		stopChanged = inner
		if code != ExitDenied || !started || calls != failing {
			t.Errorf("call %d failed: %d %v %q, %d calls", failing, code, started, se, calls)
		}
	}
}
