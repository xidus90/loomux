// Package swap replaces the pilot binary while hooks may be running it.
// Windows lets a running executable be renamed but not overwritten, so the
// current one is moved aside first.
package swap

import (
	"fmt"
	"os"
	"path/filepath"
)

// Swap moves dir/loomux.new.exe to dir/loomux.exe, keeping the previous one
// as dir/loomux.old.exe.
//
//coverage:exempt both Rename error arms need the file system to refuse a rename in a directory the Stat just read
func Swap(dir string) error {
	current := filepath.Join(dir, "loomux.exe")
	next := filepath.Join(dir, "loomux.new.exe")
	old := filepath.Join(dir, "loomux.old.exe")
	if _, err := os.Stat(next); err != nil {
		return fmt.Errorf("no new binary at %s: %w", next, err)
	}
	// Best effort: an older binary may still be running and cannot be removed.
	_ = os.Remove(old)
	if _, err := os.Stat(current); err == nil {
		if err := os.Rename(current, old); err != nil {
			return fmt.Errorf("moving %s aside: %w", current, err)
		}
	}
	if err := os.Rename(next, current); err != nil {
		return fmt.Errorf("putting %s in place: %w", next, err)
	}
	return nil
}
