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
	"github.com/xidus90/loomux/internal/verify/godotsuite"
)

// godotWorld replaces the run and the finder and hands back what they were
// asked.
func godotWorld(t *testing.T, res godotsuite.Result, runErr error) (*godotsuite.Options, *string) {
	t.Helper()
	got, configured := &godotsuite.Options{}, new(string)
	oldRun, oldFor := checkGodotRun, checkGodotFor
	t.Cleanup(func() { checkGodotRun, checkGodotFor = oldRun, oldFor })
	checkGodotRun = func(o godotsuite.Options) (godotsuite.Result, error) {
		*got = o
		return res, runErr
	}
	checkGodotFor = func(_, conf string, _ func(child.Spec) child.Result, _ func() time.Duration) func(string) (string, verify.State, string) {
		*configured = conf
		return func(string) (string, verify.State, string) { return "found-godot", "", "" }
	}
	return got, configured
}

func TestReservedNamesGodotSuite(t *testing.T) {
	if !verify.Reserved("godot-suite") {
		t.Fatal("godot-suite is not reserved")
	}
}

func TestCheckGodotSuiteRefusesABadCall(t *testing.T) {
	godotWorld(t, godotsuite.Result{}, nil)
	if code, _, _ := run("check", "godot-suite", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
	if code, _, errOut := run("check", "godot-suite", "extra"); code != 2 || !strings.Contains(errOut, `unexpected argument "extra"`) {
		t.Fatalf("code %d err %q", code, errOut)
	}
}

func TestCheckGodotSuitePrintsFindingsAndNotes(t *testing.T) {
	godotWorld(t, godotsuite.Result{Findings: []string{"a finding"}, Notes: []string{"a note"}}, nil)
	code, out, errOut := run("check", "godot-suite", "--godot", "g", "--root", t.TempDir())
	if code != 1 || out != "a finding\n" || errOut != "loomux check godot-suite: note: a note\n" {
		t.Fatalf("code %d out %q err %q", code, out, errOut)
	}
}

func TestCheckGodotSuitePassesWithoutFindings(t *testing.T) {
	godotWorld(t, godotsuite.Result{}, nil)
	if code, out, errOut := run("check", "godot-suite", "--godot", "g", "--root", t.TempDir()); code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d out %q err %q", code, out, errOut)
	}
}

func TestCheckGodotSuiteKeepsFindingsNextToAnError(t *testing.T) {
	godotWorld(t, godotsuite.Result{Findings: []string{"stuck"}}, errors.New("boom"))
	code, out, errOut := run("check", "godot-suite", "--godot", "g", "--root", t.TempDir())
	if code != 1 || out != "stuck\n" || errOut != "loomux check godot-suite: boom\n" {
		t.Fatalf("code %d out %q err %q", code, out, errOut)
	}
}

func TestCheckGodotSuiteBuildsTheOptions(t *testing.T) {
	got, _ := godotWorld(t, godotsuite.Result{}, nil)
	checkout := t.TempDir()
	writeFile(t, filepath.Join(checkout, ".loomux", "config.toml"), "")
	root := filepath.Join(checkout, "godot")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if code, _, errOut := run("check", "godot-suite", "--godot", "g", "--root", root, "--coverage", "cov"); code != 0 {
		t.Fatal(errOut)
	}
	wd, _ := os.Getwd()
	if got.Root != root || got.CheckoutRoot != checkout || got.Area != "godot" || got.Godot != "g" ||
		got.Coverage != filepath.Join(wd, "cov") || got.RestoreOnly {
		t.Fatalf("got %+v", got)
	}
	if got.RunID == "" || strings.ContainsAny(got.RunID, `/\`) || got.CPUs < 1 || got.Start == nil || got.Now == nil {
		t.Fatalf("got %+v", got)
	}
}

func TestCheckGodotSuiteNamesTheRootAreaDot(t *testing.T) {
	got, _ := godotWorld(t, godotsuite.Result{}, nil)
	checkout := t.TempDir()
	writeFile(t, filepath.Join(checkout, ".loomux", "config.toml"), "")
	if code, _, errOut := run("check", "godot-suite", "--godot", "g", "--root", checkout); code != 0 || got.Area != "." {
		t.Fatalf("code %d area %q err %q", code, got.Area, errOut)
	}
}

func TestCheckGodotSuiteFindsTheBinaryFromTheConfig(t *testing.T) {
	got, configured := godotWorld(t, godotsuite.Result{}, nil)
	checkout := t.TempDir()
	writeFile(t, filepath.Join(checkout, ".loomux", "config.toml"), "[verify.gdscript]\ngodot = \"bin/godot\"\n")
	if code, _, errOut := run("check", "godot-suite", "--root", checkout); code != 0 {
		t.Fatal(errOut)
	}
	if *configured != "bin/godot" || got.Godot != "found-godot" {
		t.Fatalf("configured %q, godot %q", *configured, got.Godot)
	}
}

func TestCheckGodotSuiteReportsAnEngineThatDoesNotFit(t *testing.T) {
	godotWorld(t, godotsuite.Result{}, nil)
	checkGodotFor = func(string, string, func(child.Spec) child.Result, func() time.Duration) func(string) (string, verify.State, string) {
		return func(string) (string, verify.State, string) { return "", verify.StateUnready, "Godot 4.1 found" }
	}
	code, out, _ := run("check", "godot-suite", "--root", t.TempDir())
	if code != 1 || out != "Godot 4.1 found\n" {
		t.Fatalf("code %d out %q", code, out)
	}
}

// The restore is the way out of an instrumentation a killed run left, so an
// engine that cannot be found must not hold it up.
func TestCheckGodotSuiteRestoresWithoutAnEngine(t *testing.T) {
	got, _ := godotWorld(t, godotsuite.Result{}, nil)
	checkGodotFor = func(string, string, func(child.Spec) child.Result, func() time.Duration) func(string) (string, verify.State, string) {
		return func(string) (string, verify.State, string) { return "", verify.StateUnready, "no Godot" }
	}
	if code, _, errOut := run("check", "godot-suite", "--root", t.TempDir(), "--restore"); code != 0 {
		t.Fatal(errOut)
	}
	if !got.RestoreOnly || got.Godot != "" {
		t.Fatalf("got %+v", got)
	}
}

func TestCheckGodotSuiteRefusesAnUnreadableConfig(t *testing.T) {
	godotWorld(t, godotsuite.Result{}, nil)
	checkout := t.TempDir()
	writeFile(t, filepath.Join(checkout, ".loomux", "config.toml"), "[verify\n")
	code, _, errOut := run("check", "godot-suite", "--root", checkout)
	if code != 1 || !strings.HasPrefix(errOut, "loomux check godot-suite: ") {
		t.Fatalf("code %d err %q", code, errOut)
	}
}
