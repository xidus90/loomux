package query

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
)

// overrideWalk replaces the corpus walk for one test and restores it after.
func overrideWalk(t *testing.T, walk func(string, corpus.Matcher) (corpus.Listing, error)) {
	t.Helper()
	old := walkCorpus
	walkCorpus = walk
	t.Cleanup(func() { walkCorpus = old })
}

// overrideRead replaces the corpus record read for one test and restores it.
func overrideRead(t *testing.T, read func(string) (corpus.Record, bool)) {
	t.Helper()
	old := readRecord
	readRecord = read
	t.Cleanup(func() { readRecord = old })
}

// walkFailsAfter first real call makes the walk fail: the drift is measured
// on the real tree, the rebuild's own walk then hits the broken filesystem.
func walkFailsAfter(real func(string, corpus.Matcher) (corpus.Listing, error), first int) func(string, corpus.Matcher) (corpus.Listing, error) {
	calls := 0
	return func(root string, m corpus.Matcher) (corpus.Listing, error) {
		calls++
		if calls > first {
			return corpus.Listing{}, errors.New("walk broken")
		}
		return real(root, m)
	}
}

func TestGrepRejectsANegativeSizeLimit(t *testing.T) {
	root := grepFixture(t)
	neg := int64(-1)
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{MaxFilesize: &neg}); err == nil {
		t.Fatal("a negative limit must fail")
	}
}

func TestGrepRejectsAnUnknownChannel(t *testing.T) {
	root := grepFixture(t)
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "bogus"}); err == nil {
		t.Fatal("an unknown channel must fail")
	}
}

func TestGrepReportsAWalkFailureWithARecord(t *testing.T) {
	root := grepFixture(t)
	overrideWalk(t, func(string, corpus.Matcher) (corpus.Listing, error) {
		return corpus.Listing{}, errors.New("walk broken")
	})
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{}); err == nil {
		t.Fatal("the walk error must surface")
	}
}

func TestGrepReportsAWalkFailureWithoutARecord(t *testing.T) {
	root := grepFixture(t)
	overrideRead(t, func(string) (corpus.Record, bool) { return corpus.Record{}, false })
	overrideWalk(t, func(string, corpus.Matcher) (corpus.Listing, error) {
		return corpus.Listing{}, errors.New("walk broken")
	})
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{}); err == nil {
		t.Fatal("the walk error must surface")
	}
}

func TestGrepReportsAWalkFailureDuringRebuild(t *testing.T) {
	root := grepFixture(t)
	real := corpus.Walk
	overrideWalk(t, walkFailsAfter(real, 1))
	write := []byte("changed\n")
	if err := os.WriteFile(filepath.Join(root, "README.md"), write, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{}); err == nil {
		t.Fatal("a failing rebuild must surface the walk error")
	}
}

func TestGrepRebuildWithoutRecordReportsAWalkFailure(t *testing.T) {
	root := grepFixture(t)
	calls := 0
	real := corpus.Read
	overrideRead(t, func(r string) (corpus.Record, bool) {
		calls++
		if calls == 1 {
			return corpus.Record{}, false
		}
		return real(r)
	})
	overrideWalk(t, walkFailsAfter(corpus.Walk, 1))
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{}); err == nil {
		t.Fatal("a failing rebuild must surface the walk error")
	}
}

func TestGrepRebuildLeavingNoRecordScans(t *testing.T) {
	root := grepFixture(t)
	overrideRead(t, func(string) (corpus.Record, bool) { return corpus.Record{}, false })
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatalf("a missing record after the rebuild must not fail: %v", err)
	}
	if res.TotalHits == 0 {
		t.Fatal("the scan must still answer")
	}
}

func TestCloudPrefixPartlyReadable(t *testing.T) {
	root := privacyRepo(t, `"secrets/**"`)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "cloud", In: "docs"})
	if err != nil {
		t.Fatalf("cloud: %v", err)
	}
	if res.TotalHits == 0 {
		t.Fatal("a readable file under the prefix must answer")
	}
}

func TestGrepReportsAnIgnoreLoadFailureWithoutARecord(t *testing.T) {
	root := grepFixture(t)
	overrideRead(t, func(string) (corpus.Record, bool) { return corpus.Record{}, false })
	old := loadIgnore
	loadIgnore = func(string) (*corpus.Rules, []corpus.Source, error) {
		return nil, nil, errors.New("ignore broken")
	}
	t.Cleanup(func() { loadIgnore = old })
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{}); err == nil {
		t.Fatal("the ignore load error must surface")
	}
}
