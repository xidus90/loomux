package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestDevCovergateFailsOnUnparsableOutput(t *testing.T) {
	coverFunc = func(string) ([]byte, error) { return []byte("garbage\n"), nil }
	defer func() { coverFunc = runCoverFunc }()
	code, _, errOut := run("dev", "covergate")
	if code != 1 || !strings.Contains(errOut, "unexpected cover line") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}
