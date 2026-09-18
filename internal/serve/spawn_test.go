package serve_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/serve"
)

// keepCmd is a spawner that starts nothing and keeps what it was handed. It is
// the whole reason Spawn takes a spawner: the process attributes and the
// redirected stdio can be read off the command without a process existing.
func keepCmd(kept *[]*exec.Cmd) func(*exec.Cmd) error {
	return func(cmd *exec.Cmd) error {
		*kept = append(*kept, cmd)
		return nil
	}
}

func TestSpawnNeverInheritsStdio(t *testing.T) {
	dir := t.TempDir()
	var kept []*exec.Cmd
	brokeAway, err := serve.Spawn(dir, keepCmd(&kept))
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if !brokeAway {
		t.Error("the first attempt carried the breakaway flag and succeeded, so brokeAway must be true")
	}
	if len(kept) != 1 {
		t.Fatalf("the spawner was called %d times, want once", len(kept))
	}
	got := kept[0]
	// nil means os.DevNull. The caller's stdout may be a host's MCP pipe; an
	// inherited descriptor would wreck its framing and hold it open.
	if got.Stdin != nil || got.Stdout != nil {
		t.Error("the child inherited stdin or stdout")
	}
	// The child is the service itself, so it is told to stay in the foreground.
	// Without that flag it takes the detached path again, spawns a child of its
	// own and so on: a bare `loomux serve` is the very command that spawns.
	if len(got.Args) < 3 || got.Args[1] != "serve" || got.Args[2] != "--foreground" {
		t.Errorf("args are %v, want the serve subcommand in the foreground", got.Args)
	}
	if got.SysProcAttr == nil {
		t.Error("the child was built without detach attributes")
	}
	// stderr is the one descriptor that goes somewhere, and it goes to a file
	// rather than to any other writer: exec builds a pipe and a copying
	// goroutine for everything that is not an *os.File, and only Wait reaps
	// those. Spawn never waits.
	file, ok := got.Stderr.(*os.File)
	if !ok {
		t.Fatalf("stderr is %T, want an *os.File so that exec needs no pipe behind it", got.Stderr)
	}
	if file.Name() != serve.LogPath(dir) {
		t.Errorf("stderr is %q, want %q", file.Name(), serve.LogPath(dir))
	}
	// LogPath names the file, it does not make its directory.
	if _, err := os.Stat(filepath.Dir(serve.LogPath(dir))); err != nil {
		t.Errorf("the log directory was not created: %v", err)
	}
}

func TestSpawnTellsTheChildWhereItsStateIs(t *testing.T) {
	dir := t.TempDir()
	// A stale value in the parent must not reach the child: the state
	// directory Spawn was given is the only one that counts.
	t.Setenv("LOOMUX_STATE_DIR", filepath.Join(dir, "somewhere-else"))
	var kept []*exec.Cmd
	if _, err := serve.Spawn(dir, keepCmd(&kept)); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	env := kept[0].Env
	if want := "LOOMUX_STATE_DIR=" + dir; !contains(env, want) {
		t.Errorf("the child environment is missing %q", want)
	}
	if count := countPrefix(env, "LOOMUX_STATE_DIR="); count != 1 {
		t.Errorf("LOOMUX_STATE_DIR appears %d times, want once", count)
	}
	// Spawn knows which attempt succeeded; the child writes serve.json, so the
	// answer has to travel to it.
	if want := serve.BrokeAwayEnv + "=1"; !contains(env, want) {
		t.Errorf("the child environment is missing %q", want)
	}
	if kept[0].Dir != dir {
		t.Errorf("the child runs in %q, want the state directory %q", kept[0].Dir, dir)
	}
}

func TestSpawnFallsBackWhenBreakawayIsRefused(t *testing.T) {
	calls := 0
	var kept []*exec.Cmd
	brokeAway, err := serve.Spawn(t.TempDir(), func(cmd *exec.Cmd) error {
		calls++
		kept = append(kept, cmd)
		if calls == 1 {
			return errors.New("access is denied")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if calls != 2 {
		t.Errorf("the spawner was called %d times, want two: with and without breakaway", calls)
	}
	if brokeAway {
		t.Error("brokeAway must be false after the fallback")
	}
	// The child must not be told it broke away when it did not: serve.json
	// would then claim a service outlives its host that dies with it.
	if contains(kept[1].Env, serve.BrokeAwayEnv+"=1") {
		t.Error("the second attempt still tells the child it broke away")
	}
}

func TestSpawnReportsAStartThatFailsBothWays(t *testing.T) {
	calls := 0
	brokeAway, err := serve.Spawn(t.TempDir(), func(*exec.Cmd) error {
		calls++
		if calls == 1 {
			return errors.New("access is denied")
		}
		return errors.New("no such file")
	})
	if err == nil {
		t.Fatal("Spawn reported success although no attempt started anything")
	}
	if !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Spawn: %v, want the second attempt's own error", err)
	}
	// Both failures, not only the last one: whether a host puts us into a job
	// object is the one assumption of this stage that nobody has measured, and
	// the first attempt's error is the only evidence there is about it.
	if !strings.Contains(err.Error(), "access is denied") {
		t.Errorf("Spawn: %v, want the first attempt's error kept as evidence", err)
	}
	if calls != 2 {
		t.Errorf("the spawner was called %d times, want two", calls)
	}
	if brokeAway {
		t.Error("brokeAway must be false when nothing started")
	}
}

func TestSpawnMakesARelativeStateDirectoryAbsolute(t *testing.T) {
	// The child runs in the state directory and inherits its path. Relative,
	// the child would resolve that path against its new working directory --
	// the directory itself -- and write its state one level deeper than the
	// caller waiting for serve.json ever looks.
	t.Chdir(t.TempDir())
	// Asked back rather than remembered: a temporary directory may be reached
	// through a link or a shortened name, and only what the process itself
	// calls its working directory is what filepath.Abs resolves against.
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	var kept []*exec.Cmd
	if _, err := serve.Spawn("state", keepCmd(&kept)); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	absolute := filepath.Join(root, "state")
	if kept[0].Dir != absolute {
		t.Errorf("the child runs in %q, want %q", kept[0].Dir, absolute)
	}
	if want := "LOOMUX_STATE_DIR=" + absolute; !contains(kept[0].Env, want) {
		t.Errorf("the child environment is missing %q; it has %v", want, stateDirEntries(kept[0].Env))
	}
	// The log has to land beside that same state, not beside the caller's.
	if _, err := os.Stat(serve.LogPath(absolute)); err != nil {
		t.Errorf("no log under the absolute state directory: %v", err)
	}
}

func TestSpawnRefusesWhenTheLogDirectoryCannotBeMade(t *testing.T) {
	dir := t.TempDir()
	// A regular file where logs/ belongs: MkdirAll cannot pass through it.
	write(t, filepath.Dir(serve.LogPath(dir)), "not a directory")
	_, err := serve.Spawn(dir, func(*exec.Cmd) error {
		t.Error("the spawner ran although the log has nowhere to go")
		return nil
	})
	if err == nil {
		t.Fatal("Spawn reported success although the log directory could not be made")
	}
}

func TestSpawnRefusesWhenTheLogFileCannotBeOpened(t *testing.T) {
	dir := t.TempDir()
	// A directory where serve.log belongs: OpenFile cannot write into it.
	if err := os.MkdirAll(serve.LogPath(dir), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	_, err := serve.Spawn(dir, func(*exec.Cmd) error {
		t.Error("the spawner ran although the log file could not be opened")
		return nil
	})
	if err == nil {
		t.Fatal("Spawn reported success although the log file could not be opened")
	}
}

func TestBrokeAwayFromEnvReadsWhatSpawnWrote(t *testing.T) {
	t.Setenv(serve.BrokeAwayEnv, "1")
	if !serve.BrokeAwayFromEnv() {
		t.Error("the child does not read back the breakaway Spawn recorded")
	}
	// Anything but the one value Spawn writes means no breakaway: a leftover
	// "0" or "false" must not read as yes.
	for _, value := range []string{"", "0", "false", "yes"} {
		t.Setenv(serve.BrokeAwayEnv, value)
		if serve.BrokeAwayFromEnv() {
			t.Errorf("%q reads as a breakaway", value)
		}
	}
}

// contains reports whether the environment carries exactly this entry.
func contains(env []string, want string) bool {
	for _, entry := range env {
		if entry == want {
			return true
		}
	}
	return false
}

// countPrefix counts the entries that set this variable, whatever the value.
func countPrefix(env []string, prefix string) int {
	count := 0
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			count++
		}
	}
	return count
}

// stateDirEntries is what an environment says about the state directory, for a
// failure message that shows what was there instead.
func stateDirEntries(env []string) []string {
	var found []string
	for _, entry := range env {
		if strings.HasPrefix(entry, "LOOMUX_STATE_DIR=") {
			found = append(found, entry)
		}
	}
	return found
}

// write puts a regular file at path, making the parent directories.
func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}
