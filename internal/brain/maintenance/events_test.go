package maintenance_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

const oneLine = "C:/repo\tabc123\tdef456\tmain\t2026-09-19T10:00:00Z\n"

func TestReadEventsParsesFiveFields(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d events, want 1", len(got))
	}
	event := got[0]
	if event.Repo != "C:/repo" || event.First != "abc123" || event.Last != "def456" {
		t.Fatalf("event = %+v, want the three fields of the line", event)
	}
	if event.Branch != "main" {
		t.Fatalf("branch = %q, want %q", event.Branch, "main")
	}
	if want := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC); !event.At.Equal(want) {
		t.Fatalf("at = %s, want %s", event.At, want)
	}
}

// The hook writes UTC, but the stamp is read as it stands: a record moved from
// a machine that wrote an offset keeps its offset rather than being relabelled.
func TestReadEventsKeepsTheOffsetOfTheStamp(t *testing.T) {
	stateDir := writeEvents(t, "C:/repo\tabc\tdef\tmain\t2026-09-19T10:00:00+02:00\n")
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if _, offset := got[0].At.Zone(); offset != 2*3600 {
		t.Fatalf("offset = %d seconds, want %d", offset, 2*3600)
	}
}

// A half line is a state that exists: the hook appends from inside a merge and
// must never fail, so a truncated line is skipped rather than raised over. The
// empty line in the middle is the same decision one layer down.
func TestReadEventsSkipsALineWithoutFiveFields(t *testing.T) {
	// The six-field line carries a readable stamp in the fifth field on
	// purpose: with a broken one it would die at the stamp instead, and the
	// "too many fields" half of the count check would go untested.
	stateDir := writeEvents(t, "C:/repo\tabc\n\n"+oneLine+
		"C:/repo\ta\tb\tc\t2026-09-19T10:00:00Z\te\n")
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d events, want 1", len(got))
	}
}

// The second half of the same decision, and it needs its own case: five fields
// and an unreadable stamp is a different line from one with four fields.
func TestReadEventsSkipsALineWhoseStampDoesNotParse(t *testing.T) {
	stateDir := writeEvents(t, "C:/repo\tabc\tdef\tmain\tyesterday\n")
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("read %d events, want none", len(got))
	}
}

// A stamp without a zone is no event here, and Python would have taken it as a
// naive one. Nothing the hook writes reaches this: the difference is recorded
// in the parity list, not relied on.
func TestReadEventsSkipsAStampWithoutAZone(t *testing.T) {
	stateDir := writeEvents(t, "C:/repo\tabc\tdef\tmain\t2026-09-19T10:00:00\n")
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("read %d events, want none", len(got))
	}
}

// No log is no error: on a machine without the hook there is none.
func TestReadEventsWithoutAFile(t *testing.T) {
	got, err := maintenance.ReadEvents(t.TempDir(), "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("read %d events from nothing", len(got))
	}
}

// Not a file is not a log either, and taking a whole reconciliation down over
// it would turn a strange directory into a lost run.
func TestReadEventsWhenADirectorySitsAtThePath(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.MkdirAll(maintenance.EventsPath(stateDir, ""), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("read %d events from a directory", len(got))
	}
}

// Bytes that are not UTF-8 are the one failure this reports: Python ends in a
// UnicodeDecodeError here, and answering "no merges ever happened" would be a
// worse lie than an error naming the file.
func TestReadEventsFailsOnBytesThatAreNotUTF8(t *testing.T) {
	stateDir := writeEvents(t, "C:/repo\tabc\tdef\t\xff\t2026-09-19T10:00:00Z\n")
	if _, err := maintenance.ReadEvents(stateDir, ""); err == nil {
		t.Fatal("ReadEvents accepted bytes that are not UTF-8")
	}
}

func TestReadEventsFailsWhenTheDroppedFileIsNotUTF8(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	writeFile(t, droppedPath(stateDir), "C:/repo\tabc\t\xff\n")
	if _, err := maintenance.ReadEvents(stateDir, ""); err == nil {
		t.Fatal("ReadEvents accepted a dropped file that is not UTF-8")
	}
}

// git runs the hook again for a re-merge, and a second case over the same
// commits would say the same thing twice. The first line wins, so a re-merge
// is dated when it first landed.
func TestReadEventsKeepsTheFirstOfTwoLinesWithOneKey(t *testing.T) {
	stateDir := writeEvents(t, oneLine+"C:/repo\tabc123\tdef456\tmain\t2026-09-20T11:00:00Z\n")
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d events, want 1", len(got))
	}
	if want := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC); !got[0].At.Equal(want) {
		t.Fatalf("at = %s, want the first line's %s", got[0].At, want)
	}
}

// Two different ranges stay two events -- both landed.
func TestReadEventsKeepsTwoRanges(t *testing.T) {
	stateDir := writeEvents(t, oneLine+"C:/repo\tdef456\tghi789\tmain\t2026-09-20T11:00:00Z\n")
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("read %d events, want 2", len(got))
	}
}

// A hook installed by the Python tool writes into its state directory, which
// is this stage's fallback.
func TestReadEventsReadsTheFallbackWhenThePrimaryHasNothing(t *testing.T) {
	fallbackDir := writeEvents(t, oneLine)
	got, err := maintenance.ReadEvents(t.TempDir(), fallbackDir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d events, want 1", len(got))
	}
}

func TestEventsPathPrefersThePrimary(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	fallbackDir := writeEvents(t, oneLine)
	if got := maintenance.EventsPath(stateDir, fallbackDir); got != eventsPath(stateDir) {
		t.Fatalf("EventsPath = %q, want %q", got, eventsPath(stateDir))
	}
}

func TestKeyIsTheRangeInOneRepository(t *testing.T) {
	event := maintenance.MergeEvent{
		Repo: "C:/repo", First: "abc", Last: "def", Branch: "main", At: time.Now(),
	}
	if got, want := event.Key(), [3]string{"C:/repo", "abc", "def"}; got != want {
		t.Fatalf("Key = %v, want %v", got, want)
	}
}

// A dropped event does not come back on the next read.
func TestDropEventHidesIt(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	events, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if err := maintenance.DropEvent(stateDir, events[0]); err != nil {
		t.Fatalf("DropEvent: %v", err)
	}
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a dropped event came back: %+v", got)
	}
}

// The log lives in the old place and the drop goes into the new one: the pair
// has to work across the two directories, or a migrating machine would raise
// every merge it already handled.
func TestDropEventHidesAnEventReadFromTheFallback(t *testing.T) {
	stateDir, fallbackDir := t.TempDir(), writeEvents(t, oneLine)
	events, err := maintenance.ReadEvents(stateDir, fallbackDir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if err := maintenance.DropEvent(stateDir, events[0]); err != nil {
		t.Fatalf("DropEvent: %v", err)
	}
	got, err := maintenance.ReadEvents(stateDir, fallbackDir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a dropped event came back: %+v", got)
	}
}

// The timestamp is not part of the key: a hook that ran twice over the same
// range wrote two different stamps, and both are the one merge.
func TestDropEventHidesTheRangeWhateverTheStampSays(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	events, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	later := events[0]
	later.At = later.At.Add(time.Hour)
	later.Branch = "other"
	if err := maintenance.DropEvent(stateDir, later); err != nil {
		t.Fatalf("DropEvent: %v", err)
	}
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a dropped range came back: %+v", got)
	}
}

// Three fields and no more, byte for byte what the Python writer puts there:
// a half-migrated machine may run both tools over the same file.
func TestDropEventWritesTheTripleAndAppends(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	events, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	second := events[0]
	second.First, second.Last = "def456", "ghi789"
	for _, event := range []maintenance.MergeEvent{events[0], second} {
		if err := maintenance.DropEvent(stateDir, event); err != nil {
			t.Fatalf("DropEvent: %v", err)
		}
	}
	raw, err := os.ReadFile(droppedPath(stateDir))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	want := "C:/repo\tabc123\tdef456\nC:/repo\tdef456\tghi789\n"
	if string(raw) != want {
		t.Fatalf("dropped file = %q, want %q", raw, want)
	}
}

// The log itself is never shortened: the hook holds it open, and a line that
// arrives between the read and the write would otherwise be lost.
func TestDropEventLeavesTheLogAlone(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	events, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if err := maintenance.DropEvent(stateDir, events[0]); err != nil {
		t.Fatalf("DropEvent: %v", err)
	}
	raw, err := os.ReadFile(maintenance.EventsPath(stateDir, ""))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(raw) != oneLine {
		t.Fatalf("the log was rewritten: %q", raw)
	}
}

// A line of the dropped file that is not a triple hides nothing and takes
// nothing down -- the same tolerance the log itself gets.
func TestDropEventFileIgnoresALineWithoutThreeFields(t *testing.T) {
	stateDir := writeEvents(t, oneLine)
	writeFile(t, droppedPath(stateDir), "C:/repo\tabc123\n")
	got, err := maintenance.ReadEvents(stateDir, "")
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("read %d events, want 1", len(got))
	}
}

func TestDropEventFailsWhenAFileSitsWhereTheDirectoryGoes(t *testing.T) {
	stateDir := t.TempDir()
	writeFile(t, filepath.Join(stateDir, "maintenance"), "not a directory")
	if err := maintenance.DropEvent(stateDir, maintenance.MergeEvent{Repo: "C:/repo"}); err == nil {
		t.Fatal("DropEvent wrote into a file")
	}
}

func TestDropEventFailsWhenADirectorySitsAtItsPath(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.MkdirAll(droppedPath(stateDir), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// The refusal of the open itself, naming the path: a write through the
	// handle that never opened would fail too, but with "invalid argument"
	// and no word of where.
	err := maintenance.DropEvent(stateDir, maintenance.MergeEvent{Repo: "C:/repo"})
	var refused *fs.PathError
	if !errors.As(err, &refused) || refused.Path != droppedPath(stateDir) {
		t.Fatalf("err = %v, want the open of %s refused", err, droppedPath(stateDir))
	}
}

// writeEvents answers a state directory whose log holds content.
func writeEvents(t *testing.T, content string) string {
	t.Helper()
	stateDir := t.TempDir()
	writeFile(t, eventsPath(stateDir), content)
	return stateDir
}

func eventsPath(stateDir string) string {
	return filepath.Join(stateDir, "maintenance", "merge-events.tsv")
}

func droppedPath(stateDir string) string {
	return filepath.Join(stateDir, "maintenance", "merge-events.done.tsv")
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
