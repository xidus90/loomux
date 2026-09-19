package cli

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/dev/faketool"
)

// wantCases2a is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases2a = 53

// approved2a names every case whose replay differs from its recording, with
// the number of the deviation in docs/.superpowers/parity/stufe-2a.md that
// explains it. A case missing here must pass; a case listed here must fail.
var approved2a = map[string]string{
	// 1: the old chain crashed on CMake types; cmake --build is now cpp's types lane.
	"check/cpp-all":   "1: types ran cmake --build and passed where the old chain raised KeyError",
	"check/cpp-types": "1: types ran cmake --build and passed where the old chain raised KeyError",
	// 10: unavailable was red; a kind no stack defines is neutral now.
	"check/godot-no-coverage-all":      "10: GDScript has no types or coverage lane: neutral, no longer red",
	"check/godot-no-coverage-coverage": "10: GDScript has no coverage lane: not-applicable exits 0",
	"check/godot-no-coverage-types":    "10: GDScript has no types lane: not-applicable exits 0",
	"check/godot-unready-all":          "10: GDScript has no types or coverage lane: neutral, no longer red",
	"check/godot-unready-coverage":     "10: GDScript has no coverage lane: not-applicable exits 0",
	"check/godot-unready-types":        "10: GDScript has no types lane: not-applicable exits 0",
	// 16: the old chain had no Go preset; the Go lanes run now.
	"check/go-only-all":      "16: the Go lanes run and pass where the old chain found no preset",
	"check/go-only-coverage": "16: the Go coverage lane runs gocover and passes",
	"check/go-only-lint":     "16: the Go lint lane runs go vet and gofmt and passes",
	"check/go-only-test":     "16: the Go test lane runs and passes",
	"check/go-only-types":    "16: Go has no types lane: not-applicable exits 0",
}

// TestCases2a replays the recordings of `ultraloom check` against loomux
// check. The fixture the recording's fake tools answered from answers here
// too, at the seam every lane starts its processes through.
func TestCases2a(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "2a"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases2a {
		t.Fatalf("found %d cases, want %d", len(all), wantCases2a)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				useFakeTools(t, dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			reason, approved := approved2a[c.Verb+"/"+c.Name]
			switch {
			case approved && outcome.Passed:
				t.Fatalf("listed as a deviation (%s) but passes; remove it from approved2a and parity/stufe-2a.md", reason)
			case !approved && !outcome.Passed:
				t.Fatalf("%s\nstdout:\n%s", strings.Join(outcome.Mismatches, "\n"), outcome.ActualStdout)
			}
		})
	}
}

// fakeSelf is what {loomux} names in the replay.
const fakeSelf = "loomux"

// coverProfileToken in a write's path stands for the profile the command
// line names with -coverprofile=: the path carries the run's ID, which no
// fixture can know. Without that flag the tool writes no profile.
const coverProfileToken = "{{COVERPROFILE}}"

// useFakeTools points the seams of check at the world's fixture until the
// case ends. {loomux} runs in this process, in its own buffers and without a
// change of directory, since the lanes run in parallel: gocover reads the
// profile a fake `go test` wrote. Every other tool answers from the fixture,
// and a command line it has no answer for exits 127, so a gap is loud.
func useFakeTools(t *testing.T, dir string) {
	t.Helper()
	fixture, err := faketool.Load(filepath.Join(dir, faketool.FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{fakeSelf: true}
	for _, a := range fixture.Answers {
		known[strings.Fields(a.Prefix)[0]] = true
	}
	oldS, oldL, oldE := checkStart, checkLook, checkExecutable
	t.Cleanup(func() { checkStart, checkLook, checkExecutable = oldS, oldL, oldE })
	checkExecutable = func() (string, error) { return fakeSelf, nil }
	checkLook = func(name string) (string, error) {
		if known[name] {
			return name, nil
		}
		return "", exec.ErrNotFound
	}
	checkStart = func(spec child.Spec) child.Result {
		if spec.Argv[0] == fakeSelf {
			args := slices.Clone(spec.Argv[1:])
			// gocover reads go.mod and the sources from --dir; the other
			// built-in checks take paths relative to the world, which is
			// the working directory already.
			if len(args) >= 2 && args[0] == "check" && args[1] == "gocover" {
				args = append(args, "--dir", spec.Dir)
			}
			var out, errOut bytes.Buffer
			code := Run(args, strings.NewReader(""), &out, &errOut)
			return child.Result{Code: code, Stdout: out.String(), Stderr: errOut.String()}
		}
		answer, ok := fixture.Match(spec.Argv)
		if !ok {
			return child.Result{Code: 127, Stderr: "faketool: no answer\n"}
		}
		if err := fixture.Apply(withProfile(t, answer, spec), spec.Dir); err != nil {
			return child.Result{Code: 127, Stderr: "faketool: " + err.Error() + "\n"}
		}
		return child.Result{Code: answer.Exit, Stdout: answer.Stdout}
	}
}

// withProfile puts the profile the command line names in place of
// coverProfileToken, relative to the directory the answer is applied in.
func withProfile(t *testing.T, a faketool.Answer, spec child.Spec) faketool.Answer {
	var profile string
	for _, arg := range spec.Argv {
		if p, ok := strings.CutPrefix(arg, "-coverprofile="); ok {
			profile = p
		}
	}
	writes := []faketool.Write{}
	for _, w := range a.Writes {
		if w.Path == coverProfileToken {
			if profile == "" {
				continue
			}
			rel, err := filepath.Rel(spec.Dir, profile)
			if err != nil {
				t.Errorf("profile %s outside %s: %v", profile, spec.Dir, err)
				continue
			}
			w.Path = filepath.ToSlash(rel)
		}
		writes = append(writes, w)
	}
	a.Writes = writes
	return a
}
