package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/serve"
)

// serveStateDir points the state directory at a fresh temporary one, so that
// no test of this file can reach the state directory of the machine it runs
// on -- where a real service may well be listening.
func serveStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(config.StateDirEnv, dir)
	return dir
}

// stubServeRun replaces the service with something that records what it was
// asked for and returns at once. Without it the command would bind two ports
// and block for the life of the suite.
func stubServeRun(t *testing.T, err error) *serve.Options {
	t.Helper()
	seen := &serve.Options{}
	saved := serveRun
	serveRun = func(_ context.Context, opts serve.Options) error {
		*seen = opts
		return err
	}
	t.Cleanup(func() { serveRun = saved })
	return seen
}

// spawnCall is one recorded detached start.
type spawnCall struct {
	stateDir string
	calls    int
}

// stubServeSpawn replaces the only real process start in this package.
func stubServeSpawn(t *testing.T, brokeAway bool, err error) *spawnCall {
	t.Helper()
	seen := &spawnCall{}
	saved := serveSpawn
	serveSpawn = func(stateDir string, _ func(*exec.Cmd) error) (bool, error) {
		seen.stateDir, seen.calls = stateDir, seen.calls+1
		return brokeAway, err
	}
	t.Cleanup(func() { serveSpawn = saved })
	return seen
}

// stubServeNotify takes the place of signal.NotifyContext. It answers a
// context of its own and reports whether the registration was given up again:
// a command that keeps it holds the process's interrupt handler for good, and
// in a test it would hold the suite's.
func stubServeNotify(t *testing.T) *bool {
	t.Helper()
	released := new(bool)
	saved := serveNotify
	serveNotify = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		return ctx, func() { *released = true; cancel() }
	}
	t.Cleanup(func() { serveNotify = saved })
	return released
}

func TestServeInTheForegroundRunsTheServiceHere(t *testing.T) {
	dir := serveStateDir(t)
	seen := stubServeRun(t, nil)
	released := stubServeNotify(t)
	// The opposite seam is poisoned rather than left real. Under exactly the
	// regression spawn.go guards against -- --foreground reaching the detached
	// branch -- an unpoisoned serveSpawn would start a real process whose
	// executable is this test binary, and detach `cli.test.exe serve
	// --foreground`. The detector must not be the thing that detonates.
	stubServeSpawn(t, false, errors.New("the detached path was taken"))

	code, out, errOut := run("serve", "--foreground")
	if !*released {
		t.Error("the foreground run kept the interrupt registration after it ended")
	}
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if seen.StateDir != dir || seen.RegistryDir != dir {
		t.Errorf("the service got %q and %q, want the state directory %q", seen.StateDir, seen.RegistryDir, dir)
	}
	if seen.LegacyDir != config.LegacyBrainDirUntilStage3() {
		t.Errorf("the service got legacy directory %q", seen.LegacyDir)
	}
	// The flag is what tells the service it is not the detached child, and it
	// is the only thing that gives serve.Options.Foreground a meaning.
	if !seen.Foreground {
		t.Error("the service was not told it runs in the foreground")
	}
}

// TestServeInTheForegroundReadsTheBreakawayFromTheEnvironment closes a gap the
// spawn leaves open: only the parent that spawned this service knows whether the
// breakaway held, and it says so in the environment. Read nowhere, every
// serve.json says false and `serve status` quietly stops telling a service
// that outlives its host from one that does not.
func TestServeInTheForegroundReadsTheBreakawayFromTheEnvironment(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{{"1", true}, {"0", false}, {"", false}} {
		serveStateDir(t)
		t.Setenv(serve.BrokeAwayEnv, tc.value)
		seen := stubServeRun(t, nil)
		stubServeSpawn(t, false, errors.New("the detached path was taken"))
		if code, _, _ := run("serve", "--foreground"); code != 0 {
			t.Fatalf("code %d", code)
		}
		if seen.BrokeAway != tc.want {
			t.Errorf("with %q the service was told BrokeAway=%v, want %v", tc.value, seen.BrokeAway, tc.want)
		}
	}
}

func TestServeInTheForegroundReportsAFailedStart(t *testing.T) {
	serveStateDir(t)
	stubServeRun(t, serve.ErrAlreadyRunning)
	stubServeSpawn(t, false, errors.New("the detached path was taken"))

	code, out, errOut := run("serve", "--foreground")
	if code != 1 || out != "" || !strings.Contains(errOut, serve.ErrAlreadyRunning.Error()) {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestServeWithoutTheFlagStartsDetached(t *testing.T) {
	dir := serveStateDir(t)
	seen := stubServeSpawn(t, true, nil)
	// A service run here as well would mean the flag decides nothing.
	stubServeRun(t, errors.New("the foreground path was taken"))

	code, out, errOut := run("serve")
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if seen.calls != 1 || seen.stateDir != dir {
		t.Fatalf("the spawner was called %d times with %q", seen.calls, seen.stateDir)
	}
	if !strings.Contains(out, serve.LogPath(dir)) {
		t.Errorf("the start did not say where the service writes: %q", out)
	}
	if strings.Contains(out, "breakaway") {
		t.Errorf("the breakaway held, so nothing is to be warned about: %q", out)
	}
}

func TestADetachedStartWithoutBreakawaySaysSo(t *testing.T) {
	serveStateDir(t)
	stubServeSpawn(t, false, nil)

	code, out, _ := run("serve")
	if code != 0 || !strings.Contains(out, "dies with its host") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestADetachedStartReportsItsFailure(t *testing.T) {
	serveStateDir(t)
	stubServeSpawn(t, false, errors.New("no process for you"))

	code, out, errOut := run("serve")
	if code != 1 || out != "" || !strings.Contains(errOut, "no process for you") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

// TestADetachedStartIsRefusedWhileTheLockIsHeld keeps the answer where the
// user is. Without it the second `loomux serve` succeeds, and the refusal
// lands in a log file nobody opens.
func TestADetachedStartIsRefusedWhileTheLockIsHeld(t *testing.T) {
	dir := serveStateDir(t)
	handle, held, err := lock.TryAcquire(serve.LockPath(dir))
	if err != nil || !held {
		t.Fatalf("TryAcquire: %v, held %v", err, held)
	}
	defer handle.Release()
	seen := stubServeSpawn(t, true, nil)

	code, out, errOut := run("serve")
	if code != 1 || out != "" || !strings.Contains(errOut, serve.ErrAlreadyRunning.Error()) {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if seen.calls != 0 {
		t.Errorf("a second service was spawned although the lock is held")
	}
}

// TestADetachedStartProceedsWhenTheLockCannotBeOpened is the fresh machine:
// nothing of the state directory exists yet, the lock says nothing, and the
// service is the one that makes the directory.
func TestADetachedStartProceedsWhenTheLockCannotBeOpened(t *testing.T) {
	root := t.TempDir()
	t.Setenv(config.StateDirEnv, filepath.Join(root, "nothing", "here"))
	seen := stubServeSpawn(t, true, nil)

	if code, _, errOut := run("serve"); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if seen.calls != 1 {
		t.Errorf("the spawner was called %d times, want once", seen.calls)
	}
}

func TestServeRefusesAnUnknownSubcommand(t *testing.T) {
	serveStateDir(t)
	code, out, errOut := run("serve", "stat")
	if code != 2 || out != "" {
		t.Fatalf("code %d, out %q", code, out)
	}
	want := serveUsage() + "\nloomux serve: error: argument command: invalid choice: 'stat' (choose from 'status', 'stop')\n"
	if errOut != want {
		t.Fatalf("err %q, want %q", errOut, want)
	}
}

func TestServeRefusesAnUnknownFlag(t *testing.T) {
	serveStateDir(t)
	code, _, errOut := run("serve", "--background")
	if code != 2 || !strings.Contains(errOut, "unrecognized arguments: --background") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// stubServeStatus and stubServeStop keep the two control verbs away from a
// real listener.
func stubServeStatus(t *testing.T, report string, err error) *string {
	t.Helper()
	seen := new(string)
	saved := serveStatus
	serveStatus = func(stateDir string) (string, error) {
		*seen = stateDir
		return report, err
	}
	t.Cleanup(func() { serveStatus = saved })
	return seen
}

func TestServeStatusPrintsTheReport(t *testing.T) {
	dir := serveStateDir(t)
	seen := stubServeStatus(t, "loomux serve is running\n", nil)

	code, out, errOut := run("serve", "status")
	if code != 0 || out != "loomux serve is running\n" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if *seen != dir {
		t.Errorf("status asked %q, want %q", *seen, dir)
	}
}

func TestServeStatusReportsAnUnreadableState(t *testing.T) {
	serveStateDir(t)
	stubServeStatus(t, "", errors.New("serve.json is not JSON"))

	code, out, errOut := run("serve", "status")
	if code != 1 || out != "" || !strings.Contains(errOut, "serve.json is not JSON") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestServeStatusTakesNoArguments(t *testing.T) {
	serveStateDir(t)
	code, _, errOut := run("serve", "status", "--force")
	if code != 2 || !strings.Contains(errOut, "unrecognized arguments: --force") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// stopCall is one recorded stop, with the flag that decides whether a PID is
// killed.
type stopCall struct {
	stateDir string
	force    bool
	calls    int
}

func stubServeStop(t *testing.T, err error) *stopCall {
	t.Helper()
	seen := &stopCall{}
	saved := serveStop
	serveStop = func(stateDir string, force bool) error {
		seen.stateDir, seen.force, seen.calls = stateDir, force, seen.calls+1
		return err
	}
	t.Cleanup(func() { serveStop = saved })
	return seen
}

func TestServeStopIsSilentWhenItWorked(t *testing.T) {
	dir := serveStateDir(t)
	seen := stubServeStop(t, nil)

	code, out, errOut := run("serve", "stop")
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if seen.stateDir != dir || seen.force {
		t.Errorf("stop asked %q with force %v", seen.stateDir, seen.force)
	}
}

// TestServeStopForcePassesTheFlagOn: without --force the PID in serve.json is
// never killed, and a stop command that swallowed the flag would look exactly
// the same from outside.
func TestServeStopForcePassesTheFlagOn(t *testing.T) {
	serveStateDir(t)
	seen := stubServeStop(t, nil)

	if code, _, errOut := run("serve", "stop", "--force"); code != 0 {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if !seen.force {
		t.Error("--force did not reach the stop")
	}
}

func TestServeStopSaysSoWhenNothingIsRunning(t *testing.T) {
	serveStateDir(t)
	stubServeStop(t, os.ErrNotExist)

	code, out, errOut := run("serve", "stop")
	if code != 0 || out != "" || !strings.Contains(errOut, "nothing is running") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

// The second stop in a row: serve.Run leaves serve.json behind, so the file is
// read and the service it names is gone. That is the same "nothing is running"
// as a missing file, and not a reason to advise --force.
func TestServeStopSaysSoWhenTheStateFileOutlivedItsService(t *testing.T) {
	serveStateDir(t)
	stubServeStop(t, serve.ErrNotRunning)

	code, out, errOut := run("serve", "stop")
	if code != 0 || out != "" || !strings.Contains(errOut, "nothing is running") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestServeStopReportsARefusedStop(t *testing.T) {
	serveStateDir(t)
	stubServeStop(t, errors.New("the service did not answer"))

	code, _, errOut := run("serve", "stop")
	if code != 1 || !strings.Contains(errOut, "the service did not answer") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestServeStopRefusesAnUnknownFlag(t *testing.T) {
	serveStateDir(t)
	seen := stubServeStop(t, nil)

	code, _, errOut := run("serve", "stop", "--hard")
	want := serveStopUsage() + "\nloomux serve stop: error: unrecognized arguments: --hard\n"
	if code != 2 || errOut != want {
		t.Fatalf("code %d, err %q, want %q", code, errOut, want)
	}
	if seen.calls != 0 {
		t.Error("a refused stop still reached the service")
	}
}

func TestServeForegroundHandsTheServiceAnUpdatePass(t *testing.T) {
	serveStateDir(t)
	seen := stubServeRun(t, nil)
	stubServeNotify(t)
	calls := fakeSelfUpdate(t, selfupdate.Result{Outcome: selfupdate.Current})
	if code := serveForeground(&bytes.Buffer{}); code != 0 {
		t.Fatalf("code = %d", code)
	}
	if seen.Update == nil {
		t.Fatal("serve got no update pass")
	}
	seen.Update(context.Background())
	if *calls != 1 {
		t.Fatalf("the pass ran selfupdate %d times", *calls)
	}
}

// serve cancels the context it hands the pass when it stops; a pass that ran
// under any other context would keep downloading after the service is gone.
func TestServeUpdatePassRunsUnderServesContext(t *testing.T) {
	serveStateDir(t)
	seen := stubServeRun(t, nil)
	stubServeNotify(t)
	var got context.Context
	selfUpdateRun = func(ctx context.Context, _ selfupdate.Options) selfupdate.Result {
		got = ctx
		return selfupdate.Result{}
	}
	t.Cleanup(func() { selfUpdateRun = selfupdate.Run })
	serveForeground(&bytes.Buffer{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	seen.Update(ctx)
	if got != ctx {
		t.Fatal("the pass did not run under serve's context")
	}
}

// serve's pass records itself as serve's: only that record tells session
// start where the service runs from.
func TestServeUpdatePassRunsAsServe(t *testing.T) {
	serveStateDir(t)
	seen := stubServeRun(t, nil)
	stubServeNotify(t)
	var source string
	selfUpdateRun = func(_ context.Context, o selfupdate.Options) selfupdate.Result {
		source = o.Source
		return selfupdate.Result{}
	}
	t.Cleanup(func() { selfUpdateRun = selfupdate.Run })
	serveForeground(&bytes.Buffer{})

	seen.Update(context.Background())
	if source != selfupdate.SourceServe {
		t.Fatalf("source = %q", source)
	}
}
