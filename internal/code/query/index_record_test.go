package query

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/trigram"
)

// A failed index write keeps the old record: the record never names an index
// that was not written, so the old pair stays valid.
func TestBuildKeepsTheRecordWhenTheIndexFails(t *testing.T) {
	root, _ := builtWith(t, BuildOptions{})
	before, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no record after the first build")
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n\n// moved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := writeIndex
	writeIndex = func(string, [32]byte, string, []byte, trigram.Counters, []trigram.File) error {
		return errors.New("index mapped")
	}
	t.Cleanup(func() { writeIndex = old })
	_, stats, err := BuildWith(root, BuildOptions{}, ignore)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := corpus.Read(root)
	if after.Digest != before.Digest || stats.Corpus.Indexed {
		t.Fatalf("record digest %q indexed %v, want the old %q and not indexed", after.Digest, stats.Corpus.Indexed, before.Digest)
	}
}

// mismatchIndex writes an index whose digest is not the record's, the state a
// build leaves that wrote one of the two.
func mismatchIndex(t *testing.T, root string) {
	t.Helper()
	rec, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no record")
	}
	if err := trigram.Write(root, [32]byte{1}, all.Version(), rec.Settings, trigram.Counters{IgnoreCase: rec.IgnoreCase}, nil); err != nil {
		t.Fatal(err)
	}
}

// An index that does not match the record is rebuilt, on a clean tree and
// instead of the overlay, rather than answered stale until the next drift.
func TestGrepRebuildsAnIndexThatDoesNotMatchTheRecord(t *testing.T) {
	for name, edit := range map[string]bool{"clean tree": false, "small drift": true} {
		t.Run(name, func(t *testing.T) {
			root := grepFixture(t)
			mismatchIndex(t, root)
			if edit {
				writeTree(t, root, map[string]string{"README.md": "# Repo\n\nzzzUniqueWord moved\n"})
			}
			builds := countBuilds(t)
			res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if *builds != 1 || res.Index != "used" {
				t.Fatalf("builds = %d index = %q, want 1 and used (notes %v)", *builds, res.Index, notes)
			}
		})
	}
}

// The watcher's check heals the same state.
func TestRefreshIfDriftedRebuildsAnIndexThatDoesNotMatchTheRecord(t *testing.T) {
	root := grepFixture(t)
	mismatchIndex(t, root)
	if rebuilt, err := RefreshIfDrifted(root, nil); !rebuilt || err != nil {
		t.Fatalf("rebuilt = %v err = %v, want a rebuild", rebuilt, err)
	}
}
