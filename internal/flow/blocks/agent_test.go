package blocks_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
	"github.com/xidus90/loomux/internal/flow/model"
)

func agentNode(keys map[string]any) flow.Node {
	node := flow.Node{Name: "draft", Kind: "agent", MaxVisits: flow.Cap{Add: 1}, Keys: keys}
	return node
}

func agentKeys() map[string]any {
	return map[string]any{
		"instruction": "instructions/draft.md",
		"tools":       "edit",
		"effort":      "high",
		"reply":       map[string]any{"verdict": "string", "count": "int"},
	}
}

// agentGraph is built per call: its texts are a map, and a test that replaced
// one file would replace it for every test after.
func agentGraph() *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml", Texts: fstest.MapFS{
		"instructions/draft.md": {Data: []byte("Draft {{topic}} in {{max_rounds}} rounds.\n")},
	}}
}

func agentConfig() config.Agent {
	return config.Agent{
		Default: "writer",
		Models:  map[string]config.ModelSpec{"writer": {Provider: "claude", Model: "claude-opus-5"}},
	}
}

// agentState and agentEnv carry what the instruction's placeholders need.
// Render fails on a placeholder without a value, so a test that means to reach
// the model has to bring both halves: {{topic}} from the state, {{max_rounds}}
// from the parameters.
func agentState() flow.State {
	return flow.State{Fields: map[string]flow.Value{"topic": "the spec"}}
}

func agentEnv(m model.Model, agent config.Agent) flow.Env {
	return flow.Env{Model: m, Agent: agent, Params: flow.Params{"max_rounds": 5}}
}

func TestAgentAsksTheModelWhatTheInstructionSays(t *testing.T) {
	fake := model.NewFake(model.Answer{Reply: model.Reply{
		Fields: map[string]any{"verdict": "done", "count": float64(2)},
		Tokens: 41,
		Model:  "claude-opus-5-20260501",
	}})

	got, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), agentState(), agentEnv(fake, agentConfig()))
	if err != nil {
		t.Fatal(err)
	}
	if got.Delta["verdict"] != "done" || got.Delta["count"] != 2 {
		t.Fatalf("delta = %#v", got.Delta)
	}
	if got.Tokens != 41 {
		t.Fatalf("tokens = %d", got.Tokens)
	}
	// The provider in front of the bare name the adapter reported: a journal
	// line that said "claude-opus-5-20260501" alone would not say who answered.
	if got.Model != "claude:claude-opus-5-20260501" {
		t.Fatalf("model = %q", got.Model)
	}

	seen := fake.Seen()
	if len(seen) != 1 {
		t.Fatalf("%d requests", len(seen))
	}
	if seen[0].Prompt != "Draft the spec in 5 rounds.\n" {
		t.Fatalf("prompt = %q", seen[0].Prompt)
	}
	if !slices.Equal(seen[0].Tools, []string{"Edit", "Glob", "Grep", "Read", "Write"}) {
		t.Fatalf("tools = %v", seen[0].Tools)
	}
	if seen[0].Effort != "high" || seen[0].Provider != "claude" || seen[0].Model != "claude-opus-5" {
		t.Fatalf("request = %+v", seen[0])
	}
	if seen[0].Reply["verdict"] != model.ReplyString || seen[0].Reply["count"] != model.ReplyInt {
		t.Fatalf("reply schema = %v", seen[0].Reply)
	}
}

// The adapter may not know which model answered. Then the chain's own answer
// stands, which is what the journal would otherwise have nothing to record.
func TestAgentFallsBackToTheResolvedLabel(t *testing.T) {
	fake := model.NewFake(model.Answer{Reply: model.Reply{
		Fields: map[string]any{"verdict": "done", "count": 1},
	}})
	got, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), agentState(), agentEnv(fake, agentConfig()))
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "claude:claude-opus-5" {
		t.Fatalf("model = %q", got.Model)
	}
}

// No stage of the chain names a model: the adapter passes none and the CLI
// takes its own default. The journal still says which provider was asked.
func TestAgentWithoutAnyModelNamesTheCliDefault(t *testing.T) {
	fake := model.NewFake(model.Answer{Reply: model.Reply{
		Fields: map[string]any{"verdict": "done", "count": 1},
	}})
	got, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), agentState(), agentEnv(fake, config.Agent{}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "claude:cli-default" {
		t.Fatalf("model = %q", got.Model)
	}
	if fake.Seen()[0].Model != "" {
		t.Fatalf("the adapter was handed a model: %q", fake.Seen()[0].Model)
	}
}

func TestAgentTakesTheNodesRoleOverTheFlows(t *testing.T) {
	agent := config.Agent{
		Models: map[string]config.ModelSpec{
			"writer": {Provider: "claude", Model: "claude-opus-5"},
			"cheap":  {Provider: "agy", Model: "gemini"},
		},
		Roles: map[string]string{"planner": "writer", "reviewer": "cheap"},
	}
	graph := agentGraph()
	graph.Role = "planner"
	node := agentNode(agentKeys())
	node.Role = "reviewer"
	fake := model.NewFake(model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "d", "count": 1}}})

	got, err := (blocks.Agent{}).Run(context.Background(), graph, node, agentState(), agentEnv(fake, agent))
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "agy:gemini" {
		t.Fatalf("model = %q", got.Model)
	}
}

// The role goes into the definition next to the model it resolved to. The
// definition hash takes the model, not the role: a changed binding changes
// it, a node's own role changes it through the node's entry, and the flow's
// role only when it resolves to another model.
func TestAgentDefinitionCarriesTheRoleAndTheBoundModel(t *testing.T) {
	graph := &flow.Graph{Texts: fstest.MapFS{"instructions/d.md": {Data: []byte("Draft.")}}, Role: "planner"}
	node := flow.Node{Name: "draft", Kind: "agent", Role: "reviewer",
		Keys: map[string]any{"instruction": "instructions/d.md", "reply": map[string]any{"ok": "bool"}}}
	env := flow.Env{Agent: config.Agent{
		Models: map[string]config.ModelSpec{"gemini": {Provider: "agy", Model: "gemini-3"}},
		Roles:  map[string]string{"reviewer": "gemini"},
	}}
	got, err := blocks.Agent{}.Define(graph, node, env)
	if err != nil || got.Role != "reviewer" || got.Model != "agy:gemini-3" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// A node that names no role plays the flow's, and the definition says so.
func TestAgentWithoutARoleOfItsOwnPlaysTheFlows(t *testing.T) {
	graph := agentGraph()
	graph.Role = "planner"
	agent := agentConfig()
	agent.Models["cheap"] = config.ModelSpec{Provider: "agy", Model: "gemini"}
	agent.Roles = map[string]string{"planner": "cheap"}
	got, err := (blocks.Agent{}).Define(graph, agentNode(agentKeys()), flow.Env{Agent: agent})
	if err != nil || got.Role != "planner" || got.Model != "agy:gemini" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// Every field, and no other. A reply short of one field is a model that did
// not do what it was told, and a reply with one too many is a model that
// answered a different question.
func TestAgentRefusesAReplyThatIsNotExactlyWhatWasAsked(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]any
		want   string
	}{
		{"a field missing", map[string]any{"verdict": "done"}, `no value for "count"`},
		{"a field too many", map[string]any{"verdict": "d", "count": 1, "extra": true}, `unexpected field "extra"`},
		{"a field of the wrong type", map[string]any{"verdict": "d", "count": "two"}, `field "count": cannot read two as int`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := model.NewFake(model.Answer{Reply: model.Reply{Fields: c.fields}})
			_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), agentState(), agentEnv(fake, agentConfig()))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q in it", err, c.want)
			}
		})
	}
}

func TestAgentPassesOnWhatTheModelRefused(t *testing.T) {
	boom := errors.New("the CLI is not installed")
	fake := model.NewFake(model.Answer{Err: boom})
	_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), agentState(), agentEnv(fake, agentConfig()))
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

// A run without a model is refused before the runtime spends anything, and
// not inside a nil call. The runner turns this into an error entry.
func TestAgentWithoutAModelSaysSo(t *testing.T) {
	env := flow.Env{Agent: agentConfig()}
	_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), flow.State{}, env)
	if err == nil || err.Error() != `node "draft" needs a model, and this run has none` {
		t.Fatalf("err = %v", err)
	}
}

// The instruction is read at every visit, so a file deleted between two of them
// is a node failure and not a load finding.
func TestAgentReportsAnInstructionItCannotRead(t *testing.T) {
	keys := agentKeys()
	keys["instruction"] = "instructions/gone.md"
	fake := model.NewFake()
	_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(keys), agentState(), agentEnv(fake, agentConfig()))
	if err == nil || !strings.Contains(err.Error(), "reading instruction instructions/gone.md") {
		t.Fatalf("err = %v", err)
	}
	if len(fake.Seen()) != 0 {
		t.Fatal("the model was asked although the instruction was never read")
	}
}

// A carriage return in an instruction file reaches neither the definition nor
// the prompt: the hash and the question asked would otherwise depend on the
// editor that saved the file.
func TestAgentReadsItsInstructionWithoutCarriageReturns(t *testing.T) {
	graph := agentGraph()
	graph.Texts = fstest.MapFS{
		"instructions/draft.md": {Data: []byte("Draft {{topic}}\r\nin {{max_rounds}} rounds.\r\n")},
	}
	definition, err := (blocks.Agent{}).Define(graph, agentNode(agentKeys()), flow.Env{Agent: agentConfig()})
	if err != nil {
		t.Fatal(err)
	}
	if string(definition.Text) != "Draft {{topic}}\nin {{max_rounds}} rounds.\n" {
		t.Fatalf("text = %q", definition.Text)
	}
	fake := model.NewFake(model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "d", "count": 1}}})
	if _, err := (blocks.Agent{}).Run(context.Background(), graph, agentNode(agentKeys()), agentState(), agentEnv(fake, agentConfig())); err != nil {
		t.Fatal(err)
	}
	if prompt := fake.Seen()[0].Prompt; prompt != "Draft the spec\nin 5 rounds.\n" {
		t.Fatalf("prompt = %q", prompt)
	}
}

// A graph built by hand rather than loaded never went through Check.
func TestAgentDefineReportsAToolProfileItDoesNotKnow(t *testing.T) {
	keys := agentKeys()
	keys["tools"] = "root"
	_, err := (blocks.Agent{}).Define(agentGraph(), agentNode(keys), flow.Env{Agent: agentConfig()})
	if err == nil || !strings.Contains(err.Error(), `unknown tool profile "root"`) {
		t.Fatalf("err = %v", err)
	}
}

// Nothing is asked when the prompt cannot be filled: an instruction with a hole
// in it would be a paid call that asks the wrong question.
func TestAgentReportsAPlaceholderWithNoValue(t *testing.T) {
	fake := model.NewFake()
	_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(agentKeys()), flow.State{}, agentEnv(fake, agentConfig()))
	if err == nil || !strings.Contains(err.Error(), "no value for {{topic}}") {
		t.Fatalf("err = %v", err)
	}
	if len(fake.Seen()) != 0 {
		t.Fatal("the model was asked with an unrendered instruction")
	}
}

func TestAgentDefineFingerprintsEverythingTheNodeWasTold(t *testing.T) {
	got, err := (blocks.Agent{}).Define(agentGraph(), agentNode(agentKeys()), flow.Env{Agent: agentConfig()})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got.Text), "Draft {{topic}}") {
		t.Fatalf("text = %q", got.Text)
	}
	if got.Role != "" || got.Model != "claude:claude-opus-5" || got.Effort != "high" || got.Profile != "edit" {
		t.Fatalf("definition = %+v", got)
	}
	if !slices.Equal(got.Tools, []string{"Edit", "Glob", "Grep", "Read", "Write"}) {
		t.Fatalf("tools = %v", got.Tools)
	}
}

// The config reader refuses a binding to a model it does not know; an Agent
// built by hand has not been through it.
func TestAgentDefineReportsARoleBoundToAModelNobodyDeclared(t *testing.T) {
	agent := agentConfig()
	agent.Roles = map[string]string{"reviewer": "wrter"}
	node := agentNode(agentKeys())
	node.Role = "reviewer"
	_, err := (blocks.Agent{}).Define(agentGraph(), node, flow.Env{Agent: agent})
	if err == nil || !strings.Contains(err.Error(), `model "wrter" is not under [agent.models]`) {
		t.Fatalf("err = %v", err)
	}
}

// Check catches this for a loaded graph; a graph built in Go never went through
// Check, and a reply type nobody knows would otherwise reach the adapter as an
// empty one — a malformed request that is paid for before anyone notices.
func TestAgentRefusesToAskWithAReplyTypeItDoesNotKnow(t *testing.T) {
	keys := agentKeys()
	keys["reply"] = map[string]any{"notes": "list[string]"}
	fake := model.NewFake()
	_, err := (blocks.Agent{}).Run(context.Background(), agentGraph(), agentNode(keys), agentState(), agentEnv(fake, agentConfig()))
	if err == nil || !strings.Contains(err.Error(), `reply field "notes" is list[string]; a reply holds string, int or bool`) {
		t.Fatalf("err = %v", err)
	}
	if len(fake.Seen()) != 0 {
		t.Fatal("the model was asked with a reply type nobody knows")
	}
}

func TestAgentWritesExactlyItsReplyFields(t *testing.T) {
	written := (blocks.Agent{}).Writes(agentNode(agentKeys()))
	if len(written) != 2 || written["verdict"] != flow.String || written["count"] != flow.Int {
		t.Fatalf("writes = %v", written)
	}
}

func TestAgentNamesItsInstructionAsAText(t *testing.T) {
	texts := (blocks.Agent{}).Texts(agentNode(agentKeys()))
	if len(texts) != 1 || texts[0].Key != "instruction" || texts[0].Path != "instructions/draft.md" {
		t.Fatalf("texts = %+v", texts)
	}
	if (blocks.Agent{}).Kind() != "agent" {
		t.Fatal("kind")
	}
	if (blocks.Agent{}).NeedsBaseline() {
		t.Fatal("an agent node reads what its instruction says, not the working tree")
	}
}

func TestAgentCheckFindsEveryProblemAtOnce(t *testing.T) {
	cases := []struct {
		name string
		keys map[string]any
		want []string
	}{
		{"nothing at all", map[string]any{}, []string{
			"instruction must name a file under instructions/",
			"reply must name at least one field with its type",
		}},
		{"a reply type that is no scalar", map[string]any{
			"instruction": "i.md", "reply": map[string]any{"notes": "list[string]"},
		}, []string{`reply field "notes" is list[string]; a reply holds string, int or bool`}},
		{"an unknown tool profile", map[string]any{
			"instruction": "i.md", "reply": map[string]any{"v": "string"}, "tools": "root",
		}, []string{`unknown tool profile "root"; known profiles: edit, mcp, read_only, shell`}},
		// Not a silent default: tools = 3 would read as read_only, and effort = 3
		// as no effort at all -- one of the inputs of the definition hash.
		{"tools and an effort that are not text", map[string]any{
			"instruction": "i.md", "reply": map[string]any{"v": "string"}, "tools": int64(3), "effort": int64(3),
		}, []string{"tools is 3, not text", "effort is 3, not text"}},
		{"an unknown key", map[string]any{
			"instruction": "i.md", "reply": map[string]any{"v": "string"}, "instructon": "i.md",
		}, []string{`unknown key "instructon"; known keys: effort, instruction, kind, max_visits, name, reply, role, tools`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := (blocks.Agent{}).Check(agentNode(c.keys))
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

// The default, so that a node that says nothing about tools cannot write.
func TestAgentWithoutToolsIsReadOnly(t *testing.T) {
	keys := agentKeys()
	delete(keys, "tools")
	got, err := (blocks.Agent{}).Define(agentGraph(), agentNode(keys), flow.Env{Agent: agentConfig()})
	if err != nil {
		t.Fatal(err)
	}
	if got.Profile != "read_only" || !slices.Equal(got.Tools, []string{"Glob", "Grep", "Read"}) {
		t.Fatalf("definition = %+v", got)
	}
}

func TestAgentPassesTheConfiguredMcpServers(t *testing.T) {
	keys := agentKeys()
	keys["tools"] = "mcp"
	agent := agentConfig()
	agent.MCPServers = []string{"brain"}
	got, err := (blocks.Agent{}).Define(agentGraph(), agentNode(keys), flow.Env{Agent: agent})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Tools, []string{"Glob", "Grep", "Read", "mcp__brain"}) {
		t.Fatalf("tools = %v", got.Tools)
	}
}
