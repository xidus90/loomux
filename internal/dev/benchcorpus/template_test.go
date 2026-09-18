package benchcorpus

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRepoSlug(t *testing.T) {
	tests := []struct {
		audit *RepoAudit
		want  string
	}{
		{
			audit: &RepoAudit{RepoURL: "https://github.com/django/django"},
			want:  "django_django",
		},
		{
			audit: &RepoAudit{RepoURL: "https://github.com/pallets/flask.git"},
			want:  "pallets_flask",
		},
		{
			audit: &RepoAudit{RepoURL: "http://singlepart"},
			want:  "singlepart",
		},
		{
			audit: &RepoAudit{Dir: "/path/to/my-project"},
			want:  "my-project",
		},
		{
			audit: &RepoAudit{Dir: "."},
			want:  "loomux",
		},
		{
			audit: &RepoAudit{Dir: ""},
			want:  "loomux",
		},
		{
			audit: &RepoAudit{Dir: "/"},
			want:  "loomux",
		},
	}

	for _, tt := range tests {
		got := RepoSlug(tt.audit)
		if got != tt.want {
			t.Errorf("RepoSlug(%+v) = %q, want %q", tt.audit, got, tt.want)
		}
	}
}

func TestLanguageSlug(t *testing.T) {
	tests := []struct {
		lang   string
		stacks []string
		want   string
	}{
		{"Python", nil, "python"},
		{"py", nil, "python"},
		{"Go", nil, "go"},
		{"golang", nil, "go"},
		{"JavaScript", nil, "javascript"},
		{"js", nil, "javascript"},
		{"node", nil, "javascript"},
		{"TypeScript", nil, "typescript"},
		{"ts", nil, "typescript"},
		{"Rust", nil, "rust"},
		{"rs", nil, "rust"},
		{"C#", nil, "csharp"},
		{"csharp", nil, "csharp"},
		{"dotnet", nil, "csharp"},
		{"C++", nil, "cpp"},
		{"cpp", nil, "cpp"},
		{"Java", nil, "java"},
		{"PHP", nil, "php"},
		{"Ruby", nil, "ruby"},
		{"rb", nil, "ruby"},
		{"C", nil, "c"},
		{"Elixir", nil, "elixir"},
		{"", []string{"go"}, "go"},
		{"", []string{"unknown", "rust"}, "rust"},
		{"", nil, "other"},
	}

	for _, tt := range tests {
		got := LanguageSlug(tt.lang, tt.stacks)
		if got != tt.want {
			t.Errorf("LanguageSlug(%q, %v) = %q, want %q", tt.lang, tt.stacks, got, tt.want)
		}
	}
}

func TestFormatDetailMarkdown(t *testing.T) {
	audit := &RepoAudit{
		RepoURL:        "https://github.com/django/django",
		Language:       "Python",
		Framework:      "Django",
		Tier:           "Sehr viel",
		CommitSHA:      "65c1568",
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
			Total: 30 * time.Millisecond,
			Components: []ComponentTiming{
				{Name: "pre-tool-use", Applicable: true, Elapsed: 10 * time.Millisecond},
				{Name: "post-tool-use", Applicable: true, Elapsed: 20 * time.Millisecond},
				{Name: "graph build", Applicable: false, Elapsed: 0},
				{Name: "custom", Applicable: false, Elapsed: 0},
			},
		},
		Warm: []TimingRun{
			{
				Total: 25 * time.Millisecond,
				Components: []ComponentTiming{
					{Name: "pre-tool-use", Applicable: true, Elapsed: 8 * time.Millisecond},
					{Name: "post-tool-use", Applicable: true, Elapsed: 17 * time.Millisecond},
					{Name: "graph build", Applicable: false, Elapsed: 0},
					{Name: "custom", Applicable: false, Elapsed: 0},
				},
			},
		},
		WarmMedian:    25 * time.Millisecond,
		WarmMin:       25 * time.Millisecond,
		WarmMax:       25 * time.Millisecond,
		ClaudeWarmMed: 100 * time.Millisecond,
		Speedup:       4.0,
	}

	t.Run("German format", func(t *testing.T) {
		var buf bytes.Buffer
		if err := FormatDetailMarkdown(audit, "de", &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "# Benchmark & Lücken-Audit: [django/django](https://github.com/django/django)") {
			t.Errorf("missing German title: %s", out)
		}
		if !strings.Contains(out, "[← Zurück zur Gesamt-Matrix](../matrix.md)") {
			t.Errorf("missing back link: %s", out)
		}
		if !strings.Contains(out, "## 1. Performance & Latenzen") {
			t.Errorf("missing German latencies header")
		}
		if !strings.Contains(out, "## 2. Test- & Lücken-Audit (Gap Analysis)") {
			t.Errorf("missing German gap header")
		}
		if !strings.Contains(out, "## 3. Erkannte Stacks & Lanes") {
			t.Errorf("missing German stacks header")
		}
		if !strings.Contains(out, "**Baseline Claude Hook:** 100.0 ms (Speedup: 4.0x)") {
			t.Errorf("missing speedup: %s", out)
		}
		if !strings.Contains(out, "✅ Aktiv") || !strings.Contains(out, "❌ Nicht im PATH") || !strings.Contains(out, "⚠️ Fehlt in Loomux") {
			t.Errorf("missing check status indicators")
		}
	})

	t.Run("English format", func(t *testing.T) {
		var buf bytes.Buffer
		if err := FormatDetailMarkdown(audit, "en", &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()

		if !strings.Contains(out, "# Benchmark & Gap Audit: [django/django](https://github.com/django/django)") {
			t.Errorf("missing English title: %s", out)
		}
		if !strings.Contains(out, "[← Back to Matrix](../matrix.md)") {
			t.Errorf("missing English back link")
		}
		if !strings.Contains(out, "## 1. Performance & Latencies") {
			t.Errorf("missing English latencies header")
		}
		if !strings.Contains(out, "## 2. Check & Gap Audit (Gap Analysis)") {
			t.Errorf("missing English gap header")
		}
		if !strings.Contains(out, "## 3. Detected Stacks & Lanes") {
			t.Errorf("missing English stacks header")
		}
		if !strings.Contains(out, "✅ Active") || !strings.Contains(out, "❌ Missing from PATH") || !strings.Contains(out, "⚠️ Missing in Loomux") {
			t.Errorf("missing English check status indicators")
		}
	})

	t.Run("Local repo without URL", func(t *testing.T) {
		localAudit := &RepoAudit{
			Dir:          "/repos/loomux",
			CoverageRate: 100.0,
			Cold: TimingRun{
				Total: 10 * time.Millisecond,
				Components: []ComponentTiming{
					{Name: "pre-tool-use", Applicable: true, Elapsed: 10 * time.Millisecond},
				},
			},
			Warm: []TimingRun{
				{
					Total: 8 * time.Millisecond,
					Components: []ComponentTiming{
						{Name: "pre-tool-use", Applicable: true, Elapsed: 8 * time.Millisecond},
					},
				},
			},
			WarmMedian: 8 * time.Millisecond,
			WarmMin:    8 * time.Millisecond,
			WarmMax:    8 * time.Millisecond,
		}
		var buf bytes.Buffer
		if err := FormatDetailMarkdown(localAudit, "de", &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "# Benchmark & Lücken-Audit: loomux") {
			t.Errorf("expected local title, got: %s", out)
		}
	})

	t.Run("Repo with single part URL", func(t *testing.T) {
		singleURLAudit := &RepoAudit{
			RepoURL:      "http://singlename",
			CoverageRate: 100.0,
			Cold: TimingRun{
				Total: 10 * time.Millisecond,
				Components: []ComponentTiming{
					{Name: "pre-tool-use", Applicable: true, Elapsed: 10 * time.Millisecond},
				},
			},
			Warm: []TimingRun{
				{
					Total: 8 * time.Millisecond,
					Components: []ComponentTiming{
						{Name: "pre-tool-use", Applicable: true, Elapsed: 8 * time.Millisecond},
					},
				},
			},
			WarmMedian: 8 * time.Millisecond,
			WarmMin:    8 * time.Millisecond,
			WarmMax:    8 * time.Millisecond,
		}
		var buf bytes.Buffer
		if err := FormatDetailMarkdown(singleURLAudit, "en", &buf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "[singlename](http://singlename)") {
			t.Errorf("expected single part URL title, got: %s", out)
		}
	})
}
