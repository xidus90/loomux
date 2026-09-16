package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchhooks"
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
