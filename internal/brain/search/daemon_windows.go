//go:build windows

package search

import "syscall"

const detachedProcess = 0x00000008

func detachAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: detachedProcess,
	}
}
