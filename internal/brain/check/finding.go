// Package check carries the result model of a check run: one finding, the
// two axes it can belong to, and the three degrees of severity.
//
// The axes answer different questions that used to share one name. "okf" asks
// whether a foreign reader of the Open Knowledge Format may consume this
// bundle; "house" asks whether it also satisfies the stricter rules of this
// repository. A bundle can fail the second and still be perfectly consumable,
// so a run that reported both under one verdict said less than it seemed to.
// The axis of `check code` is gone with that command: the check chain of
// `loomux check` owns the code lanes.
package check

import (
	"fmt"
	"sort"
	"strings"
)

// Axis names which of the two questions a rule answers.
type Axis string

const (
	AxisOKF   Axis = "okf"
	AxisHouse Axis = "house"
)

// Severity names how far a finding goes. Only Error moves the exit code.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
	Note    Severity = "note"
)

// Finding is one rule violation on one page.
type Finding struct {
	// Scope is the area the page belongs to. It stays empty for a run over a
	// single file, which knows a path but no area.
	Scope    string
	Relative string
	Axis     Axis
	// Rule carries no axis prefix -- "type-missing", not "okf/type-missing".
	// The prefixed form is what Name builds, and building it in one place
	// keeps the two spellings from drifting apart.
	Rule     string
	Severity Severity
	Message  string
}

// Name is the qualified rule name: the axis, then the rule. Two axes may hold
// rules of the same name, so the bare rule is not an identifier.
func (f Finding) Name() string {
	return string(f.Axis) + "/" + f.Rule
}

// Sort orders findings by area, then path, then axis, then rule. Two runs over
// an unchanged bundle must render byte for byte the same output, otherwise no
// two runs can be compared. The order in which the rules produce findings does
// not give that on its own: it follows the directory walk and, once a run
// covers several areas at a time, the order in which those finish.
//
// The sort is stable: findings equal on every key keep the order in which the
// rules produced them, which is the only remaining tie-break that does not
// depend on the sort implementation.
func Sort(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		if a.Relative != b.Relative {
			return a.Relative < b.Relative
		}
		if a.Axis != b.Axis {
			return a.Axis < b.Axis
		}
		return a.Rule < b.Rule
	})
}

// ExitCode is 1 only when at least one finding is an Error. Warnings and notes
// never move it: a check that failed the build over an advisory would push its
// users towards switching it off.
func ExitCode(findings []Finding) int {
	for _, f := range findings {
		if f.Severity == Error {
			return 1
		}
	}
	return 0
}

// Render writes one line per finding, in the order it is given -- sorting is
// Sort's job, so that a caller can render an already sorted slice twice.
//
// Notes never reach the default run. A message one learns to skip protects
// nothing -- the same reason this repository has dropped messages before.
// Their purpose is the one-off survey, not the daily output.
func Render(findings []Finding, withNotes bool) string {
	var b strings.Builder
	for _, f := range findings {
		if f.Severity == Note && !withNotes {
			continue
		}
		fmt.Fprintf(&b, "[%s] %s %s: %s\n", f.Severity, f.Name(), location(f), f.Message)
	}
	return b.String()
}

// location is the path as a reader has to find it: prefixed by the area when
// the run covered several, bare when the run covered one file and knows no
// area -- a leading slash would read as an absolute path.
func location(f Finding) string {
	if f.Scope == "" {
		return f.Relative
	}
	return f.Scope + "/" + f.Relative
}
