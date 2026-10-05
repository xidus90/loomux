package verify

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func out(name string, s State, origin string) Outcome {
	return Outcome{Job: Job{Name: name, Kind: strings.Split(name, "/")[0], Origin: origin}, State: s, Duration: 1200 * time.Millisecond}
}

func TestWriteCheckFormatsEveryState(t *testing.T) {
	outs := []Outcome{
		out("lint/go", StateOK, "config"),
		{Job: Job{Name: "lint/python", Kind: "lint", Origin: "preset"}, State: StateFailed, Output: "E1 bad\n", Duration: 800 * time.Millisecond},
		{Job: Job{Name: "types/go", Kind: "types", Origin: "preset"}, State: StateNotApplicable, Output: "no command"},
		{Job: Job{Name: "coverage/go", Kind: "coverage", Origin: "preset"}, State: StateBlocked, BlockedBy: "test/go"},
	}
	var b strings.Builder
	WriteCheck(&b, outs, false)
	want := "lint/go: ok [config] 1.2s\n" +
		"lint/python: failed [preset] 0.8s\nE1 bad\n" +
		"types/go: not-applicable [preset] no command\n" +
		"coverage/go: blocked [preset] by test/go\n"
	if b.String() != want {
		t.Fatalf("%q", b.String())
	}
}

// Verbose shows what a lane that ran printed, green or not, but repeats no
// note the header already carries, and an in-process lane names no layer.
func TestWriteCheckVerboseAndInProcess(t *testing.T) {
	fn := func() (string, error) { return "", errors.New("unused") }
	outs := []Outcome{
		{Job: Job{Name: "lint/go", Origin: "preset"}, State: StateOK, Output: "all fine", Duration: 50 * time.Millisecond},
		{Job: Job{Name: "lint/wiki", Origin: "preset", Fn: fn}, State: StateOK, Duration: 0},
		{Job: Job{Name: "test/go", Origin: "preset"}, State: StateBudget, Output: "part\n", Duration: 2 * time.Second},
		{Job: Job{Name: "lint/python", Origin: "preset"}, State: StateMissingTool, Output: `"ruff" is not on PATH: ruff check .`},
		{Job: Job{Name: "test/gdscript", Origin: "preset"}, State: StateUnready, Output: "import first\n"},
		{Job: Job{Name: "types/go", Origin: "preset"}, State: StateTimedOut, Output: "", Duration: time.Second},
	}
	var b strings.Builder
	WriteCheck(&b, outs, true)
	want := "lint/go: ok [preset] 0.1s\nall fine\n" +
		"lint/wiki: ok [in-process] 0.0s\n" +
		"test/go: budget [preset] 2.0s\npart\n" +
		"lint/python: missing-tool [preset] \"ruff\" is not on PATH: ruff check .\n" +
		"test/gdscript: unready [preset] import first\n" +
		"types/go: timed-out [preset] 1.0s\n"
	if b.String() != want {
		t.Fatalf("%q", b.String())
	}
	b.Reset()
	WriteCheck(&b, outs[:1], false)
	if b.String() != "lint/go: ok [preset] 0.1s\n" {
		t.Fatalf("a green lane speaks only when asked: %q", b.String())
	}
}

func TestCheckVerdictPerKind(t *testing.T) {
	cases := []struct {
		name  string
		kinds []string
		outs  []Outcome
		code  int
		note  string
	}{
		{"green", []string{"lint", "types"}, []Outcome{out("lint/go", StateOK, "p"), out("types/go", StateNotApplicable, "p")}, 0, ""},
		{"a kind with nothing run", []string{"test"}, []Outcome{out("test/go", StateUnavailable, "p")}, 1, "nothing to check for `test`"},
		{"a kind with no lane", []string{"types"}, []Outcome{out("lint/go", StateOK, "p")}, 1, "nothing to check for `types`"},
		{"red", []string{"lint"}, []Outcome{out("lint/go", StateMissingTool, "p")}, 1, ""},
		{"unready is red, not nothing", []string{"test"}, []Outcome{out("test/gdscript", StateUnready, "p")}, 1, ""},
		{"blocked is red, not nothing", []string{"coverage"}, []Outcome{out("coverage/go", StateBlocked, "p")}, 1, ""},
		{"unavailable beside a run lane", []string{"test"}, []Outcome{out("test/go", StateUnavailable, "p"), out("test/python", StateOK, "p")}, 0, ""},
		{"a graph lane that stood aside", []string{"graph"}, []Outcome{out("graph/go", StateNotApplicable, "p"), out("graph/python", StateNotApplicable, "p")}, 0, ""},
		{"budget is no finding", []string{"test"}, []Outcome{out("test/go", StateBudget, "p")}, 0, ""},
	}
	for _, c := range cases {
		code, notes := CheckVerdict(c.kinds, c.outs, true)
		if code != c.code || (c.note == "" && len(notes) != 0) || (c.note != "" && (len(notes) != 1 || notes[0] != c.note)) {
			t.Errorf("%s: %d %v", c.name, code, notes)
		}
	}
}

// Not asked for by name, a kind the project has no lane for is left out with
// a note -- as long as one kind of the request had something to check.
func TestCheckVerdictLeavesOutAKindNobodyNamed(t *testing.T) {
	cases := []struct {
		name  string
		kinds []string
		outs  []Outcome
		code  int
		notes []string
	}{
		{"lint alone, as in a vault", []string{"lint", "types", "test", "coverage"}, []Outcome{out("lint/wiki", StateOK, "p")}, 0,
			[]string{"no lane for `types` here, left out", "no lane for `test` here, left out", "no lane for `coverage` here, left out"}},
		{"a red lane stays red", []string{"lint", "test"}, []Outcome{out("lint/go", StateFailed, "p")}, 1,
			[]string{"no lane for `test` here, left out"}},
		{"a lane that stood aside checked nothing", []string{"graph", "test"}, []Outcome{out("graph/go", StateNotApplicable, "p")}, 1,
			[]string{"nothing to check for `test`"}},
		{"a lane that stood aside beside one that ran", []string{"lint", "graph", "test"}, []Outcome{out("lint/wiki", StateOK, "p"), out("graph/go", StateNotApplicable, "p")}, 0,
			[]string{"no lane for `test` here, left out"}},
		{"nothing of the request ran", []string{"types", "test"}, []Outcome{out("lint/go", StateOK, "p")}, 1,
			[]string{"nothing to check for `types`", "nothing to check for `test`"}},
		{"unavailable alone is nothing", []string{"test", "types"}, []Outcome{out("test/go", StateUnavailable, "p")}, 1,
			[]string{"nothing to check for `test`", "nothing to check for `types`"}},
	}
	for _, c := range cases {
		code, notes := CheckVerdict(c.kinds, c.outs, false)
		if code != c.code || !slices.Equal(notes, c.notes) {
			t.Errorf("%s: %d %q", c.name, code, notes)
		}
	}
}

// The budget's notice is the one cli-reference.md shows, for a lane and for
// a file alike.
func TestBudgetSkippedIsTheDocumentedSentence(t *testing.T) {
	want := "loomux hook post-tool-use: lane skipped, the edit budget ran out: lint/go"
	if got := BudgetSkipped("lint/go"); got != want {
		t.Fatalf("%q", got)
	}
}

// A skipped lane passes the edit and is said twice: as a notice for the
// model at exit 0, and on stderr for a host that reads it at exit 2.
func TestEditReportSkipsOutLoudAndBlocksOnRed(t *testing.T) {
	var se strings.Builder
	skip := Outcome{Job: Job{Name: "lint/python"}, State: StateMissingTool, Output: `"ruff" is not on PATH: ruff check .`}
	said := skipPrefix + `"ruff" is not on PATH: ruff check .`
	red, notices := EditReport(&se, []Outcome{skip}, "")
	if red || se.String() != said+"\n" || !slices.Equal(notices, []string{said}) {
		t.Fatalf("%v %q %q", red, se.String(), notices)
	}
	se.Reset()
	failed := Outcome{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x\n"}
	if red, _ := EditReport(&se, []Outcome{failed}, ""); !red || !strings.Contains(se.String(), "vet: x") {
		t.Fatalf("%v %q", red, se.String())
	}
}

func TestEditReportNamesEverySkipAndEveryRed(t *testing.T) {
	var se strings.Builder
	outs := []Outcome{
		{Job: Job{Name: "types/go"}, State: StateOK, Output: "fine\n"},
		{Job: Job{Name: "test/go"}, State: StateBudget, Output: "part"},
		{Job: Job{Name: "test/gdscript"}, State: StateUnready, Output: "run the Godot editor once to import the project"},
		{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x"},
		{Job: Job{Name: "lint/sql"}, State: StateBlocked, BlockedBy: "x"},
	}
	red, notices := EditReport(&se, outs, "")
	if !red {
		t.Fatal("red lanes block the edit")
	}
	wantErr := "loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go\n" +
		"loomux hook post-tool-use: lane skipped, run the Godot editor once to import the project\n" +
		"lint/go: failed\nvet: x\nlint/sql: blocked\n"
	if se.String() != wantErr {
		t.Fatalf("%q", se.String())
	}
	want := []string{
		"loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go",
		"loomux hook post-tool-use: lane skipped, run the Godot editor once to import the project",
	}
	if !slices.Equal(notices, want) {
		t.Fatalf("%q", notices)
	}
}

// A red lane blocks the edit and a host reads stderr alone: the lane the
// edit could not check is named there beside the finding.
func TestEditReportNamesASkippedLaneOnStderr(t *testing.T) {
	var se strings.Builder
	outs := []Outcome{
		{Job: Job{Name: "lint/python"}, State: StateFailed, Output: "E1 bad"},
		{Job: Job{Name: "types/python"}, State: StateMissingTool, Output: `"uv" is not on PATH: uv run mypy`},
	}
	red, _ := EditReport(&se, outs, "")
	if !red || !strings.Contains(se.String(), skipPrefix+`"uv" is not on PATH: uv run mypy`+"\n") || !strings.Contains(se.String(), "E1 bad") {
		t.Fatalf("%v %q", red, se.String())
	}
}

// The helper both streams go through: stderr gets the line, the notices get
// it appended, and nothing else is written.
func TestSkippedSaysTheNoticeOnStderrAndKeepsIt(t *testing.T) {
	var se strings.Builder
	notices := Skipped(&se, []string{"first"}, "second")
	if se.String() != "second\n" || len(notices) != 2 || notices[1] != "second" {
		t.Fatalf("%q %q", se.String(), notices)
	}
}

// A run with nothing skipped and nothing to add has nothing to say.
func TestEditReportGreenIsSilent(t *testing.T) {
	var se strings.Builder
	if red, notices := EditReport(&se, []Outcome{{Job: Job{Name: "lint/go"}, State: StateOK}}, ""); red || notices != nil || se.Len() != 0 {
		t.Fatalf("%v %q %q", red, notices, se.String())
	}
}

func TestEditReportCarriesTheAsideOnAGreenRun(t *testing.T) {
	var se strings.Builder
	aside := "[graph] a.go: changed F; callers in other files:\n  G (b.go)"
	red, notices := EditReport(&se, []Outcome{{Job: Job{Name: "lint/go"}, State: StateOK}}, aside)
	if red || se.Len() != 0 || !slices.Equal(notices, []string{aside}) {
		t.Fatalf("%v %q %q", red, se.String(), notices)
	}
}

func TestEditReportPutsTheAsideAfterTheSkips(t *testing.T) {
	var se strings.Builder
	skip := Outcome{Job: Job{Name: "test/go"}, State: StateBudget}
	red, notices := EditReport(&se, []Outcome{skip}, "[graph] x")
	want := []string{"loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go", "[graph] x"}
	if red || !slices.Equal(notices, want) {
		t.Fatalf("%v %q", red, notices)
	}
}

// A red lane matters more than who calls the edited code; the aside goes,
// the skips of another lane stay as they were.
func TestEditReportDropsTheAsideOnRed(t *testing.T) {
	var se strings.Builder
	failed := Outcome{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x"}
	if red, notices := EditReport(&se, []Outcome{failed}, "[graph] x"); !red || notices != nil {
		t.Fatalf("%v %q", red, notices)
	}
	skip := Outcome{Job: Job{Name: "test/go"}, State: StateBudget}
	red, notices := EditReport(&se, []Outcome{failed, skip}, "[graph] x")
	if !red || !slices.Equal(notices, []string{BudgetSkipped("test/go")}) {
		t.Fatalf("%v %q", red, notices)
	}
}

func TestFailsIsRedOutsideProbation(t *testing.T) {
	for _, s := range []State{StateOK, StateFailed, StateTimedOut, StateBudget, StateBlocked, StateMissingTool, StateUnready, StateUnavailable, StateNotApplicable} {
		for _, scope := range []Scope{ScopeCheck, ScopeEdit} {
			if got := Fails(Outcome{State: s}, scope); got != Red(s, scope) {
				t.Errorf("%s armed: fails %v, red %v", s, got, Red(s, scope))
			}
			if Fails(Outcome{State: s, Probation: true}, scope) {
				t.Errorf("%s in probation fails", s)
			}
		}
	}
}

func probing(name, key string, s State, output string) Outcome {
	kind, rest, _ := strings.Cut(key, "/")
	stack, area, _ := strings.Cut(rest, "@")
	return Outcome{Job: Job{Name: name, Kind: kind, Stack: stack, Area: area, Origin: "preset"},
		State: s, Output: output, Duration: 800 * time.Millisecond, Probation: true}
}

// The state stays the real one; the header says that it does not count. A
// green lane in probation reads like any green lane.
func TestWriteCheckMarksARedLaneInProbation(t *testing.T) {
	blocked := probing("coverage/go", "coverage/go@.", StateBlocked, "")
	blocked.BlockedBy = "test/go"
	outs := []Outcome{
		probing("lint/python", "lint/python@.", StateFailed, "E1 bad\n"),
		probing("lint/go", "lint/go@.", StateOK, ""),
		probing("types/python", "types/python@.", StateMissingTool, `"mypy" is not on PATH: mypy .`),
		blocked,
		probing("types/go", "types/go@.", StateNotApplicable, "no command"),
	}
	var b strings.Builder
	WriteCheck(&b, outs, false)
	want := "lint/python: failed (probation) [preset] 0.8s\nE1 bad\n" +
		"lint/go: ok [preset] 0.8s\n" +
		"types/python: missing-tool (probation) [preset] \"mypy\" is not on PATH: mypy .\n" +
		"coverage/go: blocked (probation) [preset] by test/go\n" +
		"types/go: not-applicable [preset] no command\n"
	if b.String() != want {
		t.Fatalf("%q", b.String())
	}
}

func TestCheckVerdictPassesARunRedOnlyInProbation(t *testing.T) {
	red := probing("lint/go", "lint/go@.", StateFailed, "x\n")
	if code, notes := CheckVerdict([]string{"lint"}, []Outcome{red}, true); code != 0 || len(notes) != 0 {
		t.Fatalf("probation alone: %d %v", code, notes)
	}
	armed := red
	armed.Probation = false
	if code, _ := CheckVerdict([]string{"lint"}, []Outcome{red, armed}, true); code != 1 {
		t.Fatalf("an armed red lane beside it: %d", code)
	}
	// The second rule is untouched: a kind with nothing to check is red
	// whatever the file says.
	if code, notes := CheckVerdict([]string{"lint", "test"}, []Outcome{red}, true); code != 1 || len(notes) != 1 {
		t.Fatalf("nothing to check for test: %d %v", code, notes)
	}
}

func TestEditReportSaysARedLaneInProbationWithoutHoldingTheEdit(t *testing.T) {
	var se strings.Builder
	red, notices := EditReport(&se, []Outcome{probing("lint/go", "lint/go@.", StateFailed, "a.go:1: bad\n")}, "aside")
	want := "lint/go: failed (probation)\na.go:1: bad"
	if red || !slices.Equal(notices, []string{want, "aside"}) || se.String() != want+"\n" {
		t.Fatalf("red %v, notices %q, stderr %q", red, notices, se.String())
	}
	se.Reset()
	armed := probing("lint/go", "lint/go@.", StateFailed, "a.go:1: bad\n")
	armed.Probation = false
	red, notices = EditReport(&se, []Outcome{armed}, "aside")
	if !red || len(notices) != 0 || se.String() != "lint/go: failed\na.go:1: bad\n" {
		t.Fatalf("armed: red %v, notices %q, stderr %q", red, notices, se.String())
	}
}

func TestProbationLineNamesTheLanesThatHaveSomethingToCheck(t *testing.T) {
	// The file arms the test lane and nothing else.
	armsOnlyTests := func(j Job) bool { return j.Kind == "test" }
	outs := []Outcome{
		probing("lint/python", "lint/python@.", StateFailed, ""),
		probing("lint/go", "lint/go@.", StateOK, ""),
		probing("types/python", "types/python@.", StateMissingTool, ""),
		probing("types/go", "types/go@.", StateNotApplicable, ""),
		probing("coverage/go", "coverage/go@.", StateUnavailable, ""),
		probing("test/go", "test/go@.", StateFailed, ""),
		probing("lint/go@tools", "lint/go@tools", StateOK, ""),
		// A second job of the same lane names it once.
		probing("lint/python", "lint/python@.", StateFailed, ""),
	}
	want := "probation: lint/go@., lint/go@tools, lint/python@., types/python@. (warn only until a green commit arms them)"
	if got := ProbationLine(outs, armsOnlyTests); got != want {
		t.Fatalf("%q", got)
	}
	if got := ProbationLine(outs, nil); got != "" {
		t.Fatalf("no set: %q", got)
	}
	if got := ProbationLine(outs, func(Job) bool { return true }); got != "" {
		t.Fatalf("every lane armed: %q", got)
	}
}

func TestGreenKeysAreTheLanesThatEndedOK(t *testing.T) {
	outs := []Outcome{
		probing("test/go", "test/go@.", StateOK, ""),
		probing("lint/go", "lint/go@.", StateOK, ""),
		probing("lint/python", "lint/python@.", StateFailed, ""),
		probing("types/go", "types/go@.", StateNotApplicable, ""),
		probing("coverage/go", "coverage/go@.", StateBudget, ""),
		probing("test/go", "test/go@.", StateOK, ""),
	}
	if got := GreenKeys(outs); !slices.Equal(got, []string{"lint/go@.", "test/go@."}) {
		t.Fatalf("%v", got)
	}
}
