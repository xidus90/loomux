package benchcorpus

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

func TestMergeAudits(t *testing.T) {
	audit1 := &RepoAudit{
		RepoURL:        "https://github.com/django/django",
		Language:       "Python",
		Framework:      "Django",
		CoverageRate:   60.0,
		DetectedStacks: []string{"python"},
	}
	audit2 := &RepoAudit{
		RepoURL:        "https://github.com/gin-gonic/gin",
		Language:       "Go",
		Framework:      "Gin",
		CoverageRate:   100.0,
		DetectedStacks: []string{"go"},
	}
	audit3 := &RepoAudit{
		RepoURL:        "https://github.com/django/django",
		Language:       "Python",
		Framework:      "Django",
		CoverageRate:   80.0, // updated coverage
		DetectedStacks: []string{"python"},
	}

	// Initial merge into empty list
	list := MergeAudits(nil, audit1)
	if len(list) != 1 {
		t.Fatalf("expected 1 audit, got %d", len(list))
	}

	// Add second audit
	list = MergeAudits(list, audit2)
	if len(list) != 2 {
		t.Fatalf("expected 2 audits, got %d", len(list))
	}
	// Go should come before Python alphabetically by LanguageSlug
	if list[0].Language != "Go" || list[1].Language != "Python" {
		t.Errorf("expected deterministic sort (Go before Python), got %s, %s", list[0].Language, list[1].Language)
	}

	// Add third audit with same language but different framework
	audit4 := &RepoAudit{
		RepoURL:        "https://github.com/fastapi/fastapi",
		Language:       "Python",
		Framework:      "FastAPI",
		CoverageRate:   70.0,
		DetectedStacks: []string{"python"},
	}
	list = MergeAudits(list, audit4)
	// FastAPI should come before Django alphabetically
	if list[1].Framework != "Django" || list[2].Framework != "FastAPI" {
		t.Errorf("expected Django before FastAPI, got %s, %s", list[1].Framework, list[2].Framework)
	}

	// Add fourth audit with same language and framework but different repo slug
	audit5 := &RepoAudit{
		RepoURL:        "https://github.com/django/django-cms",
		Language:       "Python",
		Framework:      "Django",
		CoverageRate:   65.0,
		DetectedStacks: []string{"python"},
	}
	list = MergeAudits(list, audit5)
	if list[1].RepoURL != "https://github.com/django/django" || list[2].RepoURL != "https://github.com/django/django-cms" {
		t.Errorf("expected django before django-cms, got %s, %s", list[1].RepoURL, list[2].RepoURL)
	}

	// Update first audit
	list = MergeAudits(list, audit3)
	if len(list) != 4 {
		t.Fatalf("expected 4 audits after update, got %d", len(list))
	}
	if list[1].CoverageRate != 80.0 {
		t.Errorf("expected updated coverage 80.0, got %f", list[1].CoverageRate)
	}
}

func TestFormatMatrixMarkdown_Skipped(t *testing.T) {
	report := &BenchmarkReport{
		Skipped: []SkippedRepo{{
			RepoURL:   "https://github.com/membraneframework/membrane_core",
			Language:  "Elixir",
			Framework: "Membrane",
			Reason:    "error: invalid path 'a|b.md'",
		}},
	}
	for lang, heading := range map[string]string{"en": "## Skipped Repositories", "de": "## Übersprungene Repositories"} {
		var buf bytes.Buffer
		if err := FormatMatrixMarkdown(report, lang, &buf); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.Contains(out, heading) {
			t.Errorf("%s: missing %q in\n%s", lang, heading, out)
		}
		if !strings.Contains(out, "- `membraneframework_membrane_core` (Elixir / Membrane): error: invalid path 'a|b.md'") {
			t.Errorf("%s: missing skipped entry in\n%s", lang, out)
		}
	}
}

func TestFormatMatrixMarkdown(t *testing.T) {
	auditGo := &RepoAudit{
		Dir:            "/repos/loomux",
		CoverageRate:   100.0,
		DetectedStacks: []string{"go"},
		Timings: timings(benchreport.Summarize("", 20, []float64{18}),
			comp("pre-tool-use", 10, 7),
			comp("post-tool-use", 10, 11),
			comp("graph build", 80, 75)),
		ClaudeWarmMed: 90,
		Speedup:       5.0,
	}

	auditPy := &RepoAudit{
		RepoURL:        "https://github.com/django/django",
		Language:       "Python",
		Framework:      "Django",
		Tier:           "Sehr viel",
		CoverageRate:   66.7,
		DetectedStacks: []string{"python"},
		Timings: timings(benchreport.Summarize("", 25, []float64{20}),
			comp("pre-tool-use", 8, 6),
			comp("post-tool-use", 17, 14),
			notApplicable("graph build")),
	}

	report := &BenchmarkReport{
		Timestamp: "2026-09-18T18:00:00Z",
		Mode:      "corpus",
		WarmRuns:  3,
		Repos:     []*RepoAudit{auditGo, auditPy},
	}

	t.Run("German matrix formatting", func(t *testing.T) {
		var buf bytes.Buffer
		if err := FormatMatrixMarkdown(report, "de", &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "# Loomux Open-Source Benchmark-Matrix") {
			t.Errorf("missing German title: %s", out)
		}
		if !strings.Contains(out, "[Details](go/loomux.md)") {
			t.Errorf("missing Go details link: %s", out)
		}
		if !strings.Contains(out, "[Details](python/django_django.md)") {
			t.Errorf("missing Python details link: %s", out)
		}
		if !strings.Contains(out, "75.0 ms") {
			t.Errorf("missing graph build timing for Go")
		}
		if !strings.Contains(out, "n/a") {
			t.Errorf("missing n/a for Python graph build")
		}
		if !strings.Contains(out, "5.0x") {
			t.Errorf("missing speedup")
		}
	})

	t.Run("English matrix formatting", func(t *testing.T) {
		var buf bytes.Buffer
		if err := FormatMatrixMarkdown(report, "en", &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "# Loomux Open-Source Benchmark Matrix") {
			t.Errorf("missing English title: %s", out)
		}
		if !strings.Contains(out, "| Language | Framework | Repository |") {
			t.Errorf("missing English table header")
		}
		if !strings.Contains(out, "[Details](go/loomux.md)") {
			t.Errorf("missing Go details link in English")
		}
	})

	t.Run("Empty report", func(t *testing.T) {
		emptyReport := &BenchmarkReport{
			Timestamp: "2026-09-18T18:00:00Z",
		}
		var buf bytes.Buffer
		if err := FormatMatrixMarkdown(emptyReport, "en", &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "**Average Coverage Rate:** 0.0 %") {
			t.Errorf("expected 0.0%% average coverage for empty report, got: %s", buf.String())
		}
	})

	t.Run("FormatComponentTiming missing or empty warm", func(t *testing.T) {
		auditEmpty := &RepoAudit{Timings: timings(benchreport.Timing{}, benchreport.Timing{Name: "other-tool", ColdMS: 5})}
		// Component not measured at all
		if res := formatComponentTiming(auditEmpty, "non-existent"); res != "n/a" {
			t.Errorf("expected n/a for non-existent component, got %s", res)
		}
		// Component measured cold only
		if res := formatComponentTiming(auditEmpty, "other-tool"); res != "n/a" {
			t.Errorf("expected n/a for empty warm, got %s", res)
		}
	})
}
