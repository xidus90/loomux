package hostfile

import (
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/shellwords"
)

// Entry is one hook a host runs: on Event, for tools matching Matcher (empty
// for events without tools), the shell command Command, stopped after
// Timeout seconds (0 leaves the host's default). Flat puts the handler into
// the event's list itself instead of a block with a "hooks" list, the form
// agy requires for its events without tools.
type Entry struct {
	Event, Matcher, Command string
	Timeout                 int
	Flat                    bool
}

// Canonical and Checkout are the two binaries an entry may call.
const Canonical = `"${LOCALAPPDATA}/loomux/bin/loomux.exe"`
const Checkout = `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe"`

// AntigravityBinary is the installed binary in the form Antigravity's hook
// entries call it. agy (measured with 1.2.8 on 2026-09-24) runs a hook
// command through cmd.exe, which expands %LOCALAPPDATA% and leaves
// ${LOCALAPPDATA} as it stands, and it hands a quoted program path on as
// \"…\", which breaks it -- so the path goes unquoted, with forward slashes.
const AntigravityBinary = "%LOCALAPPDATA%/loomux/bin/loomux.exe"

// Entries are the hooks loomux installs for host. Claude's call binary;
// Antigravity's always call AntigravityBinary, also in a checkout, because
// ${CLAUDE_PROJECT_DIR} means nothing to agy. A host without a hook file
// gets none.
func Entries(host hosts.Host, binary string) []Entry {
	switch host {
	case hosts.HostClaude:
		root := ` --root "${CLAUDE_PROJECT_DIR}"`
		hook := func(name string) string { return binary + " hook " + name + " --host claude" + root }
		return []Entry{
			{Event: "SessionStart", Command: hook("session-start"), Timeout: 20},
			{Event: "PreToolUse", Matcher: "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell",
				Command: hook("pre-tool-use"), Timeout: 15},
			{Event: "PostToolUse", Matcher: "Write|Edit|MultiEdit|NotebookEdit",
				Command: hook("post-tool-use"), Timeout: 60},
			{Event: "Stop", Command: hook("stop") + " --budget 270s", Timeout: 300},
			{Event: "SubagentStart", Command: hook("subagent-start"), Timeout: 30},
			{Event: "SubagentStop", Command: hook("subagent-stop"), Timeout: 30},
		}
	case hosts.HostAntigravity:
		// agy runs a hook from .agents/, one below the project root. Its
		// PreInvocation and Stop take a flat list of handlers: agy 1.2.11
		// refuses the whole file when either holds a {"hooks": [...]} block
		// (measured 2026-09-25), which would leave the guard unloaded.
		writers := "write_to_file|replace_file_content|multi_replace_file_content"
		hook := func(name string) string {
			return AntigravityBinary + " hook " + name + " --host antigravity --root .."
		}
		return []Entry{
			{Event: "PreInvocation", Command: hook("session-start"), Timeout: 20, Flat: true},
			{Event: "PreToolUse", Matcher: writers + "|run_command|send_command_input|manage_task", Command: hook("pre-tool-use"), Timeout: 15},
			{Event: "PostToolUse", Matcher: writers, Command: hook("post-tool-use"), Timeout: 60},
			{Event: "Stop", Command: hook("stop") + " --budget 270s", Timeout: 300, Flat: true},
		}
	}
	return nil
}

// Path is the hook file of host relative to the project root, or "" for a
// host that has none.
func Path(host hosts.Host) string {
	switch host {
	case hosts.HostClaude:
		return ".claude/settings.json"
	case hosts.HostAntigravity:
		return ".agents/hooks.json"
	}
	return ""
}

// container is the top-level key of the hook file that holds the events.
func container(host hosts.Host) string {
	if host == hosts.HostAntigravity {
		return "loomux"
	}
	return "hooks"
}

// Owned says whether command calls a loomux binary: its first word, under
// any directory and in any case, is loomux or loomux.exe. That is the whole
// mark of an entry of ours; the file carries no owner key.
func Owned(command string) bool {
	name := strings.ToLower(filepath.Base(firstWord(command)))
	return name == "loomux" || name == "loomux.exe"
}

// hookEventOf is the hook a loomux command runs -- the word after "hook",
// as in `loomux hook pre-tool-use` -- or "" when it runs none.
func hookEventOf(command string) string {
	words, err := shellwords.Split(strings.ReplaceAll(command, `\`, `\\`))
	if err != nil {
		return ""
	}
	for i := 1; i+1 < len(words); i++ {
		if words[i] == "hook" {
			return words[i+1]
		}
	}
	return ""
}

// firstWord is the program of command with every backslash turned into a
// slash, or "" when command has no words or does not parse. Backslashes are
// doubled before the split so a Windows path survives it instead of being
// read as escapes.
func firstWord(command string) string {
	words, err := shellwords.Split(strings.ReplaceAll(command, `\`, `\\`))
	if err != nil || len(words) == 0 {
		return ""
	}
	return strings.ReplaceAll(words[0], `\`, "/")
}
