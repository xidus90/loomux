package blast_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

// The walk vectors of this file are ported (see "Ported sources" in NOTICE.md)
// from test/graph-traverse.test.ts (MIT).

// diamondGraph: A and B both call X, C calls both A and B. Walking incoming
// edges from X reaches C twice at the same depth.
func diamondGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "X", Name: "X", Kind: "function", Path: "x.ts"},
			{ID: "A", Name: "A", Kind: "function", Path: "a.ts"},
			{ID: "B", Name: "B", Kind: "function", Path: "b.ts"},
			{ID: "C", Name: "C", Kind: "function", Path: "c.ts"},
		},
		Edges: []model.Edge{
			{Source: "A", Target: "X", Relation: model.RelationCalls},
			{Source: "B", Target: "X", Relation: model.RelationCalls},
			{Source: "C", Target: "A", Relation: model.RelationCalls},
			{Source: "C", Target: "B", Relation: model.RelationCalls},
		},
	}
}

// shortcutDiamondGraph is diamondGraph plus the shortcut C -> X: C now calls X
// directly and again the long way round through A and B. Walking incoming
// edges from X therefore finds C at two different depths, which is the only
// way to tell "first reached" apart from "last reached".
func shortcutDiamondGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "X", Name: "X", Kind: "function", Path: "x.ts"},
			{ID: "A", Name: "A", Kind: "function", Path: "a.ts"},
			{ID: "B", Name: "B", Kind: "function", Path: "b.ts"},
			{ID: "C", Name: "C", Kind: "function", Path: "c.ts"},
		},
		Edges: []model.Edge{
			{Source: "A", Target: "X", Relation: model.RelationCalls},
			{Source: "B", Target: "X", Relation: model.RelationCalls},
			{Source: "C", Target: "A", Relation: model.RelationCalls},
			{Source: "C", Target: "B", Relation: model.RelationCalls},
			{Source: "C", Target: "X", Relation: model.RelationCalls},
		},
	}
}

func idsAndDepths(hits []blast.Hit) map[model.NodeID]int {
	got := map[model.NodeID]int{}
	for _, h := range hits {
		got[h.ID] = h.Depth
	}
	return got
}

func TestReachDiamondReportsEachNodeOnce(t *testing.T) {
	got := blast.New(diamondGraph()).Reach([]model.NodeID{"X"}, blast.In, 2)

	if len(got) != 3 {
		t.Fatalf("got %d hits, want 3: A and B at depth 1, C once at depth 2", len(got))
	}
	depths := idsAndDepths(got)
	if depths["A"] != 1 || depths["B"] != 1 || depths["C"] != 2 {
		t.Errorf("got depths %v, want A=1 B=1 C=2", depths)
	}
}

func TestReachReportsTheDepthFirstReached(t *testing.T) {
	got := blast.New(shortcutDiamondGraph()).Reach([]model.NodeID{"X"}, blast.In, 2)

	if len(got) != 3 {
		t.Fatalf("got %d hits, want 3: A, B and C once each", len(got))
	}
	depths := idsAndDepths(got)
	if depths["A"] != 1 || depths["B"] != 1 {
		t.Errorf("got depths %v, want A=1 B=1", depths)
	}
	if depths["C"] != 1 {
		t.Errorf("got C at depth %d, want 1: C reaches X both directly and "+
			"through A and B, and is reported at the depth it was first "+
			"reached, not the last", depths["C"])
	}
}

func TestReachDepthCap(t *testing.T) {
	got := blast.New(diamondGraph()).Reach([]model.NodeID{"X"}, blast.In, 1)

	depths := idsAndDepths(got)
	if len(depths) != 2 || depths["A"] != 1 || depths["B"] != 1 {
		t.Errorf("got %v, want A and B alone", depths)
	}
}

func TestReachAllFollowsTheWholeChain(t *testing.T) {
	g := &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "a", Kind: "function", Path: "a.ts"},
			{ID: "b", Kind: "function", Path: "b.ts"},
			{ID: "c", Kind: "function", Path: "c.ts"},
			{ID: "d", Kind: "function", Path: "d.ts"},
		},
		Edges: []model.Edge{
			{Source: "a", Target: "b", Relation: model.RelationCalls},
			{Source: "b", Target: "c", Relation: model.RelationCalls},
			{Source: "c", Target: "d", Relation: model.RelationCalls},
		},
	}
	got := blast.New(g).Reach([]model.NodeID{"a"}, blast.Out, blast.All)

	depths := idsAndDepths(got)
	if len(depths) != 3 || depths["b"] != 1 || depths["c"] != 2 || depths["d"] != 3 {
		t.Errorf("got %v, want b=1 c=2 d=3", depths)
	}
}

func TestReachExcludesItsOwnStartAndDedupsAcrossThem(t *testing.T) {
	// C calls into both A and B; reached from two starts at the same depth it
	// is still reported once. B also calls A, so the walk runs into the start
	// B and must not report it. D calls only B, so it is reachable through the
	// second start alone: a walk that stopped at the first start would miss it.
	g := &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "A", Kind: "function", Path: "a.ts"},
			{ID: "B", Kind: "function", Path: "b.ts"},
			{ID: "C", Kind: "function", Path: "c.ts"},
			{ID: "D", Kind: "function", Path: "d.ts"},
		},
		Edges: []model.Edge{
			{Source: "C", Target: "A", Relation: model.RelationCalls},
			{Source: "C", Target: "B", Relation: model.RelationCalls},
			{Source: "B", Target: "A", Relation: model.RelationCalls},
			{Source: "D", Target: "B", Relation: model.RelationCalls},
		},
	}
	got := blast.New(g).Reach([]model.NodeID{"A", "B"}, blast.In, 2)

	depths := idsAndDepths(got)
	if len(got) != 2 || depths["C"] != 1 || depths["D"] != 1 {
		t.Fatalf("got %v, want C and D once each at depth 1", got)
	}
	if _, ok := hitOf(got, "B"); ok {
		t.Errorf("got %v, want no hit for B: a start id is never its own hit, "+
			"not even when another start's walk runs into it", got)
	}
}

// fileAndSymbolGraph: b.ts imports a.ts as a file AND calls a function defined
// in it. A file's dependents are only complete when the walk starts on the
// file node and on every symbol of that file.
func fileAndSymbolGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "src/a.ts", Name: "a.ts", Kind: model.KindFile, Path: "src/a.ts"},
			{ID: "src/a.ts#helper", Name: "helper", Kind: "function", Path: "src/a.ts"},
			{ID: "src/b.ts", Name: "b.ts", Kind: model.KindFile, Path: "src/b.ts"},
			{ID: "src/b.ts#useB", Name: "useB", Kind: "function", Path: "src/b.ts"},
		},
		Edges: []model.Edge{
			{Source: "src/b.ts", Target: "src/a.ts", Relation: model.RelationImports},
			{Source: "src/b.ts#useB", Target: "src/a.ts#helper", Relation: model.RelationCalls},
		},
	}
}

func TestReachFromAFileNodeAloneMissesTheCallers(t *testing.T) {
	g := fileAndSymbolGraph()
	x := blast.New(g)

	// The file node alone sees only the file-level import edge: the call
	// targets the symbol id and is invisible from there.
	fileOnly := idsAndDepths(x.Reach([]model.NodeID{"src/a.ts"}, blast.In, 2))
	if len(fileOnly) != 1 || fileOnly["src/b.ts"] != 1 {
		t.Fatalf("got %v, want the importing file alone", fileOnly)
	}

	// Starting on the file and its symbols recovers both dependents. This is
	// what a caller builds with model.SymbolsInFile.
	start := append([]model.NodeID{"src/a.ts"}, model.SymbolsInFile(g, "src/a.ts")...)
	both := idsAndDepths(x.Reach(start, blast.In, 2))
	if len(both) != 2 || both["src/b.ts"] != 1 || both["src/b.ts#useB"] != 1 {
		t.Errorf("got %v, want both the importing file and the calling symbol", both)
	}
}
