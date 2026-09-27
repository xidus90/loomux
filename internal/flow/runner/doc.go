// Package runner walks a loaded graph, journals every step, retraces a journal
// on resume, and pauses at gates.
//
// Three ways to walk the same graph, and one loop for all of them. A run
// executes every node it reaches, from the start node to END, writing one
// journal line per step. A resume retraces the journal up to the point where
// the earlier run stopped and executes from there -- which is what makes
// answering a gate cheap without making a bounded loop toothless. A replay
// executes nothing at all: it reconstructs the whole run from the journal, so a
// reader can ask what happened without paying for it a second time. That
// includes a run that ended at an exit, with its message but without its code:
// the journal records an exit as an error entry and carries no code.
//
// A node is found in the journal by its name and the input it saw, never by its
// implementation. Editing a block and replaying the same journal reproduces the
// old result for the new code; hashing an implementation instead would throw a
// journal away on a cosmetic edit. What the node was told is fingerprinted
// separately, and a change to it is reported rather than refused.
//
// The runner does not check the graph. internal/flow/load refuses a dangling
// edge, an island and an unbounded cycle before a *flow.Graph exists at all,
// and a second pass here would mean writing those rules twice, in two places
// that could drift. What this package checks at the start of a run is narrower
// and about the run rather than the flow: that every declared parameter is
// there in its declared type, that every visit ceiling is usable and at least
// one, that a node needing the run's baseline has one, and that every node kind
// has a block in this build. Those four come back as a Go
// error, because a run refused that way has not happened and leaves no journal
// behind.
package runner
