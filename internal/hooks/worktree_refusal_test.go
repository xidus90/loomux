package hooks

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docs/en/hooks.md: "If git refuses, the junctions are already out, and the
// next `link` puts them back." That holds for the configured mirrors only.
// Since every other link is swept before git is asked, a refusal by git -- a
// locked worktree is the one a single --force does not override -- leaves the
// tree standing without the user's own links, and no `link` brings them back.
func TestWorktreeRemoveKeepsForeignLinksWhenGitRefuses(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	assets := filepath.Join(main, "assets")
	mkdirAll(t, assets)
	writeFile(t, filepath.Join(assets, "big.bin"), "x")
	own := filepath.Join(worktree, "assets")
	mklink(t, own, assets)
	git(t, main, "worktree", "lock", worktree)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "locked") {
		t.Fatalf("stderr = %q, want git's lock refusal", stderr)
	}
	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, worktree); code != ExitOK {
		t.Fatalf("worktree-link exit = %d, want 0", code)
	}
	if _, err := os.Lstat(own); err != nil {
		t.Fatalf("git refused and the worktree stands, but its own junction is gone for good: %v", err)
	}
}

// The refusal names the way out, and the way out works: once the lock is
// lifted the same call removes the tree, links and all.
func TestWorktreeRemoveOfALockedWorktreeSaysHowToUnlockIt(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	assets := filepath.Join(main, "assets")
	mkdirAll(t, assets)
	mklink(t, filepath.Join(worktree, "assets"), assets)
	git(t, main, "worktree", "lock", "--reason", "on a stick", worktree)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "git worktree unlock") {
		t.Fatalf("stderr = %q, want the unlock command named", stderr)
	}

	git(t, main, "worktree", "unlock", worktree)
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitOK {
		t.Fatalf("exit after unlock = %d, want 0 (stderr: %s)", code, stderr)
	}
	if _, err := os.Lstat(worktree); !os.IsNotExist(err) {
		t.Fatalf("the unlocked worktree still stands: %v", err)
	}
	if _, err := os.Stat(assets); err != nil {
		t.Fatalf("the link's target was touched: %v", err)
	}
}

// The configured mirrors are links like any other: a lock keeps them in place
// too, and git is not asked at all.
func TestWorktreeRemoveOfALockedWorktreeLeavesItsMirrorsInPlace(t *testing.T) {
	main, worktree := linkedFixture(t, ".tools")
	git(t, main, "worktree", "lock", worktree)
	stubGit(t, func(string, string) ([]byte, error) {
		t.Error("git was asked to remove a locked worktree")
		return nil, nil
	})

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stderr.String(), "locked") {
		t.Fatalf("stderr = %q, want the lock named", stderr)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".tools", "godot")); err != nil {
		t.Fatalf("the mirror was taken out although the worktree was refused: %v", err)
	}
}

// A hand-made junction at a configured path that leads elsewhere is the
// user's own arrangement. Dropped before git refuses, it would be gone and the
// next link would make the mirror's junction on its path: replaced, not just
// lost.
func TestWorktreeRemoveOfALockedWorktreeKeepsAHandMadeJunctionAtAMirrorPath(t *testing.T) {
	requireWindows(t)
	main, worktree := worktreeFixture(t)
	writeConfig(t, main, "[worktree]\nmirror = ['.tools']\n")
	mkdirAll(t, filepath.Join(main, ".tools"))
	elsewhere := t.TempDir()
	writeFile(t, filepath.Join(elsewhere, "own.txt"), "own")
	mklink(t, filepath.Join(worktree, ".tools"), elsewhere)
	git(t, main, "worktree", "lock", worktree)

	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := WorktreeRemove(stdout, stderr, worktree); code != ExitInternal {
		t.Fatalf("exit = %d, want ExitInternal (stderr: %s)", code, stderr)
	}
	if code := WorktreeLink(&bytes.Buffer{}, &bytes.Buffer{}, worktree); code != ExitOK {
		t.Fatalf("worktree-link exit = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".tools", "own.txt")); err != nil {
		t.Fatalf("the user's junction at the mirror path no longer leads to its own target: %v", err)
	}
}
