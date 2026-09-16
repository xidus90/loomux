package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchhooks"
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

func TestDevCovergateReadsTheCoverTool(t *testing.T) {
	coverFunc = func(profile string) ([]byte, error) {
		return []byte("github.com/xidus90/loomux/internal/cli/cli.go:1:\tRun\t100.0%\n"), nil
	}
	defer func() { coverFunc = runCoverFunc }()
	if code, _, errOut := run("dev", "covergate", "--profile", "c.out"); code != 0 {
		t.Fatalf("code %d: %s", code, errOut)
	}
}

func TestDevCovergateRefusesAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("dev", "covergate", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestDevCovergateFailsWhenTheCoverToolFails(t *testing.T) {
	coverFunc = func(string) ([]byte, error) { return nil, errors.New("no profile") }
	defer func() { coverFunc = runCoverFunc }()
	code, _, errOut := run("dev", "covergate")
	if code != 1 || !strings.Contains(errOut, "loomux dev covergate: no profile") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevCovergateRefusesAProfileWithoutFunctions(t *testing.T) {
	coverFunc = func(string) ([]byte, error) { return []byte("total:\t(statements)\t0.0%\n"), nil }
	defer func() { coverFunc = runCoverFunc }()
	code, _, errOut := run("dev", "covergate")
	if code != 1 || !strings.Contains(errOut, "loomux dev covergate: no functions in") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestRunCoverFuncReadsAValidProfile(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "c.out")
	if err := os.WriteFile(profile, []byte("mode: set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runCoverFunc(profile); err != nil {
		t.Fatal(err)
	}
}

func TestRunCoverFuncReportsTheToolsStderr(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "missing.out")
	_, err := runCoverFunc(profile)
	if err == nil || !strings.Contains(err.Error(), "missing.out") {
		t.Fatalf("err %v", err)
	}
}

func TestDevCovergateFailsOnUnparsableOutput(t *testing.T) {
	coverFunc = func(string) ([]byte, error) { return []byte("garbage\n"), nil }
	defer func() { coverFunc = runCoverFunc }()
	code, _, errOut := run("dev", "covergate")
	if code != 1 || !strings.Contains(errOut, "unexpected cover line") {
		t.Fatalf("code %d, err %q", code, errOut)
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

func TestDevBenchHooksMeasuresTheCases(t *testing.T) {
	benchExec = func(benchhooks.Case, benchhooks.Step) (int, error) { return 0, nil }
	defer func() { benchExec = benchhooks.Exec }()
	code, out, errOut := run("dev", "bench-hooks", benchCases(t, oneCase), "-n", "2")
	if code != 0 || !strings.Contains(out, "| probe |") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestDevBenchHooksTakesTheFlagBeforeTheFile(t *testing.T) {
	benchExec = func(benchhooks.Case, benchhooks.Step) (int, error) { return 0, nil }
	defer func() { benchExec = benchhooks.Exec }()
	code, out, _ := run("dev", "bench-hooks", "-n", "1", benchCases(t, oneCase))
	if code != 0 || !strings.Contains(out, "| probe |") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestDevBenchHooksNeedsACaseFile(t *testing.T) {
	code, _, errOut := run("dev", "bench-hooks")
	if code != 2 || !strings.Contains(errOut, "loomux dev bench-hooks: a case file is required") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesAnUnknownFlag(t *testing.T) {
	if code, _, _ := run("dev", "bench-hooks", "--bogus"); code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestDevBenchHooksReportsAnUnreadableCaseFile(t *testing.T) {
	code, _, errOut := run("dev", "bench-hooks", filepath.Join(t.TempDir(), "gone.json"))
	if code != 1 || !strings.Contains(errOut, "loomux dev bench-hooks:") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksReportsABrokenMeasurement(t *testing.T) {
	benchExec = func(benchhooks.Case, benchhooks.Step) (int, error) {
		return 0, errors.New("no such binary")
	}
	defer func() { benchExec = benchhooks.Exec }()
	code, _, errOut := run("dev", "bench-hooks", benchCases(t, oneCase))
	if code != 1 || !strings.Contains(errOut, "no such binary") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesAnUnknownFlagBehindTheFile(t *testing.T) {
	code, _, _ := run("dev", "bench-hooks", benchCases(t, oneCase), "--bogus")
	if code != 2 {
		t.Fatalf("code %d", code)
	}
}

func TestDevBenchHooksRefusesLessThanOneWarmRun(t *testing.T) {
	code, _, errOut := run("dev", "bench-hooks", "-n", "0", benchCases(t, oneCase))
	if code != 2 || !strings.Contains(errOut, "-n must be at least 1, got 0") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesANegativeWarmRunBehindTheFile(t *testing.T) {
	code, _, errOut := run("dev", "bench-hooks", benchCases(t, oneCase), "-n", "-3")
	if code != 2 || !strings.Contains(errOut, "-n must be at least 1, got -3") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksRefusesAnExtraArgumentBehindTheFile(t *testing.T) {
	code, _, errOut := run("dev", "bench-hooks", benchCases(t, oneCase), "leftover")
	if code != 2 || !strings.Contains(errOut, `unexpected argument "leftover" after the case file`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevBenchHooksReportsBrokenJSON(t *testing.T) {
	code, _, errOut := run("dev", "bench-hooks", benchCases(t, "{"))
	if code != 1 || !strings.Contains(errOut, "loomux dev bench-hooks:") {
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
