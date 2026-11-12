package sessions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkOnceIsTrueOnlyTheFirstTime(t *testing.T) {
	root := t.TempDir()
	for i, want := range []bool{true, false, false} {
		first, err := MarkOnce(root, "s1", "", "graph-hint")
		if err != nil || first != want {
			t.Errorf("call %d: first %v, %v; want %v", i+1, first, err, want)
		}
	}
}

func TestMarkOnceKeepsAgentsApart(t *testing.T) {
	root := t.TempDir()
	for _, agent := range []string{"", "a", "b"} {
		if first, err := MarkOnce(root, "s1", agent, "graph-hint"); err != nil || !first {
			t.Errorf("agent %q: first %v, %v; want its own first mark", agent, first, err)
		}
	}
	if first, _ := MarkOnce(root, "s2", "", "graph-hint"); !first {
		t.Error("another session must get its own first mark")
	}
}

func TestMarkOnceWithoutASessionMarksNothing(t *testing.T) {
	root := t.TempDir()
	if first, err := MarkOnce(root, "", "", "graph-hint"); first || err != nil {
		t.Errorf("no session: first %v, %v; want false, nil", first, err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(StateDir))); !os.IsNotExist(err) {
		t.Errorf("no session must write nothing; stat %v", err)
	}
}

func TestForgetRemovesTheMark(t *testing.T) {
	root := t.TempDir()
	if _, err := MarkOnce(root, "s1", "a", "graph-hint"); err != nil {
		t.Fatal(err)
	}
	if err := Forget(root, "s1"); err != nil {
		t.Fatal(err)
	}
	if first, _ := MarkOnce(root, "s1", "a", "graph-hint"); !first {
		t.Error("a forgotten session starts with no mark")
	}
}

func TestMarkOnceReportsAMarkItCannotCreate(t *testing.T) {
	// No file system takes a NUL in a name.
	if first, err := MarkOnce(t.TempDir(), "s1", "", "bad\x00name"); first || err == nil {
		t.Errorf("uncreatable mark: first %v, %v; want false and an error", first, err)
	}
}

func TestMarkOnceReportsAnUnwritableState(t *testing.T) {
	root := t.TempDir()
	// The state directory is a file, so no session folder can be made in it.
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(StateDir)), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if first, err := MarkOnce(root, "s1", "", "graph-hint"); first || err == nil {
		t.Errorf("unwritable state: first %v, %v; want false and an error", first, err)
	}
}
