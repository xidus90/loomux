package golang_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/model"
)

// The forms each supported Go release added to the language, per the release
// notes at https://go.dev/doc/go1.26 and https://go.dev/doc/go1.27.

func versionsResult(t *testing.T) extract.Result {
	t.Helper()
	r, err := golang.File("pkg/versions.go", fixture(t, "versions"))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestFileFollowsACallInsideNewOfAnExpression(t *testing.T) {
	calls := edgesOf(t, versionsResult(t), model.RelationCalls)
	if !hasEdge(calls, extract.RawEdge{Source: "pkg/versions.go#ptr", Name: "g"}) {
		t.Errorf("new(g()) must carry the call to g; got %+v", calls)
	}
}

func TestFileReadsASelfReferentialTypeParameter(t *testing.T) {
	r := versionsResult(t)
	for _, id := range []model.NodeID{"pkg/versions.go#Adder", "pkg/versions.go#algo"} {
		if nodeByID(r, id) == nil {
			t.Errorf("node %q missing", id)
		}
	}
	for _, n := range r.Nodes {
		if n.Name == "A" {
			t.Errorf("a type parameter is not a symbol; got node %+v", n)
		}
	}
	for _, e := range edgesOf(t, r, model.RelationCalls) {
		if e.Name == "A" || e.Name == "Adder" {
			t.Errorf("a constraint is not a call; got %+v", e)
		}
	}
}

func TestFileReadsAGenericMethodAndItsInstantiatedCall(t *testing.T) {
	r := versionsResult(t)
	n := nodeByID(r, "pkg/versions.go#Stack.Map")
	if n == nil || n.Kind != "method" || n.Owner != "Stack" {
		t.Fatalf("a generic method must be a method of its receiver; got %+v", n)
	}
	calls := edgesOf(t, r, model.RelationCalls)
	// `s.Map[int](h)` takes the path `s.Push(h)` would: s is a parameter of
	// type *Stack[int], so the edge carries the owner.
	if !hasEdge(calls, extract.RawEdge{Source: "pkg/versions.go#use", Name: "Map", Owner: "Stack"}) {
		t.Errorf("an instantiated method call must carry its owner; got %+v", calls)
	}
	if !hasEdge(calls, extract.RawEdge{Source: "pkg/versions.go#Stack.Map", Name: "f"}) {
		t.Errorf("a generic method's body must emit its calls; got %+v", calls)
	}
}

func TestFileFollowsACallUnderAPromotedFieldKey(t *testing.T) {
	calls := edgesOf(t, versionsResult(t), model.RelationCalls)
	if !hasEdge(calls, extract.RawEdge{Source: "pkg/versions.go#build", Name: "v"}) {
		t.Errorf("Outer{B: v()} must carry the call to v; got %+v", calls)
	}
}

func TestFileEmitsNoEdgeForAnInferredFunctionValue(t *testing.T) {
	for _, e := range edgesOf(t, versionsResult(t), model.RelationCalls) {
		if e.Name == "id" {
			t.Errorf("assigning a generic function is a value, not a call; got %+v", e)
		}
	}
}
