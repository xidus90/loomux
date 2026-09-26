package benchreport

import (
	"encoding/json"
	"time"
)

// Schema is the version of the report's JSON; it rises when a field changes
// meaning or goes away.
const Schema = 1

// Timing is one measured thing of a run, all times in milliseconds.
type Timing struct {
	Name       string    `json:"name"`
	ColdMS     float64   `json:"cold_ms"`
	WarmMS     []float64 `json:"warm_ms"`
	MedianMS   float64   `json:"median_ms"`
	MinMS      float64   `json:"min_ms"`
	MaxMS      float64   `json:"max_ms"`
	ExitCodes  []int     `json:"exit_codes,omitempty"`
	Applicable *bool     `json:"applicable,omitempty"` // nil: applicable
	TimedOut   int       `json:"timed_out,omitempty"`  // runs that hit their deadline
}

// Environment is where a run happened, so two reports can be told apart.
type Environment struct {
	OS      string            `json:"os"`
	Arch    string            `json:"arch"`
	CPU     string            `json:"cpu"`
	Go      string            `json:"go"`
	Loomux  string            `json:"loomux"`
	Qmd     string            `json:"qmd,omitempty"`
	Models  map[string]string `json:"models,omitempty"`
	Profile string            `json:"profile,omitempty"`
	Port    string            `json:"port,omitempty"` // "daemon" | "cli"
	// An index embedded on one backbone and searched on another ranks
	// nonsense, so two runs that differ only here are no series.
	Backbone string `json:"backbone,omitempty"` // see Backbone
}

// Report is one run of one bench command, as written next to its markdown.
type Report struct {
	Schema      int             `json:"schema"`
	Command     string          `json:"command"` // "hooks" | "repos" | "search"
	Stamp       string          `json:"stamp"`
	Environment Environment     `json:"environment"`
	Timings     []Timing        `json:"timings"`
	Payload     json.RawMessage `json:"payload,omitempty"`
}

// Stamp is the minute a run started, in UTC, as file names and heads carry it.
func Stamp(now time.Time) string { return now.UTC().Format("2006-01-02-1504") }

// JSON is the report as written to disk: indented by two, one newline at the end.
func (r Report) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
