package blocks

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/flow"
)

// Gate is the node kind that stops a run and asks a person. It matches the
// answer against its choices and runs none of the flow's own code on it: a
// decision point that can run arbitrary code is a decision point nobody can
// read.
type Gate struct{}

// Kind is "gate".
func (Gate) Kind() string { return "gate" }

// Check reads the gate's own keys.
func (Gate) Check(node flow.Node) []string {
	var findings []string
	if node.StringKey("question") == "" {
		findings = append(findings, "question must name a file under questions/")
	}
	if node.StringKey("answer") == "" {
		findings = append(findings, "answer must name a state field")
	}
	choices, ok := gateChoices(node)
	if !ok {
		findings = append(findings, "choices must be a list of at least two texts")
	}
	for index, choice := range choices {
		// A choice is matched by equality or as a prefix before ":" or space.
		// One that holds either of those separators could not be told from a
		// choice plus its reason.
		if strings.ContainsAny(choice, " \t\n:") {
			findings = append(findings, fmt.Sprintf("choice %q must not hold whitespace or \":\"", choice))
		}
		if slices.Index(choices, choice) != index {
			findings = append(findings, fmt.Sprintf("choice %q is listed twice", choice))
		}
	}
	findings = append(findings, node.KeyFindings("question", "choices", "answer")...)
	return findings
}

// Texts names the question file.
func (Gate) Texts(node flow.Node) []flow.Text {
	return []flow.Text{{Key: "question", Path: node.StringKey("question")}}
}

// Writes names the answer field and the field that carries the rest of the
// answer, both strings.
func (Gate) Writes(node flow.Node) map[string]flow.Type {
	answer := node.StringKey("answer")
	return map[string]flow.Type{answer: flow.String, answer + "_text": flow.String}
}

// NeedsBaseline is false: a gate asks a person, not the working tree.
func (Gate) NeedsBaseline() bool { return false }

// Define is the question's bytes and nothing else. A gate calls no model, so
// there is no model, effort or tool list to fingerprint.
func (Gate) Define(graph *flow.Graph, node flow.Node, _ flow.Env) (flow.Definition, error) {
	raw, err := flow.ReadText(graph.Texts, Gate{}.Texts(node)[0])
	if err != nil {
		return flow.Definition{}, err
	}
	return flow.Definition{Text: raw}, nil
}

// Run asks, or takes the answer a resume brought.
func (Gate) Run(_ context.Context, graph *flow.Graph, node flow.Node, state flow.State, env flow.Env) (flow.Result, error) {
	if env.Answer == nil {
		raw, err := flow.ReadText(graph.Texts, Gate{}.Texts(node)[0])
		if err != nil {
			return flow.Result{}, err
		}
		question, err := flow.Render(raw, state, env.Params)
		if err != nil {
			return flow.Result{}, err
		}
		return flow.Result{Question: &question}, nil
	}

	answer := strings.TrimSpace(*env.Answer)
	choices, _ := gateChoices(node)
	field := node.StringKey("answer")
	for _, choice := range choices {
		rest, ok := gateMatch(answer, choice)
		if !ok {
			continue
		}
		return flow.Result{Delta: flow.Delta{field: choice, field + "_text": rest}}, nil
	}
	return flow.Result{}, fmt.Errorf("%w; the choices are %s", flow.ErrInvalidAnswer, strings.Join(choices, ", "))
}

// gateMatch reports whether the answer is this choice, and what it said beyond
// it. Upper and lower case count: "no" and "No" are not quietly the same word.
func gateMatch(answer, choice string) (string, bool) {
	if answer == choice {
		return "", true
	}
	if !strings.HasPrefix(answer, choice) {
		return "", false
	}
	rest := answer[len(choice):]
	if !strings.ContainsAny(rest[:1], " \t\n:") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(rest, ":")), true
}

// gateChoices reads the choices as TOML decoded them. It reports false for
// anything that is not a list of at least two texts, so that Check has one
// finding to make and Run has a list it can trust.
func gateChoices(node flow.Node) ([]string, bool) {
	raw, ok := node.Keys["choices"].([]any)
	if !ok || len(raw) < 2 {
		return nil, false
	}
	choices := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		choices = append(choices, text)
	}
	return choices, true
}
