package query

import (
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
)

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

	// main.go imports the package, never the file name lib.go.
	ans, _, err := Callers(root, "lib/lib.go", CallersOptions{Direction: blast.In, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range ans.Hits {
		if h.Path == "main.go" {
			if h.QuotePath != "" || h.Line != 0 || h.Quote != "" {
				t.Errorf("importer main.go = %+v, want no quote", h)
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
