package search

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// ReconcileInterval is core.RECONCILE_INTERVAL: a full reconciliation older than one day is
// reported by search and by status alike.
const ReconcileInterval = 24 * time.Hour

// ReconcileAdvice is core._RECONCILE_ADVICE; the backticks belong to the text. Until stage 3
// it names the Python command the user has.
const ReconcileAdvice = "run `brain reconcile`"

// ReadLastRun reads <stateDir>/maintenance/last-run.txt as reconcile.read_last_run does. A
// stamp that is missing, not UTF-8, unreadable as an ISO time or without a zone is no stamp
// (false, nil): Python answers None for all four, UnicodeDecodeError being a ValueError there.
// Only a stamp that exists and cannot be read is an error. Existence is Path.exists, which
// swallows every stat error.
func ReadLastRun(stateDir string) (time.Time, bool, error) {
	path := filepath.Join(stateDir, "maintenance", "last-run.txt")
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
func StaleReconcile(stateDir string, now time.Time) ([]string, error) {
	stamp, ok, err := ReadLastRun(stateDir)
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
