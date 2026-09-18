package serve

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// probeTimeout bounds one question to a listener. The answer comes from
// 127.0.0.1 and is a status line without a body, so anything slower than this
// is not a slow answer but none.
const probeTimeout = 3 * time.Second

// stopTimeout is how long the lock may still be held after the service took
// the stop request. It is longer than shutdownTimeout on purpose: the service
// gives its requests that long to finish before it lets go, and a stop that
// gave up first would report a failure that is merely a slow request.
//
// It is a variable rather than a constant because it is also the seam: a
// service that takes the stop request and then keeps the lock cannot be staged
// with the real one, and the wait for that arm would otherwise stand in every
// run of the suite.
var stopTimeout = 10 * time.Second

// Status asks the service how it is, and asks the listener rather than only
// the files: a state file is a hint, a listener that answers is the truth.
//
// A missing state file is an answer and not a failure -- nothing runs. Only a
// state file that is there and unreadable is an error, because then something
// is wrong that the user has to see rather than read past.
func Status(stateDir string) (string, error) {
	state, err := ReadState(stateDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "loomux serve is not running: there is no " + StatePath(stateDir) + "\n", nil
		}
		return "", err
	}
	alive, why := probe(state.Local)

	var report strings.Builder
	if alive {
		report.WriteString("loomux serve is running\n")
	} else {
		fmt.Fprintf(&report, "loomux serve is not running: %s\n", why)
	}
	fmt.Fprintf(&report, "  pid         %d\n", state.PID)
	fmt.Fprintf(&report, "  local       %s\n", state.Local.URL)
	fmt.Fprintf(&report, "  cloud       %s\n", state.Cloud.URL)
	fmt.Fprintf(&report, "  executable  %s\n", state.Executable)
	fmt.Fprintf(&report, "  size        %d\n", state.Size)
	fmt.Fprintf(&report, "  built       %s\n", state.ModTime.Format(time.RFC3339))
	// The lock is the second question. A lock held while no listener answers
	// is the one failure a user cannot fix with a restart, and the reason
	// stop --force exists.
	fmt.Fprintf(&report, "  lock        %s\n", lockState(stateDir))
	// Only a service that is there can die with its host. The line would be a
	// statement about a process that no longer exists otherwise.
	if alive {
		fmt.Fprintf(&report, "  breakaway   %s\n", breakaway(state.BrokeAway))
	}
	// The tokens stay out of the report. It is what a user pastes into a bug
	// report, and a token in it is a token on somebody else's screen.
	return report.String(), nil
}

// breakaway is the one line that tells a service which outlives its host from
// one that does not.
func breakaway(brokeAway bool) string {
	if brokeAway {
		return "yes"
	}
	return "no -- this service dies with its host"
}

// lockState says who holds serve.lock, by taking it rather than by reading it:
// the file is empty on purpose, and on Windows a held byte range denies the
// read anyway.
func lockState(stateDir string) string {
	handle, free, err := lock.TryAcquire(LockPath(stateDir))
	if err != nil {
		return "unknown (" + err.Error() + ")"
	}
	if !free {
		return "held"
	}
	// Taking it was the question; keeping it would refuse the next service.
	_ = handle.Release()
	return "free"
}

// probe asks the local listener whether it is there and whether it is ours,
// and returns why it is not when it is not.
//
// The question is a GET on the stop route. The route is registered for POST
// only, so our own mux answers 405 and nothing is stopped; a listener that is
// not ours answers something else, and one that carries a different token
// answers 401 before the mux is reached at all. The cross-origin protection in
// front lets a GET through: it guards the unsafe methods.
func probe(endpoint Endpoint) (bool, string) {
	response, err := ask(http.MethodGet, endpoint, probeTimeout)
	if err != nil {
		return false, err.Error()
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		return false, fmt.Sprintf("%s answered %s, so it is not the service this state describes",
			StopURL(endpoint.URL), response.Status)
	}
	return true, ""
}

// ErrNotRunning is what a stop gets when there is nothing to stop although
// serve.json is still there: no listener answered and serve.lock is free.
//
// Run does not remove serve.json on an orderly stop, so the file outlives the
// service it describes and the fs.ErrNotExist branch a caller has for the
// harmless second stop never fires. The file is never the truth; the lock and
// the listener are, and both say the same thing here. serve status already
// answers this situation with "not running", and stop now agrees with it
// instead of advising --force, which would kill a dead or reused PID.
var ErrNotRunning = errors.New("nothing is running: no listener answered and " +
	"serve.lock is free, so serve.json describes a service that is already gone")

// Stop ends the service through its own endpoint, which is what makes it an
// orderly stop: the listeners close, what is in flight finishes, and the lock
// comes free because the process that holds it exits.
//
// force is the second way, and it is never taken on its own. Three things have
// to hold before the PID in serve.json is killed: the endpoint did not answer,
// the caller asked for it, and something still holds serve.lock -- or the lock
// cannot even be asked. A stop that killed by itself would kill a service that
// was merely busy; a state file naming a reused PID would kill a stranger; and
// a free lock says there is no service left to end, which is ErrNotRunning and
// not something to force.
func Stop(stateDir string, force bool) error {
	state, err := ReadState(stateDir)
	if err != nil {
		return err
	}
	if err := requestStop(state.Local); err != nil {
		// Before force, and before the advice to use it: a free lock settles
		// it. Nothing holds the service's lock, so nothing is the service,
		// and the PID in the file belongs to a dead process or to a stranger
		// that inherited the number.
		if free, taken := lockFree(stateDir); taken && free {
			return ErrNotRunning
		}
		if !force {
			return fmt.Errorf("%w; run `loomux serve status` to see what is there, and `loomux serve stop --force` to end it by its PID", err)
		}
		return kill(state.PID)
	}
	// The request is answered before the shutdown begins, so the service is
	// still holding the lock at this point. A caller that starts the next
	// service immediately must not be refused by the one it just ended.
	if err := lock.WaitFree(LockPath(stateDir), stopTimeout); err != nil {
		return fmt.Errorf("the service took the stop request but kept %s: %w", LockPath(stateDir), err)
	}
	return nil
}

// lockFree asks serve.lock whether anybody holds it, and says whether the
// question could be answered at all. A lock that cannot even be opened is no
// answer: the caller then keeps the path it would have taken without this
// question, rather than reading an unopenable lock as an absent service.
//
// It takes the lock rather than reading it, as lockState does, and for the
// same reason: the file is empty on purpose.
func lockFree(stateDir string) (free, taken bool) {
	handle, free, err := lock.TryAcquire(LockPath(stateDir))
	if err != nil {
		return false, false
	}
	if free {
		// Taking it was the question; keeping it would refuse the next
		// service the caller starts.
		_ = handle.Release()
	}
	return free, true
}

// requestStop is the POST the stop route waits for.
func requestStop(endpoint Endpoint) error {
	response, err := ask(http.MethodPost, endpoint, probeTimeout)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%s answered %s", StopURL(endpoint.URL), response.Status)
	}
	return nil
}

// ask sends one request to the stop route of a channel, with that channel's
// token. Both questions this file asks go to the same address and differ only
// in the method.
func ask(method string, endpoint Endpoint, timeout time.Duration) (*http.Response, error) {
	request, err := http.NewRequest(method, StopURL(endpoint.URL), nil)
	if err != nil {
		return nil, fmt.Errorf("ask %s: %w", StopURL(endpoint.URL), err)
	}
	request.Header.Set("Authorization", "Bearer "+endpoint.Token)
	client := &http.Client{Timeout: timeout}
	return client.Do(request)
}

// kill is the last resort: the PID out of serve.json, because the lock file
// cannot carry one and there is nowhere else a PID is written down.
func kill(pid int) error {
	// A PID that is not a PID is refused before any system call. force is by
	// definition the mode that acts on a file nothing vouches for any more,
	// and on Unix a negative number is not a process but a process group:
	// os.Process.Kill guards only 0 and -1, so -2 out of a damaged serve.json
	// would kill every process in group 2.
	if pid <= 0 {
		return fmt.Errorf("serve.json names %d, which is not a process", pid)
	}
	process, err := os.FindProcess(pid)
	if err == nil {
		err = process.Kill()
	}
	if err != nil {
		return fmt.Errorf("kill process %d, the PID in serve.json: %w", pid, err)
	}
	return nil
}
