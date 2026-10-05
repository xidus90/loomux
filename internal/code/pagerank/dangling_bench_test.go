package pagerank_test

import (
	"fmt"
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

// danglingGraph builds the shape TestRankBroadSeedsOnMostlyDanglingGraph
// exercises for correctness: n nodes, a short chain of 100 wired together and
// the rest -- a growing share as n grows, here 19,900 of 20,000 -- with no
// outgoing edge at all. Every node is seeded, which is what forces the walk
// to redistribute the dangling mass on every step.
func danglingGraph(n int) ([]model.NodeID, [][2]int, map[model.NodeID]float64) {
	ids := make([]model.NodeID, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, model.NodeID(fmt.Sprintf("n%d", i)))
	}
	var adjPairs [][2]int
	for i := 0; i < 100 && i+1 < n; i++ {
		adjPairs = append(adjPairs, [2]int{i, i + 1})
	}
	seeds := make(map[model.NodeID]float64, n)
	for _, id := range ids {
		seeds[id] = 1
	}
	return ids, adjPairs, seeds
}

func tripleEdges(ids []model.NodeID, pairs [][2]int) [][3]string {
	edges := make([][3]string, 0, len(pairs))
	for _, p := range pairs {
		edges = append(edges, [3]string{string(ids[p[0]]), string(ids[p[1]]), ""})
	}
	return edges
}

// BenchmarkDanglingPooled is the production path: dangling mass is summed
// once per step and handed back to the restart distribution in one pass.
func BenchmarkDanglingPooled(b *testing.B) {
	ids, pairs, seeds := danglingGraph(20000)
	g := graphOf(ids, tripleEdges(ids, pairs))
	topo := pagerank.Prepare(g, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pagerank.Rank(topo, seeds, pagerank.Options{})
	}
}

// BenchmarkDanglingPerNode is the pre-optimisation shape: instead of pooling
// the dangling mass once per step, it is handed back once per dangling node,
// which makes every step O(dangling x seeds) rather than O(dangling + seeds).
// It exists only to measure the cost the pooled path in rank.go avoids;
// nothing in the production graph package still does this.
func BenchmarkDanglingPerNode(b *testing.B) {
	ids, pairs, seeds := danglingGraph(20000)
	adj := adjacencyOf(len(ids), pairs)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rankPerNodeDangling(ids, adj, seeds, 0.25, 25)
	}
}

// adjacencyOf is the undirected adjacency over the pairs, indexed like ids.
func adjacencyOf(n int, pairs [][2]int) [][]int {
	adj := make([][]int, n)
	for _, p := range pairs {
		adj[p[0]] = append(adj[p[0]], p[1])
		adj[p[1]] = append(adj[p[1]], p[0])
	}
	return adj
}

// rankPerNodeDangling is Rank's algorithm with the one step the optimisation
// in rank.go replaced: the dangling redistribution loops over every dangling
// node and adds its share to every restart entry, instead of pooling the mass
// first and distributing it in one pass. It is not exported and not called by
// any non-test code; it exists solely as the "before" side of the benchmark
// above, so the pooled path can be measured against what it replaced.
func rankPerNodeDangling(ids []model.NodeID, adj [][]int, seed map[model.NodeID]float64, alpha float64, iterations int) []pagerank.Scored {
	n := len(ids)
	index := make(map[model.NodeID]int, n)
	for i, id := range ids {
		index[id] = i
	}
	restart := make([]float64, n)
	for id, w := range seed {
		i, ok := index[id]
		if !ok || w <= 0 {
			continue
		}
		restart[i] = w
	}
	total := 0.0
	for _, w := range restart {
		total += w
	}
	if total <= 0 {
		return nil
	}
	for i := range restart {
		restart[i] /= total
	}

	rank := make([]float64, n)
	copy(rank, restart)
	next := make([]float64, n)
	for step := 0; step < iterations; step++ {
		for i, r := range restart {
			next[i] = alpha * r
		}
		for i, mass := range rank {
			nbrs := adj[i]
			if len(nbrs) == 0 {
				// Per-node redistribution: this node's own share of the
				// dangling mass is handed straight back to every restart
				// entry, one full pass over the seeds per dangling node.
				dm := (1 - alpha) * mass
				for j, r := range restart {
					if r > 0 {
						next[j] += dm * r
					}
				}
				continue
			}
			share := (1 - alpha) * mass / float64(len(nbrs))
			for _, j := range nbrs {
				next[j] += share
			}
		}
		rank, next = next, rank
	}

	max := 0.0
	for _, v := range rank {
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return nil
	}
	out := make([]pagerank.Scored, 0, n)
	for i, v := range rank {
		if v <= 0 {
			continue
		}
		out = append(out, pagerank.Scored{ID: ids[i], Score: v / max})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}
