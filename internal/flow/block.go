package flow

import (
	"context"
	"errors"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow/model"
)

// ErrInvalidAnswer marks a gate answer that matches none of the gate's choices.
// The runner refuses such a resume and leaves the gate open; it is not a node
// failure and takes no error edge.
var ErrInvalidAnswer = errors.New("the answer matches none of the choices")

// Text is one text a node renders: a file among the flow's texts, or an inline
// text.
type Text struct {
	Key    string // the node key the text came from, e.g. "instruction"
	Path   string // a path in Graph.Texts; empty for an inline text
	Inline string // the text itself when Path is empty
}

// Definition is what a node's result depends on besides its input. The runner
// hashes it into the journal's definition_hash.
type Definition struct {
	Text    []byte   // the raw bytes of the instruction or question; nil when there is none
	Role    string   // the role the node plays; empty without one
	Model   string   // "<provider>:<model>" or "<provider>:cli-default"; empty without a model
	Effort  string   // empty when the node has none
	Profile string   // the tool profile name the journal records; empty when there is none
	Tools   []string // the resolved tool list; nil when there is none
}

// Env is what a running block may use besides the graph, the node and the state.
type Env struct {
	Root     string       // the project root
	Params   Params       // the run's parameters
	Answer   *string      // the answer a resume brought for this gate; nil otherwise
	Baseline *Baseline    // nil when the run has none
	Model    model.Model  // nil when the run has no model
	Agent    config.Agent // the [agent] table a node's role resolves against
}

// Result is what one execution of a block produced.
type Result struct {
	Delta    Delta
	Tokens   int
	Model    string  // the model that answered, as the journal records it; empty without a model
	Question *string // non-nil: the node pauses with this question and the run ends paused
	Exit     *Exit   // non-nil: the run ends with this code and message
}

// Exit is a block's decision to end the run with a code of its own.
type Exit struct {
	Code    int
	Message string
}

// Block implements one node kind. The runtime knows no block by name; it finds
// them through a Registry.
type Block interface {
	// Kind is the value of `kind` this block implements.
	Kind() string
	// Check returns one finding per problem with the node's own keys.
	Check(node Node) []string
	// Texts names every text the node renders, so placeholders can be checked
	// before a run.
	Texts(node Node) []Text
	// Writes names the state fields the node writes, with their types.
	Writes(node Node) map[string]Type
	// NeedsBaseline reports whether a node of this kind needs the run's baseline.
	NeedsBaseline() bool
	// Define describes what the node's result depends on besides its input.
	Define(graph *Graph, node Node, env Env) (Definition, error)
	// Run executes the node once.
	Run(ctx context.Context, graph *Graph, node Node, state State, env Env) (Result, error)
}

// Registry finds blocks and predicates by name.
type Registry interface {
	Block(kind string) (Block, bool)
	Predicate(name string) (Predicate, bool)
}
