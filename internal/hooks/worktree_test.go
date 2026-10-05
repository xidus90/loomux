package hooks

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/testlock"
)

func requireWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("junctions exist only on windows")
	}
}

func git(t *testing.T, dir string, argv ...string) {
	t.Helper()
	command := exec.Command("git", argv...)
	command.Dir = dir
	// The same clean environment the code under test uses. Without it a
	// leaked GIT_DIR builds the fixture inside the repository whose hook is
	// running: on 2026-09-07 fixtures of this shape committed onto this
	// repository's own master that way.
	command.Env = gitenv.Environ()
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", argv, err, out)
	}
}

// A main checkout with one committed file and one registered worktree, in the
// `.worktrees` convention worktreetopo scans.
//
// main is resolved to its long spelling, so every path derived from it is the
// one git reports. Where TEMP is an 8.3 short path -- C:\Users\RUNNER~1 on a
// GitHub runner -- t.TempDir() hands out the short form, git answers with the
// long one, and every comparison by text between the two fails.
// filepath.EvalSymlinks expands 8.3 names, measured on 2026-09-18.
func worktreeFixture(t *testing.T) (main string, worktree string) {
	t.Helper()
	main, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git(t, main, "init", "-q", "-b", "main")
	writeFile(t, filepath.Join(main, "a.txt"), "x\n")
	git(t, main, "add", "-A")
	git(t, main, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "first")

	worktree = filepath.Join(main, ".worktrees", "one")
	git(t, main, "worktree", "add", "-q", worktree, "-b", "one")
	return main, worktree
}

// longestDirPath is the length of the longest path this machine makes a
// directory at, measured in a temp dir of its own: a chain of components of
// 200 characters as deep as it goes, then the longest last component that
// still fits. The temp dir is taken in its long spelling, the one every other
// path in these tests is built from.
func longestDirPath(t *testing.T) int {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for {
		next := filepath.Join(dir, strings.Repeat("p", 200))
		if os.Mkdir(next, 0o755) != nil {
			break
		}
		dir = next
	}
	// Mkdir of dir/<n characters> succeeds for n below some bound in 1..200;
	// lo is known to succeed (0 means dir itself), hi known to fail.
	lo, hi := 0, 200
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		probe := filepath.Join(dir, strings.Repeat("q", mid))
		if os.Mkdir(probe, 0o755) == nil {
			_ = os.Remove(probe)
			lo = mid
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return len(dir)
	}
	return len(dir) + 1 + lo
}

func writeConfig(t *testing.T, main string, body string) {
	t.Helper()
	mkdirAll(t, filepath.Join(main, ".loomux"))
	writeFile(t, filepath.Join(main, ".loomux", "config.toml"), body)
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// unregister takes git's entry away and leaves the directory standing -- which
// is what git itself does when a junction is in the way. Measured on
// 2026-09-07: with a junction inside the worktree, `git worktree remove
// --force` exits 0, the registration is gone, and the directory with its
// junction is still there. Without one the same call deletes the directory,
// untracked files included, so the MkdirAll below puts it back.
func unregister(t *testing.T, main string, worktree string) {
	t.Helper()
	git(t, main, "worktree", "remove", "--force", worktree)
	mkdirAll(t, worktree)
}

// mklink makes a junction the way a hand at a prompt does. Its stored target
// carries no trailing separator, unlike the one junction.Create writes, and
// the links this repository's own worktrees already hold came from here.
func mklink(t *testing.T, link string, target string) {
	t.Helper()
	command := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("mklink /J %s %s: %v (%s)", link, target, err, out)
	}
}

// The case the whole mechanism is for: a fresh worktree without .tools and
// without .loomux/vendor, both configured, both put in place.
func TestWorktreeLinkPutsTheConfiguredDirectoriesInPlace(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\", \".loomux/vendor\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools", "godot"))
	mkdirAll(t, filepath.Join(main, ".loomux", "vendor", "ultraloom"))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, worktree); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}

	for _, relative := range []string{".tools/godot", ".loomux/vendor/ultraloom"} {
		if _, err := os.Stat(filepath.Join(worktree, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("%s is not reachable from the worktree: %v", relative, err)
		}
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// A directory the worktree already has of its own is not touched: it may be a
// build output that belongs to this tree.
func TestWorktreeLinkLeavesARealDirectoryAlone(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools"))
	own := filepath.Join(worktree, ".tools", "mine.txt")
	mkdirAll(t, filepath.Dir(own))
	writeFile(t, own, "mine")

	assertSilentOK(t, worktree)
	if _, err := os.Stat(own); err != nil {
		t.Fatalf("the worktree's own directory was replaced: %v", err)
	}
}

// Running twice must be the same as running once: this fires at every session
// start.
func TestWorktreeLinkIsIdempotent(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools", "godot"))

	for round := 1; round <= 2; round++ {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := WorktreeLink(stdout, stderr, worktree); code != ExitOK {
			t.Fatalf("round %d: exit = %d (stderr: %s)", round, code, stderr)
		}
	}
	if _, err := os.Stat(filepath.Join(worktree, ".tools", "godot")); err != nil {
		t.Fatalf("the junction did not survive the second round: %v", err)
	}
}

// A configured path that is not in the main checkout either is nothing to
// mirror -- and nothing to complain about.
func TestWorktreeLinkSkipsAPathTheMainCheckoutDoesNotHave(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")

	assertSilentOK(t, worktree)
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); !os.IsNotExist(err) {
		t.Fatalf("something was created for a path that does not exist: %v", err)
	}
}

// A configured path the main checkout holds as a file is not a directory to
// mirror either.
func TestWorktreeLinkSkipsAPathThatIsNotADirectory(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	writeFile(t, filepath.Join(main, ".tools"), "not a directory")

	assertSilentOK(t, worktree)
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); !os.IsNotExist(err) {
		t.Fatalf("something was created for a file: %v", err)
	}
}

// In the main checkout there is nothing to mirror: the directories are there.
func TestWorktreeLinkDoesNothingInTheMainCheckout(t *testing.T) {
	main, _ := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools"))

	assertSilentOK(t, main)
}

// The three no-op cases, exit 0 and silent. This hook fires in every project
// on the machine.
func TestTheNoOpCasesAreSilentAndSuccessful(t *testing.T) {
	t.Run("no repository", func(t *testing.T) {
		assertSilentOK(t, t.TempDir())
	})
	t.Run("no config", func(t *testing.T) {
		_, worktree := worktreeFixture(t)
		assertSilentOK(t, worktree)
	})
	t.Run("config without the section", func(t *testing.T) {
		main, worktree := worktreeFixture(t)
		writeConfig(t, main, "[verify]\nlint = \"ruff check .\"\n")
		assertSilentOK(t, worktree)
	})
}

func assertSilentOK(t *testing.T, root string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, root); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// Damage is the one case that is a failure: read as "nothing to mirror", it
// would switch the mechanism off without a word.
func TestBrokenConfigIsAFailure(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree\n")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if stderr.Len() == 0 {
		t.Fatal("a failure said nothing")
	}
}

// A configured path that cannot be made is the one thing this reports, because
// the directory behind it may be the pinned runtime every other hook needs.
//
// The fault is the length of the path and not a denied right, which an
// elevated token creates through. How long a path Windows still makes a
// directory at is a property of the machine: 32762 characters here, measured
// on 2026-09-18, and less on the GitHub runner, where a fixture sized for
// 32762 failed to make its own target. longestDirPath measures it on the
// machine the test runs on. The worktree lies 15 characters deeper than the
// main checkout (`.worktrees/one`), so a configured path whose target in the
// main checkout ends just short of that limit has a parent in the worktree
// that ends past it. parentsPlainOrAbsent lets it through: `.tools` is absent
// in the worktree, and nothing below an absent component is looked at. The
// MkdirAll of the parent is what fails.
//
// A *file* at `.tools` used to be this fixture and no longer reaches MkdirAll:
// parentsPlainOrAbsent refuses it one step earlier, and
// TestWorktreeLinkDoesNotCreateThroughAnIntermediateLink pins that.
func TestWorktreeLinkReportsAPathItCannotMakeRoomFor(t *testing.T) {
	requireWindows(t)
	limit := longestDirPath(t)
	main, worktree := worktreeFixture(t)
	// The target ends 4 characters short of the limit; its parent in the
	// worktree then ends 15 - len("/deep") - 4 = 6 past it.
	targetLength := limit - 4
	relative := ".tools"
	for rest := targetLength - len(main) - len("/.tools/deep"); rest > 0; {
		pad := min(rest-1, 200)
		if rest-pad-1 == 1 {
			// A remainder of one would be an empty component.
			pad--
		}
		relative += "/" + strings.Repeat("d", pad)
		rest -= pad + 1
	}
	relative += "/deep"
	target := filepath.Join(main, filepath.FromSlash(relative))
	parent := filepath.Dir(filepath.Join(worktree, filepath.FromSlash(relative)))
	if len(target) > limit || len(parent) <= limit {
		t.Fatalf("fixture precondition: want the target in the main checkout at most %d "+
			"characters and its parent in the worktree more than that, got %d and %d "+
			"(main checkout %s, %d characters)",
			limit, len(target), len(parent), main, len(main))
	}
	writeConfig(t, main, "[worktree]\nmirror = ['"+relative+"']\n")
	mkdirAll(t, filepath.Join(main, filepath.FromSlash(relative)))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	// junction.Create words its own Mkdir failure as "making room for the
	// junction", and with the MkdirAll error swallowed that is what fails
	// next; the second condition keeps it from passing for this branch.
	if msg := stderr.String(); !strings.Contains(msg, "making room for") ||
		strings.Contains(msg, "making room for the junction") {
		t.Fatalf("stderr = %q, want the MkdirAll branch reported", stderr)
	}
}

// The create side of the containment check: a configured path whose parent is
// a link is not inside the worktree at all, and creating it there writes into
// whatever that link points at.
//
// Windows follows every component of a path but the last, so with a junction
// at `<worktree>/.tools` the Lstat of `<worktree>/.tools/godot` asks about
// `<elsewhere>/godot` -- absent, so before the guard link took that for "ours
// to fill", and the MkdirAll and junction.Create that followed landed under
// the main checkout, at a path no sweep of ours ever looks at. It creates and
// never deletes.
//
// The file case is the same guard from the other side: it, too, is a component
// that is not a plain directory, and refusing it keeps link from writing
// anywhere the spelling did not promise.
func TestWorktreeLinkDoesNotCreateThroughAnIntermediateLink(t *testing.T) {
	t.Run("a junction at an intermediate component", func(t *testing.T) {
		requireWindows(t)
		main, worktree := worktreeFixture(t)
		writeConfig(t, main, "[worktree]\nmirror = [\".tools/godot\"]\n")
		mkdirAll(t, filepath.Join(main, ".tools", "godot"))
		elsewhere := filepath.Join(main, "elsewhere")
		mkdirAll(t, elsewhere)
		mklink(t, filepath.Join(worktree, ".tools"), elsewhere)

		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		code := WorktreeLink(stdout, stderr, worktree)
		escaped := filepath.Join(elsewhere, "godot")
		if _, err := os.Lstat(escaped); !os.IsNotExist(err) {
			t.Fatalf("%s was created outside the worktree: %v", escaped, err)
		}
		if code != ExitInternal {
			t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q, want silence", stdout)
		}
		if !strings.Contains(stderr.String(), "worktree link") {
			t.Fatalf("stderr = %q, want the subcommand's name in it", stderr)
		}
	})

	t.Run("a file at an intermediate component", func(t *testing.T) {
		main, worktree := worktreeFixture(t)
		writeConfig(t, main, "[worktree]\nmirror = [\".tools/godot\"]\n")
		mkdirAll(t, filepath.Join(main, ".tools", "godot"))
		writeFile(t, filepath.Join(worktree, ".tools"), "not a directory")

		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := WorktreeLink(stdout, stderr, worktree); code != ExitInternal {
			t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q, want silence", stdout)
		}
	})
}

// The other side of the same check, and the one that has to keep working: a
// configured path whose parents are merely *missing* is exactly what link is
// for. Two levels, so an off-by-one that stopped one component short would
// show here -- twice on this branch a containment check has silently disabled
// the feature in this direction, and both times the second test caught it.
func TestWorktreeLinkCreatesAPathWhoseParentsAreMissing(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\"space/.tools/godot\"]\n")
	mkdirAll(t, filepath.Join(main, "space", ".tools", "godot", "bin"))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, worktree); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	linked := filepath.Join(worktree, "space", ".tools", "godot")
	if _, err := os.Stat(filepath.Join(linked, "bin")); err != nil {
		t.Fatalf("the mirrored path is not reachable from the worktree: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// The sweep: a directory git no longer knows, with our junction still in it.
func TestWorktreeLinkSweepsAnOrphanedJunction(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools", "godot"))
	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, worktree); code != ExitOK {
		t.Fatal("the fixture's own link was not created")
	}
	unregister(t, main, worktree)
	// The junction is what stopped git from deleting the directory, so it has
	// to be standing before the sweep is asked about it.
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); err != nil {
		t.Fatalf("git took the junction with it, so this tests nothing: %v", err)
	}

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, main); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); !os.IsNotExist(err) {
		t.Fatalf("the orphaned junction survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(main, ".tools", "godot")); err != nil {
		t.Fatalf("the sweep reached through the junction: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// A hand-made junction is swept too, and it is the spelling that matters: its
// stored target has no trailing separator, so a comparison made as text
// against an absolute path would miss exactly the links this repository's own
// worktrees already carry.
func TestTheSweepTakesAHandMadeJunctionAsWell(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools", "godot"))
	mklink(t, filepath.Join(worktree, ".tools"), filepath.Join(main, ".tools"))
	unregister(t, main, worktree)

	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, main); code != ExitOK {
		t.Fatal("the sweep failed")
	}
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); !os.IsNotExist(err) {
		t.Fatalf("the hand-made junction survived: %v", err)
	}
}

// A real directory in an orphaned worktree is somebody's data, not our link.
//
// Written after `unregister`, not before: measured on 2026-09-07, `git
// worktree remove --force` deletes the whole directory including untracked
// files when no reparse point blocks it, so a file put there first would be
// gone before the sweep ever ran.
func TestTheSweepLeavesARealDirectoryInAnOrphanAlone(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools"))
	unregister(t, main, worktree)
	keep := filepath.Join(worktree, ".tools", "mine.txt")
	mkdirAll(t, filepath.Dir(keep))
	writeFile(t, keep, "mine")

	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, main); code != ExitOK {
		t.Fatal("the sweep failed")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("the sweep removed a real directory: %v", err)
	}
}

// A junction pointing somewhere else is somebody's own arrangement. Being a
// reparse point at a configured path is not enough to make it ours.
func TestTheSweepLeavesAJunctionPointingElsewhereAlone(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools"))
	elsewhere := t.TempDir()
	unregister(t, main, worktree)
	mklink(t, filepath.Join(worktree, ".tools"), elsewhere)

	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, main); code != ExitOK {
		t.Fatal("the sweep failed")
	}
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); err != nil {
		t.Fatalf("a junction into another directory was removed: %v", err)
	}
}

// A junction at a path nobody configured is not ours either, wherever it
// points. The configuration is what says which paths this mechanism owns.
func TestTheSweepLeavesAJunctionAtAnUnconfiguredPathAlone(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools"))
	mkdirAll(t, filepath.Join(main, "somewhere"))
	unregister(t, main, worktree)
	mklink(t, filepath.Join(worktree, "somewhere"), filepath.Join(main, "somewhere"))

	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, main); code != ExitOK {
		t.Fatal("the sweep failed")
	}
	if _, err := os.Lstat(filepath.Join(worktree, "somewhere")); err != nil {
		t.Fatalf("a junction at an unconfigured path was removed: %v", err)
	}
}

// The other side of the containment check: a nested configured path whose
// intermediate directories are real is still swept. Without this, a check
// that skipped one component too many would stop cleaning
// `.loomux/vendor` and nothing would say so.
func TestTheSweepReachesANestedPathThroughRealDirectories(t *testing.T) {
	requireWindows(t)
	main, _ := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".loomux/vendor\"]\n")
	mkdirAll(t, filepath.Join(main, ".loomux", "vendor", "ultraloom"))

	orphan := filepath.Join(main, ".worktrees", "gone")
	mkdirAll(t, filepath.Join(orphan, ".loomux"))
	link := filepath.Join(orphan, ".loomux", "vendor")
	mklink(t, link, filepath.Join(main, ".loomux", "vendor"))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, main); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("the nested orphaned junction survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(main, ".loomux", "vendor", "ultraloom")); err != nil {
		t.Fatalf("the sweep reached through the junction: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// A candidate whose path runs through a link is not in the orphan at all, and
// the one thing behind such a path may be the main checkout's own junction.
//
// Windows opens the final component of a path with
// FILE_FLAG_OPEN_REPARSE_POINT and follows every earlier one, so
// `<orphan>/.loomux/vendor` with a junction at `<orphan>/.loomux` reads
// the reparse point of `<main>/.loomux/vendor`. Before the containment
// check that junction satisfied all three conditions and was removed -- the
// mirrored directory the main checkout needs, gone silently, and `link`
// cannot put it back because IsWorktree says false about the main checkout.
func TestTheSweepDoesNotReachThroughAnIntermediateLink(t *testing.T) {
	requireWindows(t)
	main, _ := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".loomux/vendor\"]\n")

	// The main checkout's own mirror path is a junction here as well, which is
	// what makes it look like a candidate once an earlier component leads to
	// it.
	runtimeDir := filepath.Join(main, "runtime")
	mkdirAll(t, runtimeDir)
	mainVendor := filepath.Join(main, ".loomux", "vendor")
	mklink(t, mainVendor, runtimeDir)

	orphan := filepath.Join(main, ".worktrees", "gone")
	mkdirAll(t, orphan)
	mklink(t, filepath.Join(orphan, ".loomux"), filepath.Join(main, ".loomux"))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, main); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Lstat(mainVendor); err != nil {
		t.Fatalf("the main checkout's own junction was removed: %v", err)
	}
	if _, err := os.Stat(runtimeDir); err != nil {
		t.Fatalf("what the main checkout's junction pointed at is gone: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// A candidate the sweep cannot even look at is a failure and not a skip: the
// sweep must not go quiet about a path it could not decide. `?` is legal in a
// configured path -- filepath.IsLocal accepts it, measured on 2026-09-07 --
// and illegal in a Windows filename, so Lstat answers with something other
// than IsNotExist.
func TestTheSweepReportsAPathItCannotInspect(t *testing.T) {
	requireWindows(t)
	main, _ := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['bad?name']\n")
	mkdirAll(t, filepath.Join(main, ".worktrees", "gone"))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, main); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if stderr.Len() == 0 {
		t.Fatal("a failure said nothing")
	}
}

// And a convention directory the sweep cannot read is a failure for the same
// reason.
func TestTheSweepReportsADirectoryItCannotScan(t *testing.T) {
	requireWindows(t)
	main, _ := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools"))

	// A handle without any share mode on the directory the orphan scan reads,
	// so os.ReadDir of it fails with a sharing violation. Not a denied right:
	// an elevated token lists a directory through a deny ACE.
	testlock.LockDir(t, filepath.Join(main, ".worktrees"))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, main); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if stderr.Len() == 0 {
		t.Fatal("a failure said nothing")
	}
}

// The loud case itself: a junction that was needed and did not come about.
// Nothing usable stands at the path, there is room to be made and the junction
// still cannot be created -- which is what the missing directory being the
// pinned runtime would look like.
func TestWorktreeLinkReportsAJunctionItCouldNotMake(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = [\".tools\"]\n")
	mkdirAll(t, filepath.Join(main, ".tools"))

	// A directory pending deletion at the path, and not a denied right, which
	// an elevated token creates through. Its Lstat fails, and link skips only
	// on an Lstat that succeeds; a one-component path has no parent for
	// parentsPlainOrAbsent to refuse; and the os.Mkdir inside junction.Create
	// fails with access denied, because the name is still taken.
	pending := filepath.Join(worktree, ".tools")
	mkdirAll(t, pending)
	testlock.DeletePending(t, pending)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeLink(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), ".tools") {
		t.Fatalf("stderr = %q, want the configured path named in it", stderr)
	}
}

// leadsInto has to answer about a stored target, and a stored target comes in
// more than one spelling: the trailing separator depends on who made the
// junction, and the prefix on what wrote it. Measured on 2026-09-07:
// junction.Create stores `\??\C:\dir\`, `mklink /J` stores `\??\C:\dir`, and
// both resolve.
func TestLeadsIntoReadsEverySpellingOfAStoredTarget(t *testing.T) {
	main := t.TempDir()
	inside := filepath.Join(main, "vendor")
	mkdirAll(t, inside)

	for _, stored := range []string{
		`\??\` + inside + `\`,
		`\??\` + inside,
		`\\?\` + inside,
		inside,
		filepath.Join(inside, "gone"),
	} {
		if !leadsInto(stored, main) {
			t.Fatalf("leadsInto(%q, %q) = false, want true", stored, main)
		}
	}

	for _, stored := range []string{t.TempDir(), inside} {
		outside := t.TempDir()
		if leadsInto(stored, outside) {
			t.Fatalf("leadsInto(%q, %q) = true, want false", stored, outside)
		}
	}
	if leadsInto(inside, filepath.Join(main, "not-there")) {
		t.Fatal("a main checkout that is not there was accepted")
	}
}

// writeSessionState puts a session's file where the Python hooks put theirs.
// Only the place matters -- nothing on the Go side reads the body -- and the
// directory comes from the constant rather than from a second literal, which
// is the drift that constant exists to prevent.
func writeSessionState(t *testing.T, worktree string, id string) string {
	t.Helper()
	dir := filepath.Join(worktree, filepath.FromSlash(sessions.StateDir))
	mkdirAll(t, dir)
	path := filepath.Join(dir, id+".json")
	writeFile(t, path, `{"blocks":0,"snapshots":{}}`)
	return path
}

// unlinkAs runs the subcommand the way the SessionEnd hook does: one payload on
// stdin, and nothing else to go on.
func unlinkAs(t *testing.T, root string, id string) (stdout, stderr *bytes.Buffer, code int) {
	t.Helper()
	stdout, stderr = &bytes.Buffer{}, &bytes.Buffer{}
	payload := bytes.NewBufferString(`{"session_id":"` + id + `"}`)
	return stdout, stderr, WorktreeUnlink(stdout, stderr, payload, root)
}

// linkedFixture is the state a session ends in: the configured directories put
// in place by the same code that will take them out again.
func linkedFixture(t *testing.T, mirror string) (main, worktree string) {
	t.Helper()
	requireWindows(t)
	main, worktree = worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['"+mirror+"']\n")
	mkdirAll(t, filepath.Join(main, filepath.FromSlash(mirror), "godot"))
	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, worktree); code != ExitOK {
		t.Fatal("the fixture's own link was not created")
	}
	return main, worktree
}

// The reason unlink counts first: CLAUDE.md documents sessions that share a
// checkout, and .tools must not vanish under a running Godot editor.
func TestWorktreeUnlinkKeepsTheJunctionWhileAnotherSessionHoldsIt(t *testing.T) {
	_, worktree := linkedFixture(t, ".tools")
	writeSessionState(t, worktree, "other")

	stdout, stderr, code := unlinkAs(t, worktree, "mine")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".tools", "godot")); err != nil {
		t.Fatalf("the junction was removed while another session held it: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// A file from a session that has long since ended must not hold the junction
// for ever: nothing deletes those files, so the age is all there is to go on.
func TestWorktreeUnlinkIgnoresAStaleSessionFile(t *testing.T) {
	_, worktree := linkedFixture(t, ".tools")
	ancient := writeSessionState(t, worktree, "ancient")
	// Derived from the constant and not a number of its own: the point is
	// "past the cutoff, whatever the cutoff is", and a literal here would go
	// quietly wrong the next time the cutoff moves.
	when := time.Now().Add(-2 * sessionStale)
	if err := os.Chtimes(ancient, when, when); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, code := unlinkAs(t, worktree, "mine")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); !os.IsNotExist(err) {
		t.Fatalf("an abandoned session's file held the junction: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

func TestWorktreeUnlinkTakesTheJunctionWhenItWasTheLastSession(t *testing.T) {
	main, worktree := linkedFixture(t, ".tools")
	mine := writeSessionState(t, worktree, "mine")

	stdout, stderr, code := unlinkAs(t, worktree, "mine")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); !os.IsNotExist(err) {
		t.Fatalf("the junction survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(main, ".tools", "godot")); err != nil {
		t.Fatalf("unlink reached through the junction: %v", err)
	}
	// Marked ended, not removed: a resume under the same id reads it again,
	// as it was.
	if body, err := os.ReadFile(mine); err != nil || string(body) != `{"blocks":0,"snapshots":{}}` {
		t.Fatalf("this session's own state file changed: %q, %v", body, err)
	}
	if n, err := sessions.Others(worktree, "other", sessionStale); err != nil || n != 0 {
		t.Fatalf("the ended session still counts: %d, %v", n, err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// The whole row: a session ends, its junction goes, and when it resumes under
// the same id it finds its state as it left it and counts for the others
// again.
func TestAnUnlinkedSessionThatResumesCountsAgain(t *testing.T) {
	_, worktree := linkedFixture(t, ".tools")
	t.Setenv(config.StateDirEnv, t.TempDir())
	if err := sessions.WriteState(worktree, "mine", sessions.SessionState{Base: "abc", Blocks: 1}); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := unlinkAs(t, worktree, "mine"); code != ExitOK {
		t.Fatalf("unlink: %d %s", code, stderr)
	}
	var stdout, stderr bytes.Buffer
	if code := SessionStart(strings.NewReader(`{"session_id":"mine","source":"resume"}`), &stdout, &stderr, worktree, "claude", "3.3.0"); code != ExitOK {
		t.Fatalf("session-start: %d %q", code, stderr.String())
	}
	// The base stays; the row of blocks starts again with the resume.
	if got := sessions.ReadState(worktree, "mine"); got.Base != "abc" || got.Blocks != 0 {
		t.Fatalf("state %+v", got)
	}
	if n, err := sessions.Others(worktree, "other", sessionStale); err != nil || n != 1 {
		t.Fatalf("others = %d, %v; the resumed session does not count", n, err)
	}
}

// A payload without an id is not a reason to unlink somebody else's link.
func TestWorktreeUnlinkWithoutASessionIdDoesNothing(t *testing.T) {
	_, worktree := linkedFixture(t, ".tools")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeUnlink(stdout, stderr, bytes.NewBufferString("{}"), worktree); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".tools", "godot")); err != nil {
		t.Fatalf("the junction was removed without an id to go on: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// And neither is a payload that is not a payload.
func TestWorktreeUnlinkWithoutAReadablePayloadDoesNothing(t *testing.T) {
	_, worktree := linkedFixture(t, ".tools")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeUnlink(stdout, stderr, bytes.NewBufferString("not json"), worktree); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".tools", "godot")); err != nil {
		t.Fatalf("the junction was removed over an unreadable payload: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

func TestWorktreeUnlinkInTheMainCheckoutDoesNothing(t *testing.T) {
	main, _ := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['.tools']\n")
	mkdirAll(t, filepath.Join(main, ".tools", "godot"))

	stdout, stderr, code := unlinkAs(t, main, "mine")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(main, ".tools", "godot")); err != nil {
		t.Fatalf("the main checkout's own directory was touched: %v", err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// The no-op cases, exit 0 and silent. This one fires at every session end in
// every project on the machine, and a session ends in plenty of directories
// that have nothing to do with any of this.
func TestTheUnlinkNoOpCasesAreSilentAndSuccessful(t *testing.T) {
	t.Run("no repository", func(t *testing.T) {
		assertUnlinkSilentOK(t, t.TempDir())
	})
	t.Run("no config", func(t *testing.T) {
		_, worktree := worktreeFixture(t)
		assertUnlinkSilentOK(t, worktree)
	})
	t.Run("config without the section", func(t *testing.T) {
		main, worktree := worktreeFixture(t)
		writeConfig(t, main, "[verify]\nlint = \"ruff check .\"\n")
		assertUnlinkSilentOK(t, worktree)
	})
}

func assertUnlinkSilentOK(t *testing.T, root string) {
	t.Helper()
	stdout, stderr, code := unlinkAs(t, root, "mine")
	if code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q; want silence", stdout, stderr)
	}
}

// A real directory at a configured path was never ours: it may be this tree's
// own build output, and `link` leaves such a path alone for the same reason.
func TestUnlinkLeavesARealDirectoryAlone(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['.tools']\n")
	mkdirAll(t, filepath.Join(main, ".tools"))
	own := filepath.Join(worktree, ".tools", "mine.txt")
	mkdirAll(t, filepath.Dir(own))
	writeFile(t, own, "mine")
	writeSessionState(t, worktree, "mine")

	assertUnlinkSilentOK(t, worktree)
	if _, err := os.Stat(own); err != nil {
		t.Fatalf("unlink removed a real directory: %v", err)
	}
}

// A junction pointing somewhere else is somebody's own arrangement, exactly as
// it is for the sweep. Being a reparse point at a configured path is not
// enough to make it ours.
func TestUnlinkLeavesAJunctionPointingElsewhereAlone(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['.tools']\n")
	mkdirAll(t, filepath.Join(main, ".tools"))
	mklink(t, filepath.Join(worktree, ".tools"), t.TempDir())
	writeSessionState(t, worktree, "mine")

	assertUnlinkSilentOK(t, worktree)
	if _, err := os.Lstat(filepath.Join(worktree, ".tools")); err != nil {
		t.Fatalf("a junction into another directory was removed: %v", err)
	}
}

// The other side of the containment check: a nested configured path whose
// intermediate directories are real is still unlinked. Without this, a check
// that skipped one component too many would stop taking `.loomux/vendor`
// out and nothing would say so.
func TestUnlinkReachesANestedPathThroughRealDirectories(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['.loomux/vendor']\n")
	mkdirAll(t, filepath.Join(main, ".loomux", "vendor", "ultraloom"))
	// A real directory, so that the junction lands in the worktree and the
	// state file below it does too.
	mkdirAll(t, filepath.Join(worktree, ".loomux"))
	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, worktree); code != ExitOK {
		t.Fatal("the fixture's own link was not created")
	}
	writeSessionState(t, worktree, "mine")

	assertUnlinkSilentOK(t, worktree)
	if _, err := os.Lstat(filepath.Join(worktree, ".loomux", "vendor")); !os.IsNotExist(err) {
		t.Fatalf("the nested junction survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(main, ".loomux", "vendor", "ultraloom")); err != nil {
		t.Fatalf("unlink reached through the junction: %v", err)
	}
}

// The same exposure the sweep had: Windows opens the final component of a path
// with FILE_FLAG_OPEN_REPARSE_POINT and follows every earlier one, so
// `<worktree>/.loomux/vendor` with a junction at `<worktree>/.loomux`
// reads the reparse point of `<main>/.loomux/vendor`. Without the
// containment check that junction satisfies both conditions unlink asks about
// -- a link leading into the main checkout -- and session end would take a
// mirrored directory out of the main checkout.
//
// No state file here: writing one would land in the main checkout's hooks
// directory through that very junction.
func TestUnlinkDoesNotReachThroughAnIntermediateLink(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['.loomux/vendor']\n")

	// The main checkout's own mirror path is a junction here as well, which is
	// what makes it look like a candidate once an earlier component leads to it.
	runtimeDir := filepath.Join(main, "runtime")
	mkdirAll(t, runtimeDir)
	mainVendor := filepath.Join(main, ".loomux", "vendor")
	mklink(t, mainVendor, runtimeDir)
	mklink(t, filepath.Join(worktree, ".loomux"), filepath.Join(main, ".loomux"))

	assertUnlinkSilentOK(t, worktree)
	if _, err := os.Lstat(mainVendor); err != nil {
		t.Fatalf("the main checkout's own junction was removed: %v", err)
	}
	if _, err := os.Stat(runtimeDir); err != nil {
		t.Fatalf("what the main checkout's junction pointed at is gone: %v", err)
	}
}

// Damage is a failure here too: read as "nothing to mirror", a broken config
// would switch the mechanism off without a word.
func TestUnlinkReportsABrokenConfig(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree\n")

	_, stderr, code := unlinkAs(t, worktree, "mine")
	if code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "worktree unlink") {
		t.Fatalf("stderr = %q, want the subcommand's name in it", stderr)
	}
}

// A session that cannot be marked ended must not read as "nobody else is
// here": it would then count as somebody else's on the next run, and the
// junction would stay for ever. A directory with something in it, where the
// marker goes, is the portable way to make the write refuse.
func TestUnlinkReportsASessionItCannotRetire(t *testing.T) {
	_, worktree := linkedFixture(t, ".tools")
	writeSessionState(t, worktree, "mine")
	busy := filepath.Join(worktree, filepath.FromSlash(sessions.StateDir), "mine.ended")
	mkdirAll(t, busy)
	writeFile(t, filepath.Join(busy, "inside"), "x")

	_, stderr, code := unlinkAs(t, worktree, "mine")
	if code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "retiring ") {
		t.Fatalf("stderr = %q, want Retire's error in it", stderr)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".tools", "godot")); err != nil {
		t.Fatalf("the junction went although the session could not retire: %v", err)
	}
}

// A count that could not be taken is not a count of zero. The state directory
// is held open without any share mode -- not denied by ACL, which an elevated
// token reads through -- so os.ReadDir of it fails with a sharing violation,
// while the marker is still written into it and Retire passes.
func TestUnlinkReportsAStateDirectoryItCannotRead(t *testing.T) {
	_, worktree := linkedFixture(t, ".tools")
	hooks := filepath.Join(worktree, filepath.FromSlash(sessions.StateDir))
	mkdirAll(t, hooks)
	testlock.LockDir(t, hooks)

	_, stderr, code := unlinkAs(t, worktree, "mine")
	if code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	// Others' own prefix. Under this lock Retire fails at nothing, but it
	// stands earlier in the same function and would produce the same exit code
	// and the same non-empty stderr, so the message is what tells them apart.
	if !strings.Contains(stderr.String(), "reading ") {
		t.Fatalf("stderr = %q, want the count's error in it", stderr)
	}
}

// A candidate unlink cannot even look at is a failure and not a skip, for the
// sweep's reason: it must not go quiet about a path it could not decide. `?` is
// legal in a configured path and illegal in a Windows filename.
func TestUnlinkReportsAPathItCannotInspect(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['bad?name']\n")
	writeSessionState(t, worktree, "mine")

	_, stderr, code := unlinkAs(t, worktree, "mine")
	if code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if stderr.Len() == 0 {
		t.Fatal("a failure said nothing")
	}
}

// registered answers whether git still holds `worktree` as a working tree.
//
// Compared as cleaned, case-folded text and not with os.SameFile, although
// every other filesystem-identity comparison in worktree.go does: the path
// this is asked about has just been deleted, and SameFile answers false for
// anything absent -- so the check would pass no matter what git still holds.
// filepath.Clean puts git's forward slashes into the platform spelling; the
// fold is for a drive letter neither side promises the case of.
func registered(t *testing.T, main string, worktree string) bool {
	t.Helper()
	command := exec.Command("git", "worktree", "list", "--porcelain")
	command.Dir = main
	// The clean environment the `git` helper above explains.
	command.Env = gitenv.Environ()
	out, err := command.Output()
	if err != nil {
		t.Fatalf("git worktree list: %v", err)
	}
	wanted := filepath.Clean(worktree)
	for _, line := range strings.Split(string(out), "\n") {
		rest, found := strings.CutPrefix(strings.TrimRight(line, "\r"), "worktree ")
		if found && strings.EqualFold(filepath.Clean(rest), wanted) {
			return true
		}
	}
	return false
}

// The measured reason this command exists. On 2026-09-07 in a t.TempDir()
// fixture, `git worktree remove --force` on a worktree holding a junction
// exited 0 with no output, dropped the porcelain entry, and left both the
// directory and the junction standing.
func TestWorktreeRemoveLeavesNothingBehind(t *testing.T) {
	main, worktree := linkedFixture(t, ".tools")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}

	if _, err := os.Lstat(worktree); !os.IsNotExist(err) {
		t.Fatalf("the worktree directory survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(main, ".tools", "godot")); err != nil {
		t.Fatalf("remove reached through the junction: %v", err)
	}
	if registered(t, main, worktree) {
		t.Fatal("git still holds the worktree")
	}
	// Unlike the two hook commands this one speaks: a person deleting
	// something should read what was deleted.
	if !strings.Contains(stdout.String(), "removed ") {
		t.Fatalf("stdout = %q, want what was removed named in it", stdout)
	}
}

// The refusal a wrapper whose worst outcome is deleting the repository has to
// make first.
func TestWorktreeRemoveRefusesTheMainCheckout(t *testing.T) {
	main, _ := worktreeFixture(t)

	// Three spellings of one directory. The refusal rests on os.SameFile and
	// not on text, and that is what these pin: a spelling that slips past it
	// falls through to the registration check, whose message would then be
	// false about the main checkout -- and the porcelain lists Main, so it
	// would not refuse at all.
	for _, spelling := range []string{
		main,
		main + string(os.PathSeparator),
		filepath.Join(main, "."),
	} {
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := WorktreeRemove(stdout, stderr, spelling); code != ExitInternal {
			t.Fatalf("exit = %d for %q, want ExitInternal (stderr: %s)", code, spelling, stderr)
		}
		if !strings.Contains(stderr.String(), "main checkout") {
			t.Fatalf("stderr = %q for %q, want the main checkout named in it", stderr, spelling)
		}
		if _, err := os.Stat(main); err != nil {
			t.Fatalf("the main checkout was touched: %v", err)
		}
	}
}

// git's own spelling of the path is what the removal gets, and not the
// caller's argument: `command.Dir` is the main checkout, so a relative
// argument would resolve there while the refusals above resolved it here. A
// trailing separator is the cheapest spelling that differs from the porcelain
// one and still opens the same directory.
func TestWorktreeRemoveUsesGitsSpellingOfThePath(t *testing.T) {
	main, worktree := worktreeFixture(t)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	spelling := worktree + string(os.PathSeparator)
	if code := WorktreeRemove(stdout, stderr, spelling); code != ExitOK {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr)
	}
	if stdout.String() != "removed "+filepath.Clean(worktree)+"\n" {
		t.Fatalf("stdout = %q, want git's cleaned spelling of %q", stdout, worktree)
	}
	if registered(t, main, worktree) {
		t.Fatal("git still holds the worktree")
	}
}

// A junction nothing here took out -- no mirror configured, so unlink is a
// no-op over it -- is exactly the leftover this subcommand exists to prevent,
// and git reports success over it. Measured on 2026-09-07: exit 0, no output,
// porcelain entry gone, directory and junction standing. So the directory is
// checked before anything says "removed".
func TestWorktreeRemoveDoesNotClaimSuccessOverALeftover(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	mkdirAll(t, filepath.Join(main, ".tools", "godot"))
	mklink(t, filepath.Join(worktree, ".tools"), filepath.Join(main, ".tools"))

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want nothing claimed", stdout)
	}
	if !strings.Contains(stderr.String(), "still stands; nothing here removed it") {
		t.Fatalf("stderr = %q, want the leftover reported and no cause claimed", stderr)
	}
	if _, err := os.Lstat(worktree); err != nil {
		t.Fatalf("the leftover the message names is not there: %v", err)
	}
}

func TestWorktreeRemoveOfADirectoryGitDoesNotHoldIsAFailure(t *testing.T) {
	main, _ := worktreeFixture(t)
	stranger := filepath.Join(main, ".worktrees", "stranger")
	mkdirAll(t, stranger)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, stranger); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if _, err := os.Stat(stranger); err != nil {
		t.Fatalf("a directory git does not hold was removed anyway: %v", err)
	}
}

// Unlike the hook commands, a directory without a repository is a fault here
// and not a no-op: this one was named by hand, and the name was wrong.
func TestWorktreeRemoveOutsideARepositoryIsAFailure(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, t.TempDir()); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if stderr.Len() == 0 {
		t.Fatal("a failure said nothing")
	}
}

// A configuration that cannot be read leaves it unknown which junctions are
// ours, and asking git while that is unknown is the one order this command
// exists to avoid.
func TestWorktreeRemoveReportsABrokenConfig(t *testing.T) {
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree\n")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "worktree remove") {
		t.Fatalf("stderr = %q, want the subcommand's name in it", stderr)
	}
	if !registered(t, main, worktree) {
		t.Fatal("git was asked although the junctions were not taken out")
	}
}

// And a candidate unlink cannot even look at stops it in the same place. `?`
// is legal in a configured path and illegal in a Windows filename.
func TestWorktreeRemoveReportsAPathItCannotInspect(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['bad?name']\n")

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !registered(t, main, worktree) {
		t.Fatal("git was asked although a configured path was undecided")
	}
}

// git's own refusal has to reach the caller. A lock is what makes it refuse
// under a single --force: measured on 2026-09-07, `git worktree remove
// --force` on a locked tree exited 128 with "cannot remove a locked working
// tree; use 'remove -f -f' to override or unlock first", and the directory
// stood.
//
// On a linked fixture, so the state after the refusal is the one the ordering
// produces and not an empty case: junctions out, tree still registered. That
// is the price of asking git last, and it is paid back at the next session
// start -- which is asserted here rather than argued.
func TestWorktreeRemoveReportsGitsRefusal(t *testing.T) {
	main, worktree := linkedFixture(t, ".tools")
	git(t, main, "worktree", "lock", worktree)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "locked") {
		t.Fatalf("stderr = %q, want git's own words in it", stderr)
	}
	if _, err := os.Stat(worktree); err != nil {
		t.Fatalf("the worktree went away although git refused: %v", err)
	}
	junctionPath := filepath.Join(worktree, ".tools")
	if _, err := os.Lstat(junctionPath); !os.IsNotExist(err) {
		t.Fatalf("the junction was not taken out before git was asked: %v", err)
	}
	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, worktree); code != ExitOK {
		t.Fatalf("worktree-link exit = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(junctionPath, "godot")); err != nil {
		t.Fatalf("the next session start did not put the junction back: %v", err)
	}
}
