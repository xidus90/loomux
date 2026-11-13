package stylesheet_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/stylesheet"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not CSS{}.File: the code graph
// follows a package selector, and so sees these tests reach the extractor.
func extractSrc(t *testing.T, rel, src string) extract.Result {
	t.Helper()
	r, err := stylesheet.File(rel, src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// view is a node as most tests compare it: everything but its body.
func view(n model.Node) string {
	return fmt.Sprintf("%s %s %s exported=%v sig=%s", n.ID, n.Kind, n.Span, n.Exported, n.Signature)
}

func views(r extract.Result) []string {
	var out []string
	for _, n := range r.Nodes {
		out = append(out, view(n))
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

func contains(file string, src, dst model.NodeID) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: file}
}

func call(src model.NodeID, name, receiver string, member model.Kind) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, Receiver: receiver, File: "m.scss", Member: member}
}

func extends(src model.NodeID, name string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationExtends, Name: name, File: "m.scss"}
}

func imports(file, spec string) extract.RawEdge {
	return extract.RawEdge{Source: model.NodeID(file), Relation: model.RelationImports, Specifier: spec, File: file}
}

func offer(ns extract.Namespace, name string, target model.NodeID) extract.Export {
	return extract.Export{Namespace: ns, Name: name, Target: target}
}

func same[T any](t *testing.T, what string, got, want []T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %v\nwant %v", what, got, want)
	}
}

func TestLanguagesDescribeThemselves(t *testing.T) {
	for _, c := range []struct {
		l        extract.Language
		name     string
		version  string
		ext      []string
		fileLang string
	}{
		{stylesheet.CSS{}, "css", "css/1@" + treesitter.Parser, []string{".css"}, "a.css"},
		{stylesheet.SCSS{}, "scss", "scss/1@" + treesitter.Parser, []string{".scss"}, "a.scss"},
	} {
		if c.l.Name() != c.name || c.l.Version() != c.version || !reflect.DeepEqual(c.l.Extensions(), c.ext) {
			t.Errorf("%T = %s %s %v, want %s %s %v", c.l, c.l.Name(), c.l.Version(), c.l.Extensions(), c.name, c.version, c.ext)
		}
		r, err := c.l.File(c.fileLang, ".x { }\n")
		if err != nil || r.Language != c.name || r.Path != c.fileLang {
			t.Errorf("%T.File = %s %s, %v; want language %s", c.l, r.Path, r.Language, err, c.name)
		}
	}
}

func TestRulesGetNodes(t *testing.T) {
	r := extractSrc(t, "x.css", `.a
  .b { color: red; }
.a ~ .b { }
.btn { }
.btn { }
@media (max-width: 3px) {
  @supports (display: grid) { .m { } }
}
@layer base { .l { } }
@keyframes k { from { a: b } 50% { a: c } }
`)
	same(t, "nodes", views(r), []string{
		"x.css file L1-L11 exported=true sig=",
		"x.css#.a .b rule L1-L2 exported=true sig=.a\n  .b",
		"x.css#.a ~ .b rule L3-L3 exported=true sig=.a ~ .b",
		"x.css#.btn rule L4-L4 exported=true sig=.btn",
		"x.css#.btn~2 rule L5-L5 exported=true sig=.btn",
		"x.css#.m rule L7-L7 exported=true sig=.m",
		"x.css#.l rule L9-L9 exported=true sig=.l",
	})
	if r.Nodes[1].Name != ".a .b" {
		t.Errorf("name = %q, want the selector with its whitespace folded", r.Nodes[1].Name)
	}
	var want []extract.RawEdge
	for _, n := range r.Nodes[1:] {
		want = append(want, contains("x.css", "x.css", n.ID))
	}
	same(t, "contains", edgesOf(r, model.RelationContains), want)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
}

func TestSCSSDefinitionsGetNodes(t *testing.T) {
	r := extractSrc(t, "m.scss", `.card {
  &:hover { }
  .title { }
  @media (max-width: 3px) { .in { } }
}
%p { }
%-q { }
@mixin -m($a) { .x { } }
@mixin pub { }
@function _f($x) { @return $x; }
@function g() { @return 1; }
`)
	same(t, "nodes", views(r)[1:], []string{
		"m.scss#.card rule L1-L5 exported=true sig=.card",
		"m.scss#.card › &:hover rule L2-L2 exported=true sig=&:hover",
		"m.scss#.card › .title rule L3-L3 exported=true sig=.title",
		"m.scss#.card › .in rule L4-L4 exported=true sig=.in",
		"m.scss#%p rule L6-L6 exported=true sig=%p",
		"m.scss#%-q rule L7-L7 exported=false sig=%-q",
		"m.scss#-m mixin L8-L8 exported=false sig=@mixin -m($a)",
		"m.scss#.x rule L8-L8 exported=true sig=.x",
		"m.scss#pub mixin L9-L9 exported=true sig=@mixin pub",
		"m.scss#_f function L10-L10 exported=false sig=@function _f($x)",
		"m.scss#g function L11-L11 exported=true sig=@function g()",
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.scss", "m.scss", "m.scss#.card"),
		contains("m.scss", "m.scss#.card", "m.scss#.card › &:hover"),
		contains("m.scss", "m.scss#.card", "m.scss#.card › .title"),
		contains("m.scss", "m.scss#.card", "m.scss#.card › .in"),
		contains("m.scss", "m.scss", "m.scss#%p"),
		contains("m.scss", "m.scss", "m.scss#%-q"),
		contains("m.scss", "m.scss", "m.scss#-m"),
		contains("m.scss", "m.scss#-m", "m.scss#.x"),
		contains("m.scss", "m.scss", "m.scss#pub"),
		contains("m.scss", "m.scss", "m.scss#_f"),
		contains("m.scss", "m.scss", "m.scss#g"),
	})
}

func TestOffersClassesAndIDs(t *testing.T) {
	r := extractSrc(t, "x.css", ".a, #b { }\n.a.c > .a:not(.d) { }\nd:hover { }\n")
	same(t, "exports", r.Exports, []extract.Export{
		offer(extract.NamespaceCSSClass, "a", "x.css#.a, #b"),
		offer(extract.NamespaceCSSID, "b", "x.css#.a, #b"),
		offer(extract.NamespaceCSSClass, "c", "x.css#.a.c > .a:not(.d)"),
		offer(extract.NamespaceCSSClass, "a", "x.css#.a.c > .a:not(.d)"),
		offer(extract.NamespaceCSSClass, "d", "x.css#.a.c > .a:not(.d)"),
	})
}

func TestOffersBEMSuffixes(t *testing.T) {
	r := extractSrc(t, "m.scss", `.block, .x .other {
  &__el {
    &--on { }
  }
  &.is { }
  &:hover { }
  &-#{$s} { }
}
.e-#{$k} { }
`)
	same(t, "exports", r.Exports, []extract.Export{
		offer(extract.NamespaceCSSClass, "block", "m.scss#.block, .x .other"),
		offer(extract.NamespaceCSSClass, "x", "m.scss#.block, .x .other"),
		offer(extract.NamespaceCSSClass, "other", "m.scss#.block, .x .other"),
		offer(extract.NamespaceCSSClass, "block__el", "m.scss#.block, .x .other › &__el"),
		offer(extract.NamespaceCSSClass, "other__el", "m.scss#.block, .x .other › &__el"),
		offer(extract.NamespaceCSSClass, "block__el--on", "m.scss#.block, .x .other › &__el › &--on"),
		offer(extract.NamespaceCSSClass, "other__el--on", "m.scss#.block, .x .other › &__el › &--on"),
		offer(extract.NamespaceCSSClass, "is", "m.scss#.block, .x .other › &.is"),
	})
}

func TestImportsInEveryForm(t *testing.T) {
	css := extractSrc(t, "x.css", "@import \"a.css\";\n@import url(\"b.css\");\n@import url(c.css) screen;\n@import 'd.css';\n")
	same(t, "css imports", css.Imports, []extract.Import{{Path: "a.css"}, {Path: "b.css"}, {Path: "c.css"}, {Path: "d.css"}})
	same(t, "css edges", edgesOf(css, model.RelationImports), []extract.RawEdge{
		imports("x.css", "a.css"), imports("x.css", "b.css"), imports("x.css", "c.css"), imports("x.css", "d.css"),
	})

	scss := extractSrc(t, "m.scss", `@use "sass:math";
@use "x" as y;
@use 'z' as *;
@forward "f" show g;
@import "a", "b";
@import url(c.css);
`)
	same(t, "scss imports", scss.Imports, []extract.Import{
		{Path: "sass:math", Kind: "use"},
		{Path: "x", Alias: "y", Kind: "use"},
		{Path: "z", Alias: "*", Kind: "use"},
		{Path: "f", Kind: "forward"},
		{Path: "a", Kind: "import"},
		{Path: "b", Kind: "import"},
		{Path: "c.css", Kind: "import"},
	})
	var want []extract.RawEdge
	for _, s := range []string{"sass:math", "x", "z", "f", "a", "b", "c.css"} {
		want = append(want, imports("m.scss", s))
	}
	same(t, "scss edges", edgesOf(scss, model.RelationImports), want)
}

func TestSCSSRawEdges(t *testing.T) {
	r := extractSrc(t, "m.scss", `@include top;
.a {
  @include m;
  @include n(1);
  @include y.m;
  @include y.p(1);
  @include y.q { color: f(2); }
  width: g(1) + ns.h(rgba(0, 0, 0, .5));
  @extend .k;
}
@mixin mx { @include inner; }
@function fn() { @return z(1); }
`)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.scss", "top", "", "mixin"),
		call("m.scss#.a", "m", "", "mixin"),
		call("m.scss#.a", "n", "", "mixin"),
		call("m.scss#.a", "m", "y", "mixin"),
		call("m.scss#.a", "p", "y", "mixin"),
		call("m.scss#.a", "q", "y", "mixin"),
		call("m.scss#.a", "f", "", "function"),
		call("m.scss#.a", "g", "", "function"),
		call("m.scss#.a", "h", "ns", "function"),
		call("m.scss#.a", "rgba", "", "function"),
		call("m.scss#mx", "inner", "", "mixin"),
		call("m.scss#fn", "z", "", "function"),
	})
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{extends("m.scss#.a", ".k")})
	// The tail of `@include y.q { … }` is no rule of its own.
	same(t, "nodes", views(r)[1:], []string{
		"m.scss#.a rule L2-L10 exported=true sig=.a",
		"m.scss#mx mixin L11-L11 exported=true sig=@mixin mx",
		"m.scss#fn function L12-L12 exported=true sig=@function fn()",
	})
	if len(r.Exports) != 1 {
		t.Errorf("exports = %v, want .a alone", r.Exports)
	}
}

func TestPlainCSSMakesNoCalls(t *testing.T) {
	r := extractSrc(t, "x.css", ".a { color: rgba(0, 0, 0, .5); background: url(x.png); }\n")
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)
}

// The forms the grammar rejects stay local: what the text of an ERROR
// says about @use and @extend counts, and the rules after it get nodes.
func TestRejectedFormsStayLocal(t *testing.T) {
	r := extractSrc(t, "m.scss", `@use "x" as y;
.a {
  @extend %p;
  color: red;
}
@media (width >= 576px) { .r { } }
.after { @include y.m; }
`)
	same(t, "imports", r.Imports, []extract.Import{{Path: "x", Alias: "y", Kind: "use"}})
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{extends("m.scss#.a", "%p")})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.scss#.after", "m", "y", "mixin")})
	same(t, "nodes", views(r)[1:], []string{
		"m.scss#.a rule L2-L5 exported=true sig=.a",
		"m.scss#.r rule L6-L6 exported=true sig=.r",
		"m.scss#.after rule L7-L7 exported=true sig=.after",
	})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the rejected forms counted")
	}

	css := extractSrc(t, "x.css", "@media (width >= 576px) { .r { } }\n.after { }\n")
	same(t, "css nodes", views(css)[1:], []string{
		"x.css#.r rule L1-L1 exported=true sig=.r",
		"x.css#.after rule L2-L2 exported=true sig=.after",
	})
	if css.ParseErrors == 0 {
		t.Error("css ParseErrors = 0, want the range query counted")
	}
}

// A mixin whose header the grammar rejected gets no node, and what its
// body holds belongs to the scope around; an @include of an interpolated
// name calls nothing.
func TestRejectedHeadersNameNothing(t *testing.T) {
	r := extractSrc(t, "m.scss", "@mixin bad($a,, $b) { .x { @include m; } }\n.y { @include #{$v}; }\n")
	same(t, "nodes", views(r)[1:], []string{
		"m.scss#.x rule L1-L1 exported=true sig=.x",
		"m.scss#.y rule L2-L2 exported=true sig=.y",
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.scss#.x", "m", "", "mixin")})
}

// The grammar closes the block around an @extend %p early when a nested
// rule follows; the rules after it keep their node and their parent all
// the same, and the edge still names the placeholder.
func TestExtendOfAPlaceholderKeepsTheNesting(t *testing.T) {
	r := extractSrc(t, "m.scss", ".a {\n  @extend %p;\n  &-sm { }\n  &:hover { }\n  .b { }\n}\n")
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{extends("m.scss#.a", "%p")})
	same(t, "nodes", views(r)[1:], []string{
		"m.scss#.a rule L1-L6 exported=true sig=.a",
		"m.scss#.a › &-sm rule L3-L3 exported=true sig=&-sm",
		"m.scss#.a › &:hover rule L4-L4 exported=true sig=&:hover",
		"m.scss#.a › .b rule L5-L5 exported=true sig=.b",
	})
	same(t, "exports", r.Exports, []extract.Export{
		offer(extract.NamespaceCSSClass, "a", "m.scss#.a"),
		offer(extract.NamespaceCSSClass, "a-sm", "m.scss#.a › &-sm"),
		offer(extract.NamespaceCSSClass, "b", "m.scss#.a › .b"),
	})
}

// A map literal as the argument of a call, and an interpolation inside a
// name in a value, can take the rest of the file into one ERROR; the rules
// around them keep their nodes and their calls. The parameters of a mixin
// are not read as a map.
func TestMapsAndInterpolationsInValuesStayLocal(t *testing.T) {
	r := extractSrc(t, "m.scss", `.t {
  @each $k in $m { $keys: map.merge($keys, ($k: true)); }
}
@each $v in $m {
  .c-#{$v} { --x: color-mix(in oklch, var(--bs-#{$v}-bg), transparent 50%); }
}
@mixin m($a: 1, $b: ("k": 2)) { width: f($a); }
.z { }
`)
	same(t, "nodes", views(r)[1:], []string{
		"m.scss#.t rule L1-L3 exported=true sig=.t",
		"m.scss#.c-#{$v} rule L5-L5 exported=true sig=.c-#{$v}",
		"m.scss#m mixin L7-L7 exported=true sig=@mixin m($a: 1, $b: (\"k\": 2))",
		"m.scss#.z rule L8-L8 exported=true sig=.z",
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.scss#.t", "merge", "map", "function"),
		call("m.scss#.c-#{$v}", "color-mix", "", "function"),
		call("m.scss#.c-#{$v}", "var", "", "function"),
		call("m.scss#m", "f", "", "function"),
	})
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want none", r.ParseErrors)
	}
}

// The mask leaves alone what only looks like a map: parameters after a
// space, m ($a: 1); and it keeps the line breaks of what it masks, so
// that spans stay those of the source.
func TestMaskKeepsParametersAndLines(t *testing.T) {
	r := extractSrc(t, "m.scss", "@mixin m ($a: 1) { }\n@function f ($a: 1) { @return $a; }\n.a-#{$x\n  + 1} { }\n.z { }\n")
	same(t, "nodes", views(r)[1:], []string{
		"m.scss#m mixin L1-L1 exported=true sig=@mixin m ($a: 1)",
		"m.scss#f function L2-L2 exported=true sig=@function f ($a: 1)",
		"m.scss#.a-#{$x + 1} rule L3-L4 exported=true sig=.a-#{$x\n  + 1}",
		"m.scss#.z rule L5-L5 exported=true sig=.z",
	})
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want none", r.ParseErrors)
	}
}

// A selector the parser had to invent, {} in SCSS or #{} in CSS, names no
// rule.
func TestEmptySelectorsNameNothing(t *testing.T) {
	for rel, src := range map[string]string{"m.scss": "{ }\n.z { }\n", "a.css": "#{ }\n.z { }\n"} {
		r := extractSrc(t, rel, src)
		same(t, rel+" nodes", views(r)[1:], []string{rel + "#.z rule L2-L2 exported=true sig=.z"})
	}
	// A nested selector that opens with a combinator still names its rule;
	// the grammar leaves the combinator out of the selectors here.
	r := extractSrc(t, "m.scss", ".a {\n  > .b { }\n  + .c { }\n}\n")
	same(t, "combinator nodes", views(r)[1:], []string{
		"m.scss#.a rule L1-L4 exported=true sig=.a",
		"m.scss#.a › .b rule L2-L2 exported=true sig=.b",
		"m.scss#.a › .c rule L3-L3 exported=true sig=.c",
	})
}

func TestPartHasNoFileNode(t *testing.T) {
	page := "<p>\n<style>\n.btn { }\n@import \"s.css\";\n</style>\n"
	masked := "   \n       \n.btn { }\n@import \"s.css\";\n        \n"
	r, err := stylesheet.Part("p.html", masked)
	if err != nil {
		t.Fatal(err)
	}
	if len(masked) != len(page) {
		t.Fatal("the fixture's mask must keep the page's length")
	}
	if r.Path != "p.html" || r.Language != "css" {
		t.Errorf("Part = %s %s, want p.html css", r.Path, r.Language)
	}
	same(t, "nodes", views(r), []string{"p.html#.btn rule L3-L3 exported=true sig=.btn"})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{contains("p.html", "p.html", "p.html#.btn")})
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{imports("p.html", "s.css")})
	same(t, "exports", r.Exports, []extract.Export{offer(extract.NamespaceCSSClass, "btn", "p.html#.btn")})
}
