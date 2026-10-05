package blast

import "github.com/xidus90/loomux/internal/code/model"

// EdgeWalk traverses the graph from start along dir.
//
// At depth 1, it performs a direct single-step edge scan:
// - duplicate edges to the same neighbour are preserved as separate hits
// - self-loops (recursion) are included, reporting start as its own caller/callee
// - file node start walks only the edges of the file node itself (does not expand contained symbols)
//
// At depth > 1 (or All), it performs a breadth-first search via Reach:
// - diamond converges to one hit
// - start nodes are never reported as hits (even across cycles)
// - file node start expands to include all symbols contained in that file as seeds
//
// Ported from src/graph/traverse.ts (edgeWalk) (MIT; origin under "Ported
// sources" in NOTICE.md).
func (x *Index) EdgeWalk(start *model.Node, dir Direction, depth Depth) []Hit {
	if x == nil || start == nil {
		return nil
	}
	if depth == 1 {
		adj := x.out
		if dir == In {
			adj = x.in
		}
		links := adj[start.ID]
		if len(links) == 0 {
			return nil
		}
		hits := make([]Hit, len(links))
		for i, l := range links {
			hits[i] = Hit{
				ID:       l.other,
				Node:     x.nodes[l.other],
				Relation: l.relation,
				Depth:    1,
			}
		}
		return hits
	}
	if depth < 1 && depth != All {
		return nil
	}

	startIDs := []model.NodeID{start.ID}
	if start.Kind == model.KindFile {
		startIDs = append(startIDs, x.fileSymbols[start.Path]...)
	}
	return x.Reach(startIDs, dir, depth)
}
