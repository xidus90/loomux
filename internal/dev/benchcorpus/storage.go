package benchcorpus

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// StorageOps encapsulates filesystem operations to permit clean dependency injection and testing.
type StorageOps struct {
	WriteFile func(path string, data []byte, perm os.FileMode) error
	MkdirAll  func(path string, perm os.FileMode) error
	ReadFile  func(path string) ([]byte, error)
}

// DefaultStorageOps provides live filesystem operations using the standard os package.
func DefaultStorageOps() StorageOps {
	return StorageOps{
		WriteFile: os.WriteFile,
		MkdirAll:  os.MkdirAll,
		ReadFile:  os.ReadFile,
	}
}

// SaveReport persists repository benchmark detail pages into language subdirectories
// (docs/{en,de}/benchmarks/<language>/<repo_slug>.md) and updates the central matrix
// (docs/{en,de}/benchmarks/matrix.md).
func SaveReport(report *BenchmarkReport, docsDir string, ops StorageOps) error {
	def := DefaultStorageOps()
	if ops.WriteFile == nil {
		ops.WriteFile = def.WriteFile
	}
	if ops.MkdirAll == nil {
		ops.MkdirAll = def.MkdirAll
	}
	if ops.ReadFile == nil {
		ops.ReadFile = def.ReadFile
	}

	if docsDir == "" {
		docsDir = "docs"
	}

	dataPath := filepath.Join(docsDir, "benchmarks.json")
	var existing []*RepoAudit

	if raw, err := ops.ReadFile(dataPath); err == nil {
		_ = json.Unmarshal(raw, &existing)
	}

	for _, incoming := range report.Repos {
		existing = MergeAudits(existing, incoming)
	}

	jsonBytes, _ := json.MarshalIndent(existing, "", "  ")
	if err := ops.WriteFile(dataPath, jsonBytes, 0644); err != nil {
		return err
	}

	// Skipped repositories live beside the audits rather than inside them, so
	// benchmarks.json keeps its shape; a run that does not reach a skipped
	// repository must not erase it from the matrix.
	skippedPath := filepath.Join(docsDir, "benchmarks-skipped.json")
	var skipped []SkippedRepo
	if raw, err := ops.ReadFile(skippedPath); err == nil {
		_ = json.Unmarshal(raw, &skipped)
	}
	skipped = mergeSkipped(skipped, report)
	skippedBytes, _ := json.MarshalIndent(skipped, "", "  ")
	if err := ops.WriteFile(skippedPath, skippedBytes, 0644); err != nil {
		return err
	}

	// 1. Write detail pages for all repos in current report
	for _, repo := range report.Repos {
		langSlug := LanguageSlug(repo.Language, repo.DetectedStacks)
		repoSlug := RepoSlug(repo)

		for _, lang := range []string{"en", "de"} {
			detailDir := filepath.Join(docsDir, lang, "benchmarks", langSlug)
			if err := ops.MkdirAll(detailDir, 0755); err != nil {
				return err
			}

			detailFile := filepath.Join(detailDir, repoSlug+".md")
			var buf bytes.Buffer
			_ = FormatDetailMarkdown(repo, lang, &buf)
			if err := ops.WriteFile(detailFile, buf.Bytes(), 0644); err != nil {
				return err
			}
		}
	}

	// 2. Write central matrix incorporating all merged audits
	matrixReport := *report
	matrixReport.Repos = existing
	matrixReport.Skipped = skipped
	if matrixReport.Timestamp == "" {
		matrixReport.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	for _, lang := range []string{"en", "de"} {
		matrixDir := filepath.Join(docsDir, lang, "benchmarks")
		if err := ops.MkdirAll(matrixDir, 0755); err != nil {
			return err
		}

		matrixFile := filepath.Join(matrixDir, "matrix.md")
		var buf bytes.Buffer
		_ = FormatMatrixMarkdown(&matrixReport, lang, &buf)
		if err := ops.WriteFile(matrixFile, buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

// mergeSkipped drops every stored entry the report has now benchmarked, then
// upserts the report's own skips by URL, sorted by language and URL.
func mergeSkipped(stored []SkippedRepo, report *BenchmarkReport) []SkippedRepo {
	benchmarked := make(map[string]bool, len(report.Repos))
	for _, a := range report.Repos {
		benchmarked[RepoSlug(a)] = true
	}
	byURL := make(map[string]SkippedRepo)
	for _, s := range stored {
		if !benchmarked[RepoSlug(&RepoAudit{RepoURL: s.RepoURL})] {
			byURL[s.RepoURL] = s
		}
	}
	for _, s := range report.Skipped {
		byURL[s.RepoURL] = s
	}
	merged := make([]SkippedRepo, 0, len(byURL))
	for _, s := range byURL {
		merged = append(merged, s)
	}
	slices.SortFunc(merged, func(a, b SkippedRepo) int {
		if a.Language != b.Language {
			return strings.Compare(a.Language, b.Language)
		}
		return strings.Compare(a.RepoURL, b.RepoURL)
	})
	return merged
}
