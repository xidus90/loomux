package resolve_test

import (
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/cpp"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// cppResults extracts a tree of C++ files, path to source, with the real
// extractor, handed over in reverse byte order: which file the resolver
// picks must not depend on it.
func cppResults(t *testing.T, tree map[string]string) []extract.Result {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var files []extract.Result
	for _, p := range paths {
		r, err := cpp.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	return files
}

// cppGraph resolves a tree of C++ files.
func cppGraph(t *testing.T, tree map[string]string) (*model.Graph, []resolve.Unresolved) {
	t.Helper()
	g, un := resolve.GraphWith(cppResults(t, tree), nil, stamp)
	return validated(t, g), un
}

func TestCppIncludes(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"src/a/m.cpp": "#include \"x.h\"\n#include \"a/y.h\"\n#include <a/y.h>\n#include <vector>\n#include <z.h>\n",
		// The quoted include finds its neighbour first.
		"src/a/x.h": "",
		"lib/x.h":   "",
		// Several hits: the one with the fewest segments.
		"inc/a/y.h":   "",
		"src/b/a/y.h": "",
		// A system include skips the file's own directory.
		"src/a/z.h": "",
		"lib/z.h":   "",
	})
	wantTargets(t, g, "src/a/m.cpp", model.RelationImports, "inc/a/y.h", "lib/z.h", "src/a/x.h")
	if e := edgeBetween(g, "src/a/m.cpp", "inc/a/y.h", model.RelationImports); e == nil || e.Line != 2 {
		t.Errorf("two includes of one file make one edge at the first line, got %+v", e)
	}
}

func TestCppIncludeAmbiguousSuffixGivesNoEdge(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"q/m.cpp":       "#include \"a/x.h\"\n",
		"include/a/x.h": "",
		"src/a/x.h":     "",
	})
	wantTargets(t, g, "q/m.cpp", model.RelationImports)
}

func TestCppTypeResolutionOrder(t *testing.T) {
	for _, c := range []struct {
		name       string
		tree       map[string]string
		from, want string
	}{
		{"nested type of the enclosing type", map[string]string{
			"a.h": "namespace N { struct Outer { struct B {}; struct A : B {}; }; }\n",
			"b.h": "namespace N { struct B {}; }\n",
		}, "a.h#Outer.A", "a.h#Outer.B"},
		{"own namespace", map[string]string{
			"a.h": "using namespace U;\nnamespace X::Y { struct A : B {}; }\n",
			"b.h": "namespace X::Y { struct B {}; }\n",
			"c.h": "namespace X { struct B {}; }\n",
			"u.h": "namespace U { struct B {}; }\n",
		}, "a.h#A", "b.h#B"},
		{"outer namespace", map[string]string{
			"a.h": "using namespace U;\nnamespace X::Y { struct A : B {}; }\n",
			"c.h": "namespace X { struct B {}; }\n",
			"u.h": "namespace U { struct B {}; }\n",
		}, "a.h#A", "c.h#B"},
		{"using namespace", map[string]string{
			"a.h": "using namespace U;\nstruct A : B {};\n",
			"g.h": "struct B {};\n",
			"u.h": "namespace U { struct B {}; }\n",
		}, "a.h#A", "u.h#B"},
		{"using namespace of another file", map[string]string{
			"a.h": "struct A : B {};\n",
			"g.h": "struct B {};\n",
			"o.h": "using namespace U;\n",
			"u.h": "namespace U { struct B {}; }\n",
		}, "a.h#A", "g.h#B"},
		{"global namespace", map[string]string{
			"a.h": "namespace X { struct A : B {}; }\n",
			"g.h": "struct B {};\n",
		}, "a.h#A", "g.h#B"},
		{"qualified name", map[string]string{
			"a.h": "struct A : X::Y::B {};\n",
			"b.h": "namespace X::Y { struct B {}; }\n",
			"g.h": "struct B {};\n",
		}, "a.h#A", "b.h#B"},
	} {
		t.Run(c.name, func(t *testing.T) {
			g, un := cppGraph(t, c.tree)
			wantTargets(t, g, model.NodeID(c.from), model.RelationExtends, c.want)
			wantUnresolved(t, un)
		})
	}
}

func TestCppUnresolvedBases(t *testing.T) {
	_, un := cppGraph(t, map[string]string{
		"a.h": "using namespace U;\nusing namespace V;\nstruct A : B, std::exception {};\n",
		"u.h": "namespace U { struct B {}; }\n",
		"v.h": "namespace V { struct B {}; }\n",
	})
	wantUnresolved(t, un, "B ambiguous", "std::exception none")
}

// The same class name lives in two namespaces and two headers; a method
// defined outside its class finds the right one over its qualifier.
func TestCppOutOfClassMethodFindsItsHeaderClass(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"x/b.h": "namespace x { class B { public: void h() {} }; }\n",
		"y/b.h": "namespace y { class B { public: void h() {} }; }\n",
		"a/b.h": "namespace a { class B { public: void h() {} }; }\n",
		"x/b.cpp": "#include \"b.h\"\nnamespace x {\nvoid B::m() {\n  h();\n  this->k();\n}\nvoid B::k() {}\n}\n" +
			"void x::B::n() {\n  this->h();\n}\n",
		"y/b.cpp": "using namespace y;\nvoid B::m() {\n  this->h();\n}\n",
		// No n::a::B: the qualifier is read inside n, and a::B is global.
		"n/b.cpp": "namespace n {\nvoid a::B::m() {\n  h();\n}\n}\n",
	})
	wantCalls(t, g, "x/b.cpp#B.m", "x/b.cpp#B.k extracted", "x/b.h#B.h extracted")
	wantCalls(t, g, "x/b.cpp#B.n", "x/b.h#B.h extracted")
	wantCalls(t, g, "y/b.cpp#B.m", "y/b.h#B.h extracted")
	wantCalls(t, g, "n/b.cpp#B.m", "a/b.h#B.h extracted")
}

func TestCppBasesAndCalls(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"a.h": "struct Base { void b() {} };\n" +
			"struct D : Base {\n  void f() {}\n  void m() {\n    f();\n    b();\n    this->b();\n    nowhere();\n  }\n};\n" +
			"struct P : Q { void m() { this->gone(); } };\nstruct Q : P {};\n" +
			"struct A { static void s() {} A() {} };\nstruct E {};\n" +
			"void f() {}\n",
		"n.cpp": "namespace a { void f() {} }\nnamespace a::b {\nvoid g() {\n  f();\n}\n}\n" +
			"namespace u { void uf() {} }\nusing namespace u;\n" +
			"namespace util { void z(); }\nvoid util::z() {}\n" +
			"void h(Thing obj) {\n  uf();\n  A::s();\n  a::f();\n  util::z();\n  obj.f();\n  new A();\n  new E();\n  new Ext();\n  A();\n  E();\n  this->f();\n}\n",
	})
	wantCalls(t, g, "a.h#D.m", "a.h#Base.b extracted", "a.h#Base.b extracted", "a.h#D.f extracted")
	wantCalls(t, g, "a.h#P.m")
	wantCalls(t, g, "n.cpp#g", "n.cpp#f extracted")
	wantCalls(t, g, "n.cpp#h",
		"a.h#A.A extracted", "a.h#A.A extracted", "a.h#A.s extracted", "a.h#E extracted", "a.h#E extracted",
		"n.cpp#f extracted", "n.cpp#uf extracted", "n.cpp#util.z extracted")
}

// A member call names an object, never a class or namespace, even when one
// of the same name is in scope.
func TestCppMemberCallStaysUnresolved(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"a.h":   "struct foo { static void bar() {} };\nnamespace p { void bar() {} }\n",
		"m.cpp": "void m(Foo foo, Q* p) {\n  foo.bar();\n  p->bar();\n  foo::bar();\n  p::bar();\n}\n",
	})
	wantCalls(t, g, "m.cpp#m", "a.h#bar extracted", "a.h#foo.bar extracted")
}

// A static function and anything in an anonymous namespace answer in their
// own file alone, and before a definition of another file.
func TestCppInternalLinkageStaysInItsFile(t *testing.T) {
	tree := map[string]string{
		"g.h": "void helper() {}\nstruct Impl { void run() {} };\n",
	}
	for _, f := range []string{"a.cpp", "b.cpp"} {
		tree[f] = "static void helper() {}\nnamespace {\nvoid anon() {}\nstruct Impl {\n  void run() {}\n  void go() { run(); }\n  struct In {};\n};\n}\n" +
			"void f() {\n  helper();\n  anon();\n  new Impl();\n  new Impl::In();\n}\n"
	}
	tree["c.cpp"] = "void f() {\n  helper();\n  anon();\n  new Impl();\n}\n"
	g, _ := cppGraph(t, tree)
	for _, f := range []string{"a.cpp", "b.cpp"} {
		wantCalls(t, g, model.NodeID(f+"#Impl.go"), f+"#Impl.run extracted")
		wantCalls(t, g, model.NodeID(f+"#f"), f+"#Impl extracted", f+"#Impl.In extracted", f+"#anon extracted", f+"#helper extracted")
	}
	wantCalls(t, g, "c.cpp#f", "g.h#Impl extracted", "g.h#helper extracted")
}

// A class whose body holds an ERROR gets no node; its whole methods keep
// it as their owner and belong to no type, while a whole class nested in it
// still is one.
func TestCppMethodOfABrokenClassBelongsToNoType(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"m.cpp": "struct K {\n  struct In {\n    void g() {}\n    void h() { this->g(); }\n  };\n" +
			"  void ok() { top(); }\n  void bad() { f(; ) }\n};\nvoid top() {}\n",
	})
	wantCalls(t, g, "m.cpp#K.In.h", "m.cpp#K.In.g extracted")
	wantCalls(t, g, "m.cpp#K.ok", "m.cpp#top extracted")
}

// A call inside namespace a finds a::helper before a file-static global
// one: the enclosing namespace is the nearer scope, whichever table holds it.
func TestCppEnclosingNamespaceBeatsTheFilesOwnGlobal(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"a.cpp": "static void helper() {}\nnamespace a {\nvoid run() {\n  helper();\n}\n}\n",
		"h.h":   "namespace a { void helper() {} }\n",
	})
	wantCalls(t, g, "a.cpp#run", "h.h#helper extracted")
}

// Two classes of one name in two namespaces of one file are two types, each
// with its own members.
func TestCppSameClassNameInTwoNamespacesOfOneFile(t *testing.T) {
	g, _ := cppGraph(t, map[string]string{
		"m.cpp": "namespace a { struct K { void f() {} void m() { this->f(); } }; }\n" +
			"namespace b { struct K { void f() {} void m() { this->f(); } }; }\n",
	})
	wantCalls(t, g, "m.cpp#K.m", "m.cpp#K.f extracted")
	wantCalls(t, g, "m.cpp#K.m~2", "m.cpp#K.f~2 extracted")
}

const gdexampleH = "#include <godot_cpp/classes/sprite2d.hpp>\nnamespace godot {\n" +
	"class GDExample : public Sprite2D {\n  GDCLASS(GDExample, Sprite2D)\nprotected:\n  static void _bind_methods();\n" +
	"public:\n  void spin() {}\n};\n}\n"

const gdexampleCpp = "#include \"gdexample.h\"\nusing namespace godot;\n" +
	"void GDExample::_bind_methods() {\n" +
	"  ClassDB::bind_method(D_METHOD(\"spin_it\"), &GDExample::spin);\n" +
	"  ClassDB::bind_method(D_METHOD(\"gone\"), &GDExample::missing);\n" +
	"  ClassDB::bind_method(D_METHOD(\"far\"), &Nope::far);\n" +
	"  ADD_SIGNAL(MethodInfo(\"hit\"));\n}\n" +
	"void Unknown::_bind_methods() {\n  ADD_SIGNAL(MethodInfo(\"lost\"));\n}\n"

// A binding is a references edge to the bound method and an offer of the
// class's member; a scene asking for the member on the class reaches it.
func TestCppBindingsOfferClassMembers(t *testing.T) {
	files := cppResults(t, map[string]string{"src/gdexample.h": gdexampleH, "src/gdexample.cpp": gdexampleCpp})
	scene := extract.Result{Path: "game/main.tscn", Language: "gdscript", Nodes: []model.Node{extract.FileNode("game/main.tscn", "", nil)}}
	for _, b := range []struct {
		ns          extract.Namespace
		name, class string
	}{
		{extract.NamespaceGodotMethod, "spin_it", "GDExample"},
		{extract.NamespaceGodotMethod, "gone", "GDExample"},
		{extract.NamespaceGodotMethod, "far", "Nope"},
		{extract.NamespaceGodotSignal, "hit", "GDExample"},
		{extract.NamespaceGodotSignal, "lost", "Unknown"},
	} {
		scene.Bridges = append(scene.Bridges, extract.Bridge{
			Source: "game/main.tscn", Relation: model.RelationReferences,
			Namespace: b.ns, Name: b.name, Class: b.class, File: "game/main.tscn", Line: 3,
		})
	}
	project := extract.Result{Path: "game/project.godot", Language: "gdscript", Nodes: []model.Node{extract.FileNode("game/project.godot", "", nil)}}
	g, un := resolve.GraphWith(append(files, scene, project), nil, stamp)
	validated(t, g)
	wantEdges(t, g,
		"src/gdexample.cpp -imports-> src/gdexample.h extracted",
		"src/gdexample.cpp#GDExample._bind_methods -references-> src/gdexample.h#GDExample.spin extracted",
		"game/main.tscn -references-> src/gdexample.h#GDExample.spin extracted",
		"game/main.tscn -references-> src/gdexample.h#GDExample extracted",
	)
	wantUnresolved(t, un, "Sprite2D none")
}
