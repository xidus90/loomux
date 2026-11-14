package serve

import "syscall"

const (
	// detachedProcess gives the child no console at all. It is not the same as
	// CREATE_NO_WINDOW: that one still attaches a console, only hidden, and a
	// console is a thing the parent's exit can take away.
	detachedProcess = 0x00000008
	// createNoWindow gives the child a console of its own without a window.
	// Every console program the child starts shares it; under
	// DETACHED_PROCESS Windows opens a new, visible console for each of them.
	createNoWindow = 0x08000000
	// createBreakawayFromJob asks to leave the job object the parent is in. A
	// job without JOB_OBJECT_LIMIT_BREAKAWAY_OK refuses the request and the
	// creation fails outright, which is why Spawn has a second attempt.
	createBreakawayFromJob = 0x01000000
)

// detachAttrs are the creation flags of one attempt.
func detachAttrs(breakaway, console bool) *syscall.SysProcAttr {
	flags := uint32(detachedProcess)
	if console {
		flags = createNoWindow
	}
	if breakaway {
		flags |= createBreakawayFromJob
	}
	return &syscall.SysProcAttr{CreationFlags: flags}
}
