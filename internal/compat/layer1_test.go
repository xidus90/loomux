package compat

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

// planEnv is a plan without tools: every probe answers yes, but one into
// node_modules/ only where a run installs it, from the fixture's lock.
func planEnv(root string) verify.PlanEnv {
	_, err := os.Stat(filepath.Join(root, "package-lock.json"))
	installed := err == nil
	return verify.PlanEnv{
		Root: root, Loomux: "{loomux}", RunID: "R",
		HasTests: func(_ string, patterns []string) bool {
			return installed || !slices.ContainsFunc(patterns, func(p string) bool { return strings.HasPrefix(p, "node_modules/") })
		},
		ImportReady: func(string) bool { return true },
		GraphReady:  func(string) (bool, string) { return true, "" },
		Godot:       func(string) (string, verify.State, string) { return "godot", "", "" },
	}
}

func TestEveryEntryPassesLayerOne(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		t.Run(e.Key(), func(t *testing.T) {
			root := t.TempDir()
			if err := CopyFixture(e.Fixture, root); err != nil {
				t.Fatal(err)
			}
			want, err := LoadExpect(e.Fixture)
			if err != nil {
				t.Fatal(err)
			}
			// A run configures a C++ fixture before its lanes plan; the
			// cache file is all the plan asks of it.
			if e.Runtime == "cpp" {
				if err := os.MkdirAll(filepath.Join(root, "build"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "build", "CMakeCache.txt"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			facts := detect.Detect(os.DirFS(root))
			for _, s := range append(slices.Clone(want.Stacks), want.Variants...) {
				if !slices.Contains(facts.Stacks, s) {
					t.Errorf("detected %v, want %q among them", facts.Stacks, s)
				}
			}

			cfg, err := verify.ReadConfig(root)
			if err != nil {
				t.Fatal(err)
			}
			presets, err := verify.LoadPresets()
			if err != nil {
				t.Fatal(err)
			}
			eff, err := verify.Resolve(cfg, presets, facts)
			if err != nil {
				t.Fatal(err)
			}
			jobs, err := verify.Plan(eff, verify.Request{Kinds: e.Kinds}, planEnv(root))
			if err != nil {
				t.Fatal(err)
			}
			for _, kind := range e.Kinds {
				prefix, ok := want.Lanes[kind+"/"+e.Stack]
				if !ok {
					prefix, ok = want.Lanes[kind]
				}
				if !ok {
					t.Errorf("expect.toml has no lane for %s", kind)
					continue
				}
				// Graph and rules run once at the root, for whichever stack.
				atRoot := kind == "graph" || kind == "rules"
				i := slices.IndexFunc(jobs, func(j verify.Job) bool {
					return j.Kind == kind && j.Pre == "" && (atRoot || j.Stack == e.Stack)
				})
				if i < 0 {
					t.Errorf("no %s lane is planned: %+v", kind, jobs)
					continue
				}
				// {gradle} is the wrapper by its absolute path in the copy.
				got := strings.TrimPrefix(strings.Join(jobs[i].Argvs[0], " "), root+string(filepath.Separator))
				if !strings.HasPrefix(got, prefix) {
					t.Errorf("%s plans %q, want it to begin with %q", kind, got, prefix)
				}
			}

			g, _, err := query.Build(root, func(string) {})
			if err != nil {
				t.Fatal(err)
			}
			for _, edge := range want.Edges {
				if !slices.ContainsFunc(g.Edges, func(x model.Edge) bool {
					return string(x.Source) == edge[0] && string(x.Relation) == edge[1] && string(x.Target) == edge[2]
				}) {
					t.Errorf("the graph has no edge %v", edge)
				}
			}

			for _, kind := range e.Kinds {
				if kind == "graph" {
					continue
				}
				if !slices.ContainsFunc(want.Breaks, func(b Break) bool { return b.Kind == kind && b.ranBy(e) }) {
					t.Errorf("expect.toml has no break for %s", kind)
				}
			}
			for _, b := range want.Breaks {
				if b.Stack != "" && !slices.Contains(want.Stacks, b.Stack) {
					t.Errorf("break of %s: stack %q is none of %v", b.Kind, b.Stack, want.Stacks)
				}
				// The break changes one file, so its stack's kind must be one
				// lane, not one per area.
				if n := len(slices.DeleteFunc(slices.Clone(jobs), func(j verify.Job) bool { return j.Kind != b.Kind || j.Stack != b.Stack })); b.Stack != "" && n != 1 {
					t.Errorf("break of %s/%s: %d lanes are planned, want 1", b.Kind, b.Stack, n)
				}
				for _, k := range append([]string{b.Kind}, b.Also...) {
					if !slices.Contains(e.Kinds, k) {
						t.Errorf("break of kind %q: the entry has kinds %v", k, e.Kinds)
					}
				}
				if _, err := os.Stat(filepath.Dir(filepath.Join(root, b.File))); err != nil {
					t.Errorf("break file %s: %v", b.File, err)
				}
				if b.Content == "" {
					t.Errorf("break of %s has no content", b.Kind)
				}
			}
		})
	}
}

// The real addons are third-party code in their own style; gdlint and,
// from gdtoolkit 4 on, gdformat read the directories to skip from their
// config. gdtoolkit 3's gdformat reads none, so a Godot 3 fixture with addons
// cannot lint: godot3-gut proves test and graph, godot3-plain lint. Only an
// entry that lints a fixture with addons/ is held to the rule; a lint break
// under addons/ would be skipped with them.
func TestEveryGdscriptFixtureLintsWithoutItsAddons(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if e.Stack != "gdscript" {
			continue
		}
		if _, err := fs.Stat(Fixtures(), e.Fixture+"/addons"); err != nil || !slices.Contains(e.Kinds, "lint") {
			continue
		}
		files := []string{".gdlintrc"}
		if e.Tools["gdtoolkit"] != "3.*" {
			files = append(files, ".gdformatrc")
		}
		for _, f := range files {
			data, err := fs.ReadFile(Fixtures(), e.Fixture+"/"+f)
			if err != nil || !slices.Contains(strings.Split(string(data), "\n"), "  - addons") {
				t.Errorf("%s: %s does not exclude addons (%v)", e.Key(), f, err)
			}
		}
		want, err := LoadExpect(e.Fixture)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range want.Breaks {
			if b.Kind == "lint" && strings.HasPrefix(b.File, "addons/") {
				t.Errorf("%s: the lint break %s sits under addons/", e.Key(), b.File)
			}
		}
	}
}

// gdformat 3 cannot skip addons/, so a Godot 3 entry whose fixture carries
// addons does not claim lint.
func TestAGodot3FixtureWithAddonsClaimsNoLint(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range list {
		if e.Runtime != "godot" || !strings.HasPrefix(e.Version, "3.") {
			continue
		}
		if _, err := fs.Stat(Fixtures(), e.Fixture+"/addons"); err == nil && slices.Contains(e.Kinds, "lint") {
			t.Errorf("%s: gdformat 3 would check the addons", e.Key())
		}
	}
}

func TestEveryFixtureIsUsed(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range entries {
		// A plain file is a recorded answer for a test, not a fixture.
		if !d.IsDir() {
			continue
		}
		if !slices.ContainsFunc(list, func(e Entry) bool { return e.Fixture == d.Name() }) {
			t.Errorf("fixture %s is used by no entry", d.Name())
		}
	}
}

// The grammar misses four forms of C# 15 (docs/en/support.md): the union,
// the closed record and the type holding an extension indexer leave no node,
// the labelled break only an ERROR inside First. A grammar that learns one
// fails here, and the support page is to follow.
func TestTheCSharp15GapsLeaveTheRestOfTheFile(t *testing.T) {
	root := t.TempDir()
	if err := CopyFixture("csharp-syntax-11.0", root); err != nil {
		t.Fatal(err)
	}
	g, _, err := query.Build(root, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range g.Nodes {
		if strings.HasSuffix(n.Path, ".cs") {
			got = append(got, string(n.ID))
		}
	}
	slices.Sort(got)
	want := []string{"Syntax.cs", "Syntax.cs#Cat", "Syntax.cs#Dog", "Syntax.cs#Loops",
		"Syntax.cs#Loops.First", "Syntax.cs#Loops.Make", "Syntax.cs#Loops.Sized"}
	if !slices.Equal(got, want) {
		t.Errorf("nodes %v, want %v", got, want)
	}
}

// The grammar loses the rest of the file at C++23's if consteval
// (docs/en/support.md): pick, which holds it, leaves no node, the re-parse
// brings back after, and last, before the closing brace of a namespace the
// re-parse no longer sees, is lost. A grammar that learns the form fails
// here, and the support page is to follow.
func TestTheCpp23GapLeavesTheRestOfTheFile(t *testing.T) {
	root := t.TempDir()
	if err := CopyFixture("cpp-syntax-23", root); err != nil {
		t.Fatal(err)
	}
	g, _, err := query.Build(root, func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, n := range g.Nodes {
		got = append(got, string(n.ID))
	}
	slices.Sort(got)
	want := []string{"syntax.cpp", "syntax.cpp#Counter", "syntax.cpp#Counter.doubled", "syntax.cpp#Counter.get",
		"syntax.cpp#after", "syntax.cpp#twice"}
	if !slices.Equal(got, want) {
		t.Errorf("nodes %v, want %v", got, want)
	}
}

// The grammar misses two forms of Java 25 (docs/en/support.md): import
// module is an ERROR in the import, a statement before super(...) one in the
// constructor body; the rest of each file arrives. java-syntax-27 carries the
// same forms, as 27 brought no final syntax of its own. A grammar that
// learns a form fails here, and the support page is to follow.
func TestTheJava25GapsLeaveTheRestOfTheFile(t *testing.T) {
	want := []string{"Main.java", "Main.java#greet", "Main.java#main",
		"Shape.java", "Shape.java#Shape", "Shape.java#Shape.Shape",
		"Shape.java#Square", "Shape.java#Square.Square", "Shape.java#Square.area", "Shape.java#Square.twice"}
	for _, fixture := range []string{"java-syntax-25", "java-syntax-27"} {
		root := t.TempDir()
		if err := CopyFixture(fixture, root); err != nil {
			t.Fatal(err)
		}
		g, _, err := query.Build(root, func(string) {})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, n := range g.Nodes {
			if strings.HasSuffix(n.Path, ".java") {
				got = append(got, string(n.ID))
			}
		}
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("%s: nodes %v, want %v", fixture, got, want)
		}
	}
}

// The grammar reads PHP 8.2 to 8.4 in full and misses two forms of 8.5
// (docs/en/support.md): clone($o, [...]) and the (void) cast leave an error
// in the body, every declaration arrives. A grammar that learns both fails
// here, and the support page is to follow.
func TestThePHP85GapsLeaveTheRestOfTheFile(t *testing.T) {
	for _, c := range []struct {
		fixture string
		errors  bool
	}{{"php-syntax-8.2", false}, {"php-syntax-8.3", false}, {"php-syntax-8.4", false}, {"php-syntax-8.5", true}} {
		root := t.TempDir()
		if err := CopyFixture(c.fixture, root); err != nil {
			t.Fatal(err)
		}
		g, stats, err := query.Build(root, func(string) {})
		if err != nil {
			t.Fatal(err)
		}
		if got := stats.PerLanguage["php"].ParseErrors > 0; got != c.errors {
			t.Errorf("%s: parse errors %v, want %v", c.fixture, stats.PerLanguage["php"], c.errors)
		}
		if c.fixture != "php-syntax-8.5" {
			continue
		}
		var got []string
		for _, n := range g.Nodes {
			got = append(got, string(n.ID))
		}
		slices.Sort(got)
		want := []string{"Syntax.php", "Syntax.php#Shape", "Syntax.php#Shape.__construct", "Syntax.php#Shape.grow",
			"Syntax.php#Shape.twice", "Syntax.php#double", "Syntax.php#main"}
		if !slices.Equal(got, want) {
			t.Errorf("nodes %v, want %v", got, want)
		}
	}
}

// The grammars read the syntax of Node 22 to 26 in full, the TypeScript of
// js-syntax-26 too. A grammar that loses one of the forms fails here, and the
// support page is to follow.
func TestTheJSSyntaxFixturesParseInFull(t *testing.T) {
	for _, fixture := range []string{"js-syntax-22", "js-syntax-24", "js-syntax-26"} {
		root := t.TempDir()
		if err := CopyFixture(fixture, root); err != nil {
			t.Fatal(err)
		}
		_, stats, err := query.Build(root, func(string) {})
		if err != nil {
			t.Fatal(err)
		}
		for lang, s := range stats.PerLanguage {
			if s.ParseErrors > 0 {
				t.Errorf("%s: %s %+v", fixture, lang, s)
			}
		}
		if s := stats.PerLanguage["javascript"]; s.Files != 1 {
			t.Errorf("%s: javascript %+v", fixture, s)
		}
	}
}

func TestCopyFixtureRenamesTheModuleFile(t *testing.T) {
	root := t.TempDir()
	if err := CopyFixture("go-module", root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.mod", "calc/calc.go"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Error(err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod.txt")); err == nil {
		t.Error("go.mod.txt was copied as it is")
	}
	if err := CopyFixture("nowhere", t.TempDir()); err == nil {
		t.Error("a missing fixture copied")
	}
	if err := CopyFixture("go-module", filepath.Join(root, "go.mod")+"/x"); err == nil {
		t.Error("a destination below a file was written")
	}
	if _, err := LoadExpect("nowhere"); err == nil {
		t.Error("a missing expect.toml loaded")
	}
}

func TestAMalformedExpectIsRefused(t *testing.T) {
	if _, err := decodeExpect("x", []byte("stacks = ")); err == nil || !strings.Contains(err.Error(), "x/expect.toml") {
		t.Errorf("err = %v", err)
	}
}
