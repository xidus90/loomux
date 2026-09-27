package runs_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xidus90/loomux/internal/flow/runs"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNextIDOfAProjectWithoutRunsIsTheFirst(t *testing.T) {
	if got := runs.NextID(t.TempDir()); got != "0001" {
		t.Fatalf("got %s", got)
	}
}

// A counter over journals and markers, and never a clock. A marker counts
// because a run claims its number with its marker before its journal exists.
// Names that are not numbers, and a number no int holds, do not count.
func TestNextIDCountsPastTheHighestNumberARunFileCarries(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"0001.jsonl", "0009.jsonl", "0012.flow", "notes.jsonl", "12a.jsonl", ".jsonl", "99999999999999999999.jsonl"} {
		touch(t, filepath.Join(root, runs.Dir, name))
	}
	if got := runs.NextID(root); got != "0013" {
		t.Fatalf("got %s, want 0013", got)
	}
}

// Measured on 2026-09-11: on Windows, reading a file as a directory reports
// "does not exist", so a runs path that is a file is a project without runs.
func TestNextIDOfARunsPathThatIsAFile(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, runs.Dir))
	if got := runs.NextID(root); got != "0001" {
		t.Fatalf("got %s", got)
	}
}

func TestTheFilesOfARun(t *testing.T) {
	root := t.TempDir()
	if got, want := runs.JournalPath(root, "0007"), filepath.Join(root, ".loomux", "state", "runs", "0007.jsonl"); got != want {
		t.Fatalf("journal: got %s, want %s", got, want)
	}
	if got, want := runs.MarkerPath(root, "0007"), filepath.Join(root, ".loomux", "state", "runs", "0007.flow"); got != want {
		t.Fatalf("marker: got %s, want %s", got, want)
	}
	want := []string{".loomux/state/runs/0007.flow", ".loomux/state/runs/0007.jsonl"}
	if got := runs.Files("0007"); !slices.Equal(got, want) {
		t.Fatalf("files: got %v, want %v", got, want)
	}
}
