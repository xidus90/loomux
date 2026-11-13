package ecmascript_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/ecmascript"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not JavaScript{}.File: the
// code graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, rel, src string) extract.Result {
	t.Helper()
	r, err := ecmascript.File(rel, src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// view is a node as most tests compare it: everything but its body.
func view(n model.Node) string {
	return fmt.Sprintf("%s %s owner=%s %s exported=%v sig=%s", n.ID, n.Kind, n.Owner, n.Span, n.Exported, n.Signature)
}

// views are the views of every node but the file's.
func views(r extract.Result) []string {
	var out []string
	for _, n := range r.Nodes[1:] {
		out = append(out, view(n))
	}
	return out
}

// exported is the id of every node but the file's whose Exported is set.
func exported(r extract.Result) []model.NodeID {
	var out []model.NodeID
	for _, n := range r.Nodes[1:] {
		if n.Exported {
			out = append(out, n.ID)
		}
	}
	return out
}

func edgesOf(r extract.Result, rel model.Relation) []extract.RawEdge {
	var out []extract.RawEdge
	for _, e := range r.Edges {
		if e.Relation == rel {
			e.Line = 0 // lines are asserted in TestEdgeLines
			out = append(out, e)
		}
	}
	return out
}

func call(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, Receiver: receiver, File: "m.ts"}
}

func construct(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "new"
	return e
}

func base(src model.NodeID, rel model.Relation, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: rel, Name: name, Receiver: receiver, File: "m.ts"}
}

// same compares two lists and treats nil and empty as equal.
func same[T any](t *testing.T, what string, got, want []T) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s =\n %+v\nwant\n %+v", what, got, want)
	}
}

func TestLanguagesDescribeThemselves(t *testing.T) {
	for _, c := range []struct {
		lang    extract.Language
		name    string
		exts    []string
		rel     string
		version string
	}{
		{ecmascript.JavaScript{}, "javascript", []string{".js", ".mjs", ".cjs", ".jsx"}, "src/a.js", "javascript/1@"},
		{ecmascript.TypeScript{}, "typescript", []string{".ts", ".mts", ".cts", ".tsx"}, "src/a.ts", "typescript/1@"},
	} {
		if c.lang.Name() != c.name {
			t.Errorf("Name = %q, want %q", c.lang.Name(), c.name)
		}
		if want := c.version + treesitter.Parser; c.lang.Version() != want {
			t.Errorf("Version = %q, want %q", c.lang.Version(), want)
		}
		same(t, c.name+" Extensions", c.lang.Extensions(), c.exts)
		r, err := c.lang.File(c.rel, "function f() {}\n")
		if err != nil || r.Path != c.rel || r.Language != c.name || len(r.Nodes) != 2 {
			t.Errorf("%s File = %d nodes at %q (%q), %v; want the file and f", c.name, len(r.Nodes), r.Path, r.Language, err)
		}
	}
}

func TestEachExtensionReadsItsLanguageAndGrammar(t *testing.T) {
	// A JSX attribute class= breaks the JavaScript grammar and not the tsx
	// one, and a type annotation breaks the JavaScript grammar alone: the
	// error count tells the grammars apart.
	const jsx = "function C() { return <p class=\"a\">t</p>; }\n"
	const typed = "function f(a: number): string { return ''; }\n"
	for rel, want := range map[string]struct {
		lang      string
		jsxErrors bool
		tsErrors  bool
	}{
		"a.js": {"javascript", true, true}, "a.mjs": {"javascript", true, true}, "a.cjs": {"javascript", true, true},
		"a.jsx": {"javascript", false, false},
		"a.ts":  {"typescript", true, false}, "a.mts": {"typescript", true, false}, "a.cts": {"typescript", true, false},
		"a.tsx": {"typescript", false, false},
	} {
		r := extractSrc(t, rel, jsx)
		if r.Language != want.lang || (r.ParseErrors > 0) != want.jsxErrors {
			t.Errorf("%s: language %q, %d errors on class=; want %q, errors %v", rel, r.Language, r.ParseErrors, want.lang, want.jsxErrors)
		}
		if r := extractSrc(t, rel, typed); (r.ParseErrors > 0) != want.tsErrors {
			t.Errorf("%s: %d errors on a type annotation, want errors %v", rel, r.ParseErrors, want.tsErrors)
		}
	}
}

func TestDeclarationFileKeepsTheFileNodeAlone(t *testing.T) {
	const src = "import { a } from './a';\nexport declare function f(): void;\nexport class C { m() { g(); } }\n"
	for _, rel := range []string{"types/a.d.ts", "types/a.d.mts", "types/a.d.cts"} {
		r := extractSrc(t, rel, src)
		if r.Language != "typescript" || len(r.Nodes) != 1 || r.Nodes[0].ID != model.NodeID(rel) || r.Nodes[0].Kind != model.KindFile {
			t.Errorf("%s: nodes = %v (%s), want the file node alone", rel, views(r), r.Language)
		}
		if len(r.Edges) != 0 || len(r.Imports) != 0 || r.ParseErrors != 0 {
			t.Errorf("%s: edges %v, imports %v, %d errors; want none", rel, r.Edges, r.Imports, r.ParseErrors)
		}
		if r.Nodes[0].BodyText != extract.Collapse(src) {
			t.Errorf("%s: BodyText = %q, want the whole file", rel, r.Nodes[0].BodyText)
		}
	}
	// TypeScript knows no .d.tsx: it is an ordinary source.
	if r := extractSrc(t, "a.d.tsx", src); len(r.Nodes) != 3 {
		t.Errorf("a.d.tsx: nodes = %v, want C and C.m read", views(r))
	}
}

func TestDeclarationsGetNodes(t *testing.T) {
	const src = `function f(a) { return a; }
function* gen() {}
export async function af() {}
class C extends B {
  constructor() { super(); }
  m() {}
  static s() {}
  get p() { return 1; }
  #priv() {}
  private hidden() {}
  protected prot() {}
  over(): void;
  over(a?: number): void {}
}
export abstract class A { abstract am(): void; }
export interface I extends J { x: number }
export type T = { a: string };
export enum E { A, B }
const enum CE { A }
export const ar = (x) => x, fe = function () {};
let lv = async () => 1;
var vv = function* () {};
const notFn = 1;
export default function () {}
namespace N.Inner { export function nf() {} class NC { m() {} } }
module M { function mf() {} }
declare module "ext" { function hidden(): void; }
declare function df(): void;
function outer() { function inner() {} const local = () => 1; }
const o = { meth() {} };
`
	r := extractSrc(t, "m.ts", src)
	same(t, "nodes", views(r), []string{
		"m.ts#f function owner= L1-L1 exported=false sig=function f(a)",
		"m.ts#gen function owner= L2-L2 exported=false sig=function* gen()",
		"m.ts#af function owner= L3-L3 exported=true sig=async function af()",
		"m.ts#C class owner= L4-L14 exported=false sig=class C extends B",
		"m.ts#C.constructor method owner=C L5-L5 exported=false sig=constructor()",
		"m.ts#C.m method owner=C L6-L6 exported=false sig=m()",
		"m.ts#C.s method owner=C L7-L7 exported=false sig=static s()",
		"m.ts#C.p method owner=C L8-L8 exported=false sig=get p()",
		"m.ts#C.#priv method owner=C L9-L9 exported=false sig=#priv()",
		"m.ts#C.hidden method owner=C L10-L10 exported=false sig=private hidden()",
		"m.ts#C.prot method owner=C L11-L11 exported=false sig=protected prot()",
		"m.ts#C.over method owner=C L13-L13 exported=false sig=over(a?: number): void",
		"m.ts#A class owner= L15-L15 exported=true sig=abstract class A",
		"m.ts#A.am method owner=A L15-L15 exported=true sig=abstract am(): void",
		"m.ts#I interface owner= L16-L16 exported=true sig=interface I extends J",
		"m.ts#T type owner= L17-L17 exported=true sig=type T",
		"m.ts#E class owner= L18-L18 exported=true sig=enum E",
		"m.ts#CE class owner= L19-L19 exported=false sig=const enum CE",
		"m.ts#ar function owner= L20-L20 exported=true sig=ar = (x) =>",
		"m.ts#fe function owner= L20-L20 exported=true sig=fe = function ()",
		"m.ts#lv function owner= L21-L21 exported=false sig=lv = async () =>",
		"m.ts#vv function owner= L22-L22 exported=false sig=vv = function* ()",
		"m.ts#default function owner= L24-L24 exported=true sig=function ()",
		"m.ts#N.Inner.nf function owner=N.Inner L25-L25 exported=false sig=function nf()",
		"m.ts#N.Inner.NC class owner=N.Inner L25-L25 exported=false sig=class NC",
		"m.ts#N.Inner.NC.m method owner=N.Inner.NC L25-L25 exported=false sig=m()",
		"m.ts#M.mf function owner=M L26-L26 exported=false sig=function mf()",
		"m.ts#outer function owner= L29-L29 exported=false sig=function outer()",
	})
	// The file contains every module-level node, a class or namespace its own.
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#f", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#gen", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#af", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#C", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.constructor", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.m", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.s", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.p", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.#priv", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.hidden", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.prot", File: "m.ts"},
		{Source: "m.ts#C", Relation: model.RelationContains, TargetID: "m.ts#C.over", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#A", File: "m.ts"},
		{Source: "m.ts#A", Relation: model.RelationContains, TargetID: "m.ts#A.am", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#I", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#T", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#E", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#CE", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#ar", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#fe", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#lv", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#vv", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#default", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#N.Inner.nf", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#N.Inner.NC", File: "m.ts"},
		{Source: "m.ts#N.Inner.NC", Relation: model.RelationContains, TargetID: "m.ts#N.Inner.NC.m", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#M.mf", File: "m.ts"},
		{Source: "m.ts", Relation: model.RelationContains, TargetID: "m.ts#outer", File: "m.ts"},
	})
}

func TestDefaultExportsAndJavaScriptClasses(t *testing.T) {
	r := extractSrc(t, "m.js", "export default class {}\n")
	same(t, "anonymous class", views(r), []string{"m.js#default class owner= L1-L1 exported=true sig=class"})

	r = extractSrc(t, "m.js", "export default async function named() {}\nexport default class K extends B { m() {} }\n")
	same(t, "named defaults", views(r), []string{
		"m.js#named function owner= L1-L1 exported=true sig=async function named()",
		"m.js#K class owner= L2-L2 exported=true sig=class K extends B",
		"m.js#K.m method owner=K L2-L2 exported=true sig=m()",
	})
	// A named default is also bound under default, as an export list would.
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "", Name: "named", Alias: "default", Kind: "export"},
		{Path: "", Name: "K", Alias: "default", Kind: "export"},
	})
}

func TestExportedFollowsEveryExportForm(t *testing.T) {
	const src = `function a() {}
function b() {}
class K { m() {} #p() {} }
function d() {}
function e() {}
function f() {}
function g() {}
function h() {}
function local() {}
export { a, b as bee };
export { K };
export default d;
module.exports = { e, alias: f };
exports.g = g;
module.exports.hh = h;
export { a as a };
exports.made = function () {};
module.exports.arrow = () => {};
`
	r := extractSrc(t, "m.js", src)
	same(t, "exported", exported(r), []model.NodeID{
		"m.js#a", "m.js#b", "m.js#K", "m.js#K.m", "m.js#d", "m.js#e", "m.js#f", "m.js#g", "m.js#h",
		"m.js#made", "m.js#arrow",
	})
	same(t, "export imports", r.Imports, []extract.Import{
		{Name: "a", Kind: "export"},
		{Name: "b", Alias: "bee", Kind: "export"},
		{Name: "K", Kind: "export"},
		{Name: "d", Alias: "default", Kind: "export"},
		{Name: "e", Kind: "export"},
		{Name: "f", Alias: "alias", Kind: "export"},
		{Name: "g", Kind: "export"},
		{Name: "h", Alias: "hh", Kind: "export"},
		{Name: "a", Kind: "export"},
	})
	// An export list names no file: no imports edge.
	same(t, "imports edges", edgesOf(r, model.RelationImports), nil)

	r = extractSrc(t, "m.js", "module.exports = function () { f(); };\nmodule.exports = run;\n")
	same(t, "module.exports function", views(r), []string{"m.js#default function owner= L1-L1 exported=true sig=module.exports = function ()"})
	same(t, "module.exports name", r.Imports, []extract.Import{{Name: "run", Alias: "default", Kind: "export"}})
	same(t, "its call", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		{Source: "m.js#default", Relation: model.RelationCalls, Name: "f", File: "m.js"},
	})

	// A chained assignment exports its value through whichever side names
	// the exports.
	for _, src := range []string{
		"function run() {}\nexports = module.exports = run;\n",
		"function run() {}\nmodule.exports = exports = run;\n",
	} {
		r = extractSrc(t, "m.js", src)
		same(t, "chained "+src, r.Imports, []extract.Import{{Name: "run", Alias: "default", Kind: "export"}})
	}
	r = extractSrc(t, "m.js", "exports = module.exports = function () {};\n")
	same(t, "chained function", exported(r), []model.NodeID{"m.js#default"})

	// module.exports = require("./x") passes on the whole module, its
	// default too; exports.f = require("./x") passes it on as f.
	r = extractSrc(t, "m.js", "module.exports = require(\"./x\");\nexports.f = require(\"./y\");\n")
	same(t, "re-exported module", r.Imports, []extract.Import{
		{Path: "./x", Name: "*", Kind: "reexport"},
		{Path: "./x", Name: "default", Kind: "reexport"},
		{Path: "./y", Name: "*", Alias: "f", Kind: "reexport"},
	})
	if n := len(edgesOf(r, model.RelationImports)); n != 3 {
		t.Errorf("imports edges = %d, want one per import", n)
	}
	// A chain exports its value under every target it names; a function
	// gets its node under the first.
	r = extractSrc(t, "m.js", "function fn() {}\nmodule.exports.f = exports.g = fn;\n")
	same(t, "chained targets", r.Imports, []extract.Import{
		{Name: "fn", Alias: "f", Kind: "export"}, {Name: "fn", Alias: "g", Kind: "export"},
	})
	r = extractSrc(t, "m.js", "exports.f = exports.g = function () {};\n")
	same(t, "chained function node", exported(r), []model.NodeID{"m.js#f"})
	same(t, "chained function", r.Imports, []extract.Import{{Name: "f", Alias: "g", Kind: "export"}})
	r = extractSrc(t, "m.js", "exports.f = module.exports = require(\"./x\");\n")
	same(t, "chained require", r.Imports, []extract.Import{
		{Path: "./x", Name: "*", Alias: "f", Kind: "reexport"},
		{Path: "./x", Name: "*", Kind: "reexport"},
		{Path: "./x", Name: "default", Kind: "reexport"},
	})
	r = extractSrc(t, "m.js", "exports.x = module.exports = { a };\nexports.y = { b };\n")
	same(t, "chained object", r.Imports, []extract.Import{{Name: "a", Kind: "export"}})

	// A declaration exported by its own keyword and renamed by a list keeps
	// its own name: the list alone would only say the new one.
	r = extractSrc(t, "m.ts", "export function foo() {}\nexport default foo;\nexport function bar() {}\nexport { bar };\nexport default function main() {}\n")
	same(t, "own name kept", r.Imports, []extract.Import{
		{Name: "foo", Alias: "default", Kind: "export"}, {Name: "bar", Kind: "export"},
		{Name: "main", Alias: "default", Kind: "export"}, {Name: "foo", Kind: "export"},
	})

	// A chain naming no exports, or a require of no literal, exports nothing.
	r = extractSrc(t, "m.js", "a = b = 2;\nmodule.exports = require(name);\n")
	if len(r.Imports) != 0 {
		t.Errorf("plain chains made %v", r.Imports)
	}

	// export = in TypeScript is CommonJS's module.exports.
	r = extractSrc(t, "m.ts", "function run() {}\nexport = run;\n")
	same(t, "export =", r.Imports, []extract.Import{{Name: "run", Alias: "default", Kind: "export"}})
	same(t, "export = exported", exported(r), []model.NodeID{"m.ts#run"})

	// Other assignments make nothing; what they call belongs to the file.
	r = extractSrc(t, "m.js", "x = function () {};\nexports.n = 1;\nmodule.exports = 1;\na.b.c = () => 1;\nmodule.exports = { f: function () { g(); } };\nexport default { a: h() };\n")
	if len(r.Nodes) != 1 || len(r.Imports) != 0 {
		t.Errorf("plain assignments made %v, %v", views(r), r.Imports)
	}
	same(t, "calls of plain values", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		{Source: "m.js", Relation: model.RelationCalls, Name: "g", File: "m.js"},
		{Source: "m.js", Relation: model.RelationCalls, Name: "h", File: "m.js"},
	})
}

func TestImportsInEveryForm(t *testing.T) {
	const src = `import d, { a, b as c, default as dd, "x-y" as xy } from "./x";
import * as ns from "./y";
import "./s.css";
import type { T } from "./t";
import j from "./j.json" with { type: "json" };
import eq = require("./eq");
export { e as f } from "./z";
export * from "./w";
export * as nn from "./v";
const m = require("./m");
const { k, l: ll } = require("./k");
require("./side");
function lazy() { return import("./dyn"); }
import(name);
require(name, other);
`
	r := extractSrc(t, "m.ts", src)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "./x", Name: "default", Alias: "d"},
		{Path: "./x", Name: "a"},
		{Path: "./x", Name: "b", Alias: "c"},
		{Path: "./x", Name: "default", Alias: "dd"},
		{Path: "./x", Name: "x-y", Alias: "xy"},
		{Path: "./y", Name: "*", Alias: "ns"},
		{Path: "./s.css"},
		{Path: "./t", Name: "T"},
		{Path: "./j.json", Name: "default", Alias: "j"},
		{Path: "./eq", Name: "*", Alias: "eq", Kind: "require"},
		{Path: "./z", Name: "e", Alias: "f", Kind: "reexport"},
		{Path: "./w", Name: "*", Kind: "reexport"},
		{Path: "./v", Name: "*", Alias: "nn", Kind: "reexport"},
		{Path: "./m", Name: "*", Alias: "m", Kind: "require"},
		{Path: "./k", Name: "k", Kind: "require"},
		{Path: "./k", Name: "l", Alias: "ll", Kind: "require"},
		{Path: "./side", Kind: "require"},
		{Path: "./dyn", Kind: "dynamic"},
	})
	var specs []string
	for _, e := range edgesOf(r, model.RelationImports) {
		if e.Source != "m.ts" || e.File != "m.ts" {
			t.Errorf("imports edge %+v, want one from the file", e)
		}
		specs = append(specs, e.Specifier)
	}
	same(t, "imports edges", specs, []string{
		"./x", "./x", "./x", "./x", "./x", "./y", "./s.css", "./t", "./j.json", "./eq", "./z", "./w", "./v",
		"./m", "./k", "./k", "./side", "./dyn",
	})
	// require and import() are no calls.
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.ts", "require", ""),
	})
}

func TestCallsInEveryForm(t *testing.T) {
	const src = `function top() {
  f();
  ns.g();
  this.m();
  super.n();
  new X();
  new ns.Y();
  f?.();
  a.b.c();
  (() => h())();
  function inner() { i(); }
  new (k())();
  new a.b.C();
}
class C extends Base {
  field = init();
  m() { j(); }
}
run();
const ar = () => z();
`
	r := extractSrc(t, "m.ts", src)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.ts#top", "f", ""),
		call("m.ts#top", "g", "ns"),
		call("m.ts#top", "m", "this"),
		call("m.ts#top", "n", "super"),
		construct("m.ts#top", "X", ""),
		construct("m.ts#top", "Y", "ns"),
		call("m.ts#top", "f", ""),
		call("m.ts#top", "h", ""),
		call("m.ts#top", "i", ""),
		call("m.ts#top", "k", ""),
		call("m.ts", "init", ""),
		call("m.ts#C.m", "j", ""),
		call("m.ts", "run", ""),
		call("m.ts#ar", "z", ""),
	})
}

func TestBasesInEveryForm(t *testing.T) {
	const src = `class A extends B implements I<T>, ns.J {}
class G extends Base<T> {}
class N extends ns.Base {}
class Mix extends mixin(D) {}
interface IX extends P<T>, ns.Q, R {}
`
	r := extractSrc(t, "m.ts", src)
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		base("m.ts#A", model.RelationExtends, "B", ""),
		base("m.ts#G", model.RelationExtends, "Base", ""),
		base("m.ts#N", model.RelationExtends, "Base", "ns"),
		base("m.ts#IX", model.RelationExtends, "P", ""),
		base("m.ts#IX", model.RelationExtends, "Q", "ns"),
		base("m.ts#IX", model.RelationExtends, "R", ""),
	})
	same(t, "implements", edgesOf(r, model.RelationImplements), []extract.RawEdge{
		base("m.ts#A", model.RelationImplements, "I", ""),
		base("m.ts#A", model.RelationImplements, "J", "ns"),
	})
	// The JavaScript grammar puts the base straight into the heritage.
	r = extractSrc(t, "m.js", "class A extends B {}\nclass C extends ns.D {}\n")
	same(t, "js extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		{Source: "m.js#A", Relation: model.RelationExtends, Name: "B", File: "m.js"},
		{Source: "m.js#C", Relation: model.RelationExtends, Name: "D", Receiver: "ns", File: "m.js"},
	})
}

func TestWebHandovers(t *testing.T) {
	const src = `import "./a.css";
import "../b.scss";
import "pkg/c.css";
import "./d.js";
function wire(el) {
  document.getElementById("x");
  document.querySelector("#y");
  document.querySelectorAll(".a");
  el.querySelectorAll("#z, .b");
  el.querySelector("div .no");
  el.querySelector("#ok, div");
  el.getElementsByClassName("c d");
  el.classList.add("e", "f");
  el.classList.remove("g");
  el.classList.toggle("h", true);
  el.classList.contains("i");
  el.classList.add(name);
  set.add("not-a-class");
  document.getElementById(name);
}
`
	r := extractSrc(t, "src/m.js", src)
	ref := func(member, name string) extract.RawEdge {
		return extract.RawEdge{Source: "src/m.js#wire", Relation: model.RelationReferences, Name: name, File: "src/m.js", Member: model.Kind(member)}
	}
	same(t, "references", edgesOf(r, model.RelationReferences), []extract.RawEdge{
		ref("html.id", "x"),
		ref("html.id", "y"), ref("css.id", "y"),
		ref("html.id", "z"), ref("css.id", "z"),
	})
	var classes []string
	for _, b := range r.Bridges {
		if b.Source != "src/m.js#wire" || b.Relation != model.RelationReferences || b.Namespace != extract.NamespaceCSSClass || b.File != "src/m.js" || b.Line == 0 {
			t.Errorf("bridge %+v, want a css.class reference from wire with a line", b)
		}
		same(t, "scope of "+b.Name, b.Scope, []string{"src/a.css", "b.scss"})
		classes = append(classes, b.Name)
	}
	same(t, "classes", classes, []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"})
}

func TestJSXHandoversAndOffers(t *testing.T) {
	const src = `function App() {
  return <div id="root" className="a  b"><span class="c" id={dyn} className={x} /></div>;
}
const top = <p id="top" />;
`
	r := extractSrc(t, "ui/App.jsx", src)
	same(t, "exports", r.Exports, []extract.Export{
		{Namespace: extract.NamespaceHTMLID, Name: "root", Target: "ui/App.jsx#App"},
		{Namespace: extract.NamespaceHTMLID, Name: "top", Target: "ui/App.jsx"},
	})
	var classes []string
	for _, b := range r.Bridges {
		if b.Source != "ui/App.jsx#App" || b.Scope != nil || b.Line != 2 {
			t.Errorf("bridge %+v, want one from App on line 2 without a scope", b)
		}
		classes = append(classes, b.Name)
	}
	same(t, "classes", classes, []string{"a", "b", "c"})
}

func TestBrokenDefinitionsKeepTheRest(t *testing.T) {
	// The JavaScript grammar cannot read the JSX attribute class= and
	// ends its tree there; the definitions after it still arrive.
	const src = `function before() { a(); }
function K() {
  return <p class="z">t</p>;
}
function after() { q(); }
class Later { m() { r(); } }
export const z = () => s();
`
	r := extractSrc(t, "m.js", src)
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the broken JSX counted")
	}
	var ids []model.NodeID
	for _, n := range r.Nodes[1:] {
		ids = append(ids, n.ID)
	}
	same(t, "nodes", ids, []model.NodeID{"m.js#before", "m.js#after", "m.js#Later", "m.js#Later.m", "m.js#z"})
	var calls []string
	for _, e := range edgesOf(r, model.RelationCalls) {
		calls = append(calls, string(e.Source)+">"+e.Name)
	}
	same(t, "calls", calls, []string{"m.js#before>a", "m.js#after>q", "m.js#Later.m>r", "m.js#z>s"})

	// The same file as .jsx reads with the tsx grammar, cleanly.
	r = extractSrc(t, "m.jsx", src)
	if r.ParseErrors != 0 || len(r.Nodes) != 7 {
		t.Errorf(".jsx: %d errors, nodes %v; want none and every definition", r.ParseErrors, views(r))
	}

	// A definition whose header the parser rebuilt gets no node, nor does
	// anything inside it; an ERROR in a body loses what it holds alone.
	r = extractSrc(t, "m.ts", `function bad(a b) { x(); }
function good() {}
class A extends B C { m() {} }
class D { m(a b) { y(); } n() {} }
function f() { a(); @@@; b(); }
`)
	same(t, "nodes", views(r), []string{
		"m.ts#good function owner= L2-L2 exported=false sig=function good()",
		"m.ts#D class owner= L4-L4 exported=false sig=class D",
		"m.ts#D.n method owner=D L4-L4 exported=false sig=n()",
		"m.ts#f function owner= L5-L5 exported=false sig=function f()",
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.ts#f", "a", ""), call("m.ts#f", "b", "")})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the broken headers counted")
	}
}

func TestNamespaceMembersAreExportedThroughEveryNamespace(t *testing.T) {
	const src = `namespace N { export function a() {} }
export namespace E {
  export function g() {}
  function h() {}
  export namespace In { export function i() {} }
  namespace Hid { export function j() {} }
  export class C { m() {} }
}
`
	r := extractSrc(t, "m.ts", src)
	same(t, "exported", exported(r), []model.NodeID{"m.ts#E.g", "m.ts#E.In.i", "m.ts#E.C", "m.ts#E.C.m"})
}

func TestNamespaceForms(t *testing.T) {
	const src = `export namespace E { export function g() { h(); } }
module "ambient" { function hidden() {} }
`
	r := extractSrc(t, "m.ts", src)
	same(t, "nodes", views(r), []string{"m.ts#E.g function owner=E L1-L1 exported=true sig=function g()"})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.ts#E.g", "h", "")})
}

// ids are the ids of every node but the file's.
func ids(r extract.Result) []model.NodeID {
	var out []model.NodeID
	for _, n := range r.Nodes[1:] {
		out = append(out, n.ID)
	}
	return out
}

func TestABreakSwallowingTheRestKeepsTheDefinitionsAfterIt(t *testing.T) {
	// The / of </p> opens a regex that a later / closes: the tree reaches
	// the end of the file, one ERROR spans the definitions in between, and
	// each of them is read again on its own.
	const comment = `function K() {
  return <p class="z">t</p>;
}
function after() { q(); }
class Later { m() { r(); } }
// a note
function last() { return a / b; }
`
	r := extractSrc(t, "m.js", comment)
	same(t, "after a comment", ids(r), []model.NodeID{"m.js#after", "m.js#Later", "m.js#Later.m", "m.js#last"})
	var calls []string
	for _, e := range edgesOf(r, model.RelationCalls) {
		calls = append(calls, string(e.Source)+">"+e.Name)
	}
	same(t, "calls", calls, []string{"m.js#after>q", "m.js#Later.m>r"})

	const alternating = `function B1() {
  return <p class="a">t</p>;
}
function G1() { work(); }
function B2() {
  return <p class="a">t</p>;
}
function G2() { work(); }
function B3() {
  return <p class="a">t</p>;
}
function G3() { work(); }
`
	r = extractSrc(t, "m.js", alternating)
	same(t, "alternating", ids(r), []model.NodeID{"m.js#G1", "m.js#G2", "m.js#G3"})
}

func TestARestartNeverStartsInATemplateLiteral(t *testing.T) {
	// The line in column 0 inside the template is text; read on its own it
	// would make a function.
	const src = "function K() {\n  return <p class=\"z\">t</p>;\n}\nconst tpl = `\nfunction ghost() {}\n${ { a: `\nfunction nested() {}\n` } }\n`;\nfunction after() { q(); }\n// x / y\n"
	r := extractSrc(t, "m.js", src)
	same(t, "nodes", ids(r), []model.NodeID{"m.js#after"})
}

func TestGrammarProbeParsesCleanly(t *testing.T) {
	const src = `import j from "./j.json" with { type: "json" };
const re = /[\p{L}--[a-z]]/v;
using res = getRes();
namespace N { export const x = 1; }
enum E { A, B }
abstract class A { abstract m(): void; }
class P { #x = 1; #m() { return this.#x; } }
`
	if r := extractSrc(t, "m.ts", src); r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
}

// TypeScript's export = of a function or class is the default, as
// module.exports = function is; what any other value calls belongs to the
// file.
func TestExportAssignmentOfAFunctionOrClass(t *testing.T) {
	r := extractSrc(t, "m.ts", "export = function z() { f(); };\n")
	same(t, "function nodes", exported(r), []model.NodeID{"m.ts#default"})
	same(t, "function calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.ts#default", "f", "")})
	r = extractSrc(t, "m.ts", "export = class { m() { g(); } };\n")
	same(t, "class nodes", ids(r), []model.NodeID{"m.ts#default", "m.ts#default.m"})
	same(t, "class calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.ts#default.m", "g", "")})
	r = extractSrc(t, "m.ts", "export = make(h);\n")
	same(t, "value calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.ts", "make", "")})
}

func TestJSXComponentUseIsACall(t *testing.T) {
	// A capitalised tag is a component, rendered as a call; a lowercase
	// one is a DOM element, a chain names no receiver.
	const src = `function App() {
  return <Layout><Btn label="x" /><ui.Card /><div /><a.b.C /><svg:path /></Layout>;
}
`
	for _, rel := range []string{"m.js", "m.jsx", "m.tsx"} {
		r := extractSrc(t, rel, src)
		c := func(name, receiver string) extract.RawEdge {
			return extract.RawEdge{Source: model.NodeID(rel + "#App"), Relation: model.RelationCalls, Name: name, Receiver: receiver, File: rel}
		}
		same(t, rel+" calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{c("Layout", ""), c("Btn", ""), c("Card", "ui")})
	}
}

func TestPartHasNoFileNode(t *testing.T) {
	// The page with everything but its script blanked: the mask keeps the
	// offsets, so spans are lines of the page.
	masked := "      \n        \nfunction f() { g(); }\n;        \n"
	r, err := ecmascript.Part("p.html", masked, false)
	if err != nil {
		t.Fatal(err)
	}
	if r.Path != "p.html" || r.Language != "javascript" {
		t.Errorf("Part = %s %s, want p.html javascript", r.Path, r.Language)
	}
	var got []string
	for _, n := range r.Nodes {
		got = append(got, view(n))
	}
	same(t, "nodes", got, []string{"p.html#f function owner= L3-L3 exported=false sig=function f()"})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		{Source: "p.html", Relation: model.RelationContains, TargetID: "p.html#f", File: "p.html"},
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		{Source: "p.html#f", Relation: model.RelationCalls, Name: "g", File: "p.html"},
	})
}

func TestPartReadsTypeScriptWithItsGrammar(t *testing.T) {
	masked := "      \nfunction f(a: number): void { }\nenum E { A }\n"
	r, err := ecmascript.Part("p.html", masked, true)
	if err != nil {
		t.Fatal(err)
	}
	if r.Language != "typescript" || r.ParseErrors != 0 || len(r.Nodes) != 2 || r.Nodes[0].ID != "p.html#f" || r.Nodes[1].ID != "p.html#E" {
		t.Errorf("Part(ts) = %s, %d errors, %v; want typescript, no error, f and E", r.Language, r.ParseErrors, r.Nodes)
	}
	// The JavaScript grammar breaks on the annotation.
	if js, _ := ecmascript.Part("p.html", masked, false); js.ParseErrors == 0 {
		t.Error("Part(js) read TypeScript without an error; the test proves nothing")
	}
}
