package linecov

import (
	"math"
	"slices"
	"testing"
)

func TestReportSumsHitsPerLineAndKeepsAFileWithoutLines(t *testing.T) {
	r := Report{}
	r.Add("a.gd", 3, 2)
	r.Add("a.gd", 3, 5)
	r.Add("b.gd", 1, 0)
	r.File("c.gd")
	if r["a.gd"][3] != 7 || r["b.gd"][1] != 0 {
		t.Fatalf("%v", r)
	}
	if _, ok := r["c.gd"]; !ok {
		t.Fatal("a file without lines was dropped")
	}
	if r.Lines() != 2 {
		t.Fatalf("lines %d", r.Lines())
	}
}

func TestReportSaturatesInsteadOfWrapping(t *testing.T) {
	r := Report{}
	r.Add("a", 1, math.MaxUint64)
	r.Add("a", 1, 1)
	if r["a"][1] != math.MaxUint64 {
		t.Fatal(r["a"][1])
	}
	if addSat(2, 3) != 5 {
		t.Fatal(addSat(2, 3))
	}
}

func TestFindingPrintsLineFileAndBareForms(t *testing.T) {
	for _, row := range []struct {
		f    Finding
		want string
	}{
		{Finding{"a.gd", 3, "not covered"}, "a.gd:3 not covered"},
		{Finding{"a.gd", 0, "is absent from the report"}, "a.gd is absent from the report"},
		{Finding{"", 0, "no coverage report found"}, "no coverage report found"},
	} {
		if got := row.f.String(); got != row.want {
			t.Errorf("%+v: %q", row.f, got)
		}
	}
}

func TestSortFindingsOrdersByPathLineTextAndDropsDuplicates(t *testing.T) {
	got := SortFindings([]Finding{{"b", 1, "x"}, {"a", 2, "y"}, {"a", 1, "z"}, {"a", 1, "y"}, {"a", 1, "y"}})
	want := []Finding{{"a", 1, "y"}, {"a", 1, "z"}, {"a", 2, "y"}, {"b", 1, "x"}}
	if !slices.Equal(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestTextReplacesInvalidUTF8(t *testing.T) {
	if got := text([]byte("a\xffb")); got != "a�b" {
		t.Fatalf("%q", got)
	}
}
