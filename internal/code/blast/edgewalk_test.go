package blast_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
)

func walkGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "src/file.go", Path: "src/file.go", Name: "file.go", Kind: model.KindFile, Span: "L1-L100"},
			{ID: "src/file.go#A", Path: "src/file.go", Name: "A", Kind: "func", Span: "L10-L30"},
			{ID: "src/file.go#B", Path: "src/file.go", Name: "B", Kind: "func", Span: "L40-L60"},
			{ID: "src/other.go#C", Path: "src/other.go", Name: "C", Kind: "func", Span: "L10-L20"},
			{ID: "src/other.go#D", Path: "src/other.go", Name: "D", Kind: "func", Span: "L30-L50"},
		},
		Edges: []model.Edge{
			// file contains A and B
			{Source: "src/file.go", Target: "src/file.go#A", Relation: model.RelationContains},
			{Source: "src/file.go", Target: "src/file.go#B", Relation: model.RelationContains},
			// file imports npm:pkg
			{Source: "src/file.go", Target: "npm:pkg", Relation: model.RelationImports},
			// A calls B twice (duplicate edges with different relations)
			{Source: "src/file.go#A", Target: "src/file.go#B", Relation: model.RelationCalls},
			{Source: "src/file.go#A", Target: "src/file.go#B", Relation: model.RelationReferences},
			// A calls A (self-loop / recursion)
			{Source: "src/file.go#A", Target: "src/file.go#A", Relation: model.RelationCalls},
			// B calls C
			{Source: "src/file.go#B", Target: "src/other.go#C", Relation: model.RelationCalls},
			// C calls D
			{Source: "src/other.go#C", Target: "src/other.go#D", Relation: model.RelationCalls},
		},
	}
}

func TestEdgeWalkDepth1(t *testing.T) {
	g := walkGraph()
	x := blast.New(g)
	nodeA := &g.Nodes[1] // A

	// 1. Outgoing from A at depth 1:
	// Should preserve duplicate edges to B, and include self-loop A -> A
	hits := x.EdgeWalk(nodeA, blast.Out, 1)
	if len(hits) != 3 {
		t.Fatalf("want 3 hits for A depth 1, got %d: %v", len(hits), hits)
	}

	bCount := 0
	aCount := 0
	for _, h := range hits {
		if h.Depth != 1 {
			t.Errorf("expected depth 1, got %d", h.Depth)
		}
		if h.ID == "src/file.go#B" {
			bCount++
		}
		if h.ID == "src/file.go#A" {
			aCount++
		}
	}
	if bCount != 2 {
		t.Errorf("want 2 duplicate hits for B, got %d", bCount)
	}
	if aCount != 1 {
		t.Errorf("want 1 self-loop hit for A, got %d", aCount)
	}

	// 2. Incoming to B at depth 1: walks In edges
	nodeB := &g.Nodes[2]
	inHits := x.EdgeWalk(nodeB, blast.In, 1)
	if len(inHits) != 2 {
		t.Fatalf("want 2 incoming hits for B, got %d: %v", len(inHits), inHits)
	}

	// 3. Node with no outgoing edges at depth 1 returns nil
	nodeD := &g.Nodes[4]
	if noHits := x.EdgeWalk(nodeD, blast.Out, 1); noHits != nil {
		t.Fatalf("want nil for node with no outgoing edges, got %v", noHits)
	}

	// 4. File node at depth 1: only walks file node edges, does not expand contained symbols
	fileNode := &g.Nodes[0]
	fileHits := x.EdgeWalk(fileNode, blast.Out, 1)
	if len(fileHits) != 1 || fileHits[0].ID != "npm:pkg" {
		t.Fatalf("want 1 import hit for file at depth 1, got %v", fileHits)
	}
}

func TestEdgeWalkDepthGreater1(t *testing.T) {
	g := walkGraph()
	x := blast.New(g)
	nodeA := &g.Nodes[1] // A

	// At depth > 1 (e.g. 2):
	// A -> B (calls & refs converged to 1 hit) -> C
	// Start node A is NOT reported in BFS Reach
	hits := x.EdgeWalk(nodeA, blast.Out, 2)
	if len(hits) != 2 {
		t.Fatalf("want 2 hits (B and C), got %d: %v", len(hits), hits)
	}
	if hits[0].ID != "src/file.go#B" || hits[0].Depth != 1 {
		t.Errorf("hit 0: want B at depth 1, got %v", hits[0])
	}
	if hits[1].ID != "src/other.go#C" || hits[1].Depth != 2 {
		t.Errorf("hit 1: want C at depth 2, got %v", hits[1])
	}

	// File node at depth > 1 expands all symbols in file
	fileNode := &g.Nodes[0]
	fileHits := x.EdgeWalk(fileNode, blast.Out, blast.All)
	// file has npm:pkg (depth 1), A calls B (depth 1), B calls C (depth 2), C calls D (depth 3)
	foundC := false
	foundD := false
	for _, h := range fileHits {
		if h.ID == "src/other.go#C" {
			foundC = true
		}
		if h.ID == "src/other.go#D" {
			foundD = true
		}
	}
	if !foundC || !foundD {
		t.Errorf("file node depth All should reach C and D via contained symbols, got %v", fileHits)
	}
}

func TestEdgeWalkNilAndZero(t *testing.T) {
	g := walkGraph()
	x := blast.New(g)
	nodeA := &g.Nodes[1]

	if hits := (*blast.Index)(nil).EdgeWalk(nodeA, blast.Out, 1); hits != nil {
		t.Errorf("want nil for nil Index, got %v", hits)
	}
	if hits := x.EdgeWalk(nil, blast.Out, 1); hits != nil {
		t.Errorf("want nil for nil Node, got %v", hits)
	}
	if hits := x.EdgeWalk(nodeA, blast.Out, 0); hits != nil {
		t.Errorf("want nil for depth 0, got %v", hits)
	}
	if hits := x.EdgeWalk(nodeA, blast.Out, -2); hits != nil {
		t.Errorf("want nil for negative depth not All, got %v", hits)
	}
}

func TestInDegree(t *testing.T) {
	g := walkGraph()
	x := blast.New(g)

	// B has: contains from file (ignored), calls from A, references from A -> inDegree = 2
	if deg := x.InDegree("src/file.go#B"); deg != 2 {
		t.Errorf("want inDegree 2 for B, got %d", deg)
	}

	// A has: contains from file (ignored), calls from A (self-loop) -> inDegree = 1
	if deg := x.InDegree("src/file.go#A"); deg != 1 {
		t.Errorf("want inDegree 1 for A, got %d", deg)
	}

	// Unknown node has inDegree 0
	if deg := x.InDegree("unknown"); deg != 0 {
		t.Errorf("want inDegree 0 for unknown, got %d", deg)
	}

	// Nil index returns 0
	if deg := (*blast.Index)(nil).InDegree("src/file.go#B"); deg != 0 {
		t.Errorf("want inDegree 0 for nil index, got %d", deg)
	}
}
