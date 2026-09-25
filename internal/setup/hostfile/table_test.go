package hostfile

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
)

const claudeRoot = `--root "${CLAUDE_PROJECT_DIR}"`

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

// The Antigravity rows wait for the measurement of the hook file; until it
// has been made, the installer writes none.
func TestEntriesForAntigravityWaitForTheMeasurement(t *testing.T) {
	if got := Entries(hosts.HostAntigravity, Canonical); got != nil {
		t.Fatalf("Entries(antigravity) = %v, want nil while unmeasured", got)
	}
	b := Checkout
	want := []Entry{
		{Event: "PreToolUse", Matcher: "write_to_file|replace_file_content|multi_replace_file_content|run_command",
			Command: b + " hook pre-tool-use --host antigravity --root .", Timeout: 15},
		{Event: "PostToolUse", Matcher: "write_to_file|replace_file_content|multi_replace_file_content",
			Command: b + " hook post-tool-use --host antigravity --root .", Timeout: 60},
	}
	if got := entries(hosts.HostAntigravity, b, true); !reflect.DeepEqual(got, want) {
		t.Fatalf("entries(antigravity, measured) =\n%v\nwant\n%v", got, want)
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
		`C:\x\loomux.exe hook stop`,
		`"C:\Program Files\loomux\loomux.exe" hook stop`,
	}
	for _, command := range owned {
		if !Owned(command) {
			t.Errorf("Owned(%q) = false, want true", command)
		}
	}
	foreign := []string{
		`ulguard --root "${CLAUDE_PROJECT_DIR}"`,
		"brain guard",
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
