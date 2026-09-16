package blast_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// The walk vectors of this file are ported from trailhq/Graft @ 1e352a3
// (MIT), test/graph-traverse.test.ts.

// baseGraph is Graft's traverse fixture: a file containing a method, that
// method calling another, and an import nobody resolved.
func baseGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "src/cache.ts", Name: "cache.ts", Kind: model.KindFile, Path: "src/cache.ts"},
			{ID: "src/cache.ts#Cache.get", Name: "get", Kind: "method", Path: "src/cache.ts"},
			{ID: "src/widget.ts#Widget.render", Name: "render", Kind: "method", Path: "src/widget.ts"},
			{ID: "pkg/hash.go#Hash", Name: "Hash", Kind: "function", Path: "pkg/hash.go"},
		},
		Edges: []model.Edge{
			{Source: "src/cache.ts", Target: "src/cache.ts#Cache.get", Relation: model.RelationContains},
			{Source: "src/cache.ts#Cache.get", Target: "src/widget.ts#Widget.render", Relation: model.RelationCalls},
			{Source: "src/cache.ts#Cache.get", Target: "npm:lodash", Relation: model.RelationImports},
		},
	}
}

func hitOf(hits []blast.Hit, id model.NodeID) (blast.Hit, bool) {
	for _, h := range hits {
		if h.ID == id {
			return h, true
		}
	}
	return blast.Hit{}, false
}

func TestReachOutKeepsUnresolvedTargets(t *testing.T) {
	got := blast.New(baseGraph()).Reach([]model.NodeID{"src/cache.ts#Cache.get"}, blast.Out, 1)

	if len(got) != 2 {
		t.Fatalf("got %d hits, want 2: %v", len(got), got)
	}
	called, ok := hitOf(got, "src/widget.ts#Widget.render")
	if !ok {
		t.Fatal("the called method is missing")
	}
	if called.Node == nil || called.Node.Name != "render" {
		t.Errorf("got node %v, want the render method", called.Node)
	}
	if called.Relation != model.RelationCalls || called.Depth != 1 {
		t.Errorf("got %v at depth %d, want calls at depth 1", called.Relation, called.Depth)
	}

	unresolved, ok := hitOf(got, "npm:lodash")
	if !ok {
		t.Fatal("an unresolved import target is kept -- who asks what this depends on wants to see it")
	}
	if unresolved.Node != nil {
		t.Errorf("got node %v, want none for a module string", unresolved.Node)
	}
	if unresolved.Relation != model.RelationImports {
		t.Errorf("got %v, want imports", unresolved.Relation)
	}
}

func TestReachInExcludesContains(t *testing.T) {
	x := blast.New(baseGraph())

	got := x.Reach([]model.NodeID{"src/widget.ts#Widget.render"}, blast.In, 1)
	if len(got) != 1 || got[0].ID != "src/cache.ts#Cache.get" || got[0].Relation != model.RelationCalls {
		t.Fatalf("got %v, want the calling method alone", got)
	}

	// The file that contains the method must never show up as its caller.
	if got := x.Reach([]model.NodeID{"src/cache.ts#Cache.get"}, blast.In, 1); len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestReachWithoutEdges(t *testing.T) {
	x := blast.New(baseGraph())
	if got := x.Reach([]model.NodeID{"pkg/hash.go#Hash"}, blast.In, 1); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := x.Reach([]model.NodeID{"pkg/hash.go#Hash"}, blast.Out, 1); got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if got := x.Reach(nil, blast.In, 1); got != nil {
		t.Errorf("got %v, want nil for no start", got)
	}
	if got := blast.New(nil).Reach([]model.NodeID{"a"}, blast.In, 1); got != nil {
		t.Errorf("got %v, want nil for a nil graph", got)
	}
}

// TestReachConverges pins the three promises the brief's fixture cannot make:
// a diamond reports its tip once at the depth it was first reached, a cycle
// back to a start id yields no hit for that start, and a start id repeated by
// the caller changes nothing.
func TestReachConverges(t *testing.T) {
	g := &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "A", Name: "A", Kind: "function", Path: "a.go"},
			{ID: "B", Name: "B", Kind: "function", Path: "b.go"},
			{ID: "C", Name: "C", Kind: "function", Path: "c.go"},
			{ID: "D", Name: "D", Kind: "function", Path: "d.go"},
		},
		Edges: []model.Edge{
			{Source: "A", Target: "B", Relation: model.RelationCalls},
			{Source: "A", Target: "C", Relation: model.RelationCalls},
			{Source: "B", Target: "D", Relation: model.RelationCalls},
			{Source: "C", Target: "D", Relation: model.RelationCalls},
			{Source: "D", Target: "A", Relation: model.RelationCalls},
		},
	}

	got := blast.New(g).Reach([]model.NodeID{"A", "A"}, blast.Out, blast.All)

	want := []model.NodeID{"B", "C", "D"}
	if len(got) != len(want) {
		t.Fatalf("got %d hits, want %d: %v", len(got), len(want), got)
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Fatalf("got hit %d for %q, want %q: %v", i, got[i].ID, id, got)
		}
	}
	if got[2].Depth != 2 {
		t.Errorf("got the diamond tip at depth %d, want 2", got[2].Depth)
	}
	if _, ok := hitOf(got, "A"); ok {
		t.Errorf("a start id is never its own hit, not even around a cycle: %v", got)
	}
}
