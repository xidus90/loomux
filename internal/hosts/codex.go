package hosts

import (
	"errors"
	"fmt"
	"io"
)

// ErrNoAdapter is returned by a host arm that exists as a promise and not as
// an implementation.
//
// Wrapped rather than returned bare, so a caller can tell "this host has no
// adapter yet" apart from "this host does not exist" while the message still
// names which host and what is missing.
var ErrNoAdapter = errors.New("no adapter for this host")

// readCodex is the seam and not an adapter.
//
// Codex is not installed on the development machine, and its hook contract
// could not be computed from there: what is documented is that it has a
// `hooks/hooks.json` mechanism -- suppressed by an empty `hooks` object in
// `.codex-plugin/plugin.json` -- and that it "runs no session-start hook"
// (superpowers/6.3.0/docs/porting-to-a-new-harness.md:243 and the harness
// table at :788). That same page warns that a hook *system* is not a
// session-start *event*: one harness carried the string SessionStart in its
// binary while firing only pre/post-tool and stop.
//
// So this refuses. A guessed adapter would look exactly like a working one,
// and ultra-brain has already paid for that shape once: on 2026-09-06 its
// deny envelope went unread on Antigravity and a probe file landed on disk
// anyway, with only exit 2 having any effect
// (docs/.superpowers/specs/2026-09-10-go-hooks-drei-hosts-design.md:104-108).
func readCodex(io.Reader) (Payload, error) {
	return Payload{}, fmt.Errorf("codex: %w -- its hook contract is unmeasured, see the design", ErrNoAdapter)
}

func writeCodexContext(io.Writer, string, []string) error {
	return fmt.Errorf("codex: %w -- its hook contract is unmeasured, see the design", ErrNoAdapter)
}

// readAntigravity is a seam too, because no payload was ever recorded.
//
// The measurement of 2026-09-10 against agy 1.1.24 set out to capture Stop and
// PreInvocation payloads and captured none: in print mode neither event fires
// at all, across three configurations, and the counter-probe by hand ruled out
// the script, the path, the schema and the workspace trust as causes
// (docs/.superpowers/specs/2026-09-10-antigravity-hook-messung.md, finding 4).
//
// Forwarding to readClaude would therefore assert a shape nobody has seen. It
// may well be the same object-shaped payload with the event under another key,
// but "may well be" is what a guessed adapter is made of, and it would look
// exactly like a working one.
func readAntigravity(io.Reader) (Payload, error) {
	return Payload{}, fmt.Errorf("antigravity: %w -- its payload shape is unmeasured, no Stop or PreInvocation payload was ever recorded", ErrNoAdapter)
}

// writeAntigravityContext is the arm that waits on the same measurement.
//
// Whether `PreInvocation` can write into the model's context at all is
// question 2 of the design's three, and it is unanswered for the reason above:
// the event did not fire, so nothing could be observed reaching the model.
// Answering it needs an interactive session or one of the two permission
// relaxations agy itself names, both of which are the user's decision. The
// fallback if the answer turns out to be no is prose in GEMINI.md rather than
// a hook.
func writeAntigravityContext(io.Writer, string, []string) error {
	return fmt.Errorf("antigravity: whether PreInvocation can write context is unmeasured: %w", ErrNoAdapter)
}
