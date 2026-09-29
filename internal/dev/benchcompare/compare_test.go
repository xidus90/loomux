package benchcompare

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

func timing(name string, cold float64, warm ...float64) benchreport.Timing {
	return benchreport.Summarize(name, cold, warm)
}

func TestCompareSortsCasesByWhoMeasuredThem(t *testing.T) {
	before := []benchreport.Timing{timing("guard", 40, 30, 32), timing("wiki lint", 90, 80, 82), timing("both", 5, 5)}
	after := []benchreport.Timing{timing("guard", 20, 10, 12), timing("graph", 15, 14, 16), timing("both", 6, 6)}
	r, err := Compare(before, after)
	if err != nil {
		t.Fatal(err)
	}
	names := func(ts []benchreport.Timing) []string {
		var out []string
		for _, x := range ts {
			out = append(out, x.Name)
		}
		return out
	}
	if len(r.Compared) != 2 || r.Compared[0].Name != "guard" || r.Compared[1].Name != "both" {
		t.Fatalf("compared = %+v, want guard then both in the order of the earlier run", r.Compared)
	}
	if got := names(r.Added); len(got) != 1 || got[0] != "graph" {
		t.Errorf("added = %v, want [graph]", got)
	}
	if got := names(r.Dropped); len(got) != 1 || got[0] != "wiki lint" {
		t.Errorf("dropped = %v, want [wiki lint]", got)
	}
}

func TestCompareRefusesTheSameNameTwiceInOneRun(t *testing.T) {
	dup := []benchreport.Timing{timing("guard", 1, 1), timing("guard", 2, 2)}
	if _, err := Compare(dup, nil); err == nil || !strings.Contains(err.Error(), `"guard"`) {
		t.Errorf("before: err = %v, want one naming guard", err)
	}
	if _, err := Compare(nil, dup); err == nil {
		t.Errorf("after: no error for a duplicated name")
	}
}

func TestCompareIgnoresACaseThatDoesNotApply(t *testing.T) {
	no := false
	gone := timing("x", 1, 1)
	gone.Applicable = &no
	r, err := Compare([]benchreport.Timing{gone}, []benchreport.Timing{timing("x", 1, 1)})
	if err != nil {
		t.Fatal(err)
	}
	// The earlier side does not apply: the later one is new, nothing is compared.
	if len(r.Compared) != 0 || len(r.Added) != 1 || len(r.Dropped) != 0 {
		t.Errorf("result = %+v", r)
	}
}

func TestFactorIsHowManyTimesFasterTheLaterRunIs(t *testing.T) {
	cases := []struct {
		before, after, want float64
	}{
		{30, 10, 3},
		{10, 30, 1.0 / 3},
		{0, 10, 0},
		{10, 0, 0},
		{-1, 10, 0},
	}
	for _, c := range cases {
		if got := Factor(c.before, c.after); got != c.want {
			t.Errorf("Factor(%v, %v) = %v, want %v", c.before, c.after, got, c.want)
		}
	}
}

func TestMarkdownCarriesTitleValuesAndTheThreeLists(t *testing.T) {
	r, _ := Compare(
		[]benchreport.Timing{timing("guard", 40, 30, 30, 30, 30, 30), timing("wiki lint", 90, 80, 80, 80, 80, 80)},
		[]benchreport.Timing{timing("guard", 20, 10, 10, 10, 10, 10), timing("graph", 15, 14, 14, 14, 14, 14)})
	de, err := Markdown(r, "Beispielprojekt 3", "de")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Beispielprojekt 3", "## Verglichen", "## Neu", "## Weggefallen",
		"guard", "3.00×", "graph", "wiki lint", "1 verglichen", "1 schneller"} {
		if !strings.Contains(de, want) {
			t.Errorf("de: %q missing in\n%s", want, de)
		}
	}
	en, _ := Markdown(r, "Example project 3", "en")
	for _, want := range []string{"## Compared", "## New", "## Dropped", "1 compared", "1 faster"} {
		if !strings.Contains(en, want) {
			t.Errorf("en: %q missing in\n%s", want, en)
		}
	}
	if _, err := Markdown(r, "x", "fr"); err == nil {
		t.Errorf("an unknown language must be refused")
	}
}

func TestMarkdownCountsSlowerAndUnclearApart(t *testing.T) {
	r, _ := Compare(
		[]benchreport.Timing{timing("a", 1, 10), timing("b", 1, 0)},
		[]benchreport.Timing{timing("a", 1, 30), timing("b", 1, 5)})
	out, _ := Markdown(r, "t", "en")
	if !strings.Contains(out, "1 slower") || !strings.Contains(out, "1 unclear") {
		t.Errorf("summary wrong:\n%s", out)
	}
}

func names(ts []benchreport.Timing) []string {
	var out []string
	for _, x := range ts {
		out = append(out, x.Name)
	}
	return out
}

func rowNames(rows []Row) []string {
	var out []string
	for _, x := range rows {
		out = append(out, x.Name)
	}
	return out
}

func TestCompareKeepsTheOrderOfEachRun(t *testing.T) {
	before := []benchreport.Timing{timing("c", 1, 1), timing("a", 1, 1), timing("d", 1, 1), timing("b", 1, 1), timing("z", 1, 1)}
	after := []benchreport.Timing{timing("y", 1, 1), timing("b", 1, 1), timing("x", 1, 1), timing("c", 1, 1), timing("w", 1, 1)}
	r, err := Compare(before, after)
	if err != nil {
		t.Fatal(err)
	}
	// Compared and dropped follow the earlier run, new follows the later one.
	if got := strings.Join(rowNames(r.Compared), ","); got != "c,b" {
		t.Errorf("compared = %s, want c,b", got)
	}
	if got := strings.Join(names(r.Dropped), ","); got != "a,d,z" {
		t.Errorf("dropped = %s, want a,d,z", got)
	}
	if got := strings.Join(names(r.Added), ","); got != "y,x,w" {
		t.Errorf("added = %s, want y,x,w", got)
	}
}

func TestCompareKeepsBothTimingsOfAPair(t *testing.T) {
	r, err := Compare([]benchreport.Timing{timing("a", 1, 2)}, []benchreport.Timing{timing("a", 3, 4)})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Compared) != 1 || r.Compared[0].Before.ColdMS != 1 || r.Compared[0].After.ColdMS != 3 {
		t.Errorf("compared = %+v, want before cold 1 and after cold 3", r.Compared)
	}
}

func TestCompareTreatsEachSideWhereACaseDoesNotApply(t *testing.T) {
	no := false
	off := func(name string) benchreport.Timing {
		x := timing(name, 1, 1)
		x.Applicable = &no
		return x
	}
	yes := true
	on := timing("flagged", 1, 1)
	on.Applicable = &yes
	before := []benchreport.Timing{timing("later-off", 1, 1), off("both-off"), on}
	after := []benchreport.Timing{off("later-off"), off("both-off"), off("only-later-off"), timing("flagged", 2, 2)}
	r, err := Compare(before, after)
	if err != nil {
		t.Fatal(err)
	}
	// A case that does not apply in the later run is gone from it, not new;
	// one that does not apply in either run is nowhere; an explicit true is nil.
	if got := strings.Join(names(r.Dropped), ","); got != "later-off" {
		t.Errorf("dropped = %q, want later-off", got)
	}
	if len(r.Added) != 0 {
		t.Errorf("added = %v, want none", names(r.Added))
	}
	if got := strings.Join(rowNames(r.Compared), ","); got != "flagged" {
		t.Errorf("compared = %q, want flagged", got)
	}
}

func TestMarkdownSummaryKeepsEveryCountApart(t *testing.T) {
	var before, after []benchreport.Timing
	pair := func(name string, b, a float64) {
		before = append(before, timing(name, 1, b))
		after = append(after, timing(name, 1, a))
	}
	pair("f1", 10, 5)  // 2.0: faster
	pair("f2", 9, 6)   // 1.5: faster
	pair("s1", 5, 10)  // 0.5: slower
	pair("s2", 6, 9)   // 0.67: slower
	pair("s3", 5, 6)   // 0.83: slower
	pair("u1", 10, 10) // 1.0 exactly: neither faster nor slower
	for _, n := range []string{"n1", "n2", "n3", "n4"} {
		after = append(after, timing(n, 1, 1))
	}
	for _, n := range []string{"d1", "d2", "d3", "d4", "d5"} {
		before = append(before, timing(n, 1, 1))
	}
	r, err := Compare(before, after)
	if err != nil {
		t.Fatal(err)
	}
	en, _ := Markdown(r, "t", "en")
	if want := "6 compared, 2 faster, 3 slower, 1 unclear, 4 new, 5 dropped\n"; !strings.Contains(en, want) {
		t.Errorf("en summary: %q missing in\n%s", want, en)
	}
	de, _ := Markdown(r, "t", "de")
	if want := "6 verglichen, 2 schneller, 3 langsamer, 1 unklar, 4 neu, 5 weggefallen\n"; !strings.Contains(de, want) {
		t.Errorf("de summary: %q missing in\n%s", want, de)
	}
}

// goldenResult gives every figure of a row its own value, so a swapped or
// mislabelled column changes the text.
func goldenResult() Result {
	return Result{
		Compared: []Row{
			{Name: "guard",
				Before: benchreport.Timing{Name: "guard", ColdMS: 40, WarmMS: []float64{30, 50, 70}, MedianMS: 55, ExitCodes: []int{0}},
				After:  benchreport.Timing{Name: "guard", ColdMS: 20, WarmMS: []float64{8, 12, 16}, MedianMS: 9, ExitCodes: []int{0, 1}}},
			{Name: "idle",
				Before: benchreport.Timing{Name: "idle", ColdMS: 0, MedianMS: 3},
				After:  benchreport.Timing{Name: "idle", ColdMS: 7, WarmMS: []float64{5}, MedianMS: 4, ExitCodes: []int{2}}},
		},
		Added:   []benchreport.Timing{{Name: "graph", ColdMS: 15, WarmMS: []float64{14, 16}, MedianMS: 13}},
		Dropped: []benchreport.Timing{{Name: "wiki lint", ColdMS: 90, WarmMS: []float64{80, 82}, MedianMS: 70}},
	}
}

func TestMarkdownIsExactlyThisText(t *testing.T) {
	cases := []struct {
		name, lang string
		r          Result
		want       []string
	}{
		{"de", "de", goldenResult(), []string{
			"# T",
			"",
			"2 verglichen, 1 schneller, 0 langsamer, 1 unklar, 1 neu, 1 weggefallen",
			"",
			"## Verglichen",
			"",
			"| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |",
			"|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|",
			"| guard | 40.0 ms | 20.0 ms | 2.00× | 50.0 ms | 12.0 ms | 4.17× | 55.0 ms | 9.0 ms | [0] | [0 1] |",
			"| idle | 0.0 ms | 7.0 ms | – | 0.0 ms | 5.0 ms | – | 3.0 ms | 4.0 ms | [] | [2] |",
			"",
			"## Neu",
			"",
			"| Fall | kalt | warm Ø | Median |",
			"|---|---:|---:|---:|",
			"| graph | 15.0 ms | 15.0 ms | 13.0 ms |",
			"",
			"## Weggefallen",
			"",
			"| Fall | kalt | warm Ø | Median |",
			"|---|---:|---:|---:|",
			"| wiki lint | 90.0 ms | 81.0 ms | 70.0 ms |",
			"",
		}},
		{"en", "en", goldenResult(), []string{
			"# T",
			"",
			"2 compared, 1 faster, 0 slower, 1 unclear, 1 new, 1 dropped",
			"",
			"## Compared",
			"",
			"| case | cold before | cold after | × | warm mean before | warm mean after | × | median before | median after | exit before | exit after |",
			"|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|",
			"| guard | 40.0 ms | 20.0 ms | 2.00× | 50.0 ms | 12.0 ms | 4.17× | 55.0 ms | 9.0 ms | [0] | [0 1] |",
			"| idle | 0.0 ms | 7.0 ms | – | 0.0 ms | 5.0 ms | – | 3.0 ms | 4.0 ms | [] | [2] |",
			"",
			"## New",
			"",
			"| case | cold | warm mean | median |",
			"|---|---:|---:|---:|",
			"| graph | 15.0 ms | 15.0 ms | 13.0 ms |",
			"",
			"## Dropped",
			"",
			"| case | cold | warm mean | median |",
			"|---|---:|---:|---:|",
			"| wiki lint | 90.0 ms | 81.0 ms | 70.0 ms |",
			"",
		}},
		{"de empty", "de", Result{}, []string{
			"# T",
			"",
			"0 verglichen, 0 schneller, 0 langsamer, 0 unklar, 0 neu, 0 weggefallen",
			"",
			"## Verglichen",
			"",
			"| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |",
			"|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|",
			"",
			"## Neu",
			"",
			"keine",
			"",
			"## Weggefallen",
			"",
			"keine",
			"",
		}},
		{"en empty", "en", Result{}, []string{
			"# T",
			"",
			"0 compared, 0 faster, 0 slower, 0 unclear, 0 new, 0 dropped",
			"",
			"## Compared",
			"",
			"| case | cold before | cold after | × | warm mean before | warm mean after | × | median before | median after | exit before | exit after |",
			"|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|",
			"",
			"## New",
			"",
			"none",
			"",
			"## Dropped",
			"",
			"none",
			"",
		}},
	}
	for _, c := range cases {
		got, err := Markdown(c.r, "T", c.lang)
		if err != nil {
			t.Fatal(err)
		}
		if want := strings.Join(c.want, "\n"); got != want {
			t.Errorf("%s: got\n%s\nwant\n%s", c.name, got, want)
		}
	}
}
