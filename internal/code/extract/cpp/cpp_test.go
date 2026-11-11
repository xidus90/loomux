package cpp_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/cpp"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, src string) extract.Result {
	t.Helper()
	r, err := cpp.File("m.cpp", src)
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
	for _, n := range r.Nodes[1:] {
		out = append(out, view(n))
	}
	return out
}

func ids(r extract.Result) []model.NodeID {
	var out []model.NodeID
	for _, n := range r.Nodes[1:] {
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
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: "m.cpp"}
}

func call(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, Receiver: receiver, File: "m.cpp"}
}

func construct(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "new"
	return e
}

func access(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "."
	return e
}

func extends(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationExtends, Name: name, Receiver: receiver, File: "m.cpp"}
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

func TestLanguageDescribesCpp(t *testing.T) {
	var l extract.Language = cpp.Language{}
	if l.Name() != "cpp" {
		t.Errorf("Name = %q, want cpp", l.Name())
	}
	if want := "cpp/1@" + treesitter.Parser; l.Version() != want {
		t.Errorf("Version = %q, want %q", l.Version(), want)
	}
	same(t, "Extensions", l.Extensions(), []string{
		".cpp", ".cc", ".cxx", ".c++", ".hpp", ".hh", ".hxx", ".h++", ".h", ".inl", ".ipp", ".tpp",
	})
	r, err := l.File("src/m.cpp", "class K { };\n")
	if err != nil || r.Path != "src/m.cpp" || r.Language != "cpp" || len(r.Nodes) != 2 {
		t.Errorf("File = %d nodes at %q (%q), %v; want the file and K at src/m.cpp", len(r.Nodes), r.Path, r.Language, err)
	}
}

func TestTypesAndFunctionsGetNodes(t *testing.T) {
	const src = `class C : public B {
public:
    C() {}
    ~C() {}
    C& operator+=(const C& o) { return *this; }
    template <typename T> void tm(T t) {}
    struct In { int x() { return 0; } };
    void decl();
    C(const C&) = default;
private:
    void hidden() {}
};
struct S { void m() {} private: int n() { return 1; } };
union U { int a; };
enum class E { A };
void free_f() {}
static void local_f() {}
template <typename T>
T tf(T x) { return x; }
void over(int) {}
void over(double) {}
void* operator new(unsigned long n) { return 0; }
struct Fwd;
typedef struct { int a; } Anon;
`
	r := extractSrc(t, src)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	same(t, "nodes", views(r), []string{
		`m.cpp#C class owner= L1-L12 exported=true sig="class C : public B"`,
		`m.cpp#C.C method owner=C L3-L3 exported=true sig="C()"`,
		`m.cpp#C.~C method owner=C L4-L4 exported=true sig="~C()"`,
		`m.cpp#C.operator+= method owner=C L5-L5 exported=true sig="C& operator+=(const C& o)"`,
		`m.cpp#C.tm method owner=C L6-L6 exported=true sig="void tm(T t)"`,
		`m.cpp#C.In class owner=C L7-L7 exported=true sig="struct In"`,
		`m.cpp#C.In.x method owner=C.In L7-L7 exported=true sig="int x()"`,
		`m.cpp#C.hidden method owner=C L11-L11 exported=false sig="void hidden()"`,
		`m.cpp#S class owner= L13-L13 exported=true sig="struct S"`,
		`m.cpp#S.m method owner=S L13-L13 exported=true sig="void m()"`,
		`m.cpp#S.n method owner=S L13-L13 exported=false sig="int n()"`,
		`m.cpp#U class owner= L14-L14 exported=true sig="union U"`,
		`m.cpp#E class owner= L15-L15 exported=true sig="enum class E"`,
		`m.cpp#free_f function owner= L16-L16 exported=true sig="void free_f()"`,
		`m.cpp#local_f function owner= L17-L17 exported=false sig="static void local_f()"`,
		`m.cpp#tf function owner= L18-L19 exported=true sig="T tf(T x)"`,
		`m.cpp#over function owner= L20-L20 exported=true sig="void over(int)"`,
		`m.cpp#over~2 function owner= L21-L21 exported=true sig="void over(double)"`,
		`m.cpp#operator new function owner= L22-L22 exported=true sig="void* operator new(unsigned long n)"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.cpp", "m.cpp#C"),
		contains("m.cpp#C", "m.cpp#C.C"),
		contains("m.cpp#C", "m.cpp#C.~C"),
		contains("m.cpp#C", "m.cpp#C.operator+="),
		contains("m.cpp#C", "m.cpp#C.tm"),
		contains("m.cpp#C", "m.cpp#C.In"),
		contains("m.cpp#C.In", "m.cpp#C.In.x"),
		contains("m.cpp#C", "m.cpp#C.hidden"),
		contains("m.cpp", "m.cpp#S"),
		contains("m.cpp#S", "m.cpp#S.m"),
		contains("m.cpp#S", "m.cpp#S.n"),
		contains("m.cpp", "m.cpp#U"),
		contains("m.cpp", "m.cpp#E"),
		contains("m.cpp", "m.cpp#free_f"),
		contains("m.cpp", "m.cpp#local_f"),
		contains("m.cpp", "m.cpp#tf"),
		contains("m.cpp", "m.cpp#over"),
		contains("m.cpp", "m.cpp#over~2"),
		contains("m.cpp", "m.cpp#operator new"),
	})
	// A template's header belongs to the definition's text.
	if body := r.Nodes[16].BodyText; !strings.HasPrefix(body, "template <typename T> T tf") {
		t.Errorf("tf body = %q, want it to start at the template header", body)
	}
}

func TestOutOfClassMethod(t *testing.T) {
	r := extractSrc(t, `void ns::A::m() {}
A::A() : x(1) {}
A::~A() {}
bool A::operator==(const A& o) const { return true; }
template <typename T> void Box<T>::put(T v) {}
namespace outer {
void ns::A::n() {}
}
A::operator bool() const { return true; }
struct V { operator int() { return 0; } };
`)
	same(t, "nodes", views(r), []string{
		`m.cpp#A.m method owner=A L1-L1 exported=true sig="void ns::A::m()"`,
		`m.cpp#A.A method owner=A L2-L2 exported=true sig="A::A()"`,
		`m.cpp#A.~A method owner=A L3-L3 exported=true sig="A::~A()"`,
		`m.cpp#A.operator== method owner=A L4-L4 exported=true sig="bool A::operator==(const A& o) const"`,
		`m.cpp#Box.put method owner=Box L5-L5 exported=true sig="void Box<T>::put(T v)"`,
		`m.cpp#A.n method owner=A L7-L7 exported=true sig="void ns::A::n()"`,
		`m.cpp#A.operator bool method owner=A L9-L9 exported=true sig="A::operator bool() const"`,
		`m.cpp#V class owner= L10-L10 exported=true sig="struct V"`,
		`m.cpp#V.operator int method owner=V L10-L10 exported=true sig="operator int()"`,
	})
	want := map[model.NodeID]string{
		"m.cpp#A.m": "ns::A", "m.cpp#A.A": "A", "m.cpp#A.~A": "A", "m.cpp#A.operator==": "A",
		"m.cpp#Box.put": "Box", "m.cpp#A.n": "outer::ns::A", "m.cpp#A.operator bool": "A", "m.cpp#V": "",
	}
	if !reflect.DeepEqual(r.Enclosing, want) {
		t.Errorf("Enclosing = %q, want %q", r.Enclosing, want)
	}
}

func TestNamespacesFillEnclosing(t *testing.T) {
	r := extractSrc(t, `class G { void m() {} };
namespace a { namespace b { struct K { struct N {}; }; } }
namespace c::d { void f() {} }
namespace { class Hidden {}; void anon() {} }
void top() {}
`)
	same(t, "nodes", views(r), []string{
		`m.cpp#G class owner= L1-L1 exported=true sig="class G"`,
		`m.cpp#G.m method owner=G L1-L1 exported=false sig="void m()"`,
		`m.cpp#K class owner= L2-L2 exported=true sig="struct K"`,
		`m.cpp#K.N class owner=K L2-L2 exported=true sig="struct N"`,
		`m.cpp#f function owner= L3-L3 exported=true sig="void f()"`,
		`m.cpp#Hidden class owner= L4-L4 exported=false sig="class Hidden"`,
		`m.cpp#anon function owner= L4-L4 exported=false sig="void anon()"`,
		`m.cpp#top function owner= L5-L5 exported=true sig="void top()"`,
	})
	want := map[model.NodeID]string{
		"m.cpp#G": "", "m.cpp#K": "a::b", "m.cpp#K.N": "a::b", "m.cpp#f": "c::d",
		"m.cpp#Hidden": "", "m.cpp#anon": "", "m.cpp#top": "",
	}
	if !reflect.DeepEqual(r.Enclosing, want) {
		t.Errorf("Enclosing = %q, want %q", r.Enclosing, want)
	}

	// Without a definition there is nothing to name, and the map stays
	// nil as it reads back from the extract cache.
	if r := extractSrc(t, "namespace a { int x; }\n"); r.Enclosing != nil {
		t.Errorf("Enclosing = %q, want nil", r.Enclosing)
	}
}

func TestMacroLinesAreMasked(t *testing.T) {
	const src = `class GDExample : public Sprite2D {
    GDCLASS(GDExample, Sprite2D)
    Q_OBJECT
public:
    void m() {}
};
`
	r := extractSrc(t, src)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	same(t, "nodes", views(r), []string{
		`m.cpp#GDExample class owner= L1-L6 exported=true sig="class GDExample : public Sprite2D"`,
		`m.cpp#GDExample.m method owner=GDExample L5-L5 exported=true sig="void m()"`,
	})
	if body := r.Nodes[1].BodyText; !strings.Contains(body, "GDCLASS(GDExample, Sprite2D) Q_OBJECT") {
		t.Errorf("class body = %q, want the macro lines from the original", body)
	}
	if r.Nodes[0].BodyHash != extract.Hash(src) {
		t.Error("file hash is not the original's")
	}

	// A macro line with a ';', a '{' or a '}' in it, or a lowercase one,
	// is code and stays.
	r = extractSrc(t, "struct K {\n    FIELD(int; x)\n    void m() {}\n};\n")
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the unmasked line to break the parse")
	}
}

func TestAllmanMacroIsKept(t *testing.T) {
	r := extractSrc(t, `TEST_F(Suite, Name)
{
    run();
}
TEST(Suite, Other) {
    run();
}
`)
	same(t, "nodes", ids(r), []model.NodeID{"m.cpp#TEST_F", "m.cpp#TEST"})
}

func TestDefinitionUnderBrokenAncestor(t *testing.T) {
	// The unknown macro breaks the class head: the parser keeps m whole
	// under an ERROR, and m has no class to belong to.
	r := extractSrc(t, `class Broken : public Base {
    some_macro(x)
public:
    void m() { g(); }
    int n() { return h(); }
};
void tail() { k(); }
`)
	same(t, "nodes", views(r), []string{
		// The ERROR took the return type along.
		`m.cpp#m function owner= L4-L4 exported=true sig="m()"`,
		`m.cpp#tail function owner= L7-L7 exported=true sig="void tail()"`,
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.cpp#m", "g", ""),
		call("m.cpp#tail", "k", ""),
	})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the broken class counted")
	}

	// A definition holding an ERROR gets no node, and its calls count for
	// nobody.
	r = extractSrc(t, "struct K {\n  int i = g();\n  T* p = new T();\n  void ok() {}\n  void bad() { f(; ) }\n};\n")
	same(t, "nodes", ids(r), []model.NodeID{"m.cpp#K.ok"})
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)
}

func TestIfConstevalDamage(t *testing.T) {
	// The parser stops at C++23's if consteval and drops the rest of the
	// file; reading the rest again brings back what follows the function.
	const ce = `constexpr int ce(int x) {
    if consteval {
        return 1;
    } else {
        return x;
    }
}
`
	r := extractSrc(t, "void before() {}\n"+ce+"void after1() { f(); }\nvoid after2() {}\n")
	same(t, "nodes", views(r), []string{
		`m.cpp#before function owner= L1-L1 exported=true sig="void before()"`,
		`m.cpp#after1 function owner= L9-L9 exported=true sig="void after1()"`,
		`m.cpp#after2 function owner= L10-L10 exported=true sig="void after2()"`,
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.cpp#after1", "f", "")})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the cut counted")
	}

	// A namespace open at the cut is lost: what follows names none, and its
	// closing brace breaks the definition before it.
	r = extractSrc(t, "namespace n {\n"+ce+"void inside() {}\nvoid last() {}\n}\nvoid after() {}\n")
	same(t, "nodes", ids(r), []model.NodeID{"m.cpp#inside", "m.cpp#after"})
	want := map[model.NodeID]string{"m.cpp#inside": "", "m.cpp#after": ""}
	if !reflect.DeepEqual(r.Enclosing, want) {
		t.Errorf("Enclosing = %q, want %q", r.Enclosing, want)
	}
}

func TestTemplateAliasDoesNotEndTheTree(t *testing.T) {
	// gotreesitter ends the tree at an alias of a nested template's member;
	// its right side is masked, and the file reads to the end.
	const src = `template <typename T>
struct S {
    using X = typename A<B<T>>::type;
    using Y =
        A<B<T> >::type;
    void m() {}
};
void after() {}
`
	r := extractSrc(t, src)
	same(t, "nodes", ids(r), []model.NodeID{"m.cpp#S", "m.cpp#S.m", "m.cpp#after"})
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	if len(r.Nodes) < 2 || !strings.Contains(r.Nodes[1].BodyText, "using X = typename A<B<T>>::type;") {
		t.Errorf("nodes = %+v, want S with the alias from the original", r.Nodes)
	}
}

func TestEmptyFile(t *testing.T) {
	r := extractSrc(t, "")
	same(t, "nodes", ids(r), nil)
}

func TestIncludesBecomeImports(t *testing.T) {
	r := extractSrc(t, "#include \"x.h\"\n#include <a/b.hpp>\n#include CONFIG_H\n")
	same(t, "imports", r.Imports, []extract.Import{{Path: "x.h"}, {Path: "a/b.hpp", System: true}})
	same(t, "import edges", edgesOf(r, model.RelationImports), []extract.RawEdge{
		{Source: "m.cpp", Relation: model.RelationImports, Specifier: "x.h", File: "m.cpp"},
		{Source: "m.cpp", Relation: model.RelationImports, Specifier: "a/b.hpp", File: "m.cpp"},
	})
}

func TestUsingNamespaceBecomesImport(t *testing.T) {
	r := extractSrc(t, "using namespace a::b;\nnamespace n { using namespace c; }\nusing std::string;\nusing namespace ::d;\n")
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "a::b", Alias: "namespace"}, {Path: "c", Alias: "namespace"}, {Path: "d", Alias: "namespace"},
	})
	same(t, "import edges", edgesOf(r, model.RelationImports), nil)
}

func TestCallsBelongToInnermostFunction(t *testing.T) {
	r := extractSrc(t, `int g = init();
struct K {
    int f = field();
    void m() {
        f();
        a::b::f();
        ::top();
        obj.f();
        p->f();
        a.b->f();
        this->f();
        new T();
        new a::T<int>(arg());
        tf<int>();
        obj.template tm<int>();
        get().f();
        get().x.f();
        decltype(v)::f();
        a::b.f();
        auto l = [] { in_lambda(); };
    }
};
void K::n() { this->m(); }
`)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.cpp", "init", ""),
		call("m.cpp#K", "field", ""),
		call("m.cpp#K.m", "f", ""),
		call("m.cpp#K.m", "f", "a::b"),
		call("m.cpp#K.m", "top", ""),
		access("m.cpp#K.m", "f", "obj"),
		access("m.cpp#K.m", "f", "p"),
		access("m.cpp#K.m", "f", "a.b"),
		call("m.cpp#K.m", "f", "this"),
		construct("m.cpp#K.m", "T", ""),
		construct("m.cpp#K.m", "T", "a"),
		call("m.cpp#K.m", "arg", ""),
		call("m.cpp#K.m", "tf", ""),
		access("m.cpp#K.m", "tm", "obj"),
		call("m.cpp#K.m", "get", ""),
		call("m.cpp#K.m", "get", ""),
		access("m.cpp#K.m", "f", "a::b"),
		call("m.cpp#K.m", "in_lambda", ""),
		call("m.cpp#K.n", "m", "this"),
	})
}

func TestBaseClauseBecomesExtends(t *testing.T) {
	r := extractSrc(t, `class K : public Base<int>, private ns::Other, virtual Q, protected a::T<int> {};
struct L : K {};
`)
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		extends("m.cpp#K", "Base", ""),
		extends("m.cpp#K", "Other", "ns"),
		extends("m.cpp#K", "Q", ""),
		extends("m.cpp#K", "T", "a"),
		extends("m.cpp#L", "K", ""),
	})
}

func TestGodotBindings(t *testing.T) {
	r := extractSrc(t, `void GDExample::_bind_methods() {
    ClassDB::bind_method(D_METHOD("get_speed"), &GDExample::get_speed);
    ClassDB::bind_static_method("GDExample", D_METHOD("make", "a"), &GDExample::make);
    ClassDB::bind_vararg_method(METHOD_FLAGS_DEFAULT, "va", &GDExample::va, mi);
    ADD_SIGNAL(MethodInfo("hit", PropertyInfo(Variant::INT, "n")));
    ClassDB::bind_method(D_METHOD(name), &GDExample::skipped);
    ClassDB::bind_static_method("GDExample", D_METHOD(name), &GDExample::skipped2);
    ClassDB::bind_method(D_METHOD("ptr"), fp);
    ADD_SIGNAL(info);
}
void other() {
    ClassDB::bind_method(D_METHOD("no"), &GDExample::no);
    ADD_SIGNAL(MethodInfo("nope"));
}
`)
	ref := func(member model.Kind, spec, receiver, name string) extract.RawEdge {
		return extract.RawEdge{
			Source: "m.cpp#GDExample._bind_methods", Relation: model.RelationReferences,
			Member: member, Specifier: spec, Receiver: receiver, Name: name, File: "m.cpp",
		}
	}
	same(t, "references", edgesOf(r, model.RelationReferences), []extract.RawEdge{
		ref("godot.method", "get_speed", "GDExample", "get_speed"),
		ref("godot.method", "make", "GDExample", "make"),
		ref("godot.method", "va", "GDExample", "va"),
		ref("godot.signal", "hit", "GDExample", ""),
	})
}

func TestGDCLASSOffersTheClass(t *testing.T) {
	r := extractSrc(t, `namespace godot {
class GDExample : public Sprite2D {
    GDCLASS(GDExample, Sprite2D)
    class Inner : public Object {
        GDCLASS(Inner, Object)
    };
};
class Plain {};
class Semi : public Node {
    GDCLASS(Semi, Node);
};
class Broken : public Object {
    GDCLASS(Broken, Object)
    void bad() { f(; ) }
};
}
`)
	// A nested class claims its own line; a class without a node offers
	// nothing.
	same(t, "exports", r.Exports, []extract.Export{
		{Namespace: extract.NamespaceGodotClass, Name: "Inner", Target: "m.cpp#GDExample.Inner"},
		{Namespace: extract.NamespaceGodotClass, Name: "GDExample", Target: "m.cpp#GDExample"},
		{Namespace: extract.NamespaceGodotClass, Name: "Semi", Target: "m.cpp#Semi"},
	})
}
