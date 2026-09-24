// Package swap replaces a loomux binary while processes may be running it:
// the pilot binary of a checkout, and the machine-wide one the self-update
// installs.
// Windows lets a running executable be renamed but not overwritten, so the
// current one is moved aside first.
//
// It cannot be moved aside to a single fixed name. A process keeps running
// from the image it was started as, and Windows holds that file: a predecessor
// from an earlier swap still owns loomux.old.exe, which is then neither
// removable nor a rename target, and every swap after it fails with "access
// denied" on the first rename. That was not felt while nothing lived long --
// `loomux serve` and a bridge do, so running the service in a worktree and
// committing in it would exclude each other.
//
// So the old images are numbered. Each swap first frees every slot whose
// process has ended and then takes the first free one. The directory therefore
// holds at most one image per generation that is still running -- two in the
// steady state of one service and one bridge -- and never grows without bound.
package swap

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// oldSlots bounds how many generations may be held at once. It is generations,
// not processes: everything started from one image shares its slot, and the
// slot returns on the day the last of them ends.
const oldSlots = 16

// Swap moves dir/loomux.new.exe to dir/loomux.exe, keeping the previous one
// as dir/loomux.old.exe, or as the first free numbered slot beside it when a
// running process still holds that name.
func Swap(dir string) error {
	current := filepath.Join(dir, "loomux.exe")
	next := filepath.Join(dir, "loomux.new.exe")
	if _, err := os.Stat(next); err != nil {
		return fmt.Errorf("no new binary at %s: %w", next, err)
	}
	if _, err := os.Stat(current); err == nil {
		aside, err := freeSlot(dir)
		if err != nil {
			return err
		}
		if err := os.Rename(current, aside); err != nil {
			return fmt.Errorf("moving %s aside: %w", current, err)
		}
	}
	if err := os.Rename(next, current); err != nil {
		return fmt.Errorf("putting %s in place: %w", next, err)
	}
	return nil
}

// freeSlot sweeps and picks in one pass: it removes every old image that has
// become removable and answers with the first slot that nothing holds. The
// sweep does not stop at that first free slot, because a slot released by a
// process that ended would otherwise lie on disk until a swap happened to need
// exactly its number.
func freeSlot(dir string) (string, error) {
	free := ""
	for i := range oldSlots {
		name := filepath.Join(dir, slotName(i))
		// A removal that is refused means the slot is taken, whatever refuses
		// it. Windows refuses it for an image a process runs from (measured on
		// this box: the delete is denied, a rename of the same file goes
		// through) and for a file held open without FILE_SHARE_DELETE, which
		// Go's own os.Open is -- syscall.Open asks for FILE_SHARE_READ and
		// FILE_SHARE_WRITE and no more. The portable tests use a non-empty
		// directory, which refuses it too. A holder that does share delete
		// would let the removal through; none of the holders this guards
		// against is one.
		if err := os.Remove(name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if free == "" {
			free = name
		}
	}
	if free == "" {
		return "", fmt.Errorf("all %d old-binary slots in %s are held; find the holders with: Get-Process loomux, then end the ones nothing needs any more and swap again", oldSlots, dir)
	}
	return free, nil
}

// slotName is loomux.old.exe for the first slot and loomux.old.<n>.exe after
// it. The first keeps the name the gate has always written, so the ordinary
// case still looks the way a human knows it.
func slotName(i int) string {
	if i == 0 {
		return "loomux.old.exe"
	}
	return fmt.Sprintf("loomux.old.%d.exe", i)
}
