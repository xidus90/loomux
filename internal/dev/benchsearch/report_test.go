package benchsearch

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

// sampleRun is an everyday run under profile fast: a hit at rank one, a miss
// beyond the top three, a miss not found at all, and one finding. With
// latency it carries the five operations of a latency pass.
func sampleRun(latency bool) Run {
	r := Run{
		Stamp:   "2026-09-26-1200",
		Scope:   "knowledge",
		Profile: "fast",
		Environment: benchreport.Environment{
			OS: "windows", Arch: "amd64", CPU: "Test CPU", Go: "go1.26.0", Loomux: "1.2.3",
			Qmd: "2.8.3",
			// Declared out of order: the report sorts the keys.
			Models:   map[string]string{"rerank": "qwen3-reranker", "embedding": "embeddinggemma", "query_expansion": "qmd-query-expansion"},
			Port:     "daemon",
			Backbone: "unknown",
		},
		Documents:   12,
		QuestionSet: "C:/sets/questions.yaml",
		Outcomes: []Outcome{
			{Question: Question{ID: "c01", Kind: Exact, Query: "stamp cache"}, Rank: 1, Hit: true, ElapsedMS: 12.4},
			{Question: Question{ID: "c02", Kind: Paraphrase, Query: "wie heißt das Ding | mit dem Strich"}, Rank: 5, ElapsedMS: 30.6},
			{Question: Question{ID: "c03", Kind: CrossLingual, Query: "how does the index stay fresh"}, ElapsedMS: 41.2},
		},
		Findings: []string{"qmd: 1 document skipped (unreadable)"},
	}
	if latency {
		r.Latency = &Latency{
			Document: "notes/alpha.md",
			Query:    "how does the index stay fresh",
			Timings: []benchreport.Timing{
				benchreport.Summarize("catalog", 3.2, []float64{1.1, 0.9, 1.4}),
				benchreport.Summarize("read", 5.6, []float64{2.4, 2.7, 2.2}),
				benchreport.Summarize("keyword", 41.0, []float64{18.3, 17.9, 19.6}),
				benchreport.Summarize("vector", 812.4, []float64{10.2, 11.6, 9.4}),
				benchreport.Summarize("hybrid", 2410.7, []float64{640.2, 655.9, 701.3}),
			},
		}
	}
	return r
}

func readGolden(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// diffLines names every line where the two texts part, both sides quoted so
// a trailing space or a missing newline shows.
func diffLines(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	for i := range max(len(w), len(g)) {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl || (i >= len(w)) != (i >= len(g)) {
			fmt.Fprintf(&b, "line %d\n  want %q\n  got  %q\n", i+1, wl, gl)
		}
	}
	return b.String()
}

func TestMarkdownOfAnEverydayRun(t *testing.T) {
	got := Markdown(sampleRun(false))
	want := readGolden(t, "testdata/report-everyday.md")
	if got != want {
		t.Fatalf("diff:\n%s", diffLines(want, got))
	}
}

func TestMarkdownOfACorpusRunCarriesTheCaveat(t *testing.T) {
	r := sampleRun(true)
	r.Profile = "full"
	r.Corpus = "v1"
	r.Environment.Backbone = "vulkan"
	got := Markdown(r)
	want := readGolden(t, "testdata/report-corpus.md")
	if got != want {
		t.Fatalf("diff:\n%s", diffLines(want, got))
	}
}

// An empty list still gets its heading and says so: an omitted one would
// read as "not looked at", and an empty group has no median, not a zero.
func TestMarkdownOfACleanRunSaysNone(t *testing.T) {
	r := sampleRun(false)
	r.Outcomes, r.Findings = r.Outcomes[:1], nil
	got := Markdown(r)
	for _, want := range []string{
		"Median misses: —\n",
		"## Misses\n\nnone\n",
		"failure of the engine.\n\nnone\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

func TestEnvelopeCarriesTheQuestionsAndTheLatencyTarget(t *testing.T) {
	r := sampleRun(true)
	r.Corpus = "v1"
	env, err := Envelope(r)
	if err != nil {
		t.Fatal(err)
	}
	if env.Command != "search" || env.Schema != 1 || len(env.Timings) != 5 {
		t.Fatalf("%+v", env)
	}
	// The run names its profile once; the head of the shared shape must
	// carry it whether or not the caller filled it there too.
	if env.Stamp != r.Stamp || env.Environment.Profile != "fast" || env.Environment.Qmd != "2.8.3" {
		t.Fatalf("%+v", env)
	}
	var payload struct {
		QuestionSet string  `json:"question_set"`
		Corpus      *string `json:"corpus"`
		Documents   int     `json:"documents"`
		Questions   []struct {
			ID        string  `json:"id"`
			Sort      string  `json:"sort"`
			Rank      *int    `json:"rank"`
			Hit       bool    `json:"hit"`
			ElapsedMS float64 `json:"elapsed_ms"`
		} `json:"questions"`
		Findings []string                          `json:"findings"`
		Latency  *struct{ Document, Query string } `json:"latency"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Questions[2].Rank != nil { // the miss not found
		t.Fatalf("a miss must carry rank null")
	}
	q := payload.Questions[1]
	if q.ID != "c02" || q.Sort != "umschreibung" || q.Rank == nil || *q.Rank != 5 || q.Hit || q.ElapsedMS != 30.6 {
		t.Fatalf("%+v", q)
	}
	if payload.QuestionSet != r.QuestionSet || payload.Corpus == nil || *payload.Corpus != "v1" || payload.Documents != 12 ||
		len(payload.Findings) != 1 || payload.Latency == nil || payload.Latency.Document != "notes/alpha.md" ||
		payload.Latency.Query != r.Latency.Query {
		t.Fatalf("%+v", payload)
	}
}

// Outside corpus mode and without latency the payload says null for what was
// not measured, and empty lists stay lists: a reader tells "none" from "absent".
func TestEnvelopeOfAPlainRunSaysNull(t *testing.T) {
	r := sampleRun(false)
	r.Findings = nil
	env, err := Envelope(r)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(env.Payload, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["corpus"]) != "null" || string(raw["latency"]) != "null" || string(raw["findings"]) != "[]" {
		t.Fatalf("%s", env.Payload)
	}
	data, err := env.JSON()
	if err != nil || !strings.Contains(string(data), `"timings": []`) {
		t.Fatalf("%s %v", data, err)
	}
}

func TestEnvelopeRefusesATimeJSONCannotHold(t *testing.T) {
	r := sampleRun(false)
	r.Outcomes[0].ElapsedMS = math.NaN()
	if _, err := Envelope(r); err == nil {
		t.Fatal("want an error for NaN")
	}
}
