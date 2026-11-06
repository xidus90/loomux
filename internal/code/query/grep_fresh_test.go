package query

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/corpus"
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
	if res.Index != "used" {
		t.Fatalf("index = %q, want used after the rebuild", res.Index)
	}
	if res.TotalHits == 0 {
		t.Fatal("the rebuilt index did not answer")
	}
}

func TestLockedRebuildAnswersFromDisk(t *testing.T) {
	root := grepFixture(t)
	if err := os.WriteFile(filepath.Join(root, "README.md"),
		[]byte("# Repo\n\nzzzFreshNeedle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
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
