package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/lock"
)

func TestTheAreaLockLiesBesideTheSwappedDirectory(t *testing.T) {
	state := t.TempDir()
	area := Area{Scope: "project/a", ReadOnly: true}
	if got, want := AreaLockPath(area, state), ManifestDir(area, state)+".lock"; got != want {
		t.Fatalf("%s, want %s", got, want)
	}
	// A writable area has no directory there and gets the lock all the same.
	if got := AreaLockPath(Area{Scope: "project/a", Path: "/repo"}, state); got != filepath.Join(state, "areas", "project-a.lock") {
		t.Fatal(got)
	}
}

// Within one process a second handle on a held lock is refused, which is
// what internal/lock's TestTryAcquireRefusesAHeldLock records.
func TestLockAreaHoldsUntilReleased(t *testing.T) {
	state := t.TempDir()
	area := Area{Scope: "project/a"}
	release, err := LockArea(area, state)
	if err != nil {
		t.Fatal(err)
	}
	if handle, free, _ := lock.TryAcquire(AreaLockPath(area, state)); free {
		_ = handle.Release()
		t.Fatal("the lock was free while held")
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	handle, free, err := lock.TryAcquire(AreaLockPath(area, state))
	if err != nil || !free {
		t.Fatal(free, err)
	}
	_ = handle.Release()
}

func TestLockAreaPassesOnWhatStopsIt(t *testing.T) {
	area := Area{Scope: "project/a"}
	t.Run("no areas directory", func(t *testing.T) {
		state := t.TempDir()
		if err := os.WriteFile(filepath.Join(state, "areas"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LockArea(area, state); err == nil {
			t.Fatal("locked below a file")
		}
	})
	t.Run("a directory where the lock goes", func(t *testing.T) {
		state := t.TempDir()
		if err := os.MkdirAll(AreaLockPath(area, state), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := LockArea(area, state); err == nil {
			t.Fatal("locked a directory")
		}
	})
}
