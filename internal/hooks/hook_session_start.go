package hooks

import (
	"fmt"
	"io"

	"github.com/xidus90/loomux/internal/gitwork"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/sessions"
)

// SessionStart writes down the commit the session starts on.
//
// Never blocks. This is an announcement, so the only codes it can leave with
// are 0 and 1 -- exit 2 in payload.py's protocol (payload.py:13-15) means
// blocked, and there is nothing here to hold a turn over.
//
// Waiting flow runs are not announced here: internal/journal does not move in
// stage 1a, so the report comes back with the flow migration.
//
//coverage:exempt hosts.WriteContext cannot fail here: its claude arm writes nothing for an empty document, and the other two hosts are already refused by hosts.Read above
func SessionStart(stdin io.Reader, stdout, stderr io.Writer, root, hostName string) int {
	host, err := hosts.ParseHost(hostName)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}
	payload, err := hosts.Read(host, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}

	if err := recordBase(payload.SessionID, root); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}

	if err := hosts.WriteContext(host, stdout, nil); err != nil {
		fmt.Fprintf(stderr, "loomux hook session-start: %v\n", err)
		return ExitInternal
	}
	return ExitOK
}

// recordBase keeps the commit this session starts on, if there is one to keep.
//
// Here and nowhere else: by the time the first Stop fires, the turn has
// already run, and anything it committed would sit inside the baseline that is
// supposed to expose it.
//
// Silent in two of the three cases, which was _record_base's decision
// (session_start.py (fa3dd38):43-59, deleted in 6a7037a). Without a session id
// there is nowhere to file it,
// and outside a repository there is nothing to file -- neither is a defect of
// the project, and neither is worth a line in every session of every checkout
// that is not a git repository. The stop gate is where the absence matters,
// and that is where it is said out loud.
//
// A write that fails is the third case and it is *not* silent, because the
// Python original is not silent about it either: state.py's `write`
// (state.py:56-65) catches nothing, so an OSError there leaves `run` by
// itself. Swallowing it here would be a departure dressed up as parity.
func recordBase(sessionID, root string) error {
	if sessionID == "" {
		// hosts.Read hands a missing id and a wrongly typed one over as the
		// empty string alike; that contract is on hosts.Payload.SessionID, and
		// the type assertion keeping it is in the Claude adapter. It is the
		// `isinstance(session_id, str)` test of session_start.py (fa3dd38):52
		// with the one divergence that a literal `"session_id": ""` files
		// nothing here while Python filed it under `unnamed` (state.py:75).
		return nil
	}
	commit, err := gitwork.HeadCommit(root)
	if err != nil {
		return nil
	}
	state := sessions.ReadState(root, sessionID)
	state.Base = commit
	return sessions.WriteState(root, sessionID, state)
}
