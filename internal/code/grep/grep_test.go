package grep_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/model"
)

func sampleGrepGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "pkg/service.go", Path: "pkg/service.go", Name: "service.go", Kind: model.KindFile, Span: "L1-L100"},
			{ID: "pkg/service.go#Foo", Path: "pkg/service.go", Name: "Foo", Kind: "func", Span: "L4-L6"},
			{ID: "pkg/service.go#Bar", Path: "pkg/service.go", Name: "Bar", Kind: "func", Span: "L8-L11"},
			{ID: "pkg/other.go", Path: "pkg/other.go", Name: "other.go", Kind: model.KindFile, Span: "L1-L50"},
			{ID: "other/file.go", Path: "other/file.go", Name: "file.go", Kind: model.KindFile, Span: "L1-L30"},
		},
		Edges: []model.Edge{
			// Bar has 2 callers (inDegree = 2)
			{Source: "c1", Target: "pkg/service.go#Bar", Relation: model.RelationCalls},
			{Source: "c2", Target: "pkg/service.go#Bar", Relation: model.RelationCalls},
			// Foo has 1 caller (inDegree = 1)
			{Source: "c1", Target: "pkg/service.go#Foo", Relation: model.RelationCalls},
		},
	}
}

func sampleFiles() map[string]string {
	return map[string]string{
		"pkg/service.go": `package pkg
// target at file level line 2

func Foo() {
	// target inside Foo line 12
}

func Bar() {
	// target inside Bar line 32
	// target again in Bar line 33
}
`,
		"pkg/other.go": `package pkg
// no match here
`,
		"other/file.go": `package other
// target in other package line 2
`,
	}
}

func makeReader(files map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		content, ok := files[p]
		if !ok {
			return nil, errors.New("file not found")
		}
		return []byte(content), nil
	}
}

func TestSearchSymbolAndFileGroups(t *testing.T) {
	g := sampleGrepGraph()
	x := blast.New(g)
	spans := model.FileSpans(g)
	reader := makeReader(sampleFiles())

	res, err := grep.Search(g, x, spans, "target", grep.Options{In: "pkg"}, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalHits != 4 {
		t.Fatalf("want 4 total hits, got %d", res.TotalHits)
	}
	if len(res.Groups) != 3 {
		t.Fatalf("want 3 groups (Bar, Foo, file level), got %d", len(res.Groups))
	}

	// Group 0: Bar (inDegree 2)
	if res.Groups[0].Symbol == nil || res.Groups[0].Symbol.Name != "Bar" || res.Groups[0].InDegree != 2 {
		t.Errorf("group 0: want Bar inDegree 2, got %v", res.Groups[0])
	}
	if len(res.Groups[0].Hits) != 2 || res.Groups[0].Hits[0].Line != 9 || res.Groups[0].Hits[1].Line != 10 {
		t.Errorf("group 0 hits mismatch: %v", res.Groups[0].Hits)
	}

	// Group 1: Foo (inDegree 1)
	if res.Groups[1].Symbol == nil || res.Groups[1].Symbol.Name != "Foo" || res.Groups[1].InDegree != 1 {
		t.Errorf("group 1: want Foo inDegree 1, got %v", res.Groups[1])
	}
	if len(res.Groups[1].Hits) != 1 || res.Groups[1].Hits[0].Line != 5 {
		t.Errorf("group 1 hits mismatch: %v", res.Groups[1].Hits)
	}

	// Group 2: File level (Symbol: nil, inDegree 0)
	if res.Groups[2].Symbol != nil || res.Groups[2].InDegree != 0 {
		t.Errorf("group 2: want file level, got %v", res.Groups[2])
	}
	if len(res.Groups[2].Hits) != 1 || res.Groups[2].Hits[0].Line != 2 {
		t.Errorf("group 2 hits mismatch: %v", res.Groups[2].Hits)
	}
}

func TestSearchLineTruncation160(t *testing.T) {
	longLine := "// " + strings.Repeat("A", 150) + "target" + strings.Repeat("B", 50)
	files := map[string]string{
		"a.go": longLine + "\n",
	}
	g := &model.Graph{
		Nodes: []model.Node{
			{ID: "a.go", Path: "a.go", Name: "a.go", Kind: model.KindFile, Span: "L1-L10"},
		},
	}
	x := blast.New(g)
	reader := makeReader(files)

	res, err := grep.Search(g, x, nil, "target", grep.Options{}, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Groups) != 1 || len(res.Groups[0].Hits) != 1 {
		t.Fatalf("want 1 hit, got %v", res)
	}
	hitText := res.Groups[0].Hits[0].Text
	if len([]rune(hitText)) != 160 {
		t.Errorf("want hitText capped at 160 runes, got %d runes", len([]rune(hitText)))
	}
}

func TestSearchMaxHitsAndTruncation(t *testing.T) {
	files := map[string]string{
		"a.go": "target 1\ntarget 2\ntarget 3\ntarget 4\ntarget 5\n",
	}
	g := &model.Graph{
		Nodes: []model.Node{
			{ID: "a.go", Path: "a.go", Name: "a.go", Kind: model.KindFile, Span: "L1-L10"},
		},
	}
	x := blast.New(g)
	reader := makeReader(files)

	res, err := grep.Search(g, x, nil, "target", grep.Options{MaxHits: 3}, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalHits != 3 {
		t.Errorf("want TotalHits=3, got %d", res.TotalHits)
	}
	if res.TruncatedHits != 2 {
		t.Errorf("want TruncatedHits=2, got %d", res.TruncatedHits)
	}
}

func TestSearchFixedAndIgnoreCase(t *testing.T) {
	files := map[string]string{
		"a.go": "Target[0]\ntarget[1]\n",
	}
	g := &model.Graph{
		Nodes: []model.Node{
			{ID: "a.go", Path: "a.go", Name: "a.go", Kind: model.KindFile, Span: "L1-L10"},
		},
	}
	x := blast.New(g)
	reader := makeReader(files)

	// Fixed string search for "Target[" (ignoring case)
	res, err := grep.Search(g, x, nil, "target[", grep.Options{Fixed: true, IgnoreCase: true}, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalHits != 2 {
		t.Fatalf("want 2 hits for fixed case-insensitive search, got %d", res.TotalHits)
	}

	// Regex search with IgnoreCase
	res, err = grep.Search(g, x, nil, "tArGeT\\[\\d\\]", grep.Options{Fixed: false, IgnoreCase: true}, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalHits != 2 {
		t.Fatalf("want 2 hits for regex case-insensitive search, got %d", res.TotalHits)
	}
}

func TestSearchGroupTieBreakers(t *testing.T) {
	files := map[string]string{
		"pkg/b.go": "// file hit line 1\nfunc Beta() { match() }\nfunc Alpha() { match() }\n",
		"pkg/a.go": "func Gamma() { match() }\n",
	}
	g := &model.Graph{
		Nodes: []model.Node{
			{ID: "pkg/b.go", Path: "pkg/b.go", Name: "b.go", Kind: model.KindFile, Span: "L1-L10"},
			{ID: "pkg/b.go#Beta", Path: "pkg/b.go", Name: "Beta", Kind: "func", Span: "L2-L2"},
			{ID: "pkg/b.go#Alpha", Path: "pkg/b.go", Name: "Alpha", Kind: "func", Span: "L3-L3"},
			{ID: "pkg/a.go", Path: "pkg/a.go", Name: "a.go", Kind: model.KindFile, Span: "L1-L10"},
			{ID: "pkg/a.go#Gamma", Path: "pkg/a.go", Name: "Gamma", Kind: "func", Span: "L1-L1"},
		},
	}
	x := blast.New(g)
	reader := makeReader(files)

	res, err := grep.Search(g, x, nil, "(match|file hit)", grep.Options{}, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Groups) != 4 {
		t.Fatalf("want 4 groups, got %d: %v", len(res.Groups), res.Groups)
	}

	// 1. pkg/a.go#Gamma (path pkg/a.go < pkg/b.go)
	if res.Groups[0].Path != "pkg/a.go" || res.Groups[0].Symbol.Name != "Gamma" {
		t.Errorf("group 0 mismatch: %v", res.Groups[0])
	}
	// 2. pkg/b.go#Alpha (symbol name Alpha < Beta)
	if res.Groups[1].Path != "pkg/b.go" || res.Groups[1].Symbol.Name != "Alpha" {
		t.Errorf("group 1 mismatch: %v", res.Groups[1])
	}
	// 3. pkg/b.go#Beta (symbol before file level)
	if res.Groups[2].Path != "pkg/b.go" || res.Groups[2].Symbol.Name != "Beta" {
		t.Errorf("group 2 mismatch: %v", res.Groups[2])
	}
	// 4. pkg/b.go file level (Symbol: nil)
	if res.Groups[3].Path != "pkg/b.go" || res.Groups[3].Symbol != nil {
		t.Errorf("group 3 mismatch: %v", res.Groups[3])
	}
}

func TestSearchRegexRejections(t *testing.T) {
	g := sampleGrepGraph()
	x := blast.New(g)
	reader := makeReader(sampleFiles())

	// Lookahead
	_, err := grep.Search(g, x, nil, "foo(?=bar)", grep.Options{}, reader)
	if err == nil || !strings.Contains(err.Error(), "lookaround") {
		t.Fatalf("want lookaround error, got %v", err)
	}

	// Negative lookahead
	_, err = grep.Search(g, x, nil, "foo(?!bar)", grep.Options{}, reader)
	if err == nil || !strings.Contains(err.Error(), "lookaround") {
		t.Fatalf("want lookaround error, got %v", err)
	}

	// Lookbehind
	_, err = grep.Search(g, x, nil, "(?<=foo)bar", grep.Options{}, reader)
	if err == nil || !strings.Contains(err.Error(), "lookaround") {
		t.Fatalf("want lookaround error, got %v", err)
	}

	// Negative lookbehind
	_, err = grep.Search(g, x, nil, "(?<!foo)bar", grep.Options{}, reader)
	if err == nil || !strings.Contains(err.Error(), "lookaround") {
		t.Fatalf("want lookaround error, got %v", err)
	}

	// Backreference
	_, err = grep.Search(g, x, nil, `(foo)\1`, grep.Options{}, reader)
	if err == nil || !strings.Contains(err.Error(), "backreference") {
		t.Fatalf("want backreference error, got %v", err)
	}

	// Invalid regex syntax
	_, err = grep.Search(g, x, nil, `[unclosed`, grep.Options{}, reader)
	if err == nil || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("want invalid regex error, got %v", err)
	}
}

func TestSearchEdgeCases(t *testing.T) {
	g := sampleGrepGraph()
	x := blast.New(g)
	reader := makeReader(sampleFiles())

	// Empty pattern
	_, err := grep.Search(g, x, nil, "", grep.Options{}, reader)
	if err == nil {
		t.Fatal("want error for empty pattern, got nil")
	}

	// Unindexed prefix
	_, err = grep.Search(g, x, nil, "target", grep.Options{In: "nonexistent"}, reader)
	if err == nil || !strings.Contains(err.Error(), "prefix not indexed") {
		t.Fatalf("want unindexed prefix error, got %v", err)
	}

	// Nil graph
	res, err := grep.Search(nil, nil, nil, "target", grep.Options{}, reader)
	if err != nil || res.TotalHits != 0 {
		t.Fatalf("want empty result for nil graph, got %v err=%v", res, err)
	}

	// Nil reader
	res, err = grep.Search(g, x, nil, "target", grep.Options{}, nil)
	if err != nil || res.TotalHits != 0 {
		t.Fatalf("want empty result for nil reader, got %v err=%v", res, err)
	}

	// Unreadable file count
	errReader := func(string) ([]byte, error) { return nil, errors.New("read error") }
	res, err = grep.Search(g, x, nil, "target", grep.Options{}, errReader)
	if err != nil || res.Unreadable == 0 {
		t.Fatalf("want unreadable counted, got %v err=%v", res, err)
	}
}

// The rejections read the pattern the way RE2 does: a backslash escapes the
// character after it, so `\\1` is a literal backslash before a 1 and `\(?=` an
// optional literal parenthesis, while a third backslash or an unescaped group
// brings the unsupported form back.
func TestSearchRejectsOnlyUnescapedForms(t *testing.T) {
	g := sampleGrepGraph()
	x := blast.New(g)
	reader := makeReader(map[string]string{
		"pkg/service.go": `line with \1 backslash and f(=x)` + "\n",
	})
	for pattern, want := range map[string]string{
		`\\1`:     "",
		`\(?=`:    "",
		`\\\1`:    "backreference",
		`\\(?=x)`: "lookaround",
		`x\`:      "invalid regex",
	} {
		res, err := grep.Search(g, x, nil, pattern, grep.Options{}, reader)
		switch {
		case want == "" && (err != nil || res.TotalHits != 1):
			t.Errorf("%s: want 1 hit, got %d hits, err %v", pattern, res.TotalHits, err)
		case want != "" && (err == nil || !strings.Contains(err.Error(), want)):
			t.Errorf("%s: want a %s error, got %v", pattern, want, err)
		}
	}
}

// oneFileGraph is s.go with a function per span, all named as given.
func oneFileGraph(names ...string) *model.Graph {
	g := &model.Graph{Nodes: []model.Node{{ID: "s.go", Path: "s.go", Name: "s.go", Kind: model.KindFile, Span: "L1-L10"}}}
	for i, name := range names {
		line := strconv.Itoa(i + 1)
		g.Nodes = append(g.Nodes, model.Node{
			ID: model.NodeID("s.go#" + line + "." + name), Path: "s.go", Name: name, Kind: "function", Span: model.Span("L" + line + "-L" + line),
		})
	}
	return g
}

func groupIDs(res grep.Result) []string {
	var out []string
	for _, grp := range res.Groups {
		if grp.Symbol == nil {
			out = append(out, grp.Path)
		} else {
			out = append(out, string(grp.Symbol.ID))
		}
	}
	return out
}

func TestSearchOrdersGroupsOfOneFile(t *testing.T) {
	for _, c := range []struct {
		names []string
		text  string
		want  []string
	}{
		{[]string{"A", "B"}, "m\nm\n", []string{"s.go#1.A", "s.go#2.B"}},         // already in name order
		{[]string{"A"}, "m\nm\n", []string{"s.go#1.A", "s.go"}},                  // file level after the symbol
		{[]string{"Get", "Get"}, "m\nm\n", []string{"s.go#1.Get", "s.go#2.Get"}}, // equal names keep line order
	} {
		g := oneFileGraph(c.names...)
		res, err := grep.Search(g, blast.New(g), nil, "m", grep.Options{}, makeReader(map[string]string{"s.go": c.text}))
		if err != nil {
			t.Fatal(err)
		}
		if got := groupIDs(res); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v: groups = %v, want %v", c.names, got, c.want)
		}
	}
}

func TestSearchRanksTheHigherInDegreeFirst(t *testing.T) {
	g := &model.Graph{
		Nodes: []model.Node{
			{ID: "a.go#X", Path: "a.go", Name: "X", Kind: "function", Span: "L1-L1"},
			{ID: "b.go#Y", Path: "b.go", Name: "Y", Kind: "function", Span: "L1-L1"},
		},
		Edges: []model.Edge{{Source: "a.go#X", Target: "b.go#Y", Relation: model.RelationCalls}},
	}
	res, err := grep.Search(g, blast.New(g), nil, "m", grep.Options{}, makeReader(map[string]string{"a.go": "m", "b.go": "m"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := groupIDs(res); !reflect.DeepEqual(got, []string{"b.go#Y", "a.go#X"}) {
		t.Errorf("groups = %v, want b.go#Y (in-degree 1) before a.go#X", got)
	}
}

func TestSearchIsCaseSensitiveUnlessAsked(t *testing.T) {
	g := oneFileGraph("A")
	read := makeReader(map[string]string{"s.go": "FOO"})
	for _, opts := range []grep.Options{{Fixed: true}, {}} {
		res, err := grep.Search(g, nil, nil, "foo", opts, read)
		if err != nil || res.TotalHits != 0 {
			t.Errorf("%+v: hits = %d, err = %v, want no hit on FOO", opts, res.TotalHits, err)
		}
	}
}

func TestSearchRefusesTheNinthBackreference(t *testing.T) {
	g := oneFileGraph("A")
	if _, err := grep.Search(g, nil, nil, `(a)(b)(c)(d)(e)(f)(g)(h)(i)\9`, grep.Options{}, makeReader(nil)); err == nil || !strings.Contains(err.Error(), "backreference") {
		t.Errorf(`\9: err = %v, want the backreference refusal`, err)
	}
}

func TestSearchUsesTheSpansItIsGiven(t *testing.T) {
	g := oneFileGraph("A")
	res, err := grep.Search(g, nil, map[string][]model.SymbolSpan{}, "m", grep.Options{}, makeReader(map[string]string{"s.go": "m"}))
	if err != nil {
		t.Fatal(err)
	}
	if got := groupIDs(res); !reflect.DeepEqual(got, []string{"s.go"}) {
		t.Errorf("groups = %v, want only the file level -- the given spans hold no symbol", got)
	}
}
