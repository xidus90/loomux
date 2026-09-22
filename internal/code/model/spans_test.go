package model_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func TestFileSpansAndEnclosing(t *testing.T) {
	// Nil graph handling
	if spans := model.FileSpans(nil); len(spans) != 0 {
		t.Fatalf("want empty map for nil graph, got %v", spans)
	}

	g := &model.Graph{
		Nodes: []model.Node{
			{ID: "a.go#A", Path: "a.go", Name: "A", Span: "L10-L40"},
			{ID: "a.go#A.Inner", Path: "a.go", Name: "Inner", Span: "L20-L30"},
			{ID: "a.go", Path: "a.go", Kind: model.KindFile, Span: "L1-L50"},
			{ID: "a.go#Broken", Path: "a.go", Name: "Broken", Span: "invalid"},
			{ID: "b.go#B1", Path: "b.go", Name: "B1", Span: "L10-L20"},
			{ID: "b.go#B2", Path: "b.go", Name: "B2", Span: "L10-L15"}, // same start, different end
		},
	}
	fs := model.FileSpans(g)
	spansA := fs["a.go"]
	if len(spansA) != 2 {
		t.Fatalf("want 2 symbol spans in a.go, got %d", len(spansA))
	}
	if spansA[0].From != 10 || spansA[1].From != 20 {
		t.Fatalf("spans not sorted by start line: %+v", spansA)
	}

	// Tie-breaking: same start line sorted by To desc
	spansB := fs["b.go"]
	if len(spansB) != 2 || spansB[0].To != 20 || spansB[1].To != 15 {
		t.Fatalf("tie breaking wrong: %+v", spansB)
	}

	// grep.ts rule: largest span start (innermost in enclosing sense)
	enc := model.Enclosing(spansA, 25)
	if enc == nil || enc.ID != "a.go#A.Inner" {
		t.Fatalf("want Inner, got %v", enc)
	}
	encOuter := model.Enclosing(spansA, 35)
	if encOuter == nil || encOuter.ID != "a.go#A" {
		t.Fatalf("want A, got %v", encOuter)
	}
	encNone := model.Enclosing(spansA, 5)
	if encNone != nil {
		t.Fatalf("want nil for line outside symbols, got %v", encNone)
	}

	// blast.ts rule: innermost contains no other hit symbol
	inner := model.Innermost(spansA, 15, 25)
	if len(inner) != 1 || inner[0].ID != "a.go#A.Inner" {
		t.Fatalf("want Inner, got %v", inner)
	}

	// Range that touches only A
	innerA := model.Innermost(spansA, 32, 38)
	if len(innerA) != 1 || innerA[0].ID != "a.go#A" {
		t.Fatalf("want A, got %v", innerA)
	}
}
