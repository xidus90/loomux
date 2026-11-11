package csharp_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/csharp"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, rel, src string) extract.Result {
	t.Helper()
	r, err := csharp.File(rel, src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// view is a node as most tests compare it: everything but its body.
func view(n model.Node) string {
	return fmt.Sprintf("%s %s owner=%s %s exported=%v sig=%q", n.ID, n.Kind, n.Owner, n.Span, n.Exported, n.Signature)
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

func contains(src, dst model.NodeID) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: "m.cs"}
}

func call(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, Receiver: receiver, File: "m.cs"}
}

func construct(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "new"
	return e
}

func imports(spec string) extract.RawEdge {
	return extract.RawEdge{Source: "m.cs", Relation: model.RelationImports, Specifier: spec, File: "m.cs"}
}

func extends(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationExtends, Name: name, Receiver: receiver, File: "m.cs"}
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

func TestLanguageDescribesCSharp(t *testing.T) {
	var l extract.Language = csharp.Language{}
	if l.Name() != "csharp" {
		t.Errorf("Name = %q, want csharp", l.Name())
	}
	if want := "csharp/1@" + treesitter.Parser; l.Version() != want {
		t.Errorf("Version = %q, want %q", l.Version(), want)
	}
	same(t, "Extensions", l.Extensions(), []string{".cs"})
	r, err := l.File("src/m.cs", "class K { }\n")
	if err != nil || r.Path != "src/m.cs" || r.Language != "csharp" || len(r.Nodes) != 2 {
		t.Errorf("File = %d nodes at %q (%q), %v; want the file and K at src/m.cs", len(r.Nodes), r.Path, r.Language, err)
	}
}

func TestTypesAndMembersGetNodes(t *testing.T) {
	const src = `public class C
{
    public C() { }
    public void M(int a) { }
    int P { get { return 1; } }
    int Auto { get; set; }
    internal int Arrow => 2;
    [Signal] public delegate void HitEventHandler();
    delegate void Plain();
    public static C operator +(C a, C b) { return a; }
    public int this[int i] { get => i; }
    protected internal void Shared() { }
    private void Hidden() { }
    class Inner { void N() { } }
    public class Open { }
}
struct S { }
record R(int A);
record struct RS(int A);
interface I { void X(); private void Y() { } }
enum E { A }
static class Ext { extension(int x) { public int Twice() => x; } }
`
	r := extractSrc(t, "m.cs", src)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	same(t, "nodes", views(r)[1:], []string{
		`m.cs#C class owner= L1-L16 exported=true sig="public class C"`,
		`m.cs#C.C method owner=C L3-L3 exported=true sig="public C()"`,
		`m.cs#C.M method owner=C L4-L4 exported=true sig="public void M(int a)"`,
		`m.cs#C.P method owner=C L5-L5 exported=false sig="int P"`,
		`m.cs#C.Arrow method owner=C L7-L7 exported=true sig="internal int Arrow"`,
		`m.cs#C.HitEventHandler signal owner=C L8-L8 exported=true sig="public delegate void HitEventHandler()"`,
		`m.cs#C.operator+ method owner=C L10-L10 exported=true sig="public static C operator +(C a, C b)"`,
		`m.cs#C.this method owner=C L11-L11 exported=true sig="public int this[int i]"`,
		`m.cs#C.Shared method owner=C L12-L12 exported=true sig="protected internal void Shared()"`,
		`m.cs#C.Hidden method owner=C L13-L13 exported=false sig="private void Hidden()"`,
		`m.cs#C.Inner class owner=C L14-L14 exported=false sig="class Inner"`,
		`m.cs#C.Inner.N method owner=C.Inner L14-L14 exported=false sig="void N()"`,
		`m.cs#C.Open class owner=C L15-L15 exported=true sig="public class Open"`,
		`m.cs#S class owner= L17-L17 exported=true sig="struct S"`,
		`m.cs#R class owner= L18-L18 exported=true sig="record R(int A)"`,
		`m.cs#RS class owner= L19-L19 exported=true sig="record struct RS(int A)"`,
		`m.cs#I interface owner= L20-L20 exported=true sig="interface I"`,
		`m.cs#I.X method owner=I L20-L20 exported=true sig="void X()"`,
		`m.cs#I.Y method owner=I L20-L20 exported=false sig="private void Y()"`,
		`m.cs#E class owner= L21-L21 exported=true sig="enum E"`,
		`m.cs#Ext class owner= L22-L22 exported=true sig="static class Ext"`,
		`m.cs#Ext.Twice method owner=Ext L22-L22 exported=true sig="public int Twice()"`,
	})
	if n := r.Nodes[0]; n.ID != "m.cs" || n.Kind != model.KindFile {
		t.Errorf("first node = %s, want the file node", view(n))
	}
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.cs", "m.cs#C"),
		contains("m.cs#C", "m.cs#C.C"),
		contains("m.cs#C", "m.cs#C.M"),
		contains("m.cs#C", "m.cs#C.P"),
		contains("m.cs#C", "m.cs#C.Arrow"),
		contains("m.cs#C", "m.cs#C.HitEventHandler"),
		contains("m.cs#C", "m.cs#C.operator+"),
		contains("m.cs#C", "m.cs#C.this"),
		contains("m.cs#C", "m.cs#C.Shared"),
		contains("m.cs#C", "m.cs#C.Hidden"),
		contains("m.cs#C", "m.cs#C.Inner"),
		contains("m.cs#C.Inner", "m.cs#C.Inner.N"),
		contains("m.cs#C", "m.cs#C.Open"),
		contains("m.cs", "m.cs#S"),
		contains("m.cs", "m.cs#R"),
		contains("m.cs", "m.cs#RS"),
		contains("m.cs", "m.cs#I"),
		contains("m.cs#I", "m.cs#I.X"),
		contains("m.cs#I", "m.cs#I.Y"),
		contains("m.cs", "m.cs#E"),
		contains("m.cs", "m.cs#Ext"),
		contains("m.cs#Ext", "m.cs#Ext.Twice"),
	})
}

func TestConversionOperatorsGetNodes(t *testing.T) {
	r := extractSrc(t, "m.cs", `class K {
    public static implicit operator int(K k) => F();
    public static explicit operator K(int i) { return new K(); }
}
`)
	same(t, "nodes", views(r)[1:], []string{
		`m.cs#K class owner= L1-L4 exported=true sig="class K"`,
		`m.cs#K.operator int method owner=K L2-L2 exported=true sig="public static implicit operator int(K k)"`,
		`m.cs#K.operator K method owner=K L3-L3 exported=true sig="public static explicit operator K(int i)"`,
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.cs#K.operator int", "F", ""),
		construct("m.cs#K.operator K", "K", ""),
	})
}

func TestNamespacesStayOutOfIdsAndOwners(t *testing.T) {
	r := extractSrc(t, "m.cs", `class G { }
namespace A.B {
    class K { class N { } }
    namespace C { class L { void M() { } } }
}
`)
	same(t, "nodes", views(r)[1:], []string{
		`m.cs#G class owner= L1-L1 exported=true sig="class G"`,
		`m.cs#K class owner= L3-L3 exported=true sig="class K"`,
		`m.cs#K.N class owner=K L3-L3 exported=false sig="class N"`,
		`m.cs#L class owner= L4-L4 exported=true sig="class L"`,
		`m.cs#L.M method owner=L L4-L4 exported=false sig="void M()"`,
	})
	if r.Package != "" {
		t.Errorf("Package = %q, want the global namespace of G", r.Package)
	}

	r = extractSrc(t, "m.cs", "using X;\nnamespace F.S;\nclass T { }\n")
	same(t, "nodes", ids(r), []model.NodeID{"m.cs", "m.cs#T"})
	if r.Package != "F.S" {
		t.Errorf("Package = %q, want F.S", r.Package)
	}
}

func TestEnclosingNamesTheNamespaceOfEveryType(t *testing.T) {
	r := extractSrc(t, "m.cs", `class G { void M() { } }
namespace A.B {
    class K { class N { } }
    namespace C { interface L { } }
}
`)
	want := map[model.NodeID]string{"m.cs#G": "", "m.cs#K": "A.B", "m.cs#K.N": "A.B", "m.cs#L": "A.B.C"}
	if !reflect.DeepEqual(r.Enclosing, want) {
		t.Errorf("Enclosing = %q, want %q", r.Enclosing, want)
	}

	r = extractSrc(t, "m.cs", "namespace F.S;\nenum E { A }\n")
	if want := map[model.NodeID]string{"m.cs#E": "F.S"}; !reflect.DeepEqual(r.Enclosing, want) {
		t.Errorf("Enclosing = %q, want %q", r.Enclosing, want)
	}

	// Without a type there is nothing to name, and the map stays nil as
	// it reads back from the extract cache.
	r = extractSrc(t, "m.cs", "namespace F.S;\nusing X;\n")
	if r.Enclosing != nil || r.Package != "" {
		t.Errorf("Enclosing, Package = %q, %q; want nil and none", r.Enclosing, r.Package)
	}
}

func TestUsingFormsBecomeImports(t *testing.T) {
	r := extractSrc(t, "m.cs", `global using System;
using static System.Math;
using A = X.Y.T;
using System.Collections.Generic;
using global::System.Text;
namespace N { using Inner.Ns; class K { } }
`)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "System", Global: true},
		{Path: "System.Math", Alias: "static"},
		{Path: "X.Y.T", Alias: "A"},
		{Path: "System.Collections.Generic"},
		{Path: "Inner.Ns"},
	})
	same(t, "import edges", edgesOf(r, model.RelationImports), []extract.RawEdge{
		imports("System"),
		imports("System.Math"),
		imports("X.Y.T"),
		imports("System.Collections.Generic"),
		imports("Inner.Ns"),
	})
}

func TestBaseListBecomesExtendsEdges(t *testing.T) {
	r := extractSrc(t, "m.cs", `class A<T> : B<T>, IFoo, X.Y.Z { }
interface I : J { }
record D(int A) : Base(A);
`)
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		extends("m.cs#A", "B", ""),
		extends("m.cs#A", "IFoo", ""),
		extends("m.cs#A", "Z", "X.Y"),
		extends("m.cs#I", "J", ""),
		extends("m.cs#D", "Base", ""),
	})
}

func TestCallsBelongToInnermostMember(t *testing.T) {
	r := extractSrc(t, "m.cs", `Run();
class K : B {
    int f = Init();
    [Obsolete] void M() {
        N();
        this.N();
        base.N();
        T.S();
        A.B.T.S();
        new T(Arg());
        new A.B.T();
        new int();
        G<int>();
        x.G<int>();
        F().N();
        nameof(N);
    }
    int P { get { return Q(); } }
    int Auto { get; } = Later();
}
class L { void Z() => K.Make(); }
`)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.cs", "Run", ""),
		call("m.cs#K", "Init", ""),
		call("m.cs#K.M", "N", ""),
		call("m.cs#K.M", "N", "this"),
		call("m.cs#K.M", "N", "base"),
		call("m.cs#K.M", "S", "T"),
		call("m.cs#K.M", "S", "A.B.T"),
		construct("m.cs#K.M", "T", ""),
		call("m.cs#K.M", "Arg", ""),
		construct("m.cs#K.M", "T", "A.B"),
		call("m.cs#K.M", "G", ""),
		call("m.cs#K.M", "G", "x"),
		call("m.cs#K.M", "F", ""),
		call("m.cs#K.P", "Q", ""),
		call("m.cs#K", "Later", ""),
		call("m.cs#L.Z", "Make", "K"),
	})
}

func TestPreprocessorAroundMembers(t *testing.T) {
	r := extractSrc(t, "m.cs", `class K {
#if A
    void D() { }
#elif B
    void E() { }
#else
    void F() { }
#endif
    #region R
    void G() { }
    #endregion
}
`)
	same(t, "nodes", ids(r), []model.NodeID{"m.cs", "m.cs#K", "m.cs#K.D", "m.cs#K.E", "m.cs#K.F", "m.cs#K.G"})
}

func TestGodotOffers(t *testing.T) {
	const src = `using Godot;
[GlobalClass]
public partial class Player : Node {
    [Signal] public delegate void HitEventHandler();
    [Signal] public delegate void Hit();
    public void OnHit(int a) { }
    public void OnHit() { }
    public Player() { }
    void _Ready() { }
    class Nested { void Deep() { } }
}
[GlobalClass, Tool]
public partial class Other : Resource { void Foo() { } }
`
	r := extractSrc(t, "scenes/Player.cs", src)
	same(t, "exports", r.Exports, []extract.Export{
		{Namespace: extract.NamespaceGodotClass, Name: "Player", Target: "scenes/Player.cs#Player"},
		{Namespace: extract.NamespaceGodotSignal, Name: "Hit", Target: "scenes/Player.cs#Player.HitEventHandler"},
		{Namespace: extract.NamespaceGodotMethod, Name: "OnHit", Target: "scenes/Player.cs#Player.OnHit"},
		{Namespace: extract.NamespaceGodotMethod, Name: "_Ready", Target: "scenes/Player.cs#Player._Ready"},
		{Namespace: extract.NamespaceGodotClass, Name: "Other", Target: "scenes/Player.cs#Other"},
	})

	// A class named like the file but deriving from nothing is no Godot
	// script: every script class extends a Godot type.
	r = extractSrc(t, "Player.cs", "class Player {\n    [Signal] delegate void HitEventHandler();\n    void OnHit() { }\n}\n")
	same(t, "exports", r.Exports, nil)

	// Another file name: no class is the script, and only GlobalClass is offered.
	r = extractSrc(t, "Enemy.cs", src)
	same(t, "exports", r.Exports, []extract.Export{
		{Namespace: extract.NamespaceGodotClass, Name: "Player", Target: "Enemy.cs#Player"},
		{Namespace: extract.NamespaceGodotClass, Name: "Other", Target: "Enemy.cs#Other"},
	})

	// A qualified name or the Attribute suffix names the same attribute.
	r = extractSrc(t, "Mob.cs", `[Godot.GlobalClass]
public partial class Mob : Node {
    [Godot.Signal] public delegate void HitEventHandler();
    [SignalAttribute] public delegate void DiedEventHandler();
    [Other.SignalX] public delegate void NoEventHandler();
}
[GlobalClassAttribute]
public partial class Two : Node { }
`)
	same(t, "exports", r.Exports, []extract.Export{
		{Namespace: extract.NamespaceGodotClass, Name: "Mob", Target: "Mob.cs#Mob"},
		{Namespace: extract.NamespaceGodotSignal, Name: "Hit", Target: "Mob.cs#Mob.HitEventHandler"},
		{Namespace: extract.NamespaceGodotSignal, Name: "Died", Target: "Mob.cs#Mob.DiedEventHandler"},
		{Namespace: extract.NamespaceGodotClass, Name: "Two", Target: "Mob.cs#Two"},
	})
}

func TestDefinitionInErrorGetsNoNode(t *testing.T) {
	r := extractSrc(t, "m.cs", `class Top {
    void Ok() { H(); }
    void Broken(int a b) { X(); }
    void After() { Y(); }
}
class Next { void Fine() { } }
`)
	same(t, "nodes", ids(r), []model.NodeID{"m.cs", "m.cs#Top", "m.cs#Top.Ok", "m.cs#Top.After", "m.cs#Next", "m.cs#Next.Fine"})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.cs#Top.Ok", "H", ""),
		call("m.cs#Top.After", "Y", ""),
	})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the rejected parameter counted")
	}

	// A broken type header drops the type with everything in it.
	r = extractSrc(t, "m.cs", "class K { class N(int a b) { void D() { F(); } } void M() { } }\n")
	same(t, "nodes", ids(r), []model.NodeID{"m.cs", "m.cs#K", "m.cs#K.M"})
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)

	// Rejected text in a body is skipped, and the calls around it stay.
	r = extractSrc(t, "m.cs", "class K { void M() { F(); ) G(); } }\n")
	same(t, "nodes", ids(r), []model.NodeID{"m.cs", "m.cs#K", "m.cs#K.M"})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.cs#K.M", "F", ""), call("m.cs#K.M", "G", "")})
}
