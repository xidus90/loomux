package model_test

import (
	"encoding/json"
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
		{"wrong version", `{"meta":{"version":3},"nodes":[],"edges":[]}`, "graph version 3"},
		{"node without id", `{"meta":{"version":2,"extractor":"test/1"},"nodes":[{"name":"x","kind":"function","path":"a.ts"}],"edges":[]}`, "node 0 has no id"},
		{"node without path", `{"meta":{"version":2,"extractor":"test/1"},"nodes":[{"id":"a.ts#x","name":"x","kind":"function"}],"edges":[]}`, `node "a.ts#x" has no path`},
		{"node without kind", `{"meta":{"version":2,"extractor":"test/1"},"nodes":[{"id":"a.ts#x","name":"x","path":"a.ts"}],"edges":[]}`, `node "a.ts#x" has no kind`},
		{"edge without source", `{"meta":{"version":2,"extractor":"test/1"},"nodes":[],"edges":[{"target":"b","relation":"calls"}]}`, "edge 0 has no source"},
		{"edge without target", `{"meta":{"version":2,"extractor":"test/1"},"nodes":[],"edges":[{"source":"a","relation":"calls"}]}`, "edge 0 has no target"},
		{"unknown relation", `{"meta":{"version":2,"extractor":"test/1"},"nodes":[],"edges":[{"source":"a","target":"b","relation":"summons"}]}`, `edge 0 has relation "summons"`},
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
	g := &model.Graph{Meta: model.Meta{Version: 2, Extractor: "test/1"}}
	if err := g.Validate(); err != nil {
		t.Errorf("an empty graph of the right version is valid, got %v", err)
	}
}

func TestValidateRejectsDuplicateNodeIDs(t *testing.T) {
	// pagerank.Prepare keeps the first node of an id, blast.New overwrites --
	// two answers to the same question out of one graph.
	g := &model.Graph{
		Meta: model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: []model.Node{
			{ID: "a.go#F", Kind: "function", Path: "a.go", BodyHash: "h1"},
			{ID: "a.go#F", Kind: "function", Path: "a.go", BodyHash: "h2"},
		},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "a.go#F") {
		t.Fatalf("got %v, want an error naming the duplicated id", err)
	}
}

func TestValidateRejectsAnEdgeSourceThatIsNoNode(t *testing.T) {
	// A loose end is foreseen on the target side of an import only. An invented
	// source shows up in blast as a hit without a node and is then
	// indistinguishable from a genuinely unresolved import.
	g := &model.Graph{
		Meta:  model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: []model.Node{{ID: "a.go", Kind: model.KindFile, Path: "a.go", BodyHash: "h"}},
		Edges: []model.Edge{{
			Source: "a.go#Ghost", Target: "a.go", Relation: model.RelationCalls,
			Confidence: model.ConfidenceExtracted,
		}},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "a.go#Ghost") {
		t.Fatalf("got %v, want an error naming the invented source", err)
	}
}

func TestValidateRejectsAnEmptyBodyHash(t *testing.T) {
	// Without it `graph check` cannot judge the node and would have to call it
	// fresh in silence.
	g := &model.Graph{
		Meta:  model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: []model.Node{{ID: "a.go", Kind: model.KindFile, Path: "a.go"}},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "body hash") {
		t.Fatalf("got %v, want an error about the missing body hash", err)
	}
}

func TestValidateRejectsAnEmptyExtractorStamp(t *testing.T) {
	// `check` compares the stamp before it diffs nodes: a graph from another
	// extractor is not stale, it is foreign.
	g := &model.Graph{
		Meta:  model.Meta{Version: 2},
		Nodes: []model.Node{{ID: "a.go", Kind: model.KindFile, Path: "a.go", BodyHash: "h"}},
	}
	err := g.Validate()
	if err == nil || !strings.Contains(err.Error(), "extractor") {
		t.Fatalf("got %v, want an error about the missing extractor stamp", err)
	}
}

func TestDecodeRefusesSchemaOne(t *testing.T) {
	// A writer that bumped the version changed something; failing loudly beats
	// ranking a stale shape.
	_, err := model.Decode(strings.NewReader(`{"meta":{"version":1},"nodes":[],"edges":[]}`))
	if err == nil || !strings.Contains(err.Error(), "version 1") {
		t.Fatalf("got %v, want a refusal naming version 1", err)
	}
}

func TestBodyTextNeverReachesTheWire(t *testing.T) {
	// It is ~65% of wiring.json's bytes and lives tokenized in the ask sidecar
	// instead. The json:"-" tag is how that decision is enforced.
	out, err := json.Marshal(model.Node{ID: "a.go#F", BodyText: "secret body"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "secret body") {
		t.Fatalf("body text reached the wire: %s", out)
	}
}
