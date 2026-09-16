package model_test

import (
	"os"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func TestDecodeSample(t *testing.T) {
	f, err := os.Open("testdata/wiring.json")
	if err != nil {
		t.Fatalf("open sample: %v", err)
	}
	defer f.Close()

	g, err := model.Decode(f)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Nodes) != 7 || len(g.Edges) != 6 {
		t.Fatalf("got %d nodes and %d edges, want 7 and 6", len(g.Nodes), len(g.Edges))
	}
	if g.Nodes[0].Signature != "" {
		t.Errorf("a null signature must decode as empty, got %q", g.Nodes[0].Signature)
	}
	if g.Nodes[0].Kind != model.KindFile {
		t.Errorf("a renamed \"kind\" tag would hide every file node, got %q", g.Nodes[0].Kind)
	}
	if g.Nodes[0].Name != "cache.ts" {
		t.Errorf("a renamed \"name\" tag would leave every node nameless, got %q", g.Nodes[0].Name)
	}
	if g.Nodes[0].Span != "L1-L90" {
		t.Errorf("a renamed \"span\" tag would cost every node its line range, got %q", g.Nodes[0].Span)
	}
	if g.Nodes[2].Owner != "Cache" {
		t.Errorf("got owner %q, want %q", g.Nodes[2].Owner, "Cache")
	}
	if !g.Nodes[2].Exported {
		t.Errorf("a renamed \"exported\" tag would make every symbol private, got %v", g.Nodes[2].Exported)
	}
	if g.Nodes[2].Signature != "get(k: string): number" {
		t.Errorf("a renamed \"signature\" tag would silently empty every signature, got %q", g.Nodes[2].Signature)
	}
	if g.Edges[0].Relation != model.RelationContains {
		t.Errorf("a renamed \"relation\" tag would leave every edge meaningless, got %q", g.Edges[0].Relation)
	}
	if g.Edges[2].Target != "npm:lodash" {
		t.Errorf("an unresolved import target must survive decoding, got %q", g.Edges[2].Target)
	}
	if g.Edges[1].Confidence != model.ConfidenceLSPResolved {
		t.Errorf("got confidence %q, want %q", g.Edges[1].Confidence, model.ConfidenceLSPResolved)
	}
}

func TestDecodeRejects(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"broken json", `{"meta":`, "read graph:"},
		{"wrong version", `{"meta":{"version":2},"nodes":[],"edges":[]}`, "graph version 2"},
		{"node without id", `{"meta":{"version":1},"nodes":[{"name":"x","kind":"function","path":"a.ts"}],"edges":[]}`, "node 0 has no id"},
		{"node without path", `{"meta":{"version":1},"nodes":[{"id":"a.ts#x","name":"x","kind":"function"}],"edges":[]}`, `node "a.ts#x" has no path`},
		{"node without kind", `{"meta":{"version":1},"nodes":[{"id":"a.ts#x","name":"x","path":"a.ts"}],"edges":[]}`, `node "a.ts#x" has no kind`},
		{"edge without source", `{"meta":{"version":1},"nodes":[],"edges":[{"target":"b","relation":"calls"}]}`, "edge 0 has no source"},
		{"edge without target", `{"meta":{"version":1},"nodes":[],"edges":[{"source":"a","relation":"calls"}]}`, "edge 0 has no target"},
		{"unknown relation", `{"meta":{"version":1},"nodes":[],"edges":[{"source":"a","target":"b","relation":"summons"}]}`, `edge 0 has relation "summons"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := model.Decode(strings.NewReader(c.in))
			if err == nil {
				t.Fatalf("got no error, want one containing %q", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("got error %q, want it to contain %q", err, c.want)
			}
		})
	}
}

func TestValidateAcceptsEmptyGraph(t *testing.T) {
	g := &model.Graph{Meta: model.Meta{Version: 1}}
	if err := g.Validate(); err != nil {
		t.Errorf("an empty graph of the right version is valid, got %v", err)
	}
}
