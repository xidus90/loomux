package lock

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// AsideSuffix names the directory a ReplaceDir puts the old stock in while it
// swaps. It is a fixed sibling and not a random name on purpose: only a name
// the next run can recognise lets that run finish an interrupted swap, and a
// random one would leave a directory nobody dares touch.
const AsideSuffix = ".loomux-aside"

// dirOps are the three filesystem calls a swap makes. They are values and not
// calls into the package os, because the promise of a swap is about the
// instants between them -- a process killed after the first rename, a remove
// the operating system refuses -- and no timing trick reaches those instants
// from outside.
type dirOps struct {
	rename func(source, target string) error
	remove func(path string) error
	stat   func(path string) (fs.FileInfo, error)
}

// realDirs is what the exported functions swap with.
func realDirs() dirOps {
	return dirOps{rename: replaceEventually, remove: os.RemoveAll, stat: os.Stat}
}

// ReplaceDir puts staging in the place of target, whole or not at all.
//
// The state directory resolves an area by its directory, not by the single
// file (config.ManifestDir), and privacy.VisibleAreas gives up on the
// first missing declaration for every area at once. A target written file by
// file therefore has a window in which the area exists but is incomplete, and
// in that window the whole vault answers nothing. Writing beside it and
// swapping once closes that window: at every instant target is either the old
// stock whole, the new stock whole, or absent.
//
// Absent is not a harmless third state, and saying so would be the comfortable
// answer rather than the true one. For a registered read-only area an absent
// directory is a missing declaration, and privacy.VisibleAreas gives up on the
// first of those for every area at once -- the very refusal this movement
// exists to prevent. An absent target is a missing declaration for a
// read-only area, and every brain reader refuses it; Recover, which `index`
// calls before it reads and `approve` before it writes, closes that window.
//
// Neither rename is atomic against the other, so a process killed between
// them leaves the old stock under target+AsideSuffix and no target. Recover
// undoes exactly that on the next call, which is why the aside is named and
// not random -- but only a caller that asks. Today the index run is the one
// that does, and it asks for the area it is about to rebuild.
func ReplaceDir(staging, target string) error {
	return replaceDir(staging, target, realDirs())
}

// replaceDir is ReplaceDir over given filesystem calls.
func replaceDir(staging, target string, ops dirOps) error {
	aside := target + AsideSuffix
	if err := recoverAside(target, ops); err != nil {
		return err
	}
	moved, err := moveAway(target, aside, ops)
	if err != nil {
		return err
	}
	if err := ops.rename(staging, target); err != nil {
		if moved {
			// Best effort, and the error of the swap is the one worth
			// reporting: did this fail too, Recover finishes the job on the
			// next run, and until then target is absent rather than half.
			_ = ops.rename(aside, target)
		}
		return fmt.Errorf("%s: %w", target, err)
	}
	if moved {
		if err := ops.remove(aside); err != nil {
			return fmt.Errorf("%s: %w", aside, err)
		}
	}
	return nil
}

// moveAway puts an existing target aside and says whether it did. A leftover
// aside cannot be in the way here: recoverAside ran first and either moved it
// back or found a target that outranks it.
//
// A target that is neither there nor readable is not an absence: taken for
// one, the swap would rename over a stock this process merely cannot see.
func moveAway(target, aside string, ops dirOps) (bool, error) {
	if _, err := ops.stat(target); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := ops.rename(target, aside); err != nil {
		return false, fmt.Errorf("%s: %w", target, err)
	}
	return true, nil
}

// Recover finishes a swap a killed process left half-done: an aside beside a
// target that is not there is the old stock, and it goes back.
//
// Is the target there, the aside is the remains of a swap that did get
// through and never cleaned up, and it goes. Both arms leave the directory
// the readers see whole; neither ever prefers the aside over a target that
// exists. No aside, nothing to do -- the ordinary case, and the reason this
// is cheap enough to run before every swap.
func Recover(target string) error {
	return recoverAside(target, realDirs())
}

// recoverAside is Recover over given filesystem calls.
func recoverAside(target string, ops dirOps) error {
	aside := target + AsideSuffix
	if _, err := ops.stat(aside); err != nil {
		return nil
	}
	if _, err := ops.stat(target); err == nil {
		if err := ops.remove(aside); err != nil {
			return fmt.Errorf("%s: %w", aside, err)
		}
		return nil
	}
	if err := ops.rename(aside, target); err != nil {
		return fmt.Errorf("%s: %w", aside, err)
	}
	return nil
}

// StagingDir makes an empty directory beside target for a ReplaceDir to fill.
// Beside it and not in the temporary directory, because a rename across two
// volumes is no rename at all -- it fails, and on the one path where this
// matters that failure would come after the whole area was written.
func StagingDir(target string) (string, error) {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(parent, filepath.Base(target)+".*.staging")
}
