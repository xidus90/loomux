package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// Outcome is what one update pass came to.
type Outcome string

const (
	Current Outcome = "current"
	Updated Outcome = "updated"
	Skipped Outcome = "skipped"
	Failed  Outcome = "failed"
	// Busy means another pass held the lock. It is never written: the pass
	// that holds the lock writes its own result.
	Busy Outcome = "busy"
)

// Who ran a pass. Both write update.json, but only serve's record says where
// the service runs from: a pass by hand may well run from a checkout.
const (
	SourceServe = "serve"
	SourceCLI   = "cli"
)

// Status is update.json: what the last pass found, for session start to read
// without asking the network or the service.
type Status struct {
	Source     string    `json:"source"`
	CheckedAt  time.Time `json:"checked_at"`
	Executable string    `json:"executable"`
	Running    string    `json:"running"`
	Result     Outcome   `json:"result"`
	Version    string    `json:"version"`
	Error      string    `json:"error"`
}

// StatusPath is update.json in the state directory.
func StatusPath(stateDir string) string { return filepath.Join(stateDir, "update.json") }

// Canonical is where the machine-wide binary lives. Nothing else is ever
// updated.
func Canonical(stateDir string) string { return filepath.Join(stateDir, "bin", "loomux.exe") }

// IsCanonical asks the file system rather than comparing strings: on Windows
// os.Executable may spell the same file in other letters or another form.
func IsCanonical(path, stateDir string) bool {
	have, err := os.Stat(path)
	if err != nil {
		return false
	}
	want, err := os.Stat(Canonical(stateDir))
	if err != nil {
		return false
	}
	return os.SameFile(have, want)
}

// WriteStatus replaces update.json whole, so a reader never sees half of it.
func WriteStatus(stateDir string, s Status) error {
	// Encoded first, so a status that cannot be written creates nothing. The
	// time is what can fail: JSON holds years 0 to 9999 only.
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", StatusPath(stateDir), err)
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", stateDir, err)
	}
	return lock.ReplaceText(StatusPath(stateDir), string(data)+"\n")
}

// ReadStatus is update.json, or nil when no pass has written one yet.
func ReadStatus(stateDir string) (*Status, error) {
	data, err := os.ReadFile(StatusPath(stateDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", StatusPath(stateDir), err)
	}
	var s Status
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", StatusPath(stateDir), err)
	}
	return &s, nil
}
