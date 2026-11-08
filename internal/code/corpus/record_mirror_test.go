package corpus

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

// shortBudget keeps a swap that cannot succeed from waiting out the default.
func shortBudget(t *testing.T) {
	t.Helper()
	old := lock.SidecarBudget
	lock.SidecarBudget = 50 * time.Millisecond
	t.Cleanup(func() { lock.SidecarBudget = old })
}

func TestReadPrefersTheMirror(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	rec := buildCorpus(t, root, DefaultMaxFilesize)
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}

	old := readFile
	readFile = func(name string) ([]byte, error) {
		if filepath.Clean(name) == filepath.Clean(RecordPath(root)) {
			return nil, errors.New("the JSON file must not be read")
		}
		return old(name)
	}
	defer func() { readFile = old }()

	back, ok := Read(root)
	if !ok || back.Digest != rec.Digest || len(back.Files) != len(rec.Files) {
		t.Fatalf("Read = %+v, %v; want the record from the mirror", back, ok)
	}
}

func TestReadIgnoresAStaleMirror(t *testing.T) {
	root := newRepo(t, map[string]string{"keep.go": "package x\n"}, nil, nil)
	rec := buildCorpus(t, root, DefaultMaxFilesize)
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	// The JSON file is the canonical one: another writer's record replaces it
	// and the mirror of the old bytes no longer fits.
	other := rec
	other.Digest = "other"
	body, err := json.Marshal(other)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(RecordPath(root), body, 0o644); err != nil {
		t.Fatal(err)
	}

	back, ok := Read(root)
	if !ok || back.Digest != "other" {
		t.Fatalf("Read = %q, %v; want the JSON file's record", back.Digest, ok)
	}
}

// An empty file list does not survive gob (a zero value is not sent), so the
// mirror of such a record fails the reader's checks and the JSON file answers.
func TestReadFallsBackWhenTheMirrorLacksWhatARecordNeeds(t *testing.T) {
	root := t.TempDir()
	rec := NewRecord(nil, map[string]Print{}, 1)
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	back, ok := Read(root)
	if !ok || back.Files == nil || back.Digest != rec.Digest {
		t.Fatalf("Read = %+v, %v; want the JSON file's record", back, ok)
	}
}

func TestWriteReportsWhatItCannotStore(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".loomux"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := NewRecord(nil, map[string]Print{}, 1)
	if err := Write(root, rec); err == nil {
		t.Fatal("Write created a state directory through a file")
	}

	root = t.TempDir()
	if err := os.MkdirAll(RecordPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	shortBudget(t)
	if err := Write(root, rec); err == nil {
		t.Fatal("Write replaced a directory")
	}
}

func TestWriteKeepsTheRecordWhenTheMirrorFails(t *testing.T) {
	root := t.TempDir()
	// A directory in the mirror's place: its swap fails, the record stands.
	if err := os.MkdirAll(mirrorPath(RecordPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	shortBudget(t)

	rec := NewRecord(nil, map[string]Print{}, 1)
	// The failure is reported, as one callers can tell from a lost record.
	if err := Write(root, rec); !errors.Is(err, ErrMirror) {
		t.Fatalf("Write: %v, want ErrMirror", err)
	}
	if back, ok := Read(root); !ok || back.Digest != rec.Digest {
		t.Fatalf("Read = %+v, %v; want the JSON file's record", back, ok)
	}
}
