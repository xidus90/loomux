package resolve_test

import (
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/ecmascript"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// esGraph extracts a tree of JavaScript and TypeScript files, path to
// source, with the real extractor, adds the results of other languages,
// and resolves them with the modules mods. The files go in in reverse
// byte order: no answer may depend on it.
func esGraph(t *testing.T, tree map[string]string, mods []resolve.Module, foreign ...extract.Result) (*model.Graph, []resolve.Unresolved) {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var files []extract.Result
	for _, p := range paths {
		r, err := ecmascript.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	g, un := resolve.GraphWith(append(files, foreign...), mods, stamp)
	return validated(t, g), un
}

func TestECMAScriptImportsByExtension(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"src/main.ts": `import "./a";
import "./b";
import "./c";
import "./d";
import "./e";
import "./f";
import "./g";
import "./h";
import "./i.mjs";
import "../top";
import "./x.css";
import "react";
import "./types";
import "./missing";
import "./main";
import "./a";
const lazy = () => import("./c");
import "..";
`,
		"src/a.ts":       "",
		"src/b.tsx":      "",
		"src/c.js":       "",
		"src/d.jsx":      "",
		"src/e.mjs":      "",
		"src/f.cjs":      "",
		"src/g/index.ts": "",
		"src/h/index.js": "",
		"src/i.mjs":      "",
		"top.ts":         "",
		"src/types.d.ts": "",
		"src/types.js":   "",
		"index.js":       "",
	}, nil)
	wantEdges(t, g,
		"src/main.ts -imports-> src/a.ts extracted",
		"src/main.ts -imports-> src/b.tsx extracted",
		"src/main.ts -imports-> src/c.js extracted",
		"src/main.ts -imports-> src/d.jsx extracted",
		"src/main.ts -imports-> src/e.mjs extracted",
		"src/main.ts -imports-> src/f.cjs extracted",
		"src/main.ts -imports-> src/g/index.ts extracted",
		"src/main.ts -imports-> src/h/index.js extracted",
		"src/main.ts -imports-> src/i.mjs extracted",
		"src/main.ts -imports-> top.ts extracted",
		"src/main.ts -imports-> index.js extracted",
	)
	if e := edgeBetween(g, "src/main.ts", "src/a.ts", model.RelationImports); e == nil || e.Line != 1 {
		t.Errorf("two imports of one file make one edge at the smaller line: %+v", e)
	}
}

// A TypeScript file imports what it will import once compiled: ./a.js
// finds a.ts, but a.js itself wins when both are there. A JavaScript file
// gets no such rewrite.
func TestECMAScriptImportsJSFromTypeScript(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"m.ts":     `import "./a.js"; import "./b.js"; import "./c.mjs"; import "./d.jsx";`,
		"plain.js": `import "./a.js";`,
		"a.ts":     "",
		"b.js":     "",
		"b.ts":     "",
		"c.mts":    "",
		"d.tsx":    "",
	}, nil)
	wantEdges(t, g,
		"m.ts -imports-> a.ts extracted",
		"m.ts -imports-> b.js extracted",
		"m.ts -imports-> c.mts extracted",
		"m.ts -imports-> d.tsx extracted",
	)
}

func TestECMAScriptImportsThroughTSConfig(t *testing.T) {
	mods := []resolve.Module{
		{Dir: "app", TS: &resolve.TSConfig{Paths: map[string][]string{
			"@/*":      {"src/*"},
			"@/deep/*": {"other/*"},
			"#cfg":     {"src/config.ts"},
			"$m/*":     {"gone/*", "src/*"},
			"*.gen":    {"gen/*.ts"},
		}}},
		{Dir: "pkg", TS: &resolve.TSConfig{BaseURL: "lib"}},
		{Dir: "both", TS: &resolve.TSConfig{BaseURL: "base", Paths: map[string][]string{"~/*": {"sub/*"}}}},
	}
	g, _ := esGraph(t, map[string]string{
		"app/main.ts":        `import "@/util"; import "@/deep/x"; import "#cfg"; import "$m/util"; import "a.gen"; import "react";`,
		"app/src/util.ts":    "",
		"app/src/config.ts":  "",
		"app/other/x.ts":     "",
		"app/gen/a.ts":       "",
		"pkg/main.js":        `import "util"; import "react";`,
		"pkg/lib/util.js":    "",
		"both/main.ts":       `import "~/x"; import "y";`,
		"both/base/sub/x.ts": "",
		"both/base/y.ts":     "",
		"loose/main.ts":      `import "@/util";`,
	}, mods)
	wantEdges(t, g,
		"app/main.ts -imports-> app/src/util.ts extracted",
		"app/main.ts -imports-> app/other/x.ts extracted",
		"app/main.ts -imports-> app/src/config.ts extracted",
		"app/main.ts -imports-> app/gen/a.ts extracted",
		"pkg/main.js -imports-> pkg/lib/util.js extracted",
		"both/main.ts -imports-> both/base/sub/x.ts extracted",
		"both/main.ts -imports-> both/base/y.ts extracted",
	)
}

func TestECMAScriptBindings(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"b.ts": `export default function main() {}
export function f() {}
function g() {}
export { g as h };
export class K {}
function hidden() {}
`,
		"anon.ts":  "export default function () {}\n",
		"named.js": "function v() {}\nexport default v;\n",
		"use.ts": `import main, { f, h as hh, nope } from "./b";
import * as ns from "./b";
import d from "./anon";
import v from "./named.js";
function run() {
  main();
  f();
  hh();
  nope();
  ns.f();
  ns.hidden();
  new ns.K();
  d();
  v();
  local();
  missing();
  unknown.f();
}
function local() {}
`,
	}, nil)
	wantCalls(t, g, "use.ts#run",
		"anon.ts#default extracted", "b.ts#K extracted", "b.ts#f extracted", "b.ts#f extracted",
		"b.ts#g extracted", "b.ts#main extracted", "named.js#v extracted", "use.ts#local extracted")
}

func TestECMAScriptCommonJS(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"m.js": `function a() {}
function c() {}
module.exports = { a, b: c };
exports.f = function () {};
module.exports.g = () => {};
`,
		"n.js": "module.exports = function () {};\n",
		"k.js": "function k() {}\nmodule.exports = k;\n",
		"user.js": `const m = require("./m");
const { a, b: bb } = require("./m");
const n = require("./n");
const k = require("./k");
const pkg = require("lodash");
function run() {
  m.b();
  m.f();
  m.g();
  a();
  bb();
  n();
  k();
  pkg.x();
  pkg();
}
`,
		"esm.ts": `import n from "./n.js";
import k from "./k";
function run() { n(); k(); }
`,
	}, nil)
	wantCalls(t, g, "user.js#run",
		"k.js#k extracted", "m.js#a extracted", "m.js#c extracted", "m.js#c extracted",
		"m.js#f extracted", "m.js#g extracted", "n.js#default extracted")
	wantCalls(t, g, "esm.ts#run", "k.js#k extracted", "n.js#default extracted")
}

func TestECMAScriptReExports(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"b.ts":  "export function f() {}\nexport default function main() {}\n",
		"c.ts":  "export function deep() {}\nexport function dup() {}\nexport default function cdefault() {}\n",
		"c2.ts": "export function dup() {}\n",
		"r.ts": `export { f as g, absent } from "./b";
export * from "./c";
export * from "./c2";
export * as nsx from "./b";
export * from "./gone";
export { x } from "./gone";
import { f } from "./b";
export { f as ff };
`,
		"use.ts": `import { g, deep, nsx, absent, ff, dup, x } from "./r";
import main from "./r";
function run() { g(); deep(); nsx.f(); absent(); ff(); dup(); x(); main(); }
`,
	}, nil)
	wantCalls(t, g, "use.ts#run", "b.ts#f extracted", "b.ts#f extracted", "b.ts#f extracted", "c.ts#deep extracted")
}

// export * in a cycle ends, and a name neither side has gives nothing.
func TestECMAScriptReExportCycle(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"cyc/a.ts": "export * from \"./b\";\nexport function own() {}\n",
		"cyc/b.ts": "export * from \"./a\";\n",
		"use.ts":   "import { own, nothere } from \"./cyc/b\";\nfunction run() { own(); nothere(); }\n",
	}, nil)
	wantCalls(t, g, "use.ts#run", "cyc/a.ts#own extracted")
}

func TestECMAScriptClassCalls(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"base.ts": `export class Base {
  constructor() {}
  inherited() {}
  shadowed() {}
}
export function Legacy() {}
`,
		"child.js": `import { Base, Legacy } from "./base";
class Child extends Base {
  shadowed() { super.shadowed(); }
  run() {
    this.inherited();
    this.shadowed();
    this.absent();
    new Child();
    new Base();
    new Legacy();
    new Gone();
  }
}
function free() { this.run(); }
`,
	}, nil)
	wantCalls(t, g, "child.js#Child.shadowed", "base.ts#Base.shadowed extracted")
	wantCalls(t, g, "child.js#Child.run",
		"base.ts#Base.constructor extracted", "base.ts#Base.inherited extracted", "base.ts#Legacy extracted",
		"child.js#Child extracted", "child.js#Child.shadowed extracted")
	wantCalls(t, g, "child.js#free")
}

func TestECMAScriptBases(t *testing.T) {
	g, un := esGraph(t, map[string]string{
		"lib.ts": "export class K {}\nexport interface I {}\nexport function fn() {}\n",
		"m.ts": `import { K, I, fn, Missing } from "./lib";
import * as ns from "./lib";
import * as gone from "./gone";
import React from "react";
import { Aliased } from "@/lib";
interface J {}
interface L extends J {}
class A extends K implements I, J {}
class B extends ns.K {}
class C extends Missing {}
class D extends ns.Nope {}
class E extends React.Component {}
class F extends Error {}
class G extends gone.X {}
class H extends fn {}
class P extends ns {}
class Q extends A.B {}
class R extends Aliased {}
`,
	}, []resolve.Module{{Dir: "", TS: &resolve.TSConfig{Paths: map[string][]string{"@/*": {"*"}}}}})
	wantEdges(t, g,
		"m.ts -imports-> lib.ts extracted",
		"m.ts#L -extends-> m.ts#J extracted",
		"m.ts#A -extends-> lib.ts#K extracted",
		"m.ts#A -implements-> lib.ts#I extracted",
		"m.ts#A -implements-> m.ts#J extracted",
		"m.ts#B -extends-> lib.ts#K extracted",
		"m.ts#H -extends-> lib.ts#fn extracted",
	)
	wantUnresolved(t, un, "Missing none", "ns.Nope none", "gone.X none", "ns none", "Aliased none")
}

// The pages of an HTML result give a script the scope of its element ids
// and make it offer its functions; an inline part has its own page.
func TestECMAScriptWeb(t *testing.T) {
	page := "web/index.html"
	main := model.Node{ID: "web/index.html#main", Name: "main", Kind: "element", Path: page, BodyHash: "h"}
	button := model.Node{ID: "web/index.html#button", Name: "button", Kind: "element", Path: page, BodyHash: "h"}
	inline, err := ecmascript.File(page, "function inline() { document.getElementById(\"main\"); }\n")
	if err != nil {
		t.Fatal(err)
	}
	inline.Nodes = inline.Nodes[1:] // the page's file node stands for the part
	// A classic inline function is a global of the page, which the HTML
	// extractor offers.
	inline.Exports = []extract.Export{{Namespace: extract.NamespaceJSFunction, Name: "inline", Target: "web/index.html#inline"}}
	html := extract.Result{
		Path: page, Language: "html",
		Imports: []extract.Import{{Kind: "script", Path: "app.js"}, {Kind: "script", Path: "/web/cjs.js"}},
		Nodes:   []model.Node{fileNode(page), main, button},
		Exports: []extract.Export{{Namespace: extract.NamespaceHTMLID, Name: "main", Target: main.ID}},
		Bridges: []extract.Bridge{
			{Source: button.ID, Relation: model.RelationCalls, Namespace: extract.NamespaceJSFunction, Name: "greet", File: page, Line: 3},
			{Source: button.ID, Relation: model.RelationCalls, Namespace: extract.NamespaceJSFunction, Name: "helper", File: page, Line: 4},
			{Source: button.ID, Relation: model.RelationCalls, Namespace: extract.NamespaceJSFunction, Name: "lonely", File: page, Line: 5},
			{Source: button.ID, Relation: model.RelationCalls, Namespace: extract.NamespaceJSFunction, Name: "inline", File: page, Line: 6},
		},
		Embedded: []extract.Result{inline},
	}
	rule := model.Node{ID: "web/app.css#.card", Name: ".card", Kind: "rule", Path: "web/app.css", BodyHash: "h"}
	css := extract.Result{
		Path: "web/app.css", Language: "css",
		Nodes:   []model.Node{fileNode("web/app.css"), rule},
		Exports: []extract.Export{{Namespace: extract.NamespaceCSSClass, Name: "card", Target: rule.ID}},
	}
	g, _ := esGraph(t, map[string]string{
		"web/app.js":    "function greet() { document.getElementById(\"main\"); }\n",
		"web/cjs.js":    "function helper() {}\nmodule.exports = { helper };\n",
		"web/orphan.js": "function lonely() {}\n",
		"web/comp.jsx":  "import \"./app.css\";\nexport function C() { return <div className=\"card\" />; }\n",
	}, nil, html, css)
	wantEdges(t, g,
		"web/app.js#greet -references-> web/index.html#main extracted",
		"web/index.html#inline -references-> web/index.html#main extracted",
		"web/comp.jsx#C -references-> web/app.css#.card extracted",
		"web/comp.jsx -imports-> web/app.css extracted",
		// A page's call of a script function is not scoped to its scripts.
		"web/index.html#button -calls-> web/app.js#greet inferred",
		"web/index.html#button -calls-> web/index.html#inline inferred",
	)
}

// A class a module hands over reaches the stylesheets it imports: by a
// relative path, not by a bare name without a tsconfig, nor by a URL.
func TestECMAScriptWebStyleScope(t *testing.T) {
	g := webGraph(t, map[string]string{
		"src/m.js": "import \"./a.css\";\nimport \"../b.scss\";\nimport \"pkg/c.css\";\nimport \"https://cdn.example/d.css\";\n" +
			"function wire(el) { el.classList.add(\"k\"); }\n",
		"src/a.css": ".k {}\n",
		"b.scss":    ".k {}\n",
		"pkg/c.css": ".k {}\n",
	})
	wantEdges(t, g,
		"src/m.js -imports-> src/a.css extracted",
		"src/m.js -imports-> b.scss extracted",
		"src/m.js#wire -references-> src/a.css#.k extracted",
		"src/m.js#wire -references-> b.scss#.k extracted",
	)
}

// A classic script reaches the stylesheets of the pages that load it, and
// their own <style>.
func TestECMAScriptWebClassicScriptReachesThePageStyles(t *testing.T) {
	app := "function w(el) { el.classList.add(\"btn\"); }\n"
	g := webGraph(t, map[string]string{
		"p.html":   "<link rel=\"stylesheet\" href=\"site.css\">\n<script src=\"app.js\"></script>\n",
		"site.css": ".btn {}\n",
		"app.js":   app,
	})
	wantEdges(t, g,
		"p.html -imports-> site.css extracted",
		"p.html -imports-> app.js extracted",
		"app.js#w -references-> site.css#.btn extracted",
	)
	g = webGraph(t, map[string]string{
		"p.html":    "<style>.btn {}</style>\n<script src=\"app.js\"></script>\n",
		"other.css": ".btn {}\n",
		"app.js":    app,
	})
	wantEdges(t, g,
		"p.html -imports-> app.js extracted",
		"app.js#w -references-> p.html#.btn extracted",
	)
}

// css.class takes every hit: one script on two pages, each with its own
// site.css, reaches both.
func TestECMAScriptWebClassOnTwoPagesReachesBoth(t *testing.T) {
	page := "<link rel=\"stylesheet\" href=\"site.css\">\n<script src=\"/app.js\"></script>\n"
	g := webGraph(t, map[string]string{
		"a/p.html":   page,
		"b/p.html":   page,
		"a/site.css": ".btn {}\n",
		"b/site.css": ".btn {}\n",
		"app.js":     "function w(el) { el.classList.add(\"btn\"); }\n",
	})
	wantEdges(t, g,
		"a/p.html -imports-> a/site.css extracted",
		"a/p.html -imports-> app.js extracted",
		"b/p.html -imports-> b/site.css extracted",
		"b/p.html -imports-> app.js extracted",
		"app.js#w -references-> a/site.css#.btn extracted",
		"app.js#w -references-> b/site.css#.btn extracted",
	)
}

// A selector's id reaches the page's element and the linked rule.
func TestECMAScriptWebSelectorIDReachesElementAndRule(t *testing.T) {
	g := webGraph(t, map[string]string{
		"p.html":   "<link rel=\"stylesheet\" href=\"site.css\">\n<div id=\"app\"></div>\n<script src=\"app.js\"></script>\n",
		"site.css": "#app {}\n",
		"app.js":   "function w() { document.querySelector(\"#app\"); }\n",
	})
	wantEdges(t, g,
		"p.html -imports-> site.css extracted",
		"p.html -imports-> app.js extracted",
		"app.js#w -references-> p.html##app extracted",
		"app.js#w -references-> site.css##app extracted",
	)
}

// A module with no page and no stylesheet import has no scope for its
// classes: no edge.
func TestECMAScriptWebClassWithoutScopeGivesNoEdge(t *testing.T) {
	g := webGraph(t, map[string]string{
		"C.jsx":    "export function C() { return <div className=\"a\" />; }\n",
		"site.css": ".a {}\n",
	})
	wantEdges(t, g)
}

// An aliased stylesheet import loads the file under the tsconfig's paths;
// a query after the path does not hide it.
func TestECMAScriptWebAliasedAndQueriedStylesheets(t *testing.T) {
	files := webFiles(t, map[string]string{
		"src/main.tsx":     "import \"@/styles/x.css\";\nimport \"./y.css?inline\";\nexport function C() { return <div className=\"a\" />; }\n",
		"src/styles/x.css": ".a {}\n",
		"src/y.css":        "",
	})
	mods := []resolve.Module{{Dir: "", TS: &resolve.TSConfig{Paths: map[string][]string{"@/*": {"src/*"}}}}}
	g, _ := resolve.GraphWith(files, mods, stamp)
	wantEdges(t, validated(t, g),
		"src/main.tsx -imports-> src/styles/x.css extracted",
		"src/main.tsx -imports-> src/y.css extracted",
		"src/main.tsx#C -references-> src/styles/x.css#.a extracted",
	)
}

// express's entry: index.js passes on lib/express.js, which exports its
// factory by a chained assignment; a test calls the module it requires.
func TestECMAScriptCommonJSEntry(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"index.js":       "module.exports = require(\"./lib/express\");\n",
		"lib/express.js": "exports = module.exports = createApplication;\nfunction createApplication() {}\n",
		"test/app.js":    "var express = require(\"../\");\nfunction run() { express(); }\n",
	}, nil)
	wantCalls(t, g, "test/app.js#run", "lib/express.js#createApplication extracted")
}

// A name an export list renames is exported under its new name alone.
func TestECMAScriptExportListRenames(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"b.ts":   "function g() {}\nexport { g as h };\nfunction v() {}\nexport default v;\nfunction w() {}\nexport { w };\n",
		"f.tsx":  "export function foo() {}\nexport default foo;\n",
		"m.js":   "function c() {}\nmodule.exports = { b: c };\n",
		"use.ts": "import { g, v, w } from \"./b\";\nimport { c } from \"./m.js\";\nimport d, { foo } from \"./f\";\nfunction run() { g(); v(); w(); c(); foo(); d(); }\n",
	}, nil)
	wantCalls(t, g, "use.ts#run", "b.ts#w extracted", "f.tsx#foo extracted", "f.tsx#foo extracted")
}

// A declaration file answers a TypeScript importer alone: Node loads the
// JavaScript beside it.
func TestECMAScriptDeclarationFileHidesNothingFromJavaScript(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"a.js":   "require(\"./b\");\n",
		"a.ts":   "import \"./b\";\n",
		"b.js":   "",
		"b.d.ts": "",
	}, nil)
	wantEdges(t, g, "a.js -imports-> b.js extracted")
}

// An inline TypeScript part imports by what its imports compile to, like a
// .ts file.
func TestECMAScriptInlineTypeScriptPart(t *testing.T) {
	part, err := ecmascript.File("p.html", "import \"./a.js\";\n")
	if err != nil {
		t.Fatal(err)
	}
	part.Language, part.Nodes = "typescript", nil
	html := extract.Result{Path: "p.html", Language: "html", Nodes: []model.Node{fileNode("p.html")}, Embedded: []extract.Result{part}}
	g, _ := esGraph(t, map[string]string{"a.ts": ""}, nil, html)
	wantEdges(t, g, "p.html -imports-> a.ts extracted")
}

// A result the family's extractor would not make -- a method without its
// class, a base on a function -- is skipped, not trusted.
func TestECMAScriptForeignShapes(t *testing.T) {
	m := model.Node{ID: "x.js#C.m", Name: "m", Kind: "method", Owner: "C", Path: "x.js", BodyHash: "h"}
	f := fn("x.js", "f", true)
	k := model.Node{ID: "x.js#K", Name: "K", Kind: "class", Path: "x.js", BodyHash: "h", Exported: true}
	files := []extract.Result{{
		Path: "x.js", Language: "javascript",
		Nodes: []model.Node{fileNode("x.js"), m, f, k},
		Edges: []extract.RawEdge{
			{Source: f.ID, Relation: model.RelationExtends, Name: "K", File: "x.js", Line: 1},
			{Source: m.ID, Relation: model.RelationCalls, Name: "n", Receiver: "this", File: "x.js", Line: 2},
		},
	}}
	g, _ := resolve.GraphWith(files, nil, stamp)
	wantEdges(t, validated(t, g), "x.js#f -extends-> x.js#K extracted")
}

// A bundler's query (vite's ?worker, ?raw, ?url) or a fragment names the
// file without it, as a <script src> does; an alias starting with # keeps
// its #.
func TestECMAScriptImportsDropQueryAndFragment(t *testing.T) {
	mods := []resolve.Module{{TS: &resolve.TSConfig{Paths: map[string][]string{"#lib/*": {"lib/*"}}}}}
	g, _ := esGraph(t, map[string]string{
		"main.ts":  `import W from "./w?worker"; import "./b.js?raw#top"; import "./c%20d"; import "#lib/x?url"; import "node:fs";`,
		"w.ts":     "",
		"b.js":     "",
		"c d.ts":   "",
		"lib/x.ts": "",
	}, mods)
	wantEdges(t, g,
		"main.ts -imports-> b.js extracted",
		"main.ts -imports-> c d.ts extracted",
		"main.ts -imports-> lib/x.ts extracted",
		"main.ts -imports-> w.ts extracted",
	)
}

// A JSX tag calls the component it names, local or imported.
func TestECMAScriptJSXComponents(t *testing.T) {
	g, _ := esGraph(t, map[string]string{
		"btn.tsx": "export function Btn() { return <button />; }\nexport const Card = () => <div />;\n",
		"app.tsx": `import { Btn } from "./btn";
import * as ui from "./btn";
function Local() { return null; }
export function App() { return <Local><Btn label="x" /><ui.Card /></Local>; }
`,
	}, nil)
	wantCalls(t, g, "app.tsx#App", "app.tsx#Local extracted", "btn.tsx#Btn extracted", "btn.tsx#Card extracted")
}
