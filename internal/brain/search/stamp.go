package search

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// ReconcileInterval is core.RECONCILE_INTERVAL: a full reconciliation older than one day is
// reported by search and by status alike.
const ReconcileInterval = 24 * time.Hour

// ReconcileAdvice is core._RECONCILE_ADVICE; the backticks belong to the text. It still
// names the Python command, although `loomux reconcile` exists since stage 3a: the recorded
// cases of stage 1b-1 hold this text. Once loomux writes the stamp, `brain reconcile` writes
// only into the old directory, which loomux no longer reads first: followed, the advice
// would never end the warning it comes with. It moves to `loomux reconcile` with the switch.
const ReconcileAdvice = "run `brain reconcile`"

// ReadLastRun reads maintenance/last-run.txt as reconcile.read_last_run does: from
// stateDir, with fallbackDir -- ultra-brain's -- as the fallback. Both are arguments
// and neither is read from the environment, so that a caller's state directory is the
// only one this reads (internal/serve's promise). A
// stamp that is missing, not UTF-8, unreadable as an ISO time or without a zone is no stamp
// (false, nil): Python answers None for all four, UnicodeDecodeError being a ValueError there.
// Only a stamp that exists and cannot be read is an error. Existence is Path.exists, which
// swallows every stat error.
func ReadLastRun(stateDir, fallbackDir string) (time.Time, bool, error) {
	lookup := config.ArtifactLookup{Primary: stateDir, Fallback: fallbackDir}
	path := lookup.Resolve(filepath.Join("maintenance", "last-run.txt"))
	if _, err := os.Stat(path); err != nil {
		return time.Time{}, false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, false, err
	}
	if !utf8.Valid(data) {
		return time.Time{}, false, nil
	}
	stamp, ok := pytext.ParseAwareIsoFormat(pytext.Strip(string(data)))
	if !ok {
		return time.Time{}, false, nil
	}
	return stamp, true, nil
}

// Stale is core._stale with the clock handed in: a full day or more, the day itself included.
func Stale(stamp, now time.Time) bool {
	return now.Sub(stamp) >= ReconcileInterval
}

// StaleReconcile is core.stale_reconcile: the one finding a search carries for a stamp that
// exists and has aged, never for a missing one.
func StaleReconcile(stateDir, fallbackDir string, now time.Time) ([]string, error) {
	stamp, ok, err := ReadLastRun(stateDir, fallbackDir)
	if err != nil {
		return nil, err
	}
	if !ok || !Stale(stamp, now) {
		return nil, nil
	}
	return []string{fmt.Sprintf(
		"the last full reconciliation was %s, more than 24 hours ago: "+
			"a source may have changed without this answer knowing (%s)",
		pytext.IsoFormat(stamp), ReconcileAdvice,
	)}, nil
}
