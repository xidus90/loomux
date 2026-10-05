package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/brain/search/backbonetest"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/dev/benchcorpus"
	"github.com/xidus90/loomux/internal/dev/benchhooks"
	"github.com/xidus90/loomux/internal/dev/benchreport"
	"github.com/xidus90/loomux/internal/dev/benchsearch"
	"github.com/xidus90/loomux/internal/dev/faketool"
	"github.com/xidus90/loomux/internal/dev/mutants"
	"github.com/xidus90/loomux/internal/notices"
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
	for _, want := range []string{"usage: loomux dev bench <hooks|repos|search|compare|cases>", "hooks", "repos",
		"  search  measure the rank of search hits and the chain's latency",
		"  compare set two hook runs side by side (faster, new, dropped)",
		"  cases   build the case file of a project's hooks from its settings"} {
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
		!strings.Contains(errOut, "usage: loomux dev bench <hooks|repos|search|compare|cases>") {
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
func killEveryMutant(_, overlay string, _ time.Duration) (mutants.Outcome, error) {
	if overlay == "" {
		return mutants.Passed, nil
	}
	return mutants.Failed, nil
}

func TestDevMutantsRunsARound(t *testing.T) {
	mutantsWorld(t, killEveryMutant)
	code, out, errOut := run("dev", "mutants", "p", "--workers", "2")
	if code != 0 || !strings.HasPrefix(out, "p: each mutant run is bounded at 1m0s, the floor ") ||
		!strings.Contains(out, ")\n[1/4] killed    (a1) p.go:4  if a > 0 {  ->  if true {\n") ||
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
	mutantsWorld(t, func(string, string, time.Duration) (mutants.Outcome, error) { return mutants.Failed, nil })
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
	mutantsWorld(t, func(string, string, time.Duration) (mutants.Outcome, error) { return 0, errors.New("go: not found") })
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
	mutantsWorld(t, func(_, overlay string, _ time.Duration) (mutants.Outcome, error) {
		if overlay != "" {
			if entries, err := os.ReadDir(tmp); err != nil || len(entries) != 1 {
				t.Errorf("while the mutant runs: %v, %v", entries, err)
			}
			interrupt()
		}
		return mutants.Passed, nil
	})
	code, out, errOut := run("dev", "mutants", "p", "--workers", "1")
	// The bound line comes before the first mutant runs, the verdicts do not.
	if code != 1 || !strings.HasPrefix(out, "p: each mutant run is bounded at 1m0s, the floor ") ||
		strings.Count(out, "\n") != 1 || errOut != "loomux dev mutants: context canceled\n" {
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

func TestUntilInterruptedHandsTheBoundToTheSuite(t *testing.T) {
	var got time.Duration
	test := untilInterrupted(context.Background(), func() {}, func(_, _ string, bound time.Duration) (mutants.Outcome, error) {
		got = bound
		return mutants.Failed, nil
	})
	if outcome, err := test("p", "", 150*time.Second); err != nil || outcome != mutants.Failed || got != 150*time.Second {
		t.Fatalf("outcome %d, err %v, bound %s", outcome, err, got)
	}
}

func TestUntilInterruptedStartsNoRunAfterTheInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started, stopped := false, false
	test := untilInterrupted(ctx, func() { stopped = true }, func(string, string, time.Duration) (mutants.Outcome, error) {
		started = true
		return mutants.Passed, nil
	})
	if _, err := test("p", "", time.Minute); !errors.Is(err, context.Canceled) || started || !stopped {
		t.Fatalf("err %v, started %t, stopped %t", err, started, stopped)
	}
}

// The signal registration has to end with the first interrupt: while it
// stands, a second Ctrl+C is swallowed and the user cannot force the drain to
// end.
func TestUntilInterruptedStopsTheSignalRegistrationOfARunningSuite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stopped := false
	test := untilInterrupted(ctx, func() { stopped = true }, func(string, string, time.Duration) (mutants.Outcome, error) {
		cancel()
		return mutants.Passed, nil
	})
	if _, err := test("p", "", time.Minute); !errors.Is(err, context.Canceled) || !stopped {
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

// stubBenchSearch replaces the search bench with one that records what it
// was asked and answers text or err.
func stubBenchSearch(t *testing.T, text string, err error) *benchsearch.Options {
	t.Helper()
	// The deps read the machine-wide file; never the real one.
	t.Setenv(config.StateDirEnv, t.TempDir())
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

// benchState is a state directory of the test's own whose machine-wide
// config.toml says text.
func benchState(t *testing.T, text string) string {
	t.Helper()
	state := t.TempDir()
	t.Setenv(config.StateDirEnv, state)
	if err := os.WriteFile(filepath.Join(state, "config.toml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestBenchSearchDepsReachTheRealSystem(t *testing.T) {
	backbonetest.Clear(t)
	state := benchState(t, "[search]\nbackbone = \"cpu\"\n")
	var stderr strings.Builder
	d, err := benchSearchDeps(&stderr)
	if err != nil {
		t.Fatal(err)
	}
	if port, ok := d.Daemon().(*search.QmdMcpPort); !ok || port.Backbone() != search.BackboneCPU {
		t.Fatal("the daemon is no MCP port on the machine's backbone")
	}
	if port, ok := d.CLI("loomux-bench-x").(*search.QmdPort); !ok || port.Index != "loomux-bench-x" || port.Executable != "qmd" ||
		port.Backbone != search.BackboneCPU || port.Runner != nil {
		t.Fatalf("cli = %+v", port)
	}
	if d.Backbone != "cpu" || d.StateDir != state {
		t.Fatalf("backbone %q, state %q", d.Backbone, d.StateDir)
	}
	// The user's variable is what qmd runs on, and so what the report names.
	t.Setenv("QMD_LLAMA_GPU", "vulkan")
	t.Setenv("QMD_FORCE_CPU", "")
	if d, _ = benchSearchDeps(&stderr); d.Backbone != "vulkan" {
		t.Fatalf("backbone %q", d.Backbone)
	}
	if first, second := d.Random(), d.Random(); first == "" || first == second || strings.ContainsAny(first, `/\. `) {
		t.Fatalf("random %q, %q", first, second)
	}
	d.Warn("careful")
	if stderr.String() != "warning: careful\n" || d.Loomux != Version || d.StateDir == "" {
		t.Fatalf("stderr %q, deps %+v", stderr.String(), d)
	}
}

// A [search] block that does not read stops the bench before it measures,
// naming the file; a report must not name a backbone nobody could read.
func TestDevBenchSearchRefusesABrokenBackbone(t *testing.T) {
	asked := stubBenchSearch(t, "unreached", nil)
	state := benchState(t, "[search]\nbackbone = \"metal\"\n")
	code, out, errOut := run("dev", "bench", "search")
	if code != 1 || out != "" || !strings.Contains(errOut, "error: "+filepath.Join(state, "config.toml")) || asked.Scope != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

// The notice tests run from the repository's root, where go list finds
// ./cmd/loomux, and write to a temporary --out: a test run never rewrites the
// committed NOTICE.md.
func TestDevNoticesWritesTheEmbeddedNotice(t *testing.T) {
	out := filepath.Join(t.TempDir(), "NOTICE.md")
	t.Chdir("../..")
	code, stdout, errOut := run("dev", "notices", "--out", out)
	if code != 0 || stdout != out+"\n" {
		t.Fatalf("code %d, out %q, err %q", code, stdout, errOut)
	}
	written, err := os.ReadFile(out)
	if err != nil || string(written) != notices.Text() {
		t.Fatalf("the written notice is not the embedded one (%v); run loomux dev notices", err)
	}
}

func TestDevNoticesReportsAnUnwritableOut(t *testing.T) {
	out := filepath.Join(t.TempDir(), "missing", "NOTICE.md")
	t.Chdir("../..")
	code, stdout, errOut := run("dev", "notices", "--out", out)
	if code != 1 || stdout != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, stdout, errOut)
	}
}

func TestDevNoticesRefusesAnUnknownFlag(t *testing.T) {
	if code, _, errOut := run("dev", "notices", "--frobnicate"); code != 2 || !strings.Contains(errOut, "frobnicate") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestDevNoticesReportsWhatGoCannotList(t *testing.T) {
	out := filepath.Join(t.TempDir(), "NOTICE.md")
	t.Chdir(t.TempDir())
	code, _, errOut := run("dev", "notices", "--out", out)
	if code != 1 || !strings.Contains(errOut, "go list: ") || !strings.Contains(errOut, "go.mod") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if _, err := os.Stat(out); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a failed render wrote %s: %v", out, err)
	}
}

// The recording must replay: pdftotext.extract asks with the PDF's directory
// as working directory and the bare name, so the recorder asks the same, and
// the answer it writes is the one faketool finds for that very argv.
func TestDevRecordPopplerWritesOneAnswerPerPDF(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.pdf"), "%PDF")
	writeFile(t, filepath.Join(dir, "b.PDF"), "%PDF")
	writeFile(t, filepath.Join(dir, "c.txt"), "x")
	saved := popplerRun
	t.Cleanup(func() { popplerRun = saved })
	var specs []child.Spec
	popplerRun = func(spec child.Spec) child.Result {
		specs = append(specs, spec)
		if spec.Argv[1] == "-v" {
			return child.Result{Stdout: "banner\n", Stderr: "pdftotext version 25.07.0\n"}
		}
		return child.Result{Code: len(specs), Stdout: "Text of " + spec.Argv[len(spec.Argv)-2] + "\f", Stderr: "Syntax Error\r\n"}
	}
	exe := filepath.Join(t.TempDir(), "bin", "pdftotext.exe")
	out := filepath.Join(t.TempDir(), "faketool.json")
	code, stdout, _ := run("dev", "record-poppler", "--exe", exe, "--dir", dir, "--out", out)
	fixture, err := faketool.Load(out)
	if code != 0 || err != nil || len(fixture.Answers) != 3 || len(specs) != 3 {
		t.Fatalf("%d %v %+v %d", code, err, fixture, len(specs))
	}
	if specs[0].Argv[0] != exe || specs[0].Argv[1] != "-v" || len(specs[0].Argv) != 2 {
		t.Errorf("-v asked %q", specs[0].Argv)
	}
	version, ok := fixture.Match([]string{"pdftotext", "-v"})
	if !ok || version.Stdout != "banner\npdftotext version 25.07.0\n" || version.Exit != 0 {
		t.Errorf("-v recorded as %+v %v", version, ok)
	}
	for i, name := range []string{"a.pdf", "b.PDF"} {
		spec := specs[i+1]
		want := []string{exe, "-layout", "-enc", "UTF-8", "-eol", "unix", name, "-"}
		if spec.Dir != dir || strings.Join(spec.Argv, "|") != strings.Join(want, "|") {
			t.Errorf("%s asked %q in %q", name, spec.Argv, spec.Dir)
		}
		answer, ok := fixture.Match(spec.Argv)
		if !ok || answer.Stdout != "Text of "+name+"\f" || answer.Exit != i+2 {
			t.Errorf("%s replays as %+v %v", name, answer, ok)
		}
	}
	if strings.Count(stdout, "\n") != 2 || !strings.Contains(stdout, `b.PDF: exit 3, 14 bytes, stderr "Syntax Error"`) {
		t.Errorf("stdout %q", stdout)
	}
	for _, args := range [][]string{
		{"dev", "record-poppler", "--dir", dir, "--out", out},
		{"dev", "record-poppler", "--exe", "x", "--out", out},
		{"dev", "record-poppler", "--exe", "x", "--dir", dir},
		{"dev", "record-poppler", "--nope"},
	} {
		if code, _, _ := run(args...); code != 2 {
			t.Errorf("%q: %d", args, code)
		}
	}
	if code, _, _ := run("dev", "record-poppler", "--exe", "x", "--dir", filepath.Join(dir, "gone"), "--out", out); code != 1 {
		t.Errorf("an unreadable --dir: %d", code)
	}
	if code, _, _ := run("dev", "record-poppler", "--exe", "x", "--dir", dir, "--out", filepath.Join(dir, "gone", "f.json")); code != 1 {
		t.Errorf("an unwritable --out: %d", code)
	}
}
