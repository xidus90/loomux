package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/journal"
)

// walk is the whole behaviour contract in one loop. It is one function on
// purpose: every early return is a way a run can end, and they read best next
// to each other.
func (r *Runner) walk(ctx context.Context, state flow.State, answer *pendingAnswer) (Result, error) {
	name := r.graph.Start
	for name != flow.End {
		// Not a second validation pass -- the loader owns the graph's five
		// rules. This is the lookup that turns a name into a node, saying so
		// when the name was not one. Without it the zero Node walks on: an
		// empty kind and a ceiling of 0, so the check below would report a
		// ceiling the flow never wrote, and a run that is not a replay would
		// panic inside some block's Define with nothing naming the edge that
		// was wrong.
		node, declared := r.nodes[name]
		if !declared {
			return Result{}, fmt.Errorf("the flow has no node %q", name)
		}
		state = withVisit(state, name)

		hash, err := inputHash(name, state)
		if err != nil {
			return Result{}, err
		}

		// The ceiling is checked before the block runs, and the entry it writes
		// carries 0 tokens and 0 seconds: nothing ran.
		if limit := r.limits[name]; state.Visits[name] > limit {
			detail := fmt.Sprintf("node %q exceeded max_visits=%d", name, limit)
			if err := r.write(node, hash, step{outcome: "error", detail: &detail}); err != nil {
				return Result{}, err
			}
			return Result{Status: "error", Node: name, Detail: detail, State: state}, nil
		}

		// Matched on the pause's own key, not on the node's name: a gate on a
		// cycle pauses once per pass, and an earlier pass is already answered in
		// the journal. Keying on the name would spend the answer there -- where
		// retracing short-circuits before it is even read -- and the open pause
		// would be asked again with nothing recorded.
		var given *string
		if answer != nil && answer.key == hash {
			given = &answer.text
			answer = nil
		}

		if r.retracing {
			// The most recent *successful* entry, not the most recent one: a
			// visit ceiling and a gate's pause both write a non-ok entry under
			// the key of an entry that succeeded.
			if cached, found := journal.Lookup(r.entries, name, hash, "ok"); found {
				if err := r.warnOnChangedDefinition(node, cached); err != nil {
					return Result{}, err
				}
				next, target, stop := r.advance(name, state, cached.Delta)
				if stop != nil {
					return *stop, nil
				}
				state, name = next, target
				continue
			}
			if r.replay {
				// A failure and an exit both journal an error for the visit they
				// ended, and the retrace above takes only ok. (A ceiling does too,
				// but under another key, and the check above has reproduced it
				// before this is reached.) The replay follows the error edge when
				// the flow has one and the run evidently took it: the journal goes
				// on past the entry, or the edge goes to END, where nothing would
				// follow it anyway. In every other case it ends at the entry, with
				// its message and without an exit code, which the journal does not
				// carry. An exit node that itself has an error edge to END is the
				// one ending this reads wrongly -- its journal is the same entry --
				// and replays as done.
				if failed, continued, found := r.recordedFailure(name, hash); found {
					fallback, hasFallback := r.errorName(name)
					if hasFallback && (continued || fallback == flow.End) {
						name = fallback
						continue
					}
					if continued {
						r.warn(fmt.Sprintf("node %q failed and the run went on by an error edge the flow no longer has", name))
					}
					detail := ""
					if failed.Detail != nil {
						detail = *failed.Detail
					}
					return Result{Status: "error", Node: name, Detail: detail, State: state}, nil
				}
				// No entry at all. Still not an error outcome: that would be
				// offered the node's error edge, and taking a fallback the
				// original run never took would make a broken replay look like a
				// run that handled a failure.
				return Result{
					Status: "error", Node: name, State: state,
					Detail: fmt.Sprintf("node %q is not in the journal", name),
				}, nil
			}
			// A resume has caught up with where the earlier run stopped.
			// Everything from here is new work, including a second pass through
			// a node the journal already covers.
			r.retracing = false
		}

		started := r.clock()
		result, runErr := r.blocks[name].Run(ctx, r.graph, node, state, r.envWith(given))
		seconds := r.clock().Sub(started).Seconds()

		// A delta the flow cannot hold is the node's failure: it is journalled as
		// one and the node is offered its error edge. So the next state is built
		// before anything is written, not after an ok line already stands.
		var next flow.State
		if runErr == nil && result.Exit == nil && result.Question == nil {
			next, runErr = r.merge(state, result.Delta)
		}

		switch {
		case errors.Is(runErr, flow.ErrInvalidAnswer):
			// Not a node failure: no entry, no error edge, and the visit is not
			// spent. The gate stays open and the caller is told what it refused.
			return Result{}, runErr

		case runErr != nil:
			detail := runErr.Error()
			// Tokens and model are what a delta the flow refused still cost; a
			// block that failed outright reports neither.
			entry := step{outcome: "error", tokens: result.Tokens, seconds: seconds, detail: &detail, model: result.Model}
			if err := r.write(node, hash, entry); err != nil {
				return Result{}, err
			}
			fallback, hasFallback := r.errorName(name)
			if !hasFallback {
				return Result{Status: "error", Node: name, Detail: detail, State: state}, nil
			}
			name = fallback

		case result.Exit != nil:
			// An exit is an ending with a reason, not a failure, so no error
			// edge is offered: a fallback that swallowed the code would leave
			// the caller with 1 and no way to tell the endings apart.
			detail := result.Exit.Message
			if err := r.write(node, hash, step{outcome: "error", seconds: seconds, detail: &detail}); err != nil {
				return Result{}, err
			}
			code := result.Exit.Code
			return Result{Status: "error", Node: name, Detail: detail, State: state, ExitCode: &code}, nil

		case result.Question != nil:
			// Only when this very visit is not already recorded as paused: a
			// resume without an answer walks back to here, and a run someone
			// checked on ten times would otherwise read as ten pauses. Unless
			// this walk has appended something since: the open gate is what
			// the journal's last entry says, and a pause that is no longer
			// last is a gate nobody can find to answer.
			if _, waiting := journal.Lookup(r.entries, name, hash, "paused"); !waiting || r.appended {
				question := *result.Question
				entry := step{outcome: "paused", seconds: seconds, detail: &question, model: result.Model}
				if err := r.write(node, hash, entry); err != nil {
					return Result{}, err
				}
			}
			return Result{Status: "paused", Node: name, Question: *result.Question, State: state}, nil

		default:
			entry := step{
				delta:   result.Delta,
				outcome: "ok",
				tokens:  result.Tokens,
				seconds: seconds,
				model:   result.Model,
			}
			if given != nil {
				// This entry records an answer that arrived from outside the
				// process, not a step this run executed and timed.
				entry.seconds = 0
				answered := "answered: " + *given
				entry.detail = &answered
			}
			if err := r.write(node, hash, entry); err != nil {
				return Result{}, err
			}
			target, stop := r.follow(name, next)
			if stop != nil {
				return *stop, nil
			}
			state, name = next, target
		}
	}
	return Result{Status: "done", State: state}, nil
}

// advance merges a delta read back from the journal into the state and names
// the node after it. Both can fail on the flow's own terms -- a field the flow
// no longer declares, no edge that applies -- so both answers come back as the
// one Result the walk then returns. A retrace has nothing to journal and no
// error edge to offer: the journal describes a state this flow cannot hold.
func (r *Runner) advance(name string, state flow.State, delta map[string]any) (flow.State, string, *Result) {
	next, err := r.merge(state, delta)
	if err != nil {
		return state, "", &Result{Status: "error", Node: name, Detail: err.Error(), State: state}
	}
	target, stop := r.follow(name, next)
	return next, target, stop
}

// follow names the node after this one, or ends the run where no edge applies.
// Retracing and working share it, so the message for a dead end exists once.
func (r *Runner) follow(name string, next flow.State) (string, *Result) {
	for _, edge := range r.graph.Edges {
		if edge.From != name || edge.OnError {
			continue
		}
		if edge.When == nil || edge.When.Holds(next, r.env.Params) {
			return edge.To, nil
		}
	}
	return "", &Result{
		Status: "error", Node: name, State: next,
		Detail: fmt.Sprintf("no edge out of %q applies to the current state", name),
	}
}

// recordedFailure is the latest error entry for this visit, and whether the
// journal goes on past it.
func (r *Runner) recordedFailure(name, hash string) (journal.Entry, bool, bool) {
	for i := len(r.entries) - 1; i >= 0; i-- {
		entry := r.entries[i]
		if entry.Node == name && entry.InputHash == hash && entry.Outcome == "error" {
			return entry, i < len(r.entries)-1, true
		}
	}
	return journal.Entry{}, false, false
}

// errorName is where to go when a node failed, or false to end the run. An
// error edge carries no condition and is invisible to the normal path, so a
// fallback is a visible edge in the flow rather than retry logic in here.
func (r *Runner) errorName(name string) (string, bool) {
	for _, edge := range r.graph.Edges {
		if edge.From == name && edge.OnError {
			return edge.To, true
		}
	}
	return "", false
}

// merge builds the next state from a delta. Every value goes through
// flow.Coerce, because a delta read back from a journal holds json.Number where
// the flow declared an int, and a state holds the declared type or nothing.
func (r *Runner) merge(state flow.State, delta map[string]any) (flow.State, error) {
	fields := make(map[string]flow.Value, len(state.Fields))
	for name, value := range state.Fields {
		fields[name] = value
	}
	for name, raw := range delta {
		field, declared := r.graph.State[name]
		if !declared {
			return flow.State{}, fmt.Errorf("the delta names %q, which the flow does not declare", name)
		}
		value, err := flow.Coerce(field.Type, raw)
		if err != nil {
			return flow.State{}, fmt.Errorf("%s: %w", name, err)
		}
		fields[name] = value
	}
	return flow.State{Fields: fields, Visits: state.Visits}, nil
}

// withVisit counts one more visit of a node. A State is never changed in place:
// a resume reconstructs what every node saw, and a shared map would let a later
// pass rewrite an earlier one's input.
func withVisit(state flow.State, name string) flow.State {
	visits := make(map[string]int, len(state.Visits)+1)
	for node, count := range state.Visits {
		visits[node] = count
	}
	visits[name]++
	return flow.State{Fields: state.Fields, Visits: visits}
}

// inputHash is the key a node's result is filed and found under: what it saw,
// the visit counts included. A loop whose passes leave the fields alone would
// otherwise be served its first pass forever.
func inputHash(name string, state flow.State) (string, error) {
	return journal.InputHash(name, map[string]any{"fields": state.Fields, "visits": state.Visits})
}

// step is one visit's outcome, in the shape the journal records it.
type step struct {
	delta   map[string]any
	outcome string
	tokens  int
	seconds float64
	detail  *string
	model   string
}

// write is the one place that builds a journal entry, so that no path -- not
// the ceiling, not a pause, not a failure -- escapes the replay guard: a replay
// that appended to the journal it is reading would not be reproducing a run.
func (r *Runner) write(node flow.Node, hash string, s step) error {
	if r.replay {
		return nil
	}
	definition, definitionHash, err := r.definition(node)
	if err != nil {
		return err
	}
	delta := make(map[string]any, len(s.delta))
	for name, value := range s.delta {
		delta[name] = value
	}
	err = r.journal.Append(journal.Entry{
		Node:      node.Name,
		Kind:      node.Kind,
		InputHash: hash,
		Delta:     delta,
		Outcome:   s.outcome,
		// The profile name, not the resolved list: the list is derived from it,
		// and the name is what a reader recognises the node by.
		Tools:          optional(definition.Profile),
		Effort:         optional(definition.Effort),
		Tokens:         s.tokens,
		Seconds:        s.seconds,
		Detail:         s.detail,
		Model:          optional(s.model),
		Role:           optional(definition.Role),
		DefinitionHash: &definitionHash,
	})
	if err != nil {
		return err
	}
	r.appended = true
	return nil
}

// warnOnChangedDefinition reports a node whose instruction, model, effort or
// tool list has changed since the entry being retraced was written. Reported,
// never refused: improving the wording of a question must stay allowed.
func (r *Runner) warnOnChangedDefinition(node flow.Node, cached journal.Entry) error {
	_, hash, err := r.definition(node)
	if err != nil {
		return err
	}
	// Every entry this package writes carries the hash, so a line that holds
	// null recorded no definition, and none can match the one the node has now.
	if cached.DefinitionHash == nil || hash != *cached.DefinitionHash {
		r.warn(fmt.Sprintf("node %q ran on a definition that has changed since", node.Name))
	}
	return nil
}

// definition is what the node was told, and its fingerprint. The hash is taken
// over the node's raw file entry and not over the parsed struct, so it is the
// flow file that is fingerprinted and not this package's field names.
func (r *Runner) definition(node flow.Node) (flow.Definition, string, error) {
	definition, err := r.blocks[node.Name].Define(r.graph, node, r.envWith(nil))
	if err != nil {
		return flow.Definition{}, "", fmt.Errorf("node %q: %w", node.Name, err)
	}
	hash, err := journal.DefinitionHash(node.Raw, definition.Text, definition.Model, definition.Effort, definition.Tools)
	if err != nil {
		return flow.Definition{}, "", fmt.Errorf("node %q: %w", node.Name, err)
	}
	return definition, hash, nil
}

// optional is the journal's spelling of nothing here: null, not an empty string
// that would claim a value was recorded.
func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
