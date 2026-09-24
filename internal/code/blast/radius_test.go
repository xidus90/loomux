package blast_test

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/diff"
	"github.com/xidus90/loomux/internal/code/model"
)

// radiusGraph is a small Go package written out by hand: a library with a
// function that is called, one that is not and a struct; a main that calls the
// library; a test that reaches the library only through a helper in another
// test file; and a file nobody uses.
func radiusGraph() *model.Graph {
	file := func(p string) model.Node {
		return model.Node{ID: model.NodeID(p), Name: p, Kind: model.KindFile, Path: p}
	}
	sym := func(p, name string, kind model.Kind, span model.Span) model.Node {
		return model.Node{ID: model.NodeID(p + "#" + name), Name: name, Kind: kind, Path: p, Span: span}
	}
	contains := func(p, name string) model.Edge {
		return model.Edge{Source: model.NodeID(p), Target: model.NodeID(p + "#" + name), Relation: model.RelationContains}
	}
	calls := func(from, to model.NodeID) model.Edge {
		return model.Edge{Source: from, Target: to, Relation: model.RelationCalls}
	}
	return &model.Graph{
		Meta: model.Meta{Version: 1},
		Nodes: []model.Node{
			file("lib.go"),
			sym("lib.go", "Add", "function", "L3-L5"),
			sym("lib.go", "Sub", "function", "L7-L9"),
			sym("lib.go", "T", "struct", "L11-L13"),
			file("main.go"),
			sym("main.go", "main", "function", "L3-L6"),
			file("lib_test.go"),
			sym("lib_test.go", "TestAdd", "function", "L3-L6"),
			file("helper_test.go"),
			sym("helper_test.go", "helper", "function", "L3-L5"),
			file("other.go"),
		},
		Edges: []model.Edge{
			contains("lib.go", "Add"),
			contains("lib.go", "Sub"),
			contains("lib.go", "T"),
			contains("main.go", "main"),
			contains("lib_test.go", "TestAdd"),
			contains("helper_test.go", "helper"),
			calls("main.go#main", "lib.go#Add"),
			calls("lib_test.go#TestAdd", "helper_test.go#helper"),
			calls("helper_test.go#helper", "lib.go#Add"),
			{Source: "main.go", Target: "lib.go", Relation: model.RelationImports},
		},
	}
}

func radius(g *model.Graph, depth blast.Depth, files ...diff.File) blast.Report {
	return blast.Radius(g, blast.New(g), files, nil, depth)
}

func changed(path string, lines ...int) diff.File {
	f := diff.File{Status: diff.Modified, Path: path}
	for _, l := range lines {
		f.Hunks = append(f.Hunks, diff.Hunk{From: l, To: l, Lines: []string{"+x"}})
	}
	return f
}

type hitView struct {
	ID    model.NodeID
	Depth int
	From  string
}

func hitViews(hits []blast.RadiusHit) []hitView {
	out := []hitView{}
	for _, h := range hits {
		out = append(out, hitView{h.ID, h.Depth, strings.Join(h.From, ",")})
	}
	return out
}

func seedIDs(a blast.Area) []model.NodeID {
	var out []model.NodeID
	for _, s := range a.Seeds {
		out = append(out, s.Node.ID)
	}
	return out
}

func TestRadiusSeedsTheTouchedSymbolAndWalksItsCallers(t *testing.T) {
	g := radiusGraph()

	rep := radius(g, 1, changed("lib.go", 4))

	if len(rep.Areas) != 1 {
		t.Fatalf("got %d areas, want 1", len(rep.Areas))
	}
	a := rep.Areas[0]
	if got := seedIDs(a); !reflect.DeepEqual(got, []model.NodeID{"lib.go#Add"}) {
		t.Errorf("seeds = %v, want [lib.go#Add]", got)
	}
	if a.Seeds[0].InDegree != 2 {
		t.Errorf("in-degree = %d, want 2", a.Seeds[0].InDegree)
	}
	want := []hitView{{"main.go#main", 1, "lib.go"}, {"helper_test.go#helper", 1, "lib.go"}}
	if got := hitViews(rep.Hits); !reflect.DeepEqual(got, want) {
		t.Errorf("hits at depth 1 = %v, want %v", got, want)
	}
	if a.Signal != blast.SignalStale {
		t.Errorf("signal = %q, want stale", a.Signal)
	}
	if want := []string{"helper_test.go", "lib_test.go"}; !reflect.DeepEqual(a.Tests, want) {
		t.Errorf("tests = %v, want %v -- the signal walks the closure whatever the depth", a.Tests, want)
	}

	all := radius(g, blast.All, changed("lib.go", 4))
	want = append(want, hitView{"lib_test.go#TestAdd", 2, "lib.go"})
	if got := hitViews(all.Hits); !reflect.DeepEqual(got, want) {
		t.Errorf("hits at depth all = %v, want %v", got, want)
	}
}

func TestRadiusSignalIsChangedWhenAReachingTestChangedToo(t *testing.T) {
	rep := radius(radiusGraph(), 1, changed("lib.go", 4), changed("lib_test.go", 4))

	if rep.Areas[0].Signal != blast.SignalChanged {
		t.Errorf("signal = %q, want changed", rep.Areas[0].Signal)
	}
}

// A withheld path is in the diff for the signal and nowhere else: no area,
// no walk from it, no evidence.
func TestRadiusSignalCountsAWithheldTestThatChanged(t *testing.T) {
	g := radiusGraph()
	rep := blast.Radius(g, blast.New(g), []diff.File{changed("lib.go", 4)}, []string{"lib_test.go"}, 1)

	if len(rep.Areas) != 1 || rep.Areas[0].Signal != blast.SignalChanged {
		t.Fatalf("areas = %+v, want lib.go changed", rep.Areas)
	}
	for _, e := range rep.Evidence {
		if e.Node.Path != "lib.go" {
			t.Errorf("evidence from %s", e.Node.Path)
		}
	}
	for _, h := range rep.Hits {
		if !reflect.DeepEqual(h.From, []string{"lib.go"}) {
			t.Errorf("hit %s from %v", h.ID, h.From)
		}
	}
}

func TestRadiusSignalIsNoneWithoutACaller(t *testing.T) {
	rep := radius(radiusGraph(), blast.All, changed("lib.go", 8))

	a := rep.Areas[0]
	if a.Signal != blast.SignalNone || a.Tests != nil {
		t.Errorf("signal = %q, tests = %v, want none and no tests", a.Signal, a.Tests)
	}
	if len(rep.Hits) != 0 {
		t.Errorf("hits = %v, want none", rep.Hits)
	}
}

func TestRadiusSignalIsNAWithoutABehaviouralSeed(t *testing.T) {
	rep := radius(radiusGraph(), blast.All, changed("lib.go", 12))

	a := rep.Areas[0]
	if got := seedIDs(a); !reflect.DeepEqual(got, []model.NodeID{"lib.go#T"}) {
		t.Errorf("seeds = %v, want [lib.go#T]", got)
	}
	if a.Signal != blast.SignalNA {
		t.Errorf("signal = %q, want na", a.Signal)
	}
}

func TestRadiusSeedsTheFileNodeOnlyWhenNoSymbolIsHitAndDoesNotExpandIt(t *testing.T) {
	rep := radius(radiusGraph(), blast.All, changed("lib.go", 1))

	a := rep.Areas[0]
	if got := seedIDs(a); !reflect.DeepEqual(got, []model.NodeID{"lib.go"}) {
		t.Errorf("seeds = %v, want the file node alone", got)
	}
	want := []hitView{{"main.go", 1, "lib.go"}}
	if got := hitViews(rep.Hits); !reflect.DeepEqual(got, want) {
		t.Errorf("hits = %v, want only the importer %v -- the file's symbols are not seeds", got, want)
	}
	if rep.Hits[0].Relation != model.RelationImports {
		t.Errorf("relation = %q, want imports", rep.Hits[0].Relation)
	}
	if a.Signal != blast.SignalNA {
		t.Errorf("signal = %q, want na for a file seed", a.Signal)
	}
	if len(rep.Evidence) != 0 {
		t.Errorf("evidence = %v, want none -- a file node is quoted nowhere", rep.Evidence)
	}
}

func TestRadiusSignalIsNAForATestFile(t *testing.T) {
	rep := radius(radiusGraph(), blast.All, changed("lib_test.go", 4))

	if a := rep.Areas[0]; a.Path != "lib_test.go" || a.Signal != blast.SignalNA {
		t.Errorf("area %q signal = %q, want lib_test.go with na", a.Path, a.Signal)
	}
}

func TestRadiusListsDeletedAndUnindexedFilesWithoutWalking(t *testing.T) {
	gone := diff.File{Status: diff.Deleted, Path: "lib.go", Hunks: []diff.Hunk{{From: 1, To: 1}}}

	rep := radius(radiusGraph(), blast.All, gone, changed("new.go", 1))

	if !reflect.DeepEqual(rep.Deleted, []string{"lib.go"}) {
		t.Errorf("deleted = %v, want [lib.go]", rep.Deleted)
	}
	if !reflect.DeepEqual(rep.Unindexed, []string{"new.go"}) {
		t.Errorf("unindexed = %v, want [new.go]", rep.Unindexed)
	}
	if len(rep.Areas) != 0 || len(rep.Hits) != 0 {
		t.Errorf("areas = %v, hits = %v, want neither", rep.Areas, rep.Hits)
	}
}

func TestRadiusMergesAHitReachedFromTwoFilesAtTheShallowestDepth(t *testing.T) {
	rep := radius(radiusGraph(), blast.All, changed("lib.go", 4), changed("helper_test.go", 4))

	want := []hitView{
		{"main.go#main", 1, "lib.go"},
		{"helper_test.go#helper", 1, "lib.go"},
		{"lib_test.go#TestAdd", 1, "lib.go,helper_test.go"},
	}
	if got := hitViews(rep.Hits); !reflect.DeepEqual(got, want) {
		t.Errorf("hits = %v, want %v", got, want)
	}
	if rep.Hits[2].Relation != model.RelationCalls {
		t.Errorf("relation = %q, want calls", rep.Hits[2].Relation)
	}
}

func TestRadiusKeepsTheFromListFreeOfRepeats(t *testing.T) {
	// A file named twice in the diff reaches the same hits twice.
	rep := radius(radiusGraph(), 1, changed("lib.go", 4), changed("lib.go", 4))

	for _, h := range rep.Hits {
		if !reflect.DeepEqual(h.From, []string{"lib.go"}) {
			t.Errorf("hit %s from = %v, want [lib.go] once", h.ID, h.From)
		}
	}
}

// evidenceGraph is one file with five functions of two lines each and one
// function spanning ten lines.
func evidenceGraph() *model.Graph {
	g := &model.Graph{Nodes: []model.Node{{ID: "e.go", Kind: model.KindFile, Path: "e.go"}}}
	for i, name := range []string{"A", "B", "C", "D", "E"} {
		from := 3 + 3*i
		g.Nodes = append(g.Nodes, model.Node{
			ID: model.NodeID("e.go#" + name), Name: name, Kind: "function", Path: "e.go",
			Span: model.Span("L" + strconv.Itoa(from) + "-L" + strconv.Itoa(from+1)),
		})
	}
	g.Nodes = append(g.Nodes, model.Node{ID: "e.go#Long", Name: "Long", Kind: "function", Path: "e.go", Span: "L30-L40"})
	return g
}

func TestRadiusQuotesAtMostFourSymbolsAndSixLinesEach(t *testing.T) {
	g := evidenceGraph()
	five := diff.File{Status: diff.Modified, Path: "e.go", Hunks: []diff.Hunk{
		{From: 3, To: 3, Lines: []string{"+a"}},
		{From: 6, To: 6, Lines: []string{"+b"}},
		{From: 9, To: 9, Lines: []string{"+c"}},
		{From: 12, To: 12, Lines: []string{"+d"}},
		{From: 15, To: 15, Lines: []string{"+e"}},
	}}

	rep := radius(g, 1, five)

	if len(rep.Evidence) != blast.MaxEvidence || rep.MoreEvidence != 1 {
		t.Fatalf("evidence = %d, more = %d, want 4 and 1", len(rep.Evidence), rep.MoreEvidence)
	}
	if e := rep.Evidence[1]; e.Node.ID != "e.go#B" || !reflect.DeepEqual(e.Lines, []string{"+b"}) {
		t.Errorf("second evidence = %s %v, want B with only its own hunk", e.Node.ID, e.Lines)
	}

	nine := []string{"+1", "+2", "+3", "+4", "+5", "+6", "+7", "+8", "+9"}
	long := diff.File{Status: diff.Modified, Path: "e.go", Hunks: []diff.Hunk{{From: 31, To: 39, Lines: nine}}}

	rep = radius(g, 1, long)

	if len(rep.Evidence) != 1 || rep.MoreEvidence != 0 {
		t.Fatalf("evidence = %d, more = %d, want 1 and 0", len(rep.Evidence), rep.MoreEvidence)
	}
	e := rep.Evidence[0]
	if !reflect.DeepEqual(e.Lines, nine[:blast.MaxEvidenceLines]) || e.More != 3 {
		t.Errorf("lines = %v, more = %d, want the first six and 3", e.Lines, e.More)
	}
}

func TestRadiusDropsAHitWithoutANode(t *testing.T) {
	g := radiusGraph()
	g.Edges = append(g.Edges, model.Edge{Source: "ghost", Target: "lib.go#Sub", Relation: model.RelationCalls})

	rep := radius(g, blast.All, changed("lib.go", 8))

	if len(rep.Hits) != 0 {
		t.Errorf("hits = %v, want none -- an id without a node is no area to name", rep.Hits)
	}
	if rep.Areas[0].Signal != blast.SignalNone {
		t.Errorf("signal = %q, want none", rep.Areas[0].Signal)
	}
}

func TestRadiusReportEncodesItsFieldNames(t *testing.T) {
	rep := radius(radiusGraph(), 1, changed("lib.go", 4))

	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"areas"`, `"hits"`, `"id"`, `"relation"`, `"depth"`, `"from"`, `"status":"M"`, `"signal":"stale"`, `"in_degree"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("json misses %s: %s", key, raw)
		}
	}
}

func TestIsTestPath(t *testing.T) {
	for path, want := range map[string]bool{"a_test.go": true, "x/b_test.go": true, "a.go": false, "test.go": false} {
		if got := blast.IsTestPath(path); got != want {
			t.Errorf("IsTestPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestRadiusKeepsTheShallowerDepthWhenALaterFileReachesDeeper(t *testing.T) {
	// helper_test.go reaches TestAdd at depth 1, lib.go only at depth 2.
	rep := radius(radiusGraph(), blast.All, changed("helper_test.go", 4), changed("lib.go", 4))

	found := false
	for _, h := range rep.Hits {
		if h.ID == "lib_test.go#TestAdd" {
			found = true
			if h.Depth != 1 {
				t.Errorf("TestAdd depth = %d, want 1 -- a deeper second reach must not overwrite it", h.Depth)
			}
		}
	}
	if !found {
		t.Fatalf("TestAdd is not among the hits %+v", rep.Hits)
	}
}

func TestRadiusKeepsTheFirstRelationAtAnEqualDepth(t *testing.T) {
	g := radiusGraph()
	g.Edges = append(g.Edges,
		model.Edge{Source: "other.go", Target: "lib.go#Add", Relation: model.RelationCalls},
		model.Edge{Source: "other.go", Target: "main.go#main", Relation: model.RelationImports})

	rep := radius(g, 1, changed("lib.go", 4), changed("main.go", 4))

	found := false
	for _, h := range rep.Hits {
		if h.ID == "other.go" {
			found = true
			if h.Relation != model.RelationCalls {
				t.Errorf("other.go relation = %q, want calls -- the first file's edge at the same depth stays", h.Relation)
			}
		}
	}
	if !found {
		t.Fatalf("other.go is not among the hits %+v", rep.Hits)
	}
}

func TestRadiusSeedsASymbolTwoHunksTouchOnce(t *testing.T) {
	rep := radius(radiusGraph(), 1, changed("lib.go", 3, 4))

	if got := seedIDs(rep.Areas[0]); !reflect.DeepEqual(got, []model.NodeID{"lib.go#Add"}) {
		t.Errorf("seeds = %v, want [lib.go#Add] once", got)
	}
}

func TestRadiusCountsAMethodAsBehaviour(t *testing.T) {
	g := radiusGraph()
	g.Nodes[1].Kind = "method"

	rep := radius(g, 1, changed("lib.go", 4))

	if a := rep.Areas[0]; a.Signal != blast.SignalStale {
		t.Errorf("signal = %q, want stale for a method its tests reach", a.Signal)
	}
}

func TestRadiusQuotesAHunkThatStartsOnTheSpansLastLine(t *testing.T) {
	g := evidenceGraph()
	last := diff.File{Status: diff.Modified, Path: "e.go", Hunks: []diff.Hunk{{From: 4, To: 4, Lines: []string{"+z"}}}}

	rep := radius(g, 1, last)

	if len(rep.Evidence) != 1 || rep.Evidence[0].Node.ID != "e.go#A" || !reflect.DeepEqual(rep.Evidence[0].Lines, []string{"+z"}) {
		t.Errorf("evidence = %+v, want A quoting +z -- a hunk overlapping the span's last line is inside it", rep.Evidence)
	}
}
