package load

import (
	"testing"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
)

// realCatalog holds the blocks a real run uses, not the doubles the loader's
// own stage tests register.
func realCatalog(t *testing.T) *flow.Catalog {
	t.Helper()
	catalog, err := flow.NewCatalog([]flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

// The stage tests load against doubles, and the blocks are tested without a
// loader. This is where they meet: the planning flow passes all six stages
// against the blocks a real run uses.
func TestThePlanningFlowLoadsAgainstTheRealBlocks(t *testing.T) {
	graph, err := Load(testFlow(t, "planning"), realCatalog(t))
	if err != nil {
		t.Fatalf("the planning flow does not load:\n%v", err)
	}
	if len(graph.Nodes) != 12 {
		t.Fatalf("%d nodes", len(graph.Nodes))
	}
}

// The planning flow has no exit node, and no other fixture's exit gets through
// the real Exit block. The example flow of the spec holds all three kinds, so
// exit meets the loader here.
func TestTheExampleFlowLoadsAgainstTheRealBlocks(t *testing.T) {
	graph, err := Load(testFlow(t, "example"), realCatalog(t))
	if err != nil {
		t.Fatalf("the example flow does not load:\n%v", err)
	}
	kinds := map[string]bool{}
	for _, node := range graph.Nodes {
		kinds[node.Kind] = true
	}
	if !kinds["agent"] || !kinds["gate"] || !kinds["exit"] {
		t.Fatalf("kinds = %v", kinds)
	}
}
