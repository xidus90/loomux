package rust_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/rust"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, src string) extract.Result {
	t.Helper()
	r, err := rust.File("m.rs", src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// view is a node as most tests compare it: everything but its body.
func view(n model.Node) string {
	return fmt.Sprintf("%s %s owner=%s %s exported=%v test=%v sig=%q", n.ID, n.Kind, n.Owner, n.Span, n.Exported, n.Test, n.Signature)
}

func views(r extract.Result) []string {
	var out []string
	for _, n := range r.Nodes {
		out = append(out, view(n))
	}
	return out
}

func ids(r extract.Result) []model.NodeID {
	var out []model.NodeID
	for _, n := range r.Nodes {
		out = append(out, n.ID)
	}
	return out
}

func node(t *testing.T, r extract.Result, id model.NodeID) model.Node {
	t.Helper()
	for _, n := range r.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("no node %s among %v", id, ids(r))
	return model.Node{}
}

func edgesOf(r extract.Result, rel model.Relation) []extract.RawEdge {
	var out []extract.RawEdge
	for _, e := range r.Edges {
		if e.Relation == rel {
			e.Line = 0 // lines are asserted in TestSpansAndBindingLines
			out = append(out, e)
		}
	}
	return out
}

func contains(src, dst model.NodeID) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: "m.rs"}
}

func binding(src model.NodeID, member extract.Namespace, godot, recv, name string) extract.RawEdge {
	return extract.RawEdge{
		Source: src, Relation: model.RelationReferences, Member: model.Kind(member),
		Specifier: godot, Receiver: recv, Name: name, File: "m.rs",
	}
}

func same[T any](t *testing.T, what string, got, want []T) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got %v\nwant %v", what, got, want)
	}
}

func TestLanguageDescribesRust(t *testing.T) {
	var l extract.Language = rust.Language{}
	if l.Name() != "rust" {
		t.Errorf("Name() = %q", l.Name())
	}
	if want := "rust/1@" + treesitter.Parser; l.Version() != want {
		t.Errorf("Version() = %q, want %q", l.Version(), want)
	}
	same(t, "Extensions()", l.Extensions(), []string{".rs"})
	r, err := l.File("m.rs", "fn f() {}\n")
	if err != nil {
		t.Fatal(err)
	}
	if r.Language != "rust" || r.Path != "m.rs" {
		t.Errorf("Result = %q %q", r.Language, r.Path)
	}
}

func TestItemsGetNodes(t *testing.T) {
	src := `pub fn run(a: i32) -> i32 { a }
pub(crate) struct P<T> where T: X { a: i32 }
struct U;
struct T(i32);
enum E { A, B }
union Un { a: u8 }
trait Tr {
    fn d(&self) {}
    fn s(&self);
}
type A = i32;
macro_rules! m { () => {}; }
`
	r := extractSrc(t, src)
	same(t, "nodes", views(r)[1:], []string{
		`m.rs#run function owner= L1-L1 exported=true test=false sig="pub fn run(a: i32) -> i32"`,
		`m.rs#P struct owner= L2-L2 exported=true test=false sig="pub(crate) struct P<T> where T: X"`,
		`m.rs#U struct owner= L3-L3 exported=false test=false sig="struct U"`,
		`m.rs#T struct owner= L4-L4 exported=false test=false sig="struct T"`,
		`m.rs#E enum owner= L5-L5 exported=false test=false sig="enum E"`,
		`m.rs#Un union owner= L6-L6 exported=false test=false sig="union Un"`,
		`m.rs#Tr trait owner= L7-L10 exported=false test=false sig="trait Tr"`,
		`m.rs#Tr.d method owner=Tr L8-L8 exported=false test=false sig="fn d(&self)"`,
		`m.rs#Tr.s method owner=Tr L9-L9 exported=false test=false sig="fn s(&self)"`,
		`m.rs#A type owner= L11-L11 exported=false test=false sig="type A = i32"`,
		`m.rs#m macro owner= L12-L12 exported=false test=false sig="macro_rules! m"`,
	})
	if r.Nodes[0].ID != "m.rs" || r.Nodes[0].Kind != model.KindFile {
		t.Errorf("first node = %v, want the file", r.Nodes[0].ID)
	}
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.rs", "m.rs#run"), contains("m.rs", "m.rs#P"), contains("m.rs", "m.rs#U"),
		contains("m.rs", "m.rs#T"), contains("m.rs", "m.rs#E"), contains("m.rs", "m.rs#Un"),
		contains("m.rs", "m.rs#Tr"), contains("m.rs#Tr", "m.rs#Tr.d"), contains("m.rs#Tr", "m.rs#Tr.s"),
		contains("m.rs", "m.rs#A"), contains("m.rs", "m.rs#m"),
	})
}

func TestTraitMethodsAreExportedWithTheirTrait(t *testing.T) {
	r := extractSrc(t, "pub trait Tr { fn d(&self); }\n")
	if n := node(t, r, "m.rs#Tr.d"); !n.Exported {
		t.Errorf("%s exported = false, want true: a trait's methods are as visible as the trait", n.ID)
	}
}

func TestImplMethods(t *testing.T) {
	src := `struct Foo<T> { t: T }
impl<T> Foo<T> {
    pub fn a() {}
}
impl Display for Foo<i32> {
    fn fmt(&self) {}
}
impl Debug for Foo<i32> {
    fn fmt(&self) {}
}
impl Other for &Bar {
    fn o() {}
}
impl x::y::Baz {
    fn p() {}
}
impl Marker for (A, B) {
    fn q() {}
}
`
	r := extractSrc(t, src)
	same(t, "nodes", views(r)[1:], []string{
		`m.rs#Foo struct owner= L1-L1 exported=false test=false sig="struct Foo<T>"`,
		`m.rs#Foo.a method owner=Foo L3-L3 exported=true test=false sig="pub fn a()"`,
		`m.rs#Foo.fmt method owner=Foo L6-L6 exported=false test=false sig="fn fmt(&self)"`,
		`m.rs#Foo.fmt~2 method owner=Foo L9-L9 exported=false test=false sig="fn fmt(&self)"`,
		`m.rs#Bar.o method owner=Bar L12-L12 exported=false test=false sig="fn o()"`,
		`m.rs#Baz.p method owner=Baz L15-L15 exported=false test=false sig="fn p()"`,
		`m.rs#(A, B).q method owner=(A, B) L18-L18 exported=false test=false sig="fn q()"`,
	})
	// A type in the file contains its methods; one that is not here, and
	// impls of other crates' types, hang from the file.
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.rs", "m.rs#Foo"),
		contains("m.rs#Foo", "m.rs#Foo.a"), contains("m.rs#Foo", "m.rs#Foo.fmt"), contains("m.rs#Foo", "m.rs#Foo.fmt~2"),
		contains("m.rs", "m.rs#Bar.o"), contains("m.rs", "m.rs#Baz.p"), contains("m.rs", "m.rs#(A, B).q"),
	})
}

func TestImplBeforeItsStructStillHangsFromIt(t *testing.T) {
	r := extractSrc(t, "impl Foo { fn a() {} }\nstruct Foo;\n")
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.rs#Foo", "m.rs#Foo.a"), contains("m.rs", "m.rs#Foo"),
	})
}

func TestInlineModulesQualify(t *testing.T) {
	src := `mod a {
    pub struct S;
    impl S { fn m(&self) {} }
    mod b {
        fn f() {}
    }
}
mod ext;
mod c { fn f() {} }
`
	r := extractSrc(t, src)
	same(t, "ids", ids(r), []model.NodeID{
		"m.rs", "m.rs#a::S", "m.rs#a::S.m", "m.rs#a::b::f", "m.rs#c::f",
	})
	for id, name := range map[model.NodeID]string{"m.rs#a::b::f": "f", "m.rs#a::S.m": "m", "m.rs#a::S": "S"} {
		if n := node(t, r, id); n.Name != name {
			t.Errorf("%s name = %q, want %q", id, n.Name, name)
		}
	}
	if n := node(t, r, "m.rs#a::S.m"); n.Owner != "S" {
		t.Errorf("owner = %q, want S", n.Owner)
	}
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.rs", "m.rs#a::S"), contains("m.rs#a::S", "m.rs#a::S.m"),
		contains("m.rs", "m.rs#a::b::f"), contains("m.rs", "m.rs#c::f"),
	})
}

func TestBrokenHeaderGetsNoNode(t *testing.T) {
	// The first definition's header holds text the grammar rejects, and the
	// parser rebuilt the definition around it: its extent would claim
	// the lines of its neighbour.
	r := extractSrc(t, brokenHeader+"fn ok() {}\n")
	if got := ids(r); !reflect.DeepEqual(got, []model.NodeID{"m.rs", "m.rs#ok"}) {
		t.Errorf("ids = %v, want the file and ok alone", got)
	}
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the rejected text counted")
	}
}

// brokenHeader is a function whose parameter list the parser rejects.
const brokenHeader = "fn broken(a: i32, @@@ ) -> u8 { 1 }\n"

func TestNeighbourOfBrokenItemArrives(t *testing.T) {
	r := extractSrc(t, "fn first() {}\n"+brokenHeader+"struct After;\nimpl After { fn m() {} }\n")
	same(t, "ids", ids(r), []model.NodeID{"m.rs", "m.rs#first", "m.rs#After", "m.rs#After.m"})
}

func TestBrokenImplAndMethodGetNoNode(t *testing.T) {
	r := extractSrc(t, "impl Foo { "+brokenHeader+" fn ok() {} }\nimpl @@@ Bar { fn lost() {} }\nfn tail() {}\n")
	same(t, "ids", ids(r), []model.NodeID{"m.rs", "m.rs#Foo.ok", "m.rs#tail"})
}

func TestTestMarking(t *testing.T) {
	src := `fn product() {}
#[test]
fn plain() {}
#[tokio::test(flavor = "multi_thread")]
async fn tokio_one() {}
#[cfg(test)]
mod tests {
    fn helper() {}
    mod inner { fn deep() {} }
    #[test]
    fn adds() {}
}
#[cfg( test )]
fn alone() {}
#[cfg(all(test, unix))]
fn mixed() {}
struct S;
impl S {
    #[test]
    fn method_test(&self) {}
    fn method(&self) {}
}
#[cfg(test)]
impl S { fn only_in_tests() {} }
`
	r := extractSrc(t, src)
	got := map[model.NodeID]bool{}
	for _, n := range r.Nodes[1:] {
		got[n.ID] = n.Test
	}
	want := map[model.NodeID]bool{
		"m.rs#product": false, "m.rs#plain": true, "m.rs#tokio_one": true,
		"m.rs#tests::helper": true, "m.rs#tests::inner::deep": true, "m.rs#tests::adds": true,
		"m.rs#alone": true, "m.rs#mixed": false, "m.rs#S": false,
		"m.rs#S.method_test": true, "m.rs#S.method": false, "m.rs#S.only_in_tests": true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Test marks:\n got %v\nwant %v", got, want)
	}
}

func TestGodotClassOffer(t *testing.T) {
	src := `#[derive(GodotClass)]
#[class(init, base=Node)]
struct Plain { base: Base<Node> }

#[derive(Debug, GodotClass)]
#[class(init, rename = Renamed)]
struct Original;

#[derive(godot::prelude::GodotClass)]
struct Qualified;

#[derive(Debug, Clone)]
struct NotGodot;

#[class(rename = Orphan)]
struct NoDerive;

mod m {
    #[derive(GodotClass)]
    struct InMod;
}
`
	r := extractSrc(t, src)
	offer := func(name string, target model.NodeID) extract.Export {
		return extract.Export{Namespace: extract.NamespaceGodotClass, Name: name, Target: target}
	}
	same(t, "exports", r.Exports, []extract.Export{
		offer("Plain", "m.rs#Plain"), offer("Renamed", "m.rs#Original"),
		offer("Qualified", "m.rs#Qualified"), offer("InMod", "m.rs#m::InMod"),
	})
}

func TestGodotBindings(t *testing.T) {
	src := `struct P;
#[godot_api]
impl P {
    #[func]
    fn jump(&mut self) {}
    #[func(rename=do_move)]
    pub fn moved(&mut self) {}
    #[rpc(any_peer)]
    fn remote(&mut self) {}
    #[func]
    #[rpc(authority)]
    fn both(&mut self) {}
    #[rpc(any_peer)]
    #[func(rename=x)]
    fn rpc_first(&mut self) {}
    #[signal]
    #[func]
    fn signal_and_func(&mut self) {}
    #[signal]
    fn hit(d: i32);
    fn private(&self) {}
    #[cfg(test)]
    fn other(&self) {}
}
impl P {
    #[func]
    fn not_exposed(&self) {}
}
#[godot::prelude::godot_api(secondary)]
impl P {
    #[signal]
    fn second();
}
`
	r := extractSrc(t, src)
	refs := edgesOf(r, model.RelationReferences)
	same(t, "bindings", refs, []extract.RawEdge{
		binding("m.rs#P.jump", extract.NamespaceGodotMethod, "jump", "P", "jump"),
		binding("m.rs#P.moved", extract.NamespaceGodotMethod, "do_move", "P", "moved"),
		binding("m.rs#P.remote", extract.NamespaceGodotMethod, "remote", "P", "remote"),
		binding("m.rs#P.both", extract.NamespaceGodotMethod, "both", "P", "both"),
		binding("m.rs#P.rpc_first", extract.NamespaceGodotMethod, "x", "P", "rpc_first"),
		binding("m.rs#P.signal_and_func", extract.NamespaceGodotMethod, "signal_and_func", "P", "signal_and_func"),
		binding("m.rs#P.hit", extract.NamespaceGodotSignal, "hit", "P", "hit"),
		binding("m.rs#P.second", extract.NamespaceGodotSignal, "second", "P", "second"),
	})
	for _, e := range r.Edges {
		if e.Relation == model.RelationReferences && e.Line == 0 {
			t.Errorf("binding %s has no line", e.Name)
		}
	}
}
