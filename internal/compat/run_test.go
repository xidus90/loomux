package compat

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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

// A break may name kinds it turns red as well: a compile error fails Gradle's
// checkstyle tasks, which compile first. Those must fail too.
func TestABreakHoldsTheKindsItAlsoBreaksToFailed(t *testing.T) {
	b := &Break{Kind: "types", Also: []string{"lint"}}
	kinds := []string{"lint", "types", "test"}
	results := []LaneResult{{Lane: "lint/java", State: "failed"}, {Lane: "types/java", State: "failed"}, {Lane: "test/java", State: "blocked"}}
	if got := judge(results, kinds, b); len(got) != 0 {
		t.Fatalf("judge = %q", got)
	}
	results[0].State = "ok"
	if got := judge(results, kinds, b); !slices.Equal(got, []string{"lint/java: ok, want failed"}) {
		t.Fatalf("judge = %q", got)
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

func TestADotnetEntryTargetsItsVersion(t *testing.T) {
	rep, work := runEntryKey(t, "csharp/dotnet-11.0/csharp-xunit", fakeCheck(honest))
	if !rep.Green {
		t.Fatalf("report %+v", rep)
	}
	for _, p := range []string{"Lib/Lib.csproj", "Lib.Tests/Lib.Tests.csproj"} {
		data, err := os.ReadFile(filepath.Join(work, "entry", p))
		if err != nil || !strings.Contains(string(data), "<TargetFramework>net11.0</TargetFramework>") {
			t.Errorf("%s: %v\n%s", p, err, data)
		}
	}
}

// The highest installed SDK wins without a global.json; the pin keeps the
// run in the entry's major, a pre-release probe on that major's previews.
func TestADotnetEntryPinsTheSDKOfItsMajor(t *testing.T) {
	for _, c := range []struct {
		e    Entry
		want string
	}{
		{Entry{Runtime: "dotnet", Version: "10.0", Fixture: "csharp-xunit"},
			`{"sdk":{"version":"10.0.100","rollForward":"latestFeature","allowPrerelease":false}}`},
		{Entry{Runtime: "dotnet", Version: "12.0", Pre: "12.0.0-preview.7", Fixture: "csharp-xunit"},
			`{"sdk":{"version":"12.0.100-preview.1","rollForward":"latestFeature","allowPrerelease":true}}`},
	} {
		dir := filepath.Join(t.TempDir(), "w")
		if err := prepare(c.e, RunOptions{}, dir, nil); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(dir, "global.json")); err != nil || strings.TrimSpace(string(got)) != c.want {
			t.Errorf("%s: global.json %s, %v; want %s", c.e.Key(), got, err, c.want)
		}
	}
	// A fixture that ships its own pin keeps it.
	dir := t.TempDir()
	own := []byte(`{"sdk":{"version":"9.0.100"}}`)
	if err := os.WriteFile(filepath.Join(dir, "global.json"), own, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := pinDotnet(dir, Entry{Runtime: "dotnet", Version: "10.0"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "global.json")); string(got) != string(own) {
		t.Errorf("the fixture's pin became %s", got)
	}
	if err := pinDotnet(filepath.Join(dir, "global.json", "below"), Entry{Runtime: "dotnet", Version: "10.0"}); err == nil {
		t.Error("a pin below a file was written")
	}
}

// fakeCMake stands in for CMake and records each call as its work copy and
// arguments; fail, when not empty, is the output of a call that fails.
func fakeCMake(t *testing.T, fail string) *[]string {
	t.Helper()
	var calls []string
	old := cmake
	t.Cleanup(func() { cmake = old })
	cmake = func(dir string, args ...string) ([]byte, error) {
		calls = append(calls, filepath.Base(dir)+": "+strings.Join(args, " "))
		if fail != "" {
			return []byte(fail), errors.New("exit status 1")
		}
		return nil, nil
	}
	return &calls
}

func TestACppEntryConfiguresItsStandard(t *testing.T) {
	calls := fakeCMake(t, "")
	rep, _ := runEntryKey(t, "cpp/cpp-20/cpp-cmake-gtest", fakeCheck(honest))
	if !rep.Green {
		t.Fatalf("report %+v", rep)
	}
	want := []string{"entry", "break-1-lint", "break-2-types", "break-3-test", "break-4-coverage"}
	for i := range want {
		want[i] += ": -S . -B build -DCMAKE_CXX_STANDARD=20"
	}
	if !slices.Equal(*calls, want) {
		t.Errorf("cmake calls %q, want %q", *calls, want)
	}
	// A fixture without CMakeLists.txt is configured by no one.
	*calls = nil
	if rep, _ := runEntryKey(t, "cpp/cpp-20/cpp-syntax-20", fakeCheck(honest)); !rep.Green || len(*calls) != 0 {
		t.Errorf("report %+v, cmake calls %q", rep, *calls)
	}
}

func TestAFailedConfigureIsRed(t *testing.T) {
	fakeCMake(t, "CMake Error: no CXX compiler")
	rep, _ := runEntryKey(t, "cpp/cpp-23/cpp-cmake-gtest", fakeCheck(honest))
	if rep.Green || len(rep.Lines) != 5 || rep.Lines[0] != "configure: exit status 1\nCMake Error: no CXX compiler" {
		t.Fatalf("report %+v", rep)
	}
}

func TestCMakeRunsTheTool(t *testing.T) {
	t.Setenv("PATH", "")
	if _, err := cmake(t.TempDir(), "--version"); err == nil {
		t.Error("cmake ran without a PATH")
	}
}

// One Maven fixture serves several versions: a run compiles it at the
// entry's, through source and target on Java 8, whose javac takes no
// --release. The Gradle fixture's toolchain follows the same way, and its
// wrapper is a program again.
func TestAJavaEntryCompilesAtItsVersion(t *testing.T) {
	for _, c := range []struct {
		e          Entry
		file, want string
	}{
		{Entry{Runtime: "java", Version: "11", Fixture: "java-maven-junit5"}, "pom.xml",
			"<maven.compiler.source>11</maven.compiler.source>\n    <maven.compiler.target>11</maven.compiler.target>"},
		{Entry{Runtime: "java", Version: "27", Fixture: "java-maven-junit6"}, "pom.xml",
			"<maven.compiler.release>27</maven.compiler.release>"},
		{Entry{Runtime: "java", Version: "25", Fixture: "java-gradle-junit6"}, "build.gradle",
			"JavaLanguageVersion.of(25)"},
	} {
		dir := filepath.Join(t.TempDir(), "w")
		if err := prepare(c.e, RunOptions{}, dir, nil); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(dir, c.file)); err != nil || !strings.Contains(string(got), c.want) {
			t.Errorf("%s %s: %v\n%s", c.e.Fixture, c.file, err, got)
		}
	}
	dir := filepath.Join(t.TempDir(), "w")
	if err := prepare(Entry{Runtime: "java", Version: "21", Fixture: "java-gradle-junit6"}, RunOptions{}, dir, nil); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dir, "gradlew")); err != nil || (runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0) {
		t.Errorf("gradlew: %v %v", info, err)
	}
}

const syntaxKey = "go/go-1.27/go-syntax-1.27"

func TestASyntaxEntryRunsOnlyItsGraph(t *testing.T) {
	// The real check refuses an empty request, so a graph-only entry must never ask it.
	rep, _ := runEntryKey(t, syntaxKey, func(root string, kinds []string) ([]LaneResult, error) {
		if len(kinds) == 0 {
			return nil, errors.New(`"" names no check`)
		}
		return fakeCheck(honest)(root, kinds)
	})
	if !rep.Green {
		t.Fatalf("report %+v", rep)
	}
	old := buildGraph
	t.Cleanup(func() { buildGraph = old })
	buildGraph = func(string, func(string)) (*model.Graph, query.Stats, error) {
		return &model.Graph{}, query.Stats{}, nil
	}
	rep, _ = runEntryKey(t, syntaxKey, fakeCheck(honest))
	if rep.Green || len(rep.Lines) == 0 || !strings.HasPrefix(rep.Lines[0], "graph: no edge syntax.go#") {
		t.Fatalf("report %+v", rep)
	}
}
