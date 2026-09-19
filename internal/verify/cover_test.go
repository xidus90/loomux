package verify

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestNewRunID(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 15, 0, 0, time.UTC)
	if got := NewRunID(now, 1234); got != "20260919T101500-1234" {
		t.Fatal(got)
	}
	// A clock in another zone names the same instant: the ID is UTC, so no
	// run has to load the local zone to format it.
	east := time.Date(2026, 9, 19, 12, 15, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	if got := NewRunID(east, 1234); got != "20260919T101500-1234" {
		t.Fatal(got)
	}
}

func TestCoverPaths(t *testing.T) {
	dir := filepath.Join("r", ".loomux", "state", "cover")
	cases := []struct{ area, base string }{
		{".", "R-go-root"},
		{"web", "R-go-web"},
		{"apps/web", "R-go-apps_web"},
	}
	for _, c := range cases {
		profile, data := CoverPaths("r", "R", "go", c.area)
		if profile != filepath.Join(dir, c.base+".out") || data != filepath.Join(dir, c.base+".data") {
			t.Fatalf("%s: %s %s", c.area, profile, data)
		}
	}
}

func coverFiles(t *testing.T, root string, files ...string) string {
	t.Helper()
	dir := filepath.Join(root, ".loomux", "state", "cover")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func left(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := []string{}
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func age(t *testing.T, path string, at time.Time) {
	t.Helper()
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func TestCleanCover(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		green bool
		want  []string
	}{
		{false, []string{"A-go-fresh.out", "B-go-root.data", "B-go-root.out"}},
		{true, []string{"A-go-fresh.out"}},
	} {
		root := t.TempDir()
		dir := coverFiles(t, root, "A-go-fresh.out", "A-go-root.out", "B-go-root.out", "B-go-root.data", "BB-go-root.out")
		for _, f := range []string{"A-go-root.out", "BB-go-root.out", "B-go-root.out", "B-go-root.data"} {
			age(t, filepath.Join(dir, f), now.Add(-25*time.Hour))
		}
		age(t, filepath.Join(dir, "A-go-fresh.out"), now.Add(-23*time.Hour))
		if err := cleanCover(root, "B", c.green, now); err != nil {
			t.Fatal(err)
		}
		if got := left(t, dir); !slices.Equal(got, c.want) {
			t.Fatalf("green=%v: %v", c.green, got)
		}
	}
}

func TestCleanCoverSparesARunStillGoing(t *testing.T) {
	root := t.TempDir()
	dir := coverFiles(t, root, "A-go-root.out")
	if err := CleanCover(root, "B", true); err != nil {
		t.Fatal(err)
	}
	if got := left(t, dir); !slices.Equal(got, []string{"A-go-root.out"}) {
		t.Fatalf("%v", got)
	}
}

func TestCleanCoverWithoutADirectory(t *testing.T) {
	if err := CleanCover(t.TempDir(), "B", true); err != nil {
		t.Fatal(err)
	}
}

func TestCleanCoverReportsWhatItCannotRemove(t *testing.T) {
	root := t.TempDir()
	dir := coverFiles(t, root)
	stuck := filepath.Join(dir, "A-go-root.out")
	if err := os.MkdirAll(filepath.Join(stuck, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cleanCover(root, "B", true, time.Now().Add(48*time.Hour)); err == nil {
		t.Fatal("no error for a directory that cannot be removed")
	}
}

func TestCleanCoverReportsAnUnreadableDirectory(t *testing.T) {
	// Windows reports a file in place of the directory as missing, so only a
	// path no system accepts fails the same way everywhere.
	if err := CleanCover("bad\x00root", "B", true); err == nil {
		t.Fatal("no error for a path that cannot be read")
	}
}

// Every run makes the cover directory the same way, whoever started it, and
// a second run finds it there.
func TestPrepareCoverMakesTheDirectory(t *testing.T) {
	root := t.TempDir()
	for range 2 {
		if err := PrepareCover(root); err != nil {
			t.Fatal(err)
		}
	}
	if info, err := os.Stat(filepath.Join(root, ".loomux", "state", "cover")); err != nil || !info.IsDir() {
		t.Fatalf("%v %v", info, err)
	}
	os.WriteFile(filepath.Join(root, "file"), nil, 0o644)
	if err := PrepareCover(filepath.Join(root, "file")); err == nil {
		t.Fatal("a root that is a file holds no directory")
	}
}
