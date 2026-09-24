package edit

import "testing"

func TestDiffShowsChangedLinesWithContext(t *testing.T) {
	got := Diff("a\nb\nc\nd\n", "a\nB\nc\nd\n")
	want := "  a\n- b\n+ B\n  c\n"
	if got != want {
		t.Fatalf("got\n%s", got)
	}
	if Diff("x\n", "x\n") != "" {
		t.Fatal("no change, no diff")
	}
}

func TestDiffAtTheEdgesHasNoContextToShow(t *testing.T) {
	if got := Diff("a\n", "a\nb\n"); got != "  a\n+ b\n" {
		t.Fatalf("%q", got)
	}
	if got := Diff("a\nb\n", "b\n"); got != "- a\n  b\n" {
		t.Fatalf("%q", got)
	}
}

// TestDiffWhereOneSideIsAPrefixOfTheOther: the common head and tail run into
// the end of the shorter side and stop there.
func TestDiffWhereOneSideIsAPrefixOfTheOther(t *testing.T) {
	if got := Diff("a\nb\n", "a\n"); got != "  a\n- b\n" {
		t.Fatalf("%q", got)
	}
	if got := Diff("b\n", "a\nb\n"); got != "+ a\n  b\n" {
		t.Fatalf("%q", got)
	}
}
