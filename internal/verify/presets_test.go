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
