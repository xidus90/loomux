package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SessionState is what one session carries between two calls of the stop
// gate: how many turn ends in a row it held, the commit it measures from,
// and the tree it last found green.
//
// Base is a string and not a pointer: the empty string is never a commit,
// so both absences are one, and an absent base is written as JSON null.
// Green is the tree SHA of the last green run, or "" before the first.
type SessionState struct {
	Blocks int
	Base   string
	Green  string
}

// stateFile is the shape on disk. Pointers, because a missing key and a
// null must both arrive as absent: a file written before `base` existed
// still carries a block counter, and reading it as damaged would throw that
// counter away.
//
// The fields are alphabetical: Marshal follows field order for a struct --
// it sorts keys only for a map -- so one state is one key order whoever
// wrote it.
//
// The snapshots of the subagents lived here until stage 2c and now have a
// file each (agents.go): two subagents that start in one message run their
// hooks in parallel, and one file for both lost one of them. An older file
// still reads -- its snapshots key is ignored.
type stateFile struct {
	Base   *string `json:"base"`
	Blocks *int    `json:"blocks"`
	Green  string  `json:"green,omitempty"`
}

// ReadState is what this session left behind, or an empty state. Every
// failure answers empty and none is reported: raising would end a turn over
// a counter whose worst case is a few extra rounds.
func ReadState(root, sessionID string) SessionState {
	raw, err := os.ReadFile(statePath(root, sessionID))
	if err != nil {
		return SessionState{}
	}
	var file stateFile
	// blocks is required, base and green are not: the counter has been in the
	// file since it existed, so its absence is damage, while a file without
	// the other two is merely older.
	if err := json.Unmarshal(raw, &file); err != nil || file.Blocks == nil {
		return SessionState{}
	}
	state := SessionState{Blocks: *file.Blocks, Green: file.Green}
	if file.Base != nil {
		state.Base = *file.Base
	}
	return state
}

// WriteState keeps this state for the next call.
//
//coverage:exempt the json.Marshal err arm needs a stateFile field that does not encode; an int and strings always do
func WriteState(root, sessionID string, state SessionState) error {
	path := statePath(root, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating the directory for %s: %w", path, err)
	}
	file := stateFile{Blocks: &state.Blocks, Green: state.Green}
	if state.Base != "" {
		base := state.Base
		file.Base = &base
	}
	body, err := json.Marshal(file)
	if err != nil {
		// Unreachable: an int and strings all encode. Kept because dropping
		// the error would hide a later field that does not.
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
