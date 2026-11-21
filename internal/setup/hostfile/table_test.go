package hostfile

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/mcptools"
)

const claudeRoot = `--root "${CLAUDE_PROJECT_DIR}"`

// A graph query spends the session's graph hint, so the pre-tool hook must
// see every graph tool that answers a question; the freshness check answers
// none.
func TestThePreToolMatcherCoversEveryAnsweringGraphTool(t *testing.T) {
	matcher := ""
	for _, e := range Entries(hosts.HostClaude, "loomux") {
		if e.Event == "PreToolUse" {
			matcher = "|" + e.Matcher + "|"
		}
	}
	for _, tool := range mcptools.Tools() {
		if !strings.HasPrefix(tool.Name, "graph_") || tool.Name == "graph_check_freshness" {
			continue
		}
		if !strings.Contains(matcher, "|mcp__loomux__"+tool.Name+"|") {
			t.Errorf("the PreToolUse matcher lacks mcp__loomux__%s", tool.Name)
		}
	}
}

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
		{Event: "PreToolUse", Matcher: "Write|Edit|MultiEdit|NotebookEdit|Bash|PowerShell|Grep|mcp__loomux__graph_find_code|mcp__loomux__graph_file_api|mcp__loomux__graph_trace_calls|mcp__loomux__graph_find_all|mcp__loomux__graph_repo_map|mcp__loomux__graph_blast",
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
		{Event: "PreToolUse", Matcher: "write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file|run_command|send_command_input|manage_task",
			Command: "%LOCALAPPDATA%/loomux/bin/loomux.exe hook pre-tool-use --host antigravity --root ..", Timeout: 15},
		{Event: "PostToolUse", Matcher: "write_to_file|replace_file_content|multi_replace_file_content|write_file|edit_file",
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

// DSH's table is Claude's six events with Claude's timeouts, and the tool
// matchers are `.*`: DSH's writing and running tools answer to names of
// their own (write, edit, pwsh), so a matcher of Claude's tool list would
// let them pass ungated.
func TestEntriesDsh(t *testing.T) {
	for _, b := range []string{Canonical, Checkout} {
		hook := func(name string) string { return b + " hook " + name + " --host dsh " + claudeRoot }
		want := []Entry{
			{Event: "SessionStart", Command: hook("session-start"), Timeout: 20},
			{Event: "PreToolUse", Matcher: ".*",
				Command: hook("pre-tool-use"), Timeout: 15},
			{Event: "PostToolUse", Matcher: ".*",
				Command: hook("post-tool-use"), Timeout: 60},
			{Event: "Stop", Command: hook("stop") + " --budget 270s", Timeout: 300},
			{Event: "SubagentStart", Command: hook("subagent-start"), Timeout: 30},
			{Event: "SubagentStop", Command: hook("subagent-stop"), Timeout: 30},
		}
		got := Entries(hosts.HostDsh, b)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Entries(dsh, %s) =\n%v\nwant\n%v", b, got, want)
		}
		for _, e := range got {
			if !Owned(e.Command) || hookEventOf(e.Command) == "" {
				t.Errorf("%q is not recognised as ours", e.Command)
			}
		}
	}
}

// DSH's hook file lives in loomux's own directory, where DSH reads it.
func TestPathDsh(t *testing.T) {
	if got := Path(hosts.HostDsh); got != ".loomux/dsh/hooks.json" {
		t.Fatalf("Path(dsh) = %q, want .loomux/dsh/hooks.json", got)
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
