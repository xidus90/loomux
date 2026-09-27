package runner_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"
	"time"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/model"
	"github.com/xidus90/loomux/internal/flow/runner"
)

// memJournal is the journal a test run writes into. A file would test the file
// system; what is under test is what the runner writes, and when.
type memJournal struct{ lines []journal.Entry }

func (j *memJournal) Entries() ([]journal.Entry, error) { return j.lines, nil }

func (j *memJournal) Append(entry journal.Entry) error {
	j.lines = append(j.lines, entry)
	return nil
}

func (j *memJournal) Pending() (*journal.PendingGate, error) {
	var open *journal.PendingGate
	for _, entry := range j.lines {
		switch entry.Outcome {
		case "paused":
			detail := ""
			if entry.Detail != nil {
				detail = *entry.Detail
			}
			open = &journal.PendingGate{Node: entry.Node, Question: detail, InputHash: entry.InputHash}
		case "ok":
			if open != nil && open.Node == entry.Node && open.InputHash == entry.InputHash {
				open = nil
			}
		}
	}
	return open, nil
}

// stepBlock is a node that does one thing: apply a fixed delta, or fail, or
// end the run. Everything the runner is asked about is visible in what it
// journals afterwards.
type stepBlock struct {
	delta flow.Delta
	err   error
	exit  *flow.Exit
	runs  *int
	model string
}

func (stepBlock) Kind() string                { return "step" }
func (stepBlock) Check(flow.Node) []string    { return nil }
func (stepBlock) Texts(flow.Node) []flow.Text { return nil }
func (stepBlock) NeedsBaseline() bool         { return false }

func (b stepBlock) Writes(flow.Node) map[string]flow.Type {
	written := make(map[string]flow.Type, len(b.delta))
	for name := range b.delta {
		written[name] = flow.String
	}
	return written
}

func (stepBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{Text: []byte("v1")}, nil
}

func (b stepBlock) Run(context.Context, *flow.Graph, flow.Node, flow.State, flow.Env) (flow.Result, error) {
	if b.runs != nil {
		*b.runs++
	}
	if b.err != nil {
		return flow.Result{}, b.err
	}
	if b.exit != nil {
		return flow.Result{Exit: b.exit}, nil
	}
	return flow.Result{Delta: b.delta, Tokens: 7, Model: b.model}, nil
}

// baselineBlock is a step that needs the run's baseline, so that a run without
// one can be refused before the node spends anything finding that out.
type baselineBlock struct{ stepBlock }

func (baselineBlock) NeedsBaseline() bool { return true }

// fixBlock is a step of a kind of its own, so that a fallback can succeed where
// the node before it failed.
type fixBlock struct{ stepBlock }

func (fixBlock) Kind() string { return "fix" }

// fieldIs is the smallest condition an edge can carry: one field, one value.
// The loader parses real conditions; this lane only needs one that decides.
func fieldIs(name, want string) flow.Condition {
	return fieldEquals{name: name, want: want}
}

type fieldEquals struct {
	name string
	want string
}

func (c fieldEquals) Holds(state flow.State, _ flow.Params) bool {
	return state.Fields[c.name] == c.want
}

// ticks hands out a clock that moves half a second per call, so a duration in
// the journal is a number a test can name.
func ticks() runner.Clock {
	now := time.Unix(0, 0)
	return func() time.Time {
		now = now.Add(500 * time.Millisecond)
		return now
	}
}

func node(name string, visits int) flow.Node {
	// Cap{} means no visit at all, so every hand-built node says what it allows.
	return flow.Node{Name: name, Kind: "step", MaxVisits: flow.Cap{Add: visits}}
}

func graphOf(nodes []flow.Node, edges []flow.Edge, state map[string]flow.Field) *flow.Graph {
	return &flow.Graph{Name: "f", File: "f.toml", Start: nodes[0].Name, Nodes: nodes, Edges: edges, State: state}
}

func catalogOf(t *testing.T, blocks ...flow.Block) *flow.Catalog {
	t.Helper()
	made, err := flow.NewCatalog(blocks, nil)
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func TestARunWalksToEndAndJournalsEveryStep(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("b", 1)},
		[]flow.Edge{{From: "a", To: "b"}, {From: "b", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	run := runner.New(runner.Options{
		Graph:   graph,
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}}),
		Journal: log,
		Clock:   ticks(),
	})

	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("status = %q, detail = %q", got.Status, got.Detail)
	}
	if len(log.lines) != 2 {
		t.Fatalf("%d entries", len(log.lines))
	}
	if log.lines[0].Node != "a" || log.lines[0].Outcome != "ok" || log.lines[0].Tokens != 7 {
		t.Fatalf("entry = %+v", log.lines[0])
	}
	if log.lines[0].Seconds != 0.5 {
		t.Fatalf("seconds = %v", log.lines[0].Seconds)
	}
	if log.lines[0].DefinitionHash == nil || *log.lines[0].DefinitionHash == "" {
		t.Fatal("every entry carries a definition hash")
	}
	if log.lines[0].Model != nil {
		t.Fatal("a block without a model records none")
	}
	if got.State.Fields["note"] != "done" {
		t.Fatalf("state = %#v", got.State.Fields)
	}
}

// The role is what the node's definition resolved it to, so it is recorded on
// the agent's entry and is null on the gate's: a gate asks no model and plays
// no role.
func TestAnEntryRecordsTheRoleItsNodePlayed(t *testing.T) {
	log := &memJournal{}
	graph := &flow.Graph{
		Name: "f", File: "f.toml", Start: "review",
		Texts: fstest.MapFS{
			"instructions/review.md": {Data: []byte("Review the change.")},
			"questions/ship.md":      {Data: []byte("Ship it?")},
		},
		Nodes: []flow.Node{
			{
				Name: "review", Kind: "agent", Role: "reviewer", MaxVisits: flow.Cap{Add: 1},
				Keys: map[string]any{"instruction": "instructions/review.md", "reply": map[string]any{"verdict": "string"}},
			},
			{
				Name: "ask", Kind: "gate", MaxVisits: flow.Cap{Add: 1},
				Keys: map[string]any{"question": "questions/ship.md", "answer": "decision", "choices": []any{"yes", "no"}},
			},
		},
		Edges: []flow.Edge{{From: "review", To: "ask"}, {From: "ask", To: flow.End}},
		State: map[string]flow.Field{
			"verdict":       {Type: flow.String, Default: ""},
			"decision":      {Type: flow.String, Default: ""},
			"decision_text": {Type: flow.String, Default: ""},
		},
	}
	fake := model.NewFake(model.Answer{Reply: model.Reply{Fields: map[string]any{"verdict": "fine"}}})
	run := runner.New(runner.Options{
		Graph:   graph,
		Catalog: catalogOf(t, blocks.Agent{}, blocks.Gate{}),
		Journal: log,
		Env:     flow.Env{Model: fake, Agent: config.Agent{}},
		Clock:   ticks(),
	})

	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paused" || len(log.lines) != 2 {
		t.Fatalf("result = %+v, entries = %+v", got, log.lines)
	}
	if role := log.lines[0].Role; role == nil || *role != "reviewer" {
		t.Fatalf("the agent's role = %v", role)
	}
	if role := log.lines[1].Role; role != nil {
		t.Fatalf("the gate's role = %q", *role)
	}
}

// File order, and a condition that holds decides. An edge without a condition
// always holds, which is why it is written last.
func TestTheFirstEdgeWhoseConditionHoldsIsTaken(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("yes", 1), node("no", 1)},
		[]flow.Edge{
			{From: "a", To: "yes", When: fieldIs("note", "done")},
			{From: "a", To: "no"},
			{From: "yes", To: flow.End},
			{From: "no", To: flow.End},
		},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	run := runner.New(runner.Options{
		Graph:   graph,
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}}),
		Journal: log,
		Clock:   ticks(),
	})
	if _, err := run.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if log.lines[1].Node != "yes" {
		t.Fatalf("went to %q", log.lines[1].Node)
	}
}

func TestARunWithNoEdgeThatAppliesEndsWithAnError(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("b", 1)},
		[]flow.Edge{{From: "a", To: "b", When: fieldIs("note", "never")}, {From: "b", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	run := runner.New(runner.Options{Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Detail != `no edge out of "a" applies to the current state` {
		t.Fatalf("result = %+v", got)
	}
}

// The runaway-loop guard. 0 tokens and 0 seconds, because nothing ran.
func TestANodeOverItsCapEndsTheRun(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 2)},
		[]flow.Edge{{From: "a", To: "a"}},
		map[string]flow.Field{"n": {Type: flow.Int, Default: 0}},
	)
	run := runner.New(runner.Options{Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Detail != `node "a" exceeded max_visits=2` {
		t.Fatalf("result = %+v", got)
	}
	last := log.lines[len(log.lines)-1]
	if last.Outcome != "error" || last.Tokens != 0 || last.Seconds != 0 {
		t.Fatalf("entry = %+v", last)
	}
}

func TestANodeThatFailsTakesItsErrorEdge(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("fix", 1)},
		[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: "fix", OnError: true}, {From: "fix", To: flow.End}},
		nil,
	)
	catalog := catalogOf(t, stepBlock{err: errors.New("the tool is gone")})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Both nodes are the same block here, so "fix" fails too and there is no
	// second error edge: the run ends where the fallback ran out.
	if got.Status != "error" || got.Node != "fix" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 2 || log.lines[0].Outcome != "error" {
		t.Fatalf("entries = %+v", log.lines)
	}
	if log.lines[0].Detail == nil || *log.lines[0].Detail != "the tool is gone" {
		t.Fatalf("detail = %v", log.lines[0].Detail)
	}
}

// An exit is an ending with a reason, not a failure: no error edge is offered,
// because a fallback that swallowed the code would leave the caller with 1.
func TestAnExitEndsTheRunWithItsCodeAndSkipsTheErrorEdge(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), node("fix", 1)},
		[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: "fix", OnError: true}, {From: "fix", To: flow.End}},
		nil,
	)
	catalog := catalogOf(t, stepBlock{exit: &flow.Exit{Code: 4, Message: "rejected"}})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.ExitCode == nil || *got.ExitCode != 4 || got.Detail != "rejected" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 1 || log.lines[0].Outcome != "error" {
		t.Fatalf("entries = %+v", log.lines)
	}
}

func TestAStartIsRefusedBeforeAnyEntryIsWritten(t *testing.T) {
	cases := []struct {
		name  string
		graph *flow.Graph
		env   flow.Env
		want  string
	}{
		{
			name:  "a cap that is zero",
			graph: graphOf([]flow.Node{node("a", 0)}, []flow.Edge{{From: "a", To: flow.End}}, nil),
		},
		{
			name: "a cap whose parameter the run does not have",
			graph: graphOf(
				[]flow.Node{{Name: "a", Kind: "step", MaxVisits: flow.Cap{Param: "rounds", Add: 1}}},
				[]flow.Edge{{From: "a", To: flow.End}}, nil),
		},
		{
			name: "a cap that a parameter makes zero",
			graph: graphOf(
				[]flow.Node{{Name: "a", Kind: "step", MaxVisits: flow.Cap{Param: "rounds"}}},
				[]flow.Edge{{From: "a", To: flow.End}}, nil),
			env: flow.Env{Params: flow.Params{"rounds": 0}},
		},
		{
			name:  "a kind for which the catalog has no block",
			graph: graphOf([]flow.Node{{Name: "a", Kind: "widget", MaxVisits: flow.Cap{Add: 1}}}, []flow.Edge{{From: "a", To: flow.End}}, nil),
		},
		{
			name:  "a declared parameter the run does not have",
			graph: declaring(flow.Int),
			want:  `parameter "rounds" is declared, and the run does not have it`,
		},
		{
			name:  "a declared parameter of another Go type than the declared one",
			graph: declaring(flow.Int),
			env:   flow.Env{Params: flow.Params{"rounds": int64(3)}},
			want:  `parameter "rounds" is declared as int and holds int64`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := &memJournal{}
			run := runner.New(runner.Options{
				Graph: c.graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Env: c.env, Clock: ticks(),
			})
			_, err := run.Run(context.Background())
			if err == nil {
				t.Fatal("want a refusal")
			}
			if c.want != "" && err.Error() != c.want {
				t.Fatalf("err = %v", err)
			}
			if len(log.lines) != 0 {
				t.Fatalf("a refused run wrote %d entries", len(log.lines))
			}
		})
	}
}

// declaring is a one-node flow that declares the parameter rounds and never
// reads it. A condition that read it would panic on a missing one; the refusal
// has to come before any edge is evaluated.
func declaring(typ flow.Type) *flow.Graph {
	graph := graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil)
	graph.Params = map[string]flow.Field{"rounds": {Type: typ}}
	return graph
}

// A parameter the run holds in the declared Go type starts the run: the
// refusal above is about the value, not about declaring one.
func TestARunStartsWithEveryDeclaredParameterInItsType(t *testing.T) {
	cases := []struct {
		typ   flow.Type
		value flow.Value
	}{
		{flow.Int, 3},
		{flow.String, "x"},
		{flow.Bool, true},
		{flow.StringList, []string{"a"}},
	}
	for _, c := range cases {
		t.Run(string(c.typ), func(t *testing.T) {
			run := runner.New(runner.Options{
				Graph: declaring(c.typ), Catalog: catalogOf(t, stepBlock{}), Journal: &memJournal{},
				Env: flow.Env{Params: flow.Params{"rounds": c.value}}, Clock: ticks(),
			})
			got, err := run.Run(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != "done" {
				t.Fatalf("result = %+v", got)
			}
		})
	}
}

// A flow whose node needs the baseline and a run that has none: refused, so
// that the node does not first spend a model call and then find out.
func TestARunIsRefusedWhenANodeNeedsABaselineAndThereIsNone(t *testing.T) {
	log := &memJournal{}
	graph := graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil)
	run := runner.New(runner.Options{
		Graph: graph, Catalog: catalogOf(t, baselineBlock{}), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err == nil {
		t.Fatal("want a refusal")
	}
	if len(log.lines) != 0 {
		t.Fatalf("a refused run wrote %d entries", len(log.lines))
	}
}

// The same flow with a baseline runs: the refusal above is about the baseline
// and not about the block.
func TestARunWithABaselineReachesANodeThatNeedsOne(t *testing.T) {
	log := &memJournal{}
	graph := graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil)
	run := runner.New(runner.Options{
		Graph:   graph,
		Catalog: catalogOf(t, baselineBlock{}),
		Journal: log,
		Env:     flow.Env{Baseline: &flow.Baseline{Commit: "abc"}},
		Clock:   ticks(),
	})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("result = %+v", got)
	}
}

// A node error, and the journal says so: a line reading ok would tell show and
// every later resume that the step succeeded.
func TestADeltaTheFlowDoesNotDeclareIsANodeError(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1)},
		[]flow.Edge{{From: "a", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	catalog := catalogOf(t, stepBlock{delta: flow.Delta{"nope": "x"}})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := `the delta names "nope", which the flow does not declare`
	if got.Status != "error" || got.Node != "a" || got.Detail != want {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 1 || log.lines[0].Outcome != "error" || log.lines[0].Detail == nil || *log.lines[0].Detail != want {
		t.Fatalf("entries = %+v", log.lines)
	}
}

// The declared type is what reaches the state, so a block that writes the
// wrong shape is caught here rather than at the next node that reads it.
func TestADeltaOfTheWrongTypeIsANodeError(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1)},
		[]flow.Edge{{From: "a", To: flow.End}},
		map[string]flow.Field{"n": {Type: flow.Int, Default: 0}},
	)
	catalog := catalogOf(t, stepBlock{delta: flow.Delta{"n": "seven"}})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Node != "a" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 1 || log.lines[0].Outcome != "error" {
		t.Fatalf("entries = %+v", log.lines)
	}
}

// A delta the flow cannot hold is a node error like any other, so the node gets
// the fallback its flow declares. What the node spent is recorded all the same.
func TestADeltaTheFlowCannotHoldTakesTheErrorEdge(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1), {Name: "fix", Kind: "fix", MaxVisits: flow.Cap{Add: 1}}},
		[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: "fix", OnError: true}, {From: "fix", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	catalog := catalogOf(t,
		stepBlock{delta: flow.Delta{"nope": "x"}, model: "fake:m1"},
		fixBlock{stepBlock{delta: flow.Delta{"note": "fixed"}}})
	run := runner.New(runner.Options{Graph: graph, Catalog: catalog, Journal: log, Clock: ticks()})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" || got.State.Fields["note"] != "fixed" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 2 || log.lines[0].Outcome != "error" || log.lines[0].Tokens != 7 || log.lines[1].Node != "fix" {
		t.Fatalf("entries = %+v", log.lines)
	}
	if log.lines[0].Model == nil || *log.lines[0].Model != "fake:m1" {
		t.Fatalf("model = %v", log.lines[0].Model)
	}
}

// brokenDefine is a step whose definition cannot be made. Nothing can be
// journalled about such a node, so the call ends rather than the run.
type brokenDefine struct{ stepBlock }

func (brokenDefine) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{}, errors.New("the instruction file is gone")
}

func TestADefinitionThatCannotBeMadeEndsTheCall(t *testing.T) {
	log := &memJournal{}
	graph := graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil)
	run := runner.New(runner.Options{
		Graph: graph, Catalog: catalogOf(t, brokenDefine{}), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err == nil {
		t.Fatal("want the definition error")
	}
}

// A node entry that cannot be serialized cannot be hashed either, and an entry
// without a definition hash is one no replay could check.
func TestANodeEntryThatCannotBeHashedEndsTheCall(t *testing.T) {
	log := &memJournal{}
	raw := map[string]any{"kind": func() {}}
	graph := graphOf(
		[]flow.Node{{Name: "a", Kind: "step", MaxVisits: flow.Cap{Add: 1}, Raw: raw}},
		[]flow.Edge{{From: "a", To: flow.End}}, nil,
	)
	run := runner.New(runner.Options{
		Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err == nil {
		t.Fatal("want the hash error")
	}
}

// The input hash is what a resume finds a node's earlier result by. A state
// that cannot be hashed has no key, so the run cannot start.
func TestAStateThatCannotBeHashedEndsTheCall(t *testing.T) {
	log := &memJournal{}
	graph := graphOf(
		[]flow.Node{node("a", 1)},
		[]flow.Edge{{From: "a", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: func() {}}},
	)
	run := runner.New(runner.Options{
		Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err == nil {
		t.Fatal("want the hash error")
	}
}

// The one place a name becomes a node says so when the name was not one. The
// zero node walking on would be reported as a ceiling of 0 that no flow ever
// wrote, and outside a replay it would panic inside a block's Define instead.
func TestANameNoNodeDeclaresEndsTheCall(t *testing.T) {
	cases := []struct {
		name    string
		graph   *flow.Graph
		entries int
	}{
		{
			name: "a start that is not a node",
			graph: &flow.Graph{
				Name: "f", File: "f.toml", Start: "x",
				Nodes: []flow.Node{node("a", 1)},
				Edges: []flow.Edge{{From: "a", To: flow.End}},
			},
			entries: 0,
		},
		{
			name:    "an edge to a name no node carries",
			graph:   graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: "x"}}, nil),
			entries: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := &memJournal{}
			run := runner.New(runner.Options{
				Graph: c.graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(),
			})
			_, err := run.Run(context.Background())
			if err == nil || err.Error() != `the flow has no node "x"` {
				t.Fatalf("err = %v", err)
			}
			if len(log.lines) != c.entries {
				t.Fatalf("entries = %+v", log.lines)
			}
		})
	}
}

// The start checks guard a resume as much as a run: a flow whose ceilings
// cannot be read is no more walkable the second time.
func TestAResumeIsRefusedTheSameWayARunIs(t *testing.T) {
	log := &memJournal{}
	graph := graphOf([]flow.Node{node("a", 0)}, []flow.Edge{{From: "a", To: flow.End}}, nil)
	run := runner.New(runner.Options{
		Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(),
	})
	if _, err := run.Resume(context.Background(), nil); err == nil {
		t.Fatal("want a refusal")
	}
}

// failJournal refuses one of the three methods, so each way a journal can be
// unusable arrives at the caller as an error and not as a wrong answer.
// failAppend names the outcome whose entry it refuses, so that every path that
// writes one can be reached in turn.
type failJournal struct {
	memJournal
	onEntries  bool
	onPending  bool
	failAppend string
}

func (j *failJournal) Entries() ([]journal.Entry, error) {
	if j.onEntries {
		return nil, errors.New("the journal cannot be read")
	}
	return j.memJournal.Entries()
}

func (j *failJournal) Append(entry journal.Entry) error {
	if entry.Outcome == j.failAppend {
		return errors.New("the journal cannot be written")
	}
	return j.memJournal.Append(entry)
}

func (j *failJournal) Pending() (*journal.PendingGate, error) {
	if j.onPending {
		return nil, errors.New("the journal cannot be read")
	}
	return j.memJournal.Pending()
}

func TestAJournalThatCannotBeReadEndsTheCall(t *testing.T) {
	answer := "yes"
	cases := []struct {
		name   string
		log    *failJournal
		replay bool
		resume bool
		answer *string
	}{
		{name: "a replay that cannot read", log: &failJournal{onEntries: true}, replay: true},
		{name: "a replay whose gate cannot be read", log: &failJournal{onPending: true}, replay: true},
		{name: "a resume that cannot read", log: &failJournal{onEntries: true}, resume: true},
		{name: "an answer whose gate cannot be read", log: &failJournal{onPending: true}, resume: true, answer: &answer},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			graph := graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil)
			run := runner.New(runner.Options{
				Graph: graph, Catalog: catalogOf(t, stepBlock{}), Journal: c.log, Clock: ticks(), Replay: c.replay,
			})
			var err error
			if c.resume {
				_, err = run.Resume(context.Background(), c.answer)
			} else {
				_, err = run.Run(context.Background())
			}
			if err == nil {
				t.Fatal("want the journal error")
			}
		})
	}
}

// Every outcome a walk can write, and the run that cannot write it. A step
// nobody could journal is a step no resume could reconstruct, so the call ends
// rather than walking on.
func TestAJournalThatCannotBeWrittenEndsTheCall(t *testing.T) {
	cases := []struct {
		name    string
		graph   *flow.Graph
		block   flow.Block
		outcome string
	}{
		{
			name:    "a step that succeeded",
			graph:   graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil),
			block:   stepBlock{},
			outcome: "ok",
		},
		{
			name:    "a node over its ceiling",
			graph:   graphOf([]flow.Node{node("a", 2)}, []flow.Edge{{From: "a", To: "a"}}, nil),
			block:   stepBlock{},
			outcome: "error",
		},
		{
			name:    "a node that failed",
			graph:   graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil),
			block:   stepBlock{err: errors.New("the tool is gone")},
			outcome: "error",
		},
		{
			name:    "a node that asked for an exit",
			graph:   graphOf([]flow.Node{node("a", 1)}, []flow.Edge{{From: "a", To: flow.End}}, nil),
			block:   stepBlock{exit: &flow.Exit{Code: 4, Message: "rejected"}},
			outcome: "error",
		},
		{
			name:    "a gate that paused",
			graph:   oneGate(),
			block:   askBlock(nil),
			outcome: "paused",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := &failJournal{failAppend: c.outcome}
			run := runner.New(runner.Options{
				Graph: c.graph, Catalog: catalogOf(t, c.block), Journal: log, Clock: ticks(),
			})
			if _, err := run.Run(context.Background()); err == nil {
				t.Fatal("want the journal error")
			}
		})
	}
}
