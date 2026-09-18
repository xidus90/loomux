//go:build !windows

package serve

import "syscall"

// detachAttrs puts the child in a session of its own, so that no terminal
// signal meant for the caller reaches the service.
//
// The flag is ignored: there is no job object here and nothing to break away
// from, so both of Spawn's attempts ask for the same thing. The second attempt
// is then a repetition, which costs one failed start on a platform where the
// first attempt fails for a reason a flag cannot fix. That is cheaper than a
// platform-dependent number of attempts, which would make Spawn's contract
// differ between the operating systems it is tested on.
func detachAttrs(bool) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
