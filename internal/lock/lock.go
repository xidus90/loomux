// Package lock is the cross-process lock, ported from the reference's
// locking.py.
//
// The guarantee is the operating system's: a handle holds the lock for as long
// as it is open, and a process that dies releases it without anyone cleaning
// up. The lock file stays empty: the byte range a Windows lock holds is denied
// to every other handle, reads included, so anything written there would be
// unreadable for exactly as long as the lock is held -- the only time it would
// mean anything. PID() is the holder's own PID, for its own logging; a running
// service's PID for display lives in serve.json.
package lock

import (
	"fmt"
	"os"
	"time"
)

// waitFreeTick is how often WaitFree asks again.
const waitFreeTick = 250 * time.Millisecond

// Handle is a held lock. Release it exactly once.
type Handle struct {
	file *os.File
	pid  int
}

// TryAcquire takes the lock at path, or reports false when someone else holds
// it.
func TryAcquire(path string) (*Handle, bool, error) {
	return acquire(path, tryLock)
}

// Acquire blocks until the lock at path is free.
func Acquire(path string) (*Handle, error) {
	handle, _, err := acquire(path, func(file *os.File) (bool, error) {
		err := waitLock(file)
		return err == nil, err
	})
	return handle, err
}

// WaitFree waits until nobody holds the lock at path: every 250 ms it takes
// the lock and releases it again, and returns once that worked. It leaves
// nothing behind -- the file is never written -- but it is not free of effect:
// while it probes, it holds the lock itself, so a TryAcquire racing it can be
// refused although the previous holder is long gone. It also makes no promise
// about the moment after it returns; the caller that wants the lock takes it
// itself, and may find it taken.
func WaitFree(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		handle, free, err := TryAcquire(path)
		if err != nil {
			return err
		}
		if free {
			return handle.Release()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("lock %s still held after %s", path, timeout)
		}
		time.Sleep(waitFreeTick)
	}
}

// PID is the process that holds this handle.
func (h *Handle) PID() int { return h.pid }

// Release gives the lock up.
func (h *Handle) Release() error {
	if err := unlock(h.file); err != nil {
		h.file.Close()
		return err
	}
	return h.file.Close()
}

// acquire is the one path both forms take; take is what turns the open file
// into a held lock, blocking or not.
func acquire(path string, take func(*os.File) (bool, error)) (*Handle, bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("open lock %s: %w", path, err)
	}
	held, err := take(file)
	if err != nil {
		file.Close()
		return nil, false, fmt.Errorf("lock %s: %w", path, err)
	}
	if !held {
		file.Close()
		return nil, false, nil
	}
	return &Handle{file: file, pid: os.Getpid()}, true, nil
}
