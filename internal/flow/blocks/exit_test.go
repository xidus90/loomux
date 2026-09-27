package blocks_test

import (
	"context"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
)

func exitNode(keys map[string]any) flow.Node {
	return flow.Node{Name: "stop", Kind: "exit", MaxVisits: flow.Cap{Add: 1}, Keys: keys}
}

// exitGraph has no texts: an exit's message is inline and never read from them.
func exitGraph() *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml"}
}

func TestExitEndsTheRunWithItsCodeAndRenderedMessage(t *testing.T) {
	node := exitNode(map[string]any{"code": int64(4), "message": "rejected after {{count}} rounds"})
	state := flow.State{Fields: map[string]flow.Value{"count": 3}}
	got, err := blocks.Exit{}.Run(context.Background(), exitGraph(), node, state, flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Exit == nil {
		t.Fatal("an exit node ends the run")
	}
	if got.Exit.Code != 4 || got.Exit.Message != "rejected after 3 rounds" {
		t.Fatalf("exit = %+v", got.Exit)
	}
	if got.Delta != nil || got.Question != nil || got.Tokens != 0 {
		t.Fatalf("an exit changes no field: %+v", got)
	}
}

func TestExitWritesNothingAndNeedsNoBaseline(t *testing.T) {
	node := exitNode(map[string]any{"code": int64(4), "message": "done"})
	// The parentheses are not decoration: `if blocks.Exit{}.Kind()` parses the
	// brace as the start of the if's body.
	if (blocks.Exit{}).Writes(node) != nil {
		t.Fatal("an exit writes no field")
	}
	if (blocks.Exit{}).NeedsBaseline() {
		t.Fatal("an exit reads no working tree")
	}
	if (blocks.Exit{}).Kind() != "exit" {
		t.Fatal("kind")
	}
}

// Inline, not a path: a message is a sentence, and a file per sentence would
// be a file per sentence.
func TestExitNamesItsMessageAsAnInlineText(t *testing.T) {
	texts := blocks.Exit{}.Texts(exitNode(map[string]any{"message": "done"}))
	if len(texts) != 1 || texts[0].Key != "message" || texts[0].Path != "" || texts[0].Inline != "done" {
		t.Fatalf("texts = %+v", texts)
	}
}

func TestExitDefineCarriesTheMessageBytes(t *testing.T) {
	got, err := blocks.Exit{}.Define(exitGraph(), exitNode(map[string]any{"message": "done"}), flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Text) != "done" || got.Model != "" {
		t.Fatalf("definition = %+v", got)
	}
}

// The runtime spends 0 for done, 1 for failed and 3 for paused. A block that
// could choose one of them would make an exit code unreadable: a 3 would mean
// either "waiting at a gate" or "this flow said so".
func TestExitRefusesTheRuntimesOwnCodes(t *testing.T) {
	for _, code := range []int64{0, 1, 3} {
		node := exitNode(map[string]any{"code": code, "message": "m"})
		got := blocks.Exit{}.Check(node)
		if len(got) != 1 {
			t.Fatalf("code %d: %v", code, got)
		}
	}
}

func TestExitCheckFindsEveryProblemAtOnce(t *testing.T) {
	cases := []struct {
		name string
		keys map[string]any
		want []string
	}{
		{"nothing at all", map[string]any{}, []string{
			"code must be an integer other than 0, 1 and 3",
			"message must be a text",
		}},
		{"a code out of range", map[string]any{"code": int64(300), "message": "m"},
			[]string{"code 300 is outside 0..255"}},
		{"a code that is no integer", map[string]any{"code": "four", "message": "m"},
			[]string{"code must be an integer other than 0, 1 and 3"}},
		{"an unknown key", map[string]any{"code": int64(4), "message": "m", "mesage": "m"},
			[]string{`unknown key "mesage"; known keys: code, kind, max_visits, message, name, role`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := blocks.Exit{}.Check(exitNode(c.keys))
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("finding %d is %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestExitReportsAMessageItCannotRender(t *testing.T) {
	node := exitNode(map[string]any{"code": int64(4), "message": "{{ name }}"})
	if _, err := (blocks.Exit{}).Run(context.Background(), exitGraph(), node, flow.State{}, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
}
