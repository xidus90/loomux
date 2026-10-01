package verify

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// timed are the states of a lane that started a process or a function; only
// they have a duration and an output of their own rather than a note.
func timed(s State) bool {
	return slices.Contains([]State{StateOK, StateFailed, StateTimedOut, StateBudget}, s)
}

// WriteCheck reports every lane of a check in job order: a header, and below
// it what the lane printed when it is red, or when asked and it ran.
func WriteCheck(w io.Writer, outs []Outcome, verbose bool) {
	for _, o := range outs {
		origin := o.Job.Origin
		if o.Job.Fn != nil {
			origin = "in-process"
		}
		state := string(o.State)
		if o.Probation && Red(o.State, ScopeCheck) {
			state += " (probation)"
		}
		head := fmt.Sprintf("%s: %s [%s] ", o.Job.Name, state, origin)
		switch {
		case timed(o.State):
			fmt.Fprintf(w, "%s%.1fs\n", head, o.Duration.Seconds())
		case o.State == StateBlocked:
			fmt.Fprintf(w, "%sby %s\n", head, o.BlockedBy)
			continue
		default:
			// The note is the whole output; the header carries it already.
			fmt.Fprintf(w, "%s%s\n", head, strings.TrimSuffix(o.Output, "\n"))
			continue
		}
		if o.Output != "" && (Red(o.State, ScopeCheck) || verbose) {
			fmt.Fprintf(w, "%s\n", strings.TrimSuffix(o.Output, "\n"))
		}
	}
}

// CheckVerdict judges a check: each requested kind must have had something
// to check, and no lane may be red. The notes follow the report on stdout.
func CheckVerdict(kinds []string, outs []Outcome) (code int, notes []string) {
	for _, kind := range kinds {
		ran, na := false, false
		for _, o := range outs {
			if o.Job.Kind != kind {
				continue
			}
			switch o.State {
			case StateNotApplicable:
				na = true
			case StateUnavailable:
			default:
				ran = true
			}
		}
		if !ran && !na {
			notes = append(notes, "nothing to check for `"+kind+"`")
			code = 1
		}
	}
	if slices.ContainsFunc(outs, func(o Outcome) bool { return Fails(o, ScopeCheck) }) {
		code = 1
	}
	return code, notes
}

// Fails says whether an outcome fails a run: a red state of a lane that is
// not in probation.
func Fails(o Outcome, scope Scope) bool {
	return Red(o.State, scope) && !o.Probation
}

// ProbationLine ends the report of a run with a lane in probation: the keys
// of the lanes armed does not arm, sorted. A lane with nothing to check here
// -- no command, no tests -- is left out: it cannot turn green, so it would
// stand in the line for good. "" without such a lane, and with a nil armed.
func ProbationLine(outs []Outcome, armed func(Job) bool) string {
	if armed == nil {
		return ""
	}
	var keys []string
	for _, o := range outs {
		if o.State != StateNotApplicable && o.State != StateUnavailable && !armed(o.Job) {
			keys = append(keys, LaneKey(o.Job))
		}
	}
	if len(keys) == 0 {
		return ""
	}
	slices.Sort(keys)
	return "probation: " + strings.Join(slices.Compact(keys), ", ") + " (warn only until a green commit arms them)"
}

// GreenKeys are the keys of the lanes that ended ok, sorted, each once: what
// a green run may arm.
func GreenKeys(outs []Outcome) []string {
	var keys []string
	for _, o := range outs {
		if o.State == StateOK {
			keys = append(keys, LaneKey(o.Job))
		}
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}

// skipPrefix begins every notice of a lane or a file post-edit did not run.
const skipPrefix = "loomux hook post-tool-use: lane skipped, "

// BudgetSkipped is the notice for a lane or a file the edit budget did not
// reach, named by name: one sentence for both, so the model reads one form.
func BudgetSkipped(name string) string {
	return skipPrefix + "the edit budget ran out: " + name
}

// Skipped says notice, a lane or a file post-edit did not run, on stderr and
// adds it to notices. A host reads stderr at exit 2 and the notices at exit
// 0, so a skip is heard whatever the call ends with. It never writes stdout:
// the notices go there once, when the call's code is known.
func Skipped(stderr io.Writer, notices []string, notice string) []string {
	fmt.Fprintln(stderr, notice)
	return append(notices, notice)
}

// EditReport reports the lanes of one edited file: an armed red lane on
// stderr, which blocks the edit; a red lane in probation, marked so, and
// every lane it had to skip through Skipped, in lane order, without blocking;
// then whatever else the hook has to say, one notice a line. The aside is
// dropped when an armed lane is red: the finding matters more, and stderr
// stays the finding's. The caller hands the notices of every file of a call
// to the host's adapter together, because a host reads stdout as one
// document, and only when the call ends with 0.
func EditReport(stderr io.Writer, outs []Outcome, aside string) (red bool, notices []string) {
	for _, o := range outs {
		switch {
		case Fails(o, ScopeEdit):
			red = true
			fmt.Fprintf(stderr, "%s: %s\n", o.Job.Name, o.State)
			if o.Output != "" {
				fmt.Fprintf(stderr, "%s\n", strings.TrimSuffix(o.Output, "\n"))
			}
		case Red(o.State, ScopeEdit):
			// In probation: said on both streams like a skip, and the edit stands.
			said := fmt.Sprintf("%s: %s (probation)", o.Job.Name, o.State)
			if o.Output != "" {
				said += "\n" + strings.TrimSuffix(o.Output, "\n")
			}
			notices = Skipped(stderr, notices, said)
		case o.State == StateBudget:
			notices = Skipped(stderr, notices, BudgetSkipped(o.Job.Name))
		case o.State == StateMissingTool, o.State == StateUnready:
			notices = Skipped(stderr, notices, skipPrefix+o.Output)
		}
	}
	if aside != "" && !red {
		notices = append(notices, aside)
	}
	return red, notices
}
