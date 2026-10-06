package release

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/notices"
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
	names, err := Build("1.2.3", out, fakeGo(&calls))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"loomux_1.2.3_windows_amd64.exe", "loomux_1.2.3_linux_amd64", "loomux_1.2.3_linux_arm64",
		"loomux_1.2.3_darwin_amd64", "loomux_1.2.3_darwin_arm64", "NOTICE.md", "SHA256SUMS",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("names %v", names)
	}
	first := strings.Join(calls[0], " ")
	for _, part := range []string{"CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64", "-trimpath",
		"-ldflags -X github.com/xidus90/loomux/internal/cli.Version=1.2.3 -o",
		"./cmd/loomux"} {
		if !strings.Contains(first, part) {
			t.Fatalf("call %q lacks %q", first, part)
		}
	}
	// A new build leaves the channel unset: only a binary of the old count
	// carries one, and that is how it recognises itself.
	if strings.Contains(first, "Channel") {
		t.Fatalf("call %q sets a channel", first)
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
	if _, err := Build("1.0.0", t.TempDir(), fail); err == nil || !strings.Contains(err.Error(), "windows/amd64: boom") {
		t.Fatalf("err %v", err)
	}
}

func TestBuildReportsAnUnwritableOutput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	var calls [][]string
	if _, err := Build("1.0.0", file, fakeGo(&calls)); err == nil {
		t.Fatal("want error for an output that is a file")
	}
}

func TestBuildReportsAMissingBinary(t *testing.T) {
	noop := func([]string, ...string) error { return nil }
	if _, err := Build("1.0.0", t.TempDir(), noop); err == nil {
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
	if _, err := Build("1.0.0", out, blockSums); err == nil {
		t.Fatal("want error when SHA256SUMS cannot be written")
	}
}

func TestBuildShipsTheNoticeBesideTheBinaries(t *testing.T) {
	out := t.TempDir()
	names, err := Build("1.2.3", out, func(env []string, args ...string) error {
		return os.WriteFile(args[len(args)-2], []byte("binary"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if names[len(names)-2] != "NOTICE.md" || names[len(names)-1] != "SHA256SUMS" {
		t.Fatalf("%q", names)
	}
	notice, err := os.ReadFile(filepath.Join(out, "NOTICE.md"))
	if err != nil || string(notice) != notices.Text() {
		t.Fatal("NOTICE.md is not the embedded notice")
	}
	sums, err := os.ReadFile(filepath.Join(out, "SHA256SUMS"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(notice)
	if !strings.Contains(string(sums), hex.EncodeToString(sum[:])+"  NOTICE.md\n") {
		t.Fatalf("SHA256SUMS: %s", sums)
	}
}

func TestBuildStopsWhenTheNoticeCannotBeWritten(t *testing.T) {
	out := t.TempDir()
	notice := filepath.Join(out, "NOTICE.md")
	_, err := Build("1.2.3", out, func(env []string, args ...string) error {
		if err := os.MkdirAll(notice, 0o755); err != nil {
			return err
		}
		return os.WriteFile(args[len(args)-2], []byte("binary"), 0o644)
	})
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != notice {
		t.Fatalf("want the failed write of %s, got %v", notice, err)
	}
	if _, err := os.Stat(filepath.Join(out, "SHA256SUMS")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("SHA256SUMS written without the notice: %v", err)
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
