package model

import (
	"encoding/json"
	"fmt"
	"io"
)

// schemaVersion is the only wiring.json this model reads. A writer that bumps
// it changed something; failing loudly beats ranking stale shapes.
const schemaVersion = 1

// Decode reads a wiring.json and validates it.
func Decode(r io.Reader) (*Graph, error) {
	var g Graph
	if err := json.NewDecoder(r).Decode(&g); err != nil {
		return nil, fmt.Errorf("read graph: %w", err)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}

// Validate reports the first thing about a graph that no consumer can work
// around: a version it does not know, a node without identity or place, an
// edge with a loose end or a relation the schema never had.
//
// It does not check that every edge end is a node: an unresolved import names
// its module, and that is a fact about the code, not a defect.
func (g *Graph) Validate() error {
	if g.Meta.Version != schemaVersion {
		return fmt.Errorf("graph version %d, want %d", g.Meta.Version, schemaVersion)
	}
	for i, n := range g.Nodes {
		switch {
		case n.ID == "":
			return fmt.Errorf("node %d has no id", i)
		case n.Path == "":
			return fmt.Errorf("node %q has no path", n.ID)
		case n.Kind == "":
			return fmt.Errorf("node %q has no kind", n.ID)
		}
	}
	for i, e := range g.Edges {
		switch {
		case e.Source == "":
			return fmt.Errorf("edge %d has no source", i)
		case e.Target == "":
			return fmt.Errorf("edge %d has no target", i)
		case !e.Relation.IsWalk() && e.Relation != RelationContains:
			return fmt.Errorf("edge %d has relation %q, which the schema does not define", i, e.Relation)
		}
	}
	return nil
}
