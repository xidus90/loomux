package gdscript_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/gdscript"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

const rel = "a/s.gd"

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, path, src string) extract.Result {
	t.Helper()
	r, err := gdscript.File(path, src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// views is every node as most tests compare it: everything but its body.
func views(r extract.Result) []string {
	var out []string
	for _, n := range r.Nodes {
		out = append(out, fmt.Sprintf("%s %s owner=%s %s exported=%v sig=%q", n.ID, n.Kind, n.Owner, n.Span, n.Exported, n.Signature))
	}
	return out
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
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: rel}
}

// same compares two lists and treats nil and empty as equal.
func same[T any](t *testing.T, what string, got, want []T) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s:\n got  %v\n want %v", what, got, want)
	}
}

const named = "@tool\nclass_name Sample\nextends Node\n\nsignal changed(value: int)\n\nstatic func make() -> Sample:\n\treturn null\n\nfunc _ready() -> void:\n\tpass\n\nclass Inner extends Node:\n\tfunc inner_fn():\n\t\tpass\n\n\tclass Deeper:\n\t\tfunc deep():\n\t\t\tpass\n"

func TestANamedScriptIsOneClassOverTheWholeFile(t *testing.T) {
	r := extractSrc(t, rel, named)
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L20 exported=true sig=""`,
		`a/s.gd#Sample class owner= L1-L19 exported=true sig="class_name Sample"`,
		`a/s.gd#Sample.changed signal owner=Sample L5-L5 exported=true sig="signal changed(value: int)"`,
		`a/s.gd#Sample.make method owner=Sample L7-L8 exported=true sig="static func make() -> Sample"`,
		`a/s.gd#Sample._ready method owner=Sample L10-L11 exported=false sig="func _ready() -> void"`,
		`a/s.gd#Sample.Inner class owner=Sample L13-L19 exported=true sig="class Inner extends Node"`,
		`a/s.gd#Sample.Inner.inner_fn method owner=Sample.Inner L14-L15 exported=true sig="func inner_fn()"`,
		`a/s.gd#Sample.Inner.Deeper class owner=Sample.Inner L17-L19 exported=true sig="class Deeper"`,
		`a/s.gd#Sample.Inner.Deeper.deep method owner=Sample.Inner.Deeper L18-L19 exported=true sig="func deep()"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("a/s.gd", "a/s.gd#Sample"),
		contains("a/s.gd#Sample", "a/s.gd#Sample.changed"),
		contains("a/s.gd#Sample", "a/s.gd#Sample.make"),
		contains("a/s.gd#Sample", "a/s.gd#Sample._ready"),
		contains("a/s.gd#Sample", "a/s.gd#Sample.Inner"),
		contains("a/s.gd#Sample.Inner", "a/s.gd#Sample.Inner.inner_fn"),
		contains("a/s.gd#Sample.Inner", "a/s.gd#Sample.Inner.Deeper"),
		contains("a/s.gd#Sample.Inner.Deeper", "a/s.gd#Sample.Inner.Deeper.deep"),
	})
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		{Source: "a/s.gd#Sample", Relation: model.RelationExtends, Name: "Node", File: rel},
		{Source: "a/s.gd#Sample.Inner", Relation: model.RelationExtends, Name: "Node", File: rel},
	})
	if r.Package != "Sample" || r.Language != "gdscript" || r.Path != rel || r.ParseErrors != 0 {
		t.Errorf("Package %q, Language %q, Path %q, ParseErrors %d", r.Package, r.Language, r.Path, r.ParseErrors)
	}
}

func TestAnUnnamedScriptHangsItsDefinitionsOnTheFile(t *testing.T) {
	r := extractSrc(t, rel, "extends Node\n\nsignal done\n\nfunc run():\n\tpass\n\nclass Helper:\n\tfunc help():\n\t\tpass\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L11 exported=true sig=""`,
		`a/s.gd#done signal owner= L3-L3 exported=true sig="signal done"`,
		`a/s.gd#run function owner= L5-L6 exported=true sig="func run()"`,
		`a/s.gd#Helper class owner= L8-L10 exported=true sig="class Helper"`,
		`a/s.gd#Helper.help method owner=Helper L9-L10 exported=true sig="func help()"`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("a/s.gd", "a/s.gd#done"),
		contains("a/s.gd", "a/s.gd#run"),
		contains("a/s.gd", "a/s.gd#Helper"),
		contains("a/s.gd#Helper", "a/s.gd#Helper.help"),
	})
	if r.Package != "" {
		t.Errorf("Package = %q, want none", r.Package)
	}
}

// Godot takes class_name anywhere at the top of the script, after extends
// too; the class still spans the file and owns every function.
func TestAClassNameAfterExtendsStillNamesTheScript(t *testing.T) {
	r := extractSrc(t, rel, "extends Node\nclass_name Late\nfunc f():\n\tpass\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L5 exported=true sig=""`,
		`a/s.gd#Late class owner= L1-L4 exported=true sig="class_name Late"`,
		`a/s.gd#Late.f method owner=Late L3-L4 exported=true sig="func f()"`,
	})
}

// gotreesitter v0.55.1 reads `class_name` with no name on its line as a
// class named after the first word of the next line: here "func". Godot
// wants the name on the keyword's line, and so does the extractor.
func TestAClassNameWithoutANameOnItsLineNamesNothing(t *testing.T) {
	r := extractSrc(t, rel, "class_name\nfunc f():\n\tpass\n")
	same(t, "nodes", views(r), []string{`a/s.gd file owner= L1-L4 exported=true sig=""`})
	if r.Package != "" || r.ParseErrors == 0 {
		t.Errorf("Package %q, ParseErrors %d; want none and some", r.Package, r.ParseErrors)
	}
}

// A function the parser rebuilt around text it rejected gets no node; the
// next one it swallowed goes with it, the one after that is whole.
func TestABrokenHeaderGetsNoNode(t *testing.T) {
	r := extractSrc(t, rel, "func broken(:\n\tpass\n\nfunc after():\n\tpass\n\nfunc ok():\n\tpass\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L9 exported=true sig=""`,
		`a/s.gd#ok function owner= L7-L8 exported=true sig="func ok()"`,
	})
	if r.ParseErrors == 0 {
		t.Error("ParseErrors = 0, want the rejected text counted")
	}
}

// func _init() is a constructor_definition without a name field; it is a
// method like any other, and what it calls and names is its own.
func TestAConstructorIsAMethodNamedInit(t *testing.T) {
	r := extractSrc(t, rel, "class_name X\n\nfunc _init(a: Foo):\n\tbar()\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L5 exported=true sig=""`,
		`a/s.gd#X class owner= L1-L4 exported=true sig="class_name X"`,
		`a/s.gd#X._init method owner=X L3-L4 exported=false sig="func _init(a: Foo)"`,
	})
	const init = "a/s.gd#X._init"
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{edge(init, model.RelationCalls, "bar", "", "")})
	var refs []string
	for _, e := range edgesOf(r, model.RelationReferences) {
		refs = append(refs, string(e.Source)+" "+e.Name)
	}
	same(t, "references", refs, []string{init + " a", init + " Foo"})
}

func TestAnEmptyScriptIsItsFileNode(t *testing.T) {
	r := extractSrc(t, rel, "")
	same(t, "nodes", views(r), []string{`a/s.gd file owner= L1-L1 exported=true sig=""`})
}

// Godot on Windows may save with CRLF; the nodes must not change with it.
func TestCRLFGivesTheSameNodes(t *testing.T) {
	lf := extractSrc(t, rel, named)
	crlf := extractSrc(t, rel, strings.ReplaceAll(named, "\n", "\r\n"))
	same(t, "nodes", views(crlf), views(lf))
	same(t, "edges", crlf.Edges, lf.Edges)
}

func TestLanguageDescribesTheExtractor(t *testing.T) {
	l := gdscript.Language{}
	if l.Name() != "gdscript" || l.Version() != "gdscript/1@"+treesitter.Parser || !reflect.DeepEqual(l.Extensions(), []string{".gd"}) {
		t.Fatalf("Language = %q %q %v", l.Name(), l.Version(), l.Extensions())
	}
	r, err := l.File(rel, "func f():\n\tpass\n")
	if err != nil || len(r.Nodes) != 2 {
		t.Fatalf("Language.File = %v, %v", views(r), err)
	}
}

// edgeSrc holds every edge form of a script, probed on gotreesitter v0.55.1.
const edgeSrc = "class_name Sample extends \"res://base.gd\"\n\nconst Helper := preload(\"res://lib/helper.gd\")\nvar Scene = load(\"scenes/x.tscn\")\nvar nothing = preload()\n\nsignal changed\n\nfunc run(a: Foo = Foo.new()) -> Bar:\n\tbump()\n\tself.bump()\n\tHelper.make(1)\n\tOuter.Inner.go()\n\tself.box.open()\n\tselfish.go()\n\tsuper.run(a)\n\tsuper()\n\tchanged.emit()\n\temit_signal(\"changed\")\n\tget_node(\"x\").poke()\n\tvar r = load(\"res://y.gd\")\n\tif a is Qux:\n\t\tpass\n\treturn a as Quux\n\nclass Inner extends Outer.Base:\n\tpass\n"

func edge(src model.NodeID, r model.Relation, name, receiver, spec string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: r, Name: name, Receiver: receiver, Specifier: spec, File: rel}
}

func TestEveryEdgeFormOfAScript(t *testing.T) {
	r := extractSrc(t, rel, edgeSrc)
	const run = "a/s.gd#Sample.run"
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		edge("a/s.gd#Sample", model.RelationExtends, "", "", "res://base.gd"),
		edge("a/s.gd#Sample.Inner", model.RelationExtends, "Base", "Outer", ""),
	})
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{
		edge("a/s.gd", model.RelationImports, "", "", "res://lib/helper.gd"),
		edge("a/s.gd", model.RelationImports, "", "", "res://scenes/x.tscn"),
		edge("a/s.gd", model.RelationImports, "", "", "res://y.gd"),
	})
	same(t, "aliases", r.Imports, []extract.Import{
		{Alias: "Helper", Path: "res://lib/helper.gd"},
		{Alias: "Scene", Path: "res://scenes/x.tscn"},
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		edge(run, model.RelationCalls, "new", "Foo", ""),
		edge(run, model.RelationCalls, "bump", "", ""),
		edge(run, model.RelationCalls, "bump", "", ""),
		edge(run, model.RelationCalls, "make", "Helper", ""),
		edge(run, model.RelationCalls, "go", "Outer.Inner", ""),
		edge(run, model.RelationCalls, "open", "box", ""),
		// Only "self." is cut off; a name that starts with self is a name.
		edge(run, model.RelationCalls, "go", "selfish", ""),
		edge(run, model.RelationCalls, "run", "super", ""),
		edge(run, model.RelationCalls, "run", "super", ""),
		edge(run, model.RelationCalls, "emit", "changed", ""),
		edge(run, model.RelationCalls, "emit", "changed", ""),
		edge(run, model.RelationCalls, "get_node", "", ""),
	})
	var refs []string
	for _, e := range edgesOf(r, model.RelationReferences) {
		refs = append(refs, string(e.Source)+" "+e.Name)
	}
	same(t, "references", refs, []string{
		run + " a", run + " Foo", run + " Bar", run + " self", run + " Helper",
		run + " Outer", run + " selfish", run + " super", run + " changed", run + " Qux", run + " Quux",
	})
}

// A signal's parameter types are names it uses; its own name is none.
func TestASignalReferencesItsParameterTypes(t *testing.T) {
	r := extractSrc(t, rel, "signal hit(t: Enemy)\n")
	same(t, "references", edgesOf(r, model.RelationReferences), []extract.RawEdge{
		edge("a/s.gd#hit", model.RelationReferences, "t", "", ""),
		edge("a/s.gd#hit", model.RelationReferences, "Enemy", "", ""),
	})
}

// `class_name A extends` with no base on its line: the parser takes the next
// line's first word for the base; the statement names nothing, as without a
// name, and the function after it is read as of a script without a class.
func TestAnExtendsWithoutABaseOnItsLineNamesNothing(t *testing.T) {
	r := extractSrc(t, rel, "class_name A extends\nfunc f():\n\tpass\n")
	same(t, "nodes", views(r), []string{`a/s.gd file owner= L1-L4 exported=true sig=""`})
	same(t, "extends", edgesOf(r, model.RelationExtends), nil)
	if r.Package != "" {
		t.Errorf("Package = %q, want none", r.Package)
	}
}

// A script without class_name has the file as the source of its base, its
// top-level calls and its references.
func TestAnUnnamedScriptIsTheSourceOfItsTopLevel(t *testing.T) {
	r := extractSrc(t, rel, "extends Base\n\nvar x: Thing = make()\n")
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{edge("a/s.gd", model.RelationExtends, "Base", "", "")})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{edge("a/s.gd", model.RelationCalls, "make", "", "")})
	same(t, "references", edgesOf(r, model.RelationReferences), []extract.RawEdge{edge("a/s.gd", model.RelationReferences, "Thing", "", "")})
}

// super() outside a function has no name to look for, and an alias only
// counts at the top of the script.
func TestSuperOutsideAFunctionAndAnAliasInAClassGiveNothing(t *testing.T) {
	r := extractSrc(t, rel, "var v = super()\n\nclass C:\n\tconst H = preload(\"res://h.gd\")\n")
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)
	same(t, "aliases", r.Imports, nil)
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{edge("a/s.gd", model.RelationImports, "", "", "res://h.gd")})
}

// emit_signal and preload with no string literal name nothing.
func TestCallsWithoutALiteralNameNothing(t *testing.T) {
	r := extractSrc(t, rel, "func f(p):\n\temit_signal(p)\n\tload(p)\n\tpreload()\n")
	same(t, "calls", edgesOf(r, model.RelationCalls), nil)
	same(t, "imports", edgesOf(r, model.RelationImports), nil)
}

// A base that is no plain name, no chain of names and no path names nothing;
// a chain whose last link is a name is a receiver and a name.
func TestABaseThatIsNoNameNamesNothing(t *testing.T) {
	for _, src := range []string{"extends Array[int]\n", "extends Foo.bar()\n", "class C extends :\n\tpass\n"} {
		r := extractSrc(t, rel, src)
		same(t, src, edgesOf(r, model.RelationExtends), nil)
	}
	r := extractSrc(t, rel, "extends Foo.Bar.baz\n")
	same(t, "chain", edgesOf(r, model.RelationExtends), []extract.RawEdge{edge("a/s.gd", model.RelationExtends, "baz", "Foo.Bar", "")})
}

// A class or a signal the parser rebuilt around rejected text gets no node;
// the class after it is whole.
func TestABrokenClassOrSignalGetsNoNode(t *testing.T) {
	r := extractSrc(t, rel, "class C(:\n\tpass\n\nclass D:\n\tpass\n\nsignal s(a,\n")
	same(t, "nodes", views(r), []string{
		`a/s.gd file owner= L1-L8 exported=true sig=""`,
		`a/s.gd#D class owner= L4-L5 exported=true sig="class D"`,
	})
}

func TestABrokenSignalGetsNoNode(t *testing.T) {
	for _, src := range []string{"signal s(a b)\n", "signal s(1)\n", "signal s(a:\n"} {
		same(t, src, views(extractSrc(t, rel, src))[1:], nil)
	}
}

// A class_name whose statement holds rejected text names nothing.
func TestAClassNameWithRejectedTextNamesNothing(t *testing.T) {
	for _, src := range []string{"class_name A extends B..C\n", "class_name A extends (B)\n"} {
		r := extractSrc(t, rel, src)
		same(t, src, views(r)[1:], nil)
		if r.Package != "" {
			t.Errorf("%q: Package = %q, want none", src, r.Package)
		}
	}
}

// A call inside text the parser rejected is no call.
func TestACallInRejectedTextCountsForNothing(t *testing.T) {
	for _, src := range []string{"func f():\n\tx = (bar()\n\ty = 2\n", "func f():\n\tx = [bar(]\n"} {
		r := extractSrc(t, rel, src)
		same(t, src, edgesOf(r, model.RelationCalls), nil)
	}
}

// Names inside the arguments of a call count, whatever form the call has.
func TestTheArgumentsOfACallAreWalked(t *testing.T) {
	r := extractSrc(t, rel, "func f():\n\tplain(one)\n\tobj.member(two)\n\ta.b().c(three)\n")
	var refs []string
	for _, e := range edgesOf(r, model.RelationReferences) {
		refs = append(refs, e.Name)
	}
	same(t, "references", refs, []string{"one", "obj", "two", "a", "three"})
	// After a call in the chain the receiver is no chain of names any more.
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		edge("a/s.gd#f", model.RelationCalls, "plain", "", ""),
		edge("a/s.gd#f", model.RelationCalls, "member", "obj", ""),
		edge("a/s.gd#f", model.RelationCalls, "b", "a", ""),
	})
}

// Single quotes quote a path as double quotes do.
func TestASingleQuotedPathIsAPath(t *testing.T) {
	r := extractSrc(t, rel, "const X = preload('res://a.gd')\nvar y = preload.foo(\"res://b.gd\")\n")
	same(t, "aliases", r.Imports, []extract.Import{{Alias: "X", Path: "res://a.gd"}})
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{edge("a/s.gd", model.RelationImports, "", "", "res://a.gd")})
}

// load() resolves a path without a scheme from res://, preload() from the
// script's own folder; a path with a scheme is left as it is.
func TestALoadPathWithoutASchemeIsFromTheProjectRoot(t *testing.T) {
	r := extractSrc(t, "ui/s.gd", "const A = load(\"data/a.gd\")\nconst B = preload(\"data/b.gd\")\nconst C = load(\"uid://c\")\nconst D = load(\"user://d\")\n")
	same(t, "aliases", r.Imports, []extract.Import{
		{Alias: "A", Path: "res://data/a.gd"}, {Alias: "B", Path: "data/b.gd"},
		{Alias: "C", Path: "uid://c"}, {Alias: "D", Path: "user://d"},
	})
	same(t, "imports", edgesOf(r, model.RelationImports), []extract.RawEdge{
		{Source: "ui/s.gd", Relation: model.RelationImports, Specifier: "res://data/a.gd", File: "ui/s.gd"},
		{Source: "ui/s.gd", Relation: model.RelationImports, Specifier: "data/b.gd", File: "ui/s.gd"},
		{Source: "ui/s.gd", Relation: model.RelationImports, Specifier: "uid://c", File: "ui/s.gd"},
		{Source: "ui/s.gd", Relation: model.RelationImports, Specifier: "user://d", File: "ui/s.gd"},
	})
}

// emit_signal takes a StringName literal as it takes a string.
func TestEmitSignalTakesAStringNameLiteral(t *testing.T) {
	r := extractSrc(t, rel, "const S = load(&\"res://s.gd\")\n\nfunc f():\n\temit_signal(&\"sig\")\n\tload(&\"res://x.gd\")\n")
	same(t, "aliases", r.Imports, nil)
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{edge("a/s.gd#f", model.RelationCalls, "emit", "sig", "")})
	same(t, "imports", edgesOf(r, model.RelationImports), nil)
}

func TestCRLFGivesTheSameEdges(t *testing.T) {
	lf := extractSrc(t, rel, edgeSrc)
	crlf := extractSrc(t, rel, strings.ReplaceAll(edgeSrc, "\n", "\r\n"))
	same(t, "edges", crlf.Edges, lf.Edges)
	same(t, "aliases", crlf.Imports, lf.Imports)
}
