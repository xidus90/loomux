package query

import (
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/spanidx"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/code/trigram"
	"github.com/xidus90/loomux/internal/lock"
	"github.com/xidus90/loomux/internal/testlock"
)

// built writes the sample repository, builds it and returns root and stats.
func builtWith(t *testing.T, opts BuildOptions) (string, Stats) {
	t.Helper()
	root := repo(t, sample())
	_, stats, err := BuildWith(root, opts, ignore)
	if err != nil {
		t.Fatalf("BuildWith: %v", err)
	}
	return root, stats
}

func TestBuildWritesIndex(t *testing.T) {
	root, _ := builtWith(t, BuildOptions{})
	ix, err := trigram.Read(root)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	defer ix.Close()
	want := []string{"go.mod", "lib/lib.go", "lib/lib_test.go", "main.go"}
	if got := ix.Files(); !slices.Equal(got, want) {
		t.Fatalf("index files = %v, want %v", got, want)
	}
}

func TestBuildIndexDigestMatchesRecord(t *testing.T) {
	root, stats := builtWith(t, BuildOptions{})
	ix, err := trigram.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("the build wrote no search corpus record")
	}
	if ix.Digest() != corpus.Digest(rec) {
		t.Fatal("index digest and record digest disagree")
	}
	d := ix.Digest()
	if stats.Corpus.Digest != d {
		t.Fatalf("stats digest = %x, index digest = %x", stats.Corpus.Digest, d)
	}
	if want := hex.EncodeToString(d[:]); rec.Digest != want {
		t.Fatalf("record digest = %q, want %q", rec.Digest, want)
	}
	if ix.Extractor() == "" {
		t.Fatal("index carries no extractor")
	}
	// The graph's own record is a separate sidecar and keeps its shape.
	if _, err := os.Stat(freshness.Path(root)); err != nil {
		t.Fatalf("freshness record: %v", err)
	}
}

func TestBuildSkipsUnreadableSearchFile(t *testing.T) {
	files := sample()
	files["notes.md"] = "searchable text\n"
	root := repo(t, files)
	broken := filepath.Join(root, "notes.md")
	testlock.Lock(t, broken)
	_, stats, err := BuildWith(root, BuildOptions{}, ignore)
	if err != nil {
		t.Fatalf("an unreadable search file must not fail the build: %v", err)
	}
	if stats.Corpus.Unreadable != 1 {
		t.Fatalf("Corpus.Unreadable = %d, want 1", stats.Corpus.Unreadable)
	}
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no record")
	}
	if print, known := rec.Files["notes.md"]; !known || print.Hash != "" {
		t.Fatalf("record entry = %+v, %v; want the file with an empty hash", print, known)
	}
	ix, err := trigram.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	if slices.Contains(ix.Files(), "notes.md") {
		t.Fatal("an unreadable file reached the index")
	}
}

func TestBuildGraphFileUnreadableAborts(t *testing.T) {
	root := repo(t, sample())
	testlock.Lock(t, filepath.Join(root, "main.go"))
	if _, _, err := BuildWith(root, BuildOptions{}, ignore); err == nil {
		t.Fatal("an unreadable graph file must fail the build")
	}
}

func TestBuildRecordsSettings(t *testing.T) {
	limit := int64(1234)
	root, stats := builtWith(t, BuildOptions{MaxFilesize: &limit})
	if stats.Corpus.Files == 0 {
		t.Fatal("the corpus is empty")
	}
	ix, err := trigram.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no record")
	}
	want := corpus.CanonicalSettings(limit, false)
	if string(ix.Settings()) != string(want) || string(rec.Settings) != string(want) {
		t.Fatalf("settings index %q record %q, want %q", ix.Settings(), rec.Settings, want)
	}
}

func TestBuildIndexFailureIsNote(t *testing.T) {
	old := lock.SidecarBudget
	lock.SidecarBudget = 50 * time.Millisecond
	t.Cleanup(func() { lock.SidecarBudget = old })
	root := repo(t, sample())
	if err := os.MkdirAll(store.CachePath(root, "trigram.idx"), 0o755); err != nil {
		t.Fatal(err)
	}
	var heard []string
	g, _, err := BuildWith(root, BuildOptions{}, func(s string) { heard = append(heard, s) })
	if err != nil || g == nil {
		t.Fatalf("BuildWith = %v, %v; want the graph and no error", g, err)
	}
	found := false
	for _, s := range heard {
		if strings.Contains(s, "trigram index not written") {
			found = true
		}
	}
	if !found {
		t.Fatalf("notices = %q, want the unwritten index", heard)
	}
}

func TestBuildIndexSameBytesAsSource(t *testing.T) {
	files := sample()
	files["notes.md"] = "searchable text\n"
	root := repo(t, files)

	old := readFile
	reads := map[string]int{}
	readFile = func(name string) ([]byte, error) {
		reads[name]++
		return old(name)
	}
	defer func() { readFile = old }()

	if _, _, err := BuildWith(root, BuildOptions{}, ignore); err != nil {
		t.Fatal(err)
	}
	for rel, n := range reads {
		got, err := filepath.Rel(root, rel)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s was read %d times, want once", filepath.ToSlash(got), n)
		}
	}
	for _, rel := range []string{"go.mod", "main.go", "lib/lib.go", "notes.md"} {
		if reads[filepath.Join(root, filepath.FromSlash(rel))] != 1 {
			t.Errorf("%s was not read exactly once", rel)
		}
	}
}

func TestBuildCountsBinaryFiles(t *testing.T) {
	files := sample()
	files["blob.bin"] = "text\x00binary\n"
	root := repo(t, files)
	_, stats, err := BuildWith(root, BuildOptions{}, ignore)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Corpus.SkippedBinary != 1 {
		t.Fatalf("SkippedBinary = %d, want 1", stats.Corpus.SkippedBinary)
	}
	ix, err := trigram.Read(root)
	if err != nil {
		t.Fatal(err)
	}
	defer ix.Close()
	if slices.Contains(ix.Files(), "blob.bin") {
		t.Fatal("a binary file reached the index")
	}
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no record")
	}
	if print, known := rec.Files["blob.bin"]; !known || !print.Binary {
		t.Fatalf("record entry = %+v, %v; want it marked binary", print, known)
	}
}

func TestBuildCorpusErrorsAreNotes(t *testing.T) {
	root := repo(t, sample())
	// A directory where the record belongs: the write fails and says so, the
	// index is still written and the build stands.
	if err := os.MkdirAll(corpus.RecordPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	var heard []string
	if _, _, err := BuildWith(root, BuildOptions{}, func(s string) { heard = append(heard, s) }); err != nil {
		t.Fatalf("BuildWith: %v", err)
	}
	found := false
	for _, s := range heard {
		if strings.Contains(s, "search corpus record not written") {
			found = true
		}
	}
	if !found {
		t.Fatalf("notices = %q, want the unwritten corpus record", heard)
	}
	ix, err := trigram.Read(root)
	if err != nil {
		t.Fatalf("the index must still be written: %v", err)
	}
	ix.Close()
}

func TestBuildNegativeLimitIsAnError(t *testing.T) {
	limit := int64(-1)
	root := repo(t, sample())
	if _, _, err := BuildWith(root, BuildOptions{MaxFilesize: &limit}, ignore); err == nil {
		t.Fatal("a negative limit must be refused")
	} else if !strings.Contains(err.Error(), "negative") {
		t.Fatalf("err = %v", err)
	}
}

func TestWriteCorpusLoadErrorIsNote(t *testing.T) {
	old := loadIgnore
	loadIgnore = func(string) (*corpus.Rules, []corpus.Source, error) {
		return nil, nil, errors.New("boom")
	}
	t.Cleanup(func() { loadIgnore = old })
	var heard []string
	if st := writeCorpus(t.TempDir(), corpus.DefaultMaxFilesize, nil, func(s string) { heard = append(heard, s) }); st.Indexed {
		t.Fatal("a corpus that could not be selected must not be indexed")
	}
	if len(heard) != 1 || !strings.Contains(heard[0], "search corpus skipped") {
		t.Fatalf("notices = %q", heard)
	}
}

func TestWriteCorpusWalkErrorIsNote(t *testing.T) {
	old := walkCorpus
	walkCorpus = func(string, corpus.Matcher) (corpus.Listing, error) {
		return corpus.Listing{}, errors.New("boom")
	}
	t.Cleanup(func() { walkCorpus = old })
	var heard []string
	if st := writeCorpus(t.TempDir(), corpus.DefaultMaxFilesize, nil, func(s string) { heard = append(heard, s) }); st.Indexed {
		t.Fatal("a corpus that could not be walked must not be indexed")
	}
	if len(heard) != 1 || !strings.Contains(heard[0], "search corpus skipped") {
		t.Fatalf("notices = %q", heard)
	}
}

// The spans are bound to the corpus digest, so they are written after the
// corpus, and a build that writes no corpus leaves none behind.
func TestBuildWritesSpansLast(t *testing.T) {
	root := repo(t, sample())
	_, stats, err := BuildWith(root, BuildOptions{}, ignore)
	if err != nil {
		t.Fatal(err)
	}
	r, err := spanidx.Open(root, stats.Corpus.Digest)
	if err != nil {
		t.Fatalf("Open after a build: %v", err)
	}
	r.Close()

	old := walkCorpus
	walkCorpus = func(string, corpus.Matcher) (corpus.Listing, error) {
		return corpus.Listing{}, errors.New("walk failed")
	}
	t.Cleanup(func() { walkCorpus = old })
	if _, _, err := BuildWith(root, BuildOptions{}, ignore); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.CachePath(root, spanidx.Name)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("spans after a build without corpus: %v", err)
	}
}

func TestBuildSpansFailureIsNote(t *testing.T) {
	old := lock.SidecarBudget
	lock.SidecarBudget = 50 * time.Millisecond
	t.Cleanup(func() { lock.SidecarBudget = old })
	root := repo(t, sample())
	if err := os.MkdirAll(filepath.Join(store.CachePath(root, spanidx.Name)+"."+strconv.Itoa(os.Getpid())+".tmp", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	var heard []string
	g, _, err := BuildWith(root, BuildOptions{}, func(s string) { heard = append(heard, s) })
	if err != nil || g == nil {
		t.Fatalf("BuildWith = %v, %v; want the graph and no error", g, err)
	}
	if len(heard) != 1 || !strings.Contains(heard[0], "symbol spans not written") {
		t.Fatalf("notices = %q, want the unwritten spans", heard)
	}
}

// Spans that cannot be removed would outlive a graph whose corpus digest did
// not change, so the build fails rather than leave them to grep.
func TestBuildFailsWhenOldSpansStay(t *testing.T) {
	old := lock.SidecarBudget
	lock.SidecarBudget = 50 * time.Millisecond
	t.Cleanup(func() { lock.SidecarBudget = old })
	root := repo(t, sample())
	if err := os.MkdirAll(filepath.Join(store.CachePath(root, spanidx.Name), "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	g, _, err := BuildWith(root, BuildOptions{}, ignore)
	if err == nil || g != nil || !strings.Contains(err.Error(), "remove stale symbol spans") {
		t.Fatalf("BuildWith = %v, %v; want the stale spans named in the error", g, err)
	}
}

// A grep opening the old spans holds a handle until the mapping stands; on
// Windows that handle refuses the removal (Go opens without
// FILE_SHARE_DELETE), and the build waits instead of failing. The reader lets
// go at the first refused attempt; where removal of an open file is allowed,
// no attempt is refused and the handle closes afterwards.
func TestBuildWaitsForOpenSpans(t *testing.T) {
	root, _ := builtWith(t, BuildOptions{})
	f, err := os.Open(store.CachePath(root, spanidx.Name))
	if err != nil {
		t.Fatal(err)
	}
	var release sync.Once
	closeReader := func() { release.Do(func() { f.Close() }) }
	t.Cleanup(closeReader)
	oldHook := lock.OnRetry
	lock.OnRetry = closeReader
	t.Cleanup(func() { lock.OnRetry = oldHook })

	if _, _, err := BuildWith(root, BuildOptions{}, ignore); err != nil {
		t.Fatalf("BuildWith while the spans are open: %v", err)
	}
}
