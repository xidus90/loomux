package hooks

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/sessions"
)

// Nothing to announce writes no envelope at all, rather than one carrying an
// empty additionalContext -- hosts.WriteContext makes that decision in its
// Claude arm, and this only shows it reaching the hook's own output. Named
// rather than cited by line: the decision has already moved between files once.
func TestHookSessionStartIsSilentWithNothingWaiting(t *testing.T) {
	root := project(t)

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitOK || stdout.Len() != 0 {
		t.Fatalf("expected exit 0 and no output, got %d and %q", code, stdout.String())
	}
}

// The base commit is written here and nowhere else: by the time the first Stop
// fires, the turn has already run, and anything it committed would sit inside
// the baseline that is supposed to expose it.
//
// Read back through the package's own door rather than by parsing the file
// again: a second reader here would pass while ReadState was broken.
func TestHookSessionStartRecordsTheBaseCommit(t *testing.T) {
	root := project(t)
	gitInit(t, root)

	var stdout, stderr bytes.Buffer
	SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if state := sessions.ReadState(root, "s1"); len(state.Base) != 40 {
		t.Fatalf("expected a full sha as the base, got %q", state.Base)
	}
}

// Silent in both failure cases, as session_start.py (fa3dd38):43-59's
// _record_base was:
// without a session id there is nowhere to file it, and outside a repository
// there is nothing to file. Neither is a defect of the project, and neither is
// worth a line in every session of every checkout that is not a repository.
// The stop gate is where the absence matters and where it is said out loud.
func TestHookSessionStartRecordsNoBaseWithoutARepository(t *testing.T) {
	root := project(t)

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitOK {
		t.Fatalf("a checkout that is not a repository is not a failure, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("and it is not worth a word, got %q", stderr.String())
	}
}

func TestHookSessionStartRecordsNoBaseWithoutASessionID(t *testing.T) {
	root := project(t)
	gitInit(t, root)

	var stdout, stderr bytes.Buffer
	SessionStart(strings.NewReader(`{"hook_event_name":"SessionStart"}`), &stdout, &stderr, root, "claude")

	dir := filepath.Join(root, filepath.FromSlash(sessions.StateDir))
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		t.Fatalf("no session id means nowhere to file it, got %d files", len(entries))
	}
}

// A base that cannot be written is said out loud and ends the call with exit
// 1, unlike the two absences above. state.py:56-65's `write` catches nothing,
// so an OSError there leaves the Python `run` by itself; keeping quiet here
// would be a departure from the original rather than the parity the two
// silences are.
func TestHookSessionStartReportsABaseItCannotWrite(t *testing.T) {
	root := project(t)
	gitInit(t, root)
	// A file where the state directory belongs, so creating it must fail.
	stateDir := filepath.Join(root, filepath.FromSlash(sessions.StateDir))
	if err := os.MkdirAll(filepath.Dir(stateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateDir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "hooks") {
		t.Fatalf("the refusal names the path it could not write, got %q", stderr.String())
	}
}

// payload.py's exit protocol: 1 for an internal failure, and never 2 -- this
// hook is an announcement and has nothing to block.
func TestHookSessionStartOnABadPayload(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader("{not json"), &stdout, &stderr, project(t), "claude")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if strings.TrimSpace(stderr.String()) == "" {
		t.Fatal("a refusal says why")
	}
}

func TestHookSessionStartOnAnUnknownHost(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, project(t), "gemini-cli")

	if code != ExitInternal {
		t.Fatalf("an unknown host is refused, got %d", code)
	}
}

// A host whose adapter is still a promise is refused where it is first asked a
// question -- hosts.Read -- and the refusal carries that host's own words.
func TestHookSessionStartOnAHostWithoutAnAdapter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, project(t), "antigravity")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "antigravity") {
		t.Fatalf("the refusal names the host, got %q", stderr.String())
	}
}

// pilot is a project holding a binary and one source, each aged as asked.
func pilot(t *testing.T, binaryAge, sourceAge time.Duration) (root, binary string) {
	t.Helper()
	root = t.TempDir()
	binary = filepath.Join(root, "bin", "loomux.exe")
	source := filepath.Join(root, "internal", "cli", "cli.go")
	for path, age := range map[string]time.Duration{binary: binaryAge, source: sourceAge} {
		write(t, path, age)
	}
	return root, binary
}

// write puts a file in place and dates it `age` into the past.
func write(t *testing.T, path string, age time.Duration) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().Add(-age)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatal(err)
	}
}

// runningAs points the stale check at a binary of the test's choosing: in a
// test os.Executable names the test binary in a temp directory of its own,
// which lies outside any project, so nothing would ever be compared.
func runningAs(t *testing.T, path string) {
	t.Helper()
	executable = func() (string, error) { return path, nil }
	t.Cleanup(func() { executable = os.Executable })
}

// The comparison is against the sources, deliberately not against HEAD: the
// pre-commit gate builds the binary before the commit exists, so a HEAD
// comparison would warn after every single commit.
func TestSessionStartWarnsWhenASourceIsNewerThanThePilotBinary(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	runningAs(t, binary)

	lines := staleBinary(root)

	if len(lines) != 1 || !strings.Contains(lines[0], "internal/cli/cli.go") {
		t.Fatalf("expected one line naming the newer source, got %v", lines)
	}
	if !strings.Contains(lines[0], "bin/loomux.exe") {
		t.Fatalf("the warning names the binary it is about, got %v", lines)
	}
}

// go.mod and go.sum decide the build as much as the .go files do.
func TestSessionStartWarnsWhenTheModuleFilesAreNewer(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, 3*time.Hour)
	write(t, filepath.Join(root, "go.sum"), 3*time.Hour)
	write(t, filepath.Join(root, "go.mod"), time.Minute)
	runningAs(t, binary)

	lines := staleBinary(root)

	if len(lines) != 1 || !strings.Contains(lines[0], "go.mod") {
		t.Fatalf("expected one line naming go.mod, got %v", lines)
	}
}

func TestSessionStartIsQuietWhenTheBinaryIsNewest(t *testing.T) {
	root, binary := pilot(t, time.Minute, 2*time.Hour)
	runningAs(t, binary)

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

// Only what is compiled into the binary counts: a .go file outside cmd/ and
// internal/ never reached the build.
func TestSessionStartIgnoresGoFilesOutsideTheBuiltDirectories(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "bin", "loomux.exe")
	write(t, binary, 2*time.Hour)
	write(t, filepath.Join(root, "internal", "cli", "cli.go"), 3*time.Hour)
	write(t, filepath.Join(root, "internal", "cli", "notes.md"), 3*time.Hour)
	write(t, filepath.Join(root, "testdata", "x.go"), time.Minute)
	runningAs(t, binary)

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

func TestSessionStartSaysNothingAboutABinaryOutsideTheProject(t *testing.T) {
	root, _ := pilot(t, 2*time.Hour, time.Minute)
	runningAs(t, filepath.Join(t.TempDir(), "loomux.exe"))

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("a binary outside the project is none of the project's business, got %v", lines)
	}
}

func TestSessionStartSaysNothingWhenTheRunningBinaryCannotBeNamed(t *testing.T) {
	root, _ := pilot(t, 2*time.Hour, time.Minute)
	executable = func() (string, error) { return "", errors.New("no executable") }
	t.Cleanup(func() { executable = os.Executable })

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

func TestSessionStartSaysNothingWhenTheRootCannotBeResolved(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	runningAs(t, binary)
	absPath = func(string) (string, error) { return "", errors.New("no cwd") }
	t.Cleanup(func() { absPath = filepath.Abs })

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

func TestSessionStartSaysNothingAboutABinaryThatIsNotThere(t *testing.T) {
	root, _ := pilot(t, 2*time.Hour, time.Minute)
	runningAs(t, filepath.Join(root, "bin", "gone.exe"))

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

// A directory that will not be read leaves the age of the sources unknown, and
// an unknown age is no reason to tell the session anything.
func TestSessionStartSaysNothingWhenASourceDirectoryCannotBeRead(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	runningAs(t, binary)
	walkDir = func(string, fs.WalkDirFunc) error { return errors.New("permission denied") }
	t.Cleanup(func() { walkDir = filepath.WalkDir })

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

func TestSessionStartSaysNothingWhenAWalkHandsUpAnError(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	runningAs(t, binary)
	walkDir = func(dir string, fn fs.WalkDirFunc) error {
		return fn(dir, nil, errors.New("permission denied"))
	}
	t.Cleanup(func() { walkDir = filepath.WalkDir })

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

// A file that disappears between the walk and the question about its age: the
// walk hands up the entry, and the entry can no longer answer.
func TestSessionStartSaysNothingWhenASourceLosesItsAgeMidWalk(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	runningAs(t, binary)
	walkDir = func(dir string, fn fs.WalkDirFunc) error {
		return fn(filepath.Join(dir, "vanished.go"), vanishing{}, nil)
	}
	t.Cleanup(func() { walkDir = filepath.WalkDir })

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

// vanishing is a walk entry for a file that is gone by the time it is asked
// how old it is.
type vanishing struct{ fs.DirEntry }

func (vanishing) IsDir() bool                { return false }
func (vanishing) Info() (fs.FileInfo, error) { return nil, fs.ErrNotExist }

// The warning travels the same way as anything else the hook has to say, so a
// stdout that will not take it ends the call with exit 1.
func TestHookSessionStartReportsAWarningItCannotWrite(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	write(t, filepath.Join(root, ".loomux", "config.toml"), 3*time.Hour)
	runningAs(t, binary)

	var stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), refusingWriter{}, &stderr, root, "claude")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if strings.TrimSpace(stderr.String()) == "" {
		t.Fatal("a refusal says why")
	}
}

// refusingWriter is a stdout that takes nothing.
type refusingWriter struct{}

func (refusingWriter) Write([]byte) (int, error) { return 0, errors.New("closed") }

// The session hears about a stale binary through the hook itself, not only
// through staleBinary: a pilot hook judges with yesterday's rules until it is
// rebuilt.
func TestHookSessionStartAnnouncesAStaleBinary(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	write(t, filepath.Join(root, ".loomux", "config.toml"), 3*time.Hour)
	runningAs(t, binary)

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude")

	if code != ExitOK {
		t.Fatalf("a warning does not fail the hook, got %d (%s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "internal/cli/cli.go") {
		t.Fatalf("the session is told which source is newer, got %q", stdout.String())
	}
}

// project is a directory loomux recognises as a root.
func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func gitInit(t *testing.T, root string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "t@example.invalid"},
		{"config", "user.name", "Test"},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = gitenv.Environ()
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "a.txt"}, {"commit", "-m", "first"}} {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = gitenv.Environ()
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// SessionStart fires on resume, clear and compact under the same session id,
// not only at startup. A base it moved to HEAD there would swallow what the
// session committed since its last green run: the stop gate would measure
// from that commit, find nothing new, and end a red turn unchecked.
func TestHookSessionStartKeepsABaseTheSessionAlreadyHas(t *testing.T) {
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:1}}","green":"`+goneSHA+`","blocks":0}`)
	base := stateOf(t, root).Base

	var stdout, stderr bytes.Buffer
	if code := SessionStart(strings.NewReader(`{"session_id":"s1","source":"compact"}`), &stdout, &stderr, root, "claude"); code != ExitOK {
		t.Fatalf("%d %q", code, stderr.String())
	}

	if state := stateOf(t, root); state.Base != base {
		t.Fatalf("the base moved from %s to %s", base, state.Base)
	}
	env, started := countTools(redVet())
	if code, se := runStop(t, root, s1, env); code != ExitDenied || started.Load() == 0 {
		t.Fatalf("the commit since the base went unchecked: %d %q, %d tools started", code, se, started.Load())
	}
}
