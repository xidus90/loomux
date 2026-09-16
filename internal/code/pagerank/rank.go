package pagerank

import (
	"sort"

	"github.com/xidus90/loomux/internal/code/model"
)

// The defaults of the reference: a quarter of the mass restarts at the seeds
// every step, and 25 steps settle a graph of this size.
const (
	defaultAlpha      = 0.25
	defaultIterations = 25
)

// Options tunes the walk. A zero value means the defaults.
type Options struct {
	// Alpha is the restart probability: the share of mass that teleports back
	// to the seed set each step. Higher keeps the walk closer to the seeds.
	Alpha float64
	// Iterations is the power-iteration count.
	Iterations int
}

func (o Options) alpha() float64 {
	if o.Alpha <= 0 || o.Alpha >= 1 {
		return defaultAlpha
	}
	return o.Alpha
}

func (o Options) iterations() int {
	if o.Iterations <= 0 {
		return defaultIterations
	}
	return o.Iterations
}

// Scored is one node and its rank.
type Scored struct {
	ID    model.NodeID
	Score float64
}

// Rank walks the topology from seed and returns the nodes the walk reached,
// best first, ties broken alphabetically by id.
//
// Scores are normalized so the best node scores 1; a node the walk never
// reached is absent rather than zero. Seeds naming no node, and weights that
// are not positive, are ignored -- a seed set that is empty after that yields
// no result at all.
func Rank(t Topology, seed map[model.NodeID]float64, opts Options) []Scored {
	restart := make([]float64, len(t.ids))
	for id, w := range seed {
		i, ok := t.index[id]
		if !ok || w <= 0 {
			continue
		}
		restart[i] = w
	}
	// The weights are summed over the slice, not while reading the map: map
	// order is randomized per range, and the same floats added in another order
	// differ in the last bits. That difference divides into every score.
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

	alpha, iterations := opts.alpha(), opts.iterations()
	rank := make([]float64, len(restart))
	copy(rank, restart)
	next := make([]float64, len(restart))
	for step := 0; step < iterations; step++ {
		// Teleport first: every step, alpha of the mass returns to the seeds.
		for i, r := range restart {
			next[i] = alpha * r
		}
		dangling := 0.0
		for i, mass := range rank {
			nbrs := t.adj[i]
			if len(nbrs) == 0 {
				dangling += mass
				continue
			}
			share := (1 - alpha) * mass / float64(len(nbrs))
			for _, j := range nbrs {
				next[j] += share
			}
		}
		// The mass of nodes with no walk edges is pooled and returned to the
		// seeds in one pass, weighted like the restart distribution. Doing it
		// per dangling node is the same arithmetic at O(dangling x seeds), and
		// a graph of 20k isolated nodes seeded broadly takes minutes that way.
		if dangling > 0 {
			dm := (1 - alpha) * dangling
			for i, r := range restart {
				if r > 0 {
					next[i] += dm * r
				}
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
	out := make([]Scored, 0, len(rank))
	for i, v := range rank {
		if v <= 0 {
			continue
		}
		out = append(out, Scored{ID: t.ids[i], Score: v / max})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}
