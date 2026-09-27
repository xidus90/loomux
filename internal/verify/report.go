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
		head := fmt.Sprintf("%s: %s [%s] ", o.Job.Name, o.State, origin)
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
	if slices.ContainsFunc(outs, func(o Outcome) bool { return Red(o.State, ScopeCheck) }) {
		code = 1
	}
	return code, notes
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

// EditReport reports the lanes of one edited file: red lanes on stderr,
// which blocks the edit, and every lane it had to skip through Skipped, in
// lane order, then whatever else the hook has to say, one notice a line. The
// aside is dropped when a lane is red: the finding matters more, and stderr
// stays the finding's. The caller hands the notices of every file of a call
// to the host's adapter together, because a host reads stdout as one
// document, and only when the call ends with 0.
func EditReport(stderr io.Writer, outs []Outcome, aside string) (red bool, notices []string) {
	for _, o := range outs {
		switch {
		case Red(o.State, ScopeEdit):
			red = true
			fmt.Fprintf(stderr, "%s: %s\n", o.Job.Name, o.State)
			if o.Output != "" {
				fmt.Fprintf(stderr, "%s\n", strings.TrimSuffix(o.Output, "\n"))
			}
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
