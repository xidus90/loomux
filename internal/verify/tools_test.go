package verify

import (
	"slices"
	"testing"
)

// A tool placeholder is one whole argument: {tool:x} is the requirement the
// tool runs from, {from:x} the option uvx takes one from, and an unpinned
// tool keeps the arguments the presets had before pins existed.
func TestToolArg(t *testing.T) {
	for _, c := range []struct {
		arg   string
		tools map[string]string
		want  string
		keep  bool
	}{
		{"{tool:mypy}", nil, "mypy", true},
		{"{tool:mypy}", map[string]string{"mypy": "1.16.*"}, "mypy==1.16.*", true},
		{"{tool:mypy}", map[string]string{"mypy": ">=1.16,<2"}, "mypy>=1.16,<2", true},
		{"{tool:x}", map[string]string{"mypy": "1"}, "x", true},
		{"{from:ruff}", nil, "", false},
		{"{from:ruff}", map[string]string{"ruff": "0.6.*"}, "--from=ruff==0.6.*", true},
		{"check", map[string]string{"ruff": "1"}, "check", true},
		{"{tool:x", nil, "{tool:x", true},
		{"{file}", nil, "{file}", true},
		{"x{tool:ruff}", map[string]string{"ruff": "1"}, "x{tool:ruff}", true},
	} {
		got, keep := toolArg(c.arg, c.tools)
		if got != c.want || keep != c.keep {
			t.Errorf("toolArg(%q, %v) = %q, %v; want %q, %v", c.arg, c.tools, got, keep, c.want, c.keep)
		}
	}
}

// expand drops an unpinned {from:} and keeps every other argument in place.
func TestExpandDropsAnUnpinnedFrom(t *testing.T) {
	argv, err := expand("uvx {from:ruff} ruff check .", nil, nil)
	if err != nil || !slices.Equal(argv, []string{"uvx", "ruff", "check", "."}) {
		t.Fatalf("%q %v", argv, err)
	}
	argv, err = expand("uvx {from:ruff} ruff check .", nil, map[string]string{"ruff": "0.6.9"})
	if err != nil || !slices.Equal(argv, []string{"uvx", "--from=ruff==0.6.9", "ruff", "check", "."}) {
		t.Fatalf("%q %v", argv, err)
	}
}

// Without pins every preset of a stack with tools runs the arguments it ran
// before tools could be pinned: the replay cases under testdata/cases record
// them, and a project that pins nothing must not notice.
func TestUnpinnedPresetsKeepTheirArguments(t *testing.T) {
	before := map[string]string{
		"python/lint":              "uvx ruff check . --output-format=concise",
		"python/types":             "uv run --with mypy mypy --no-error-summary --no-pretty --exclude-gitignore .",
		"python/test":              "uv run --with pytest pytest -q --tb=short --no-header",
		"python/test measuring":    "uv run --with coverage --with pytest coverage run -m pytest -q --tb=short --no-header",
		"python/coverage":          "uv run --with coverage coverage report --skip-covered --skip-empty -m",
		"python/coverage measure":  "uv run --with coverage --with pytest coverage run -m pytest -q --tb=short --no-header",
		"gdscript/lint #1":         "uvx --from gdtoolkit gdlint .",
		"gdscript/lint #2":         "uvx --from gdtoolkit gdformat --check .",
		"gdscript/lint on_file #1": "uvx --from gdtoolkit gdlint {file}",
		"gdscript/lint on_file #2": "uvx --from gdtoolkit gdformat --check {file}",
	}
	p, err := LoadPresets()
	if err != nil {
		t.Fatal(err)
	}
	now := map[string]string{}
	for _, stack := range []string{"python", "gdscript"} {
		lanes := p.Stacks[stack].Lanes
		for kind, l := range lanes {
			key := stack + "/" + kind
			for i, c := range l.Commands {
				k := key
				if len(l.Commands) > 1 {
					k = key + " #" + string(rune('1'+i))
				}
				now[k] = c
			}
			for i, c := range l.OnFile {
				if len(l.OnFile) > 1 {
					now[key+" on_file #"+string(rune('1'+i))] = c
				}
			}
			if l.Measuring != "" {
				now[key+" measuring"] = l.Measuring
			}
			if l.Measure != "" {
				now[key+" measure"] = l.Measure
			}
		}
	}
	for k, old := range before {
		c, ok := now[k]
		if !ok {
			t.Errorf("%s: no such preset command", k)
			continue
		}
		got, err := expand(c, nil, nil)
		want, err2 := expand(old, nil, nil)
		if err != nil || err2 != nil || !slices.Equal(got, want) {
			t.Errorf("%s: unpinned %q, before %q", k, got, want)
		}
		if c == old {
			t.Errorf("%s: %q names no tool placeholder", k, c)
		}
	}
}

// A command shown to a person names its tools as they run: pinned where the
// project pins them, as before where it does not, every other word as it is.
func TestShowTools(t *testing.T) {
	for _, c := range []struct {
		cmd   string
		tools map[string]string
		want  string
	}{
		{"uvx {from:ruff} ruff check {file}", nil, "uvx ruff check {file}"},
		{"uvx {from:ruff} ruff check {file}", map[string]string{"ruff": "0.6.*"}, "uvx --from=ruff==0.6.* ruff check {file}"},
		{"uv run --with {tool:mypy} mypy .", map[string]string{"mypy": ">=1.16"}, "uv run --with mypy>=1.16 mypy ."},
		{"go vet ./...", nil, "go vet ./..."},
	} {
		if got := ShowTools(c.cmd, c.tools); got != c.want {
			t.Errorf("ShowTools(%q) = %q, want %q", c.cmd, got, c.want)
		}
	}
}
