// Package blast walks the wiring of a code graph: who breaks if this changes,
// and what this depends on. It answers the callers, the callees and the blast
// radius of a refactoring from the same index.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/traverse.ts.
package blast

import "github.com/xidus90/loomux/internal/code/model"

// Direction is which way a walk follows its edges.
type Direction int

const (
	// In walks incoming edges: who calls, references or imports this.
	In Direction = iota
	// Out walks outgoing edges: what this calls, references or imports.
	Out
)

// Depth caps how far a walk goes. All follows the edges as far as they lead.
type Depth int

// All is the transitive closure.
const All Depth = -1

// Hit is one node a walk reached.
//
// Node is nil when the id is not a node of the graph: an unresolved import
// names its module, and a walk reports it rather than hiding the dependency.
type Hit struct {
	ID       model.NodeID
	Node     *model.Node
	Relation model.Relation
	Depth    int
}

type link struct {
	other    model.NodeID
	relation model.Relation
}

// Index is the adjacency of one graph, built once and walked many times.
type Index struct {
	in    map[model.NodeID][]link
	out   map[model.NodeID][]link
	nodes map[model.NodeID]*model.Node
}

// New indexes a graph for walking. Only walk relations enter the adjacency;
// "contains" would make every file a hub.
func New(g *model.Graph) *Index {
	x := &Index{
		in:    map[model.NodeID][]link{},
		out:   map[model.NodeID][]link{},
		nodes: map[model.NodeID]*model.Node{},
	}
	if g == nil {
		return x
	}
	for i := range g.Nodes {
		x.nodes[g.Nodes[i].ID] = &g.Nodes[i]
	}
	for _, e := range g.Edges {
		if !e.Relation.IsWalk() {
			continue
		}
		x.out[e.Source] = append(x.out[e.Source], link{other: e.Target, relation: e.Relation})
		x.in[e.Target] = append(x.in[e.Target], link{other: e.Source, relation: e.Relation})
	}
	return x
}
