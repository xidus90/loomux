package ask

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/child"
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

// Status is how a refresh ended when it did not fail.
type Status int

const (
	// StatusClean means the graph on disk already matched the tree.
	StatusClean Status = iota
	// StatusRebuilt means this run rebuilt the graph.
	StatusRebuilt
)

// RefreshOptions tunes one refresh.
type RefreshOptions struct {
	Force  string        // a reason to rebuild although the probe is clean; "" probes
	Wait   time.Duration // how long to wait for a held lock; 0 does not wait
	Notice func(string)
}

// ProbeError is a freshness probe that could not run.
type ProbeError struct{ Err error }

func (e *ProbeError) Error() string { return "freshness probe failed: " + e.Err.Error() }
func (e *ProbeError) Unwrap() error { return e.Err }

// RebuildError is a rebuild that ran and failed.
type RebuildError struct{ Err error }

func (e *RebuildError) Error() string { return "rebuild failed: " + e.Err.Error() }
func (e *RebuildError) Unwrap() error { return e.Err }

// LockedError is a rebuild lock another run still held when the wait ran out.
type LockedError struct {
	Path string
	Age  time.Duration
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("another run holds the rebuild lock %s (%s old)", e.Path, e.Age)
}

// lockPoll is how often a waiting run looks at the lock again.
const lockPoll = 100 * time.Millisecond

// Refresh probes the tree and rebuilds when it moved, and says how that ended.
//
// Freshness belongs in the query path, and the reference explains why from
// experience: with the rebuild hanging off a session hook, every question
// between the first edit and the end of a turn answered from a graph that did
// not know the file just changed -- and an edit made outside the agent
// triggered nothing at all.
//
// Three properties, all of them load-bearing:
//
//   - a failure is typed, never a panic: a question answers from the graph
//     on disk, a gate turns red
//   - no stampede: a lock, and the loser waits at most opts.Wait, then yields
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
//
// opts.Force skips the probe: the caller already knows the graph cannot stand,
// for a reason the record does not see (an outdated schema, another extractor).
func Refresh(root, extractor string, rebuild Rebuild, opts RefreshOptions) (Status, error) {
	say := func(format string, args ...any) {
		if opts.Notice != nil {
			opts.Notice(fmt.Sprintf(format, args...))
		}
	}
	if opts.Force != "" {
		say("%s", opts.Force)
	} else {
		drift, err := freshness.Probe(root, extractor)
		if err != nil {
			return StatusClean, &ProbeError{err}
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
			return StatusClean, nil
		}
	}
	deadline := time.Now().Add(opts.Wait)
	for !lock(root) {
		if !time.Now().Before(deadline) {
			locked := &LockedError{Path: LockPath(root)}
			if info, err := os.Stat(locked.Path); err == nil {
				locked.Age = time.Since(info.ModTime()).Round(time.Second)
			}
			return StatusClean, locked
		}
		time.Sleep(lockPoll)
	}
	defer unlock(root)
	if err := rebuild(); err != nil {
		return StatusClean, &RebuildError{err}
	}
	return StatusRebuilt, nil
}

// EnsureFresh is Refresh for a question: it never fails, and says why it
// answers from the graph on disk.
func EnsureFresh(root, extractor string, rebuild Rebuild, notice func(string)) {
	_, err := Refresh(root, extractor, rebuild, RefreshOptions{Notice: notice})
	if err == nil || notice == nil {
		return
	}
	var probe *ProbeError
	var locked *LockedError
	var failed *RebuildError
	switch {
	case errors.As(err, &probe):
		notice(fmt.Sprintf("freshness probe failed, answering from the graph on disk: %v", probe.Err))
	case errors.As(err, &locked):
		notice("another run is rebuilding, answering from the graph on disk")
	case errors.As(err, &failed):
		notice(fmt.Sprintf("rebuild failed, answering from the graph on disk: %v", failed.Err))
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
	if statErr != nil || !stale(path, info) {
		return false
	}
	if !takeOver(path) {
		return false
	}
	return lock(root)
}

// stale says whether the lock at path guards nothing any more: older than
// lockStale, or written by a process that is gone. lock writes its number
// right after the create, so a lock without one is a young one or not ours,
// and only its age can tell. A reused number keeps a dead holder's lock alive;
// the age still ends it.
func stale(path string, info fs.FileInfo) bool {
	if time.Since(info.ModTime()) >= lockStale {
		return true
	}
	// A lock gone before the read has no number either.
	b, _ := os.ReadFile(path)
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	return err == nil && !child.Alive(pid)
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
	if err != nil || !stale(claim, got) {
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
