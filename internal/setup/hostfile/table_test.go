package hostfile

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

const claudeRoot = `--root "${CLAUDE_PROJECT_DIR}"`

// The hook is the word after "hook" wherever it stands, and a command
// without that word runs none.
func TestHookEventOfReadsTheWordAfterHook(t *testing.T) {
	for command, want := range map[string]string{
		"loomux --root . hook stop": "stop",
		"loomux hook pre-tool-use":  "pre-tool-use",
		"loomux run stop":           "",
	} {
		if got := hookEventOf(command); got != want {
			t.Errorf("hookEventOf(%q) = %q, want %q", command, got, want)
		}
	}
}

func TestEntriesMatchTheTable(t *testing.T) {
	b := Canonical
	want := []Entry{
		{Event: "SessionStart", Command: b + " hook session-start --host claude " + claudeRoot, Timeout: 20},
		{Event: "PreToolUse", Matcher: "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell",
			Command: b + " hook pre-tool-use --host claude " + claudeRoot, Timeout: 15},
		{Event: "PostToolUse", Matcher: "Write|Edit|MultiEdit|NotebookEdit",
			Command: b + " hook post-tool-use --host claude " + claudeRoot, Timeout: 60},
		{Event: "Stop", Command: b + " hook stop --host claude " + claudeRoot + " --budget 270s", Timeout: 300},
		{Event: "SubagentStart", Command: b + " hook subagent-start --host claude " + claudeRoot, Timeout: 30},
		{Event: "SubagentStop", Command: b + " hook subagent-stop --host claude " + claudeRoot, Timeout: 30},
	}
	if got := Entries(hosts.HostClaude, b); !reflect.DeepEqual(got, want) {
		t.Fatalf("Entries(claude) =\n%v\nwant\n%v", got, want)
	}
}

// Antigravity's entries call the installed binary in the form cmd.exe
// expands, whichever binary Claude's entries call: agy runs them through
// cmd.exe from .agents/, keeps ${LOCALAPPDATA} literal and breaks a quoted
// program path.
func TestAntigravityEntriesCallTheInstalledBinaryThroughCmd(t *testing.T) {
	if AntigravityBinary != "%LOCALAPPDATA%/loomux/bin/loomux.exe" {
		t.Fatalf("AntigravityBinary = %q", AntigravityBinary)
	}
	want := []Entry{
		{Event: "PreInvocation", Command: "%LOCALAPPDATA%/loomux/bin/loomux.exe hook session-start --host antigravity --root ..", Timeout: 20, Flat: true},
		{Event: "PreToolUse", Matcher: "write_to_file|replace_file_content|multi_replace_file_content|run_command|send_command_input|manage_task",
			Command: "%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root ..", Timeout: 15},
		{Event: "PostToolUse", Matcher: "write_to_file|replace_file_content|multi_replace_file_content",
			Command: "%LOCALAPPDATA%/loomux/bin/loomux.exe hook post-tool-use --host antigravity --root ..", Timeout: 60},
		{Event: "Stop", Command: "%LOCALAPPDATA%/loomux/bin/loomux.exe hook stop --host antigravity --root .. --budget 270s", Timeout: 300, Flat: true},
	}
	for _, binary := range []string{Canonical, Checkout} {
		got := Entries(hosts.HostAntigravity, binary)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Entries(antigravity, %s) =\n%v\nwant\n%v", binary, got, want)
		}
		for _, e := range got {
			if !Owned(e.Command) || hookEventOf(e.Command) == "" {
				t.Errorf("%q is not recognised as ours", e.Command)
			}
		}
	}
}

func TestAHostWithoutAHookFileGetsNothing(t *testing.T) {
	if got := Entries(hosts.HostCodex, Canonical); got != nil {
		t.Fatalf("Entries(codex) = %v, want nil", got)
	}
	if got := Path(hosts.HostCodex); got != "" {
		t.Fatalf("Path(codex) = %q, want empty", got)
	}
}

func TestPathNamesTheHookFile(t *testing.T) {
	if got := Path(hosts.HostClaude); got != ".claude/settings.json" {
		t.Fatalf("Path(claude) = %q", got)
	}
	if got := Path(hosts.HostAntigravity); got != ".agents/hooks.json" {
		t.Fatalf("Path(antigravity) = %q", got)
	}
}

func TestOwnedKnowsEveryLoomuxBinary(t *testing.T) {
	owned := []string{
		"loomux hook stop",
		"loomux.exe hook stop",
		"LOOMUX.EXE hook stop",
		`"${CLAUDE_PROJECT_DIR}/bin/loomux.exe" hook stop`,
		Canonical + " hook stop",
		AntigravityBinary + " hook stop",
		`C:\x\loomux.exe hook stop`,
		`"C:\Program Files\loomux\loomux.exe" hook stop`,
	}
	for _, command := range owned {
		if !Owned(command) {
			t.Errorf("Owned(%q) = false, want true", command)
		}
	}
	foreign := []string{
		`other-guard --root "${CLAUDE_PROJECT_DIR}"`,
		"notes-guard",
		"uv run loomux-ish",
		`"loomuxer.exe"`,
		"",
		`"loomux.exe`,
	}
	for _, command := range foreign {
		if Owned(command) {
			t.Errorf("Owned(%q) = true, want false", command)
		}
	}
}
