//go:build windows

package testlock

import (
	"syscall"
	"testing"
)

// Lock holds path open without any share mode until the test ends, the way an
// editor or a sync tool can: a stat still succeeds, a read fails with a sharing
// violation. os.Chmod cannot deny a read on Windows.
//
//coverage:exempt test helper; its t.Fatalf arms run only when encoding the path or CreateFile fails
func Lock(t testing.TB, path string) {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("lock %s: %v", path, err)
	}
	// Registered after t.TempDir's own cleanup, so it runs before the
	// directory is removed: a held handle would keep the removal from working.
	t.Cleanup(func() { _ = syscall.CloseHandle(handle) })
}
