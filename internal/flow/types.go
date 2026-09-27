package flow

import "github.com/xidus90/loomux/internal/flow/runs"

// Type is the declared type of a state field or a parameter.
type Type string

// The four types a flow can declare.
const (
	String     Type = "string"
	Int        Type = "int"
	Bool       Type = "bool"
	StringList Type = "list[string]"
)

// Value is one typed value. Its dynamic type is exactly one of string, int,
// bool or []string, matching the Type it was declared with; no other Go type
// ever appears. TOML's int64 and JSON's float64 are converted before a value
// reaches a State.
type Value = any

// Field declares a state field or a parameter.
type Field struct {
	Type    Type
	Default Value
}

// State is what a node sees: every declared field, and how often each node has
// run so far.
//
// A State is never changed in place. Blocks return a Delta and the runner
// builds the next State from it; otherwise a resume could not reconstruct what
// a node saw.
type State struct {
	Fields map[string]Value
	Visits map[string]int
}

// Delta is what one node changes: field name to new value.
type Delta map[string]Value

// Params are a run's parameters after defaults and --option were applied.
type Params map[string]Value

// Baseline is the run's starting point; it is defined beside the marker that
// carries it (internal/flow/runs).
type Baseline = runs.Baseline
