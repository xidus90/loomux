package journal_test

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/journal"
)

func text(s string) *string { return &s }

func TestAppendWritesOneCanonicalLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "0001.jsonl")
	entry := journal.Entry{
		Node: "draft", Kind: "agent", InputHash: "h",
		Delta:   map[string]any{"z": 1, "a": "<b>"},
		Outcome: "ok", Tools: text("edit"), Effort: text("low"), Tokens: 3, Seconds: 1.5,
		Model: text("claude:cli-default"), Role: text("author"), DefinitionHash: text("d"),
	}
	if err := journal.Append(path, entry); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"definition_hash":"d","delta":{"a":"<b>","z":1},"detail":null,"effort":"low","input_hash":"h",` +
		`"kind":"agent","model":"claude:cli-default","node":"draft","outcome":"ok","role":"author","seconds":1.5,"tokens":3,"tools":"edit"}` + "\n"
	if string(got) != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

// What Append writes, Entries reads back -- including the null a gate writes
// for its model.
func TestAppendThenEntriesRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0002.jsonl")
	gate := journal.Entry{Node: "approve", Kind: "gate", InputHash: "h1", Delta: map[string]any{},
		Outcome: "paused", Detail: text("ship it?"), DefinitionHash: text("d1")}
	agent := journal.Entry{Node: "draft", Kind: "agent", InputHash: "h2", Delta: map[string]any{"n": 2},
		Outcome: "ok", Tokens: 9, Seconds: 0.25, Model: text("claude:claude-opus-5"), DefinitionHash: text("d2")}
	for _, entry := range []journal.Entry{gate, agent} {
		if err := journal.Append(path, entry); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"model":null`) {
		t.Fatalf("a gate writes its model as null: %s", raw)
	}
	got, err := journal.Entries(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Model != nil || *got[0].Detail != "ship it?" ||
		*got[1].Model != "claude:claude-opus-5" || got[1].Delta["n"] != json.Number("2") {
		t.Fatalf("round trip lost something: %+v", got)
	}
}

// Refused before the file is touched, so a bad entry leaves no half line.
func TestAppendRefusesAnUnserializableEntryBeforeTouchingTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs", "0003.jsonl")
	err := journal.Append(path, journal.Entry{Node: "draft", Delta: map[string]any{"x": math.NaN()}})
	if err == nil || !strings.Contains(err.Error(), "draft") {
		t.Fatalf("want an error naming the node, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Dir(path)); !os.IsNotExist(statErr) {
		t.Fatalf("nothing may be created for a refused entry, stat says %v", statErr)
	}
}

func TestAppendReportsAParentThatIsAFile(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := journal.Append(filepath.Join(blocker, "0001.jsonl"), journal.Entry{Node: "n"})
	if err == nil || !strings.Contains(err.Error(), "creating") {
		t.Fatalf("got %v", err)
	}
}

func TestAppendReportsAPathThatIsADirectory(t *testing.T) {
	err := journal.Append(t.TempDir(), journal.Entry{Node: "n"})
	if err == nil || !strings.Contains(err.Error(), "opening") {
		t.Fatalf("got %v", err)
	}
}
