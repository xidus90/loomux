package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// State is what ui.json holds: who serves the interface and where. It is a
// hint, never the truth -- the truth is whether /health on that port names
// that PID -- and it holds no secret, so any process may read it.
type State struct {
	PID     int       `json:"pid"`
	Port    int       `json:"port"`
	Started time.Time `json:"started"`
}

// StatePath is ui.json under the state directory.
func StatePath(stateDir string) string { return filepath.Join(stateDir, "ui.json") }

// LogPath is where the detached interface writes, since nobody reads its
// stderr.
func LogPath(stateDir string) string { return filepath.Join(stateDir, "logs", "ui.log") }

// ReadState reads ui.json.
func ReadState(stateDir string) (*State, error) {
	data, err := os.ReadFile(StatePath(stateDir))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", StatePath(stateDir), err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", StatePath(stateDir), err)
	}
	return &s, nil
}

// writeState writes ui.json so that a reader never sees half a file.
func writeState(stateDir string, s State) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", stateDir, err)
	}
	// s holds an int, an int and a time: marshalling it cannot fail.
	data, _ := json.Marshal(s)
	return lock.ReplaceText(StatePath(stateDir), string(data))
}

// removeState deletes ui.json; a file that is not there is what the caller
// wanted.
func removeState(stateDir string) error {
	if err := os.Remove(StatePath(stateDir)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove %s: %w", StatePath(stateDir), err)
	}
	return nil
}

// healthClient asks one local port, never follows a redirect and gives up
// soon: what answers there is a stranger until its PID says otherwise.
var healthClient = &http.Client{
	Timeout:       2 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// Health asks the interface on a local port for its PID. Whatever is not a
// loomux interface -- nothing, another service, a garbled answer -- is an
// error.
func Health(port int) (int, error) {
	resp, err := healthClient.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("port %d answered %s", port, resp.Status)
	}
	var answer struct {
		PID int `json:"pid"`
	}
	// A stranger may send any amount; the answer of ours is a few bytes.
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<10)).Decode(&answer); err != nil {
		return 0, fmt.Errorf("port %d: %w", port, err)
	}
	return answer.PID, nil
}

// alive says whether state names an interface that is really running: the
// port answers and names the PID the file does.
func alive(s *State) bool {
	pid, err := Health(s.Port)
	return err == nil && s.PID > 0 && pid == s.PID
}

// imageOf is the seam for what a PID runs: no test may point at a real process
// it does not own.
var imageOf = processImage

// isLoomuxImage says whether an executable path is a loomux binary: its file
// name, in any case and without the .exe a Windows binary carries. A binary
// replaced while it runs keeps running under its new name: on Windows the
// swap moves it to loomux.old.exe or loomux.old.<n>.exe (package swap), and
// Linux marks its /proc/<pid>/exe with " (deleted)".
func isLoomuxImage(image string) bool {
	image = strings.TrimSuffix(image, " (deleted)")
	name := strings.ToLower(filepath.Base(strings.ReplaceAll(image, `\`, "/")))
	name = strings.TrimSuffix(name, ".exe")
	if name == "loomux" || name == "loomux.old" {
		return true
	}
	slot, found := strings.CutPrefix(name, "loomux.old.")
	return found && slot != "" && strings.Trim(slot, "0123456789") == ""
}

// killProcess is the seam of the one irreversible step. The real one ends the
// process outright: the interface holds nothing that outlives it.
var killProcess = endProcess

// endProcess ends the process with this PID.
//
//coverage:exempt it kills a real process, which no test may do with a PID it read from a file
func endProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

// Stop ends the running interface and reports whether there was one.
//
// ui.json says which process to end and /health on its port says whether that
// process is the interface: only a port that names the same PID, and a PID
// that runs a loomux binary, is ended. A file whose port is silent or answered
// by a stranger names nothing and is removed. One whose port names another
// PID, or whose process is not loomux, is an error and is left as it is:
// nothing is ended on its word.
func Stop(stateDir string) (bool, error) {
	state, err := ReadState(stateDir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	case err != nil:
		// Not readable as JSON: it names no process, and what names none is
		// removed rather than left to fail every later start.
		return false, removeState(stateDir)
	}
	pid, err := Health(state.Port)
	if err != nil || state.PID <= 0 {
		return false, removeState(stateDir)
	}
	if pid != state.PID {
		return false, fmt.Errorf("port %d answers as process %d, but %s names %d; nothing ended", state.Port, pid, StatePath(stateDir), state.PID)
	}
	image, err := imageOf(pid)
	if err != nil {
		return false, fmt.Errorf("cannot tell what process %d runs, so nothing ended: %w", pid, err)
	}
	if !isLoomuxImage(image) {
		return false, fmt.Errorf("process %d runs %s, not loomux; nothing ended", pid, image)
	}
	if err := killProcess(state.PID); err != nil {
		return false, fmt.Errorf("end process %d: %w", state.PID, err)
	}
	return true, removeState(stateDir)
}

// WaitReady waits until ui.json names the interface with the given PID --
// the child the starter just spawned -- and that interface answers, and
// returns that state. The start is detached, so this is the only way the
// starter learns the port; a ui.json of any other process is not the child,
// and the login code goes to the child only.
func WaitReady(stateDir string, pid int, timeout time.Duration) (*State, error) {
	deadline := time.Now().Add(timeout)
	for {
		if state, err := ReadState(stateDir); err == nil && state.PID == pid && alive(state) {
			return state, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("the interface did not answer within %s; see %s", timeout, LogPath(stateDir))
		}
		time.Sleep(50 * time.Millisecond)
	}
}
