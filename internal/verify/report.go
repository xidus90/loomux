package verify

import (
	"encoding/json"
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

// WriteEdit reports a post-edit run: red lanes on stderr, which blocks the
// edit with 2, and as an aside for the model on stdout the lanes it had to
// skip and whatever else the hook has to say. The aside is dropped when a
// lane is red: the finding matters more, and stderr stays the finding's.
func WriteEdit(stdout, stderr io.Writer, outs []Outcome, aside string) int {
	code := 0
	var skipped strings.Builder
	const prefix = "loomux hook post-tool-use: lane skipped, "
	for _, o := range outs {
		switch {
		case Red(o.State, ScopeEdit):
			code = 2
			fmt.Fprintf(stderr, "%s: %s\n", o.Job.Name, o.State)
			if o.Output != "" {
				fmt.Fprintf(stderr, "%s\n", strings.TrimSuffix(o.Output, "\n"))
			}
		case o.State == StateBudget:
			skipped.WriteString(prefix + "the edit budget ran out: " + o.Job.Name + "\n")
		case o.State == StateMissingTool, o.State == StateUnready:
			skipped.WriteString(prefix + o.Output + "\n")
		}
	}
	notices := skipped.String()
	if aside != "" && code == 0 {
		notices += aside + "\n"
	}
	writeSkipped(stdout, notices)
	return code
}

// writeSkipped puts the dropped lanes where a PostToolUse hook exiting 0 is
// read: `hookSpecificOutput.additionalContext`, the field the Claude adapter
// writes for the model. Nothing is written when nothing was skipped, because
// stdout that is not valid JSON turns a passed hook into a hook-error notice.
func writeSkipped(stdout io.Writer, notices string) {
	if notices == "" {
		return
	}
	type specific struct {
		AdditionalContext string `json:"additionalContext"`
		HookEventName     string `json:"hookEventName"`
	}
	document := struct {
		HookSpecificOutput specific `json:"hookSpecificOutput"`
	}{specific{strings.TrimRight(notices, "\n"), "PostToolUse"}}
	// A struct of strings always encodes; there is no error to report.
	encoded, _ := json.Marshal(document)
	fmt.Fprintf(stdout, "%s\n", encoded)
}
