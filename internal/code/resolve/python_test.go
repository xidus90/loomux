package resolve_test

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/python"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// pyFiles extracts a tree of Python files, path to source, with the real
// extractor, in the byte order sourceset's walk would hand them over.
func pyFiles(t *testing.T, tree map[string]string) []extract.Result {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var files []extract.Result
	for _, p := range paths {
		r, err := python.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	return files
}

// pyGraph extracts a tree of Python files and resolves them.
func pyGraph(t *testing.T, tree map[string]string) *model.Graph {
	t.Helper()
	return validated(t, resolve.Graph(pyFiles(t, tree), nil, stamp))
}

// validated fails the test unless g validates, and returns it.
func validated(t *testing.T, g *model.Graph) *model.Graph {
	t.Helper()
	if err := g.Validate(); err != nil {
		t.Fatalf("a resolved Python graph must validate: %v", err)
	}
	return g
}

// wantCalls checks the calls edges out of from, in the graph's order, each as
// "target confidence": a call may be a guess, and which one it is belongs to
// the answer.
func wantCalls(t *testing.T, g *model.Graph, from model.NodeID, want ...string) {
	t.Helper()
	var got []string
	for _, e := range g.Edges {
		if e.Source == from && e.Relation == model.RelationCalls {
			got = append(got, string(e.Target)+" "+string(e.Confidence))
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s -calls-> %q, want %q", from, got, want)
	}
}

// wantTargets checks the targets of from's edges of one relation, in the
// graph's order, and that each of them is extracted: an import or a base
// either names its target or gives no edge.
func wantTargets(t *testing.T, g *model.Graph, from model.NodeID, rel model.Relation, want ...string) {
	t.Helper()
	var got []string
	for _, e := range g.Edges {
		if e.Source != from || e.Relation != rel {
			continue
		}
		got = append(got, string(e.Target))
		if e.Confidence != model.ConfidenceExtracted {
			t.Errorf("%s -%s-> %s is %s, want extracted", from, rel, e.Target, e.Confidence)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s -%s-> %q, want %q", from, rel, got, want)
	}
}

func TestPythonModulePathsFromRootAndSrc(t *testing.T) {
	tree := map[string]string{
		"pkg/__init__.py":     "",
		"pkg/a.py":            "",
		"src/lib/__init__.py": "",
		"src/lib/b.py":        "",
	}
	want := map[string]string{
		"pkg": "pkg/__init__.py", "pkg.a": "pkg/a.py",
		// src holds a package, so it is a root of its own: what installs
		// src/lib imports it as lib.
		"lib": "src/lib/__init__.py", "lib.b": "src/lib/b.py",
		// And the repository root still sees it as src.lib.
		"src.lib": "src/lib/__init__.py", "src.lib.b": "src/lib/b.py",
	}
	for spec := range want {
		tree["use_"+strings.ReplaceAll(spec, ".", "_")+".py"] = "import " + spec + "\n"
	}
	g := pyGraph(t, tree)
	for spec, target := range want {
		wantTargets(t, g, model.NodeID("use_"+strings.ReplaceAll(spec, ".", "_")+".py"), model.RelationImports, target)
	}

	// A loose module directly under src/, and one in a directory without an
	// __init__.py: neither is a package, so src/ is no root, and only the
	// repository root's names reach them.
	g = pyGraph(t, map[string]string{
		"src/tool.py":        "",
		"src/scripts/run.py": "",
		"use.py":             "import tool\nimport scripts.run\nimport src.tool\nimport src.scripts.run\n",
	})
	wantTargets(t, g, "use.py", model.RelationImports, "src/scripts/run.py", "src/tool.py")
}

func TestPythonManagePyDirectoryIsASourceRoot(t *testing.T) {
	for _, inits := range []bool{false, true} {
		tree := map[string]string{
			"backend/manage.py":            "def main():\n    pass\n",
			"backend/apps/users/models.py": "def helper():\n    pass\n\n\nclass Base:\n    def save(self):\n        pass\n",
			"backend/apps/orders/services.py": "from apps.users.models import helper, Base\n\n\ndef run():\n    helper()\n\n\n" +
				"class Svc(Base):\n    def go(self):\n        self.save()\n",
			// A second helper, so that the call can only come from the import.
			"other.py": "def helper():\n    pass\n",
		}
		if inits {
			for _, p := range []string{"backend/apps/__init__.py", "backend/apps/users/__init__.py", "backend/apps/orders/__init__.py"} {
				tree[p] = ""
			}
		}
		// Django runs manage.py with its own directory on the path, and its
		// apps import each other from there, packages or not.
		g := pyGraph(t, tree)
		wantTargets(t, g, "backend/apps/orders/services.py", model.RelationImports, "backend/apps/users/models.py")
		wantCalls(t, g, "backend/apps/orders/services.py#run", "backend/apps/users/models.py#helper extracted")
		wantTargets(t, g, "backend/apps/orders/services.py#Svc", model.RelationExtends, "backend/apps/users/models.py#Base")
		wantCalls(t, g, "backend/apps/orders/services.py#Svc.go", "backend/apps/users/models.py#Base.save extracted")
	}

	// One module path under two roots: apps.x is apps/x.py from the
	// repository root and backend/apps/x.py from backend/. Each file finds
	// the one of the deepest root that holds it and maps the path. main.py
	// lies under the repository root alone, which `python main.py` puts
	// first on the path; manage.py runs from backend/.
	g := pyGraph(t, map[string]string{
		"apps/__init__.py":  "",
		"apps/x.py":         "",
		"backend/manage.py": "import apps.x\n",
		"backend/apps/x.py": "",
		"main.py":           "import apps.x\n",
	})
	wantTargets(t, g, "main.py", model.RelationImports, "apps/x.py")
	wantTargets(t, g, "backend/manage.py", model.RelationImports, "backend/apps/x.py")
}

// wantEdges checks every edge of g but containment, each as "source
// -relation-> target confidence", in any order.
func wantEdges(t *testing.T, g *model.Graph, want ...string) {
	t.Helper()
	var got []string
	for _, e := range g.Edges {
		if e.Relation != model.RelationContains {
			got = append(got, fmt.Sprintf("%s -%s-> %s %s", e.Source, e.Relation, e.Target, e.Confidence))
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("edges:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

func TestPythonModuleResolvesUnderTheRootsThatHoldTheImporter(t *testing.T) {
	for _, c := range []struct {
		name string
		tree map[string]string
		want []string
	}{{
		// examples/demo holds a top-level package config, so it is a root
		// that maps config as well. main.py lies under the repository root
		// alone, and the repository root's config is the one it imports.
		name: "a copy of a package in an example",
		tree: map[string]string{
			"config/__init__.py":               "def load():\n    return {}\n",
			"config/settings.py":               "X = 1\n",
			"examples/demo/config/__init__.py": "def load():\n    return {}\n",
			"examples/demo/config/settings.py": "X = 1\n",
			"main.py":                          "import config.settings\nfrom config import load\n\n\ndef main():\n    load()\n",
		},
		want: []string{
			"main.py -imports-> config/__init__.py extracted",
			"main.py -imports-> config/settings.py extracted",
			"main.py#main -calls-> config/__init__.py#load extracted",
		},
	}, {
		// Two packages tests, one at the repository root and one under the
		// root tools/cli: each test module imports the helpers beside it.
		name: "two test packages of one name",
		tree: map[string]string{
			"tests/__init__.py":           "",
			"tests/helpers.py":            "def make():\n    return 1\n",
			"tests/test_core.py":          "from tests.helpers import make\n\n\ndef test_compute():\n    make()\n",
			"tools/cli/tests/__init__.py": "",
			"tools/cli/tests/helpers.py":  "def make():\n    return 2\n",
			"tools/cli/tests/test_cli.py": "from tests.helpers import make\n\n\ndef test_cli():\n    make()\n",
		},
		want: []string{
			"tests/test_core.py -imports-> tests/helpers.py extracted",
			"tests/test_core.py#test_compute -calls-> tests/helpers.py#make extracted",
			"tools/cli/tests/test_cli.py -imports-> tools/cli/tests/helpers.py extracted",
			"tools/cli/tests/test_cli.py#test_cli -calls-> tools/cli/tests/helpers.py#make extracted",
		},
	}, {
		// No root that holds main.py maps apps.x; the one root that does,
		// backend/, still answers for it.
		name: "a file outside the root of the module",
		tree: map[string]string{
			"backend/apps/x.py": "def helper():\n    return 1\n",
			"backend/manage.py": "from apps.x import helper\n\n\ndef run():\n    helper()\n",
			"main.py":           "from apps.x import helper\n\n\ndef top():\n    helper()\n",
		},
		want: []string{
			"backend/manage.py -imports-> backend/apps/x.py extracted",
			"backend/manage.py#run -calls-> backend/apps/x.py#helper extracted",
			"main.py -imports-> backend/apps/x.py extracted",
			"main.py#top -calls-> backend/apps/x.py#helper extracted",
		},
	}, {
		// config is ambiguous across all roots, the repository root's and the
		// one of examples/demo. The deepest root that holds orders.py,
		// backend/, maps no config; the next one up, the repository root,
		// does, and answers before the ambiguous map across every root.
		name: "a shallower root that maps the path",
		tree: map[string]string{
			"backend/manage.py":                "",
			"backend/apps/orders.py":           "import config\n",
			"config/__init__.py":               "",
			"examples/demo/config/__init__.py": "",
		},
		want: []string{
			"backend/apps/orders.py -imports-> config/__init__.py extracted",
		},
	}} {
		t.Run(c.name, func(t *testing.T) {
			wantEdges(t, pyGraph(t, c.tree), c.want...)
		})
	}
}

func TestPythonEveryModuleLookupStartsAtTheImporter(t *testing.T) {
	helpers := "def make():\n    pass\n\n\nclass Maker:\n    def __init__(self):\n        pass\n"
	g := pyGraph(t, map[string]string{
		"tests/__init__.py":           "",
		"tests/helpers.py":            helpers,
		"tools/cli/tests/__init__.py": "",
		"tools/cli/tests/helpers.py":  helpers,
		"tools/cli/use.py": "import tests.helpers\nimport tests.helpers as h\nfrom tests import helpers\nfrom tests.helpers import Maker\n\n\n" +
			"def receiver():\n    tests.helpers.make()\n\n\ndef alias():\n    h.make()\n\n\ndef submodule():\n    helpers.make()\n\n\n" +
			"def construct():\n    h.Maker()\n\n\nclass Sub(Maker):\n    pass\n\n\nclass ViaModule(h.Maker):\n    pass\n",
		// A re-export at the repository root, followed from a file under
		// tools/cli/: the import in shim.py is looked up from shim.py.
		"shim.py":               "from tests.helpers import make\n",
		"tools/cli/via_shim.py": "from shim import make\n\n\ndef go():\n    make()\n",
	})
	cli := "tools/cli/tests/helpers.py"
	// The package and its submodule, a module receiver, an alias, a
	// submodule a from-import binds, a constructor through a module, and a
	// base named either way.
	wantTargets(t, g, "tools/cli/use.py", model.RelationImports, "tools/cli/tests/__init__.py", cli)
	wantCalls(t, g, "tools/cli/use.py#receiver", cli+"#make extracted")
	wantCalls(t, g, "tools/cli/use.py#alias", cli+"#make extracted")
	wantCalls(t, g, "tools/cli/use.py#submodule", cli+"#make extracted")
	wantCalls(t, g, "tools/cli/use.py#construct", cli+"#Maker.__init__ extracted")
	wantTargets(t, g, "tools/cli/use.py#Sub", model.RelationExtends, cli+"#Maker")
	wantTargets(t, g, "tools/cli/use.py#ViaModule", model.RelationExtends, cli+"#Maker")
	wantTargets(t, g, "tools/cli/via_shim.py", model.RelationImports, "shim.py")
	wantCalls(t, g, "tools/cli/via_shim.py#go", "tests/helpers.py#make extracted")
}

func TestPythonParentOfATopLevelPackageIsASourceRoot(t *testing.T) {
	g := pyGraph(t, map[string]string{
		// proj/pkg is a package whose parent is none: proj/ is where it is
		// imported from, as a project in a monorepo is.
		"proj/pkg/__init__.py": "",
		"proj/pkg/a.py":        "def f():\n    pass\n",
		"proj/main.py":         "from pkg.a import f\n\n\ndef run():\n    f()\n",
		"other.py":             "def f():\n    pass\n",
		// lib/sub lies in a package: lib/ is no root, and sub alone names
		// nothing.
		"lib/__init__.py":     "",
		"lib/sub/__init__.py": "",
		"lib/sub/b.py":        "",
		"use.py":              "import sub.b\nimport lib.sub.b\n",
	})
	wantTargets(t, g, "proj/main.py", model.RelationImports, "proj/pkg/a.py")
	wantCalls(t, g, "proj/main.py#run", "proj/pkg/a.py#f extracted")
	wantTargets(t, g, "use.py", model.RelationImports, "lib/sub/b.py")
}

func TestPythonUnresolvedModuleOfTheRepoStillAllowsTheGuess(t *testing.T) {
	g := pyGraph(t, map[string]string{
		// apps is a directory here that no source root reaches: the module is
		// the repository's, only unresolved, and the one helper is a guess.
		"backend/apps/users/models.py":    "def helper():\n    pass\n",
		"backend/apps/orders/services.py": "from apps.users.models import helper\n\n\ndef run():\n    helper()\n",
		// lib.util is left out as ambiguous across the roots one/ and two/,
		// neither of which holds amb.py, and the repository root maps no
		// lib.util; lib is still a directory here.
		"one/lib/__init__.py": "",
		"one/lib/util.py":     "def load():\n    pass\n",
		"two/lib/__init__.py": "",
		"two/lib/util.py":     "",
		"amb.py":              "from lib.util import load\n\n\ndef run():\n    load()\n",
		// tools is the name of a module file here, one no root reaches.
		"scripts/tools.py": "def tool():\n    pass\n",
		"t.py":             "from tools import tool\n\n\ndef run():\n    tool()\n",
		// One binding outside, one of the repository: the name may well be
		// the repository's.
		"mixed.py": "try:\n    from ujson import tool\nexcept ImportError:\n    from tools import tool\n\n\ndef run():\n    tool()\n",
		// A relative import names a place in the repository by its form.
		"pkg/m.py":     "from .missing import gone\n\n\ndef run():\n    gone()\n",
		"elsewhere.py": "def gone():\n    pass\n",
		// json names no directory and no module here: outside, no guess.
		"loader.py": "def loads():\n    pass\n",
		"j.py":      "from json import loads\n\n\ndef run():\n    loads()\n",
	})
	wantCalls(t, g, "backend/apps/orders/services.py#run", "backend/apps/users/models.py#helper inferred")
	wantCalls(t, g, "amb.py#run", "one/lib/util.py#load inferred")
	wantCalls(t, g, "t.py#run", "scripts/tools.py#tool inferred")
	wantCalls(t, g, "mixed.py#run", "scripts/tools.py#tool inferred")
	wantCalls(t, g, "pkg/m.py#run", "elsewhere.py#gone inferred")
	wantCalls(t, g, "j.py#run")
}

func TestPythonPackageBeatsModule(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"a.py":          "",
		"a/__init__.py": "",
		"main.py":       "import a\n",
		// The same pair one level down, reached by a relative import.
		"pkg/b.py":          "",
		"pkg/b/__init__.py": "",
		"pkg/m.py":          "from . import b\n",
	})
	// CPython's finder looks for the package directory before the module
	// file, so the package wins.
	wantTargets(t, g, "main.py", model.RelationImports, "a/__init__.py")
	// pkg has no __init__.py, so "." names no file; ".b" is the package.
	wantTargets(t, g, "pkg/m.py", model.RelationImports, "pkg/b/__init__.py")
}

func TestPythonRelativeImports(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"pkg/__init__.py":     "",
		"pkg/top.py":          "",
		"pkg/x.py":            "def y():\n    pass\n",
		"pkg/w/__init__.py":   "",
		"pkg/w/v.py":          "",
		"pkg/sub/__init__.py": "",
		"pkg/sub/sib.py":      "",
		"pkg/sub/m.py":        "from . import sib\nfrom .. import top\nfrom ..x import y\nfrom ..w import v\n",
	})
	// Each statement reaches the package it imports from, and each name that
	// is a module of that package reaches its file too; y is a function of
	// pkg/x.py and names no file.
	wantTargets(t, g, "pkg/sub/m.py", model.RelationImports,
		"pkg/__init__.py", "pkg/sub/__init__.py", "pkg/sub/sib.py", "pkg/top.py",
		"pkg/w/__init__.py", "pkg/w/v.py", "pkg/x.py")
}

func TestPythonRelativeImportAboveTheRootHasNoEdge(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"__init__.py": "",
		"top.py":      "",
		"m.py":        "from .. import top\n",
		"pkg/n.py":    "from ... import top\n",
		// One dot at the root stays at the root.
		"r.py": "from . import top\n",
	})
	wantTargets(t, g, "m.py", model.RelationImports)
	wantTargets(t, g, "pkg/n.py", model.RelationImports)
	wantTargets(t, g, "r.py", model.RelationImports, "__init__.py", "top.py")
}

func TestPythonImportOfSubmoduleName(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"pkg/__init__.py": "def helper():\n    pass\n",
		"pkg/a.py":        "",
		"main.py":         "from pkg import a, helper\n",
	})
	// a is a submodule and gets an edge of its own; helper is a function of
	// the package and adds nothing to the edge the package already has.
	wantTargets(t, g, "main.py", model.RelationImports, "pkg/__init__.py", "pkg/a.py")
}

func TestPythonExternalImportHasNoEdge(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"main.py": "import os\nimport os.path as p\nfrom django.db import models\nfrom . import nothing\n",
	})
	// Unlike Go, which keeps an outside package path as the target, a module
	// name that is no file here says nothing about where it would come from.
	wantTargets(t, g, "main.py", model.RelationImports)
}

func TestPythonImportEdgesAreDedupedAndNeverSelf(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"pkg/__init__.py": "from . import a\nfrom . import b\n",
		"pkg/a.py":        "",
		"pkg/b.py":        "",
		"pkg/c.py":        "from . import a\nfrom . import b\nimport pkg.a\n",
	})
	// "." inside the package's own __init__.py is the file itself.
	wantTargets(t, g, "pkg/__init__.py", model.RelationImports, "pkg/a.py", "pkg/b.py")
	// Two statements from one package and two ways to one module are one
	// dependency each.
	wantTargets(t, g, "pkg/c.py", model.RelationImports, "pkg/__init__.py", "pkg/a.py", "pkg/b.py")
}

func TestPythonModuleInTwoRoots(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"lib/__init__.py":     "",
		"lib/c.py":            "",
		"lib/m.py":            "from . import c\n",
		"src/lib/__init__.py": "",
		"src/lib/c.py":        "",
		"src/run.py":          "import lib.c\n",
		"main.py":             "import lib\nimport lib.c\nimport src.lib.c\n",
		// Two roots below the repository root, one/ and two/, each with a
		// package pkg, and none at the repository root.
		"one/pkg/__init__.py": "",
		"two/pkg/__init__.py": "",
		"one/use.py":          "import pkg\n",
		"tools/use.py":        "import pkg\n",
	})
	// lib and lib.c are a package at the root and one under src/. A file
	// finds the one of the deepest root that holds it and maps the path:
	// main.py the repository root's, src/run.py the one of src/.
	wantTargets(t, g, "main.py", model.RelationImports, "lib/__init__.py", "lib/c.py", "src/lib/c.py")
	wantTargets(t, g, "src/run.py", model.RelationImports, "src/lib/c.py")
	wantTargets(t, g, "one/use.py", model.RelationImports, "one/pkg/__init__.py")
	// tools/use.py lies under neither one/ nor two/, and the repository root
	// maps no pkg: which one Python finds depends on its search path, which
	// no file here states.
	wantTargets(t, g, "tools/use.py", model.RelationImports)
	// A relative import names a directory and is never ambiguous.
	wantTargets(t, g, "lib/m.py", model.RelationImports, "lib/__init__.py", "lib/c.py")
}

func TestPythonDottedFileNameIsNoModule(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"a.b.py":  "",
		"a/b.py":  "",
		"main.py": "import a.b\n",
	})
	// No import can name a.b.py: Python would look for b in a directory a.
	wantTargets(t, g, "main.py", model.RelationImports, "a/b.py")
}

func TestPythonExtendsSameFileAndImported(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py":   "class A:\n    def run(self):\n        pass\n\n\nclass B(A):\n    pass\n",
		"n.py":   "from m import A\n\n\nclass C(A):\n    pass\n",
		"mod.py": "class A:\n    pass\n",
		"o.py": "import mod\n\n\nclass C(mod.A):\n    pass\n\n\nclass D(Unknown):\n    pass\n\n\n" +
			"class E(mod.Missing):\n    pass\n\n\nclass F(nope.A):\n    pass\n",
		// Defined here after the import, A is this module's own.
		"p.py": "from m import A\n\n\nclass A:\n    pass\n\n\nclass L(A):\n    pass\n",
	})
	wantTargets(t, g, "m.py#B", model.RelationExtends, "m.py#A")
	wantTargets(t, g, "n.py#C", model.RelationExtends, "m.py#A")
	wantTargets(t, g, "o.py#C", model.RelationExtends, "mod.py#A")
	wantTargets(t, g, "o.py#D", model.RelationExtends)
	wantTargets(t, g, "o.py#E", model.RelationExtends)
	wantTargets(t, g, "o.py#F", model.RelationExtends)
	wantTargets(t, g, "p.py#L", model.RelationExtends, "p.py#A")
	// Containment is the extractor's and passes through.
	wantTargets(t, g, "m.py", model.RelationContains, "m.py#A", "m.py#B")
	wantTargets(t, g, "m.py#A", model.RelationContains, "m.py#A.run")
}

func TestPythonExtendsThroughAliasesAndRelativeImports(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"pkg/__init__.py": "class Base:\n    pass\n",
		"pkg/models.py":   "class User:\n    pass\n",
		"pkg/views.py": "from . import models\nfrom .models import User as U\nimport pkg.models\nimport pkg.models as pm\n\n\n" +
			"class V(models.User):\n    pass\n\n\nclass W(U):\n    pass\n\n\nclass X(pkg.models.User):\n    pass\n\n\n" +
			"class Y(pkg.Base):\n    pass\n\n\nclass Z(pm.User):\n    pass\n",
	})
	// A submodule bound by a from-import, a class bound under an alias, the
	// dotted path a plain import binds, the package that same import binds
	// with it, and a module alias.
	wantTargets(t, g, "pkg/views.py#V", model.RelationExtends, "pkg/models.py#User")
	wantTargets(t, g, "pkg/views.py#W", model.RelationExtends, "pkg/models.py#User")
	wantTargets(t, g, "pkg/views.py#X", model.RelationExtends, "pkg/models.py#User")
	wantTargets(t, g, "pkg/views.py#Y", model.RelationExtends, "pkg/__init__.py#Base")
	wantTargets(t, g, "pkg/views.py#Z", model.RelationExtends, "pkg/models.py#User")
}

func TestPythonExtendsNeedsExactlyOneClass(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "def f():\n    pass\n\n\nif X:\n    class Dup:\n        pass\nelse:\n    class Dup:\n        pass\n\n\n" +
			"class A:\n    pass\n\n\nclass G(f):\n    pass\n\n\nclass H(Dup):\n    pass\n",
		"n.py": "from m import *\nfrom ext import Base\nfrom django.db import models\n" +
			"try:\n    from fast import Impl\nexcept ImportError:\n    from m import A as Impl\n\n\n" +
			"class I(A):\n    pass\n\n\nclass J(Base):\n    pass\n\n\nclass K(models.Model):\n    pass\n\n\n" +
			"class M(Impl):\n    pass\n",
	})
	// A function is no base, two classes of one name are a guess, a
	// wildcard binds nothing this graph can name, and a module outside the
	// repository has no class to point at.
	for _, id := range []model.NodeID{"m.py#G", "m.py#H", "n.py#I", "n.py#J", "n.py#K"} {
		wantTargets(t, g, id, model.RelationExtends)
	}
	// A binding to no file here is passed over for the next one of the same
	// name.
	wantTargets(t, g, "n.py#M", model.RelationExtends, "m.py#A")
}

func TestBareCallSameModule(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "def foo():\n    pass\n\n\ndef bar():\n    foo()\n\n\nbar()\n",
		"n.py": "def foo():\n    pass\n",
	})
	// foo is defined twice in the repository; the module's own is the one.
	wantCalls(t, g, "m.py#bar", "m.py#foo extracted")
	// A call at module level belongs to the file.
	wantCalls(t, g, "m.py", "m.py#bar extracted")
}

func TestBareCallViaFromImport(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "def foo():\n    pass\n",
		// A second foo, so that no edge below can come from the repo-wide
		// guess.
		"n.py": "def foo():\n    pass\n",
		"a.py": "from m import foo\n\n\ndef run():\n    foo()\n",
		"b.py": "from m import foo as bar\n\n\ndef run():\n    bar()\n",
	})
	wantCalls(t, g, "a.py#run", "m.py#foo extracted")
	wantCalls(t, g, "b.py#run", "m.py#foo extracted")
}

func TestBareCallUniqueName(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "def foo():\n    pass\n",
		"a.py": "def run():\n    foo()\n",
		// A wildcard binds nothing the resolver can name, so its names are
		// found the same way.
		"b.py": "from m import *\n\n\ndef run():\n    foo()\n",
	})
	wantCalls(t, g, "a.py#run", "m.py#foo inferred")
	wantCalls(t, g, "b.py#run", "m.py#foo inferred")
}

func TestBareCallAmbiguousDropped(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "def foo():\n    pass\n",
		"n.py": "class foo:\n    pass\n",
		// The module m binds no bar, and nothing in the repository is bar.
		"a.py": "from m import bar\n\n\ndef run():\n    foo()\n    bar()\n    nothing()\n",
	})
	wantCalls(t, g, "a.py#run")
}

func TestBareCallSkipsBuiltins(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"store.py": "def open(path):\n    pass\n\n\ndef read(path):\n    open(path)\n",
		"out.py":   "def print(text):\n    pass\n",
		"main.py":  "def run():\n    open('x')\n    print('y')\n",
		"use.py":   "from store import open\n\n\ndef run():\n    open('x')\n",
	})
	// A module that defines or imports its own open means its own.
	wantCalls(t, g, "store.py#read", "store.py#open extracted")
	wantCalls(t, g, "use.py#run", "store.py#open extracted")
	// Anywhere else open and print are Python's, and one repo function of
	// the name must not collect every call of it.
	wantCalls(t, g, "main.py#run")
}

func TestBareCallBoundOutsideTheRepoHasNoEdge(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"loader.py": "def load(path):\n    pass\n\n\ndef helper():\n    pass\n",
		"main.py":   "from json import load\n\n\ndef run(f):\n    load(f)\n    helper()\n",
		"alias.py":  "from json import loads as load\n\n\ndef run(f):\n    load(f)\n",
		// Both bindings name modules outside the repository.
		"twice.py": "try:\n    from ujson import load\nexcept ImportError:\n    from json import load\n\n\n" +
			"def run(f):\n    load(f)\n",
	})
	// The file says load is json's: the one load of the repository is not
	// it. A name the file does not bind is still guessed.
	wantCalls(t, g, "main.py#run", "loader.py#helper inferred")
	wantCalls(t, g, "alias.py#run")
	wantCalls(t, g, "twice.py#run")
}

func TestBareCallFollowsAReExport(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"pkg/__init__.py": "from .impl import foo\n",
		"pkg/impl.py":     "def foo():\n    pass\n",
		"main.py":         "from pkg import foo\n\n\ndef run():\n    foo()\n",
		// One binding outside the repository, one to a file here that passes
		// on the load of json: the chain ends outside the repository too, and
		// the one load here is not it.
		"compat.py": "from json import load\n",
		"loader.py": "def load(path):\n    pass\n",
		"use.py": "try:\n    from ujson import load\nexcept ImportError:\n    from compat import load\n\n\n" +
			"def run(f):\n    load(f)\n",
	})
	// pkg does not define foo itself but imports it from pkg/impl.py, and
	// that is the foo main.py calls.
	wantCalls(t, g, "main.py#run", "pkg/impl.py#foo extracted")
	wantCalls(t, g, "use.py#run")
}

func TestPythonReExportThroughAPackage(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"lib/__init__.py": "from .impl import Base, work, Made\nfrom .impl import work as job\n",
		"lib/impl.py": "class Base:\n    def m(self):\n        pass\n\n\ndef work():\n    pass\n\n\n" +
			"class Made:\n    def __init__(self):\n        pass\n",
		// A second of every name, so that no edge below can come from a guess.
		"other.py": "class Base:\n    def m(self):\n        pass\n\n\ndef work():\n    pass\n\n\ndef job():\n    pass\n\n\n" +
			"class Made:\n    pass\n",
		"app.py": "import lib\nfrom lib import Base, work, job, Made\n\n\nclass C(Base):\n    def run(self):\n        self.m()\n\n\n" +
			"class D(lib.Base):\n    pass\n\n\ndef via_module():\n    lib.work()\n\n\ndef bare():\n    work()\n\n\n" +
			"def alias():\n    job()\n\n\ndef construct():\n    Made()\n\n\ndef module_construct():\n    lib.Made()\n\n\n" +
			"def class_receiver():\n    lib.Base.m(None)\n",
	})
	// lib/__init__.py defines none of the names; its own from-import binds
	// each to lib/impl.py, where the definition is, for a base, a self call
	// through that base, a module selector, a bare call, an alias the
	// package gives, and a constructor reached either way.
	wantTargets(t, g, "app.py#C", model.RelationExtends, "lib/impl.py#Base")
	wantTargets(t, g, "app.py#D", model.RelationExtends, "lib/impl.py#Base")
	wantCalls(t, g, "app.py#C.run", "lib/impl.py#Base.m extracted")
	wantCalls(t, g, "app.py#via_module", "lib/impl.py#work extracted")
	wantCalls(t, g, "app.py#bare", "lib/impl.py#work extracted")
	wantCalls(t, g, "app.py#alias", "lib/impl.py#work extracted")
	wantCalls(t, g, "app.py#construct", "lib/impl.py#Made.__init__ extracted")
	wantCalls(t, g, "app.py#module_construct", "lib/impl.py#Made.__init__ extracted")
	wantCalls(t, g, "app.py#class_receiver", "lib/impl.py#Base.m extracted")
}

func TestPythonReExportThatCannotBeFollowed(t *testing.T) {
	g := pyGraph(t, map[string]string{
		// A wildcard binds no name the resolver can follow: a dead end inside
		// the repository, where the one bar of the repository is a guess.
		"wild/__init__.py": "from .impl import *\n",
		"wild/impl.py":     "def bar():\n    pass\n",
		"w.py":             "from wild import bar\n\n\ndef run():\n    bar()\n",
		// Two modules passing x on to each other: a cycle, no edge, and no
		// guess at the one x elsewhere.
		"a.py":     "from b import x\n",
		"b.py":     "from a import x\n",
		"cycle.py": "from a import x\n\n\ndef run():\n    x()\n",
		"other.py": "def x():\n    pass\n\n\ndef y():\n    pass\n",
		// The chain ends in two definitions of foo: a guess between them, and
		// no guess at the y the name is bound to here either.
		"amb.py":   "if X:\n    def foo():\n        pass\nelse:\n    def foo():\n        pass\n",
		"reexp.py": "from amb import foo as y\n",
		"amb2.py":  "from reexp import y\n\n\ndef run():\n    y()\n",
	})
	wantCalls(t, g, "w.py#run", "wild/impl.py#bar inferred")
	wantCalls(t, g, "cycle.py#run")
	wantCalls(t, g, "amb2.py#run")
}

func TestPythonReExportChainIsFollowedEightDeep(t *testing.T) {
	tree := map[string]string{
		"use.py": "from m0 import f\nfrom n0 import g\n\n\ndef eight():\n    f()\n\n\ndef nine():\n    g()\n",
		"m8.py":  "def f():\n    pass\n",
		"n9.py":  "def g():\n    pass\n",
	}
	for i := range 9 {
		tree[fmt.Sprintf("n%d.py", i)] = fmt.Sprintf("from n%d import g\n", i+1)
		if i < 8 {
			tree[fmt.Sprintf("m%d.py", i)] = fmt.Sprintf("from m%d import f\n", i+1)
		}
	}
	g := pyGraph(t, tree)
	// Eight re-exports, m0 to m8, are followed; the ninth is not, and the
	// g of n9 is then no guess either, though it is the only g.
	wantCalls(t, g, "use.py#eight", "m8.py#f extracted")
	wantCalls(t, g, "use.py#nine")
}

func TestSelfCallOwnClass(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class A:\n    def run(self):\n        self.step()\n\n    def step(self):\n        pass\n\n" +
			"    @classmethod\n    def make(cls):\n        cls.step()\n\n\n" +
			"class B:\n    def step(self):\n        pass\n",
		// Two classes of one name: each self is the class the method is in.
		"n.py": "if X:\n    class Dup:\n        def a(self):\n            self.b()\n\n        def b(self):\n            pass\n" +
			"else:\n    class Dup:\n        def a(self):\n            self.b()\n\n        def b(self):\n            pass\n",
	})
	wantCalls(t, g, "m.py#A.run", "m.py#A.step extracted")
	wantCalls(t, g, "m.py#A.make", "m.py#A.step extracted")
	wantCalls(t, g, "n.py#Dup.a", "n.py#Dup.b extracted")
	wantCalls(t, g, "n.py#Dup.a~2", "n.py#Dup.b~2 extracted")
}

func TestSelfCallBaseClass(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"base.py": "class Base:\n    def helper(self):\n        pass\n\n    def shared(self):\n        pass\n",
		"child.py": "from base import Base\n\n\nclass Child(Base):\n    def run(self):\n        self.helper()\n" +
			"        self.shared()\n\n    def shared(self):\n        pass\n",
		"grand.py": "import child\n\n\nclass Grand(child.Child):\n    def go(self):\n        self.helper()\n" +
			"        self.shared()\n",
		// With two bases, the first one written is searched first.
		"multi.py": "class X:\n    def f(self):\n        pass\n\n\nclass Y:\n    def f(self):\n        pass\n\n" +
			"    def g(self):\n        pass\n\n\nclass M(X, Y):\n    def run(self):\n        self.f()\n        self.g()\n",
	})
	// The base in another module, and the class's own method before the
	// base's.
	wantCalls(t, g, "child.py#Child.run", "base.py#Base.helper extracted", "child.py#Child.shared extracted")
	// Two levels up.
	wantCalls(t, g, "grand.py#Grand.go", "base.py#Base.helper extracted", "child.py#Child.shared extracted")
	wantCalls(t, g, "multi.py#M.run", "multi.py#X.f extracted", "multi.py#Y.g extracted")
}

func TestSelfCallTwiceDefinedShadowsTheBase(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class P:\n    def f(self):\n        pass\n\n\nclass Q(P):\n    def f(self):\n        pass\n\n" +
			"    def f(self):\n        pass\n\n    def run(self):\n        self.f()\n",
	})
	// Q defines f itself, twice: which one runs is a guess, and P's is never
	// the one.
	wantCalls(t, g, "m.py#Q.run")
}

func TestSelfCallUnknownDropped(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class A(Unknown):\n    def run(self):\n        self.missing()\n        self.attr.method()\n\n\n" +
			"def missing():\n    pass\n\n\nclass B:\n    def method(self):\n        pass\n",
	})
	// Not in the class, and its base is no class here: a module function of
	// the name, or a unique method elsewhere, is not what self means.
	wantCalls(t, g, "m.py#A.run")
}

func TestCyclicBasesTerminate(t *testing.T) {
	files := pyFiles(t, map[string]string{
		"m.py": "class A(B):\n    def run(self):\n        self.missing()\n\n\nclass B(A):\n    pass\n",
		"x.py": "from y import Y\n\n\nclass X(Y):\n    def run(self):\n        self.missing()\n        self.found()\n",
		"y.py": "from x import X\n\n\nclass Y(X):\n    def found(self):\n        pass\n",
	})
	// A walk that loops forever would hang the whole test binary; this
	// fails the one test instead.
	done := make(chan *model.Graph, 1)
	go func() { done <- resolve.Graph(files, nil, stamp) }()
	var g *model.Graph
	select {
	case g = <-done:
	case <-time.After(time.Minute):
		t.Fatal("resolving calls over cyclic bases did not end")
	}
	validated(t, g)
	wantTargets(t, g, "m.py#A", model.RelationExtends, "m.py#B")
	wantTargets(t, g, "m.py#B", model.RelationExtends, "m.py#A")
	wantCalls(t, g, "m.py#A.run")
	wantCalls(t, g, "x.py#X.run", "y.py#Y.found extracted")
}

func TestPythonClassNeverExtendsItself(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"base.py": "class S:\n    def foo(self):\n        pass\n\n\nclass W:\n    def foo(self):\n        pass\n",
		// The base is read before the name is bound to the class being
		// defined: it is the S the import bound.
		"sub.py": "from base import S\n\n\nclass S(S):\n    def run(self):\n        self.foo()\n",
		// With no other S, the name is not bound yet: no base.
		"lone.py": "class S(S):\n    def run(self):\n        self.foo()\n",
		// Two classes T in one module, and which of them the second means
		// needs the order the resolver does not read: no base.
		"redef.py": "class T:\n    def foo(self):\n        pass\n\n\nclass T(T):\n    def run(self):\n        self.foo()\n",
		// A W defined after the class that names it is not its base; the
		// two are ambiguous, and the import decides.
		"fwd.py": "from base import W\n\n\nclass W(W):\n    def run(self):\n        self.foo()\n\n\nclass W:\n    pass\n",
		// Each branch's W extends the W the import bound, not the other
		// branch's.
		"branch.py": "from base import W\n\nif X:\n    class W(W):\n        pass\nelse:\n    class W(W):\n        pass\n",
		// The same with a base outside the repository: no edge, and above
		// all no cycle between the two branches.
		"ext.py": "from django.test import TestCase\n\nif X:\n    class TestCase(TestCase):\n        pass\n" +
			"else:\n    class TestCase(TestCase):\n        pass\n",
	})
	wantTargets(t, g, "sub.py#S", model.RelationExtends, "base.py#S")
	wantCalls(t, g, "sub.py#S.run", "base.py#S.foo extracted")
	wantTargets(t, g, "lone.py#S", model.RelationExtends)
	wantCalls(t, g, "lone.py#S.run")
	wantTargets(t, g, "redef.py#T~2", model.RelationExtends)
	wantCalls(t, g, "redef.py#T.run")
	wantTargets(t, g, "fwd.py#W", model.RelationExtends, "base.py#W")
	wantCalls(t, g, "fwd.py#W.run", "base.py#W.foo extracted")
	wantTargets(t, g, "fwd.py#W~2", model.RelationExtends)
	wantTargets(t, g, "branch.py#W", model.RelationExtends, "base.py#W")
	wantTargets(t, g, "branch.py#W~2", model.RelationExtends, "base.py#W")
	wantTargets(t, g, "ext.py#TestCase", model.RelationExtends)
	wantTargets(t, g, "ext.py#TestCase~2", model.RelationExtends)
}

func TestModuleReceiverCall(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py":          "def foo():\n    pass\n",
		"a/__init__.py": "def top():\n    pass\n",
		"a/b.py":        "def foo():\n    pass\n",
		"use.py": "import m\nimport a.b\nimport a.b as c\nfrom a import b as bee\n\n\n" +
			"def via_module():\n    m.foo()\n\n\ndef via_chain():\n    a.b.foo()\n\n\n" +
			"def via_alias():\n    c.foo()\n\n\ndef via_package():\n    a.top()\n\n\n" +
			"def via_submodule():\n    bee.foo()\n\n\n" +
			"def unresolved():\n    m.missing()\n    m.top()\n    os.getcwd()\n",
	})
	wantCalls(t, g, "use.py#via_module", "m.py#foo extracted")
	wantCalls(t, g, "use.py#via_chain", "a/b.py#foo extracted")
	wantCalls(t, g, "use.py#via_alias", "a/b.py#foo extracted")
	// import a.b binds a as well.
	wantCalls(t, g, "use.py#via_package", "a/__init__.py#top extracted")
	wantCalls(t, g, "use.py#via_submodule", "a/b.py#foo extracted")
	// A name the module does not define is not looked for anywhere else,
	// and a module that is no file here has nothing to point at.
	wantCalls(t, g, "use.py#unresolved")
}

func TestClassReceiverCall(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"models.py": "class Base:\n    def make(self):\n        pass\n\n\nclass User(Base):\n" +
			"    @staticmethod\n    def guest():\n        pass\n\n\ndef helper():\n    pass\n",
		"use.py": "import models\nfrom models import User as U\n\n\nclass Local:\n    def ping(self):\n        pass\n\n\n" +
			"def same_module():\n    Local.ping(None)\n\n\ndef imported():\n    U.guest()\n\n\n" +
			"def through_module():\n    models.User.guest()\n\n\ndef inherited():\n    U.make(None)\n\n\n" +
			"def unresolved():\n    U.missing()\n    models.helper.attr()\n",
	})
	wantCalls(t, g, "use.py#same_module", "use.py#Local.ping extracted")
	wantCalls(t, g, "use.py#imported", "models.py#User.guest extracted")
	wantCalls(t, g, "use.py#through_module", "models.py#User.guest extracted")
	// The class's own methods first, then its bases, as for self.
	wantCalls(t, g, "use.py#inherited", "models.py#Base.make extracted")
	// A method the class and its bases lack, and a receiver that is a
	// function and no class.
	wantCalls(t, g, "use.py#unresolved")
}

func TestUnknownReceiverDropped(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class Foo:\n    def foo(self):\n        pass\n\n\ndef foo():\n    pass\n\n\n" +
			"def run(obj):\n    obj.foo()\n    obj.inner.foo()\n",
	})
	// foo is unique as a method and as a function, and still neither is
	// what obj.foo means.
	wantCalls(t, g, "m.py#run")
}

func TestConstructorHitsInit(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class Foo:\n    def __init__(self):\n        pass\n\n\ndef make():\n    Foo()\n",
		"n.py": "def other():\n    Foo()\n",
	})
	wantCalls(t, g, "m.py#make", "m.py#Foo.__init__ extracted")
	// Found by the repo-wide guess, the constructor keeps that guess.
	wantCalls(t, g, "n.py#other", "m.py#Foo.__init__ inferred")
}

func TestConstructorWithoutInitHitsClass(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class Foo:\n    def run(self):\n        pass\n\n\n" +
			"class Twice:\n    def __init__(self):\n        pass\n\n    def __init__(self, a):\n        pass\n\n\n" +
			"def make():\n    Foo()\n\n\ndef make_twice():\n    Twice()\n",
	})
	wantCalls(t, g, "m.py#make", "m.py#Foo extracted")
	// Two __init__ in one class: which one runs is a guess, the class is
	// not.
	wantCalls(t, g, "m.py#make_twice", "m.py#Twice extracted")
}

func TestConstructorInheritedInitHitsClass(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class Base:\n    def __init__(self):\n        pass\n\n\nclass Foo(Base):\n    pass\n\n\ndef make():\n    Foo()\n",
	})
	// Only an __init__ the class defines itself stands for its constructor.
	wantCalls(t, g, "m.py#make", "m.py#Foo extracted")
}

func TestImportedConstructor(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py": "class Foo:\n    def __init__(self):\n        pass\n\n\nclass Bar:\n    pass\n",
		// A second Foo, so that the edge can only come from the import.
		"n.py":   "class Foo:\n    pass\n",
		"use.py": "from m import Foo, Bar as B\n\n\ndef make():\n    Foo()\n    B()\n",
	})
	wantCalls(t, g, "use.py#make", "m.py#Bar extracted", "m.py#Foo.__init__ extracted")
}

func TestModuleConstructor(t *testing.T) {
	g := pyGraph(t, map[string]string{
		"m.py":   "class Foo:\n    def __init__(self):\n        pass\n\n\nclass Bar:\n    pass\n",
		"use.py": "import m\n\n\ndef make():\n    m.Foo()\n    m.Bar()\n",
	})
	wantCalls(t, g, "use.py#make", "m.py#Bar extracted", "m.py#Foo.__init__ extracted")
}
