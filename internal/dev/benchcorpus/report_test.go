package benchcorpus

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

func TestFormatMarkdown(t *testing.T) {
	t.Run("Single repository report", func(t *testing.T) {
		report := &BenchmarkReport{
			Timestamp: "2026-09-18T15:35:00Z",
			Mode:      "single",
			WarmRuns:  3,
			Repos: []*RepoAudit{
				{
					Dir:            "/path/to/project",
					DetectedStacks: []string{"python"},
					ExecutedLanes:  []string{"ruff check ."},
					CoverageRate:   66.7,
					MissingGaps: []string{
						"pytest deklariert, aber keine Lane in Loomux vorhanden",
					},
					Audit: []CheckAudit{
						{Tool: "ruff", Category: "lint", Native: "pyproject.toml", Lane: "ruff check .", OnPath: true},
						{Tool: "mypy", Category: "typecheck", Native: "pyproject.toml", Lane: "mypy .", OnPath: false},
						{Tool: "pytest", Category: "test", Native: "pytest.ini", Lane: "", OnPath: true},
					},
					Timings: timings(benchreport.Summarize("", 50, []float64{30, 32}),
						comp("pre-tool-use", 10, 8, 9),
						comp("post-tool-use", 40, 22, 23),
						notApplicable("graph build")),
					ClaudeWarmMed: 90,
					Speedup:       2.9,
				},
			},
		}

		var buf bytes.Buffer
		if err := FormatMarkdown(report, &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "# Loomux Benchmark & Lücken-Audit") {
			t.Errorf("missing header in markdown")
		}
		if !strings.Contains(out, "2026-09-18T15:35:00Z") {
			t.Errorf("missing timestamp")
		}
		if !strings.Contains(out, "pre-tool-use") {
			t.Errorf("missing pre-tool-use row")
		}
		if !strings.Contains(out, "graph build") || !strings.Contains(out, "n/a") {
			t.Errorf("missing graph build n/a row")
		}
		if !strings.Contains(out, "**Baseline Claude Hook:** 90.0 ms (Speedup: 2.9x)") {
			t.Errorf("missing Claude baseline speedup")
		}
		if !strings.Contains(out, "✅ Aktiv") || !strings.Contains(out, "❌ Nicht im PATH") || !strings.Contains(out, "⚠️ Fehlt in Loomux") {
			t.Errorf("missing check audit status indicators")
		}
		if !strings.Contains(out, "**Abdeckungsquote:** **66.7 %**") {
			t.Errorf("missing coverage rate: %s", out)
		}
	})

	t.Run("Corpus report with multiple repositories", func(t *testing.T) {
		report := &BenchmarkReport{
			Timestamp: "2026-09-18T16:00:00Z",
			Mode:      "corpus",
			WarmRuns:  1,
			Repos: []*RepoAudit{
				{
					RepoURL:      "https://github.com/foo/repo1",
					Language:     "Go",
					Tier:         "Sehr viel",
					CommitSHA:    "abc1234",
					CoverageRate: 100.0,
					Timings: timings(benchreport.Summarize("", 20, []float64{18}),
						comp("pre-tool-use", 5, 4),
						comp("post-tool-use", 10, 9),
						comp("graph build", 5, 5)),
				},
				{
					Dir:          "/local/repo2",
					CoverageRate: 100.0,
					Timings: timings(benchreport.Summarize("", 10, []float64{9}),
						comp("pre-tool-use", 4, 3),
						comp("post-tool-use", 6, 6),
						notApplicable("graph build")),
				},
			},
		}

		var buf bytes.Buffer
		if err := FormatMarkdown(report, &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		out := buf.String()
		if !strings.Contains(out, "Repository #1: https://github.com/foo/repo1") {
			t.Errorf("missing Repo 1 header")
		}
		if !strings.Contains(out, "**Commit:** abc1234") {
			t.Errorf("missing commit sha")
		}
		if !strings.Contains(out, "Repository #2: /local/repo2") {
			t.Errorf("missing Repo 2 header")
		}
	})
}

// The JSON of a run is the shared report: each repository's total under the
// name its markdown section carries, the rows themselves in the payload.
func TestReportJSONIsTheSharedReport(t *testing.T) {
	report := &BenchmarkReport{
		Timestamp: "2026-09-18T15:35:00Z",
		Mode:      "corpus",
		WarmRuns:  2,
		Repos: []*RepoAudit{
			{RepoURL: "https://github.com/foo/repo1", Timings: timings(benchreport.Timing{MedianMS: 12.5})},
			{Dir: "/local/repo2", Timings: timings(benchreport.Timing{MedianMS: 7})},
			{Dir: "/local/unmeasured"},
		},
		Skipped: []SkippedRepo{{RepoURL: "https://github.com/x/y", Reason: "clone failed"}},
	}
	env := benchreport.Environment{OS: "windows", Arch: "amd64", CPU: "cpu", Go: "go1.27", Loomux: "v9"}

	data, err := ReportJSON(report, "2026-09-18-1535", env)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		benchreport.Report
		Payload struct {
			Repos   []RepoAudit   `json:"repos"`
			Skipped []SkippedRepo `json:"skipped"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("%v\n%s", err, data)
	}
	if parsed.Schema != benchreport.Schema || parsed.Command != "repos" || parsed.Stamp != "2026-09-18-1535" || !reflect.DeepEqual(parsed.Environment, env) {
		t.Fatalf("head: %+v", parsed.Report)
	}
	if len(parsed.Timings) != 2 || parsed.Timings[0].Name != "https://github.com/foo/repo1" ||
		parsed.Timings[0].MedianMS != 12.5 || parsed.Timings[1].Name != "/local/repo2" {
		t.Fatalf("timings: %+v", parsed.Timings)
	}
	if len(parsed.Payload.Repos) != 3 || len(parsed.Payload.Skipped) != 1 || parsed.Payload.Skipped[0].Reason != "clone failed" {
		t.Fatalf("payload: %s", data)
	}
	// The repository keeps its own total: the renamed copy lives in the head only.
	if report.Repos[0].Timings[0].Name != TotalTiming {
		t.Fatalf("the audit's total was renamed: %+v", report.Repos[0].Timings)
	}
}

// A run without repositories or skips says so with empty lists, not null.
func TestReportJSONWritesEmptyListsForAnEmptyRun(t *testing.T) {
	data, err := ReportJSON(&BenchmarkReport{}, "s", benchreport.Environment{})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{`"timings": []`, `"repos": []`, `"skipped": []`} {
		if !strings.Contains(text, want) {
			t.Fatalf("lacks %s:\n%s", want, text)
		}
	}
}

func TestReportJSONRefusesARowJSONCannotCarry(t *testing.T) {
	report := &BenchmarkReport{Repos: []*RepoAudit{{Dir: "/r", Speedup: math.NaN()}}}
	if _, err := ReportJSON(report, "s", benchreport.Environment{}); err == nil {
		t.Fatal("NaN encoded")
	}
}
