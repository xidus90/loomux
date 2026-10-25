package cli

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow/model"
)

// fakeQoder answers each call with the next stdout and records every Command.
func fakeQoder(t *testing.T, stdouts []string, got *[]Command) model.Model {
	t.Helper()
	return NewQoder(Options{Exec: func(_ context.Context, c Command) ([]byte, []byte, error) {
		*got = append(*got, c)
		return []byte(stdouts[len(*got)-1]), nil, nil
	}})
}

func qoderResult(result string) string {
	b, _ := json.Marshal(map[string]any{"is_error": false, "result": result})
	return string(b)
}

const qoderOK = `{"verdict":"x"}`

func TestQoderArgs(t *testing.T) {
	var got []Command
	req := claudeReq(model.Request{Model: "m", Effort: "high", Agent: "a", Tools: []string{"Read", "Grep"}})
	if _, err := fakeQoder(t, []string{qoderResult(qoderOK)}, &got).Ask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	want := []string{"-p", "-o", "json", "-m", "m", "--reasoning-effort", "high", "--agent", "a", "--tools", "Read,Grep"}
	if got[0].Name != "qodercli" || !reflect.DeepEqual(got[0].Args, want) {
		t.Fatalf("got %s %q", got[0].Name, got[0].Args)
	}
	args := strings.Join(got[0].Args, " ")
	if strings.Contains(args, "bypass") || strings.Contains(args, "skip-permissions") {
		t.Fatal("must not bypass permissions")
	}
}

func TestQoderNoToolsIsEmptyList(t *testing.T) {
	var got []Command
	if _, err := fakeQoder(t, []string{qoderResult(qoderOK)}, &got).Ask(context.Background(), claudeReq(model.Request{})); err != nil {
		t.Fatal(err)
	}
	want := []string{"-p", "-o", "json", "--tools", ""}
	if !reflect.DeepEqual(got[0].Args, want) {
		t.Fatalf("got %q", got[0].Args)
	}
}

func TestQoderToolsDropsUnknown(t *testing.T) {
	kept, dropped := qoderTools([]string{"Read", "mcp__qmd", "Bash"})
	if !reflect.DeepEqual(kept, []string{"Read", "Bash"}) || !reflect.DeepEqual(dropped, []string{"mcp__qmd"}) {
		t.Fatalf("kept %q dropped %q", kept, dropped)
	}
}

func TestQoderPromptCarriesSchema(t *testing.T) {
	var got []Command
	req := claudeReq(model.Request{})
	if _, err := fakeQoder(t, []string{qoderResult(qoderOK)}, &got).Ask(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	stdin := string(got[0].Stdin)
	if !strings.HasPrefix(stdin, "hello") || !strings.Contains(stdin[strings.LastIndex(stdin, "\n"):], Schema(req.Reply)) {
		t.Fatalf("stdin %q", stdin)
	}
}

func TestQoderRetriesOnce(t *testing.T) {
	var got []Command
	reply, err := fakeQoder(t, []string{qoderResult("nope"), qoderResult("```json\n" + qoderOK + "\n```")}, &got).
		Ask(context.Background(), claudeReq(model.Request{}))
	if err != nil || reply.Fields["verdict"] != "x" || len(got) != 2 {
		t.Fatalf("got %+v %v after %d calls", reply, err, len(got))
	}
	if !strings.Contains(string(got[1].Stdin), "not a JSON object") {
		t.Fatalf("second prompt lacks the error: %q", got[1].Stdin)
	}
	got = nil
	if _, err := fakeQoder(t, []string{qoderResult("nope"), qoderResult("nope")}, &got).Ask(context.Background(), claudeReq(model.Request{})); err == nil || len(got) != 2 {
		t.Fatalf("want error after two calls, got %v after %d", err, len(got))
	}
}

func TestQoderReply(t *testing.T) {
	raw, err := os.ReadFile("testdata/qoder-reply.json")
	if err != nil {
		t.Fatal(err)
	}
	var got []Command
	reply, err := fakeQoder(t, []string{string(raw)}, &got).Ask(context.Background(), model.Request{Prompt: "p", Reply: claudeReplyTypes})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Fields["verdict"] != "ok" || reply.Fields["n"] != 1 || reply.Tokens != 15 || reply.Model != "auto" {
		t.Fatalf("got %+v", reply)
	}
}

func TestQoderModelAmbiguous(t *testing.T) {
	var got []Command
	out := `{"result":"{\"verdict\":\"x\"}","modelUsage":{"a":{},"b":{}}}`
	r, err := fakeQoder(t, []string{out}, &got).Ask(context.Background(), claudeReq(model.Request{}))
	if err != nil || r.Model != "" {
		t.Fatalf("got %+v %v", r, err)
	}
}

func TestQoderRefusals(t *testing.T) {
	for name, out := range map[string]string{
		"not json": `nope`,
		"is_error": `{"is_error":true,"result":"boom"}`,
	} {
		var got []Command
		if _, err := fakeQoder(t, []string{out}, &got).Ask(context.Background(), claudeReq(model.Request{})); err == nil || len(got) != 1 {
			t.Errorf("%s: want error after one call, got %v after %d", name, err, len(got))
		}
	}
}

func TestQoderDenied(t *testing.T) {
	var got []Command
	out := `{"is_error":false,"result":"I could not run it","permission_denials":[{"tool_name":"Bash"}]}`
	_, err := fakeQoder(t, []string{out}, &got).Ask(context.Background(), claudeReq(model.Request{}))
	want := "qoder denied Bash in headless mode and returned no reply; allow them in qodercli's settings"
	if err == nil || err.Error() != want || len(got) != 1 {
		t.Fatalf("got %v after %d calls", err, len(got))
	}
}

func TestQoderRefusesWhenNoToolSurvives(t *testing.T) {
	var got []Command
	_, err := fakeQoder(t, nil, &got).Ask(context.Background(), claudeReq(model.Request{Tools: []string{"mcp__qmd"}}))
	if err == nil || err.Error() != "qoder has none of the tools the profile allows" || len(got) != 0 {
		t.Fatalf("got %v after %d calls", err, len(got))
	}
}

func TestQoderRunError(t *testing.T) {
	m := NewQoder(Options{Exec: func(context.Context, Command) ([]byte, []byte, error) {
		return nil, nil, context.Canceled
	}})
	if _, err := m.Ask(context.Background(), claudeReq(model.Request{})); err == nil {
		t.Fatal("want error")
	}
}
