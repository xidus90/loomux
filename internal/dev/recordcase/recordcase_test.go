package recordcase

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	case "env":
		for _, name := range args {
			value := os.Getenv(name)
			if name == "PATH" {
				value, _, _ = strings.Cut(value, string(os.PathListSeparator))
			}
			os.Stdout.WriteString(name + "=" + value + "\n")
		}
		os.Exit(0)
	case "stderr":
		os.Stderr.WriteString(strings.Join(args, " ") + "\n")
		os.Exit(0)
	case "crlf":
		os.Stdout.WriteString("one\r\ntwo\r\n")
		os.Exit(0)
	case "mcp":
		// The reference's two shapes: a daemon verb that only has to exit, and
		// the front, which speaks a whole session. helperMCP knows both.
		helperMCP(args)
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

// helperArgvSpec runs the test binary as a program with leading arguments:
// the program, -test.run and "--" stand before the command's own arguments,
// as "uv run --project <ub> brain-mcp" stands before "status".
func helperArgvSpec(t *testing.T, mode, cmd string) Spec {
	t.Helper()
	s := helperSpec(t, mode, cmd)
	s.Exe = ""
	s.Argv = []string{os.Args[0], "-test.run=TestHelperProcess", "--"}
	s.Cmd = "brain-mcp " + cmd
	return s
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
	if _, err := os.Stat(filepath.Join(s.Out, "stderr")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("stderr is written only when the run said something: %v", err)
	}
}

// The old hooks said everything on stderr, so a case keeps it as evidence --
// nothing compares it, and a deviation list has to be able to quote it.
func TestRecordKeepsStderrAsEvidence(t *testing.T) {
	s := helperSpec(t, "stderr", "gone wrong in {{WORLD}}")
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s.Out, "stderr"); got != "gone wrong in {{WORLD}}\n" {
		t.Errorf("stderr %q", got)
	}
}

// A world that declares commits is built before the run, and the repository
// the build made is the bench rather than the world: it stays out of the case.
func TestRecordBuildsTheGitWorldAndLeavesItOutOfWorldAfter(t *testing.T) {
	s := helperSpec(t, "write", "{{WORLD}}/made.txt")
	if err := os.WriteFile(filepath.Join(s.World, "git.toml"), []byte("[[commit]]\nmessage = \"base\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.World, "head.txt"), []byte("{{COMMIT:1}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Out, "world_after", ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("world_after/.git: %v", err)
	}
	if got := read(t, s.Out, "world_after", "head.txt"); len(got) != 40 {
		t.Errorf("token not replaced: %q", got)
	}
	if got := read(t, s.Out, "world", "head.txt"); got != "{{COMMIT:1}}" {
		t.Errorf("the recorded world keeps its declaration: %q", got)
	}
}

// A linked worktree carries .git as a file rather than a directory; the skip
// holds for both shapes.
func TestRecordLeavesAGitFileOutOfTheCorpus(t *testing.T) {
	s := helperSpec(t, "echo", "x")
	if err := os.WriteFile(filepath.Join(s.World, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Out, "world", ".git")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("world/.git: %v", err)
	}
}

// A git.toml git cannot make is the recorder's error, not a case.
func TestRecordReportsAGitWorldItCannotBuild(t *testing.T) {
	s := helperSpec(t, "echo", "x")
	if err := os.WriteFile(filepath.Join(s.World, "git.toml"), []byte("[[commit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Record(s); err == nil {
		t.Fatal("want error for a git.toml that does not parse")
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

// A command of blanks splits into no token at all, so there is no old name
// for the program to replace.
func TestRecordRefusesACommandWithoutTokens(t *testing.T) {
	s := helperSpec(t, "echo", "")
	s.Cmd = "   "
	if err := Record(s); err == nil || err.Error() != "empty command" {
		t.Fatalf("err %v", err)
	}
}

// The Python reference runs as "uv run --project <ub> brain-mcp", reads its
// state directory and a fake qmd from the environment, and finds that qmd
// through PATH. All three have to reach the recorded process.
func TestRecordRunsAProgramWithLeadingArgumentsInItsEnvironment(t *testing.T) {
	prepend := t.TempDir()
	s := helperArgvSpec(t, "env", "BRAIN_STATE_DIR PYTHONUTF8 LOOMUX_FAKE_QMD_FIXTURE PATH")
	s.Env = []string{"LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json"}
	s.PathPrepend = prepend
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	want := "BRAIN_STATE_DIR={{WORLD}}\nPYTHONUTF8=1\nLOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json\nPATH=" + prepend + "\n"
	if got := read(t, s.Out, "stdout"); got != want {
		t.Errorf("stdout %q, want %q", got, want)
	}
	if got := read(t, s.Out, "cmd"); got != "brain-mcp BRAIN_STATE_DIR PYTHONUTF8 LOOMUX_FAKE_QMD_FIXTURE PATH\n" {
		t.Errorf("cmd %q", got)
	}
}

// Python on Windows ends every printed line with \r\n in a pipe.
// The recorded process runs git under the environment the replay sets too, so
// the user's configuration cannot shape what the reference printed.
func TestRecordRunsTheProgramUnderTheReplaysGitEnvironment(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "")
	t.Setenv("HOME", t.TempDir())
	s := helperArgvSpec(t, "env", "GIT_CONFIG_NOSYSTEM HOME")
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	want := "GIT_CONFIG_NOSYSTEM=1\nHOME={{WORLD}}/.no-git-home\n"
	if got := read(t, s.Out, "stdout"); got != want {
		t.Errorf("stdout %q, want %q", got, want)
	}
}

func TestRecordFoldsCRLFInStdout(t *testing.T) {
	s := helperSpec(t, "crlf", "")
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s.Out, "stdout"); got != "one\ntwo\n" {
		t.Errorf("stdout %q", got)
	}
}

func TestRecordRefusesAProgramNamedTwiceOrNotAtAll(t *testing.T) {
	s := helperArgvSpec(t, "echo", "x")
	s.Exe = os.Args[0]
	if err := Record(s); err == nil {
		t.Error("Exe and Argv: want error")
	}
	s.Exe, s.Argv = "", nil
	if err := Record(s); err == nil {
		t.Error("neither Exe nor Argv: want error")
	}
}

func TestRecordRefusesAnEnvironmentEntryWithoutAValue(t *testing.T) {
	s := helperSpec(t, "echo", "x")
	s.Env = []string{"NOVALUE"}
	if err := Record(s); err == nil || !strings.Contains(err.Error(), `"NOVALUE"`) {
		t.Fatalf("err %v", err)
	}
}

func TestRecordNamesTheArgvProgramItCannotRun(t *testing.T) {
	s := helperArgvSpec(t, "echo", "x")
	s.Argv = []string{filepath.Join(t.TempDir(), "no-such-program.exe"), "run"}
	if err := Record(s); err == nil || !strings.Contains(err.Error(), "no-such-program.exe") {
		t.Fatalf("err %v", err)
	}
}

func TestMergeEnvReplacesAKeyAsThePlatformSpellsIt(t *testing.T) {
	base := []string{`Path=C:\a`, "X=1"}
	for _, c := range []struct {
		goos string
		env  []string
		set  []string
		want []string
	}{
		{"windows", base, []string{`PATH=C:\b`}, []string{"X=1", `PATH=C:\b`}},
		{"linux", []string{"Path=/a", "X=1"}, []string{"PATH=/b"}, []string{"Path=/a", "X=1", "PATH=/b"}},
		{"linux", []string{"X=1", "Y=2"}, []string{"X=3", "Z=4"}, []string{"Y=2", "X=3", "Z=4"}},
	} {
		if got := mergeEnv(c.goos, c.env, c.set...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %v + %v: got %v, want %v", c.goos, c.env, c.set, got, c.want)
		}
	}
	if !reflect.DeepEqual(base, []string{`Path=C:\a`, "X=1"}) {
		t.Errorf("the environment it was given changed: %v", base)
	}
}

func TestPrependPathPutsTheDirectoryFirst(t *testing.T) {
	for _, c := range []struct {
		goos string
		env  []string
		dir  string
		want []string
	}{
		{"windows", []string{`Path=C:\a;C:\b`, "X=1"}, `D:\fake`, []string{"X=1", `PATH=D:\fake;C:\a;C:\b`}},
		{"linux", []string{"PATH=/a:/b"}, "/fake", []string{"PATH=/fake:/a:/b"}},
		{"linux", []string{"Path=/a"}, "/fake", []string{"Path=/a", "PATH=/fake"}},
		{"linux", []string{"PATH="}, "/fake", []string{"PATH=/fake"}},
		{"windows", []string{"X=1"}, `D:\fake`, []string{"X=1", `PATH=D:\fake`}},
	} {
		if got := prependPath(c.goos, c.env, c.dir); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %v: got %v, want %v", c.goos, c.env, got, c.want)
		}
	}
}

// A recording of a command that commits carries git.after in world_after, so
// the replay compares the commit rather than the files alone.
func TestRecordWritesGitAfterWhenAsked(t *testing.T) {
	s := helperSpec(t, "echo", "x")
	s.GitAfter = true
	if err := os.WriteFile(filepath.Join(s.World, "git.toml"), []byte("dir = \"repo\"\n[[commit]]\nmessage = \"base\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Record(s); err != nil {
		t.Fatal(err)
	}
	got := read(t, s.Out, "world_after", "repo", "git.after")
	// An empty commit on a clean worktree: the subject and the identity, and
	// no path, diff or status.
	if got != "base\nloomux cases <cases@loomux.invalid> / loomux cases <cases@loomux.invalid>\n" {
		t.Errorf("git.after %q", got)
	}
}

func TestRecordRefusesGitAfterWithoutACommit(t *testing.T) {
	for name, decl := range map[string]string{"no git world": "", "no commit": "dir = \"repo\"\n"} {
		t.Run(name, func(t *testing.T) {
			s := helperSpec(t, "echo", "x")
			s.GitAfter = true
			if decl != "" {
				if err := os.WriteFile(filepath.Join(s.World, "git.toml"), []byte(decl), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := Record(s); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}
