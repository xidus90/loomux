package graphview

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

func fileNode(p string) model.Node {
	return model.Node{ID: model.NodeID(p), Name: p, Kind: model.KindFile, Path: p}
}

func symbol(p, name string) model.Node {
	return model.Node{ID: model.NodeID(p + "#" + name), Name: name, Kind: "function", Path: p}
}

func edge(from, to string, rel model.Relation, conf model.Confidence) model.Edge {
	return model.Edge{Source: model.NodeID(from), Target: model.NodeID(to), Relation: rel, Confidence: conf}
}

func testGraph() *model.Graph {
	const x, y = "internal/a/x.go", "internal/b/y.go"
	return &model.Graph{
		Nodes: []model.Node{
			fileNode("cmd/loomux/main.go"),
			fileNode(x), symbol(x, "A1"), symbol(x, "A2"),
			fileNode(y), symbol(y, "B1"),
		},
		Edges: []model.Edge{
			edge(x+"#A1", y+"#B1", model.RelationCalls, model.ConfidenceExtracted),
			edge(x+"#A2", y+"#B1", model.RelationCalls, model.ConfidenceInferred),
			edge(x, x+"#A1", model.RelationContains, model.ConfidenceExtracted),
			edge(x+"#A1", "npm:lodash", model.RelationImports, model.ConfidenceExtracted),
		},
	}
}

func ids(v View) []string {
	out := []string{}
	for _, n := range v.Nodes {
		out = append(out, n.ID)
	}
	return out
}

func find(t *testing.T, v View, id string) Node {
	t.Helper()
	for _, n := range v.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("node %q not in %v", id, ids(v))
	return Node{}
}

func TestCode_StartsAtDirectoriesAndOpensUnderTheLimit(t *testing.T) {
	v := Code(testGraph(), nil, 3)
	if want := []string{"internal"}; !reflect.DeepEqual(v.Open, want) {
		t.Fatalf("Open = %v, want %v", v.Open, want)
	}
	want := []string{"dir:cmd", "dir:internal/a", "dir:internal/b"}
	if got := ids(v); !reflect.DeepEqual(got, want) {
		t.Fatalf("nodes = %v, want %v", got, want)
	}
	a := find(t, v, "dir:internal/a")
	if a.Kind != "dir" || a.Label != "a" || a.Path != "internal/a" || a.Group != "internal" || a.Members != 3 {
		t.Fatalf("dir:internal/a = %+v", a)
	}
}

func TestCode_AutoOpenStopsAtTheLimit(t *testing.T) {
	v := Code(testGraph(), nil, 2)
	if len(v.Open) != 0 || !reflect.DeepEqual(ids(v), []string{"dir:cmd", "dir:internal"}) {
		t.Fatalf("view = %+v", v)
	}
}

func TestCode_DefaultLimitApplies(t *testing.T) {
	v := Code(testGraph(), nil, 0)
	// 150 is ample: every directory with two children is opened.
	if want := []string{"internal"}; !reflect.DeepEqual(v.Open, want) {
		t.Fatalf("Open = %v, want %v", v.Open, want)
	}
}

func TestCode_EdgesCountRelationAndConfidence(t *testing.T) {
	v := Code(testGraph(), nil, 3)
	if len(v.Edges) != 1 {
		t.Fatalf("edges = %+v", v.Edges)
	}
	e := v.Edges[0]
	want := map[string]int{"calls/extracted": 1, "calls/inferred": 1}
	if e.Source != "dir:internal/a" || e.Target != "dir:internal/b" || e.Kind != "code" || !reflect.DeepEqual(e.Counts, want) {
		t.Fatalf("edge = %+v", e)
	}
}

func TestCode_ContainsAndForeignTargetsMakeNoEdge(t *testing.T) {
	g := testGraph()
	g.Edges = g.Edges[2:] // contains + import of a module that is no node
	if v := Code(g, nil, 3); len(v.Edges) != 0 {
		t.Fatalf("edges = %+v", v.Edges)
	}
}

func TestCode_EdgesInsideOneNodeVanish(t *testing.T) {
	// Everything folded into dir:internal at limit 2.
	if v := Code(testGraph(), nil, 2); len(v.Edges) != 0 {
		t.Fatalf("edges = %+v", v.Edges)
	}
}

func TestCode_OpenFileShowsItsSymbols(t *testing.T) {
	v := Code(testGraph(), []string{"internal", "internal/a", "internal/a/x.go"}, 3)
	want := []string{"dir:cmd", "dir:internal/b", "internal/a/x.go#A1", "internal/a/x.go#A2"}
	if got := ids(v); !reflect.DeepEqual(got, want) {
		t.Fatalf("nodes = %v, want %v", got, want)
	}
	if s := find(t, v, "internal/a/x.go#A1"); s.Kind != "symbol" || s.Label != "A1" || s.Group != "internal" || s.Members != 1 {
		t.Fatalf("symbol = %+v", s)
	}
	if len(v.Edges) != 2 || v.Edges[0].Source != "internal/a/x.go#A1" || v.Edges[0].Target != "dir:internal/b" {
		t.Fatalf("edges = %+v", v.Edges)
	}
}

func TestCode_ClosedFileCountsItselfAndItsSymbols(t *testing.T) {
	v := Code(testGraph(), []string{"internal", "internal/a"}, 3)
	if f := find(t, v, "file:internal/a/x.go"); f.Kind != "file" || f.Label != "x.go" || f.Members != 3 {
		t.Fatalf("file = %+v", f)
	}
}

func TestCode_SizeIsSummedRank(t *testing.T) {
	v := Code(testGraph(), nil, 3)
	if b, c := find(t, v, "dir:internal/b"), find(t, v, "dir:cmd"); b.Size <= c.Size {
		t.Fatalf("size b = %v, cmd = %v", b.Size, c.Size)
	}
}

// rootView is the view with nothing open: the top directories.
var rootView = []string{"dir:cmd", "dir:internal"}

func TestCode_UnknownOrEscapingOpenIsIgnored(t *testing.T) {
	// The entries are dropped and nothing is opened in their place.
	v := Code(testGraph(), []string{"nope", "../x", "internal/../.."}, 3)
	if len(v.Open) != 0 || !reflect.DeepEqual(ids(v), rootView) {
		t.Fatalf("view = %+v", v)
	}
}

func TestCode_OpenNeedsEveryAncestorOpen(t *testing.T) {
	// "internal/a" is dropped for want of "internal", and so is its file.
	v := Code(testGraph(), []string{"internal/a", "internal/a/x.go"}, 3)
	if len(v.Open) != 0 || !reflect.DeepEqual(ids(v), rootView) {
		t.Fatalf("view = %+v", v)
	}
}

func TestCode_OnlyAnAbsentOpenListOpensAutomatically(t *testing.T) {
	if v := Code(testGraph(), []string{}, 3); len(v.Open) != 0 || !reflect.DeepEqual(ids(v), rootView) {
		t.Fatalf("empty open: view = %+v", v)
	}
	if v := Code(testGraph(), nil, 3); !reflect.DeepEqual(v.Open, []string{"internal"}) {
		t.Fatalf("nil open: Open = %v", v.Open)
	}
}

func TestCode_OpenIsSortedAndDeduplicated(t *testing.T) {
	v := Code(testGraph(), []string{"internal/a", "internal", "internal"}, 3)
	if want := []string{"internal", "internal/a"}; !reflect.DeepEqual(v.Open, want) {
		t.Fatalf("Open = %v, want %v", v.Open, want)
	}
}

func TestCode_RootFileAndDuplicatesAndPathlessNodes(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{fileNode("go.mod"), fileNode("go.mod"), {ID: "ghost", Name: "ghost"}}}
	v := Code(g, nil, 3)
	if got := ids(v); !reflect.DeepEqual(got, []string{"file:go.mod"}) {
		t.Fatalf("nodes = %v", got)
	}
	if n := v.Nodes[0]; n.Members != 1 || n.Group != "go.mod" {
		t.Fatalf("node = %+v", n)
	}
}

func TestCode_NilGraphIsEmpty(t *testing.T) {
	v := Code(nil, nil, 3)
	if v.Nodes == nil || v.Edges == nil || v.Open == nil || len(v.Nodes)+len(v.Edges)+len(v.Open) != 0 {
		t.Fatalf("view = %+v", v)
	}
}

func TestCode_EdgesAreSortedBySourceThenTarget(t *testing.T) {
	g := testGraph()
	g.Edges = append(g.Edges, edge("internal/a/x.go#A1", "cmd/loomux/main.go", model.RelationReferences, model.ConfidenceLSPResolved))
	v := Code(g, nil, 3)
	if len(v.Edges) != 2 || v.Edges[0].Target != "dir:cmd" || v.Edges[1].Target != "dir:internal/b" {
		t.Fatalf("edges = %+v", v.Edges)
	}
}

func TestCode_ExplicitOpenNeverExceedsMaxNodes(t *testing.T) {
	g := &model.Graph{}
	for i := 0; i < MaxNodes+10; i++ {
		g.Nodes = append(g.Nodes, fileNode(fmt.Sprintf("big/f%04d.go", i)))
	}
	for i := 0; i < 3; i++ {
		g.Nodes = append(g.Nodes, fileNode(fmt.Sprintf("small/f%d.go", i)))
	}
	v := Code(g, []string{"big", "small"}, 3)
	if want := []string{"small"}; !reflect.DeepEqual(v.Open, want) {
		t.Fatalf("Open = %v, want %v", v.Open, want)
	}
	if len(v.Nodes) != 4 { // dir:big plus the three files of small
		t.Fatalf("nodes = %d", len(v.Nodes))
	}
}

func TestCode_OpenFileWithoutSymbolsIsIgnored(t *testing.T) {
	v := Code(testGraph(), []string{"cmd", "cmd/loomux", "cmd/loomux/main.go"}, 3)
	if want := []string{"cmd", "cmd/loomux"}; !reflect.DeepEqual(v.Open, want) {
		t.Fatalf("Open = %v, want %v", v.Open, want)
	}
	find(t, v, "file:cmd/loomux/main.go")
}
