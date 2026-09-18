package ask

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/store"
)

// lockStale is when a lock left behind by a killed run stops counting.
//
// Without a rule like this, one run killed mid-rebuild would stop that checkout
// from ever refreshing again -- and the failure would be silent, which is the
// worst kind.
const lockStale = time.Hour

// linkFile is the hard link takeOver restores a live lock with, swapped in a
// test: every filesystem this repository is developed on supports links, so
// the arm that matters -- the one a network share or an exFAT volume takes --
// is not reachable otherwise.
var linkFile = os.Link

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
//
// The last one is why a clean tree is not enough to return on. The freshness
// record knows about source files and nothing else, so a sidecar that was
// deleted, or written by a binary with another index version, leaves the
// record clean and the question ranking without the body text -- forever, or
// until a source file happens to move. A sidecar that is missing, or of
// another index version, therefore counts as drift. One that passes that
// cheap check and only then fails to parse does not: the probe reads the
// version field, not the whole file.
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
	switch {
	case drift == nil:
		// No record: unknown, never clean.
		say("no freshness record, building the graph")
	case !drift.Clean():
		say("%d files moved, rebuilding the graph", drift.Count())
	case !lexicon.Usable(root):
		say("no ask index, rebuilding the graph")
	default:
		return
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
//
// The restore is a link where links work and a rename where they do not. The
// link stays first because it cannot overwrite a lock a third run created in
// the meantime -- an existing target is the one case where the live lock is
// back already and this run has only its own copy left to drop. Any other
// error is usually a filesystem with no hard links at all: a network share,
// exFAT, a container bind mount. Removing the claim there, as this used to,
// deleted the live lock and let both runs rebuild -- the stampede the lock
// exists to prevent. A rename needs no link support and puts the file back.
//
// Usually, not always: EPERM under protected_hardlinks, EMLINK, ENOSPC and a
// cross-device error reach the rename too, and there it can still overwrite a
// lock a third run created in the window. That is the residual -- narrower
// than what it replaces, which lost the live lock on every link failure.
func takeOver(path string) bool {
	claim := path + "." + strconv.Itoa(os.Getpid()) + ".stale"
	if err := os.Rename(path, claim); err != nil {
		return false
	}
	got, err := os.Stat(claim)
	if err != nil || time.Since(got.ModTime()) < lockStale {
		if err := linkFile(claim, path); err == nil || errors.Is(err, fs.ErrExist) {
			_ = os.Remove(claim)
			return false
		}
		_ = os.Rename(claim, path)
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
