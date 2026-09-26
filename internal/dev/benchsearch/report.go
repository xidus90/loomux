package benchsearch

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// Latency is the latency pass together with what it read and asked: read
// latency follows the document's size and the query decides which paths of
// the chain ran, so timings without both compare with nothing.
type Latency struct {
	Document, Query string
	Timings         []benchreport.Timing
}

// Run is everything one search bench leaves behind. The head is not
// decoration: two numbers produced under different models are no series,
// and without the head nobody can tell a year later.
type Run struct {
	Stamp       string
	Profile     string
	Environment benchreport.Environment
	Documents   int
	QuestionSet string
	Corpus      string // "" outside corpus mode
	Outcomes    []Outcome
	Findings    []string
	Latency     *Latency
}

// The notes that stand beside the numbers, because the misreading happens
// while reading the table.
const (
	fastCaveat = "This run is **purely vectorial** (`qmd vsearch`) and therefore measures " +
		"**both** the embedding model's share of the language bridge and whether `fast` " +
		"carries across languages."
	corpusCaveat = "These numbers measure **regression** against an artificial stock. They say " +
		"whether the chain got worse than at the last stand — and they can carry **no decision**, " +
		"neither about the architecture nor about a model. That is what the real question set is for."
	findingsIntro = "Messages of the search chain during the run. They belong beside the numbers: " +
		"a miss with a finding may be no measurement at all but a silent failure of the engine."
	warmth = `"Warm" here means the chain has already searched in this run: ` +
		"the latency runs after the quality pass."
)

// Markdown is the run as a page to read.
func Markdown(r Run) string {
	env := r.Environment
	lines := []string{
		fmt.Sprintf("# Search bench %s — profile `%s`", r.Stamp, r.Profile),
		"",
		"- qmd: " + env.Qmd,
		"- models: " + models(env.Models),
		fmt.Sprintf("- system: %s/%s, %s", env.OS, env.Arch, env.CPU),
		"- loomux: " + env.Loomux,
		"- search path: " + env.Port,
		fmt.Sprintf("- indexed documents: %d", r.Documents),
		"- question set: " + r.QuestionSet,
	}
	// The stand is a fact about the run like the machine, and the caveat
	// keeps a reader from spending its numbers on a decision.
	if r.Corpus != "" {
		lines = append(lines, "- corpus: "+r.Corpus, "", corpusCaveat)
	}
	if r.Profile == string(search.ProfileFast) {
		lines = append(lines, "", fastCaveat)
	}
	lines = append(lines, quality(r.Outcomes)...)
	lines = append(lines, findings(r.Findings)...)
	if r.Latency != nil {
		lines = append(lines, latency(*r.Latency)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

// models labels every name, sorted: a map has no order a reader could guess.
func models(names map[string]string) string {
	pairs := make([]string, 0, len(names))
	for _, key := range slices.Sorted(maps.Keys(names)) {
		pairs = append(pairs, key+"="+names[key])
	}
	return strings.Join(pairs, ", ")
}

// quality is the hit table by kind, every kind listed even when empty, and
// the list of misses.
func quality(outcomes []Outcome) []string {
	hits, totals := map[Kind]int{}, map[Kind]int{}
	var hitMS, missMS []float64
	var misses []string
	for _, o := range outcomes {
		totals[o.Question.Kind]++
		if o.Hit {
			hits[o.Question.Kind]++
			hitMS = append(hitMS, o.ElapsedMS)
			continue
		}
		missMS = append(missMS, o.ElapsedMS)
		where := "not found"
		if o.Rank > 0 {
			where = fmt.Sprintf("rank %d", o.Rank)
		}
		misses = append(misses, fmt.Sprintf("- %s (%s): %s — %s", o.Question.ID, o.Question.Kind, where, o.Question.Query))
	}
	lines := []string{"", "## Hit quality", "", "| Sort | Hits |", "|---|---|"}
	for _, k := range kinds {
		lines = append(lines, fmt.Sprintf("| %s | %d/%d |", k, hits[k], totals[k]))
	}
	lines = append(lines,
		fmt.Sprintf("| total | %d/%d |", len(hitMS), len(outcomes)),
		"",
		"Median hits: "+median(hitMS),
		"Median misses: "+median(missMS),
		"",
		"## Misses",
		"",
	)
	return append(lines, orNone(misses)...)
}

// findings is what the chain said about its own answers, beside the numbers.
func findings(found []string) []string {
	items := make([]string, 0, len(found))
	for _, f := range found {
		items = append(items, "- "+f)
	}
	return append([]string{"", "## Findings", "", findingsIntro, ""}, orNone(items)...)
}

// orNone says so for an empty list: an omitted one would read as "not looked at".
func orNone(items []string) []string {
	if len(items) == 0 {
		return []string{"none"}
	}
	return items
}

// median is the middle in whole milliseconds; an empty group has no median,
// not a zero.
func median(ms []float64) string {
	if len(ms) == 0 {
		return "—"
	}
	return wholeMS(benchreport.Median(ms))
}

func wholeMS(ms float64) string { return fmt.Sprintf("%.0f ms", ms) }

func latency(l Latency) []string {
	lines := []string{
		"", "## Latency", "", warmth, "",
		fmt.Sprintf("- document read: `%s`", l.Document),
		fmt.Sprintf("- query: `%s`", l.Query),
		"",
		"| Operation | cold | warm median | min | max |",
		"|---|---|---|---|---|",
	}
	for _, t := range l.Timings {
		lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s | %s |", t.Name,
			wholeMS(t.ColdMS), wholeMS(t.MedianMS), wholeMS(t.MinMS), wholeMS(t.MaxMS)))
	}
	return lines
}

type payload struct {
	QuestionSet string           `json:"question_set"`
	Corpus      *string          `json:"corpus"` // null outside corpus mode
	Documents   int              `json:"documents"`
	Questions   []questionRecord `json:"questions"`
	Findings    []string         `json:"findings"` // [] never null
	Latency     *latencyTarget   `json:"latency"`  // null without --latency
}

type questionRecord struct {
	ID        string  `json:"id"`
	Sort      Kind    `json:"sort"`
	Rank      *int    `json:"rank"` // null: not found
	Hit       bool    `json:"hit"`
	ElapsedMS float64 `json:"elapsed_ms"`
}

type latencyTarget struct {
	Document string `json:"document"`
	Query    string `json:"query"`
}

// Envelope is the run in the shared shape of every bench. Null and an empty
// list say different things: null was not measured, an empty list found
// nothing.
func Envelope(r Run) (benchreport.Report, error) {
	p := payload{QuestionSet: r.QuestionSet, Documents: r.Documents,
		Questions: make([]questionRecord, 0, len(r.Outcomes)), Findings: []string{}}
	if r.Corpus != "" {
		p.Corpus = &r.Corpus
	}
	for _, o := range r.Outcomes {
		rec := questionRecord{ID: o.Question.ID, Sort: o.Question.Kind, Hit: o.Hit, ElapsedMS: o.ElapsedMS}
		if o.Rank > 0 {
			rec.Rank = &o.Rank
		}
		p.Questions = append(p.Questions, rec)
	}
	p.Findings = append(p.Findings, r.Findings...)
	timings := []benchreport.Timing{}
	if r.Latency != nil {
		p.Latency = &latencyTarget{Document: r.Latency.Document, Query: r.Latency.Query}
		timings = append(timings, r.Latency.Timings...)
	}
	data, err := json.Marshal(p)
	if err != nil {
		return benchreport.Report{}, err
	}
	// The run names its profile once; the head carries it for every reader
	// of the shared shape.
	env := r.Environment
	env.Profile = r.Profile
	return benchreport.Report{Schema: benchreport.Schema, Command: "search", Stamp: r.Stamp,
		Environment: env, Timings: timings, Payload: data}, nil
}
