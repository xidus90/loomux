package query

import (
	"errors"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/grep"
)

func TestGrepMissingGraph(t *testing.T) {
	root := t.TempDir()
	_, _, err := Grep(root, "Run", GrepOptions{})
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("want ErrNoGraph, got %v", err)
	}
}

func TestGrepAnswersAndReports(t *testing.T) {
	root := built(t)

	res, _, err := Grep(root, "Run", GrepOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalHits == 0 {
		t.Fatalf("want matches for Run, got none")
	}

	report := GrepReport(res)
	if !strings.Contains(report, "Run") || !strings.Contains(report, "matches") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestGrepPrivacyAndEdgeCases(t *testing.T) {
	root := built(t)

	// Keep rejecting files
	res, _, err := Grep(root, "Run", GrepOptions{
		Keep: func(p string) bool { return false },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.TotalHits != 0 || res.Unreadable == 0 {
		t.Fatalf("expected 0 hits and unreadable files when keep rejects all, got hits=%d unreadable=%d", res.TotalHits, res.Unreadable)
	}

	// Empty report
	emptyReport := GrepReport(res)
	if !strings.Contains(emptyReport, "no matches") {
		t.Errorf("expected no matches in report:\n%s", emptyReport)
	}

	// Truncated hits report
	truncReport := GrepReport(grep.Result{
		Pattern:       "test",
		TotalHits:     1,
		FilesSearched: 1,
		TruncatedHits: 5,
		Groups: []grep.Group{
			{
				Path: "a.go",
				Hits: []grep.Hit{{Line: 1, Text: "test"}},
			},
		},
	})
	if !strings.Contains(truncReport, "truncated") {
		t.Errorf("expected truncated notice in report:\n%s", truncReport)
	}
}
