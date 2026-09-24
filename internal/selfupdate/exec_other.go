//go:build !windows

package selfupdate

import "syscall"

// hiddenAttrs asks for nothing: a program started here opens no window of its
// own, whatever its parent has or lacks.
func hiddenAttrs() *syscall.SysProcAttr {
	return nil
}
