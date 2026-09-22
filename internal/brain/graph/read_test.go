package graph_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/config"
)

func TestReadGraph_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	graphContent := `{
  "scope": "project/test",
  "nodes": [
    {"id": "a.md", "title": "A", "tags": ["tag1"]},
    {"id": "b.md", "title": "B", "tags": []}
  ],
  "edges": [
    {"from": "a.md", "to": "b.md"}
  ],
  "links": {
    "total": 1,
    "resolved": 1,
    "dropped": {"external": 0}
  }
}`
	graphPath := filepath.Join(tmpDir, "graph.json")
	if err := os.WriteFile(graphPath, []byte(graphContent), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{
		Scope: "project/test",
		Path:  tmpDir,
	}

	g, err := graph.ReadGraph(area, tmpDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if g.Scope != "project/test" {
		t.Errorf("expected scope project/test, got %q", g.Scope)
	}
	if len(g.Nodes) != 2 || g.Nodes[0].ID != "a.md" || g.Nodes[0].Title != "A" || len(g.Nodes[0].Tags) != 1 || g.Nodes[0].Tags[0] != "tag1" {
		t.Errorf("nodes not parsed properly: %+v", g.Nodes)
	}
	if len(g.Edges) != 1 || g.Edges[0].From != "a.md" || g.Edges[0].To != "b.md" {
		t.Errorf("edges not parsed properly: %+v", g.Edges)
	}
	if g.Links.Total != 1 || g.Links.Resolved != 1 || g.Links.Dropped["external"] != 0 {
		t.Errorf("links not parsed properly: %+v", g.Links)
	}
}

func TestReadGraph_ReadOnlyAreaInStateDir(t *testing.T) {
	stateDir := t.TempDir()
	areaStateDir := filepath.Join(stateDir, "areas", "project-readonly")
	if err := os.MkdirAll(areaStateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	graphContent := `{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": {}}}`
	if err := os.WriteFile(filepath.Join(areaStateDir, "graph.json"), []byte(graphContent), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{
		Scope:    "project/readonly",
		Path:     filepath.Join(stateDir, "nonexistent"),
		ReadOnly: true,
	}

	g, err := graph.ReadGraph(area, stateDir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(g.Edges) != 0 {
		t.Errorf("expected empty edges, got %d", len(g.Edges))
	}
}

func TestReadGraph_NeverIndexed(t *testing.T) {
	tmpDir := t.TempDir()
	area := config.Area{
		Scope: "project/missing",
		Path:  tmpDir,
	}

	_, err := graph.ReadGraph(area, tmpDir, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "project/missing: never indexed; run `brain reindex`"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestParseGraph_InvalidJSON(t *testing.T) {
	path := "/path/to/graph.json"
	_, err := graph.ParseGraph([]byte("not json"), path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.HasPrefix(err.Error(), path+": graph is not valid JSON") || !strings.HasSuffix(err.Error(), "; delete it and run `brain reindex`") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestParseGraph_MissingKeys(t *testing.T) {
	path := "/path/to/graph.json"

	tests := []struct {
		json string
		want string
	}{
		{`{}`, path + ": graph is missing edges, links; delete it and run `brain reindex`"},
		{`{"links": {}}`, path + ": graph is missing edges; delete it and run `brain reindex`"},
		{`{"edges": []}`, path + ": graph is missing links; delete it and run `brain reindex`"},
	}

	for _, tc := range tests {
		_, err := graph.ParseGraph([]byte(tc.json), path)
		if err == nil {
			t.Fatalf("expected error for %s, got nil", tc.json)
		}
		if err.Error() != tc.want {
			t.Errorf("got %q, want %q", err.Error(), tc.want)
		}
	}
}

func TestParseGraph_InvalidShape(t *testing.T) {
	path := "/path/to/graph.json"

	tests := []struct {
		json string
		want string
	}{
		// Edges not a list
		{
			`{"edges": "not a list", "links": {"total": 0, "resolved": 0, "dropped": {}}}`,
			path + ": edges is not a list of from/to entries; delete it and run `brain reindex`",
		},
		// Edge element not dict or missing from/to
		{
			`{"edges": [1], "links": {"total": 0, "resolved": 0, "dropped": {}}}`,
			path + ": edges is not a list of from/to entries; delete it and run `brain reindex`",
		},
		{
			`{"edges": [{"from": "a"}], "links": {"total": 0, "resolved": 0, "dropped": {}}}`,
			path + ": edges is not a list of from/to entries; delete it and run `brain reindex`",
		},
		{
			`{"edges": [{"to": "b"}], "links": {"total": 0, "resolved": 0, "dropped": {}}}`,
			path + ": edges is not a list of from/to entries; delete it and run `brain reindex`",
		},
		// Links not dict
		{
			`{"edges": [], "links": "not a dict"}`,
			path + ": links is not a total/resolved/dropped record; delete it and run `brain reindex`",
		},
		// Links missing total, resolved, or dropped
		{
			`{"edges": [], "links": {"total": 0, "resolved": 0}}`,
			path + ": links is not a total/resolved/dropped record; delete it and run `brain reindex`",
		},
		// Links.dropped not dict
		{
			`{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": []}}`,
			path + ": links.dropped is not a table of reasons; delete it and run `brain reindex`",
		},
		// Link counts not numbers
		{
			`{"edges": [], "links": {"total": "not-int", "resolved": 0, "dropped": {}}}`,
			path + ": the link counts are not numbers; delete it and run `brain reindex`",
		},
		{
			`{"edges": [], "links": {"total": 0, "resolved": "not-int", "dropped": {}}}`,
			path + ": the link counts are not numbers; delete it and run `brain reindex`",
		},
		{
			`{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": {"external": "one"}}}`,
			path + ": the link counts are not numbers; delete it and run `brain reindex`",
		},
	}

	for _, tc := range tests {
		_, err := graph.ParseGraph([]byte(tc.json), path)
		if err == nil {
			t.Fatalf("expected error for %s, got nil", tc.json)
		}
		if err.Error() != tc.want {
			t.Errorf("got %q, want %q", err.Error(), tc.want)
		}
	}
}

func TestParseGraph_InvalidNodesType(t *testing.T) {
	path := "/path/to/graph.json"
	json := `{"edges": [], "links": {"total": 0, "resolved": 0, "dropped": {}}, "nodes": "not-a-list"}`
	_, err := graph.ParseGraph([]byte(json), path)
	if err == nil {
		t.Fatal("expected error for invalid nodes, got nil")
	}
}

func TestReadGraph_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	graphPath := filepath.Join(tmpDir, "graph.json")
	if err := os.Mkdir(graphPath, 0o755); err != nil {
		t.Fatal(err)
	}

	area := config.Area{
		Scope: "project/test",
		Path:  tmpDir,
	}

	_, err := graph.ReadGraph(area, tmpDir, "")
	if err == nil {
		t.Fatal("expected error reading directory as file, got nil")
	}
}
