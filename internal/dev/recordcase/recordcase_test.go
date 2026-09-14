package recordcase

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const helperEnv = "LOOMUX_RECORDCASE_HELPER"

// TestHelperProcess stands in for an old binary: run as a child with
// helperEnv set, it acts on the arguments after "--" and exits.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv(helperEnv)
	if mode == "" {
		return
	}
	var args []string
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	switch mode {
	case "echo":
		os.Stdout.WriteString(strings.Join(args, " ") + "\n")
		os.Exit(0)
	case "write":
		os.WriteFile(args[0], []byte("made at "+args[0]+"\n"), 0o644)
		os.Exit(0)
	default:
		os.Stdout.WriteString("refused\n")
		os.Exit(2)
	}
}

// helperSpec builds a Spec that runs the test binary as the old tool.
func helperSpec(t *testing.T, mode, cmd string) Spec {
	t.Helper()
	t.Setenv(helperEnv, mode)
	return Spec{
		Exe:   os.Args[0],
		Cmd:   "ulguard -test.run=TestHelperProcess -- " + cmd,
		World: t.TempDir(),
		Out:   filepath.Join(t.TempDir(), "guard", "one"),
		Notes: "tag loomux-1a-source, ulguard, what the case shows",
	}
}

func read(t *testing.T, parts ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("reading %v: %v", parts, err)
	}
	return string(data)
}

func TestRecordKeepsTheWorldTokenAndTheRecordedWorld(t *testing.T) {
	s := helperSpec(t, "echo", "{{WORLD}}/a.txt")
	os.WriteFile(filepath.Join(s.World, "a.txt"), []byte("first\n"), 0o644)
	if err := Record(s); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.Out, "cmd"); got != s.Cmd+"\n" {
		t.Errorf("cmd %q", got)
	}
	if got := read(t, s.Out, "stdout"); got != "{{WORLD}}/a.txt\n" {
		t.Errorf("stdout %q", got)
	}
	if got := read(t, s.Out, "exit"); got != "0\n" {
		t.Errorf("exit %q", got)
	}
	if got := read(t, s.Out, "notes.md"); got != s.Notes+"\n" {
		t.Errorf("notes %q", got)
	}
	if got := read(t, s.Out, "world", "a.txt"); got != "first\n" {
		t.Errorf("world %q", got)
	}
	if _, err := os.Stat(filepath.Join(s.Out, "world_after")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a case that changes nothing must not carry world_after: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Out, "compare")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("compare is written only when it is set: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Out, "stdin")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("stdin is written only when there is one: %v", err)
	}
}

func TestRecordWritesTheWorldAfterWhenTheRunChangesIt(t *testing.T) {
	s := helperSpec(t, "write", "{{WORLD}}/made.txt")
	s.Compare = "message"
	stdin := filepath.Join(t.TempDir(), "payload.json")
	os.WriteFile(stdin, []byte(`{"path":"{{WORLD}}/made.txt"}`), 0o644)
	s.Stdin = stdin
	if err := Record(s); err != nil {
		t.Fatal(err)
	}

	if got := read(t, s.Out, "world_after", "made.txt"); got != "made at {{WORLD}}/made.txt\n" {
		t.Errorf("world_after %q", got)
	}
	if got := read(t, s.Out, "compare"); got != "message\n" {
		t.Errorf("compare %q", got)
	}
	if got := read(t, s.Out, "stdin"); got != `{"path":"{{WORLD}}/made.txt"}` {
		t.Errorf("stdin %q", got)
	}
}

func TestRecordKeepsANonZeroExit(t *testing.T) {
	s := helperSpec(t, "refuse", "{{WORLD}}")
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s.Out, "exit"); got != "2\n" {
		t.Errorf("exit %q", got)
	}
}

func TestRecordReportsABinaryItCannotRun(t *testing.T) {
	s := helperSpec(t, "echo", "{{WORLD}}")
	s.Exe = filepath.Join(t.TempDir(), "no-such-binary.exe")
	if err := Record(s); err == nil {
		t.Fatal("want error")
	}
}

func TestRecordReportsWhatItCannotRead(t *testing.T) {
	s := helperSpec(t, "echo", "{{WORLD}}")
	s.World = filepath.Join(t.TempDir(), "gone")
	if err := Record(s); err == nil {
		t.Fatal("want error for a missing world")
	}

	s = helperSpec(t, "echo", "{{WORLD}}")
	s.Cmd = `ulguard "unclosed`
	if err := Record(s); err == nil {
		t.Fatal("want error for a command that cannot be split")
	}

	s = helperSpec(t, "echo", "{{WORLD}}")
	s.Stdin = filepath.Join(t.TempDir(), "gone.json")
	if err := Record(s); err == nil {
		t.Fatal("want error for a missing stdin file")
	}
}

func TestRecordReportsWhatItCannotWrite(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "file")
	os.WriteFile(blocked, []byte("x"), 0o644)

	s := helperSpec(t, "echo", "{{WORLD}}")
	s.Out = filepath.Join(blocked, "guard", "one")
	if err := Record(s); err == nil {
		t.Fatal("want error for an out directory under a file")
	}
}

func TestRecordReportsATempDirItCannotMake(t *testing.T) {
	mkdirTemp = func(string, string) (string, error) { return "", errors.New("no temp") }
	defer func() { mkdirTemp = os.MkdirTemp }()
	if err := Record(helperSpec(t, "echo", "{{WORLD}}")); err == nil {
		t.Fatal("want error")
	}
}

func TestRecordReportsACaseFileItCannotWrite(t *testing.T) {
	s := helperSpec(t, "echo", "{{WORLD}}")
	if err := os.MkdirAll(filepath.Join(s.Out, "exit"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Record(s); err == nil {
		t.Fatal("want error for a case file blocked by a directory")
	}
}

func TestRecordReportsAWorldItCannotCopy(t *testing.T) {
	s := helperSpec(t, "echo", "{{WORLD}}")
	if err := os.MkdirAll(s.Out, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Out, "world"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Record(s); err == nil {
		t.Fatal("want error for a world directory blocked by a file")
	}
}

func TestRecordCopiesNestedDirectoriesAndReportsAWorldAfterItCannotWrite(t *testing.T) {
	s := helperSpec(t, "write", "{{WORLD}}/sub/made.txt")
	if err := os.MkdirAll(filepath.Join(s.World, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.World, "sub", "kept.txt"), []byte("kept\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(s.Out, "world_after", "sub", "made.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Record(s); err == nil {
		t.Fatal("want error for a recorded file blocked by a directory")
	}
	if got := read(t, s.Out, "world", "sub", "kept.txt"); got != "kept\n" {
		t.Errorf("nested world file %q", got)
	}
}
