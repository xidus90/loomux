//go:build !windows

package testlock

import (
	"os"
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
