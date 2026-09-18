package ask

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/store"
)

// lockStale is when a lock left behind by a killed run stops counting.
//
// Without a rule like this, one run killed mid-rebuild would stop that checkout
// from ever refreshing again -- and the failure would be silent, which is the
// worst kind.
const lockStale = time.Hour

// Rebuild is the build a caller hands in.
//
// A function and not an import: this package stays free of the extractor, the
// same separation freshness keeps, so the query path and the hook path can both
// use it later without dragging the parser along.
type Rebuild func() error

// LockPath is the file that keeps two rebuilds apart.
func LockPath(root string) string { return store.CachePath(root, "rebuild.lock") }

// EnsureFresh probes the tree and rebuilds when it moved.
//
// Freshness belongs in the query path, and the reference explains why from
// experience: with the rebuild hanging off a session hook, every question
// between the first edit and the end of a turn answered from a graph that did
// not know the file just changed -- and an edit made outside the agent
// triggered nothing at all.
//
// Three properties, all of them load-bearing:
//
//   - never fatal: a failed probe or rebuild answers from the graph on disk
//   - no stampede: a lock, and the loser does not queue -- it answers
//   - writes only what a question reads: the graph, the sidecar, the record
func EnsureFresh(root, extractor string, rebuild Rebuild, notice func(string)) {
	say := func(format string, args ...any) {
		if notice != nil {
			notice(fmt.Sprintf(format, args...))
		}
	}

	drift, err := freshness.Probe(root, extractor)
	if err != nil {
		say("freshness probe failed, answering from the graph on disk: %v", err)
		return
	}
	if drift != nil && drift.Clean() {
		return
	}
	if drift != nil {
		say("%d files moved, rebuilding the graph", drift.Count())
	} else {
		// No record: unknown, never clean.
		say("no freshness record, building the graph")
	}

	if !lock(root) {
		say("another run is rebuilding, answering from the graph on disk")
		return
	}
	defer unlock(root)

	if err := rebuild(); err != nil {
		say("rebuild failed, answering from the graph on disk: %v", err)
	}
}

// lock takes the rebuild lock, or reports that someone else holds it.
//
// O_EXCL is the whole mechanism: the create either wins or it does not, with no
// window between the check and the take.
func lock(root string) bool {
	path := LockPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		fmt.Fprint(f, os.Getpid())
		f.Close()
		return true
	}
	// Held. Stale?
	info, statErr := os.Stat(path)
	if statErr != nil || time.Since(info.ModTime()) < lockStale {
		return false
	}
	if !takeOver(path) {
		return false
	}
	return lock(root)
}

// takeOver breaks a stale lock, and never a live one.
//
// Removing by path would not do: two runs that both saw the lock stale would
// both remove it -- the second one deleting the fresh lock the first had
// created in between -- and both would rebuild. A rename to a name only this
// run uses moves exactly one file, and that file's own age then tells whether
// it is the stale one. If it is not, this run has taken a live lock away; it
// links it back, which cannot overwrite a lock a third run may have created
// meanwhile, and yields.
//
// The age and not os.SameFile against the earlier Stat: on Windows SameFile
// looks the earlier file up again by its path, and after the rename that path
// names nothing.
func takeOver(path string) bool {
	claim := path + "." + strconv.Itoa(os.Getpid()) + ".stale"
	if err := os.Rename(path, claim); err != nil {
		return false
	}
	got, err := os.Stat(claim)
	if err != nil || time.Since(got.ModTime()) < lockStale {
		_ = os.Link(claim, path)
		_ = os.Remove(claim)
		return false
	}
	_ = os.Remove(claim)
	return true
}

// unlock releases the lock. A failure here is not worth reporting: the next run
// finds it stale.
func unlock(root string) {
	_ = os.Remove(LockPath(root))
}
