package query

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// In the sample, lib_test.go calls Run in another file of its package, which
// only a repo-wide unique name resolves; main.go names lib.Run outright.
func TestCallersMarksAnInferredCaller(t *testing.T) {
	ans, _, err := Callers(built(t), "Run", CallersOptions{Direction: blast.In, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	inferred := map[string]bool{}
	for _, h := range ans.Hits {
		inferred[h.Name] = h.Inferred
	}
	if !inferred["TestRun"] || inferred["main"] {
		t.Fatalf("inferred = %v, want TestRun guessed and main sure", inferred)
	}
	report := CallersReport(ans)
	if !strings.Contains(report, "  - TestRun (function, depth 1, calls, inferred) in lib/lib_test.go:") ||
		!strings.Contains(report, "  - main (function, depth 1, calls) in main.go:") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestCallersReportMarksAnInferredHitWithoutANode(t *testing.T) {
	report := CallersReport(CallersAnswer{
		Start: &model.Node{Name: "Run", Kind: "function", Path: "lib/lib.go", Span: "L4-L4"},
		Hits:  []CallerHit{{ID: "ext", Relation: model.RelationImports, Depth: 1, Inferred: true}},
	})
	if !strings.Contains(report, "  - ext (depth 1, imports, inferred)\n") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestCallerHitEncodesInferredOnlyWhenSet(t *testing.T) {
	for _, c := range []struct {
		hit  CallerHit
		want bool
	}{{CallerHit{ID: "a", Inferred: true}, true}, {CallerHit{ID: "a"}, false}} {
		raw, err := json.Marshal(c.hit)
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(string(raw), `"inferred"`); got != c.want {
			t.Errorf("%s: inferred key present = %v, want %v", raw, got, c.want)
		}
	}
}

func TestCallersMissingGraph(t *testing.T) {
	root := t.TempDir()
	_, _, err := Callers(root, "Run", CallersOptions{})
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("want ErrNoGraph, got %v", err)
	}
}

func TestCallersAnswersAndReports(t *testing.T) {
	root := built(t)

	ans, _, err := Callers(root, "Run", CallersOptions{Direction: blast.In, Depth: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ans.Start == nil || ans.Start.Name != "Run" {
		t.Fatalf("want start Run, got %v", ans.Start)
	}
	if len(ans.Hits) == 0 {
		t.Fatalf("want callers of Run, got none")
	}

	report := CallersReport(ans)
	if !strings.Contains(report, "Run") || !strings.Contains(report, "calls") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestCallersQuotesTheCallSiteInsideTheStartForCallees(t *testing.T) {
	root := built(t)

	ans, _, err := Callers(root, "main", CallersOptions{Direction: blast.Out, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range ans.Hits {
		if h.Name != "Run" {
			continue
		}
		if h.QuotePath != "main.go" || h.Line != 5 || h.Quote != "func main() { lib.Run() }" {
			t.Fatalf("callee Run quoted as %s:%d %q, want main.go:5 with the call", h.QuotePath, h.Line, h.Quote)
		}
		if !strings.Contains(CallersReport(ans), "    main.go:L5: func main() { lib.Run() }") {
			t.Errorf("a quote outside the hit's file needs its path:\n%s", CallersReport(ans))
		}
		return
	}
	t.Fatalf("want Run among the callees of main, got %+v", ans.Hits)
}

func TestCallersQuotesTheLineOfTheEdgeAndNotAnEarlierMention(t *testing.T) {
	root := repo(t, map[string]string{
		"go.mod": "module example.com/repo\n",
		"lib/lib.go": "package lib\n\ntype Index struct{}\n\nfunc New() *Index { return &Index{} }\n\n" +
			"func (x *Index) Reach() {}\n\n" +
			"func use() {\n\t// Reach is named here first.\n\tNew().Reach()\n}\n",
	})
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}
	ans, _, err := Callers(root, "Reach", CallersOptions{Direction: blast.In, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range ans.Hits {
		if h.Name == "use" {
			if h.Line != 11 || h.Quote != "New().Reach()" {
				t.Errorf("use quoted at L%d %q, want the call at L11", h.Line, h.Quote)
			}
			return
		}
	}
	t.Fatalf("want use among the callers of Reach, got %+v", ans.Hits)
}

func TestEvidenceFallsBackToTheWordWhenTheEdgeHasNoLine(t *testing.T) {
	src := "package p\n\nfunc use() {\n\tReach()\n}\n"
	read := func(string) ([]byte, error) { return []byte(src), nil }
	caller := &model.Node{Name: "use", Path: "p.go", Span: "L3-L5"}
	callee := &model.Node{Name: "Reach", Path: "p.go"}
	// No line on the edge, and a line past the end of the file, both fall
	// back to the word in the caller's span.
	for _, at := range []int{0, 99} {
		if path, line, quote := evidence(read, callee, caller, blast.In, at); path != "p.go" || line != 4 || quote != "Reach()" {
			t.Errorf("at %d: evidence = %s:%d %q, want p.go:4 Reach()", at, path, line, quote)
		}
	}
	other := &model.Node{Name: "Elsewhere", Path: "p.go"}
	if path, line, quote := evidence(read, other, caller, blast.In, 0); path != "" || line != 0 || quote != "" {
		t.Errorf("no line and no word is no evidence; got %s:%d %q", path, line, quote)
	}
	unreadable := func(string) ([]byte, error) { return nil, errors.New("gone") }
	if _, ok := quoteAt(unreadable, "p.go", 4); ok {
		t.Error("an unreadable file has no line to quote")
	}
}

func TestCallersNamesTheOtherSymbolsOfTheSameName(t *testing.T) {
	root := repo(t, map[string]string{
		"go.mod":                  "module example.com/repo\n",
		"corpus/probe.go":         "package corpus\n\nfunc Probe() {}\n",
		"freshness/probe.go":      "package freshness\n\nfunc Probe() {}\n\nfunc use() { Probe() }\n",
		"freshness/probe_test.go": "package freshness\n\nfunc probe() {}\n",
	})
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}
	ans, _, err := Callers(root, "Probe", CallersOptions{Direction: blast.In, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	if ans.Start.Path != "corpus/probe.go" || len(ans.Also) != 1 || ans.Also[0].Path != "freshness/probe.go" {
		t.Fatalf("start %s, also %+v; want corpus first and only the other exact Probe", ans.Start.Path, ans.Also)
	}
	report := CallersReport(ans)
	if !strings.Contains(report, "also matching: freshness.Probe (function) in freshness/probe.go; name the package to pick one") {
		t.Errorf("report must name the other matches:\n%s", report)
	}
	// A qualified name picks one, and nothing else matches.
	ans, _, err = Callers(root, "freshness.Probe", CallersOptions{Direction: blast.In, Depth: 1})
	if err != nil || ans.Start.Path != "freshness/probe.go" || len(ans.Also) != 0 || strings.Contains(CallersReport(ans), "also matching") {
		t.Errorf("qualified: start %v, also %+v, err %v", ans.Start, ans.Also, err)
	}
}

func TestAlsoListCountsPastFiveAndKeepsARootNameBare(t *testing.T) {
	var nodes []*model.Node
	for _, p := range []string{"main.go", "a/x.go", "b/x.go", "c/x.go", "d/x.go", "e/x.go", "f/x.go"} {
		nodes = append(nodes, &model.Node{Name: "Run", Kind: "function", Path: p})
	}
	want := "Run (function) in main.go, a.Run (function) in a/x.go, b.Run (function) in b/x.go, " +
		"c.Run (function) in c/x.go, d.Run (function) in d/x.go, and 2 more"
	if got := alsoList(nodes); got != want {
		t.Errorf("alsoList =\n%s\nwant\n%s", got, want)
	}
}

func TestCallersQuotesOnlyDirectNeighbours(t *testing.T) {
	root := repo(t, map[string]string{
		"go.mod": "module example.com/repo\n",
		"chain/chain.go": "package chain\n\nfunc A() {}\n\nfunc B() { A() }\n\n" +
			"func C() { B() /* A */ }\n",
	})
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}

	ans, _, err := Callers(root, "A", CallersOptions{Direction: blast.In, Depth: blast.All})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]CallerHit{}
	for _, h := range ans.Hits {
		got[h.Name] = h
	}
	if b := got["B"]; b.Depth != 1 || b.Line != 5 || b.QuotePath != "chain/chain.go" {
		t.Errorf("direct caller B = %+v, want depth 1 quoted at chain/chain.go:5", b)
	}
	// C never calls A; the word in its comment is no evidence of an edge.
	if c := got["C"]; c.Depth != 2 || c.Quote != "" || c.Line != 0 || c.QuotePath != "" {
		t.Errorf("transitive caller C = %+v, want depth 2 without a quote", c)
	}
}

func TestCallersKeepsADirectHitWhoseSpanNeverNamesTheStart(t *testing.T) {
	root := built(t)

	// main.go imports the package, never the file name lib.go: no word search
	// finds it, but the import edge knows its line.
	ans, _, err := Callers(root, "lib/lib.go", CallersOptions{Direction: blast.In, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range ans.Hits {
		if h.Path == "main.go" {
			if h.QuotePath != "main.go" || h.Line != 3 || h.Quote != `import "example.com/repo/lib"` {
				t.Errorf("importer main.go = %+v, want the import quoted at L3", h)
			}
			return
		}
	}
	t.Fatalf("want main.go among the importers of lib/lib.go, got %+v", ans.Hits)
}

func TestCallersResolvesPastAHiddenFirstMatch(t *testing.T) {
	root := repo(t, map[string]string{
		"go.mod":  "module example.com/repo\n",
		"a/a.go":  "package a\n\nfunc Load() {}\n",
		"b/b.go":  "package b\n\nfunc Load() {}\n",
		"main.go": "package main\n\nimport \"example.com/repo/b\"\n\nfunc main() { b.Load() }\n",
	})
	if _, _, err := Build(root, ignore); err != nil {
		t.Fatal(err)
	}
	// Precondition: without a filter the hidden file's Load comes first.
	ans, _, err := Callers(root, "Load", CallersOptions{})
	if err != nil || ans.Start.Path != "a/a.go" {
		t.Fatalf("precondition: start = %v, err %v; want a/a.go first", ans.Start, err)
	}

	ans, _, err = Callers(root, "Load", CallersOptions{
		Keep: func(p string) bool { return p != "a/a.go" },
	})
	if err != nil {
		t.Fatalf("a readable match exists, got %v", err)
	}
	if ans.Start.Path != "b/b.go" {
		t.Errorf("start = %s, want the readable b/b.go", ans.Start.Path)
	}
}

func TestCallersPrivacyAndEdgeCases(t *testing.T) {
	root := built(t)

	// Keep rejecting start symbol
	_, _, err := Callers(root, "Run", CallersOptions{
		Keep: func(p string) bool { return false },
	})
	if err == nil || !strings.Contains(err.Error(), "symbol not found") {
		t.Fatalf("want symbol not found when start is rejected by keep, got %v", err)
	}

	// Keep rejecting callers (hidden count)
	ans, _, err := Callers(root, "Run", CallersOptions{
		Direction: blast.In,
		Depth:     1,
		Keep: func(p string) bool {
			return p == "lib/lib.go" // keep lib.go (where Run is), reject main.go (where caller is)
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ans.Hidden == 0 {
		t.Fatalf("want hidden callers, got %d", ans.Hidden)
	}
	report := CallersReport(ans)
	if !strings.Contains(report, "hidden by privacy") {
		t.Fatalf("expected privacy notice in report:\n%s", report)
	}

	// Unknown symbol
	_, _, err = Callers(root, "NonExistentSymbol", CallersOptions{})
	if err == nil || !strings.Contains(err.Error(), "symbol not found") {
		t.Fatalf("want error for unknown symbol, got %v", err)
	}

	// Invalid In prefix
	_, _, err = Callers(root, "Run", CallersOptions{In: "nonexistent"})
	if err == nil || !strings.Contains(err.Error(), "prefix not indexed") {
		t.Fatalf("want error for invalid In prefix, got %v", err)
	}

	// Depth 0 defaults to 1
	ansDepth0, _, err := Callers(root, "Run", CallersOptions{Depth: 0})
	if err != nil || len(ansDepth0.Hits) == 0 {
		t.Fatalf("want default depth 1 hits, got %v err=%v", ansDepth0, err)
	}

	// CallersReport with unnamed hit (e.g. unresolved import)
	unnamedReport := CallersReport(CallersAnswer{
		Start: ans.Start,
		Hits: []CallerHit{
			{ID: "npm:lodash", Depth: 1, Relation: "imports"},
		},
	})
	if !strings.Contains(unnamedReport, "npm:lodash") {
		t.Errorf("expected unnamed hit formatted in report:\n%s", unnamedReport)
	}

	// CallersReport empty
	if CallersReport(CallersAnswer{}) != "no start symbol\n" {
		t.Error("expected no start symbol message")
	}
	emptyHitsReport := CallersReport(CallersAnswer{Start: ans.Start})
	if !strings.Contains(emptyHitsReport, "no callers found") {
		t.Errorf("expected no callers found in report:\n%s", emptyHitsReport)
	}
}
