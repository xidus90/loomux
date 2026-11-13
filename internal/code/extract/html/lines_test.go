package html_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
)

// The parts are read from a masked copy of the page: their spans and lines
// are those of the page, and their bodies stay out of the file node's.
func TestLines(t *testing.T) {
	r := extractSrc(t, `<!doctype html>
<link rel=stylesheet href="s.css">
<style>
.a { }
</style>
<div class="k" onclick="go()">
<script>
function f() {
  g();
}
</script>
<p class="k" onclick="go()"></p>
</div>
`)
	same(t, "css", views(part(t, r, "css").Nodes), []string{"p.html#.a rule L4-L4 exported=true sig=.a"})
	js := part(t, r, "javascript")
	same(t, "js", views(js.Nodes), []string{"p.html#f function L8-L10 exported=false sig=function f()"})
	if c := js.Edges[len(js.Edges)-1]; c.Name != "g" || c.Line != 9 {
		t.Errorf("call = %s line %d, want g on line 9", c.Name, c.Line)
	}
	type key struct {
		rel  model.Relation
		name string
	}
	got := map[key]int{}
	for _, e := range r.Edges {
		got[key{e.Relation, e.Name + e.Specifier}] = e.Line
	}
	// The second class="k" and onclick="go()" are the same handovers; the
	// first line is kept.
	same(t, "lines", []map[key]int{got}, []map[key]int{{
		{model.RelationImports, "s.css"}: 2,
		{model.RelationReferences, "k"}:  6,
		{model.RelationCalls, "go"}:      6,
	}})
	file := r.Nodes[0]
	if file.Span != "L1-L14" || !strings.Contains(file.BodyText, "doctype") || strings.Contains(file.BodyText, "g();") || strings.Contains(file.BodyText, ".a {") {
		t.Errorf("file = %s %q, want L1-L14 without the bodies of the parts", file.Span, file.BodyText)
	}
}
