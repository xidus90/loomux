package resolve_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/csharp"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// csGraph extracts a tree of C# files, path to source, with the real
// extractor, and resolves them. The files are handed over in reverse byte
// order: which part or file the resolver picks must not depend on it.
func csGraph(t *testing.T, tree map[string]string) (*model.Graph, []resolve.Unresolved) {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var files []extract.Result
	for _, p := range paths {
		r, err := csharp.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	g, un := resolve.GraphWith(files, nil, stamp)
	return validated(t, g), un
}

func TestCSharpTypeResolutionOrder(t *testing.T) {
	for _, c := range []struct {
		name       string
		tree       map[string]string
		from, want string
	}{
		{"nested type of the enclosing type", map[string]string{
			"a.cs": "namespace N;\nclass Outer { class B { } class A : B { } }\n",
			"b.cs": "namespace N;\nclass B { }\n",
		}, "a.cs#Outer.A", "a.cs#Outer.B"},
		{"own namespace", map[string]string{
			"a.cs": "using U;\nnamespace X.Y;\nclass A : B { }\n",
			"b.cs": "namespace X.Y;\nclass B { }\n",
			"c.cs": "namespace X;\nclass B { }\n",
			"u.cs": "namespace U;\nclass B { }\n",
		}, "a.cs#A", "b.cs#B"},
		{"outer namespace", map[string]string{
			"a.cs": "using U;\nnamespace X.Y;\nclass A : B { }\n",
			"c.cs": "namespace X;\nclass B { }\n",
			"u.cs": "namespace U;\nclass B { }\n",
		}, "a.cs#A", "c.cs#B"},
		{"alias of a type", map[string]string{
			"a.cs": "using U;\nusing B = P.Real;\nclass A : B { }\n",
			"p.cs": "namespace P;\nclass Real { }\n",
			"u.cs": "namespace U;\nclass B { }\n",
		}, "a.cs#A", "p.cs#Real"},
		{"alias of a namespace", map[string]string{
			"a.cs": "using PQ = P;\nclass A : PQ.Real { }\n",
			"p.cs": "namespace P;\nclass Real { }\n",
		}, "a.cs#A", "p.cs#Real"},
		{"using", map[string]string{
			"a.cs": "using U;\nclass A : B { }\n",
			"g.cs": "class B { }\n",
			"u.cs": "namespace U;\nclass B { }\n",
		}, "a.cs#A", "u.cs#B"},
		{"global using of another file", map[string]string{
			"GlobalUsings.cs": "global using Game.Util;\n",
			"a.cs":            "class A : Helper { }\n",
			"g.cs":            "class Helper { }\n",
			"u.cs":            "namespace Game.Util;\nclass Helper { }\n",
		}, "a.cs#A", "u.cs#Helper"},
		{"plain using of another file", map[string]string{
			"a.cs":     "class A : B { }\n",
			"g.cs":     "class B { }\n",
			"other.cs": "using U;\n",
			"u.cs":     "namespace U;\nclass B { }\n",
		}, "a.cs#A", "g.cs#B"},
		{"global namespace", map[string]string{
			"a.cs": "namespace X;\nclass A : B { }\n",
			"g.cs": "class B { }\n",
		}, "a.cs#A", "g.cs#B"},
		{"qualified name", map[string]string{
			"a.cs": "class A : X.Y.B { }\n",
			"b.cs": "namespace X.Y;\nclass B { }\n",
		}, "a.cs#A", "b.cs#B"},
	} {
		t.Run(c.name, func(t *testing.T) {
			g, un := csGraph(t, c.tree)
			wantTargets(t, g, model.NodeID(c.from), model.RelationExtends, c.want)
			if len(un) != 0 {
				t.Errorf("unresolved = %+v, want none", un)
			}
		})
	}
}

func TestCSharpUnresolvedBases(t *testing.T) {
	g, un := csGraph(t, map[string]string{
		"a.cs": "using U;\nusing V;\nclass A : B, C, Godot.Node { void M() { Nowhere(); } }\n",
		"u.cs": "namespace U;\nclass B { }\n",
		"v.cs": "namespace V;\nclass B { }\n",
	})
	wantEdges(t, g, "a.cs -imports-> u.cs extracted", "a.cs -imports-> v.cs extracted")
	want := []resolve.Unresolved{
		{Source: "a.cs#A", File: "a.cs", Line: 3, Name: "B", Reason: resolve.ReasonAmbiguous},
		{Source: "a.cs#A", File: "a.cs", Line: 3, Name: "C", Reason: resolve.ReasonNone},
		{Source: "a.cs#A", File: "a.cs", Line: 3, Name: "Godot.Node", Reason: resolve.ReasonNone},
	}
	if !reflect.DeepEqual(un, want) {
		t.Errorf("unresolved = %+v\nwant %+v", un, want)
	}
}

func TestCSharpInterfacesAreImplemented(t *testing.T) {
	g, _ := csGraph(t, map[string]string{
		"a.cs": "class A : B, I { }\nclass B { }\ninterface I { }\ninterface J : I { }\n",
	})
	wantEdges(t, g,
		"a.cs#A -extends-> a.cs#B extracted",
		"a.cs#A -implements-> a.cs#I extracted",
		"a.cs#J -implements-> a.cs#I extracted",
	)
}

func TestCSharpImportsReachTheNamespacesFile(t *testing.T) {
	g, _ := csGraph(t, map[string]string{
		"a.cs": "using System;\nusing N;\nusing N;\nusing static N.T;\nusing Al = N;\nusing T;\n" +
			"class A { }\n",
		"n/NTests.cs":   "namespace N;\nclass NTests { }\n",
		"n/b.cs":        "namespace N;\nclass T { }\n",
		"n/c.cs":        "using N;\nnamespace N;\nclass C { }\n",
		"t/FooTests.cs": "namespace T;\nclass FooTests { }\n",
		// A test file imports like any other.
		"Tests/x.cs": "using N;\nnamespace N.Sub;\nclass X { }\n",
	})
	var got []string
	for _, e := range g.Edges {
		if e.Relation == model.RelationImports {
			got = append(got, fmt.Sprintf("%s -> %s L%d", e.Source, e.Target, e.Line))
		}
	}
	want := []string{
		"Tests/x.cs -> n/b.cs L1",
		"a.cs -> n/b.cs L2",
		"a.cs -> t/FooTests.cs L6",
		"n/c.cs -> n/b.cs L1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("imports = %q, want %q", got, want)
	}
}

func TestCSharpPartialTypesAreOneType(t *testing.T) {
	g, un := csGraph(t, map[string]string{
		"a.cs":    "namespace G;\npartial class Player : Base { void M() { } }\n",
		"b.cs":    "namespace G;\npartial class Player : IFoo { void Run() { this.M(); Hit(); } }\n",
		"base.cs": "namespace G;\nclass Base { void Hit() { } }\n",
		"ifoo.cs": "namespace G;\ninterface IFoo { }\n",
		"user.cs": "namespace G;\nclass Hero : Player { void S() { new Player(); } }\n",
	})
	wantEdges(t, g,
		"a.cs#Player -extends-> base.cs#Base extracted",
		"b.cs#Player -implements-> ifoo.cs#IFoo extracted",
		"user.cs#Hero -extends-> a.cs#Player extracted",
		"user.cs#Hero.S -calls-> a.cs#Player extracted",
		"b.cs#Player.Run -calls-> a.cs#Player.M extracted",
		"b.cs#Player.Run -calls-> base.cs#Base.Hit extracted",
	)
	if len(un) != 0 {
		t.Errorf("unresolved = %+v, want none", un)
	}
}

func TestCSharpCalls(t *testing.T) {
	g, un := csGraph(t, map[string]string{
		"c.cs": `namespace C;
using static C.Util;
using static C.Two;
class A : B {
    public A() { }
    public A(int x) { }
    void Own() { }
    void Own(int x) { }
    void Run() {
        Own();
        Inherited();
        Missing();
        this.Own();
        this.Helper();
        base.Inherited();
        base.Own();
        Util.Helper();
        Helper();
        Other();
        new A();
        new B();
        new Outer.Inner();
        local.Thing();
        new Nowhere();
    }
}
class B { void Inherited() { } }
static class Util { public static void Helper() { } public static void Other() { } }
static class Two { public static void Other() { } }
class Outer { public class Inner { public Inner() { } } }
`,
		// Outside every type, this and base name nothing.
		"top.cs": "Run();\nthis.Run();\nbase.Run();\n",
	})
	wantEdges(t, g,
		"c.cs#A -extends-> c.cs#B extracted",
		"c.cs#A.Run -calls-> c.cs#A.Own extracted",
		"c.cs#A.Run -calls-> c.cs#B.Inherited extracted",
		"c.cs#A.Run -calls-> c.cs#A.Own extracted",
		"c.cs#A.Run -calls-> c.cs#B.Inherited extracted",
		"c.cs#A.Run -calls-> c.cs#Util.Helper extracted",
		"c.cs#A.Run -calls-> c.cs#Util.Helper extracted",
		"c.cs#A.Run -calls-> c.cs#A.A extracted",
		"c.cs#A.Run -calls-> c.cs#B extracted",
		"c.cs#A.Run -calls-> c.cs#Outer.Inner.Inner extracted",
	)
	if len(un) != 0 {
		t.Errorf("unresolved = %+v, want none: a call never gives one", un)
	}
}

func TestCSharpUsingStaticInForceTwiceIsNoGuess(t *testing.T) {
	g, _ := csGraph(t, map[string]string{
		"GlobalUsings.cs": "global using static C.Util;\n",
		"a.cs":            "using static C.Util;\nclass A { void Run() { Helper(); } }\n",
		"u.cs":            "namespace C;\nstatic class Util { public static void Helper() { } }\n",
	})
	wantTargets(t, g, "a.cs#A.Run", model.RelationCalls, "u.cs#Util.Helper")
}

func TestCSharpNestedTypeCallsEnclosingMembers(t *testing.T) {
	g, _ := csGraph(t, map[string]string{
		"a.cs": `class Outer : Base {
    static void Shared() { }
    static void Both() { }
    class Inner : InnerBase {
        void Both() { }
        void Run() { Shared(); Both(); Inherited(); FromOuterBase(); }
    }
}
class Base { static void FromOuterBase() { } static void Inherited() { } }
class InnerBase { void Inherited() { } }
`,
	})
	wantTargets(t, g, "a.cs#Outer.Inner.Run", model.RelationCalls,
		"a.cs#Base.FromOuterBase", "a.cs#InnerBase.Inherited", "a.cs#Outer.Inner.Both", "a.cs#Outer.Shared")
}

func TestCSharpCyclicBasesTerminate(t *testing.T) {
	g, _ := csGraph(t, map[string]string{
		"a.cs": "class A : B { void M() { Missing(); base.Missing(); } }\nclass B : A { }\n",
	})
	wantEdges(t, g,
		"a.cs#A -extends-> a.cs#B extracted",
		"a.cs#B -extends-> a.cs#A extracted",
	)
}
