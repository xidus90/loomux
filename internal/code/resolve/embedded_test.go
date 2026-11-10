package resolve_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// page is an HTML file with one inline script part, as an extractor of C
// would hand it over.
func page() extract.Result {
	body := model.Node{ID: "p.html#body", Name: "body", Kind: "element", Path: "p.html", BodyHash: "h"}
	f := model.Node{ID: "p.html#script.f", Name: "f", Kind: "function", Path: "p.html", BodyHash: "h"}
	return extract.Result{
		Path: "p.html", Language: "html",
		Nodes: []model.Node{fileNode("p.html"), body},
		Edges: []extract.RawEdge{{Source: "p.html", Relation: model.RelationContains, TargetID: body.ID, File: "p.html"}},
		Embedded: []extract.Result{{
			Path: "p.html", Language: "js",
			Nodes: []model.Node{f},
			Edges: []extract.RawEdge{{Source: body.ID, Relation: model.RelationContains, TargetID: f.ID, File: "p.html"}},
		}},
	}
}

func TestGraphWithFlattensEmbedded(t *testing.T) {
	g, _ := resolve.GraphWith([]extract.Result{page()}, nil, stamp)
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.Meta.Languages, []string{"html", "js"}) {
		t.Errorf("languages = %v, want html and js", g.Meta.Languages)
	}
	if len(g.Nodes) != 3 {
		t.Errorf("nodes = %v, want the file, body and the script's f", g.Nodes)
	}
	if edgeBetween(g, "p.html#body", "p.html#script.f", model.RelationContains) == nil {
		t.Errorf("the embedded part lost its containment: %v", g.Edges)
	}
}

func TestGraphWithEmbeddedIsDeterministic(t *testing.T) {
	a, _ := resolve.GraphWith([]extract.Result{page(), result("a.go", []model.Node{fileNode("a.go")}, nil)}, nil, stamp)
	b, _ := resolve.GraphWith([]extract.Result{page(), result("a.go", []model.Node{fileNode("a.go")}, nil)}, nil, stamp)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("the same input gave two graphs")
	}
}

func TestGraphWithResolvesExtractorBridges(t *testing.T) {
	cs := model.Node{ID: "game/p.cs#OnHit", Name: "OnHit", Kind: "method", Path: "game/p.cs", BodyHash: "h"}
	files := []extract.Result{
		{Path: "game/project.godot", Language: "conf", Nodes: []model.Node{fileNode("game/project.godot")}},
		{Path: "game/p.cs", Language: "csharp", Nodes: []model.Node{fileNode("game/p.cs"), cs},
			Exports: []extract.Export{{Namespace: extract.NamespaceGodotMethod, Name: "OnHit", Target: cs.ID}}},
		{Path: "game/q.cs", Language: "csharp", Nodes: []model.Node{fileNode("game/q.cs")},
			Bridges: []extract.Bridge{{Source: "game/q.cs", Relation: model.RelationReferences, Namespace: extract.NamespaceGodotMethod,
				Name: "OnHit", Scope: []string{"game/p.cs"}, File: "game/q.cs", Line: 4}}},
	}
	g, _ := resolve.GraphWith(files, nil, stamp)
	e := edgeBetween(g, "game/q.cs", cs.ID, model.RelationReferences)
	if e == nil || e.Confidence != model.ConfidenceExtracted || e.Line != 4 {
		t.Fatalf("bridge edge = %+v in %v", e, g.Edges)
	}
	if g.Meta.EdgeCount != len(g.Edges) {
		t.Errorf("edge count %d, edges %d", g.Meta.EdgeCount, len(g.Edges))
	}
}

func TestGraphWithEmbeddedFileNodeIsRejected(t *testing.T) {
	p := page()
	p.Embedded[0].Nodes = append(p.Embedded[0].Nodes, fileNode("p.html"))
	g, _ := resolve.GraphWith([]extract.Result{p}, nil, stamp)
	if err := g.Validate(); err == nil || !strings.Contains(err.Error(), "p.html") {
		t.Fatalf("Validate = %v, want the doubled file node named", err)
	}
}
