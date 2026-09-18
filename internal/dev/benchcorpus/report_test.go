package benchcorpus

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
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
					Cold: TimingRun{
						Total: 50 * time.Millisecond,
						Components: []ComponentTiming{
							{Name: "pre-tool-use", Applicable: true, Elapsed: 10 * time.Millisecond},
							{Name: "post-tool-use", Applicable: true, Elapsed: 40 * time.Millisecond},
							{Name: "graph build", Applicable: false, Elapsed: 0},
						},
					},
					Warm: []TimingRun{
						{
							Total: 30 * time.Millisecond,
							Components: []ComponentTiming{
								{Name: "pre-tool-use", Applicable: true, Elapsed: 8 * time.Millisecond},
								{Name: "post-tool-use", Applicable: true, Elapsed: 22 * time.Millisecond},
								{Name: "graph build", Applicable: false, Elapsed: 0},
							},
						},
						{
							Total: 32 * time.Millisecond,
							Components: []ComponentTiming{
								{Name: "pre-tool-use", Applicable: true, Elapsed: 9 * time.Millisecond},
								{Name: "post-tool-use", Applicable: true, Elapsed: 23 * time.Millisecond},
								{Name: "graph build", Applicable: false, Elapsed: 0},
							},
						},
					},
					WarmMedian:    31 * time.Millisecond,
					WarmMin:       30 * time.Millisecond,
					WarmMax:       32 * time.Millisecond,
					ClaudeWarmMed: 90 * time.Millisecond,
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
					Cold: TimingRun{
						Total: 20 * time.Millisecond,
						Components: []ComponentTiming{
							{Name: "pre-tool-use", Applicable: true, Elapsed: 5 * time.Millisecond},
							{Name: "post-tool-use", Applicable: true, Elapsed: 10 * time.Millisecond},
							{Name: "graph build", Applicable: true, Elapsed: 5 * time.Millisecond},
						},
					},
					Warm: []TimingRun{
						{
							Total: 18 * time.Millisecond,
							Components: []ComponentTiming{
								{Name: "pre-tool-use", Applicable: true, Elapsed: 4 * time.Millisecond},
								{Name: "post-tool-use", Applicable: true, Elapsed: 9 * time.Millisecond},
								{Name: "graph build", Applicable: true, Elapsed: 5 * time.Millisecond},
							},
						},
					},
					WarmMedian: 18 * time.Millisecond,
					WarmMin:    18 * time.Millisecond,
					WarmMax:    18 * time.Millisecond,
				},
				{
					Dir:          "/local/repo2",
					CoverageRate: 100.0,
					Cold: TimingRun{
						Total: 10 * time.Millisecond,
						Components: []ComponentTiming{
							{Name: "pre-tool-use", Applicable: true, Elapsed: 4 * time.Millisecond},
							{Name: "post-tool-use", Applicable: true, Elapsed: 6 * time.Millisecond},
							{Name: "graph build", Applicable: false, Elapsed: 0},
						},
					},
					Warm: []TimingRun{
						{
							Total: 9 * time.Millisecond,
							Components: []ComponentTiming{
								{Name: "pre-tool-use", Applicable: true, Elapsed: 3 * time.Millisecond},
								{Name: "post-tool-use", Applicable: true, Elapsed: 6 * time.Millisecond},
								{Name: "graph build", Applicable: false, Elapsed: 0},
							},
						},
					},
					WarmMedian: 9 * time.Millisecond,
					WarmMin:    9 * time.Millisecond,
					WarmMax:    9 * time.Millisecond,
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

func TestFormatJSON(t *testing.T) {
	report := &BenchmarkReport{
		Timestamp: "2026-09-18T15:35:00Z",
		Mode:      "single",
		WarmRuns:  2,
		Repos: []*RepoAudit{
			{
				Dir:          "/repo",
				CoverageRate: 100.0,
			},
		},
	}

	var buf bytes.Buffer
	if err := FormatJSON(report, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed BenchmarkReport
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if parsed.Timestamp != report.Timestamp || parsed.Mode != report.Mode || len(parsed.Repos) != 1 {
		t.Errorf("parsed JSON does not match: %+v", parsed)
	}
}
