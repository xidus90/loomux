package rust_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

func TestUseForms(t *testing.T) {
	src := `use a::b;
use a::b as c;
use a::{self, C as D, e::{f, g}};
use a::*;
pub use x::y;
pub(crate) use x::z;
use crate::p::q;
use self::m;
use super::*;
use ::std::fmt;
use {u, v};
use ::{w};
use a::{*};
use a::b as _;
use Vec::<i32>::new;
use <T as Tr>::f;
extern crate x as y;
extern crate w;
pub extern crate k;
`
	r := extractSrc(t, src)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "a::b", Name: "b"},
		{Path: "a::b", Name: "b", Alias: "c"},
		{Path: "a", Name: "a"},
		{Path: "a::C", Name: "C", Alias: "D"},
		{Path: "a::e::f", Name: "f"},
		{Path: "a::e::g", Name: "g"},
		{Path: "a", Name: "*"},
		{Path: "x::y", Name: "y", Reexport: true},
		{Path: "x::z", Name: "z", Reexport: true},
		{Path: "crate::p::q", Name: "q"},
		{Path: "self::m", Name: "m"},
		{Path: "super", Name: "*"},
		{Path: "::std::fmt", Name: "fmt"},
		{Path: "u", Name: "u"},
		{Path: "v", Name: "v"},
		{Path: "w", Name: "w"},
		{Path: "a", Name: "*"},
		{Path: "a::b", Name: "b", Alias: "_"},
		{Path: "Vec::new", Name: "new"},
		{Path: "x", Alias: "y"},
		{Path: "w"},
		{Path: "k", Reexport: true},
	})
	if got := edgesOf(r, model.RelationImports); len(got) != 0 {
		t.Errorf("a use makes no imports edge, got %v", got)
	}
}

func TestUseInsideBodies(t *testing.T) {
	// A use in a function or an inline module binds as one of the file.
	r := extractSrc(t, "fn f() { use a::b; }\nmod m { use c::d; }\n")
	same(t, "imports", r.Imports, []extract.Import{{Path: "a::b", Name: "b"}, {Path: "c::d", Name: "d"}})
}

func TestUseInInlineModules(t *testing.T) {
	// The resolver binds per file, so self:: and super:: are read from the
	// file's own module.
	src := `mod tests { use super::*; use super::x; use self::y; }
mod a {
    fn f() { use super::q; }
    mod b {
        use super::x;
        use self::y;
        use super::super::z;
        use super::super::super::w;
        use super::super::super::super::v;
        use super::{self};
        use super::{k, m::n};
        use crate::c;
        use ::d;
        use e::f;
    }
}
`
	r := extractSrc(t, src)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "self", Name: "*"},
		{Path: "self::x", Name: "x"},
		{Path: "self::tests::y", Name: "y"},
		{Path: "self::q", Name: "q"},
		{Path: "self::a::x", Name: "x"},
		{Path: "self::a::b::y", Name: "y"},
		{Path: "self::z", Name: "z"},
		{Path: "super::w", Name: "w"},
		{Path: "super::super::v", Name: "v"},
		{Path: "self::a", Name: "a"},
		{Path: "self::a::k", Name: "k"},
		{Path: "self::a::m::n", Name: "n"},
		{Path: "crate::c", Name: "c"},
		{Path: "::d", Name: "d"},
		{Path: "e::f", Name: "f"},
	})
}

func TestOutOfLineModule(t *testing.T) {
	src := `mod x;
#[path = "y.rs"]
mod x2;
pub mod inline { mod z; }
mod body { fn f() {} }
mod a { mod b { mod w; } }
`
	r := extractSrc(t, src)
	edge := func(in, name, spec string) extract.RawEdge {
		return extract.RawEdge{Source: "m.rs", Relation: model.RelationImports, Receiver: in, Name: name, Specifier: spec, File: "m.rs"}
	}
	// Receiver is the inline module the declaration sits in.
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{edge("", "x", ""), edge("", "x2", "y.rs"), edge("inline", "z", ""), edge("a::b", "w", "")})
	if len(r.Imports) != 0 {
		t.Errorf("mod x; is no import: %v", r.Imports)
	}
}

func TestBrokenModuleHeaderMakesNoEdge(t *testing.T) {
	r := extractSrc(t, "mod x y;\nfn f() {}\n")
	same(t, "imports", edgesOf(r, model.RelationImports), nil)
}

type callOpt func(*extract.RawEdge)

func withRecv(r string) callOpt       { return func(e *extract.RawEdge) { e.Receiver = r } }
func withOwner(o string) callOpt      { return func(e *extract.RawEdge) { e.Owner = o } }
func withMember(m model.Kind) callOpt { return func(e *extract.RawEdge) { e.Member = m } }

func callWith(src model.NodeID, name string, opts ...callOpt) extract.RawEdge {
	e := extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, File: "m.rs"}
	for _, o := range opts {
		o(&e)
	}
	return e
}

func TestCallShapes(t *testing.T) {
	src := `fn f() {
    g();
    a::b::h();
    Foo::new();
    Self::s();
    self.m();
    x.k();
    a.b.k();
    self.inner.k();
    Foo(1);
    foo::<i32>();
    Vec::<i32>::new();
    x.t::<u8>();
    make().chain();
    ::top();
    (g)();
    <T as Tr>::q();
}
impl Foo {
    fn m(&self) {
        self.n();
        Self::o();
    }
}
`
	r := extractSrc(t, src)
	want := []extract.RawEdge{
		callWith("m.rs#f", "g"),
		callWith("m.rs#f", "h", withRecv("a::b")),
		callWith("m.rs#f", "new", withRecv("Foo")),
		callWith("m.rs#f", "s", withRecv("Self")),
		// self outside an impl names no type, so it is an ordinary object.
		callWith("m.rs#f", "m", withRecv("self"), withMember(".")),
		callWith("m.rs#f", "k", withRecv("x"), withMember(".")),
		callWith("m.rs#f", "k", withRecv("a.b"), withMember(".")),
		callWith("m.rs#f", "k", withRecv("self.inner"), withMember(".")),
		callWith("m.rs#f", "Foo"),
		callWith("m.rs#f", "foo"),
		callWith("m.rs#f", "new", withRecv("Vec")),
		callWith("m.rs#f", "t", withRecv("x"), withMember(".")),
		callWith("m.rs#f", "chain", withMember(".")),
		callWith("m.rs#f", "make"),
		callWith("m.rs#f", "top"),
		callWith("m.rs#Foo.m", "n", withOwner("Foo")),
		callWith("m.rs#Foo.m", "o", withRecv("Self")),
	}
	// make() comes after chain(): the walk is pre-order, and the outer call
	// is the method call.
	got := edgesOf(r, model.RelationCalls)
	same(t, "calls", got, want)
}

func TestCallLines(t *testing.T) {
	r := extractSrc(t, "fn f() {\n    g();\n    m!(h(1));\n}\n")
	var lines []int
	for _, e := range r.Edges {
		if e.Relation == model.RelationCalls {
			lines = append(lines, e.Line)
		}
	}
	same(t, "lines", lines, []int{2, 3, 3})
}

func TestCallsInMacroArgs(t *testing.T) {
	src := `fn f() {
    assert_eq!(helper(1), 2);
    println!("{}", a::g());
    vec![v.iter().map(|x| x.f())];
    format!("{x}");
    m!(self.go(2), Self::new(1), crate::a::b(3), x.y.z(), f::<T>(), if (a) {}, T { a: (b) });
    m!(outer(inner(1)));
    assert!(v.len() == 3);
}
`
	r := extractSrc(t, src)
	guess := func(name string, opts ...callOpt) extract.RawEdge {
		return callWith("m.rs#f", name, append(opts, withMember("guess"))...)
	}
	method := func(name string) extract.RawEdge { return callWith("m.rs#f", name, withMember("guess.")) }
	macro := func(name string) extract.RawEdge { return callWith("m.rs#f", name, withMember("macro")) }
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		macro("assert_eq"), guess("helper"),
		macro("println"), guess("g", withRecv("a")),
		macro("vec"), method("iter"), method("map"), method("f"),
		macro("format"),
		macro("m"), method("go"), guess("new", withRecv("Self")), guess("b", withRecv("crate::a")), method("z"),
		macro("m"), guess("outer"), guess("inner"),
		macro("assert"), method("len"),
	})
}

func TestMacroCall(t *testing.T) {
	r := extractSrc(t, "fn f() { my_macro!(); std::dbg!(1); }\nmy_macro!{ }\n")
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		callWith("m.rs#f", "my_macro", withMember("macro")),
		callWith("m.rs#f", "dbg", withRecv("std"), withMember("macro")),
		callWith("m.rs", "my_macro", withMember("macro")),
	})
}

func TestMacroNameThatIsNoPathMakesNoEdge(t *testing.T) {
	r := extractSrc(t, "fn f() { <T>::m!(); }\n")
	for _, e := range edgesOf(r, model.RelationCalls) {
		t.Errorf("unexpected call %+v", e)
	}
}

func TestCallBelongsToInnermost(t *testing.T) {
	src := `const C: i32 = k();
static S: i32 = l();
fn outer() {
    let c = |x| helper(x);
    fn nested() { deep(); }
    c(1);
}
struct Foo;
impl Foo {
    const A: i32 = assoc();
    fn m(&self) {
        let f = || self.n();
        f();
    }
}
trait Tr {
    fn d(&self) { self.e(); }
    fn s(&self);
}
mod inner {
    fn g() { h(); }
}
declare!();
`
	r := extractSrc(t, src)
	by := map[model.NodeID][]string{}
	for _, e := range edgesOf(r, model.RelationCalls) {
		by[e.Source] = append(by[e.Source], e.Name)
	}
	want := map[model.NodeID][]string{
		"m.rs":          {"k", "l", "assoc", "declare"},
		"m.rs#outer":    {"helper", "deep", "c"},
		"m.rs#Foo.m":    {"n", "f"},
		"m.rs#Tr.d":     {"e"},
		"m.rs#inner::g": {"h"},
	}
	for id, names := range want {
		same(t, string(id), by[id], names)
	}
	if len(by) != len(want) {
		t.Errorf("calls come from %v, want %v", by, want)
	}
	owners := map[string]string{}
	for _, e := range edgesOf(r, model.RelationCalls) {
		owners[e.Name] = e.Owner
	}
	// self names the impl's type, also in a closure, and the trait in a default method.
	if owners["n"] != "Foo" || owners["e"] != "Tr" || owners["assoc"] != "" {
		t.Errorf("owners = %v", owners)
	}
}

func TestTraitEdges(t *testing.T) {
	src := `trait Trait {}
trait A: B + c::C + Gen<X> + 'static + ?Sized {}
struct Foo;
impl Trait for Foo {}
impl<T: X> a::Tr for Foo<T> {}
impl Late for Later {}
impl Into<u8> for x::y::Baz {}
impl Foo {}
impl !Send for Foo {}
struct Later;
impl<T> Trait for Vec<T> {}
`
	r := extractSrc(t, src)
	rel := func(src model.NodeID, rel model.Relation, recv, name, owner string) extract.RawEdge {
		return extract.RawEdge{Source: src, Relation: rel, Receiver: recv, Name: name, Owner: owner, File: "m.rs"}
	}
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		rel("m.rs#A", model.RelationExtends, "", "B", ""),
		rel("m.rs#A", model.RelationExtends, "c", "C", ""),
		rel("m.rs#A", model.RelationExtends, "", "Gen", ""),
	})
	same(t, "implements", edgesOf(r, model.RelationImplements), []extract.RawEdge{
		rel("m.rs#Foo", model.RelationImplements, "", "Trait", "Foo"),
		rel("m.rs#Foo", model.RelationImplements, "a", "Tr", "Foo"),
		rel("m.rs#Later", model.RelationImplements, "", "Late", "Later"),
		// Baz and Vec are declared elsewhere: the edge starts at the file.
		rel("m.rs", model.RelationImplements, "", "Into", "Baz"),
		rel("m.rs#Foo", model.RelationImplements, "", "Send", "Foo"),
		rel("m.rs", model.RelationImplements, "", "Trait", "Vec"),
	})
}
