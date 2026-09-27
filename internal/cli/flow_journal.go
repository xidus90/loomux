package cli

import "github.com/xidus90/loomux/internal/flow/journal"

// flowJournal is a run's journal file as the runner reads and writes it. The
// runner takes an interface, not a path, so that its own tests need no files;
// this is where the path comes back.
type flowJournal struct{ path string }

func (j flowJournal) Entries() ([]journal.Entry, error) { return journal.Entries(j.path) }

func (j flowJournal) Append(entry journal.Entry) error { return journal.Append(j.path, entry) }

func (j flowJournal) Pending() (*journal.PendingGate, error) { return journal.Pending(j.path) }
