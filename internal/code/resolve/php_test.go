package resolve_test

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/php"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// phpGraph extracts a tree of PHP files, path to source, with the real
// extractor, and resolves them. The files are handed over in reverse byte
// order: which file the resolver picks must not depend on it.
func phpGraph(t *testing.T, tree map[string]string) (*model.Graph, []resolve.Unresolved) {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var files []extract.Result
	for _, p := range paths {
		r, err := php.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	g, un := resolve.GraphWith(files, nil, stamp)
	return validated(t, g), un
}

func TestPHPClassNames(t *testing.T) {
	g, un := phpGraph(t, map[string]string{
		"src/A.php": `<?php
namespace App;
use Lib\Base;
use Lib\Other as Alias;
use Lib;
class Own { }
class Q extends \Lib\Base { }
class R extends Base { }
class S extends Alias { }
class T extends Own { }
class U extends Lib\Base { }
class V extends namespace\Own { }
class W extends Missing { }
class X extends \Exception { }
`,
		"lib/Base.php": "<?php\nnamespace Lib;\nclass Base { }\nclass Other { }\n",
		// A twin in a greater path loses.
		"lib/zz/Base.php": "<?php\nnamespace Lib;\nclass Base { }\n",
		"global.php":      "<?php\nclass Own { }\n",
	})
	wantEdges(t, g,
		"src/A.php -imports-> lib/Base.php extracted",
		"src/A.php#Q -extends-> lib/Base.php#Base extracted",
		"src/A.php#R -extends-> lib/Base.php#Base extracted",
		"src/A.php#S -extends-> lib/Base.php#Other extracted",
		"src/A.php#T -extends-> src/A.php#Own extracted",
		"src/A.php#U -extends-> lib/Base.php#Base extracted",
		"src/A.php#V -extends-> src/A.php#Own extracted",
	)
	want := []resolve.Unresolved{
		{Source: "src/A.php#W", File: "src/A.php", Line: 13, Name: "Missing", Reason: resolve.ReasonNone},
		{Source: "src/A.php#X", File: "src/A.php", Line: 14, Name: `\Exception`, Reason: resolve.ReasonNone},
	}
	if !reflect.DeepEqual(un, want) {
		t.Errorf("unresolved = %+v, want %+v", un, want)
	}
}

func TestPHPBasesByKind(t *testing.T) {
	g, un := phpGraph(t, map[string]string{
		"a.php": `<?php
interface I { }
interface J extends I { }
trait Tr { }
class P { }
class C extends P implements I, J { use Tr; }
enum E implements I { case A; }
class D { use Tr; }
`,
	})
	wantEdges(t, g,
		"a.php#J -implements-> a.php#I extracted",
		"a.php#C -extends-> a.php#P extracted",
		"a.php#C -implements-> a.php#I extracted",
		"a.php#C -implements-> a.php#J extracted",
		"a.php#C -extends-> a.php#Tr extracted",
		"a.php#E -implements-> a.php#I extracted",
		"a.php#D -extends-> a.php#Tr extracted",
	)
	if len(un) != 0 {
		t.Errorf("unresolved = %+v, want none", un)
	}
}

func TestPHPImports(t *testing.T) {
	g, _ := phpGraph(t, map[string]string{
		"src/App.php": `<?php
namespace App;
use Lib\Zed;
use Lib\Zed as Again;
use Lib\Missing;
use Lib;
use Symfony\Component\Console;
use function Lib\helper;
use function Lib\nothing;
use const Lib\X;
use App\Own;
require 'boot.php';
require_once __DIR__ . '/../lib/Zed.php';
include dirname(__FILE__) . '/sub/inc.php';
require 'gone.php';
require '../../outside.php';
class Own { }
`,
		"src/boot.php":    "<?php\n",
		"src/sub/inc.php": "<?php\n",
		"lib/Zed.php":     "<?php\nnamespace Lib;\nclass Zed { }\nconst X = 1;\n",
		"lib/fn.php":      "<?php\nnamespace Lib;\nfunction helper() { }\n",
	})
	var got []string
	for _, e := range g.Edges {
		if e.Relation == model.RelationImports {
			got = append(got, fmt.Sprintf("%s -> %s L%d", e.Source, e.Target, e.Line))
		}
	}
	want := []string{
		"src/App.php -> lib/Zed.php L3",
		"src/App.php -> lib/fn.php L8",
		"src/App.php -> src/boot.php L12",
		"src/App.php -> src/sub/inc.php L14",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("imports = %q, want %q", got, want)
	}
}

func TestPHPFunctionCalls(t *testing.T) {
	g, _ := phpGraph(t, map[string]string{
		"src/a.php": `<?php
namespace App;
use function Lib\helper;
use function Lib\helper as h;
use Lib as L;
function own() { }
function run() {
    own();
    shared();
    global_only();
    helper();
    h();
    \global_only();
    \Lib\helper();
    L\helper();
    Sub\deep();
    namespace\own();
    strlen('x');
}
`,
		"src/b.php":   "<?php\nnamespace App;\nfunction shared() { }\nnamespace App\\Sub;\nfunction deep() { }\n",
		"global.php":  "<?php\nfunction shared() { }\nfunction global_only() { }\n",
		"lib/lib.php": "<?php\nnamespace Lib;\nfunction helper() { }\n",
	})
	wantCalls(t, g, "src/a.php#run",
		"global.php#global_only extracted",
		"global.php#global_only extracted",
		"lib/lib.php#helper extracted",
		"lib/lib.php#helper extracted",
		"lib/lib.php#helper extracted",
		"lib/lib.php#helper extracted",
		"src/a.php#own extracted",
		"src/a.php#own extracted",
		"src/b.php#deep extracted",
		"src/b.php#shared extracted",
	)
}

func TestPHPAnImportedFunctionNeverFallsBackToTheGlobalNamespace(t *testing.T) {
	g, _ := phpGraph(t, map[string]string{
		"a.php":      "<?php\nnamespace App;\nuse function Lib\\nothing;\nfunction run() { nothing(); }\n",
		"global.php": "<?php\nfunction nothing() { }\n",
	})
	wantCalls(t, g, "a.php#run")
}

func TestPHPNamesIgnoreCase(t *testing.T) {
	g, un := phpGraph(t, map[string]string{
		"a.php": `<?php
namespace App;
use LIB\base AS B;
class Foo extends b {
    function Run() {
        new foo();
        NEW SELF();
        $this->RUN();
        PARENT::Up();
        \app\foo::run();
        HELPER();
        // Variables keep their case: $THIS is not $this.
        $THIS->run();
    }
}
function helper() { }
`,
		"lib.php": "<?php\nnamespace Lib;\nclass Base { function up() { } }\n",
	})
	wantCalls(t, g, "a.php#Foo.Run",
		"a.php#Foo extracted",
		"a.php#Foo extracted",
		"a.php#Foo.Run extracted",
		"a.php#Foo.Run extracted",
		"a.php#helper extracted",
		"lib.php#Base.up extracted",
	)
	if len(un) != 0 {
		t.Errorf("unresolved = %+v, want none", un)
	}
}

func TestPHPStaticCallsAndNamespacedFunctionsStayApart(t *testing.T) {
	g, _ := phpGraph(t, map[string]string{
		"a.php": `<?php
namespace App;
function run() {
    X::g();
    X\g();
    X::onlyFn();
    X\onlyMethod();
}
`,
		"x.php": `<?php
namespace App;
class X { static function g() { } static function onlyMethod() { } }
`,
		"xf.php": "<?php\nnamespace App\\X;\nfunction g() { }\nfunction onlyFn() { }\n",
	})
	wantCalls(t, g, "a.php#run",
		"x.php#X.g extracted",
		"xf.php#g extracted",
	)
}

func TestPHPClassNamesNeverFallBackToTheGlobalNamespace(t *testing.T) {
	g, un := phpGraph(t, map[string]string{
		"a.php":      "<?php\nnamespace App;\nclass C extends G { function m() { G::s(); new G(); } }\n",
		"global.php": "<?php\nclass G { static function s() { } }\n",
	})
	wantCalls(t, g, "a.php#C.m")
	if len(un) != 1 || un[0].Name != "G" {
		t.Errorf("unresolved = %+v, want G alone", un)
	}
}

func TestPHPMethodCalls(t *testing.T) {
	g, _ := phpGraph(t, map[string]string{
		"a.php": `<?php
namespace App;
trait Tr { function t() { } function both() { } }
class P { function p() { } function both() { } function __construct() { } }
class C extends P {
    use Tr;
    function own() { }
    function run($obj) {
        $this->own();
        $this->t();
        $this->p();
        $this->both();
        self::own();
        static::t();
        parent::p();
        parent::t();
        C::own();
        \App\C::p();
        $obj->own();
        $obj::own();
        $this->missing();
    }
}
`,
	})
	wantCalls(t, g, "a.php#C.run",
		// $this->own(), self::own(), C::own()
		"a.php#C.own extracted",
		"a.php#C.own extracted",
		"a.php#C.own extracted",
		// $this->p(), parent::p(), \App\C::p()
		"a.php#P.p extracted",
		"a.php#P.p extracted",
		"a.php#P.p extracted",
		// A trait's method wins over the parent's.
		"a.php#Tr.both extracted",
		// $this->t(), static::t(); parent::t() is not the parent's.
		"a.php#Tr.t extracted",
		"a.php#Tr.t extracted",
	)
}

func TestPHPConstructorCalls(t *testing.T) {
	g, _ := phpGraph(t, map[string]string{
		"a.php": `<?php
namespace App;
class WithCtor { function __construct() { } }
class Plain { }
class Child extends WithCtor {
    function make() {
        new WithCtor();
        new Plain;
        new Child();
        new static();
        new self();
        new \App\Plain();
        new Nowhere();
    }
}
function outside() { new static(); }
`,
	})
	wantCalls(t, g, "a.php#Child.make",
		"a.php#Plain extracted",
		"a.php#Plain extracted",
		// Its own, then the inherited one through Child, static and self.
		"a.php#WithCtor.__construct extracted",
		"a.php#WithCtor.__construct extracted",
		"a.php#WithCtor.__construct extracted",
		"a.php#WithCtor.__construct extracted",
	)
	wantCalls(t, g, "a.php#outside")
}

func TestPHPCallsOutsideATypeAndOnCyclicBases(t *testing.T) {
	g, _ := phpGraph(t, map[string]string{
		"a.php": `<?php
namespace App;
class A extends B { function run() { $this->none(); parent::none(); } }
class B extends A { }
function f() { self::x(); parent::x(); $this->x(); }
f();
`,
	})
	wantCalls(t, g, "a.php#A.run")
	wantCalls(t, g, "a.php#f")
	wantCalls(t, g, "a.php", "a.php#f extracted")
}
