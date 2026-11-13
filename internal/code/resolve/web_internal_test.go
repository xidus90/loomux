package resolve

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestPageRef(t *testing.T) {
	for _, c := range []struct {
		from, ref, want string
		ok              bool
	}{
		{"site/p.html", "app.js", "site/app.js", true},
		{"site/p.html", "../app.js?v=3#x", "app.js", true},
		{"site/p.html", "/js//app.js", "js/app.js", true},
		{"site/p.html", "a%20b.css", "site/a b.css", true},
		{"site/p.html", "https://cdn.example/x.js", "", false},
		{"site/p.html", "//cdn/x.js", "", false},
		{"site/p.html", "%zz", "", false},
		{"main.scss", "sass:math", "", false},
	} {
		got, ok := pageRef(c.from, c.ref)
		if got != c.want || ok != c.ok {
			t.Errorf("pageRef(%q, %q) = %q, %v, want %q, %v", c.from, c.ref, got, ok, c.want, c.ok)
		}
	}
}

func TestSassFile(t *testing.T) {
	paths := map[string]bool{
		"s/_a.scss": true, "s/a.scss": true, "s/b.scss": true, "s/c/_index.scss": true,
		"s/d.scss": true, "s/_e.scss": true, "s/f.css": true,
	}
	for ref, want := range map[string]string{
		"a":       "s/_a.scss",
		"b":       "s/b.scss",
		"c":       "s/c/_index.scss",
		"d.scss":  "s/d.scss",
		"e.scss":  "s/_e.scss",
		"f.css":   "s/f.css",
		"missing": "",
		"g.scss":  "",
		"%zz":     "",
	} {
		got, ok := sassFile("s/main.scss", ref, paths)
		if got != want || ok != (want != "") {
			t.Errorf("sassFile(%q) = %q, %v, want %q", ref, got, ok, want)
		}
	}
}

// A page's own family never answers its handover: the element an id
// names on the page is no class or function another language defines.
func TestBridgeSkipsAnOfferOfTheHandingFamily(t *testing.T) {
	offers := []offer{{Export: extract.Export{Namespace: extract.NamespaceHTMLID, Name: "app", Target: "p.html##app"}, file: "p.html", lang: "html"}}
	h := handover{bridge: extract.Bridge{
		Source: "q.html", Relation: model.RelationReferences, Namespace: extract.NamespaceHTMLID, Name: "app", File: "q.html", Line: 1,
	}, lang: "html", unresolved: -1}
	if edges, _ := bridgeEdges(builtinRules, nil, offers, []handover{h}, nil, nil); len(edges) != 0 {
		t.Fatalf("edges = %v, want none", edges)
	}
	h.lang = "javascript"
	if edges, _ := bridgeEdges(builtinRules, nil, offers, []handover{h}, nil, nil); len(edges) != 1 {
		t.Fatalf("edges = %v, want the element", edges)
	}
}
