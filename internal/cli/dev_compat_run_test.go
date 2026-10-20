package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/compat"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/godotsuite"
)

// stubCompatRun stands in for compat.Run and keeps the options it was given.
func stubCompatRun(t *testing.T, rep compat.Report, err error) *compat.RunOptions {
	t.Helper()
	got := &compat.RunOptions{}
	old := compatRun
	t.Cleanup(func() { compatRun = old })
	compatRun = func(o compat.RunOptions) (compat.Report, error) { *got = o; return rep, err }
	return got
}

func firstKey() string {
	list, _ := compat.Load()
	return list[0].Key()
}

func TestDevCompatRunRefusesABadCall(t *testing.T) {
	stubCompatRun(t, compat.Report{}, nil)
	k := firstKey()
	for _, args := range [][]string{
		{"dev", "compat", "run"},
		{"dev", "compat", "run", "--work", "w"},
		{"dev", "compat", "run", "nope", "--work", "w"},
		{"dev", "compat", "run", k},
		{"dev", "compat", "run", k, "--work", "w", "extra"},
		{"dev", "compat", "run", k, "--bogus"},
	} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%v: code %d", args, code)
		}
	}
}

func TestDevCompatRunReportsAndExits(t *testing.T) {
	k := firstKey()
	jsonPath := filepath.Join(t.TempDir(), "report.json")
	got := stubCompatRun(t, compat.Report{Entry: k, Green: true, Lines: []string{}}, nil)
	code, out, errOut := run("dev", "compat", "run", k, "--work", "w", "--addons", "a", "--json", jsonPath)
	if code != 0 || out != "### "+k+": green\n" {
		t.Fatalf("green: code %d, out %q, err %q", code, out, errOut)
	}
	if got.Key != k || got.Work != "w" || got.Addons != "a" || got.Check == nil || got.Rescan == nil {
		t.Fatalf("options %+v", got)
	}
	var rep compat.Report
	data, _ := os.ReadFile(jsonPath)
	if err := json.Unmarshal(data, &rep); err != nil || !rep.Green || rep.Entry != k {
		t.Fatalf("json %s: %v", data, err)
	}

	stubCompatRun(t, compat.Report{Entry: k, Lines: []string{"lint/go: failed, want ok"}}, nil)
	if code, out, _ := run("dev", "compat", "run", k, "--work", "w"); code != 1 || !strings.Contains(out, "- lint/go: failed, want ok") {
		t.Fatalf("red: code %d, out %q", code, out)
	}
	if code, _, errOut := run("dev", "compat", "run", k, "--work", "w", "--json", t.TempDir()); code != 1 || errOut == "" {
		t.Fatalf("unwritable json: code %d, err %q", code, errOut)
	}
	stubCompatRun(t, compat.Report{}, errors.New("addon GUT: missing"))
	if code, _, errOut := run("dev", "compat", "run", k, "--work", "w"); code != 1 || errOut != "loomux dev compat run: addon GUT: missing\n" {
		t.Fatalf("error: code %d, err %q", code, errOut)
	}
}

func TestDevCompatRunChecksInProcess(t *testing.T) {
	got := stubCompatRun(t, compat.Report{}, nil)
	run("dev", "compat", "run", firstKey(), "--work", "w")
	stubCheck(t, green)
	root := goWorld(t)
	results, err := got.Check(root, []string{"lint", "test"})
	if err != nil {
		t.Fatal(err)
	}
	var lanes []string
	for _, r := range results {
		lanes = append(lanes, r.Lane+"="+r.State)
	}
	if !slices.Equal(lanes, []string{"lint/go=ok", "test/go=ok"}) {
		t.Fatalf("lanes %v", lanes)
	}

	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[verify\n")
	if _, err := got.Check(root, []string{"lint"}); err == nil {
		t.Error("a config that does not read was checked")
	}
	os.Remove(filepath.Join(root, ".loomux", "config.toml"))
	old := checkPlan
	t.Cleanup(func() { checkPlan = old })
	checkPlan = func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) {
		return nil, errors.New("no plan")
	}
	if _, err := got.Check(root, []string{"lint"}); err == nil || err.Error() != "no plan" {
		t.Errorf("err %v", err)
	}
}

func TestDevCompatRunRescansWithTheEngineItFinds(t *testing.T) {
	got := stubCompatRun(t, compat.Report{}, nil)
	run("dev", "compat", "run", firstKey(), "--work", "w")
	opts := rescanWorld(t, nil, nil)
	godotWorld(t, godotsuite.Result{}, nil)
	root := t.TempDir()
	if err := got.Rescan(root); err != nil || opts.Godot != "found-godot" || opts.Root != root {
		t.Fatalf("err %v, opts %+v", err, opts)
	}
	// The work copy is no git checkout, so its status is always empty.
	if status, err := opts.Status(root); status != "" || err != nil {
		t.Fatalf("status %q, err %v", status, err)
	}
	rescanWorld(t, nil, errors.New("the engine failed"))
	if err := got.Rescan(root); err == nil || err.Error() != "the engine failed" {
		t.Fatalf("err %v", err)
	}
	checkGodotFor = func(string, string, func(child.Spec) child.Result, func() time.Duration) func(string) (string, verify.State, string) {
		return func(string) (string, verify.State, string) { return "", verify.StateUnready, "Godot 4.1 found" }
	}
	if err := got.Rescan(root); err == nil || err.Error() != "Godot 4.1 found" {
		t.Fatalf("err %v", err)
	}
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[verify\n")
	if err := got.Rescan(root); err == nil {
		t.Fatal("a config that does not read found an engine")
	}
}

func TestDevCompatRunDescribesTheEntry(t *testing.T) {
	stubCompatRun(t, compat.Report{}, errors.New("describe must not run the entry"))
	code, out, errOut := run("dev", "compat", "run", "gdscript/godot-3.6/godot3-gut", "--describe")
	if code != 0 || !strings.HasPrefix(out, `{"stack":"gdscript","runtime":"godot","version":"3.6","addons":[{"name":"GUT"`) || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if code, _, _ := run("dev", "compat", "run", "nope", "--describe"); code != 2 {
		t.Errorf("unknown entry: code %d", code)
	}
}
