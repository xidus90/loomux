package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// closedFile is a file whose handle is gone: every call against it fails the
// way a lock call fails when the operating system refuses, which no open mode
// this process can request would produce.
func closedFile(t *testing.T) *os.File {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "closed")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return file
}

func TestAcquireReportsARefusedLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "serve.lock")
	refused := errors.New("refused")
	_, ok, err := acquire(path, func(*os.File) (bool, error) { return false, refused })
	if ok {
		t.Fatal("expected no handle")
	}
	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want it to wrap %v", err, refused)
	}
}

func TestReleaseReportsAFailedUnlock(t *testing.T) {
	handle := &Handle{file: closedFile(t), pid: os.Getpid()}
	if err := handle.Release(); err == nil {
		t.Fatal("expected an error unlocking a closed file")
	}
}

func TestTryLockReportsAnErrorThatIsNotAHeldLock(t *testing.T) {
	if _, err := tryLock(closedFile(t)); err == nil {
		t.Fatal("expected an error locking a closed file")
	}
}
