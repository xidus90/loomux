package compat

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
)

const (
	pyKey     = "python/python-3.11/python-mypy"
	gdunitKey = "gdscript/godot-4.7/godot4-gdunit4"
)

// runOf names the run a work copy belongs to: "" for the entry itself, else
// the kind its break breaks.
func runOf(root string) string {
	base := filepath.Base(root)
	if !strings.HasPrefix(base, "break-") {
		return ""
	}
	return strings.SplitN(base, "-", 3)[2]
}

// honest answers what a working toolchain would: every lane ok, and in a
// break's run its kind failed and coverage blocked behind a failed test.
func honest(run, kind string) (state, output string) {
	switch {
	case run == kind:
		return "failed", "boom"
	case run == "test" && kind == "coverage":
		return "blocked", ""
	}
	return "ok", ""
}

// fakeCheck reports one lane per kind asked for, in the state answer gives.
func fakeCheck(answer func(run, kind string) (string, string)) func(string, []string) ([]LaneResult, error) {
	return func(root string, kinds []string) ([]LaneResult, error) {
		var out []LaneResult
		for _, k := range kinds {
			state, output := answer(runOf(root), k)
			out = append(out, LaneResult{Lane: k + "/stack", State: state, Output: output})
		}
		return out, nil
	}
}

func runEntryKey(t *testing.T, key string, check func(string, []string) ([]LaneResult, error)) (Report, string) {
	t.Helper()
	work := t.TempDir()
	rep, err := Run(RunOptions{Key: key, Work: work, Check: check})
	if err != nil {
		t.Fatal(err)
	}
	return rep, work
}

func TestAnEntryWhoseLanesAreAllOkIsGreen(t *testing.T) {
	var asked [][]string
	check := fakeCheck(honest)
	rep, work := runEntryKey(t, pyKey, func(root string, kinds []string) ([]LaneResult, error) {
		asked = append(asked, kinds)
		return check(root, kinds)
	})
	if !rep.Green || len(rep.Lines) != 0 || rep.Entry != pyKey {
		t.Fatalf("report %+v", rep)
	}
	// The entry and each of its four breaks; graph is never a lane.
	if len(asked) != 5 || !slices.Equal(asked[0], []string{"lint", "types", "test", "coverage"}) {
		t.Fatalf("asked %v", asked)
	}
	cfg, err := os.ReadFile(filepath.Join(work, "entry", ".loomux", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[verify.python.tools]", `coverage = "7.16.*"`, `mypy = "2.4.*"`, `pytest = "9.1.*"`, `ruff = "0.17.*"`} {
		if !strings.Contains(string(cfg), want) {
			t.Errorf("config.toml lacks %s:\n%s", want, cfg)
		}
	}
	if v, _ := os.ReadFile(filepath.Join(work, "entry", ".python-version")); string(v) != "3.11\n" {
		t.Errorf(".python-version = %q", v)
	}
	if _, err := os.Stat(filepath.Join(work, "break-1-lint", "pkg", "ambiguous.py")); err != nil {
		t.Errorf("the lint break was not applied: %v", err)
	}
}

func TestASkippedLaneIsRed(t *testing.T) {
	rep, _ := runEntryKey(t, pyKey, fakeCheck(func(run, kind string) (string, string) {
		if run == "" && kind == "types" {
			return "missing-tool", "uv not found\nlooked on PATH"
		}
		return honest(run, kind)
	}))
	if rep.Green || len(rep.Lines) != 1 {
		t.Fatalf("report %+v", rep)
	}
	if l := rep.Lines[0]; !strings.HasPrefix(l, "types/stack: missing-tool, want ok\n") || !strings.Contains(l, "looked on PATH") {
		t.Fatalf("line %q", l)
	}
}

func TestAKindWithoutALaneIsRed(t *testing.T) {
	rep, _ := runEntryKey(t, pyKey, func(root string, kinds []string) ([]LaneResult, error) {
		return fakeCheck(honest)(root, slices.DeleteFunc(slices.Clone(kinds), func(k string) bool { return k == "types" && runOf(root) == "" }))
	})
	if rep.Green || !slices.Equal(rep.Lines, []string{"types: no lane ran"}) {
		t.Fatalf("report %+v", rep)
	}
}

func TestABreakThatLeavesItsKindGreenIsRed(t *testing.T) {
	rep, _ := runEntryKey(t, pyKey, fakeCheck(func(run, kind string) (string, string) {
		if run == "lint" {
			return "ok", ""
		}
		return honest(run, kind)
	}))
	if rep.Green || !slices.Equal(rep.Lines, []string{"break lint: lint/stack: ok, want failed"}) {
		t.Fatalf("report %+v", rep)
	}
}

func TestABreakThatAlsoBreaksAnotherKindIsRed(t *testing.T) {
	rep, _ := runEntryKey(t, pyKey, fakeCheck(func(run, kind string) (string, string) {
		if run == "lint" && kind == "test" {
			return "failed", "AssertionError"
		}
		return honest(run, kind)
	}))
	if rep.Green || !slices.Equal(rep.Lines, []string{"break lint: test/stack: failed, want ok\nAssertionError"}) {
		t.Fatalf("report %+v", rep)
	}
}

func TestABreakWhoseKindDoesNotRunIsRed(t *testing.T) {
	rep, _ := runEntryKey(t, pyKey, fakeCheck(func(run, kind string) (string, string) {
		if run == "types" && kind == "types" {
			return "missing-tool", "no mypy"
		}
		return honest(run, kind)
	}))
	if rep.Green || !slices.Equal(rep.Lines, []string{"break types: types/stack: missing-tool, want failed\nno mypy"}) {
		t.Fatalf("report %+v", rep)
	}
}

func TestAMissingEdgeIsRed(t *testing.T) {
	check := fakeCheck(honest)
	rep, _ := runEntryKey(t, pyKey, func(root string, kinds []string) ([]LaneResult, error) {
		// The graph is built after the check, from what the check left.
		if runOf(root) == "" {
			os.Remove(filepath.Join(root, "tests", "test_core.py"))
		}
		return check(root, kinds)
	})
	if rep.Green || !slices.Equal(rep.Lines, []string{"graph: no edge tests/test_core.py imports pkg/core.py"}) {
		t.Fatalf("report %+v", rep)
	}
}

func TestOnlyTheTailOfAnOutputIsReported(t *testing.T) {
	long := strings.Repeat("line\n", 30) + "last\n"
	rep, _ := runEntryKey(t, pyKey, fakeCheck(func(run, kind string) (string, string) {
		if run == "" && kind == "lint" {
			return "failed", long
		}
		return honest(run, kind)
	}))
	got := strings.Split(rep.Lines[0], "\n")
	if len(got) != 1+tailLines || got[len(got)-1] != "last" {
		t.Fatalf("line %q", rep.Lines[0])
	}
}

func addonsDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"gdUnit4", "NanoCoverage"} {
		if err := os.MkdirAll(filepath.Join(dir, name, "bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, name, "plugin.cfg"), []byte("real "+name), 0o644)
		os.WriteFile(filepath.Join(dir, name, "bin", "tool.gd"), []byte("x"), 0o644)
	}
	return dir
}

func TestAGodotEntryGetsItsAddonsItsVersionAndARescan(t *testing.T) {
	work := t.TempDir()
	var rescanned []string
	rep, err := Run(RunOptions{Key: gdunitKey, Work: work, Addons: addonsDir(t), Check: fakeCheck(honest),
		Rescan: func(root string) error { rescanned = append(rescanned, filepath.Base(root)); return nil }})
	if err != nil || !rep.Green {
		t.Fatalf("report %+v, err %v", rep, err)
	}
	if !slices.Equal(rescanned, []string{"entry", "break-1-lint", "break-2-test", "break-3-coverage"}) {
		t.Fatalf("rescanned %v", rescanned)
	}
	entry := filepath.Join(work, "entry")
	for file, want := range map[string]string{
		"addons/gdUnit4/plugin.cfg":              "real gdUnit4",
		"addons/gdUnit4/bin/tool.gd":             "x",
		"addons/nano_coverage_godot/plugin.cfg":  "real NanoCoverage",
		"addons/nano_coverage_godot/bin/tool.gd": "x",
	} {
		if got, _ := os.ReadFile(filepath.Join(entry, file)); string(got) != want {
			t.Errorf("%s = %q, want %q", file, got, want)
		}
	}
	project, _ := os.ReadFile(filepath.Join(entry, "project.godot"))
	if !strings.Contains(string(project), `config/features=PackedStringArray("4.7")`) {
		t.Errorf("project.godot:\n%s", project)
	}
	cfg, _ := os.ReadFile(filepath.Join(entry, ".loomux", "config.toml"))
	if !strings.Contains(string(cfg), "[verify.gdscript.tools]\ngdtoolkit = \"4.*\"\n") {
		t.Errorf("config.toml:\n%s", cfg)
	}
}

func TestAFailedRescanIsRed(t *testing.T) {
	rep, err := Run(RunOptions{Key: gdunitKey, Work: t.TempDir(), Addons: addonsDir(t), Check: fakeCheck(honest),
		Rescan: func(root string) error { return errors.New("the engine failed") }})
	if err != nil || rep.Green || len(rep.Lines) != 4 || rep.Lines[0] != "rescan: the engine failed" || rep.Lines[1] != "break lint: rescan: the engine failed" {
		t.Fatalf("report %+v, err %v", rep, err)
	}
}

func TestRunRefusesWhatItCannotPrepare(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	failing := func(string, []string) ([]LaneResult, error) { return nil, errors.New("no plan") }
	for name, o := range map[string]RunOptions{
		"unknown entry":  {Key: "nope", Work: t.TempDir(), Check: fakeCheck(honest)},
		"missing addons": {Key: gdunitKey, Work: t.TempDir(), Addons: t.TempDir(), Check: fakeCheck(honest)},
		"work in a file": {Key: pyKey, Work: file, Check: fakeCheck(honest)},
		"failing check":  {Key: pyKey, Work: t.TempDir(), Check: failing},
		"failing break check": {Key: pyKey, Work: t.TempDir(), Check: func(root string, kinds []string) ([]LaneResult, error) {
			if runOf(root) != "" {
				return failing(root, kinds)
			}
			return fakeCheck(honest)(root, kinds)
		}},
	} {
		if _, err := Run(o); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	for name, e := range map[string]Entry{
		"unknown addon":     {Stack: "gdscript", Fixture: "godot4-plain", Addons: []Addon{{Name: "Nope"}}},
		"no project.godot":  {Stack: "gdscript", Fixture: "go-module", Version: "4.5"},
		"no pyproject.toml": {Stack: "python", Fixture: "go-module", Version: "3.11"},
		"no expect.toml":    {Stack: "go", Fixture: "nowhere"},
	} {
		if _, err := runEntry(e, RunOptions{Work: t.TempDir(), Check: fakeCheck(honest)}); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestTheReportReadsAsMarkdown(t *testing.T) {
	if got := (Report{Entry: "k", Green: true}).Markdown(); got != "### k: green\n" {
		t.Errorf("green: %q", got)
	}
	got := Report{Entry: "k", Lines: []string{"graph: no edge a b c", "lint/go: failed, want ok\nout"}}.Markdown()
	want := "### k: red\n\n- graph: no edge a b c\n- lint/go: failed, want ok\n\n  ```\n  out\n  ```\n"
	if got != want {
		t.Errorf("red:\n%q\nwant\n%q", got, want)
	}
}

func TestAGoEntryNeedsNoPins(t *testing.T) {
	rep, work := runEntryKey(t, "go/go-1.27/go-module", fakeCheck(honest))
	if !rep.Green {
		t.Fatalf("report %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(work, "entry", ".loomux", "config.toml")); err == nil {
		t.Error("a config was written without pins")
	}
}

func TestAGraphThatDoesNotBuildIsRed(t *testing.T) {
	old := buildGraph
	t.Cleanup(func() { buildGraph = old })
	buildGraph = func(string, func(string)) (*model.Graph, query.Stats, error) {
		return nil, query.Stats{}, errors.New("no extractor")
	}
	rep, _ := runEntryKey(t, pyKey, fakeCheck(honest))
	if rep.Green || !slices.Equal(rep.Lines, []string{"graph: no extractor"}) {
		t.Fatalf("report %+v", rep)
	}
}
