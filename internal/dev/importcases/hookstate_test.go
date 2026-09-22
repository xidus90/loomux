package importcases

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/sessions"
	"github.com/xidus90/loomux/internal/testlock"
)

func TestParseOldSnapshot(t *testing.T) {
	got := ParseOldSnapshot("c1\tHEAD\nc1\trefs/heads/master\nHEAD\tc0\n")
	want := &sessions.Snapshot{Head: "c0", Refs: map[string]string{"HEAD": "c1", "refs/heads/master": "c1"}, Remote: sessions.RemoteOK}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	// Python stored "" for a remote that did not answer.
	if got := ParseOldSnapshot("HEAD\tc0\n"); got.Remote != sessions.RemoteUnavailable || got.Refs != nil {
		t.Fatalf("got %+v", got)
	}
}

func TestFoldHookStateMovesTheOldFiles(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, ".ultraloom", "hooks")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "s1.json"), []byte(`{"base": "b", "blocks": 2, "snapshots": {"a1": "c1\trefs/heads/master\nHEAD\tc1\n"}}`), 0o644)
	// Neither a directory nor a file the hooks never wrote is a session.
	os.MkdirAll(filepath.Join(old, "sub.json"), 0o755)
	os.WriteFile(filepath.Join(old, "notes.txt"), []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	os.WriteFile(filepath.Join(dir, ".claude", ".no-verify"), nil, 0o644)
	if err := foldHookState(dir); err != nil {
		t.Fatal(err)
	}
	if got := sessions.ReadState(dir, "s1"); got != (sessions.SessionState{Blocks: 2, Base: "b"}) {
		t.Fatalf("state %+v", got)
	}
	agent, ok := sessions.ReadAgent(dir, "s1", "a1")
	if !ok || agent.Snapshot.Head != "c1" || agent.Snapshot.Heads != nil {
		t.Fatalf("agent %+v %v", agent.Snapshot, ok)
	}
	for _, gone := range []string{".ultraloom/hooks", ".claude/.no-verify"} {
		if _, err := os.Stat(filepath.Join(dir, gone)); !os.IsNotExist(err) {
			t.Fatalf("%s: %v", gone, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".loomux", "no-verify")); err != nil {
		t.Fatal(err)
	}
}

func TestFoldHookStateWithoutOldFiles(t *testing.T) {
	if err := foldHookState(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestFoldHookStateRefusesADamagedFile(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, ".ultraloom", "hooks")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "s1.json"), []byte("{"), 0o644)
	if err := foldHookState(dir); err == nil {
		t.Fatal("want an error: a recording whose state cannot be read is no evidence")
	}
}

// TestFoldHookStateRefusesWhatItCannotRead covers the arms that only a
// filesystem in the way reaches: a held directory and a held file, and a file
// standing where a directory belongs.
func TestFoldHookStateRefusesWhatItCannotRead(t *testing.T) {
	// The old directory cannot be listed.
	dir := t.TempDir()
	old := filepath.Join(dir, ".ultraloom", "hooks")
	os.MkdirAll(old, 0o755)
	testlock.LockDir(t, old)
	if err := foldHookState(dir); err == nil {
		t.Fatal("want an error for a directory that cannot be listed")
	}

	// One recording cannot be read.
	dir = t.TempDir()
	old = filepath.Join(dir, ".ultraloom", "hooks")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "s1.json"), []byte(`{"blocks": 1}`), 0o644)
	testlock.Lock(t, filepath.Join(old, "s1.json"))
	if err := foldHookState(dir); err == nil {
		t.Fatal("want an error for a recording that cannot be read")
	}

	// The state cannot be written: .loomux is a file.
	dir = t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ultraloom", "hooks"), 0o755)
	os.WriteFile(filepath.Join(dir, ".ultraloom", "hooks", "s1.json"), []byte(`{"blocks": 1}`), 0o644)
	os.WriteFile(filepath.Join(dir, ".loomux"), nil, 0o644)
	if err := foldHookState(dir); err == nil {
		t.Fatal("want an error for a state that cannot be written")
	}

	// The agent file cannot be written: the session's directory is a file.
	dir = t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ultraloom", "hooks"), 0o755)
	os.WriteFile(filepath.Join(dir, ".ultraloom", "hooks", "s1.json"), []byte(`{"blocks": 1, "snapshots": {"a1": "HEAD\tc1\n"}}`), 0o644)
	os.MkdirAll(filepath.Join(dir, ".loomux", "state", "hooks"), 0o755)
	os.WriteFile(filepath.Join(dir, ".loomux", "state", "hooks", "s1"), nil, 0o644)
	if err := foldHookState(dir); err == nil {
		t.Fatal("want an error for an agent file that cannot be written")
	}
}

// TestFoldMarkerRefusesWhatItCannotMove covers the two arms of the move: the
// target directory cannot be made, and the rename lands on a directory.
func TestFoldMarkerRefusesWhatItCannotMove(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	os.WriteFile(filepath.Join(dir, ".claude", ".no-verify"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, ".loomux"), nil, 0o644)
	if err := foldMarker(dir); err == nil {
		t.Fatal("want an error for a target directory that cannot be made")
	}

	dir = t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	os.WriteFile(filepath.Join(dir, ".claude", ".no-verify"), nil, 0o644)
	os.MkdirAll(filepath.Join(dir, ".loomux", "no-verify", "held"), 0o755)
	if err := foldMarker(dir); err == nil {
		t.Fatal("want an error for a rename onto a directory that holds something")
	}
}

// TestTranslateWorldReportsADamagedHookState pins that the fold runs first:
// a world whose old state cannot be read is no world to translate.
func TestTranslateWorldReportsADamagedHookState(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".ultraloom", "hooks"), 0o755)
	os.WriteFile(filepath.Join(dir, ".ultraloom", "hooks", "s1.json"), []byte("{"), 0o644)
	if err := TranslateWorld(dir); err == nil {
		t.Fatal("want an error")
	}
}
