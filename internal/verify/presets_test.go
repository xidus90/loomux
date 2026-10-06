package verify

import (
	"slices"
	"strings"
	"testing"
)

func TestTheEmbeddedPresetsLoad(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	if p.Extensions[".go"] != "go" || !slices.Contains(p.Ignored, ".lock") {
		t.Fatalf("%+v", p.Extensions)
	}
	if p.Stacks["python"].Variants[0].When != "pyright" {
		t.Fatal("pyright variant")
	}
	if p.Stacks["go"].Lanes["coverage"].After != "test" {
		t.Fatal("go coverage after test")
	}
	again, _ := LoadPresets()
	if again != p {
		t.Fatal("parsed twice")
	}
}

func TestTheEmbeddedPresetsCarryWhatTheBriefSays(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	goStack := p.Stacks["go"]
	if !slices.Equal(goStack.TestsWhen, []string{"*_test.go"}) || !goStack.Lanes["lint"].Threaded {
		t.Fatalf("%+v", goStack)
	}
	ts := p.Stacks["typescript"].Variants[0]
	if ts.When != "biome" || !slices.Equal(ts.Lanes["lint"].OnFile, []string{"npx biome check {file}"}) {
		t.Fatalf("%+v", ts)
	}
	if len(p.Stacks["shell"].Lanes["lint"].Commands) != 0 || p.Extensions[".md"] != "wiki" {
		t.Fatalf("%+v", p.Stacks["shell"])
	}
}

func TestParsePresetsRefuses(t *testing.T) {
	cases := []struct{ src, want string }{
		{"[stack.cobol.lint]\ncommands=[\"x\"]", "cobol"},
		{"[stack.go.style]\ncommands=[\"x\"]", "style"},
		{"[[stack.go.variant]]\nwhen = \"nothing\"", `"nothing"`},
		{"[extensions]\ngo = \"go\"", `"go"`},
		{"[extensions]\n\".x\" = \"cobol\"", "cobol"},
		{"[stack.go.lint]\ncommands=[]", "empty"},
		{"[stack.go.coverage]\ncommands=[\"{loomux} check gocover --profile {coverprofile}\"]", "coverprofile"},
		{"x = [", "presets.toml"},
		{"ignored = \".txt\"", "ignored"},
		{"ignored = [1]", "ignored"},
		{"extensions = 1", "extensions"},
		{"[extensions]\n\".x\" = 1", `".x"`},
		{"stack = 1", "stack"},
		{"unknown = 1", `"unknown"`},
		{"[stack]\ngo = 1", "[stack.go]"},
		{"[stack.go]\ntests_when = \"*_test.go\"", "tests_when"},
		{"[stack.go]\ntests_when = [1]", "tests_when"},
		{"[stack.go]\ntests_when = [\"\"]", "tests_when"},
		{"[stack.go]\nvariant = 1", "variant"},
		{"[stack.go]\nvariant = [1]", "variant"},
		{"[[stack.go.variant]]\n[stack.go.variant.lint]\ncommands=[\"x\"]", "when"},
		{"[[stack.go.variant]]\nwhen = 1", "when"},
		{"[[stack.go.variant]]\nwhen = \"biome\"\nstyle = 1", "style"},
		{"[[stack.go.variant]]\nwhen = \"biome\"\n[stack.go.variant.lint]\ncommands=[]", "empty"},
		{"[stack.go.test]\nafter = \"coverage\"\n[stack.go.coverage]\ncommands=[\"x\"]\nafter = \"test\"", "cycle"},
		{"[[stack.go.variant]]\nwhen = \"biome\"\n[stack.go.variant.coverage]\ncommands=[\"x {coverprofile}\"]", "coverprofile"},
	}
	for _, c := range cases {
		if _, err := parsePresets(c.src); err == nil || !strings.Contains(err.Error(), c.want) || !strings.HasPrefix(err.Error(), "presets.toml: ") {
			t.Errorf("%q: %v", c.src, err)
		}
	}
}

// A Python project that declares neither mypy's files nor coverage and
// pytest as dependencies still gets lanes that run: mypy is handed a target,
// and the tools come in through uv's --with. Where mypy is configured, the
// variant keeps the command that lets its configuration pick the files.
func TestThePythonLanesRunWithoutAMypyTargetOrToolDependencies(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	py := p.Stacks["python"]
	measure := "uv run --with coverage --with pytest coverage run -m pytest -q --tb=short --no-header"
	want := map[string][]string{
		"types":    {"uv run --with mypy mypy --no-error-summary --no-pretty --exclude-gitignore ."},
		"test":     {"uv run --with pytest pytest -q --tb=short --no-header"},
		"coverage": {"uv run --with coverage coverage report --skip-covered --skip-empty -m"},
	}
	for kind, cmds := range want {
		if got := py.Lanes[kind].Commands; !slices.Equal(got, cmds) {
			t.Errorf("%s = %q, want %q", kind, got, cmds)
		}
	}
	if py.Lanes["test"].Measuring != measure || py.Lanes["coverage"].Measure != measure {
		t.Errorf("measuring %q, measure %q", py.Lanes["test"].Measuring, py.Lanes["coverage"].Measure)
	}
	i := slices.IndexFunc(py.Variants, func(v Variant) bool { return v.When == "mypy" })
	if i < 0 || !slices.Equal(py.Variants[i].Lanes["types"].Commands, []string{"uv run mypy --no-error-summary --no-pretty"}) {
		t.Fatalf("mypy variant: %+v", py.Variants)
	}
	if i < slices.IndexFunc(py.Variants, func(v Variant) bool { return v.When == "pyright" }) {
		t.Fatal("pyright must come first: the first detected variant wins")
	}
}

// gdlint is a command of the package gdtoolkit; no package of its own name
// exists, so uvx has to be told where the command comes from.
func TestTheGdscriptLintTakesGdlintFromGdtoolkit(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	gd := p.Stacks["gdscript"].Lanes
	if got := gd["lint"].Commands; !slices.Equal(got, []string{"uvx --from gdtoolkit gdlint ."}) {
		t.Errorf("commands = %q", got)
	}
	if got := gd["lint"].OnFile; !slices.Equal(got, []string{"uvx --from gdtoolkit gdlint {file}"}) {
		t.Errorf("on_file = %q", got)
	}
}

// Booting Godot is no test: it exits 0 on a script that does not parse, and
// its binary is seldom on the PATH as `godot`. Lint and the root's graph lane
// are all the preset runs; a project names its own test command.
func TestTheGdscriptPresetHasOnlyALintAndAGraphLane(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	gd := p.Stacks["gdscript"].Lanes
	if _, ok := gd["lint"]; !ok || len(gd) != 2 || len(gd["graph"].Commands) == 0 {
		t.Errorf("lanes = %+v", gd)
	}
}

func TestAVariantInheritsTheLanesItDoesNotName(t *testing.T) {
	src := "[stack.go.test]\nmeasuring = \"go test -coverprofile={coverprofile}\"\n" +
		"[[stack.go.variant]]\nwhen = \"biome\"\n[stack.go.variant.coverage]\ncommands=[\"x {coverprofile}\"]\n"
	p, err := parsePresets(src)
	if err != nil {
		t.Fatal(err)
	}
	if p.Stacks["go"].Variants[0].Lanes["coverage"].Commands[0] != "x {coverprofile}" {
		t.Fatalf("%+v", p.Stacks["go"])
	}
}

func TestCheckCoverProfile(t *testing.T) {
	reads := Lane{Commands: []string{"report {coverprofile}"}}
	writes := "go test -coverprofile={coverprofile}"
	ok := []map[string]Lane{
		{},
		{"coverage": {Commands: []string{"report"}}},
		{"coverage": reads, "test": {Commands: []string{writes}}},
		{"coverage": reads, "test": {Measuring: writes}},
		{"coverage": {Commands: reads.Commands, Measure: writes}},
	}
	for _, lanes := range ok {
		if err := checkCoverProfile("stack.go", lanes); err != nil {
			t.Errorf("%+v: %v", lanes, err)
		}
	}
	err := checkCoverProfile("stack.go", map[string]Lane{"coverage": reads, "test": {Commands: []string{"go test"}}})
	want := "[stack.go].coverage reads {coverprofile}, but nothing in test or coverage.measure writes it"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v", err)
	}
}

// BenchmarkLoadPresets measures the parse the first LoadPresets pays, apart
// from the process start that hides it in every command.
func BenchmarkLoadPresets(b *testing.B) {
	for b.Loop() {
		if _, err := parsePresets(presetsText); err != nil {
			b.Fatal(err)
		}
	}
}

// The Go graph lane first drives the graph to fresh, then audits the index
// against it; both commands, since a lane that only audits reads a stale graph.
func TestTheGoGraphLaneRefreshesThenAudits(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	lane := p.Stacks["go"].Lanes["graph"]
	want := []string{"{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"}
	if !slices.Equal(lane.Commands, want) || lane.OnFile != nil || lane.Threaded {
		t.Fatalf("%+v", lane)
	}
}

// The graph belongs to the root, not to a stack, so the graph lanes of the
// tree-sitter languages run the very commands of Go's; the plan keeps one of
// them per run.
func TestTreeSitterGraphLanesAreTheGoOne(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	goLane := p.Stacks["go"].Lanes["graph"]
	for _, stack := range []string{"python", "gdscript"} {
		lane := p.Stacks[stack].Lanes["graph"]
		if len(lane.Commands) == 0 || !slices.Equal(lane.Commands, goLane.Commands) || lane.OnFile != nil || lane.Threaded {
			t.Errorf("%s %+v, go %+v", stack, lane, goLane)
		}
	}
}

// The C++ lanes that read the build tree need it configured, clang-tidy's
// lint among them; the edit form of the lint formats one file, and needs do
// not guard it.
func TestTheCppLanesOnTheBuildTreeNeedItConfigured(t *testing.T) {
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	cpp := p.Stacks["cpp"].Lanes
	for _, kind := range []string{"lint", "types", "test", "coverage"} {
		if !slices.Equal(cpp[kind].Needs, []string{"build/CMakeCache.txt"}) {
			t.Errorf("%s: needs %q", kind, cpp[kind].Needs)
		}
	}
}
