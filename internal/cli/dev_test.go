package cli

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchcorpus"
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

func TestDevBench(t *testing.T) {
	origRepoRun := benchRepoRun
	origCorpusRun := benchCorpusRun
	origReadFile := benchReadFile
	origWriteFile := benchWriteFile
	origSaveReport := benchSaveReport
	origStorageOps := benchStorageOps
	defer func() {
		benchRepoRun = origRepoRun
		benchCorpusRun = origCorpusRun
		benchReadFile = origReadFile
		benchWriteFile = origWriteFile
		benchSaveReport = origSaveReport
		benchStorageOps = origStorageOps
	}()

	mockAudit := &benchcorpus.RepoAudit{
		Dir:          "/mock/repo",
		CoverageRate: 100.0,
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
		if code, _, _ := run("dev", "bench", "--bogus"); code != 2 {
			t.Errorf("expected 2 for unknown flag, got %d", code)
		}
		if code, _, errOut := run("dev", "bench", "--warm", "0"); code != 2 || !strings.Contains(errOut, "--warm") {
			t.Errorf("expected 2 with --warm error, got code=%d, err=%s", code, errOut)
		}
		if code, _, errOut := run("dev", "bench", "--corpus", "matrix.md", "--languages", "0"); code != 2 || !strings.Contains(errOut, "--languages") {
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
		run("dev", "bench", "--dir", ".")
		run("dev", "bench", "--dir", ".", "--component-timeout", "2s")
		if len(got) != 2 || got[0] != 60*time.Second || got[1] != 2*time.Second {
			t.Errorf("component timeouts = %v, want [1m0s 2s]", got)
		}
	})

	t.Run("Single repo success to stdout", func(t *testing.T) {
		code, out, _ := run("dev", "bench", "--dir", ".")
		if code != 0 {
			t.Fatalf("expected 0, got %d", code)
		}
		if !strings.Contains(out, "# Loomux Benchmark & Lücken-Audit") {
			t.Errorf("expected markdown report on stdout, got: %s", out)
		}
	})

	t.Run("Single repo with out and json-out files", func(t *testing.T) {
		var writtenFiles = make(map[string]string)
		benchWriteFile = func(name string, data []byte, perm os.FileMode) error {
			writtenFiles[name] = string(data)
			return nil
		}

		code, _, _ := run("dev", "bench", "--dir", ".", "--out", "bench.md", "--json-out", "bench.json")
		if code != 0 {
			t.Fatalf("expected 0, got %d", code)
		}
		if _, ok := writtenFiles["bench.md"]; !ok {
			t.Errorf("expected bench.md to be written")
		}
		if _, ok := writtenFiles["bench.json"]; !ok {
			t.Errorf("expected bench.json to be written")
		}
	})

	t.Run("Single repo benchRepo failure", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return nil, errors.New("inspection crashed")
		}

		code, _, errOut := run("dev", "bench", "--dir", ".")
		if code != 1 || !strings.Contains(errOut, "inspection crashed") {
			t.Fatalf("expected code 1 with error, got code=%d err=%s", code, errOut)
		}
	})

	t.Run("Single repo write markdown file failure", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return mockAudit, nil
		}
		benchWriteFile = func(name string, data []byte, perm os.FileMode) error {
			return errors.New("disk full")
		}

		code, _, errOut := run("dev", "bench", "--dir", ".", "--out", "out.md")
		if code != 1 || !strings.Contains(errOut, "disk full") {
			t.Fatalf("expected code 1 with disk full error, got code=%d err=%s", code, errOut)
		}
	})

	t.Run("Single repo write JSON file failure", func(t *testing.T) {
		benchRepoRun = func(dir string, opts benchcorpus.Options, runner benchcorpus.ProcessRunner, clock func() time.Time, openFS func(string) (fs.FS, error), lookPath func(string) (string, error)) (*benchcorpus.RepoAudit, error) {
			return mockAudit, nil
		}
		benchWriteFile = func(name string, data []byte, perm os.FileMode) error {
			if strings.HasSuffix(name, ".json") {
				return errors.New("json disk full")
			}
			return nil
		}

		code, _, errOut := run("dev", "bench", "--dir", ".", "--out", "out.md", "--json-out", "out.json")
		if code != 1 || !strings.Contains(errOut, "json disk full") {
			t.Fatalf("expected code 1 with json error, got code=%d err=%s", code, errOut)
		}
	})

	t.Run("Corpus mode success", func(t *testing.T) {
		benchReadFile = func(name string) ([]byte, error) {
			return []byte("# matrix"), nil
		}

		code, out, errOut := run("dev", "bench", "--corpus", "matrix.md")
		if code != 0 {
			t.Fatalf("expected 0, got %d", code)
		}
		if !strings.Contains(errOut, "skipped https://github.com/x/y: fatal: unable to checkout working tree") {
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

		code, _, errOut := run("dev", "bench", "--corpus", "missing.md")
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

		code, _, errOut := run("dev", "bench", "--corpus", "matrix.md")
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

		code, _, _ := run("dev", "bench", "--dir", ".", "--save", "--report-dir", "custom/docs")
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

		code, _, errOut := run("dev", "bench", "--dir", ".", "--save")
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
