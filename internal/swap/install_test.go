package swap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevDirIsTheCheckoutsFolderUnderLocalAppData(t *testing.T) {
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	root := filepath.Join(t.TempDir(), "compat-version-breadth")
	want := filepath.Join(local, "loomux", "dev", "compat-version-breadth")
	if got := DevDir(root); got != want {
		t.Fatalf("DevDir = %q; want %q", got, want)
	}
}

func TestDevDirIsEmptyWithoutLocalAppData(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	if got := DevDir("."); got != "" {
		t.Fatalf("DevDir = %q; want empty", got)
	}
}

func TestInstallReportsACopyItCannotWrite(t *testing.T) {
	old := create
	t.Cleanup(func() { create = old })
	readOnly := filepath.Join(t.TempDir(), "ro")
	write(t, readOnly, "")
	create = func(string) (*os.File, error) { return os.Open(readOnly) }
	src := filepath.Join(t.TempDir(), "loomux.exe")
	write(t, src, "x")
	if err := Install(src, t.TempDir()); err == nil {
		t.Fatal("a copy that cannot be written must fail")
	}
}

func TestInstallCopiesAndKeepsTheSource(t *testing.T) {
	src := filepath.Join(t.TempDir(), "loomux.exe")
	write(t, src, "build")
	dir := filepath.Join(t.TempDir(), "dev", "loomux")
	if err := Install(src, dir); err != nil {
		t.Fatal(err)
	}
	if read(t, src) != "build" || read(t, filepath.Join(dir, "loomux.exe")) != "build" {
		t.Fatal("the source must stay and the copy must be in place")
	}
}

func TestInstallRotatesThePreviousBinary(t *testing.T) {
	src := filepath.Join(t.TempDir(), "loomux.exe")
	write(t, src, "new")
	dir := t.TempDir()
	write(t, filepath.Join(dir, "loomux.exe"), "old")
	if err := Install(src, dir); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(dir, "loomux.exe")) != "new" || read(t, filepath.Join(dir, "loomux.old.exe")) != "old" {
		t.Fatal("the previous binary must be kept aside")
	}
}

func TestInstallReportsAMissingSource(t *testing.T) {
	src := filepath.Join(t.TempDir(), "missing.exe")
	if err := Install(src, t.TempDir()); err == nil || !strings.Contains(err.Error(), src) {
		t.Fatalf("err %v must name %s", err, src)
	}
}

func TestInstallReportsADirItCannotMake(t *testing.T) {
	src := filepath.Join(t.TempDir(), "loomux.exe")
	write(t, src, "x")
	blocker := filepath.Join(t.TempDir(), "file")
	write(t, blocker, "")
	if err := Install(src, filepath.Join(blocker, "dev")); err == nil {
		t.Fatal("a dir under a file cannot be made")
	}
}

func TestInstallReportsACopyItCannotCreate(t *testing.T) {
	old := create
	t.Cleanup(func() { create = old })
	create = func(string) (*os.File, error) { return nil, errors.New("refused") }
	src := filepath.Join(t.TempDir(), "loomux.exe")
	write(t, src, "x")
	dir := t.TempDir()
	if err := Install(src, dir); err == nil || !strings.Contains(err.Error(), dir) {
		t.Fatalf("err %v must name %s", err, dir)
	}
}
