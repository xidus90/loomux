package benchcorpus

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// exitStatus condenses the outcomes of every run of timings into one table
// cell: a single timeout outweighs every exit code, since its code says nothing.
func exitStatus(timings ...benchreport.Timing) string {
	var codes []int
	for _, t := range timings {
		if t.TimedOut > 0 {
			return "timeout"
		}
		for _, code := range t.ExitCodes {
			if !slices.Contains(codes, code) {
				codes = append(codes, code)
			}
		}
	}
	if len(codes) == 0 {
		return "n/a"
	}
	slices.Sort(codes)
	parts := make([]string, len(codes))
	for i, code := range codes {
		parts[i] = fmt.Sprint(code)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// allRuns gathers every applicable component, whose runs the total row reports.
func allRuns(audit *RepoAudit) []benchreport.Timing {
	var runs []benchreport.Timing
	for _, c := range audit.Components() {
		if c.Applicable == nil || *c.Applicable {
			runs = append(runs, c)
		}
	}
	return runs
}

// baselineRuns gathers the Claude hooks the row was compared against.
func baselineRuns(audit *RepoAudit) []benchreport.Timing {
	var runs []benchreport.Timing
	for _, t := range audit.Timings {
		if strings.HasPrefix(t.Name, baselinePrefix) {
			runs = append(runs, t)
		}
	}
	return runs
}

// baselineStatus appends the exit status of the Claude baseline hooks to the
// baseline line, when any were measured.
func baselineStatus(audit *RepoAudit) string {
	runs := baselineRuns(audit)
	if len(runs) == 0 {
		return ""
	}
	return ", Status: " + exitStatus(runs...)
}

// hookComparison names the loomux side of the speedup, which counts the two
// edit hooks only and leaves graph build out.
func hookComparison(audit *RepoAudit) string {
	if audit.HookWarmMedian == 0 {
		return ""
	}
	return " vs. loomux hooks " + benchreport.FormatMS(audit.HookWarmMedian)
}

// unavailableWord marks a baseline that could not be measured.
func unavailableWord(isDE bool) string {
	if isDE {
		return "nicht verfügbar"
	}
	return "unavailable"
}
