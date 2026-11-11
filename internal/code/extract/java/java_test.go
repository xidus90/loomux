package java_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/java"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, rel, src string) extract.Result {
	t.Helper()
	r, err := java.File(rel, src)
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
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: "m.java"}
}

func call(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, Receiver: receiver, File: "m.java"}
}

func onObject(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "."
	return e
}

func construct(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "new"
	return e
}

func imports(spec string) extract.RawEdge {
	return extract.RawEdge{Source: "m.java", Relation: model.RelationImports, Specifier: spec, File: "m.java"}
}

func extends(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationExtends, Name: name, Receiver: receiver, File: "m.java"}
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

func TestLanguageDescribesJava(t *testing.T) {
	var l extract.Language = java.Language{}
	if l.Name() != "java" {
		t.Errorf("Name = %q, want java", l.Name())
	}
	if want := "java/1@" + treesitter.Parser; l.Version() != want {
		t.Errorf("Version = %q, want %q", l.Version(), want)
	}
	same(t, "Extensions", l.Extensions(), []string{".java"})
	r, err := l.File("src/M.java", "class K { }\n")
	if err != nil || r.Path != "src/M.java" || r.Language != "java" || len(r.Nodes) != 2 {
		t.Errorf("File = %d nodes at %q (%q), %v; want the file and K at src/M.java", len(r.Nodes), r.Path, r.Language, err)
	}
}

func TestTypesAndMembersGetNodes(t *testing.T) {
	const src = `package a.b;
@Deprecated public class C<T> {
    public C() { }
    protected void m(int a) { }
    void plain() { }
    private void hidden() { }
    @Override public String toString() { return ""; }
    abstract void bare();
    class Inner { void n() { } }
    public static class Open { }
}
interface I { void x(); default void y() { } private void z() { } class Nested { } }
enum E { A, B { void constant() { } }; E() { } public void w() { } }
record R(int a) { R { } static void s() { } }
@interface Ann { int v() default 1; }
void main() { }
`
	r := extractSrc(t, "m.java", src)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	if r.Package != "a.b" {
		t.Errorf("Package = %q, want a.b", r.Package)
	}
	same(t, "nodes", views(r)[1:], []string{
		`m.java#C class owner= L2-L11 exported=true sig="public class C<T>"`,
		`m.java#C.C method owner=C L3-L3 exported=true sig="public C()"`,
		`m.java#C.m method owner=C L4-L4 exported=true sig="protected void m(int a)"`,
		`m.java#C.plain method owner=C L5-L5 exported=false sig="void plain()"`,
		`m.java#C.hidden method owner=C L6-L6 exported=false sig="private void hidden()"`,
		`m.java#C.toString method owner=C L7-L7 exported=true sig="public String toString()"`,
		`m.java#C.bare method owner=C L8-L8 exported=false sig="abstract void bare()"`,
		`m.java#C.Inner class owner=C L9-L9 exported=false sig="class Inner"`,
		`m.java#C.Inner.n method owner=C.Inner L9-L9 exported=false sig="void n()"`,
		`m.java#C.Open class owner=C L10-L10 exported=true sig="public static class Open"`,
		`m.java#I interface owner= L12-L12 exported=false sig="interface I"`,
		`m.java#I.x method owner=I L12-L12 exported=true sig="void x()"`,
		`m.java#I.y method owner=I L12-L12 exported=true sig="default void y()"`,
		`m.java#I.z method owner=I L12-L12 exported=false sig="private void z()"`,
		`m.java#I.Nested class owner=I L12-L12 exported=true sig="class Nested"`,
		`m.java#E class owner= L13-L13 exported=false sig="enum E"`,
		`m.java#E.E method owner=E L13-L13 exported=false sig="E()"`,
		`m.java#E.w method owner=E L13-L13 exported=true sig="public void w()"`,
		`m.java#R class owner= L14-L14 exported=false sig="record R(int a)"`,
		`m.java#R.R method owner=R L14-L14 exported=false sig="R"`,
		`m.java#R.s method owner=R L14-L14 exported=false sig="static void s()"`,
		`m.java#Ann interface owner= L15-L15 exported=false sig="@interface Ann"`,
		`m.java#main function owner= L16-L16 exported=false sig="void main()"`,
	})
	if n := r.Nodes[0]; n.ID != "m.java" || n.Kind != model.KindFile {
		t.Errorf("first node = %s, want the file node", view(n))
	}
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.java", "m.java#C"),
		contains("m.java#C", "m.java#C.C"),
		contains("m.java#C", "m.java#C.m"),
		contains("m.java#C", "m.java#C.plain"),
		contains("m.java#C", "m.java#C.hidden"),
		contains("m.java#C", "m.java#C.toString"),
		contains("m.java#C", "m.java#C.bare"),
		contains("m.java#C", "m.java#C.Inner"),
		contains("m.java#C.Inner", "m.java#C.Inner.n"),
		contains("m.java#C", "m.java#C.Open"),
		contains("m.java", "m.java#I"),
		contains("m.java#I", "m.java#I.x"),
		contains("m.java#I", "m.java#I.y"),
		contains("m.java#I", "m.java#I.z"),
		contains("m.java#I", "m.java#I.Nested"),
		contains("m.java", "m.java#E"),
		contains("m.java#E", "m.java#E.E"),
		contains("m.java#E", "m.java#E.w"),
		contains("m.java", "m.java#R"),
		contains("m.java#R", "m.java#R.R"),
		contains("m.java#R", "m.java#R.s"),
		contains("m.java", "m.java#Ann"),
		contains("m.java", "m.java#main"),
	})
}

func TestOverloadsGetOrdinals(t *testing.T) {
	r := extractSrc(t, "m.java", "class K { void f() { } void f(int a) { } }\n")
	same(t, "nodes", ids(r), []model.NodeID{"m.java", "m.java#K", "m.java#K.f", "m.java#K.f~2"})
}

func TestNoPackageIsTheUnnamedOne(t *testing.T) {
	if r := extractSrc(t, "m.java", "class K { }\n"); r.Package != "" {
		t.Errorf("Package = %q, want the unnamed package", r.Package)
	}
	if r := extractSrc(t, "m.java", "package p;\nclass K { }\n"); r.Package != "p" {
		t.Errorf("Package = %q, want p", r.Package)
	}
}

func TestImportFormsBecomeImports(t *testing.T) {
	r := extractSrc(t, "m.java", `import a.b.C;
import a.b.*;
import static a.b.C.m;
import static a.b.C.*;
import module java.base;
import module foo;
class K { }
`)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "a.b.C"},
		{Path: "a.b.*"},
		{Path: "a.b.C.m", Alias: "static"},
		{Path: "a.b.C.*", Alias: "static"},
		{Path: "java.base", Alias: "module"},
		{Path: "foo", Alias: "module"},
	})
	same(t, "import edges", edgesOf(r, model.RelationImports), []extract.RawEdge{
		imports("a.b.C"),
		imports("a.b.*"),
		imports("a.b.C.m"),
		imports("a.b.C.*"),
		imports("java.base"),
		imports("foo"),
	})
}

// Java 25's module import and flexible constructor bodies are beyond the
// grammar: the module import still becomes an import, and the members after
// a statement before super(...) arrive.
func TestJava25GapsKeepTheRestOfTheFile(t *testing.T) {
	// After a static on-demand import the grammar ends the module import at
	// "module" and reads the name as a statement of its own.
	r := extractSrc(t, "m.java", `import static a.b.C.*;
import module java.base;
class Flex {
    Flex(int x) { int y = 1; super(x); }
    void after() { g(); }
}
class Next { }
`)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "a.b.C.*", Alias: "static"},
		{Path: "java.base", Alias: "module"},
	})
	same(t, "nodes", ids(r), []model.NodeID{"m.java", "m.java#Flex", "m.java#Flex.Flex", "m.java#Flex.after", "m.java#Next"})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.java#Flex.after", "g", "")})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the gaps counted")
	}

	// Anywhere else the grammar keeps the name in the import and wraps
	// "module" in an ERROR.
	r = extractSrc(t, "m.java", "package p;\nimport a.B;\nimport module java.base;\nclass K { }\n")
	same(t, "imports", r.Imports, []extract.Import{{Path: "a.B"}, {Path: "java.base", Alias: "module"}})
	same(t, "nodes", ids(r), []model.NodeID{"m.java", "m.java#K"})

	// A module import whose name the grammar dropped names nothing.
	r = extractSrc(t, "m.java", "import static a.b.C.*;\nimport module\n")
	same(t, "imports", r.Imports, []extract.Import{{Path: "a.b.C.*", Alias: "static"}})
}

func TestImportInErrorIsLeftOut(t *testing.T) {
	r := extractSrc(t, "m.java", "import a.b.C;\nimport 1.b;\nimport a.(b);\nimport d.E;\nclass K { }\n")
	same(t, "imports", r.Imports, []extract.Import{{Path: "a.b.C"}, {Path: "d.E"}})
}

func TestPackageNameAfterAnnotations(t *testing.T) {
	if r := extractSrc(t, "package-info.java", "@Deprecated\npackage a.b;\n"); r.Package != "a.b" {
		t.Errorf("Package = %q, want a.b", r.Package)
	}
}

func TestBasesBecomeExtendsEdges(t *testing.T) {
	r := extractSrc(t, "m.java", `class A<T> extends B<T> implements IFoo, x.y.Z<T> { }
interface I extends J, a.L { }
enum E implements I { X }
record R(int a) implements I { }
sealed class S permits T1 { }
`)
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		extends("m.java#A", "B", ""),
		extends("m.java#A", "IFoo", ""),
		extends("m.java#A", "Z", "x.y"),
		extends("m.java#I", "J", ""),
		extends("m.java#I", "L", "a"),
		extends("m.java#E", "I", ""),
		extends("m.java#R", "I", ""),
	})
}

func TestCallsBelongToInnermostMember(t *testing.T) {
	r := extractSrc(t, "m.java", `class K extends B {
    int f = init();
    Other field;
    static { boot(); }
    void m(String s) {
        n();
        this.n();
        super.n();
        T.s();
        a.b.T.s();
        s.trim();
        field.go();
        this.field.go();
        get().n();
        new T(arg());
        new a.b.T<String>();
        new int[3];
        for (var e : xs) e.run();
        Runnable q = () -> z.m();
        java.util.Collections.sort(xs);
    }
    void lambdas() { F g = (p, q) -> p.m(); F h = r -> r.m(); }
}
enum E { A { void z() { inside(); } }; void w() { A.go(); } }
`)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.java#K", "init", ""),
		call("m.java#K", "boot", ""),
		call("m.java#K.m", "n", ""),
		call("m.java#K.m", "n", "this"),
		call("m.java#K.m", "n", "super"),
		call("m.java#K.m", "s", "T"),
		call("m.java#K.m", "s", "a.b.T"),
		onObject("m.java#K.m", "trim", "s"),
		onObject("m.java#K.m", "go", "field"),
		onObject("m.java#K.m", "go", "this.field"),
		call("m.java#K.m", "get", ""),
		construct("m.java#K.m", "T", ""),
		call("m.java#K.m", "arg", ""),
		construct("m.java#K.m", "T", "a.b"),
		onObject("m.java#K.m", "run", "e"),
		call("m.java#K.m", "m", "z"),
		call("m.java#K.m", "sort", "java.util.Collections"),
		onObject("m.java#K.lambdas", "m", "p"),
		onObject("m.java#K.lambdas", "m", "r"),
		call("m.java#E", "inside", ""),
		onObject("m.java#E.w", "go", "A"),
	})
}

// Every way Java declares a variable makes a receiver of that name an
// object, wherever in the file it is declared.
func TestDeclaredVariablesMarkObjects(t *testing.T) {
	r := extractSrc(t, "m.java", `class K {
    void m(int... va) {
        try (var rs = open()) { rs.a(); } catch (Exception ex) { ex.a(); }
        if (o instanceof Foo f) f.a();
        switch (o) { case Bar b -> b.a(); default -> { } }
        va.a();
        Q.a();
    }
}
`)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.java#K.m", "open", ""),
		onObject("m.java#K.m", "a", "rs"),
		onObject("m.java#K.m", "a", "ex"),
		onObject("m.java#K.m", "a", "f"),
		onObject("m.java#K.m", "a", "b"),
		onObject("m.java#K.m", "a", "va"),
		call("m.java#K.m", "a", "Q"),
	})
}

func TestDefinitionInErrorGetsNoNode(t *testing.T) {
	r := extractSrc(t, "m.java", `class K<T extends> { void m() { m(); } }
class L { void n(int x int y) { n(); } void o() { p(); } }
class After { void a() { } }
`)
	same(t, "nodes", ids(r), []model.NodeID{"m.java", "m.java#L", "m.java#L.o", "m.java#After", "m.java#After.a"})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.java#L.o", "p", "")})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the rejected text counted")
	}
}
