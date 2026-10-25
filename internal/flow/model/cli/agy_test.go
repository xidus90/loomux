package cli

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/flow/model"
)

func fakeAgy(t *testing.T, o Options, stdout []byte, got *Command) model.Model {
	t.Helper()
	o.Exec = func(_ context.Context, c Command) ([]byte, []byte, error) {
		*got = c
		return stdout, nil, nil
	}
	return NewAgy(o)
}

const agyOK = `{"status":"SUCCESS","structured_output":{"verdict":"x"}}`

func TestAgyArgs(t *testing.T) {
	for profile, mode := range map[string]string{"read_only": "plan", "edit": "accept-edits", "": "accept-edits"} {
		var got Command
		req := claudeReq(model.Request{Profile: profile, Model: "m", Effort: "high", Agent: "a"})
		if _, err := fakeAgy(t, Options{}, []byte(agyOK), &got).Ask(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		// -p takes the next word as the prompt, so it stays last.
		want := []string{"--output-format", "json", "--json-schema", Schema(req.Reply),
			"--model", "m", "--effort", "high", "--agent", "a", "--mode", mode, "-p", "hello"}
		if got.Name != "agy" || !reflect.DeepEqual(got.Args, want) {
			t.Fatalf("%q: got %s %q", profile, got.Name, got.Args)
		}
		if strings.Contains(strings.Join(got.Args, " "), "--dangerously-skip-permissions") {
			t.Fatal("must not skip permissions")
		}
	}
}

func TestAgyOmitsEmptyFlags(t *testing.T) {
	var got Command
	if _, err := fakeAgy(t, Options{}, []byte(agyOK), &got).Ask(context.Background(), claudeReq(model.Request{})); err != nil {
		t.Fatal(err)
	}
	args := strings.Join(got.Args, " ")
	for _, f := range []string{"--model", "--effort", "--agent", "--print-timeout"} {
		if strings.Contains(args, f) {
			t.Errorf("unexpected %s in %q", f, args)
		}
	}
}

func TestAgyTimeout(t *testing.T) {
	var got Command
	if _, err := fakeAgy(t, Options{Timeout: 90 * time.Second}, []byte(agyOK), &got).Ask(context.Background(), claudeReq(model.Request{})); err != nil {
		t.Fatal(err)
	}
	i := len(got.Args) - 6
	if !reflect.DeepEqual(got.Args[i:i+2], []string{"--print-timeout", "1m30s"}) {
		t.Fatalf("got %q", got.Args)
	}
}

func TestAgyPromptTooLong(t *testing.T) {
	var got Command
	req := claudeReq(model.Request{})
	req.Prompt = strings.Repeat("x", 30001)
	_, err := fakeAgy(t, Options{}, []byte(agyOK), &got).Ask(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "30000") || got.Name != "" {
		t.Fatalf("got %v, ran %q", err, got.Name)
	}
}

func TestAgyReply(t *testing.T) {
	raw, err := os.ReadFile("testdata/agy-reply.json")
	if err != nil {
		t.Fatal(err)
	}
	var got Command
	reply, err := fakeAgy(t, Options{}, raw, &got).Ask(context.Background(), model.Request{Prompt: "p", Reply: claudeReplyTypes})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Fields["verdict"] != "ok" || reply.Fields["n"] != 1 || reply.Tokens != 15634+1849 || reply.Model != "" {
		t.Fatalf("got %+v", reply)
	}
}

func TestAgyStatusNotSuccess(t *testing.T) {
	var got Command
	_, err := fakeAgy(t, Options{}, []byte(`{"status":"ERROR","response":"boom"}`), &got).Ask(context.Background(), claudeReq(model.Request{}))
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("got %v", err)
	}
}

func TestAgyRefusals(t *testing.T) {
	for name, out := range map[string]string{
		"not json":        `nope`,
		"no structured":   `{"status":"SUCCESS","response":"plain text"}`,
		"null structured": `{"status":"SUCCESS","structured_output":null}`,
		"bad fields":      `{"status":"SUCCESS","structured_output":{"other":1}}`,
	} {
		var got Command
		if _, err := fakeAgy(t, Options{}, []byte(out), &got).Ask(context.Background(), claudeReq(model.Request{})); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestAgyRunError(t *testing.T) {
	m := NewAgy(Options{Exec: func(context.Context, Command) ([]byte, []byte, error) {
		return nil, nil, context.Canceled
	}})
	if _, err := m.Ask(context.Background(), claudeReq(model.Request{})); err == nil {
		t.Fatal("want error")
	}
}

func TestAgyDeniedFallsBackToAction(t *testing.T) {
	var got Command
	out := `{"status":"SUCCESS","denied_actions":[{"action":"command","display_name":""}]}`
	_, err := fakeAgy(t, Options{}, []byte(out), &got).Ask(context.Background(), claudeReq(model.Request{}))
	if err == nil || !strings.Contains(err.Error(), "agy denied command in") {
		t.Fatalf("got %v", err)
	}
}

func TestAgyDenied(t *testing.T) {
	var got Command
	out := `{"status":"SUCCESS","response":"","denied_actions":[{"action":"command","display_name":"RunCommand"},{"action":"x","display_name":"Edit"}]}`
	_, err := fakeAgy(t, Options{}, []byte(out), &got).Ask(context.Background(), claudeReq(model.Request{}))
	want := "agy denied RunCommand, Edit in headless mode and returned no reply; allow it under permissions.allow in agy's settings.json"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v", err)
	}
}
