package hooks

import (
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/sessions"
)

// writeAgent is the seam for a file that cannot be written where a file was
// read a moment ago: both ends of the stop hook touch one path, so no world
// can hold that open on every platform.
var writeAgent = sessions.WriteAgent

// SubagentStart writes down where origin, the local branches and HEAD stand
// before a subagent runs. It never blocks: 0, or 1 for a payload it cannot
// file.
func SubagentStart(stdin io.Reader, stderr io.Writer, root, hostName string) int {
	payload, root, ok := subagentPayload(stdin, stderr, root, hostName, "subagent-start")
	if !ok {
		return ExitInternal
	}
	snap := takeSnapshot(root)
	file := sessions.AgentFile{Snapshot: &snap}
	// A subagent continued under the same id parked a finding when it last
	// stopped, and the main agent's turn has not ended since. The new
	// snapshot joins it rather than replacing the file: a finding nobody has
	// read is not this hook's to drop.
	if before, found := sessions.ReadAgent(root, payload.SessionID, payload.AgentID); found {
		file.Finding = before.Finding
	}
	if err := writeAgent(root, payload.SessionID, payload.AgentID, file); err != nil {
		fmt.Fprintf(stderr, "loomux hook subagent-start: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

// SubagentStop compares against the snapshot and leaves what moved for the
// main agent's stop gate. Its own stdout and exit 2 would reach the subagent,
// not the main agent -- the only one who should hear about it.
func SubagentStop(stdin io.Reader, stderr io.Writer, root, hostName string) int {
	payload, root, ok := subagentPayload(stdin, stderr, root, hostName, "subagent-stop")
	if !ok {
		return ExitInternal
	}
	before, found := sessions.ReadAgent(root, payload.SessionID, payload.AgentID)
	if !found || before.Snapshot == nil {
		// Silent: this hook may have been switched on midway, and nobody
		// would have read the line Python printed here.
		return ExitOK
	}
	// A finding is added to, never dropped: what an earlier run of this id
	// left is still unread, and its lines come first because they happened
	// first. A line both runs report stays twice -- both runs did report it,
	// and the main agent reads two turns of history, not one list.
	lines := append(before.Finding, compareSnapshots(root, *before.Snapshot, takeSnapshot(root))...)
	var err error
	if len(lines) == 0 {
		err = sessions.RemoveAgent(root, payload.SessionID, payload.AgentID)
	} else {
		err = writeAgent(root, payload.SessionID, payload.AgentID, sessions.AgentFile{Finding: lines})
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook subagent-stop: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

func subagentPayload(stdin io.Reader, stderr io.Writer, root, hostName, event string) (hosts.Payload, string, bool) {
	host, err := hosts.ParseHost(hostName)
	if err == nil {
		var p hosts.Payload
		if p, err = hosts.Read(host, stdin); err == nil {
			if missing := missingFields(p); missing != "" {
				fmt.Fprintf(stderr, "loomux hook %s: payload carries no %s\n", event, missing)
				return hosts.Payload{}, "", false
			}
			if abs, err := filepath.Abs(root); err == nil {
				root = abs
			}
			return p, root, true
		}
	}
	fmt.Fprintf(stderr, "loomux hook %s: %v\n", event, err)
	return hosts.Payload{}, "", false
}

// missingFields names the ids a subagent payload lacks, "" when it has both.
func missingFields(p hosts.Payload) string {
	var missing []string
	if p.SessionID == "" {
		missing = append(missing, "session_id")
	}
	if p.AgentID == "" {
		missing = append(missing, "agent_id")
	}
	return strings.Join(missing, " and ")
}

// takeSnapshot never fails: an unreachable remote is a fact about the
// machine, not a finding about the subagent, and it is written down as such.
// A repository without origin is written down apart from one whose origin
// did not answer: the first has nothing to report, the second may hide a push.
// Heads is a map even when git refuses to name the branches -- nil is what a
// snapshot translated from Python carries, and it would silence every branch
// line at the other end.
func takeSnapshot(root string) sessions.Snapshot {
	snap := sessions.Snapshot{Remote: sessions.RemoteOK, Heads: map[string]string{}}
	if !gitwork.HasRemote(root, "origin") {
		snap.Remote = sessions.RemoteNone
	} else if refs, err := gitwork.LsRemote(root, "origin"); err == nil {
		snap.Refs = refs
	} else {
		snap.Remote = sessions.RemoteUnavailable
	}
	if heads, elsewhere, err := gitwork.LocalBranches(root); err == nil {
		snap.Heads, snap.Elsewhere = heads, elsewhere
	}
	snap.Head, _ = gitwork.Head(root)
	return snap
}

// compareSnapshots names every ref of origin and every local branch that
// moved, appeared or vanished while the subagent ran, then the commits HEAD
// and the moved branches gained, each once. It observes what moved, not who
// moved it: another session's push reads here as well, and no filter can
// tell it apart.
//
// A branch checked out in another worktree at either end is left out
// entirely. Local branches are shared by every worktree, and such a branch
// moves with the session working there. That includes a subagent started in a
// worktree of its own; the host's answer for that subagent names its
// worktree and branch.
func compareSnapshots(root string, before, after sessions.Snapshot) []string {
	skip := slices.Concat(before.Elsewhere, after.Elsewhere)
	before.Heads = withoutRefs(before.Heads, skip)
	after.Heads = withoutRefs(after.Heads, skip)
	var lines []string
	switch {
	case before.Remote == sessions.RemoteNone && after.Remote == sessions.RemoteNone:
		// No origin at either end: nothing to push to, nothing to say.
	case before.Remote != sessions.RemoteOK:
		lines = append(lines, "remote could not be read at start")
	case after.Remote != sessions.RemoteOK:
		lines = append(lines, "remote could not be read at stop")
	default:
		lines = append(lines, refLines("origin ", before.Refs, after.Refs, nil)...)
	}
	short := func(ref string) string { return strings.TrimPrefix(ref, "refs/heads/") }
	if before.Heads != nil {
		lines = append(lines, refLines("branch ", before.Heads, after.Heads, short)...)
	}
	var ranges [][2]string
	if before.Head != "" && after.Head != "" && before.Head != after.Head {
		ranges = append(ranges, [2]string{before.Head, after.Head})
	}
	if before.Heads != nil {
		for _, ref := range slices.Sorted(maps.Keys(before.Heads)) {
			was, now := before.Heads[ref], after.Heads[ref]
			if now != "" && was != now {
				ranges = append(ranges, [2]string{was, now})
			}
		}
	}
	seen := map[string]bool{}
	for _, r := range ranges {
		commits, _ := gitwork.LogOneline(root, r[0], r[1])
		for _, c := range commits {
			if !seen[c] {
				seen[c] = true
				lines = append(lines, "new commit "+c)
			}
		}
	}
	return lines
}

// withoutRefs is heads less the refs named in skip. A nil table stays nil:
// it is a snapshot that never looked at the branches, not one that found
// none.
func withoutRefs(heads map[string]string, skip []string) map[string]string {
	if heads == nil {
		return nil
	}
	kept := maps.Clone(heads)
	for _, ref := range skip {
		delete(kept, ref)
	}
	return kept
}

// refLines compares two ref tables in sorted order, both directions: a ref
// deleted is as much a push as one created.
func refLines(label string, old, now map[string]string, name func(string) string) []string {
	if name == nil {
		name = func(ref string) string { return ref }
	}
	all := map[string]bool{}
	for ref := range old {
		all[ref] = true
	}
	for ref := range now {
		all[ref] = true
	}
	var lines []string
	for _, ref := range slices.Sorted(maps.Keys(all)) {
		was, is := old[ref], now[ref]
		switch {
		case was == is:
		case was == "":
			lines = append(lines, fmt.Sprintf("%s%s is new at %s", label, name(ref), is))
		case is == "":
			lines = append(lines, fmt.Sprintf("%s%s is gone; it was %s", label, name(ref), was))
		default:
			lines = append(lines, fmt.Sprintf("%s%s moved %s -> %s", label, name(ref), was, is))
		}
	}
	return lines
}
