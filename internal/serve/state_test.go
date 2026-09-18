package serve_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/serve"
)

func newState() *serve.State {
	return &serve.State{
		Local:      serve.Endpoint{URL: "http://127.0.0.1:1/mcp", Token: "l"},
		Cloud:      serve.Endpoint{URL: "http://127.0.0.1:2/mcp", Token: "c"},
		PID:        4711,
		Executable: `C:\loomux\bin\loomux.exe`,
		Size:       17,
		ModTime:    time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC),
		// True rather than the zero value: a field that is false on both sides
		// survives a misspelled json tag without anyone noticing.
		BrokeAway: true,
	}
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	want := newState()
	if err := serve.WriteState(dir, want); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	got, err := serve.ReadState(dir)
	if err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if !got.ModTime.Equal(want.ModTime) {
		t.Fatalf("ModTime = %v, want %v", got.ModTime, want.ModTime)
	}
	// The time is the one field that cannot be compared with ==: the same
	// instant may carry another location after the detour through JSON. Once it
	// is known equal, the whole struct can be compared at once, and every field
	// added later is covered without anyone having to remember this test.
	got.ModTime = want.ModTime
	if *got != *want {
		t.Errorf("round trip lost data:\n got %+v\nwant %+v", *got, *want)
	}
}

// TestWriteKeepsTheStateToItsOwner pins what nothing else pins: the state file
// carries two tokens, and the explicit chmod was dropped because os.CreateTemp
// already makes the file 0600. Should that ever stop being true, this test is
// the only place it shows.
func TestWriteKeepsTheStateToItsOwner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no POSIX file modes; the ACL of the user profile guards the state directory there")
	}
	// A directory WriteState has to create itself: t.TempDir's own mode is not
	// the one under test.
	dir := filepath.Join(t.TempDir(), "state")
	if err := serve.WriteState(dir, newState()); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	for _, c := range []struct {
		path string
		want os.FileMode
	}{
		{dir, 0o700},
		{serve.StatePath(dir), 0o600},
	} {
		info, err := os.Stat(c.path)
		if err != nil {
			t.Fatalf("Stat %s: %v", c.path, err)
		}
		if got := info.Mode().Perm(); got != c.want {
			t.Errorf("%s has mode %o, want %o", c.path, got, c.want)
		}
	}
}

func TestWriteLeavesNoTemporaryFileBehind(t *testing.T) {
	dir := t.TempDir()
	if err := serve.WriteState(dir, newState()); err != nil {
		t.Fatalf("WriteState: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(serve.StatePath(dir)) {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("state directory holds %v, want only the state file", names)
	}
}

func TestReadMissingStateSaysSo(t *testing.T) {
	if _, err := serve.ReadState(t.TempDir()); err == nil {
		t.Fatal("expected an error for a missing state file")
	}
}

func TestReadBrokenStateSaysSo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(serve.StatePath(dir), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := serve.ReadState(dir); err == nil {
		t.Fatal("expected an error for a broken state file")
	}
}

func TestOlderThanComparesTheBuild(t *testing.T) {
	s := newState()
	newer := s.ModTime.Add(time.Minute)
	older := s.ModTime.Add(-time.Minute)

	if !s.OlderThan(newer) {
		t.Error("a newer build should win")
	}
	if s.OlderThan(older) {
		t.Error("an older build must never win")
	}
	if s.OlderThan(s.ModTime) {
		t.Error("the same build is not newer")
	}
}

func TestPathsAllSitUnderTheStateDir(t *testing.T) {
	dir := t.TempDir()
	for name, got := range map[string]string{
		"state":    serve.StatePath(dir),
		"lock":     serve.LockPath(dir),
		"qmd lock": serve.QmdLockPath(dir),
		"log":      serve.LogPath(dir),
	} {
		if rel, err := filepath.Rel(dir, got); err != nil || strings.HasPrefix(rel, "..") {
			t.Errorf("%s path %q is not under the state dir", name, got)
		}
	}
}

// The three tests below are not in the brief. They cover the two failure arms
// of WriteState that a test can reach without touching the operating system,
// and the happy path of BuildIdentity.

func TestWriteSaysSoWhenTheStateDirIsAFile(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "state")
	if err := os.WriteFile(blocked, nil, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := serve.WriteState(blocked, newState()); err == nil {
		t.Fatal("expected an error for a state directory that is a file")
	}
}

func TestWriteRemovesTheTemporaryFileWhenTheRenameFails(t *testing.T) {
	dir := t.TempDir()
	// A directory where serve.json belongs: the rename refuses a directory as
	// its target, which is the last arm before the state file is in place.
	if err := os.Mkdir(serve.StatePath(dir), 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if err := serve.WriteState(dir, newState()); err == nil {
		t.Fatal("expected an error when the state file cannot be renamed into place")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("state directory holds %v, want no temporary file", names)
	}
}

func TestBuildIdentityDescribesTheRunningProgram(t *testing.T) {
	path, size, modTime, err := serve.BuildIdentity()
	if err != nil {
		t.Fatalf("BuildIdentity: %v", err)
	}
	if path == "" {
		t.Error("BuildIdentity gave no path")
	}
	if size <= 0 {
		t.Errorf("size = %d, want the size of the test binary", size)
	}
	if modTime.IsZero() {
		t.Error("BuildIdentity gave no modification time")
	}
}
