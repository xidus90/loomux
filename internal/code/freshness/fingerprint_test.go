package freshness_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/testlock"
)

// build writes a tree, records its fingerprint, and returns the root -- the
// state right after a `graph build`.
func build(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	record(t, root)
	return root
}

// record writes the fingerprint of the tree as it is now.
func record(t *testing.T, root string) {
	t.Helper()
	stat, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, f := range stat {
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		hashes[f.Rel] = hex.EncodeToString(sum[:])
	}
	if err := freshness.Write(root, "go/1", stat, hashes); err != nil {
		t.Fatal(err)
	}
}

func TestProbeIsCleanRightAfterABuild(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || !d.Clean() {
		t.Fatalf("got %+v, want clean", d)
	}
}

func TestProbeReportsNoRecordAsUnknownAndNotAsClean(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	// Never built, or built by a version that wrote no record. A caller must
	// rebuild -- treating this as clean is how a query answers from a graph
	// that does not exist.
	if d != nil {
		t.Fatalf("got %+v, want nil for unknown", d)
	}
}

func TestProbeTreatsAForeignExtractorAsUnknown(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})

	d, err := freshness.Probe(root, "go/2")
	if err != nil {
		t.Fatal(err)
	}
	// A changed extractor matches the tree byte for byte. Without this check it
	// reports clean and queries keep answering from nodes the old extractor
	// built.
	if d != nil {
		t.Fatalf("got %+v, want nil: the record belongs to another extractor", d)
	}
}

func TestProbeSeesAChangedFile(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || len(d.Changed) != 1 || d.Changed[0] != "a.go" {
		t.Fatalf("got %+v, want a.go changed", d)
	}
	if d.Count() != 1 || d.Clean() {
		t.Errorf("Count = %d, Clean = %v", d.Count(), d.Clean())
	}
}

func TestProbeIgnoresATouchThatChangedNoBytes(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	abs := filepath.Join(root, "a.go")
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(abs, later, later); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	// The stat fast path suspects the file, the hash clears it. Without this a
	// `touch` or a checkout restoring identical bytes would cost a rebuild.
	if d == nil || !d.Clean() {
		t.Fatalf("got %+v, want clean after a touch", d)
	}
}

func TestProbeSeesAnAddedAndARemovedFile(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n", "b.go": "package a\n"})
	if err := os.Remove(filepath.Join(root, "b.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "c.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || len(d.Added) != 1 || d.Added[0] != "c.go" {
		t.Fatalf("got %+v, want c.go added", d)
	}
	if len(d.Removed) != 1 || d.Removed[0] != "b.go" {
		t.Fatalf("got %+v, want b.go removed", d)
	}
	if d.Count() != 2 {
		t.Errorf("Count = %d, want 2", d.Count())
	}
}

func TestProbeSortsEachCategory(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	for _, rel := range []string{"z.go", "m.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("package a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"b.go", "m.go", "z.go"}
	if len(d.Added) != 3 {
		t.Fatalf("got %+v", d.Added)
	}
	for i := range want {
		if d.Added[i] != want[i] {
			t.Fatalf("got %v, want %v: a report a human reads is sorted", d.Added, want)
		}
	}
}

func TestProbeReportsAMissingRoot(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	// The record survives, the tree does not.
	if err := os.Remove(filepath.Join(root, "a.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := freshness.Probe(filepath.Join(root, "gone"), "go/1"); err == nil {
		t.Fatal("got nil, want an error for a root that is not there")
	}
}

func TestProbeLeavesAnUnreadableFileToTheNextProbe(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	abs := filepath.Join(root, "a.go")
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(abs, later, later); err != nil {
		t.Fatal(err)
	}
	testlock.Lock(t, abs)

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	// The file is suspect by mtime but cannot be read right now. It is left to
	// the next probe rather than reported as drift a rebuild could not repair
	// either.
	if d == nil || !d.Clean() {
		t.Fatalf("got %+v, want clean: an unreadable file is left for the next probe", d)
	}
}

func TestWriteReportsAMkdirFailure(t *testing.T) {
	root := t.TempDir()
	// A file where the state directory needs to be: MkdirAll cannot create a
	// directory through it.
	if err := os.WriteFile(filepath.Join(root, ".loomux"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := freshness.Write(root, "go/1", nil, nil); err == nil {
		t.Fatal("got nil, want an error: the state directory cannot be created")
	}
}

func TestWriteReportsAWriteFailure(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	testlock.Lock(t, freshness.Path(root))

	if err := freshness.Write(root, "go/1", nil, nil); err == nil {
		t.Fatal("got nil, want an error: the sidecar cannot be written")
	}
}

func TestProbeTreatsABrokenRecordAsUnknown(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	if err := os.WriteFile(freshness.Path(root), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	// A sidecar is a cache. A broken one costs the next probe its fast path and
	// nothing else -- it must never be an error a query dies on.
	if d != nil {
		t.Fatalf("got %+v, want nil for a record that cannot be read", d)
	}
}

func TestProbeTreatsARecordOfAnotherVersionAsUnknown(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	// The extractor field matches; only the schema version does not. A reader
	// that checked the extractor alone would accept this record and diff
	// against files whose recorded shape it never actually wrote.
	body := `{"version":99,"extractor":"go/1","files":{"a.go":{"size":1,"mtime":1,"hash":"x"}}}`
	if err := os.WriteFile(freshness.Path(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d != nil {
		t.Fatalf("got %+v, want nil: the record's version does not match what this code writes", d)
	}
}

func TestProbeAlwaysChecksAFileWhoseRecordedSizeDisagrees(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	abs := filepath.Join(root, "a.go")
	info, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	orig := info.ModTime()
	// Longer content, but the mtime is put back exactly where it was: only the
	// size disagrees with the record.
	if err := os.WriteFile(abs, []byte("package a\n\nfunc Longer() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(abs, orig, orig); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || len(d.Changed) != 1 || d.Changed[0] != "a.go" {
		t.Fatalf("got %+v, want a.go changed even though only its size moved", d)
	}
}

func TestProbeAlwaysChecksAFileWhoseRecordedModTimeDisagrees(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	abs := filepath.Join(root, "a.go")
	// Same length, different bytes -- the size still matches the record --
	// with the mtime moved: only the mtime disagrees with the record.
	if err := os.WriteFile(abs, []byte("package b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(abs, later, later); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || len(d.Changed) != 1 || d.Changed[0] != "a.go" {
		t.Fatalf("got %+v, want a.go changed even though only its mtime moved and its size stayed put", d)
	}
}

func TestProbeAlwaysChecksAFileTheLastBuildNeverHashed(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "a.go")
	if err := os.WriteFile(abs, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	// hashes carries no entry for a.go, so Write records it with an empty hash
	// -- exactly what a build does for a file it could not read.
	if err := freshness.Write(root, "go/1", stat, map[string]string{}); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || len(d.Changed) != 1 || d.Changed[0] != "a.go" {
		t.Fatalf("got %+v, want a.go changed: an empty recorded hash is never trusted, size and mtime notwithstanding", d)
	}
}
