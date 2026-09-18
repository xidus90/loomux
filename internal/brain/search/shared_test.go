package search

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// scriptedSession is a session whose every call is the test's own.
type scriptedSession struct {
	call func(name string, args map[string]any) (map[string]any, error)
}

func (s *scriptedSession) Call(name string, args map[string]any) (map[string]any, error) {
	return s.call(name, args)
}

func (s *scriptedSession) Close() error { return nil }

// freePort reserves a port and gives it back again, so that a stub daemon can
// take it later. Nothing else binds it in between on a machine that runs one
// test suite.
func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("give the reserved port back: %v", err)
	}
	return port
}

// answersLikeQmd is a daemon that is up: a JSON-RPC result for everything it
// is asked, with one hit in it, which is more than Reachable asks for and just
// enough for a query.
func answersLikeQmd(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{
		"structuredContent": map[string]any{
			"results": []any{map[string]any{"file": "c/a.md", "title": "A"}},
		},
	}})
}

// daemonOn puts handler on port and gives the listener back, so that a test
// can take the port away again. The bind is retried for a second: a daemon
// that has just let go of the port is not always done with it at once.
func daemonOn(t *testing.T, port int, handler http.Handler) (net.Listener, bool) {
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	var listener net.Listener
	var err error
	for i := 0; i < 50; i++ {
		if listener, err = net.Listen("tcp", address); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Errorf("the stub daemon could not take port %d: %v", port, err)
		return nil, false
	}
	server := &http.Server{Handler: handler}
	t.Cleanup(func() { _ = server.Close() })
	go func() { _ = server.Serve(listener) }()
	return listener, true
}

// stubDaemon is a daemon that is up, on port, for as long as the test runs.
func stubDaemon(t *testing.T, port int) bool {
	_, ok := daemonOn(t, port, http.HandlerFunc(answersLikeQmd))
	return ok
}

// TestTwoStartersStartOneDaemon is the whole point of the shared lock, without
// a socket in it: the second starter must find the daemon of the first, and it
// only does when the wait for that daemon happened inside the critical section.
// A lock released after the spawn leaves a window as wide as the daemon's boot.
func TestTwoStartersStartOneDaemon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "qmd.lock")
	var mu sync.Mutex
	up, starts := false, 0
	steps := func() daemonSteps {
		return daemonSteps{
			probe: func() bool {
				mu.Lock()
				defer mu.Unlock()
				return up
			},
			start: func() error {
				mu.Lock()
				defer mu.Unlock()
				starts++
				return nil
			},
			started: func() {},
			wait: func() error {
				// The daemon of a real start answers only after its boot; the
				// second starter must not see the port before then.
				time.Sleep(30 * time.Millisecond)
				mu.Lock()
				defer mu.Unlock()
				up = true
				return nil
			},
		}
	}

	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { errs <- ensureDaemon(path, steps()) }()
	}
	for i := 0; i < 2; i++ {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatalf("a starter refused: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("a starter never returned")
		}
	}
	if starts != 1 {
		t.Errorf("two starters started %d daemons, want 1", starts)
	}
}

// TestTwoConnectsStartOneDaemon is the same claim through the connect both
// callers use, with a stub daemon that comes up late, as a real one does.
func TestTwoConnectsStartOneDaemon(t *testing.T) {
	path := filepath.Join(t.TempDir(), "qmd.lock")
	port := freePort(t)
	var spawned int32
	connect := DefaultConnectWith(path, port,
		func(string) ([]string, error) { return []string{"qmd"}, nil },
		func([]string, []string) error {
			atomic.AddInt32(&spawned, 1)
			go func() {
				time.Sleep(30 * time.Millisecond)
				stubDaemon(t, port)
			}()
			return nil
		},
		5*time.Second, nil)

	type outcome struct {
		session Session
		err     error
	}
	done := make(chan outcome, 2)
	for i := 0; i < 2; i++ {
		go func() {
			session, err := connect(nil)
			done <- outcome{session, err}
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case got := <-done:
			if got.err != nil || got.session == nil {
				t.Fatalf("connect: session %v, error %v", got.session != nil, got.err)
			}
		case <-time.After(20 * time.Second):
			t.Fatal("a connect never returned")
		}
	}
	if got := atomic.LoadInt32(&spawned); got != 1 {
		t.Errorf("two connects spawned %d daemons, want 1", got)
	}
}

// TestEnsureDaemonProbesUnderTheLock pins that the probe waits for the lock
// rather than running beside it: with the lock in someone else's hand, a
// reachable daemon is not yet an answer.
func TestEnsureDaemonProbesUnderTheLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "qmd.lock")
	holder, held, err := lock.TryAcquire(path)
	if err != nil || !held {
		t.Fatalf("TryAcquire: %v, held=%v", err, held)
	}

	probed := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- ensureDaemon(path, daemonSteps{
			probe:   func() bool { close(probed); return true },
			start:   func() error { t.Error("a reachable daemon was started again"); return nil },
			started: func() {},
			wait:    func() error { return nil },
		})
	}()

	select {
	case <-probed:
		_ = holder.Release()
		t.Fatal("the probe ran while another holder had the lock")
	case <-time.After(200 * time.Millisecond):
	}
	if err := holder.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ensureDaemon after the release: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ensureDaemon did not return after the holder released")
	}
}

// TestEnsureDaemonPassesOnWhatTheStepsRefuse: neither a start nor a wait that
// failed is a daemon, and the lock is given back either way.
func TestEnsureDaemonPassesOnWhatTheStepsRefuse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		steps   daemonSteps
		want    string
		started int
	}{
		{
			name: "a start that failed",
			steps: daemonSteps{
				probe: func() bool { return false },
				start: func() error { return errors.New("cannot launch qmd") },
				wait:  func() error { t.Error("waited for a daemon that was never started"); return nil },
			},
			want: "cannot launch qmd",
		},
		{
			name: "a daemon that never answered",
			steps: daemonSteps{
				probe: func() bool { return false },
				start: func() error { return nil },
				wait:  func() error { return errors.New("no qmd daemon answered") },
			},
			want:    "no qmd daemon answered",
			started: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "qmd.lock")
			starts := 0
			steps := tc.steps
			steps.started = func() { starts++ }
			err := ensureDaemon(path, steps)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
			if starts != tc.started {
				t.Errorf("the start was announced %d times, want %d", starts, tc.started)
			}
			// The lock is free again, or the next starter would hang here.
			handle, held, err := lock.TryAcquire(path)
			if err != nil || !held {
				t.Fatalf("the lock was not given back: %v, held=%v", err, held)
			}
			_ = handle.Release()
		})
	}
}

// TestEnsureDaemonReportsALockItCannotTake: a path that cannot be opened is
// the caller's error, not a silent start without the lock.
func TestEnsureDaemonReportsALockItCannotTake(t *testing.T) {
	dir := t.TempDir()
	err := ensureDaemon(dir, daemonSteps{
		probe:   func() bool { t.Error("probed without the lock"); return true },
		start:   func() error { return nil },
		started: func() {},
		wait:    func() error { return nil },
	})
	if err == nil || !strings.Contains(err.Error(), "open lock") {
		t.Fatalf("got %v, want an error naming the lock", err)
	}
}

// TestEnsureDaemonTakesTheRunningSystemsStarter pins what the exported form
// hands the connect: the caller's lock, port and environment, and the default
// launcher and spawner. It runs against the seam, because the real one would
// start a daemon.
func TestEnsureDaemonTakesTheRunningSystemsStarter(t *testing.T) {
	var gotPath string
	var gotPort int
	var gotEnv map[string]string
	var gotWait time.Duration
	connectWith = func(path string, port int, launcher func(string) ([]string, error), spawner DaemonSpawner, wait time.Duration, notice func(string)) ConnectFunc {
		gotPath, gotPort, gotWait = path, port, wait
		if launcher == nil || spawner == nil || notice != nil {
			t.Errorf("launcher %v, spawner %v, notice %v", launcher != nil, spawner != nil, notice != nil)
		}
		return func(env map[string]string) (Session, error) {
			gotEnv = env
			return nil, errors.New("no daemon")
		}
	}
	defer func() { connectWith = DefaultConnectWith }()

	env := map[string]string{"QMD_FORCE_CPU": "1"}
	err := EnsureDaemon("C:/state/qmd.lock", env, 9001)
	if err == nil || err.Error() != "no daemon" {
		t.Fatalf("got %v", err)
	}
	if gotPath != "C:/state/qmd.lock" || gotPort != 9001 || gotWait != DaemonWait {
		t.Errorf("connect got %q, port %d, wait %v", gotPath, gotPort, gotWait)
	}
	if gotEnv["QMD_FORCE_CPU"] != "1" {
		t.Errorf("the environment was not handed on: %v", gotEnv)
	}
}

// TestQmdLockPathLiesInTheStateDirectory: the name is spelled once, and serve
// reads the same file.
func TestQmdLockPathLiesInTheStateDirectory(t *testing.T) {
	if got, want := QmdLockPath("C:/state"), filepath.Join("C:/state", "qmd.lock"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	dir := t.TempDir()
	t.Setenv(config.StateDirEnv, dir)
	if got, want := DefaultQmdLockPath(), filepath.Join(dir, "qmd.lock"); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TestNewQmdMcpPortHandsOnTheLockPath: the port's connect gets the lock it was
// given, and the state directory's when it was given none.
func TestNewQmdMcpPortHandsOnTheLockPath(t *testing.T) {
	var got string
	connectDefault = func(path string, port int, notice func(string)) ConnectFunc {
		got = path
		return nil
	}
	defer func() { connectDefault = DefaultConnect }()

	NewQmdMcpPort(WithQmdLock("C:/elsewhere/qmd.lock"))
	if got != "C:/elsewhere/qmd.lock" {
		t.Errorf("the port took %q instead of the lock it was given", got)
	}

	dir := t.TempDir()
	t.Setenv(config.StateDirEnv, dir)
	NewQmdMcpPort()
	if want := filepath.Join(dir, "qmd.lock"); got != want {
		t.Errorf("the port took %q, want %q", got, want)
	}
}

// TestADeadDaemonIsRetriedExactlyOnce is the rule of this stage on the qmd
// hop, as serve asks for it (serve.qmdOptions passes the two): the call is
// repeated once against a daemon that was re-probed and, if need be, restarted
// under the lock -- and then it is an error, never an empty hit list.
func TestADeadDaemonIsRetriedExactlyOnce(t *testing.T) {
	connects, calls := 0, 0
	port := NewQmdMcpPort(WithColdAttempts(2), WithConnect(func(map[string]string) (Session, error) {
		connects++
		return &scriptedSession{call: func(string, map[string]any) (map[string]any, error) {
			calls++
			return nil, errors.New("the daemon hung up")
		}}, nil
	}))

	hits, err := port.Search("q", []string{"c"}, ProfileFast, 1)
	if err == nil {
		t.Fatal("a dead daemon answered")
	}
	if hits != nil {
		t.Errorf("an outage came back as %d hits", len(hits))
	}
	if calls != 2 {
		t.Errorf("the call was made %d times, want 2 -- one try and one retry", calls)
	}
	if connects != 2 {
		t.Errorf("the daemon was ensured %d times, want 2", connects)
	}
}

// TestTheRetryAnswers: the second call is a real second chance, not a formality.
func TestTheRetryAnswers(t *testing.T) {
	calls := 0
	port := NewQmdMcpPort(WithConnect(func(map[string]string) (Session, error) {
		return &scriptedSession{call: func(string, map[string]any) (map[string]any, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("the daemon hung up")
			}
			return map[string]any{"structuredContent": map[string]any{
				"results": []any{map[string]any{"file": "c/a.md", "title": "A"}},
			}}, nil
		}}, nil
	}))

	hits, err := port.Search("q", []string{"c"}, ProfileFast, 1)
	if err != nil {
		t.Fatalf("the retry did not answer: %v", err)
	}
	if len(hits) != 1 || hits[0].Relative != "a.md" {
		t.Fatalf("got %v", hits)
	}
	if calls != 2 {
		t.Errorf("the call was made %d times, want 2", calls)
	}
}

// TestColdAttemptsIsTheReferencesThree: the command line keeps what stage 1b-1
// signed off. The rule of one retry belongs to the long-lived service, and
// serve asks for it by option; a 1b-2 task does not narrow a released parity
// value for everyone.
func TestColdAttemptsIsTheReferencesThree(t *testing.T) {
	if ColdAttempts != 3 {
		t.Fatalf("ColdAttempts is %d; the reference's cold_attempts is 3", ColdAttempts)
	}
}

// TestTheRetryProbesAndRestartsUnderTheLock closes the gap the stubbed connect
// leaves: here the retry goes through the connect of the running system, so
// what happens between the two calls is a real probe and a real start under
// the lock. The daemon hangs up mid-query and takes its port with it.
func TestTheRetryProbesAndRestartsUnderTheLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "qmd.lock")
	port := freePort(t)

	queries := 0
	var dying net.Listener
	var hungUp sync.Once
	listener, ok := daemonOn(t, port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err == nil && strings.Contains(string(body), "tools/call") {
			queries++
			// The daemon dies: the port goes first, so that the next probe
			// finds nothing, then the answer this call is waiting for.
			hungUp.Do(func() { _ = dying.Close() })
			if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
				_ = conn.Close()
			}
			return
		}
		answersLikeQmd(w, r)
	}))
	if !ok {
		t.Fatal("the first daemon never came up")
	}
	dying = listener

	spawned := 0
	connect := DefaultConnectWith(path, port,
		func(string) ([]string, error) { return []string{"qmd"}, nil },
		func([]string, []string) error {
			spawned++
			if !stubDaemon(t, port) {
				return errors.New("the restarted daemon could not bind")
			}
			return nil
		},
		10*time.Second, nil)

	hits, err := NewQmdMcpPort(WithConnect(connect), WithColdAttempts(2), WithPort(port)).
		Search("q", []string{"c"}, ProfileFast, 1)
	if err != nil {
		t.Fatalf("the retry against the restarted daemon: %v", err)
	}
	if len(hits) != 1 || hits[0].Relative != "a.md" {
		t.Fatalf("got %v", hits)
	}
	if spawned != 1 {
		t.Errorf("the retry started %d daemons, want 1", spawned)
	}
	if queries != 1 {
		t.Errorf("the daemon that died answered %d queries, want 1", queries)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the restart left no lock file behind: %v", err)
	}
}
