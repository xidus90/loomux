package selfupdate

import "syscall"

// createNoWindow gives a console program a console without a window. serve,
// which runs the pass, has no console to share (DETACHED_PROCESS), so without
// this flag Windows opens a new, visible one for every call.
const createNoWindow = 0x08000000

// hiddenAttrs are the creation flags of every external call.
func hiddenAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNoWindow, HideWindow: true}
}
