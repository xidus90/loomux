package target

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// unlistable makes path a link to itself: opening it fails with an error
// that is not "does not exist", so a listing of it is an error.
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

func proposalFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// Without hard links the name is claimed by an exclusive create instead.
func TestStoreProposalWithoutHardLinks(t *testing.T) {
	restore := linkFile
	linkFile = func(string, string) error { return errors.New("not supported") }
	t.Cleanup(func() { linkFile = restore })
	dir := t.TempDir()
	now := time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC)
	first, err1 := storeProposal(dir, now, Proposal{Op: "set", Key: "a"})
	second, err2 := storeProposal(dir, now, Proposal{Op: "set", Key: "b"})
	if err1 != nil || err2 != nil || first != "20260924T101530Z-001" || second != "20260924T101530Z-002" {
		t.Fatalf("%s %v %s %v", first, err1, second, err2)
	}
	data, _ := os.ReadFile(filepath.Join(dir, second+".json"))
	if names := proposalFiles(t, dir); len(names) != 2 || !strings.Contains(string(data), `"key": "b"`) {
		t.Fatalf("%v %s", names, data)
	}
	// A directory that cannot be made is reported, not retried.
	if _, err := storeProposal(filepath.Join(dir, "missing", "\x00"), now, Proposal{}); err == nil {
		t.Fatal("no error")
	}
}

// Two proposals of one second get two names, in the order they were made.
func TestStoreProposalTakesTheNextFreeName(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC)
	first, err := storeProposal(dir, now, Proposal{Op: "set", Key: "a"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := storeProposal(dir, now, Proposal{Op: "set", Key: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if first != "20260924T101530Z-001" || second != "20260924T101530Z-002" {
		t.Fatalf("%s %s", first, second)
	}
	if names := proposalFiles(t, dir); len(names) != 2 {
		t.Fatalf("leftovers: %v", names)
	}
}

// stubProposalLink replaces the hard link for one test.
func stubProposalLink(t *testing.T, link func(string, string) error) {
	t.Helper()
	restore := linkFile
	linkFile = link
	t.Cleanup(func() { linkFile = restore })
}

// The name is claimed by linking the complete temporary file: what is
// linked is a file that exists and already holds the proposal.
func TestStoreProposalLinksTheWrittenTemporaryFile(t *testing.T) {
	var linked []string
	stubProposalLink(t, func(oldname, newname string) error {
		data, err := os.ReadFile(oldname)
		if err != nil || !strings.Contains(string(data), `"key": "a"`) {
			t.Errorf("linked %q: %q %v", oldname, data, err)
		}
		linked = append(linked, oldname)
		return os.Link(oldname, newname)
	})
	dir := t.TempDir()
	id, err := storeProposal(dir, time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC), Proposal{Op: "set", Key: "a"})
	if err != nil || id != "20260924T101530Z-001" || len(linked) != 1 {
		t.Fatalf("%s %v %v", id, err, linked)
	}
	if names := proposalFiles(t, dir); len(names) != 1 {
		t.Fatalf("files %v", names)
	}
}

// A temporary file that cannot be written ends the store; no name is
// claimed for a proposal that was never written.
func TestStoreProposalStopsWhenTheTemporaryFileFails(t *testing.T) {
	stubProposalLink(t, func(oldname, newname string) error {
		t.Errorf("link %q -> %q after the temporary file failed", oldname, newname)
		return errors.New("not supported")
	})
	_, err := storeProposal(filepath.Join(t.TempDir(), "missing", "\x00"), time.Now(), Proposal{})
	if err == nil {
		t.Fatal("no error")
	}
}

// A link that finds its name taken moves on to the next name; the exclusive
// create is the stand-in for a file system without links, not a second try
// at a name another writer holds.
func TestStoreProposalMovesOnWhenTheLinkFindsTheNameTaken(t *testing.T) {
	calls := 0
	stubProposalLink(t, func(oldname, newname string) error {
		calls++
		if calls == 1 {
			return &fs.PathError{Op: "link", Path: newname, Err: fs.ErrExist}
		}
		return os.Link(oldname, newname)
	})
	dir := t.TempDir()
	id, err := storeProposal(dir, time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC), Proposal{Op: "set", Key: "a"})
	if err != nil || id != "20260924T101530Z-002" {
		t.Fatalf("%s %v", id, err)
	}
	if names := proposalFiles(t, dir); strings.Join(names, " ") != "20260924T101530Z-002.json" {
		t.Fatalf("files %v", names)
	}
}
