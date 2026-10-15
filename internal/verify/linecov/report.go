// Package linecov holds every executable line of a project to one rule: it
// ran, or it carries an exemption with a reason. Reports, scope and markers
// are read from the project itself; see docs/en/cli-reference.md,
// "loomux check linecov".
package linecov

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
)

// Report maps a slash path relative to the root to the hits of each line.
type Report map[string]map[int]uint64

// File returns the lines of path, adding the file when the report lacks it:
// a file a report names without lines still counts as reported.
func (r Report) File(path string) map[int]uint64 {
	lines, ok := r[path]
	if !ok {
		lines = map[int]uint64{}
		r[path] = lines
	}
	return lines
}

// Add sums hits into a line; merged reports add up, they never replace.
func (r Report) Add(path string, line int, hits uint64) {
	lines := r.File(path)
	lines[line] = addSat(lines[line], hits)
}

// Lines counts the line records over every file.
func (r Report) Lines() int {
	n := 0
	for _, lines := range r {
		n += len(lines)
	}
	return n
}

// addSat adds without wrapping: a wrapped sum could read as zero hits.
func addSat(a, b uint64) uint64 {
	if a > math.MaxUint64-b {
		return math.MaxUint64
	}
	return a + b
}

// Finding is one line of the gate's output. Line 0 is about a whole file,
// an empty Path about the run.
type Finding struct {
	Path string
	Line int
	Text string
}

func (f Finding) String() string {
	switch {
	case f.Path == "":
		return f.Text
	case f.Line > 0:
		return fmt.Sprintf("%s:%d %s", f.Path, f.Line, f.Text)
	}
	return f.Path + " " + f.Text
}

// SortFindings orders by path, line and text and drops repeats: two profiles
// that know the same marker must not report it twice.
func SortFindings(fs []Finding) []Finding {
	slices.SortFunc(fs, func(a, b Finding) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), strings.Compare(a.Text, b.Text))
	})
	return slices.Compact(fs)
}

// text decodes leniently: a refused byte would fail the whole gate on one
// mis-encoded file, and a replaced one cannot hide a marker. A leading BOM is
// dropped, so the first line reads like any other.
func text(content []byte) string {
	return strings.TrimPrefix(strings.ToValidUTF8(string(content), "�"), string(rune(0xFEFF)))
}
