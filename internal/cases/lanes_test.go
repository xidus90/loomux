package cases

import (
	"maps"
	"testing"
)

// The old form is copied from the original recordings (the archive release
// archive/parity-recordings): `unavailable` and `blocked` stand in brackets
// there, as the source of a failed kind, so the old side reads them as red, as
// Python judged them.
func TestKindVerdicts(t *testing.T) {
	old := []byte("lint: ok [config]\ntypes: failed [unavailable]\nno preset\ntest: failed [preset]\nE1\ncoverage: failed [blocked]\n")
	neu := []byte("lint/go: ok [config] 1.2s\nlint/python: ok [preset] 0.1s\ntypes/go: not-applicable [preset] no command\ntest/go: failed [preset] 3.0s\ncoverage/go: blocked [preset] by test/go\n")
	o, n := KindVerdicts(old), KindVerdicts(neu)
	if o["lint"] != "ok" || o["types"] != "red" || o["test"] != "red" || o["coverage"] != "red" {
		t.Fatalf("old %v", o)
	}
	if n["lint"] != "ok" || n["types"] != "neutral" || n["test"] != "red" || n["coverage"] != "red" {
		t.Fatalf("new %v", n)
	}
}

// A red lane decides its kind wherever it stands among the others, and every
// state the check counts as red is read as red.
func TestKindVerdictsRedWinsOverOkAndNeutral(t *testing.T) {
	for _, state := range []string{"failed", "timed-out", "blocked", "missing-tool", "unready", "error"} {
		out := []byte("test/go: ok [preset] 1.0s\ntest/python: " + state + " [preset]\ntest/cpp: not-applicable [preset] no command\n")
		if got := KindVerdicts(out)["test"]; got != "red" {
			t.Errorf("%s: %q", state, got)
		}
	}
	out := []byte("lint/go: not-applicable [preset] no command\nlint/python: ok [preset] 0.1s\nlint/cpp: budget [preset]\n")
	if got := KindVerdicts(out)["lint"]; got != "ok" {
		t.Fatalf("ok beside neutral: %q", got)
	}
}

// Tool output and notes carry no verdict, even where they begin with a word
// that looks like a kind.
func TestKindVerdictsIgnoresEverythingButVerdictLines(t *testing.T) {
	out := []byte("faketool: go\n$ go vet ./...\nnote: the threshold is not enforced\nnothing to check for `style`\ntesting: ok\n  test: failed\n")
	if got := KindVerdicts(out); len(got) != 0 {
		t.Fatalf("%v", got)
	}
	want := map[string]string{"types": "neutral"}
	if got := KindVerdicts([]byte("types/gdscript: unavailable [preset] no tests found\n")); !maps.Equal(got, want) {
		t.Fatalf("%v", got)
	}
}

// The note of a requested kind that had nothing to check fails the check,
// so it is a red verdict for that kind, over any line of its own.
func TestKindVerdictsReadsTheNothingToCheckNoteAsRed(t *testing.T) {
	out := []byte("types/go: not-applicable [preset] no command\nnothing to check for `lint`\nnothing to check for `types`\n")
	want := map[string]string{"lint": "red", "types": "red"}
	if got := KindVerdicts(out); !maps.Equal(got, want) {
		t.Fatalf("%v", got)
	}
}
