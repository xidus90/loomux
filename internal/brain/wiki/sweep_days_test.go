package wiki

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	_ "time/tzdata"
)

// Python subtracts `timedelta(days=n)`, which is exactly n * 24 hours. A
// calendar step instead lands on the same wall-clock time n days back and,
// across a daylight-saving change, an hour away from it -- so a page modified
// inside that hour would be reported by one and not by the other.
func TestTheAgeRulesCountDaysAsTwentyFourHours(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	// Noon on the day the clocks went forward: one calendar day back is
	// 11:00 UTC, 24 hours back is 10:00 UTC.
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, berlin)
	root := t.TempDir()
	sweepWrite(t, root, "index.md", "# c\n* [a](a.md)\n")
	sweepWrite(t, root, "a.md", "---\ntype: Topic\nrealization: planned\n---\n")
	touched := time.Date(2026, 3, 28, 10, 30, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, "a.md"), touched, touched); err != nil {
		t.Fatal(err)
	}
	found, err := SweepBundle(SweepContext{Root: root, UntouchedDays: 1, Now: now, IsProject: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		if f.Rule == "untouched" || f.Rule == "long-planned" {
			t.Errorf("a page 23.5 hours old was reported: %s: %s", f.Rule, f.Message)
		}
	}
}
