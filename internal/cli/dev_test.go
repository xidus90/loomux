package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/dev/benchcorpus"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/benchreport"
	"github.com/xidus90/loomux/internal/dev/benchsearch"
	"github.com/xidus90/loomux/internal/dev/mutants"
)

func TestDevNeedsASubcommand(t *testing.T) {
	code, _, errOut := run("dev")
	if code != 2 || !strings.Contains(errOut, "loomux dev: subcommand required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRefusesAnUnknownSubcommand(t *testing.T) {
	code, _, errOut := run("dev", "frobnicate")
	if code != 2 || !strings.Contains(errOut, `loomux dev: unknown subcommand "frobnicate"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevSwapBinary(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "loomux.new.exe"), []byte("x"), 0o755)
	if code, _, errOut := run("dev", "swap-binary", "--dir", dir); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if code, _, _ := run("dev", "swap-binary", "--dir", dir); code != 1 {
		t.Fatalf("second swap without a new binary must fail, got %d", code)
	}
}

func TestDevSwapBinaryRefusesAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("dev", "swap-binary", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestDevRecordCaseNeedsItsFlags(t *testing.T) {
	if code, _, _ := run("dev", "record-case", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
	code, _, errOut := run("dev", "record-case", "--cmd", "ulguard {{WORLD}}")
	if code != 2 || !strings.Contains(errOut, "loomux dev record-case: --exe or --argv, --cmd, --world and --out are required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRecordCaseRecordsABinary(t *testing.T) {
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go binary on PATH")
	}
	out := filepath.Join(t.TempDir(), "demo", "version")
	code, _, errOut := run("dev", "record-case",
		"--exe", goExe, "--cmd", "go version", "--world", t.TempDir(),
		"--out", out, "--notes", "the go version", "--compare", "message")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(out, "cmd")); err != nil {
		t.Fatal(err)
	}
}

// The flag reaches the recorder: a world without a repository has no commit
// for git.after to name.
func TestDevRecordCasePassesGitAfterOn(t *testing.T) {
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go binary on PATH")
	}
	code, _, errOut := run("dev", "record-case", "--git-after",
		"--exe", goExe, "--cmd", "go version", "--world", t.TempDir(),
		"--out", filepath.Join(t.TempDir(), "demo", "version"))
	if code != 1 || !strings.Contains(errOut, "git world") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRecordCaseReportsAFailedRecording(t *testing.T) {
	code, _, errOut := run("dev", "record-case",
		"--exe", filepath.Join(t.TempDir(), "gone.exe"), "--cmd", "ulguard x",
		"--world", t.TempDir(), "--out", filepath.Join(t.TempDir(), "v", "n"))
	if code != 1 || !strings.Contains(errOut, "loomux dev record-case:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRecordCaseRefusesExeAndArgvTogether(t *testing.T) {
	code, _, errOut := run("dev", "record-case", "--exe", "brain.exe", "--argv", "uv run brain-mcp",
		"--cmd", "brain-mcp status", "--world", t.TempDir(), "--out", t.TempDir())
	if code != 2 || !strings.Contains(errOut, "loomux dev record-case: --exe and --argv exclude each other") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRecordCaseRefusesAnArgvItCannotSplit(t *testing.T) {
	code, _, errOut := run("dev", "record-case", "--argv", "'unclosed",
		"--cmd", "brain-mcp status", "--world", t.TempDir(), "--out", t.TempDir())
	if code != 2 || !strings.Contains(errOut, "loomux dev record-case: --argv: unclosed quote") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevRecordCaseRefusesAnEnvWithoutAValue(t *testing.T) {
	code, _, errOut := run("dev", "record-case", "--env", "NOVALUE")
	if code != 2 || !strings.Contains(errOut, `"NOVALUE" is not KEY=VALUE`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// The argv form with a real program: go stands in for uv, "env" for the
// leading arguments, GOWORK for what the command asks after.
func TestDevRecordCaseRecordsAProgramWithLeadingArguments(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("no go binary on PATH")
	}
	out := filepath.Join(t.TempDir(), "demo", "gowork")
	code, _, errOut := run("dev", "record-case",
		"--argv", "go env", "--env", "GOWORK=off", "--env", "LOOMUX_UNUSED={{WORLD}}",
		"--path-prepend", t.TempDir(), "--cmd", "old GOWORK", "--world", t.TempDir(),
		"--out", out, "--notes", "the go workspace setting")
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	if got, err := os.ReadFile(filepath.Join(out, "stdout")); err != nil || string(got) != "off\n" {
		t.Fatalf("%v %q", err, got)
	}
}

func TestDevImportCasesNeedsItsFlags(t *testing.T) {
	if code, _, _ := run("dev", "import-cases", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
	code, _, errOut := run("dev", "import-cases", "--map", "m.toml")
	if code != 2 || !strings.Contains(errOut, "loomux dev import-cases: --map, --from and --to are required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevImportCasesTranslatesACorpus(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	caseDir := filepath.Join(from, "guard", "one")
	if err := os.MkdirAll(filepath.Join(caseDir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"cmd": "ulguard --root {{WORLD}}\n", "exit": "2\n", "stdout": ""} {
		if err := os.WriteFile(filepath.Join(caseDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mapFile := filepath.Join(t.TempDir(), "map.toml")
	if err := os.WriteFile(mapFile, []byte("[[command]]\nfrom = \"ulguard --root {{WORLD}}\"\nto   = \"loomux hook pre-tool-use --host claude --root {{WORLD}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, errOut := run("dev", "import-cases", "--map", mapFile, "--from", from, "--to", to)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	got, err := os.ReadFile(filepath.Join(to, "guard", "one", "cmd"))
	if err != nil || string(got) != "loomux hook pre-tool-use --host claude --root {{WORLD}}\n" {
		t.Fatalf("%v %q", err, got)
	}
}

func TestDevImportCasesReportsABrokenMapAndAFailedImport(t *testing.T) {
	broken := filepath.Join(t.TempDir(), "map.toml")
	if err := os.WriteFile(broken, []byte("[[command\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("dev", "import-cases", "--map", broken, "--from", t.TempDir(), "--to", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "loomux dev import-cases:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}

	good := filepath.Join(t.TempDir(), "map.toml")
	if err := os.WriteFile(good, []byte("[[command]]\nfrom = \"ulguard\"\nto = \"loomux\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut = run("dev", "import-cases", "--map", good, "--from", filepath.Join(t.TempDir(), "gone"), "--to", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "loomux dev import-cases:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// benchCases writes a one-case file and returns its path.
func benchCases(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cases.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const oneCase = `[{"name":"probe","mode":"single","steps":[{"argv":["x"]}]}]`

func TestDevBenchHooksKeepsTheOldBehaviour(t *testing.T) {
	benchExec = func(benchhooks.Case, benchhooks.Step) (int, error) { return 0, nil }
	defer func() { benchExec = benchhooks.Exec }()
	code, out, errOut := run("dev", "bench", "hooks", benchCases(t, oneCase), "-n", "2")
	if code != 0 || !strings.Contains(out, "| probe |") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestDevBenchHooksTakesTheFlagBeforeTheFile(t *testing.T) {
	benchExec = func(benchhooks.Case, benchhooks.Step) (int, error) { return 0, nil }
	defer func() { benchExec = benchhooks.Exec }()
	code, out, _ := run("dev", "bench", "hooks", "-n", "1", benchCases(t, oneCase))
	if code != 0 || !strings.Contains(out, "| probe |") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestDevBenchHooksNeedsACaseFile(t *testing.T) {
	code, _, errOut := run("dev", "bench", "hooks")
	if code != 2 || !strings.Contains(errOut, "loomux dev bench hooks: a case file is required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("dev", "bench", "hooks", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestDevBenchHooksReportsAnUnreadableCaseFile(t *testing.T) {
	code, _, errOut := run("dev", "bench", "hooks", filepath.Join(t.TempDir(), "gone.json"))
	if code != 1 || !strings.Contains(errOut, "loomux dev bench hooks:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksReportsABrokenMeasurement(t *testing.T) {
	benchExec = func(benchhooks.Case, benchhooks.Step) (int, error) {
		return 0, errors.New("no such binary")
	}
	defer func() { benchExec = benchhooks.Exec }()
	code, _, errOut := run("dev", "bench", "hooks", benchCases(t, oneCase))
	if code != 1 || !strings.Contains(errOut, "no such binary") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesAnUnknownFlagBehindTheFile(t *testing.T) {
	code, _, _ := run("dev", "bench", "hooks", benchCases(t, oneCase), "--bogus")
	if code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestDevBenchHooksRefusesLessThanOneWarmRun(t *testing.T) {
	code, _, errOut := run("dev", "bench", "hooks", "-n", "0", benchCases(t, oneCase))
	if code != 2 || !strings.Contains(errOut, "-n must be at least 1, got 0") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesANegativeWarmRunBehindTheFile(t *testing.T) {
	code, _, errOut := run("dev", "bench", "hooks", benchCases(t, oneCase), "-n", "-3")
	if code != 2 || !strings.Contains(errOut, "-n must be at least 1, got -3") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesAnExtraArgumentBehindTheFile(t *testing.T) {
	code, _, errOut := run("dev", "bench", "hooks", benchCases(t, oneCase), "leftover")
	if code != 2 || !strings.Contains(errOut, `unexpected argument "leftover" after the case file`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksReportsBrokenJSON(t *testing.T) {
	code, _, errOut := run("dev", "bench", "hooks", benchCases(t, "{"))
	if code != 1 || !strings.Contains(errOut, "loomux dev bench hooks:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchWithoutSubcommandPrintsTheGroupsHelp(t *testing.T) {
	code, out, errOut := run("dev", "bench")
	if code != 2 || out != "" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	for _, want := range []string{"usage: loomux dev bench <hooks|repos|search>", "hooks", "repos",
		"  search  measure the rank of search hits and the chain's latency"} {
		if !strings.Contains(errOut, want) {
			t.Fatalf("help lacks %q:\n%s", want, errOut)
		}
	}
}

func TestTheOldBenchNamesAreGone(t *testing.T) {
	code, _, errOut := run("dev", "bench-hooks", "x")
	if code != 2 || !strings.Contains(errOut, `loomux dev: unknown subcommand "bench-hooks"`) {
		t.Fatalf("bench-hooks: code %d, stderr %q", code, errOut)
	}
	code, _, errOut = run("dev", "bench", "--dir", ".")
	if code != 2 || !strings.Contains(errOut, `loomux dev bench: unknown subcommand "--dir"`) ||
		!strings.Contains(errOut, "usage: loomux dev bench <hooks|repos|search>") {
		t.Fatalf("bench --dir: code %d, stderr %q", code, errOut)
	}
}

// pinBenchClock fixes the clock a bench run takes its stamp from and returns
// the stamp.
func pinBenchClock(t *testing.T) string {
	t.Helper()
	now := time.Date(2026, 9, 26, 14, 7, 0, 0, time.UTC)
	orig := benchClock
	benchClock = func() time.Time { return now }
	t.Cleanup(func() { benchClock = orig })
	return benchreport.Stamp(now)
}

// countingBenchExec replaces the process start with one that succeeds and
// counts its calls.
func countingBenchExec(t *testing.T) *int {
	t.Helper()
	calls := 0
	benchExec = func(benchhooks.Case, benchhooks.Step) (int, error) {
		calls++
		return 0, nil
	}
	t.Cleanup(func() { benchExec = benchhooks.Exec })
	return &calls
}

func TestDevBenchHooksWritesBothFilesWithOut(t *testing.T) {
	stamp := pinBenchClock(t)
	countingBenchExec(t)
	dir := t.TempDir()
	code, out, errOut := run("dev", "bench", "hooks", benchCases(t, oneCase), "-n", "1", "--out", dir)
	if code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	base := filepath.Join(dir, "bench-"+stamp+"-hooks")
	text, err := os.ReadFile(base + ".md")
	if err != nil {
		t.Fatal(err)
	}
	head := "# loomux dev bench hooks — " + stamp + "\n\n- system: "
	if !strings.HasPrefix(string(text), head) || !strings.Contains(string(text), "- loomux: "+Version) ||
		!strings.Contains(string(text), "| probe |") {
		t.Fatalf("markdown:\n%s", text)
	}
	// The markdown stays on stdout as well, as the file has it.
	if out != string(text) {
		t.Fatalf("stdout %q, file %q", out, text)
	}
	data, err := os.ReadFile(base + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var report benchreport.Report
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Command != "hooks" || report.Stamp != stamp || report.Schema != benchreport.Schema ||
		report.Environment.Loomux != Version || len(report.Timings) != 1 || report.Timings[0].Name != "probe" {
		t.Fatalf("report %+v", report)
	}
}

// A run that could not save its report must not measure first.
func TestDevBenchHooksRefusesATakenTargetBeforeMeasuring(t *testing.T) {
	stamp := pinBenchClock(t)
	calls := countingBenchExec(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bench-"+stamp+"-hooks.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("dev", "bench", "hooks", benchCases(t, oneCase), "--out", dir)
	if code != 1 || !strings.Contains(errOut, "loomux dev bench hooks:") || !strings.Contains(errOut, "already exists") || *calls != 0 {
		t.Fatalf("code %d, calls %d, err %q", code, *calls, errOut)
	}
	code, _, errOut = run("dev", "bench", "hooks", benchCases(t, oneCase), "--out", filepath.Join(dir, "gone"))
	if code != 1 || !strings.Contains(errOut, "no directory at") || *calls != 0 {
		t.Fatalf("code %d, calls %d, err %q", code, *calls, errOut)
	}
}

func TestDevBenchHooksReportsAFailedWrite(t *testing.T) {
	countingBenchExec(t)
	orig := benchWriteBoth
	benchWriteBoth = func(string, string, []byte, []byte) error { return errors.New("disk full") }
	t.Cleanup(func() { benchWriteBoth = orig })
	code, _, errOut := run("dev", "bench", "hooks", benchCases(t, oneCase), "--out", t.TempDir())
	if code != 1 || !strings.Contains(errOut, "loomux dev bench hooks: disk full") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// mutantsWorld lays out a root with one package p, points the dev mutants
// seams at it and at test, and restores both when the test ends.
func mutantsWorld(t *testing.T, test mutants.TestFunc) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "p"), 0o755); err != nil {
		t.Fatal(err)
	}
	source := "package p\n\nfunc Sign(a int) int {\n\tif a > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"
	if err := os.WriteFile(filepath.Join(root, "p", "p.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	mutantsRoot = func() (string, error) { return root, nil }
	mutantsTest = func(ctx context.Context, dir string) mutants.TestFunc {
		if dir != root {
			t.Errorf("go test runs in %q, want %q", dir, root)
		}
		// Only a cancellable context has a Done channel: handing GoTest the
		// background one would leave every go test running after an interrupt.
		if ctx.Done() == nil {
			t.Error("go test runs under a context no interrupt can end")
		}
		return test
	}
	t.Cleanup(func() {
		mutantsRoot = os.Getwd
		mutantsTest = mutants.GoTest
	})
}

// killEveryMutant is a suite that is green alone and red under any overlay.
func killEveryMutant(_, overlay string) (mutants.Outcome, error) {
	if overlay == "" {
		return mutants.Passed, nil
	}
	return mutants.Failed, nil
}

func TestDevMutantsRunsARound(t *testing.T) {
	mutantsWorld(t, killEveryMutant)
	code, out, errOut := run("dev", "mutants", "p", "--workers", "2")
	if code != 0 || !strings.Contains(out, "[1/4] killed    (a1) p.go:4  if a > 0 {  ->  if true {\n") ||
		!strings.Contains(out, "\n4 mutants over p, oracle go\n0 do not compile and are no mutants\n0 survived:\n") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestDevMutantsTakesFlagsBetweenPackages(t *testing.T) {
	mutantsWorld(t, killEveryMutant)
	code, out, errOut := run("dev", "mutants", "--family", "a3", "p", "--only", "p.go", "p")
	if code != 0 || strings.Count(out, "\n1 mutants over p, oracle go\n") != 2 {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestDevMutantsRefusesBadArguments(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"dev", "mutants"}, "loomux dev mutants: at least one package is required\n"},
		{[]string{"dev", "mutants", "p", "--family", "a5"}, "loomux dev mutants: --family must be one of a1, a2, a3, a4, got \"a5\"\n"},
		{[]string{"dev", "mutants", "--workers", "0", "p"}, "loomux dev mutants: --workers must be at least 1, got 0\n"},
	} {
		if code, _, errOut := run(c.args...); code != 2 || errOut != c.want {
			t.Errorf("%v: code %d, err %q", c.args, code, errOut)
		}
	}
	for _, args := range [][]string{{"dev", "mutants", "--bogus"}, {"dev", "mutants", "p", "--bogus"}} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%v: code %d", args, code)
		}
	}
}

func TestDevMutantsReportsAMissingWorkingDirectory(t *testing.T) {
	mutantsWorld(t, killEveryMutant)
	mutantsRoot = func() (string, error) { return "", errors.New("getwd: gone") }
	code, _, errOut := run("dev", "mutants", "p")
	if code != 1 || errOut != "loomux dev mutants: getwd: gone\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevMutantsRefusesARedSuiteAndAnEmptyPackage(t *testing.T) {
	mutantsWorld(t, func(string, string) (mutants.Outcome, error) { return mutants.Failed, nil })
	code, out, errOut := run("dev", "mutants", "p")
	if code != 2 || out != "" || errOut != "loomux dev mutants: p: the suite is not green before the round\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	code, _, errOut = run("dev", "mutants", "p", "--only", "nothing")
	if code != 2 || errOut != "loomux dev mutants: p: no source files\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevMutantsReportsABrokenRun(t *testing.T) {
	mutantsWorld(t, func(string, string) (mutants.Outcome, error) { return 0, errors.New("go: not found") })
	code, _, errOut := run("dev", "mutants", "p")
	if code != 1 || errOut != "loomux dev mutants: go: not found\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// Ctrl+C cannot be sent to a test process on Windows; the seam hands the
// round a context the test cancels while the first mutant runs.
func TestDevMutantsCleansUpAnInterruptedRound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	t.Setenv("TMPDIR", tmp)
	var interrupt context.CancelFunc
	stops := 0
	mutantsNotify = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		interrupt = cancel
		return ctx, func() { stops++; cancel() }
	}
	t.Cleanup(func() { mutantsNotify = signal.NotifyContext })
	mutantsWorld(t, func(_, overlay string) (mutants.Outcome, error) {
		if overlay != "" {
			if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 1 {
				t.Errorf("while the mutant runs: %v, %v", entries, err)
			}
			interrupt()
		}
		return mutants.Passed, nil
	})
	code, out, errOut := run("dev", "mutants", "p", "--workers", "1")
	if code != 1 || out != "" || errOut != "loomux dev mutants: context canceled\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 0 {
		t.Fatalf("left behind: %v, %v", entries, err)
	}
	// Twice: once where the round saw the interrupt, once in the defer. Only
	// the defer would mean a second Ctrl+C stays swallowed through the drain.
	if stops != 2 {
		t.Fatalf("the signal registration was stopped %d times, want 2", stops)
	}
}

func TestDevFakeOllamaNeedsAFixture(t *testing.T) {
	if code, _, _ := run("dev", "fake-ollama", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
	code, _, errOut := run("dev", "fake-ollama")
	if code != 2 || errOut != "loomux dev fake-ollama: --fixture is required\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// interruptedFakeOllama hands the command a context that has already ended,
// as a Ctrl+C would; the fake then stops as soon as it listens.
func interruptedFakeOllama(t *testing.T) {
	t.Helper()
	fakeOllamaNotify = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		cancel()
		return ctx, cancel
	}
	t.Cleanup(func() { fakeOllamaNotify = signal.NotifyContext })
}

func ollamaFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ollama-fixture.json")
	if err := os.WriteFile(path, []byte(`{"response":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDevFakeOllamaServesUntilInterrupted(t *testing.T) {
	interruptedFakeOllama(t)
	logPath := filepath.Join(t.TempDir(), "ollama.log")
	if err := os.WriteFile(logPath, []byte("earlier\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := run("dev", "fake-ollama", "--fixture", ollamaFixture(t), "--addr", "127.0.0.1:0", "--log", logPath)
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if data, err := os.ReadFile(logPath); err != nil || string(data) != "earlier\n" {
		t.Fatalf("the log was not appended to: %q, %v", data, err)
	}
	if code, _, errOut := run("dev", "fake-ollama", "--fixture", ollamaFixture(t), "--addr", "127.0.0.1:0"); code != 0 {
		t.Fatalf("without --log: code %d, err %q", code, errOut)
	}
}

func TestDevFakeOllamaReportsWhatKeepsItFromServing(t *testing.T) {
	interruptedFakeOllama(t)
	dir := t.TempDir()
	for name, args := range map[string][]string{
		"fixture": {"--fixture", filepath.Join(dir, "missing.json")},
		"log":     {"--fixture", ollamaFixture(t), "--log", filepath.Join(dir, "no", "such", "dir", "ollama.log")},
		"addr":    {"--fixture", ollamaFixture(t), "--addr", "127.0.0.1:-1"},
	} {
		code, _, errOut := run(append([]string{"dev", "fake-ollama"}, args...)...)
		if code != 1 || !strings.HasPrefix(errOut, "loomux dev fake-ollama: ") {
			t.Errorf("%s: code %d, err %q", name, code, errOut)
		}
	}
}

func TestUntilInterruptedStartsNoRunAfterTheInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started, stopped := false, false
	test := untilInterrupted(ctx, func() { stopped = true }, func(string, string) (mutants.Outcome, error) {
		started = true
		return mutants.Passed, nil
	})
	if _, err := test("p", ""); !errors.Is(err, context.Canceled) || started || !stopped {
		t.Fatalf("err %v, started %t, stopped %t", err, started, stopped)
	}
}

// The signal registration has to end with the first interrupt: while it
// stands, a second Ctrl+C is swallowed and the user cannot force the drain to
// end.
func TestUntilInterruptedStopsTheSignalRegistrationOfARunningSuite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stopped := false
	test := untilInterrupted(ctx, func() { stopped = true }, func(string, string) (mutants.Outcome, error) {
		cancel()
		return mutants.Passed, nil
	})
	if _, err := test("p", ""); !errors.Is(err, context.Canceled) || !stopped {
		t.Fatalf("err %v, stopped %t", err, stopped)
	}
}

func TestDevBenchRepos(t *testing.T) {
	origRepoRun := benchRepoRun
	origCorpusRun := benchCorpusRun
	origReadFile := benchReadFile
	origWriteBoth := benchWriteBoth
	origSaveReport := benchSaveReport
	origStorageOps := benchStorageOps
	defer func() {
		benchRepoRun = origRepoRun
		benchCorpusRun = origCorpusRun
		benchReadFile = origReadFile
		benchWriteBoth = origWriteBoth
		benchSaveReport = origSaveReport
		benchStorageOps = origStorageOps
	}()

	mockAudit := &benchcorpus.RepoAudit{
		Dir:          "/mock/repo",
		CoverageRate: 100.0,
		Timings:      []benchreport.Timing{{Name: benchcorpus.TotalTiming, MedianMS: 42}},
	}

	benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
		return mockAudit, nil
	}

	benchCorpusRun = func(matrixData []byte, opts benchcorpus.Options, cloner benchcorpus.Cloner, benchRepo func(string, benchcorpus.Options) (*benchcorpus.RepoAudit, error)) (*benchcorpus.BenchmarkReport, error) {
		audit, err := benchRepo("/cached/repo", opts)
		if err != nil {
			return nil, err
		}
		return &benchcorpus.BenchmarkReport{
			Timestamp: "2026-09-18T12:00:00Z",
			Mode:      "corpus",
			WarmRuns:  opts.WarmRuns,
			Repos:     []*benchcorpus.RepoAudit{audit},
			Skipped:   []benchcorpus.SkippedRepo{{RepoURL: "https://github.com/x/y", Reason: "fatal: unable to checkout working tree"}},
		}, nil
	}

	t.Run("Flag validation", func(t *testing.T) {
		if code, _, _ := run("dev", "bench", "repos", "--bogus"); code != 2 {
			t.Errorf("expected 2 for unknown flag, got %d", code)
		}
		if code, _, _ := run("dev", "bench", "repos", "--json-out", "x.json"); code != 2 {
			t.Errorf("expected 2 for the dropped --json-out, got %d", code)
		}
		if code, _, errOut := run("dev", "bench", "repos", "--warm", "0"); code != 2 || !strings.Contains(errOut, "loomux dev bench repos: --warm") {
			t.Errorf("expected 2 with --warm error, got code=%d, err=%s", code, errOut)
		}
		if code, _, errOut := run("dev", "bench", "repos", "--corpus", "matrix.md", "--languages", "0"); code != 2 || !strings.Contains(errOut, "--languages") {
			t.Errorf("expected 2 with --languages error, got code=%d, err=%s", code, errOut)
		}
	})

	t.Run("Component timeout flag", func(t *testing.T) {
		mockRun := benchRepoRun
		defer func() { benchRepoRun = mockRun }()
		var got []time.Duration
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			got = append(got, opts.ComponentTimeout)
			return mockAudit, nil
		}
		run("dev", "bench", "repos", "--dir", ".")
		run("dev", "bench", "repos", "--dir", ".", "--component-timeout", "2s")
		if len(got) != 2 || got[0] != 60*time.Second || got[1] != 2*time.Second {
			t.Errorf("component timeouts = %v, want [1m0s 2s]", got)
		}
	})

	t.Run("Single repo success to stdout", func(t *testing.T) {
		stamp := pinBenchClock(t)
		code, out, _ := run("dev", "bench", "repos", "--dir", ".")
		if code != 0 {
			t.Fatalf("expected 0, got %d", code)
		}
		if !strings.HasPrefix(out, "# loomux dev bench repos — "+stamp+"\n") || !strings.Contains(out, "# Loomux Benchmark & Lücken-Audit") {
			t.Errorf("expected head and markdown report on stdout, got: %s", out)
		}
	})

	t.Run("Single repo with out writes both files", func(t *testing.T) {
		stamp := pinBenchClock(t)
		dir := t.TempDir()
		code, out, errOut := run("dev", "bench", "repos", "--dir", ".", "--out", dir)
		if code != 0 {
			t.Fatalf("expected 0, got %d: %s", code, errOut)
		}
		if out != "" {
			t.Errorf("the markdown went to its file, yet stdout has %q", out)
		}
		base := filepath.Join(dir, "bench-"+stamp+"-repos")
		text, err := os.ReadFile(base + ".md")
		if err != nil || !strings.HasPrefix(string(text), "# loomux dev bench repos — "+stamp) || !strings.Contains(string(text), "# Loomux Benchmark & Lücken-Audit") {
			t.Fatalf("%v\n%s", err, text)
		}
		data, err := os.ReadFile(base + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var report benchreport.Report
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if report.Command != "repos" || report.Stamp != stamp || report.Environment.Loomux != Version ||
			len(report.Timings) != 1 || report.Timings[0].Name != "/mock/repo" || report.Timings[0].MedianMS != 42 {
			t.Fatalf("report %s", data)
		}
	})

	t.Run("Out refuses a taken target before measuring", func(t *testing.T) {
		stamp := pinBenchClock(t)
		mockRun, mockCorpus := benchRepoRun, benchCorpusRun
		defer func() { benchRepoRun, benchCorpusRun = mockRun, mockCorpus }()
		calls := 0
		benchRepoRun = func(string, benchcorpus.Options, benchcorpus.ProcessRunner, func() time.Time, func(string) (fs.FS, error), func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			calls++
			return mockAudit, nil
		}
		benchCorpusRun = func([]byte, benchcorpus.Options, benchcorpus.Cloner, func(string, benchcorpus.Options) (*benchcorpus.RepoAudit, error)) (*benchcorpus.BenchmarkReport, error) {
			calls++
			return nil, errors.New("measured")
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bench-"+stamp+"-repos.json"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		code, _, errOut := run("dev", "bench", "repos", "--dir", ".", "--out", dir)
		if code != 1 || !strings.Contains(errOut, "loomux dev bench repos:") || !strings.Contains(errOut, "already exists") || calls != 0 {
			t.Fatalf("code %d, calls %d, err %q", code, calls, errOut)
		}
		code, _, errOut = run("dev", "bench", "repos", "--corpus", "matrix.md", "--out", "bench.md")
		if code != 1 || !strings.Contains(errOut, "no directory at bench.md") || calls != 0 {
			t.Fatalf("code %d, calls %d, err %q", code, calls, errOut)
		}
	})

	t.Run("Single repo benchRepo failure", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return nil, errors.New("inspection crashed")
		}

		code, _, errOut := run("dev", "bench", "repos", "--dir", ".")
		if code != 1 || !strings.Contains(errOut, "loomux dev bench repos: repository benchmark: inspection crashed") {
			t.Fatalf("expected code 1 with error, got code=%d err=%s", code, errOut)
		}
	})

	t.Run("Single repo write failure", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return mockAudit, nil
		}
		benchWriteBoth = func(string, string, []byte, []byte) error { return errors.New("disk full") }
		defer func() { benchWriteBoth = origWriteBoth }()

		code, _, errOut := run("dev", "bench", "repos", "--dir", ".", "--out", t.TempDir())
		if code != 1 || !strings.Contains(errOut, "loomux dev bench repos: writing the report: disk full") {
			t.Fatalf("expected code 1 with disk full error, got code=%d err=%s", code, errOut)
		}
	})

	t.Run("Single repo JSON failure", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return &benchcorpus.RepoAudit{Dir: "/r", Speedup: math.NaN()}, nil
		}
		dir := t.TempDir()
		code, _, errOut := run("dev", "bench", "repos", "--dir", ".", "--out", dir)
		if code != 1 || !strings.Contains(errOut, "loomux dev bench repos: encoding the report:") {
			t.Fatalf("expected code 1 with an encoding error, got code=%d err=%s", code, errOut)
		}
		if entries, _ := os.ReadDir(dir); len(entries) != 0 {
			t.Fatalf("a half report was written: %v", entries)
		}
	})

	t.Run("Corpus mode success", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return mockAudit, nil
		}
		benchReadFile = func(name string) ([]byte, error) {
			return []byte("# matrix"), nil
		}

		code, out, errOut := run("dev", "bench", "repos", "--corpus", "matrix.md")
		if code != 0 {
			t.Fatalf("expected 0, got %d", code)
		}
		if !strings.Contains(errOut, "loomux dev bench repos: skipped https://github.com/x/y: fatal: unable to checkout working tree") {
			t.Errorf("expected skipped repository on stderr, got %s", errOut)
		}
		if !strings.Contains(out, "# Loomux Benchmark & Lücken-Audit") {
			t.Errorf("expected markdown report on stdout")
		}
	})

	t.Run("Corpus mode read file failure", func(t *testing.T) {
		benchReadFile = func(name string) ([]byte, error) {
			return nil, errors.New("missing matrix file")
		}

		code, _, errOut := run("dev", "bench", "repos", "--corpus", "missing.md")
		if code != 1 || !strings.Contains(errOut, "missing matrix file") {
			t.Fatalf("expected code 1 with read error, got code=%d err=%s", code, errOut)
		}
	})

	t.Run("Corpus mode corpus benchmark failure", func(t *testing.T) {
		benchReadFile = func(name string) ([]byte, error) {
			return []byte("# matrix"), nil
		}
		benchCorpusRun = func(matrixData []byte, opts benchcorpus.Options, cloner benchcorpus.Cloner, benchRepo func(string, benchcorpus.Options) (*benchcorpus.RepoAudit, error)) (*benchcorpus.BenchmarkReport, error) {
			return nil, errors.New("corpus run failed")
		}

		code, _, errOut := run("dev", "bench", "repos", "--corpus", "matrix.md")
		if code != 1 || !strings.Contains(errOut, "corpus run failed") {
			t.Fatalf("expected code 1 with corpus error, got code=%d err=%s", code, errOut)
		}
	})

	t.Run("Single repo with save flag success", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return mockAudit, nil
		}
		var savedReport *benchcorpus.BenchmarkReport
		var savedDir string
		benchSaveReport = func(report *benchcorpus.BenchmarkReport, docsDir string, ops benchcorpus.StorageOps) error {
			savedReport = report
			savedDir = docsDir
			return nil
		}

		code, _, _ := run("dev", "bench", "repos", "--dir", ".", "--save", "--report-dir", "custom/docs")
		if code != 0 {
			t.Fatalf("expected 0, got %d", code)
		}
		if savedReport == nil || len(savedReport.Repos) != 1 {
			t.Errorf("expected saved report with 1 repo")
		}
		if savedDir != "custom/docs" {
			t.Errorf("expected custom/docs dir, got %s", savedDir)
		}
	})

	t.Run("Single repo with save flag failure", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return mockAudit, nil
		}
		benchSaveReport = func(report *benchcorpus.BenchmarkReport, docsDir string, ops benchcorpus.StorageOps) error {
			return errors.New("cannot write docs")
		}

		code, _, errOut := run("dev", "bench", "repos", "--dir", ".", "--save")
		if code != 1 || !strings.Contains(errOut, "cannot write docs") {
			t.Fatalf("expected code 1 with save error, got code=%d err=%s", code, errOut)
		}
	})
}

// --merge-fixture appends the extra answers to every translated world; extra
// answers it cannot read fail the import after the cases were written.
func TestDevImportCasesMergesTheExtraAnswers(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	caseDir := filepath.Join(from, "check", "one")
	if err := os.MkdirAll(filepath.Join(caseDir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"cmd": "ultraloom check all --root {{WORLD}}\n", "exit": "1\n", "stdout": ""} {
		if err := os.WriteFile(filepath.Join(caseDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mapFile := filepath.Join(t.TempDir(), "map.toml")
	if err := os.WriteFile(mapFile, []byte("[[command]]\nfrom = \"ultraloom check \"\nto = \"loomux check \"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	extra := filepath.Join(t.TempDir(), "extra.json")
	if err := os.WriteFile(extra, []byte(`{"answers": [{"prefix": "cmake --build", "exit": 0}]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, errOut := run("dev", "import-cases", "--map", mapFile, "--from", from, "--to", to, "--merge-fixture", extra)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
	got, err := os.ReadFile(filepath.Join(to, "check", "one", "world", "faketool.json"))
	if err != nil || !strings.Contains(string(got), `"prefix": "cmake --build"`) {
		t.Fatalf("%v %s", err, got)
	}

	code, _, errOut = run("dev", "import-cases", "--map", mapFile, "--from", from, "--to", to, "--merge-fixture", filepath.Join(t.TempDir(), "gone.json"))
	if code != 1 || !strings.Contains(errOut, "loomux dev import-cases: reading the extra answers") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// stubBenchSearch replaces the search bench with one that records what it
// was asked and answers text or err.
func stubBenchSearch(t *testing.T, text string, err error) *benchsearch.Options {
	t.Helper()
	var asked benchsearch.Options
	benchSearchRun = func(o benchsearch.Options, _ benchsearch.Deps) (string, error) {
		asked = o
		return text, err
	}
	t.Cleanup(func() { benchSearchRun = benchsearch.Bench })
	return &asked
}

func stubRepoRoot(t *testing.T, root string, err error) {
	t.Helper()
	benchRepoRoot = func() (string, error) { return root, err }
	t.Cleanup(func() { benchRepoRoot = gitTopLevel })
}

func TestDevBenchSearchRefusesBadFlags(t *testing.T) {
	stubBenchSearch(t, "", errors.New("not reached"))
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--frobnicate"}, "flag provided but not defined"},
		{[]string{"--profile", "slow"}, `--profile must be keyword, fast or full, got "slow"`},
		{[]string{"--channel", "radio"}, "invalid channel"},
		{[]string{"--repeat", "0"}, "--repeat must be at least 1, got 0"},
		{[]string{"leftover"}, `unexpected argument "leftover"`},
	} {
		code, _, errOut := run(append([]string{"dev", "bench", "search"}, c.args...)...)
		if code != 2 || !strings.Contains(errOut, c.want) {
			t.Errorf("%v: code %d, err %q", c.args, code, errOut)
		}
	}
}

func TestDevBenchSearchHandsOnTheFlags(t *testing.T) {
	asked := stubBenchSearch(t, "# report\n", nil)
	code, out, errOut := run("dev", "bench", "search", "--scope", "all", "--profile", "full", "--channel", "cloud",
		"--out", "o", "--questions", "q.yaml", "--latency", "--latency-query", "x", "--repeat", "3")
	if code != 0 || out != "# report\n" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	want := benchsearch.Options{Scope: "all", ScopeSet: true, Profile: search.ProfileFull, Channel: privacy.ChannelCloud,
		Out: "o", Questions: "q.yaml", Latency: true, LatencyQuery: "x", Repeat: 3}
	if *asked != want {
		t.Fatalf("asked %+v", *asked)
	}
	run("dev", "bench", "search")
	want = benchsearch.Options{Scope: "knowledge", Profile: search.ProfileFast, Channel: privacy.ChannelLocal, LatencyQuery: "latenz", Repeat: 10}
	if *asked != want {
		t.Fatalf("defaults %+v", *asked)
	}
}

func TestDevBenchSearchFindsCorpusV1InTheCheckout(t *testing.T) {
	asked := stubBenchSearch(t, "", nil)
	root := t.TempDir()
	stand := filepath.Join(root, "testdata", "bench", "search", "v1")
	if err := os.MkdirAll(stand, 0o755); err != nil {
		t.Fatal(err)
	}
	stubRepoRoot(t, root, nil)
	if code, _, errOut := run("dev", "bench", "search", "--corpus", "v1", "--out", "o"); code != 0 || asked.Corpus != stand {
		t.Fatalf("code %d, corpus %q, err %q", code, asked.Corpus, errOut)
	}
	if run("dev", "bench", "search", "--corpus", "elsewhere"); asked.Corpus != "elsewhere" {
		t.Fatalf("corpus %q", asked.Corpus)
	}
}

func TestDevBenchSearchNeedsACheckoutForCorpusV1(t *testing.T) {
	stubBenchSearch(t, "", errors.New("not reached"))
	for _, err := range []error{nil, errors.New("not a git repository")} {
		stubRepoRoot(t, t.TempDir(), err)
		code, _, errOut := run("dev", "bench", "search", "--corpus", "v1")
		if code != 1 || errOut != "error: --corpus v1 needs a loomux checkout; name the stand's directory instead\n" {
			t.Fatalf("code %d, err %q", code, errOut)
		}
	}
}

func TestDevBenchSearchPrintsEveryProblem(t *testing.T) {
	stubBenchSearch(t, "", benchsearch.Problems{"one", "two"})
	code, out, errOut := run("dev", "bench", "search")
	if code != 1 || out != "" || errOut != "error: one\nerror: two\n" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	stubBenchSearch(t, "", errors.New("refused"))
	if code, _, errOut := run("dev", "bench", "search"); code != 1 || errOut != "error: refused\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	// qmd's stderr, a crash dump among it, arrives inside one error.
	stubBenchSearch(t, "", errors.New("qmd exited with 1: first\n\nsecond\n"))
	if code, _, errOut := run("dev", "bench", "search"); code != 1 || errOut != "error: qmd exited with 1: first\nerror: second\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestBenchSearchDepsReachTheRealSystem(t *testing.T) {
	t.Setenv("QMD_LLAMA_GPU", "vulkan")
	t.Setenv("QMD_FORCE_CPU", "")
	var stderr strings.Builder
	d := benchSearchDeps(&stderr)
	if _, ok := d.Daemon().(*search.QmdMcpPort); !ok {
		t.Fatal("the daemon is no MCP port")
	}
	if port, ok := d.CLI("loomux-bench-x").(*search.QmdPort); !ok || port.Index != "loomux-bench-x" || port.Executable != "qmd" {
		t.Fatalf("cli = %+v", port)
	}
	if backbone := benchreport.Backbone(d.Getenv); backbone != "vulkan" {
		t.Fatalf("backbone %q", backbone)
	}
	if first, second := d.Random(), d.Random(); first == "" || first == second || strings.ContainsAny(first, `/\. `) {
		t.Fatalf("random %q, %q", first, second)
	}
	d.Warn("careful")
	if stderr.String() != "warning: careful\n" || d.Loomux != Version || d.StateDir == "" {
		t.Fatalf("stderr %q, deps %+v", stderr.String(), d)
	}
}
