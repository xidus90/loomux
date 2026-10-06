package maintenance_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// scan is Scan over the one area of a one-area world.
func scan(t *testing.T, w areaWorld) (int, int, map[string]maintenance.Changed) {
	t.Helper()
	checked, hashed, changed, err := maintenance.Scan(w.Area, w.Manifest, w.StateDir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	return checked, hashed, changed
}

// only is the single entry of changed, which most cases have exactly one of.
func only(t *testing.T, changed map[string]maintenance.Changed) maintenance.Changed {
	t.Helper()
	if len(changed) != 1 {
		t.Fatalf("changed = %d entries, want 1", len(changed))
	}
	for _, item := range changed {
		return item
	}
	panic("unreachable")
}

// statsPath is where the cache of this world's one area lies. Derived rather
// than spelt out, because a world with a second scope is one `addArea` away.
func statsPath(w areaWorld) string {
	return filepath.Join(w.StateDir, "maintenance",
		search.CollectionName(w.Area.Scope), "stats.tsv")
}

func readStatsFile(t *testing.T, w areaWorld) string {
	t.Helper()
	data, err := os.ReadFile(statsPath(w))
	if err != nil {
		t.Fatalf("ReadFile stats: %v", err)
	}
	return string(data)
}

// A register that cannot be inspected is refused, not read as empty: an
// empty one would report an area without a single source.
func TestScanRefusesARegisterThatCannotBeInspected(t *testing.T) {
	// The area itself is sound: only the register, which a read-only area
	// keeps in the state directory, cannot be reached.
	area := config.Area{Scope: "project/odd", Path: t.TempDir(), ReadOnly: true}
	checked, hashed, changed, err := maintenance.Scan(area, &config.Manifest{}, filepath.Join(t.TempDir(), "a\x00b"))
	if err == nil || checked != 0 || hashed != 0 || changed != nil {
		t.Fatalf("got %d, %d, %v, %v", checked, hashed, changed, err)
	}
	// The register is what failed, not a later step that shares the path.
	if !strings.Contains(err.Error(), "_identities.tsv") {
		t.Fatalf("error does not name the register: %v", err)
	}
}

// A source whose digest still matches the register has not moved: it is
// counted, and it stands in nothing.
func TestScanFindsNothingWhenNothingMoved(t *testing.T) {
	w := newArea(t, map[string]string{"src/a.go": "package a\n"})

	checked, hashed, changed := scan(t, w)

	if checked != 1 {
		t.Fatalf("checked = %d, want 1", checked)
	}
	if hashed != 1 {
		t.Fatalf("hashed = %d, want 1", hashed)
	}
	if len(changed) != 0 {
		t.Fatalf("changed = %+v, want none", changed)
	}
}

// A source whose content has moved stands in changed with its new digest and
// its text.
func TestScanReportsAChangedSource(t *testing.T) {
	w := newArea(t, map[string]string{"src/a.go": "package a\n"})
	writeUnder(t, w.Area.Path, "src/a.go", "package a\n\nfunc B() {}\n")

	_, hashed, changed := scan(t, w)

	if hashed != 1 {
		t.Fatalf("hashed = %d, want 1", hashed)
	}
	item := only(t, changed)
	if item.Relative != "src/a.go" {
		t.Fatalf("relative = %q", item.Relative)
	}
	if item.Text != "package a\n\nfunc B() {}\n" {
		t.Fatalf("text = %q", item.Text)
	}
	if item.Revision != 1 {
		t.Fatalf("revision = %d, want 1", item.Revision)
	}
	if !strings.HasPrefix(item.ContentHash, "sha256:") {
		t.Fatalf("content hash = %q", item.ContentHash)
	}
	if changed[item.DocID].Relative != item.Relative {
		t.Fatalf("changed is not keyed by doc id: %+v", changed)
	}
}

// Rule 1: a stamp that matches spares the hash. The first run has no cache and
// hashes; the second finds its own stamp and does not.
func TestScanUsesTheStampCache(t *testing.T) {
	w := newArea(t, map[string]string{"src/a.go": "package a\n"})

	if _, hashed, _ := scan(t, w); hashed != 1 {
		t.Fatalf("first run hashed = %d, want 1", hashed)
	}
	if _, hashed, _ := scan(t, w); hashed != 0 {
		t.Fatalf("second run hashed = %d, want 0", hashed)
	}
}

// Rule 1, the half the cache exists for: only the fresh are written. A changed
// source that entered the cache would be reported once and never again, because
// its stamp would match on every later run while its content still differs from
// the register.
func TestScanKeepsAChangedSourceOutOfTheStampCache(t *testing.T) {
	w := newArea(t, map[string]string{"src/a.go": "package a\n"})
	writeUnder(t, w.Area.Path, "src/a.go", "package a\n\nfunc B() {}\n")

	if _, hashed, changed := scan(t, w); hashed != 1 || len(changed) != 1 {
		t.Fatalf("first run: hashed = %d, changed = %d", hashed, len(changed))
	}
	_, hashed, changed := scan(t, w)
	if hashed != 1 {
		t.Fatalf("second run hashed = %d, want 1", hashed)
	}
	if len(changed) != 1 {
		t.Fatalf("second run changed = %d entries, want 1", len(changed))
	}
	if strings.Contains(readStatsFile(t, w), "src/a.go") {
		t.Fatalf("the changed source entered the cache:\n%s", readStatsFile(t, w))
	}
}

// Both halves of the stamp decide, and each gets its own case: a cache that
// compared the size alone would let an edit of the same length through.
func TestScanHashesAgainWhenTheSizeChanged(t *testing.T) {
	w := newArea(t, map[string]string{"src/a.go": "package a\n"})
	scan(t, w)

	path := filepath.Join(w.Area.Path, "src", "a.go")
	stamp := modTime(t, path)
	writeUnder(t, w.Area.Path, "src/a.go", "package a\n\n")
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if _, hashed, _ := scan(t, w); hashed != 1 {
		t.Fatalf("hashed = %d, want 1", hashed)
	}
}

// And a cache that compared the mtime alone would miss nothing here, but would
// hash a file whose stamp a touch moved -- the case that proves the mtime is
// read at all.
func TestScanHashesAgainWhenTheMtimeChanged(t *testing.T) {
	w := newArea(t, map[string]string{"src/a.go": "package a\n"})
	scan(t, w)

	path := filepath.Join(w.Area.Path, "src", "a.go")
	moved := modTime(t, path).Add(-2 * time.Hour)
	if err := os.Chtimes(path, moved, moved); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if _, hashed, _ := scan(t, w); hashed != 1 {
		t.Fatalf("hashed = %d, want 1", hashed)
	}
}

func modTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	return info.ModTime()
}

// Rule 2: a vanished source is `source_missing`, which the index run's rename
// detection decides on -- not a change of content. It is not even counted.
func TestScanSkipsAVanishedSource(t *testing.T) {
	w := newArea(t, map[string]string{"src/a.go": "package a\n"})
	if err := os.Remove(filepath.Join(w.Area.Path, "src", "a.go")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	checked, hashed, changed := scan(t, w)

	if checked != 0 || hashed != 0 {
		t.Fatalf("checked = %d, hashed = %d, want 0, 0", checked, hashed)
	}
	if len(changed) != 0 {
		t.Fatalf("changed = %+v, want none", changed)
	}
	if got := readStatsFile(t, w); got != "pfad\tmtime_ns\tsize\n" {
		t.Fatalf("stats = %q, want the header alone", got)
	}
}

// A file the walk finds but the register does not know is not this stage's
// business either: the register is what is iterated, and a new file is the
// index run's to name.
func TestScanIgnoresAnUnregisteredFile(t *testing.T) {
	w := newAreaWith(t, areaOptions{
		Files:      map[string]string{"src/a.go": "package a\n", "src/b.go": "package b\n"},
		Registered: []string{"src/a.go"},
	})

	checked, _, changed := scan(t, w)

	if checked != 1 {
		t.Fatalf("checked = %d, want 1", checked)
	}
	if len(changed) != 0 {
		t.Fatalf("changed = %+v, want none", changed)
	}
}

// Rule 3: only CRLF is folded, never a lone CR. `identity.ContentHash` folds
// exactly that much, and decoding the two sides of a diff by different rules
// would make the package claim a change to a line nobody touched.
func TestScanFoldsOnlyCRLF(t *testing.T) {
	w := newArea(t, map[string]string{
		"crlf.txt": "a\r\nb\n",
		"lf.txt":   "a\nb\n",
		"cr.txt":   "a\rb\n",
	})
	for _, name := range []string{"crlf.txt", "lf.txt", "cr.txt"} {
		writeUnder(t, w.Area.Path, name, "x\r\ny\rz\n")
	}

	_, _, changed := scan(t, w)

	if len(changed) != 3 {
		t.Fatalf("changed = %d entries, want 3", len(changed))
	}
	for _, item := range changed {
		if item.Text != "x\ny\rz\n" {
			t.Fatalf("%s: text = %q, want the CRLF folded and the CR kept", item.Relative, item.Text)
		}
	}
}

// Rule 4, the half that has to work: where the committed bytes hash to what the
// register describes, they are the baseline.
func TestScanReadsTheBaselineFromHead(t *testing.T) {
	w := newAreaWith(t, areaOptions{Git: true, Files: map[string]string{"a.md": "first\n"}})
	writeUnder(t, w.Area.Path, "a.md", "second\n")

	item := only(t, third(scan(t, w)))

	if item.Baseline == nil {
		t.Fatal("baseline = nil, want the committed text")
	}
	if *item.Baseline != "first\n" {
		t.Fatalf("baseline = %q", *item.Baseline)
	}
}

// The committed bytes are folded the way the register folds them before they
// are hashed against it. Without that, every source committed with CRLF would
// lose its baseline, because the register's hash is taken over LF.
func TestScanFoldsTheBaselineLikeTheRegister(t *testing.T) {
	w := newAreaWith(t, areaOptions{Git: true, Files: map[string]string{"a.md": "first\r\nline\r\n"}})
	writeUnder(t, w.Area.Path, "a.md", "second\n")

	item := only(t, third(scan(t, w)))

	if item.Baseline == nil {
		t.Fatal("baseline = nil, want the committed text")
	}
	if *item.Baseline != "first\nline\n" {
		t.Fatalf("baseline = %q, want the CRLF folded", *item.Baseline)
	}
}

// Rule 4, the half that is the point: a hand commit between two approvals puts
// a third state in HEAD, and believing it would describe a change that never
// happened.
func TestBaselineIsEmptyWhenHeadDisagrees(t *testing.T) {
	w := newAreaWith(t, areaOptions{Git: true, Files: map[string]string{"a.md": "first\n"}})
	writeUnder(t, w.Area.Path, "a.md", "by hand\n")
	gitRun(t, w.Area.Path, "add", "--all")
	gitCommit(t, w.Area.Path, "a hand commit between two approvals")
	writeUnder(t, w.Area.Path, "a.md", "second\n")

	item := only(t, third(scan(t, w)))

	if item.Baseline != nil {
		t.Fatalf("baseline = %q, want none", *item.Baseline)
	}
}

// A vault without git is a legitimate setup: no baseline, no error.
func TestScanHasNoBaselineWithoutARepository(t *testing.T) {
	requireGit(t)
	w := newArea(t, map[string]string{"a.md": "first\n"})
	writeUnder(t, w.Area.Path, "a.md", "second\n")

	item := only(t, third(scan(t, w)))

	if item.Baseline != nil {
		t.Fatalf("baseline = %q, want none", *item.Baseline)
	}
}

// The same without a repository for a source registered empty: git has no
// version to hand over, and the hash of nothing happening to equal the
// register's does not make nothing a baseline.
func TestScanHasNoBaselineForAnEmptySourceWithoutARepository(t *testing.T) {
	requireGit(t)
	w := newArea(t, map[string]string{"a.md": ""})
	writeUnder(t, w.Area.Path, "a.md", "no longer empty\n")

	item := only(t, third(scan(t, w)))

	if item.Baseline != nil {
		t.Fatalf("baseline = %q, want none", *item.Baseline)
	}
}

// A cache that did not change is not written again: it keeps its own
// timestamp, the way `write_if_changed` leaves an untouched file.
func TestScanLeavesAnUnchangedStampFileAlone(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n"})
	scan(t, w)
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(statsPath(w), past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	scan(t, w)
	info, err := os.Stat(statsPath(w))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.ModTime().Equal(past) {
		t.Fatalf("the unchanged cache was written again at %v", info.ModTime())
	}
}

// A committed empty file is a baseline that happens to be empty, and it is not
// the same answer as "there is none": the package draws a real diff from the
// one and prints its no-baseline note over the other. A `string` field with ""
// for both would fold them together here, where vcs.ShowBlob went out of its
// way to keep them apart.
func TestScanDistinguishesAnEmptyBaselineFromNone(t *testing.T) {
	w := newAreaWith(t, areaOptions{Git: true, Files: map[string]string{"a.md": ""}})
	writeUnder(t, w.Area.Path, "a.md", "no longer empty\n")

	item := only(t, third(scan(t, w)))

	if item.Baseline == nil {
		t.Fatal("baseline = nil, want an empty baseline")
	}
	if *item.Baseline != "" {
		t.Fatalf("baseline = %q, want empty", *item.Baseline)
	}
}

// A read-only area keeps its register in the state directory, because writing
// a catalog into it would be the first forbidden write. Reading it from the
// area would find nothing there and report an area without a single source.
func TestScanReadsTheRegisterOfAReadOnlyArea(t *testing.T) {
	w := newAreaWith(t, areaOptions{ReadOnly: true, Files: map[string]string{"a.md": "first\n"}})
	if _, err := os.Stat(filepath.Join(w.Area.Path, "_identities.tsv")); err == nil {
		t.Fatal("the fixture wrote the register into the area")
	}
	writeUnder(t, w.Area.Path, "a.md", "second\n")

	checked, _, changed := scan(t, w)

	if checked != 1 {
		t.Fatalf("checked = %d, want 1", checked)
	}
	if only(t, changed).Relative != "a.md" {
		t.Fatalf("changed = %+v", changed)
	}
}

// Two areas of one world never share a doc id. This is a promise of the test
// world rather than of Scan, and it is pinned here because Reconcile merges the
// findings of several areas: with a counter that restarted per area, the first
// file of the second area would carry the first area's first key, and a merge
// by doc id would drop one of the two without a word.
func TestTheWorldGivesEveryAreaItsOwnDocIDs(t *testing.T) {
	w := newWorld(t)
	first, firstManifest := w.addArea(t, areaOptions{
		Scope: "project/one",
		Files: map[string]string{"a.md": "one\n"},
	})
	second, secondManifest := w.addArea(t, areaOptions{
		Scope: "project/two",
		Files: map[string]string{"a.md": "two\n"},
	})
	writeUnder(t, first.Path, "a.md", "one changed\n")
	writeUnder(t, second.Path, "a.md", "two changed\n")

	seen := map[string]string{}
	for _, item := range []struct {
		area     config.Area
		manifest *config.Manifest
	}{{first, firstManifest}, {second, secondManifest}} {
		_, _, changed, err := maintenance.Scan(item.area, item.manifest, w.StateDir)
		if err != nil {
			t.Fatalf("Scan %s: %v", item.area.Scope, err)
		}
		for docID := range changed {
			if other, clash := seen[docID]; clash {
				t.Fatalf("%s and %s share the doc id %q", other, item.area.Scope, docID)
			}
			seen[docID] = item.area.Scope
		}
	}
	if len(seen) != 2 {
		t.Fatalf("saw %d doc ids, want 2", len(seen))
	}
}

// A register a run cannot read is refused rather than taken for empty: taking
// it for empty would report an area of no sources at all, which reads as "all
// is well" and is the one wrong answer here.
func TestScanRefusesABrokenRegister(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n"})
	w.writeRegisterText(t, w.Area, "doc_id\tpfad\tcontent_hash\trevision\nonly\ttwo\n")

	_, _, _, err := maintenance.Scan(w.Area, w.Manifest, w.StateDir)

	if err == nil {
		t.Fatal("Scan accepted a broken register")
	}
}

// The walk's own refusal is passed on: an area whose path is gone is not an
// area with no files.
func TestScanRefusesAnAreaThatIsGone(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n"})
	area := w.Area
	area.Path = filepath.Join(area.Path, "gone")

	_, _, _, err := maintenance.Scan(area, w.Manifest, w.StateDir)

	if err == nil {
		t.Fatal("Scan accepted an area that is gone")
	}
}

// The cache is written under `maintenance/<collection>/`, a directory that does
// not exist on the first run. A file standing where that directory belongs is
// the one way to see the MkdirAll refused.
func TestScanRefusesWhenTheStatsDirectoryCannotBeMade(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n"})
	blocker := filepath.Join(w.StateDir, "maintenance", "project-one")
	if err := os.MkdirAll(filepath.Dir(blocker), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, _, err := maintenance.Scan(w.Area, w.Manifest, w.StateDir)

	if err == nil {
		t.Fatal("Scan wrote its cache over a file")
	}
}

// And a directory standing where the file belongs is the one way to see the
// swap refused.
func TestScanRefusesWhenTheStatsFileCannotBeWritten(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n"})
	if err := os.MkdirAll(statsPath(w), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	_, _, _, err := maintenance.Scan(w.Area, w.Manifest, w.StateDir)

	if err == nil {
		t.Fatal("Scan wrote its cache over a directory")
	}
}

// A damaged cache line is dropped and not fatal: the file is a throwaway, so
// the worst a bad line costs is one extra hash, while refusing over it would
// take the whole run down for something the run can simply rebuild.
func TestScanDropsDamagedCacheLines(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n", "b.md": "second\n"})
	scan(t, w)
	good := readStatsFile(t, w)

	writeStats(t, w, strings.Join([]string{
		"pfad\tmtime_ns\tsize",
		"a.md\t1\t2\t3",
		"b.md\tnot a number\t7",
		"",
	}, "\n"))

	if _, hashed, _ := scan(t, w); hashed != 2 {
		t.Fatalf("hashed = %d, want both sources hashed again", hashed)
	}
	if readStatsFile(t, w) != good {
		t.Fatalf("the rebuilt cache differs from the one before it was damaged")
	}
}

// A cache entry whose numbers are out of range is a damaged line too. Python's
// int() has no ceiling; this side drops the line rather than carry a wrapped
// stamp that could match a real file by accident.
func TestScanDropsACacheLineOutOfRange(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n"})
	writeStats(t, w, "pfad\tmtime_ns\tsize\na.md\t99999999999999999999\t7\n")

	if _, hashed, _ := scan(t, w); hashed != 1 {
		t.Fatalf("hashed = %d, want 1", hashed)
	}
}

// The cache file this run writes is the one the next run reads: the stamps in
// it match the files on disk, so a second run hashes nothing.
func TestScanWritesTheStampsItRead(t *testing.T) {
	w := newArea(t, map[string]string{"a.md": "first\n"})
	scan(t, w)

	info, err := os.Stat(filepath.Join(w.Area.Path, "a.md"))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	want := "pfad\tmtime_ns\tsize\na.md\t" +
		strconv.FormatInt(info.ModTime().UnixNano(), 10) + "\t" + strconv.FormatInt(info.Size(), 10) + "\n"
	if got := readStatsFile(t, w); got != want {
		t.Fatalf("stats = %q, want %q", got, want)
	}
}

func writeStats(t *testing.T, w areaWorld, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(statsPath(w)), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(statsPath(w), []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile stats: %v", err)
	}
}

// third drops the counters of a scan so a case that only cares about the one
// changed source reads as one line.
func third(_, _ int, changed map[string]maintenance.Changed) map[string]maintenance.Changed {
	return changed
}
