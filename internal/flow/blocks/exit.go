package blocks

import (
	"context"
	"fmt"

	"github.com/xidus90/loomux/internal/flow"
)

// Exit is the node kind that ends a run with a code of its own. A flow with
// more than one way of failing needs more than one way of saying so: a hook
// cannot tell "the checks stayed red" from "the repairer touched a test file"
// if both arrive as 1.
type Exit struct{}

// Kind is "exit".
func (Exit) Kind() string { return "exit" }

// Check reads the exit's own keys.
//
// 0, 1 and 3 are refused here rather than at the command line: the runtime
// spends them on done, failed and paused, and a 3 that could also mean "this
// flow said so" is a code no caller can act on.
func (Exit) Check(node flow.Node) []string {
	var findings []string
	code, ok := node.Keys["code"].(int64)
	switch {
	case !ok || code == 0 || code == 1 || code == 3:
		findings = append(findings, "code must be an integer other than 0, 1 and 3")
	case code < 0 || code > 255:
		findings = append(findings, fmt.Sprintf("code %d is outside 0..255", code))
	}
	if _, ok := node.Keys["message"].(string); !ok {
		findings = append(findings, "message must be a text")
	}
	findings = append(findings, node.KeyFindings("code", "message")...)
	return findings
}

// Texts names the message, which is inline: a message is a sentence, and a
// file per sentence would be a file per sentence.
func (Exit) Texts(node flow.Node) []flow.Text {
	return []flow.Text{{Key: "message", Inline: node.StringKey("message")}}
}

// Writes nothing: an exit ends the run, it does not change it.
func (Exit) Writes(flow.Node) map[string]flow.Type { return nil }

// NeedsBaseline is false.
func (Exit) NeedsBaseline() bool { return false }

// Define is the message's bytes.
func (Exit) Define(graph *flow.Graph, node flow.Node, _ flow.Env) (flow.Definition, error) {
	// The message is inline, so ReadText reads no file and cannot fail; a
	// branch for an error it never returns would be code no test can reach.
	raw, _ := flow.ReadText(graph.Texts, Exit{}.Texts(node)[0])
	return flow.Definition{Text: raw}, nil
}

// Run renders the message and hands the runner the code to end with.
func (Exit) Run(_ context.Context, graph *flow.Graph, node flow.Node, state flow.State, env flow.Env) (flow.Result, error) {
	// Inline again, and again without a failure to handle; see Define.
	raw, _ := flow.ReadText(graph.Texts, Exit{}.Texts(node)[0])
	message, err := flow.Render(raw, state, env.Params)
	if err != nil {
		return flow.Result{}, err
	}
	code, _ := node.Keys["code"].(int64)
	return flow.Result{Exit: &flow.Exit{Code: int(code), Message: message}}, nil
}
