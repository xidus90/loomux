package compat

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func goEntries(t *testing.T) (all, syntax []Entry) {
	t.Helper()
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if e.Runtime != "go" {
			continue
		}
		all = append(all, e)
		if e.syntax() {
			syntax = append(syntax, e)
		}
	}
	if len(syntax) == 0 {
		t.Fatal("the list has no Go syntax entry")
	}
	return all, syntax
}

// A syntax fixture is named <prefix>-syntax-<version>, its prefix per runtime.
func TestEveryVersionHasASyntaxEntry(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// Java 27 brought no final syntax of its own: java-syntax-27 carries the
	// forms of 25 to 27 (TestTheJava25GapsLeaveTheRestOfTheFile holds the two
	// alike), and Node 26 none either: js-syntax-26 carries the forms of 24
	// and TypeScript beside them. These two fixtures do not hold what their
	// version introduced.
	prefixes := map[string]string{"go": "go", "dotnet": "csharp", "cpp": "cpp", "java": "java", "php": "php", "node": "js"}
	for _, e := range list {
		prefix, ok := prefixes[e.Runtime]
		if !ok {
			continue
		}
		fixture := prefix + "-syntax-" + e.Version
		if !slices.ContainsFunc(list, func(s Entry) bool { return s.syntax() && s.Version == e.Version && s.Fixture == fixture }) {
			t.Errorf("%s %s has no syntax entry with fixture %s", e.Runtime, e.Version, fixture)
		}
	}
	for runtime := range prefixes {
		if !slices.ContainsFunc(list, func(e Entry) bool { return e.Runtime == runtime }) {
			t.Errorf("the list has no %s entry", runtime)
		}
	}
}

// go/parser reads every version alike; only the type checker gates a form
// by the version it came with, so that is what proves the fixture new.
func TestSyntaxFixtureNeedsItsVersion(t *testing.T) {
	_, syntax := goEntries(t)
	for _, e := range syntax {
		t.Run(e.Fixture, func(t *testing.T) {
			fset := token.NewFileSet()
			var files []*ast.File
			paths, _ := fs.Glob(Fixtures(), e.Fixture+"/*.go")
			for _, p := range paths {
				src, _ := fs.ReadFile(Fixtures(), p)
				f, err := parser.ParseFile(fset, p, src, 0)
				if err != nil {
					t.Fatal(err)
				}
				files = append(files, f)
			}
			if len(files) == 0 {
				t.Fatal("no Go file")
			}
			errs := func(version string) []error {
				var out []error
				cfg := types.Config{GoVersion: version, Error: func(err error) { out = append(out, err) }}
				cfg.Check("syntax", fset, files, nil)
				return out
			}
			v, _ := parseVer(e.Version)
			if got := errs(fmt.Sprintf("go%d.%d", v[0], v[1])); len(got) != 0 {
				t.Errorf("go%s: %v", e.Version, got)
			}
			if prev := fmt.Sprintf("go%d.%d", v[0], v[1]-1); len(errs(prev)) == 0 {
				t.Errorf("%s accepts the fixture of %s", prev, e.Version)
			}
		})
	}
}

// A listed Go version above the toolchain loomux is built with could not be
// read by its go/parser.
func TestListGoVersionsNotAboveGoModToolchain(t *testing.T) {
	all, _ := goEntries(t)
	mod, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^toolchain go(\d+\.\d+)`).FindSubmatch(mod)
	if m == nil {
		t.Fatal("go.mod has no toolchain line")
	}
	top, _ := parseVer(string(m[1]))
	for _, e := range all {
		if v, _ := parseVer(e.Version); cmpVer(v, top) > 0 {
			t.Errorf("go %s is above the toolchain go%s of go.mod", e.Version, m[1])
		}
	}
}

func TestTheListLoads(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("the list is empty")
	}
	for _, e := range list {
		if e.Stack == "" || e.Runtime == "" || e.Version == "" || e.Fixture == "" || len(e.Kinds) == 0 || e.Line == 0 {
			t.Errorf("incomplete entry %+v", e)
		}
	}
	if got, want := list[0].Key(), "gdscript/godot-3.6/godot3-gut"; got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
	if list[0].Addons[0] != (Addon{Name: "GUT", Version: "7.4.3"}) || list[1].Tools["gdtoolkit"] != "3.*" {
		t.Errorf("addons or tools lost: %+v %+v", list[0], list[1])
	}
	if got, want := list[1].Key(), "gdscript/godot-3.6/godot3-plain"; got != want {
		t.Errorf("Key() = %q, want %q", got, want)
	}
}

const good = `[[entry]]
stack = "python"
runtime = "python"
version = "3.12"
fixture = "python-mypy"
kinds = ["lint", "types"]
tools = { ruff = "0.17.*" }
`

func TestTheListRefuses(t *testing.T) {
	cases := []struct{ name, doc, want string }{
		{"syntax", "[[entry", "compat.toml"},
		{"unknown key", good + "colour = 1\n", `compat.toml:8: unknown key "colour"`},
		{"unknown stack", strings.Replace(good, `stack = "python"`, `stack = "cobol"`, 1), `compat.toml:1: stack "cobol"; stacks are cpp, csharp, css, gdscript, go, html, java, php, python, typescript`},
		{"wrong runtime", strings.Replace(good, `runtime = "python"`, `runtime = "godot"`, 1), `compat.toml:1: runtime "godot"`},
		{"bad version", strings.Replace(good, `"3.12"`, `"3.12.1"`, 1), `compat.toml:1: version "3.12.1" is not MAJOR or MAJOR.MINOR`},
		{"unknown kind", strings.Replace(good, `"types"`, `"magic"`, 1), `compat.toml:1: kind "magic"`},
		{"duplicate", good + "\n" + good, `compat.toml:9: duplicate entry python/python-3.12/python-mypy`},
		{"unknown addon", good + `addons = [{ name = "Nope", version = "1" }]` + "\n", `compat.toml:1: addon "Nope"`},
		{"unknown tool", strings.Replace(good, "ruff", "hammer", 1), `compat.toml:1: tool "hammer"`},
		{"no fixture folder", strings.Replace(good, `"python-mypy"`, `"nowhere"`, 1), `compat.toml:1: fixture "nowhere"`},
		{"empty held", good + "held = \" \"\n", `compat.toml:1: held is empty; name what holds the line back`},
		{"no kinds", strings.Replace(good, "kinds = [\"lint\", \"types\"]", "kinds = []", 1), `compat.toml:1: entry has no kinds`},
		{"no fixture", strings.Replace(good, "fixture = \"python-mypy\"\n", "", 1), `compat.toml:1: entry has no fixture`},
		// A pre-release is probed, never listed.
		{"pre-release", good + "pre = \"3.12.0rc1\"\n", `unknown key "pre"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parse([]byte(c.doc), "compat.toml")
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want it to contain %q", err, c.want)
			}
		})
	}
}

// Java and Node name a version by its major alone, .NET by major.minor.
func TestAVersionIsAMajorOrAMajorMinor(t *testing.T) {
	for _, v := range []string{"21", "10.0"} {
		if _, err := parse([]byte(strings.Replace(good, `"3.12"`, `"`+v+`"`, 1)), "compat.toml"); err != nil {
			t.Errorf("%s: %v", v, err)
		}
	}
	if _, err := parse([]byte(strings.Replace(good, `"3.12"`, `"1.2.3"`, 1)), "compat.toml"); err == nil {
		t.Error("1.2.3 passed")
	}
}

func TestAnInlineKeyHasNoLine(t *testing.T) {
	doc := good + `addons = [{ name = "GUT", version = "1", colour = "x" }]` + "\n"
	_, err := parse([]byte(doc), "compat.toml")
	if err == nil || !strings.HasPrefix(err.Error(), `compat.toml: unknown key "colour"`) {
		t.Errorf("err = %v", err)
	}
}

func TestTheListLoadsFromAnyDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(Fixtures(), "go-module/go.mod.txt"); err != nil {
		t.Error(err)
	}
	if _, err := fs.Stat(Fixtures(), "godot3-gut/project.godot"); err != nil {
		t.Error(err)
	}
}

func TestAProbeMayCarryAPreReleaseOfItsVersion(t *testing.T) {
	dir := t.TempDir()
	write := func(json string) string {
		p := filepath.Join(dir, "probe.json")
		if err := os.WriteFile(p, []byte(json), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	e, err := LoadEntry(write(`{"stack":"go","runtime":"go","version":"1.28","pre":"1.28rc1","fixture":"go-module","kinds":["lint"]}`))
	if err != nil || e.Pre != "1.28rc1" {
		t.Fatalf("%+v %v", e, err)
	}
	for _, pre := range []string{"1.29rc1", "1.28", "soon"} {
		if _, err := LoadEntry(write(`{"stack":"go","runtime":"go","version":"1.28","pre":"` + pre + `","fixture":"go-module","kinds":["lint"]}`)); err == nil || !strings.Contains(err.Error(), "pre-release") {
			t.Errorf("pre %q: %v", pre, err)
		}
	}
}
