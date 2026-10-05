package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/hooks"
	"github.com/xidus90/loomux/internal/verify"
	"github.com/xidus90/loomux/internal/verify/commit"
)

func TestCheckNeedsARequest(t *testing.T) {
	want := "loomux check: name a profile or kinds (lint,types,test,coverage,graph), or one of: commit-msg, gofmt, gocover, graph-fresh, blast-audit\n"
	if code, _, errOut := run("check"); code != 2 || errOut != want {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func stubCheck(t *testing.T, answer func(child.Spec) child.Result) *[]string {
	t.Helper()
	var mu sync.Mutex
	seen := []string{}
	oldS, oldL, oldE := checkStart, checkLook, checkExecutable
	checkStart = func(s child.Spec) child.Result {
		mu.Lock()
		seen = append(seen, strings.Join(s.Argv, " "))
		mu.Unlock()
		return answer(s)
	}
	checkLook = func(s string) (string, error) { return s, nil }
	checkExecutable = func() (string, error) { return "loomux", nil }
	t.Cleanup(func() { checkStart, checkLook, checkExecutable = oldS, oldL, oldE })
	return &seen
}

func goWorld(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module m\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "a_test.go"), []byte("package m\n"), 0o644)
	return dir
}

// answering stands in for a tool that ends with code and, like go test,
// writes the profile it was told to: the coverage lane reads it afterwards.
func answering(code int) func(child.Spec) child.Result {
	return func(s child.Spec) child.Result {
		for _, a := range s.Argv {
			if p, ok := strings.CutPrefix(a, "-coverprofile="); ok {
				os.WriteFile(p, []byte("mode: set\n"), 0o644)
			}
		}
		return child.Result{Code: code}
	}
}

var green = answering(0)

func coverDirOf(root string) string { return filepath.Join(root, ".loomux", "state", "cover") }

func TestCheckRunsTheGoPresetsInTheirOrder(t *testing.T) {
	root := goWorld(t)
	seen := stubCheck(t, green)
	var so, se bytes.Buffer
	code := Run([]string{"check", "all", "--root", root}, nil, &so, &se)
	if code != 0 {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
	for _, want := range []string{"lint/go: ok [preset]", "types/go: not-applicable [preset] no command", "test/go: ok [preset]", "coverage/go: ok [preset]"} {
		if !strings.Contains(so.String(), want) {
			t.Errorf("missing %q in %q", want, so.String())
		}
	}
	if !strings.Contains(strings.Join(*seen, "\n"), "go test ./... -count=1 -covermode=set -coverprofile=") {
		t.Errorf("test must measure: %v", *seen)
	}
	if _, err := os.Stat(coverDirOf(root)); err != nil {
		t.Errorf("the cover directory must exist: %v", err)
	}
}

// A string override of test drops the preset's measuring; coverage then
// measures by itself instead of reading a profile nobody wrote.
func TestCheckCoverageMeasuresItselfBesideATestOverride(t *testing.T) {
	root := goWorld(t)
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify.go]\ntest = \"go test ./...\"\n"), 0o644)
	seen := stubCheck(t, green)
	code, out, errOut := run("check", "test,coverage", "--root", root)
	if code != 0 || !strings.Contains(out, "coverage/go: ok") {
		t.Fatalf("code %d, out %q, err %q, ran %v", code, out, errOut, *seen)
	}
}

// The same for Python, where every lane carries COVERAGE_FILE: a plain
// pytest writes no data, so coverage runs its own measure before reporting.
func TestCheckPythonCoverageMeasuresItselfBesideATestOverride(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.pytest.ini_options]\n"), 0o644)
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify.python]\ntest = \"uv run pytest\"\n"), 0o644)
	seen := stubCheck(t, func(s child.Spec) child.Result {
		// Like coverage.py, only a measuring run writes where COVERAGE_FILE points.
		if slices.Contains(s.Argv, "coverage") && slices.Contains(s.Argv, "-m") {
			for _, e := range s.Env {
				if p, ok := strings.CutPrefix(e, "COVERAGE_FILE="); ok {
					os.WriteFile(p, []byte("data"), 0o644)
				}
			}
		}
		return child.Result{}
	})
	code, out, errOut := run("check", "test,coverage", "--root", root)
	if code != 0 || !strings.Contains(out, "coverage/python: ok") || !slices.Contains(*seen, "uv run --with coverage --with pytest coverage run -m pytest -q --tb=short --no-header") {
		t.Fatalf("code %d, out %q, err %q, ran %v", code, out, errOut, *seen)
	}
}

func TestCheckFailsAKindWithNothingToRun(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644)
	stubCheck(t, green)
	var so, se bytes.Buffer
	if code := Run([]string{"check", "test", "--root", root}, nil, &so, &se); code != 1 || !strings.Contains(so.String(), "nothing to check for `test`\n") {
		t.Fatalf("%d %q", code, so.String())
	}
}

// The built-in precommit profile leaves out the kinds a module without tests
// has no lane for; the same profile set by the project names them.
func TestCheckLeavesOutAKindOnlyTheBuiltInProfileNames(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module m\n"), 0o644)
	stubCheck(t, green)
	var so, se bytes.Buffer
	if code := Run([]string{"check", "precommit", "--root", root}, nil, &so, &se); code != 0 ||
		!strings.Contains(so.String(), "no lane for `test` here, left out\n") || strings.Contains(so.String(), "nothing to check") {
		t.Fatalf("built in: %d %q %q", code, so.String(), se.String())
	}
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify.profiles]\nprecommit = [\"lint\", \"test\"]\n"), 0o644)
	so.Reset()
	if code := Run([]string{"check", "precommit", "--root", root}, nil, &so, &se); code != 1 || !strings.Contains(so.String(), "nothing to check for `test`\n") {
		t.Fatalf("set by the project: %d %q", code, so.String())
	}
}

func TestCheckRefusesMalformedCalls(t *testing.T) {
	root := goWorld(t)
	stubCheck(t, green)
	for _, args := range [][]string{
		{"check", "lint", "test", "--root", root},
		{"check", "lint", "--root", root, "test"},
		{"check", "lint", "--bogus"},
		{"check", "-v", "lint"},
	} {
		code, out, errOut := run(args...)
		if code != 2 || out != "" || errOut == "" {
			t.Errorf("%v: code %d, out %q, err %q", args, code, out, errOut)
		}
	}
}

func TestCheckNamesTheRequestFirst(t *testing.T) {
	if _, _, errOut := run("check", "--root", "x", "all"); !strings.Contains(errOut, "loomux check: name the profile or kinds before the flags, got \"--root\"") {
		t.Fatalf("%q", errOut)
	}
}

func TestCheckReportsLoadErrors(t *testing.T) {
	stubCheck(t, green)
	for _, c := range []struct{ config, request, want string }{
		{"[verify\n", "all", "config.toml"},
		{"", "style", `unknown check "style"`},
		{"[verify.go.test]\nafter = \"coverage\"\n", "all", "after forms a cycle"},
	} {
		root := goWorld(t)
		os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
		os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte(c.config), 0o644)
		code, out, errOut := run("check", c.request, "--root", root)
		if code != 1 || out != "" || !strings.HasPrefix(errOut, "loomux check: ") || !strings.Contains(errOut, c.want) {
			t.Errorf("%q: code %d, out %q, err %q", c.config, code, out, errOut)
		}
	}
}

func TestCheckReportsAPresetError(t *testing.T) {
	old := checkPresets
	checkPresets = func() (*verify.Presets, error) { return nil, errors.New("presets broke") }
	t.Cleanup(func() { checkPresets = old })
	if code, _, errOut := run("check", "all", "--root", goWorld(t)); code != 1 || errOut != "loomux check: presets broke\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckReportsAPlanError(t *testing.T) {
	stubCheck(t, green)
	old := checkPlan
	checkPlan = func(verify.Effective, verify.Request, verify.PlanEnv) ([]verify.Job, error) {
		return nil, errors.New("plan broke")
	}
	t.Cleanup(func() { checkPlan = old })
	if code, _, errOut := run("check", "all", "--root", goWorld(t)); code != 1 || errOut != "loomux check: plan broke\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckShowPrintsThePlanAndRunsNothing(t *testing.T) {
	root := goWorld(t)
	seen := stubCheck(t, green)
	code, out, errOut := run("check", "lint", "--root", root, "--show")
	if code != 0 || !strings.Contains(out, "[verify.go.lint]") || !strings.Contains(out, "tests found") || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if len(*seen) != 0 {
		t.Fatalf("show ran %v", *seen)
	}
	if _, err := os.Stat(filepath.Join(root, ".loomux")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("show must not touch the project: %v", err)
	}
}

func TestCheckVerboseShowsAGreenLanesOutput(t *testing.T) {
	root := goWorld(t)
	stubCheck(t, func(child.Spec) child.Result { return child.Result{Stdout: "all tidy\n"} })
	if _, out, _ := run("check", "lint", "--root", root); strings.Contains(out, "all tidy") {
		t.Fatalf("quiet check printed a green lane: %q", out)
	}
	if code, out, _ := run("check", "lint", "--root", root, "-v"); code != 0 || !strings.Contains(out, "all tidy\n") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestCheckFailsOnARedLaneAndStillCleansStaleFiles(t *testing.T) {
	root := goWorld(t)
	stubCheck(t, func(s child.Spec) child.Result {
		if slices.Contains(s.Argv, "gofmt") {
			return child.Result{Code: 1, Stdout: "x.go\n"}
		}
		return child.Result{}
	})
	os.MkdirAll(coverDirOf(root), 0o755)
	// A stale profile of an earlier run shows that the red run still ran
	// CleanCover, which would have removed a green run's own files as well.
	stale := filepath.Join(coverDirOf(root), "old.out")
	os.WriteFile(stale, nil, 0o644)
	past := time.Now().Add(-48 * time.Hour)
	os.Chtimes(stale, past, past)
	code, out, _ := run("check", "lint", "--root", root)
	if code != 1 || !strings.Contains(out, "lint/go: failed [preset]") || !strings.Contains(out, "x.go\n") {
		t.Fatalf("code %d, out %q", code, out)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the stale profile must go: %v", err)
	}
}

func TestCheckKeepsTheOwnProfilesOfARedRunOnly(t *testing.T) {
	for _, c := range []struct {
		code int
		kept []string
	}{{0, nil}, {1, []string{"go-root.out"}}} {
		root := goWorld(t)
		stubCheck(t, answering(c.code))
		if code, _, _ := run("check", "test,coverage", "--root", root); code != c.code {
			t.Fatalf("tool exit %d, check exit %d", c.code, code)
		}
		entries, _ := os.ReadDir(coverDirOf(root))
		kept := []string{}
		for _, e := range entries {
			// The run ID in front differs per run; the stack and area do not.
			kept = append(kept, e.Name()[strings.Index(e.Name(), "go-"):])
		}
		if !slices.Equal(kept, append([]string{}, c.kept...)) {
			t.Errorf("tool exit %d kept %q", c.code, kept)
		}
	}
}

func TestCheckFindsTheRootAboveTheWorkingDirectory(t *testing.T) {
	root := goWorld(t)
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), nil, 0o644)
	sub := filepath.Join(root, "sub")
	os.MkdirAll(sub, 0o755)
	t.Chdir(sub)
	stubCheck(t, green)
	if code, out, errOut := run("check", "lint"); code != 0 || !strings.Contains(out, "lint/go: ok") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if _, err := os.Stat(coverDirOf(root)); err != nil {
		t.Fatalf("the found root holds the state: %v", err)
	}
}

func TestCheckWithoutAConfigChecksTheWorkingDirectory(t *testing.T) {
	root := goWorld(t)
	t.Chdir(root)
	stubCheck(t, green)
	if code, out, errOut := run("check", "lint"); code != 0 || !strings.Contains(out, "lint/go: ok") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if _, err := os.Stat(coverDirOf(root)); err != nil {
		t.Fatalf("the working directory holds the state: %v", err)
	}
}

func TestCheckNamesItsOwnBinaryForLoomux(t *testing.T) {
	root := goWorld(t)
	seen := stubCheck(t, green)
	checkExecutable = func() (string, error) { return `C:\bin\loomux.exe`, nil }
	run("check", "lint", "--root", root)
	checkExecutable = func() (string, error) { return "", errors.New("no path") }
	run("check", "lint", "--root", root)
	gofmt := slices.DeleteFunc(slices.Clone(*seen), func(s string) bool { return !strings.Contains(s, "check gofmt") })
	want := []string{`C:\bin\loomux.exe check gofmt .`, "loomux check gofmt ."}
	if !slices.Equal(gofmt, want) {
		t.Fatalf("%q", *seen)
	}
}

func TestCheckFailsWhenItCannotMakeTheCoverDirectory(t *testing.T) {
	root := goWorld(t)
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "state"), nil, 0o644)
	seen := stubCheck(t, green)
	code, _, errOut := run("check", "lint", "--root", root)
	if code != 1 || !strings.HasPrefix(errOut, "loomux check: ") || len(*seen) != 0 {
		t.Fatalf("code %d, err %q, ran %v", code, errOut, *seen)
	}
}

func TestCheckWarnsWhenItCannotCleanTheCoverDirectory(t *testing.T) {
	root := goWorld(t)
	// A stale entry that is a directory with something in it cannot be
	// removed, on any platform.
	stale := filepath.Join(coverDirOf(root), "old")
	os.MkdirAll(stale, 0o755)
	os.WriteFile(filepath.Join(stale, "x"), nil, 0o644)
	past := time.Now().Add(-48 * time.Hour)
	os.Chtimes(stale, past, past)
	stubCheck(t, green)
	code, _, errOut := run("check", "lint", "--root", root)
	if code != 0 || !strings.HasPrefix(errOut, "loomux check: cleaning coverage files: ") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckCommitMsgAcceptsEnglish(t *testing.T) {
	file := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(file, []byte("feat: add the first loomux command\n"), 0o644)
	if code, _, errOut := run("check", "commit-msg", file); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestCheckCommitMsgRefusesGerman(t *testing.T) {
	file := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(file, []byte("Füge den ersten Befehl hinzu und prüfe die Änderung\n"), 0o644)
	if code, _, _ := run("check", "commit-msg", file); code != 1 {
		t.Fatalf("code %d", code)
	}
}

func TestCheckGofmtNamesUnformattedFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\nfunc  F(){}\n"), 0o644)
	code, out, _ := run("check", "gofmt", dir)
	if code != 1 || !strings.Contains(out, "x.go") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestCheckCommitMsgWithoutAFileIsAUsageError(t *testing.T) {
	code, _, errOut := run("check", "commit-msg")
	if code != 2 || !strings.Contains(errOut, "exactly one message file required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckCommitMsgOnAnUnreadableFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent")
	code, _, errOut := run("check", "commit-msg", missing)
	if code != 1 || !strings.Contains(errOut, "loomux check commit-msg:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGofmtOnACleanDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\n\nfunc F() {}\n"), 0o644)
	code, out, errOut := run("check", "gofmt", dir)
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

// Without paths the check reads the working directory.
func TestCheckGofmtWithoutPathsChecksTheWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.go"), []byte("package x\nfunc  F(){}\n"), 0o644)
	t.Chdir(dir)
	code, out, _ := run("check", "gofmt")
	if code != 1 || !strings.Contains(out, "x.go") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestCheckGofmtReportsAPathItCannotRead(t *testing.T) {
	code, out, errOut := run("check", "gofmt", "invalid\x00path")
	if code != 1 || out != "" || !strings.Contains(errOut, "loomux check gofmt:") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestCheckCommitMsgRefusesAMissingHeader(t *testing.T) {
	file := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(file, []byte("Add the first loomux command\n"), 0o644)
	if code, _, errOut := run("check", "commit-msg", file); code != 1 || !strings.Contains(errOut, "<type>[(<scope>)][!]: <description>") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func stubCoverFunc(t *testing.T, out string, err error) *[2]string {
	t.Helper()
	var seen [2]string
	old := coverFunc
	coverFunc = func(dir, profile string) ([]byte, error) {
		seen = [2]string{dir, profile}
		return []byte(out), err
	}
	t.Cleanup(func() { coverFunc = old })
	return &seen
}

// gocoverModule writes a module with one function on line 3 of a.go.
func gocoverModule(t *testing.T, source string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const halfCovered = "example.com/m/a.go:3:\tF\t50.0%\ntotal:\t(statements)\t50.0%\n"

func TestCheckGocoverPerFunctionAndFloor(t *testing.T) {
	dir := gocoverModule(t, "package m\n\nfunc F() {}\n")
	t.Chdir(dir)
	stubCoverFunc(t, halfCovered, nil)
	var so, se bytes.Buffer
	if code := Run([]string{"check", "gocover", "--profile", "p.out"}, nil, &so, &se); code != 1 || !strings.Contains(so.String(), "not covered: a.go:3 F 50.0%") {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
	so.Reset()
	if code := Run([]string{"check", "gocover", "--profile", "p.out", "--floor", "40"}, nil, &so, &se); code != 0 || so.String() != "coverage 50.0%\n" {
		t.Fatalf("%d %q %q", code, so.String(), se.String())
	}
	se.Reset()
	if code := Run([]string{"check", "gocover", "--profile", "p.out", "--floor", "60"}, nil, &so, &se); code != 1 || !strings.Contains(se.String(), "coverage 50.0% is below the floor of 60%") {
		t.Fatalf("%d %q", code, se.String())
	}
}

func TestCheckGocoverReadsSourcesAndTheProfileInDir(t *testing.T) {
	dir := gocoverModule(t, "package m\n\n//coverage:exempt needs a full disk\nfunc F() {}\n")
	seen := stubCoverFunc(t, "example.com/m/a.go:4:\tF\t50.0%\n", nil)
	if code, out, errOut := run("check", "gocover", "--profile", "p.out", "--dir", dir); code != 0 {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if *seen != [2]string{dir, "p.out"} {
		t.Fatalf("cover tool called with %q", *seen)
	}
}

func TestCheckGocoverNeedsAProfile(t *testing.T) {
	if code, _, errOut := run("check", "gocover"); code != 2 || !strings.Contains(errOut, "loomux check gocover: --profile is required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGocoverRefusesAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("check", "gocover", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestCheckGocoverNeedsAGoMod(t *testing.T) {
	stubCoverFunc(t, halfCovered, nil)
	code, _, errOut := run("check", "gocover", "--profile", "p.out", "--dir", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "loomux check gocover: ") || !strings.Contains(errOut, "go.mod") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGocoverNeedsAModuleLine(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go 1.25.0\n"), 0o644)
	stubCoverFunc(t, halfCovered, nil)
	code, _, errOut := run("check", "gocover", "--profile", "p.out", "--dir", dir)
	if code != 1 || !strings.Contains(errOut, "loomux check gocover: go.mod has no module line") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGocoverFailsWhenTheCoverToolFails(t *testing.T) {
	dir := gocoverModule(t, "package m\n")
	stubCoverFunc(t, "", errors.New("no profile"))
	code, _, errOut := run("check", "gocover", "--profile", "p.out", "--dir", dir)
	if code != 1 || !strings.Contains(errOut, "loomux check gocover: no profile") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGocoverRefusesAProfileWithoutFunctions(t *testing.T) {
	dir := gocoverModule(t, "package m\n")
	stubCoverFunc(t, "total:\t(statements)\t0.0%\n", nil)
	code, _, errOut := run("check", "gocover", "--profile", "p.out", "--dir", dir)
	if code != 1 || !strings.Contains(errOut, "loomux check gocover: no functions in p.out") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGocoverFailsOnUnparsableOutput(t *testing.T) {
	dir := gocoverModule(t, "package m\n")
	stubCoverFunc(t, "garbage\n", nil)
	code, _, errOut := run("check", "gocover", "--profile", "p.out", "--dir", dir)
	if code != 1 || !strings.Contains(errOut, "unexpected cover line") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGocoverFloorNeedsATotalLine(t *testing.T) {
	dir := gocoverModule(t, "package m\n")
	stubCoverFunc(t, "example.com/m/a.go:3:\tF\t100.0%\n", nil)
	code, _, errOut := run("check", "gocover", "--profile", "p.out", "--dir", dir, "--floor", "1")
	if code != 1 || !strings.Contains(errOut, "loomux check gocover: no total line") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevCovergateIsGone(t *testing.T) {
	if code, _, _ := run("dev", "covergate"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestRunCoverFuncReadsAProfileRelativeToDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "c.out"), []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runCoverFunc(dir, "c.out"); err != nil {
		t.Fatal(err)
	}
}

func TestRunCoverFuncReportsTheToolsStderr(t *testing.T) {
	_, err := runCoverFunc(t.TempDir(), "missing.out")
	if err == nil || !strings.Contains(err.Error(), "missing.out") {
		t.Fatalf("err %v", err)
	}
}

func TestCheckCommitMsgCalibrate(t *testing.T) {
	oldRunner := commit.GitRunner
	t.Cleanup(func() { commit.GitRunner = oldRunner })
	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		return child.Result{
			Code:   0,
			Stdout: "feat: add feature\x00fix: bug fix\x00",
		}, nil
	}

	code, out, errOut := run("check", "commit-msg", "--calibrate", "10")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if !strings.Contains(out, "threshold") {
		t.Fatalf("expected calibrate output, got %q", out)
	}

	code, out, errOut = run("check", "commit-msg", "--calibrate", "5", "--language", "de")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if !strings.Contains(out, "threshold") {
		t.Fatalf("expected calibrate output, got %q", out)
	}

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(dir, ".loomux", "config.toml"), []byte("[commit]\nlanguage = \"de\"\n"), 0o644)
	code, out, errOut = run("check", "commit-msg", "--root", dir, "--calibrate", "5")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if !strings.Contains(out, "threshold") {
		t.Fatalf("expected calibrate output, got %q", out)
	}

	emptyDir := t.TempDir()
	code, out, errOut = run("check", "commit-msg", "--root", emptyDir, "--calibrate", "5")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if !strings.Contains(out, "threshold") {
		t.Fatalf("expected calibrate output, got %q", out)
	}

	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		return child.Result{Code: 1, Stderr: "git error"}, nil
	}
	code, _, errOut = run("check", "commit-msg", "--calibrate", "5")
	if code != 1 || !strings.Contains(errOut, "git error") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckCommitMsgCalibrateErrors(t *testing.T) {
	if code, _, _ := run("check", "commit-msg", "--unknown-flag"); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}

	if code, _, errOut := run("check", "commit-msg", "--calibrate", "5", "file.txt"); code != 2 || !strings.Contains(errOut, "cannot pass both a file and --calibrate") {
		t.Fatalf("expected 2, got %d (%s)", code, errOut)
	}

	if code, _, errOut := run("check", "commit-msg", "--calibrate", "0"); code != 2 || !strings.Contains(errOut, "--calibrate needs a count of at least 1") {
		t.Fatalf("expected 2, got %d (%s)", code, errOut)
	}

	if code, _, errOut := run("check", "commit-msg", "--calibrate", "5", "--language", "fr"); code != 2 || !strings.Contains(errOut, "--language must be one of") {
		t.Fatalf("expected 2, got %d (%s)", code, errOut)
	}

	badDir := t.TempDir()
	os.MkdirAll(filepath.Join(badDir, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(badDir, ".loomux", "config.toml"), []byte("bogus toml [[["), 0o644)
	if code, _, errOut := run("check", "commit-msg", "--root", badDir, "--calibrate", "5"); code != 1 || !strings.Contains(errOut, "loomux check commit-msg:") {
		t.Fatalf("expected 1, got %d (%s)", code, errOut)
	}
}

func TestCheckCommitMsgFileErrors(t *testing.T) {
	file := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(file, []byte("feat: hello\n"), 0o644)

	if code, _, errOut := run("check", "commit-msg", "--language", "en", file); code != 2 || !strings.Contains(errOut, "--language cannot be used when checking a file") {
		t.Fatalf("expected 2, got %d (%s)", code, errOut)
	}

	if code, _, errOut := run("check", "commit-msg", file, file); code != 2 || !strings.Contains(errOut, "exactly one message file required") {
		t.Fatalf("expected 2, got %d (%s)", code, errOut)
	}

	badDir := t.TempDir()
	os.MkdirAll(filepath.Join(badDir, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(badDir, ".loomux", "config.toml"), []byte("bogus toml [[["), 0o644)
	if code, _, errOut := run("check", "commit-msg", "--root", badDir, file); code != 1 || !strings.Contains(errOut, "loomux check commit-msg:") {
		t.Fatalf("expected 1, got %d (%s)", code, errOut)
	}
}

// wikiWorld is a project that declares a wiki bundle and holds one page in
// it, the way a loomux project with docs/wiki does.
func wikiWorld(t *testing.T, pages map[string]string) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".loomux"), 0o755)
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[area]\nscope = \"project/x\"\n[layout]\nwiki = \"docs/wiki\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Join(root, "docs", "wiki"), 0o755)
	for name, body := range pages {
		if err := os.WriteFile(filepath.Join(root, "docs", "wiki", name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const wikiPage = "---\ntitle: Sample Concept\ntype: concept\ndescription: A valid OKF test document\n---\n\n# Sample Concept\nThis is a test concept.\n"

// A check for lint runs the wiki gate as a lane of the chain, in this
// process: no tool starts for it, and its verdict is the run's.
func TestCheckRunsTheWikiGateAsALintLane(t *testing.T) {
	root := wikiWorld(t, map[string]string{"page.md": wikiPage})
	stubCheck(t, green)
	code, out, errOut := run("check", "lint", "--root", root)
	if code != 0 || !strings.Contains(out, "lint/wiki: ok [in-process]") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestCheckFailsOnABrokenWikiBundle(t *testing.T) {
	root := wikiWorld(t, map[string]string{"page.md": wikiPage, "broken.md": "no frontmatter at all\n"})
	stubCheck(t, green)
	code, out, _ := run("check", "lint", "--root", root)
	if code != 1 || !strings.Contains(out, "lint/wiki: failed") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

// graphFresh runs `check graph-fresh` with args.
func graphFresh(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := checkCommand(append([]string{"graph-fresh"}, args...), nil, &out, &errOut)
	return code, out.String(), errOut.String()
}

// builtRepo is sample() with a graph built for it.
func builtRepo(t *testing.T) string {
	t.Helper()
	root := repo(t, sample())
	if _, _, err := query.Build(root, func(string) {}); err != nil {
		t.Fatal(err)
	}
	return root
}

// moveFile edits a source file so the graph no longer matches the tree.
func moveFile(t *testing.T, root string) {
	t.Helper()
	body := "package lib\n\n// Run does the thing.\nfunc Run() { _ = 1 }\n"
	if err := os.WriteFile(filepath.Join(root, "lib", "lib.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckGraphFreshIsGreenOnAFreshGraph(t *testing.T) {
	code, out, errOut := graphFresh("--root", builtRepo(t))
	if code != 0 || out != "graph is fresh\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestCheckGraphFreshRebuildsOnDriftAndStaysGreen(t *testing.T) {
	root := builtRepo(t)
	moveFile(t, root)
	code, out, errOut := graphFresh("--root", root)
	if code != 0 || out != "graph rebuilt\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if !strings.Contains(errOut, "loomux check graph-fresh: 1 files moved, rebuilding the graph") {
		t.Fatalf("stderr %q does not say why it rebuilt", errOut)
	}
}

func TestCheckGraphFreshIsRedOnAHeldLock(t *testing.T) {
	root := builtRepo(t)
	moveFile(t, root)
	lock := ask.LockPath(root)
	// This process: a lock whose holder is gone would be broken at once.
	if err := os.WriteFile(lock, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := graphFresh("--root", root, "--wait", "0s")
	if code != 1 || !strings.Contains(errOut, lock) {
		t.Fatalf("code %d, err %q; want 1 and the lock path", code, errOut)
	}
}

func TestCheckGraphFreshIsRedWithoutAGraph(t *testing.T) {
	code, _, errOut := graphFresh("--root", repo(t, sample()))
	if code != 1 || !strings.Contains(errOut, query.ErrNoGraph.Error()) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckGraphFreshRejectsABadWait(t *testing.T) {
	if code, _, _ := graphFresh("--wait", "x"); code != 2 {
		t.Fatalf("code %d, want 2", code)
	}
}

func TestCheckGraphFreshFailsWhenTheRootCannotBeFound(t *testing.T) {
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { getwd = saved })
	code, _, errOut := graphFresh()
	if code != 1 || !strings.HasPrefix(errOut, "loomux check graph-fresh: ") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// blastAudit runs `check blast-audit` with args.
func blastAudit(args ...string) (int, string, string) {
	var out, errOut bytes.Buffer
	code := checkCommand(append([]string{"blast-audit"}, args...), nil, &out, &errOut)
	return code, out.String(), errOut.String()
}

// stagedRun is builtRepo committed, with Run's body changed and staged:
// main and TestRun call Run, and the test did not change with it.
func stagedRun(t *testing.T) string {
	t.Helper()
	root := committedRun(t, false)
	gitIn(t, root, "add", "lib")
	return root
}

func TestCheckBlastAuditIsRedOnAFinding(t *testing.T) {
	code, out, errOut := blastAudit("--cached", "--threshold", "2", "--root", stagedRun(t))
	want := "blast audit: index against HEAD, threshold 2\nlib/lib.go [stale]: Run in-degree 2\n"
	if code != 1 || out != want {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestCheckBlastAuditIsGreenBelowTheThreshold(t *testing.T) {
	code, out, errOut := blastAudit("--cached", "--threshold", "3", "--root", stagedRun(t))
	if code != 0 || out != "no area at or above 3 callers lacks a changed test (1 areas)\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestCheckBlastAuditSkipsTestCallers(t *testing.T) {
	code, out, errOut := blastAudit("--cached", "--threshold", "2", "--skip-test-callers", "--root", stagedRun(t))
	if code != 0 || !strings.HasPrefix(out, "no area at or above 2 callers") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestCheckBlastAuditRefusesBaseAndCached(t *testing.T) {
	code, _, errOut := blastAudit("--base", "X", "--cached", "--root", t.TempDir())
	if code != 1 || !strings.Contains(errOut, query.ErrBaseAndCached.Error()) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckBlastAuditRejectsABadThreshold(t *testing.T) {
	if code, _, _ := blastAudit("--threshold", "x"); code != 2 {
		t.Fatalf("code %d, want 2", code)
	}
}

func TestCheckBlastAuditFailsWhenTheRootCannotBeFound(t *testing.T) {
	saved := getwd
	getwd = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { getwd = saved })
	code, _, errOut := blastAudit()
	if code != 1 || !strings.HasPrefix(errOut, "loomux check blast-audit: ") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// committedRun is builtRepo committed, with Run's body changed in the working
// tree and nothing staged: what a turn end sees.
func committedRun(t *testing.T, withTest bool) string {
	t.Helper()
	root := builtRepo(t)
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
		{"add", "go.mod", "main.go", "lib"}, {"commit", "-qm", "init"},
	} {
		gitIn(t, root, args...)
	}
	moveFile(t, root)
	if withTest {
		body := "package lib\n\nimport \"testing\"\n\nfunc TestNew(t *testing.T) { Run() }\n"
		if err := os.WriteFile(filepath.Join(root, "lib", "new_test.go"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// stopEnvMu guards GIT_INDEX_FILE: lanes run in goroutines, and the stub
// sets it for the in-process blast audit.
var stopEnvMu sync.Mutex

// stopStub answers every lane green and runs the blast audit in the process,
// with the lane's environment and a threshold the sample reaches.
func stopStub(t *testing.T, root string) {
	t.Helper()
	stubCheck(t, func(s child.Spec) child.Result {
		if len(s.Argv) < 3 {
			return green(s)
		}
		if s.Argv[2] == "graph-fresh" {
			code, out, errOut := graphFresh("--root", root)
			return child.Result{Code: code, Stdout: out, Stderr: errOut}
		}
		if s.Argv[2] != "blast-audit" {
			return green(s)
		}
		stopEnvMu.Lock()
		defer stopEnvMu.Unlock()
		old, had := os.LookupEnv("GIT_INDEX_FILE")
		for _, kv := range s.Env {
			if v, ok := strings.CutPrefix(kv, "GIT_INDEX_FILE="); ok {
				os.Setenv("GIT_INDEX_FILE", v)
			}
		}
		defer func() {
			if had {
				os.Setenv("GIT_INDEX_FILE", old)
			} else {
				os.Unsetenv("GIT_INDEX_FILE")
			}
		}()
		args := append(append([]string{}, s.Argv[3:]...), "--threshold", "2", "--root", root)
		code, out, errOut := blastAudit(args...)
		return child.Result{Code: code, Stdout: out, Stderr: errOut}
	})
}

// noIndexCopy fails when a lane left a copy of the index in the git directory.
func noIndexCopy(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "index" && strings.Contains(e.Name(), "index") {
			t.Fatalf("left behind: %s", e.Name())
		}
	}
}

// check stop judges the working tree against HEAD as the stop gate does:
// an unstaged change with a new test beside it is green.
func TestCheckStopJudgesTheWorkingTreeLikeTheHook(t *testing.T) {
	root := committedRun(t, true)
	stopStub(t, root)
	code, out, errOut := run("check", "stop", "--root", root)
	if code != 0 || !strings.Contains(out, "graph/go: ok") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	noIndexCopy(t, root)
}

func TestCheckStopFindsAnUntestedChange(t *testing.T) {
	root := committedRun(t, false)
	stopStub(t, root)
	code, out, errOut := run("check", "stop", "-v", "--root", root)
	if code != 1 || !strings.Contains(out, "lib/lib.go [stale]") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	noIndexCopy(t, root)
}

// check precommit keeps the real index, where nothing is staged.
func TestCheckPrecommitKeepsTheRealIndex(t *testing.T) {
	root := committedRun(t, true)
	stopStub(t, root)
	code, out, errOut := run("check", "precommit", "--root", root)
	if code != 0 || !strings.Contains(out, "graph/go: not-applicable") || !strings.Contains(out, "nothing staged") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	noIndexCopy(t, root)
}

// A copy that cannot be made fails the check before any lane runs.
func TestCheckStopFailsWhenTheCopyCannotBeMade(t *testing.T) {
	root := committedRun(t, true)
	if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	seen := stubCheck(t, green)
	code, _, errOut := run("check", "stop", "--root", root)
	if code != 1 || !strings.HasPrefix(errOut, "loomux check: ") || len(*seen) != 0 {
		t.Fatalf("code %d, err %q, ran %q", code, errOut, *seen)
	}
}

// The table runs nothing and asks no probe, so it makes no copy of the index
// either: a broken index does not stop it.
func TestCheckStopShowMakesNoCopy(t *testing.T) {
	root := committedRun(t, true)
	if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	seen := stubCheck(t, green)
	code, out, errOut := run("check", "stop", "--show", "--root", root)
	if code != 0 || !strings.Contains(out, "graph") || len(*seen) != 0 {
		t.Fatalf("code %d, out %q, err %q, ran %q", code, out, errOut, *seen)
	}
	noIndexCopy(t, root)
}

// check stop and the stop hook give one verdict over one world: green passes
// both, an untested change fails the check and holds the hook.
func TestCheckStopGivesTheHooksVerdict(t *testing.T) {
	for name, c := range map[string]struct {
		withTest    bool
		check, hook int
	}{
		"green": {true, 0, hooks.ExitOK},
		"red":   {false, 1, hooks.ExitDenied},
	} {
		t.Run(name, func(t *testing.T) {
			root := committedRun(t, c.withTest)
			stopStub(t, root)
			code, out, errOut := run("check", "stop", "--root", root)
			if code != c.check {
				t.Fatalf("check stop: code %d, out %q, err %q", code, out, errOut)
			}
			var hookErr bytes.Buffer
			hook := hooks.RunStop(strings.NewReader(`{"session_id":"s1"}`), &hookErr, root, "claude", hooks.StopEnv{
				Start: checkStart, Look: checkLook, Loomux: "loomux", Budget: time.Minute, Now: time.Now,
			})
			if hook != c.hook {
				t.Fatalf("hook stop: code %d, err %q", hook, hookErr.String())
			}
			noIndexCopy(t, root)
		})
	}
}

// With a graph and a staged change the graph lane runs both its commands, in
// the order the preset gives them.
func TestCheckGraphRunsTheLaneWhereTheProbeAgrees(t *testing.T) {
	root := stagedRun(t)
	seen := stubCheck(t, green)
	code, out, errOut := run("check", "graph", "--root", root)
	if code != 0 || !strings.Contains(out, "graph/go: ok [preset]") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	want := []string{"loomux check graph-fresh", "loomux check blast-audit --cached --threshold 5"}
	if !slices.Equal(*seen, want) {
		t.Fatalf("ran %q, want %q", *seen, want)
	}
}

// Without a graph the lane stands aside, and a check of the kind alone is
// green: in CI and in a fresh clone there is nothing for it to read.
func TestCheckGraphWithoutAGraphIsNotApplicable(t *testing.T) {
	root := goWorld(t)
	seen := stubCheck(t, green)
	code, out, errOut := run("check", "graph", "--root", root)
	if code != 0 || !strings.Contains(out, "graph/go: not-applicable [preset] no graph at .loomux/state/graph/wiring.json") || len(*seen) != 0 {
		t.Fatalf("code %d, out %q, err %q, ran %q", code, out, errOut, *seen)
	}
}
