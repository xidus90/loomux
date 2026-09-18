package mutants

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// fixtureSource is the source the expected mutants were measured on with the
// script's own _mutants_of (tools/go_mutants.py under the reference's Python
// 3.14.7): 52 mutants, a1 14, a2 12, a3 19, a4 7. It is not Go that compiles;
// it gathers the cases the script's rules tell apart.
const fixtureSource = "package fixture\n" +
	"\n" +
	"// if a == b {\n" +
	"func decide(a, b int, s, t string, ok bool, ch chan int) int {\n" +
	"\tif a > 0 {\n" +
	"\t\treturn 1\n" +
	"\t}\n" +
	"\tif a != b && s == \"x&&y\" || ok {\n" +
	"\t\treturn 2\n" +
	"\t}\n" +
	"\tif f(a < b && ok) || t != \"\" {\n" +
	"\t\treturn 3\n" +
	"\t} else if a >= b {\n" +
	"\t\treturn 4\n" +
	"\t}\n" +
	"\tif v, ok := any(s).(string); !ok {\n" +
	"\t\treturn len(v)\n" +
	"\t}\n" +
	"\tif r := s[0]; r == '\"' && t != \"\" {\n" +
	"\t\treturn 5\n" +
	"\t}\n" +
	"\tif s == \"\\\"&&\" || t == `a||b` {\n" +
	"\t\treturn 6\n" +
	"\t}\n" +
	"\tfor i := 0; i < b; i++ {\n" +
	"\t\ta <<= 1\n" +
	"\t}\n" +
	"\tx := a == b // a != b\n" +
	"\ty := \"a < b\" + s\n" +
	"\tz := <-ch\n" +
	"\tif a <= b { // trailing comment keeps the brace off the end\n" +
	"\t\treturn 7\n" +
	"\t}\n" +
	"\tif \"é\" < s && t >= \"ü\" {\n" +
	"\t\treturn 8\n" +
	"\t}\n" +
	"\tswitch {\n" +
	"\tcase a > b:\n" +
	"\t\treturn 9\n" +
	"\t}\n" +
	"\thtml := `\n" +
	"<p>\n" +
	"\t<b>`\n" +
	"\t_, _, _, _ = x, y, z, html\n" +
	"\treturn 0\n" +
	"}\n"

// site is what a mutant says, without the file it was read from.
type site struct {
	Family string
	Line   int
	Now    string
}

func sites(ms []Mutant) []site {
	out := make([]site, len(ms))
	for i, m := range ms {
		out[i] = site{m.Family, m.Line, m.Now}
	}
	return out
}

// fixtureSites is _mutants_of(fixture) in the script's order, measured.
var fixtureSites = []site{
	{"a1", 5, "\tif true {"},
	{"a1", 5, "\tif false {"},
	{"a4", 5, "\tif !(a > 0) {"},
	{"a3", 5, "\tif a >= 0 {"},
	{"a1", 8, "\tif true {"},
	{"a1", 8, "\tif false {"},
	{"a4", 8, "\tif !(a != b && s == \"x&&y\" || ok) {"},
	{"a2", 8, "\tif a != b {"},
	{"a2", 8, "\tif s == \"x&&y\" || ok {"},
	{"a2", 8, "\tif a != b && s == \"x&&y\" {"},
	{"a2", 8, "\tif ok {"},
	{"a3", 8, "\tif a != b && s != \"x&&y\" || ok {"},
	{"a3", 8, "\tif a == b && s == \"x&&y\" || ok {"},
	{"a1", 11, "\tif true {"},
	{"a1", 11, "\tif false {"},
	{"a4", 11, "\tif !(f(a < b && ok) || t != \"\") {"},
	{"a2", 11, "\tif f(a < b && ok) {"},
	{"a2", 11, "\tif t != \"\" {"},
	{"a3", 11, "\tif f(a < b && ok) || t == \"\" {"},
	{"a3", 11, "\tif f(a <= b && ok) || t != \"\" {"},
	{"a3", 13, "\t} else if a > b {"},
	{"a1", 16, "\tif true {"},
	{"a1", 16, "\tif false {"},
	{"a4", 16, "\tif !(v, ok := any(s).(string); !ok) {"},
	{"a1", 19, "\tif true {"},
	{"a1", 19, "\tif false {"},
	{"a4", 19, "\tif !(r := s[0]; r == '\"' && t != \"\") {"},
	{"a2", 19, "\tif r := s[0]; r == '\"' {"},
	{"a2", 19, "\tif t != \"\" {"},
	{"a3", 19, "\tif r := s[0]; r != '\"' && t != \"\" {"},
	{"a1", 22, "\tif true {"},
	{"a1", 22, "\tif false {"},
	{"a4", 22, "\tif !(s == \"\\\"&&\" || t == `a||b`) {"},
	{"a2", 22, "\tif s == \"\\\"&&\" {"},
	{"a2", 22, "\tif t == `a||b` {"},
	{"a3", 22, "\tif s != \"\\\"&&\" || t == `a||b` {"},
	{"a3", 25, "\tfor i := 0; i <= b; i++ {"},
	{"a3", 26, "\t\ta << 1"},
	{"a3", 26, "\t\ta <=<= 1"},
	{"a3", 28, "\tx := a != b // a != b"},
	{"a3", 30, "\tz := <=-ch"},
	{"a3", 31, "\tif a < b { // trailing comment keeps the brace off the end"},
	{"a1", 34, "\tif true {"},
	{"a1", 34, "\tif false {"},
	{"a4", 34, "\tif !(\"é\" < s && t >= \"ü\") {"},
	{"a2", 34, "\tif \"é\" < s {"},
	{"a2", 34, "\tif t >= \"ü\" {"},
	{"a3", 34, "\tif \"é\" < s && t > \"ü\" {"},
	{"a3", 34, "\tif \"é\" <= s && t >= \"ü\" {"},
	{"a3", 38, "\tcase a >= b:"},
	{"a3", 43, "\t<=b>`"},
	{"a3", 43, "\t<b>=`"},
}

func TestGenerateMatchesTheScriptOnTheFixture(t *testing.T) {
	ms := Generate("fixture.go.txt", []byte(fixtureSource))
	if got := sites(ms); !slices.Equal(got, fixtureSites) {
		t.Fatalf("got %d mutants:\n%+v", len(got), got)
	}
	families := map[string]int{}
	lines := strings.Split(fixtureSource, "\n")
	for _, m := range ms {
		families[m.Family]++
		if m.Path != "fixture.go.txt" || m.Was != lines[m.Line-1] {
			t.Fatalf("%+v", m)
		}
	}
	if len(ms) != 52 || families["a1"] != 14 || families["a2"] != 12 || families["a3"] != 19 || families["a4"] != 7 {
		t.Fatalf("%d mutants, families %v", len(ms), families)
	}
}

func TestGenerateReadsCRLFAsTheScriptDoes(t *testing.T) {
	crlf := strings.ReplaceAll(fixtureSource, "\n", "\r\n")
	ms := Generate("fixture.go.txt", []byte(crlf))
	if got := sites(ms); !slices.Equal(got, fixtureSites) {
		t.Fatalf("got %d mutants:\n%+v", len(got), got)
	}
	for _, m := range ms {
		if strings.ContainsRune(m.Was, '\r') {
			t.Fatalf("line %d keeps its CR: %q", m.Line, m.Was)
		}
	}
}

// breaks holds every line break Python's str.splitlines knows, and \x1f,
// which it does not know.
const breaks = "package t\n\tif a == 1 {\v\tif b == 2 {\f\tx := c < d\x1c\ty := e > f\x1d\tz := g != h\x1e\tw := i <= j\xc2\x85\tv := k >= l\xe2\x80\xa8\tu := m == n\xe2\x80\xa9\tif o {\r\ts := p < q\x1f\x1fr > s\r\n"

func TestGenerateNumbersLinesLikeTheScript(t *testing.T) {
	// _mutants_of on a file holding breaks, measured: 19 mutants.
	want := []site{
		{"a1", 2, "\tif true {"},
		{"a1", 2, "\tif false {"},
		{"a4", 2, "\tif !(a == 1) {"},
		{"a3", 2, "\tif a != 1 {"},
		{"a1", 3, "\tif true {"},
		{"a1", 3, "\tif false {"},
		{"a4", 3, "\tif !(b == 2) {"},
		{"a3", 3, "\tif b != 2 {"},
		{"a3", 4, "\tx := c <= d"},
		{"a3", 5, "\ty := e >= f"},
		{"a3", 6, "\tz := g == h"},
		{"a3", 7, "\tw := i < j"},
		{"a3", 8, "\tv := k > l"},
		{"a3", 9, "\tu := m != n"},
		{"a1", 10, "\tif true {"},
		{"a1", 10, "\tif false {"},
		{"a4", 10, "\tif !(o) {"},
		{"a3", 11, "\ts := p <= q\x1f\x1fr > s"},
		{"a3", 11, "\ts := p < q\x1f\x1fr >= s"},
	}
	if got := sites(Generate("breaks.go", []byte(breaks))); !slices.Equal(got, want) {
		t.Fatalf("got %d mutants:\n%+v", len(got), got)
	}
}

func TestLineSpansCutWhereSplitlinesCuts(t *testing.T) {
	// Every want is str.splitlines() of the text, measured.
	for _, c := range []struct {
		text string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a\n", []string{"a"}},
		{"\n", []string{""}},
		{"a\r\nb", []string{"a", "b"}},
		{"a\rb", []string{"a", "b"}},
		{"a\n\nb", []string{"a", "", "b"}},
		{"a\vb\fc", []string{"a", "b", "c"}},
		{"a\x1cb\x1dc\x1ed\x1fe", []string{"a", "b", "c", "d\x1fe"}},
		{"a\xc2\x85b\xe2\x80\xa8c\xe2\x80\xa9d", []string{"a", "b", "c", "d"}},
		{"a\r\r\nb", []string{"a", "", "b"}},
		{"a\n\r", []string{"a", ""}},
	} {
		var got []string
		for _, s := range lineSpans([]byte(c.text)) {
			got = append(got, c.text[s.start:s.end])
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%q: got %q, want %q", c.text, got, c.want)
		}
	}
}

// lineSpans is a twin of pytext.SplitLines that keeps offsets; both must cut
// the same pieces.
func TestLineSpansCutWhatPytextSplitLinesCuts(t *testing.T) {
	for _, source := range []string{fixtureSource, breaks, "a\r\r\nb\n\r", ""} {
		var got []string
		for _, s := range lineSpans([]byte(source)) {
			got = append(got, source[s.start:s.end])
		}
		if want := pytext.SplitLines(source); !slices.Equal(got, want) {
			t.Errorf("%q: got %q, want %q", source, got, want)
		}
	}
}

func TestSplitTopFindsTheOperandsOfTheCondition(t *testing.T) {
	// Every want is _split_top(condition, operator), measured.
	for _, c := range []struct {
		condition, operator string
		want                []string
	}{
		{"a && b", "&&", []string{"a", "b"}},
		{"a || b", "&&", nil},
		{"f(a && b) && c", "&&", []string{"f(a && b)", "c"}},
		{`s == "x&&y" && t`, "&&", []string{`s == "x&&y"`, "t"}},
		{`s == "\"&&" && t`, "&&", []string{`s == "\"&&"`, "t"}},
		{"t == `a\\&&b` && u", "&&", []string{"t == `a\\&&b`", "u"}},
		{`r == '"' && t`, "&&", []string{`r == '"'`, "t"}},
		{"a && b || c && d", "||", []string{"a && b", "c && d"}},
		{"a &&", "&&", []string{"a", ""}},
		{") && (", "&&", nil},
		{"m[a && b] && c", "&&", []string{"m[a && b]", "c"}},
		{"x && y && z", "&&", []string{"x", "y", "z"}},
	} {
		if got := splitTop(c.condition, c.operator); !slices.Equal(got, c.want) {
			t.Errorf("%q %s: got %q, want %q", c.condition, c.operator, got, c.want)
		}
	}
}

func TestColumnsFindTheOperatorAsAnOperator(t *testing.T) {
	// Every want is _columns(line, operator), measured.
	for _, c := range []struct {
		line, operator string
		want           []int
	}{
		{"a < b", "<", []int{2}},
		{"a <= b", "<", nil},
		{"a <= b", "<=", []int{2}},
		{"<p>", "<", nil},
		{"<p>", ">", nil},
		{"\t<b>`", "<", []int{1}},
		{"\t<b>`", ">", []int{3}},
		{"x := a == b // a != b", "!=", nil},
		{`y := "a < b" + s`, "<", nil},
		{"a << b", "<", []int{2}},
		{"a >>= b", ">=", []int{3}},
		{"a >>= b", ">", []int{2}},
		{"a == b ==", "==", []int{2}},
		{"x <- ch", "<", []int{2}},
		{"a != b", "!=", []int{2}},
		{"a >= b", ">", nil},
		{`r == '"' && t != ""`, "!=", nil},
	} {
		if got := columns(c.line, c.operator); !slices.Equal(got, c.want) {
			t.Errorf("%q %s: got %v, want %v", c.line, c.operator, got, c.want)
		}
	}
}

func TestMutateReplacesOneLineAndKeepsItsBreak(t *testing.T) {
	crlf := strings.ReplaceAll(fixtureSource, "\n", "\r\n")
	m := Generate("fixture.go.txt", []byte(crlf))[3] // (a3) line 5: if a >= 0 {
	got := string(mutate([]byte(crlf), m))
	if want := strings.Replace(crlf, "\tif a > 0 {\r\n", "\tif a >= 0 {\r\n", 1); got != want {
		t.Fatalf("%q", got)
	}
	last := Mutant{Line: 2, Now: "\tif true {"}
	if got := string(mutate([]byte("x\n\tif a > 0 {"), last)); got != "x\n\tif true {" {
		t.Fatalf("%q", got)
	}
}
