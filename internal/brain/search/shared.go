package search

import (
	"path/filepath"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// DaemonWait is how long a starter waits for the daemon it started. A cold qmd
// pays a model load of about 5.7 s; a minute is that with room to spare.
const DaemonWait = 60 * time.Second

// QmdLockPath is the lock both qmd starters take. It lies in the state
// directory beside serve.json, and it is deliberately not serve.lock: serve
// holds that one for its whole life, so a second taker would never get in.
func QmdLockPath(stateDir string) string { return filepath.Join(stateDir, "qmd.lock") }

// DefaultQmdLockPath is the lock of the state directory this run works in. It
// is read on use, never once at start: a test that moves the state directory
// moves the lock with it.
func DefaultQmdLockPath() string { return QmdLockPath(config.StateDir()) }

// connectWith is the connect EnsureDaemon builds; a test replaces it to see
// what EnsureDaemon hands on without starting a daemon.
var connectWith = DefaultConnectWith

// EnsureDaemon makes sure a qmd daemon answers on port, taking qmdLockPath
// across the probe and the start so that two starters never produce two
// daemons. It is the same path a search takes, not a second one beside it: the
// session it opens on the way is thrown away, since HTTP keeps nothing open.
func EnsureDaemon(qmdLockPath string, env map[string]string, port int) error {
	_, err := connectWith(qmdLockPath, port, Launcher, DefaultSpawner, DaemonWait, nil)(env)
	return err
}

// daemonSteps are the four steps a starter takes under the lock. They are an
// argument rather than a fixed body so that a test sees the order without a
// socket and without a daemon; started is heard between the start and the wait
// for it.
type daemonSteps struct {
	probe   func() bool
	start   func() error
	started func()
	wait    func() error
}

// ensureDaemon runs the steps under the lock at path.
//
// The wait belongs inside the critical section, although the brief calls the
// section short. A lock released the moment the spawn returns leaves a window
// as wide as the daemon's boot -- measured at 5.7 s -- in which the second
// starter takes the lock, probes a port nobody listens on yet and starts a
// second daemon. That is the very thing the shared lock exists to prevent.
//
// The state directory is not created here. It is the registry's own directory,
// and nothing searches without a registry; a lock file made in a directory
// that does not exist would be the first file of a state that has none.
func ensureDaemon(path string, steps daemonSteps) error {
	handle, err := lock.Acquire(path)
	if err != nil {
		return err
	}
	defer func() { _ = handle.Release() }()

	if steps.probe() {
		return nil
	}
	if err := steps.start(); err != nil {
		return err
	}
	steps.started()
	return steps.wait()
}
