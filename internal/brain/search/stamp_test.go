package search_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
)

// stampWorld writes content as the reconcile stamp of a fresh state directory.
func stampWorld(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "maintenance"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "maintenance", "last-run.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestReconcileConstantsAreThePythonOnes(t *testing.T) {
	if search.ReconcileInterval != 24*time.Hour {
		t.Errorf("interval %v", search.ReconcileInterval)
	}
	if search.ReconcileAdvice != "run `brain reconcile`" {
		t.Errorf("advice %q", search.ReconcileAdvice)
	}
}

// Expected values measured against the reference on 2026-09-15:
// datetime.fromisoformat(text.strip()) and .isoformat(), None for naive stamps.
func TestReadLastRunReadsWhatPythonReads(t *testing.T) {
	for _, tc := range []struct {
		name, content, iso string
		ok                 bool
	}{
		{"writer form", "2026-09-13T08:00:00+00:00\n", "2026-09-13T08:00:00+00:00", true},
		{"padded with microseconds and an offset", "  2026-09-13T08:00:00.123456+02:00\r\n", "2026-09-13T08:00:00.123456+02:00", true},
		{"zulu", "2026-09-13T08:00:00Z", "2026-09-13T08:00:00+00:00", true},
		{"three fraction digits", "2026-09-13T08:00:00.500+00:00", "2026-09-13T08:00:00.500000+00:00", true},
		{"naive", "2026-09-13T08:00:00\n", "", false},
		{"garbage", "garbage", "", false},
		{"byte order mark", "\xef\xbb\xbf2026-09-13T08:00:00+00:00", "", false},
		{"not utf-8", "\xff2026-09-13T08:00:00+00:00", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stamp, ok, err := search.ReadLastRun(stampWorld(t, tc.content), "")
			if err != nil {
				t.Fatal(err)
			}
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && pytext.IsoFormat(stamp) != tc.iso {
				t.Fatalf("isoformat %q, want %q", pytext.IsoFormat(stamp), tc.iso)
			}
			if !ok && !stamp.IsZero() {
				t.Fatalf("no stamp must come back zero, got %v", stamp)
			}
		})
	}
}

func TestReadLastRunWithoutAStampIsNoStamp(t *testing.T) {
	stamp, ok, err := search.ReadLastRun(t.TempDir(), "")
	if err != nil || ok || !stamp.IsZero() {
		t.Fatalf("got %v %v %v", stamp, ok, err)
	}
}

func TestReadLastRunOfAStampThatCannotBeReadIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "maintenance", "last-run.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := search.ReadLastRun(dir, ""); err == nil || ok {
		t.Fatalf("a directory in the stamp's place must be an error, got ok=%v err=%v", ok, err)
	}
}

// Measured: a difference of exactly 24 h is stale, 24 h less one microsecond is not.
func TestStaleCountsTheFullDay(t *testing.T) {
	stamp := time.Date(2026, 9, 13, 8, 0, 0, 0, time.UTC)
	if !search.Stale(stamp, stamp.Add(24*time.Hour)) {
		t.Error("exactly one day must be stale")
	}
	if search.Stale(stamp, stamp.Add(24*time.Hour-time.Microsecond)) {
		t.Error("one microsecond short of a day must not be stale")
	}
	if search.Stale(stamp, stamp.Add(-time.Hour)) {
		t.Error("a stamp from the future must not be stale")
	}
}

func TestStaleReconcileNamesAnAgedStampOnly(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		dir  string
		want []string
	}{
		{"aged", stampWorld(t, "2000-01-01T00:00:00+00:00\n"), []string{
			"the last full reconciliation was 2000-01-01T00:00:00+00:00, more than 24 hours ago: a source may have changed without this answer knowing (run `brain reconcile`)",
		}},
		{"fresh", stampWorld(t, "2999-01-01T00:00:00+00:00\n"), nil},
		{"naive and aged", stampWorld(t, "2000-01-01T00:00:00\n"), nil},
		{"missing", t.TempDir(), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := search.StaleReconcile(tc.dir, "", now)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStaleReconcileHandsAReadErrorOn(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "maintenance", "last-run.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := search.StaleReconcile(dir, "", time.Now()); err == nil || got != nil {
		t.Fatalf("got %q, %v", got, err)
	}
}
