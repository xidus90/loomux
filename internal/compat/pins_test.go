package compat

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/verify"
)

// A tool is pinned when uv's constraints bind it, or when every command of
// the copy's resolved lanes that names it names it through a placeholder.
func TestEveryToolOfAPythonEntryIsPinnedAfterPreparation(t *testing.T) {
	entries, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	presets, err := verify.LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Stack != "python" {
			continue
		}
		dir := filepath.Join(t.TempDir(), "w")
		if err := prepare(e, RunOptions{}, dir, nil); err != nil {
			t.Fatal(err)
		}
		var py struct {
			Tool struct {
				UV struct {
					Constraints []string `toml:"constraint-dependencies"`
				} `toml:"uv"`
			} `toml:"tool"`
		}
		if _, err := toml.DecodeFile(filepath.Join(dir, "pyproject.toml"), &py); err != nil {
			t.Fatal(err)
		}
		cfg, err := verify.ReadConfig(dir)
		if err != nil {
			t.Fatal(err)
		}
		eff, err := verify.Resolve(cfg, presets, detect.Detect(os.DirFS(dir)))
		if err != nil {
			t.Fatal(err)
		}
		var commands []string
		for _, r := range eff.Stacks["python"] {
			commands = append(commands, r.Lane.Commands...)
			commands = append(commands, r.Lane.Measure, r.Lane.Measuring)
		}
		for name, spec := range e.Tools {
			if slices.Contains(py.Tool.UV.Constraints, name+"=="+spec) {
				continue
			}
			for _, c := range commands {
				words := strings.Fields(c)
				if slices.Contains(words, name) && !slices.Contains(words, "{tool:"+name+"}") && !slices.Contains(words, "{from:"+name+"}") {
					t.Errorf("%s: %s runs unpinned in %q", e.Key(), name, c)
				}
			}
		}
	}
}

func TestARequirementKeepsTheOperatorOfItsSpec(t *testing.T) {
	if got := requirement("mypy", ">=2,<3"); got != "mypy>=2,<3" {
		t.Errorf("requirement = %q", got)
	}
	if got := requirement("mypy", "2.4.*"); got != "mypy==2.4.*" {
		t.Errorf("requirement = %q", got)
	}
}
