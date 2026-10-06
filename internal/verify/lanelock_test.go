package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/lock"
)

func TestLockPathPerStackAndArea(t *testing.T) {
	root := filepath.Join("C:", "repo")
	if got := LockPath(root, "gdscript", "."); got != filepath.Join(root, ".loomux", "state", "locks", "gdscript-root.lock") {
		t.Fatal(got)
	}
	if got := LockPath(root, "go", "a/b"); got != filepath.Join(root, ".loomux", "state", "locks", "go-a_b.lock") {
		t.Fatal(got)
	}
}

// holdLock takes path in this process, as another run would; LockFileEx and
// flock lock per handle, so a second TryAcquire here is refused.
func holdLock(t *testing.T, path, who string) *lock.Handle {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	h, ok, err := lock.TryAcquire(path)
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	if who != "" {
		if err := os.WriteFile(path+".who", []byte(who), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { h.Release() })
	return h
}

// lockedJob builds its lock path by hand, so a LockPath that answers "" does
// not send a test's files into the package directory.
func lockedJob(root string) Job {
	j := job("test/gdscript", -1, "t")
	j.Lock = filepath.Join(root, ".loomux", "state", "locks", "gdscript-root.lock")
	return j
}

func TestRunTakesAndReleasesTheLaneLock(t *testing.T) {
	root := t.TempDir() // no .loomux/state yet: a fresh clone
	j := lockedJob(root)
	var whoDuring string
	f := &fakeStart{answer: func(child.Spec) child.Result {
		data, _ := os.ReadFile(j.Lock + ".who")
		whoDuring = string(data)
		return child.Result{}
	}}
	o := opts(f)
	o.Caller = "check precommit"
	if out := Run([]Job{j}, o); out[0].State != StateOK {
		t.Fatalf("%+v", out[0])
	}
	if !strings.Contains(whoDuring, "caller=check precommit") || !strings.Contains(whoDuring, "pid=") {
		t.Fatalf("holder file while running: %q", whoDuring)
	}
	if _, err := os.Stat(j.Lock + ".who"); !os.IsNotExist(err) {
		t.Fatalf("holder file left behind: %v", err)
	}
	h, ok, err := lock.TryAcquire(j.Lock)
	if err != nil || !ok {
		t.Fatalf("lock not released: %v %v", ok, err)
	}
	h.Release()
}

func TestRunWaitsForTheLockUntilTheBudgetIsSpent(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	holdLock(t, j.Lock, "pid=4711\ncaller=check precommit\nsince=2026-10-06T20:00:00Z\n")
	f := &fakeStart{answer: ok}
	c := &clock{t: time.Date(2026, 10, 6, 20, 0, 0, 0, time.UTC)}
	var told []string
	o := opts(f)
	o.Scope, o.Budget, o.Now, o.Sleep = ScopeCheck, time.Minute, c.now, func(time.Duration) {}
	o.Waiting = func(j Job, who string) { told = append(told, j.Name+": "+who) }
	out := Run([]Job{j}, o)
	if out[0].State != StateBudget || len(f.started) != 0 || !strings.Contains(out[0].Output, "held by loomux pid 4711, check precommit") {
		t.Fatalf("%+v", out[0])
	}
	if len(told) != 1 || !strings.HasPrefix(told[0], "test/gdscript: held by loomux pid 4711, check precommit, since ") {
		t.Fatalf("told %q", told)
	}
}

// Lanes of one stack and area share a lock, so the holder a lane waits for can
// be a lane of the same run; it is not named as another process.
func TestRunSaysWhenTheLockIsHeldByAnotherLaneOfTheRun(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	holdLock(t, j.Lock, fmt.Sprintf("pid=%d\ncaller=check precommit\nsince=2026-10-06T20:00:00Z\n", os.Getpid()))
	c := &clock{t: time.Date(2026, 10, 6, 20, 0, 0, 0, time.UTC)}
	o := opts(&fakeStart{answer: ok})
	o.Scope, o.Budget, o.Now, o.Sleep = ScopeCheck, time.Minute, c.now, func(time.Duration) {}
	out := Run([]Job{j}, o)
	if want := "held by another lane of this run, check precommit, since "; out[0].State != StateBudget || !strings.Contains(out[0].Output, want) {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunWaitsForTheLockUntilTheTimeoutWithoutABudget(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	holdLock(t, j.Lock, "")
	c := &clock{t: time.Now()}
	o := opts(&fakeStart{answer: ok})
	o.Timeout, o.Now, o.Sleep = time.Minute, c.now, func(time.Duration) {}
	if out := Run([]Job{j}, o); out[0].State != StateTimedOut || !strings.Contains(out[0].Output, "held by another run") {
		t.Fatalf("%+v", out[0])
	}
}

// With neither a budget nor a timeout nothing ends the wait but the lock, and
// whoever waits is told once, however long it takes.
func TestRunWaitsForTheLockWithoutALimit(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	h := holdLock(t, j.Lock, "")
	told, sleeps := 0, 0
	o := opts(&fakeStart{answer: ok})
	o.Timeout = 0
	o.Waiting = func(Job, string) { told++ }
	o.Sleep = func(time.Duration) {
		if sleeps++; sleeps == 3 {
			h.Release()
		}
	}
	if out := Run([]Job{j}, o); out[0].State != StateOK || sleeps != 3 || told != 1 {
		t.Fatalf("%+v, slept %d, told %d", out[0], sleeps, told)
	}
}

// While one lane waits for its lock it holds no slot: with one slot, the free
// lane is held back until the other is waiting, must then start, and only
// then is the lock let go. A waiting lane that kept its slot would leave the
// free lane unstarted, and the lock would go only after 400 sleeps.
func TestRunHoldsNoSlotWhileWaitingForTheLock(t *testing.T) {
	root := t.TempDir()
	locked := lockedJob(root)
	h := holdLock(t, locked.Lock, "")
	f := &fakeStart{answer: ok}
	o := opts(f)
	o.MaxParallel = 1
	waiting := make(chan struct{})
	o.Waiting = func(Job, string) { close(waiting) }
	o.Look = func(s string) (string, error) {
		if s == "l" {
			select {
			case <-waiting:
			case <-time.After(5 * time.Second):
			}
		}
		return s, nil
	}
	var once sync.Once
	sleeps := 0
	o.Sleep = func(time.Duration) {
		sleeps++
		f.mu.Lock()
		freeRan := len(f.started) > 0
		f.mu.Unlock()
		if freeRan || sleeps > 400 {
			once.Do(func() { h.Release() })
		}
		time.Sleep(5 * time.Millisecond)
	}
	out := Run([]Job{locked, job("lint/gdscript", -1, "l")}, o)
	if sleeps > 400 || out[0].State != StateOK || out[1].State != StateOK || !slices.Equal(f.started, []string{"l", "t"}) {
		t.Fatalf("sleeps %d, started %v, %+v", sleeps, f.started, out)
	}
}

// A holder file a dead run left beside a free lock is not read.
func TestRunTakesAFreeLockBesideAStaleHolderFile(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	if err := os.MkdirAll(filepath.Dir(j.Lock), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(j.Lock+".who", []byte("pid=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var told int
	o := opts(&fakeStart{answer: ok})
	o.Waiting = func(Job, string) { told++ }
	if out := Run([]Job{j}, o); out[0].State != StateOK || told != 0 {
		t.Fatalf("%+v told %d", out[0], told)
	}
}

func TestRunLocksTwoAreasApart(t *testing.T) {
	root := t.TempDir()
	a := lockedJob(root)
	b := job("test/gdscript@b", -1, "t")
	b.Lock = filepath.Join(root, ".loomux", "state", "locks", "gdscript-b.lock")
	holdLock(t, a.Lock, "")
	o := opts(&fakeStart{answer: ok})
	if out := Run([]Job{b}, o); out[0].State != StateOK {
		t.Fatalf("%+v", out[0])
	}
}

func TestRunFailsALockItCannotOpen(t *testing.T) {
	root := t.TempDir()
	// The locks directory is a file: nothing can be made below it.
	put := filepath.Join(root, ".loomux", "state")
	if err := os.MkdirAll(put, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(put, "locks"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if out := Run([]Job{lockedJob(root)}, opts(&fakeStart{answer: ok})); out[0].State != StateFailed {
		t.Fatalf("%+v", out[0])
	}
}

// The lock path is a directory: it can be made but not opened as a file.
func TestRunFailsALockItCannotTake(t *testing.T) {
	root := t.TempDir()
	j := lockedJob(root)
	if err := os.MkdirAll(j.Lock, 0o755); err != nil {
		t.Fatal(err)
	}
	f := &fakeStart{answer: ok}
	if out := Run([]Job{j}, opts(f)); out[0].State != StateFailed || len(f.started) != 0 {
		t.Fatalf("%+v", out[0])
	}
}

func TestWaitingToWritesOneLine(t *testing.T) {
	var out strings.Builder
	WaitingTo(&out)(Job{Name: "test/go"}, "held by another run")
	if out.String() != "test/go: waiting for the lock (held by another run)\n" {
		t.Fatalf("%q", out.String())
	}
}
