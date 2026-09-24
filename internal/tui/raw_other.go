//go:build !windows

package tui

import "os"

// enableVT has nothing to do: POSIX terminals interpret escape sequences
// without being asked.
//
//coverage:exempt nothing to switch on outside Windows
func enableVT(*os.File) func() { return func() {} }
