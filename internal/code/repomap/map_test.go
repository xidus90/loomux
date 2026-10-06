package repomap_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/repomap"
)

func sampleMapGraph() *model.Graph {
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			{ID: "main.go", Path: "main.go", Name: "main.go", Kind: model.KindFile, Span: "L1-L20"},
			{ID: "main.go#main", Path: "main.go", Name: "main", Kind: "func", Span: "L5-L15"},

			{ID: "pkg/cache.go", Path: "pkg/cache.go", Name: "cache.go", Kind: model.KindFile, Span: "L1-L100"},
			{ID: "pkg/cache.go#Cache", Path: "pkg/cache.go", Name: "Cache", Kind: "struct", Span: "L10-L50"},
			{ID: "pkg/cache.go#Get", Path: "pkg/cache.go", Name: "Get", Kind: "method", Span: "L20-L30"},

			{ID: "pkg/store.go", Path: "pkg/store.go", Name: "store.go", Kind: model.KindFile, Span: "L1-L50"},
			{ID: "pkg/store.go#Store", Path: "pkg/store.go", Name: "Store", Kind: "struct", Span: "L5-L25"},

			{ID: "web/app.ts", Path: "web/app.ts", Name: "app.ts", Kind: model.KindFile, Span: "L1-L50"},
			{ID: "web/app.ts#App", Path: "web/app.ts", Name: "App", Kind: "class", Span: "L5-L30"},
		},
		Edges: []model.Edge{
			// Cache has 3 callers (inDegree 3)
			{Source: "a", Target: "pkg/cache.go#Cache", Relation: model.RelationCalls},
			{Source: "b", Target: "pkg/cache.go#Cache", Relation: model.RelationCalls},
			{Source: "c", Target: "pkg/cache.go#Cache", Relation: model.RelationCalls},
			// Get has 2 callers (inDegree 2)
			{Source: "a", Target: "pkg/cache.go#Get", Relation: model.RelationCalls},
			{Source: "b", Target: "pkg/cache.go#Get", Relation: model.RelationCalls},
			// Store has 2 callers (inDegree 2)
			{Source: "a", Target: "pkg/store.go#Store", Relation: model.RelationCalls},
			{Source: "b", Target: "pkg/store.go#Store", Relation: model.RelationCalls},
			// main has 0 callers
			// App has 1 caller
			{Source: "a", Target: "web/app.ts#App", Relation: model.RelationCalls},
		},
	}
}

func TestBuildBasicAndTotals(t *testing.T) {
	g := sampleMapGraph()
	x := blast.New(g)

	m := repomap.Build(g, x, repomap.Options{})

	// Totals
	if m.Totals.Files != 4 {
		t.Errorf("want 4 files, got %d", m.Totals.Files)
	}
	if m.Totals.Symbols != 5 {
		t.Errorf("want 5 symbols, got %d", m.Totals.Symbols)
	}
	if m.Totals.Edges != 8 {
		t.Errorf("want 8 edges, got %d", m.Totals.Edges)
	}
	// Languages should be ["go", "typescript"]
	if len(m.Totals.Languages) != 2 || m.Totals.Languages[0] != "go" || m.Totals.Languages[1] != "typescript" {
		t.Errorf("want [go, typescript], got %v", m.Totals.Languages)
	}

	// Hotspots: top nodes with inDegree > 0
	// 1. Cache (3)
	// 2. Get (2, Name "Get" < "Store")
	// 3. Store (2)
	// 4. App (1)
	// main has inDegree 0, must be excluded
	if len(m.Hotspots) != 4 {
		t.Fatalf("want 4 hotspots (excluding inDegree 0), got %d: %v", len(m.Hotspots), m.Hotspots)
	}
	if m.Hotspots[0].Name != "Cache" || m.Hotspots[0].InDegree != 3 {
		t.Errorf("hotspot 0: want Cache (3), got %v", m.Hotspots[0])
	}
	if m.Hotspots[1].Name != "Get" || m.Hotspots[1].InDegree != 2 {
		t.Errorf("hotspot 1: want Get (2), got %v", m.Hotspots[1])
	}
	if m.Hotspots[2].Name != "Store" || m.Hotspots[2].InDegree != 2 {
		t.Errorf("hotspot 2: want Store (2), got %v", m.Hotspots[2])
	}
	if m.Hotspots[3].Name != "App" || m.Hotspots[3].InDegree != 1 {
		t.Errorf("hotspot 3: want App (1), got %v", m.Hotspots[3])
	}

	// Dirs: main.go (root file), pkg (dir), web (dir)
	if len(m.Dirs) != 3 {
		t.Fatalf("want 3 dir entries, got %d: %v", len(m.Dirs), m.Dirs)
	}
	// Dirs sorted alphabetically: main.go, pkg, web
	if m.Dirs[0].Path != "main.go" || !m.Dirs[0].IsFile || m.Dirs[0].Files != 1 || m.Dirs[0].Symbols != 1 {
		t.Errorf("dir 0 mismatch: %v", m.Dirs[0])
	}
	if m.Dirs[1].Path != "pkg" || m.Dirs[1].IsFile || m.Dirs[1].Files != 2 || m.Dirs[1].Symbols != 3 {
		t.Errorf("dir 1 mismatch: %v", m.Dirs[1])
	}
	// Hubs in pkg: Cache (3), Get (2), Store (2)
	if len(m.Dirs[1].Hubs) != 3 {
		t.Errorf("want 3 hubs in pkg, got %d", len(m.Dirs[1].Hubs))
	}
}

func TestMonolithRefinement(t *testing.T) {
	// 10 files total. If internal/ has 7 files (70% > 60%), internal/ is refined
	var nodes []model.Node
	// 3 files outside internal
	nodes = append(nodes,
		model.Node{ID: "main.go", Path: "main.go", Name: "main.go", Kind: model.KindFile},
		model.Node{ID: "pkg/a.go", Path: "pkg/a.go", Name: "a.go", Kind: model.KindFile},
		model.Node{ID: "pkg/b.go", Path: "pkg/b.go", Name: "b.go", Kind: model.KindFile},
	)
	// 4 files in internal/code
	for i := 1; i <= 4; i++ {
		path := "internal/code/file" + string(rune('0'+i)) + ".go"
		nodes = append(nodes, model.Node{ID: model.NodeID(path), Path: path, Name: path, Kind: model.KindFile})
	}
	// 3 files in internal/cli
	for i := 1; i <= 3; i++ {
		path := "internal/cli/file" + string(rune('0'+i)) + ".go"
		nodes = append(nodes, model.Node{ID: model.NodeID(path), Path: path, Name: path, Kind: model.KindFile})
	}
	// 1 file directly under internal/ (len(parts) == 2 in split dir)
	nodes = append(nodes, model.Node{ID: "internal/version.go", Path: "internal/version.go", Name: "version.go", Kind: model.KindFile})

	g := &model.Graph{Nodes: nodes}
	x := blast.New(g)

	m := repomap.Build(g, x, repomap.Options{})

	paths := make(map[string]bool)
	for _, d := range m.Dirs {
		paths[d.Path] = true
	}
	if !paths["internal/code"] || !paths["internal/cli"] || !paths["internal"] {
		t.Errorf("want internal/code, internal/cli and internal in dirs, got %v", m.Dirs)
	}
}

func TestMaxDirsCappingAndTieBreakers(t *testing.T) {
	var nodes []model.Node
	// Create directories: dir1 has 2 symbols, dir2 has 1 symbol, others have 0
	dirNames := []string{"dir1", "dir2", "dir3", "dir4", "dir5"}
	for _, d := range dirNames {
		path := d + "/file.go"
		nodes = append(nodes, model.Node{ID: model.NodeID(path), Path: path, Name: "file.go", Kind: model.KindFile})
	}
	// dir1 has 2 symbols
	nodes = append(nodes,
		model.Node{ID: "dir1/file.go#S1", Path: "dir1/file.go", Name: "S1", Kind: "func"},
		model.Node{ID: "dir1/file.go#S2", Path: "dir1/file.go", Name: "S2", Kind: "func"},
		model.Node{ID: "dir2/file.go#S3", Path: "dir2/file.go", Name: "S3", Kind: "func"},
	)
	g := &model.Graph{Nodes: nodes}
	x := blast.New(g)

	// Cap at MaxDirs = 3
	m := repomap.Build(g, x, repomap.Options{MaxDirs: 3})
	if len(m.Dirs) != 3 {
		t.Fatalf("want 3 dirs, got %d", len(m.Dirs))
	}
	if m.Dropped != 2 {
		t.Errorf("want 2 dropped dirs, got %d", m.Dropped)
	}
	// dir1 (2 symbols) and dir2 (1 symbol) must be preferred over dir3/dir4/dir5 (0 symbols)
	if m.Dirs[0].Path != "dir1" || m.Dirs[1].Path != "dir2" {
		t.Errorf("expected dir1 and dir2 kept by symbol count tie breaker, got %v", m.Dirs)
	}
}

func TestCappingAndLanguages(t *testing.T) {
	nodes := []model.Node{
		{ID: "a.js", Path: "a.js", Name: "a.js", Kind: model.KindFile},
		{ID: "b.py", Path: "b.py", Name: "b.py", Kind: model.KindFile},
		{ID: "data.json", Path: "data.json", Name: "data.json", Kind: model.KindFile},
		{ID: "main.gd", Path: "main.gd", Name: "main.gd", Kind: model.KindFile},
		{ID: "main.tscn", Path: "main.tscn", Name: "main.tscn", Kind: model.KindFile},
		{ID: "project.godot", Path: "project.godot", Name: "project.godot", Kind: model.KindFile},
		{ID: "x.tres", Path: "x.tres", Name: "x.tres", Kind: model.KindFile},
		{ID: "Makefile", Path: "Makefile", Name: "Makefile", Kind: model.KindFile},

		{ID: "pkg/file.go", Path: "pkg/file.go", Name: "file.go", Kind: model.KindFile},
		{ID: "pkg/other.go", Path: "pkg/other.go", Name: "other.go", Kind: model.KindFile},

		// Symbols in same directory with same inDegree and same Name in different files for path tie-breaker
		{ID: "pkg/file.go#Dup", Path: "pkg/file.go", Name: "Dup", Kind: "func"},
		{ID: "pkg/other.go#Dup", Path: "pkg/other.go", Name: "Dup", Kind: "func"},
		{ID: "pkg/file.go#Extra", Path: "pkg/file.go", Name: "Extra", Kind: "func"},
		// Orphaned symbol with no file node in graph
		{ID: "orphan/file.go#Orphan", Path: "orphan/file.go", Name: "Orphan", Kind: "func"},
	}
	edges := []model.Edge{
		{Source: "a", Target: "pkg/file.go#Dup", Relation: model.RelationCalls},
		{Source: "b", Target: "pkg/other.go#Dup", Relation: model.RelationCalls},
		{Source: "c", Target: "pkg/file.go#Extra", Relation: model.RelationCalls},
	}
	g := &model.Graph{Nodes: nodes, Edges: edges}

	// Passing nil x to test auto-indexing, and limits to test capping
	m := repomap.Build(g, nil, repomap.Options{Hotspots: 2, HubsPerDir: 1})

	if len(m.Hotspots) != 2 {
		t.Errorf("want 2 hotspots capped, got %d", len(m.Hotspots))
	}
	// Check languages include javascript, python, json, unknown
	langs := strings.Join(m.Totals.Languages, ",")
	if !strings.Contains(langs, "javascript") || !strings.Contains(langs, "python") || !strings.Contains(langs, "json") || !strings.Contains(langs, "unknown") || !slices.Contains(m.Totals.Languages, "gdscript") || slices.Contains(m.Totals.Languages, "gd") || slices.Contains(m.Totals.Languages, "tscn") || slices.Contains(m.Totals.Languages, "godot") || slices.Contains(m.Totals.Languages, "tres") {
		t.Errorf("unexpected languages: %s", langs)
	}
}

func TestFormatAndEdgeCases(t *testing.T) {
	g := sampleMapGraph()
	x := blast.New(g)

	m := repomap.Build(g, x, repomap.Options{MaxDirs: 2})
	formatted := repomap.Format(m)

	if !strings.Contains(formatted, "files") || !strings.Contains(formatted, "Hotspots") || !strings.Contains(formatted, "Directories") {
		t.Fatalf("unexpected format output: %s", formatted)
	}
	if !strings.Contains(formatted, "dropped") {
		t.Fatalf("expected dropped mention in formatted output: %s", formatted)
	}

	// Nil graph
	nilMap := repomap.Build(nil, nil, repomap.Options{})
	if nilMap.Totals.Files != 0 {
		t.Errorf("want 0 files for nil graph, got %d", nilMap.Totals.Files)
	}
	nilFormatted := repomap.Format(nilMap)
	if nilFormatted == "" {
		t.Error("expected non-empty string for empty map format")
	}
}
