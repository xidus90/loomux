package serve_test

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/serve"
)

// helperEnv turns the test binary into the process a force test kills.
const helperEnv = "LOOMUX_SERVE_TEST_HELPER"

// TestHelperProcess is not a case of its own: it is the body of the child that
// the force test needs, a process that does nothing but stay alive. It ends by
// itself after a minute, so that no interrupted run leaves it behind.
func TestHelperProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		return
	}
	time.Sleep(time.Minute)
}

// helper is a started child and the one Wait that reaps it. done is closed
// rather than written to: both the case and the cleanup ask whether the child
// is gone, and a value would answer only the first of them.
type helper struct {
	cmd  *exec.Cmd
	done chan struct{}
}

// startHelper starts the test binary as a process that waits to be killed. The
// case may kill it or not; either way it is reaped before the case ends.
func startHelper(t *testing.T) *helper {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), helperEnv+"=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the helper process: %v", err)
	}
	h := &helper{cmd: cmd, done: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		close(h.done)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		select {
		case <-h.done:
		case <-time.After(20 * time.Second):
			t.Error("the helper process outlived its case")
		}
	})
	return h
}

// ended reports whether the child is gone, waiting up to d for a kill that may
// still be on its way. A case that expects it to live passes a short d; one
// that expects it to die passes a generous one, so that a slow machine fails
// the case only when the kill truly did not happen.
func (h *helper) ended(d time.Duration) bool {
	select {
	case <-h.done:
		return true
	case <-time.After(d):
		return false
	}
}

// runServe runs a service in this process on a state directory of its own. The
// case never force-kills this one: the PID in that serve.json is the test
// runner's.
func runServe(t *testing.T, brokeAway bool) string {
	t.Helper()
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serve.Run(ctx, serve.Options{
			StateDir:    dir,
			RegistryDir: dir,
			BrokeAway:   brokeAway,
		})
	}()
	// The service must be gone before the case's temporary directory is: on
	// Windows an open log or lock file makes the removal fail.
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(20 * time.Second):
			t.Error("Run did not return; it is stuck holding the lock")
		}
	})
	waitForState(t, dir)
	return dir
}

// waitForState waits until serve.json names a service, and fails rather than
// letting the case hang on a start that never finished.
func waitForState(t *testing.T, dir string) *serve.State {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		state, err := serve.ReadState(dir)
		if err == nil {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatalf("no %s after 20s: %v", serve.StatePath(dir), err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// deadEndpoint is an address that nothing listens on: a port taken from the
// operating system and given straight back. Anything else risks answering.
func deadEndpoint(t *testing.T) serve.Endpoint {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return serve.Endpoint{URL: "http://" + address + serve.MCPPath, Token: "no listener wants this"}
}

// holdLock takes serve.lock for the length of one test, the way a service that
// is there holds it. Every arm below that is about a listener which does not
// answer needs it: a free lock is Stop's own answer -- ErrNotRunning -- and
// would settle the call before the arm under test is reached.
func holdLock(t *testing.T, dir string) {
	t.Helper()
	handle, free, err := lock.TryAcquire(serve.LockPath(dir))
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if !free {
		t.Fatal("serve.lock is held although this test just made the directory")
	}
	t.Cleanup(func() { handle.Release() })
}

// writeState puts a state file where Status and Stop read one.
func writeState(t *testing.T, dir string, state *serve.State) {
	t.Helper()
	if err := serve.WriteState(dir, state); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
}

func TestStatusNamesTheServiceThatDiesWithItsHost(t *testing.T) {
	// A state written with BrokeAway=false must make Status say so, because
	// the user otherwise has no way to learn it: the service looks exactly
	// like one that outlives its host until the host goes away.
	dir := runServe(t, false)
	report, err := serve.Status(dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(report, "is running") {
		t.Errorf("Status says\n%s\nwant a running service", report)
	}
	if !strings.Contains(report, "dies with its host") {
		t.Errorf("Status says\n%s\nwant the lost breakaway named", report)
	}
	if !strings.Contains(report, strconv.Itoa(os.Getpid())) {
		t.Errorf("Status says\n%s\nwant the PID of the running service", report)
	}
	// The lock is the second question, and its answer must not come from the
	// state file: a held lock with a silent listener is the failure that
	// stop --force exists for.
	if !strings.Contains(report, "held") {
		t.Errorf("Status says\n%s\nwant the lock reported as held", report)
	}
	state := waitForState(t, dir)
	if strings.Contains(report, state.Local.Token) || strings.Contains(report, state.Cloud.Token) {
		t.Error("Status prints a listener token")
	}
}

func TestStatusNamesTheServiceThatBrokeAway(t *testing.T) {
	dir := runServe(t, true)
	report, err := serve.Status(dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if strings.Contains(report, "dies with its host") {
		t.Errorf("Status says\n%s\nwant no warning for a service that broke away", report)
	}
	if !strings.Contains(report, "breakaway") {
		t.Errorf("Status says\n%s\nwant the breakaway reported", report)
	}
}

func TestStatusSaysNothingRunsWithoutAStateFile(t *testing.T) {
	dir := t.TempDir()
	report, err := serve.Status(dir)
	if err != nil {
		t.Fatalf("Status: %v; a missing state file is an answer, not a failure", err)
	}
	if !strings.Contains(report, "not running") || !strings.Contains(report, serve.StatePath(dir)) {
		t.Errorf("Status says\n%s\nwant that nothing runs and where it looked", report)
	}
}

func TestStatusReportsAStateFileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	write(t, serve.StatePath(dir), "{not json")
	if _, err := serve.Status(dir); err == nil {
		t.Fatal("Status read a broken state file without complaint")
	}
}

func TestStatusAsksTheListenerRatherThanTheStateFile(t *testing.T) {
	// The state file says a service runs, and no listener answers. Only a
	// question to the address tells them apart.
	dir := t.TempDir()
	writeState(t, dir, &serve.State{Local: deadEndpoint(t), PID: 1, BrokeAway: true})
	report, err := serve.Status(dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(report, "not running") {
		t.Errorf("Status says\n%s\nwant that the listener does not answer", report)
	}
	// Nothing holds the lock either, and a state file that outlived its
	// service is exactly the case where that matters.
	if !strings.Contains(report, "free") {
		t.Errorf("Status says\n%s\nwant the lock reported as free", report)
	}
	if strings.Contains(report, "breakaway") {
		t.Errorf("Status says\n%s\nwant no breakaway line for a service that is not there", report)
	}
}

func TestStatusRefusesAListenerThatIsNotOurs(t *testing.T) {
	// The port was reused, or the state is older than the token: something
	// answers, and it is not the service this state describes.
	dir := runServe(t, true)
	state := waitForState(t, dir)
	stale := *state
	stale.Local.Token = serve.NewToken()
	writeState(t, dir, &stale)

	report, err := serve.Status(dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(report, "not running") || !strings.Contains(report, "401") {
		t.Errorf("Status says\n%s\nwant a refused listener, not a running service", report)
	}
}

func TestStatusReportsALockItCannotTake(t *testing.T) {
	dir := t.TempDir()
	// A directory where the lock file belongs: it can neither be opened nor
	// locked, and Status has to say so rather than claim the lock is free.
	if err := os.MkdirAll(serve.LockPath(dir), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeState(t, dir, &serve.State{Local: deadEndpoint(t), PID: 1})
	report, err := serve.Status(dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !strings.Contains(report, "unknown") {
		t.Errorf("Status says\n%s\nwant the lock reported as unknown", report)
	}
}

func TestStopEndsTheServiceThroughItsEndpoint(t *testing.T) {
	dir := runServe(t, true)
	if err := serve.Stop(dir, false); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	// Stop returns only once the lock is free: a caller that starts a new
	// service immediately afterwards must not be refused by the old one.
	handle, free, err := lock.TryAcquire(serve.LockPath(dir))
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if !free {
		t.Fatal("Stop returned while the service still held serve.lock")
	}
	if err := handle.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

func TestStopWithoutAStateFileSaysThereIsNothingToStop(t *testing.T) {
	if err := serve.Stop(t.TempDir(), false); err == nil {
		t.Fatal("Stop reported success although there is no state file")
	}
}

func TestStopRejectsAnAddressItCannotAsk(t *testing.T) {
	dir := t.TempDir()
	// A control character makes the URL unusable for a request. A state file
	// is written by a service, but it is also a file on disk that anything
	// may have damaged.
	writeState(t, dir, &serve.State{Local: serve.Endpoint{URL: "http://127.0.0.1:1\x7f/mcp"}, PID: 1})
	holdLock(t, dir)
	if err := serve.Stop(dir, false); err == nil {
		t.Fatal("Stop reported success although the address cannot be asked")
	}
}

func TestStopRefusesAListenerThatDoesNotTakeTheRequest(t *testing.T) {
	// A listener answers, and refuses: the token in the state file is not the
	// one it was started with. Stop must not read that as a service ended.
	dir := runServe(t, true)
	state := waitForState(t, dir)
	stale := *state
	stale.Local.Token = serve.NewToken()
	writeState(t, dir, &stale)

	err := serve.Stop(dir, false)
	if err == nil {
		t.Fatal("Stop reported success although the listener refused the request")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Stop: %v, want the refusal of the listener named", err)
	}
}

func TestStopWithForceRefusesAPIDThatIsNotOne(t *testing.T) {
	// A negative number in serve.json is not a process: on Unix it names a
	// process group, and os.Process.Kill guards only 0 and -1. force is the
	// mode that trusts this file least, so the refusal happens before any
	// system call and is therefore safe to run on every platform.
	dir := t.TempDir()
	writeState(t, dir, &serve.State{Local: deadEndpoint(t), PID: -2})
	holdLock(t, dir)
	err := serve.Stop(dir, true)
	if err == nil {
		t.Fatal("Stop acted on a PID that cannot be one")
	}
	if !strings.Contains(err.Error(), "-2") {
		t.Errorf("Stop: %v, want the impossible PID named", err)
	}
}

func TestStopWithoutForceDoesNotKill(t *testing.T) {
	// No listener, no PID kill: Stop returns an error naming `serve status`.
	dir := t.TempDir()
	h := startHelper(t)
	holdLock(t, dir)
	writeState(t, dir, &serve.State{Local: deadEndpoint(t), PID: h.cmd.Process.Pid})

	err := serve.Stop(dir, false)
	if err == nil {
		t.Fatal("Stop reported success although nothing answered")
	}
	if !strings.Contains(err.Error(), "serve status") {
		t.Errorf("Stop: %v, want an error naming `loomux serve status`", err)
	}
	if h.ended(2 * time.Second) {
		t.Error("Stop killed the process although --force was not given")
	}
}

func TestStopWithForceKillsThePIDFromTheStateFile(t *testing.T) {
	dir := t.TempDir()
	h := startHelper(t)
	holdLock(t, dir)
	writeState(t, dir, &serve.State{Local: deadEndpoint(t), PID: h.cmd.Process.Pid})

	if err := serve.Stop(dir, true); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !h.ended(20 * time.Second) {
		t.Fatal("--force left the process running")
	}
	// The same call again has nothing left to kill. It must say so: a stop
	// that reports success over a process that is not there would hide a
	// state file naming a PID that now belongs to somebody else.
	if err := serve.Stop(dir, true); err == nil {
		t.Error("Stop reported success although the process is long gone")
	}
}

// A second `loomux serve stop` meets the state file the first one left behind:
// serve.Run does not remove it. Both ways it must say that nothing is running
// rather than POST to a port nobody holds and advise --force, which would kill
// a dead PID or the stranger that inherited the number. serve status gets the
// same situation right, and stop now agrees with it.
func TestStopSaysNothingRunsWhenTheLockIsFreeAndNobodyAnswers(t *testing.T) {
	for _, force := range []bool{false, true} {
		dir := t.TempDir()
		h := startHelper(t)
		writeState(t, dir, &serve.State{Local: deadEndpoint(t), PID: h.cmd.Process.Pid})

		err := serve.Stop(dir, force)
		if !errors.Is(err, serve.ErrNotRunning) {
			t.Fatalf("Stop(force=%v): %v, want ErrNotRunning", force, err)
		}
		if h.ended(2 * time.Second) {
			t.Errorf("Stop(force=%v) killed the PID from a state file nothing vouches for", force)
		}
	}
}

// A lock that cannot even be opened is no answer, and must not be read as an
// absent service: the caller keeps the path it would have taken without the
// question, which with --force is the kill.
func TestStopKeepsForcingWhenTheLockCannotBeAsked(t *testing.T) {
	dir := t.TempDir()
	// A directory where the lock file belongs can be neither opened nor
	// locked, the same staging TestStatusReportsALockItCannotTake uses.
	if err := os.MkdirAll(serve.LockPath(dir), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	h := startHelper(t)
	writeState(t, dir, &serve.State{Local: deadEndpoint(t), PID: h.cmd.Process.Pid})

	if err := serve.Stop(dir, true); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !h.ended(20 * time.Second) {
		t.Fatal("--force left the process running")
	}
}
