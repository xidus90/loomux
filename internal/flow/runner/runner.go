package runner

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/journal"
)

// Journal is the run journal from the runner's side: it writes to it, reads it
// back when it retraces, and asks it for the gate a resume has to answer.
//
// An interface and not a path, because the runner writes and reads the same
// journal and a test that had to create a file for that would be testing the
// file system as well. Pending belongs to it because journal.Pending takes a
// path.
type Journal interface {
	Entries() ([]journal.Entry, error)
	Append(entry journal.Entry) error
	Pending() (*journal.PendingGate, error)
}

// Clock is where every duration in the journal comes from. Injected, so a run
// of the same flow over the same journal writes the same numbers twice.
type Clock func() time.Time

// Options are everything a Runner needs. Every field but Warn and Replay is
// required; a run without a clock has no duration to record.
type Options struct {
	Graph   *flow.Graph
	Catalog *flow.Catalog
	Journal Journal
	Env     flow.Env
	Clock   Clock
	// Warn takes the messages a run reports without stopping for. Writing to
	// stderr is internal/cli's job, not this package's.
	Warn func(string)
	// Replay retraces the journal from the first node to the last and executes
	// nothing.
	Replay bool
}

// Result is how a run ended, and where.
type Result struct {
	Status   string
	Node     string
	Question string
	Detail   string
	State    flow.State
	ExitCode *int
}

// Runner walks a graph, journalling every step.
//
// A node is recognised by its name and the input it saw, never by its
// implementation: editing a block's body and replaying the same journal
// reproduces the old result. Hashing the implementation instead would throw a
// journal away on a cosmetic edit.
type Runner struct {
	graph   *flow.Graph
	catalog *flow.Catalog
	journal Journal
	env     flow.Env
	clock   Clock
	warn    func(string)
	replay  bool

	// nodes and limits are filled by the start checks, before the walk: the
	// ceiling of every node is known to be usable by the time the first one runs.
	nodes  map[string]flow.Node
	limits map[string]int
	blocks map[string]flow.Block

	// retracing says whether this walk is reconstructing a journal that already
	// exists. A replay retraces from the first node to the last; a resume
	// retraces only up to the point where the earlier run stopped, and the walk
	// switches it off there.
	retracing bool
	entries   []journal.Entry

	// appended says whether this walk has written to the journal, which moves
	// an earlier pause away from the end where the open gate is read.
	appended bool
}

// New makes a Runner. It checks nothing: what a run refuses, it refuses in Run
// or Resume, where a caller is asking for something to happen.
func New(opts Options) *Runner {
	warn := opts.Warn
	if warn == nil {
		warn = func(string) {}
	}
	return &Runner{
		graph:   opts.Graph,
		catalog: opts.Catalog,
		journal: opts.Journal,
		env:     opts.Env,
		clock:   opts.Clock,
		warn:    warn,
		replay:  opts.Replay,
		nodes:   make(map[string]flow.Node, len(opts.Graph.Nodes)),
		limits:  make(map[string]int, len(opts.Graph.Nodes)),
		blocks:  make(map[string]flow.Block, len(opts.Graph.Nodes)),
	}
}

// errWaitsAtAGate refuses a replay of a run that is still open. A replay
// reproduces a run that ended; one waiting at a gate has no ending to
// reproduce. A replay looks up only ok and error entries, so retracing it would
// end at the gate with "not in the journal", and an open run would read as a
// broken replay.
var errWaitsAtAGate = errors.New("this run waits at a gate; answer it with resume")

// Run walks the flow from its start node and executes every node it reaches.
// A replay is the exception: it retraces the journal instead.
func (r *Runner) Run(ctx context.Context) (Result, error) {
	if err := r.startChecks(); err != nil {
		return Result{}, err
	}
	// Stated here rather than inherited from whatever this Runner did last: one
	// that resumed once would otherwise carry that retrace into every later run
	// and be served the old journal instead of working. The snapshot goes with
	// the flag -- a run that kept a Resume's entries would find this visit's
	// pause in them and skip the entry it owes, losing a journal line quietly
	// rather than failing.
	r.retracing = r.replay
	r.entries = nil
	r.appended = false
	if r.replay {
		entries, err := r.journal.Entries()
		if err != nil {
			return Result{}, err
		}
		r.entries = entries
		gate, err := r.journal.Pending()
		if err != nil {
			return Result{}, err
		}
		if gate != nil {
			return Result{}, errWaitsAtAGate
		}
	}
	return r.walk(ctx, r.initialState(), nil)
}

// Resume carries a paused run onward, applying the answer to its open gate.
//
// Without an answer the gate pauses again: treating a missing answer as consent
// would make the approval point decorative.
//
// The walk still starts at the flow's start node, so every node before the gate
// is reconstructed from the journal and the gate sees the state it actually
// saw. Jumping straight to the gate with a fresh state would key the answer
// under a hash no later replay can find.
func (r *Runner) Resume(ctx context.Context, answer *string) (Result, error) {
	if r.replay && answer != nil {
		// Applying an answer runs the gate's block live, which is precisely the
		// promise a replay makes it will not do.
		return Result{}, errors.New("a replay cannot take an answer; resume the run instead")
	}
	if err := r.startChecks(); err != nil {
		return Result{}, err
	}
	entries, err := r.journal.Entries()
	if err != nil {
		return Result{}, err
	}
	r.entries = entries
	r.retracing = true
	r.appended = false
	gate, err := r.journal.Pending()
	if err != nil {
		return Result{}, err
	}
	if r.replay && gate != nil {
		return Result{}, errWaitsAtAGate
	}
	if answer == nil {
		return r.walk(ctx, r.initialState(), nil)
	}
	if gate == nil {
		// An answer with nothing to answer is a mistake worth reporting: a
		// silent run from the start would discard the answer and charge for
		// every node again, which is the worst reading of a user's "yes".
		return Result{}, errors.New("no gate is waiting for an answer")
	}
	return r.walk(ctx, r.initialState(), &pendingAnswer{key: gate.InputHash, text: *answer})
}

// startChecks are the four things a run is refused for, before the first
// journal entry exists. They come back as a Go error and not as a Result with
// status "error": a run refused this way has not happened, and it leaves no
// journal behind.
//
// The graph itself is not checked. The loader does that, before a *flow.Graph
// exists at all; a second pass here would mean writing its five rules twice.
func (r *Runner) startChecks() error {
	if err := r.paramChecks(); err != nil {
		return err
	}
	for _, node := range r.graph.Nodes {
		// The block first, because the baseline check below needs it. Each of
		// the three refusals stands on its own; only their order is arbitrary.
		block, known := r.catalog.Block(node.Kind)
		if !known {
			return fmt.Errorf("node %q is of kind %q, for which this build has no block", node.Name, node.Kind)
		}
		limit, err := node.MaxVisits.Limit(r.env.Params)
		if err != nil {
			return fmt.Errorf("node %q: %w", node.Name, err)
		}
		if limit < 1 {
			return fmt.Errorf("node %q allows %d visits; a node that may never run cannot be walked", node.Name, limit)
		}
		if block.NeedsBaseline() && r.env.Baseline == nil {
			return fmt.Errorf("node %q needs the run's baseline, and this run has none", node.Name)
		}
		r.nodes[node.Name] = node
		r.limits[node.Name] = limit
		r.blocks[node.Name] = block
	}
	return nil
}

// paramChecks holds the run's parameters against the flow's declarations. A
// parameter that is missing, or held in another Go type than the one Coerce
// makes of it, would not be named anywhere later: an int comparison panics on
// it, and a string or bool comparison silently takes another edge. Params arrive with defaults applied;
// filling a gap is the caller's job, not this package's.
func (r *Runner) paramChecks() error {
	names := make([]string, 0, len(r.graph.Params))
	for name := range r.graph.Params {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		declared := r.graph.Params[name]
		value, present := r.env.Params[name]
		if !present {
			return fmt.Errorf("parameter %q is declared, and the run does not have it", name)
		}
		coerced, err := flow.Coerce(declared.Type, value)
		if err != nil || reflect.TypeOf(coerced) != reflect.TypeOf(value) {
			return fmt.Errorf("parameter %q is declared as %s and holds %T", name, declared.Type, value)
		}
	}
	return nil
}

// initialState is every declared field at its default, and no visit yet.
func (r *Runner) initialState() flow.State {
	fields := make(map[string]flow.Value, len(r.graph.State))
	for name, field := range r.graph.State {
		fields[name] = field.Default
	}
	return flow.State{Fields: fields, Visits: make(map[string]int, len(r.graph.Nodes))}
}

// envWith is the environment one node run sees: the run's own, plus the answer
// that belongs to this visit, if there is one.
func (r *Runner) envWith(answer *string) flow.Env {
	env := r.env
	env.Answer = answer
	return env
}

// pendingAnswer is an answer travelling through the walk towards the pause it
// belongs to. key is the paused entry's input hash, which names one visit of
// one node and not the node.
type pendingAnswer struct {
	key  string
	text string
}
