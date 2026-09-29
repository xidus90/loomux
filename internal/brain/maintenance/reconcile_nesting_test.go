package maintenance_test

import (
	"os"
	"path/filepath"
	"testing"
)

// nestedHub is the registry found on this machine on 2026-09-29, reduced:
// "project/inner" is `local_only`, has its source tree of its own and its
// wiki inside the tree of "hub", whose tree is its own wiki and which holds
// the review centre. The inner page cites the inner source, which has
// changed since the register was written.
func nestedHub(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	inner, _ := w.addArea(t, areaOptions{
		Scope:       "project/inner",
		Wiki:        "docs/wiki",
		PrivacyMode: "local_only",
		Files:       map[string]string{"src/a.go": "package a\n", "docs/wiki/a.md": ""},
		Cites:       map[string][]string{"docs/wiki/a.md": {"src/a.go"}},
		Registered:  []string{"src/a.go"},
	})
	hub, _ := w.addArea(t, areaOptions{
		Scope:  "hub",
		Review: reviewLayout,
		Files:  map[string]string{"top.md": "# Top\n"},
	})
	nested := filepath.Join(hub.Path, "inner")
	if err := os.Rename(inner.WikiPath, nested); err != nil {
		t.Fatal(err)
	}
	w.Areas[0].WikiPath = nested
	w.Areas[1].WikiPath = hub.Path
	w.Change(t, "project/inner", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	return w
}

// The second leak of 2026-09-29: a changed source of the inner area opened a
// case in the inner area and a second one in the hub, whose package carried
// the diff of the `local_only` source into a `manual_cloud` area. A page
// belongs to the deepest area whose wiki holds it, so there is one case, the
// inner area's.
func TestReconcileRaisesTheCaseOfANestedWikiInItsOwnAreaOnly(t *testing.T) {
	w := nestedHub(t)
	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want 1: %+v", len(report.Cases), report.Cases)
	}
	if got := report.Cases[0]; got.Area != "project/inner" || got.Target != "a.md" {
		t.Fatalf("the case is about %q of %q, want a.md of project/inner", got.Target, got.Area)
	}
}

// The hub's own pages are still its own: nesting carves the inner wiki out,
// nothing else.
func TestReconcileKeepsTheCasesOfTheEnclosingWikisOwnPages(t *testing.T) {
	w := nestedHub(t)
	hub := w.Areas[1]
	writeUnder(t, hub.Path, "own.md", pageCiting(w, "project/inner", "src/a.go"))
	report := mustReconcile(t, w, w.Now())
	got := map[string]string{}
	for _, c := range report.Cases {
		got[c.Area+" "+c.Target] = c.ID
	}
	if len(got) != 2 || got["project/inner a.md"] == "" || got["hub own.md"] == "" {
		t.Fatalf("cases %v, want a.md of project/inner and own.md of hub", got)
	}
}

// A merge asks every open page of the area's wiki, and a wiki nested in it
// is another area's: its open page is no candidate of the enclosing area.
func TestAMergeAsksNoPageOfANestedWiki(t *testing.T) {
	w := mergeWorld(t)
	one := w.Areas[0]
	w.addArea(t, areaOptions{Scope: "project/nested", Files: map[string]string{"x.go": "package x\n"}})
	w.Areas[1].WikiPath = filepath.Join(one.WikiPath, "nested")
	writeUnder(t, w.Areas[1].WikiPath, "open.md", citingPage("open.md", nil, nil, "planned"))
	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 || report.Cases[0].Area != "project/one" || report.Cases[0].Target != "a.md" {
		t.Fatalf("cases %+v, want the one case for a.md of project/one", report.Cases)
	}
}
