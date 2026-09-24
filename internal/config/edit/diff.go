package edit

import "strings"

// Diff is a small line diff: the changed lines with one line of context
// around each run. Configuration files are short; an LCS is not needed to
// show a one-line change, and every change here is one key.
func Diff(before, after string) string {
	a := strings.Split(strings.TrimSuffix(before, "\n"), "\n")
	b := strings.Split(strings.TrimSuffix(after, "\n"), "\n")
	start := 0
	for start < len(a) && start < len(b) && a[start] == b[start] {
		start++
	}
	endA, endB := len(a), len(b)
	for endA > start && endB > start && a[endA-1] == b[endB-1] {
		endA--
		endB--
	}
	if start == endA && start == endB {
		return ""
	}
	var out strings.Builder
	if start > 0 {
		out.WriteString("  " + a[start-1] + "\n")
	}
	for _, l := range a[start:endA] {
		out.WriteString("- " + l + "\n")
	}
	for _, l := range b[start:endB] {
		out.WriteString("+ " + l + "\n")
	}
	if endA < len(a) {
		out.WriteString("  " + a[endA] + "\n")
	}
	return out.String()
}
