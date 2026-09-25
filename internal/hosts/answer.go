package hosts

import (
	"encoding/json"
	"io"
	"strings"
)

// exitOK and exitDenied are the hooks package's ExitOK and ExitDenied, which
// this package cannot import.
const (
	exitOK     = 0
	exitDenied = 2
)

// Answer turns what a hook produced -- its exit code, its stdout and what it
// told stderr -- into what the host reads, and answers the exit code to end
// with. event is the subcommand's name (`stop`, `post-tool-use`, ...).
//
// Claude Code and Codex read the codes as the hooks write them, so their
// output passes through untouched. Antigravity reads a code differently per
// event (measured with agy 1.2.8 and 1.2.11, 2026-09-25): pre-tool-use's exit 2
// refuses the call, and post-tool-use's exit 2 hands stderr to the model as a
// warning without aborting, so both stay as they are. A stop is held by
// `continue` on stdout with exit 0. Every other non-zero code ends with 0,
// its message on stderr for agy's log: 1 is a hook that could not judge and
// holds nothing.
//
// A stdout that cannot be written leaves the code alone: the host sees what
// it would have seen of a hook that wrote its answer itself and lost it.
func Answer(host Host, event string, w io.Writer, code int, out []byte, reason string) int {
	if host != HostAntigravity || event == "pre-tool-use" {
		_, _ = w.Write(out)
		return code
	}
	switch {
	case event == "post-tool-use":
		// What post-edit writes on stdout is Claude's additionalContext,
		// which agy does not read. Its guide names {} as a PostToolUse's
		// answer, and an empty stdout passes as well (measured with agy
		// 1.2.11 on 2026-09-25, no hook error in its log).
		if code == exitDenied {
			return exitDenied
		}
		return exitOK
	case code == exitDenied && event == "stop":
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false)
		err := encoder.Encode(struct {
			Decision string `json:"decision"`
			Reason   string `json:"reason"`
		}{"continue", strings.TrimSpace(reason)})
		if err != nil {
			// Nothing reached the host; a failed command is the nearest
			// thing to the hold it was meant to be.
			return exitDenied
		}
		return exitOK
	}
	_, _ = w.Write(out)
	return exitOK
}
