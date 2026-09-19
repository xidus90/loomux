package verify

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
)

func env(root string) PlanEnv {
	return PlanEnv{
		Root: root, Loomux: `C:\bin\loomux.exe`, RunID: "R",
		HasTests:    func(string, []string) bool { return true },
		ImportReady: func(string) bool { return true },
	}
}

func effFor(t *testing.T, src string, facts detect.Facts) Effective {
	t.Helper()
	cfg, err := parse(t, src)
	if err != nil {
		t.Fatal(err)
	}
	eff, err := Resolve(cfg, presetsFor(t), facts)
	if err != nil {
		t.Fatal(err)
	}
	return eff
}

var goOnly = detect.Facts{Stacks: []string{"go"}, Areas: map[string][]string{"go": {"."}}}

var pythonOnly = detect.Facts{Stacks: []string{"python"}, Areas: map[string][]string{"python": {"."}}}

func names(jobs []Job) string {
	out := []string{}
	for _, j := range jobs {
		out = append(out, j.Name)
	}
	return strings.Join(out, " ")
}

func TestPlanLinksCoverageToTestAndMeasuresInTest(t *testing.T) {
	root := t.TempDir()
	jobs, err := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"lint", "types", "test", "coverage"}}, env(root))
	if err != nil {
		t.Fatal(err)
	}
	if names(jobs) != "lint/go types/go test/go coverage/go" {
		t.Fatalf("%v", names(jobs))
	}
	if jobs[1].Pre != StateNotApplicable || jobs[1].Note != "no command" {
		t.Fatalf("types/go: %+v", jobs[1])
	}
	profile, _ := CoverPaths(root, "R", "go", ".")
	if !slices.Contains(jobs[2].Argvs[0], "-coverprofile="+profile) {
		t.Fatalf("test measures: %v", jobs[2].Argvs)
	}
	if jobs[3].After != 2 || jobs[3].Measure != nil {
		t.Fatalf("coverage: %+v", jobs[3])
	}
	if jobs[3].Argvs[0][0] != `C:\bin\loomux.exe` {
		t.Fatalf("{loomux}: %v", jobs[3].Argvs[0])
	}
	if !slices.Equal(jobs[3].Reads, []string{profile}) || jobs[2].Reads != nil {
		t.Fatalf("reads: %v / %v", jobs[3].Reads, jobs[2].Reads)
	}
	if !jobs[0].Threaded || jobs[0].Origin != "preset" || jobs[0].Dir != root || jobs[0].After != -1 {
		t.Fatalf("lint/go: %+v", jobs[0])
	}
}

func TestPlanLetsCoverageMeasureAlone(t *testing.T) {
	root := t.TempDir()
	jobs, _ := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"coverage"}}, env(root))
	if jobs[0].After != -1 || jobs[0].Measure == nil {
		t.Fatalf("%+v", jobs[0])
	}
	profile, _ := CoverPaths(root, "R", "go", ".")
	if !slices.Contains(jobs[0].Measure, "-coverprofile="+profile) || !slices.Equal(jobs[0].Reads, []string{profile}) {
		t.Fatalf("%+v", jobs[0])
	}
}

func TestPlanTestWithoutCoverageDoesNotMeasure(t *testing.T) {
	jobs, _ := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"test"}}, env(t.TempDir()))
	if strings.Join(jobs[0].Argvs[0], " ") != "go test ./... -count=1" {
		t.Fatalf("%v", jobs[0].Argvs)
	}
}

func TestPlanFailsCoverageThatCannotMeasure(t *testing.T) {
	src := "[verify.go]\ncoverage = \"{loomux} check gocover --profile {coverprofile}\"\n"
	jobs, _ := Plan(effFor(t, src, goOnly), Request{Kinds: []string{"coverage"}}, env(t.TempDir()))
	if jobs[0].Pre != StateFailed || !strings.Contains(jobs[0].Note, "no measure step") || jobs[0].Reads != nil {
		t.Fatalf("%+v", jobs[0])
	}
	if jobs[0].Origin != "config" {
		t.Fatalf("origin: %q", jobs[0].Origin)
	}
}

func TestPlanLeavesCoverageThatReadsNothingAlone(t *testing.T) {
	src := "[verify.go]\ncoverage = \"gcov-check\"\n"
	jobs, _ := Plan(effFor(t, src, goOnly), Request{Kinds: []string{"coverage"}}, env(t.TempDir()))
	if jobs[0].Pre != "" || jobs[0].After != -1 || jobs[0].Measure != nil || jobs[0].Reads != nil {
		t.Fatalf("%+v", jobs[0])
	}
}

func TestPlanMarksMissingTestsUnavailable(t *testing.T) {
	root := t.TempDir()
	e := env(root)
	var seen []string
	e.HasTests = func(dir string, patterns []string) bool {
		seen = append(seen, dir+" "+strings.Join(patterns, ","))
		return false
	}
	jobs, _ := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"test", "coverage"}}, e)
	if jobs[0].Pre != StateUnavailable || jobs[1].Pre != StateUnavailable || jobs[0].Note != "no tests found" {
		t.Fatalf("%+v", jobs)
	}
	if len(seen) != 2 || seen[0] != root+" *_test.go" {
		t.Fatalf("%v", seen)
	}
}

func TestPlanTrustsAConfiguredTestLane(t *testing.T) {
	e := env(t.TempDir())
	e.HasTests = func(string, []string) bool { return false }
	jobs, _ := Plan(effFor(t, "[verify.go]\ntest = \"go test ./...\"\n", goOnly), Request{Kinds: []string{"test"}}, e)
	if jobs[0].Pre != "" {
		t.Fatalf("%+v", jobs[0])
	}
}

func TestPlanEditScopeRunsOnFileInTheFilesArea(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"typescript"}, Areas: map[string][]string{"typescript": {"admin", "web"}}}
	root := t.TempDir()
	jobs, _ := Plan(effFor(t, "", facts), Request{Kinds: []string{"lint", "types"}, Scope: ScopeEdit, File: "web/src/a.ts"}, env(root))
	if len(jobs) != 2 || jobs[0].Name != "lint/typescript@web" || jobs[0].Dir != filepath.Join(root, "web") {
		t.Fatalf("%+v", jobs)
	}
	if strings.Join(jobs[0].Argvs[0], " ") != "npx eslint --cache src/a.ts" {
		t.Fatalf("%v", jobs[0].Argvs)
	}
	if strings.Join(jobs[1].Argvs[0], " ") != "npx tsc --noEmit" {
		t.Fatalf("types falls back to commands: %v", jobs[1].Argvs)
	}
}

func TestPlanEditScopePicksTheLongestArea(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"typescript"}, Areas: map[string][]string{"typescript": {"web", "web/app"}}}
	eff := effFor(t, "", facts)
	req := Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "web/app/X.TS"}
	jobs, _ := Plan(eff, req, env(t.TempDir()))
	if names(jobs) != "lint/typescript@web/app" || jobs[0].Argvs[0][3] != "X.TS" {
		t.Fatalf("%+v", jobs)
	}
	req.File = "top.ts"
	jobs, _ = Plan(eff, req, env(t.TempDir()))
	if names(jobs) != "lint/typescript@." || jobs[0].Argvs[0][3] != "top.ts" {
		t.Fatalf("%+v", jobs)
	}
}

func TestPlanEditScopeSkipsUndefinedLanes(t *testing.T) {
	jobs, _ := Plan(effFor(t, "", goOnly), Request{Kinds: []string{"lint", "types"}, Scope: ScopeEdit, File: "a.go"}, env(t.TempDir()))
	if names(jobs) != "lint/go" {
		t.Fatalf("%v", names(jobs))
	}
}

func TestPlanEditScopeWithoutAStack(t *testing.T) {
	eff := effFor(t, "", goOnly)
	for _, file := range []string{"a.py", "notes.xyz", "Makefile"} {
		jobs, err := Plan(eff, Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: file}, env(t.TempDir()))
		if err != nil || len(jobs) != 0 {
			t.Fatalf("%s: %+v %v", file, jobs, err)
		}
	}
}

func TestPlanProjectRunsInEditScopeOnlyWithOnFile(t *testing.T) {
	req := Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "sub/a.go"}
	jobs, _ := Plan(effFor(t, "[verify.project]\nlint = \"make lint\"\n", goOnly), req, env(t.TempDir()))
	if names(jobs) != "lint/go" {
		t.Fatalf("%v", names(jobs))
	}
	root := t.TempDir()
	src := "[verify.project.lint]\non_file = [\"fmt {file} {area}\"]\n"
	jobs, _ = Plan(effFor(t, src, goOnly), req, env(root))
	if names(jobs) != "lint/go lint/project" {
		t.Fatalf("%v", names(jobs))
	}
	if strings.Join(jobs[1].Argvs[0], " ") != "fmt sub/a.go "+root {
		t.Fatalf("%v", jobs[1].Argvs)
	}
	jobs, _ = Plan(effFor(t, "[verify.project]\nlint = \"make lint\"\n", goOnly), Request{Kinds: []string{"lint"}}, env(root))
	if names(jobs) != "lint/go lint/project" || strings.Join(jobs[1].Argvs[0], " ") != "make lint" {
		t.Fatalf("%+v", jobs)
	}
}

func TestPlanProjectNeedsAnActiveFileStack(t *testing.T) {
	eff := effFor(t, "[verify.project.lint]\non_file = [\"x {file}\"]\n", goOnly)
	for _, file := range []string{"notes.xyz", "a.py"} {
		jobs, err := Plan(eff, Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: file}, env(t.TempDir()))
		if err != nil || len(jobs) != 0 {
			t.Fatalf("%s: %+v %v", file, jobs, err)
		}
	}
	jobs, _ := Plan(eff, Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "a.go"}, env(t.TempDir()))
	if names(jobs) != "lint/go lint/project" || strings.Join(jobs[1].Argvs[0], " ") != "x a.go" {
		t.Fatalf("%+v", jobs)
	}
}

func TestPlanMarksGodotUnready(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"gdscript"}, Areas: map[string][]string{"gdscript": {"game"}}}
	root := t.TempDir()
	e := env(root)
	var asked string
	e.ImportReady = func(dir string) bool { asked = dir; return false }
	jobs, _ := Plan(effFor(t, "", facts), Request{Kinds: []string{"lint", "test"}}, e)
	if jobs[0].Pre != "" || jobs[1].Pre != StateUnready || jobs[1].Note != "run the Godot editor once to import the project" {
		t.Fatalf("%+v", jobs)
	}
	if asked != filepath.Join(root, "game") {
		t.Fatalf("asked %q", asked)
	}
	jobs, _ = Plan(effFor(t, "[verify.gdscript]\nimport_check = false\n", facts), Request{Kinds: []string{"test"}}, e)
	if jobs[0].Pre != "" {
		t.Fatalf("%+v", jobs)
	}
}

func TestPlanGivesPythonItsOwnCoverageFile(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"go", "python"}, Areas: map[string][]string{"python": {"py"}}}
	root := t.TempDir()
	jobs, _ := Plan(effFor(t, "", facts), Request{Kinds: []string{"coverage"}}, env(root))
	if names(jobs) != "coverage/go coverage/python" {
		t.Fatalf("%v", names(jobs))
	}
	_, data := CoverPaths(root, "R", "python", "py")
	if jobs[0].Env != nil || !slices.Equal(jobs[1].Env, []string{"COVERAGE_FILE=" + data}) {
		t.Fatalf("%v / %v", jobs[0].Env, jobs[1].Env)
	}
	if jobs[1].Measure == nil || jobs[1].Reads != nil {
		t.Fatalf("%+v", jobs[1])
	}
}

func TestPlanNamesEveryAreaInOrder(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"typescript"}, Areas: map[string][]string{"typescript": {"web", "admin"}}}
	jobs, _ := Plan(effFor(t, "", facts), Request{Kinds: []string{"test", "lint"}}, env(t.TempDir()))
	if names(jobs) != "test/typescript@admin test/typescript@web lint/typescript@admin lint/typescript@web" {
		t.Fatalf("%v", names(jobs))
	}
}

func TestPlanLinksPerArea(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"go"}, Areas: map[string][]string{"go": {"b", "a"}}}
	root := t.TempDir()
	jobs, _ := Plan(effFor(t, "", facts), Request{Kinds: []string{"test", "coverage"}}, env(root))
	if names(jobs) != "test/go@a test/go@b coverage/go@a coverage/go@b" || jobs[2].After != 0 || jobs[3].After != 1 {
		t.Fatalf("%+v", jobs)
	}
	profile, data := CoverPaths(root, "R", "go", "b")
	if !slices.Equal(jobs[3].Reads, []string{profile}) || data == profile {
		t.Fatalf("%v", jobs[3].Reads)
	}
}

func TestPlanKeepsUnknownPlaceholders(t *testing.T) {
	root := t.TempDir()
	src := "[verify.go]\nlint = [\"echo {x} {area}\", \"echo {coverdata}\"]\n"
	jobs, _ := Plan(effFor(t, src, goOnly), Request{Kinds: []string{"lint"}}, env(root))
	_, data := CoverPaths(root, "R", "go", ".")
	if strings.Join(jobs[0].Argvs[0], " ") != "echo {x} "+root || jobs[0].Argvs[1][1] != data {
		t.Fatalf("%v", jobs[0].Argvs)
	}
}

func TestPlanDoesNotWriteIntoTheLane(t *testing.T) {
	eff := effFor(t, "", goOnly)
	before := slices.Clone(eff.Stacks["go"]["coverage"].Lane.Commands)
	if _, err := Plan(eff, Request{Kinds: []string{"coverage"}}, env(t.TempDir())); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eff.Stacks["go"]["coverage"].Lane.Commands, before) {
		t.Fatalf("%v", eff.Stacks["go"]["coverage"].Lane.Commands)
	}
}

func TestPlanReturnsASplitError(t *testing.T) {
	bad := func(l Lane) Effective {
		return Effective{
			Stacks: map[string]map[string]Resolved{"go": {"coverage": {Lane: l, Defined: true}}},
			Areas:  map[string][]string{"go": {"."}},
			Active: []string{"go"},
		}
	}
	for _, l := range []Lane{
		{Commands: []string{"a 'b"}},
		{Commands: []string{"{coverprofile}"}, Measure: "a 'b", After: "test"},
	} {
		if _, err := Plan(bad(l), Request{Kinds: []string{"coverage"}}, env(t.TempDir())); err == nil || !strings.Contains(err.Error(), "unclosed quote") {
			t.Fatalf("%+v: %v", l, err)
		}
	}
}

func TestExpandProfile(t *testing.T) {
	cfg, _ := parse(t, "")
	cases := []struct {
		in   string
		want []string
		err  string
	}{
		{"all", []string{"lint", "types", "test", "coverage"}, ""},
		{"edit", []string{"lint", "types"}, ""},
		{" test , lint,test ", []string{"test", "lint"}, ""},
		{"coverage", []string{"coverage"}, ""},
		{"lint,style", nil, `unknown check "style"; kinds: lint, types, test, coverage; profiles: edit, precommit`},
		{"", nil, `"" names no check`},
		{" , ", nil, `" , " names no check`},
	}
	for _, c := range cases {
		got, err := ExpandProfile(cfg, c.in)
		if c.err != "" {
			if err == nil || err.Error() != c.err {
				t.Fatalf("%q: %v", c.in, err)
			}
			continue
		}
		if err != nil || !slices.Equal(got, c.want) {
			t.Fatalf("%q: %v %v", c.in, got, err)
		}
	}
	got, _ := ExpandProfile(cfg, "precommit")
	got[0] = "x"
	if cfg.Profiles["precommit"][0] != "lint" {
		t.Fatal("ExpandProfile hands out the profile itself")
	}
}

func TestImportReadyAsksForTheClassCache(t *testing.T) {
	dir := t.TempDir()
	if ImportReady(dir) {
		t.Fatal("a project never imported is not ready")
	}
	os.MkdirAll(filepath.Join(dir, ".godot"), 0o755)
	os.WriteFile(filepath.Join(dir, ".godot", "global_script_class_cache.cfg"), nil, 0o644)
	if !ImportReady(dir) {
		t.Fatal("the class cache marks an imported project")
	}
}

// A test lane switched off names no command, so nobody has vouched for tests:
// the coverage lane still asks whether there are any.
func TestPlanDoesNotTrustATestLaneSwitchedOff(t *testing.T) {
	e := env(t.TempDir())
	e.HasTests = func(string, []string) bool { return false }
	jobs, err := Plan(effFor(t, "[verify.go]\ntest = false\n", goOnly), Request{Kinds: []string{"coverage"}}, e)
	if err != nil || len(jobs) != 1 || jobs[0].Pre != StateUnavailable {
		t.Fatalf("%v %+v", err, jobs)
	}
}

// A test lane switched off measures nothing, so coverage cannot wait for it:
// it measures by itself, or fails where it has no measure step.
func TestPlanCoverageMeasuresItselfWhenTestIsSwitchedOff(t *testing.T) {
	req := Request{Kinds: []string{"test", "coverage"}}
	jobs, err := Plan(effFor(t, "[verify.go]\ntest = false\n", goOnly), req, env(t.TempDir()))
	if err != nil || len(jobs) != 2 || jobs[0].Pre != StateNotApplicable {
		t.Fatalf("%v %+v", err, jobs)
	}
	if cov := jobs[1]; cov.After != -1 || len(cov.Measure) == 0 || cov.Pre != "" {
		t.Fatalf("%+v", cov)
	}

	// {coverdata}: the load check already refuses a lone {coverprofile}.
	src := "[verify.go]\ntest = false\ncoverage = \"report {coverdata}\"\n"
	jobs, err = Plan(effFor(t, src, goOnly), req, env(t.TempDir()))
	if err != nil || jobs[1].Pre != StateFailed || jobs[1].After != -1 {
		t.Fatalf("%v %+v", err, jobs[1])
	}
}

// A test lane that found no tests still hands its state on: coverage waits
// for it and inherits unavailable.
func TestSettleStillLinksATestLaneWithoutTests(t *testing.T) {
	jobs := []Job{
		{Name: "test/go", Kind: "test", Stack: "go", Area: ".", After: -1, Pre: StateUnavailable},
		{Name: "coverage/go", Kind: "coverage", Stack: "go", Area: ".", After: -1},
	}
	l := link{after: "test", measure: "go test ./...", repl: strings.NewReplacer()}
	if err := settle(jobs, []link{{}, l}, 1, Request{Kinds: []string{"test", "coverage"}}); err != nil {
		t.Fatal(err)
	}
	if jobs[1].After != 0 || len(jobs[1].Measure) != 0 {
		t.Fatalf("%+v", jobs[1])
	}
}

// A string override of test drops the preset's measuring, so the test lane
// writes no profile: coverage must not wait for it to, and measures itself.
func TestPlanCoverageMeasuresItselfWhenTestWritesNothingItReads(t *testing.T) {
	root := t.TempDir()
	req := Request{Kinds: []string{"test", "coverage"}}
	jobs, err := Plan(effFor(t, "[verify.go]\ntest = \"go test ./...\"\n", goOnly), req, env(root))
	if err != nil || len(jobs) != 2 {
		t.Fatalf("%v %+v", err, jobs)
	}
	profile, _ := CoverPaths(root, "R", "go", ".")
	cov := jobs[1]
	if cov.After != -1 || !slices.Contains(cov.Measure, "-coverprofile="+profile) || !slices.Equal(cov.Reads, []string{profile}) {
		t.Fatalf("%+v", cov)
	}

	// Without a measure step nothing in the run writes the profile.
	src := "[verify.go]\ntest = \"go test ./...\"\ncoverage = \"report {coverdata}\"\n"
	jobs, err = Plan(effFor(t, src, goOnly), req, env(root))
	if err != nil || jobs[1].Pre != StateFailed || jobs[1].After != -1 || !strings.Contains(jobs[1].Note, "does not write") {
		t.Fatalf("%v %+v", err, jobs[1])
	}
}

// Every Python lane gets COVERAGE_FILE, measuring or not, so the environment
// cannot tell a writer. The preset's test runs its measuring form: coverage
// waits for it, and reads the data file through that variable, unchecked.
func TestPlanLinksPythonCoverageToTheMeasuringTestLane(t *testing.T) {
	jobs, err := Plan(effFor(t, "", pythonOnly), Request{Kinds: []string{"test", "coverage"}}, env(t.TempDir()))
	if err != nil || jobs[1].After != 0 || jobs[1].Measure != nil || jobs[1].Reads != nil {
		t.Fatalf("%v %+v", err, jobs[1])
	}
}

// A string override of test drops the measuring form; the test lane still
// carries COVERAGE_FILE but writes nothing, so coverage measures itself, and
// without a measure step it has nothing to report on.
func TestPlanPythonCoverageMeasuresItselfBesideATestOverride(t *testing.T) {
	req := Request{Kinds: []string{"test", "coverage"}}
	src := "[verify.python]\ntest = \"uv run pytest\"\n"
	jobs, err := Plan(effFor(t, src, pythonOnly), req, env(t.TempDir()))
	if err != nil || jobs[1].After != -1 || !slices.Contains(jobs[1].Measure, "coverage") || jobs[1].Reads != nil {
		t.Fatalf("%v %+v", err, jobs[1])
	}
	src += "coverage = \"uv run coverage report\"\n"
	jobs, err = Plan(effFor(t, src, pythonOnly), req, env(t.TempDir()))
	if err != nil || jobs[1].Pre != StateFailed || jobs[1].After != -1 || !strings.Contains(jobs[1].Note, "does not write") {
		t.Fatalf("%v %+v", err, jobs[1])
	}
}

// A coverage lane that does not report with coverage.py reads no data file,
// however the environment is set: alone it just runs, and beside a test
// that measures nothing it waits for it like any lane that reads nothing.
func TestPlanPythonCoverageWithoutCoveragePyReadsNothing(t *testing.T) {
	src := "[verify.python]\ncoverage = \"uv run pytest --cov=src --cov-fail-under=100\"\n"
	jobs, err := Plan(effFor(t, src, pythonOnly), Request{Kinds: []string{"coverage"}}, env(t.TempDir()))
	if err != nil || jobs[0].Pre != "" || jobs[0].After != -1 || jobs[0].Measure != nil || len(jobs[0].Argvs) != 1 {
		t.Fatalf("%v %+v", err, jobs[0])
	}
	src += "test = \"uv run pytest\"\n"
	jobs, err = Plan(effFor(t, src, pythonOnly), Request{Kinds: []string{"test", "coverage"}}, env(t.TempDir()))
	if err != nil || jobs[1].Pre != "" || jobs[1].After != 0 || jobs[1].Measure != nil {
		t.Fatalf("%v %+v", err, jobs[1])
	}
}

func TestReportsWithCoveragePy(t *testing.T) {
	for argv, want := range map[string]bool{
		"uv run coverage report -m":        true,
		"coverage xml":                     true,
		"python -m coverage json -o x":     true,
		"coverage html":                    true,
		"coverage lcov":                    true,
		"uv run pytest --cov=src":          false,
		"report coverage":                  false,
		"uv run coverage run -m pytest -q": false,
	} {
		if got := reportsWithCoveragePy([][]string{strings.Fields(argv)}); got != want {
			t.Errorf("%q: %v", argv, got)
		}
	}
}

// An override that names the data file in its argv is a writer by that.
func TestPlanLinksPythonCoverageToATestOverrideNamingTheDataFile(t *testing.T) {
	src := "[verify.python]\ntest = \"uv run coverage run --data-file={coverdata} -m pytest\"\n"
	jobs, err := Plan(effFor(t, src, pythonOnly), Request{Kinds: []string{"test", "coverage"}}, env(t.TempDir()))
	if err != nil || jobs[1].After != 0 || jobs[1].Measure != nil {
		t.Fatalf("%v %+v", err, jobs[1])
	}
}
