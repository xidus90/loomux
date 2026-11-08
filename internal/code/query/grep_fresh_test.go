package query

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/spanidx"
	"github.com/xidus90/loomux/internal/code/store"
)

// recordSettings reads the settings the written record carries.
func recordSettings(t *testing.T, root string) []byte {
	t.Helper()
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no corpus record")
	}
	return rec.Settings
}

func TestRebuildRefreshesIndex(t *testing.T) {
	root := grepFixture(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"),
		[]byte("# Repo\n\nzzzFreshNeedle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := Grep(root, "zzzFreshNeedle", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" {
		t.Fatalf("index = %q, want overlay for a small content drift", res.Index)
	}
	if res.TotalHits == 0 {
		t.Fatal("the overlay did not answer")
	}
}

// touchIgnore moves an ignore source without changing what it ignores, a
// drift only a rebuild answers.
func touchIgnore(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, ".gitignore")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, []byte("nothing-matches-this\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLockedRebuildAnswersFromDisk(t *testing.T) {
	root := grepFixture(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"),
		[]byte("# Repo\n\nzzzFreshNeedle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	touchIgnore(t, root)
	lock := ask.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte(fmt.Sprint(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	res, notes, err := Grep(root, "zzzFreshNeedle", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale while the lock is held", res.Index)
	}
	joined := strings.Join(notes, "\n")
	if !strings.Contains(joined, "another run") {
		t.Fatalf("notes = %q, want the lock note", notes)
	}
	if res.TotalHits == 0 {
		t.Fatal("the answer from disk is empty")
	}
}

func TestFlagSettingsNoRebuild(t *testing.T) {
	root := grepFixture(t)
	before := recordSettings(t, root)
	limit := int64(16)
	res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{MaxFilesize: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale", res.Index)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "settings differ from index") {
		t.Fatalf("notes = %q, want the settings note", notes)
	}
	if after := recordSettings(t, root); !bytes.Equal(before, after) {
		t.Fatalf("the flag rewrote the record: %q -> %q", before, after)
	}
}

func TestNoFlagRebuildsToDefault(t *testing.T) {
	root := grepFixture(t)
	limit := int64(16)
	if _, _, err := BuildWith(root, BuildOptions{MaxFilesize: &limit}, nil); err != nil {
		t.Fatal(err)
	}
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "used" {
		t.Fatalf("index = %q, want used after the rebuild", res.Index)
	}
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no corpus record")
	}
	want := corpus.CanonicalSettings(corpus.DefaultMaxFilesize, rec.IgnoreCase)
	if got := rec.Settings; !bytes.Equal(got, want) {
		t.Fatalf("settings = %q, want the default", got)
	}
}

func TestNoFlagNoRefreshKeepsScan(t *testing.T) {
	root := grepFixture(t)
	limit := int64(16)
	if _, _, err := BuildWith(root, BuildOptions{MaxFilesize: &limit}, nil); err != nil {
		t.Fatal(err)
	}
	before := recordSettings(t, root)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale", res.Index)
	}
	if after := recordSettings(t, root); !bytes.Equal(before, after) {
		t.Fatalf("--no-refresh rewrote the record: %q -> %q", before, after)
	}
}

func TestIgnoreEditMakesStale(t *testing.T) {
	root := grepFixture(t)
	body, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"),
		append(body, []byte("big.txt\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale after the ignore edit", res.Index)
	}

	res, _, err = Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "used" {
		t.Fatalf("index = %q, want used after the rebuild", res.Index)
	}
	if slicesHasPath(res, "big.txt") {
		t.Fatal("the rebuilt corpus still searches the now ignored file")
	}
}

func TestGitAddForceMakesStale(t *testing.T) {
	files := grepFiles()
	root := grepGitRepo(t, files, nil)
	if _, _, err := BuildWith(root, BuildOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	if res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{NoRefresh: true}); err != nil {
		t.Fatal(err)
	} else if slicesHasPath(res, "ignored.log") {
		t.Fatal("an untracked ignored file is in the corpus")
	}

	grepRunGit(t, root, "add", "-f", "ignored.log")
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale after git add -f", res.Index)
	}

	res, _, err = Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "used" {
		t.Fatalf("index = %q, want used after the rebuild", res.Index)
	}
	if !slicesHasPath(res, "ignored.log") {
		t.Fatal("the force-added file did not enter the search corpus")
	}
}

func TestNoRecordRebuildsBeforeAnswer(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(corpus.RecordPath(root)); err != nil {
		t.Fatal(err)
	}
	res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := corpus.Read(root); !ok {
		t.Fatal("the rebuild did not write a record")
	}
	if res.Index != "used" {
		t.Fatalf("index = %q, want used after the rebuild (notes %v)", res.Index, notes)
	}
	if res.TotalHits == 0 {
		t.Fatal("the rebuilt corpus is empty")
	}
}

func TestNoRecordNoRefreshScans(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(corpus.RecordPath(root)); err != nil {
		t.Fatal(err)
	}
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale without a record and with --no-refresh", res.Index)
	}
	if res.TotalHits == 0 {
		t.Fatal("the scan is empty")
	}
}

func TestNoRecordLockedRebuildScans(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(corpus.RecordPath(root)); err != nil {
		t.Fatal(err)
	}
	lock := ask.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lock, []byte(fmt.Sprint(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale while the lock is held", res.Index)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "another run") {
		t.Fatalf("notes = %q, want the lock note", notes)
	}
}

// overrideGraph replaces the graph read for one test and restores it after.
func overrideGraph(t *testing.T, read func(string) (*model.Graph, error)) {
	t.Helper()
	old := readGraph
	readGraph = read
	t.Cleanup(func() { readGraph = old })
}

func failGraph(string) (*model.Graph, error) { return nil, errors.New("graph broken") }

func spansPath(root string) string { return store.CachePath(root, spanidx.Name) }

// symbolGroups is the JSON of the groups of an answer, the part the symbols
// decide.
func symbolGroups(t *testing.T, res grep.Result) string {
	t.Helper()
	raw, err := json.Marshal(res.Groups)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestGrepUsesTheSpansSidecar(t *testing.T) {
	root := grepFixture(t)
	overrideGraph(t, failGraph)
	res, _, err := Grep(root, "Run", GrepOptions{})
	if err != nil {
		t.Fatalf("the sidecar must answer without the graph: %v", err)
	}
	if !strings.Contains(symbolGroups(t, res), `"name":"Run"`) {
		t.Fatalf("groups = %s, want the symbol Run", symbolGroups(t, res))
	}
}

func TestGrepFallsBackToTheGraphWithoutSidecar(t *testing.T) {
	root := grepFixture(t)
	with, _, err := Grep(root, "Run", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(spansPath(root)); err != nil {
		t.Fatal(err)
	}
	without, _, err := Grep(root, "Run", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if a, b := symbolGroups(t, with), symbolGroups(t, without); a != b {
		t.Fatalf("groups differ:\nwith    %s\nwithout %s", a, b)
	}
	if _, err := os.Stat(spansPath(root)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a missing sidecar must not rebuild: %v", err)
	}
}

func TestGrepReportsAGraphReadFailureWithoutSidecar(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(spansPath(root)); err != nil {
		t.Fatal(err)
	}
	overrideGraph(t, failGraph)
	if _, _, err := Grep(root, "Run", GrepOptions{}); err == nil {
		t.Fatal("the graph read error must surface")
	}
}

// A digest that is not 32 hex bytes binds neither the index nor the sidecar.
func TestGrepTreatsAnUndecodableDigestAsStale(t *testing.T) {
	root := grepFixture(t)
	want, _, err := Grep(root, "Run", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	overrideRead(t, func(r string) (corpus.Record, bool) {
		rec, ok := corpus.Read(r)
		rec.Digest = "zz"
		return rec, ok
	})
	res, _, err := Grep(root, "Run", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale", res.Index)
	}
	if a, b := symbolGroups(t, want), symbolGroups(t, res); a != b {
		t.Fatalf("groups differ:\nsidecar %s\ngraph   %s", a, b)
	}
}

// Windows refuses to truncate a file while a view of it is mapped, even where
// it lets a mapped file be removed or replaced.
func TestGrepClosesTheIndex(t *testing.T) {
	root := grepFixture(t)
	if _, _, err := Grep(root, "Run", GrepOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{indexPath(root), spansPath(root)} {
		if err := os.Truncate(path, 0); err != nil {
			t.Fatalf("still mapped after the answer: %v", err)
		}
	}
}

func TestGrepRebuildsAGraphWithoutFingerprint(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(freshness.Path(root)); err != nil {
		t.Fatal(err)
	}
	res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "no freshness record, building the graph") {
		t.Fatalf("notes = %q, want the graph rebuild", notes)
	}
	if _, err := os.Stat(freshness.Path(root)); err != nil {
		t.Fatalf("the rebuild wrote no fingerprint: %v", err)
	}
	if res.Index != "used" {
		t.Fatalf("index = %q, want used after the rebuild", res.Index)
	}
}

func TestNeedsRebuild(t *testing.T) {
	root := grepFixture(t)
	clean := grepState{hasRecord: true, corpusDrift: &corpus.Drift{}, graphDrift: &freshness.Drift{}}
	cases := []struct {
		name   string
		root   string
		modify func(s *grepState)
		want   string
	}{
		{"clean", root, func(*grepState) {}, ""},
		{"no fingerprint", root, func(s *grepState) { s.graphDrift = nil }, "no freshness record, building the graph"},
		{"graph drift", root, func(s *grepState) { s.graphDrift = &freshness.Drift{Changed: []string{"a.go"}} },
			"1 files moved, rebuilding the graph"},
		{"no ask index", t.TempDir(), func(*grepState) {}, "no ask index, rebuilding the graph"},
		{"no record", root, func(s *grepState) { s.hasRecord, s.corpusDrift = false, nil }, "no search corpus record, rebuilding"},
		{"corpus drift", root, func(s *grepState) { s.corpusDrift = &corpus.Drift{Added: []string{"a", "b"}} },
			"search corpus moved (2 entries), rebuilding"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := clean
			c.modify(&s)
			if got := needsRebuild(s, c.root); got != c.want {
				t.Fatalf("reason = %q, want %q", got, c.want)
			}
		})
	}
}

// A flag that only moves the settings does not spare a drifted graph its
// rebuild: the answer would otherwise carry the old graph's symbols.
func TestGrepFlagWithoutRecordRebuildsWithTheDefault(t *testing.T) {
	root := grepFixture(t)
	overrideRead(t, func(string) (corpus.Record, bool) { return corpus.Record{}, false })
	limit := int64(16)
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{MaxFilesize: &limit}); err != nil {
		t.Fatal(err)
	}
	want := corpus.CanonicalSettings(corpus.DefaultMaxFilesize, true)
	if got := recordSettings(t, root); !bytes.Equal(got, want) {
		t.Fatalf("record settings = %s, want the default %s", got, want)
	}
}

func TestGrepFlagSettingsStillRebuildADriftedGraph(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(freshness.Path(root)); err != nil {
		t.Fatal(err)
	}
	before := recordSettings(t, root)
	limit := int64(16)
	res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{MaxFilesize: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "building the graph") {
		t.Fatalf("notes = %q, want the graph rebuild", notes)
	}
	// The flag never rewrites the record: the rebuild keeps its limit, and the
	// answer is a scan at the flag's.
	if after := recordSettings(t, root); !bytes.Equal(after, before) {
		t.Fatalf("record settings = %s, want them unchanged (%s)", after, before)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale", res.Index)
	}
	if res.SkippedSize == 0 {
		t.Fatal("the counters must describe the scan at the flag's limit, not the record's")
	}
	if !strings.Contains(strings.Join(notes, "\n"), "settings differ from index") {
		t.Fatalf("notes = %q, want the settings difference said", notes)
	}
}
