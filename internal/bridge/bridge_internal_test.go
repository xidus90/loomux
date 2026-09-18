package bridge

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/serve"
)

// waitTimeout is what the waiting tests allow themselves. Everything they wait
// for is a file this process writes, so anything longer is a hang.
const waitTimeout = 5 * time.Second

func TestTheDefaultTransportIsTheHostsStdio(t *testing.T) {
	if _, ok := (Options{}).transport().(*mcp.StdioTransport); !ok {
		t.Errorf("a bridge without a transport speaks %T", (Options{}).transport())
	}
}

func TestTheDefaultSpawnerStartsTheDetachedService(t *testing.T) {
	// The seam is what keeps this test from starting a second loomux serve:
	// everything up to the start is real, the start itself is not.
	attempts := 0
	previous := startProcess
	startProcess = func(*exec.Cmd) error { attempts++; return nil }
	t.Cleanup(func() { startProcess = previous })

	brokeAway, err := (Options{}).spawn()(t.TempDir())
	if err != nil {
		t.Fatalf("the default spawner: %v", err)
	}
	if !brokeAway {
		t.Error("the first attempt asks to break away and it was accepted")
	}
	if attempts != 1 {
		t.Errorf("the default spawner made %d attempts, want 1", attempts)
	}
}

func TestTheDefaultStopNeverForces(t *testing.T) {
	// The kill switch belongs to a human at a command line, never to a bridge
	// tidying up in the background. The seam reads the flag that arrives:
	// without it the test would pass on force just as well, because serve.Stop
	// fails at the state file long before force means anything.
	var forced bool
	var stopped string
	previous := stopService
	stopService = func(stateDir string, force bool) error {
		stopped, forced = stateDir, force
		return nil
	}
	t.Cleanup(func() { stopService = previous })

	dir := t.TempDir()
	if err := (Options{}).deps().Stop(dir); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if stopped != dir {
		t.Errorf("the stop went to %q, want %q", stopped, dir)
	}
	if forced {
		t.Error("the bridge asked for the forced stop")
	}
}

func TestTheDefaultStopIsServeStop(t *testing.T) {
	// And the seam really stands in for serve.Stop: without a service there is
	// nothing to read, and that error is what comes back.
	if err := (Options{}).deps().Stop(t.TempDir()); err == nil {
		t.Fatal("stopping a service that is not there succeeded")
	}
}

func TestAFreeLockIsNotHeld(t *testing.T) {
	if held(filepath.Join(t.TempDir(), "serve.lock")) {
		t.Error("a lock nobody took counts as held")
	}
}

func TestATakenLockIsHeld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	handle, free, err := lock.TryAcquire(path)
	if err != nil || !free {
		t.Fatalf("take %s: free=%v err=%v", path, free, err)
	}
	t.Cleanup(func() { _ = handle.Release() })
	if !held(path) {
		t.Error("a lock this process holds counts as free")
	}
}

func TestALockThatCannotBeOpenedCountsAsHeld(t *testing.T) {
	// The question is "may I start a second service", and an unknown is the one
	// answer that must not lead to a second one.
	if !held(filepath.Join(t.TempDir(), "no-such-directory", "serve.lock")) {
		t.Error("a lock that cannot be opened counts as free")
	}
}

func TestPidOfAMissingStateIsNoProcess(t *testing.T) {
	if pid := pidOf(nil); pid != 0 {
		t.Errorf("a missing state file names process %d", pid)
	}
	if pid := pidOf(&serve.State{PID: 17}); pid != 17 {
		t.Errorf("the recorded process is %d", pid)
	}
}

func TestWaitingEndsWhenAServiceAnnouncesItself(t *testing.T) {
	dir := t.TempDir()
	writeState(t, dir, 4711, time.Now())
	if err := waitForService(context.Background(), dir, 0, waitTimeout); err != nil {
		t.Errorf("waitForService: %v", err)
	}
}

func TestWaitingIgnoresTheStateFileOfTheServiceItReplaced(t *testing.T) {
	// The old serve.json is still there while the new service comes up. Only a
	// different PID is proof that the new one has announced itself.
	dir := t.TempDir()
	writeState(t, dir, 4711, time.Now())
	err := waitForService(context.Background(), dir, 4711, 10*time.Millisecond)
	if err == nil {
		t.Fatal("the old state file counted as the new service")
	}
}

func TestWaitingGivesUpWhenTheHostGoesAway(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForService(ctx, t.TempDir(), 0, waitTimeout); !errors.Is(err, context.Canceled) {
		t.Errorf("waitForService: %v, want the cancelled context", err)
	}
}

func TestWaitingTicksUntilTheStateFileAppears(t *testing.T) {
	// The tick is what makes the wait a wait and not a single look.
	dir := t.TempDir()
	go func() {
		time.Sleep(StartTick / 2)
		_ = serve.WriteState(dir, &serve.State{PID: 4711, ModTime: time.Now()})
	}()
	if err := waitForService(context.Background(), dir, 0, waitTimeout); err != nil {
		t.Errorf("waitForService: %v", err)
	}
}

func TestARunningServiceOfTheSameBuildIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	writeState(t, dir, 4711, time.Now().Add(time.Hour))
	takeServeLock(t, dir)
	var done moves
	if err := ensure(context.Background(), Options{StateDir: dir}, recordingDeps(&done, nil)); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	// None of the three, and the stop is the one that matters: an older bridge
	// that ended a newer service would be two hosts shooting each other down.
	if done.stopped {
		t.Error("the bridge stopped a service that is newer than itself")
	}
	if done.waited {
		t.Error("the bridge waited for a lock it had no business taking")
	}
	if done.spawned {
		t.Error("the bridge started a second service beside a running one")
	}
}

func TestAStateFileWhoseServiceIsGoneStartsANewOne(t *testing.T) {
	// A newer build in serve.json and a free lock: the file is stale, and
	// without this the bridge would never start a service again.
	dir := t.TempDir()
	writeState(t, dir, 4711, time.Now().Add(time.Hour))
	var done moves
	deps := recordingDeps(&done, func(stateDir string) { writeState(t, stateDir, 4712, time.Now()) })
	if err := ensure(context.Background(), Options{StateDir: dir}, deps); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if !done.spawned {
		t.Error("the bridge kept a service that is not there")
	}
	if done.stateDir != dir {
		t.Errorf("the service was started in %q, want %q", done.stateDir, dir)
	}
	// A stale file is nothing to stop: Start, not Restart.
	if done.stopped {
		t.Error("the bridge stopped a service that was already gone")
	}
}

func TestANewerBridgeStopsTheOldServiceAndWaitsForTheNewOne(t *testing.T) {
	dir := t.TempDir()
	writeState(t, dir, 4711, time.Time{})
	var order []string
	deps := Deps{
		Stop:     func(string) error { order = append(order, "stop"); return nil },
		WaitFree: func(string, time.Duration) error { order = append(order, "wait"); return nil },
		Spawn: func(stateDir string) (bool, error) {
			order = append(order, "spawn")
			writeState(t, stateDir, 4712, time.Now())
			return true, nil
		},
	}
	if err := ensure(context.Background(), Options{StateDir: dir}, deps); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if len(order) != 3 || order[0] != "stop" || order[1] != "wait" || order[2] != "spawn" {
		t.Errorf("the restart went %v", order)
	}
}

func TestAFailedRestartIsReported(t *testing.T) {
	dir := t.TempDir()
	writeState(t, dir, 4711, time.Time{})
	deps := Deps{
		Stop:     func(string) error { return nil },
		WaitFree: func(string, time.Duration) error { return errors.New("still held") },
		Spawn:    func(string) (bool, error) { return true, nil },
	}
	if err := ensure(context.Background(), Options{StateDir: dir}, deps); err == nil {
		t.Fatal("a restart that never got the lock free came back as a success")
	}
}

func TestAFailedStartIsReported(t *testing.T) {
	deps := Deps{Spawn: func(string) (bool, error) { return false, errors.New("no such program") }}
	if err := ensure(context.Background(), Options{StateDir: t.TempDir()}, deps); err == nil {
		t.Fatal("a failed start came back as a success")
	}
}

func TestABridgeThatCannotNameItsOwnBuildStartsNothing(t *testing.T) {
	previous := buildIdentity
	buildIdentity = func() (string, int64, time.Time, error) {
		return "", 0, time.Time{}, errors.New("no program of that name")
	}
	t.Cleanup(func() { buildIdentity = previous })
	var done moves
	if err := ensure(context.Background(), Options{StateDir: t.TempDir()}, recordingDeps(&done, nil)); err == nil {
		t.Fatal("a bridge that cannot compare builds acted anyway")
	}
	if done.spawned || done.stopped {
		t.Error("the bridge acted on a service without knowing its own build")
	}
}

func TestNoSessionIsClosedUnderTheLock(t *testing.T) {
	// The invariant itself, measured instead of asserted in a comment. Closing
	// a session waits for what it still has in flight -- a note being
	// dispatched, a call still outstanding -- so a close under connecting
	// deadlocks the bridge against its own progress handler and holds every
	// other call behind the slowest question on the dying session.
	//
	// The state directory is empty, so the handshake fails and the close is
	// reached by the shortest way there is.
	stale := &mcp.ClientSession{}
	closed := false
	b := &bridge{opts: Options{StateDir: t.TempDir()}}
	b.closeSession = func(session *mcp.ClientSession) {
		closed = true
		if session != stale {
			t.Error("a session other than the stale one was closed")
		}
		// This is the whole test: TryLock fails exactly when the close runs
		// under the lock.
		if !b.connecting.TryLock() {
			t.Error("a session was closed while connecting was held")
			return
		}
		b.connecting.Unlock()
	}
	b.session.Store(stale)
	if _, err := b.connect(context.Background(), stale); err == nil {
		t.Fatal("a handshake without a state file succeeded")
	}
	if !closed {
		t.Error("the stale session was never closed")
	}
	if b.session.Load() != nil {
		t.Error("a session that was closed is still on offer")
	}
}

func TestNoSessionIsClosedUnderTheLockWhenTheHostGoesAway(t *testing.T) {
	// The same invariant at the other of the two places that close anything.
	// The type doc claims it for both, so both are measured -- and the
	// shutdown is the one where a stall would hide behind a host that is gone
	// anyway, which is how it would stay unnoticed.
	stale := &mcp.ClientSession{}
	closed := false
	b := &bridge{ready: make(chan struct{})}
	b.closeSession = func(session *mcp.ClientSession) {
		closed = true
		if session != stale {
			t.Error("a session other than the one on offer was closed")
		}
		if !b.connecting.TryLock() {
			t.Error("a session was closed while connecting was held")
			return
		}
		b.connecting.Unlock()
	}
	b.session.Store(stale)
	// The background start has had its say; the shutdown waits for that and
	// for nothing else.
	close(b.ready)
	b.close()
	if !closed {
		t.Error("the shutdown left the session open")
	}
	if b.session.Load() != nil {
		t.Error("a session that was closed is still on offer")
	}
}

func TestAReconnectSomebodyElseAlreadyMadeIsTakenOver(t *testing.T) {
	// Two calls of one host fail on the same session at the same moment. The
	// second one must take the session the first has just made, not close it
	// and build a third: closing it would leave the first call retrying on a
	// session that is gone. This is the question asked before the lock, which
	// is why a call that only wants a live session waits for nothing.
	//
	// The state directory is empty, so a call that went on to read serve.json
	// would fail rather than pass quietly.
	current, mine := &mcp.ClientSession{}, &mcp.ClientSession{}
	b := &bridge{opts: Options{StateDir: t.TempDir()}}
	b.session.Store(current)
	session, err := b.connect(context.Background(), mine)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if session != current {
		t.Error("the bridge threw away a session another call had just made")
	}
}

func TestASessionThatArrivedDuringTheWaitIsTakenOver(t *testing.T) {
	// The same question a second time, now the one asked with the lock: a call
	// that saw nothing, queued behind somebody else's handshake and got its
	// turn must take what that handshake produced instead of building a second
	// session.
	//
	// renew is called directly, which is exactly the state such a call is in
	// when the lock comes free -- its own look said "nothing there", and the
	// field has a session in it now. Staging it with two goroutines would race
	// the very look this measures.
	arrived := &mcp.ClientSession{}
	b := &bridge{opts: Options{StateDir: t.TempDir()}}
	b.session.Store(arrived)
	session, err := b.renew(context.Background(), nil)
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if session != arrived {
		t.Error("the waiting call built a session of its own")
	}
	if b.connecting.TryLock() {
		b.connecting.Unlock()
	} else {
		t.Error("renew kept the lock")
	}
}

func TestAProgressNoteWithoutAHostLapses(t *testing.T) {
	// The first call is what makes the host known. A note that arrived before
	// it has nowhere to go, and a message with no recipient is not one.
	b := &bridge{}
	b.relay(context.Background(), &mcp.ProgressNotificationClientRequest{
		Params: &mcp.ProgressNotificationParams{Message: "warming"},
	})
}

func TestABridgeWithoutARunContextSaysSo(t *testing.T) {
	// Only Run sets hostGone, and every test in this file builds the bridge by
	// hand. The message names the field and the way out, because the panic
	// from context.AfterFunc names neither.
	defer func() {
		message, ok := recover().(string)
		if !ok || !strings.Contains(message, "hostGone") {
			t.Fatalf("recovered %v, want a panic naming hostGone", message)
		}
	}()
	(&bridge{}).untilTheHostIsGone(context.Background())
}

// moves records every one of the three, not only the spawn: "an older bridge
// never stops a newer service" is the dangerous half of the rule, and a test
// that watches the spawn alone would not see a stop at all.
type moves struct {
	stopped  bool
	waited   bool
	spawned  bool
	stateDir string
}

// recordingDeps are deps that note what was asked of them, and optionally act
// as a started service would.
func recordingDeps(done *moves, act func(stateDir string)) Deps {
	return Deps{
		Stop:     func(string) error { done.stopped = true; return nil },
		WaitFree: func(string, time.Duration) error { done.waited = true; return nil },
		Spawn: func(stateDir string) (bool, error) {
			done.spawned, done.stateDir = true, stateDir
			if act != nil {
				act(stateDir)
			}
			return true, nil
		},
	}
}

func writeState(t *testing.T, dir string, pid int, modTime time.Time) {
	t.Helper()
	if err := serve.WriteState(dir, &serve.State{PID: pid, ModTime: modTime}); err != nil {
		t.Fatalf("write the state file: %v", err)
	}
}

func takeServeLock(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("create %s: %v", dir, err)
	}
	handle, free, err := lock.TryAcquire(serve.LockPath(dir))
	if err != nil || !free {
		t.Fatalf("take %s: free=%v err=%v", serve.LockPath(dir), free, err)
	}
	t.Cleanup(func() { _ = handle.Release() })
}
