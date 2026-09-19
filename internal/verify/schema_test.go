package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

func parse(t *testing.T, src string) (Config, error) {
	t.Helper()
	doc := map[string]any{}
	if _, err := toml.Decode(src, &doc); err != nil {
		t.Fatalf("fixture is not TOML: %v", err)
	}
	return ParseConfig("cfg.toml", doc)
}

func TestParseConfigRefuses(t *testing.T) {
	cases := []struct{ src, want string }{
		{"[verify]\nparallelism = 4", `[verify] has unknown key "parallelism"`},
		{"[verify]\ntypes = \"mypy\"", "[verify].types is the old form; name the stack: [verify.<stack>].types"},
		{"[verify.cobol]\nlint = \"x\"", "[verify.cobol] is not a stack; stacks are: go, python, typescript,"},
		{"[verify.go]\nstyle = \"x\"", `[verify.go] has unknown key "style"`},
		{"[verify.go]\nimport_check = true", `[verify.go] has unknown key "import_check"`},
		{"[verify.wiki]\nlint = \"x\"", "[verify.wiki].lint can only be false"},
		{"[verify.wiki]\ntypes = false", "[verify.wiki].lint can only be false"},
		{"[verify.go]\nlint = true", "[verify.go].lint = true is not a command; leave the key out to keep the preset"},
		{"[verify.go]\nlint = []", "[verify.go].lint is empty"},
		{"[verify.go]\nlint = [\"a\", \" \"]", "[verify.go].lint #2 is empty"},
		{"[verify.go]\nlint = \"a 'b\"", "[verify.go].lint #1: unclosed quote"},
		{"[verify.go]\nlint = \"x {file}\"", "[verify.go].lint uses {file}, which only on_file knows"},
		{"[verify.go.lint]\ncommands = [\"x\"]\nfoo = 1", `[verify.go.lint] has unknown key "foo"`},
		{"[verify.go.lint]\ncommands = [\"x\"]\nmeasuring = \"y\"", "[verify.go.lint] cannot have measuring"},
		{"[verify.go.types]\nafter = \"lint\"", "[verify.go.types] cannot have after"},
		{"[verify.go.coverage]\ncommands = [\"x\"]\nafter = \"style\"", `[verify.go.coverage].after names unknown kind "style"`},
		{"[verify.go.test]\ncommands = [\"x\"]\nafter = \"coverage\"\n[verify.go.coverage]\ncommands = [\"y\"]\nafter = \"test\"", "[verify.go] after forms a cycle: test -> coverage -> test"},
		{"[verify.go.test]\nafter = \"test\"", "[verify.go] after forms a cycle: test -> test"},
		{"[verify]\nmax_parallel = 0", "[verify].max_parallel must be a positive integer, found 0"},
		{"[verify]\nmax_parallel = \"4\"", "[verify].max_parallel must be a positive integer, found 4"},
		{"[verify]\ntimeout = \"10s\"", "[verify].timeout must be a positive number of seconds, found 10s"},
		{"[verify]\ntimeout = 0", "[verify].timeout must be a positive number of seconds, found 0"},
		{"[verify.profiles]\nlint = [\"lint\"]", "[verify.profiles].lint collides with a reserved name"},
		{"[verify.profiles]\nall = [\"lint\"]", "[verify.profiles].all collides with a reserved name"},
		{"[verify.profiles]\nx = []", "[verify.profiles].x is empty"},
		{"[verify.profiles]\nx = [\"style\"]", `[verify.profiles].x names unknown kind "style"`},
		{"[verify.profiles]\nx = [1]", `[verify.profiles].x names unknown kind "1"`},
		// Shapes TOML allows but the schema does not.
		{"verify = 1", "[verify] must be a table"},
		{"[verify]\ngo = 1", "[verify.go] must be a table"},
		{"[verify]\nprofiles = 1", "[verify.profiles] must be a table"},
		{"[verify.profiles]\nx = \"lint\"", "[verify.profiles].x must be a list of kinds"},
		{"[verify.go]\nlint = [1]", "[verify.go].lint #1 must be a string"},
		{"[verify.go]\nlint = 1", "[verify.go].lint must be false, a command, a list of commands or a table"},
		{"[verify.go.lint]\nthreaded = 1", "[verify.go.lint].threaded must be a boolean"},
		{"[verify.go.lint]", "[verify.go].lint is empty"},
		{"[verify.go.lint]\ncommands = \"x\"", "[verify.go.lint].commands must be a list of commands"},
		{"[verify.go.lint]\ncommands = []", "[verify.go.lint] commands is empty"},
		{"[verify.go.lint]\non_file = [\"a 'b\"]", "[verify.go.lint] on_file #1: unclosed quote"},
		{"[verify.go.test]\nmeasuring = 1", "[verify.go.test].measuring must be a string"},
		{"[verify.go.test]\nmeasure = \" \"", "[verify.go.test] measure #1 is empty"},
		{"[verify.go.test]\nmeasure = \"x {file}\"", "[verify.go.test] measure uses {file}, which only on_file knows"},
		{"[verify.gdscript]\nimport_check = 1", "[verify.gdscript].import_check must be a boolean"},
	}
	for _, c := range cases {
		_, err := parse(t, c.src)
		if err == nil || !strings.HasPrefix(err.Error(), "cfg.toml: ") || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want %q", c.src, err, c.want)
		}
	}
}

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := parse(t, "")
	if err != nil || cfg.MaxParallel < 1 || cfg.Timeout != 600*time.Second || !cfg.ImportCheck {
		t.Fatalf("%+v %v", cfg, err)
	}
	if strings.Join(cfg.Profiles["edit"], ",") != "lint,types" || len(cfg.Profiles["precommit"]) != 4 {
		t.Fatalf("%v", cfg.Profiles)
	}
}

func TestParseConfigForms(t *testing.T) {
	cfg, err := parse(t, `
[verify]
max_parallel = 3
timeout = 90
[verify.profiles]
edit = ["lint"]
fast = ["lint", "test"]
[verify.go]
lint = "go vet ./..."
types = false
[verify.go.test]
measuring = "go test -coverprofile={coverprofile} ./..."
[verify.go.coverage]
measure = "go tool cover -func={coverprofile}"
after = "test"
[verify.python]
lint = ["ruff check .", "ruff format --check ."]
[verify.typescript.lint]
commands = ["npx eslint ."]
on_file = ["npx eslint --cache {file}"]
threaded = true
[verify.gdscript]
import_check = false
[verify.wiki]
lint = false
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxParallel != 3 || cfg.Timeout != 90*time.Second || cfg.ImportCheck {
		t.Fatalf("%+v", cfg)
	}
	if strings.Join(cfg.Profiles["edit"], ",") != "lint" || strings.Join(cfg.Profiles["fast"], ",") != "lint,test" || len(cfg.Profiles["precommit"]) != 4 {
		t.Fatalf("%v", cfg.Profiles)
	}
	golint := cfg.Stacks["go"]["lint"]
	if !golint.Replace || golint.Lane.Commands[0] != "go vet ./..." {
		t.Fatalf("%+v", golint)
	}
	if types := cfg.Stacks["go"]["types"]; !types.Lane.Off || !types.Replace {
		t.Fatalf("types = false: %+v", types)
	}
	gotest := cfg.Stacks["go"]["test"]
	if gotest.Replace || !gotest.Set["measuring"] || gotest.Set["commands"] || gotest.Lane.Measuring != "go test -coverprofile={coverprofile} ./..." {
		t.Fatalf("%+v", gotest)
	}
	cover := cfg.Stacks["go"]["coverage"]
	if cover.Lane.After != "test" || cover.Lane.Measure != "go tool cover -func={coverprofile}" || !cover.Set["after"] || !cover.Set["measure"] {
		t.Fatalf("%+v", cover)
	}
	py := cfg.Stacks["python"]["lint"]
	if !py.Replace || strings.Join(py.Lane.Commands, "|") != "ruff check .|ruff format --check ." {
		t.Fatalf("%+v", py)
	}
	ts := cfg.Stacks["typescript"]["lint"]
	if !ts.Lane.Threaded || ts.Lane.OnFile[0] != "npx eslint --cache {file}" || ts.Lane.Commands[0] != "npx eslint ." {
		t.Fatalf("%+v", ts)
	}
	if !ts.Set["commands"] || !ts.Set["on_file"] || !ts.Set["threaded"] || len(ts.Set) != 3 {
		t.Fatalf("%+v", ts.Set)
	}
	if !cfg.Stacks["wiki"]["lint"].Lane.Off {
		t.Fatal("wiki lint = false")
	}
}

func TestParseLaneNamesItsOwner(t *testing.T) {
	_, err := parseLane("stack.go", "lint", true)
	if err == nil || err.Error() != "[stack.go].lint = true is not a command; leave the key out to keep the preset" {
		t.Fatalf("%v", err)
	}
	_, err = parseLane("stack.go", "lint", map[string]any{"foo": 1})
	if err == nil || err.Error() != `[stack.go.lint] has unknown key "foo"` {
		t.Fatalf("%v", err)
	}
}

func TestCheckCycleIgnoresAnAfterOutsideTheLanes(t *testing.T) {
	lanes := map[string]Override{"coverage": {Lane: Lane{After: "test"}}}
	if err := checkCycle("verify.go", lanes); err != nil {
		t.Fatal(err)
	}
}

func TestNames(t *testing.T) {
	if strings.Join(Kinds(), ",") != "lint,types,test,coverage" {
		t.Fatal(Kinds())
	}
	if strings.Join(StackNames(), ",") != "go,python,typescript,vue,svelte,css,html,gdscript,cpp,shell,sql,rust,wiki,project" {
		t.Fatal(StackNames())
	}
	for _, name := range []string{"gofmt", "commit-msg", "gocover", "all", "lint", "types", "test", "coverage"} {
		if !Reserved(name) {
			t.Errorf("%s is not reserved", name)
		}
	}
	if Reserved("edit") {
		t.Error("edit is reserved")
	}
}

func TestReadConfigOfAMissingFileIsTheDefault(t *testing.T) {
	cfg, err := ReadConfig(t.TempDir())
	if err != nil || cfg.Timeout != 600*time.Second {
		t.Fatalf("%+v %v", cfg, err)
	}
}

func writeManifest(t *testing.T, src string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestReadConfigNamesTheFileOfBrokenTOML(t *testing.T) {
	root := writeManifest(t, "[verify")
	if _, err := ReadConfig(root); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("%v", err)
	}
}

func TestReadConfigNamesTheFileItCannotRead(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(root); err == nil || !strings.Contains(err.Error(), "config.toml") {
		t.Fatalf("%v", err)
	}
}

func TestReadConfigParsesTheVerifySection(t *testing.T) {
	root := writeManifest(t, "[verify]\ntimeout = 5\n")
	cfg, err := ReadConfig(root)
	if err != nil || cfg.Timeout != 5*time.Second {
		t.Fatalf("%+v %v", cfg, err)
	}
	root = writeManifest(t, "[verify]\ntimeout = 0\n")
	if _, err := ReadConfig(root); err == nil || !strings.HasPrefix(err.Error(), filepath.Join(root, ".loomux", "config.toml")+": ") {
		t.Fatalf("%v", err)
	}
}
