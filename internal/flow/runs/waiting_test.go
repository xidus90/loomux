package runs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/runs"
)

// run writes a run's marker and, with a question, a journal paused at a gate;
// without one the journal ends in a finished node.
func run(t *testing.T, root, id, question string) {
	t.Helper()
	if err := runs.WriteMarker(runs.MarkerPath(root, id), runs.Marker{Flow: "example", Origin: "bundled"}); err != nil {
		t.Fatal(err)
	}
	entry := journal.Entry{Node: "approve", Kind: "gate", InputHash: "h", Delta: map[string]any{}, Outcome: "paused", Detail: &question}
	if question == "" {
		entry = journal.Entry{Node: "draft", Kind: "work", InputHash: "h", Delta: map[string]any{}, Outcome: "ok"}
	}
	if err := journal.Append(runs.JournalPath(root, id), entry); err != nil {
		t.Fatal(err)
	}
}

func TestWaitingRunsListsOnlyTheRunsAtAGateInRunOrder(t *testing.T) {
	root := t.TempDir()
	run(t, root, "0002", "Plan ok?")
	run(t, root, "0001", "Ship it?")
	run(t, root, "0003", "")
	got, err := runs.WaitingRuns(root)
	if err != nil || len(got) != 2 {
		t.Fatalf("WaitingRuns = %+v, %v, want two runs", got, err)
	}
	if got[0].ID != "0001" || got[0].Gate.Question != "Ship it?" || got[0].Gate.Node != "approve" || got[0].Marker == nil || got[0].Marker.Flow != "example" {
		t.Errorf("first run = %+v", got[0])
	}
	if got[1].ID != "0002" {
		t.Errorf("second run = %s, want 0002", got[1].ID)
	}
}

func TestWaitingRunsOfAProjectWithoutRunsIsEmpty(t *testing.T) {
	if got, err := runs.WaitingRuns(t.TempDir()); err != nil || len(got) != 0 {
		t.Fatalf("WaitingRuns = %+v, %v, want none", got, err)
	}
}

func TestWaitingRunsSaysWhyAJournalOrAMarkerDoesNotRead(t *testing.T) {
	root := t.TempDir()
	run(t, root, "0001", "Ship it?")
	if err := os.WriteFile(runs.JournalPath(root, "0002"), []byte("{not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runs.MarkerPath(root, "0001"), []byte("example\norigin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := runs.WaitingRuns(root)
	if err != nil || len(got) != 2 {
		t.Fatalf("WaitingRuns = %+v, %v, want two runs", got, err)
	}
	if got[0].MarkerErr == nil || got[0].Gate == nil {
		t.Errorf("0001 = %+v, want the gate and a marker that did not read", got[0])
	}
	if got[1].Err == nil || got[1].Gate != nil || got[1].Marker != nil || got[1].MarkerErr != nil {
		t.Errorf("0002 = %+v, want a journal error and no marker", got[1])
	}
}

func TestWaitingRunsLeavesFoldersAndOtherFilesAlone(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(runs.Dir))
	if err := os.MkdirAll(filepath.Join(dir, "0001.jsonl"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0002.flow"), []byte("example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := runs.WaitingRuns(root); err != nil || len(got) != 0 {
		t.Fatalf("WaitingRuns = %+v, %v, want none", got, err)
	}
}

func TestWaitingRunsNamesARunsPathThatIsAFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash(runs.Dir))
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runs.WaitingRuns(root); err == nil || err.Error() != "it is a file" {
		t.Fatalf("err = %v, want it is a file", err)
	}
}
