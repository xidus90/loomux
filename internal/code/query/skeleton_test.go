package query

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/skeleton"
)

func TestSkeletonMissingGraph(t *testing.T) {
	root := t.TempDir()
	_, _, err := Skeleton(root, "lib.go", SkeletonOptions{})
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("want ErrNoGraph, got %v", err)
	}
}

func TestSkeletonAnswersAndReports(t *testing.T) {
	root := built(t)

	ans, _, err := Skeleton(root, "lib.go", SkeletonOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ans.Path != "lib/lib.go" {
		t.Fatalf("want lib/lib.go, got %s", ans.Path)
	}
	if len(ans.Entries) == 0 {
		t.Fatalf("want entries in lib.go, got none")
	}

	report := SkeletonReport(ans)
	if !strings.Contains(report, "lib/lib.go") || !strings.Contains(report, "Run") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestSkeletonPrivacyAndEdgeCases(t *testing.T) {
	root := built(t)

	// Keep rejecting file
	_, _, err := Skeleton(root, "lib.go", SkeletonOptions{
		Keep: func(p string) bool { return false },
	})
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("want file not found for rejected path, got %v", err)
	}

	// Missing file error from skeleton.Extract
	_, _, err = Skeleton(root, "missing.go", SkeletonOptions{})
	if err == nil || !strings.Contains(err.Error(), "file not found") {
		t.Fatalf("want file not found for missing.go, got %v", err)
	}

	// Empty report
	if SkeletonReport(SkeletonAnswer{}) != "no file\n" {
		t.Error("expected no file message")
	}

	emptyEntriesReport := SkeletonReport(SkeletonAnswer{Path: "empty.go"})
	if !strings.Contains(emptyEntriesReport, "no symbols defined") {
		t.Errorf("expected no symbols defined in report:\n%s", emptyEntriesReport)
	}

	// Entry with empty signature falls back to Name
	sigFallbackReport := SkeletonReport(SkeletonAnswer{
		Path: "test.go",
		Entries: []skeleton.Entry{
			{Name: "MyFunc", Kind: "func", Span: "L1-L5"},
		},
	})
	if !strings.Contains(sigFallbackReport, "MyFunc") {
		t.Errorf("expected MyFunc in report:\n%s", sigFallbackReport)
	}
}

func TestSkeletonRefreshesOnDrift(t *testing.T) {
	root := built(t)

	// Modify lib.go to create drift
	libPath := filepath.Join(root, "lib", "lib.go")
	if err := os.WriteFile(libPath, []byte("package lib\n\nfunc NewFunc() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ans, notes, err := Skeleton(root, "lib.go", SkeletonOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "rebuilding") {
		t.Fatalf("expected rebuild note on drift, got %v", notes)
	}
	if len(ans.Entries) == 0 || ans.Entries[0].Name != "NewFunc" {
		t.Fatalf("expected NewFunc from rebuilt graph, got %v", ans.Entries)
	}
}
