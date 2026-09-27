package runner_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/runner"
)

// noted is the one-node flow the retracing tests start from: it writes a field
// and ends. A journal is made by running it, never by spelling a hash out by
// hand -- a test that computed the key itself would restate the code it checks.
func noted(value string) *flow.Graph {
	return graphOf(
		[]flow.Node{node("a", 1)},
		[]flow.Edge{{From: "a", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: value}},
	)
}

func primed(t *testing.T, graph *flow.Graph, block flow.Block) *memJournal {
	t.Helper()
	log := &memJournal{}
	run := runner.New(runner.Options{
		Graph: graph, Catalog: catalogOf(t, block), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	return log
}

func TestAResumeDoesNotRunWhatTheJournalCovers(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	ran := 0
	again := runner.New(runner.Options{
		Graph:   noted(""),
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}, runs: &ran}),
		Journal: log,
		Clock:   ticks(),
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" || got.State.Fields["note"] != "done" {
		t.Fatalf("result = %+v", got)
	}
	if ran != 0 {
		t.Fatalf("the block ran %d times on a resume the journal covers", ran)
	}
}

// A node can hold two successful entries under one key when an earlier run was
// resumed and worked past the same point. The younger one is what happened last.
func TestRetracingTakesTheLatestOkEntryForNameAndInput(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "first"}})
	later := log.lines[0]
	later.Delta = map[string]any{"note": "second"}
	log.lines = append(log.lines, later)

	ran := 0
	again := runner.New(runner.Options{
		Graph:   noted(""),
		Catalog: catalogOf(t, stepBlock{runs: &ran}),
		Journal: log,
		Clock:   ticks(),
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.State.Fields["note"] != "second" {
		t.Fatalf("state = %#v", got.State.Fields)
	}
	if ran != 0 {
		t.Fatalf("the block ran %d times", ran)
	}
}

// The journal of a shorter run, retraced against a flow that grew a node: the
// part the journal covers is reconstructed, and the rest is worked.
func TestRetracingStopsWhereTheJournalStops(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	grown := graphOf(
		[]flow.Node{node("a", 1), node("b", 1)},
		[]flow.Edge{{From: "a", To: "b"}, {From: "b", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	ran := 0
	again := runner.New(runner.Options{
		Graph:   grown,
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}, runs: &ran}),
		Journal: log,
		Clock:   ticks(),
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("result = %+v", got)
	}
	if ran != 1 {
		t.Fatalf("the block ran %d times; only \"b\" is new work", ran)
	}
	if len(log.lines) != 2 || log.lines[1].Node != "b" {
		t.Fatalf("entries = %+v", log.lines)
	}
}

// A journal read from disk hands numbers over as json.Number. What reaches the
// state is the declared type, so a condition comparing an int compares an int.
func TestARetracedDeltaArrivesInTheDeclaredType(t *testing.T) {
	counted := func() *flow.Graph {
		return graphOf(
			[]flow.Node{node("a", 1)},
			[]flow.Edge{{From: "a", To: flow.End}},
			map[string]flow.Field{"n": {Type: flow.Int, Default: 0}},
		)
	}
	log := primed(t, counted(), stepBlock{delta: flow.Delta{"n": 2}})
	log.lines[0].Delta = map[string]any{"n": json.Number("2")}

	again := runner.New(runner.Options{
		Graph: counted(), Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(),
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := got.State.Fields["n"].(int); !ok || value != 2 {
		t.Fatalf("n = %#v", got.State.Fields["n"])
	}
}

// A delta the flow no longer declares stops the retrace: the journal describes
// a state this flow cannot hold.
func TestARetracedDeltaTheFlowDoesNotDeclareEndsTheRun(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	log.lines[0].Delta = map[string]any{"nope": "x"}

	again := runner.New(runner.Options{
		Graph: noted(""), Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(),
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Node != "a" {
		t.Fatalf("result = %+v", got)
	}
}

func TestADefinitionThatCannotBeMadeStopsARetrace(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	again := runner.New(runner.Options{
		Graph: noted(""), Catalog: catalogOf(t, brokenDefine{}), Journal: log, Clock: ticks(),
	})
	if _, err := again.Resume(context.Background(), nil); err == nil {
		t.Fatal("want the definition error")
	}
}

func TestReplayRunsNoNode(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	ran := 0
	again := runner.New(runner.Options{
		Graph:   noted(""),
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}, runs: &ran}),
		Journal: log,
		Clock:   ticks(),
		Replay:  true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" || got.State.Fields["note"] != "done" {
		t.Fatalf("result = %+v", got)
	}
	if ran != 0 {
		t.Fatalf("a replay ran %d nodes", ran)
	}
	if len(log.lines) != 1 {
		t.Fatalf("a replay wrote to the journal it reads: %+v", log.lines)
	}
}

// No entry at all, and still not an error outcome: that would be offered the
// node's error edge, and taking a fallback the original run never took would
// make a broken replay look like a run that handled a failure.
func TestReplayOfANodeWithoutAnEntryIsAnError(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	grown := graphOf(
		[]flow.Node{node("a", 1), node("b", 1), node("fix", 1)},
		[]flow.Edge{
			{From: "a", To: "b"},
			{From: "b", To: flow.End},
			{From: "b", To: "fix", OnError: true},
			{From: "fix", To: flow.End},
		},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
	again := runner.New(runner.Options{
		Graph:   grown,
		Catalog: catalogOf(t, stepBlock{delta: flow.Delta{"note": "done"}}),
		Journal: log,
		Clock:   ticks(),
		Replay:  true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Node != "b" {
		t.Fatalf("result = %+v", got)
	}
	if got.Detail != `node "b" is not in the journal` {
		t.Fatalf("detail = %q", got.Detail)
	}
}

// fallingBack is a flow whose first node has a fallback. Which of the two
// endings a run reaches depends on the blocks it is given.
func fallingBack() *flow.Graph {
	return graphOf(
		[]flow.Node{node("a", 1), {Name: "fix", Kind: "fix", MaxVisits: flow.Cap{Add: 1}}},
		[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: "fix", OnError: true}, {From: "fix", To: flow.End}},
		map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
	)
}

// An exit writes an error entry, so the ok lookup misses. The replay ends where
// the run ended and with its message; the code is not in the journal, so the
// replay has none to give.
func TestReplayReproducesAnExit(t *testing.T) {
	blocks := func(ran *int) *flow.Catalog {
		return catalogOf(t,
			stepBlock{exit: &flow.Exit{Code: 4, Message: "rejected"}, runs: ran},
			fixBlock{stepBlock{delta: flow.Delta{"note": "fixed"}, runs: ran}})
	}
	log := &memJournal{}
	first := runner.New(runner.Options{Graph: fallingBack(), Catalog: blocks(nil), Journal: log, Clock: ticks()})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	ran := 0
	again := runner.New(runner.Options{
		Graph: fallingBack(), Catalog: blocks(&ran), Journal: log, Clock: ticks(), Replay: true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Node != "a" || got.Detail != "rejected" || got.ExitCode != nil {
		t.Fatalf("result = %+v", got)
	}
	if ran != 0 || len(log.lines) != 1 {
		t.Fatalf("ran %d, entries = %+v", ran, log.lines)
	}
}

// A node that failed and a journal that goes on past it: the run took the error
// edge, and the replay takes it too.
func TestReplayFollowsTheErrorEdgeTheRunTook(t *testing.T) {
	blocks := func(ran *int) *flow.Catalog {
		return catalogOf(t,
			stepBlock{err: errors.New("the tool is gone"), runs: ran},
			fixBlock{stepBlock{delta: flow.Delta{"note": "fixed"}, runs: ran}})
	}
	log := &memJournal{}
	first := runner.New(runner.Options{Graph: fallingBack(), Catalog: blocks(nil), Journal: log, Clock: ticks()})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	ran := 0
	again := runner.New(runner.Options{
		Graph: fallingBack(), Catalog: blocks(&ran), Journal: log, Clock: ticks(), Replay: true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" || got.State.Fields["note"] != "fixed" {
		t.Fatalf("result = %+v", got)
	}
	if ran != 0 || len(log.lines) != 2 {
		t.Fatalf("ran %d, entries = %+v", ran, log.lines)
	}
}

// A failure that ended the run although the node has a fallback -- the flow
// gained its error edge after the run. The journal stops at the failure, and so
// does the replay.
func TestReplayEndsAtAFailureTheJournalDoesNotGoPast(t *testing.T) {
	log := primed(t, noted(""), stepBlock{err: errors.New("the tool is gone")})

	again := runner.New(runner.Options{
		Graph:   fallingBack(),
		Catalog: catalogOf(t, stepBlock{}, fixBlock{}),
		Journal: log,
		Clock:   ticks(),
		Replay:  true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Node != "a" || got.Detail != "the tool is gone" {
		t.Fatalf("result = %+v", got)
	}
}

// The run went on past the failure, but the flow has lost the error edge it
// went on by. The replay cannot follow it and ends at the failure -- and says
// why, since the journal holds more than the replay reproduces.
func TestReplayWarnsWhenTheErrorEdgeTheRunTookIsGone(t *testing.T) {
	log := &memJournal{}
	first := runner.New(runner.Options{
		Graph:   fallingBack(),
		Catalog: catalogOf(t, stepBlock{err: errors.New("the tool is gone")}, fixBlock{stepBlock{delta: flow.Delta{"note": "fixed"}}}),
		Journal: log,
		Clock:   ticks(),
	})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	var said []string
	again := runner.New(runner.Options{
		Graph:   noted(""),
		Catalog: catalogOf(t, stepBlock{}),
		Journal: log,
		Clock:   ticks(),
		Replay:  true,
		Warn:    func(text string) { said = append(said, text) },
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Node != "a" || got.Detail != "the tool is gone" {
		t.Fatalf("result = %+v", got)
	}
	if len(said) != 1 || said[0] != `node "a" failed and the run went on by an error edge the flow no longer has` {
		t.Fatalf("warnings = %#v", said)
	}
}

// A failure whose error edge goes to END finished the run, and its journal is
// the one error entry with nothing after it. The replay takes the edge the run
// took and ends done, not at the failure.
func TestReplayOfAFailureWhoseErrorEdgeEndsTheRunIsDone(t *testing.T) {
	ending := func() *flow.Graph {
		return graphOf(
			[]flow.Node{node("a", 1)},
			[]flow.Edge{{From: "a", To: flow.End}, {From: "a", To: flow.End, OnError: true}},
			map[string]flow.Field{"note": {Type: flow.String, Default: ""}},
		)
	}
	log := primed(t, ending(), stepBlock{err: errors.New("the tool is gone")})

	again := runner.New(runner.Options{
		Graph: ending(), Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(), Replay: true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("result = %+v", got)
	}
}

// The walk always writes a failure's message, but an entry is data read back
// from a file. One without a detail ends the replay with an empty one.
func TestReplayOfAFailureWithoutADetailEndsWithoutOne(t *testing.T) {
	log := primed(t, noted(""), stepBlock{err: errors.New("the tool is gone")})
	log.lines[0].Detail = nil

	again := runner.New(runner.Options{
		Graph: noted(""), Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(), Replay: true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Node != "a" || got.Detail != "" {
		t.Fatalf("result = %+v", got)
	}
}

func TestReplayRefusesAnAnswer(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	again := runner.New(runner.Options{
		Graph: noted(""), Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(), Replay: true,
	})
	yes := "yes"
	_, err := again.Resume(context.Background(), &yes)
	if err == nil || err.Error() != "a replay cannot take an answer; resume the run instead" {
		t.Fatalf("err = %v", err)
	}
}

// A replay reproduces a run that ended. One that is still open has no ending to
// reproduce, and retracing it would end at the gate with "not in the journal",
// so an open run would read as a broken replay.
func TestReplayRefusesARunThatWaitsAtAGate(t *testing.T) {
	log := &memJournal{}
	first := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	again := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(), Replay: true,
	})
	if _, err := again.Run(context.Background()); err == nil {
		t.Fatal("want the refusal")
	}
}

// Resume without an answer refuses a replay of a waiting run the way Run does,
// rather than reporting that the gate is not in the journal.
func TestReplayResumeRefusesARunThatWaitsAtAGate(t *testing.T) {
	log := &memJournal{}
	first := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := first.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	again := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(), Replay: true,
	})
	_, err := again.Resume(context.Background(), nil)
	if err == nil || err.Error() != "this run waits at a gate; answer it with resume" {
		t.Fatalf("err = %v", err)
	}
}

// Reported, never refused: improving the wording at a gate stays allowed.
func TestAChangedDefinitionWarnsAndCarriesOn(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	other := "0000"
	log.lines[0].DefinitionHash = &other

	var said []string
	again := runner.New(runner.Options{
		Graph:   noted(""),
		Catalog: catalogOf(t, stepBlock{}),
		Journal: log,
		Clock:   ticks(),
		Warn:    func(text string) { said = append(said, text) },
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("result = %+v", got)
	}
	if len(said) != 1 || !strings.Contains(said[0], `"a"`) {
		t.Fatalf("warnings = %#v", said)
	}
}

// A Runner that resumed and then runs again is doing a fresh run, the journal
// snapshot included. One that kept the resume's entries would find this very
// pause among them and skip the entry it owes -- losing a journal line quietly
// rather than failing.
func TestARunAfterAResumeStillWritesItsPause(t *testing.T) {
	log := &memJournal{}
	run := runner.New(runner.Options{
		Graph: oneGate(), Catalog: catalogOf(t, askBlock(nil)), Journal: log, Clock: ticks(),
	})
	if _, err := run.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := run.Resume(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(log.lines) != 1 {
		t.Fatalf("the resume wrote a second pause: %+v", log.lines)
	}
	got, err := run.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "paused" {
		t.Fatalf("result = %+v", got)
	}
	if len(log.lines) != 2 {
		t.Fatalf("the run kept the resume's snapshot and wrote no pause: %+v", log.lines)
	}
}

// Warn is optional. A caller that hands none still gets the run, not a crash.
func TestAChangedDefinitionWithoutAWarnIsHarmless(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	other := "0000"
	log.lines[0].DefinitionHash = &other

	again := runner.New(runner.Options{
		Graph: noted(""), Catalog: catalogOf(t, stepBlock{}), Journal: log, Clock: ticks(),
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("result = %+v", got)
	}
}

// A replay reproduces how a run ended, the ceiling it ran into included -- and
// it writes nothing, not even that. A replay that appended to the journal it
// reads would not be reproducing a run.
func TestAReplayReproducesACeilingAndWritesNothing(t *testing.T) {
	looping := func() *flow.Graph {
		return graphOf([]flow.Node{node("a", 2)}, []flow.Edge{{From: "a", To: "a"}}, nil)
	}
	log := primed(t, looping(), stepBlock{})
	before := len(log.lines)

	ran := 0
	again := runner.New(runner.Options{
		Graph:   looping(),
		Catalog: catalogOf(t, stepBlock{runs: &ran}),
		Journal: log,
		Clock:   ticks(),
		Replay:  true,
	})
	got, err := again.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.Detail != `node "a" exceeded max_visits=2` {
		t.Fatalf("result = %+v", got)
	}
	if ran != 0 {
		t.Fatalf("a replay ran %d nodes", ran)
	}
	if len(log.lines) != before {
		t.Fatalf("a replay wrote to the journal it reads: %+v", log.lines)
	}
}

// Every line the runner writes carries a definition hash. A line that reads
// `"definition_hash":null` recorded none, so it cannot match the definition the
// node has now, and the retrace says so rather than trusting it silently.
func TestAnEntryWithANullDefinitionHashWarnsAsChanged(t *testing.T) {
	log := primed(t, noted(""), stepBlock{delta: flow.Delta{"note": "done"}})
	log.lines[0].DefinitionHash = nil

	var said []string
	again := runner.New(runner.Options{
		Graph:   noted(""),
		Catalog: catalogOf(t, stepBlock{}),
		Journal: log,
		Clock:   ticks(),
		Warn:    func(text string) { said = append(said, text) },
	})
	got, err := again.Resume(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "done" {
		t.Fatalf("result = %+v", got)
	}
	if len(said) != 1 || said[0] != `node "a" ran on a definition that has changed since` {
		t.Fatalf("warnings = %#v", said)
	}
}
