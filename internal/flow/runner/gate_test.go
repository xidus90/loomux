package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/runner"
)

// gateBlock asks its question while the run brings no answer, and turns an
// answer it knows into a delta. An answer it does not know is not a node
// failure: it comes back as flow.ErrInvalidAnswer and leaves the gate open.
type gateBlock struct {
	question string
	answers  map[string]flow.Delta
	runs     *int
}

func (gateBlock) Kind() string                { return "gate" }
func (gateBlock) Check(flow.Node) []string    { return nil }
func (gateBlock) Texts(flow.Node) []flow.Text { return nil }
func (gateBlock) NeedsBaseline() bool         { return false }

func (gateBlock) Writes(flow.Node) map[string]flow.Type { return nil }

// The profile, the effort and the model are what a journal entry records
// besides the result, so the test gate names all three.
func (gateBlock) Define(*flow.Graph, flow.Node, flow.Env) (flow.Definition, error) {
	return flow.Definition{
		Text:    []byte("may I?"),
		Model:   "test:gate",
		Effort:  "low",
		Profile: "read_only",
		Tools:   []string{"Read"},
	}, nil
}

func (b gateBlock) Run(_ context.Context, _ *flow.Graph, _ flow.Node, _ flow.State, env flow.Env) (flow.Result, error) {
	if b.runs != nil {
		*b.runs++
	}
	if env.Answer == nil {
		question := b.question
		return flow.Result{Question: &question}, nil
	}
	delta, known := b.answers[*env.Answer]
	if !known {
		return flow.Result{}, flow.ErrInvalidAnswer
	}
	return flow.Result{Delta: delta, Model: "test:gate"}, nil
}

func gateNode(name string, visits int) flow.Node {
	return flow.Node{Name: name, Kind: "gate", MaxVisits: flow.Cap{Add: visits}}
}

// A resume without an answer works a failed node again, and its entry lands
// after the pause. journal.Pending reads only the last entry, so the pause has
// to be written again behind it -- otherwise the gate is buried, and the answer
// that follows finds nothing waiting.
func TestAPauseIsWrittenAgainOnceSomethingWasAppendedPastIt(t *testing.T) {
	graph := graphOf(
		[]flow.Node{node("a", 2), gateNode("ask", 2)},
		[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: "ask", OnError: true}, {From: "ask", To: flow.End}},
		map[string]flow.Field{"answer": {Type: flow.String, Default: ""}},
	)
	catalog := func() *flow.Catalog {
		return catalogOf(t, stepBlock{err: errors.New("the tool is gone")}, askBlock(nil))
	}
	log := &memJournal{}
	first := runner.New(runner.Options{Graph: graph, Catalog: catalog(), Journal: log, Clock: ticks()})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	again := runner.New(runner.Options{Graph: graph, Catalog: catalog(), Journal: log, Clock: ticks()})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paused" {
		t.Fatalf("result = %+v", got)
	}
	if last := log.lines[len(log.lines)-1]; last.Outcome != "paused" || last.Node != "ask" {
		t.Fatalf("the journal no longer ends at the pause: %+v", log.lines)
	}
}

// oneGate is the smallest flow with a gate in it: ask, then end.
func oneGate() *flow.Graph {
	return graphOf(
		[]flow.Node{gateNode("ask", 1)},
		[]flow.Edge{{From: "ask", To: flow.End}},
		map[string]flow.Field{"answer": {Type: flow.String, Default: ""}},
	)
}

func askBlock(runs *int) gateBlock {
	return gateBlock{
		question: "ship it?",
		answers:  map[string]flow.Delta{"yes": {"answer": "yes"}, "no": {"answer": "no"}},
		runs:     runs,
	}
}

func TestAGateWritesOnePausedEntryAndEndsPaused(t *testing.T) {
	log := &memJournal{}
	run := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paused" || got.Node != "ask" || got.Question != "ship it?" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 1 {
		t.Fatalf("entries = %+v", log.lines)
	}
	entry := log.lines[0]
	if entry.Outcome != "paused" || entry.Tokens != 0 {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.Detail == nil || *entry.Detail != "ship it?" {
		t.Fatalf("detail = %v", entry.Detail)
	}
}

// A run somebody checks on ten times is still one pause. The key is the visit,
// so a second pass through the same gate still writes an entry of its own.
func TestASecondPassAtTheSameVisitWritesNoSecondPausedEntry(t *testing.T) {
	log := &memJournal{}
	run := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		again := runner.New(runner.Options{
			Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
		})
		got, err := again.Resume(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != "paused" {
			t.Fatalf("result = %+v", got)
		}
	}
	paused := 0
	for _, entry := range log.lines {
		if entry.Outcome == "paused" {
			paused++
		}
	}
	if paused != 1 {
		t.Fatalf("%d paused entries in %+v", paused, log.lines)
	}
}

// A gate on a cycle pauses once per pass. The answer belongs to the pause that
// is open, not to the node: keyed by name it would be spent on the first pass,
// which the journal has already answered.
func TestAnAnswerBelongsToTheVisitNotToTheNode(t *testing.T) {
	log := &memJournal{}
	cycle := func() *flow.Graph {
		return graphOf(
			[]flow.Node{gateNode("ask", 2)},
			[]flow.Edge{
				{From: "ask", To: flow.End, When: fieldIs("answer", "no")},
				{From: "ask", To: "ask"},
			},
			map[string]flow.Field{"answer": {Type: flow.String, Default: ""}},
		)
	}
	fresh := func() *runner.Runner {
		return runner.New(runner.Options{
			Graph: cycle(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
		})
	}

	if _, err := fresh().Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	yes := "yes"
	if _, err := fresh().Resume(context.Background(), &yes); err != nil {
		t.Fatal(err)
	}
	no := "no"
	got, err := fresh().Resume(context.Background(), &no)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("result = %+v", got)
	}

	// paused(first), ok(first), paused(second), ok(second).
	if len(log.lines) != 4 {
		t.Fatalf("entries = %+v", log.lines)
	}
	if log.lines[3].InputHash != log.lines[2].InputHash {
		t.Fatal("the second answer landed on a visit other than the pause it answers")
	}
	if log.lines[3].InputHash == log.lines[1].InputHash {
		t.Fatal("the second answer landed on the first pass, which was already answered")
	}
}

func TestTheAnswerEntryIsOkWithZeroSecondsAndItsDetail(t *testing.T) {
	log := &memJournal{}
	first := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	yes := "yes"
	second := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	got, err := second.Resume(context.Background(), &yes)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" || got.State.Fields["answer"] != "yes" {
		t.Fatalf("result = %+v", got)
	}
	entry := log.lines[len(log.lines)-1]
	if entry.Outcome != "ok" || entry.Seconds != 0 {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.Detail == nil || *entry.Detail != "answered: yes" {
		t.Fatalf("detail = %v", entry.Detail)
	}
	if entry.Delta["answer"] != "yes" {
		t.Fatalf("delta = %v", entry.Delta)
	}
}

// The three fields an entry carries besides its result: what the node was
// told, how hard it was told to think, and which model answered.
func TestAnEntryCarriesTheProfileTheEffortAndTheModel(t *testing.T) {
	log := &memJournal{}
	first := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	yes := "yes"
	second := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := second.Resume(context.Background(), &yes); err != nil {
		t.Fatal(err)
	}
	entry := log.lines[len(log.lines)-1]
	if entry.Tools == nil || *entry.Tools != "read_only" {
		t.Fatalf("tools = %v", entry.Tools)
	}
	if entry.Effort == nil || *entry.Effort != "low" {
		t.Fatalf("effort = %v", entry.Effort)
	}
	if entry.Model == nil || *entry.Model != "test:gate" {
		t.Fatalf("model = %v", entry.Model)
	}
}

func TestResumeWithAnAnswerAndNoWaitingGateIsAnError(t *testing.T) {
	log := &memJournal{}
	run := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	yes := "yes"
	_, err := run.Resume(context.Background(), &yes)
	if err == nil || err.Error() != "no gate is waiting for an answer" {
		t.Fatalf("err = %v", err)
	}
}

// An answer the gate refuses is not a node failure: no entry, no error edge,
// and the visit is not spent. The gate is still there to be answered.
func TestAnAnswerTheGateRefusesLeavesItOpen(t *testing.T) {
	log := &memJournal{}
	fresh := func() *runner.Runner {
		return runner.New(runner.Options{
			Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
		})
	}
	if _, err := fresh().Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	maybe := "maybe"
	if _, err := fresh().Resume(context.Background(), &maybe); err == nil {
		t.Fatal("want the refusal")
	}
	if len(log.lines) != 1 {
		t.Fatalf("a refused answer wrote to the journal: %+v", log.lines)
	}
	got, err := fresh().Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paused" || got.Question != "ship it?" {
		t.Fatalf("result = %+v", got)
	}
}
