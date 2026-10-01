package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionState is what one session carries between two calls of the stop
// gate: how many turn ends in a row it held, the commit it measures from,
// and the tree it last found green.
//
// Base is a string and not a pointer: the empty string is never a commit,
// so both absences are one, and an absent base is written as JSON null.
// Green is the tree SHA of the last green run, or "" before the first.
// Seen is the last tree found red only in lanes in probation, nil without one.
type SessionState struct {
	Blocks int
	Base   string
	Green  string
	Seen   *Seen
}

// Seen is a tree the stop gate found red only in lanes in probation: not
// green, so the base stays, and no block. It is kept so that the same tree
// under the same HEAD starts no tool again and only says Report again. HEAD
// is part of it because the graph lane judges against HEAD. Armed is the
// armed lanes the chain ran under: a project that ignores .loomux keeps the
// file out of the tree, so a lane armed by hand changes no tree and has to
// end the stand by itself. At is when the chain ran, which is how the newest
// stand of several sessions is told.
type Seen struct {
	Tree   string    `json:"tree"`
	Head   string    `json:"head"`
	Armed  []string  `json:"armed"`
	Report string    `json:"report"`
	At     time.Time `json:"at"`
}

// LastSeen is the newest stand any session of root left behind, by the time
// its chain ran. Session start reads it for a session that has no state of
// its own yet. A file that does not read is passed by, as ReadState passes
// it: no JSON, or no block counter.
func LastSeen(root string) (Seen, bool) {
	dir := filepath.Join(root, filepath.FromSlash(StateDir))
	entries, _ := os.ReadDir(dir)
	var newest Seen
	found := false
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		// A directory named like a session file fails here and is passed by.
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var file stateFile
		if json.Unmarshal(raw, &file) != nil || file.Blocks == nil || file.Seen == nil {
			continue
		}
		if !found || file.Seen.At.After(newest.At) {
			newest, found = *file.Seen, true
		}
	}
	return newest, found
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
	Seen   *Seen   `json:"seen,omitempty"`
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
	state := SessionState{Blocks: *file.Blocks, Green: file.Green, Seen: file.Seen}
	if file.Base != nil {
		state.Base = *file.Base
	}
	return state
}

// WriteState keeps this state for the next call.
func WriteState(root, sessionID string, state SessionState) error {
	path := statePath(root, sessionID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating the directory for %s: %w", path, err)
	}
	file := stateFile{Blocks: &state.Blocks, Green: state.Green, Seen: state.Seen}
	if state.Base != "" {
		base := state.Base
		file.Base = &base
	}
	body, err := json.Marshal(file)
	if err != nil {
		// The time of a seen stand encodes only within the years 0 to 9999.
		return fmt.Errorf("encoding the state of session %s: %w", sessionID, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func statePath(root, sessionID string) string {
	return sessionFile(root, sessionID, ".json")
}

// sessionFile is one of the files beside the others in StateDir, named by the
// session and ext.
func sessionFile(root, sessionID, ext string) string {
	return filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID)+ext)
}
