//go:build !windows

package testlock

import (
	"os"
	"path/filepath"
	"testing"
)

// Lock takes every permission off path until the test ends. A process that
// reads regardless -- root does -- cannot build the world the test needs, so
// the test is skipped rather than passed on a readable file.
//
//coverage:exempt built only off Windows; the gate runs on Windows
func Lock(t testing.TB, path string) {
	t.Helper()
	if err := os.Chmod(path, 0); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("this process reads a file without permissions; the world cannot be built")
	}
}

// LockDir takes every permission off a directory until the test ends, so
// reading it back fails. A process that reads regardless -- root does -- cannot
// build the world the test needs, so the test is skipped rather than passed on
// a readable directory.
//
//coverage:exempt built only off Windows; the gate runs on Windows
func LockDir(t testing.TB, path string) {
	t.Helper()
	if err := os.Chmod(path, 0); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o755) })
	if _, err := os.ReadDir(path); err == nil {
		t.Skip("this process reads a directory without permissions; the world cannot be built")
	}
}

// LockLink takes the write permission off the link's parent until the test
// ends, so the link cannot be deleted. A process that writes regardless --
// root does -- cannot build the world the test needs, so the test is skipped
// rather than passed on a deletable link.
//
//coverage:exempt built only off Windows; the gate runs on Windows
func LockLink(t testing.TB, path string) {
	t.Helper()
	parent := filepath.Dir(path)
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatalf("chmod %s: %v", parent, err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })
	probe := filepath.Join(parent, ".testlock-probe")
	if err := os.Mkdir(probe, 0o755); err == nil {
		_ = os.Remove(probe)
		t.Skip("this process writes into a directory without permissions; the world cannot be built")
	}
}

// DeletePending has nothing to build here: a name that stays taken after its
// delete is a Windows state, and POSIX unlinks at once.
//
//coverage:exempt built only off Windows; the gate runs on Windows
func DeletePending(t testing.TB, path string) {
	t.Helper()
	t.Skip("a pending delete that keeps its name exists only on Windows")
}
