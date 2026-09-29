//go:build !windows

package reader

import "syscall"

// errNotFound is the cause os.ReadFile reports off Windows for a file that
// is not there.
var errNotFound error = syscall.ENOENT
