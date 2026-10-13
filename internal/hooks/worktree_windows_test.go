//go:build windows

package hooks

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// hold keeps a handle on `path` open until the test ends, with the access and
// the share rights the caller names -- an access of none takes no part in
// sharing checks, so it blocks nothing. It always opens the path itself and not
// what a link there points at.
func hold(t *testing.T, path string, access uint32, share uint32) {
	t.Helper()
	wide, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(wide, access, share, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		t.Fatalf("holding %s: %v", path, err)
	}
	t.Cleanup(func() { windows.CloseHandle(handle) })
}

// A link that cannot be taken out stops the removal before git is asked --
// a git that follows junctions is the reason, and handing it the link anyway
// would be the loss this refusal exists to prevent. An open handle without the
// delete share right is what makes Windows refuse the delete.
func TestWorktreeRemoveRefusesWhenALinkCannotBeTakenOut(t *testing.T) {
	main, worktree := worktreeFixture(t)
	mkdirAll(t, filepath.Join(main, ".tools", "godot"))
	link := filepath.Join(worktree, ".tools")
	mklink(t, link, filepath.Join(main, ".tools"))
	hold(t, link, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE)
	stubGit(t, func(string, string) ([]byte, error) {
		t.Error("git was asked although a link was still in the tree")
		return nil, nil
	})

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), link) || !strings.Contains(stderr.String(), "could not be removed") {
		t.Fatalf("stderr = %q, want the link named and the refusal said", stderr)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want nothing claimed", stdout)
	}
	if !registered(t, main, worktree) {
		t.Fatal("the worktree was dropped although git was never asked")
	}
}

// A directory that cannot be listed is a part of the tree nobody has looked
// at, and a link may be in it: the removal stops there too.
func TestWorktreeRemoveRefusesWhenADirectoryCannotBeListed(t *testing.T) {
	main, worktree := worktreeFixture(t)
	sealed := filepath.Join(worktree, "sealed")
	mkdirAll(t, sealed)
	hold(t, sealed, windows.GENERIC_READ, 0)
	stubGit(t, func(string, string) ([]byte, error) {
		t.Error("git was asked although a directory had not been looked at")
		return nil, nil
	})

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "listing "+sealed) {
		t.Fatalf("stderr = %q, want the directory named", stderr)
	}
	if !registered(t, main, worktree) {
		t.Fatal("the worktree was dropped although git was never asked")
	}
}
