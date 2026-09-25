package sessions

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Remote says whether the remote answered when a snapshot was taken.
const (
	RemoteOK          = "ok"
	RemoteUnavailable = "unavailable"
	// RemoteNone: no remote of that name is configured, so there was nothing
	// to read.
	RemoteNone = "none"
)

// Snapshot is where origin, the local branches and HEAD stood when a
// subagent started. Heads is nil in a snapshot that never looked at the
// local branches -- one translated from the Python hooks -- and empty in one
// that found none; the comparison only reads branches from the second.
// Elsewhere names the branches then checked out in another worktree of the
// repository, which the comparison leaves out: they move with that
// worktree's session, not with this one's subagent.
type Snapshot struct {
	Elsewhere []string          `json:"elsewhere,omitempty"`
	Head      string            `json:"head"`
	Heads     map[string]string `json:"heads"`
	Refs      map[string]string `json:"refs"`
	Remote    string            `json:"remote"`
}

// AgentFile is one subagent's file: its snapshot while it runs, what it
// changed once it stopped. The lines carry no prefix; whoever shows them
// names the agent.
type AgentFile struct {
	Snapshot *Snapshot `json:"snapshot,omitempty"`
	Finding  []string  `json:"finding,omitempty"`
}

// Finding is what one stopped subagent left for the main agent.
type Finding struct {
	AgentID string
	Lines   []string
}

func agentDir(root, sessionID string) string {
	return filepath.Join(root, filepath.FromSlash(StateDir), safeName(sessionID), "agents")
}

func agentPath(root, sessionID, agentID string) string {
	return filepath.Join(agentDir(root, sessionID), safeName(agentID)+".json")
}

// ReadAgent is one subagent's file, or false when there is none it can read.
func ReadAgent(root, sessionID, agentID string) (AgentFile, bool) {
	return readAgentFile(agentPath(root, sessionID, agentID))
}

func readAgentFile(path string) (AgentFile, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return AgentFile{}, false
	}
	var file AgentFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return AgentFile{}, false
	}
	return file, true
}

// WriteAgent replaces one subagent's file in one step: a temp file in the
// same directory, then a rename. A background subagent can stop while the
// main agent's stop gate reads the directory, and a half-written file would
// read as none.
//
//coverage:exempt the json.Marshal err arm needs a field that does not encode; strings, maps and slices of strings always do
func WriteAgent(root, sessionID, agentID string, f AgentFile) error {
	dir := agentDir(root, sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	body, err := json.Marshal(f)
	if err != nil {
		// Unreachable: strings, maps of strings and slices of strings all
		// encode. Kept because dropping it would hide a later field that
		// does not.
		return fmt.Errorf("encoding the file of agent %s: %w", agentID, err)
	}
	tmp, err := createTemp(dir, ".agent-*.tmp")
	if err != nil {
		return fmt.Errorf("writing into %s: %w", dir, err)
	}
	// Both are asked before either is reported: a close that fails after a
	// write that did not is still a file nobody may rename into place, and
	// leaving the handle open would keep the temp file undeletable.
	_, werr := tmp.Write(body)
	cerr := tmp.Close()
	if werr == nil {
		werr = cerr
	}
	if werr != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("writing %s: %w", tmp.Name(), werr)
	}
	path := agentPath(root, sessionID, agentID)
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("replacing %s: %w", path, err)
	}
	return nil
}

// createTemp is a seam: a write into a file just created fails only when the
// disk does.
var createTemp = os.CreateTemp

// RemoveAgent deletes one subagent's file; one that is not there is gone.
func RemoveAgent(root, sessionID, agentID string) error {
	return removeFile(agentPath(root, sessionID, agentID), "removing")
}

// Findings are the finding lines of the subagent files, sorted by agent id.
// What decides is the finding list alone: a file without lines -- a subagent
// that runs and has parked nothing -- is skipped, and a file with lines is
// delivered whether or not it also holds a snapshot, which it does when a
// subagent continued under the same id starts again beside an unread
// finding. A file that cannot be read is skipped, like a damaged session
// file.
func Findings(root, sessionID string) []Finding {
	dir := agentDir(root, sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Finding
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		file, ok := readAgentFile(filepath.Join(dir, name))
		if !ok || len(file.Finding) == 0 {
			continue
		}
		out = append(out, Finding{AgentID: strings.TrimSuffix(name, ".json"), Lines: file.Finding})
	}
	// os.ReadDir already sorts by name. The sort stays because this order is
	// the order `stop` prints the findings in, and that may not hang on a
	// property of ReadDir nobody here asked for.
	slices.SortFunc(out, func(a, b Finding) int { return strings.Compare(a.AgentID, b.AgentID) })
	return out
}
