package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newSource is a real source the test closes, whatever happens.
func newSource(t *testing.T) Source {
	t.Helper()
	src, err := NewSource()
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	t.Cleanup(func() { src.Close() })
	return src
}

// waitFor reads events until one matches, and fails after five seconds.
func waitFor(t *testing.T, src Source, match func(Event) bool) Event {
	t.Helper()
	timeout := time.After(5 * time.Second)
	for {
		select {
		case e := <-src.Events():
			if match(e) {
				return e
			}
		case <-timeout:
			t.Fatal("no matching event within five seconds")
			return Event{}
		}
	}
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSourceReportsAWrite(t *testing.T) {
	root := t.TempDir()
	src := newSource(t)
	if err := src.Add(root, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	file := filepath.Join(root, "a.go")
	write(t, file)
	waitFor(t, src, func(e Event) bool { return e.Root == root && (e.Overflow || e.Path == file) })
}

// A directory made after Add is watched too: a source that sees only the
// directories that existed at the start misses every new package.
func TestSourceReportsAWriteInANewDirectory(t *testing.T) {
	root := t.TempDir()
	src := newSource(t)
	if err := src.Add(root, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	dir := filepath.Join(root, "pkg")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	waitFor(t, src, func(e Event) bool { return e.Root == root && (e.Overflow || e.Path == dir) })
	file := filepath.Join(dir, "b.go")
	write(t, file)
	waitFor(t, src, func(e Event) bool { return e.Root == root && (e.Overflow || e.Path == file) })
}

// A directory deleted and made again, as a branch switch does, is watched
// again: a per-directory watch dies with its directory.
func TestSourceReportsAWriteInARecreatedDirectory(t *testing.T) {
	root := t.TempDir()
	src := newSource(t)
	if err := src.Add(root, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	dir := filepath.Join(root, "pkg")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	waitFor(t, src, func(e Event) bool { return e.Path == dir })
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Event carries no operation, so the removal's stale events would match a
	// wait on dir; only a write below it proves the new directory is watched.
	// The writes repeat because the watch is added only once the creation has
	// been routed.
	inside := func(e Event) bool { return filepath.Dir(e.Path) == dir }
	timeout := time.After(5 * time.Second)
	for i := 0; ; i++ {
		write(t, filepath.Join(dir, fmt.Sprintf("b%d.go", i)))
		select {
		case e := <-src.Events():
			if inside(e) {
				return
			}
		case <-time.After(50 * time.Millisecond):
		case <-timeout:
			t.Fatal("no write below the recreated directory reported within five seconds")
		}
	}
}

// The git files of a linked worktree lie outside its root; a write there is
// reported for the root that asked for it.
func TestSourceReportsAWriteBesideTheRoot(t *testing.T) {
	root, gitDir := t.TempDir(), t.TempDir()
	src := newSource(t)
	index := filepath.Join(gitDir, "index")
	config := filepath.Join(gitDir, "config")
	missing := filepath.Join(gitDir, "info", "exclude")
	if err := src.Add(root, []string{index, config, missing}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	write(t, index)
	waitFor(t, src, func(e Event) bool { return e.Root == root && (e.Overflow || e.Path == index) })
}

func TestSourceRefusesAMissingRoot(t *testing.T) {
	src := newSource(t)
	if err := src.Add(filepath.Join(t.TempDir(), "gone"), nil); err == nil {
		t.Fatal("Add of a missing root succeeded")
	}
}

// A removed root can be added again: Remove forgets it rather than leaving a
// stale entry behind.
func TestSourceRemoveForgetsTheRoot(t *testing.T) {
	root := t.TempDir()
	src := newSource(t)
	if err := src.Add(root, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	src.Remove(root)
	if err := src.Add(root, nil); err != nil {
		t.Fatalf("Add after Remove: %v", err)
	}
	file := filepath.Join(root, "a.go")
	write(t, file)
	waitFor(t, src, func(e Event) bool { return e.Root == root && (e.Overflow || e.Path == file) })
}

// An open handle on Windows keeps a directory from going away; after Close
// none may be left.
func TestSourceCloseReleasesTheDirectory(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "area")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := NewSource()
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	if err := src.Add(root, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// An event left unread must not keep Close waiting.
	write(t, filepath.Join(root, "a.go"))
	if err := src.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("root not deletable after Close: %v", err)
	}
}
