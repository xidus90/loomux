package skeleton_test

import (
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/skeleton"
)

func testGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "pkg/reach.go", Path: "pkg/reach.go", Name: "reach.go", Kind: model.KindFile, Span: "L1-L100"},
			{ID: "pkg/reach.go#Helper", Path: "pkg/reach.go", Name: "Helper", Kind: "func", Span: "invalid", Signature: "func Helper()"},
			{ID: "pkg/reach.go#Reach", Path: "pkg/reach.go", Name: "Reach", Kind: "func", Span: "L20-L40", Signature: "func Reach()"},
			{ID: "pkg/reach.go#Index", Path: "pkg/reach.go", Name: "Index", Kind: "struct", Span: "L5-L15", Signature: "type Index struct"},
			{ID: "pkg/reach.go#Config", Path: "pkg/reach.go", Name: "Config", Kind: "struct", Span: "L5-L10", Signature: "type Config struct"},
			{ID: "pkg/reach.go#AlphaHelper", Path: "pkg/reach.go", Name: "AlphaHelper", Kind: "func", Span: "invalid", Signature: "func AlphaHelper()"},
			{ID: "a/test.go", Path: "a/test.go", Name: "test.go", Kind: model.KindFile, Span: "L1-L10"},
			{ID: "b/test.go", Path: "b/test.go", Name: "test.go", Kind: model.KindFile, Span: "L1-L10"},
		},
	}
}

func TestExtractExactAndBasename(t *testing.T) {
	g := testGraph()

	// 1. Exact relative path
	filePath, entries, err := skeleton.Extract(g, "pkg/reach.go", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filePath != "pkg/reach.go" {
		t.Fatalf("want pkg/reach.go, got %s", filePath)
	}
	if len(entries) != 5 {
		t.Fatalf("want 5 symbol entries (excluding KindFile), got %d: %v", len(entries), entries)
	}

	// Sorted: Config (L5) before Index (L5, alphabetical) before Reach (L20) before AlphaHelper (invalid) before Helper (invalid)
	if entries[0].Name != "Config" || entries[1].Name != "Index" || entries[2].Name != "Reach" || entries[3].Name != "AlphaHelper" || entries[4].Name != "Helper" {
		t.Errorf("unexpected sort order: %v", entries)
	}

	// 2. Basename query
	filePath, entries, err = skeleton.Extract(g, "reach.go", nil)
	if err != nil || filePath != "pkg/reach.go" || len(entries) != 5 {
		t.Fatalf("basename query failed: path=%s len=%d err=%v", filePath, len(entries), err)
	}

	// Backslash path normalization
	filePath, entries, err = skeleton.Extract(g, `pkg\reach.go`, nil)
	if err != nil || filePath != "pkg/reach.go" || len(entries) != 5 {
		t.Fatalf("backslash path failed: path=%s len=%d err=%v", filePath, len(entries), err)
	}
}

func TestExtractAmbiguousAndNotFound(t *testing.T) {
	g := testGraph()

	// Ambiguous basename
	_, _, err := skeleton.Extract(g, "test.go", nil)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("want ambiguous error for test.go, got %v", err)
	}

	// Not found
	_, _, err = skeleton.Extract(g, "missing.go", nil)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not found error for missing.go, got %v", err)
	}
}

func TestExtractTreatsAHiddenFileAsUnknown(t *testing.T) {
	g := testGraph()
	hideA := func(p string) bool { return p != "a/test.go" }

	// The hidden twin neither makes the basename ambiguous nor shows up in an error.
	filePath, _, err := skeleton.Extract(g, "test.go", hideA)
	if err != nil || filePath != "b/test.go" {
		t.Fatalf("Extract = %q, %v; want the readable b/test.go", filePath, err)
	}

	_, _, err = skeleton.Extract(g, "a/test.go", hideA)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("want not found for the hidden exact path, got %v", err)
	}

	hideBoth := func(p string) bool { return p == "pkg/reach.go" }
	_, _, err = skeleton.Extract(g, "test.go", hideBoth)
	if err == nil || strings.Contains(err.Error(), "a/test.go") || strings.Contains(err.Error(), "b/test.go") {
		t.Fatalf("want not found naming no hidden path, got %v", err)
	}
}

func TestExtractEdgeCases(t *testing.T) {
	g := testGraph()

	// Nil graph
	_, _, err := skeleton.Extract(nil, "pkg/reach.go", nil)
	if err == nil {
		t.Fatal("want error for nil graph, got nil")
	}

	// Empty query
	_, _, err = skeleton.Extract(g, "", nil)
	if err == nil {
		t.Fatal("want error for empty query, got nil")
	}
}
