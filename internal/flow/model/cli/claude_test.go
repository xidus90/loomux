package cli

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/model"
)

var claudeReplyTypes = map[string]model.ReplyType{"verdict": model.ReplyString, "n": model.ReplyInt}

// fakeClaude records the Command and answers with stdout.
func fakeClaude(t *testing.T, stdout []byte, got *Command) model.Model {
	t.Helper()
	return NewClaude(Options{Exec: func(_ context.Context, c Command) ([]byte, []byte, error) {
		*got = c
		return stdout, nil, nil
	}})
}

func claudeReq(r model.Request) model.Request {
	r.Reply = map[string]model.ReplyType{"verdict": model.ReplyString}
	r.Prompt = "hello"
	return r
}

func TestClaudeArgs(t *testing.T) {
	var got Command
	m := fakeClaude(t, []byte(`{"structured_output":{"verdict":"x"}}`), &got)
	req := claudeReq(model.Request{Model: "m", Effort: "high", Agent: "a", Tools: []string{"Glob", "Grep", "Read", "mcp__qmd"}})
	if _, err := m.Ask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	want := []string{"-p", "--output-format", "json", "--json-schema", Schema(req.Reply),
		"--model", "m", "--effort", "high", "--agent", "a",
		"--tools", "Glob,Grep,Read", "--allowedTools", "mcp__qmd"}
	if got.Name != "claude" || !reflect.DeepEqual(got.Args, want) {
		t.Fatalf("got %s %q", got.Name, got.Args)
	}
	if string(got.Stdin) != "hello" {
		t.Fatalf("stdin %q", got.Stdin)
	}
	if len(got.Env) != 0 {
		t.Fatalf("env %q", got.Env)
	}
}

func TestClaudeOmitsEmptyFlags(t *testing.T) {
	var got Command
	m := fakeClaude(t, []byte(`{"structured_output":{"verdict":"x"}}`), &got)
	if _, err := m.Ask(context.Background(), claudeReq(model.Request{})); err != nil {
		t.Fatal(err)
	}
	args := strings.Join(got.Args, " ")
	for _, f := range []string{"--model", "--effort", "--agent", "--allowedTools", "--dangerously-skip-permissions"} {
		if strings.Contains(args, f) {
			t.Errorf("unexpected %s in %q", f, args)
		}
	}
	// No tools asked for means no tools, not the CLI's default set.
	if !strings.Contains(args, `--tools `) || got.Args[len(got.Args)-1] != "" {
		t.Errorf("want --tools with empty value, got %q", got.Args)
	}
}

func TestClaudeBaseURL(t *testing.T) {
	var got Command
	m := fakeClaude(t, []byte(`{"structured_output":{"verdict":"x"}}`), &got)
	if _, err := m.Ask(context.Background(), claudeReq(model.Request{BaseURL: "http://127.0.0.1:8080"})); err != nil {
		t.Fatal(err)
	}
	// Strata's README names no token; only the base URL is set.
	if !reflect.DeepEqual(got.Env, []string{"ANTHROPIC_BASE_URL=http://127.0.0.1:8080"}) {
		t.Fatalf("env %q", got.Env)
	}
}

func TestClaudeReply(t *testing.T) {
	raw, err := os.ReadFile("testdata/claude-reply.json")
	if err != nil {
		t.Fatal(err)
	}
	var got Command
	m := fakeClaude(t, raw, &got)
	reply, err := m.Ask(context.Background(), model.Request{Prompt: "p", Reply: claudeReplyTypes})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Fields["verdict"] != "ok" || reply.Fields["n"] != 1 || reply.Tokens != 243 || reply.Model != "claude-haiku-5-5" {
		t.Fatalf("got %+v", reply)
	}
}

func TestClaudeRefusals(t *testing.T) {
	for name, out := range map[string]string{
		"not json":      `nope`,
		"is_error":      `{"is_error":true,"result":"boom"}`,
		"no structured": `{"result":"{}"}`,
		"bad fields":    `{"structured_output":{"other":1}}`,
	} {
		var got Command
		if _, err := fakeClaude(t, []byte(out), &got).Ask(context.Background(), claudeReq(model.Request{})); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestClaudeDenied(t *testing.T) {
	var got Command
	out := `{"result":"","permission_denials":[{"tool_name":"Bash"},{"tool_name":""}]}`
	_, err := fakeClaude(t, []byte(out), &got).Ask(context.Background(), claudeReq(model.Request{}))
	want := "claude denied Bash, a tool in headless mode and returned no reply; allow them in claude's settings"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v", err)
	}
}

func TestClaudeRunError(t *testing.T) {
	m := NewClaude(Options{Exec: func(context.Context, Command) ([]byte, []byte, error) {
		return nil, nil, context.Canceled
	}})
	if _, err := m.Ask(context.Background(), claudeReq(model.Request{})); err == nil {
		t.Fatal("want error")
	}
}

func TestClaudeModelAmbiguous(t *testing.T) {
	var got Command
	m := fakeClaude(t, []byte(`{"structured_output":{"verdict":"x"},"modelUsage":{"a":{},"b":{}}}`), &got)
	r, err := m.Ask(context.Background(), claudeReq(model.Request{}))
	if err != nil || r.Model != "" {
		t.Fatalf("got %+v %v", r, err)
	}
}
