package web

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// unlistable makes path a link to itself: opening it fails with an error that
// is not "does not exist", so a listing of it is an error. A file where a
// folder belongs is no stand-in: Windows reports its listing as a missing
// path.
func unlistable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if filepath.Separator == '\\' {
		// A junction is the link Windows lets an unelevated user make.
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", path, path).CombinedOutput(); err != nil {
			t.Skipf("no junction on this machine: %v (%s)", err, out)
		}
	} else if err := os.Symlink(path, path); err != nil {
		t.Skipf("no symlink on this machine: %v", err)
	}
	if _, err := os.ReadDir(path); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Skipf("the loop lists or reads as absent here: %v", err)
	}
}
