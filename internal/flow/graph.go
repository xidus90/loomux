package flow

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

// End is the pseudo node an edge points to when the run is done.
const End = "END"

// Cap is a node's visit ceiling: the value of the int parameter Param plus Add,
// or Add alone when Param is empty. A node without max_visits has Cap{Add: 1}.
type Cap struct {
	Param string
	Add   int
}

// Node is one [[node]] entry of a flow file.
type Node struct {
	Name      string
	Kind      string
	Role      string // the role the node plays; empty when the node names none
	MaxVisits Cap
	// Keys are the kind's own keys as TOML decoded them. The block for Kind
	// checks and reads them; nothing else does.
	Keys map[string]any
	// Raw is the whole entry as TOML decoded it, the shared keys included. The
	// definition hash is taken over this and not over the struct: the struct
	// has no JSON tags, so its Go field names would land in the hash, and
	// MaxVisits would arrive parsed rather than as the file wrote it.
	Raw map[string]any
}

// StringKey is the node's own key `name` when it holds a string, and "" when
// it holds anything else or nothing. Three blocks and the loader read keys this
// way; written once here, none of them has to guess what TOML handed over.
func (n Node) StringKey(name string) string {
	text, _ := n.Keys[name].(string)
	return text
}

// SharedKeys are the node keys the runtime reads whatever the kind. The
// loader lifts them into Node's fields and leaves them out of Keys; every
// other key is the block's own. A fresh slice each call, so no caller can
// change what the next one reads.
func SharedKeys() []string { return []string{"kind", "max_visits", "name", "role"} }

// UnknownKeys are the node's own keys that are not in known, sorted.
func (n Node) UnknownKeys(known ...string) []string {
	var unknown []string
	for key := range n.Keys {
		if !slices.Contains(known, key) {
			unknown = append(unknown, key)
		}
	}
	slices.Sort(unknown)
	return unknown
}

// KeyFindings is one finding per key of the node's own that own does not
// hold, naming every key a node of the kind may hold: the shared keys and the
// block's. A block returns them from Check, so `instructon` is a load finding
// that shows the word it should have been, and not an instruction nobody
// reads.
func (n Node) KeyFindings(own ...string) []string {
	unknown := n.UnknownKeys(own...)
	if unknown == nil {
		return nil
	}
	known := append(SharedKeys(), own...)
	slices.Sort(known)
	findings := make([]string, len(unknown))
	for i, key := range unknown {
		findings[i] = fmt.Sprintf("unknown key %q; known keys: %s", key, strings.Join(known, ", "))
	}
	return findings
}

// Limit is the ceiling this Cap stands for, given a run's parameters. The
// loader has checked the parameter exists and is an int; a hand-built graph
// has not, and this is the last place where an unusable ceiling can be named
// rather than panicked over.
func (c Cap) Limit(params Params) (int, error) {
	if c.Param == "" {
		return c.Add, nil
	}
	raw, ok := params[c.Param]
	if !ok {
		return 0, fmt.Errorf("max_visits names parameter %q, which the run does not have", c.Param)
	}
	number, ok := raw.(int)
	if !ok {
		return 0, fmt.Errorf("max_visits names parameter %q, which is not an int", c.Param)
	}
	return number + c.Add, nil
}

// Condition decides whether an edge holds for a state.
type Condition interface {
	Holds(state State, params Params) bool
}

// Predicate is a condition written in Go and registered by name.
type Predicate func(state State, params Params) bool

// Edge is one [[edge]] entry. A nil When always holds. An error edge carries no
// When and is taken only when its source node failed.
type Edge struct {
	From    string
	To      string
	When    Condition
	Text    string // the `when` as the flow file writes it; empty when there is none
	OnError bool
}

// Graph is a loaded flow. Nodes and Edges keep file order, because the first
// edge whose condition holds is the one a run takes.
type Graph struct {
	Name string
	File string // the flow file, as messages name it
	// Texts are the flow's files as the run reads them: its folder, or a
	// bundled folder with a project's overlay laid over it. They hold
	// flow.toml, instructions/ and questions/, and may hold a README.md and
	// a _test/ folder that no run reads.
	Texts    fs.FS
	Origin   string   // where the flow came from: "project", "project (hides bundled)", "bundled" or "bundled+overlay"
	Overlays []string // the files a project's overlay replaced, e.g. "instructions/review.md"; nil without one
	Start    string
	Role     string // the flow's role; empty when it names none
	Params   map[string]Field
	State    map[string]Field
	Nodes    []Node
	Edges    []Edge
}
