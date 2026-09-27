package journal_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/journal"
)

const pausedLine = `{"definition_hash":"d1","delta":{},"detail":"which colour?","effort":null,"input_hash":"h1","kind":"gate","model":null,"node":"ask","outcome":"paused","role":null,"seconds":0.1,"tokens":0,"tools":null}`

const okLine = `{"definition_hash":"d0","delta":{"n":1},"detail":null,"effort":"low","input_hash":"h0","kind":"work","model":"claude:cli-default","node":"build","outcome":"ok","role":null,"seconds":2.5,"tokens":12,"tools":"bash"}`

// An absent file reads as empty and is not an error: journal.py's `entries`
// decides this, and a session that never ran anything still starts.
func TestEntriesOfMissingFileIsEmpty(t *testing.T) {
	got, err := journal.Entries(filepath.Join(t.TempDir(), "nothing.jsonl"))
	if err != nil {
		t.Fatalf("an absent file is not an error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no entries, got %d", len(got))
	}
}

func TestEntriesReadsInOrderAndSkipsBlankLines(t *testing.T) {
	path := write(t, okLine+"\n\n"+pausedLine+"\n")

	got, err := journal.Entries(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].Node != "build" || got[1].Node != "ask" {
		t.Fatalf("order lost: %s then %s", got[0].Node, got[1].Node)
	}
	if got[0].Tools == nil || *got[0].Tools != "bash" {
		t.Fatalf("tools not read: %+v", got[0].Tools)
	}
	if got[1].Tools != nil {
		t.Fatalf("a null tools carries no value, got %q", *got[1].Tools)
	}
	if got[0].Delta["n"] != json.Number("1") {
		t.Fatalf("delta not read: %+v", got[0].Delta)
	}
}

// The reason Entries reads with UseNumber: this number has no float64, and a
// resume that hashed the rounded value would look for a payload no run had.
func TestEntriesKeepsAnIntTooLargeForAFloat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.jsonl")
	line := `{"node":"n","kind":"agent","input_hash":"h","delta":{"big":9007199254740993},` +
		`"outcome":"ok","tools":null,"effort":null,"tokens":0,"seconds":0,"detail":null,` +
		`"model":null,"role":null,"definition_hash":null}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := journal.Entries(path)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Delta["big"] != json.Number("9007199254740993") {
		t.Fatalf("delta = %#v", got[0].Delta)
	}
}

// A damaged line is named with its number and not swallowed. journal.py raises
// JournalError with exactly this shape, and session-start prints it and carries
// on with the other runs.
func TestEntriesNamesTheDamagedLine(t *testing.T) {
	path := write(t, okLine+"\n{not json\n")

	_, err := journal.Entries(path)

	if err == nil {
		t.Fatal("expected an error for a line that is not an entry")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("the error names the line number, got %v", err)
	}
}

// A blank line is skipped but still counted, so the number in an error is the
// line a human finds in the file. journal.py enumerates before it filters.
func TestEntriesCountsBlankLinesWhenNumbering(t *testing.T) {
	path := write(t, "\n\n{not json\n")

	_, err := journal.Entries(path)

	if err == nil {
		t.Fatal("expected an error for a line that is not an entry")
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("blank lines count towards the number, got %v", err)
	}
}

// A line missing any key is damage, as journal.py's Entry -- a dataclass with no
// defaults -- raised TypeError for it. Accepting it as a zero value would let a
// paused run whose last line lost its `outcome` read as "nothing waiting" --
// unannounced, and silently.
func TestEntriesNamesALineMissingAKey(t *testing.T) {
	short := `{"definition_hash":"d1","delta":{},"detail":null,"effort":null,"input_hash":"h1","kind":"gate","model":null,"node":"ask","role":null,"seconds":0.1,"tokens":0,"tools":null}`
	path := write(t, okLine+"\n"+short+"\n")

	_, err := journal.Entries(path)

	if err == nil {
		t.Fatal("a line missing a key is not an entry")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("the error names the line number, got %v", err)
	}
	if !strings.Contains(err.Error(), "outcome") {
		t.Fatalf("the error names the missing key, got %v", err)
	}
}

// A key nobody reads means the writer and this reader disagree about the
// format, which journal.py also refuses -- `Entry(**...)` raises TypeError on an
// unexpected keyword.
func TestEntriesNamesALineWithAnUnknownKey(t *testing.T) {
	extra := strings.Replace(okLine, `{"definition_hash"`, `{"mood":"brisk","definition_hash"`, 1)
	path := write(t, extra+"\n")

	_, err := journal.Entries(path)

	if err == nil {
		t.Fatal("an unknown key is a disagreement about the format")
	}
	if !strings.Contains(err.Error(), "line 1") || !strings.Contains(err.Error(), "mood") {
		t.Fatalf("the error names the line and the key, got %v", err)
	}
}

// `null` is a value and absence is damage: the key is always there and only its
// value says "nothing". A gate asks no model and plays no role, and says so
// with null.
func TestEntriesReadsANullValueAsNoValue(t *testing.T) {
	got, err := journal.Entries(write(t, pausedLine+"\n"))
	if err != nil {
		t.Fatalf("a null value is not damage: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one entry, got %d", len(got))
	}
	if got[0].Tools != nil || got[0].Effort != nil || got[0].Model != nil || got[0].Role != nil {
		t.Fatalf("a null reads as nil, got %+v", got[0])
	}
	if got[0].Detail == nil || *got[0].Detail != "which colour?" {
		t.Fatalf("the question next to those nulls is lost: %+v", got[0])
	}
}

// Types are checked as well as keys: a string where a count belongs is damage,
// although journal.py, which never checked its annotations, took one.
func TestEntriesNamesALineWithAMistypedValue(t *testing.T) {
	wrong := strings.Replace(okLine, `"tokens":12`, `"tokens":"twelve"`, 1)

	_, err := journal.Entries(write(t, wrong+"\n"))

	if err == nil {
		t.Fatal("a string where a count belongs is not an entry")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("the error names the line number, got %v", err)
	}
}

// A line of arbitrary length is read, because nothing caps what Append writes:
// `delta` is a node's own output. A reader with a ceiling the writer does not
// share would turn a legitimate run into one that cannot be announced, so
// there is no ceiling here either.
func TestEntriesReadsAVeryLongLine(t *testing.T) {
	// A whole entry, not a partial one: this test is about the line's length,
	// and a line missing keys would also fail a reader that checked for them.
	node := strings.Repeat("x", 5*1024*1024)
	line := `{"definition_hash":null,"delta":{},"detail":null,"effort":"low","input_hash":"h0","kind":"work","model":null,"node":"` +
		node + `","outcome":"ok","role":null,"seconds":2.5,"tokens":12,"tools":"bash"}`
	path := write(t, line+"\n")

	got, err := journal.Entries(path)
	if err != nil {
		t.Fatalf("a long line is not damage: %v", err)
	}
	if len(got) != 1 || got[0].Node != node {
		t.Fatalf("expected the long node name back, got %d entries", len(got))
	}
}

// A path that exists but cannot be read is an error, not an empty journal: only
// absence means "this run wrote nothing yet".
func TestEntriesReportsAnUnreadablePath(t *testing.T) {
	if _, err := journal.Entries(t.TempDir()); err == nil {
		t.Fatal("a directory is not a readable journal")
	}
}

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run.jsonl")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const agentLine = `{"definition_hash":"d1","delta":{},"detail":null,"effort":"high","input_hash":"h2","kind":"agent","model":"claude:cli-default","node":"draft","outcome":"ok","role":"author","seconds":1,"tokens":7,"tools":"edit"}`

func TestEntriesReadsModelRoleAndDefinitionHash(t *testing.T) {
	got, err := journal.Entries(write(t, agentLine+"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Model == nil || *got[0].Model != "claude:cli-default" {
		t.Fatalf("model not read: %+v", got[0].Model)
	}
	if got[0].Role == nil || *got[0].Role != "author" {
		t.Fatalf("role not read: %+v", got[0].Role)
	}
	if got[0].DefinitionHash == nil || *got[0].DefinitionHash != "d1" {
		t.Fatalf("definition_hash not read: %+v", got[0].DefinitionHash)
	}
}

// Model, role and definition_hash are required like the ten before them, so a
// line with only those ten is damage, and the error names the three in field
// order.
func TestEntriesRefusesALineWithoutModelRoleAndDefinitionHash(t *testing.T) {
	ten := `{"delta":{"n":1},"detail":null,"effort":"low","input_hash":"h0","kind":"work","node":"build","outcome":"ok","seconds":2.5,"tokens":12,"tools":"bash"}`

	_, err := journal.Entries(write(t, ten+"\n"))

	if err == nil || !strings.Contains(err.Error(), "no value for model, role, definition_hash") {
		t.Fatalf("err = %v", err)
	}
}

func TestEntriesRefusesALineWithoutRole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.jsonl")
	line := `{"node":"a","kind":"gate","input_hash":"h","delta":{},"outcome":"ok","tools":null,"effort":null,"tokens":0,"seconds":0,"detail":null,"model":null,"definition_hash":"d"}`
	if err := os.WriteFile(path, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Entries(path); err == nil || !strings.Contains(err.Error(), "no value for role") {
		t.Fatalf("err = %v", err)
	}
}

func TestAppendWritesRoleEvenWhenThereIsNone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0001.jsonl")
	if err := journal.Append(path, journal.Entry{Node: "a", Kind: "gate", InputHash: "h", Delta: map[string]any{}, Outcome: "ok"}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"role":null`) {
		t.Fatalf("line %s", raw)
	}
	role := "reviewer"
	if err := journal.Append(path, journal.Entry{Node: "b", Kind: "agent", InputHash: "h", Delta: map[string]any{}, Outcome: "ok", Role: &role}); err != nil {
		t.Fatal(err)
	}
	entries, err := journal.Entries(path)
	if err != nil || entries[1].Role == nil || *entries[1].Role != "reviewer" {
		t.Fatalf("entries %+v, %v", entries, err)
	}
}
