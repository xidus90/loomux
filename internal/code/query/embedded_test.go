package query

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestCacheKeepsEmbeddedBodies(t *testing.T) {
	root := t.TempDir()
	r := extract.Result{
		Path: "p.html", Language: "html",
		Nodes: []model.Node{{ID: "p.html", Kind: model.KindFile, BodyText: "page"}},
		Embedded: []extract.Result{{Path: "p.html", Language: "js",
			Nodes: []model.Node{{ID: "p.html#script.f", Kind: "function", BodyText: "word"}}}},
	}
	if err := writeCache(root, map[string]cacheEntry{"p.html": {Extractor: "html/1", SHA256: "s", Result: r}}); err != nil {
		t.Fatal(err)
	}
	got, err := readCache(root)
	if err != nil {
		t.Fatal(err)
	}
	e := got["p.html"].Result
	if e.Nodes[0].BodyText != "page" || e.Embedded[0].Nodes[0].BodyText != "word" {
		t.Fatalf("bodies after the round trip: %q and %q", e.Nodes[0].BodyText, e.Embedded[0].Nodes[0].BodyText)
	}
}

func TestTallyCountsEmbeddedParts(t *testing.T) {
	r := extract.Result{
		Path: "p.html", Language: "html", Nodes: []model.Node{{ID: "p.html"}, {ID: "p.html#b"}},
		Embedded: []extract.Result{{Path: "p.html", Language: "js", ParseErrors: 1}},
	}
	per := map[string]LangStats{}
	tally(per, "p.html", r, true)
	want := map[string]LangStats{
		"html": {Files: 1, Parsed: 1},
		"js":   {Files: 1, Parsed: 1, ParseErrors: 1, ErrorFiles: []string{"p.html"}},
	}
	if !reflect.DeepEqual(per, want) {
		t.Fatalf("parsed: %+v, want %+v", per, want)
	}

	per = map[string]LangStats{}
	tally(per, "p.html", r, false)
	want = map[string]LangStats{
		"html": {Files: 1, Reused: 1},
		"js":   {Files: 1, Reused: 1, ParseErrors: 1, ErrorFiles: []string{"p.html"}},
	}
	if !reflect.DeepEqual(per, want) {
		t.Fatalf("reused: %+v, want %+v", per, want)
	}
}
