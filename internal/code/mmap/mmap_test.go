package mmap

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// writeFile puts data into a fresh file and returns its path.
func writeFile(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestOpenMapsTheFile pins that the mapping shows the file's bytes and that
// Close lets go of it: on Windows a mapped file cannot be removed.
func TestOpenMapsTheFile(t *testing.T) {
	path := writeFile(t, "hello")
	f, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := string(f.Bytes()); got != "hello" {
		t.Fatalf("Bytes() = %q, want hello", got)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("the file is still held after Close: %v", err)
	}
}

// TestOpenEmptyFile pins that an empty file needs no mapping: the system
// refuses to map zero bytes.
func TestOpenEmptyFile(t *testing.T) {
	f, err := Open(writeFile(t, ""))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := f.Bytes(); got == nil || len(got) != 0 {
		t.Fatalf("Bytes() = %v, want an empty slice", got)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// TestOpenMissingFile pins that a caller can tell "missing" apart.
func TestOpenMissingFile(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "absent")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open(absent) = %v, want fs.ErrNotExist", err)
	}
}

// TestOpenDirectoryIsAnError pins that only a regular file is mapped: a
// directory opens on Windows and would pass as an empty file.
func TestOpenDirectoryIsAnError(t *testing.T) {
	if _, err := Open(t.TempDir()); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open(directory) = %v, want an error other than ErrNotExist", err)
	}
}

// TestCloseTwiceIsSafe pins that a second Close neither fails nor unmaps
// twice, and that the bytes are gone after the first.
func TestCloseTwiceIsSafe(t *testing.T) {
	f, err := Open(writeFile(t, "abc"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if f.Bytes() != nil {
		t.Fatalf("Bytes() after Close = %v, want nil", f.Bytes())
	}
}
