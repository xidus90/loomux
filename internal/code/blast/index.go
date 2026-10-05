// Package blast walks the wiring of a code graph: who breaks if this changes,
// and what this depends on. It answers the callers, the callees and the blast
// radius of a refactoring from the same index.
//
// This walk meets the edges directed, where the package pagerank meets the very
// same edges undirected. That is deliberate: "who breaks if this changes" has a
// direction, and a callee is no answer to it. Rank asks the other question --
// understand this area -- and there a callee weighs as much as a caller.
//
// Ported from src/graph/traverse.ts (MIT; origin under
// "Ported sources" in NOTICE.md).
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
//
// Node points into the graph New was given, it is not a copy. Two consequences
// for a caller: writing through it writes into that graph, and a graph changed
// after New leaves the index and every Node it hands out stale. Treat a hit as
// read-only and build a new Index after a change.
type Hit struct {
	ID       model.NodeID   `json:"id"`
	Node     *model.Node    `json:"node"`
	Relation model.Relation `json:"relation"`
	Depth    int            `json:"depth"`
}

type link struct {
	other    model.NodeID
	relation model.Relation
}

// Index is the adjacency of one graph, built once and walked many times.
type Index struct {
	in          map[model.NodeID][]link
	out         map[model.NodeID][]link
	nodes       map[model.NodeID]*model.Node
	fileSymbols map[string][]model.NodeID
}

// New indexes a graph for walking. Only walk relations enter the adjacency;
// "contains" would make every file a hub.
func New(g *model.Graph) *Index {
	x := &Index{
		in:          map[model.NodeID][]link{},
		out:         map[model.NodeID][]link{},
		nodes:       map[model.NodeID]*model.Node{},
		fileSymbols: map[string][]model.NodeID{},
	}
	if g == nil {
		return x
	}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		x.nodes[n.ID] = n
		if n.Kind != model.KindFile {
			x.fileSymbols[n.Path] = append(x.fileSymbols[n.Path], n.ID)
		}
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

// InDegree counts incoming walk relations (calls, references, imports, implements, extends)
// to id. Contains edges are excluded.
func (x *Index) InDegree(id model.NodeID) int {
	if x == nil {
		return 0
	}
	return len(x.in[id])
}

// InDegreeWhere counts the incoming walk edges of id whose source keep
// accepts. keep sees nil for a source that is not a node of the graph.
func (x *Index) InDegreeWhere(id model.NodeID, keep func(*model.Node) bool) int {
	n := 0
	for _, l := range x.in[id] {
		if keep(x.nodes[l.other]) {
			n++
		}
	}
	return n
}
