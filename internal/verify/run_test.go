package verify

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
)

type fakeStart struct {
	mu            sync.Mutex
	started       []string
	answer        func(child.Spec) child.Result
	running, peak int
}

func (f *fakeStart) start(s child.Spec) child.Result {
	f.mu.Lock()
	f.started = append(f.started, strings.Join(s.Argv, " "))
	f.running++
	if f.running > f.peak {
		f.peak = f.running
	}
	f.mu.Unlock()
	r := f.answer(s)
	f.mu.Lock()
	f.running--
	f.mu.Unlock()
	return r
}

func ok(child.Spec) child.Result { return child.Result{Code: 0} }

func opts(f *fakeStart) RunOptions {
	return RunOptions{Scope: ScopeCheck, MaxParallel: 4, Timeout: time.Minute, Start: f.start,
		Look: func(s string) (string, error) { return s, nil }, Now: time.Now}
}

func job(name string, after int, argvs ...string) Job {
	j := Job{Name: name, After: after}
	for _, a := range argvs {
		j.Argvs = append(j.Argvs, strings.Fields(a))
	}
	return j
}

// clock is a Now that moves an hour on every call, so a budget of minutes is
// spent the moment anyone looks at the time again.
type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(time.Hour)
	return c.t
}

func TestRunBlocksTheSuccessorOfARedLane(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "t" {
			return child.Result{Code: 1, Stdout: "boom\n"}
		}
		return child.Result{}
	}}
	out := Run([]Job{job("test/go", -1, "t"), job("coverage/go", 0, "c")}, opts(f))
	if out[0].State != StateFailed || out[1].State != StateBlocked || out[1].BlockedBy != "test/go" {
		t.Fatalf("%+v", out)
	}
	if len(f.started) != 1 {
		t.Fatalf("coverage must not start: %v", f.started)
	}
}

func TestRunInheritsAPredecessorThatCouldNotRun(t *testing.T) {
	pre := job("test/go", -1)
	pre.Pre = StateUnavailable
	out := Run([]Job{pre, job("coverage/go", 0, "c")}, opts(&fakeStart{answer: ok}))
	if out[1].State != StateUnavailable {
		t.Fatalf("%+v", out[1])
	}
}

func TestRunNamesAMissingTool(t *testing.T) {
	o := opts(&fakeStart{answer: ok})
	o.Look = func(string) (string, error) { return "", errors.New("not found") }
	out := Run([]Job{job("lint/python", -1, "ruff check .")}, o)
	if out[0].State != StateMissingTool || !strings.Contains(out[0].Output, `"ruff" is not on PATH`) {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunCapsProcessesAtMaxParallel(t *testing.T) {
	f := &fakeStart{answer: func(child.Spec) child.Result { time.Sleep(20 * time.Millisecond); return child.Result{} }}
	o := opts(f)
	o.MaxParallel = 2
	jobs := []Job{}
	for i := 0; i < 6; i++ {
		jobs = append(jobs, job(fmt.Sprintf("lint/s%d", i), -1, "x"))
	}
	Run(jobs, o)
	if f.peak > 2 {
		t.Fatalf("peak %d", f.peak)
	}
}

func TestRunRunsEveryCommandAndBlocksTheOutput(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "b" {
			return child.Result{Code: 1, Stdout: "bad\n"}
		}
		return child.Result{Stdout: "fine\n"}
	}}
	j := job("lint/go", -1, "a 1", "b 2", "c 3")
	out := Run([]Job{j}, opts(f))
	want := "$ a 1\nfine\n\n$ b 2 (failed)\nbad\n\n$ c 3\nfine\n"
	if out[0].State != StateFailed || out[0].Output != want || len(f.started) != 3 {
		t.Fatalf("%q %v", out[0].Output, f.started)
	}
}

func TestRunTurnsABudgetDeadlineIntoBudget(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Timeout >= time.Minute {
			t.Errorf("the budget must cap the timeout, got %v", s.Timeout)
		}
		return child.Result{Code: -1, TimedOut: true}
	}}
	o := opts(f)
	o.Scope, o.Budget = ScopeEdit, 50*time.Millisecond
	out := Run([]Job{job("lint/go", -1, "go vet ./...")}, o)
	if out[0].State != StateBudget {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunGivesAPlannedStateAndItsNote(t *testing.T) {
	j := job("coverage/go", -1, "c")
	j.Pre, j.Note = StateFailed, "no measure step"
	f := &fakeStart{answer: ok}
	out := Run([]Job{j, job("x/go", 0, "x")}, opts(f))
	if out[0].State != StateFailed || out[0].Output != "no measure step" || out[1].State != StateBlocked {
		t.Fatalf("%+v", out)
	}
	if len(f.started) != 0 {
		t.Fatalf("%v", f.started)
	}
}

func TestRunCallsAnInProcessLaneWithoutStartingAnything(t *testing.T) {
	o := RunOptions{Scope: ScopeCheck, MaxParallel: 1, Now: time.Now,
		Start: func(child.Spec) child.Result { t.Error("Start called"); return child.Result{} },
		Look:  func(string) (string, error) { t.Error("Look called"); return "", nil }}
	green := Job{Name: "a", After: -1, Fn: func() (string, error) { return "all good\n", nil }}
	red := Job{Name: "b", After: -1, Fn: func() (string, error) { return "found\n", errors.New("two findings") }}
	bare := Job{Name: "c", After: -1, Fn: func() (string, error) { return "", errors.New("broken") }}
	open := Job{Name: "d", After: -1, Fn: func() (string, error) { return "half", errors.New("cut") }}
	out := Run([]Job{green, red, bare, open}, o)
	if out[3].Output != "half\ncut\n" {
		t.Fatalf("%q", out[3].Output)
	}
	if out[0].State != StateOK || out[0].Output != "all good\n" {
		t.Fatalf("%+v", out[0])
	}
	if out[1].State != StateFailed || out[1].Output != "found\ntwo findings\n" {
		t.Fatalf("%q", out[1].Output)
	}
	if out[2].State != StateFailed || out[2].Output != "broken\n" {
		t.Fatalf("%q", out[2].Output)
	}
}

func TestRunEndsTheLaneOnARedMeasure(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "measure" {
			return child.Result{Code: 2, Stderr: "no tests\n"}
		}
		return child.Result{}
	}}
	j := job("coverage/go", -1, "report")
	j.Measure = []string{"measure", "it"}
	out := Run([]Job{j}, opts(f))
	if out[0].State != StateFailed || out[0].Output != "no tests\n" || len(f.started) != 1 {
		t.Fatalf("%+v %v", out[0], f.started)
	}
}

func TestRunMeasuresThenReports(t *testing.T) {
	dir := t.TempDir()
	f := &fakeStart{answer: func(s child.Spec) child.Result { return child.Result{Stdout: "measured\n"} }}
	j := job("coverage/go", -1, "report")
	j.Measure, j.Reads = []string{"measure"}, []string{dir}
	out := Run([]Job{j}, opts(f))
	if out[0].State != StateOK || out[0].Output != "measured\n" || strings.Join(f.started, ";") != "measure;report" {
		t.Fatalf("%+v %v", out[0], f.started)
	}
}

func TestRunRefusesALaneWhoseCoverageFileIsMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "cover.out")
	f := &fakeStart{answer: ok}
	j := job("coverage/go", -1, "report")
	j.Reads = []string{missing}
	out := Run([]Job{j}, opts(f))
	want := missing + " is missing: the measuring run did not write it"
	if out[0].State != StateFailed || out[0].Output != want || len(f.started) != 0 {
		t.Fatalf("%+v %v", out[0], f.started)
	}
}

func TestRunStartsEveryThreadedCommand(t *testing.T) {
	var mu sync.Mutex
	arrived := 0
	release := make(chan struct{})
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		mu.Lock()
		arrived++
		if arrived == 3 {
			close(release)
		}
		mu.Unlock()
		<-release
		return child.Result{Stdout: s.Argv[0] + "\n"}
	}}
	j := job("lint/go", -1, "a", "b", "c")
	j.Threaded = true
	out := Run([]Job{j}, opts(f))
	if out[0].State != StateOK || out[0].Output != "$ a\na\n\n$ b\nb\n\n$ c\nc\n" || f.peak != 3 {
		t.Fatalf("%q peak %d", out[0].Output, f.peak)
	}
}

func TestRunTimesOutOnItsOwnDeadline(t *testing.T) {
	f := &fakeStart{answer: func(child.Spec) child.Result { return child.Result{Code: -1, TimedOut: true} }}
	o := opts(f)
	o.Budget = time.Hour
	out := Run([]Job{job("test/go", -1, "t")}, o)
	if out[0].State != StateTimedOut {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunGivesTheRestWhenThereIsNoTimeout(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Timeout <= 0 || s.Timeout > time.Hour {
			t.Errorf("timeout %v", s.Timeout)
		}
		return child.Result{Code: -1, TimedOut: true}
	}}
	o := opts(f)
	o.Timeout, o.Budget = 0, time.Hour
	if out := Run([]Job{job("test/go", -1, "t")}, o); out[0].State != StateBudget {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunReportsAToolThatCouldNotStart(t *testing.T) {
	f := &fakeStart{answer: func(child.Spec) child.Result {
		return child.Result{Code: -1, Err: errors.New("exec: access denied")}
	}}
	out := Run([]Job{job("lint/go", -1, "x")}, opts(f))
	if out[0].State != StateFailed || out[0].Output != "exec: access denied" {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunNamesAnAbandonedOutput(t *testing.T) {
	f := &fakeStart{answer: func(child.Spec) child.Result {
		return child.Result{Code: 0, Stdout: "partial\n", Stderr: "warn\n", OutputAbandoned: true}
	}}
	out := Run([]Job{job("lint/go", -1, "x")}, opts(f))
	want := "partial\nwarn\noutput abandoned: a process the tool started kept its pipe open\n"
	if out[0].State != StateFailed || out[0].Output != want {
		t.Fatalf("%q", out[0].Output)
	}
}

func TestRunNamesAnAbandonedOutputWithoutAnEnd(t *testing.T) {
	f := &fakeStart{answer: func(child.Spec) child.Result { return child.Result{Stdout: "cut", OutputAbandoned: true} }}
	out := Run([]Job{job("lint/go", -1, "x")}, opts(f))
	if out[0].Output != "cut\n"+abandoned+"\n" {
		t.Fatalf("%q", out[0].Output)
	}
}

func TestRunKeepsABudgetThatStoppedTheMeasure(t *testing.T) {
	c := &clock{}
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.Scope, o.Budget, o.Now = ScopeEdit, time.Minute, c.now
	j := job("coverage/go", -1, "report")
	j.Measure = []string{"measure"}
	out := Run([]Job{j, job("x/go", 0, "x")}, o)
	if out[0].State != StateBudget || out[1].State != StateBudget || len(f.started) != 0 {
		t.Fatalf("%+v %v", out, f.started)
	}
}

func TestRunDeductsTheWaitForASlotFromTheBudget(t *testing.T) {
	var mu sync.Mutex
	now := time.Time{}
	f := &fakeStart{answer: func(child.Spec) child.Result {
		// Long enough for the other lane to reach the cap and look at the
		// clock before this one has spent the budget.
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		now = now.Add(2 * time.Minute)
		mu.Unlock()
		return child.Result{}
	}}
	o := opts(f)
	o.MaxParallel, o.Budget = 1, time.Minute
	o.Now = func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	out := Run([]Job{job("a/go", -1, "a"), job("b/go", -1, "b")}, o)
	budget := 0
	for _, x := range out {
		if x.State == StateBudget {
			budget++
		}
	}
	if len(f.started) != 1 || budget != 1 {
		t.Fatalf("%+v %v", out, f.started)
	}
}

func TestRunMarksOnlyRedBlocks(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "b" {
			return child.Result{Code: 1, Stdout: "bad\n"}
		}
		return child.Result{Code: -1, TimedOut: true}
	}}
	o := opts(f)
	// A clock that stands still leaves the rest at the budget, below the
	// timeout, so "a" is cut off by the budget and is not red.
	o.Budget, o.Now = 30*time.Second, func() time.Time { return time.Time{} }
	j := job("lint/go", -1, "a", "b")
	out := Run([]Job{j}, o)
	// A block without output keeps its line break, as checks.py wrote it.
	if out[0].Output != "$ a\n\n\n$ b (failed)\nbad\n" {
		t.Fatalf("%q", out[0].Output)
	}
}

func TestRunStartsNothingOnceTheBudgetIsSpent(t *testing.T) {
	c := &clock{}
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.Budget, o.Now = time.Minute, c.now
	out := Run([]Job{job("lint/go", -1, "x"), job("fmt/go", -1, "a", "b")}, o)
	if out[0].State != StateBudget || out[1].State != StateBudget || len(f.started) != 0 {
		t.Fatalf("%+v %v", out, f.started)
	}
	want := "$ a\nnot started: the budget is spent\n\n$ b\nnot started: the budget is spent\n"
	if out[1].Output != want {
		t.Fatalf("%q", out[1].Output)
	}
}

func TestRunLetsAFindingOutrankTheBudget(t *testing.T) {
	c := &clock{}
	f := &fakeStart{answer: func(child.Spec) child.Result { return child.Result{Code: 1} }}
	o := opts(f)
	o.Budget, o.Now = 150*time.Minute, c.now
	j := job("lint/go", -1, "a", "b")
	j.Threaded = true
	out := Run([]Job{j}, o)
	// One of the two commands gets the half hour left, the other finds
	// nothing left; the failure of the one that ran decides the lane.
	if out[0].State != StateFailed || len(f.started) != 1 {
		t.Fatalf("%+v %v", out[0], f.started)
	}
}

func TestRunInheritsWhatAnEditCannotJudge(t *testing.T) {
	for _, s := range []State{StateBudget, StateMissingTool, StateUnready, StateNotApplicable} {
		pre := job("test/go", -1)
		pre.Pre = s
		o := opts(&fakeStart{answer: ok})
		o.Scope = ScopeEdit
		if out := Run([]Job{pre, job("coverage/go", 0, "c")}, o); out[1].State != s || out[1].BlockedBy != "" {
			t.Fatalf("%s: %+v", s, out[1])
		}
	}
}

func TestRunBlocksWhatACheckCannotSkip(t *testing.T) {
	pre := job("test/go", -1)
	pre.Pre = StateMissingTool
	if out := Run([]Job{pre, job("coverage/go", 0, "c")}, opts(&fakeStart{answer: ok})); out[1].State != StateBlocked {
		t.Fatalf("%+v", out[1])
	}
}

func TestRunLetsACheckRunPastABudgetPredecessor(t *testing.T) {
	pre := job("test/go", -1)
	pre.Pre = StateBudget
	f := &fakeStart{answer: ok}
	if out := Run([]Job{pre, job("coverage/go", 0, "c")}, opts(f)); out[1].State != StateOK || len(f.started) != 1 {
		t.Fatalf("%+v", out[1])
	}
}

func TestRunWaitsForAPredecessorFurtherDown(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result { time.Sleep(10 * time.Millisecond); return child.Result{} }}
	out := Run([]Job{job("coverage/go", 1, "c"), job("test/go", -1, "t")}, opts(f))
	if out[0].State != StateOK || strings.Join(f.started, ";") != "t;c" {
		t.Fatalf("%+v %v", out, f.started)
	}
}

func TestRunDoesNotDeadlockAChainOnOneSlot(t *testing.T) {
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.MaxParallel = 1
	done := make(chan []Outcome)
	go func() {
		done <- Run([]Job{job("coverage/go", 1, "c"), job("test/go", -1, "t"), job("lint/go", -1, "l")}, o)
	}()
	select {
	case out := <-done:
		if out[0].State != StateOK || len(f.started) != 3 {
			t.Fatalf("%+v", out)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock")
	}
}

func TestRunTreatsANonPositiveCapAsOne(t *testing.T) {
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.MaxParallel = 0
	out := Run([]Job{job("a/go", -1, "a"), job("b/go", -1, "b")}, o)
	if out[0].State != StateOK || out[1].State != StateOK || f.peak != 1 {
		t.Fatalf("%+v peak %d", out, f.peak)
	}
}

func TestRunPassesDirEnvAndOutputInOrder(t *testing.T) {
	var got child.Spec
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		got = s
		return child.Result{Stdout: "out\n", Stderr: "err\n"}
	}}
	j := job("lint/python", -1, "ruff check")
	j.Dir, j.Env = "area", []string{"K=V"}
	out := Run([]Job{j}, opts(f))
	if out[0].Output != "out\nerr\n" || got.Dir != "area" || strings.Join(got.Env, ",") != "K=V" || got.Timeout != time.Minute {
		t.Fatalf("%q %+v", out[0].Output, got)
	}
}

func TestRunMeasuresTheDurationOfALane(t *testing.T) {
	c := &clock{}
	o := opts(&fakeStart{answer: ok})
	o.Now = c.now
	if out := Run([]Job{job("a/go", -1, "a")}, o); out[0].Duration != time.Hour {
		t.Fatalf("%v", out[0].Duration)
	}
}

func TestRedDependsOnTheScope(t *testing.T) {
	cases := []struct {
		s           State
		check, edit bool
	}{
		{StateOK, false, false},
		{StateFailed, true, true},
		{StateTimedOut, true, true},
		{StateBlocked, true, true},
		{StateBudget, false, false},
		{StateMissingTool, true, false},
		{StateUnready, true, false},
		{StateUnavailable, false, false},
		{StateNotApplicable, false, false},
	}
	for _, c := range cases {
		if Red(c.s, ScopeCheck) != c.check || Red(c.s, ScopeEdit) != c.edit {
			t.Errorf("%s: check %v edit %v", c.s, Red(c.s, ScopeCheck), Red(c.s, ScopeEdit))
		}
	}
}

// lane is job with the fields a key is built from.
func lane(kind string, after int, argv string) Job {
	j := job(kind+"/go", after, argv)
	j.Kind, j.Stack, j.Area = kind, "go", "."
	return j
}

// Coverage waits for a test that fails. Whether the blocked lane is red is
// decided by the lane that blocks it, not by its own place in the file.
func TestBlockedIsRedOnlyBehindAnArmedLane(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "test" {
			return child.Result{Code: 1}
		}
		return child.Result{}
	}}
	jobs := []Job{lane("test", -1, "test run"), lane("coverage", 0, "cover run")}
	for name, c := range map[string]struct {
		armed               func(Job) bool
		testProb, coverProb bool
	}{
		"no set":                        {nil, false, false},
		"the test is in probation":      {func(j Job) bool { return j.Kind != "test" }, true, true},
		"only coverage is in probation": {func(j Job) bool { return j.Kind == "test" }, false, false},
		"both are in probation":         {func(Job) bool { return false }, true, true},
	} {
		o := opts(f)
		o.Armed = c.armed
		outs := Run(jobs, o)
		if outs[0].State != StateFailed || outs[1].State != StateBlocked || outs[1].BlockedBy != "test/go" {
			t.Fatalf("%s: states %s, %s by %q", name, outs[0].State, outs[1].State, outs[1].BlockedBy)
		}
		if outs[0].Probation != c.testProb || outs[1].Probation != c.coverProb {
			t.Errorf("%s: probation %v, %v; want %v, %v", name, outs[0].Probation, outs[1].Probation, c.testProb, c.coverProb)
		}
		if Fails(outs[1], ScopeCheck) == c.coverProb {
			t.Errorf("%s: blocked fails = %v", name, Fails(outs[1], ScopeCheck))
		}
	}
}

// A block passes down a chain with the answer of the lane that failed, not of
// the blocked lane in between: behind a test in probation nothing is red,
// behind an armed test everything is, whatever the middle lane's arming.
func TestABlockCarriesTheAnswerOfTheLaneThatFailed(t *testing.T) {
	f := &fakeStart{answer: func(s child.Spec) child.Result {
		if s.Argv[0] == "test" {
			return child.Result{Code: 1}
		}
		return child.Result{}
	}}
	jobs := []Job{lane("test", -1, "test run"), lane("coverage", 0, "cover run"), lane("lint", 1, "lint run")}
	for name, c := range map[string]struct {
		armed     func(Job) bool
		probation bool
		code      int
	}{
		"the test is in probation, the rest armed": {func(j Job) bool { return j.Kind != "test" }, true, 0},
		"the test is armed, the middle is not":     {func(j Job) bool { return j.Kind != "coverage" }, false, 1},
	} {
		o := opts(f)
		o.Armed = c.armed
		outs := Run(jobs, o)
		for i, by := range map[int]string{1: "test/go", 2: "coverage/go"} {
			if outs[i].State != StateBlocked || outs[i].BlockedBy != by || outs[i].Probation != c.probation {
				t.Errorf("%s: lane %d %s by %q, probation %v", name, i, outs[i].State, outs[i].BlockedBy, outs[i].Probation)
			}
		}
		if code, _ := CheckVerdict(nil, outs, true); code != c.code {
			t.Errorf("%s: code %d, want %d", name, code, c.code)
		}
	}
}

// A lane that ran, and one that inherits "cannot judge", carry the answer for
// their own lane.
func TestEveryOutcomeSaysWhetherItsLaneIsInProbation(t *testing.T) {
	f := &fakeStart{answer: ok}
	pre := lane("test", -1, "test run")
	pre.Pre, pre.Note = StateUnavailable, "no tests found"
	jobs := []Job{pre, lane("coverage", 0, "cover run"), lane("lint", -1, "lint run")}
	o := opts(f)
	o.Armed = func(j Job) bool { return j.Kind == "lint" }
	outs := Run(jobs, o)
	if !outs[0].Probation || !outs[1].Probation || outs[2].Probation {
		t.Fatalf("probation %v %v %v", outs[0].Probation, outs[1].Probation, outs[2].Probation)
	}
	if outs[1].State != StateUnavailable {
		t.Fatalf("coverage %s", outs[1].State)
	}
}
