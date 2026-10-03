package serve_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/serve"
)

func TestUpdateLoopWaitsFirstThenEveryInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var waits []time.Duration
	after := func(d time.Duration) <-chan time.Time {
		waits = append(waits, d)
		c := make(chan time.Time, 1)
		c <- time.Now()
		return c
	}
	runs := 0
	update := func(context.Context) {
		runs++
		if runs == 3 {
			cancel()
		}
	}
	serve.UpdateLoop(ctx, update, time.Minute, 24*time.Hour, after)
	if runs != 3 {
		t.Fatalf("runs = %d, want 3", runs)
	}
	want := []time.Duration{time.Minute, 24 * time.Hour, 24 * time.Hour}
	for i, w := range want {
		if waits[i] != w {
			t.Fatalf("waits = %v, want %v first", waits, want)
		}
	}
}

func TestUpdateLoopEndsBeforeTheFirstPassWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	never := func(time.Duration) <-chan time.Time { return nil }
	serve.UpdateLoop(ctx, func(context.Context) { t.Fatal("a cancelled loop ran a pass") }, time.Minute, time.Hour, never)
}

// Run starts the loop when it is given a pass.
func TestRunStartsTheUpdateLoop(t *testing.T) {
	dir := t.TempDir()
	ran := make(chan struct{}, 1)
	var calls atomic.Int32
	after := func(time.Duration) <-chan time.Time {
		if calls.Add(1) > 1 {
			return nil
		}
		c := make(chan time.Time, 1)
		c <- time.Now()
		return c
	}
	startWith(t, dir, serve.Options{
		StateDir: dir, RegistryDir: dir,
		Update:      func(context.Context) { ran <- struct{}{} },
		UpdateAfter: after,
	})
	select {
	case <-ran:
	case <-time.After(5 * time.Second):
		t.Fatal("serve never ran an update pass")
	}
}

// Run ends the loop with the service and waits for a pass in flight: the
// process exits once Run returns, and a pass killed between swap's two
// renames would leave no loomux.exe behind.
func TestRunWaitsForTheUpdatePassBeforeItReturns(t *testing.T) {
	dir := t.TempDir()
	started := make(chan struct{})
	release := make(chan struct{})
	now := func(time.Duration) <-chan time.Time {
		c := make(chan time.Time, 1)
		c <- time.Now()
		return c
	}
	_, cancel, done := startWith(t, dir, serve.Options{
		StateDir: dir, RegistryDir: dir,
		Update: func(ctx context.Context) {
			close(started)
			<-ctx.Done()
			<-release
		},
		UpdateAfter: now,
	})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("serve never ran an update pass")
	}
	cancel()
	select {
	case <-done:
		close(release)
		t.Fatal("Run returned while the update pass was still running")
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run = %v, want an orderly stop", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return once the pass had")
	}
}

// Without a clock of its own the loop waits on the real one: serve starts and
// stops, and the first minute never runs out while it does.
func TestRunUpdatesOnTheRealClockByDefault(t *testing.T) {
	dir := t.TempDir()
	_, cancel, done := startWith(t, dir, serve.Options{
		StateDir: dir, RegistryDir: dir,
		Update: func(context.Context) { t.Error("a pass ran before the first minute was out") },
	})
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run = %v, want an orderly stop", err)
	}
}
