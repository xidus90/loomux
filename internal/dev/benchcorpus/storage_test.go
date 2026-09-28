package benchcorpus

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/dev/benchreport"
)

type mockStorage struct {
	files map[string][]byte
	dirs  map[string]bool

	failMkdir func(path string) error
	failWrite func(path string) error
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *mockStorage) toOps() StorageOps {
	return StorageOps{
		MkdirAll: func(path string, perm os.FileMode) error {
			norm := filepath.ToSlash(path)
			if m.failMkdir != nil {
				if err := m.failMkdir(norm); err != nil {
					return err
				}
			}
			m.dirs[norm] = true
			return nil
		},
		WriteFile: func(path string, data []byte, perm os.FileMode) error {
			norm := filepath.ToSlash(path)
			if m.failWrite != nil {
				if err := m.failWrite(norm); err != nil {
					return err
				}
			}
			m.files[norm] = data
			return nil
		},
		ReadFile: func(path string) ([]byte, error) {
			norm := filepath.ToSlash(path)
			data, ok := m.files[norm]
			if !ok {
				return nil, os.ErrNotExist
			}
			return data, nil
		},
	}
}

func sampleAudits() []*RepoAudit {
	return []*RepoAudit{
		{
			Dir:            "/repos/loomux",
			Language:       "Go",
			CoverageRate:   100.0,
			DetectedStacks: []string{"go"},
			Timings: timings(benchreport.Summarize("", 20, []float64{18}),
				comp("pre-tool-use", 10, 7),
				comp("post-tool-use", 10, 11),
				comp("graph build", 80, 75)),
			ClaudeWarmMed: 90,
			Speedup:       5.0,
		},
		{
			RepoURL:        "https://github.com/django/django",
			Language:       "Python",
			Framework:      "Django",
			Tier:           "Tier 1",
			CoverageRate:   75.0,
			DetectedStacks: []string{"python"},
			Timings: timings(benchreport.Summarize("", 30, []float64{25}),
				comp("pre-tool-use", 12, 10),
				comp("post-tool-use", 18, 15),
				notApplicable("graph build")),
		},
	}
}

func TestSaveReport_SuccessAndMerge(t *testing.T) {
	t.Chdir(t.TempDir()) // see TestSaveReport_DefaultOps
	mock := newMockStorage()
	ops := mock.toOps()

	report := &BenchmarkReport{
		Timestamp: "2026-09-18T18:00:00Z",
		Mode:      "corpus",
		WarmRuns:  3,
		Repos:     sampleAudits(),
	}

	// 1. Initial save with empty docsDir (defaults to "docs")
	if err := SaveReport(report, "", ops); err != nil {
		t.Fatalf("SaveReport failed: %v", err)
	}

	// Check that language folders and detail pages exist
	expectedFiles := []string{
		"docs/en/benchmarks/go/loomux.md",
		"docs/de/benchmarks/go/loomux.md",
		"docs/en/benchmarks/python/django_django.md",
		"docs/de/benchmarks/python/django_django.md",
		"docs/en/benchmarks/matrix.md",
		"docs/de/benchmarks/matrix.md",
		"docs/benchmarks.json",
	}

	for _, ef := range expectedFiles {
		if _, ok := mock.files[ef]; !ok {
			t.Errorf("missing expected file: %s", ef)
		}
	}

	// Check content of German detail page
	deLoomux := string(mock.files["docs/de/benchmarks/go/loomux.md"])
	if !strings.Contains(deLoomux, "# Benchmark & Lücken-Audit: loomux") {
		t.Errorf("unexpected content in de loomux detail page: %s", deLoomux)
	}
	if !strings.Contains(deLoomux, "[← Zurück zur Gesamt-Matrix](../matrix.md)") {
		t.Errorf("missing back link in de loomux detail page")
	}

	// Check content of English matrix
	enMatrix := string(mock.files["docs/en/benchmarks/matrix.md"])
	if !strings.Contains(enMatrix, "[Details](go/loomux.md)") || !strings.Contains(enMatrix, "[Details](python/django_django.md)") {
		t.Errorf("missing detail links in en matrix: %s", enMatrix)
	}

	// 2. Incremental save: single repo update
	updatedDjango := &RepoAudit{
		RepoURL:        "https://github.com/django/django",
		Language:       "Python",
		Framework:      "Django",
		Tier:           "Tier 1",
		CoverageRate:   90.0, // updated
		DetectedStacks: []string{"python"},
		Timings:        timings(benchreport.Timing{ColdMS: 25, MedianMS: 20}),
	}
	singleReport := &BenchmarkReport{
		Timestamp: "2026-09-18T19:00:00Z",
		Mode:      "single",
		WarmRuns:  1,
		Repos:     []*RepoAudit{updatedDjango},
	}

	if err := SaveReport(singleReport, "docs", ops); err != nil {
		t.Fatalf("incremental SaveReport failed: %v", err)
	}

	// Verify that both repos are still present in matrix.md
	deMatrixUpdated := string(mock.files["docs/de/benchmarks/matrix.md"])
	if !strings.Contains(deMatrixUpdated, "90.0 %") {
		t.Errorf("expected updated 90.0%% coverage in matrix: %s", deMatrixUpdated)
	}
	if !strings.Contains(deMatrixUpdated, "[Details](go/loomux.md)") {
		t.Errorf("expected loomux still in matrix after incremental save")
	}
}

func TestSaveReport_SingleRepoKeepsCorpusMetadata(t *testing.T) {
	t.Chdir(t.TempDir()) // see TestSaveReport_DefaultOps
	tempDir := t.TempDir()
	corpus := &RepoAudit{
		RepoURL:        "https://github.com/gin-gonic/gin",
		Dir:            ".cache/benchcorpus/gin-gonic_gin",
		Language:       "Go",
		Framework:      "Gin",
		Tier:           "Sehr viel",
		CommitSHA:      "5c6a15f",
		DetectedStacks: []string{"go"},
		Timings:        timings(benchreport.Timing{MedianMS: 21}),
	}
	if err := SaveReport(&BenchmarkReport{Repos: []*RepoAudit{corpus}}, tempDir, StorageOps{}); err != nil {
		t.Fatalf("corpus save: %v", err)
	}

	single := &RepoAudit{
		Dir:            ".cache/benchcorpus/gin-gonic_gin",
		DetectedStacks: []string{"go"},
		Timings:        timings(benchreport.Timing{MedianMS: 20}),
	}
	if err := SaveReport(&BenchmarkReport{Repos: []*RepoAudit{single}}, tempDir, StorageOps{}); err != nil {
		t.Fatalf("single save: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(tempDir, "benchmarks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored []*RepoAudit
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored audit, got %d", len(stored))
	}
	got := stored[0]
	if got.RepoURL != corpus.RepoURL || got.Language != "Go" || got.Framework != "Gin" || got.Tier != "Sehr viel" || got.CommitSHA != "5c6a15f" {
		t.Errorf("corpus metadata lost: %+v", got)
	}
	if total, _ := got.Timing(TotalTiming); total.MedianMS != 20 {
		t.Errorf("expected new measurement, got %v", total.MedianMS)
	}

	detail, err := os.ReadFile(filepath.Join(tempDir, "en", "benchmarks", "go", "gin-gonic_gin.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(detail), "**Framework:** Gin") {
		t.Errorf("detail page lost metadata:\n%s", detail)
	}
}

func TestSaveReport_SkippedPersistsAcrossRuns(t *testing.T) {
	t.Chdir(t.TempDir()) // see TestSaveReport_DefaultOps
	tempDir := t.TempDir()
	membrane := "https://github.com/membraneframework/membrane_core"
	matrix := func() string {
		raw, err := os.ReadFile(filepath.Join(tempDir, "en", "benchmarks", "matrix.md"))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}

	corpus := &BenchmarkReport{
		Repos:   sampleAudits()[:1],
		Skipped: []SkippedRepo{{RepoURL: membrane, Language: "Elixir", Framework: "Membrane", Reason: "fatal: unable to checkout working tree"}},
	}
	if err := SaveReport(corpus, tempDir, StorageOps{}); err != nil {
		t.Fatal(err)
	}

	// A later run that does not touch the skipped repository keeps it listed.
	if err := SaveReport(&BenchmarkReport{Repos: sampleAudits()[:1]}, tempDir, StorageOps{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(matrix(), "`membraneframework_membrane_core` (Elixir / Membrane)") {
		t.Fatalf("skipped repository lost after a later run:\n%s", matrix())
	}

	// Once it is benchmarked, it is no longer skipped.
	fixed := &BenchmarkReport{Repos: []*RepoAudit{{RepoURL: membrane, Language: "Elixir", DetectedStacks: []string{"elixir"}}}}
	if err := SaveReport(fixed, tempDir, StorageOps{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(matrix(), "Skipped Repositories") {
		t.Fatalf("benchmarked repository still listed as skipped:\n%s", matrix())
	}
}

func TestSaveReport_CorruptedSkippedFile(t *testing.T) {
	t.Chdir(t.TempDir()) // see TestSaveReport_DefaultOps
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "benchmarks-skipped.json"), []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SaveReport(&BenchmarkReport{Repos: sampleAudits()[:1]}, tempDir, StorageOps{}); err != nil {
		t.Fatalf("a damaged skipped file must read as empty, got %v", err)
	}
}

func TestSaveReport_DefaultOps(t *testing.T) {
	// A mutant that forces docsDir to "docs" or swaps a mock for the real
	// filesystem writes relative to the working directory; every SaveReport
	// test runs in a temporary one to keep that out of the package.
	t.Chdir(t.TempDir())
	tempDir := t.TempDir()
	report := &BenchmarkReport{
		Repos: sampleAudits()[:1],
	}

	// Pass empty StorageOps to exercise DefaultStorageOps fallback
	if err := SaveReport(report, tempDir, StorageOps{}); err != nil {
		t.Fatalf("SaveReport with default ops failed: %v", err)
	}

	detailPath := filepath.Join(tempDir, "de", "benchmarks", "go", "loomux.md")
	if _, err := os.Stat(detailPath); err != nil {
		t.Errorf("expected detail file to exist on disk: %v", err)
	}

	matrixPath := filepath.Join(tempDir, "en", "benchmarks", "matrix.md")
	if _, err := os.Stat(matrixPath); err != nil {
		t.Errorf("expected matrix file to exist on disk: %v", err)
	}
}

func TestSaveReport_Errors(t *testing.T) {
	t.Chdir(t.TempDir()) // see TestSaveReport_DefaultOps
	report := &BenchmarkReport{
		Repos: sampleAudits(),
	}

	t.Run("write skipped fail", func(t *testing.T) {
		mock := newMockStorage()
		mock.failWrite = func(p string) error {
			if strings.HasSuffix(p, "benchmarks-skipped.json") {
				return errors.New("mock write skipped error")
			}
			return nil
		}
		err := SaveReport(report, "docs", mock.toOps())
		if err == nil || !strings.Contains(err.Error(), "mock write skipped error") {
			t.Errorf("expected write error, got %v", err)
		}
	})
	t.Run("mkdir detail fail", func(t *testing.T) {
		mock := newMockStorage()
		mock.failMkdir = func(p string) error {
			if strings.HasSuffix(p, "/go") {
				return errors.New("mock mkdir detail error")
			}
			return nil
		}
		err := SaveReport(report, "docs", mock.toOps())
		if err == nil || !strings.Contains(err.Error(), "mock mkdir detail error") {
			t.Errorf("expected mkdir error, got %v", err)
		}
	})

	t.Run("write detail fail", func(t *testing.T) {
		mock := newMockStorage()
		mock.failWrite = func(p string) error {
			if strings.HasSuffix(p, "loomux.md") {
				return errors.New("mock write detail error")
			}
			return nil
		}
		err := SaveReport(report, "docs", mock.toOps())
		if err == nil || !strings.Contains(err.Error(), "mock write detail error") {
			t.Errorf("expected write error, got %v", err)
		}
	})

	t.Run("mkdir matrix fail", func(t *testing.T) {
		mock := newMockStorage()
		mock.failMkdir = func(p string) error {
			if strings.HasSuffix(p, "/benchmarks") {
				return errors.New("mock mkdir matrix error")
			}
			return nil
		}
		err := SaveReport(report, "docs", mock.toOps())
		if err == nil || !strings.Contains(err.Error(), "mock mkdir matrix error") {
			t.Errorf("expected mkdir error, got %v", err)
		}
	})

	t.Run("write matrix fail", func(t *testing.T) {
		mock := newMockStorage()
		mock.failWrite = func(p string) error {
			if strings.HasSuffix(p, "matrix.md") {
				return errors.New("mock write matrix error")
			}
			return nil
		}
		err := SaveReport(report, "docs", mock.toOps())
		if err == nil || !strings.Contains(err.Error(), "mock write matrix error") {
			t.Errorf("expected write error, got %v", err)
		}
	})

	t.Run("write benchmarks.json fail", func(t *testing.T) {
		mock := newMockStorage()
		mock.failWrite = func(p string) error {
			if strings.HasSuffix(p, "benchmarks.json") {
				return errors.New("mock write data error")
			}
			return nil
		}
		err := SaveReport(report, "docs", mock.toOps())
		if err == nil || !strings.Contains(err.Error(), "mock write data error") {
			t.Errorf("expected write error, got %v", err)
		}
	})
}

func TestSaveReport_CorruptedDataFile(t *testing.T) {
	t.Chdir(t.TempDir()) // see TestSaveReport_DefaultOps
	mock := newMockStorage()
	mock.files["docs/benchmarks.json"] = []byte("corrupted json {")

	report := &BenchmarkReport{
		Repos: sampleAudits()[:1],
	}

	if err := SaveReport(report, "docs", mock.toOps()); err != nil {
		t.Fatalf("expected graceful recovery on corrupted json, got: %v", err)
	}

	var parsed []*RepoAudit
	if err := json.Unmarshal(mock.files["docs/benchmarks.json"], &parsed); err != nil {
		t.Fatalf("expected valid json written after recovery: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("expected 1 audit saved, got %d", len(parsed))
	}
}

func TestMergeSkipped_SortsByLanguageThenURL(t *testing.T) {
	report := &BenchmarkReport{Skipped: []SkippedRepo{
		{RepoURL: "https://github.com/b/z", Language: "Zig"},
		{RepoURL: "https://github.com/b/y", Language: "Elixir"},
		{RepoURL: "https://github.com/a/x", Language: "Elixir"},
	}}
	got := mergeSkipped(nil, report)
	want := []string{"https://github.com/a/x", "https://github.com/b/y", "https://github.com/b/z"}
	for i, s := range got {
		if s.RepoURL != want[i] {
			t.Fatalf("mergeSkipped order = %+v, want %v", got, want)
		}
	}
}

func TestSaveReportStampsTheMatrixOnlyWithoutATimestamp(t *testing.T) {
	t.Chdir(t.TempDir()) // see TestSaveReport_DefaultOps
	stamp := func(report *BenchmarkReport) string {
		mock := newMockStorage()
		if err := SaveReport(report, "out", mock.toOps()); err != nil {
			t.Fatal(err)
		}
		matrix := string(mock.files["out/en/benchmarks/matrix.md"])
		_, rest, ok := strings.Cut(matrix, "- **Last Updated:** ")
		if !ok {
			t.Fatalf("no timestamp line:\n%s", matrix)
		}
		line, _, _ := strings.Cut(rest, "\n")
		return line
	}
	if got := stamp(&BenchmarkReport{Timestamp: "2026-01-02T03:04:05Z"}); got != "2026-01-02T03:04:05Z" {
		t.Errorf("given timestamp became %q", got)
	}
	if got := stamp(&BenchmarkReport{}); got == "" {
		t.Error("a report without a timestamp got none")
	} else if _, err := time.Parse(time.RFC3339, got); err != nil {
		t.Errorf("stamped %q: %v", got, err)
	}
}

func TestMergeSkippedSortsByLanguageBeforeURL(t *testing.T) {
	// URL order contradicts language order here.
	report := &BenchmarkReport{Skipped: []SkippedRepo{
		{RepoURL: "https://github.com/a/w", Language: "Zig"},
		{RepoURL: "https://github.com/b/y", Language: "Elixir"},
		{RepoURL: "https://github.com/b/x", Language: "Elixir"},
	}}
	var got []string
	for _, s := range mergeSkipped(nil, report) {
		got = append(got, s.RepoURL)
	}
	want := []string{"https://github.com/b/x", "https://github.com/b/y", "https://github.com/a/w"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("mergeSkipped order = %v, want %v", got, want)
	}
}
