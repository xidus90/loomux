// Package serve is the long-lived MCP service: two listeners, one per channel,
// and the lifecycle around them.
package serve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xidus90/loomux/internal/brain/search"
)

// Endpoint is one channel's address and its token.
type Endpoint struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// State is what serve.json holds. It is a hint, never the truth: the truth is
// whether the listener answers and whether serve.lock is held. A state file
// whose listener is silent and whose lock is free is stale and gets
// overwritten.
//
// PID is not optional. It is the only place a human can read the PID of a
// running service: the lock file cannot carry it, because a Windows byte-range
// lock is mandatory and denies the read for exactly as long as the service
// lives.
type State struct {
	Local      Endpoint  `json:"local"`
	Cloud      Endpoint  `json:"cloud"`
	PID        int       `json:"pid"`
	Executable string    `json:"executable"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	BrokeAway  bool      `json:"broke_away"`
}

// StatePath is serve.json under the state directory.
func StatePath(stateDir string) string { return filepath.Join(stateDir, "serve.json") }

// LockPath is the lock serve holds for its whole life.
func LockPath(stateDir string) string { return filepath.Join(stateDir, "serve.lock") }

// QmdLockPath guards probing and starting qmd. brain search takes the same
// one, and names it: search owns the path, because serve reaches search
// through answer while nothing may reach serve from there.
func QmdLockPath(stateDir string) string { return search.QmdLockPath(stateDir) }

// LogPath is where a detached serve writes, since nobody reads its stderr.
func LogPath(stateDir string) string { return filepath.Join(stateDir, "logs", "serve.log") }

// ReadState reads serve.json.
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

// WriteState writes serve.json atomically: a temporary file next to it, then a
// rename. A reader never sees half a file, and no temporary file survives.
//
//coverage:exempt four arms need the OS to break mid-write: encoding a struct of strings, an int, a bool and a time cannot fail, and creating, writing to and closing a temporary file in a directory MkdirAll has just accepted fail only on a denied ACL, a full disk or a withdrawn volume
func WriteState(stateDir string, s *State) error {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", stateDir, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	// os.CreateTemp already creates the file with mode 0600, so the state file
	// carries its tokens at that mode without a chmod of our own.
	temp, err := os.CreateTemp(stateDir, "serve-*.json")
	if err != nil {
		return fmt.Errorf("create temporary state file: %w", err)
	}
	name := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(name)
		return fmt.Errorf("write temporary state file: %w", err)
	}
	if err := temp.Close(); err != nil {
		os.Remove(name)
		return fmt.Errorf("close temporary state file: %w", err)
	}
	if err := os.Rename(name, StatePath(stateDir)); err != nil {
		os.Remove(name)
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

// BuildIdentity describes the running program, taken from os.Executable rather
// than from a fixed bin/ path: serve is machine-wide while bin/loomux.exe sits
// in one checkout of several.
//
//coverage:exempt both err arms need the OS to withdraw the running program: os.Executable fails where the platform cannot name it, and the stat only if the binary is gone while it runs
func BuildIdentity() (string, int64, time.Time, error) {
	path, err := os.Executable()
	if err != nil {
		return "", 0, time.Time{}, fmt.Errorf("find the running program: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", 0, time.Time{}, fmt.Errorf("stat %s: %w", path, err)
	}
	return path, info.Size(), info.ModTime(), nil
}

// OlderThan reports whether the recorded build is older than the one described.
// Newer wins, and only newer: an older bridge never restarts a newer serve, or
// two hosts out of two checkouts would kill each other on every call. Only the
// time decides; Size is recorded for serve status to show, not to compare, and
// a second rule beside the first could contradict it on an equal timestamp.
func (s *State) OlderThan(modTime time.Time) bool {
	return s.ModTime.Before(modTime)
}
