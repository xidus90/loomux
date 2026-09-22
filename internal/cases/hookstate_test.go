package cases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/sessions"
)

func TestCompareStateReadsBaseAndBlocks(t *testing.T) {
	want, got := t.TempDir(), t.TempDir()
	sessions.WriteState(want, "s1", sessions.SessionState{Blocks: 1, Base: "b"})
	sessions.WriteState(got, "s1", sessions.SessionState{Blocks: 1, Base: "b", Green: "ignored"})
	if m := compareState(want, got); len(m) != 0 {
		t.Fatalf("mismatches %v", m)
	}
	sessions.WriteState(got, "s1", sessions.SessionState{Blocks: 2, Base: "c"})
	if m := compareState(want, got); len(m) != 2 {
		t.Fatalf("mismatches %v", m)
	}
}

func TestCompareStateOfAMissingSession(t *testing.T) {
	want := t.TempDir()
	sessions.WriteState(want, "s1", sessions.SessionState{Blocks: 1})
	if m := compareState(want, t.TempDir()); len(m) != 1 {
		t.Fatalf("mismatches %v", m)
	}
}

func TestCompareFindings(t *testing.T) {
	got := t.TempDir()
	sessions.WriteAgent(got, "s1", "a1", sessions.AgentFile{Finding: []string{"origin x is new at c"}})
	sessions.WriteAgent(got, "s1", "a2", sessions.AgentFile{Snapshot: &sessions.Snapshot{}})
	if m := compareFindings([]byte("subagent a1: origin x is new at c\n"), got); len(m) != 0 {
		t.Fatalf("mismatches %v", m)
	}
	if m := compareFindings([]byte("subagent a1: something else\n"), got); len(m) != 1 {
		t.Fatalf("mismatches %v", m)
	}
	if m := compareFindings(nil, t.TempDir()); len(m) != 0 {
		t.Fatalf("mismatches %v", m)
	}
	os.MkdirAll(filepath.Join(got, ".loomux", "state", "hooks", "s1", "agents", "x.json"), 0o755)
	if m := compareFindings([]byte("subagent a1: origin x is new at c\n"), got); len(m) != 0 {
		t.Fatalf("a directory is no agent file: %v", m)
	}
}

// TestSessionIDsSkipsWhatIsNoSessionFile pins that only the files the hooks
// wrote name a session: a directory beside them is a session's agents, and a
// file without the suffix is nobody's state.
func TestSessionIDsSkipsWhatIsNoSessionFile(t *testing.T) {
	root := t.TempDir()
	sessions.WriteState(root, "s1", sessions.SessionState{Blocks: 1})
	dir := filepath.Join(root, filepath.FromSlash(sessions.StateDir))
	os.MkdirAll(filepath.Join(dir, "s1"), 0o755)
	os.WriteFile(filepath.Join(dir, "notes.txt"), nil, 0o644)
	if ids := sessionIDs(root); len(ids) != 1 || ids[0] != "s1" {
		t.Fatalf("ids %v", ids)
	}
}
