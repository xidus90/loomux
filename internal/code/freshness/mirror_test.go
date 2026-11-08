package freshness_test

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/lock"
)

// The mirror is bound to the JSON file by size and mtime. With both put back,
// a JSON file that cannot be parsed still stands behind a mirror that can be,
// which is how the test sees that the probe reads the mirror first.
func TestProbeReadsTheMirrorBoundToTheJSON(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	info, err := os.Stat(freshness.Path(root))
	if err != nil {
		t.Fatal(err)
	}
	garbage := bytes.Repeat([]byte("x"), int(info.Size()))
	if err := os.WriteFile(freshness.Path(root), garbage, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(freshness.Path(root), info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || !d.Clean() {
		t.Fatalf("got %+v, want clean from the mirror", d)
	}
}

// A mirror that belongs to another JSON file is not a record of this one: the
// JSON file names another extractor, so the probe has to come out unknown.
func TestProbeIgnoresAMirrorOfAnotherJSON(t *testing.T) {
	root := build(t, map[string]string{"a.go": "package a\n"})
	body := `{"version":1,"extractor":"go/2","files":{"a.go":{"size":1,"mtime":1,"hash":"x"}}}`
	if err := os.WriteFile(freshness.Path(root), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(freshness.Path(root), later, later); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d != nil {
		t.Fatalf("got %+v, want nil: the mirror describes another file", d)
	}
}

// A record without files does not survive gob, so its mirror cannot answer;
// the JSON file does.
func TestProbeFallsBackToTheJSONWhenTheMirrorHasNoFiles(t *testing.T) {
	root := t.TempDir()
	if err := freshness.Write(root, "go/1", nil, nil); err != nil {
		t.Fatal(err)
	}

	d, err := freshness.Probe(root, "go/1")
	if err != nil {
		t.Fatal(err)
	}
	if d == nil || !d.Clean() {
		t.Fatalf("got %+v, want clean from the JSON file", d)
	}
}

func TestWriteKeepsTheRecordWhenTheMirrorFails(t *testing.T) {
	root := t.TempDir()
	// A directory in the mirror's place: its swap fails, the record stands.
	if err := os.MkdirAll(freshness.Path(root)+".bin", 0o755); err != nil {
		t.Fatal(err)
	}
	old := lock.SidecarBudget
	lock.SidecarBudget = 50 * time.Millisecond
	t.Cleanup(func() { lock.SidecarBudget = old })

	// The failure is reported, as one callers can tell from a lost record.
	if err := freshness.Write(root, "go/1", nil, nil); !errors.Is(err, corpus.ErrMirror) {
		t.Fatalf("Write: %v, want corpus.ErrMirror", err)
	}
	d, err := freshness.Probe(root, "go/1")
	if err != nil || d == nil || !d.Clean() {
		t.Fatalf("Probe = %+v, %v; want clean from the JSON file", d, err)
	}
}
