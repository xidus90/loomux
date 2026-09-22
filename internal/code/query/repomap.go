package query

import (
	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/repomap"
)

// MapOptions configures the repository map generation.
type MapOptions struct {
	MaxDirs    int
	HubsPerDir int
	Hotspots   int
	NoRefresh  bool
	Keep       func(path string) bool
}

// Map builds the RepoMap for root, respecting privacy filters.
func Map(root string, opts MapOptions) (repomap.RepoMap, []string, error) {
	g, notes, err := loadGraph(root, opts.NoRefresh)
	if err != nil {
		return repomap.RepoMap{}, notes, err
	}

	if opts.Keep != nil {
		var keptNodes []model.Node
		keptSet := make(map[model.NodeID]bool)
		for i := range g.Nodes {
			n := g.Nodes[i]
			if opts.Keep(n.Path) {
				keptNodes = append(keptNodes, n)
				keptSet[n.ID] = true
			}
		}
		var keptEdges []model.Edge
		for _, e := range g.Edges {
			if keptSet[e.Source] && keptSet[e.Target] {
				keptEdges = append(keptEdges, e)
			}
		}
		g = &model.Graph{
			Meta:  g.Meta,
			Nodes: keptNodes,
			Edges: keptEdges,
		}
	}

	x := blast.New(g)
	m := repomap.Build(g, x, repomap.Options{
		MaxDirs:    opts.MaxDirs,
		HubsPerDir: opts.HubsPerDir,
		Hotspots:   opts.Hotspots,
	})

	return m, notes, nil
}

// MapReport formats RepoMap into a concise terminal overview.
func MapReport(m repomap.RepoMap) string {
	return repomap.Format(m)
}
