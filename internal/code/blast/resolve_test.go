package blast_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/extract/cpp"
	"github.com/xidus90/loomux/internal/code/model"
)

func sampleGraph() *model.Graph {
	return &model.Graph{
		Nodes: []model.Node{
			{ID: "pkg/cache.go", Path: "pkg/cache.go", Name: "cache.go", Kind: model.KindFile, Span: "L1-L100"},
			{ID: "pkg/cache.go#Cache", Path: "pkg/cache.go", Name: "Cache", Kind: "struct", Span: "L10-L50"},
			{ID: "pkg/cache.go#Cache.Get", Path: "pkg/cache.go", Name: "Get", Kind: "method", Span: "L20-L30"},
			{ID: "pkg/cache.go#Cache.Get~2", Path: "pkg/cache.go", Name: "Get", Kind: "method", Span: "L35-L45"},
			{ID: "internal/hooks/write.go#Write", Path: "internal/hooks/write.go", Name: "Write", Kind: "func", Span: "L10-L25"},
			{ID: "internal/cli/write.go#Write", Path: "internal/cli/write.go", Name: "Write", Kind: "func", Span: "L10-L25"},
			{ID: "pkg/runner.go#go", Path: "pkg/runner.go", Name: "go", Kind: "func", Span: "L5-L15"},
			{ID: "pkg/runner.go", Path: "pkg/runner.go", Name: "runner.go", Kind: model.KindFile, Span: "L1-L50"},
		},
	}
}

func TestResolveExactAndSuffix(t *testing.T) {
	g := sampleGraph()

	// 1. Bare name match
	hits, err := blast.Resolve(g, "Cache", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "pkg/cache.go#Cache" {
		t.Fatalf("want Cache, got hits=%v err=%v", hits, err)
	}

	// 2. Qualified id suffix (Cache.Get) matches both bare and ~2 ordinal
	hits, err = blast.Resolve(g, "Cache.Get", "")
	if err != nil || len(hits) != 2 {
		t.Fatalf("want 2 Get methods, got hits=%v err=%v", hits, err)
	}

	// Case-insensitivity
	hits, err = blast.Resolve(g, "cache.get", "")
	if err != nil || len(hits) != 2 {
		t.Fatalf("want 2 Get methods (case-insensitive), got hits=%v err=%v", hits, err)
	}
}

func TestResolveGoPackageFilter(t *testing.T) {
	g := sampleGraph()

	// hooks.Write matches hooks package only
	hits, err := blast.Resolve(g, "hooks.Write", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "internal/hooks/write.go#Write" {
		t.Fatalf("want internal/hooks/write.go#Write, got %v err=%v", hits, err)
	}

	// cli.Write matches cli package only
	hits, err = blast.Resolve(g, "cli.Write", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "internal/cli/write.go#Write" {
		t.Fatalf("want internal/cli/write.go#Write, got %v err=%v", hits, err)
	}

	// Bare Write matches both
	hits, err = blast.Resolve(g, "Write", "")
	if err != nil || len(hits) != 2 {
		t.Fatalf("want 2 Write matches, got %v err=%v", hits, err)
	}
}

func TestResolveFileFallback(t *testing.T) {
	g := sampleGraph()

	// File by basename
	hits, err := blast.Resolve(g, "cache.go", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "pkg/cache.go" {
		t.Fatalf("want pkg/cache.go, got %v err=%v", hits, err)
	}

	// File by path
	hits, err = blast.Resolve(g, "pkg/cache.go", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "pkg/cache.go" {
		t.Fatalf("want pkg/cache.go, got %v err=%v", hits, err)
	}

	// Dateinamen-Schutz: runner.go ends with .go, must not match symbol 'go'
	hits, err = blast.Resolve(g, "runner.go", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "pkg/runner.go" {
		t.Fatalf("want file pkg/runner.go, got %v", hits)
	}
}

func TestResolveInPrefix(t *testing.T) {
	g := sampleGraph()

	// Filtering with valid prefix
	hits, err := blast.Resolve(g, "Write", "internal/hooks")
	if err != nil || len(hits) != 1 || hits[0].ID != "internal/hooks/write.go#Write" {
		t.Fatalf("want 1 hooks match, got %v err=%v", hits, err)
	}

	// Unknown prefix returns error (assertPrefixIndexed)
	_, err = blast.Resolve(g, "Write", "nonexistent")
	if err == nil {
		t.Fatalf("want error for unknown prefix, got nil")
	}
}

func TestResolveEdgeCases(t *testing.T) {
	g := sampleGraph()

	// Nil graph
	hits, err := blast.Resolve(nil, "Cache", "")
	if err != nil || len(hits) != 0 {
		t.Fatalf("want nil for nil graph, got %v err=%v", hits, err)
	}

	// Empty query
	hits, err = blast.Resolve(g, "", "")
	if err != nil || len(hits) != 0 {
		t.Fatalf("want nil for empty query, got %v err=%v", hits, err)
	}

	// Query with hash prefix #Cache
	hits, err = blast.Resolve(g, "#Cache", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "pkg/cache.go#Cache" {
		t.Fatalf("want Cache, got %v err=%v", hits, err)
	}

	// Query with dot prefix .Cache
	hits, err = blast.Resolve(g, ".Cache", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "pkg/cache.go#Cache" {
		t.Fatalf("want Cache, got %v err=%v", hits, err)
	}

	// Query with ordinal segment Cache~2.Get
	hits, err = blast.Resolve(g, "Cache~2.Get", "")
	if err != nil || len(hits) != 2 {
		t.Fatalf("want 2 Get matches, got %v err=%v", hits, err)
	}

	// Stage 2 fallback: unknownpkg.Get falls back to bare name Get
	hits, err = blast.Resolve(g, "unknownpkg.Get", "")
	if err != nil || len(hits) != 2 {
		t.Fatalf("want 2 Get matches via bare fallback, got %v err=%v", hits, err)
	}

	// Backslash path
	hits, err = blast.Resolve(g, `pkg\cache.go`, "")
	if err != nil || len(hits) != 1 || hits[0].ID != "pkg/cache.go" {
		t.Fatalf("want pkg/cache.go for backslash path, got %v err=%v", hits, err)
	}

	// Unknown file query with .go extension
	hits, err = blast.Resolve(g, "unknown.go", "")
	if err != nil || len(hits) != 0 {
		t.Fatalf("want 0 matches for unknown.go, got %v err=%v", hits, err)
	}

	// Query with dot where bare name doesn't match anything
	hits, err = blast.Resolve(g, "somepkg.UnknownFunc", "")
	if err != nil || len(hits) != 0 {
		t.Fatalf("want 0 matches for somepkg.UnknownFunc, got %v err=%v", hits, err)
	}

	// Nothing matches
	hits, err = blast.Resolve(g, "DoesNotExistAtAll", "")
	if err != nil || len(hits) != 0 {
		t.Fatalf("want 0 matches, got %v err=%v", hits, err)
	}
}

func TestResolveStageBoundaries(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{
		{ID: "pkg/util", Path: "pkg/util", Name: "util", Kind: model.KindFile},
		{ID: "pkg/x.go#util", Path: "pkg/x.go", Name: "util", Kind: "function"},
		{ID: "a/b/f.go#Foo", Path: "a/b/f.go", Name: "Foo", Kind: "function"},
		{ID: "docs/notes.txt", Path: "docs/notes.txt", Name: "notes.txt", Kind: model.KindFile},
		{ID: "n.go#Add2", Path: "n.go", Name: "Add2", Kind: "function"},
		{ID: "n.go#F~9", Path: "n.go", Name: "F~9", Kind: "function"},
		{ID: "n.go#G~0", Path: "n.go", Name: "G~0", Kind: "function"},
		{ID: "n.go#a~b", Path: "n.go", Name: "a~b", Kind: "function"},
		{ID: "n.go#", Path: "n.go", Name: "", Kind: "function"},
		{ID: "dir/x.go#y", Path: "dir/x.go", Name: "y", Kind: "function"},
	}}
	for _, c := range []struct {
		query string
		want  []model.NodeID
		why   string
	}{
		{"util", []model.NodeID{"pkg/x.go#util"}, "a file node is no exact-name hit"},
		{"pkg.util", []model.NodeID{"pkg/x.go#util"}, "a file node is no package-filter hit"},
		{"a/b.Foo", []model.NodeID{"a/b/f.go#Foo"}, "the package part may be the whole directory"},
		{"notes.txt", []model.NodeID{"docs/notes.txt"}, "no bare-name hit falls through to the files"},
		{"Ad", nil, "a digit after a letter is no ordinal"},
		{"F", []model.NodeID{"n.go#F~9"}, "~9 is an ordinal"},
		{"G", []model.NodeID{"n.go#G~0"}, "~0 is an ordinal"},
		{"ab", nil, "a ~ without digits stays"},
		{"", nil, "an empty query resolves to nothing"},
		{"dir/x.y", nil, "a slash makes a file query, never a bare name"},
		{`dir\x.y`, nil, "so does a backslash"},
	} {
		hits, err := blast.Resolve(g, c.query, "")
		var got []model.NodeID
		for _, h := range hits {
			got = append(got, h.ID)
		}
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Resolve(%q) = %v, %v, want %v: %s", c.query, got, err, c.want, c.why)
		}
	}
}

// A name ending in .cs is a file query, so "cs" never matches a symbol of that name.
func TestResolveTakesACSharpNameForAFile(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{
		{ID: "Game/Player.cs", Path: "Game/Player.cs", Name: "Player.cs", Kind: model.KindFile},
		{ID: "Game/Player.cs#cs", Path: "Game/Player.cs", Name: "cs", Kind: "function"},
	}}
	hits, err := blast.Resolve(g, "Player.cs", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "Game/Player.cs" {
		t.Errorf("Resolve(Player.cs) = %v, %v; want the file", hits, err)
	}
}

// A name ending in a JavaScript or TypeScript extension is a file query, so
// "mjs" or "cts" never matches a symbol of that name.
func TestResolveTakesAModuleNameForAFile(t *testing.T) {
	for _, ext := range []string{"mjs", "cjs", "mts", "cts"} {
		g := &model.Graph{Nodes: []model.Node{
			{ID: model.NodeID("src/a." + ext), Path: "src/a." + ext, Name: "a." + ext, Kind: model.KindFile},
			{ID: model.NodeID("src/a." + ext + "#" + ext), Path: "src/a." + ext, Name: ext, Kind: "function"},
		}}
		hits, err := blast.Resolve(g, "a."+ext, "")
		if err != nil || len(hits) != 1 || hits[0].ID != model.NodeID("src/a."+ext) {
			t.Errorf("Resolve(a.%s) = %v, %v; want the file", ext, hits, err)
		}
	}
}

// A name ending in .php is a file query, so "php" never matches a symbol of that name.
func TestResolveTakesAPHPNameForAFile(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{
		{ID: "src/User.php", Path: "src/User.php", Name: "User.php", Kind: model.KindFile},
		{ID: "src/User.php#php", Path: "src/User.php", Name: "php", Kind: "function"},
	}}
	hits, err := blast.Resolve(g, "User.php", "")
	if err != nil || len(hits) != 1 || hits[0].ID != "src/User.php" {
		t.Errorf("Resolve(User.php) = %v, %v; want the file", hits, err)
	}
}

// A name ending in .css, .scss, .html or .htm is a file query, so "css"
// never matches a mixin of that name.
func TestResolveTakesAStylesheetNameForAFile(t *testing.T) {
	for _, ext := range []string{"css", "scss", "html", "htm"} {
		g := &model.Graph{Nodes: []model.Node{
			{ID: model.NodeID("src/a." + ext), Path: "src/a." + ext, Name: "a." + ext, Kind: model.KindFile},
			{ID: model.NodeID("src/a." + ext + "#" + ext), Path: "src/a." + ext, Name: ext, Kind: "mixin"},
		}}
		hits, err := blast.Resolve(g, "a."+ext, "")
		if err != nil || len(hits) != 1 || hits[0].ID != model.NodeID("src/a."+ext) {
			t.Errorf("Resolve(a.%s) = %v, %v; want the file", ext, hits, err)
		}
	}
}

// A name ending in a C++ extension is a file query, so "hh" and "ipp" never
// match a symbol of that name.
func TestResolveTakesACppNameForAFile(t *testing.T) {
	for _, dotExt := range (cpp.Language{}).Extensions() {
		ext := strings.TrimPrefix(dotExt, ".")
		name := "Player." + ext
		g := &model.Graph{Nodes: []model.Node{
			{ID: model.NodeID("src/" + name), Path: "src/" + name, Name: name, Kind: model.KindFile},
			{ID: model.NodeID("src/" + name + "#" + ext), Path: "src/" + name, Name: ext, Kind: "function"},
		}}
		hits, err := blast.Resolve(g, name, "")
		if err != nil || len(hits) != 1 || hits[0].ID != model.NodeID("src/"+name) {
			t.Errorf("Resolve(%s) = %v, %v; want the file", name, hits, err)
		}
	}
}

// A name with a Godot extension is a file query: it must not fall through to
// the bare-name stage, where "gd" would find a symbol called gd.
func TestResolveTakesAGodotNameForAFile(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{
		{ID: "a/main.gd", Path: "a/main.gd", Name: "main.gd", Kind: model.KindFile},
		{ID: "a/main.gd#gd", Path: "a/main.gd", Name: "gd", Kind: "function"},
		{ID: "a/main.tscn", Path: "a/main.tscn", Name: "main.tscn", Kind: model.KindFile},
		{ID: "a/main.tscn#tscn", Path: "a/main.tscn", Name: "tscn", Kind: "function"},
		{ID: "a/project.godot", Path: "a/project.godot", Name: "project.godot", Kind: model.KindFile},
		{ID: "a/project.godot#godot", Path: "a/project.godot", Name: "godot", Kind: "function"},
		{ID: "a/x.tres", Path: "a/x.tres", Name: "x.tres", Kind: model.KindFile},
		{ID: "a/x.tres#tres", Path: "a/x.tres", Name: "tres", Kind: "function"},
	}}
	for q, id := range map[string]model.NodeID{
		"main.gd": "a/main.gd", "main.tscn": "a/main.tscn", "project.godot": "a/project.godot", "x.tres": "a/x.tres",
	} {
		hits, err := blast.Resolve(g, q, "")
		if err != nil || len(hits) != 1 || hits[0].ID != id {
			t.Errorf("Resolve(%q) = %v, %v; want the file %s", q, hits, err, id)
		}
	}
}

// Asked for Build, the user means an exported Build, not a test helper build: an
// exact-case name ranks before a case-insensitive one, and within each a
// symbol outside the tests before one inside, at every stage that matches
// symbols by name. Ties keep the graph's order.
func TestResolveRanksExactCaseThenNonTestFirst(t *testing.T) {
	g := &model.Graph{Nodes: []model.Node{
		{ID: "f/fp_test.go#build", Path: "f/fp_test.go", Name: "build", Kind: "function"},
		{ID: "f/x_test.go#Build", Path: "f/x_test.go", Name: "Build", Kind: "function"},
		{ID: "a/build.go#build", Path: "a/build.go", Name: "build", Kind: "function"},
		{ID: "t/format.go#Build", Path: "t/format.go", Name: "Build", Kind: "function"},
		{ID: "t/format.go#Build~2", Path: "t/format.go", Name: "Build~2", Kind: "function"},
	}}
	all := []model.NodeID{"t/format.go#Build", "t/format.go#Build~2", "f/x_test.go#Build", "a/build.go#build", "f/fp_test.go#build"}
	for _, c := range []struct {
		query string
		want  []model.NodeID
	}{
		{"Build", all},
		{"build", []model.NodeID{"a/build.go#build", "f/fp_test.go#build", "t/format.go#Build", "t/format.go#Build~2", "f/x_test.go#Build"}},
		{"t.Build", []model.NodeID{"t/format.go#Build", "t/format.go#Build~2"}},
		{"f.build", []model.NodeID{"f/fp_test.go#build", "f/x_test.go#Build"}},
		{"nope.Build", all},
	} {
		hits, err := blast.Resolve(g, c.query, "")
		var got []model.NodeID
		for _, h := range hits {
			got = append(got, h.ID)
		}
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("Resolve(%q) = %v, %v, want %v", c.query, got, err, c.want)
		}
	}
}
