package php_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/php"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// extractSrc calls the package function and not Language{}.File: the code
// graph follows a package selector, and so sees these tests reach the
// extractor.
func extractSrc(t *testing.T, rel, src string) extract.Result {
	t.Helper()
	r, err := php.File(rel, src)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// view is a node as most tests compare it: everything but its body.
func view(n model.Node) string {
	return fmt.Sprintf("%s %s owner=%s %s exported=%v sig=%s", n.ID, n.Kind, n.Owner, n.Span, n.Exported, n.Signature)
}

func views(r extract.Result) []string {
	var out []string
	for _, n := range r.Nodes {
		out = append(out, view(n))
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
	return extract.RawEdge{Source: src, Relation: model.RelationContains, TargetID: dst, File: "m.php"}
}

func call(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationCalls, Name: name, Receiver: receiver, File: "m.php"}
}

func onObject(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "."
	return e
}

func scoped(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "::"
	return e
}

func construct(src model.NodeID, name, receiver string) extract.RawEdge {
	e := call(src, name, receiver)
	e.Member = "new"
	return e
}

func imports(spec string) extract.RawEdge {
	return extract.RawEdge{Source: "m.php", Relation: model.RelationImports, Specifier: spec, File: "m.php"}
}

func extends(src model.NodeID, name, receiver string) extract.RawEdge {
	return extract.RawEdge{Source: src, Relation: model.RelationExtends, Name: name, Receiver: receiver, File: "m.php"}
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

func TestLanguageDescribesPHP(t *testing.T) {
	var l extract.Language = php.Language{}
	if l.Name() != "php" {
		t.Errorf("Name = %q, want php", l.Name())
	}
	if want := "php/1@" + treesitter.Parser; l.Version() != want {
		t.Errorf("Version = %q, want %q", l.Version(), want)
	}
	same(t, "Extensions", l.Extensions(), []string{".php"})
	r, err := l.File("src/M.php", "<?php class K { }\n")
	if err != nil || r.Path != "src/M.php" || r.Language != "php" || len(r.Nodes) != 2 {
		t.Errorf("File = %d nodes at %q (%q), %v; want the file and K at src/M.php", len(r.Nodes), r.Path, r.Language, err)
	}
}

func TestTypesMembersAndFunctionsGetNodes(t *testing.T) {
	const src = `<?php
namespace App;

#[Attr]
final class C extends Base {
    public function pub() { }
    function plain() { }
    protected function prot() { }
    private static function priv(): int { return 1; }
    abstract public function abs();
    public $field = 1;
    const K = 2;
}
interface I { function m(); }
trait T { private function t() { } }
enum E: string { case A = 'a'; public function label() { } }
function f(int $a): void { }
function f(): void { }
`
	r := extractSrc(t, "m.php", src)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	same(t, "nodes", views(r)[1:], []string{
		`m.php#C class owner= L4-L13 exported=true sig=final class C extends Base`,
		`m.php#C.pub method owner=C L6-L6 exported=true sig=public function pub()`,
		`m.php#C.plain method owner=C L7-L7 exported=true sig=function plain()`,
		`m.php#C.prot method owner=C L8-L8 exported=false sig=protected function prot()`,
		`m.php#C.priv method owner=C L9-L9 exported=false sig=private static function priv(): int`,
		`m.php#C.abs method owner=C L10-L10 exported=true sig=abstract public function abs()`,
		`m.php#I interface owner= L14-L14 exported=true sig=interface I`,
		`m.php#I.m method owner=I L14-L14 exported=true sig=function m()`,
		`m.php#T class owner= L15-L15 exported=true sig=trait T`,
		`m.php#T.t method owner=T L15-L15 exported=false sig=private function t()`,
		`m.php#E class owner= L16-L16 exported=true sig=enum E: string`,
		`m.php#E.label method owner=E L16-L16 exported=true sig=public function label()`,
		`m.php#f function owner= L17-L17 exported=true sig=function f(int $a): void`,
		`m.php#f~2 function owner= L18-L18 exported=true sig=function f(): void`,
	})
	same(t, "contains", edgesOf(r, model.RelationContains), []extract.RawEdge{
		contains("m.php", "m.php#C"),
		contains("m.php#C", "m.php#C.pub"), contains("m.php#C", "m.php#C.plain"), contains("m.php#C", "m.php#C.prot"),
		contains("m.php#C", "m.php#C.priv"), contains("m.php#C", "m.php#C.abs"),
		contains("m.php", "m.php#I"), contains("m.php#I", "m.php#I.m"),
		contains("m.php", "m.php#T"), contains("m.php#T", "m.php#T.t"),
		contains("m.php", "m.php#E"), contains("m.php#E", "m.php#E.label"),
		contains("m.php", "m.php#f"), contains("m.php", "m.php#f~2"),
	})
}

// A function or class PHP declares when the code around it runs -- inside
// an if, a function or a method -- is declared in the namespace all the
// same, and gets its node at the file.
func TestConditionalDeclarationsGetNodes(t *testing.T) {
	const src = `<?php
namespace N;
if (! function_exists('app')) {
    function app() { return make(); }
}
function outer() {
    function inner() { }
    if (true) { class Late { function m() { } } }
}
`
	r := extractSrc(t, "m.php", src)
	same(t, "nodes", views(r)[1:], []string{
		`m.php#app function owner= L4-L4 exported=true sig=function app()`,
		`m.php#outer function owner= L6-L9 exported=true sig=function outer()`,
		`m.php#inner function owner= L7-L7 exported=true sig=function inner()`,
		`m.php#Late class owner= L8-L8 exported=true sig=class Late`,
		`m.php#Late.m method owner=Late L8-L8 exported=true sig=function m()`,
	})
	if want := map[model.NodeID]string{"m.php#app": `N`, "m.php#outer": `N`, "m.php#inner": `N`, "m.php#Late": `N`}; !reflect.DeepEqual(r.Enclosing, want) {
		t.Errorf("Enclosing = %v, want %v", r.Enclosing, want)
	}
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.php", "function_exists", ""), call("m.php#app", "make", ""),
	})
}

// Review Focus 1: every type and function carries its own namespace, in
// the statement form and the block form, several of them per file.
func TestNamespacesPerDeclaration(t *testing.T) {
	statements := extractSrc(t, "m.php", `<?php
namespace A\B;
class X { function m() { } }
function fx() { }
namespace C;
interface Y { }
`)
	if statements.Package != `A\B` {
		t.Errorf("Package = %q, want A\\B", statements.Package)
	}
	if want := map[model.NodeID]string{"m.php#X": `A\B`, "m.php#fx": `A\B`, "m.php#Y": `C`}; !reflect.DeepEqual(statements.Enclosing, want) {
		t.Errorf("Enclosing = %v, want %v", statements.Enclosing, want)
	}

	blocks := extractSrc(t, "m.php", `<?php
namespace D {
    class Z { }
}
namespace {
    function g() { }
    class Glob { }
}
`)
	if blocks.Package != "D" {
		t.Errorf("Package = %q, want D", blocks.Package)
	}
	if want := map[model.NodeID]string{"m.php#Z": "D", "m.php#g": "", "m.php#Glob": ""}; !reflect.DeepEqual(blocks.Enclosing, want) {
		t.Errorf("Enclosing = %v, want %v", blocks.Enclosing, want)
	}

	none := extractSrc(t, "m.php", "<?php\nfunction h() { }\n")
	if none.Package != "" || !reflect.DeepEqual(none.Enclosing, map[model.NodeID]string{"m.php#h": ""}) {
		t.Errorf("Package %q, Enclosing %v; want the global namespace", none.Package, none.Enclosing)
	}
	empty := extractSrc(t, "m.php", "<?php\nnamespace E;\n")
	if empty.Package != "" || empty.Enclosing != nil {
		t.Errorf("Package %q, Enclosing %v; want neither without a declaration", empty.Package, empty.Enclosing)
	}
}

func TestUseInEveryForm(t *testing.T) {
	const src = `<?php
use A\B\C;
use \D\E as F, G;
use A\{X, Y\Z as W};
use function A\f;
use function A\{g, h as i};
use const A\K;
use const A\{L};
use A\{function m, const N};
`
	r := extractSrc(t, "m.php", src)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: `A\B\C`}, {Path: `D\E`, Alias: "F"}, {Path: "G"},
		{Path: `A\X`}, {Path: `A\Y\Z`, Alias: "W"},
		{Path: `A\f`, Kind: "function"},
		{Path: `A\g`, Kind: "function"}, {Path: `A\h`, Alias: "i", Kind: "function"},
		{Path: `A\K`, Kind: "const"}, {Path: `A\L`, Kind: "const"},
		{Path: `A\m`, Kind: "function"}, {Path: `A\N`, Kind: "const"},
	})
	same(t, "imports edges", edgesOf(r, model.RelationImports), []extract.RawEdge{
		imports(`A\B\C`), imports(`D\E`), imports("G"), imports(`A\X`), imports(`A\Y\Z`),
		imports(`A\f`), imports(`A\g`), imports(`A\h`), imports(`A\K`), imports(`A\L`),
		imports(`A\m`), imports(`A\N`),
	})
}

// require and include name a file relative to the one that runs them: a
// bare literal, or one after __DIR__ or dirname(__FILE__). Anything built
// at run time names nothing the graph can follow.
func TestRequireAndInclude(t *testing.T) {
	const src = `<?php
require 'lib.php';
require_once __DIR__ . '/boot.php';
include dirname( __FILE__ ) . '/../x.php';
include_once("paren.php");
function load() { require "dq.php"; }
require $dynamic;
require "$dir/z.php";
require "esc\x41.php";
require '';
require __DIR__ . $x;
require dirname(__DIR__) . '/no.php';
require strtolower('no.php');
require 'a.php' . 'b.php';
`
	r := extractSrc(t, "m.php", src)
	same(t, "imports", r.Imports, []extract.Import{
		{Path: "lib.php", Kind: "require"}, {Path: "boot.php", Kind: "require"}, {Path: "../x.php", Kind: "require"},
		{Path: "paren.php", Kind: "require"}, {Path: "dq.php", Kind: "require"},
	})
	same(t, "imports edges", edgesOf(r, model.RelationImports), []extract.RawEdge{
		imports("lib.php"), imports("boot.php"), imports("../x.php"), imports("paren.php"), imports("dq.php"),
	})
}

func TestBasesInterfacesAndTraits(t *testing.T) {
	const src = `<?php
class C extends P implements I1, \Ns\I2, namespace\I3 {
    use T1, \Ns\T2;
    use T3 { a as b; }
}
interface I extends J, K\L { }
enum E implements I { }
$o = new class extends Anon { };
`
	r := extractSrc(t, "m.php", src)
	same(t, "extends", edgesOf(r, model.RelationExtends), []extract.RawEdge{
		extends("m.php#C", "P", ""), extends("m.php#C", "I1", ""), extends("m.php#C", "I2", `\Ns`),
		extends("m.php#C", "I3", "namespace"),
		extends("m.php#C", "T1", ""), extends("m.php#C", "T2", `\Ns`), extends("m.php#C", "T3", ""),
		extends("m.php#I", "J", ""), extends("m.php#I", "L", "K"),
		extends("m.php#E", "I", ""),
	})
}

func TestCallsInEveryForm(t *testing.T) {
	const src = `<?php
namespace N;
class C {
    function run() {
        f();
        A\g();
        \A\h();
        \strlen('x');
        namespace\k();
        X::m();
        \Y\Z::m2();
        self::a();
        static::b();
        parent::c();
        $this->d();
        $this?->e();
        $obj->o();
        $obj?->o2();
        $this->prop->p();
        $this?->prop?->p2();
        $obj::s();
        new Foo();
        new \Q\R;
        new static();
        $fn();
        $obj->$m();
        X::$m();
        get()->q();
        (new Made)->r();
        new $cls();
        new class { function z() { w(); } };
    }
}
function top() { f2(); }
f3();
`
	r := extractSrc(t, "m.php", src)
	run, top := model.NodeID("m.php#C.run"), model.NodeID("m.php#top")
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call(run, "f", ""), call(run, "g", "A"), call(run, "h", `\A`), call(run, "strlen", `\`),
		call(run, "k", "namespace"),
		scoped(run, "m", "X"), scoped(run, "m2", `\Y\Z`),
		scoped(run, "a", "self"), scoped(run, "b", "static"), scoped(run, "c", "parent"),
		call(run, "d", "$this"), call(run, "e", "$this"),
		onObject(run, "o", "$obj"), onObject(run, "o2", "$obj"),
		onObject(run, "p", "$this->prop"), onObject(run, "p2", "$this->prop"),
		onObject(run, "s", "$obj"),
		construct(run, "Foo", ""), construct(run, "R", `\Q`), construct(run, "static", ""),
		call(run, "get", ""), construct(run, "Made", ""),
		call(run, "w", ""),
		call(top, "f2", ""), call("m.php", "f3", ""),
	})
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
}

// The syntax each PHP version from 8.2 to 8.6 added parses clean, and the
// declarations around it are read.
func TestSyntaxOfEachVersion(t *testing.T) {
	const src = `<?php
readonly class P82 { public function f((A&B)|null $x): true { return g82(); } }
class P83 { const string NAME = 'x'; #[\Override] public function f() { return static::{$n}; } }
class P84 {
    public private(set) string $v;
    public string $name { get => strtoupper($this->name); set { $this->x = trim($value); } }
    public function f() { return new P84()->g(); }
}
#[\NoDiscard] function p85($s) { return $s |> trim(...) |> strlen(...); }
function p86() { return str_replace('a', ?, ...); }
`
	r := extractSrc(t, "m.php", src)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	var names []model.NodeID
	for _, n := range r.Nodes[1:] {
		names = append(names, n.ID)
	}
	same(t, "nodes", names, []model.NodeID{
		"m.php#P82", "m.php#P82.f", "m.php#P83", "m.php#P83.f", "m.php#P84", "m.php#P84.f", "m.php#p85", "m.php#p86",
	})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.php#P82.f", "g82", ""),
		// A property hook has no node; its calls belong to the class.
		call("m.php#P84", "strtoupper", ""), call("m.php#P84", "trim", ""),
		// The receiver of g is no chain of names.
		construct("m.php#P84.f", "P84", ""),
		call("m.php#p85", "trim", ""), call("m.php#p85", "strlen", ""),
		call("m.php#p86", "str_replace", ""),
	})
}

// Two forms of PHP 8.5 are beyond the grammar: clone($o, [...]) and the
// (void) cast. Each is an ERROR inside the method's body and counts as a
// parse error; the declarations around it and the calls beside it are read.
func TestPHP85GapsStayLocal(t *testing.T) {
	const src = `<?php
class K {
    function m($o) {
        $c = clone($o, ['a' => inner()]);
        (void) $c;
        kept();
        return $c;
    }
}
function after() { }
`
	r := extractSrc(t, "m.php", src)
	if r.ParseErrors < 2 {
		t.Errorf("ParseErrors = %d, want both gaps counted", r.ParseErrors)
	}
	var names []model.NodeID
	for _, n := range r.Nodes[1:] {
		names = append(names, n.ID)
	}
	same(t, "nodes", names, []model.NodeID{"m.php#K", "m.php#K.m", "m.php#after"})
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{
		call("m.php#K.m", "inner", ""), call("m.php#K.m", "kept", ""),
	})
}

// Review Focus 4: the HTML around the PHP is skipped, the PHP in it read,
// and a namespace holds across a stretch of HTML.
func TestHTMLAroundThePHP(t *testing.T) {
	const src = `<html>
<?php namespace Page; ?>
<body><?php if ($x): ?>
<p><?= h($y) ?></p>
<?php endif; ?>
<?php
class K { function m() { echo '<b>'; } }
?>
<footer>class Fake { function no() { } }</footer>
`
	r := extractSrc(t, "m.php", src)
	if r.ParseErrors != 0 {
		t.Errorf("ParseErrors = %d, want 0", r.ParseErrors)
	}
	same(t, "nodes", views(r)[1:], []string{
		`m.php#K class owner= L7-L7 exported=true sig=class K`,
		`m.php#K.m method owner=K L7-L7 exported=true sig=function m()`,
	})
	if want := map[model.NodeID]string{"m.php#K": "Page"}; !reflect.DeepEqual(r.Enclosing, want) {
		t.Errorf("Enclosing = %v, want %v", r.Enclosing, want)
	}
	same(t, "calls", edgesOf(r, model.RelationCalls), []extract.RawEdge{call("m.php", "h", "")})
}

// A definition the parser rebuilt around rejected text gets no node, and
// nothing in rejected text counts.
func TestRejectedTextNamesNothing(t *testing.T) {
	const src = `<?php
class Broken extends { function m() { inside(); } }
class Fine { function n() { ok(); } }
`
	r := extractSrc(t, "m.php", src)
	if r.ParseErrors == 0 {
		t.Fatal("ParseErrors = 0, want the broken header counted")
	}
	fine := false
	for _, n := range r.Nodes {
		if n.Name == "Broken" || n.Name == "m" {
			t.Errorf("node %s from a rejected header", n.ID)
		}
		fine = fine || n.ID == "m.php#Fine"
	}
	if !fine {
		t.Errorf("no node m.php#Fine: the code after the broken definition was lost")
	}
	for _, e := range r.Edges {
		if e.Name == "inside" {
			t.Errorf("edge %+v from a rejected definition", e)
		}
	}
}
