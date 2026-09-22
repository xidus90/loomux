package importcases

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/sessions"
)

// oldState is .ultraloom/hooks/<id>.json as state.py wrote it.
type oldState struct {
	Base      *string           `json:"base"`
	Blocks    int               `json:"blocks"`
	Snapshots map[string]string `json:"snapshots"`
}

// foldHookState moves the old hooks' files into loomux's layout: the session
// file under .loomux/state/hooks, each snapshot into its agent's file, and
// the marker to .loomux/no-verify. Written through internal/sessions, so the
// bytes are the ones loomux writes and reads.
//
//coverage:exempt the os.RemoveAll arm needs a directory that lists and reads and still refuses to go
func foldHookState(dir string) error {
	old := filepath.Join(dir, ".ultraloom", "hooks")
	entries, err := os.ReadDir(old)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(old, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var s oldState
		if err := json.Unmarshal(raw, &s); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		state := sessions.SessionState{Blocks: s.Blocks}
		if s.Base != nil {
			state.Base = *s.Base
		}
		if err := sessions.WriteState(dir, id, state); err != nil {
			return err
		}
		for _, agent := range slices.Sorted(maps.Keys(s.Snapshots)) {
			if err := sessions.WriteAgent(dir, id, agent, sessions.AgentFile{Snapshot: ParseOldSnapshot(s.Snapshots[agent])}); err != nil {
				return err
			}
		}
	}
	if err := os.RemoveAll(old); err != nil {
		return err
	}
	return foldMarker(dir)
}

// foldMarker moves .claude/.no-verify to where loomux looks for it.
func foldMarker(dir string) error {
	from := filepath.Join(dir, ".claude", ".no-verify")
	if _, err := os.Stat(from); os.IsNotExist(err) {
		return nil
	}
	to := filepath.Join(dir, ".loomux", "no-verify")
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Rename(from, to)
}

// ParseOldSnapshot reads what subagent_start.py stored: the lines of
// `git ls-remote origin`, and a line `HEAD\t<sha>` for the local HEAD. The
// remote lines are `<sha>\t<ref>`, the marker the other way round, which is
// how subagent_stop.py's _split tells them apart. A snapshot without one
// remote line is a remote that did not answer: Python stored "" for it.
func ParseOldSnapshot(stored string) *sessions.Snapshot {
	s := &sessions.Snapshot{Remote: sessions.RemoteUnavailable}
	for _, line := range strings.Split(stored, "\n") {
		if head, ok := strings.CutPrefix(line, "HEAD\t"); ok {
			s.Head = head
			continue
		}
		sha, ref, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		if s.Refs == nil {
			s.Refs = map[string]string{}
			s.Remote = sessions.RemoteOK
		}
		s.Refs[ref] = sha
	}
	return s
}
