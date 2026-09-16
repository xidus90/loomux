// Package pagerank ranks a code graph by personalized PageRank: a
// random walk with restart, seeded with whatever a caller found lexically.
// Mass gathers on the nodes wired into the seeded cluster, so a node that
// merely shares a word with the query sinks. Lexical proposes, the graph
// disposes.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/ask/graphrank.ts.
package pagerank

import "github.com/xidus90/loomux/internal/code/model"

// Topology is the undirected adjacency a rank walks. Preparing it apart from
// the walk lets several seed sets share one scan of the graph.
//
// Nodes are held as a slice with an index beside it, not as a map: the
// iteration order of a map would reach the floating-point sums and make two
// runs of the same rank differ in the last bits.
type Topology struct {
	ids   []model.NodeID
	index map[model.NodeID]int
	adj   [][]int
}

// Len returns how many nodes carry rank.
func (t Topology) Len() int { return len(t.ids) }

// NeighboursOf returns the walk neighbours of a node, in graph order. It
// exists for the tests of this package and for callers that want to see the
// prepared shape.
func (t Topology) NeighboursOf(id model.NodeID) []model.NodeID {
	i, ok := t.index[id]
	if !ok {
		return nil
	}
	out := make([]model.NodeID, 0, len(t.adj[i]))
	for _, j := range t.adj[i] {
		out = append(out, t.ids[j])
	}
	return out
}

// Prepare builds the undirected topology over the walk relations. keep narrows
// it to a subgraph; an edge counts only when both ends pass, and a nil keep
// takes everything.
//
// An edge whose end is not a node -- an unresolved import naming its module --
// is dropped: it would otherwise gather rank mass and be reported as a result
// nobody can open.
func Prepare(g *model.Graph, keep func(model.NodeID) bool) Topology {
	t := Topology{index: map[model.NodeID]int{}}
	if g == nil {
		return t
	}
	for _, n := range g.Nodes {
		if keep != nil && !keep(n.ID) {
			continue
		}
		if _, seen := t.index[n.ID]; seen {
			continue
		}
		t.index[n.ID] = len(t.ids)
		t.ids = append(t.ids, n.ID)
	}
	t.adj = make([][]int, len(t.ids))
	for _, e := range g.Edges {
		if !e.Relation.IsWalk() {
			continue
		}
		from, ok := t.index[e.Source]
		if !ok {
			continue
		}
		to, ok := t.index[e.Target]
		if !ok {
			continue
		}
		t.adj[from] = append(t.adj[from], to)
		t.adj[to] = append(t.adj[to], from)
	}
	return t
}
