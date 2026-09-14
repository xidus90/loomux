package junction

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func requireWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("junctions exist only on windows")
	}
}

// sameDir compares two paths by what they open, not by how they are spelled.
func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	left, err := os.Stat(a)
	if err != nil {
		return false
	}
	right, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(left, right)
}

// What a junction is, measured rather than assumed: Lstat must not follow it,
// Stat must, and the target must be readable back.
//
// Not `Mode()&os.ModeSymlink`, which the plan expected: measured on
// 2026-09-07 with go1.27.0, Lstat answers `Lrw-rw-rw-` by default but
// `?rw-rw-rw-` under `GODEBUG=winsymlink=1,winreadlinkvolume=1` -- the default
// for every `go` directive from 1.23 on, per `Changed: 23` in the toolchain's
// `internal/godebugs/table.go`. What holds under both, and what "did not
// follow" actually means, is that Lstat sees no directory where Stat sees one.
func TestACreatedJunctionLooksLikeOneAndReadsBack(t *testing.T) {
	requireWindows(t)
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(target, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")

	if err := Create(link, target); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := os.Stat(filepath.Join(link, "inner")); err != nil {
		t.Fatalf("the junction does not lead into the target: %v", err)
	}
	unfollowed, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if unfollowed.IsDir() {
		t.Fatalf("Lstat mode = %v, want something that is not the directory behind it", unfollowed.Mode())
	}
	followed, err := os.Stat(link)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !followed.IsDir() {
		t.Fatalf("Stat mode = %v, want the directory behind the link", followed.Mode())
	}

	got, err := Target(link)
	if err != nil {
		t.Fatalf("Target: %v", err)
	}
	// The NT form the kernel stores, unnormalised, is the contract the sweep
	// compares against; stripping the prefix is the caller's business.
	if !strings.HasPrefix(got, `\??\`) {
		t.Fatalf("Target = %q, want the stored NT form with a %q prefix", got, `\??\`)
	}
	if !sameDir(strings.TrimPrefix(got, `\??\`), target) {
		t.Fatalf("Target = %q, want %q", got, target)
	}
}

// A symbolic link is a reparse point too, and reporting it as a junction would
// hand the sweep something it must not remove.
func TestTargetOfASymbolicLinkIsEmptyAndNoError(t *testing.T) {
	requireWindows(t)
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("creating a symbolic link needs a privilege or Developer Mode:", err)
	}

	got, err := Target(link)
	if err != nil || got != "" {
		t.Fatalf("Target = %q, %v; want empty and nil", got, err)
	}
}

// The whole point of the destroy side: removing the link must not touch the
// 4.2 GB behind it.
func TestRemoveTakesTheLinkAndLeavesTheTarget(t *testing.T) {
	requireWindows(t)
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(keep, []byte("PRECIOUS"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := Create(link, target); err != nil {
		t.Fatal(err)
	}

	if err := Remove(link); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("the link survived: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("Remove reached through the junction: %v", err)
	}
}

// An ordinary directory is not ours to remove, and saying so is the guard the
// sweep leans on.
func TestTargetOfAnOrdinaryDirectoryIsEmptyAndNoError(t *testing.T) {
	root := t.TempDir()
	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Target(plain)
	if err != nil || got != "" {
		t.Fatalf("Target = %q, %v; want empty and nil", got, err)
	}
}

func TestTargetOfSomethingMissingIsEmptyAndNoError(t *testing.T) {
	got, err := Target(filepath.Join(t.TempDir(), "nowhere"))
	if err != nil || got != "" {
		t.Fatalf("Target = %q, %v; want empty and nil", got, err)
	}
}

// The third case, and the only one that is a fault: a path the filesystem
// cannot even be asked about is neither "not a link" nor "not there", and
// Remove must hand that on rather than deciding for itself.
func TestAPathThatCannotBeInspectedIsAFault(t *testing.T) {
	unaskable := filepath.Join(t.TempDir(), "bad\x00name")

	if _, err := Target(unaskable); err == nil {
		t.Fatal("Target reported success about a path it cannot inspect")
	}
	if err := Remove(unaskable); err == nil {
		t.Fatal("Remove reported success about a path it cannot inspect")
	}
}

// The refusal the Remove docstring promises: what is not a link is not ours to
// delete, and that is the whole distance between this and `rm -rf`.
func TestRemoveRefusesAnOrdinaryDirectory(t *testing.T) {
	root := t.TempDir()
	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := Remove(plain); err == nil {
		t.Fatal("Remove accepted an ordinary directory")
	}
	if _, err := os.Stat(plain); err != nil {
		t.Fatalf("the refused directory is gone: %v", err)
	}
}

// The other half of Remove: a link it may read and may not delete. Deletion
// code is where a swallowed error costs the most, so this branch is measured
// rather than reasoned about.
//
// Two denies, because one is not enough: deleting a child needs DELETE on the
// child *or* FILE_DELETE_CHILD on its parent, and with only the first taken
// away the removal still went through -- measured on 2026-09-08. `/L` on the
// link's own ACE, because icacls follows a junction otherwise and would edit
// the target's ACL instead. Target() itself keeps working under both denies:
// it opens with no access rights at all.
func TestRemoveReportsALinkItMayNotDelete(t *testing.T) {
	requireWindows(t)
	if _, err := exec.LookPath("icacls"); err != nil {
		t.Skip("icacls is not on PATH")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := Create(link, target); err != nil {
		t.Fatal(err)
	}
	user := os.Getenv("USERNAME")
	// The rights come back before t.TempDir's own cleanup runs: that one was
	// registered first and t.Cleanup is LIFO, so without this order the
	// fixture cannot delete the tree it made.
	icacls(t, link, "/deny", user+":(DE)", "/L")
	icacls(t, root, "/deny", user+":(DC)")
	t.Cleanup(func() {
		icacls(t, root, "/remove:d", user)
		icacls(t, link, "/remove:d", user, "/L")
	})

	if err := Remove(link); err == nil {
		t.Fatal("Remove reported success over a link it could not delete")
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("the link the failure was about is gone: %v", err)
	}
}

func icacls(t *testing.T, path string, argv ...string) {
	t.Helper()
	command := exec.Command("icacls", append([]string{path}, argv...)...)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("icacls %s %v: %v (%s)", path, argv, err, out)
	}
}

func TestCreateRefusesAnOccupiedPath(t *testing.T) {
	requireWindows(t)
	root := t.TempDir()
	target := filepath.Join(root, "target")
	occupied := filepath.Join(root, "occupied")
	for _, dir := range []string{target, occupied} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := Create(occupied, target); err == nil {
		t.Fatal("Create overwrote an existing directory")
	}
}

func TestCreateRefusesATargetThatIsNotADirectory(t *testing.T) {
	requireWindows(t)
	root := t.TempDir()
	notADir := filepath.Join(root, "file.txt")
	if err := os.WriteFile(notADir, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Create(filepath.Join(root, "link"), notADir); err == nil {
		t.Fatal("Create accepted a file as a target")
	}
}

func TestCreateRefusesATargetThatIsNotThere(t *testing.T) {
	requireWindows(t)
	root := t.TempDir()
	if err := Create(filepath.Join(root, "link"), filepath.Join(root, "nowhere")); err == nil {
		t.Fatal("Create accepted a target that is not there")
	}
}

// On any other platform the answer is a named error, never a symlink: a
// symlink is what cost 3.7 GB once, in the shape of `ln -s`.
func TestCreateIsUnsupportedElsewhere(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("this is the other platforms' half of the contract")
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "target"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Create(filepath.Join(root, "link"), filepath.Join(root, "target"))
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Create = %v, want ErrUnsupported", err)
	}
}
