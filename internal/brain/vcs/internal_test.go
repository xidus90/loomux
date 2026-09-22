package vcs

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// Called directly, because git cannot reach it: os/exec answers empty output
// with an empty slice that is not nil (measured 2026-09-20, Go 1.27,
// windows/amd64), so no fixture can hand `baseline` a nil on a successful call.
// The promise is this package's own all the same -- nil means "no baseline",
// and a committed empty file is a baseline -- and a promise that rests only on
// what the standard library happens to do is not held by anything. Drop the
// guard in `baseline` and this case goes red; the end-to-end case over a
// committed empty file does not.
func TestBaselineTurnsNoOutputIntoAnEmptyBaselineAndNotIntoNone(t *testing.T) {
	got, err := baseline(nil, nil)

	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	if got == nil {
		t.Fatal("baseline(nil, nil) = nil; want an empty slice, which is a baseline")
	}
	if len(got) != 0 {
		t.Fatalf("baseline(nil, nil) = %q; want it empty", got)
	}
}

// The other half of the same promise: a failure is the nil, and it is not an
// error -- a vault without git, an unborn branch and a path git never saw all
// arrive here.
func TestBaselineTurnsAFailureIntoNoBaselineAndNotIntoAnError(t *testing.T) {
	got, err := baseline([]byte("ignored"), errors.New("git said no"))

	if got != nil || err != nil {
		t.Fatalf("baseline = %q, %v; want nil, nil", got, err)
	}
}

// Both ends of the length rule are inclusive: four digits is the shortest
// abbreviation git accepts, sixty-four a full SHA-256 name. Called directly,
// because a range over a real repository cannot tell a refused name from an
// unknown one without a SHA-256 repository to hand.
func TestObjectNameTakesFourToSixtyFourDigits(t *testing.T) {
	for _, testCase := range []struct {
		name string
		want bool
	}{
		{strings.Repeat("a", 3), false},
		{strings.Repeat("a", 4), true},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false},
	} {
		if got := objectName(testCase.name); got != testCase.want {
			t.Errorf("objectName(%d digits) = %v, want %v", len(testCase.name), got, testCase.want)
		}
	}
}

// A git that exits non-zero and says nothing on stderr adds nothing to the
// message -- no dangling ": ". Called directly: no fixture makes git fail
// silently on demand.
func TestSaidAddsNothingForASilentFailure(t *testing.T) {
	if got := said(&exec.ExitError{}); got != "" {
		t.Fatalf("said(silent exit) = %q, want it empty", got)
	}
}
