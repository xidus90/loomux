package godotsuite

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

var past = time.Date(2024, 3, 4, 5, 6, 8, 0, time.UTC) // even second: FAT keeps two

func gdProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := project(t, files)
	for rel := range files {
		if err := os.Chtimes(filepath.Join(root, rel), past, past); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func mtimeOf(t *testing.T, p string) time.Time {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.ModTime().Round(time.Second)
}

func TestSettleAcceptsARewriteWithTheSameContentAndRestoresItsMtime(t *testing.T) {
	root := gdProject(t, map[string]string{"core/a.gd": "extends Node\n", "readme.md": "x"})
	snap, err := Take(root)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "core", "a.gd")
	if err := os.WriteFile(p, []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("extends Node\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if mtimeOf(t, p).Equal(past) {
		t.Fatal("precondition: the rewrite must have moved the mtime")
	}
	if f := snap.Settle(root); len(f) != 0 {
		t.Fatalf("findings %q", f)
	}
	if got := mtimeOf(t, p); !got.Equal(past) {
		t.Fatalf("mtime %v, want %v", got, past)
	}
}

func TestSettleReportsAFileWithOtherContent(t *testing.T) {
	root := gdProject(t, map[string]string{"core/a.gd": "one\n"})
	snap, _ := Take(root)
	os.WriteFile(filepath.Join(root, "core", "a.gd"), []byte("two\n"), 0o644)
	want := []string{"core/a.gd differs after restoring the instrumentation"}
	if f := snap.Settle(root); !reflect.DeepEqual(f, want) {
		t.Fatalf("got %q", f)
	}
}

func TestSettleReportsADeletedFile(t *testing.T) {
	root := gdProject(t, map[string]string{"core/a.gd": "one\n"})
	snap, _ := Take(root)
	os.Remove(filepath.Join(root, "core", "a.gd"))
	want := []string{"core/a.gd differs after restoring the instrumentation"}
	if f := snap.Settle(root); !reflect.DeepEqual(f, want) {
		t.Fatalf("got %q", f)
	}
}

func TestSettleListsFindingsSortedByPath(t *testing.T) {
	root := gdProject(t, map[string]string{"b.gd": "1", "a.gd": "1", "c.gd": "1"})
	snap, _ := Take(root)
	for _, n := range []string{"b.gd", "a.gd"} {
		os.WriteFile(filepath.Join(root, n), []byte("2"), 0o644)
	}
	f := snap.Settle(root)
	if len(f) != 2 || f[0][:4] != "a.gd" || f[1][:4] != "b.gd" {
		t.Fatalf("got %q", f)
	}
}

func TestSettleDoesNotNormaliseLineEndings(t *testing.T) {
	// The engine restores byte for byte, so a CRLF/LF flip is a real change.
	root := gdProject(t, map[string]string{"a.gd": "x\r\ny\r\n"})
	snap, _ := Take(root)
	os.WriteFile(filepath.Join(root, "a.gd"), []byte("x\ny\n"), 0o644)
	if f := snap.Settle(root); len(f) != 1 {
		t.Fatalf("got %q", f)
	}
}

func TestSettleAcceptsAnUnchangedFileWithABOM(t *testing.T) {
	root := gdProject(t, map[string]string{"a.gd": "\xef\xbb\xbfextends Node\r\n"})
	snap, err := Take(root)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "a.gd"), []byte("\xef\xbb\xbfextends Node\r\n"), 0o644)
	if f := snap.Settle(root); len(f) != 0 {
		t.Fatalf("findings %q", f)
	}
}

func TestTakeOnlyVisitsWhatWalkGDVisits(t *testing.T) {
	root := gdProject(t, map[string]string{"a.gd": "1", "addons/x.gd": "1", "m.md": "1"})
	snap, _ := Take(root)
	if len(snap) != 1 {
		t.Fatalf("got %d entries", len(snap))
	}
	if _, ok := snap["a.gd"]; !ok {
		t.Fatal("a.gd missing")
	}
}

func TestTakeFailsOnAMissingRoot(t *testing.T) {
	if _, err := Take(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("want an error")
	}
}
