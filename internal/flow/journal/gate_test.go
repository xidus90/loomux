package journal_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/flow/journal"
)

// Only the last entry decides, and only when it paused: gate.py reads the tail
// and nothing else.
func TestPendingReadsTheLastEntry(t *testing.T) {
	path := write(t, okLine+"\n"+pausedLine+"\n")

	gate, err := journal.Pending(path)
	if err != nil {
		t.Fatal(err)
	}

	if gate == nil {
		t.Fatal("expected an open gate")
	}
	if gate.Node != "ask" || gate.Question != "which colour?" || gate.InputHash != "h1" {
		t.Fatalf("unexpected gate: %+v", gate)
	}
}

func TestPendingIsNilWhenTheRunMovedOn(t *testing.T) {
	path := write(t, pausedLine+"\n"+okLine+"\n")

	gate, err := journal.Pending(path)
	if err != nil {
		t.Fatal(err)
	}
	if gate != nil {
		t.Fatalf("a run whose last entry is ok waits for nothing, got %+v", gate)
	}
}

func TestPendingIsNilForAnEmptyJournal(t *testing.T) {
	gate, err := journal.Pending(write(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if gate != nil {
		t.Fatalf("an empty journal waits for nothing, got %+v", gate)
	}
}

// A pause with no question is not an address anyone can answer, so it is not
// reported as one. gate.py tests `last.detail is None` for this.
func TestPendingIsNilWhenThePauseAskedNothing(t *testing.T) {
	path := write(t, `{"definition_hash":"d1","delta":{},"detail":null,"effort":null,"input_hash":"h1","kind":"gate","model":null,"node":"ask","outcome":"paused","role":null,"seconds":0.1,"tokens":0,"tools":null}`+"\n")

	gate, err := journal.Pending(path)
	if err != nil {
		t.Fatal(err)
	}
	if gate != nil {
		t.Fatalf("a pause without a question is no open gate, got %+v", gate)
	}
}

func TestPendingPassesTheJournalErrorOn(t *testing.T) {
	if _, err := journal.Pending(write(t, "{not json\n")); err == nil {
		t.Fatal("a damaged journal must not read as nothing waiting")
	}
}
