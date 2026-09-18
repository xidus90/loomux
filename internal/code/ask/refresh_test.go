package ask_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/testlock"
)

// hashOf is the hash a build would have recorded for this file.
func hashOf(t *testing.T, abs string) string {
	t.Helper()
	b, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func TestEnsureFreshRebuildsWhenThereIsNoRecord(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	// "No record" means unknown, never clean. A fresh clone has no state at
	// all, and the first question must not answer from nothing.
	if built != 1 {
		t.Fatalf("built %d times, want 1", built)
	}
}

func TestEnsureFreshDoesNothingOnACleanTree(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, f := range stat {
		hashes[f.Rel] = hashOf(t, f.Abs)
	}
	if err := freshness.Write(root, "go/1", stat, hashes); err != nil {
		t.Fatal(err)
	}
	// Clean means the whole of what a question reads, sidecar included.
	if err := lexicon.Write(root, lexicon.Build(&model.Graph{})); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	if built != 0 {
		t.Fatalf("built %d times, want none: the probe is the whole point", built)
	}
}

func TestEnsureFreshRebuildsWhenTheSidecarIsGone(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, f := range stat {
		hashes[f.Rel] = hashOf(t, f.Abs)
	}
	if err := freshness.Write(root, "go/1", stat, hashes); err != nil {
		t.Fatal(err)
	}
	built := 0
	var notices []string

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, func(s string) { notices = append(notices, s) })

	// The record is clean and knows nothing about the sidecar. Deciding on it
	// alone would leave every later question ranking without the body text,
	// until a source file happened to move.
	if built != 1 {
		t.Fatalf("built %d times, want 1: a missing sidecar is drift", built)
	}
	if len(notices) == 0 || !strings.Contains(notices[0], "no ask index") {
		t.Fatalf("notices %v want 'no ask index'", notices)
	}
}

func TestEnsureFreshRebuildsWhenFilesMoved(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a.go")
	if err := os.WriteFile(file, []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stat, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{stat[0].Rel: hashOf(t, stat[0].Abs)}
	if err := freshness.Write(root, "go/1", stat, hashes); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("package a // edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	built := 0
	var notices []string
	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, func(s string) { notices = append(notices, s) })
	if built != 1 {
		t.Fatalf("built %d times, want 1", built)
	}
	if len(notices) == 0 || !strings.Contains(notices[0], "files moved") {
		t.Fatalf("notice %v want 'files moved'", notices)
	}
}

func TestEnsureFreshSurvivesAFailedProbe(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	var notices []string
	ask.EnsureFresh(root, "go/1", func() error { return nil }, func(s string) { notices = append(notices, s) })
	if len(notices) == 0 || !strings.Contains(notices[0], "freshness probe failed") {
		t.Fatalf("notices %v must carry probe failure", notices)
	}
}

func TestEnsureFreshSurvivesAFailedRebuild(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var notices []string

	ask.EnsureFresh(root, "go/1",
		func() error { return errors.New("disk full") },
		func(s string) { notices = append(notices, s) })

	// Never fatal: a failed rebuild answers from the graph on disk. A question
	// that works today must not start failing because a rebuild could not run.
	if len(notices) == 0 || !strings.Contains(strings.Join(notices, " "), "disk full") {
		t.Fatalf("notices %v must carry the reason", notices)
	}
}

func TestEnsureFreshSkipsWhenAnotherRunHoldsTheLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ask.LockPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ask.LockPath(root), []byte("999999"), 0o644); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	// No stampede: concurrent questions must not pile rebuilds on each other.
	// The loser answers from what is on disk.
	if built != 0 {
		t.Fatalf("built %d times while the lock was held", built)
	}
}

func TestEnsureFreshReleasesTheLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ask.EnsureFresh(root, "go/1", func() error { return nil }, nil)
	if _, err := os.Stat(ask.LockPath(root)); !os.IsNotExist(err) {
		t.Fatal("the lock must be gone once the rebuild finished")
	}
}

func TestEnsureFreshBreaksAStaleLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ask.LockPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ask.LockPath(root), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(ask.LockPath(root), old, old); err != nil {
		t.Fatal(err)
	}
	built := 0

	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	// A run killed mid-rebuild leaves its lock behind. Without a staleness
	// rule, the graph would never refresh again on that checkout.
	if built != 1 {
		t.Fatalf("built %d times, want 1: a stale lock must not be forever", built)
	}
}

func TestEnsureFreshFailsWhenStateDirectoryCannotBeCreated(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	built := 0
	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	if built != 0 {
		t.Fatal("must not build when lock directory cannot be created")
	}
}

func TestEnsureFreshCannotBreakLockedStaleFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ask.LockPath(root)), 0o755); err != nil {
		t.Fatal(err)
	}
	lockFile := ask.LockPath(root)
	if err := os.WriteFile(lockFile, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(lockFile, old, old); err != nil {
		t.Fatal(err)
	}
	testlock.Lock(t, lockFile)
	built := 0
	ask.EnsureFresh(root, "go/1", func() error { built++; return nil }, nil)
	if built != 0 {
		t.Fatal("must not build when stale lock cannot be removed")
	}
}

func TestTakeOverLeavesALiveLock(t *testing.T) {
	root := t.TempDir()
	path := ask.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// This run saw the lock stale, but another run broke it first and created
	// its own, fresh one at the same path.
	if err := os.WriteFile(path, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}

	if ask.TakeOver(path) {
		t.Fatal("took over a live lock")
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "other" {
		t.Fatalf("the other run's lock is gone: %q, %v", b, err)
	}
}

func TestTakeOverLeavesALiveLockWithoutHardLinks(t *testing.T) {
	root := t.TempDir()
	path := ask.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A network share, an exFAT volume, a container bind mount: the link fails
	// and the restore has to happen anyway. Removing the claim regardless, as
	// this used to, deleted the live lock and let both runs rebuild.
	defer ask.SwapLinkFile(func(string, string) error {
		return &os.LinkError{Op: "link", Err: errors.New("not supported")}
	})()

	if ask.TakeOver(path) {
		t.Fatal("took over a live lock")
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "other" {
		t.Fatalf("the other run's lock is gone: %q, %v", b, err)
	}
	if _, err := os.Stat(path + "." + strconv.Itoa(os.Getpid()) + ".stale"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the claim was left behind: %v", err)
	}
}

func TestTakeOverDropsItsClaimWhenTheLockIsBackAlready(t *testing.T) {
	root := t.TempDir()
	path := ask.LockPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A third run created its own lock between the rename and the restore.
	// Its lock is the live one; this run has nothing left but its copy, and a
	// restore that overwrote the target would take that run's lock away.
	defer ask.SwapLinkFile(func(_, newname string) error {
		if err := os.WriteFile(newname, []byte("third"), 0o644); err != nil {
			t.Fatal(err)
		}
		return &os.LinkError{Op: "link", Err: fs.ErrExist}
	})()

	if ask.TakeOver(path) {
		t.Fatal("took over a live lock")
	}
	b, err := os.ReadFile(path)
	if err != nil || string(b) != "third" {
		t.Fatalf("the third run's lock is gone: %q, %v", b, err)
	}
}

func TestTakeOverFailsWhenTheLockVanished(t *testing.T) {
	// Another run broke the stale lock first and has not created its own yet.
	if ask.TakeOver(ask.LockPath(t.TempDir())) {
		t.Fatal("took over a lock that no longer exists")
	}
}
