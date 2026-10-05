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
// and that shape has already been paid for once: on 2026-09-06 a
// deny envelope went unread on Antigravity and a probe file landed on disk
// anyway, with only exit 2 having any effect
// (docs/.superpowers/specs/2026-09-10-go-hooks-drei-hosts-design.md:104-108).
func readCodex(io.Reader) (Payload, error) {
	return Payload{}, fmt.Errorf("codex: %w -- its hook contract is unmeasured, see the design", ErrNoAdapter)
}

func writeCodexContext(io.Writer, string, []string) error {
	return fmt.Errorf("codex: %w -- its hook contract is unmeasured, see the design", ErrNoAdapter)
}
