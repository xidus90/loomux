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

// planEnv is a plan without tools: every probe answers yes.
func planEnv(root string) verify.PlanEnv {
	return verify.PlanEnv{
		Root: root, Loomux: "{loomux}", RunID: "R",
		HasTests:    func(string, []string) bool { return true },
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
				prefix, ok := want.Lanes[kind]
				if !ok {
					t.Errorf("expect.toml has no lane for %s", kind)
					continue
				}
				i := slices.IndexFunc(jobs, func(j verify.Job) bool { return j.Kind == kind && j.Pre == "" })
				if i < 0 {
					t.Errorf("no %s lane is planned: %+v", kind, jobs)
					continue
				}
				if got := strings.Join(jobs[i].Argvs[0], " "); !strings.HasPrefix(got, prefix) {
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
				if !slices.ContainsFunc(want.Breaks, func(b Break) bool { return b.Kind == kind }) {
					t.Errorf("expect.toml has no break for %s", kind)
				}
			}
			for _, b := range want.Breaks {
				if !slices.Contains(e.Kinds, b.Kind) {
					t.Errorf("break of kind %q: the entry has kinds %v", b.Kind, e.Kinds)
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
		if !slices.ContainsFunc(list, func(e Entry) bool { return e.Fixture == d.Name() }) {
			t.Errorf("fixture %s is used by no entry", d.Name())
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
