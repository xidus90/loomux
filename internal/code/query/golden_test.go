package query_test

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/gitenv"
)

var update = flag.Bool("update", false, "update golden files")

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyDir: %v", err)
	}
}

func checkGolden(t *testing.T, goldenPath, actual string) {
	t.Helper()
	// Normalize CRLF to LF for deterministic comparison across platforms
	actual = strings.ReplaceAll(actual, "\r\n", "\n")
	if *update {
		if err := os.WriteFile(goldenPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	expectedBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update to generate)", err)
	}
	expected := strings.ReplaceAll(string(expectedBytes), "\r\n", "\n")
	if actual != expected {
		t.Errorf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", filepath.Base(goldenPath), actual, expected)
	}
}

func TestGolden(t *testing.T) {
	casesDir := filepath.Join("..", "..", "..", "testdata", "cases", "graph")
	repoSrc := filepath.Join(casesDir, "repo")

	root := t.TempDir()
	copyDir(t, repoSrc, root)

	// Build graph
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatalf("build: %v", err)
	}

	// 1. Callers
	callersAns, _, err := query.Callers(root, "Add", query.CallersOptions{
		Direction: blast.In,
		Depth:     1,
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("callers: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "callers.golden"), query.CallersReport(callersAns))

	// 2. Skeleton
	skelAns, _, err := query.Skeleton(root, "calc/calc.go", query.SkeletonOptions{
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("skeleton: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "skeleton.golden"), query.SkeletonReport(skelAns))

	// 3. Grep
	grepAns, _, err := query.Grep(root, "Add", query.GrepOptions{
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("grep: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "grep.golden"), query.GrepReport(grepAns))

	// 4. Map
	mapAns, _, err := query.Map(root, query.MapOptions{
		MaxDirs:   16,
		NoRefresh: true,
	})
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "map.golden"), query.MapReport(mapAns))

	// 5. Stats
	statsAns, err := query.GraphStats(root)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	// Normalise Wiring size as it depends on file system/newline encoding
	statsReport := query.StatsReport(statsAns)
	lines := strings.Split(strings.ReplaceAll(statsReport, "\r\n", "\n"), "\n")
	var normLines []string
	for _, l := range lines {
		if strings.HasPrefix(l, "  Wiring size:") {
			normLines = append(normLines, "  Wiring size:  <normalized>")
		} else {
			normLines = append(normLines, l)
		}
	}
	checkGolden(t, filepath.Join(casesDir, "stats.golden"), strings.Join(normLines, "\n"))
}

func TestGoldenBlast(t *testing.T) {
	casesDir := filepath.Join("..", "..", "..", "testdata", "cases", "graph")
	root := t.TempDir()
	copyDir(t, filepath.Join(casesDir, "repo"), root)
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "."}, {"commit", "-qm", "init"},
	} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = gitenv.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Line 5 of calc/calc.go is Add's body.
	calc := filepath.Join(root, "calc", "calc.go")
	data, err := os.ReadFile(calc)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "\treturn a + b\n", "\treturn b + a\n", 1)
	if edited == string(data) {
		t.Fatal("calc.go no longer holds Add's body")
	}
	if err := os.WriteFile(calc, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	ans, _, err := query.Blast(root, query.BlastOptions{NoRefresh: true})
	if err != nil {
		t.Fatalf("blast: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "blast.golden"), query.BlastReport(ans))

	// Staged, the same change is an audit finding at threshold 2.
	add := exec.Command("git", "-C", root, "add", ".")
	add.Env = gitenv.Environ()
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	audit, _, err := query.Audit(root, query.AuditOptions{Cached: true, Threshold: 2})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	checkGolden(t, filepath.Join(casesDir, "audit.golden"), query.AuditReport(audit, 2))
}

// graphText is a graph as text, one line per node and one per edge, in the
// graph's own order.
func graphText(g *model.Graph) string {
	var b strings.Builder
	for _, n := range g.Nodes {
		fmt.Fprintf(&b, "node %s %s %s exported=%t sig=%s\n", n.ID, n.Kind, n.Span, n.Exported, n.Signature)
	}
	for _, e := range g.Edges {
		fmt.Fprintf(&b, "edge %s -%s/%s-> %s\n", e.Source, e.Relation, e.Confidence, e.Target)
	}
	return b.String()
}

// mixedCase is the Go and Python repository whose names clash on purpose:
// a Go function Run and a Python one, a Go type T with a method Run and a
// Python class T with one.
func mixedCase() string {
	return filepath.Join("..", "..", "..", "testdata", "cases", "graph", "mixed")
}

// buildCopy builds the graph of a copy of the fixture repository, less the
// directories named in drop.
func buildCopy(t *testing.T, repo string, drop ...string) *model.Graph {
	t.Helper()
	root := t.TempDir()
	copyDir(t, repo, root)
	for _, d := range drop {
		if err := os.RemoveAll(filepath.Join(root, d)); err != nil {
			t.Fatal(err)
		}
	}
	g, _, err := query.Build(root, func(string) {})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return g
}

// pythonCase is the Python repository that shows every rule of the Python
// extractor and resolver: a relative import, a from-import, a self call into
// the base class, a module and a class receiver, a constructor that reaches
// __init__ and one that reaches a class without its own, an unknown receiver,
// a nested function, a decorator, a dunder, a builtin a repo function
// shadows, a src root, a package that passes names on from its __init__.py,
// a Django project under backend/manage.py, a sibling module a script
// imports by its bare name, the test paths (tests/, test/, Django's tests.py)
// and a file the parser rejects in part.
func pythonCase() string {
	return filepath.Join("..", "..", "..", "testdata", "cases", "graph", "python")
}

func TestPythonGolden(t *testing.T) {
	root := t.TempDir()
	copyDir(t, filepath.Join(pythonCase(), "repo"), root)
	g, stats, err := query.Build(root, func(string) {})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	checkGolden(t, filepath.Join(pythonCase(), "graph.golden"), graphText(g))
	// What `graph build` reports as the parse errors of a language: its
	// report is the cli package's, the numbers are these.
	py := stats.PerLanguage["python"]
	if py.ParseErrors != 1 || !reflect.DeepEqual(py.ErrorFiles, []string{"app/broken.py"}) {
		t.Errorf("python parse errors = %d in %q, want 1 in [app/broken.py]", py.ParseErrors, py.ErrorFiles)
	}
	// The files the blast audit counts as tests, and no other.
	var tests []string
	for _, n := range g.Nodes {
		if n.Kind == "file" && blast.IsTestPath(n.Path) {
			tests = append(tests, n.Path)
		}
	}
	if want := []string{"shop/tests.py", "test/helper.py", "tests/test_service.py"}; !reflect.DeepEqual(tests, want) {
		t.Errorf("test paths %q, want %q", tests, want)
	}
}

func TestMixedGolden(t *testing.T) {
	g := buildCopy(t, filepath.Join(mixedCase(), "repo"))
	checkGolden(t, filepath.Join(mixedCase(), "graph.golden"), graphText(g))
}

func TestMixedIndexesStayApart(t *testing.T) {
	mixed := buildCopy(t, filepath.Join(mixedCase(), "repo"))
	goOnly := buildCopy(t, filepath.Join(mixedCase(), "repo"), "tool")

	// Every edge out of Go is what the Go files alone give: the Python files
	// beside them change none of it.
	goEdges := func(g *model.Graph) []model.Edge {
		var out []model.Edge
		for _, e := range g.Edges {
			file, _, _ := strings.Cut(string(e.Source), "#")
			if strings.HasSuffix(file, ".go") {
				out = append(out, e)
			}
		}
		return out
	}
	if got, want := goEdges(mixed), goEdges(goOnly); !reflect.DeepEqual(got, want) {
		t.Errorf("Go edges with the Python files =\n%v\nwithout them =\n%v", got, want)
	}
	// The edge that shows it: one index across both languages would find two
	// functions named Run and drop the call.
	if !strings.Contains(graphText(mixed), "edge main.go#main -calls/inferred-> run.go#Run\n") {
		t.Errorf("the Go call to Run must resolve beside the Python Run; graph:\n%s", graphText(mixed))
	}
	if !reflect.DeepEqual(mixed.Meta.Languages, []string{"go", "python"}) {
		t.Errorf("Meta.Languages = %v, want [go python]", mixed.Meta.Languages)
	}
}
