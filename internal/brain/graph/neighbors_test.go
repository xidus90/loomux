package graph_test

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
)

func TestNeighbors_Basic(t *testing.T) {
	g := &graph.Graph{
		Scope: "project/test",
		Nodes: []graph.Node{
			{ID: "target.md"},
			{ID: "in1.md"},
			{ID: "in2.md"},
			{ID: "out1.md"},
			{ID: "out2.md"},
		},
		Edges: []graph.Edge{
			{From: "in2.md", To: "target.md"},
			{From: "in1.md", To: "target.md"},
			{From: "target.md", To: "out2.md"},
			{From: "target.md", To: "out1.md"},
			{From: "unrelated.md", To: "other.md"},
		},
	}

	incoming, outgoing := graph.Neighbors(g, "target.md")
	wantIn := []string{"in1.md", "in2.md"}
	wantOut := []string{"out1.md", "out2.md"}

	if !reflect.DeepEqual(incoming, wantIn) {
		t.Errorf("got incoming %v, want %v", incoming, wantIn)
	}
	if !reflect.DeepEqual(outgoing, wantOut) {
		t.Errorf("got outgoing %v, want %v", outgoing, wantOut)
	}
}

func TestNeighbors_IsolatedAndNil(t *testing.T) {
	g := &graph.Graph{
		Scope: "project/test",
		Nodes: []graph.Node{
			{ID: "isolated.md"},
		},
		Edges: []graph.Edge{},
	}

	incoming, outgoing := graph.Neighbors(g, "isolated.md")
	if len(incoming) != 0 || len(outgoing) != 0 {
		t.Errorf("expected empty neighbors for isolated, got in: %v, out: %v", incoming, outgoing)
	}

	nilIn, nilOut := graph.Neighbors(nil, "any.md")
	if len(nilIn) != 0 || len(nilOut) != 0 {
		t.Errorf("expected empty neighbors for nil graph, got in: %v, out: %v", nilIn, nilOut)
	}
}

func TestRenderNeighbors(t *testing.T) {
	tests := []struct {
		name     string
		incoming []string
		outgoing []string
		want     string
	}{
		{
			name:     "both present",
			incoming: []string{"a.md", "b.md"},
			outgoing: []string{"c.md"},
			want:     "incoming: a.md, b.md\noutgoing: c.md\n",
		},
		{
			name:     "incoming empty",
			incoming: nil,
			outgoing: []string{"c.md"},
			want:     "incoming: -\noutgoing: c.md\n",
		},
		{
			name:     "outgoing empty",
			incoming: []string{"a.md"},
			outgoing: []string{},
			want:     "incoming: a.md\noutgoing: -\n",
		},
		{
			name:     "both empty",
			incoming: nil,
			outgoing: nil,
			want:     "incoming: -\noutgoing: -\n",
		},
	}

	for _, tc := range tests {
		got := graph.RenderNeighbors(tc.incoming, tc.outgoing)
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
