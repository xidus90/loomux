package pagerank_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

// graphOf builds the hand-made fixtures of this package: every id becomes a
// function node in its own file, every pair becomes one edge.
func graphOf(ids []model.NodeID, edges [][3]string) *model.Graph {
	g := &model.Graph{Meta: model.Meta{Version: 1}}
	for _, id := range ids {
		g.Nodes = append(g.Nodes, model.Node{ID: id, Name: string(id), Kind: "function", Path: string(id) + ".ts"})
	}
	for _, e := range edges {
		rel := model.Relation(e[2])
		if rel == "" {
			rel = model.RelationCalls
		}
		g.Edges = append(g.Edges, model.Edge{
			Source: model.NodeID(e[0]), Target: model.NodeID(e[1]),
			Relation: rel, Confidence: model.ConfidenceExtracted,
		})
	}
	return g
}

func TestPrepareIsUndirectedOverWalkRelations(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"hub", "a", "b"},
		[][3]string{{"hub", "a", ""}, {"hub", "b", "contains"}},
	)
	topo := pagerank.Prepare(g, nil)

	if got := topo.Len(); got != 3 {
		t.Fatalf("got %d nodes, want 3", got)
	}
	// "a" is reachable from "hub" and back; "b" hangs on a contains edge and is
	// therefore dangling.
	if got := topo.NeighboursOf("hub"); len(got) != 1 || got[0] != "a" {
		t.Errorf("got neighbours of hub = %v, want [a] -- the source-to-target direction is missing", got)
	}
	if got := topo.NeighboursOf("a"); len(got) != 1 || got[0] != "hub" {
		t.Errorf("got neighbours of a = %v, want [hub] -- the target-to-source direction is missing", got)
	}
	if got := topo.NeighboursOf("b"); got == nil || len(got) != 0 {
		t.Errorf("got neighbours of b = %v, want an empty non-nil slice -- contains is not walkable, and only an unknown node yields nil", got)
	}
}

func TestPrepareDropsEdgesWithoutTwoNodes(t *testing.T) {
	g := graphOf([]model.NodeID{"a"}, [][3]string{{"a", "npm:lodash", "imports"}})
	topo := pagerank.Prepare(g, nil)

	if got := topo.NeighboursOf("a"); len(got) != 0 {
		t.Errorf("got %v, want none -- a module string is no node", got)
	}
}

func TestPrepareDropsEdgeWithUnknownSourceAndRepeatedNode(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"a", "a", "b"},
		[][3]string{{"ghost", "a", ""}, {"a", "b", ""}},
	)
	topo := pagerank.Prepare(g, nil)

	if got := topo.Len(); got != 2 {
		t.Fatalf("got %d nodes, want 2 -- a repeated id is one node", got)
	}
	if got := topo.NeighboursOf("a"); len(got) != 1 || got[0] != "b" {
		t.Errorf("got neighbours of a = %v, want [b] -- a source that is no node is dropped", got)
	}
}

func TestPrepareKeepFilter(t *testing.T) {
	g := graphOf(
		[]model.NodeID{"hub", "a", "x"},
		[][3]string{{"hub", "a", ""}, {"a", "x", ""}},
	)
	topo := pagerank.Prepare(g, func(id model.NodeID) bool { return id != "x" })

	if got := topo.Len(); got != 2 {
		t.Fatalf("got %d nodes, want 2", got)
	}
	if got := topo.NeighboursOf("a"); len(got) != 1 || got[0] != "hub" {
		t.Errorf("got %v, want [hub] -- an edge needs both ends inside the filter", got)
	}
	if got := topo.NeighboursOf("x"); got != nil {
		t.Errorf("got %v, want nil for a filtered-out node", got)
	}
}

func TestPrepareNilGraph(t *testing.T) {
	if got := pagerank.Prepare(nil, nil).Len(); got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}
