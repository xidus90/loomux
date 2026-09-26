package python_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/python"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, src string) extract.Result {
	t.Helper()
	r, err := python.File("m.py", src)
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

// node is the node with an id, or a failed test.
func node(t *testing.T, r extract.Result, id model.NodeID) model.Node {
	t.Helper()
	for _, n := range r.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("no node %s among %v", id, views(r))
	return model.Node{}
}

func edgesOf(r extract.Result, rel model.Relation) []extract.RawEdge {
	var out []extract.RawEdge
	for _, e := range r.Edges {
		if e.Relation == rel {
			out = append(out, e)
		}
	}
	return out
}

func contains(src, dst model.NodeID) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: "m.py"}
}

func call(src model.NodeID, name, owner, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, Owner: owner, Receiver: receiver, File: "m.py"}
}

func imports(spec string) extract.RawEdge {
	return extract.RawEdge{Source: "m.py", Relation: model.RelationImports, Specifier: spec, File: "m.py"}
}

func extends(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationExtends, Name: name, Receiver: receiver, File: "m.py"}
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

func TestLanguageDescribesPython(t *testing.T) {
	var l extract.Language = python.Language{}
	if l.Name() != "python" {
		t.Errorf("Name = %q, want python", l.Name())
	}
	if want := "python/1@" + treesitter.Parser; l.Version() != want {
		t.Errorf("Version = %q, want %q", l.Version(), want)
	}
	same(t, "Extensions", l.Extensions(), []string{".py"})
	r, err := l.File("pkg/m.py", "def f():\n    pass\n")
	if err != nil || r.Path != "pkg/m.py" || len(r.Nodes) != 2 {
		t.Errorf("File = %d nodes at %q, %v; want the file and f at pkg/m.py", len(r.Nodes), r.Path, err)
	}
}

func TestTopLevelFunctionAndClass(t *testing.T) {
	const src = `"""Module doc."""
import os


def f(a, b):
    return a


class C(Base):
    x = 1
`
	r := extractSrc(t, src)
	if r.Path != "m.py" || r.Language != "python" || r.Package != "" || r.ParseErrors != 0 {
		t.Errorf("Result header = %q %q %q %d, want m.py python, no package, no parse errors", r.Path, r.Language, r.Package, r.ParseErrors)
	}
	same(t, "nodes", views(r), []string{
		`m.py file owner= L1-L11 exported=true sig=""`,
		`m.py#f function owner= L5-L6 exported=true sig="def f(a, b)"`,
		`m.py#C class owner= L9-L10 exported=true sig="class C(Base)"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.py", "m.py#f"),
		contains("m.py", "m.py#C"),
	})
	// The file node keeps what no definition covers.
	if got := r.Nodes[0].BodyText; got != `"""Module doc.""" import os` {
		t.Errorf("file BodyText = %q, want the docstring and the import", got)
	}
}

func TestMethodOwnerAndID(t *testing.T) {
	const src = `class C:
    def m(self):
        pass

    async def n(self):
        pass

def m():
    pass
`
	r := extractSrc(t, src)
	same(t, "nodes", views(r)[1:], []string{
		`m.py#C class owner= L1-L6 exported=true sig="class C"`,
		`m.py#C.m method owner=C L2-L3 exported=true sig="def m(self)"`,
		`m.py#C.n method owner=C L5-L6 exported=true sig="async def n(self)"`,
		`m.py#m function owner= L8-L9 exported=true sig="def m()"`,
	})
	if got := node(t, r, "m.py#C.m").Name; got != "m" {
		t.Errorf("method Name = %q, want the bare m: a call names it so", got)
	}
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.py", "m.py#C"),
		contains("m.py#C", "m.py#C.m"),
		contains("m.py#C", "m.py#C.n"),
		contains("m.py", "m.py#m"),
	})
}

func TestDecoratorInSpanNotInSignature(t *testing.T) {
	const src = `@app.route("/x")
def f(a):
    return a


@register(kind="c")
class C:
    @staticmethod
    @cache(size=2)
    def m(a):
        return helper(a)
`
	r := extractSrc(t, src)
	same(t, "nodes", views(r)[1:], []string{
		`m.py#f function owner= L1-L3 exported=true sig="def f(a)"`,
		`m.py#C class owner= L6-L11 exported=true sig="class C"`,
		`m.py#C.m method owner=C L8-L11 exported=true sig="def m(a)"`,
	})
	if got := node(t, r, "m.py#f").BodyText; !strings.HasPrefix(got, `@app.route("/x") def f(a):`) {
		t.Errorf("f BodyText = %q, want it to start with the decorator", got)
	}
	if got := node(t, r, "m.py#C.m").BodyText; !strings.HasPrefix(got, "@staticmethod @cache(size=2) def m(a):") {
		t.Errorf("C.m BodyText = %q, want it to start with both decorators", got)
	}
	// A decorator is part of the definition, and so is the call in it.
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.py#f", "route", "", "app"),
		call("m.py#C", "register", "", ""),
		call("m.py#C.m", "cache", "", ""),
		call("m.py#C.m", "helper", "", ""),
	})
}

func TestSignatureEndsAtTheHeaderColon(t *testing.T) {
	for src, want := range map[string]string{
		// A comment after the colon is a child of the definition ahead of its
		// block; it stays out.
		"def g(a):\n    # leading comment\n    return a\n": "def g(a)",
		"def f(a):  # noqa: E501\n    return a\n":          "def f(a)",
		"class C:  # pragma: no cover\n    pass\n":         "class C",
		// Deeper colons -- in an annotation, a default, a lambda -- never win.
		"def h(a: int = {1: 2}) -> dict[str, int]:  # c\n    pass\n": "def h(a: int = {1: 2}) -> dict[str, int]",
		"class D(B, metaclass=M):  # c\n    pass\n":                  "class D(B, metaclass=M)",
		"def f(a=lambda: 1):\n    pass\n":                            "def f(a=lambda: 1)",
		// The signature starts at the definition, which holds async.
		"async def f(a):\n    pass\n": "async def f(a)",
	} {
		r := extractSrc(t, src)
		if len(r.Nodes) != 2 {
			t.Errorf("%q: %d nodes, want the file and one definition", src, len(r.Nodes))
			continue
		}
		if got := r.Nodes[1].Signature; got != want {
			t.Errorf("Signature of %q = %q, want %q", src, got, want)
		}
	}
}

func TestNestedDefHasNoNode(t *testing.T) {
	const src = `def outer():
    def inner():
        helper()
    class Local:
        def m(self):
            other()
    return inner()
`
	r := extractSrc(t, src)
	same(t, "nodes", views(r)[1:], []string{
		`m.py#outer function owner= L1-L7 exported=true sig="def outer()"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{contains("m.py", "m.py#outer")})
	// Everything nested belongs to the body of outer, and so do its calls.
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.py#outer", "helper", "", ""),
		call("m.py#outer", "other", "", ""),
		call("m.py#outer", "inner", "", ""),
	})
}

func TestClassInClassHasNoNode(t *testing.T) {
	const src = `class Outer:
    class Inner:
        def m(self):
            self.foo()

    def run(self):
        class Local:
            def m(self):
                self.baz()
        self.bar()
`
	r := extractSrc(t, src)
	same(t, "nodes", views(r)[1:], []string{
		`m.py#Outer class owner= L1-L10 exported=true sig="class Outer"`,
		`m.py#Outer.run method owner=Outer L6-L10 exported=true sig="def run(self)"`,
	})
	// In a method of Inner or Local, self is an Inner or a Local and not an
	// Outer: the call keeps its receiver as written and no owner. It belongs
	// to the innermost definition with a node, Outer or Outer.run.
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.py#Outer", "foo", "", "self"),
		call("m.py#Outer.run", "baz", "", "self"),
		call("m.py#Outer.run", "bar", "Outer", ""),
	})
}

func TestConditionalDefIsModuleLevel(t *testing.T) {
	// A definition under a module-level if, try or with -- nested in any
	// combination of them -- binds its name in the module and is a function
	// or a class like one directly in it. A repeated name gets an ordinal. A
	// def inside such a function still gets none, and a call in those blocks
	// outside every definition is the file's.
	const src = `import sys

if sys.platform == "win32":
    def pick():
        choose()
elif sys.platform == "darwin":
    def pick():
        pass
else:
    class K:
        def m(self):
            pass

try:
    def f():
        pass
except ImportError:
    def f():
        pass
finally:
    cleanup()

with open("x") as fh:
    class W:
        pass

if DEBUG:
    try:
        def outer():
            def inner():
                pass
    except OSError:
        pass
`
	r := extractSrc(t, src)
	same(t, "nodes", views(r), []string{
		`m.py file owner= L1-L34 exported=true sig=""`,
		`m.py#pick function owner= L4-L5 exported=true sig="def pick()"`,
		`m.py#pick~2 function owner= L7-L8 exported=true sig="def pick()"`,
		`m.py#K class owner= L10-L12 exported=true sig="class K"`,
		`m.py#K.m method owner=K L11-L12 exported=true sig="def m(self)"`,
		`m.py#f function owner= L15-L16 exported=true sig="def f()"`,
		`m.py#f~2 function owner= L18-L19 exported=true sig="def f()"`,
		`m.py#W class owner= L24-L25 exported=true sig="class W"`,
		`m.py#outer function owner= L29-L31 exported=true sig="def outer()"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.py", "m.py#pick"),
		contains("m.py", "m.py#pick~2"),
		contains("m.py", "m.py#K"),
		contains("m.py#K", "m.py#K.m"),
		contains("m.py", "m.py#f"),
		contains("m.py", "m.py#f~2"),
		contains("m.py", "m.py#W"),
		contains("m.py", "m.py#outer"),
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.py#pick", "choose", "", ""),
		call("m.py", "cleanup", "", ""),
		call("m.py", "open", "", ""),
	})
}

func TestLoopDefHasNoNode(t *testing.T) {
	// Only if, try and with count as module level; a def in a loop body
	// belongs to the loop, and its calls to the file.
	r := extractSrc(t, "for i in range(2):\n    def loop():\n        work()\n")
	same(t, "nodes", views(r)[1:], nil)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.py", "range", "", ""),
		call("m.py", "work", "", ""),
	})
}

func TestRecoveredDefinitionIsSkipped(t *testing.T) {
	// `def n(self)` lacks its colon. The parser rebuilds it and the next def
	// into one definition named o, with an ERROR child ahead of the name and
	// a span over n's lines. It gets no node, and its call is credited to
	// nobody -- not to C.
	const src = "class C:\n    def m(self):\n        pass\n    def n(self)\n        pass\n    def o(self):\n        foo()\n"
	r := extractSrc(t, src)
	same(t, "nodes", views(r)[1:], []string{
		`m.py#C class owner= L1-L7 exported=true sig="class C"`,
		`m.py#C.m method owner=C L2-L3 exported=true sig="def m(self)"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.py", "m.py#C"),
		contains("m.py#C", "m.py#C.m"),
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the rejected text counted")
	}
}

func TestRecoveredDefinitionIsSkippedAtEveryLevel(t *testing.T) {
	cases := []struct {
		name, src string
		nodes     []string
		calls     []extract.RawEdge
	}{
		{"module level", "def n(a)\n    pass\ndef o(a):\n    foo()\n", nil, nil},
		// Nested, the rebuilt def has no node anyway; its call must not reach
		// the function around it.
		{"nested", "def outer():\n    def n(a)\n        pass\n    def o(a):\n        foo()\n    bar()\n",
			[]string{`m.py#outer function owner= L1-L6 exported=true sig="def outer()"`},
			[]extract.RawEdge{call("m.py#outer", "bar", "", "")}},
		// An ERROR anywhere in the header counts, not only a direct child:
		// `def n(self` without its ")" swallows the next def into its
		// parameters, and the definition would claim both.
		{"error in the parameters, module level", "def n(self\n    pass\ndef o(self):\n    foo()\n", nil, nil},
		{"error in the parameters, method", "class C:\n    def m(self):\n        pass\n    def n(self\n        pass\n    def o(self):\n        foo()\n",
			[]string{
				`m.py#C class owner= L1-L7 exported=true sig="class C"`,
				`m.py#C.m method owner=C L2-L3 exported=true sig="def m(self)"`,
			}, nil},
		{"error in the parameters, nested", "def outer():\n    def n(x\n        pass\n    def o(x):\n        foo()\n    bar()\n",
			[]string{`m.py#outer function owner= L1-L6 exported=true sig="def outer()"`},
			[]extract.RawEdge{call("m.py#outer", "bar", "", "")}},
		{"error inside the parameters", "def f(a, $):\n    foo()\n", nil, nil},
		// A decorator ahead of a def without its colon: the ERROR is a
		// sibling of the next def inside one decorated_definition, whose span
		// would claim both.
		{"decorated, module level", "@a\ndef n(x)\n    pass\n@b\ndef o(x):\n    foo()\n", nil, nil},
		{"decorated, method", "class C:\n    def m(self):\n        pass\n    @a\n    def n(x)\n        pass\n    @b\n    def o(x):\n        foo()\n",
			[]string{
				`m.py#C class owner= L1-L9 exported=true sig="class C"`,
				`m.py#C.m method owner=C L2-L3 exported=true sig="def m(self)"`,
			}, nil},
		{"decorated, nested", "def outer():\n    @a\n    def n(x)\n        pass\n    @b\n    def o(x):\n        foo()\n    bar()\n",
			[]string{`m.py#outer function owner= L1-L8 exported=true sig="def outer()"`},
			[]extract.RawEdge{call("m.py#outer", "bar", "", "")}},
		// The unclosed call of @a swallows the def after it; the ERROR sits
		// below a child of the decorated_definition, not as one, and still
		// counts.
		{"decorated, error deep in a decorator", "def m():\n    pass\n@a(x,\ndef n(self):\n    pass\n@b\ndef o():\n    foo()\n",
			[]string{`m.py#m function owner= L1-L2 exported=true sig="def m()"`}, nil},
		// Not rebuilt: a MISSING token is no ERROR, and an ERROR in the body
		// is not the header's; the definition keeps the extent the source
		// wrote, decorated or not.
		{"missing token", "def broken(:\n",
			[]string{`m.py#broken function owner= L1-L1 exported=true sig="def broken("`}, nil},
		{"error in the body", "def f():\n    x = $\n    foo()\n",
			[]string{`m.py#f function owner= L1-L3 exported=true sig="def f()"`},
			[]extract.RawEdge{call("m.py#f", "foo", "", "")}},
		{"error in the body, decorated", "@d\ndef f():\n    x = $\n    foo()\n",
			[]string{`m.py#f function owner= L1-L4 exported=true sig="def f()"`},
			[]extract.RawEdge{call("m.py#f", "foo", "", "")}},
	}
	for _, c := range cases {
		r := extractSrc(t, c.src)
		same(t, c.name+": nodes", views(r)[1:], c.nodes)
		same(t, c.name+": calls", edgesOf(r, model.RelationCalls), c.calls)
		if r.ParseErrors == 0 {
			t.Errorf("%s: ParseErrors = 0, want the error counted", c.name)
		}
	}
}

func TestExportedRule(t *testing.T) {
	const src = `def f(): pass
def _f(): pass
def __f(): pass
def __init__(): pass
def __(): pass
def ____(): pass
`
	r := extractSrc(t, src)
	got := map[string]bool{}
	for _, n := range r.Nodes[1:] {
		got[n.Name] = n.Exported
	}
	want := map[string]bool{
		"f": true, "_f": false, "__f": false,
		// A dunder is protocol, not private; "__" and "____" are no dunders.
		"__init__": true, "__": false, "____": false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Exported = %v, want %v", got, want)
	}
}

func TestImportForms(t *testing.T) {
	const src = `from __future__ import annotations
import a.b
import a.b as c
import x, y.z as w
from ..pkg.mod import t
from . import sib
from .. import up
from X import n, m as k
from X import *
from X import (p, q)


def f():
    import json
`
	r := extractSrc(t, src)
	// __future__ names a compiler feature, never a file; a function-local
	// import is still one of the file's.
	same(t, "Imports", r.Imports, []extract.Import{
		{Path: "a.b"},
		{Path: "a.b", Alias: "c"},
		{Path: "x"},
		{Path: "y.z", Alias: "w"},
		{Path: "..pkg.mod", Name: "t"},
		{Path: ".", Name: "sib"},
		{Path: "..", Name: "up"},
		{Path: "X", Name: "n"},
		{Path: "X", Name: "m", Alias: "k"},
		{Path: "X", Name: "*"},
		{Path: "X", Name: "p"},
		{Path: "X", Name: "q"},
		{Path: "json"},
	})
	// One edge per module, and one per imported name that may be a
	// submodule; a wildcard names none.
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{
		imports("a.b"),
		imports("a.b"),
		imports("x"),
		imports("y.z"),
		imports("..pkg.mod"), imports("..pkg.mod.t"),
		imports("."), imports(".sib"),
		imports(".."), imports("..up"),
		imports("X"), imports("X.n"), imports("X.m"),
		imports("X"),
		imports("X"), imports("X.p"), imports("X.q"),
		imports("json"),
	})
}

func TestImportPathIgnoresLineContinuation(t *testing.T) {
	// The path is read from its tokens, not its text: a backslash may break a
	// dotted name, and a comment may stand in a parenthesized list.
	const src = "import a.\\\n    b\nfrom X import (\n    c,  # the one\n    d,\n)\n"
	r := extractSrc(t, src)
	same(t, "Imports", r.Imports, []extract.Import{
		{Path: "a.b"},
		{Path: "X", Name: "c"},
		{Path: "X", Name: "d"},
	})
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{
		imports("a.b"),
		imports("X"), imports("X.c"), imports("X.d"),
	})
}

func TestCallShapes(t *testing.T) {
	cases := []struct {
		name, src string
		want      []extract.RawEdge
	}{
		{"bare", "foo()\n", []extract.RawEdge{call("m.py", "foo", "", "")}},
		{"receiver", "x.foo()\n", []extract.RawEdge{call("m.py", "foo", "", "x")}},
		{"chain", "a.b.foo()\n", []extract.RawEdge{call("m.py", "foo", "", "a.b")}},
		{"self and cls", "class C:\n    def m(self):\n        self.foo()\n\n    @classmethod\n    def k(cls):\n        cls.bar()\n", []extract.RawEdge{
			call("m.py#C.m", "foo", "C", ""),
			call("m.py#C.k", "bar", "C", ""),
		}},
		// A function nested in a method sees the method's self.
		{"self in a nested function", "class C:\n    def m(self):\n        def helper():\n            self.foo()\n", []extract.RawEdge{
			call("m.py#C.m", "foo", "C", ""),
		}},
		// A decorator of a class nested in a method runs in the method, where
		// self is still the method's.
		{"self in a decorator of a nested class", "class Outer:\n    def run(self):\n        @self.deco()\n        class Local:\n            pass\n", []extract.RawEdge{
			call("m.py#Outer.run", "deco", "Outer", ""),
		}},
		// Outside a method self is a name like any other.
		{"self in a function", "def f(self):\n    self.foo()\n", []extract.RawEdge{call("m.py#f", "foo", "", "self")}},
		// The outer call has no plain receiver and no edge; the inner call is a
		// bare call of its own.
		{"super", "super().foo()\n", []extract.RawEdge{call("m.py", "super", "", "")}},
		{"call of a call", "f()()\n", []extract.RawEdge{call("m.py", "f", "", "")}},
		{"subscript", "x[0].foo()\n", nil},
	}
	for _, c := range cases {
		same(t, c.name, edgesOf(extractSrc(t, c.src), model.RelationCalls), c.want)
	}
}

func TestCallSourceScopes(t *testing.T) {
	const src = `setup()


class C:
    registry = build()

    def m(self):
        def helper():
            inner()
        work()


def f():
    run()
`
	r := extractSrc(t, src)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.py", "setup", "", ""),
		call("m.py#C", "build", "", ""),
		call("m.py#C.m", "inner", "", ""),
		call("m.py#C.m", "work", "", ""),
		call("m.py#f", "run", "", ""),
	})
}

func TestMintedIDsReachEdges(t *testing.T) {
	// A repeated name gets an ordinal; every edge must name the id the node
	// was minted with, not one recomputed from the name.
	const src = `def f():
    one()


def f():
    two()


class C:
    def m(self):
        pass


class C:
    def m(self):
        three()
`
	r := extractSrc(t, src)
	var ids []model.NodeID
	for _, n := range r.Nodes[1:] {
		ids = append(ids, n.ID)
	}
	same(t, "ids", ids, []model.NodeID{"m.py#f", "m.py#f~2", "m.py#C", "m.py#C.m", "m.py#C~2", "m.py#C.m~2"})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.py", "m.py#f"),
		contains("m.py", "m.py#f~2"),
		contains("m.py", "m.py#C"),
		contains("m.py#C", "m.py#C.m"),
		contains("m.py", "m.py#C~2"),
		contains("m.py#C~2", "m.py#C.m~2"),
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.py#f", "one", "", ""),
		call("m.py#f~2", "two", "", ""),
		call("m.py#C.m~2", "three", "", ""),
	})
}

func TestExtendsForms(t *testing.T) {
	const src = `class A:
    pass


class B(A, mod.Base, pkg.sub.Mixin, Generic[T], *bases, metaclass=Meta):
    pass


class D(make_base()):
    pass
`
	r := extractSrc(t, src)
	// Names and attribute chains only: a subscript, a splat, a keyword and a
	// call name no class this extractor could point at.
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		extends("m.py#B", "A", ""),
		extends("m.py#B", "Base", "mod"),
		extends("m.py#B", "Mixin", "pkg.sub"),
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.py#D", "make_base", "", "")})
}

func TestDefinitionInErrorIsSkipped(t *testing.T) {
	// The stray ")" turns the class into an ERROR node with the method inside
	// it. Neither gets a node, and the call in the method is attributed to
	// nobody: the file did not make it.
	const src = "class C:\n    def m(self):\n        foo()\n    )\n"
	r := extractSrc(t, src)
	same(t, "nodes", views(r), []string{`m.py file owner= L1-L5 exported=true sig=""`})
	same(t, "edges", r.Edges, nil)
	if r.ParseErrors != 1 {
		t.Errorf("ParseErrors = %d, want 1", r.ParseErrors)
	}
}

// mixed holds every rule the CRLF and BOM tests must see unchanged.
const mixed = `@dec
def first(a):  # noqa
    return a


import os
from . import sib


class C(Base, mod.Mixin):
    """Doc."""

    @property
    def m(self):
        # leading comment
        return self.helper(os.path.join("a", "b"))
`

func TestCRLFMatchesLF(t *testing.T) {
	lf := extractSrc(t, mixed)
	crlf := extractSrc(t, strings.ReplaceAll(mixed, "\n", "\r\n"))
	same(t, "nodes", views(crlf), views(lf))
	if len(crlf.Nodes) != len(lf.Nodes) {
		t.FailNow()
	}
	for i, n := range crlf.Nodes {
		if n.BodyText != lf.Nodes[i].BodyText {
			t.Errorf("BodyText of %s = %q, want %q as with LF", n.ID, n.BodyText, lf.Nodes[i].BodyText)
		}
		if strings.Contains(n.Signature, "\r") {
			t.Errorf("Signature of %s = %q holds a carriage return", n.ID, n.Signature)
		}
	}
	same(t, "edges", crlf.Edges, lf.Edges)
	same(t, "Imports", crlf.Imports, lf.Imports)
	if crlf.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d with CRLF, want 0", crlf.ParseErrors)
	}
	if len(lf.Nodes) != 4 || len(edgesOf(lf, model.RelationCalls)) != 2 {
		t.Errorf("LF baseline = %d nodes and %d calls, want 4 and 2: the comparison would prove little", len(lf.Nodes), len(edgesOf(lf, model.RelationCalls)))
	}
}

func TestBOMIsParsed(t *testing.T) {
	plain := extractSrc(t, mixed)
	bom := extractSrc(t, "\uFEFF"+mixed)
	if bom.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d with a BOM, want 0", bom.ParseErrors)
	}
	// The first definition starts right after the BOM.
	if got := node(t, bom, "m.py#first").Signature; got != "def first(a)" {
		t.Errorf("Signature after a BOM = %q, want def first(a)", got)
	}
	node(t, bom, "m.py#C.m")
	same(t, "nodes", views(bom), views(plain))
	if len(bom.Nodes) != len(plain.Nodes) {
		t.FailNow()
	}
	// The file's hash covers the BOM; every definition's does not.
	for i, n := range bom.Nodes[1:] {
		if want := plain.Nodes[i+1]; n.BodyHash != want.BodyHash || n.BodyText != want.BodyText {
			t.Errorf("body of %s differs with a BOM", n.ID)
		}
	}
	same(t, "edges", bom.Edges, plain.Edges)
	same(t, "Imports", bom.Imports, plain.Imports)
}
