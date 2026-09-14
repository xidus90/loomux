//go:build !windows

package guard

import "path/filepath"

// The half of the resolver that stands where `golang.org/x/sys/windows`
// cannot build. The barrier is written for Windows -- the registration it
// reads, the spellings it closes and the two holes this file's Windows
// twin exists to shut are all Windows ones -- but a build that fails on a
// runner is a barrier nobody can ship, so the package compiles here too.
//
// Nothing in this file is covered by a test, because no test on this
// machine executes it. That is the one coverage exclusion in the package
// and it is named rather than configured: the arms it stands in for are
// measured on Windows, and a posix run of the suite would exercise these
// two functions against a file system that has neither an 8.3 alias nor
// a junction.

// finalName is `_getfinalpathname`'s posix counterpart. `EvalSymlinks`
// answers exactly what is wanted here: an existing path with every
// symlink on it followed, and an error where the path is not there --
// which is the signal `finalPath` reads to shorten and ask again.
func finalName(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// stopsResolving lets the walk shorten the path on any failure at all,
// which is the reading the Windows side reaches through a list of error
// numbers. There is no equivalent list here: `EvalSymlinks` folds every
// cause into one error, and the case the list exists to keep apart -- a
// cycle -- is caught by `readlinkDeep` on this platform as on the other.
func stopsResolving(error) bool { return true }
