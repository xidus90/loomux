package blocks_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
)

func gateNode(keys map[string]any) flow.Node {
	return flow.Node{Name: "approve", Kind: "gate", MaxVisits: flow.Cap{Add: 1}, Keys: keys}
}

// gateGraph is built per call, like agentGraph.
func gateGraph() *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml", Texts: fstest.MapFS{
		"questions/approve.md": {Data: []byte("Approve after {{count}} rounds?\n")},
	}}
}

var gateKeys = map[string]any{
	"question": "questions/approve.md",
	"choices":  []any{"yes", "no"},
	"answer":   "answer",
}

func TestGateAsksItsQuestionAndPauses(t *testing.T) {
	state := flow.State{Fields: map[string]flow.Value{"count": 2}}
	got, err := blocks.Gate{}.Run(context.Background(), gateGraph(), gateNode(gateKeys), state, flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Question == nil {
		t.Fatal("a gate without an answer pauses")
	}
	// Render copies the question's bytes as they are, the closing newline
	// included.
	if *got.Question != "Approve after 2 rounds?\n" {
		t.Fatalf("question = %q", *got.Question)
	}
	if got.Delta != nil || got.Exit != nil {
		t.Fatalf("a pause changes nothing: %+v", got)
	}
}

// A carriage return in a question file reaches neither the definition nor the
// question a person is shown, as for an agent's instruction.
func TestGateReadsItsQuestionWithoutCarriageReturns(t *testing.T) {
	graph := gateGraph()
	graph.Texts = fstest.MapFS{"questions/approve.md": {Data: []byte("Approve\r\nafter {{count}} rounds?\r\n")}}
	definition, err := blocks.Gate{}.Define(graph, gateNode(gateKeys), flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if string(definition.Text) != "Approve\nafter {{count}} rounds?\n" {
		t.Fatalf("text = %q", definition.Text)
	}
	state := flow.State{Fields: map[string]flow.Value{"count": 2}}
	got, err := blocks.Gate{}.Run(context.Background(), graph, gateNode(gateKeys), state, flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Question == nil || *got.Question != "Approve\nafter 2 rounds?\n" {
		t.Fatalf("result = %+v", got)
	}
}

func TestGateSplitsTheAnswerIntoChoiceAndRest(t *testing.T) {
	cases := []struct{ answer, choice, rest string }{
		{"yes", "yes", ""},
		{"no", "no", ""},
		{"no: the spec is thin", "no", "the spec is thin"},
		{"no the spec is thin", "no", "the spec is thin"},
		{"  yes  ", "yes", ""},
		{"no:\tstill open\n", "no", "still open"},
	}
	for _, c := range cases {
		t.Run(c.answer, func(t *testing.T) {
			answer := c.answer
			env := flow.Env{Answer: &answer}
			got, err := blocks.Gate{}.Run(context.Background(), gateGraph(), gateNode(gateKeys), flow.State{}, env)
			if err != nil {
				t.Fatal(err)
			}
			if got.Question != nil {
				t.Fatal("an answered gate does not pause again")
			}
			if got.Delta["answer"] != c.choice || got.Delta["answer_text"] != c.rest {
				t.Fatalf("delta = %#v", got.Delta)
			}
		})
	}
}

// Case matters, so that "no" and "No" cannot quietly mean the same thing: a
// gate is the place where a person's word is taken literally or not at all.
func TestGateRefusesAnAnswerThatIsNoChoice(t *testing.T) {
	for _, answer := range []string{"No", "yesterday", "maybe", "", "ye"} {
		t.Run(answer, func(t *testing.T) {
			text := answer
			env := flow.Env{Answer: &text}
			_, err := blocks.Gate{}.Run(context.Background(), gateGraph(), gateNode(gateKeys), flow.State{}, env)
			if !errors.Is(err, flow.ErrInvalidAnswer) {
				t.Fatalf("err = %v, want ErrInvalidAnswer", err)
			}
			if !strings.Contains(err.Error(), "yes, no") {
				t.Fatalf("the message names the choices: %v", err)
			}
		})
	}
}

func TestGateWritesTheAnswerFieldAndItsText(t *testing.T) {
	written := blocks.Gate{}.Writes(gateNode(gateKeys))
	if written["answer"] != flow.String || written["answer_text"] != flow.String {
		t.Fatalf("writes = %v", written)
	}
	if len(written) != 2 {
		t.Fatalf("writes = %v", written)
	}
}

func TestGateNamesItsQuestionAsAText(t *testing.T) {
	texts := blocks.Gate{}.Texts(gateNode(gateKeys))
	if len(texts) != 1 || texts[0].Key != "question" || texts[0].Path != "questions/approve.md" {
		t.Fatalf("texts = %+v", texts)
	}
}

func TestGateDefineCarriesTheQuestionBytesAndNoModel(t *testing.T) {
	got, err := blocks.Gate{}.Define(gateGraph(), gateNode(gateKeys), flow.Env{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got.Text), "Approve after {{count}}") {
		t.Fatalf("text = %q", got.Text)
	}
	if got.Model != "" || got.Effort != "" || got.Profile != "" || got.Tools != nil {
		t.Fatalf("a gate has no model and no tools: %+v", got)
	}
}

func TestGateCheckFindsEveryProblemAtOnce(t *testing.T) {
	cases := []struct {
		name string
		keys map[string]any
		want []string
	}{
		{"nothing at all", map[string]any{}, []string{
			"question must name a file under questions/",
			"answer must name a state field",
			"choices must be a list of at least two texts",
		}},
		{"an unknown key", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes", "no"}, "choicez": "x",
		}, []string{`unknown key "choicez"; known keys: answer, choices, kind, max_visits, name, question, role`}},
		{"a choice that is not one word", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes please", "no"},
		}, []string{`choice "yes please" must not hold whitespace or ":"`}},
		{"a choice twice", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes", "yes"},
		}, []string{`choice "yes" is listed twice`}},
		{"a choice that is not a text", map[string]any{
			"question": "q.md", "answer": "a", "choices": []any{"yes", 2},
		}, []string{"choices must be a list of at least two texts"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := blocks.Gate{}.Check(gateNode(c.keys))
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

func TestGateOfAWellFormedNodeHasNothingToSay(t *testing.T) {
	// The parentheses are not decoration: `if blocks.Gate{}.Kind()` parses the
	// brace as the start of the if's body.
	if got := (blocks.Gate{}).Check(gateNode(gateKeys)); got != nil {
		t.Fatalf("got %v", got)
	}
	if (blocks.Gate{}).Kind() != "gate" {
		t.Fatal("kind")
	}
	if (blocks.Gate{}).NeedsBaseline() {
		t.Fatal("a gate asks a person, not the working tree")
	}
}

func TestGateReportsAQuestionItCannotRead(t *testing.T) {
	node := gateNode(map[string]any{"question": "questions/nope.md", "choices": []any{"yes", "no"}, "answer": "answer"})
	if _, err := (blocks.Gate{}).Run(context.Background(), gateGraph(), node, flow.State{}, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
	if _, err := (blocks.Gate{}).Define(gateGraph(), node, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
}

func TestGateReportsAQuestionItCannotRender(t *testing.T) {
	graph := gateGraph()
	graph.Texts = fstest.MapFS{"questions/broken.md": {Data: []byte("Ask {{ name }} first.\n")}}
	node := gateNode(map[string]any{"question": "questions/broken.md", "choices": []any{"yes", "no"}, "answer": "answer"})
	if _, err := (blocks.Gate{}).Run(context.Background(), graph, node, flow.State{}, flow.Env{}); err == nil {
		t.Fatal("want an error")
	}
}
