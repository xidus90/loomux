package config

import (
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/lock"
)

// AreaLockPath is the lock `reindex` and `approve` share for one area. It
// lies beside `<state>/areas/<scope>`, never inside it: lock.ReplaceDir
// swaps that directory whole, and an open file in it would hold the rename
// on Windows. A writable area has no such directory and is locked there all
// the same, since its register is read and written by both commands too.
func AreaLockPath(area Area, stateDir string) string {
	return filepath.Join(stateDir, "areas", flat(area.Scope)+".lock")
}

// LockArea blocks until the area's lock is free, takes it and hands back
// its release.
func LockArea(area Area, stateDir string) (func() error, error) {
	path := AreaLockPath(area, stateDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	handle, err := lock.Acquire(path)
	if err != nil {
		return nil, err
	}
	return handle.Release, nil
}
