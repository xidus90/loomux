package benchcorpus

import (
	"fmt"
	"slices"
	"strings"
)

// exitStatus condenses the outcomes of several runs into one table cell: a
// single timeout outweighs every exit code, since its code says nothing.
func exitStatus(runs []ComponentTiming) string {
	var codes []int
	for _, run := range runs {
		if run.TimedOut {
			return "timeout"
		}
		if !slices.Contains(codes, run.ExitCode) {
			codes = append(codes, run.ExitCode)
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

// componentRuns gathers the cold and warm measurements of one component.
func componentRuns(cold TimingRun, warm []TimingRun, idx int) []ComponentTiming {
	runs := []ComponentTiming{cold.Components[idx]}
	for _, w := range warm {
		if idx < len(w.Components) {
			runs = append(runs, w.Components[idx])
		}
	}
	return runs
}

// allRuns gathers every applicable measurement, which the total row reports.
func allRuns(cold TimingRun, warm []TimingRun) []ComponentTiming {
	var runs []ComponentTiming
	for _, run := range append([]TimingRun{cold}, warm...) {
		for _, comp := range run.Components {
			if comp.Applicable {
				runs = append(runs, comp)
			}
		}
	}
	return runs
}

// baselineRuns gathers every baseline hook call of the cold and warm passes.
func baselineRuns(cold TimingRun, warm []TimingRun) []ComponentTiming {
	runs := slices.Clone(cold.Baseline)
	for _, w := range warm {
		runs = append(runs, w.Baseline...)
	}
	return runs
}

// baselineStatus appends the exit status of the Claude baseline hooks to the
// baseline line, when any were measured.
func baselineStatus(audit *RepoAudit) string {
	runs := baselineRuns(audit.Cold, audit.Warm)
	if len(runs) == 0 {
		return ""
	}
	return ", Status: " + exitStatus(runs)
}

// hookComparison names the loomux side of the speedup, which counts the two
// edit hooks only and leaves graph build out.
func hookComparison(audit *RepoAudit) string {
	if audit.HookWarmMedian == 0 {
		return ""
	}
	return " vs. loomux hooks " + formatDuration(audit.HookWarmMedian)
}

// unavailableWord marks a baseline that could not be measured.
func unavailableWord(isDE bool) string {
	if isDE {
		return "nicht verfügbar"
	}
	return "unavailable"
}
