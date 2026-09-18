// Package bridge is the stdio front a host starts: one redirector per host.
//
// Name to name, arguments to arguments, result back. If this layer ever does
// more than that, something is wrong.
package bridge

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/serve"
)

// Decision is what to do with the service described by serve.json.
type Decision int

const (
	// Keep uses the running service as it is.
	Keep Decision = iota
	// Start spawns one because none is recorded.
	Start
	// Restart replaces an older one with this build.
	Restart
)

// Decide reads the recorded build against ours. Newer wins, and only newer.
//
// It judges the file and nothing else. Whether anything is still listening is
// a second question, and ensure asks it: a state file is a hint, never the
// truth.
func Decide(state *serve.State, ours time.Time) Decision {
	if state == nil {
		return Start
	}
	if state.OlderThan(ours) {
		return Restart
	}
	return Keep
}

// Deps are the three moves a restart makes. A test replaces all three.
//
// Stop wraps serve.Stop(stateDir, false): the --force kill belongs to a human
// at the command line, never to a bridge tidying up in the background.
//
// Stop reads serve.json itself and asks the local endpoint, whatever channel
// this bridge speaks on. That is not a hole in "the channel is the address":
// the gate is the address the bridge calls, not the file it reads, and whoever
// can read serve.json holds both tokens already. Without it a cloud bridge
// could never replace a service.
type Deps struct {
	Stop     func(stateDir string) error
	WaitFree func(lockPath string, timeout time.Duration) error
	Spawn    func(stateDir string) (brokeAway bool, err error)
}

// StartTimeout and StartTick are the qmd handshake's figures from stage 1b-1,
// reused deliberately: one waiting rhythm in this project, not two.
const (
	StartTimeout = 60 * time.Second
	StartTick    = 250 * time.Millisecond
)

// RestartService replaces the running service with this build.
//
// The order is forced: a spawned serve cannot take serve.lock while the old one
// holds it. Stopping first is not politeness, it is the only order that works.
//
// It is not called Restart, although the brief calls it that: Restart is the
// decision above, and Go has one namespace for both.
func RestartService(stateDir string, deps Deps) error {
	// A stop that fails is not an error: a service that is already gone has
	// reached the goal this call has, which is a free lock.
	_ = deps.Stop(stateDir)
	if err := deps.WaitFree(serve.LockPath(stateDir), StartTimeout); err != nil {
		return fmt.Errorf("the old service did not let go of its lock: %w", err)
	}
	if _, err := deps.Spawn(stateDir); err != nil {
		return fmt.Errorf("start the service: %w", err)
	}
	return nil
}

// buildIdentity is the seam for asking what this program is. serve.BuildIdentity
// fails only where the operating system cannot name the running program, and
// nothing else in this package reaches that arm.
var buildIdentity = serve.BuildIdentity

// startProcess is the seam for actually starting a process. Without it the
// test of the default spawner would start a second loomux serve, and no test
// may leave a process behind.
var startProcess = (*exec.Cmd).Start

// defaultSpawn is what a bridge whose caller named no spawner uses: the real
// detached start out of internal/serve.
func defaultSpawn(stateDir string) (bool, error) {
	return serve.Spawn(stateDir, startProcess)
}

// stopService is the seam for the stop, and it exists so that a test can read
// the force flag this package passes. Against a state directory without a
// service, serve.Stop fails while reading serve.json and never reaches the arm
// where force means anything -- a test without the seam would pass just as
// happily on the kill switch.
var stopService = serve.Stop

// deps are the moves this bridge makes when it has to act on the service.
func (o Options) deps() Deps {
	return Deps{
		Stop:     func(stateDir string) error { return stopService(stateDir, false) },
		WaitFree: lock.WaitFree,
		Spawn:    o.spawn(),
	}
}

// spawn is the caller's spawner, or the real one. A test always brings its own;
// nothing else in this package may start a process.
func (o Options) spawn() func(string) (bool, error) {
	if o.Spawn != nil {
		return o.Spawn
	}
	return defaultSpawn
}

// ensure makes sure a service is there that this bridge can talk to, and
// returns once one has announced itself or the attempt has failed.
//
// It runs beside the host's handshake, never inside it: a cold start takes
// seconds and the host must see an answering server long before that. Its
// error goes nowhere on purpose -- the host learns of a failed start from the
// call that needed the service, not from a front that vanishes.
func ensure(ctx context.Context, opts Options, deps Deps) error {
	_, _, ours, err := buildIdentity()
	if err != nil {
		return err
	}
	// A state file that does not read is a state file that says nothing, and
	// saying nothing means nothing runs. The error is not passed on: there is
	// no service to talk about yet, only one to start.
	state, err := serve.ReadState(opts.StateDir)
	if err != nil {
		state = nil
	}
	decision := Decide(state, ours)
	// The second question, and the one the file cannot answer: a serve.json of
	// an equal or newer build left behind by a service that has since died
	// would otherwise keep this bridge from ever starting one.
	if decision == Keep && !held(serve.LockPath(opts.StateDir)) {
		decision = Start
	}
	switch decision {
	case Keep:
		return nil
	case Restart:
		err = RestartService(opts.StateDir, deps)
	default:
		_, err = deps.Spawn(opts.StateDir)
		if err != nil {
			err = fmt.Errorf("start the service: %w", err)
		}
	}
	if err != nil {
		return err
	}
	return waitForService(ctx, opts.StateDir, pidOf(state), StartTimeout)
}

// pidOf is the process a new service has to differ from, and 0 when none was
// recorded.
func pidOf(state *serve.State) int {
	if state == nil {
		return 0
	}
	return state.PID
}

// held reports whether somebody holds the lock at path. It answers by taking
// it, as serve status does: the file is empty on purpose, and on Windows a
// held byte range denies the read anyway.
//
// A lock that cannot even be opened counts as held. The question this answers
// is "may I start a second service", and an unknown is the one answer that
// must not lead to a second one.
func held(path string) bool {
	handle, free, err := lock.TryAcquire(path)
	if err != nil {
		return true
	}
	if !free {
		return true
	}
	// Taking it was the question; keeping it would refuse the next service.
	_ = handle.Release()
	return false
}

// waitForService waits until a service has written a state file of its own.
//
// serve writes serve.json last, after the lock and after both listeners, so a
// state file with a PID other than the one we started from is proof of a
// listener that answers -- and the one proof available without asking a port
// that may belong to somebody else.
//
// The lock is deliberately not probed while waiting. Probing means taking, and
// taking it would refuse the very service this call is waiting for.
//
// A spawned serve that lost the race for the lock is a normal outcome and no
// reason to spawn again: the one that won it writes the state file this loop
// is waiting for.
func waitForService(ctx context.Context, stateDir string, previousPID int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(StartTick)
	defer ticker.Stop()
	for {
		if state, err := serve.ReadState(stateDir); err == nil && state.PID != previousPID {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("no service announced itself in %s under %s", timeout, serve.StatePath(stateDir))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
