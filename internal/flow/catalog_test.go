package flow_test

import (
	"context"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
)

type namedBlock struct{ kind string }

func (b namedBlock) Kind() string                        { return b.kind }
func (namedBlock) Check(flow.Node) []string              { return nil }
func (namedBlock) Texts(flow.Node) []flow.Text           { return nil }
func (namedBlock) Writes(flow.Node) map[string]flow.Type { return nil }
func (namedBlock) NeedsBaseline() bool                   { return false }

func (namedBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, nil
}

func (namedBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	return flow.Result{}, nil
}

func catalog(t *testing.T, blocks []flow.Block, predicates map[string]flow.Predicate) *flow.Catalog {
	t.Helper()
	made, err := flow.NewCatalog(blocks, predicates)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// truthy is a predicate that holds for every state, for tests that need one
// registered and do not care what it says.
func truthy(flow.State, flow.Params) bool { return true }

func TestCatalogFindsWhatItHolds(t *testing.T) {
	holds := func(flow.State, flow.Params) bool { return true }
	made := catalog(t, []flow.Block{namedBlock{"gate"}, namedBlock{"exit"}},
		map[string]flow.Predicate{"is_clean": holds})

	block, ok := made.Block("gate")
	if !ok || block.Kind() != "gate" {
		t.Fatalf("Block(gate) = %v, %v", block, ok)
	}
	if _, ok := made.Block("command"); ok {
		t.Fatal("command is not registered")
	}
	if _, ok := made.Predicate("is_clean"); !ok {
		t.Fatal("is_clean is registered")
	}
	if _, ok := made.Predicate("nothing"); ok {
		t.Fatal("nothing is not registered")
	}
}

// Sorted, because both lists end up in a load message and a map's order does not.
func TestCatalogListsWhatItHoldsSorted(t *testing.T) {
	made := catalog(t, []flow.Block{namedBlock{"gate"}, namedBlock{"agent"}},
		map[string]flow.Predicate{"b": truthy, "a": truthy})
	if kinds := made.Kinds(); !slices.Equal(kinds, []string{"agent", "gate"}) {
		t.Fatalf("Kinds() = %v", kinds)
	}
	names := make([]string, 0, 2)
	for name := range made.Predicates() {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"a", "b"}) {
		t.Fatalf("Predicates() = %v", names)
	}
}

// A copy, so that a loader handing the map to expr.ParseCondition cannot
// widen what every later flow may name.
func TestPredicatesIsACopy(t *testing.T) {
	made := catalog(t, nil, map[string]flow.Predicate{"a": truthy})
	made.Predicates()["b"] = truthy
	if _, ok := made.Predicate("b"); ok {
		t.Fatal("the returned map is the catalog's own")
	}
}

func TestNewCatalogRefusesTwoBlocksOfOneKind(t *testing.T) {
	_, err := flow.NewCatalog([]flow.Block{namedBlock{"gate"}, namedBlock{"gate"}}, nil)
	if err == nil || err.Error() != `two blocks claim kind "gate"` {
		t.Fatalf("err = %v", err)
	}
}

// Found like any other, a nil predicate would panic inside the first condition
// that names it, far from whoever registered it.
func TestNewCatalogRefusesANilPredicate(t *testing.T) {
	_, err := flow.NewCatalog(nil, map[string]flow.Predicate{"green": truthy, "red": nil})
	if err == nil || err.Error() != `predicate "red" is nil` {
		t.Fatalf("err = %v", err)
	}
}
