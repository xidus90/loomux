package verify

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
)

func TestToolsPinsAPresetTool(t *testing.T) {
	cfg, err := parse(t, "[verify.python]\ntools = { ruff = \"0.6.*\", mypy = \">=1.16\" }\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Tools["python"]; len(got) != 2 || got["ruff"] != "0.6.*" || got["mypy"] != ">=1.16" {
		t.Fatalf("%v", cfg.Tools)
	}
}

func TestToolsRefusesUnknownNamesAndBadSpecs(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"[verify.python]\ntools = { ruf = \"1\" }", `[verify.python].tools has unknown tool "ruf"; known: coverage, mypy, pytest, ruff`},
		{"[verify.gdscript]\ntools = { ruff = \"1\" }", `[verify.gdscript].tools has unknown tool "ruff"; known: gdtoolkit`},
		{"[verify.python]\ntools = { ruff = \"\" }", "[verify.python].tools.ruff must be a version"},
		{"[verify.python]\ntools = { ruff = \"1 .2\" }", "[verify.python].tools.ruff must be a version"},
		{"[verify.python]\ntools = { ruff = \"'1'\" }", "[verify.python].tools.ruff must be a version"},
		{"[verify.python]\ntools = { ruff = 1 }", "[verify.python].tools.ruff must be a version"},
		{"[verify.python]\ntools = \"x\"", "[verify.python].tools must be a table"},
		{"[verify.go]\ntools = { x = \"1\" }", "[verify.go].tools: go has no tools to pin"},
	} {
		_, err := parse(t, c.src)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.src, err, c.want)
		}
	}
}

// Presets that do not load leave no tool to check a pin against, and the
// config says why instead of passing the pin unchecked.
func TestToolsNeedThePresets(t *testing.T) {
	old := loadPresets
	defer func() { loadPresets = old }()
	loadPresets = func() (*Presets, error) { return nil, errors.New("presets.toml: broken") }
	_, err := parse(t, "[verify.python]\ntools = { ruff = \"1\" }\n")
	if err == nil || !strings.Contains(err.Error(), "presets.toml: broken") {
		t.Fatalf("%v", err)
	}
}

// A pin reaches the arguments a lane runs; without one they are as before.
func TestPlanPinsTheToolsOfItsStack(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"python"}}
	e := env(t.TempDir())
	jobs, err := Plan(effFor(t, "[verify.python]\ntools = { ruff = \"0.6.9\" }\n", facts), Request{Kinds: []string{"lint"}}, e)
	if err != nil || len(jobs) != 1 || !slices.Equal(jobs[0].Argvs[0], []string{"uvx", "--from=ruff==0.6.9", "ruff", "check", ".", "--output-format=concise"}) {
		t.Fatalf("%+v %v", jobs, err)
	}
	jobs, err = Plan(effFor(t, "", facts), Request{Kinds: []string{"lint"}}, e)
	if err != nil || len(jobs) != 1 || !slices.Equal(jobs[0].Argvs[0], []string{"uvx", "ruff", "check", ".", "--output-format=concise"}) {
		t.Fatalf("%+v %v", jobs, err)
	}
}
