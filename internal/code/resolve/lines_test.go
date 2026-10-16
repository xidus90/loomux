package resolve_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/gdscript"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// lineOf is the Line of the one edge of the given shape.
func lineOf(t *testing.T, g *model.Graph, src, dst model.NodeID, rel model.Relation) int {
	t.Helper()
	var lines []int
	for _, e := range g.Edges {
		if e.Source == src && e.Target == dst && e.Relation == rel {
			lines = append(lines, e.Line)
		}
	}
	if len(lines) != 1 {
		t.Fatalf("%s -%s-> %s: got lines %v, want exactly one edge", src, rel, dst, lines)
	}
	return lines[0]
}

func TestEdgeCarriesTheSmallestLine(t *testing.T) {
	gd := gdGraph(t, map[string]string{
		"p/project.godot": "config_version=5\n",
		"p/foo.gd":        "class_name Foo\n\nstatic func a():\n\tpass\n",
		"p/user.gd":       "func f():\n\tpass\n\tpass\n\tpass\n\tFoo.a()\n\tpass\n\tpass\n\tpass\n\tFoo.a()\n",
	})
	if got := lineOf(t, gd, "p/user.gd#f", "p/foo.gd#Foo.a", model.RelationCalls); got != 5 {
		t.Fatalf("gdscript call line = %d, want 5", got)
	}
	py := pyGraph(t, map[string]string{
		"a.py": "x = 1\n",
		"b.py": "\n\n\n\nimport a\n\n\n\nimport a\n",
	})
	if got := lineOf(t, py, "b.py", "a.py", model.RelationImports); got != 5 {
		t.Fatalf("python import line = %d, want 5", got)
	}
}

func gdFiles(t *testing.T, tree map[string]string) []extract.Result {
	t.Helper()
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var files []extract.Result
	for _, p := range paths {
		r, err := gdscript.File(p, tree[p])
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, r)
	}
	return files
}

func sortedUnresolved(t *testing.T, files []extract.Result) []resolve.Unresolved {
	t.Helper()
	g, un := resolve.GraphWith(files, nil, stamp)
	validated(t, g)
	sort.Slice(un, func(i, j int) bool {
		if un[i].Source != un[j].Source {
			return un[i].Source < un[j].Source
		}
		return un[i].Name < un[j].Name
	})
	return un
}

func TestUnresolvedBases(t *testing.T) {
	got := sortedUnresolved(t, gdFiles(t, map[string]string{
		"p/project.godot": "config_version=5\n",
		"p/engine.gd":     "extends Node\n",
		"p/dup1.gd":       "class_name Dup\n",
		"p/dup2.gd":       "class_name Dup\n",
		"p/amb.gd":        "extends Dup\n",
		"p/ok.gd":         "class_name Ok\n",
		"p/good.gd":       "extends Ok\n",
		"p/missing.gd":    "extends \"res://missing.gd\"\n",
		"p/inner.gd":      "class A:\n\tpass\n\nclass B extends A.Nope:\n\tpass\n",
	}))
	want := []resolve.Unresolved{
		{Source: "p/amb.gd", File: "p/amb.gd", Line: 1, Name: "Dup", Reason: resolve.ReasonAmbiguous},
		{Source: "p/engine.gd", File: "p/engine.gd", Line: 1, Name: "Node", Reason: resolve.ReasonNone},
		{Source: "p/inner.gd#B", File: "p/inner.gd", Line: 4, Name: "A.Nope", Reason: resolve.ReasonNone},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("gdscript: got %+v, want %+v", got, want)
	}

	got = sortedUnresolved(t, pyFiles(t, map[string]string{
		"k.py": "class K(object, Missing):\n    pass\n",
		"m.py": "class A:\n    pass\n",
		"n.py": "class A:\n    pass\n",
		"o.py": "from m import A\nclass Q(A):\n    pass\n",
	}))
	wantPy := []resolve.Unresolved{
		{Source: "k.py#K", File: "k.py", Line: 1, Name: "Missing", Reason: resolve.ReasonNone},
		{Source: "k.py#K", File: "k.py", Line: 1, Name: "object", Reason: resolve.ReasonNone},
	}
	// Missing sorts before object only by byte order; compare as sets.
	if len(got) != 2 || !(reflect.DeepEqual(got, wantPy) || reflect.DeepEqual(got, []resolve.Unresolved{wantPy[1], wantPy[0]})) {
		t.Fatalf("python: got %+v, want %+v", got, wantPy)
	}
}

func TestUnresolvedInnerClassAmbiguity(t *testing.T) {
	got := sortedUnresolved(t, gdFiles(t, map[string]string{
		"a.gd": "class A:\n\tpass\n\nclass A:\n\tpass\n\nclass B extends A:\n\tpass\n",
	}))
	want := []resolve.Unresolved{{Source: "a.gd#B", File: "a.gd", Line: 7, Name: "A", Reason: resolve.ReasonAmbiguous}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestUnresolvedPythonModuleAmbiguityWithoutImport(t *testing.T) {
	got := sortedUnresolved(t, pyFiles(t, map[string]string{
		"m.py": "class A:\n    pass\nclass A:\n    pass\nclass Q(A):\n    pass\n",
	}))
	want := []resolve.Unresolved{{Source: "m.py#Q", File: "m.py", Line: 5, Name: "A", Reason: resolve.ReasonAmbiguous}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestUnresolvedPythonAmbiguity(t *testing.T) {
	got := sortedUnresolved(t, pyFiles(t, map[string]string{
		"m.py": "class A:\n    pass\nclass A:\n    pass\n",
		"u.py": "from m import A\n\nclass Q(A):\n    pass\n",
	}))
	want := []resolve.Unresolved{{Source: "u.py#Q", File: "u.py", Line: 3, Name: "A", Reason: resolve.ReasonAmbiguous}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestModulesFromMatchesModules(t *testing.T) {
	root := t.TempDir()
	bodies := map[string]string{
		"go.mod":       "// c\nmodule example.com/repo\n\ngo 1.25.0\n",
		"tools/go.mod": "module example.com/repo/tools\n",
		"x/go.mod":     "go 1.25.0\n",
	}
	var rels []string
	for rel, body := range bodies {
		rels = append(rels, rel)
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(rels)
	disk, err := resolve.Modules(root, append(rels, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	bodies["main.go"] = "module not.a.gomod\n"
	mem, err := resolve.ModulesFrom(bodies)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(disk, mem) || len(mem) != 2 {
		t.Fatalf("from contents %+v, from disk %+v", mem, disk)
	}
}

func TestGraphSchemaVersionIs3(t *testing.T) {
	if g := resolve.Graph(nil, nil, stamp); g.Meta.Version != 3 {
		t.Fatalf("version = %d, want 3", g.Meta.Version)
	}
}
