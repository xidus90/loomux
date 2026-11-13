package html_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/ecmascript"
	"github.com/xidus90/loomux/internal/code/extract/html"
	"github.com/xidus90/loomux/internal/code/extract/stylesheet"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, src string) extract.Result {
	t.Helper()
	r, err := html.File("p.html", src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// view is a node as most tests compare it: everything but its body.
func view(n model.Node) string {
	return fmt.Sprintf("%s %s %s exported=%v sig=%s", n.ID, n.Kind, n.Span, n.Exported, n.Signature)
}

func views(nodes []model.Node) []string {
	var out []string
	for _, n := range nodes {
		out = append(out, view(n))
	}
	return out
}

func edgesOf(r extract.Result, rel model.Relation) []extract.RawEdge {
	var out []extract.RawEdge
	for _, e := range r.Edges {
		if e.Relation == rel {
			e.Line = 0 // lines are asserted in TestLines
			out = append(out, e)
		}
	}
	return out
}

// part is the embedded part of language lang, failing the test without one.
func part(t *testing.T, r extract.Result, lang string) extract.Result {
	t.Helper()
	for _, p := range r.Embedded {
		if p.Language == lang {
			return p
		}
	}
	t.Fatalf("no %s part among %d", lang, len(r.Embedded))
	return extract.Result{}
}

func contains(dst model.NodeID) extract.RawEdge {
	return extract.RawEdge{Source: "p.html", Relation: model.RelationContains, TargetID: dst, File: "p.html"}
}

func offer(ns extract.Namespace, name string, target model.NodeID) extract.Export {
	return extract.Export{Namespace: ns, Name: name, Target: target}
}

// same compares two lists and treats nil and empty as equal.
func same[T any](t *testing.T, what string, got, want []T) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %+v\nwant %+v", what, got, want)
	}
}

func TestLanguageDescribesItself(t *testing.T) {
	l := html.Language{}
	if l.Name() != "html" || l.Version() != "html/1@"+treesitter.Parser || !reflect.DeepEqual(l.Extensions(), []string{".html", ".htm"}) {
		t.Errorf("Language = %s %s %v, want html html/1@%s [.html .htm]", l.Name(), l.Version(), l.Extensions(), treesitter.Parser)
	}
	r, err := l.File("a.htm", "<p></p>\n")
	if err != nil || r.Language != "html" || r.Path != "a.htm" {
		t.Errorf("File = %s %s, %v; want a.htm html", r.Path, r.Language, err)
	}
}

// A page's cache entry carries its parts but is keyed on html's version
// alone: when an embedded extractor's version moves, html's must move with
// it, or cached pages keep parts of the old shape.
func TestEmbeddedVersionsArePinned(t *testing.T) {
	for _, c := range []struct {
		l    extract.Language
		want string
	}{
		{ecmascript.JavaScript{}, "javascript/1@" + treesitter.Parser},
		{ecmascript.TypeScript{}, "typescript/1@" + treesitter.Parser},
		{stylesheet.CSS{}, "css/1@" + treesitter.Parser},
	} {
		if got := c.l.Version(); got != c.want {
			t.Errorf("%s is %s, pinned at %s: bump html's version with it, then the pin", c.l.Name(), got, c.want)
		}
	}
}

func TestElementsWithAnIDGetNodes(t *testing.T) {
	r := extractSrc(t, `<div id="app">
  <p id=x>t</p>
  <span id="">e</span>
  <b ID="x"></b>
</div>
`)
	same(t, "nodes", views(r.Nodes), []string{
		"p.html file L1-L6 exported=true sig=",
		`p.html##app element L1-L5 exported=true sig=<div id="app">`,
		"p.html##x element L2-L2 exported=true sig=<p id=x>",
		`p.html##x~2 element L4-L4 exported=true sig=<b ID="x">`,
	})
	if r.Nodes[1].Name != "#app" || r.Nodes[1].Owner != "" {
		t.Errorf("element = name %q owner %q, want #app and no owner", r.Nodes[1].Name, r.Nodes[1].Owner)
	}
	if r.Nodes[1].BodyHash != extract.Hash("<div id=\"app\">\n  <p id=x>t</p>\n  <span id=\"\">e</span>\n  <b ID=\"x\"></b>\n</div>") {
		t.Error("the body hash is not over the whole element")
	}
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{contains("p.html##app"), contains("p.html##x"), contains("p.html##x~2")})
	same(t, "exports", r.Exports, []extract.Export{
		offer(extract.NamespaceHTMLID, "app", "p.html##app"),
		offer(extract.NamespaceHTMLID, "x", "p.html##x"),
		offer(extract.NamespaceHTMLID, "x", "p.html##x~2"),
	})
	if r.ParseErrors != 0 || len(r.Embedded) != 0 {
		t.Errorf("ParseErrors = %d, parts = %d; want none", r.ParseErrors, len(r.Embedded))
	}
}

func TestScriptsAndStylesheetsAreImports(t *testing.T) {
	r := extractSrc(t, `<script src="app.js?v=3"></script>
<link rel="preload stylesheet" href="s.css">
<link rel="icon" href="i.png">
<a href="o.html">o</a>
<LINK REL=StyleSheet HREF=t.css>
<script src="b.js">function ignored() {}</script>
<script src="">function empty() {}</script>
<script src>function bare() {}</script>
`)
	same(t, "imports", r.Imports, []extract.Import{
		{Kind: "script", Path: "app.js?v=3"},
		{Kind: "stylesheet", Path: "s.css"},
		{Kind: "stylesheet", Path: "t.css"},
		{Kind: "script", Path: "b.js"},
	})
	imp := func(spec string) extract.RawEdge {
		return extract.RawEdge{Source: "p.html", Relation: model.RelationImports, Specifier: spec, File: "p.html"}
	}
	same(t, "edges", edgesOf(r, model.RelationImports), []extract.RawEdge{imp("app.js?v=3"), imp("s.css"), imp("t.css"), imp("b.js")})
	// A browser never runs the body of a script with src, whatever its value.
	if len(r.Embedded) != 0 {
		t.Errorf("parts = %d; want none", len(r.Embedded))
	}
}

// The spec's test: two inline scripts share one part and so one minting of
// ids, and the page keeps its single file node.
func TestInlineScriptsShareOnePart(t *testing.T) {
	r := extractSrc(t, `<script>
function init() {}
</script>
<p>x</p>
<script>
function init() {}
</script>
`)
	files := 0
	for _, p := range r.Flatten() {
		for _, n := range p.Nodes {
			if n.Kind == model.KindFile {
				files++
			}
		}
	}
	if files != 1 || len(r.Embedded) != 1 {
		t.Fatalf("file nodes = %d, parts = %d; want 1 and 1", files, len(r.Embedded))
	}
	js := part(t, r, "javascript")
	same(t, "nodes", views(js.Nodes), []string{
		"p.html#init function L2-L2 exported=false sig=function init()",
		"p.html#init~2 function L6-L6 exported=false sig=function init()",
	})
	same(t, "contains", edgesOf(js, model.RelationContains), []extract.RawEdge{contains("p.html#init"), contains("p.html#init~2")})
	same(t, "exports", js.Exports, []extract.Export{
		offer(extract.NamespaceJSFunction, "init", "p.html#init"),
		offer(extract.NamespaceJSFunction, "init", "p.html#init~2"),
	})
}

// Review Focus 1: a script of another type is data or a template, not code.
func TestOnlyJavaScriptTypesAreParts(t *testing.T) {
	for _, typ := range []string{"text/template", "application/ld+json", "importmap", "speculationrules", "text/x-handlebars"} {
		r := extractSrc(t, `<script type="`+typ+`">function f() {}</script>`+"\n")
		if len(r.Embedded) != 0 {
			t.Errorf("type %q: %d parts, want none", typ, len(r.Embedded))
		}
	}
	for _, typ := range []string{"", "module", "text/javascript", "application/javascript", "text/ecmascript", "application/ecmascript", " Text/JavaScript "} {
		r := extractSrc(t, `<script type="`+typ+`">function f() {}</script>`+"\n")
		if len(r.Embedded) != 1 || r.Embedded[0].Language != "javascript" || len(r.Embedded[0].Nodes) != 1 {
			t.Errorf("type %q: parts %+v, want one javascript part with f", typ, r.Embedded)
		}
	}
}

// Review Focus 2: without the ; the mask puts after each body, the parser
// would read b(function () { … }) across the two scripts.
func TestEachScriptEndsItsStatement(t *testing.T) {
	r := extractSrc(t, "<script>var a = b</script>\n<script>(function () { g(); })();</script>\n")
	var names []string
	for _, e := range edgesOf(part(t, r, "javascript"), model.RelationCalls) {
		names = append(names, e.Name)
	}
	same(t, "calls", names, []string{"g"})
}

// Only a classic script's functions are global; a module's stay in it.
func TestOnlyClassicFunctionsAreOffered(t *testing.T) {
	r := extractSrc(t, `<script type="module">
import "./a.js";
function m() {}
</script>
<script>
function c() {}
const k = () => {};
class K {}
</script>
`)
	js := part(t, r, "javascript")
	same(t, "exports", js.Exports, []extract.Export{
		offer(extract.NamespaceJSFunction, "c", "p.html#c"),
		offer(extract.NamespaceJSFunction, "k", "p.html#k"),
	})
	if len(js.Imports) != 1 || js.Imports[0].Path != "./a.js" || len(r.Imports) != 0 {
		t.Errorf("imports: part %+v, page %+v; want ./a.js in the part alone", js.Imports, r.Imports)
	}
}

func TestTypeScriptIsAPartOfItsOwn(t *testing.T) {
	r := extractSrc(t, `<script lang="ts">function f(a: number) {}</script>
<script type="text/typescript">function g(b: string) {}</script>
<script lang="ts" type="text/template">function h(c: string) {}</script>
`)
	if len(r.Embedded) != 1 {
		t.Fatalf("parts = %d, want the typescript part alone", len(r.Embedded))
	}
	ts := part(t, r, "typescript")
	same(t, "nodes", views(ts.Nodes), []string{
		"p.html#f function L1-L1 exported=false sig=function f(a: number)",
		"p.html#g function L2-L2 exported=false sig=function g(b: string)",
	})
	if ts.ParseErrors != 0 || len(ts.Exports) != 0 {
		t.Errorf("ParseErrors = %d, exports = %v; want none", ts.ParseErrors, ts.Exports)
	}
}

// A script without an end tag still has its body: the grammar leaves the
// start tag and the raw text side by side.
func TestAnUnclosedScriptIsRead(t *testing.T) {
	r := extractSrc(t, "<p>\n<script>\nfunction f() {}\n")
	same(t, "nodes", views(part(t, r, "javascript").Nodes), []string{"p.html#f function L3-L3 exported=false sig=function f()"})
}

// The unclosed <p> breaks the second style element up; its body is read
// all the same.
func TestStylesAreACSSPart(t *testing.T) {
	r := extractSrc(t, "<style>.btn{}</style>\n<p>\n<style>\n.k{}\n</style>\n")
	css := part(t, r, "css")
	same(t, "nodes", views(css.Nodes), []string{"p.html#.btn rule L1-L1 exported=true sig=.btn", "p.html#.k rule L4-L4 exported=true sig=.k"})
	same(t, "exports", css.Exports, []extract.Export{
		offer(extract.NamespaceCSSClass, "btn", "p.html#.btn"),
		offer(extract.NamespaceCSSClass, "k", "p.html#.k"),
	})
}

// The elements mint first, then each part in the same set: a part's id
// that is taken gets the next ordinal, and its edges and offers follow.
func TestPartsMintAfterTheElements(t *testing.T) {
	r := extractSrc(t, "<div id=\"app\"></div>\n<style>#app{}</style>\n")
	css := part(t, r, "css")
	same(t, "nodes", views(css.Nodes), []string{"p.html##app~2 rule L2-L2 exported=true sig=#app"})
	same(t, "contains", edgesOf(css, model.RelationContains), []extract.RawEdge{contains("p.html##app~2")})
	same(t, "exports", css.Exports, []extract.Export{offer(extract.NamespaceCSSID, "app", "p.html##app~2")})
	if r.Nodes[1].ID != "p.html##app" {
		t.Errorf("element = %s, want p.html##app", r.Nodes[1].ID)
	}

	r = extractSrc(t, "<script>function f() {}</script>\n<script lang=\"ts\">function f() { g(); document.body.classList.add(\"k\"); }</script>\n")
	ts := part(t, r, "typescript")
	same(t, "nodes", views(ts.Nodes), []string{`p.html#f~2 function L2-L2 exported=false sig=function f()`})
	same(t, "contains", edgesOf(ts, model.RelationContains), []extract.RawEdge{contains("p.html#f~2")})
	for _, e := range edgesOf(ts, model.RelationCalls) {
		if e.Source != "p.html#f~2" {
			t.Errorf("call %s from %s, want from p.html#f~2", e.Name, e.Source)
		}
	}
	if len(ts.Bridges) != 1 || ts.Bridges[0].Source != "p.html#f~2" {
		t.Errorf("bridges = %+v, want one from p.html#f~2", ts.Bridges)
	}
}

// Template syntax in a class attribute is no class: neither its markers
// nor the names inside them are handed over.
func TestTemplateClassesAreNotHandedOver(t *testing.T) {
	r := extractSrc(t, `<p class="{{ cls }} x {% if a %}on{% endif %} <%= k %> a{{b}} y">t</p>`+"\n")
	var names []string
	for _, e := range edgesOf(r, model.RelationReferences) {
		names = append(names, e.Name)
	}
	same(t, "classes", names, []string{"x", "y"})
}

func TestClassesAndHandlersAreHandedOver(t *testing.T) {
	r := extractSrc(t, `<div class="a b">
<p class="b c">
<span class=" a ">
</span></p></div>
<button onclick="go(1)" ONSUBMIT="return check()" onmouseover="a.b()" onblur="x = 1">
<i onclick=" go ()">
`)
	ref := func(name string) extract.RawEdge {
		return extract.RawEdge{Source: "p.html", Relation: model.RelationReferences, Name: name, Member: "css.class", File: "p.html"}
	}
	call := func(name string) extract.RawEdge {
		return extract.RawEdge{Source: "p.html", Relation: model.RelationCalls, Name: name, Member: "js.function", File: "p.html"}
	}
	same(t, "references", edgesOf(r, model.RelationReferences), []extract.RawEdge{ref("a"), ref("b"), ref("c")})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("go"), call("check")})
	if len(r.Bridges) != 0 {
		t.Errorf("bridges = %+v, want handovers as raw edges alone", r.Bridges)
	}
}

// Template placeholders are no HTML; the page is read all the same, and the
// page counts only what its own grammar reports.
func TestTemplatePlaceholdersBreakNothing(t *testing.T) {
	if r := extractSrc(t, "<div id=\"a\">{{ name }}</div>\n"); r.ParseErrors != 0 || len(r.Nodes) != 2 {
		t.Errorf("{{ }}: %d errors, %d nodes; want none and the element", r.ParseErrors, len(r.Nodes))
	}
	page := func(script string) extract.Result {
		return extractSrc(t, `<p id="b">
<% if (x) { %>
<span id="c">x</span>
<% } %>
</p>
<i id="d"></i>
<script>`+script+`</script>
`)
	}
	r := page("function f( {")
	var ids []model.NodeID
	for _, n := range r.Nodes[1:] {
		ids = append(ids, n.ID)
	}
	same(t, "elements", ids, []model.NodeID{"p.html##b", "p.html##c", "p.html##d"})
	if r.ParseErrors == 0 || part(t, r, "javascript").ParseErrors == 0 {
		t.Fatalf("ParseErrors = %d, part %d; want both broken", r.ParseErrors, part(t, r, "javascript").ParseErrors)
	}
	if clean := page("function f() {}"); clean.ParseErrors != r.ParseErrors {
		t.Errorf("ParseErrors = %d with a broken script, %d without; the part's errors stay in the part", r.ParseErrors, clean.ParseErrors)
	}
}
