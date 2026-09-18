package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckNeedsASubcommand(t *testing.T) {
	if code, _, errOut := run("check"); code != 2 || !strings.Contains(errOut, "loomux check: subcommand required") {
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

// Without paths the check reads the working directory, as ulinit did.
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

func TestCheckUnknownSubcommandIsAUsageError(t *testing.T) {
	code, _, errOut := run("check", "types")
	if code != 2 || !strings.Contains(errOut, `loomux check: unknown subcommand "types"`) {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestCheckCommitMsgRefusesAMissingHeader(t *testing.T) {
	file := filepath.Join(t.TempDir(), "msg")
	os.WriteFile(file, []byte("Add the first loomux command\n"), 0o644)
	if code, _, errOut := run("check", "commit-msg", file); code != 1 || !strings.Contains(errOut, "<type>[(<scope>)][!]: <description>") {
		t.Fatalf("code %d: %s", code, errOut)
	}
}
