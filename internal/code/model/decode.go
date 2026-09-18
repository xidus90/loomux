package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// schemaVersion is the only wiring.json this model reads. A writer that bumps
// it changed something; failing loudly beats ranking stale shapes.
const schemaVersion = 2

// ErrSchemaVersion wraps a version mismatch, so a caller can tell "this graph
// predates the schema this binary reads" from any other validation failure --
// the two want different guidance, and a schema-1 graph in particular has no
// extractor stamp at all to check next.
var ErrSchemaVersion = errors.New("graph schema version mismatch")

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
// It does not check that every edge target is a node: an unresolved import
// names its module, and that is a fact about the code, not a defect.
func (g *Graph) Validate() error {
	if g.Meta.Version != schemaVersion {
		return fmt.Errorf("%w: graph version %d, want %d", ErrSchemaVersion, g.Meta.Version, schemaVersion)
	}
	if g.Meta.Extractor == "" {
		return fmt.Errorf("graph has no extractor stamp")
	}
	seen := make(map[NodeID]struct{}, len(g.Nodes))
	for i, n := range g.Nodes {
		switch {
		case n.ID == "":
			return fmt.Errorf("node %d has no id", i)
		case n.Path == "":
			return fmt.Errorf("node %q has no path", n.ID)
		case n.Kind == "":
			return fmt.Errorf("node %q has no kind", n.ID)
		case n.BodyHash == "":
			return fmt.Errorf("node %q has no body hash", n.ID)
		}
		if _, dup := seen[n.ID]; dup {
			return fmt.Errorf("node %q appears twice; the two computers would read it differently", n.ID)
		}
		seen[n.ID] = struct{}{}
	}
	for i, e := range g.Edges {
		switch {
		case e.Source == "":
			return fmt.Errorf("edge %d has no source", i)
		case e.Target == "":
			return fmt.Errorf("edge %d has no target", i)
		case !e.Relation.IsWalk() && e.Relation != RelationContains:
			return fmt.Errorf("edge %d has relation %q, which the schema does not define", i, e.Relation)
		case !isNode(seen, e.Source):
			return fmt.Errorf("edge %d has source %q, which is no node of this graph", i, e.Source)
		}
	}
	return nil
}

// isNode reports whether id belongs to a node of the graph.
//
// Only edge sources are held to this. A target may be a loose end: an
// unresolved import names its module ("fmt"), and that is a fact about the
// code, not a defect.
func isNode(nodes map[NodeID]struct{}, id NodeID) bool {
	_, ok := nodes[id]
	return ok
}
