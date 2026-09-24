package hooks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/code/store"
)

// graphRepo is a copy of the recorded graph case with its graph built:
// main.go and calc/calc_test.go call calc.Add, nothing calls calc.Sub.
func graphRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join("..", "..", "testdata", "cases", "graph", "repo")
	for _, rel := range []string{"go.mod", "main.go", "calc/calc.go", "calc/calc_test.go"} {
		b, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		writeRepoFile(t, root, rel, string(b))
	}
	buildGraph(t, root)
	return root
}

func writeRepoFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func buildGraph(t *testing.T, root string) {
	t.Helper()
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatal(err)
	}
}

// fromDisk reads a repo-relative path under root, as the hook does.
func fromDisk(root string) func(string) ([]byte, error) {
	return func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	}
}

// fixed answers every read with one source text.
func fixed(source string) func(string) ([]byte, error) {
	return func(string) ([]byte, error) { return []byte(source), nil }
}

const calcHead = "package calc\n\n"

const addSource = "// Add sums two numbers.\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"

const subSource = "// Sub subtracts b from a.\nfunc Sub(a, b int) int {\n\treturn a - b\n}\n"

func TestBlastAsideNamesTheCallersOfAChangedFunction(t *testing.T) {
	root := graphRepo(t)
	edited := calcHead + strings.Replace(addSource, "a + b", "b + a", 1) + "\n" + subSource
	got := blastAside(root, "calc/calc.go", fixed(edited))
	if !strings.HasPrefix(got, "[graph] calc/calc.go: changed Add; callers in other files:\n") ||
		!strings.Contains(got, "  main (main.go)") || !strings.Contains(got, "  TestAdd (calc/calc_test.go)") {
		t.Fatalf("%q", got)
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatalf("the aside ends without a newline: %q", got)
	}
}

// Removed symbols come first: their callers break for sure.
func TestBlastAsideNamesARemovedFunctionFirst(t *testing.T) {
	root := graphRepo(t)
	edited := calcHead + strings.Replace(subSource, "a - b", "b - a", 1)
	got := blastAside(root, "calc/calc.go", fixed(edited))
	if !strings.HasPrefix(got, "[graph] calc/calc.go: removed Add, changed Sub; callers in other files:\n") ||
		!strings.Contains(got, "  main (main.go)") {
		t.Fatalf("%q", got)
	}
}

// Whatever changes no symbol that has a caller elsewhere is silence: a
// changed function nobody calls, a new function, a comment between symbols.
func TestBlastAsideIsSilentWithoutCallersToName(t *testing.T) {
	root := graphRepo(t)
	unchanged := calcHead + addSource + "\n" + subSource
	for name, source := range map[string]string{
		"uncalled function changed": calcHead + addSource + "\n" + strings.Replace(subSource, "a - b", "b - a", 1),
		"function added":            unchanged + "\nfunc Mul(a, b int) int { return a * b }\n",
		"comment between symbols":   unchanged + "\n// A trailing note.\n",
		"nothing changed":           unchanged,
	} {
		if got := blastAside(root, "calc/calc.go", fixed(source)); got != "" {
			t.Errorf("%s: %q", name, got)
		}
	}
}

// The Go graph has no edges to types; a changed type says so, callers or not.
func TestBlastAsideNotesAChangedType(t *testing.T) {
	root := graphRepo(t)
	test, err := os.ReadFile(filepath.Join(root, "calc", "calc_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	withType := string(test) + "\ntype T struct{ A int }\n"
	writeRepoFile(t, root, "calc/calc_test.go", withType)
	buildGraph(t, root)
	got := blastAside(root, "calc/calc_test.go", fixed(strings.Replace(withType, "A int", "A string", 1)))
	if got != typeNote {
		t.Fatalf("%q", got)
	}
	if typeNote != "[graph] Modified struct/interface/type: type coupling not wired in graph v1 (check references via grep)" {
		t.Fatalf("the note is worded by the spec: %q", typeNote)
	}
}

// Every kind a type declaration gets marks the edit as typed.
func TestBlastAsideNotesEveryKindOfType(t *testing.T) {
	for name, decl := range map[string][2]string{
		"interface":  {"type I interface{ M() }\n", "type I interface{ N() }\n"},
		"named type": {"type N int\n", "type N string\n"},
	} {
		root := graphRepo(t)
		writeRepoFile(t, root, "calc/types.go", calcHead+decl[0])
		buildGraph(t, root)
		if got := blastAside(root, "calc/types.go", fixed(calcHead+decl[1])); got != typeNote {
			t.Errorf("%s: %q", name, got)
		}
	}
}

func TestBlastAsideIsSilentWhereItCannotKnow(t *testing.T) {
	root := graphRepo(t)
	edited := calcHead + strings.Replace(addSource, "a + b", "b + a", 1)
	cases := map[string]struct {
		root, rel string
		read      func(string) ([]byte, error)
	}{
		"no graph":            {t.TempDir(), "calc/calc.go", fixed(edited)},
		"file not in graph":   {root, "calc/new.go", fixed(edited)},
		"read fails":          {root, "calc/calc.go", func(string) ([]byte, error) { return nil, errors.New("gone") }},
		"source cannot parse": {root, "calc/calc.go", fixed("package calc\n\nfunc (")},
	}
	for name, c := range cases {
		if got := blastAside(c.root, c.rel, c.read); got != "" {
			t.Errorf("%s: %q", name, got)
		}
	}
}

// A graph another extractor wrote, or one of another schema, gives body
// hashes this extractor cannot compare against.
func TestBlastAsideIsSilentOnAForeignGraph(t *testing.T) {
	edited := calcHead + strings.Replace(addSource, "a + b", "b + a", 1)
	for name, alter := range map[string]func(*model.Graph){
		"older extractor": func(g *model.Graph) { g.Meta.Extractor = "go/0" },
		"older schema":    func(g *model.Graph) { g.Meta.Version = 1 },
	} {
		root := graphRepo(t)
		g, err := store.Read(root)
		if err != nil {
			t.Fatal(err)
		}
		alter(g)
		if err := store.Write(root, g); err != nil {
			t.Fatal(err)
		}
		if got := blastAside(root, "calc/calc.go", fixed(edited)); got != "" {
			t.Errorf("%s: %q", name, got)
		}
	}
}

// The aside names at most ten callers and counts the rest.
func TestBlastAsideCountsCallersPastTen(t *testing.T) {
	root := t.TempDir()
	writeRepoFile(t, root, "go.mod", "module example.com/many\n")
	writeRepoFile(t, root, "calc/calc.go", calcHead+addSource)
	for i := 1; i <= 12; i++ {
		writeRepoFile(t, root, fmt.Sprintf("u%02d/u.go", i), fmt.Sprintf(
			"package u%02d\n\nimport \"example.com/many/calc\"\n\nfunc Use%02d() int { return calc.Add(1, 2) }\n", i, i))
	}
	buildGraph(t, root)
	got := blastAside(root, "calc/calc.go", fromDisk(root))
	if got != "" {
		t.Fatalf("an unchanged file changes nothing: %q", got)
	}
	got = blastAside(root, "calc/calc.go", fixed(calcHead+strings.Replace(addSource, "a + b", "b + a", 1)))
	lines := strings.Split(got, "\n")
	if len(lines) != 12 || lines[11] != "  … and 2 more" || !strings.HasPrefix(lines[1], "  Use") {
		t.Fatalf("%q", got)
	}
}

func TestRelInRoot(t *testing.T) {
	root := t.TempDir()
	for raw, want := range map[string]string{
		"a.go":                              "a.go",
		filepath.Join("calc", "calc.go"):    "calc/calc.go",
		filepath.Join(root, "calc", "b.go"): "calc/b.go",
	} {
		if got, ok := relInRoot(root, raw); !ok || got != want {
			t.Errorf("%s: %q %v", raw, got, ok)
		}
	}
	for _, raw := range []string{filepath.Join("..", "a.go"), filepath.Join(t.TempDir(), "a.go")} {
		if got, ok := relInRoot(root, raw); ok {
			t.Errorf("%s lies outside the root: %q", raw, got)
		}
	}
}
