package journal_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/flow/journal"
)

func TestLookupFindsTheLatestMatch(t *testing.T) {
	entries := []journal.Entry{
		{Node: "draft", InputHash: "h1", Outcome: "ok", Tokens: 1},
		{Node: "draft", InputHash: "h1", Outcome: "ok", Tokens: 2},
		{Node: "draft", InputHash: "h1", Outcome: "error", Tokens: 3},
		{Node: "draft", InputHash: "h2", Outcome: "ok", Tokens: 4},
		{Node: "review", InputHash: "h1", Outcome: "ok", Tokens: 5},
	}
	// With an outcome the search skips a later decoy instead of stopping at it:
	// a visit limit or a pause writes non-ok entries under a key that succeeded.
	if got, ok := journal.Lookup(entries, "draft", "h1", "ok"); !ok || got.Tokens != 2 {
		t.Fatalf("latest ok: got %+v, %v", got, ok)
	}
	if got, ok := journal.Lookup(entries, "draft", "h1", ""); !ok || got.Tokens != 3 {
		t.Fatalf("latest of any outcome: got %+v, %v", got, ok)
	}
	if _, ok := journal.Lookup(entries, "draft", "h3", ""); ok {
		t.Fatal("an unknown input has no entry")
	}
	if _, ok := journal.Lookup(entries, "review", "h1", "paused"); ok {
		t.Fatal("no paused entry for review")
	}
}
