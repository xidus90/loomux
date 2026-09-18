//go:build windows

package testlock

import (
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Lock holds path open without any share mode until the test ends, the way an
// editor or a sync tool can: a stat still succeeds, a read fails with a sharing
// violation. os.Chmod cannot deny a read on Windows.
//
// The handle asks for write access too although it never writes. Measured on
// 2026-09-18 on GitHub's windows-latest runner, which holds SeBackupPrivilege
// and SeRestorePrivilege enabled: a read-only handle without share mode did
// not stop os.ReadFile there, one with read and write access did.
//
//coverage:exempt test helper; its t.Fatalf arms run only when encoding the path or CreateFile fails
func Lock(t testing.TB, path string) {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("lock %s: %v", path, err)
	}
	// Registered after t.TempDir's own cleanup, so it runs before the
	// directory is removed: a held handle would keep the removal from working.
	t.Cleanup(func() { _ = syscall.CloseHandle(handle) })
}

// LockDir does the same for a directory: every attempt to read it back --
// os.ReadDir, and filepath.WalkDir walking into it -- fails with a sharing
// violation. Measured on 2026-09-15: WalkDir hands that error to its callback
// for the directory itself, which is how a walk's error arm is reached without
// a permission denial this process cannot produce.
//
// The one difference from Lock is FILE_FLAG_BACKUP_SEMANTICS, without which a
// directory cannot be opened at all.
//
//coverage:exempt test helper; its t.Fatalf arms run only when encoding the path or CreateFile fails
func LockDir(t testing.TB, path string) {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, 0, nil,
		syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatalf("lock %s: %v", path, err)
	}
	t.Cleanup(func() { _ = syscall.CloseHandle(handle) })
}

// LockLink holds a link itself open, not what it points at, until the test
// ends. The share mode leaves out FILE_SHARE_DELETE, so every attempt to delete
// the link fails with a sharing violation, while an open that asks for no
// delete access -- a read of its reparse data among them -- still succeeds.
// Unlike a denied right, a sharing violation binds an elevated token as well:
// administrator privileges and backup semantics get past an ACL, not past a
// share mode.
//
// FILE_FLAG_OPEN_REPARSE_POINT is what keeps the handle on the link: without
// it CreateFile follows a junction and locks the target instead.
//
//coverage:exempt test helper; its t.Fatalf arms run only when encoding the path or CreateFile fails
func LockLink(t testing.TB, path string) {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_OPEN_REPARSE_POINT|syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatalf("lock %s: %v", path, err)
	}
	t.Cleanup(func() { _ = syscall.CloseHandle(handle) })
}

// DeletePending marks an empty directory for deletion and holds it open until
// the test ends, so its name stays taken while nothing can use it: os.Lstat of
// it and os.Mkdir at it both fail with access denied, measured on 2026-09-18.
// A pending delete is a state of the file and not an access check, so an
// elevated token meets it as well.
//
// FileDispositionInfo and not its Ex form, whose POSIX semantics would drop the
// name at once. Closing the handle at cleanup completes the delete, before
// t.TempDir's own cleanup looks for the directory.
//
//coverage:exempt test helper; its t.Fatalf arms run only when encoding the path, CreateFile or setting the disposition fails
func DeletePending(t testing.TB, path string) {
	t.Helper()
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatalf("encode %s: %v", path, err)
	}
	handle, err := windows.CreateFile(name, windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { _ = windows.CloseHandle(handle) })
	remove := byte(1)
	if err := windows.SetFileInformationByHandle(handle, windows.FileDispositionInfo,
		&remove, uint32(unsafe.Sizeof(remove))); err != nil {
		t.Fatalf("mark %s for deletion: %v", path, err)
	}
}
