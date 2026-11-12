package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/sessions"
)

// hintWorld is world with a built graph at the root.
func hintWorld(t *testing.T, config string) (root, state string) {
	t.Helper()
	root, state = world(t, config)
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, state
}

// grepCall is a Bash grep over code as a session (and maybe a subagent) runs
// it from cwd.
func grepCall(cwd, session, agent string) string {
	payload := map[string]any{
		"tool_name": "Bash", "cwd": cwd, "session_id": session,
		"tool_input": map[string]any{"command": `grep -rn "blastAside(" --include=*.go internal`},
	}
	if agent != "" {
		payload["agent_id"] = agent
	}
	data, _ := json.Marshal(payload)
	return string(data)
}

func TestPreToolUseHintsOnceAtTheFirstCodeGrep(t *testing.T) {
	root, state := hintWorld(t, "")
	code, out, errOut := call(t, root, state, grepCall(root, "s1", ""))
	if code != 2 || !strings.Contains(out+errOut, "loomux graph callers blastAside") || !strings.Contains(out+errOut, "running the same command again goes through") {
		t.Fatalf("first code grep: code %d, out %q, err %q; want the hint", code, out, errOut)
	}
	if strings.Contains(out+errOut, "policy refused") {
		t.Errorf("the hint must not read as a policy refusal: %q", out+errOut)
	}
	if code, _, errOut := call(t, root, state, grepCall(root, "s1", "")); code != 0 {
		t.Errorf("the same grep again: code %d, err %q; want it through", code, errOut)
	}
}

// The Grep tool reaches the guard only since it joined the matcher: the
// policy and the write barrier must let it through, and the hint answers it
// like a shell grep.
func TestPreToolUseTreatsTheGrepToolLikeAShellGrep(t *testing.T) {
	root, state := hintWorld(t, "")
	grep := func(pattern string) string {
		data, _ := json.Marshal(map[string]any{
			"tool_name": "Grep", "cwd": root, "session_id": "s1",
			"tool_input": map[string]any{"pattern": pattern, "path": filepath.Join(root, "internal")},
		})
		return string(data)
	}
	if code, _, errOut := call(t, root, state, grep("timeout")); code != 0 {
		t.Fatalf("a Grep for no symbol: code %d, err %q; want it through", code, errOut)
	}
	if code, out, errOut := call(t, root, state, grep(`NewIndex\(`)); code != 2 || !strings.Contains(out+errOut, "loomux graph callers NewIndex") {
		t.Errorf("a Grep for a symbol: code %d, out %q, err %q; want the hint", code, out, errOut)
	}
	if code, _, errOut := call(t, root, state, grep(`NewIndex\(`)); code != 0 {
		t.Errorf("the same Grep again: code %d, err %q; want it through", code, errOut)
	}
}

func TestGraphQuery(t *testing.T) {
	for _, c := range []struct {
		tool, command string
		want          bool
	}{
		{"mcp__loomux__graph_blast", "", true},
		{"mcp__loomux__brain_search", "", false},
		{"Bash", "loomux graph callers X", true},
		{"Bash", "cd internal && loomux graph blast", true},
		{"Bash", "loomux check", false},
		{"Bash", "&", false},
		{"Bash", `echo "unclosed`, false},
		{"PowerShell", `& loomux.exe graph map`, true},
		{"Read", "", false},
	} {
		input := map[string]any{}
		if c.command != "" {
			input["command"] = c.command
		}
		if got := graphQuery(c.tool, input); got != c.want {
			t.Errorf("graphQuery(%s, %q) = %v, want %v", c.tool, c.command, got, c.want)
		}
	}
}

// An agent that already asked the graph needs no hint: the query spends it.
func TestPreToolUseSpendsTheHintOnAGraphQuery(t *testing.T) {
	for name, query := range map[string]map[string]any{
		"mcp tool": {"tool_name": "mcp__loomux__graph_trace_calls", "tool_input": map[string]any{"symbol": "store.Read"}},
		"shell":    {"tool_name": "Bash", "tool_input": map[string]any{"command": "loomux graph callers store.Read --depth 2"}},
		"shell exe": {"tool_name": "PowerShell", "tool_input": map[string]any{
			"command": `& "C:/tools/loomux.exe" graph callers store.Read`}},
	} {
		t.Run(name, func(t *testing.T) {
			root, state := hintWorld(t, "")
			query["cwd"], query["session_id"] = root, "s1"
			data, _ := json.Marshal(query)
			if code, _, errOut := call(t, root, state, string(data)); code != 0 {
				t.Fatalf("graph query: code %d, err %q; want it through", code, errOut)
			}
			if code, _, errOut := call(t, root, state, grepCall(root, "s1", "")); code != 0 {
				t.Errorf("grep after a graph query: code %d, err %q; want no hint", code, errOut)
			}
		})
	}
}

// The Grep tool and the graph tools reach the hook only for the hint. A
// broken policy refuses everything else as before, but it must not take away
// the search and the very tools the hint points at.
func TestPreToolUseLetsTheHintOnlyToolsThroughABrokenPolicy(t *testing.T) {
	root, state := hintWorld(t, "[guard]\ngraph_hint = \"no\"\n")
	for _, tool := range []string{"Grep", "mcp__loomux__graph_trace_calls"} {
		data, _ := json.Marshal(map[string]any{
			"tool_name": tool, "cwd": root, "session_id": "s1",
			"tool_input": map[string]any{"pattern": "NewIndex", "symbol": "NewIndex"},
		})
		if code, _, errOut := call(t, root, state, string(data)); code != 0 {
			t.Errorf("%s under a broken policy: code %d, err %q; want it through", tool, code, errOut)
		}
	}
	if code, _, _ := call(t, root, state, grepCall(root, "s1", "")); code != 2 {
		t.Errorf("a shell call under a broken policy: code %d; want the refusal it always got", code)
	}
}

func TestPreToolUseHintsEachSubagentOnce(t *testing.T) {
	root, state := hintWorld(t, "")
	for _, agent := range []string{"", "a1", "a2"} {
		if code, _, _ := call(t, root, state, grepCall(root, "s1", agent)); code != 2 {
			t.Errorf("agent %q: code %d; want its own hint", agent, code)
		}
	}
}

func TestPreToolUseHintsNotWithoutAGraph(t *testing.T) {
	root, state := world(t, "")
	if code, _, errOut := call(t, root, state, grepCall(root, "s1", "")); code != 0 {
		t.Errorf("no graph: code %d, err %q; want it through", code, errOut)
	}
}

func TestPreToolUseHintsNotWhenSwitchedOff(t *testing.T) {
	root, state := hintWorld(t, "[guard]\ngraph_hint = false\n")
	if code, _, errOut := call(t, root, state, grepCall(root, "s1", "")); code != 0 {
		t.Errorf("switched off: code %d, err %q; want it through", code, errOut)
	}
}

func TestPreToolUseHintsNotWithoutASession(t *testing.T) {
	root, state := hintWorld(t, "")
	if code, _, errOut := call(t, root, state, grepCall(root, "", "")); code != 0 {
		t.Errorf("no session: code %d, err %q; want it through", code, errOut)
	}
}

func TestPreToolUseFindsTheGraphAboveCwd(t *testing.T) {
	root, state := hintWorld(t, "")
	sub := filepath.Join(root, "internal", "hooks")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := call(t, root, state, grepCall(sub, "s1", "")); code != 2 {
		t.Errorf("cwd below the graph: code %d; want the hint", code)
	}
}

func TestPreToolUseLooksForTheGraphAtTheRootWithoutCwd(t *testing.T) {
	root, state := hintWorld(t, "")
	if code, _, _ := call(t, root, state, grepCall("", "s1", "")); code != 2 {
		t.Errorf("no cwd: code %d; want the hint from the root's graph", code)
	}
}

func TestPreToolUseLetsThroughWhenTheMarkCannotBeWritten(t *testing.T) {
	root, state := hintWorld(t, "")
	// The hooks state folder is a file: no mark can be made, so no hint.
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(sessions.StateDir)), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := call(t, root, state, grepCall(root, "s1", "")); code != 0 {
		t.Errorf("unwritable mark: code %d, err %q; want it through", code, errOut)
	}
}

func TestPreToolUseKeepsAPolicyRefusalBeforeTheHint(t *testing.T) {
	root, state := hintWorld(t, "[[policy.commands.rules]]\nregex = \"grep\"\nreason = \"no grep here\"\n")
	code, _, errOut := call(t, root, state, grepCall(root, "s1", ""))
	if code != 2 || !strings.Contains(errOut, "no grep here") || strings.Contains(errOut, "loomux graph callers") {
		t.Errorf("policy first: code %d, err %q", code, errOut)
	}
	if first, _ := sessions.MarkOnce(root, "s1", "", "graph-hint"); !first {
		t.Error("a policy refusal must not spend the session's hint")
	}
}

func TestGraphHintSymbol(t *testing.T) {
	for _, c := range []struct {
		name  string
		tool  string
		input map[string]any
		want  string
	}{
		{"grep tool call shape", "Grep", map[string]any{"pattern": `relInRoot\(`}, "relInRoot"},
		{"grep tool definition in a code dir", "Grep", map[string]any{"pattern": "func Plan", "path": "internal/verify"}, "Plan"},
		{"grep tool method definition", "Grep", map[string]any{"pattern": `func \(x \*Index\) Reach`, "glob": "*.go"}, "Reach"},
		{"grep tool over docs", "Grep", map[string]any{"pattern": "Schreibschranke", "path": "docs/wiki"}, ""},
		{"grep tool on markdown", "Grep", map[string]any{"pattern": "blastAside", "glob": "*.md"}, ""},
		{"grep tool on a python type", "Grep", map[string]any{"pattern": "TokenStore", "type": "py"}, "TokenStore"},
		{"grep tool plain word", "Grep", map[string]any{"pattern": "timeout"}, ""},
		{"grep tool short name", "Grep", map[string]any{"pattern": `Do\(`}, ""},
		{"grep tool keyword only", "Grep", map[string]any{"pattern": `func \(`}, ""},
		{"grep tool regex alternation", "Grep", map[string]any{"pattern": "CallersReport|BlastReport", "path": "internal"}, "CallersReport"},
		{"shell call shape with include", "Bash", map[string]any{"command": `grep -rn "blastAside(" --include=*.go internal`}, "blastAside"},
		{"shell pipe from git log", "Bash", map[string]any{"command": "git log --oneline | grep Radius"}, ""},
		{"shell definition in a test file", "Bash", map[string]any{"command": `grep -n "func sample" -A25 internal/code/query/x_test.go`}, "sample"},
		{"shell over a log", "Bash", map[string]any{"command": `grep -E "timed-out|red|failed" /tmp/run.log`}, ""},
		{"shell after a sed", "Bash", map[string]any{"command": `sed -i 's/a/b/' internal/x.go; grep -n "preRe" internal/compat/scout.go`}, "preRe"},
		{"shell over docs", "Bash", map[string]any{"command": `grep -rn "Mitigation|JobObject" docs/`}, ""},
		{"shell without a scope reads stdin", "Bash", map[string]any{"command": `grep "NewIndex"`}, ""},
		{"shell rg with a type", "Bash", map[string]any{"command": `rg -t go "func NewIndex"`}, "NewIndex"},
		{"shell git grep", "Bash", map[string]any{"command": `git grep -n "EdgeWalk(" -- '*.go'`}, "EdgeWalk"},
		{"shell after or-else", "Bash", map[string]any{"command": `true || grep -rn "EdgeWalk(" internal`}, "EdgeWalk"},
		{"shell -e pattern", "Bash", map[string]any{"command": `grep -rn -e "graphHint(" internal`}, "graphHint"},
		{"powershell select-string", "PowerShell", map[string]any{"command": `Select-String -Path internal\*.go -Pattern "NewIndex"`}, "NewIndex"},
		{"shell unbalanced quote", "Bash", map[string]any{"command": `grep -rn "NewIndex internal`}, ""},
		{"shell rg --type=", "Bash", map[string]any{"command": `rg --type=go "NewIndex"`}, "NewIndex"},
		{"shell rg -g glob", "Bash", map[string]any{"command": `rg -g '*.go' "NewIndex"`}, "NewIndex"},
		{"shell rg --glob=", "Bash", map[string]any{"command": `rg --glob=*.go "func NewIndex"`}, "NewIndex"},
		{"shell context flag with a value", "Bash", map[string]any{"command": `grep -A 3 -rn "NewIndex(" internal`}, "NewIndex"},
		{"shell dangling -e", "Bash", map[string]any{"command": `grep -rn -e`}, ""},
		{"shell git grep pathspec", "Bash", map[string]any{"command": `git grep "NewIndex(" -- internal`}, "NewIndex"},
		{"shell after cd and", "Bash", map[string]any{"command": `cd internal && grep -rn "NewIndex(" .`}, "NewIndex"},
		{"shell grep without a symbol in code", "Bash", map[string]any{"command": `grep -rn "timeout" internal`}, ""},
		{"powershell literal path", "PowerShell", map[string]any{"command": `Select-String "NewIndex" -LiteralPath internal\a.go`}, "NewIndex"},
		{"powershell dangling pattern", "PowerShell", map[string]any{"command": `Select-String -Path internal\a.go -Pattern`}, ""},
		{"grep tool keeps a package qualifier", "Grep", map[string]any{"pattern": `freshness\.Probe\(`}, "freshness.Probe"},
		{"shell keeps a package qualifier", "Bash", map[string]any{"command": `grep -rn "store.Read(" internal`}, "store.Read"},
		{"a one-letter receiver is no package", "Grep", map[string]any{"pattern": `s\.handle\(`}, "handle"},
		{"grep tool on an absolute docs folder", "Grep", map[string]any{"pattern": "FooBar", "path": `C:\repo\docs`}, ""},
		{"shell on ./docs", "Bash", map[string]any{"command": "grep -rn FooBar ./docs"}, ""},
		{"grep tool type md", "Grep", map[string]any{"pattern": "FooBar", "type": "md"}, ""},
		{"shell rg -t md over the tree", "Bash", map[string]any{"command": "rg FooBar -t md ."}, ""},
		{"grep tool brace glob of docs", "Grep", map[string]any{"pattern": "FooBar", "glob": "*.{md,txt}"}, ""},
		{"grep tool brace glob with code", "Grep", map[string]any{"pattern": "FooBar", "glob": "*.{go,md}"}, "FooBar"},
		{"powershell unquoted windows docs path", "PowerShell", map[string]any{"command": `Select-String -Pattern FooBar -Path docs\wiki\*`}, ""},
		{"powershell keeps a regex escape", "PowerShell", map[string]any{"command": `Select-String -Pattern "freshness\.Probe\(" -Path internal\code\*.go`}, "freshness.Probe"},
		{"marker word TODO", "Bash", map[string]any{"command": "grep -rn TODO internal"}, ""},
		{"marker word FIXME in the tool", "Grep", map[string]any{"pattern": "FIXME"}, ""},
		{"a logs folder", "Bash", map[string]any{"command": "grep -rn FooBar logs/"}, ""},
		{"a system log", "Bash", map[string]any{"command": "grep -rn ErrNotFound /var/log/syslog"}, ""},
		{"no grep at all", "Bash", map[string]any{"command": "ls internal"}, ""},
		{"a writing tool", "Write", map[string]any{"file_path": "a.go", "content": "func NewIndex()"}, ""},
		{"no command", "Bash", map[string]any{}, ""},
		{"grep tool without a pattern", "Grep", map[string]any{"path": "internal"}, ""},
	} {
		got, ok := graphHintSymbol(c.tool, c.input)
		if got != c.want || ok != (c.want != "") {
			t.Errorf("%s: graphHintSymbol = %q, %v; want %q", c.name, got, ok, c.want)
		}
	}
}
