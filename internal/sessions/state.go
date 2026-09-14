package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SessionState is state.py's SessionState: the counter, the snapshots and the
// base commit one session carries between two hook calls.
//
// Base is a string and not a pointer. state.py tells None from a value, and
// the empty string is never a commit any of these files could hold, so both
// absences are one. What must not collapse is the file: an absent base is
// written as JSON null, because the Python side keeps reading these files for
// as long as both sides run.
type SessionState struct {
	Blocks    int
	Snapshots map[string]string
	Base      string
}

// stateFile is the shape on disk, in the spelling state.py wrote and reads.
// Pointers, because a missing key and a null must both arrive as absent: a
// file written before `base` existed still carries a block counter, and
// reading it as damaged would throw that counter away.
type stateFile struct {
	// Alphabetical, and that is load-bearing: Marshal follows field order for
	// a struct -- it sorts keys only for a map -- and state.py writes these
	// files with sort_keys=True. So one state is one key order whichever side
	// wrote it, which is what a golden comparison over these files reads.
	//
	// Key order is all it buys. The two encoders do not agree on the bytes and
	// cannot be made to: measured on 2026-09-10, json.dumps puts a space after
	// every `:` and `,` where Marshal puts none, and their escaping diverges in
	// both directions -- for the snapshot key `ä<b`, Python's default
	// ensure_ascii=True writes `\u00e4<b` while Marshal writes the letter raw
	// and escapes the `<` as `\u003c`.
	Base      *string            `json:"base"`
	Blocks    *int               `json:"blocks"`
	Snapshots *map[string]string `json:"snapshots"`
}

// ReadState is state.py's `read`: what this session left behind, or an empty
// state.
//
// Every failure answers empty and none of them is reported. Raising would end
// a turn over a counter whose worst case is a few extra rounds, which is the
// trade state.py wrote down and this port keeps.
func ReadState(root, sessionID string) SessionState {
	raw, err := os.ReadFile(statePath(root, sessionID))
	if err != nil {
		return SessionState{}
	}
	var file stateFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return SessionState{}
	}
	// blocks and snapshots are required, base is not: the first two have been
	// in the file since it existed, so their absence is damage, while a file
	// from before `base` is merely older.
	if file.Blocks == nil || file.Snapshots == nil {
		return SessionState{}
	}
	state := SessionState{Blocks: *file.Blocks, Snapshots: *file.Snapshots}
	if file.Base != nil {
		state.Base = *file.Base
	}
	return state
}

// WriteState is state.py's `write`: keep this state for the next call.
//
//coverage:exempt the json.Marshal err arm needs a stateFile field that does not encode; an int, a string and a map of strings always do
func WriteState(root, sessionID string, state SessionState) error {
	path := statePath(root, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating the directory for %s: %w", path, err)
	}
	snapshots := state.Snapshots
	if snapshots == nil {
		// `{}` and not `null`: state.py refuses a snapshots field that is not
		// an object and would read the whole file as damaged.
		snapshots = map[string]string{}
	}
	file := stateFile{Blocks: &state.Blocks, Snapshots: &snapshots}
	if state.Base != "" {
		base := state.Base
		file.Base = &base
	}
	body, err := json.Marshal(file)
	if err != nil {
		// Unreachable: an int, a string and a map of strings all encode. Kept
		// because dropping the error would hide a later field that does not.
		return fmt.Errorf("encoding the state of session %s: %w", sessionID, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func statePath(root, sessionID string) string {
	return filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID)+".json")
}
