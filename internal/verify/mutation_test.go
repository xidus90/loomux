package verify

import (
	"bytes"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/detect"
)

// These tests close gaps the stage 2a mutation round found: each one pins a
// difference the suite could not see before.

func TestCleanCoverRemovesTheOwnFreshFilesOfAGreenRun(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	root := t.TempDir()
	dir := coverFiles(t, root, "B-go-root.out")
	age(t, filepath.Join(dir, "B-go-root.out"), now.Add(-time.Minute))
	if err := cleanCover(root, "B", true, now); err != nil {
		t.Fatal(err)
	}
	if got := left(t, dir); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestCleanCoverRemovesAFileExactlyADayOld(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	root := t.TempDir()
	dir := coverFiles(t, root, "A-go-root.out")
	age(t, filepath.Join(dir, "A-go-root.out"), now.Add(-staleAfter))
	if err := cleanCover(root, "B", true, now); err != nil {
		t.Fatal(err)
	}
	if got := left(t, dir); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestATableOverrideKeepsEveryKeyItDoesNotName(t *testing.T) {
	base := Lane{Commands: []string{"a"}, OnFile: []string{"f"}, Threaded: true,
		Measuring: "m", Measure: "c", After: "types"}
	got := merge(base, Override{Lane: Lane{Commands: []string{"b"}}, Set: map[string]bool{"commands": true}})
	if !slices.Equal(got.Commands, []string{"b"}) || !slices.Equal(got.OnFile, base.OnFile) ||
		got.Threaded != base.Threaded || got.Measuring != base.Measuring ||
		got.Measure != base.Measure || got.After != base.After {
		t.Fatalf("%+v", got)
	}
}

func TestTheDeepestAreaWinsWhateverTheOrder(t *testing.T) {
	if got := editArea([]string{"apps/web", "apps", "."}, "apps/web/x.go"); got != "apps/web" {
		t.Fatal(got)
	}
}

// oneLane is an effective config with a single test lane in the root, so a
// plan sees exactly what the lane says.
func oneLane(lane Lane) Effective {
	return Effective{
		Stacks:     map[string]map[string]Resolved{"x": {"test": {Lane: lane, Origin: "config", Defined: true}}},
		Areas:      map[string][]string{"x": {"."}},
		Active:     []string{"x"},
		Configured: map[string]bool{},
		TestsWhen:  map[string][]string{},
	}
}

func TestAStackWithoutTestPatternsIsNotSearchedForTests(t *testing.T) {
	e := env(t.TempDir())
	e.HasTests = HasTests
	jobs, err := Plan(oneLane(Lane{Commands: []string{"t"}}), Request{Kinds: []string{"test"}}, e)
	if err != nil || len(jobs) != 1 || jobs[0].Pre != "" {
		t.Fatalf("%+v %v", jobs, err)
	}
}

func TestAMeasureWithoutAnAfterIsNeverRun(t *testing.T) {
	jobs, err := Plan(oneLane(Lane{Commands: []string{"t"}, Measure: "m"}), Request{Kinds: []string{"test"}}, env(t.TempDir()))
	if err != nil || len(jobs) != 1 || jobs[0].Measure != nil || jobs[0].Pre != "" {
		t.Fatalf("%+v %v", jobs, err)
	}
}

func TestAVariantRefusesAKeyThatIsNoKindEvenAsACommand(t *testing.T) {
	_, err := parsePresets("[[stack.go.variant]]\nwhen = \"biome\"\nstyle = \"x\"")
	if err == nil || !strings.Contains(err.Error(), `unknown key "style"`) {
		t.Fatalf("%v", err)
	}
}

func TestAMeasureWhoseToolIsMissingStartsNothing(t *testing.T) {
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.Look = func(s string) (string, error) {
		if s == "measure" {
			return "", errors.New("not found")
		}
		return s, nil
	}
	j := job("coverage/go", -1, "report")
	j.Measure = []string{"measure"}
	out := Run([]Job{j}, o)
	if out[0].State != StateMissingTool || len(f.started) != 0 {
		t.Fatalf("%+v %v", out[0], f.started)
	}
}

func TestAMeasureThatTimedOutFailsTheLane(t *testing.T) {
	f := &fakeStart{answer: func(child.Spec) child.Result { return child.Result{Code: -1, TimedOut: true} }}
	j := job("coverage/go", -1, "report")
	j.Measure = []string{"measure"}
	out := Run([]Job{j}, opts(f))
	if out[0].State != StateFailed || len(f.started) != 1 {
		t.Fatalf("%+v %v", out[0], f.started)
	}
}

// Each command waits a moment for the other to start beside it, whichever
// comes first; in a lane that is not threaded, neither may.
func TestALaneThatIsNotThreadedRunsOneCommandAtATime(t *testing.T) {
	var mu sync.Mutex
	arrived := 0
	both := make(chan struct{})
	f := &fakeStart{answer: func(child.Spec) child.Result {
		mu.Lock()
		arrived++
		if arrived == 2 {
			close(both)
		}
		mu.Unlock()
		select {
		case <-both:
		case <-time.After(200 * time.Millisecond):
		}
		return child.Result{}
	}}
	out := Run([]Job{job("lint/go", -1, "a", "b")}, opts(f))
	if out[0].State != StateOK || f.peak != 1 {
		t.Fatalf("%+v peak %d", out[0], f.peak)
	}
}

// steady is a Now that answers start once, for the deadline, and start plus
// the budget ever after: the budget is spent to the nanosecond.
func steady(start, later time.Time) func() time.Time {
	var mu sync.Mutex
	calls := 0
	return func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls == 1 {
			return start
		}
		return later
	}
}

func TestABudgetSpentToTheNanosecondStartsNothing(t *testing.T) {
	start := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.Budget, o.Now = time.Minute, steady(start, start.Add(time.Minute))
	out := Run([]Job{job("lint/go", -1, "a")}, o)
	if out[0].State != StateBudget || len(f.started) != 0 {
		t.Fatalf("%+v %v", out[0], f.started)
	}
}

func TestATimeoutAsLongAsTheRestIsTheLanesOwn(t *testing.T) {
	start := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	f := &fakeStart{answer: func(child.Spec) child.Result { return child.Result{Code: -1, TimedOut: true} }}
	o := opts(f)
	o.Timeout, o.Budget, o.Now = time.Minute, 2*time.Minute, steady(start, start.Add(time.Minute))
	out := Run([]Job{job("lint/go", -1, "a")}, o)
	if out[0].State != StateTimedOut {
		t.Fatalf("%+v", out[0])
	}
}

func TestABudgetAfterAFindingDoesNotHideIt(t *testing.T) {
	state, _ := mergeSteps([][]string{{"a"}, {"b"}}, []step{{StateFailed, "x"}, {StateBudget, "y"}}, ScopeCheck)
	if state != StateFailed {
		t.Fatal(state)
	}
}

func TestTheSmallestCapAndTimeoutAreAccepted(t *testing.T) {
	cfg, err := parse(t, "[verify]\nmax_parallel = 1\ntimeout = 1\n")
	if err != nil || cfg.MaxParallel != 1 || cfg.Timeout != time.Second {
		t.Fatalf("%+v %v", cfg, err)
	}
}

func TestALaneWithOnlyAnOnFileFormShowsNoCommands(t *testing.T) {
	var b bytes.Buffer
	writeLane(&b, "go", "lint", Resolved{Lane: Lane{OnFile: []string{"x {file}"}}, Origin: "config", Defined: true})
	if want := "[verify.go.lint]\non_file = [\"x {file}\"]  # config\n"; b.String() != want {
		t.Fatalf("%q", b.String())
	}
}

func TestPlanEditScopeLetsAOneLetterAreaOwnItsFiles(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"typescript"}, Areas: map[string][]string{"typescript": {".", "a"}}}
	req := Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "a/X.TS"}
	jobs, _ := Plan(effFor(t, "", facts), req, env(t.TempDir()))
	if names(jobs) != "lint/typescript@a" || jobs[0].Argvs[0][3] != "X.TS" {
		t.Fatalf("%+v", jobs)
	}
}
