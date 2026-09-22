package query

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/store"
)

func TestStatsMissingGraph(t *testing.T) {
	root := t.TempDir()
	_, err := GraphStats(root)
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("want ErrNoGraph, got %v", err)
	}
}

func TestStatsAnswersAndReports(t *testing.T) {
	root := built(t)

	s, err := GraphStats(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Files == 0 || s.Symbols == 0 || s.Edges == 0 || s.WiringBytes == 0 {
		t.Fatalf("want non-zero stats, got %v", s)
	}
	if len(s.Languages) == 0 || s.Languages[0] != "go" {
		t.Errorf("want [go] language, got %v", s.Languages)
	}
	if len(s.Relations) == 0 {
		t.Errorf("want relation breakdown, got %v", s.Relations)
	}

	report := StatsReport(s)
	if !strings.Contains(report, "Code Graph Stats") || !strings.Contains(report, "Files:") || !strings.Contains(report, "Relations:") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestStatsErrors(t *testing.T) {
	// 1. Corrupt graph
	root := repo(t, sample())
	if err := os.MkdirAll(store.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(root), []byte("{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := GraphStats(root)
	if err == nil || errors.Is(err, ErrNoGraph) {
		t.Fatalf("want read error for corrupt graph, got %v", err)
	}

	// 2. Stat error other than ErrNotExist
	_, err = GraphStats("\x00invalid")
	if err == nil || errors.Is(err, ErrNoGraph) {
		t.Fatalf("want invalid path stat error, got %v", err)
	}
}
