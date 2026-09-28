//go:build !windows

package child

import (
	"errors"
	"os"
	"syscall"
)

// alive sends the null signal: delivered or refused for lack of permission,
// the process runs; gone, it does not. FindProcess never fails on Unix.
//
//coverage:exempt POSIX-only; the gate is measured on Windows, the Linux leg of ci.yml runs its tests
func alive(pid int) bool {
	p, _ := os.FindProcess(pid)
	err := p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
