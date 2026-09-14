package swap

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSwapPutsTheNewBinaryInPlaceAndKeepsTheOld(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	if err := Swap(dir); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "new" || read(t, filepath.Join(dir, "loomux.old.exe")) != "old" {
		t.Fatal("binaries not swapped")
	}
	if _, err := os.Stat(filepath.Join(dir, "loomux.new.exe")); !os.IsNotExist(err) {
		t.Fatal("new binary still there")
	}
}

func TestSwapWorksWithoutAPreviousBinary(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	if err := Swap(dir); err != nil || read(t, filepath.Join(dir, "loomux.exe")) != "new" {
		t.Fatalf("err %v", err)
	}
}

func TestSwapFailsWhenTheOldSlotCannotBeFreed(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	if err := os.MkdirAll(filepath.Join(dir, "loomux.old.exe", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Swap(dir); err == nil {
		t.Fatal("want error")
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "old" || read(t, filepath.Join(dir, "loomux.new.exe")) != "new" {
		t.Fatal("a failed swap must leave both binaries where they were")
	}
}

func TestSwapFailsWhenTheTargetIsADanglingJunction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("junctions are a Windows construct")
	}
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.new.exe"), "new")
	target := filepath.Join(dir, "gone")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(dir, "loomux.exe"), target).CombinedOutput()
	if err != nil {
		t.Fatalf("mklink: %v: %s", err, out)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := Swap(dir); err == nil {
		t.Fatal("want error")
	}
	if read(t, filepath.Join(dir, "loomux.new.exe")) != "new" {
		t.Fatal("the new binary must stay in place")
	}
}

func TestSwapRefusesWithoutANewBinary(t *testing.T) {
	if err := Swap(t.TempDir()); err == nil {
		t.Fatal("want error")
	}
}
