package verify

import (
	"errors"
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
		{"budget is no finding", []string{"test"}, []Outcome{out("test/go", StateBudget, "p")}, 0, ""},
	}
	for _, c := range cases {
		code, notes := CheckVerdict(c.kinds, c.outs)
		if code != c.code || (c.note == "" && len(notes) != 0) || (c.note != "" && (len(notes) != 1 || notes[0] != c.note)) {
			t.Errorf("%s: %d %v", c.name, code, notes)
		}
	}
}

func TestWriteEditSkipsQuietlyAndBlocksOnRed(t *testing.T) {
	var so, se strings.Builder
	skip := Outcome{Job: Job{Name: "lint/python"}, State: StateMissingTool, Output: `"ruff" is not on PATH: ruff check .`}
	if code := WriteEdit(&so, &se, []Outcome{skip}); code != 0 || se.Len() != 0 {
		t.Fatalf("%d %q", code, se.String())
	}
	if !strings.Contains(so.String(), `lane skipped, \"ruff\" is not on PATH`) || !strings.Contains(so.String(), `"hookEventName":"PostToolUse"`) {
		t.Fatalf("%q", so.String())
	}
	so.Reset()
	red := Outcome{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x\n"}
	if code := WriteEdit(&so, &se, []Outcome{red}); code != 2 || !strings.Contains(se.String(), "vet: x") {
		t.Fatalf("%d %q", code, se.String())
	}
}

func TestWriteEditNamesEverySkipAndEveryRed(t *testing.T) {
	var so, se strings.Builder
	outs := []Outcome{
		{Job: Job{Name: "types/go"}, State: StateOK, Output: "fine\n"},
		{Job: Job{Name: "test/go"}, State: StateBudget, Output: "part"},
		{Job: Job{Name: "test/gdscript"}, State: StateUnready, Output: "run the Godot editor once to import the project"},
		{Job: Job{Name: "lint/go"}, State: StateFailed, Output: "vet: x"},
		{Job: Job{Name: "lint/sql"}, State: StateBlocked, BlockedBy: "x"},
	}
	if code := WriteEdit(&so, &se, outs); code != 2 {
		t.Fatalf("red lanes block the edit: %d", code)
	}
	if se.String() != "lint/go: failed\nvet: x\nlint/sql: blocked\n" {
		t.Fatalf("%q", se.String())
	}
	want := `{"hookSpecificOutput":{"additionalContext":"loomux hook post-tool-use: lane skipped, the edit budget ran out: test/go\n` +
		`loomux hook post-tool-use: lane skipped, run the Godot editor once to import the project","hookEventName":"PostToolUse"}}` + "\n"
	if so.String() != want {
		t.Fatalf("%q", so.String())
	}
}

// Stdout that is not valid JSON turns a passed hook into a hook-error notice,
// so a run with nothing skipped leaves it empty.
func TestWriteEditGreenIsSilent(t *testing.T) {
	var so, se strings.Builder
	if code := WriteEdit(&so, &se, []Outcome{{Job: Job{Name: "lint/go"}, State: StateOK}}); code != 0 || so.Len() != 0 || se.Len() != 0 {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
}

func TestWriteSkipped(t *testing.T) {
	var b strings.Builder
	writeSkipped(&b, "")
	if b.Len() != 0 {
		t.Fatalf("nothing skipped, nothing written: %q", b.String())
	}
	writeSkipped(&b, "a <b>\nc\n")
	// Marshal escapes markup, as the notice it replaces did.
	want := `{"hookSpecificOutput":{"additionalContext":"a ` + "\\u003cb\\u003e" + `\nc","hookEventName":"PostToolUse"}}` + "\n"
	if b.String() != want {
		t.Fatalf("%q", b.String())
	}
}
