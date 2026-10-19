package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/gutsuite"
)

// gutWorld replaces the run and the finder; found is what the finder says.
func gutWorld(t *testing.T, res gutsuite.Result, runErr error, pre verify.State, note string) *gutsuite.Options {
	t.Helper()
	got := &gutsuite.Options{}
	oldRun, oldFor := checkGutRun, checkGodotFor
	t.Cleanup(func() { checkGutRun, checkGodotFor = oldRun, oldFor })
	checkGutRun = func(o gutsuite.Options) (gutsuite.Result, error) {
		*got = o
		return res, runErr
	}
	checkGodotFor = func(string, string, func(child.Spec) child.Result, func() time.Duration) func(string) (string, verify.State, string) {
		return func(string) (string, verify.State, string) { return "found-godot", pre, note }
	}
	return got
}

func TestReservedNamesGutSuite(t *testing.T) {
	if !verify.Reserved("gut-suite") {
		t.Fatal("gut-suite is not reserved")
	}
}

func TestCheckGutSuiteRefusesABadCall(t *testing.T) {
	gutWorld(t, gutsuite.Result{}, nil, "", "")
	if code, _, _ := run("check", "gut-suite", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
	if code, _, errOut := run("check", "gut-suite", "extra"); code != 2 || !strings.Contains(errOut, `unexpected argument "extra"`) {
		t.Fatalf("code %d err %q", code, errOut)
	}
}

func TestCheckGutSuiteJudgesByTheFindings(t *testing.T) {
	gutWorld(t, gutsuite.Result{}, nil, "", "")
	if code, out, errOut := run("check", "gut-suite", "--godot", "g", "--root", t.TempDir()); code != 0 || out != "" || errOut != "" {
		t.Fatalf("green: %d %q %q", code, out, errOut)
	}
	gutWorld(t, gutsuite.Result{Findings: []string{"test/test_a.gd test_x: no"}, Notes: []string{"a note"}}, nil, "", "")
	if code, out, errOut := run("check", "gut-suite", "--godot", "g", "--root", t.TempDir()); code != 1 || out != "test/test_a.gd test_x: no\n" || errOut != "loomux check gut-suite: note: a note\n" {
		t.Fatalf("red: %d %q %q", code, out, errOut)
	}
	gutWorld(t, gutsuite.Result{}, errors.New("boom"), "", "")
	if code, _, errOut := run("check", "gut-suite", "--godot", "g", "--root", t.TempDir()); code != 1 || errOut != "loomux check gut-suite: boom\n" {
		t.Fatalf("error: %d %q", code, errOut)
	}
}

// Without --godot the engine is found like {godot}, held to project.godot: a
// binary of the wrong line is a finding, not a run.
func TestCheckGutSuiteFindsTheEngineForTheProject(t *testing.T) {
	checkout := t.TempDir()
	writeFile(t, filepath.Join(checkout, ".loomux", "config.toml"), "")
	root := filepath.Join(checkout, "game")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	got := gutWorld(t, gutsuite.Result{}, nil, "", "")
	if code, _, errOut := run("check", "gut-suite", "--root", root); code != 0 || got.Godot != "found-godot" || got.Root != root || got.Checkout != checkout || got.RunID == "" {
		t.Fatalf("%d %q %+v", code, errOut, *got)
	}
	gutWorld(t, gutsuite.Result{}, nil, verify.StateUnready, "Godot 4.7.2 at x, but project.godot wants 3")
	if code, out, _ := run("check", "gut-suite", "--root", root); code != 1 || out != "Godot 4.7.2 at x, but project.godot wants 3\n" {
		t.Fatalf("%d %q", code, out)
	}
	writeFile(t, filepath.Join(checkout, ".loomux", "config.toml"), "[verify]\nbogus = 1\n")
	if code, _, errOut := run("check", "gut-suite", "--root", root); code != 1 || !strings.Contains(errOut, "loomux check gut-suite: ") {
		t.Fatalf("%d %q", code, errOut)
	}
}
