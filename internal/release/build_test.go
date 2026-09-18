package release

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGo writes the -o target with content naming its GOOS/GOARCH, so the
// checksums differ per file.
func fakeGo(calls *[][]string) GoBuild {
	return func(env []string, args ...string) error {
		*calls = append(*calls, append(append([]string{}, env...), args...))
		for i, a := range args {
			if a == "-o" {
				return os.WriteFile(args[i+1], []byte(strings.Join(env, " ")), 0o755)
			}
		}
		return errors.New("no -o")
	}
}

func TestBuildWritesAllTargetsAndSums(t *testing.T) {
	out := t.TempDir()
	var calls [][]string
	names, err := Build("1.2.3", "beta", out, fakeGo(&calls))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"loomux_1.2.3_windows_amd64.exe", "loomux_1.2.3_linux_amd64", "loomux_1.2.3_linux_arm64",
		"loomux_1.2.3_darwin_amd64", "loomux_1.2.3_darwin_arm64", "SHA256SUMS",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("names %v", names)
	}
	first := strings.Join(calls[0], " ")
	for _, part := range []string{"CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64", "-trimpath",
		"-X github.com/xidus90/loomux/internal/cli.Version=1.2.3 -X github.com/xidus90/loomux/internal/cli.Channel=beta",
		"./cmd/loomux"} {
		if !strings.Contains(first, part) {
			t.Fatalf("call %q lacks %q", first, part)
		}
	}
	sums, _ := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
	data, _ := os.ReadFile(filepath.Join(out, want[0]))
	sum := sha256.Sum256(data)
	if !strings.HasPrefix(string(sums), hex.EncodeToString(sum[:])+"  "+want[0]+"\n") {
		t.Fatalf("sums %q", sums)
	}
}

func TestBuildStopsAtTheFirstFailure(t *testing.T) {
	fail := func([]string, ...string) error { return errors.New("boom") }
	if _, err := Build("1.0.0", "", t.TempDir(), fail); err == nil || !strings.Contains(err.Error(), "windows/amd64: boom") {
		t.Fatalf("err %v", err)
	}
}

func TestBuildReportsAnUnwritableOutput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	var calls [][]string
	if _, err := Build("1.0.0", "", file, fakeGo(&calls)); err == nil {
		t.Fatal("want error for an output that is a file")
	}
}

func TestBuildReportsAMissingBinary(t *testing.T) {
	noop := func([]string, ...string) error { return nil }
	if _, err := Build("1.0.0", "", t.TempDir(), noop); err == nil {
		t.Fatal("want error when go build wrote nothing")
	}
}

func TestBuildReportsUnwritableSums(t *testing.T) {
	out := t.TempDir()
	var calls [][]string
	inner := fakeGo(&calls)
	blockSums := func(env []string, args ...string) error {
		if len(calls) == len(Targets)-1 {
			os.Mkdir(filepath.Join(out, "SHA256SUMS"), 0o755)
		}
		return inner(env, args...)
	}
	if _, err := Build("1.0.0", "", out, blockSums); err == nil {
		t.Fatal("want error when SHA256SUMS cannot be written")
	}
}

func TestExecGoBuildRunsGo(t *testing.T) {
	if err := ExecGoBuild(nil, "version"); err != nil {
		t.Fatal(err)
	}
	if err := ExecGoBuild(nil, "no-such-subcommand"); err == nil {
		t.Fatal("want error")
	}
}
