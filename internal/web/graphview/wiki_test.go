package graphview

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/code/model"
)

const (
	docX = "docs/wiki/topics/x.md"
	docZ = "docs/wiki/topics/z.md"
	docY = "docs/de/y.md"
	docP = "docs/.superpowers/p.md"
)

func wikiGraph() *graph.Graph {
	return &graph.Graph{
		Nodes: []graph.Node{
			{ID: docX, Title: "X"}, {ID: docZ, Title: "Z"}, {ID: docY}, {ID: docP, Title: "P"},
		},
		Edges: []graph.Edge{{From: docZ, To: docX}},
		CodeRefs: []graph.CodeRef{
			{From: docX, Path: "internal/b"},
			{From: docX, Path: "internal/a/x.go", Line: 5, End: 5},
			{From: docX, Path: "ci/run.ps1"},
			{From: docX, Path: docY},
			{From: docP, Path: "internal/b/y.go"},
		},
	}
}

func edgeOf(t *testing.T, v View, source, target string) Edge {
	t.Helper()
	for _, e := range v.Edges {
		if e.Source == source && e.Target == target {
			return e
		}
	}
	t.Fatalf("edge %s -> %s not in %v", source, target, v.Edges)
	return Edge{}
}

func hasEdge(v View, source, target string) bool {
	for _, e := range v.Edges {
		if e.Source == source && e.Target == target {
			return true
		}
	}
	return false
}

func TestBuild_EveryDocumentIsANodeMarkedWiki(t *testing.T) {
	v := Build(testGraph(), wikiGraph(), "docs/wiki", nil, 3)
	for id, c := range map[string]struct {
		wiki         bool
		group, label string
	}{
		"doc:" + docX: {true, "topics", "X"},
		"doc:" + docZ: {true, "topics", "Z"},
		"doc:" + docY: {false, "docs/de", "y.md"},
		"doc:" + docP: {false, "docs/.superpowers", "P"},
	} {
		n := find(t, v, id)
		if n.Kind != "doc" || n.Wiki != c.wiki || n.Group != c.group || n.Label != c.label ||
			n.Size != 1 || n.Members != 1 || n.Path != id[len("doc:"):] {
			t.Errorf("%s = %+v", id, n)
		}
	}
	for i := 1; i < len(v.Nodes); i++ {
		if v.Nodes[i-1].ID >= v.Nodes[i].ID {
			t.Fatalf("nodes not sorted: %v", ids(v))
		}
	}
}

func TestBuild_DocumentDirectlyInTheWikiGroupsByTheWikiFolder(t *testing.T) {
	w := &graph.Graph{Nodes: []graph.Node{{ID: "docs/wiki/index.md", Title: "I"}, {ID: "top.md", Title: "T"}}}
	v := Build(nil, w, "docs/wiki", nil, 0)
	if g := find(t, v, "doc:docs/wiki/index.md").Group; g != "wiki" {
		t.Errorf("index group = %q", g)
	}
	if g := find(t, v, "doc:top.md").Group; g != "" {
		t.Errorf("top group = %q", g)
	}
}

func TestBuild_LinksBetweenDocumentsAreEdges(t *testing.T) {
	v := Build(testGraph(), wikiGraph(), "docs/wiki", nil, 3)
	e := edgeOf(t, v, "doc:"+docZ, "doc:"+docX)
	if e.Kind != "link" || !reflect.DeepEqual(e.Counts, map[string]int{"link": 1}) {
		t.Fatalf("edge = %+v", e)
	}
}

func TestBuild_LinkEdgesToUnknownDocumentsOrSelfAreDropped(t *testing.T) {
	w := wikiGraph()
	w.Edges = append(w.Edges, graph.Edge{From: docX, To: "gone.md"}, graph.Edge{From: docX, To: docX})
	w.CodeRefs = []graph.CodeRef{{From: "gone.md", Path: "internal/b"}, {From: docX, Path: docX}}
	v := Build(testGraph(), w, "docs/wiki", nil, 3)
	for _, e := range v.Edges {
		if e.Kind != "code" && e.Kind != "link" {
			t.Errorf("unexpected edge %+v", e)
		}
		if e.Source == e.Target || e.Source == "doc:gone.md" || e.Target == "doc:gone.md" {
			t.Errorf("unexpected edge %+v", e)
		}
	}
	if m := find(t, v, "doc:"+docX).Missing; m != nil {
		t.Errorf("Missing = %v", m)
	}
}

func TestBuild_RefToAClosedDirectory(t *testing.T) {
	v := Build(testGraph(), wikiGraph(), "docs/wiki", nil, 3)
	e := edgeOf(t, v, "doc:"+docX, "dir:internal/b")
	if e.Kind != "ref" || !reflect.DeepEqual(e.Counts, map[string]int{"ref": 1}) {
		t.Fatalf("edge = %+v", e)
	}
	// A file below a closed directory folds into that directory.
	if e := edgeOf(t, v, "doc:"+docP, "dir:internal/b"); e.Kind != "ref" {
		t.Fatalf("edge = %+v", e)
	}
	if e := edgeOf(t, v, "doc:"+docX, "dir:internal/a"); e.Kind != "ref" {
		t.Fatalf("edge = %+v", e)
	}
}

func TestBuild_RefsToTheSameElementAreCounted(t *testing.T) {
	w := wikiGraph()
	w.CodeRefs = []graph.CodeRef{
		{From: docX, Path: "internal/b/y.go"}, {From: docX, Path: "internal/b"},
	}
	v := Build(testGraph(), w, "docs/wiki", nil, 3)
	if e := edgeOf(t, v, "doc:"+docX, "dir:internal/b"); e.Counts["ref"] != 2 {
		t.Fatalf("edge = %+v", e)
	}
}

func spanned() *model.Graph {
	g := testGraph()
	for i := range g.Nodes {
		switch g.Nodes[i].ID {
		case "internal/a/x.go#A1":
			g.Nodes[i].Span = "L1-L10"
		case "internal/a/x.go#A2":
			g.Nodes[i].Span = "L11-L20"
		}
	}
	return g
}

func TestBuild_RefWithLineHitsTheEnclosingSymbolWhenTheFileIsOpen(t *testing.T) {
	open := []string{"internal", "internal/a", "internal/a/x.go"}
	v := Build(spanned(), wikiGraph(), "docs/wiki", open, 3)
	e := edgeOf(t, v, "doc:"+docX, "internal/a/x.go#A1")
	if e.Kind != "ref" || e.Counts["ref"] != 1 {
		t.Fatalf("edge = %+v", e)
	}
	if hasEdge(v, "doc:"+docX, "internal/a/x.go#A2") {
		t.Fatalf("edge to A2: %v", v.Edges)
	}
}

func TestBuild_RefWithLineOutsideEverySymbolFansOutToTheSymbols(t *testing.T) {
	w := &graph.Graph{
		Nodes:    []graph.Node{{ID: docX}},
		CodeRefs: []graph.CodeRef{{From: docX, Path: "internal/a/x.go", Line: 99}},
	}
	open := []string{"internal", "internal/a", "internal/a/x.go"}
	v := Build(spanned(), w, "docs/wiki", open, 3)
	for _, s := range []string{"internal/a/x.go#A1", "internal/a/x.go#A2"} {
		edgeOf(t, v, "doc:"+docX, s)
	}
}

func TestBuild_RefToAnOpenDirectoryFansOutToItsVisibleChildren(t *testing.T) {
	w := &graph.Graph{
		Nodes:    []graph.Node{{ID: docX}},
		CodeRefs: []graph.CodeRef{{From: docX, Path: "internal"}},
	}
	v := Build(testGraph(), w, "docs/wiki", []string{"internal"}, 0)
	for _, d := range []string{"dir:internal/a", "dir:internal/b"} {
		if e := edgeOf(t, v, "doc:"+docX, d); e.Kind != "ref" || e.Counts["ref"] != 1 {
			t.Fatalf("edge = %+v", e)
		}
	}
	if hasEdge(v, "doc:"+docX, "dir:cmd") || hasEdge(v, "doc:"+docX, "dir:internal") {
		t.Fatalf("edges = %v", v.Edges)
	}
}

func TestBuild_RefToADocumentIsALink(t *testing.T) {
	v := Build(testGraph(), wikiGraph(), "docs/wiki", nil, 3)
	e := edgeOf(t, v, "doc:"+docX, "doc:"+docY)
	if e.Kind != "link" || !reflect.DeepEqual(e.Counts, map[string]int{"link": 1}) {
		t.Fatalf("edge = %+v", e)
	}
}

func TestBuild_RefDoesNotDoubleACountedLink(t *testing.T) {
	w := wikiGraph()
	w.CodeRefs = []graph.CodeRef{{From: docZ, Path: docX}, {From: docY, Path: docX}, {From: docY, Path: docX}}
	v := Build(testGraph(), w, "docs/wiki", nil, 3)
	if e := edgeOf(t, v, "doc:"+docZ, "doc:"+docX); e.Counts["link"] != 1 {
		t.Fatalf("edge = %+v", e)
	}
	// Mentioned twice and linked nowhere: one edge, counted per mention.
	if e := edgeOf(t, v, "doc:"+docY, "doc:"+docX); e.Counts["link"] != 2 {
		t.Fatalf("edge = %+v", e)
	}
}

func TestBuild_RefOutsideTheCodeGraphIsListedAsMissing(t *testing.T) {
	w := wikiGraph()
	w.CodeRefs = append(w.CodeRefs, graph.CodeRef{From: docX, Path: "ci/run.ps1", Line: 3}, graph.CodeRef{From: docX, Path: "a/b.toml"})
	v := Build(testGraph(), w, "docs/wiki", nil, 3)
	if got, want := find(t, v, "doc:"+docX).Missing, []string{"a/b.toml", "ci/run.ps1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Missing = %v, want %v", got, want)
	}
	for _, e := range v.Edges {
		if e.Target == "ci/run.ps1" || e.Target == "dir:ci" {
			t.Fatalf("edge to a missing path: %+v", e)
		}
	}
	if m := find(t, v, "doc:"+docZ).Missing; m != nil {
		t.Fatalf("Missing = %v", m)
	}
}

func TestBuild_NoWikiPrefixMarksEveryDocumentWiki(t *testing.T) {
	v := Build(testGraph(), wikiGraph(), "", nil, 3)
	for _, id := range []string{docX, docY, docP} {
		if n := find(t, v, "doc:"+id); !n.Wiki {
			t.Errorf("%s not marked wiki", id)
		}
	}
	if g := find(t, v, "doc:"+docX).Group; g != "docs/wiki" {
		t.Errorf("group = %q", g)
	}
}

func TestBuild_NilWikiIsCodeOnly(t *testing.T) {
	if got, want := Build(testGraph(), nil, "docs/wiki", nil, 3), Code(testGraph(), nil, 3); !reflect.DeepEqual(got, want) {
		t.Fatalf("view = %+v, want %+v", got, want)
	}
}

func TestBuild_NilCodeGraphKeepsTheDocuments(t *testing.T) {
	v := Build(nil, wikiGraph(), "docs/wiki", nil, 0)
	if len(v.Nodes) != 4 {
		t.Fatalf("nodes = %v", ids(v))
	}
	if got, want := find(t, v, "doc:"+docX).Missing, []string{"ci/run.ps1", "internal/a/x.go", "internal/b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Missing = %v, want %v", got, want)
	}
}

func TestBuild_KeepsTheCodeViewsOpenSet(t *testing.T) {
	v := Build(testGraph(), wikiGraph(), "docs/wiki", nil, 3)
	if want := []string{"internal"}; !reflect.DeepEqual(v.Open, want) {
		t.Fatalf("Open = %v, want %v", v.Open, want)
	}
}
