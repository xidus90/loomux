package lock_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/lock"
)

func TestTryAcquireTakesAFreeLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	h, ok, err := lock.TryAcquire(path)
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if !ok {
		t.Fatal("expected to take a free lock")
	}
	defer h.Release()
	if h.PID() != os.Getpid() {
		t.Errorf("PID() = %d, want %d", h.PID(), os.Getpid())
	}
}

func TestTryAcquireRefusesAHeldLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	first, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("first TryAcquire: %v, ok=%v", err, ok)
	}
	defer first.Release()

	// Same process, second handle: the file lock is per handle, not per process,
	// so this is the same refusal a second process would see.
	second, ok, err := lock.TryAcquire(path)
	if err != nil {
		t.Fatalf("second TryAcquire: %v", err)
	}
	if ok {
		second.Release()
		t.Fatal("expected the second TryAcquire to be refused")
	}
}

func TestReleaseLetsTheNextTakerIn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	first, _, err := lock.TryAcquire(path)
	if err != nil {
		t.Fatalf("TryAcquire: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	second, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("after Release: %v, ok=%v", err, ok)
	}
	second.Release()
}

func TestAcquireRefusesAnUnwritablePath(t *testing.T) {
	// A directory where a file is expected: the open fails, and the failure must
	// come back as an error rather than as a taken lock.
	dir := t.TempDir()
	if _, _, err := lock.TryAcquire(dir); err == nil {
		t.Fatal("expected an error for a directory path")
	}
	if _, err := lock.Acquire(dir); err == nil {
		t.Fatal("expected an error for a directory path")
	}
}

// TestAcquireBlocksUntilTheHolderReleases is the one check that the blocking
// form really blocks: a refusal alone would also be satisfied by a call that
// returns an error immediately.
func TestAcquireBlocksUntilTheHolderReleases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	first, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("TryAcquire: %v, ok=%v", err, ok)
	}

	type result struct {
		handle *lock.Handle
		err    error
	}
	done := make(chan result, 1)
	go func() {
		h, err := lock.Acquire(path)
		done <- result{handle: h, err: err}
	}()

	select {
	case r := <-done:
		if r.handle != nil {
			r.handle.Release()
		}
		first.Release()
		t.Fatalf("Acquire returned while the lock was held: %v", r.err)
	case <-time.After(200 * time.Millisecond):
	}

	if err := first.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("Acquire after Release: %v", r.err)
		}
		if err := r.handle.Release(); err != nil {
			t.Fatalf("Release of the second handle: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Acquire did not return after the holder released")
	}
}

func TestWaitFreeRunsIntoTheTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	held, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("TryAcquire: %v, ok=%v", err, ok)
	}
	defer held.Release()

	const timeout = 600 * time.Millisecond
	start := time.Now()
	err = lock.WaitFree(path, timeout)
	if err == nil {
		t.Fatal("expected WaitFree to run into the timeout")
	}
	// An I/O error would satisfy err != nil just as well, and it would come
	// back at once: the timeout is what this test is about.
	if waited := time.Since(start); waited < timeout {
		t.Errorf("WaitFree gave up after %v, before the %v timeout: %v", waited, timeout, err)
	}
	if !strings.Contains(err.Error(), "still held") {
		t.Errorf("WaitFree error = %v, want it to name the held lock", err)
	}
}

func TestWaitFreeReturnsAtOnceOnAFreeLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	held, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("TryAcquire: %v, ok=%v", err, ok)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}

	start := time.Now()
	if err := lock.WaitFree(path, 10*time.Second); err != nil {
		t.Fatalf("WaitFree: %v", err)
	}
	if waited := time.Since(start); waited > 250*time.Millisecond {
		t.Errorf("WaitFree waited %v on a free lock, want no tick at all", waited)
	}
}

func TestWaitFreeReportsAnUnusablePath(t *testing.T) {
	if err := lock.WaitFree(t.TempDir(), time.Second); err == nil {
		t.Fatal("expected an error for a directory path")
	}
}
