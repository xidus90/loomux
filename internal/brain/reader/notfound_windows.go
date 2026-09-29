package reader

import "syscall"

// errNotFound is the cause os.ReadFile reports on Windows for a file that
// is not there in a directory that is.
var errNotFound error = syscall.ERROR_FILE_NOT_FOUND
