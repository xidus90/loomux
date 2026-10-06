package verify

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/detect"
)

const showGo = `[verify.go.lint]
commands = ["go vet ./...", "{loomux} check gofmt ."]  # preset
on_file = ["go vet ./...", "{loomux} check gofmt {file}"]  # preset
threaded = true  # preset

# [verify.go.types] not defined

[verify.go.test]
commands = ["go test ./... -count=1"]  # preset
measuring = "go test ./... -count=1 -covermode=set -coverprofile={coverprofile}"  # preset

[verify.go.coverage]
commands = ["{loomux} check gocover --profile {coverprofile}"]  # preset
measure = "go test ./... -count=1 -covermode=set -coverprofile={coverprofile}"  # preset
after = "test"  # preset

[verify.go.graph]
commands = ["{loomux} check graph-fresh", "{loomux} check blast-audit --cached --threshold 5"]  # preset
`

func TestWriteShowGoOnly(t *testing.T) {
	for _, found := range []bool{true, false} {
		eff := effFor(t, "", goOnly)
		eff.Config.MaxParallel = 8
		e := env(`C:\repo`)
		asked := []string{}
		e.HasTests = func(dir string, patterns []string) bool {
			asked = append(asked, dir+" "+strings.Join(patterns, ","))
			return found
		}
		var b strings.Builder
		WriteShow(&b, eff, Kinds(), e)
		state := "tests found"
		if !found {
			state = "no tests found"
		}
		want := "# max_parallel = 8, timeout = 600s\n\n# go: areas ., " + state + "\n" + showGo
		if b.String() != want {
			t.Fatalf("%v:\n%s", found, b.String())
		}
		if len(asked) != 1 || asked[0] != filepath.Join(`C:\repo`, ".")+" *_test.go" {
			t.Fatalf("%v", asked)
		}
	}
}

// A stack without a test signal says so instead of guessing; a stack in
// several areas names them all, and one area with tests is enough. On-file
// commands and a config origin show as the resolved lane has them.
func TestWriteShowAreasOriginsAndNoSignal(t *testing.T) {
	facts := detect.Facts{Stacks: []string{"go", "sql"}, Areas: map[string][]string{"go": {"b", "a"}}}
	eff := effFor(t, "[verify.go.lint]\non_file = [\"gofmt -l {file}\"]\n", facts)
	eff.Config.MaxParallel = 2
	e := env(`C:\repo`)
	e.HasTests = func(dir string, _ []string) bool { return strings.HasSuffix(dir, "b") }
	eff.TestsWhen["sql"] = nil
	var b strings.Builder
	WriteShow(&b, eff, []string{"lint"}, e)
	got := b.String()
	for _, want := range []string{
		"# max_parallel = 2, timeout = 600s\n\n# go: areas a, b, tests found\n[verify.go.lint]\n",
		"on_file = [\"gofmt -l {file}\"]  # config\n",
		"\n\n# sql: areas ., tests not detected\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in\n%s", want, got)
		}
	}
}

// What --show prints loads back: an undefined lane is a comment, an off lane
// a `<kind> = false` in the stack's table, and both come back as they were.
func TestWriteShowLoadsBack(t *testing.T) {
	eff := effFor(t, "[verify.go]\nlint = false\n", goOnly)
	var b strings.Builder
	WriteShow(&b, eff, Kinds(), env(`C:\repo`))
	got := b.String()
	for _, want := range []string{
		"[verify.go]\nlint = false  # config\n\n# [verify.go.types] not defined\n\n[verify.go.test]\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in\n%s", want, got)
		}
	}
	cfg, err := parse(t, got)
	if err != nil {
		t.Fatalf("%v\n%s", err, got)
	}
	back, err := Resolve(cfg, presetsFor(t), goOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"lint", "types"} {
		a, z := eff.Stacks["go"][kind], back.Stacks["go"][kind]
		if !reflect.DeepEqual(a.Lane, z.Lane) || a.Defined != z.Defined {
			t.Errorf("%s: %+v, loaded back %+v", kind, a, z)
		}
	}
}

// What a lane needs is part of the lane, so show prints it and it loads back.
func TestWriteShowPrintsWhatALaneNeeds(t *testing.T) {
	cppOnly := detect.Facts{Stacks: []string{"cpp"}, Areas: map[string][]string{"cpp": {"."}}}
	eff := effFor(t, "", cppOnly)
	var b strings.Builder
	WriteShow(&b, eff, []string{"types"}, env(`C:\repo`))
	got := b.String()
	if !strings.Contains(got, "[verify.cpp.types]\ncommands = [\"cmake --build build --parallel\"]  # preset\nneeds = [\"build/CMakeCache.txt\"]  # preset\n") {
		t.Fatalf("%s", got)
	}
	cfg, err := parse(t, got)
	if err != nil {
		t.Fatalf("%v\n%s", err, got)
	}
	if needs := cfg.Stacks["cpp"]["types"].Lane.Needs; len(needs) != 1 || needs[0] != "build/CMakeCache.txt" {
		t.Fatalf("%q", needs)
	}
}

func TestWriteShowPrintsSkipWhenOnly(t *testing.T) {
	var out strings.Builder
	eff := effFor(t, "[verify.go.test]\nskip_when_only = [\"docs/**\"]\n", goOnly)
	WriteShow(&out, eff, []string{"test"}, env(`C:\repo`))
	if !strings.Contains(out.String(), "skip_when_only = [\"docs/**\"]  # config\n") {
		t.Fatalf("%s", out.String())
	}
}

func TestWriteShowPrintsLock(t *testing.T) {
	var out strings.Builder
	WriteShow(&out, effFor(t, "[verify.go.test]\nlock = true\n", goOnly), []string{"test"}, env(`C:\repo`))
	if !strings.Contains(out.String(), "lock = true  # config\n") {
		t.Fatalf("%s", out.String())
	}
}
