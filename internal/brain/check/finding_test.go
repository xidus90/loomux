package check

import (
	"strings"
	"testing"
)

func TestNameCarriesTheAxis(t *testing.T) {
	f := Finding{Axis: AxisOKF, Rule: "type-missing"}
	if got := f.Name(); got != "okf/type-missing" {
		t.Fatalf("Name() = %q, want okf/type-missing", got)
	}
}

func TestExitCodeIsOneOnlyForErrors(t *testing.T) {
	cases := []struct {
		name string
		in   []Finding
		want int
	}{
		{"empty", nil, 0},
		{"warning only", []Finding{{Severity: Warning}}, 0},
		{"note only", []Finding{{Severity: Note}}, 0},
		{"one error among warnings", []Finding{{Severity: Warning}, {Severity: Error}}, 1},
	}
	for _, c := range cases {
		if got := ExitCode(c.in); got != c.want {
			t.Errorf("%s: ExitCode = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestNotesAreHiddenUnlessAsked(t *testing.T) {
	in := []Finding{
		{Relative: "a.md", Axis: AxisOKF, Rule: "no-log", Severity: Note, Message: "m"},
		{Relative: "a.md", Axis: AxisHouse, Rule: "orphan", Severity: Error, Message: "m"},
	}
	if strings.Contains(Render(in, false), "no-log") {
		t.Error("a note reached the default run")
	}
	if !strings.Contains(Render(in, true), "no-log") {
		t.Error("--notes did not show the note")
	}
}

func TestSortIsStableAcrossRuns(t *testing.T) {
	in := []Finding{
		{Scope: "b", Relative: "z.md", Axis: AxisHouse, Rule: "orphan"},
		{Scope: "a", Relative: "z.md", Axis: AxisOKF, Rule: "type-missing"},
		{Scope: "a", Relative: "a.md", Axis: AxisHouse, Rule: "orphan"},
	}
	want := []string{"a/a.md", "a/z.md", "b/z.md"}
	Sort(in)
	for i, f := range in {
		if got := f.Scope + "/" + f.Relative; got != want[i] {
			t.Fatalf("position %d = %q, want %q", i, got, want[i])
		}
	}
}

// Two findings on the same page differ only by axis and rule. Without those
// two tiers the sort leaves them in the order the rules happened to produce
// them, and that order is not part of the contract -- so two runs over an
// unchanged bundle could render two different outputs.
func TestSortBreaksTiesByAxisThenRule(t *testing.T) {
	in := []Finding{
		{Scope: "a", Relative: "a.md", Axis: AxisOKF, Rule: "type-missing"},
		{Scope: "a", Relative: "a.md", Axis: AxisHouse, Rule: "orphan"},
		{Scope: "a", Relative: "a.md", Axis: AxisHouse, Rule: "no-log"},
	}
	want := []string{"house/no-log", "house/orphan", "okf/type-missing"}
	Sort(in)
	for i, f := range in {
		if got := f.Name(); got != want[i] {
			t.Fatalf("position %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestRenderNamesTheAreaOfAFinding(t *testing.T) {
	withScope := []Finding{
		{Scope: "brain", Relative: "a.md", Axis: AxisHouse, Rule: "orphan", Severity: Warning, Message: "m"},
	}
	if got := Render(withScope, false); !strings.Contains(got, "brain/a.md") {
		t.Errorf("Render = %q, want it to name brain/a.md", got)
	}
	// A run over a single file knows no area, and an empty Scope must not
	// leave a bare slash in front of the path.
	withoutScope := []Finding{
		{Relative: "a.md", Axis: AxisHouse, Rule: "orphan", Severity: Warning, Message: "m"},
	}
	if got := Render(withoutScope, false); strings.Contains(got, "/a.md") {
		t.Errorf("Render = %q, want no separator before a.md", got)
	}
}

// The two tests above never let a tier act alone: in TestSortIsStableAcrossRuns
// the axes of the two "a" findings happen to sort the same way as their paths,
// and in TestSortBreaksTiesByAxisThenRule the rule names alone already produce
// the expected order. Dropping the path tier or the axis tier therefore stays
// green there. These two cases put the tiers against each other, so that each
// one is observed on its own.
func TestSortPrefersTheAxisOverTheRule(t *testing.T) {
	in := []Finding{
		{Scope: "a", Relative: "a.md", Axis: AxisOKF, Rule: "a-rule"},
		{Scope: "a", Relative: "a.md", Axis: AxisHouse, Rule: "z-rule"},
	}
	want := []string{"house/z-rule", "okf/a-rule"}
	Sort(in)
	for i, f := range in {
		if got := f.Name(); got != want[i] {
			t.Fatalf("position %d = %q, want %q", i, got, want[i])
		}
	}
}

func TestSortPrefersThePathOverTheAxis(t *testing.T) {
	in := []Finding{
		{Scope: "a", Relative: "z.md", Axis: AxisHouse, Rule: "orphan"},
		{Scope: "a", Relative: "a.md", Axis: AxisOKF, Rule: "orphan"},
	}
	want := []string{"a.md", "z.md"}
	Sort(in)
	for i, f := range in {
		if f.Relative != want[i] {
			t.Fatalf("position %d = %q, want %q", i, f.Relative, want[i])
		}
	}
}
