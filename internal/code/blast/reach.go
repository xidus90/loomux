package blast

import "github.com/xidus90/loomux/internal/code/model"

// Reach walks from every id in start and returns what it found, in the order
// it found it.
//
// Breadth first: a node is reported once, at the depth it was first reached
// from any start, so a diamond converges to one hit. The start ids are never
// their own hits.
func (x *Index) Reach(start []model.NodeID, dir Direction, depth Depth) []Hit {
	adj := x.out
	if dir == In {
		adj = x.in
	}

	visited := make(map[model.NodeID]bool, len(start))
	frontier := make([]model.NodeID, 0, len(start))
	for _, id := range start {
		if visited[id] {
			continue
		}
		visited[id] = true
		frontier = append(frontier, id)
	}

	var hits []Hit
	for d := 1; (depth == All || Depth(d) <= depth) && len(frontier) > 0; d++ {
		var next []model.NodeID
		for _, current := range frontier {
			for _, l := range adj[current] {
				if visited[l.other] {
					continue
				}
				visited[l.other] = true
				hits = append(hits, Hit{ID: l.other, Node: x.nodes[l.other], Relation: l.relation, Depth: d})
				next = append(next, l.other)
			}
		}
		frontier = next
	}
	return hits
}
