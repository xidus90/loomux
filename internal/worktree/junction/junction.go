// Package junction makes one directory point at another on Windows.
//
// A junction and not a symlink: `os.Symlink` creates a symbolic link, which
// needs a privilege or Developer Mode, and `ln -s` under MSYS silently copies
// instead -- that mistake cost 3.7 GB of Godot editor once. A junction needs
// no privilege and is what the hand-made links in this repository's worktrees
// already are.
package junction

import (
	"errors"
	"fmt"
	"os"
)

// ErrUnsupported is the answer everywhere but Windows. Named rather than
// generic, so a caller can tell "cannot here" from "went wrong".
var ErrUnsupported = errors.New("junctions exist only on windows")

// Target is the directory `link` points at, or "" when it points at nothing.
//
// A path that is not a link, and a path that is not there at all, are both the
// empty answer with no error: the sweep asks this about every candidate it
// scans, and neither case is a fault. Anything else is a fault and says so.
//
// The answer is the target as Windows stores it, `\??\C:\dir` or `\??\C:\dir\`
// -- and not the path a caller could open. Two reasons, and neither is that
// the prefix identifies anything: the tag does that, and this checks the tag.
// First, the stored substitute name is what the kernel actually resolves; the
// print name beside it in the same buffer is decoration a tool may set to
// anything. Second, turning it into something openable means deciding what
// that is, and that decision belongs to the one caller comparing paths, in one
// place, not to this package.
//
// Whether a trailing separator is there depends on who made the junction, so a
// caller must not compare on it. Measured on 2026-09-07: `mklink /J` stores
// `\??\C:\dir` without one, `Create` below stores `\??\C:\dir\` with one, and
// both resolve. The hand-made links this package inherits came from the former.
//
// The ioctl below and not Lstat's mode bits plus `os.Readlink` -- the Lstat
// call kept here answers only "is there anything at all". Measured on
// 2026-09-07 with go1.27.0 windows/amd64: by default Lstat reports a junction
// as `Lrw-rw-rw-` and Readlink answers it, but under
// `GODEBUG=winsymlink=1,winreadlinkvolume=1` the mode is `?rw-rw-rw-`
// instead: ModeIrregular, no ModeSymlink. Both settings carry `Changed: 23` in
// the toolchain's own `internal/godebugs/table.go`, so that is the default for
// every `go` directive from 1.23 on -- and that path would answer "" for a
// real junction, which is not an error a caller could notice. Bumping this
// module's `go 1.22` line would silently turn the sweep blind. Widening the
// test to ModeIrregular is no answer either: the bit covers every other
// reparse tag as well, and what Readlink makes of those was not measured here
// -- which is reason enough not to route the sweep through it.
func Target(link string) (string, error) {
	if _, err := os.Lstat(link); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("inspecting %s: %w", link, err)
	}
	return reparseTarget(link)
}

// Remove takes the link and never what it points at.
//
// `os.Remove` and not `os.RemoveAll`: on a reparse point the first removes the
// point itself, and the second is the call that would walk into 4.2 GB of
// somebody else's directory. Measured on 2026-09-07 for the three ordinary
// delete paths -- none of them reached through the junction -- and this is the
// one place in ultraloom's own code that could.
func Remove(link string) error {
	target, err := Target(link)
	if err != nil {
		return err
	}
	if target == "" {
		return fmt.Errorf("%s is not a link; refusing to remove it", link)
	}
	if err := os.Remove(link); err != nil {
		return fmt.Errorf("removing the link %s: %w", link, err)
	}
	return nil
}
