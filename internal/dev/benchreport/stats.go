// Package benchreport is the one shape every benchmark of loomux leaves
// behind: a head naming where it ran, the timings of one run, and a payload
// of the command's own.
package benchreport

import (
	"fmt"
	"slices"
	"time"
)

// MS is a duration in milliseconds, the unit every report carries. It is
// the very division the renderers did before, so a stored value renders to
// the same text.
func MS(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// Median is the middle of the warm runs; an even count averages the two
// middle values, so a run of four does not report the upper one.
func Median(ms []float64) float64 {
	if len(ms) == 0 {
		return 0
	}
	s := slices.Clone(ms)
	slices.Sort(s)
	half := len(s) / 2
	if len(s)%2 == 1 {
		return s[half]
	}
	return (s[half-1] + s[half]) / 2
}

// Summarize keeps the cold run apart: folded into the median it would hide
// both itself and the warm spread.
func Summarize(name string, coldMS float64, warmMS []float64) Timing {
	t := Timing{Name: name, ColdMS: coldMS, WarmMS: warmMS, MedianMS: Median(warmMS)}
	if len(warmMS) > 0 {
		t.MinMS, t.MaxMS = slices.Min(warmMS), slices.Max(warmMS)
	}
	return t
}

// FormatMS is the one spelling of a time in a table.
func FormatMS(ms float64) string { return fmt.Sprintf("%.1f ms", ms) }

// Row is a timing as a table row: cold, warm median, minimum, maximum.
func (t Timing) Row() string {
	return fmt.Sprintf("| %s | %s | %s | %s | %s |", t.Name,
		FormatMS(t.ColdMS), FormatMS(t.MedianMS), FormatMS(t.MinMS), FormatMS(t.MaxMS))
}
