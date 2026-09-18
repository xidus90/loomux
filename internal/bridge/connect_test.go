package bridge_test

import (
	"errors"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/bridge"
	"github.com/xidus90/loomux/internal/serve"
)

func stateWithBuild(modTime time.Time) *serve.State {
	return &serve.State{
		Local:      serve.Endpoint{URL: "http://127.0.0.1:1/mcp", Token: "l"},
		Cloud:      serve.Endpoint{URL: "http://127.0.0.1:2/mcp", Token: "c"},
		PID:        4711,
		Executable: `C:\other-checkout\bin\loomux.exe`,
		Size:       17,
		ModTime:    modTime,
	}
}

func TestAnOlderBridgeNeverRestartsANewerService(t *testing.T) {
	// serve is machine-wide, bin/loomux.exe sits in one checkout of several.
	// "Anything different restarts it" would let two hosts out of two clones
	// kill each other on every call.
	now := time.Now()
	decision := bridge.Decide(stateWithBuild(now), now.Add(-time.Hour))
	if decision != bridge.Keep {
		t.Errorf("decision is %v, want Keep", decision)
	}
}

func TestANewerBridgeRestartsTheService(t *testing.T) {
	now := time.Now()
	decision := bridge.Decide(stateWithBuild(now), now.Add(time.Hour))
	if decision != bridge.Restart {
		t.Errorf("decision is %v, want Restart", decision)
	}
}

func TestTheSameBuildIsKept(t *testing.T) {
	now := time.Now()
	if decision := bridge.Decide(stateWithBuild(now), now); decision != bridge.Keep {
		t.Errorf("decision is %v, want Keep", decision)
	}
}

func TestNoStateMeansStart(t *testing.T) {
	if decision := bridge.Decide(nil, time.Now()); decision != bridge.Start {
		t.Errorf("decision is %v, want Start", decision)
	}
}

func TestRestartStopsTheOldServiceBeforeStarting(t *testing.T) {
	// A newly spawned serve cannot take serve.lock while the old one holds it.
	// Stop, wait for the lock, then spawn -- in that order, or the new one dies
	// on startup and the bridge waits sixty seconds for nothing.
	var order []string
	deps := bridge.Deps{
		Stop:     func(string) error { order = append(order, "stop"); return nil },
		WaitFree: func(string, time.Duration) error { order = append(order, "wait"); return nil },
		Spawn:    func(string) (bool, error) { order = append(order, "spawn"); return true, nil },
	}
	if err := bridge.RestartService(t.TempDir(), deps); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	want := []string{"stop", "wait", "spawn"}
	if len(order) != len(want) {
		t.Fatalf("order is %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order is %v, want %v", order, want)
		}
	}
}

func TestRestartStillSpawnsWhenTheOldServiceIsAlreadyGone(t *testing.T) {
	// A stop against a dead listener is not a failure: the goal is a free lock,
	// and a service that is already gone has reached it.
	deps := bridge.Deps{
		Stop:     func(string) error { return errors.New("connection refused") },
		WaitFree: func(string, time.Duration) error { return nil },
		Spawn:    func(string) (bool, error) { return true, nil },
	}
	if err := bridge.RestartService(t.TempDir(), deps); err != nil {
		t.Fatalf("Restart: %v", err)
	}
}

func TestARestartThatCannotGetTheLockFreeDoesNotSpawn(t *testing.T) {
	// Spawning against a held lock produces a serve that exits at once, and the
	// error the caller would see would name the wrong cause.
	spawned := false
	deps := bridge.Deps{
		Stop:     func(string) error { return nil },
		WaitFree: func(string, time.Duration) error { return errors.New("still held") },
		Spawn:    func(string) (bool, error) { spawned = true; return true, nil },
	}
	err := bridge.RestartService(t.TempDir(), deps)
	if err == nil {
		t.Fatal("a held lock came back as a successful restart")
	}
	if spawned {
		t.Error("the bridge spawned a service against a held lock")
	}
}

func TestARestartReportsAFailedSpawn(t *testing.T) {
	deps := bridge.Deps{
		Stop:     func(string) error { return nil },
		WaitFree: func(string, time.Duration) error { return nil },
		Spawn:    func(string) (bool, error) { return false, errors.New("no such program") },
	}
	if err := bridge.RestartService(t.TempDir(), deps); err == nil {
		t.Fatal("a failed spawn came back as a successful restart")
	}
}
