package resolve

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
)

func TestWebOf(t *testing.T) {
	script := func(src string) extract.Import { return extract.Import{Kind: "script", Path: src} }
	sheet := func(href string) extract.Import { return extract.Import{Kind: "stylesheet", Path: href} }
	files := []extract.Result{
		{Path: "site/b.html", Language: "html", Imports: []extract.Import{script("js/x.js")}},
		{Path: "site/a.html", Language: "html", Imports: []extract.Import{
			script("js/x.js"), sheet("z.css"), script("js/x.js"), script("/root.js"), script("../up.js?v=2#top"),
			script("https://cdn.example/x.js"), script("//cdn.example/y.js"), script("data:text/javascript,1"),
			script("%zz"), {Path: "other.js"}, {Kind: "style", Path: "s.js"},
			sheet("/a.css"), sheet("z.css?v=1"), sheet("https://cdn.example/x.css"),
		}},
		{Path: "site/c.js", Language: "javascript", Imports: []extract.Import{script("not-a-page.js"), sheet("not-a-page.css")}},
	}
	w := webOf(files)
	if want := map[string][]string{
		"site/js/x.js": {"site/a.html", "site/b.html"},
		"root.js":      {"site/a.html"},
		"up.js":        {"site/a.html"},
	}; !reflect.DeepEqual(w.pages, want) {
		t.Errorf("pages = %v, want %v", w.pages, want)
	}
	if want := map[string][]string{
		"site/b.html": {"site/js/x.js"},
		"site/a.html": {"site/js/x.js", "root.js", "up.js"},
	}; !reflect.DeepEqual(w.scripts, want) {
		t.Errorf("scripts = %v, want %v", w.scripts, want)
	}
	if want := map[string][]string{
		"site/a.html": {"site/z.css", "a.css"},
	}; !reflect.DeepEqual(w.sheets, want) {
		t.Errorf("sheets = %v, want %v", w.sheets, want)
	}
}
