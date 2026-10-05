package hooks

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow/runs"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/sessions"
)

// Nothing to announce writes no envelope at all, rather than one carrying an
// empty additionalContext -- hosts.WriteContext makes that decision in its
// Claude arm, and this only shows it reaching the hook's own output. Named
// rather than cited by line: the decision has already moved between files once.
func TestHookSessionStartIsSilentWithNothingWaiting(t *testing.T) {
	root := project(t)

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude", "3.3.0")

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
	SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude", "3.3.0")

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
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude", "3.3.0")

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
	SessionStart(strings.NewReader(`{"hook_event_name":"SessionStart"}`), &stdout, &stderr, root, "claude", "3.3.0")

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
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude", "3.3.0")

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
	code := SessionStart(strings.NewReader("{not json"), &stdout, &stderr, project(t), "claude", "3.3.0")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if strings.TrimSpace(stderr.String()) == "" {
		t.Fatal("a refusal says why")
	}
}

func TestHookSessionStartOnAnUnknownHost(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, project(t), "gemini-cli", "3.3.0")

	if code != ExitInternal {
		t.Fatalf("an unknown host is refused, got %d", code)
	}
}

// A host whose adapter is still a promise is refused when it writes context,
// and the refusal carries that host's own words.
func TestHookSessionStartOnAHostWithoutAnAdapter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	root, _ := pilot(t, time.Hour, 2*time.Hour)
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "codex", "3.3.0")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "codex") {
		t.Fatalf("the refusal names the host, got %q", stderr.String())
	}
}

// Antigravity encodes context warnings as injectSteps with an ephemeral message.
func TestHookSessionStartOnAntigravity(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, time.Minute)
	write(t, filepath.Join(root, ".loomux", "config.toml"), 3*time.Hour)
	runningAs(t, binary)

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"conversationId":"s1"}`), &stdout, &stderr, root, "antigravity", "3.3.0")

	if code != ExitOK {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "injectSteps") || !strings.Contains(stdout.String(), "ephemeralMessage") {
		t.Fatalf("expected injectSteps in stdout, got %q", stdout.String())
	}

	// PreInvocation fires before every model call; only the first announces.
	stdout.Reset()
	code = SessionStart(strings.NewReader(`{"conversationId":"s1","invocationNum":2}`), &stdout, &stderr, root, "antigravity", "3.3.0")
	if code != ExitOK || stdout.Len() != 0 {
		t.Fatalf("a later invocation: %d %q", code, stdout.String())
	}
}

// pilot is a project holding a binary and one source, each aged as asked.
func pilot(t *testing.T, binaryAge, sourceAge time.Duration) (root, binary string) {
	t.Helper()
	t.Setenv(config.StateDirEnv, t.TempDir())
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

// Only what is compiled into the binary counts: a .go file outside cmd/,
// internal/ and flows/ never reached the build.
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
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), refusingWriter{}, &stderr, root, "claude", "3.3.0")

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
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude", "3.3.0")

	if code != ExitOK {
		t.Fatalf("a warning does not fail the hook, got %d (%s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "internal/cli/cli.go") {
		t.Fatalf("the session is told which source is newer, got %q", stdout.String())
	}
}

// A waiting run is news on every start, not only the first: the question stays
// open until a human answers it. agy's later invocations are the starts the
// hook otherwise keeps quiet on.
func TestHookSessionStartAnnouncesAWaitingRunOnARepeatedStart(t *testing.T) {
	root := project(t)
	writeRun(t, root, "0001", runs.Marker{Flow: "example", Origin: "bundled"}, pausedLine("approve", "Ship it?"))
	flowFolder(t, root, "example/flow.toml")
	for _, payload := range []string{`{"conversationId":"s1"}`, `{"conversationId":"s1","invocationNum":2}`} {
		var stdout, stderr bytes.Buffer
		code := SessionStart(strings.NewReader(payload), &stdout, &stderr, root, "antigravity", "3.3.0")
		if code != ExitOK || !strings.Contains(stdout.String(), "run 0001 (example, bundled) is waiting at approve: Ship it?") ||
			!strings.Contains(stdout.String(), ".loomux/flows/example is ignored") {
			t.Fatalf("%s: %d %q %q", payload, code, stdout.String(), stderr.String())
		}
	}
}

// The binary compiles flows/ and embeds its catalog, so a flow edited after the
// build is as stale as a Go source.
func TestSessionStartWarnsWhenACatalogFileIsNewer(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, 3*time.Hour)
	write(t, filepath.Join(root, "flows", "catalog", "example", "flow.toml"), time.Minute)
	runningAs(t, binary)

	lines := staleBinary(root)

	if len(lines) != 1 || !strings.Contains(lines[0], "flows/catalog/example/flow.toml") {
		t.Fatalf("expected one line naming the catalog file, got %v", lines)
	}
}

// The embed directive leaves out every name that starts with "_" or ".", and
// with it a flow's _test/ folder: its script and golden journal never reach
// the binary.
func TestSessionStartIgnoresWhatTheCatalogDoesNotEmbed(t *testing.T) {
	root, binary := pilot(t, 2*time.Hour, 3*time.Hour)
	write(t, filepath.Join(root, "flows", "catalog", "example", "flow.toml"), 3*time.Hour)
	write(t, filepath.Join(root, "flows", "catalog", "example", "_test", "journal.jsonl"), time.Minute)
	write(t, filepath.Join(root, "flows", "catalog", "example", ".draft.toml"), time.Minute)
	runningAs(t, binary)

	if lines := staleBinary(root); len(lines) != 0 {
		t.Fatalf("expected no warning, got %v", lines)
	}
}

// project is a directory loomux recognises as a root. It also gives the test a
// state directory of its own: session start reads update.json from there, and
// this machine's would otherwise decide what the session hears.
func project(t *testing.T) string {
	t.Helper()
	t.Setenv(config.StateDirEnv, t.TempDir())
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
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:1}}","green":"`+goneSHA+`","blocks":0}`)
	base := stateOf(t, root).Base

	var stdout, stderr bytes.Buffer
	if code := SessionStart(strings.NewReader(`{"session_id":"s1","source":"compact"}`), &stdout, &stderr, root, "claude", "3.3.0"); code != ExitOK {
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

// A session whose end marker cannot be taken away is told so in its context,
// with exit 0: exit 1 would drop the context lines.
func TestHookSessionStartSaysAMarkerItCannotRemove(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:1}}","green":"`+goneSHA+`","blocks":0}`)
	busy := filepath.Join(root, filepath.FromSlash(sessions.StateDir), "s1.ended", "inside")
	if err := os.MkdirAll(busy, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1","source":"resume"}`), &stdout, &stderr, root, "claude", "3.3.0")
	if code != ExitOK || !strings.Contains(stdout.String(), "may not count for worktree unlink") || !strings.Contains(stdout.String(), "reviving ") {
		t.Fatalf("%d %q %q", code, stdout.String(), stderr.String())
	}
}

// Only the first PreInvocation revives, so only it says a marker it cannot
// take away: nothing retires an agy conversation between two model calls, and
// the first invocation already said it. invocationNum is 0 on the first model
// call; 2 is a later one whichever count the host keeps.
func TestHookSessionStartSaysAMarkerItCannotRemoveOnlyOnTheFirstInvocation(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		said          bool
	}{
		{"first", `{"conversationId":"s1","invocationNum":0}`, true},
		{"later", `{"conversationId":"s1","invocationNum":2}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(config.StateDirEnv, t.TempDir())
			root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:1}}","green":"`+goneSHA+`","blocks":0}`)
			busy := filepath.Join(root, filepath.FromSlash(sessions.StateDir), "s1.ended", "inside")
			if err := os.MkdirAll(busy, 0o755); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := SessionStart(strings.NewReader(tc.payload), &stdout, &stderr, root, "antigravity", "3.3.0")
			said := strings.Contains(stdout.String(), "injectSteps") && strings.Contains(stdout.String(), "may not count for worktree unlink")
			if code != ExitOK || said != tc.said || (!tc.said && stdout.Len() != 0) {
				t.Fatalf("%d %q %q", code, stdout.String(), stderr.String())
			}
		})
	}
}

// The session counts again before its base is filed, so a base that cannot be
// written leaves it counted all the same. A directory where the session's file
// belongs makes the write fail; Revive's own touch of that path still lands,
// and it is how the order shows.
func TestHookSessionStartRevivesBeforeABaseItCannotWrite(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := project(t)
	gitInit(t, root)
	file := filepath.Join(root, filepath.FromSlash(sessions.StateDir), "s1.json")
	if err := os.MkdirAll(file, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := SessionStart(strings.NewReader(`{"session_id":"s1"}`), &stdout, &stderr, root, "claude", "3.3.0")

	if code != ExitInternal {
		t.Fatalf("expected exit 1, got %d; stderr: %s", code, stderr.String())
	}
	info, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if info.ModTime().Before(time.Now().Add(-time.Hour)) {
		t.Fatalf("the session was not made young before the base failed: %s", info.ModTime())
	}
}

// A session that worktree unlink retired and that then resumes under the same
// id counts again for the others, and still has its base.
func TestHookSessionStartRevivesARetiredSession(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	root := gitWorld(t, twoCommitsOnly, `{"base":"{{COMMIT:1}}","green":"`+goneSHA+`","blocks":0}`)
	base := stateOf(t, root).Base
	if err := sessions.Retire(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if n, err := sessions.Others(root, "s2", time.Hour); err != nil || n != 0 {
		t.Fatalf("others before the resume = %d, %v", n, err)
	}

	var stdout, stderr bytes.Buffer
	if code := SessionStart(strings.NewReader(`{"session_id":"s1","source":"resume"}`), &stdout, &stderr, root, "claude", "3.3.0"); code != ExitOK {
		t.Fatalf("%d %q", code, stderr.String())
	}
	if n, err := sessions.Others(root, "s2", time.Hour); err != nil || n != 1 {
		t.Fatalf("others after the resume = %d, %v; the resumed session does not count", n, err)
	}
	if state := stateOf(t, root); state.Base != base {
		t.Fatalf("the base moved from %s to %s", base, state.Base)
	}
}

// canonicalIn is a state directory whose canonical binary exists, so that
// IsCanonical has a file to compare the recorded executable with.
func canonicalIn(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exe := selfupdate.Canonical(dir)
	if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestUpdateWarnings(t *testing.T) {
	at := time.Date(2026, 9, 24, 8, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name string
		goos string
		// windowsOnly marks a row that needs the file system of Windows.
		windowsOnly bool
		status      func(dir string) *selfupdate.Status
		want        []string
	}{
		{"no pass yet", "windows", false, func(string) *selfupdate.Status { return nil }, nil},
		{"all well", "windows", false, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: selfupdate.Canonical(dir), Result: selfupdate.Current}
		}, nil},
		{"the canonical path in other letters", "windows", true, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: strings.ToUpper(selfupdate.Canonical(dir)), Result: selfupdate.Current}
		}, nil},
		{"serve from a checkout", "windows", false, func(string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: `C:\repo\bin\loomux.exe`, Result: selfupdate.Skipped}
		}, []string{`loomux serve runs from C:\repo\bin\loomux.exe, not from `}},
		{"serve off Windows", "linux", false, func(string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: "/usr/local/bin/loomux", Result: selfupdate.Skipped}
		}, nil},
		{"a pass by hand from a checkout", "windows", false, func(string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceCLI, Executable: `C:\repo\bin\loomux.exe`, Result: selfupdate.Skipped}
		}, nil},
		{"a failed pass", "windows", false, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Executable: selfupdate.Canonical(dir), Result: selfupdate.Failed, CheckedAt: at, Error: "gh not found"}
		}, []string{"updating loomux failed at 2026-09-24T08:00:00Z: gh not found; run loomux upgrade to retry"}},
		{"a pass that kept the binary but not the channel", "windows", false, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: selfupdate.Canonical(dir), Result: selfupdate.Current, CheckedAt: at, Error: `channel file holds "nightly", not beta`}
		}, []string{`loomux update channel at 2026-09-24T08:00:00Z: channel file holds "nightly", not beta; run loomux upgrade --beta or --stable to set it`}},
		{"an updated pass that could not write the channel", "windows", false, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: selfupdate.Canonical(dir), Result: selfupdate.Updated, CheckedAt: at, Error: "open channel: denied"}
		}, []string{"loomux update channel at 2026-09-24T08:00:00Z: open channel: denied; run loomux upgrade --beta or --stable to set it"}},
		{"a skipped pass says why without a warning", "windows", false, func(dir string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceCLI, Executable: `C:\repo\bin\loomux.exe`, Result: selfupdate.Skipped, Error: "running from C:\\repo"}
		}, nil},
		{"a failed pass off Windows", "linux", false, func(string) *selfupdate.Status {
			return &selfupdate.Status{Source: selfupdate.SourceServe, Executable: "/usr/local/bin/loomux", Result: selfupdate.Failed, CheckedAt: at, Error: "gh not found"}
		}, []string{"updating loomux failed at 2026-09-24T08:00:00Z: gh not found; run loomux upgrade to retry"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.windowsOnly && runtime.GOOS != "windows" {
				t.Skip("only Windows spells one file in other letters")
			}
			dir := canonicalIn(t)
			if st := c.status(dir); st != nil {
				if err := selfupdate.WriteStatus(dir, *st); err != nil {
					t.Fatal(err)
				}
			}
			got := updateWarnings(dir, c.goos)
			if len(got) != len(c.want) {
				t.Fatalf("updateWarnings = %v, want %v", got, c.want)
			}
			for i := range c.want {
				if !strings.HasPrefix(got[i], c.want[i]) {
					t.Fatalf("line %d = %q, want prefix %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestUpdateWarningsNamesAnUnreadableStatus(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(selfupdate.StatusPath(dir), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := updateWarnings(dir, "windows")
	if len(got) != 1 || !strings.HasPrefix(got[0], "loomux cannot read the update status: ") {
		t.Fatalf("updateWarnings = %v", got)
	}
}
