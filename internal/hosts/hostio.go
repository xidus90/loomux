// Package hosts is the seam between a hook's work and the host that calls it.
//
// One normalised payload in, one normalised answer out. The core packages see
// these two types and never a JSON envelope, so a second host costs an adapter
// and not a second copy of the hook.
//
// The host arrives as a flag and is not guessed from the payload. Guessing was
// the first design -- `tool_input.file_path` for Claude against `TargetFile`
// for Antigravity, the way `brain guard` does it -- and it cannot work here:
// the four events these hooks answer are not tool events at all. SessionStart,
// Stop and the subagent pair carry no `tool_input`, so there is nothing to
// recognise. The flag is not a second place for the same truth either, because
// loomux init writes both host files from one table and knows the host as it builds
// the command.
package hosts

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Host is which harness is calling.
type Host string

const (
	HostClaude      Host = "claude"
	HostAntigravity Host = "antigravity"
	HostCodex       Host = "codex"
)

// ErrNoRoot is returned when no `.loomux/config.toml` stands above the
// starting directory.
var ErrNoRoot = errors.New("no .loomux/config.toml above this directory")

// ParseHost turns the flag's value into a Host, or refuses it.
//
// Refused and not defaulted: the value is written by loomux init, so one nobody
// knows means the two have drifted apart. Answering a hook in the wrong shape
// is worse than refusing to answer.
func ParseHost(name string) (Host, error) {
	switch Host(name) {
	case HostClaude:
		return HostClaude, nil
	case HostAntigravity:
		return HostAntigravity, nil
	case HostCodex:
		return HostCodex, nil
	}
	return "", fmt.Errorf("unknown host %q: expected claude, antigravity or codex", name)
}

// Payload is what a hook needs to know, whichever host asked.
//
// The fields the hooks read, and no more: the tool and the paths a write
// barrier reads arrive when the stage that needs them does, because a field
// nobody reads has no test holding it in place.
type Payload struct {
	// Event is the host's own name for what fired. Carried for the adapters
	// and read by no hook today -- session-start is dispatched by the
	// subcommand, not by the payload -- so it exists to be normalised once
	// rather than twice when a second event arrives.
	Event string

	// SessionID is where this session's state is filed, or the empty string.
	//
	// One string for two cases, and every adapter owes the caller this: a
	// missing id and an id of the wrong type both arrive as "". Deciding
	// between them is not the adapter's business -- what counts as a usable
	// id is the hook's question, and `recordBase` in internal/hooks answers it by
	// filing nothing. An adapter that refused a mistyped id instead would
	// exit 1 where the Python hook exited 0.
	SessionID string

	// AgentID and AgentType name the subagent a SubagentStart or
	// SubagentStop fired for, "" on every other event. Measured with Claude
	// Code 2.1.276 (stage 2c, ba6bed7; the payloads are in
	// testdata/cases/2c-payloads/): SubagentStart and SubagentStop carry the
	// same agent_id, both carry the main agent's session_id -- which files a
	// subagent's snapshot under the session whose stop gate delivers its
	// finding -- and both carry agent_type. No hook reads AgentType today.
	AgentID   string
	AgentType string
}

// Read decodes the host's payload.
func Read(host Host, r io.Reader) (Payload, error) {
	switch host {
	case HostClaude:
		return readClaude(r)
	case HostAntigravity:
		return readAntigravity(r)
	case HostCodex:
		return readCodex(r)
	}
	return Payload{}, fmt.Errorf("unknown host %q", host)
}

// WriteContext hands lines back for the model to read.
//
// Nothing to say writes nothing at all, and that test sits inside the Claude
// arm: ahead of the adapter, behind the switch. Behind it on purpose, because
// a seam's refusal *is* its arm -- an emptiness test in front of the switch
// would make ErrNoAdapter conditional on there being content and let a host
// with no adapter answer nil to an empty call. So every arm answers for its
// own host at every call size, and a second adapter that wants the same
// silence spells it out where this one does.
// The event is the host's own name for what is being answered, since one
// answer shape serves several events: a SessionStart envelope that named
// SubagentStop's answer would be read by nobody.
func WriteContext(host Host, event string, w io.Writer, lines []string) error {
	switch host {
	case HostClaude:
		if len(lines) == 0 {
			return nil
		}
		return writeClaudeContext(w, event, lines)
	case HostAntigravity:
		if len(lines) == 0 {
			return nil
		}
		return writeAntigravityContext(w, event, lines)
	case HostCodex:
		return writeCodexContext(w, event, lines)
	}
	return fmt.Errorf("unknown host %q", host)
}

// FindRoot walks up from `start` to the first directory holding
// `.loomux/config.toml`.
//
// Needed because an Antigravity hook runs with its working directory set to
// the directory holding `hooks.json`, which is `.agents/` and not the project
// root. Measured on 2026-09-10 against agy 1.1.24 -- see
// docs/.superpowers/specs/2026-09-10-antigravity-hook-messung.md, finding 2 --
// and stated the same way in the antigravity-for-claude-code plugin's
// docs/MIGRATION.md:157. Whether Antigravity sets `${CLAUDE_PROJECT_DIR}` is
// not part of either finding and is unmeasured here; the working directory
// alone is reason enough to walk. A `--root` given on the command line
// outranks this and is handled by the caller.
func FindRoot(start string) (string, error) {
	return findRoot(start, filepath.Abs)
}

// findRoot is FindRoot with its path resolver as a parameter.
//
// filepath.Abs fails only when os.Getwd does, which no test on any of the
// platforms this runs on can provoke -- so the resolver is injected here rather than
// leaving the refusal untested. The caller passes ".", so resolving is not
// optional: filepath.Dir would walk that to "." and stop there instead of
// climbing to the volume root.
func findRoot(start string, resolve func(string) (string, error)) (string, error) {
	current, err := resolve(start)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", start, err)
	}
	for {
		candidate := filepath.Join(current, ".loomux", "config.toml")
		if _, err := os.Stat(candidate); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("%s: %w", start, ErrNoRoot)
		}
		current = parent
	}
}
