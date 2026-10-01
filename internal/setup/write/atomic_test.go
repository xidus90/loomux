package write

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
)

func TestAnExistingFileIsSkippedNotOverwritten(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "keep.txt")
	if err := os.WriteFile(target, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := Prepare(root, map[string]string{"keep.txt": "theirs"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(plan.Create) != 0 || len(plan.Skip) != 1 {
		t.Fatalf("plan = %+v, want one skip and no create", plan)
	}
	if _, err := Commit(root, plan); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	body, _ := os.ReadFile(target)
	if string(body) != "mine" {
		t.Fatalf("file was overwritten: %q", body)
	}
}

func TestDirectoriesAreCreatedForNewFiles(t *testing.T) {
	root := t.TempDir()
	plan, err := Prepare(root, map[string]string{".loomux/policy.toml": "x"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if _, err := Commit(root, plan); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".loomux", "policy.toml")); err != nil {
		t.Fatalf("file missing: %v", err)
	}
}

func TestSkipIsSortedRegardlessOfMapOrder(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{}
	for _, name := range []string{"e.txt", "b.txt", "d.txt", "a.txt", "c.txt", "f.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("mine"), 0o644); err != nil {
			t.Fatal(err)
		}
		files[name] = "theirs"
	}
	plan, err := Prepare(root, files)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	want := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt", "f.txt"}
	if !slices.Equal(plan.Skip, want) {
		t.Fatalf("Skip = %v, want %v", plan.Skip, want)
	}
}

func TestHostileNamesAreRefused(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"",
		".",
		"..",
		"../escape.txt",
		"a/../../escape.txt",
		"/etc/passwd",
		"C:/Windows/system32/x",
		`..\escape.txt`,
		"a//b",
		"a/",
	} {
		if _, err := Prepare(root, map[string]string{name: "x"}); err == nil {
			t.Errorf("Prepare accepted %q", name)
		}
	}
}

func TestAFileAppearingAfterPrepareIsNotOverwritten(t *testing.T) {
	root := t.TempDir()
	plan, err := Prepare(root, map[string]string{"late.txt": "theirs"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	target := filepath.Join(root, "late.txt")
	if err := os.WriteFile(target, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit(root, plan); err == nil {
		t.Fatal("Commit overwrote a file that appeared after Prepare")
	}
	body, _ := os.ReadFile(target)
	if string(body) != "mine" {
		t.Fatalf("file was overwritten: %q", body)
	}
}

func TestAnUnreadableNameIsReported(t *testing.T) {
	// A NUL byte in the root never reaches the OS: Go's own string conversion
	// rejects it, on every platform, with something that is not "not found".
	if _, err := Prepare("bad\x00root", map[string]string{"a.txt": "x"}); err == nil {
		t.Fatal("Prepare accepted a root it cannot stat")
	}
}

func TestADirectoryThatCannotBeCreatedIsReported(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a"), []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Create: map[string]string{"a/b.txt": "x"}}
	if _, err := Commit(root, plan); err == nil {
		t.Fatal("Commit created a directory where a file stands")
	}
}

func TestCommitRefusesAHostileNameInAHandBuiltPlan(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	plan := Plan{Create: map[string]string{"../escape.txt": "x"}}
	if _, err := Commit(root, plan); err == nil {
		t.Fatal("Commit accepted a name that leaves the project")
	}
	if _, err := os.Stat(filepath.Join(root, "..", "escape.txt")); !os.IsNotExist(err) {
		t.Fatalf("Commit wrote outside root: %v", err)
	}
}

func TestTheRaceIsExplainedNotJustReported(t *testing.T) {
	root := t.TempDir()
	plan, err := Prepare(root, map[string]string{"late.txt": "theirs"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "late.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Commit(root, plan)
	if err == nil {
		t.Fatal("Commit did not report the race")
	}
	if !strings.Contains(err.Error(), "the plan had it as new, but it exists") {
		t.Fatalf("error does not explain itself: %v", err)
	}
}

func TestNamesGivesCallersTheCommitOrder(t *testing.T) {
	plan := Plan{Create: map[string]string{"e": "", "b": "", "d": "", "a": "", "c": "", "f": ""}}
	want := []string{"a", "b", "c", "d", "e", "f"}
	if !slices.Equal(plan.Names(), want) {
		t.Fatalf("Names() = %v, want %v", plan.Names(), want)
	}
}

func TestAFailureThatIsNotARaceReadsAsAFailure(t *testing.T) {
	// The distinction is what the package owes its callers, so it is a named
	// function and gets asked on its own rather than through a staged disk.
	err := commitFailure("a.txt", errors.New("disk on fire"))
	if !strings.Contains(err.Error(), "writing a.txt") || strings.Contains(err.Error(), "the plan had it as new") {
		t.Fatalf("wrong wording: %v", err)
	}
}

func TestADanglingSymlinkIsSkippedNotWritten(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "dangling.txt")
	if err := os.Symlink(filepath.Join(root, "nowhere.txt"), link); err != nil {
		skipIfLinksAreNotAllowed(t, err)
	}
	plan, err := Prepare(root, map[string]string{"dangling.txt": "theirs"})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if len(plan.Create) != 0 || !slices.Equal(plan.Skip, []string{"dangling.txt"}) {
		t.Fatalf("plan = %+v, want the link skipped", plan)
	}
	if _, err := Commit(root, plan); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("the link is gone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "nowhere.txt")); !os.IsNotExist(err) {
		t.Fatalf("Commit wrote through the link: %v", err)
	}
}

// A Commit that stops halfway has still written: the names of what landed are
// the caller's only way to report the truth about it.
func TestCommitNamesWhatItWroteBeforeItFailed(t *testing.T) {
	root := t.TempDir()
	plan, err := Prepare(root, map[string]string{"a.txt": "1", "b.txt": "2", "c.txt": "3"})
	if err != nil {
		t.Fatal(err)
	}
	// Planned as new, and there by the time Commit arrives -- the race the
	// exclusive create is for, arranged on purpose in the middle of the order.
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("theirs"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err := Commit(root, plan)
	if err == nil {
		t.Fatal("Commit overwrote a file that appeared after Prepare")
	}
	if len(written) != 1 || written[0] != "a.txt" {
		t.Fatalf("written = %v, want [a.txt]", written)
	}
}

// Nothing written is the other half, and the one a caller may report as
// "nothing happened".
func TestCommitNamesNothingWhenTheFirstFileFails(t *testing.T) {
	root := t.TempDir()
	written, err := Commit(root, Plan{Create: map[string]string{"../escape.txt": "x"}})
	if err == nil {
		t.Fatal("Commit accepted a name that leaves the project")
	}
	if len(written) != 0 {
		t.Fatalf("written = %v, want none", written)
	}
}

// privilegeNotHeld is ERROR_PRIVILEGE_NOT_HELD, what Windows answers when the
// account may not create a symlink -- which is every account without developer
// mode or an elevated shell. Go maps ERROR_ACCESS_DENIED to os.ErrPermission
// and this one to nothing, so it has to be named. The number is harmless
// elsewhere: no other platform returns it.
const privilegeNotHeld = syscall.Errno(1314)

// skipIfLinksAreNotAllowed skips only where the platform says no, and fails on
// anything else. Creating a link is a privilege on Windows and unsupported on
// some filesystems; a plain failure on Linux is a broken test, and skipping on
// every error would hide it.
func skipIfLinksAreNotAllowed(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, errors.ErrUnsupported) || errors.Is(err, os.ErrPermission) ||
		errors.Is(err, privilegeNotHeld) {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	t.Fatalf("creating a symlink failed for a reason this platform allows: %v", err)
}

// A name that is above reproach can still land outside the project: MkdirAll
// and OpenFile follow a symlinked directory without a word. Nothing is
// overwritten, but "inside root" was the promise.
func TestCommitRefusesToWriteThroughALinkedDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	linkDir(t, outside, filepath.Join(root, "elsewhere"))
	plan, err := Prepare(root, map[string]string{"elsewhere/new.txt": "ours"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Commit(root, plan); err == nil {
		t.Fatal("Commit wrote through a linked directory")
	}
	if _, err := os.Stat(filepath.Join(outside, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("a new file landed outside the project: %v", err)
	}
}

// A directory above the name that cannot be looked at is not a free path.
func TestAnUnreadableParentIsReported(t *testing.T) {
	err := CheckParents("bad"+string(rune(0))+"root", "a/b.txt")
	if err == nil {
		t.Fatal("CheckParents accepted a root it cannot stat")
	}
	if !strings.Contains(err.Error(), "above a/b.txt") {
		t.Fatalf("the name was not reported: %v", err)
	}
}

// linkDir points one name at another directory, by whichever of the two
// mechanisms this machine allows. A symlink needs a privilege on Windows that
// an ordinary account does not have; a directory junction needs none, and Go
// reports it as ModeIrregular. Both are what CheckParents refuses, so either
// one tests it.
func linkDir(t *testing.T, target, link string) {
	t.Helper()
	err := os.Symlink(target, link)
	if err == nil {
		return
	}
	if runtime.GOOS != "windows" {
		skipIfLinksAreNotAllowed(t, err)
		return
	}
	if out, jerr := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); jerr != nil {
		t.Skipf("neither a symlink (%v) nor a junction (%v: %s) can be made here", err, jerr, out)
	}
}

func TestWriteNewExecutablePermissions(t *testing.T) {
	root := t.TempDir()
	plan, err := Prepare(root, map[string]string{
		".githooks/pre-commit": "#!/bin/sh\nexit 0\n",
		"script.sh":            "#!/bin/sh\nexit 0\n",
	})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	written, err := Commit(root, plan)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("written = %v, want 2 files", written)
	}
}

// Windows keeps no execute bits, and the test chain runs there, so the mode
// is judged where it is chosen rather than read back from the disk.
func TestAScriptIsWrittenExecutableWhereverItLies(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       fs.FileMode
	}{
		{".githooks/pre-commit", "#!/bin/sh\nexit 0\n", 0o755},
		{"tools/hooks/pre-push", "#!/bin/sh\nexit 0\n", 0o755},
		{".git/hooks/commit-msg", "#!/bin/sh\nexit 0\n", 0o755},
		{"script.sh", "echo hi\n", 0o755},
		{"AGENTS.md", "# loomux\n", 0o644},
		{".mcp.json", "{}\n", 0o644},
	} {
		if got := Mode(tc.name, tc.body); got != tc.want {
			t.Errorf("Mode(%s) = %o, want %o", tc.name, got, tc.want)
		}
	}
}
