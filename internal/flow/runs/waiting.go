package runs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/flow/journal"
)

// Waiting is one run whose journal holds an open gate, or whose journal or
// marker would not read: the callers name each of those their own way.
type Waiting struct {
	ID string
	// Gate is the open question; nil when the journal did not read.
	Gate *journal.PendingGate
	// Err is why the journal did not read.
	Err error
	// Marker and MarkerErr are the run's marker, read for exactly the runs
	// listed here; Marker is nil when there is none.
	Marker    *Marker
	MarkerErr error
}

// WaitingRuns lists the runs of the project at root that wait at a gate, and
// those whose journal cannot be told, by name, which is run number order.
// Runs that are over or have no open gate are left out.
//
// The runs directory is read with ListDir: absent means no runs, and one that
// cannot be listed is the error beside what was listed before it.
func WaitingRuns(root string) ([]Waiting, error) {
	dir := filepath.Join(root, filepath.FromSlash(Dir))
	entries, err := ListDir(dir)
	var out []Waiting
	for _, entry := range entries {
		id, isJournal := strings.CutSuffix(entry.Name(), ".jsonl")
		if entry.IsDir() || !isJournal {
			continue
		}
		gate, gateErr := journal.Pending(JournalPath(root, id))
		if gateErr == nil && gate == nil {
			continue
		}
		w := Waiting{ID: id, Gate: gate, Err: gateErr}
		w.Marker, w.MarkerErr = ReadMarker(MarkerPath(root, id))
		out = append(out, w)
	}
	return out, err
}

// ListDir is dir's entries, none when it is absent. One that cannot be listed
// -- a file, or a folder the process may not read -- is an error, and what was
// listed before it is still returned. Stat tells a file from an absent folder:
// Windows lists a file as a path it cannot find, POSIX as no directory.
func ListDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	info, statErr := os.Stat(dir)
	switch {
	case errors.Is(statErr, fs.ErrNotExist):
		return nil, nil
	case statErr == nil && !info.IsDir():
		err = errors.New("it is a file")
	}
	return entries, err
}
