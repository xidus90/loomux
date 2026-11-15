package graphview

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/code/model"
)

func hitIDs(hits []Hit) []string {
	out := []string{}
	for _, h := range hits {
		out = append(out, h.ID)
	}
	return out
}

func TestSearch_SymbolNeedsItsDirectoriesAndFileOpen(t *testing.T) {
	code := testGraph()
	hits := Search(code, nil, "a1")
	want := []Hit{{
		ID: "internal/a/x.go#A1", Label: "A1", Kind: "symbol",
		Open: []string{"internal", "internal/a", "internal/a/x.go"},
	}}
	if !reflect.DeepEqual(hits, want) {
		t.Fatalf("hits = %+v, want %+v", hits, want)
	}
	// The open list must make the hit appear in the view.
	find(t, Build(code, nil, "", hits[0].Open, DefaultLimit), hits[0].ID)
}

func TestSearch_TopLevelHitIsVisibleUnderItsEmptyOpenList(t *testing.T) {
	code := testGraph()
	hits := Search(code, nil, "internal")
	if hits[0].ID != "dir:internal" || hits[0].Open == nil || len(hits[0].Open) != 0 {
		t.Fatalf("hit = %+v", hits[0])
	}
	find(t, Build(code, nil, "", hits[0].Open, DefaultLimit), hits[0].ID)
}

func TestSearch_MatchesFilesDirectoriesAndDocuments(t *testing.T) {
	code, wiki := testGraph(), wikiGraph()

	dir := Search(code, wiki, "INTERNAL/B")
	if got := dir[0]; got.ID != "dir:internal/b" || got.Kind != "dir" || got.Label != "b" ||
		!reflect.DeepEqual(got.Open, []string{"internal"}) {
		t.Fatalf("dir hit = %+v", got)
	}

	file := Search(code, wiki, "MAIN.go")
	want := []Hit{{ID: "file:cmd/loomux/main.go", Label: "main.go", Kind: "file", Open: []string{"cmd", "cmd/loomux"}}}
	if !reflect.DeepEqual(file, want) {
		t.Fatalf("file hit = %+v, want %+v", file, want)
	}

	// A document matches on its id, and carries no open list.
	byID := Search(code, wiki, "TOPICS/z")
	wantDoc := []Hit{{ID: "doc:" + docZ, Label: "Z", Kind: "doc", Open: []string{}}}
	if !reflect.DeepEqual(byID, wantDoc) {
		t.Fatalf("doc hit = %+v, want %+v", byID, wantDoc)
	}

	// ... and on its title; an untitled one shows its file name.
	w := &graph.Graph{Nodes: []graph.Node{{ID: "docs/q.md", Title: "Zebra"}, {ID: "docs/plain.md"}}}
	if got := Search(nil, w, "zebr"); !reflect.DeepEqual(hitIDs(got), []string{"doc:docs/q.md"}) {
		t.Fatalf("title hit = %v", hitIDs(got))
	}
	if got := Search(nil, w, "plain"); got[0].Label != "plain.md" {
		t.Fatalf("untitled label = %q", got[0].Label)
	}
}

func TestSearch_ExactNamesFirstThenByPath(t *testing.T) {
	code := &model.Graph{Nodes: []model.Node{
		fileNode("aa/f.go"), symbol("aa/f.go", "FooBar"),
		fileNode("zz/f.go"), symbol("zz/f.go", "Foo"),
	}}
	wiki := &graph.Graph{Nodes: []graph.Node{{ID: "docs/aaa.md", Title: "foo"}}}
	hits := Search(code, wiki, "foo")
	// Exact matches (title "foo", symbol Foo) lead, by path; then FooBar.
	want := []string{"doc:docs/aaa.md", "zz/f.go#Foo", "aa/f.go#FooBar"}
	if got := hitIDs(hits); !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestSearch_EmptyQueryAndNoMatchAreEmpty(t *testing.T) {
	for _, q := range []string{"", "  \t", "no-such-thing"} {
		hits := Search(testGraph(), wikiGraph(), q)
		if hits == nil || len(hits) != 0 {
			t.Fatalf("q %q: hits = %#v, want empty non-nil", q, hits)
		}
	}
	if hits := Search(nil, nil, "x"); hits == nil || len(hits) != 0 {
		t.Fatalf("nil graphs: hits = %#v", hits)
	}
}

func TestSearch_CapsAtMaxHits(t *testing.T) {
	g := &model.Graph{}
	for i := range MaxHits + 5 {
		g.Nodes = append(g.Nodes, fileNode(fmt.Sprintf("pkg/f%02d.go", i)))
	}
	hits := Search(g, nil, "f")
	if len(hits) != MaxHits || hits[0].ID != "file:pkg/f00.go" {
		t.Fatalf("len = %d, first = %v", len(hits), hitIDs(hits)[:1])
	}
}
