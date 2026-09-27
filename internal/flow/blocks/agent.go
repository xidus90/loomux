package blocks

import (
	"context"
	"fmt"
	"slices"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/model"
)

// defaultProfile is what a node that says nothing about tools gets. Read-only,
// because the profiles are the only ceiling between a node and Write, and a
// node that forgot to say so has not asked for one.
const defaultProfile = "read_only"

// replyTypes maps a flow's types onto what a reply may hold. A reply is flat
// on purpose: a model that has to fill a list is a model that has to be told
// what a list looks like in its answer format, and every adapter would tell it
// differently.
var replyTypes = map[string]model.ReplyType{
	string(flow.String): model.ReplyString,
	string(flow.Int):    model.ReplyInt,
	string(flow.Bool):   model.ReplyBool,
}

// Agent is the node kind that renders an instruction, asks a model and checks
// the answer against the fields the node declared.
type Agent struct{}

// Kind is "agent".
func (Agent) Kind() string { return "agent" }

// Check reads the agent's own keys.
func (Agent) Check(node flow.Node) []string {
	var findings []string
	if node.StringKey("instruction") == "" {
		findings = append(findings, "instruction must name a file under instructions/")
	}
	reply, _ := node.Keys["reply"].(map[string]any)
	if len(reply) == 0 {
		findings = append(findings, "reply must name at least one field with its type")
	}
	for _, name := range sortedNames(reply) {
		declared, _ := reply[name].(string)
		if _, ok := replyTypes[declared]; !ok {
			findings = append(findings,
				fmt.Sprintf("reply field %q is %s; a reply holds string, int or bool", name, declared))
		}
	}
	if _, err := model.Tools(agentProfile(node), nil); err != nil {
		findings = append(findings, err.Error())
	}
	// Both are read with StringKey, which turns a value of another type into
	// nothing: tools = 3 would quietly become the default profile, and
	// effort = 3 no effort -- a changed input of the definition hash nobody
	// was told about.
	for _, key := range []string{"tools", "effort"} {
		if raw, present := node.Keys[key]; present {
			if _, ok := raw.(string); !ok {
				findings = append(findings, fmt.Sprintf("%s is %#v, not text", key, raw))
			}
		}
	}
	findings = append(findings, node.KeyFindings("instruction", "reply", "tools", "effort")...)
	return findings
}

// Texts names the instruction file.
func (Agent) Texts(node flow.Node) []flow.Text {
	return []flow.Text{{Key: "instruction", Path: node.StringKey("instruction")}}
}

// Writes are the reply's fields, with the types the node declared for them.
func (Agent) Writes(node flow.Node) map[string]flow.Type {
	reply, _ := node.Keys["reply"].(map[string]any)
	written := make(map[string]flow.Type, len(reply))
	for name, declared := range reply {
		text, _ := declared.(string)
		written[name] = flow.Type(text)
	}
	return written
}

// NeedsBaseline is false: an agent node reads what its instruction says.
func (Agent) NeedsBaseline() bool { return false }

// Define is everything the node's result depends on besides its input: the
// instruction's bytes, the role and the model it resolved to, the effort and
// the tool list.
func (Agent) Define(graph *flow.Graph, node flow.Node, env flow.Env) (flow.Definition, error) {
	definition, _, err := agentDefine(graph, node, env)
	return definition, err
}

// agentDefine is Define, and hands the resolved model back besides. Run needs
// the provider and the bare name the definition only carries as one label, and
// resolving a second time would be a second chance to fail where the first has
// already decided.
func agentDefine(graph *flow.Graph, node flow.Node, env flow.Env) (flow.Definition, model.Resolved, error) {
	raw, err := flow.ReadText(graph.Texts, Agent{}.Texts(node)[0])
	if err != nil {
		return flow.Definition{}, model.Resolved{}, err
	}
	resolved, err := model.Resolve(env.Agent, node.Role, graph.Role)
	if err != nil {
		return flow.Definition{}, model.Resolved{}, err
	}
	profile := agentProfile(node)
	tools, err := model.Tools(profile, env.Agent.MCPServers)
	if err != nil {
		return flow.Definition{}, model.Resolved{}, err
	}
	return flow.Definition{
		Text:    raw,
		Role:    resolved.Role,
		Model:   resolved.Label(),
		Effort:  node.StringKey("effort"),
		Profile: profile,
		Tools:   tools,
	}, resolved, nil
}

// Run renders the instruction, asks the model and turns the answer into a delta.
func (Agent) Run(ctx context.Context, graph *flow.Graph, node flow.Node, state flow.State, env flow.Env) (flow.Result, error) {
	if env.Model == nil {
		// Named here and not only at the command line: a run that got this far
		// without an adapter would otherwise fail inside a nil call.
		return flow.Result{}, fmt.Errorf("node %q needs a model, and this run has none", node.Name)
	}
	definition, resolved, err := agentDefine(graph, node, env)
	if err != nil {
		return flow.Result{}, err
	}
	prompt, err := flow.Render(definition.Text, state, env.Params)
	if err != nil {
		return flow.Result{}, err
	}

	written := Agent{}.Writes(node)
	schema, err := agentSchema(written)
	if err != nil {
		return flow.Result{}, err
	}

	reply, err := env.Model.Ask(ctx, model.Request{
		Prompt:   prompt,
		Tools:    definition.Tools,
		Effort:   definition.Effort,
		Provider: resolved.Provider,
		Model:    resolved.Model,
		Reply:    schema,
	})
	if err != nil {
		return flow.Result{}, err
	}

	delta, err := agentDelta(reply.Fields, written)
	if err != nil {
		return flow.Result{}, err
	}
	label := definition.Model
	if reply.Model != "" {
		// The adapter reports a bare name; the journal records who answered.
		label = resolved.Provider + ":" + reply.Model
	}
	return flow.Result{Delta: delta, Tokens: reply.Tokens, Model: label}, nil
}

// agentSchema is the answer format the request carries, one reply type per
// declared field.
//
// A type the table does not know is an error and not a zero value: Check
// catches it for a loaded graph, but a graph built in Go never went through
// Check, and an empty type would reach the adapter as a malformed request that
// is paid for before anyone notices.
func agentSchema(written map[string]flow.Type) (map[string]model.ReplyType, error) {
	schema := make(map[string]model.ReplyType, len(written))
	for _, name := range sortedTypes(written) {
		declared := written[name]
		replyType, ok := replyTypes[string(declared)]
		if !ok {
			return nil, fmt.Errorf("reply field %q is %s; a reply holds string, int or bool", name, string(declared))
		}
		schema[name] = replyType
	}
	return schema, nil
}

// agentDelta checks the answer against the declared fields and converts every
// value into the Go type its type stands for. Every field, and no other.
func agentDelta(fields map[string]any, written map[string]flow.Type) (flow.Delta, error) {
	delta := make(flow.Delta, len(written))
	for _, name := range sortedTypes(written) {
		raw, ok := fields[name]
		if !ok {
			return nil, fmt.Errorf("no value for %q", name)
		}
		value, err := flow.Coerce(written[name], raw)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", name, err)
		}
		delta[name] = value
	}
	for _, name := range sortedNames(fields) {
		if _, ok := written[name]; !ok {
			return nil, fmt.Errorf("unexpected field %q", name)
		}
	}
	return delta, nil
}

// sortedNames and sortedTypes exist so that a message about a reply with two
// mistakes in it names the same one twice running.
func sortedNames(table map[string]any) []string {
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func sortedTypes(table map[string]flow.Type) []string {
	names := make([]string, 0, len(table))
	for name := range table {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// agentProfile is the node's tool profile, or the default when it names none.
func agentProfile(node flow.Node) string {
	if profile := node.StringKey("tools"); profile != "" {
		return profile
	}
	return defaultProfile
}
