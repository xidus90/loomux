package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeClock replaces the clock and the pause of the retry loop: a sleep only
// advances the clock, so the loop's arithmetic is observable without waiting.
type fakeClock struct {
	start  time.Time
	now    time.Time
	sleeps int
}

func useFakeClock(t *testing.T) *fakeClock {
	t.Helper()
	start := time.Unix(0, 0)
	clock := &fakeClock{start: start, now: start}
	oldNow, oldSleep := replaceNow, replaceSleep
	replaceNow = func() time.Time { return clock.now }
	replaceSleep = func(d time.Duration) {
		clock.sleeps++
		clock.now = clock.now.Add(d)
	}
	t.Cleanup(func() { replaceNow, replaceSleep = oldNow, oldSleep })
	return clock
}

// countRenames wraps the rename seam: the first fail calls are refused with
// err, every later call goes through to the file system.
func countRenames(t *testing.T, fail int, err error) *int {
	t.Helper()
	calls := new(int)
	old := replaceRename
	replaceRename = func(source, target string) error {
		*calls++
		if *calls <= fail {
			return err
		}
		return os.Rename(source, target)
	}
	t.Cleanup(func() { replaceRename = old })
	return calls
}

// A directory in the target's place refuses the rename on every platform, so
// the loop runs until the budget is gone. The bound is on the attempts: a loop
// that ignored the budget would never end and one that ignored the pause
// would spin through thousands of attempts.
func TestReplaceEventuallyGivesUpAfterTheBudget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "occupied")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	clock := useFakeClock(t)

	if err := ReplaceEventually(source, target, 100*time.Millisecond); err == nil {
		t.Fatal("ReplaceEventually: want error for a directory in the way")
	}
	attempts := clock.sleeps + 1
	if attempts < 3 || attempts > 6 {
		t.Fatalf("attempts = %d, want 3..6 for a 100 ms budget and a 20 ms pause", attempts)
	}
	if elapsed := clock.now.Sub(clock.start); elapsed > 100*time.Millisecond {
		t.Fatalf("waited %s, more than the budget", elapsed)
	}
}

func TestReplaceEventuallySucceedsWhenTheTargetFrees(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	useFakeClock(t)
	calls := countRenames(t, 1, errors.New("sharing violation"))
	retries := 0
	oldHook := OnRetry
	OnRetry = func() { retries++ }
	t.Cleanup(func() { OnRetry = oldHook })

	if err := ReplaceEventually(source, target, time.Second); err != nil {
		t.Fatalf("ReplaceEventually: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "new" {
		t.Fatalf("content = %q, want the source's", got)
	}
	if *calls != 2 {
		t.Fatalf("renames = %d, want 2", *calls)
	}
	if retries != 1 {
		t.Fatalf("OnRetry calls = %d, want 1: one rename failed", retries)
	}
}

// The unexported entry point keeps the old budget: five tries with four
// pauses between them.
func TestReplaceEventuallyKeepsTheOldFiveTries(t *testing.T) {
	useFakeClock(t)
	calls := countRenames(t, 100, errors.New("busy"))

	if err := replaceEventually("a", "b"); err == nil {
		t.Fatal("replaceEventually: want the last rename's error")
	}
	if *calls != replaceTries {
		t.Fatalf("renames = %d, want %d", *calls, replaceTries)
	}
}
