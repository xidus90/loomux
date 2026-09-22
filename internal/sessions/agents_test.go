package sessions

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAgentRoundTrip(t *testing.T) {
	root := t.TempDir()
	snap := &Snapshot{Head: "h", Heads: map[string]string{}, Refs: map[string]string{"HEAD": "r"}, Remote: RemoteOK}
	if err := WriteAgent(root, "s1", "a/1", AgentFile{Snapshot: snap}); err != nil {
		t.Fatal(err)
	}
	got, ok := ReadAgent(root, "s1", "a/1")
	if !ok || !reflect.DeepEqual(got.Snapshot, snap) {
		t.Fatalf("got %+v, %v", got.Snapshot, ok)
	}
	// The id is cleaned like a session id: it may not decide where the file lands.
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents", "a1.json")); err != nil {
		t.Fatal(err)
	}
}

// null and {} are two answers: a snapshot translated from Python never took
// the local branches, one taken by loomux found none.
func TestSnapshotKeepsNilHeadsApartFromEmpty(t *testing.T) {
	root := t.TempDir()
	if err := WriteAgent(root, "s1", "a", AgentFile{Snapshot: &Snapshot{Remote: RemoteOK}}); err != nil {
		t.Fatal(err)
	}
	got, _ := ReadAgent(root, "s1", "a")
	if got.Snapshot.Heads != nil {
		t.Fatalf("heads = %#v", got.Snapshot.Heads)
	}
}

func TestReadAgentOfNothing(t *testing.T) {
	if _, ok := ReadAgent(t.TempDir(), "s1", "a"); ok {
		t.Fatal("want absent")
	}
}

func TestReadAgentOfADamagedFile(t *testing.T) {
	root := t.TempDir()
	if err := WriteAgent(root, "s1", "a", AgentFile{Finding: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents", "a.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := ReadAgent(root, "s1", "a"); ok {
		t.Fatal("want absent")
	}
}

func TestFindingsAreSortedAndSkipSnapshots(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "s1", "b", AgentFile{Finding: []string{"two"}})
	writeAgent(t, root, "s1", "a", AgentFile{Finding: []string{"one"}})
	writeAgent(t, root, "s1", "c", AgentFile{Snapshot: &Snapshot{Remote: RemoteOK}})
	dir := filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents")
	// A damaged file, a temp file left behind by a write that died, and a
	// directory whose name ends in .json: none of the three is a finding.
	if err := os.WriteFile(filepath.Join(dir, "d.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".agent-x.tmp"), []byte(`{"finding":["no"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "e.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []Finding{{AgentID: "a", Lines: []string{"one"}}, {AgentID: "b", Lines: []string{"two"}}}
	if got := Findings(root, "s1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if Findings(root, "nobody") != nil {
		t.Fatal("want nil for a session without agents")
	}
}

func TestRemoveAgent(t *testing.T) {
	root := t.TempDir()
	writeAgent(t, root, "s1", "a", AgentFile{Finding: []string{"x"}})
	if err := RemoveAgent(root, "s1", "a"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAgent(root, "s1", "a"); err != nil {
		t.Fatalf("second remove: %v", err)
	}
}

// A file that is there and will not go is not the same as one that was never
// written. A directory with something in it is the portable way to make
// os.Remove refuse.
func TestRemoveAgentReportsAFileItCannotRemove(t *testing.T) {
	root := t.TempDir()
	busy := filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents", "a.json")
	if err := os.MkdirAll(busy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(busy, "inside"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveAgent(root, "s1", "a"); err == nil {
		t.Fatal("want an error")
	}
}

func TestWriteAgentReportsADirectoryItCannotMake(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(StateDir)), []byte("a file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteAgent(root, "s1", "a", AgentFile{}); err == nil {
		t.Fatal("want an error")
	}
}

// The temp file is the one step that can fail before anything was replaced;
// the old file must then still be the one on disk.
func TestWriteAgentReportsATempFileItCannotCreate(t *testing.T) {
	stubCreateTemp(t, func(string, string) (*os.File, error) {
		return nil, errors.New("no room")
	})
	if err := WriteAgent(t.TempDir(), "s1", "a", AgentFile{}); err == nil {
		t.Fatal("want an error")
	}
}

// A write into a file just created fails only when the disk does, so the seam
// hands back a file that is already closed.
func TestWriteAgentReportsAWriteThatFails(t *testing.T) {
	stubCreateTemp(t, func(dir, pattern string) (*os.File, error) {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		return file, file.Close()
	})
	if err := WriteAgent(t.TempDir(), "s1", "a", AgentFile{}); err == nil {
		t.Fatal("want an error")
	}
}

// A close that succeeds does not undo a write that failed. The seam hands
// back a file opened for reading alone: the write refuses, the close has
// nothing to refuse, and the nil it answers must not be mistaken for the
// write's verdict.
func TestWriteAgentKeepsAFailedWriteOverASuccessfulClose(t *testing.T) {
	stubCreateTemp(t, func(dir, pattern string) (*os.File, error) {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		name := file.Name()
		if err := file.Close(); err != nil {
			return nil, err
		}
		return os.OpenFile(name, os.O_RDONLY, 0o644)
	})
	if err := WriteAgent(t.TempDir(), "s1", "a", AgentFile{}); err == nil {
		t.Fatal("want an error")
	}
}

// A directory standing where the file goes makes the rename refuse, on every
// platform.
func TestWriteAgentReportsARenameThatFails(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, filepath.FromSlash(StateDir), "s1", "agents", "a.json")
	if err := os.MkdirAll(blocker, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocker, "inside"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteAgent(root, "s1", "a", AgentFile{}); err == nil {
		t.Fatal("want an error")
	}
}

func stubCreateTemp(t *testing.T, stub func(dir, pattern string) (*os.File, error)) {
	t.Helper()
	original := createTemp
	createTemp = stub
	t.Cleanup(func() { createTemp = original })
}

func writeAgent(t *testing.T, root, sessionID, agentID string, f AgentFile) {
	t.Helper()
	if err := WriteAgent(root, sessionID, agentID, f); err != nil {
		t.Fatal(err)
	}
}
