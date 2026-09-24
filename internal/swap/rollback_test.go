package swap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failRenames swaps the rename seam for one that refuses the calls whose
// 1-based numbers are listed and passes every other call through.
func failRenames(t *testing.T, refused ...int) {
	t.Helper()
	calls := 0
	rename = func(from, to string) error {
		calls++
		for _, n := range refused {
			if calls == n {
				return errors.New("refused")
			}
		}
		return os.Rename(from, to)
	}
	t.Cleanup(func() { rename = os.Rename })
}

func TestSwapPutsTheOldBinaryBackWhenTheNewOneCannotTakeItsPlace(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	failRenames(t, 2)
	err := Swap(dir)
	if err == nil || !strings.Contains(err.Error(), filepath.Join(dir, "loomux.new.exe")) || !strings.Contains(err.Error(), filepath.Join(dir, "loomux.old.exe")) {
		t.Fatalf("Swap = %v", err)
	}
	if strings.Contains(err.Error(), "rollback") {
		t.Fatalf("a rollback that went through was reported as failed: %v", err)
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "old" {
		t.Fatal("the old binary is not back in place")
	}
	if read(t, filepath.Join(dir, "loomux.new.exe")) != "new" {
		t.Fatal("the new binary must stay where it was")
	}
}

func TestSwapReportsARollbackThatFailsToo(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	failRenames(t, 2, 3)
	err := Swap(dir)
	if err == nil || !strings.Contains(err.Error(), "rollback") || !strings.Contains(err.Error(), filepath.Join(dir, "loomux.old.exe")) {
		t.Fatalf("Swap = %v", err)
	}
	if read(t, filepath.Join(dir, "loomux.old.exe")) != "old" {
		t.Fatal("the old binary is lost")
	}
}
