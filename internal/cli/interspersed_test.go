package cli

import (
	"flag"
	"io"
	"slices"
	"strings"
	"testing"
)

// interspersedFlags is a flag set of the shape `approve` has: one flag that
// takes a value and one that does not.
func interspersedFlags() (*flag.FlagSet, *string, *bool) {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags, flags.String("amend", "", ""), flags.Bool("reject", false, "")
}

// Flags stand before, between and after the positionals, and a flag's value
// is never taken for a positional, as argparse reads a command line.
func TestParseInterspersedReadsFlagsOnBothSides(t *testing.T) {
	flags, amend, reject := interspersedFlags()
	free, err := parseInterspersed(flags, []string{"--reject", "one", "--amend", "file.md", "two"})
	if err != nil {
		t.Fatalf("parseInterspersed: %v", err)
	}
	if !slices.Equal(free, []string{"one", "two"}) || *amend != "file.md" || !*reject {
		t.Fatalf("free = %q, amend = %q, reject = %v", free, *amend, *reject)
	}
}

// Past `--` everything is positional, whatever it looks like: resuming the
// parse there would make the terminator mean its opposite.
func TestParseInterspersedStopsAtTheTerminator(t *testing.T) {
	flags, amend, reject := interspersedFlags()
	free, err := parseInterspersed(flags, []string{"one", "--", "--reject", "--amend", "x"})
	if err != nil {
		t.Fatalf("parseInterspersed: %v", err)
	}
	if !slices.Equal(free, []string{"one", "--reject", "--amend", "x"}) || *amend != "" || *reject {
		t.Fatalf("free = %q, amend = %q, reject = %v", free, *amend, *reject)
	}
}

func TestParseInterspersedRefusesAnUnknownFlagAfterAPositional(t *testing.T) {
	flags, _, _ := interspersedFlags()
	if _, err := parseInterspersed(flags, []string{"one", "--nope"}); err == nil {
		t.Fatal("parseInterspersed accepted an unknown flag")
	}
}

func TestParseInterspersedAnswersNothingForNoArguments(t *testing.T) {
	flags, _, _ := interspersedFlags()
	free, err := parseInterspersed(flags, nil)
	if err != nil || len(free) != 0 {
		t.Fatalf("free = %q, err = %v", free, err)
	}
}

// argparse refuses `--amend --` and `--amend --reject`: what looks like an
// option is no value. `flag` would take it as one and hand it on as the
// amendment's path.
func TestParseInterspersedRefusesAnOptionAsAValue(t *testing.T) {
	for _, args := range [][]string{
		{"one", "--amend", "--"},
		{"-amend", "--", "one"},
		{"--amend", "--reject", "one"},
		{"one", "--amend", "-x.md"},
		{"--amend", "--nope=1"},
		{"--amend", "-inf"},
		{"--amend", "-.x"},
	} {
		flags, _, _ := interspersedFlags()
		var said strings.Builder
		flags.SetOutput(&said)
		if _, err := parseInterspersed(flags, args); err == nil {
			t.Fatalf("%q: accepted an option as the value of --amend", args)
		}
		if !strings.Contains(said.String(), "argument --amend: expected one argument") {
			t.Fatalf("%q: output = %q", args, said.String())
		}
	}
}

// `--amend=--` names the value outright, argparse takes it, and a `--` after
// a flag without a value, or after a value, stays the terminator.
func TestParseInterspersedKeepsTheTerminatorWhereItIsOne(t *testing.T) {
	flags, amend, reject := interspersedFlags()
	free, err := parseInterspersed(flags, []string{"--amend=--", "--reject", "--", "--amend", "x"})
	if err != nil || *amend != "--" || !*reject || !slices.Equal(free, []string{"--amend", "x"}) {
		t.Fatalf("free = %q, amend = %q, reject = %v, err = %v", free, *amend, *reject, err)
	}
	flags, amend, _ = interspersedFlags()
	free, err = parseInterspersed(flags, []string{"--amend", "file.md", "--", "--amend", "--"})
	if err != nil || *amend != "file.md" || !slices.Equal(free, []string{"--amend", "--"}) {
		t.Fatalf("free = %q, amend = %q, err = %v", free, *amend, err)
	}
}

// What argparse does take as a value although it starts with `-`: a lone
// `-`, a word that starts like a number (Python 3.14 matches `^-\.?\d` as a
// prefix, with Unicode digits; measured against the reference 2026-09-23),
// and a word with a space in it.
func TestParseInterspersedTakesADashValueArgparseTakes(t *testing.T) {
	for _, value := range []string{"-", "-5", "-1.5", "-.5", "-a b.md", "-1e5", "-1.", "-5x.md", "-2026-notes.md", "-.5x", "-١x"} {
		flags, amend, _ := interspersedFlags()
		free, err := parseInterspersed(flags, []string{"one", "--amend", value})
		if err != nil || *amend != value || !slices.Equal(free, []string{"one"}) {
			t.Fatalf("%q: free = %q, amend = %q, err = %v", value, free, *amend, err)
		}
	}
}
