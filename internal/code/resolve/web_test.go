package resolve_test

import (
	"encoding/json"
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/ecmascript"
	"github.com/xidus90/loomux/internal/code/extract/html"
	"github.com/xidus90/loomux/internal/code/extract/stylesheet"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// webFiles extracts a tree of pages, stylesheets and scripts, path to
// source, each with the extractor its extension picks, in reverse byte
// order: no answer may depend on it.
func webFiles(t *testing.T, tree map[string]string) []extract.Result {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var files []extract.Result
	for _, p := range paths {
		var r extract.Result
		var err error
		switch path.Ext(p) {
		case ".html":
			r, err = html.Language{}.File(p, tree[p])
		case ".css", ".scss":
			r, err = stylesheet.File(p, tree[p])
		default:
			r, err = ecmascript.File(p, tree[p])
		}
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	return files
}

// webGraph resolves the tree webFiles extracts.
func webGraph(t *testing.T, tree map[string]string) *model.Graph {
	t.Helper()
	g, _ := resolve.GraphWith(webFiles(t, tree), nil, stamp)
	return validated(t, g)
}

func TestWebStylesheetFamily(t *testing.T) {
	g := webGraph(t, map[string]string{
		"main.scss":  "@use \"base\";\n",
		"_base.scss": "",
		"a.css":      "@import \"x.css\";\n",
		"x.css":      "",
		"p.html":     "<p>x</p>\n",
		"app.js":     "",
	})
	wantEdges(t, g,
		"a.css -imports-> x.css extracted",
		"main.scss -imports-> _base.scss extracted",
	)
	if want := []string{"css", "html", "javascript", "scss"}; !reflect.DeepEqual(g.Meta.Languages, want) {
		t.Errorf("languages = %v, want %v", g.Meta.Languages, want)
	}
}

func TestWebHTMLImports(t *testing.T) {
	g := webGraph(t, map[string]string{
		"site/p.html": `<link rel="stylesheet" href="s.css">
<script src="/root.js"></script>
<script src="app.js?v=3#x"></script>
<link rel="stylesheet" href="a%20b.css">
<script src="app.js"></script>
<script src="https://cdn.example/x.js"></script>
<script src="//cdn/x.js"></script>
<script src="%zz"></script>
<script src="missing.js"></script>
`,
		"site/s.css":    "",
		"site/a b.css":  "",
		"site/app.js":   "",
		"root.js":       "",
		"site/root.js":  "",
		"cdn/x.js":      "",
		"site/cdn/x.js": "",
		"site/%zz":      "",
	})
	wantEdges(t, g,
		"site/p.html -imports-> root.js extracted",
		"site/p.html -imports-> site/a b.css extracted",
		"site/p.html -imports-> site/app.js extracted",
		"site/p.html -imports-> site/s.css extracted",
	)
	// Two srcs naming one file give one edge, at the first line.
	if e := edgeBetween(g, "site/p.html", "site/app.js", model.RelationImports); e == nil || e.Line != 3 {
		t.Errorf("app.js import = %+v, want line 3", e)
	}
}

func TestWebClassesReachTheLinkedRules(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html":    "<link rel=\"stylesheet\" href=\"site.css\">\n<div class=\"btn\"></div>\n",
		"site.css":  ".btn {}\n",
		"other.css": ".btn {}\n",
	})
	wantEdges(t, g, "p.html -imports-> site.css extracted", "p.html -references-> site.css#.btn extracted")
}

func TestWebClassesReachTheirOwnStyle(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html":    "<style>.btn {}</style>\n<div class=\"btn\"></div>\n",
		"other.css": ".btn {}\n",
	})
	wantEdges(t, g, "p.html -references-> p.html#.btn extracted")
}

func TestWebClassesWithoutAStylesheetGiveNoEdge(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html":    "<div class=\"btn\"></div>\n",
		"other.css": ".btn {}\n",
	})
	wantEdges(t, g)
}

func TestWebClassesReachEveryRule(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html":   "<link rel=\"stylesheet\" href=\"site.css\">\n<div class=\"btn\"></div>\n",
		"site.css": ".btn {}\n.btn {}\n",
	})
	wantEdges(t, g,
		"p.html -imports-> site.css extracted",
		"p.html -references-> site.css#.btn extracted",
		"p.html -references-> site.css#.btn~2 extracted",
	)
}

func TestWebHandlersReachTheLoadedScripts(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html":   "<script src=\"app.js\"></script>\n<button onclick=\"go()\"></button>\n",
		"app.js":   "function go() {}\n",
		"other.js": "function go() {}\n",
	})
	wantEdges(t, g,
		"p.html -calls-> app.js#go extracted",
		"p.html -imports-> app.js extracted",
	)
}

func TestWebHandlersReachInlineFunctions(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html":   "<script>function go() {}</script>\n<button onclick=\"go()\"></button>\n",
		"other.js": "function go() {}\n",
	})
	wantEdges(t, g, "p.html -calls-> p.html#go extracted")
}

func TestWebHandlersMissInlineModules(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html": "<script type=\"module\">function go() {}</script>\n<button onclick=\"go()\"></button>\n",
	})
	wantEdges(t, g)
}

// A module beside a classic script offers nothing itself and hides none
// of the classic functions.
func TestWebHandlersReachClassicFunctionsBesideAModule(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html": "<script>function go() {}</script>\n<script type=\"module\">import \"./x.js\";\nfunction m() {}</script>\n" +
			"<button onclick=\"go()\" ondblclick=\"m()\"></button>\n",
		"x.js": "",
	})
	wantEdges(t, g,
		"p.html -calls-> p.html#go extracted",
		"p.html -imports-> x.js extracted",
	)
}

// Two loaded scripts that both define the handler: js.function takes
// exactly one, so neither is the answer.
func TestWebHandlerWithTwoLoadedDefinitionsGivesNoEdge(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html": "<script src=\"a.js\"></script>\n<script src=\"b.js\"></script>\n<button onclick=\"init()\"></button>\n",
		"a.js":   "function init() {}\n",
		"b.js":   "function init() {}\n",
	})
	wantEdges(t, g,
		"p.html -imports-> a.js extracted",
		"p.html -imports-> b.js extracted",
	)
}

// One script on two pages that both hold id="app": html.id takes exactly
// one element, so neither is the answer.
func TestWebSharedScriptWithTwoElementsGivesNoEdge(t *testing.T) {
	page := "<div id=\"app\"></div>\n<script src=\"app.js\"></script>\n"
	app := "function f() { document.getElementById(\"app\"); }\n"
	g := webGraph(t, map[string]string{"a.html": page, "b.html": page, "app.js": app})
	wantEdges(t, g,
		"a.html -imports-> app.js extracted",
		"b.html -imports-> app.js extracted",
	)
	g = webGraph(t, map[string]string{"a.html": page, "app.js": app})
	wantEdges(t, g,
		"a.html -imports-> app.js extracted",
		"app.js#f -references-> a.html##app extracted",
	)
}

func TestWebSCSSCallsThroughUseAndImport(t *testing.T) {
	g := webGraph(t, map[string]string{
		"main.scss":    "@use \"mixins\";\n@import \"old\";\n.a { @include mixins.m; @include n; @include mixins.n; }\n",
		"_mixins.scss": "@mixin m {}\n",
		"old.scss":     "@mixin n {}\n",
	})
	wantCalls(t, g, "main.scss#.a", "_mixins.scss#m extracted", "old.scss#n extracted")
	// A call is an edge per use, as in every other language: blast counts
	// each.
	g = webGraph(t, map[string]string{
		"main.scss":    "@use \"mixins\";\n.a { @include mixins.m; @include mixins.m; }\n",
		"_mixins.scss": "@mixin m {}\n",
	})
	wantCalls(t, g, "main.scss#.a", "_mixins.scss#m extracted", "_mixins.scss#m extracted")
}

// The grammar rejects @use … as; the extractor reads the namespace from
// the rejected text.
func TestWebSCSSCallsThroughANamespace(t *testing.T) {
	g := webGraph(t, map[string]string{
		"main.scss":      "@use \"lib/x\" as y;\n.a { @include y.m; @include x.m; }\n",
		"lib/x.scss":     "@forward \"more\";\n",
		"lib/_more.scss": "@mixin m {}\n",
	})
	wantCalls(t, g, "main.scss#.a", "lib/_more.scss#m extracted")
}

func TestWebSCSSCycleEnds(t *testing.T) {
	g := webGraph(t, map[string]string{
		"a.scss": "@use \"b\";\n.a { @include m; }\n",
		"b.scss": "@use \"a\";\n",
	})
	wantEdges(t, g, "a.scss -imports-> b.scss extracted", "b.scss -imports-> a.scss extracted")
}

// The nearest files decide; two of them alike give no edge, though a
// farther one defines the mixin too.
func TestWebSCSSTwoDefinitionsAtOneDepthGiveNoEdge(t *testing.T) {
	g := webGraph(t, map[string]string{
		"main.scss": "@use \"a\";\n@use \"b\";\n.x { @include m; @include f; }\n",
		"a.scss":    "@use \"c\";\n@mixin m {}\n",
		"b.scss":    "@mixin m {}\n",
		"c.scss":    "@mixin f {}\n@mixin m {}\n",
	})
	wantCalls(t, g, "main.scss#.x", "c.scss#f extracted")
}

func TestWebSCSSFunctionsAndBuiltins(t *testing.T) {
	g := webGraph(t, map[string]string{
		"main.scss": "@use \"sass:math\";\n@function f($a) { @return $a; }\n@mixin f {}\n" +
			".a { color: rgba(0, 0, 0, f(1)); width: math.div(1, 2); }\n",
	})
	wantEdges(t, g, "main.scss#.a -calls-> main.scss#f extracted")
}

func TestWebSCSSExtends(t *testing.T) {
	g := webGraph(t, map[string]string{
		"main.scss": "@use \"base\";\n.a { @extend %p; @extend .k; }\n.k:hover { @extend .k; }\n",
		"base.scss": "%p { color: red; }\n.k { color: red; }\n.x .k { color: blue; }\n",
	})
	wantEdges(t, g,
		"main.scss -imports-> base.scss extracted",
		"main.scss#.a -extends-> base.scss#%p extracted",
		"main.scss#.a -extends-> base.scss#.k extracted",
		"main.scss#.a -extends-> base.scss#.x .k extracted",
		"main.scss#.a -extends-> main.scss#.k:hover extracted",
		"main.scss#.k:hover -extends-> base.scss#.k extracted",
		"main.scss#.k:hover -extends-> base.scss#.x .k extracted",
	)
}

// Two builds of the same pages, styles and scripts are the same bytes.
func TestWebGraphIsDeterministic(t *testing.T) {
	tree := map[string]string{
		"p.html":    "<link rel=\"stylesheet\" href=\"s.scss\">\n<script src=\"app.js\"></script>\n<style>.b {}</style>\n<div id=\"app\" class=\"a b\" onclick=\"go()\"></div>\n",
		"s.scss":    "@use \"m\";\n.a { @include m.x; @include m.x; }\n.b { @extend .a; }\n",
		"_m.scss":   "@mixin x {}\n",
		"app.js":    "function go() { document.getElementById(\"app\"); }\n",
		"other.css": ".a {}\n",
	}
	var out [2][]byte
	for i := range out {
		g, _ := resolve.GraphWith(webFiles(t, tree), nil, stamp)
		b, err := json.Marshal(validated(t, g))
		if err != nil {
			t.Fatal(err)
		}
		out[i] = b
	}
	if string(out[0]) != string(out[1]) {
		t.Fatal("the same input gave two graphs")
	}
}
