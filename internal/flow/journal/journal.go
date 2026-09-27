// Package journal is a flow run's journal: one JSONL line per step. The same
// file is the log a person reads and the only source a resume reads from;
// loomux hook session-start only reads it.
package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Entry is one journal line in the JSON spelling on disk: the ten fields of
// journal.py's Entry, field for field, followed by Model, Role and
// DefinitionHash.
//
// Tools, Effort and Detail are pointers because null is a value there and the
// difference carries a decision: `Pending` reads a nil Detail as "this pause
// asked nothing", which is not the same as an empty question.
type Entry struct {
	Node      string         `json:"node"`
	Kind      string         `json:"kind"`
	InputHash string         `json:"input_hash"`
	Delta     map[string]any `json:"delta"`
	Outcome   string         `json:"outcome"`
	Tools     *string        `json:"tools"`
	Effort    *string        `json:"effort"`
	Tokens    int            `json:"tokens"`
	Seconds   float64        `json:"seconds"`
	Detail    *string        `json:"detail"`

	// Model, Role and DefinitionHash are null on a line that has none of them
	// -- a gate asks no model and plays no role -- and never absent.
	Model          *string `json:"model"`
	Role           *string `json:"role"`
	DefinitionHash *string `json:"definition_hash"`
}

// Entries is journal.py's `entries`: every line, in order.
//
// An absent file is empty and not an error, and a line that cannot be decoded
// ends the read with its number named. Both are that module's decisions: a run
// that never started has no journal, and one unreadable line makes the whole
// journal unreadable rather than being skipped -- a journal read past its
// damage would answer about a run nobody can reconstruct. The caller decides
// what that costs; session-start prints the error and carries on with the
// other runs, so one damaged file hides its own lines and no others.
//
// A well-formed object whose keys do not match `Entry` is damage too: a missing
// key or any key Entry does not know, the same way journal.py's
// `Entry(**json.loads(line))` raises TypeError for a missing or an unexpected
// keyword. Types are checked on top of that: a string where `tokens` wants a
// count is damage.
//
// The whole file is read at once and no line length is refused: nothing caps
// what Append writes, since `delta` is a node's own output.
func Entries(path string) ([]Entry, error) {
	text, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var found []Entry
	// A blank line is skipped but still counted, so the number in an error is
	// the line a reader finds in the file.
	for number, line := range strings.Split(string(text), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		entry, err := decode(line)
		if err != nil {
			return nil, fmt.Errorf("%s: line %d is not a journal entry: %w", path, number+1, err)
		}
		found = append(found, entry)
	}
	return found, nil
}

// entryKeys are the keys every line carries, in Entry's field order.
//
// The list is spelled out rather than derived from the struct tags: a derived
// list would make every field added to Entry required without anyone deciding
// it.
var entryKeys = []string{
	"node", "kind", "input_hash", "delta", "outcome",
	"tools", "effort", "tokens", "seconds", "detail",
	"model", "role", "definition_hash",
}

// decode turns one line into an Entry, or says why it is not one.
//
// The object is read twice on purpose: the first pass sees which keys are
// *present*, which unmarshalling into a struct cannot tell apart from absent.
// The difference matters because a `null` is a written value and its key is
// always there, so for every key in entryKeys absence is damage and `null` is
// not.
func decode(line string) (Entry, error) {
	var keyed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &keyed); err != nil {
		return Entry{}, err
	}

	var missing []string
	for _, key := range entryKeys {
		if _, ok := keyed[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return Entry{}, fmt.Errorf("no value for %s", strings.Join(missing, ", "))
	}

	var unknown []string
	for key := range keyed {
		if !slices.Contains(entryKeys, key) {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		// Sorted, because a map's order is not stable and an error message that
		// changed between runs would be one no test could name.
		slices.Sort(unknown)
		return Entry{}, fmt.Errorf("unknown key %s", strings.Join(unknown, ", "))
	}

	decoder := json.NewDecoder(strings.NewReader(line))
	// A delta's int field is an int, not a float64 that happens to look like
	// one. Read the ordinary way, 2^53+1 comes back as 2^53, and a resume
	// would hash a payload the run never had. Canonical already reads this
	// way, and json.Number marshals back as the literal that stood there, so
	// the writer is unchanged.
	decoder.UseNumber()
	var entry Entry
	if err := decoder.Decode(&entry); err != nil {
		return Entry{}, err
	}
	return entry, nil
}
