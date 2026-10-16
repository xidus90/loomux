package cli

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/godotsuite"
)

func rescanWorld(t *testing.T, notes []string, err error) *godotsuite.RescanOptions {
	t.Helper()
	got := &godotsuite.RescanOptions{}
	old := godotRescan
	t.Cleanup(func() { godotRescan = old })
	godotRescan = func(o godotsuite.RescanOptions) ([]string, error) {
		*got = o
		return notes, err
	}
	return got
}

func TestGodotNeedsASubcommand(t *testing.T) {
	for _, args := range [][]string{{"godot"}, {"godot", "nope"}} {
		if code, _, errOut := run(args...); code != 2 || !strings.Contains(errOut, "name a subcommand") {
			t.Fatalf("%v: code %d err %q", args, code, errOut)
		}
	}
}

func TestGodotRescanRefusesABadCall(t *testing.T) {
	rescanWorld(t, nil, nil)
	if code, _, _ := run("godot", "rescan", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
	if code, _, errOut := run("godot", "rescan", "extra"); code != 2 || !strings.Contains(errOut, `unexpected argument "extra"`) {
		t.Fatalf("code %d err %q", code, errOut)
	}
}

func TestGodotRescanPrintsNotesOnStderr(t *testing.T) {
	got := rescanWorld(t, []string{"restored project.godot, which the engine rewrote"}, nil)
	checkout := t.TempDir()
	writeFile(t, filepath.Join(checkout, ".loomux", "config.toml"), "")
	root := filepath.Join(checkout, "godot")
	code, out, errOut := run("godot", "rescan", "--godot", "g", "--root", root)
	if code != 0 || out != "" || errOut != "loomux godot rescan: note: restored project.godot, which the engine rewrote\n" {
		t.Fatalf("code %d out %q err %q", code, out, errOut)
	}
	if got.Root != root || got.Godot != "g" || got.Start == nil || got.Status == nil {
		t.Fatalf("got %+v", got)
	}
}

func TestGodotRescanReportsAnErrorWithItsNotes(t *testing.T) {
	rescanWorld(t, []string{"n"}, errors.New("boom"))
	code, _, errOut := run("godot", "rescan", "--godot", "g", "--root", t.TempDir())
	if code != 1 || errOut != "loomux godot rescan: note: n\nloomux godot rescan: boom\n" {
		t.Fatalf("code %d err %q", code, errOut)
	}
}

func TestGodotRescanRefusesAnEngineThatDoesNotFit(t *testing.T) {
	rescanWorld(t, nil, nil)
	godotWorld(t, godotsuite.Result{}, nil)
	checkGodotFor = func(string, string, func(child.Spec) child.Result, func() time.Duration) func(string) (string, verify.State, string) {
		return func(string) (string, verify.State, string) { return "", verify.StateUnready, "Godot 4.1 found" }
	}
	code, _, errOut := run("godot", "rescan", "--root", t.TempDir())
	if code != 1 || errOut != "loomux godot rescan: Godot 4.1 found\n" {
		t.Fatalf("code %d err %q", code, errOut)
	}
}

func TestGodotRescanFindsTheBinaryFromTheConfig(t *testing.T) {
	got := rescanWorld(t, nil, nil)
	godotWorld(t, godotsuite.Result{}, nil)
	if code, _, errOut := run("godot", "rescan", "--root", t.TempDir()); code != 0 || got.Godot != "found-godot" {
		t.Fatalf("code %d godot %q err %q", code, got.Godot, errOut)
	}
}

func TestGodotStatusAsksGitFromTheCheckout(t *testing.T) {
	var spec child.Spec
	old := checkStart
	t.Cleanup(func() { checkStart = old })
	checkStart = func(s child.Spec) child.Result { spec = s; return child.Result{Stdout: " M a\n"} }
	out, err := godotStatus("/co")("/co/godot")
	want := []string{"git", "status", "--porcelain", "--", "/co/godot"}
	if err != nil || out != " M a\n" || spec.Dir != "/co" || !slices.Equal(spec.Argv, want) {
		t.Fatalf("out %q err %v spec %+v", out, err, spec)
	}
	checkStart = func(child.Spec) child.Result { return child.Result{Err: errors.New("no git")} }
	if _, err := godotStatus("/co")("/co"); err == nil || err.Error() != "no git" {
		t.Fatalf("err %v", err)
	}
	checkStart = func(child.Spec) child.Result { return child.Result{Code: 128, Stderr: "not a repo"} }
	if _, err := godotStatus("/co")("/co"); err == nil || !strings.Contains(err.Error(), "not a repo") {
		t.Fatalf("err %v", err)
	}
}
