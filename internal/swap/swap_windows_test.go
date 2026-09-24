//go:build windows

package swap

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// hold opens path with read sharing only: while the handle lives, the file is
// neither deletable nor renamable. That stands in for a running image on the
// branch freeSlot reads -- an image cannot be deleted either -- and is stricter
// on the other one: an image can be renamed, which is what this whole swap
// rests on (see the package doc). Go's own os.Open would serve for the delete
// branch as well, since it withholds delete sharing, but it lets others write.
func hold(t *testing.T, path string) {
	t.Helper()
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ,
		nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatalf("holding %s: %v", path, err)
	}
	// Before t.TempDir's own cleanup, which cannot remove a held file.
	t.Cleanup(func() { _ = syscall.CloseHandle(h) })
}

func TestSwapSucceedsWhileAProcessHoldsTheOldImage(t *testing.T) {
	// The reported failure, end to end: a bridge or a service started from an
	// earlier swap still runs from loomux.old.exe. The swap must go through.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	write(t, filepath.Join(dir, "loomux.old.exe"), "running")
	hold(t, filepath.Join(dir, "loomux.old.exe"))
	if err := Swap(dir); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "new" || read(t, filepath.Join(dir, "loomux.old.1.exe")) != "old" {
		t.Fatal("the swap did not go around the held image")
	}
	if read(t, filepath.Join(dir, "loomux.old.exe")) != "running" {
		t.Fatal("the image a process still runs from was taken away from it")
	}
}

func TestSwapGoesAroundASlotThatAnyOpenHandleHolds(t *testing.T) {
	// Not only an image somebody runs from: an ordinary read handle is enough
	// for Windows to refuse the removal, and the swap must go around that one
	// too rather than rename onto a name it could not free.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	slot := filepath.Join(dir, "loomux.old.exe")
	write(t, slot, "held")
	f, err := os.Open(slot)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := Swap(dir); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "new" || read(t, filepath.Join(dir, "loomux.old.1.exe")) != "old" {
		t.Fatal("the swap used a name it had not been able to free")
	}
}

func TestSwapFailsWhenTheCurrentBinaryCannotBeMovedAside(t *testing.T) {
	// The one thing the numbering cannot rescue: something holds the binary so
	// that it cannot be renamed at all. A running loomux.exe is not that holder
	// -- an image stays renamable -- so this stands for a foreign one, a
	// scanner or a backup agent that opened the file without delete sharing.
	// Then the swap says so and changes nothing.
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	hold(t, filepath.Join(dir, "loomux.exe"))
	if err := Swap(dir); err == nil {
		t.Fatal("want error")
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "old" || read(t, filepath.Join(dir, "loomux.new.exe")) != "new" {
		t.Fatal("a failed swap must leave both binaries where they were")
	}
}
