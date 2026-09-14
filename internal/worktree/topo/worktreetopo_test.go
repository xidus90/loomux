package topo

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
)

func git(t *testing.T, dir string, argv ...string) {
	t.Helper()
	command := exec.Command("git", argv...)
	command.Dir = dir
	// The same clean environment the code under test uses: a leaked GIT_DIR
	// would build the fixture inside the repository whose hook is running.
	command.Env = gitenv.Environ()
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", argv, err, out)
	}
}

// A main checkout with one worktree in each of the two conventions this
// repository provides for -- see conventionDirs. Whether either is occupied
// right now is no part of what this fixture claims.
func fixture(t *testing.T) (main string, claudeWt string, plainWt string) {
	t.Helper()
	main = t.TempDir()
	git(t, main, "init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(main, "a.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, main, "add", "-A")
	git(t, main, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "first")

	claudeWt = filepath.Join(main, ".claude", "worktrees", "one")
	plainWt = filepath.Join(main, ".worktrees", "two")
	git(t, main, "worktree", "add", "-q", claudeWt, "-b", "one")
	git(t, main, "worktree", "add", "-q", plainWt, "-b", "two")
	return main, claudeWt, plainWt
}

func TestReadNamesTheMainCheckoutFirst(t *testing.T) {
	main, claudeWt, plainWt := fixture(t)

	for _, from := range []string{main, claudeWt, plainWt} {
		topology, err := Read(from)
		if err != nil {
			t.Fatalf("Read(%s): %v", from, err)
		}
		if !sameDir(topology.Main, main) {
			t.Fatalf("from %s: Main = %q, want %q", from, topology.Main, main)
		}
		if len(topology.Worktrees) != 3 {
			t.Fatalf("from %s: Worktrees = %q, want three entries", from, topology.Worktrees)
		}
	}
}

// The question the whole mechanism turns on, and the one CLAUDE.md forbids
// answering by comparing --git-dir against --git-common-dir as text.
func TestIsWorktreeSeparatesTheMainCheckoutFromTheRest(t *testing.T) {
	main, claudeWt, plainWt := fixture(t)
	topology, err := Read(main)
	if err != nil {
		t.Fatal(err)
	}

	if topology.IsWorktree(main) {
		t.Fatal("the main checkout was taken for a worktree")
	}
	for _, worktree := range []string{claudeWt, plainWt} {
		if !topology.IsWorktree(worktree) {
			t.Fatalf("%s was not taken for a worktree", worktree)
		}
	}
}

func TestOrphansFindsADirectoryGitNoLongerKnows(t *testing.T) {
	main, claudeWt, _ := fixture(t)
	// What `git worktree remove` leaves behind when a junction is in the way:
	// the registration is gone, the directory is not. Measured on 2026-09-07
	// with a junction inside a scratch worktree -- `remove --force` exited 0
	// without a word, dropped the porcelain entry, and left the directory with
	// the junction in it. Rebuilt here rather than junctioned, because that
	// state and not the way into it is what Orphans has to answer about.
	git(t, main, "worktree", "remove", "--force", claudeWt)
	if err := os.MkdirAll(claudeWt, 0o755); err != nil {
		t.Fatal(err)
	}

	topology, err := Read(main)
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := topology.Orphans()
	if err != nil {
		t.Fatalf("Orphans: %v", err)
	}
	if len(orphans) != 1 || !sameDir(orphans[0], claudeWt) {
		t.Fatalf("Orphans = %q, want just %q", orphans, claudeWt)
	}
}

func TestOrphansIsEmptyWhileEveryDirectoryIsRegistered(t *testing.T) {
	main, _, _ := fixture(t)
	topology, err := Read(main)
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := topology.Orphans()
	if err != nil || len(orphans) != 0 {
		t.Fatalf("Orphans = %q, %v; want empty and nil", orphans, err)
	}
}

// Neither convention directory has to exist, and their absence is not a fault.
func TestOrphansOfARepositoryWithoutWorktreeDirectoriesIsEmpty(t *testing.T) {
	main := t.TempDir()
	git(t, main, "init", "-q", "-b", "main")
	topology, err := Read(main)
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := topology.Orphans()
	if err != nil || len(orphans) != 0 {
		t.Fatalf("Orphans = %q, %v; want empty and nil", orphans, err)
	}
}

func TestADirectoryOutsideAnyRepositoryIsANamedError(t *testing.T) {
	if _, err := Read(t.TempDir()); !errors.Is(err, ErrNoRepository) {
		t.Fatalf("Read = %v, want ErrNoRepository", err)
	}
}

// An inherited GIT_DIR outranks the working directory, so a hook would
// otherwise be told about the repository it was started from.
//
// The pointers go up before the fixture is built, not after: that way the
// fixture's own `git init` and `git worktree add` run under a dirty
// environment too, and a git helper that stopped stripping would take the
// whole test down instead of only the Read below. It points at a t.TempDir(),
// so what a regression corrupts is a temp directory.
func TestAnInheritedGitDirDoesNotRedirectTheAnswer(t *testing.T) {
	elsewhere := t.TempDir()
	git(t, elsewhere, "init", "-q", "-b", "main")
	t.Setenv("GIT_DIR", filepath.Join(elsewhere, ".git"))
	t.Setenv("GIT_WORK_TREE", elsewhere)

	main, _, _ := fixture(t)

	topology, err := Read(main)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !sameDir(topology.Main, main) {
		t.Fatalf("Main = %q, want %q", topology.Main, main)
	}
}

func TestReadIgnoresTheOrderOfUnrelatedPorcelainFields(t *testing.T) {
	parsed := parse("worktree /a\nHEAD abc\nbranch refs/heads/main\n\nworktree /b\ndetached\n\n")
	// FromSlash because parse cleans, and on Windows filepath.Clean("/a")
	// answers "\a": rewriting to the platform separator is Clean's last step.
	// So the expectation follows the separator rather than the plan's literal
	// "/a", which is not what a cleaned path looks like here.
	want := []string{filepath.FromSlash("/a"), filepath.FromSlash("/b")}
	if !slices.Equal(parsed, want) {
		t.Fatalf("parse = %q, want %q", parsed, want)
	}
}

// A directory git holds nothing about is neither the main checkout nor a
// worktree. Every caller acts only on a true answer -- stated as the contract
// rather than as a list of subcommands, which would need a census and would go
// stale as they land.
func TestIsWorktreeSaysNoAboutADirectoryGitDoesNotHold(t *testing.T) {
	main, _, _ := fixture(t)
	topology, err := Read(main)
	if err != nil {
		t.Fatal(err)
	}
	if topology.IsWorktree(t.TempDir()) {
		t.Fatal("an unregistered directory was taken for a worktree")
	}
}

// Every path this package compares comes from git or from a caller, and either
// may name something that is not there. Then the answer is "not the same
// directory", never a stat error handed upwards.
func TestSameDirIsFalseWhenAPathDoesNotOpen(t *testing.T) {
	there := t.TempDir()
	gone := filepath.Join(there, "gone")

	for _, pair := range [][2]string{
		{"", there},
		{there, ""},
		{gone, there},
		{there, gone},
	} {
		if sameDir(pair[0], pair[1]) {
			t.Errorf("sameDir(%q, %q) = true, want false", pair[0], pair[1])
		}
	}
}

// A convention directory holds whatever someone put there. Only directories
// can be worktrees, so only directories can be orphaned ones.
func TestOrphansIgnoresAFileBesideTheWorktrees(t *testing.T) {
	main, _, plainWt := fixture(t)
	if err := os.WriteFile(filepath.Join(filepath.Dir(plainWt), "note.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	topology, err := Read(main)
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := topology.Orphans()
	if err != nil || len(orphans) != 0 {
		t.Fatalf("Orphans = %q, %v; want empty and nil", orphans, err)
	}
}

// A search space that is there but cannot be read is a fault, unlike one that
// is absent -- reporting it as "no orphans" would let the sweep that takes
// stale junctions down report a clean tree it never looked at.
//
// Windows only, because taking the right to list a directory away needs an
// ACL here. Measured on 2026-09-07: the cheaper trick of putting a *file* at
// `.worktrees` does not reach this path -- os.ReadDir then answers
// ERROR_PATH_NOT_FOUND, for which os.IsNotExist is true, so Orphans skips it.
func TestOrphansReportsAnUnreadableSearchSpace(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("taking away the right to list a directory is done by ACL here")
	}
	main := t.TempDir()
	git(t, main, "init", "-q", "-b", "main")
	unreadable := filepath.Join(main, ".worktrees")
	if err := os.Mkdir(unreadable, 0o755); err != nil {
		t.Fatal(err)
	}
	deny := exec.Command("icacls", unreadable, "/deny", os.Getenv("USERNAME")+":(RD)")
	if out, err := deny.CombinedOutput(); err != nil {
		t.Skipf("this account cannot deny itself the listing right: %v (%s)", err, out)
	}

	topology, err := Read(main)
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := topology.Orphans()
	if err == nil {
		t.Fatalf("Orphans = %q, nil; want a reported failure", orphans)
	}
	if errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Orphans = %v, which Orphans itself would have skipped", err)
	}
	if !strings.Contains(err.Error(), unreadable) {
		t.Fatalf("Orphans = %v, want the unreadable path named", err)
	}
}

// The guard behind worktrees[0]. No git answers this way -- `git init --bare`
// still prints a `worktree` line -- so it takes a stand-in on PATH to reach,
// and reaching it is the point: a silent empty answer must not become an
// index panic.
func TestReadRejectsAnAnswerThatNamesNoWorkingTree(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the stand-in is a .cmd file")
	}
	standIn := t.TempDir()
	if err := os.WriteFile(filepath.Join(standIn, "git.cmd"), []byte("@exit /b 0\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// exec.Command resolves the name against this process's PATH, so the
	// stand-in has to go in front of the real git there and not in the
	// environment handed to the child.
	t.Setenv("PATH", standIn+string(os.PathListSeparator)+os.Getenv("PATH"))

	if _, err := Read(standIn); !errors.Is(err, ErrNoRepository) {
		t.Fatalf("Read = %v, want ErrNoRepository", err)
	}
}
