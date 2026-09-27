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
		code, notes := CheckVerdict(c.kinds, c.outs)
		if code != c.code || (c.note == "" && len(notes) != 0) || (c.note != "" && (len(notes) != 1 || notes[0] != c.note)) {
			t.Errorf("%s: %d %v", c.name, code, notes)
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
