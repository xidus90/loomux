package verify

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
)

func presetsFor(t *testing.T) *Presets {
	t.Helper()
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveLayersPresetVariantAndConfig(t *testing.T) {
	cfg, _ := parse(t, `
[verify.python]
lint = "ruff check ."
[verify.go.test]
measuring = "go test -coverpkg=x/... -coverprofile={coverprofile} ./..."
[verify.sql]
lint = "sqlfluff lint ."
`)
	facts := detect.Facts{
		Stacks: []string{"go", "pyright", "python"},
		Areas:  map[string][]string{"go": {"."}, "python": {"."}, "pyright": {"."}},
	}
	eff, err := Resolve(cfg, presetsFor(t), facts)
	if err != nil {
		t.Fatal(err)
	}
	if got := eff.Stacks["python"]["types"]; got.Origin != "preset, variant pyright" || got.Lane.Commands[0] != "uv run pyright" {
		t.Fatalf("%+v", got)
	}
	if got := eff.Stacks["python"]["lint"]; got.Origin != "config" || got.Lane.Commands[0] != "ruff check ." {
		t.Fatalf("%+v", got)
	}
	gotest := eff.Stacks["go"]["test"]
	if gotest.Lane.Commands[0] != "go test ./... -count=1" || !strings.Contains(gotest.Lane.Measuring, "coverpkg") {
		t.Fatalf("table merges: %+v", gotest)
	}
	if !slices.Equal(eff.Active, []string{"go", "python", "sql"}) || !slices.Equal(eff.Areas["sql"], []string{"."}) {
		t.Fatalf("%v %v", eff.Active, eff.Areas)
	}
	if eff.Stacks["go"]["types"].Defined {
		t.Fatal("go has no types")
	}
}

func TestResolveRefusesACoverProfileNobodyWrites(t *testing.T) {
	cfg, _ := parse(t, "[verify.go]\ntest = \"go test ./...\"\n[verify.go.coverage]\nmeasure = \"go test ./...\"\n")
	_, err := Resolve(cfg, presetsFor(t), detect.Facts{Stacks: []string{"go"}})
	if err == nil || !strings.Contains(err.Error(), "nothing in test or coverage.measure writes it") {
		t.Fatalf("%v", err)
	}
	if want := "[verify.go].coverage reads {coverprofile}"; !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("%v", err)
	}
}

// The config alone has no ring; only laid over the preset's coverage.after
// does it close one.
func TestResolveRefusesARingAcrossPresetAndConfig(t *testing.T) {
	cfg, err := parse(t, "[verify.go.test]\nafter = \"coverage\"\n")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Resolve(cfg, presetsFor(t), detect.Facts{Stacks: []string{"go"}})
	if err == nil || err.Error() != "[verify.go] after forms a cycle: test -> coverage -> test" {
		t.Fatalf("%v", err)
	}
}

func TestResolveConfigForms(t *testing.T) {
	cfg, err := parse(t, `
[verify.go]
coverage = "go tool cover -func={coverprofile}"
[verify.go.lint]
commands = ["golangci-lint run"]
on_file = ["gofmt -l {file}"]
threaded = false
[verify.go.test]
commands = ["go test -coverprofile={coverprofile} ./..."]
measuring = "go test -race ./..."
[verify.python]
types = false
[verify.python.coverage]
measure = "uv run coverage run -m pytest"
after = "lint"
`)
	if err != nil {
		t.Fatal(err)
	}
	eff, err := Resolve(cfg, presetsFor(t), detect.Facts{Stacks: []string{"go"}})
	if err != nil {
		t.Fatal(err)
	}
	cov := eff.Stacks["go"]["coverage"]
	if cov.Origin != "config" || cov.Lane.After != "test" || cov.Lane.Measure != "" || !cov.Defined {
		t.Fatalf("a string replaces, after stays: %+v", cov)
	}
	lint := eff.Stacks["go"]["lint"].Lane
	if lint.Commands[0] != "golangci-lint run" || lint.OnFile[0] != "gofmt -l {file}" || lint.Threaded {
		t.Fatalf("%+v", lint)
	}
	if got := eff.Stacks["go"]["test"].Lane; got.Measuring != "go test -race ./..." {
		t.Fatalf("%+v", got)
	}
	types := eff.Stacks["python"]["types"]
	if types.Defined || types.Origin != "config" {
		t.Fatalf("false switches off: %+v", types)
	}
	pycov := eff.Stacks["python"]["coverage"].Lane
	if pycov.After != "lint" || pycov.Measure != "uv run coverage run -m pytest" || pycov.Commands[0] != "uv run coverage report --skip-covered --skip-empty -m" {
		t.Fatalf("%+v", pycov)
	}
	if !eff.Configured["go"] || eff.Configured["python"] {
		t.Fatalf("%v", eff.Configured)
	}
	if !slices.Equal(eff.Active, []string{"go"}) { // off and a table without a command do not activate
		t.Fatalf("%v", eff.Active)
	}
}

func TestResolveActiveAndAreas(t *testing.T) {
	p := presetsFor(t)
	facts := detect.Facts{
		Stacks: []string{"project", "typescript", "wiki", "biome", "shell"},
		Areas:  map[string][]string{"typescript": {"web", "admin"}, "wiki": {"."}},
	}
	eff, err := Resolve(defaults(), p, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eff.Active, []string{"shell", "typescript"}) {
		t.Fatalf("project needs config, wiki is not resolved: %v", eff.Active)
	}
	if _, ok := eff.Stacks["wiki"]; ok {
		t.Fatal("wiki is not resolved here")
	}
	if !slices.Equal(eff.Areas["typescript"], []string{"web", "admin"}) || !slices.Equal(eff.Areas["shell"], []string{"."}) {
		t.Fatalf("%v", eff.Areas)
	}
	lint := eff.Stacks["typescript"]["lint"]
	if lint.Origin != "preset, variant biome" || lint.Lane.Commands[0] != "npx biome check ." {
		t.Fatalf("%+v", lint)
	}
	if got := eff.Stacks["typescript"]["types"]; got.Origin != "preset" || !got.Defined {
		t.Fatalf("a kind the variant leaves alone stays preset: %+v", got)
	}
	if !eff.Stacks["shell"]["lint"].Defined {
		t.Fatal("on_file alone defines a lane")
	}
	if !slices.Equal(eff.TestsWhen["typescript"], []string{"package.json:vitest"}) || eff.Configured["typescript"] {
		t.Fatalf("%v %v", eff.TestsWhen, eff.Configured)
	}
	if eff.Extensions[".go"] != "go" || !slices.Contains(eff.Ignored, ".lock") {
		t.Fatalf("%v %v", eff.Extensions, eff.Ignored)
	}
	eff.Ignored[0] = "changed"
	if p.Ignored[0] == "changed" {
		t.Fatal("the presets are shared; Resolve hands out a copy")
	}

	cfg, _ := parse(t, "[verify.project]\ntest = \"make test\"\n[verify.rust.test]\nthreaded = true\n")
	eff, err = Resolve(cfg, p, detect.Facts{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(eff.Active, []string{"project"}) || !slices.Equal(eff.Areas["project"], []string{"."}) {
		t.Fatalf("a table without a command does not activate: %v %v", eff.Active, eff.Areas)
	}
	if eff.Config.MaxParallel != cfg.MaxParallel {
		t.Fatal("the config travels along")
	}
}

// Only whoever writes the test command vouches for tests: a table that
// changes measuring alone keeps the preset's command and its test search.
func TestResolveConfiguredNeedsATestCommand(t *testing.T) {
	for src, want := range map[string]bool{
		"[verify.go]\ntest = \"go test ./...\"\n":                                        true,
		"[verify.go.test]\ncommands = [\"go test ./...\"]\n":                             true,
		"[verify.go.test]\nmeasuring = \"go test -coverprofile={coverprofile} ./...\"\n": false,
		"[verify.go.test]\nthreaded = true\n":                                            false,
		"[verify.go]\ntest = false\n":                                                    false,
	} {
		cfg, err := parse(t, src)
		if err != nil {
			t.Fatal(err)
		}
		eff, err := Resolve(cfg, presetsFor(t), goOnly)
		if err != nil {
			t.Fatal(err)
		}
		if eff.Configured["go"] != want {
			t.Errorf("%q: configured %v", src, eff.Configured["go"])
		}
	}
}
