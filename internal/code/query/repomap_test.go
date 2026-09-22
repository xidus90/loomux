package query

import (
	"errors"
	"strings"
	"testing"
)

func TestMapMissingGraph(t *testing.T) {
	root := t.TempDir()
	_, _, err := Map(root, MapOptions{})
	if !errors.Is(err, ErrNoGraph) {
		t.Fatalf("want ErrNoGraph, got %v", err)
	}
}

func TestMapAnswersAndReports(t *testing.T) {
	root := built(t)

	m, _, err := Map(root, MapOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Totals.Files == 0 {
		t.Fatalf("want files in repo map, got 0")
	}

	report := MapReport(m)
	if !strings.Contains(report, "Repo Map") || !strings.Contains(report, "Directories") {
		t.Fatalf("unexpected report:\n%s", report)
	}
}

func TestMapPrivacyFilter(t *testing.T) {
	root := built(t)

	// Filter out lib/
	m, _, err := Map(root, MapOptions{
		Keep: func(p string) bool { return !strings.HasPrefix(p, "lib/") },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, d := range m.Dirs {
		if strings.HasPrefix(d.Path, "lib") {
			t.Errorf("rejected path found in map dirs: %s", d.Path)
		}
	}
}
