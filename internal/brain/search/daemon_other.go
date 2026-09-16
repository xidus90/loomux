//go:build !windows

package search

import "syscall"

func detachAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
