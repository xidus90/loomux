package corpus

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writtenRecord is a record with files and git state, stored the way the
// build stores it, and the stamp of its JSON file.
func writtenRecord(t *testing.T) (string, Record, Stamp) {
	t.Helper()
	root := newRepo(t, map[string]string{"keep.go": "package x\n", "a/b.txt": "b\n"}, []string{"keep.go"}, nil)
	rec := buildCorpus(t, root, DefaultMaxFilesize)
	if err := Write(root, rec); err != nil {
		t.Fatal(err)
	}
	return root, rec, FileStamp(RecordPath(root))
}

func indented(t *testing.T, rec Record) []byte {
	t.Helper()
	body, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(body, '\n')
}

func TestRestampWritesWhatWriteWould(t *testing.T) {
	root, rec, was := writtenRecord(t)
	stamp := Stamp{Size: 12345, MTime: 678901234}
	if err := Restamp(root, was, rec, stamp); err != nil {
		t.Fatal(err)
	}
	want := rec
	want.Stamp = stamp
	got, err := os.ReadFile(RecordPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, indented(t, want)) {
		t.Fatalf("record =\n%s\nwant\n%s", got, indented(t, want))
	}
	// The mirror is bound to the new file: the reader never parses the JSON.
	old := readFile
	readFile = func(name string) ([]byte, error) {
		if filepath.Clean(name) == filepath.Clean(RecordPath(root)) {
			return nil, errors.New("the JSON file must not be read")
		}
		return old(name)
	}
	defer func() { readFile = old }()
	if back, ok := Read(root); !ok || back.Stamp != stamp || back.Digest != rec.Digest {
		t.Fatalf("Read = stamp %+v digest %q, %v; want %+v from the mirror", back.Stamp, back.Digest, ok, stamp)
	}
}

// Another writer's record keeps its stamp: the old file list with the new
// stamp would not describe its index.
func TestRestampLeavesARecordThatMovedOn(t *testing.T) {
	root, rec, was := writtenRecord(t)
	other := rec
	other.Digest = "rebuilt"
	if err := Write(root, other); err != nil {
		t.Fatal(err)
	}
	if err := Restamp(root, was, rec, Stamp{Size: 1, MTime: 2}); err != nil {
		t.Fatal(err)
	}
	if back, ok := Read(root); !ok || back.Digest != "rebuilt" || back.Stamp != rec.Stamp {
		t.Fatalf("Read = digest %q stamp %+v, %v; want the other record untouched", back.Digest, back.Stamp, ok)
	}
}

// A record in a layout Write does not produce is written whole.
func TestRestampOfAnotherLayout(t *testing.T) {
	root, rec, _ := writtenRecord(t)
	body, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(RecordPath(root), body, 0o644); err != nil {
		t.Fatal(err)
	}
	stamp := Stamp{Size: 3, MTime: 4}
	if err := Restamp(root, FileStamp(RecordPath(root)), rec, stamp); err != nil {
		t.Fatal(err)
	}
	want := rec
	want.Stamp = stamp
	if got, _ := os.ReadFile(RecordPath(root)); !bytes.Equal(got, indented(t, want)) {
		t.Fatalf("record =\n%s\nwant the whole record rewritten", got)
	}
}

func TestRestampReportsWhatItCannotDo(t *testing.T) {
	if err := Restamp(t.TempDir(), Stamp{}, Record{}, Stamp{}); err == nil {
		t.Fatal("Restamp without a record reported nothing")
	}
	root, rec, was := writtenRecord(t)
	// A directory in the temporary's place: the new file cannot be written.
	if err := os.MkdirAll(tempName(RecordPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Restamp(root, was, rec, Stamp{Size: 1}); err == nil {
		t.Fatal("Restamp reported a write that failed as done")
	}
}

func TestFileStampOfAMissingFile(t *testing.T) {
	if got := FileStamp(filepath.Join(t.TempDir(), "gone")); got != (Stamp{}) {
		t.Fatalf("FileStamp = %+v, want the empty stamp", got)
	}
}
