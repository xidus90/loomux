package resolve_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/java"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// javaGraph extracts a tree of Java files, path to source, with the real
// extractor, and resolves them. The files are handed over in reverse byte
// order: which file the resolver picks must not depend on it.
func javaGraph(t *testing.T, tree map[string]string) (*model.Graph, []resolve.Unresolved) {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var files []extract.Result
	for _, p := range paths {
		r, err := java.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	g, un := resolve.GraphWith(files, nil, stamp)
	return validated(t, g), un
}

func TestJavaTypeResolutionOrder(t *testing.T) {
	for _, c := range []struct {
		name       string
		tree       map[string]string
		from, want string
	}{
		{"nested type of the enclosing type", map[string]string{
			"p/A.java": "package p;\nclass Outer { class B { } class A extends B { } }\n",
			"p/B.java": "package p;\nclass B { }\n",
		}, "p/A.java#Outer.A", "p/A.java#Outer.B"},
		{"nested type of a type further out", map[string]string{
			"p/A.java": "package p;\nclass Outer { class B { } class Mid { class A extends B { } } }\n",
			"p/B.java": "package p;\nclass B { }\n",
		}, "p/A.java#Outer.Mid.A", "p/A.java#Outer.B"},
		{"the file's own type before a twin elsewhere and an import", map[string]string{
			"p/A.java": "package p;\nimport q.B;\nclass A extends B { }\nclass B { }\n",
			// The same package in another module, smaller by path.
			"m/p/B.java": "package p;\nclass B { }\n",
			"q/B.java":   "package q;\npublic class B { }\n",
		}, "p/A.java#A", "p/A.java#B"},
		{"single-type import before the package", map[string]string{
			"p/A.java": "package p;\nimport q.B;\nclass A extends B { }\n",
			"p/B.java": "package p;\nclass B { }\n",
			"q/B.java": "package q;\npublic class B { }\n",
		}, "p/A.java#A", "q/B.java#B"},
		{"package before an on-demand import", map[string]string{
			"p/A.java": "package p;\nimport q.*;\nclass A extends B { }\n",
			"p/B.java": "package p;\nclass B { }\n",
			"q/B.java": "package q;\npublic class B { }\n",
		}, "p/A.java#A", "p/B.java#B"},
		{"on-demand import", map[string]string{
			"p/A.java": "package p;\nimport q.*;\nclass A extends B { }\n",
			"q/B.java": "package q;\npublic class B { }\n",
		}, "p/A.java#A", "q/B.java#B"},
		{"on-demand import of a type's members", map[string]string{
			"p/A.java": "package p;\nimport q.Outer.*;\nclass A extends B { }\n",
			"q/O.java": "package q;\npublic class Outer { public static class B { } }\n",
		}, "p/A.java#A", "q/O.java#Outer.B"},
		{"nested type through a single-type import", map[string]string{
			"p/A.java": "package p;\nimport q.Outer;\nclass A extends Outer.B { }\n",
			"q/O.java": "package q;\npublic class Outer { public static class B { } }\n",
		}, "p/A.java#A", "q/O.java#Outer.B"},
		{"fully qualified name", map[string]string{
			"p/A.java": "package p;\nclass A extends q.r.B { }\n",
			"q/B.java": "package q.r;\npublic class B { }\n",
		}, "p/A.java#A", "q/B.java#B"},
	} {
		t.Run(c.name, func(t *testing.T) {
			g, un := javaGraph(t, c.tree)
			wantTargets(t, g, model.NodeID(c.from), model.RelationExtends, c.want)
			if len(un) != 0 {
				t.Errorf("unresolved = %+v, want none", un)
			}
		})
	}
}

func TestJavaTwoOnDemandImportsOfOneNameAreAGuess(t *testing.T) {
	g, un := javaGraph(t, map[string]string{
		"p/A.java": "package p;\nimport q.*;\nimport r.*;\nclass A extends B implements Missing, x.Y {\n" +
			"  void m() { B.run(); Nowhere(); }\n}\n",
		"q/B.java": "package q;\npublic class B { public static void run() { } }\n",
		"r/B.java": "package r;\npublic class B { public static void run() { } }\n",
	})
	want := []resolve.Unresolved{
		{Source: "p/A.java#A", File: "p/A.java", Line: 4, Name: "B", Reason: resolve.ReasonAmbiguous},
		{Source: "p/A.java#A", File: "p/A.java", Line: 4, Name: "Missing", Reason: resolve.ReasonNone},
		{Source: "p/A.java#A", File: "p/A.java", Line: 4, Name: "x.Y", Reason: resolve.ReasonNone},
	}
	if !reflect.DeepEqual(un, want) {
		t.Errorf("unresolved = %+v\nwant %+v", un, want)
	}
	// Neither the ambiguous call nor the missing one gives an edge, nor an
	// Unresolved entry.
	wantEdges(t, g, "p/A.java -imports-> q/B.java extracted", "p/A.java -imports-> r/B.java extracted")
}

func TestJavaFilesWithoutAPackageFormOne(t *testing.T) {
	g, un := javaGraph(t, map[string]string{
		"A.java":   "class A extends B { void m() { B.run(); } }\n",
		"B.java":   "class B { static void run() { } }\n",
		"p/B.java": "package p;\npublic class B { }\n",
		// A named package sees no type of the unnamed one.
		"p/C.java": "package p;\nclass C extends B { }\nclass D extends Z { }\n",
		"Z.java":   "class Z { }\n",
	})
	wantEdges(t, g,
		"A.java#A -extends-> B.java#B extracted",
		"A.java#A.m -calls-> B.java#B.run extracted",
		"p/C.java#C -extends-> p/B.java#B extracted",
	)
	want := []resolve.Unresolved{{Source: "p/C.java#D", File: "p/C.java", Line: 3, Name: "Z", Reason: resolve.ReasonNone}}
	if !reflect.DeepEqual(un, want) {
		t.Errorf("unresolved = %+v\nwant %+v", un, want)
	}
}

func TestJavaClassesAreExtendedAndInterfacesImplemented(t *testing.T) {
	g, _ := javaGraph(t, map[string]string{
		"p/A.java": "package p;\nclass A extends B implements I, J { }\nclass B { }\ninterface I { }\n" +
			"interface J extends I { }\nenum E implements I { X }\nrecord R(int x) implements J { }\n" +
			"@interface Ann { }\nclass C implements Ann { }\n",
	})
	wantEdges(t, g,
		"p/A.java#A -extends-> p/A.java#B extracted",
		"p/A.java#A -implements-> p/A.java#I extracted",
		"p/A.java#A -implements-> p/A.java#J extracted",
		"p/A.java#J -implements-> p/A.java#I extracted",
		"p/A.java#E -implements-> p/A.java#I extracted",
		"p/A.java#R -implements-> p/A.java#J extracted",
		"p/A.java#C -implements-> p/A.java#Ann extracted",
	)
}

func TestJavaImportsReachTheDeclaringFile(t *testing.T) {
	g, _ := javaGraph(t, map[string]string{
		"src/main/java/a/App.java": "package a;\n" +
			"import java.util.List;\n" + // foreign type: no edge
			"import java.util.*;\n" + // foreign package: no edge
			"import b.Zed;\n" +
			"import b.*;\n" +
			"import b.*;\n" +
			"import static c.Util.helper;\n" +
			"import static c.Util.*;\n" +
			"import static java.lang.Math.max;\n" +
			"import b.Single.*;\n" +
			"import a.Self;\n" +
			"import t.*;\n" +
			"import module java.base;\n" +
			"class App { }\nclass Self { }\n",
		"src/main/java/b/Single.java": "package b;\npublic class Single { }\n",
		// package-info.java declares no type, yet it is a file of b.
		"src/main/java/b/package-info.java": "package b;\n",
		"src/main/java/b/Zed.java":          "package b;\npublic class Zed { }\n",
		"src/main/java/c/Util.java":         "package c;\npublic class Util { public static void helper() { } }\n",
		"src/main/java/b/Single2.java":      "package b;\nclass Single2 { }\n",
		// A package only tests declare stands for itself by a test file.
		"src/test/java/t/Fixture.java": "package t;\npublic class Fixture { }\n",
		// Smaller by path than every file of b, but a test.
		"lib/src/test/java/b/ATest.java": "package b;\nclass ATest { }\n",
	})
	var got []string
	for _, e := range g.Edges {
		if e.Relation == model.RelationImports {
			got = append(got, fmt.Sprintf("%s -> %s L%d", e.Source, e.Target, e.Line))
		}
	}
	want := []string{
		"src/main/java/a/App.java -> src/main/java/b/Single.java L5",
		"src/main/java/a/App.java -> src/main/java/b/Zed.java L4",
		"src/main/java/a/App.java -> src/main/java/c/Util.java L7",
		"src/main/java/a/App.java -> src/test/java/t/Fixture.java L12",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("imports = %q, want %q", got, want)
	}
}

func TestJavaCalls(t *testing.T) {
	g, un := javaGraph(t, map[string]string{
		"p/A.java": `package p;
import static p.Util.helper;
import static p.Statics.*;
import static p.Twin1.*;
import static p.Twin2.*;
class A extends B {
    A() { }
    A(int x) { }
    void own() { }
    void own(int x) { }
    void run() {
        own();
        inherited();
        missing();
        this.own();
        this.helper();
        super.inherited();
        super.own();
        Util.helper();
        p.Util.helper();
        helper();
        fromStar();
        twin();
        new A();
        new B();
        new Outer.Inner();
        new p.Outer.Inner();
        B b = new B();
        b.inherited();
        Nowhere.call();
        new Nowhere();
    }
}
class B { void inherited() { } }
class Util { static void helper() { } }
class Statics { static void fromStar() { } static void helper() { } }
class Twin1 { static void twin() { } }
class Twin2 { static void twin() { } }
class Outer { static class Inner { Inner() { } } }
`,
	})
	wantTargets(t, g, "p/A.java#A.run", model.RelationCalls,
		"p/A.java#A.A",
		"p/A.java#A.own",
		"p/A.java#A.own",
		"p/A.java#B",
		"p/A.java#B",
		"p/A.java#B.inherited",
		"p/A.java#B.inherited",
		"p/A.java#Outer.Inner.Inner",
		"p/A.java#Outer.Inner.Inner",
		"p/A.java#Statics.fromStar",
		"p/A.java#Util.helper",
		"p/A.java#Util.helper",
		"p/A.java#Util.helper",
	)
	if len(un) != 0 {
		t.Errorf("unresolved = %+v, want none: a call never gives one", un)
	}
}

func TestJavaNestedTypeCallsEnclosingMembers(t *testing.T) {
	g, _ := javaGraph(t, map[string]string{
		"p/A.java": `package p;
class Outer extends Base {
    static void shared() { }
    static void both() { }
    class Inner extends InnerBase {
        void both() { }
        void run() { shared(); both(); inherited(); fromOuterBase(); }
    }
}
class Base { static void fromOuterBase() { } static void inherited() { } }
class InnerBase { void inherited() { } }
`,
	})
	wantTargets(t, g, "p/A.java#Outer.Inner.run", model.RelationCalls,
		"p/A.java#Base.fromOuterBase", "p/A.java#InnerBase.inherited", "p/A.java#Outer.Inner.both", "p/A.java#Outer.shared")
}

func TestJavaCallsOutsideATypeAndOnCyclicBases(t *testing.T) {
	g, _ := javaGraph(t, map[string]string{
		"p/A.java": "package p;\nclass A extends B { void m() { missing(); super.missing(); } }\nclass B extends A { }\n",
		// A compact source file: its methods belong to no type here, and a
		// bare call finds them after the types around it and before a static
		// import of the same name. this.m() and super.m() name no type.
		"Main.java": "import static p.U.go;\nimport static p.U.helper;\n" +
			"void main() { go(); helper(); this.helper(); super.helper(); new p.U(); }\n" +
			"void helper() { }\nvoid helper(int x) { }\nclass K { void k() { helper(); } }\n",
		"p/U.java": "package p;\npublic class U { public static void go() { } public static void helper() { } }\n",
	})
	wantEdges(t, g,
		"p/A.java#A -extends-> p/A.java#B extracted",
		"p/A.java#B -extends-> p/A.java#A extracted",
		"Main.java -imports-> p/U.java extracted",
		"Main.java#main -calls-> p/U.java#U.go extracted",
		"Main.java#main -calls-> Main.java#helper extracted",
		"Main.java#main -calls-> p/U.java#U extracted",
		"Main.java#K.k -calls-> Main.java#helper extracted",
	)
}
