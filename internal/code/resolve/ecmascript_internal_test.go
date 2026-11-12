package resolve

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
)

func TestScriptPages(t *testing.T) {
	script := func(src string) extract.Import { return extract.Import{Kind: "script", Path: src} }
	files := []extract.Result{
		{Path: "site/b.html", Language: "html", Imports: []extract.Import{script("js/x.js")}},
		{Path: "site/a.html", Language: "html", Imports: []extract.Import{
			script("js/x.js"), script("js/x.js"), script("/root.js"), script("../up.js?v=2#top"),
			script("https://cdn.example/x.js"), script("//cdn.example/y.js"), script("data:text/javascript,1"),
			script("%zz"), {Path: "other.js"}, {Kind: "style", Path: "s.js"},
		}},
		{Path: "site/c.js", Language: "javascript", Imports: []extract.Import{script("not-a-page.js")}},
	}
	want := map[string][]string{
		"site/js/x.js": {"site/a.html", "site/b.html"},
		"root.js":      {"site/a.html"},
		"up.js":        {"site/a.html"},
	}
	if got := scriptPages(files); !reflect.DeepEqual(got, want) {
		t.Errorf("scriptPages = %v, want %v", got, want)
	}
}
