package resolve_test

import (
	"encoding/json"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

// stamp is the extractor identity these tests hand to Graph. resolve writes
// whatever its caller names and knows no list of languages; query passes
// all.Version().
const stamp = "test/1"

// result is a hand-built extraction of one file, so these tests exercise
// resolve alone and never the parser. The package clause defaults to the last
// segment of the directory, which is the ordinary case.
func result(rel string, nodes []model.Node, edges []extract.RawEdge) extract.Result {
	return extract.Result{Path: rel, Language: "go", Package: path.Base(path.Dir(rel)), Nodes: nodes, Edges: edges}
}

// importing is result plus the imports the file wrote, which is what a selector
// resolves through.
func importing(rel string, imports []extract.Import, nodes []model.Node, edges []extract.RawEdge) extract.Result {
	r := result(rel, nodes, edges)
	r.Imports = imports
	return r
}

func fileNode(rel string) model.Node {
	return model.Node{ID: model.NodeID(rel), Name: rel, Kind: model.KindFile, Path: rel, BodyHash: "h", Exported: true}
}

func fn(rel, name string, exported bool) model.Node {
	return model.Node{
		ID: model.NodeID(rel + "#" + name), Name: name, Kind: "function",
		Path: rel, BodyHash: "h", Exported: exported,
	}
}

func edgeBetween(g *model.Graph, from, to model.NodeID, rel model.Relation) *model.Edge {
	for i := range g.Edges {
		e := &g.Edges[i]
		if e.Source == from && e.Target == to && e.Relation == rel {
			return e
		}
	}
	return nil
}

func TestGraphResolvesASameFileCallAsExtracted(t *testing.T) {
	files := []extract.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "caller", false), fn("a.go", "local", false)},
		[]extract.RawEdge{{Source: "a.go#caller", Relation: model.RelationCalls, Name: "local", File: "a.go"}},
	)}

	g := resolve.Graph(files, nil, stamp)
	e := edgeBetween(g, "a.go#caller", "a.go#local", model.RelationCalls)
	if e == nil {
		t.Fatalf("edge missing; got %+v", g.Edges)
	}
	// Same file, so the target is certain.
	if e.Confidence != model.ConfidenceExtracted {
		t.Errorf("confidence = %q, want extracted", e.Confidence)
	}
}

func TestGraphResolvesAUniqueCrossFileCallAsInferred(t *testing.T) {
	files := []extract.Result{
		result("a.go",
			[]model.Node{fileNode("a.go"), fn("a.go", "caller", false)},
			[]extract.RawEdge{{Source: "a.go#caller", Relation: model.RelationCalls, Name: "helper", File: "a.go"}},
		),
		result("b.go", []model.Node{fileNode("b.go"), fn("b.go", "helper", false)}, nil),
	}

	g := resolve.Graph(files, nil, stamp)
	e := edgeBetween(g, "a.go#caller", "b.go#helper", model.RelationCalls)
	if e == nil {
		t.Fatalf("edge missing; got %+v", g.Edges)
	}
	// One match across files: shadowing could in principle fool this, so it is
	// inferred and not extracted.
	if e.Confidence != model.ConfidenceInferred {
		t.Errorf("confidence = %q, want inferred", e.Confidence)
	}
}

func TestGraphDropsAnAmbiguousCall(t *testing.T) {
	files := []extract.Result{
		result("a.go",
			[]model.Node{fileNode("a.go"), fn("a.go", "caller", false)},
			[]extract.RawEdge{{Source: "a.go#caller", Relation: model.RelationCalls, Name: "helper", File: "a.go"}},
		),
		result("b_windows.go", []model.Node{fileNode("b_windows.go"), fn("b_windows.go", "helper", false)}, nil),
		result("b_other.go", []model.Node{fileNode("b_other.go"), fn("b_other.go", "helper", false)}, nil),
	}

	g := resolve.Graph(files, nil, stamp)
	// Build constraints are not evaluated: a platform pair is two definitions,
	// the call is ambiguous, and the edge falls. Sound by the rule, and stated
	// in 6.1 of the spec so nobody files it as a bug.
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("an ambiguous call must yield no edge; got %+v", e)
		}
	}
}

func TestGraphResolvesAMemberCallThroughTheOwner(t *testing.T) {
	method := model.Node{
		ID: "a.go#Cache.Get", Name: "Get", Kind: "method", Owner: "Cache",
		Path: "a.go", BodyHash: "h", Exported: true,
	}
	files := []extract.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "caller", false), method},
		[]extract.RawEdge{{
			Source: "a.go#caller", Relation: model.RelationCalls,
			Name: "Get", Owner: "Cache", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil, stamp)
	if edgeBetween(g, "a.go#caller", "a.go#Cache.Get", model.RelationCalls) == nil {
		t.Fatalf("a member call must resolve against the owner-qualified index; got %+v", g.Edges)
	}
}

func TestGraphResolvesAPackageSelectorInsideTheTargetPackageOnly(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "New", Receiver: "blast", File: "cli/run.go",
			}},
		),
		result("blast/index.go", []model.Node{fileNode("blast/index.go"), fn("blast/index.go", "New", true)}, nil),
		// A same-named function elsewhere must not be reachable this way.
		result("other/thing.go", []model.Node{fileNode("other/thing.go"), fn("other/thing.go", "New", true)}, nil),
	}

	g := resolve.Graph(files, mods, stamp)
	e := edgeBetween(g, "cli/run.go#run", "blast/index.go#New", model.RelationCalls)
	if e == nil {
		t.Fatalf("a package selector must resolve inside the target package; got %+v", g.Edges)
	}
	if e.Confidence != model.ConfidenceExtracted {
		t.Errorf("confidence = %q, want extracted: the package names the target", e.Confidence)
	}
	if edgeBetween(g, "cli/run.go#run", "other/thing.go#New", model.RelationCalls) != nil {
		t.Error("a same-named symbol outside the target package must never be reached")
	}
}

func TestGraphSelectorSkipsMethodsAndTestFiles(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	method := model.Node{
		ID: "blast/index.go#T.New", Name: "New", Kind: "method", Owner: "T",
		Path: "blast/index.go", BodyHash: "h", Exported: true,
	}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "New", Receiver: "blast", File: "cli/run.go",
			}},
		),
		// Only a method of that name, plus a function of that name in a test
		// file. An importer sees neither.
		result("blast/index.go", []model.Node{fileNode("blast/index.go"), method}, nil),
		result("blast/index_test.go", []model.Node{fileNode("blast/index_test.go"), fn("blast/index_test.go", "New", true)}, nil),
	}

	g := resolve.Graph(files, mods, stamp)
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("neither a method nor a test-file symbol is a selector candidate; got %+v", e)
		}
	}
}

func TestGraphResolvesAnInRepoImportToANonTestRepresentative(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		result("cli/run.go",
			[]model.Node{fileNode("cli/run.go")},
			[]extract.RawEdge{{
				Source: "cli/run.go", Relation: model.RelationImports,
				Specifier: "example.com/repo/blast", File: "cli/run.go",
			}},
		),
		// Sorted first by id, and invisible to an importer.
		result("blast/a_test.go", []model.Node{fileNode("blast/a_test.go")}, nil),
		result("blast/index.go", []model.Node{fileNode("blast/index.go")}, nil),
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go", "blast/index.go", model.RelationImports) == nil {
		t.Fatalf("the representative must skip test files; got %+v", g.Edges)
	}
}

func TestGraphKeepsAnExternalImportAsAString(t *testing.T) {
	files := []extract.Result{result("a.go",
		[]model.Node{fileNode("a.go")},
		[]extract.RawEdge{{
			Source: "a.go", Relation: model.RelationImports, Specifier: "fmt", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil, stamp)
	e := edgeBetween(g, "a.go", "fmt", model.RelationImports)
	if e == nil {
		t.Fatalf("an external import keeps its package path as the target; got %+v", g.Edges)
	}
	// blast keeps this as a hit without a node, pagerank drops the edge. That
	// divergence is G1's ruling and stays.
	if e.Confidence != model.ConfidenceExtracted {
		t.Errorf("confidence = %q, want extracted: pointing outward is a fact, not a doubt", e.Confidence)
	}
}

func TestGraphEmitsNoCallEdgeForAnExternalSelector(t *testing.T) {
	files := []extract.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "f", false)},
		[]extract.RawEdge{{
			Source: "a.go#f", Relation: model.RelationCalls,
			Name: "Println", Receiver: "fmt", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil, stamp)
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("a call into a package outside the repository has no node to point at; got %+v", e)
		}
	}
}

func TestGraphResolvesASelectorIntoTheRootPackage(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Root", Receiver: "repo", File: "cli/run.go",
			}},
		),
		// A file at the repository root: its directory is "" and not ".", or it
		// could never be found by the import path of the root module.
		extract.Result{
			Path: "root.go", Language: "go", Package: "repo",
			Nodes: []model.Node{fileNode("root.go"), fn("root.go", "Root", true)},
		},
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go#run", "root.go#Root", model.RelationCalls) == nil {
		t.Fatalf("the root package must be reachable; got %+v", g.Edges)
	}
}

func TestGraphBindsAPlainImportByThePackageClauseAndNotThePathTail(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo/yaml.v3"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Marshal", Receiver: "yaml", File: "cli/run.go",
			}},
		),
		// The directory's last segment is "yaml.v3", the clause is "yaml", and
		// Go binds the clause. Guessing the path tail would drop this call --
		// and versioned module paths make the case ordinary, not exotic.
		extract.Result{
			Path: "yaml.v3/marshal.go", Language: "go", Package: "yaml",
			Nodes: []model.Node{fileNode("yaml.v3/marshal.go"), fn("yaml.v3/marshal.go", "Marshal", true)},
		},
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go#run", "yaml.v3/marshal.go#Marshal", model.RelationCalls) == nil {
		t.Fatalf("a plain import binds the target's package clause; got %+v", g.Edges)
	}
}

func TestGraphIgnoresABlankAndADotImportForASelector(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{
				{Alias: "_", Path: "example.com/repo/driver"},
				{Alias: ".", Path: "example.com/repo/dsl"},
			},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Open", Receiver: "driver", File: "cli/run.go",
			}},
		),
		extract.Result{
			Path: "driver/driver.go", Language: "go", Package: "driver",
			Nodes: []model.Node{fileNode("driver/driver.go"), fn("driver/driver.go", "Open", true)},
		},
	}

	g := resolve.Graph(files, mods, stamp)
	// Neither binds a selector name. A `driver.Open` in a file that only
	// blank-imports driver is some other driver entirely.
	for _, e := range g.Edges {
		if e.Relation == model.RelationCalls {
			t.Errorf("a blank import binds no selector; got %+v", e)
		}
	}
}

func TestGraphCarriesContainsThroughAndStampsTheExtractor(t *testing.T) {
	files := []extract.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "f", false)},
		[]extract.RawEdge{{
			Source: "a.go", Relation: model.RelationContains, TargetID: "a.go#f", File: "a.go",
		}},
	)}

	g := resolve.Graph(files, nil, stamp)
	e := edgeBetween(g, "a.go", "a.go#f", model.RelationContains)
	if e == nil || e.Confidence != model.ConfidenceExtracted {
		t.Fatalf("contains is already resolved and certain; got %+v", g.Edges)
	}
	if g.Meta.Extractor != stamp {
		t.Errorf("Meta.Extractor = %q, want the stamp the caller named, %q", g.Meta.Extractor, stamp)
	}
	if len(g.Meta.Languages) != 1 || g.Meta.Languages[0] != "go" {
		t.Errorf("Meta.Languages = %v, want [go]", g.Meta.Languages)
	}
	if g.Meta.Version == 0 || g.Meta.NodeCount != len(g.Nodes) || g.Meta.EdgeCount != len(g.Edges) {
		t.Errorf("meta = %+v, want the counts of the graph it describes", g.Meta)
	}
	if err := g.Validate(); err != nil {
		t.Errorf("a resolved graph must validate: %v", err)
	}
}

func TestGraphResolvesASelectorIntoANestedModule(t *testing.T) {
	mods := []resolve.Module{
		// Longest first, as Modules itself sorts them: the nested module of a
		// monorepo must win over its parent, and a call into the parent's own
		// package must skip past this one instead of matching it by prefix.
		{Dir: "tools", Path: "example.com/repo/tools"},
		{Dir: "", Path: "example.com/repo"},
	}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo/tools"}, {Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{
				{Source: "cli/run.go#run", Relation: model.RelationCalls, Name: "Root", Receiver: "tools", File: "cli/run.go"},
				{Source: "cli/run.go#run", Relation: model.RelationCalls, Name: "New", Receiver: "blast", File: "cli/run.go"},
			},
		),
		// The nested module's root package: importDir must land exactly on
		// its Dir, not Dir plus a spurious "/" + "".
		extract.Result{Path: "tools/root.go", Language: "go", Package: "tools", Nodes: []model.Node{fileNode("tools/root.go"), fn("tools/root.go", "Root", true)}},
		// The parent module's own package, reached only because the nested
		// module's prefix does not match this import path -- exercising the
		// "continue to the next module" arm.
		result("blast/index.go", []model.Node{fileNode("blast/index.go"), fn("blast/index.go", "New", true)}, nil),
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go#run", "tools/root.go#Root", model.RelationCalls) == nil {
		t.Fatalf("a selector into a nested module's root package must resolve; got %+v", g.Edges)
	}
	if edgeBetween(g, "cli/run.go#run", "blast/index.go#New", model.RelationCalls) == nil {
		t.Fatalf("a selector into the parent module must still resolve past the nested one; got %+v", g.Edges)
	}
}

func TestGraphResolvesASelectorIntoANestedModuleSubpackage(t *testing.T) {
	mods := []resolve.Module{
		{Dir: "tools", Path: "example.com/repo/tools"},
		{Dir: "", Path: "example.com/repo"},
	}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo/tools/gen"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Run", Receiver: "gen", File: "cli/run.go",
			}},
		),
		// A subpackage of the nested module: importDir must join the module's
		// Dir with the remaining path segment.
		extract.Result{Path: "tools/gen/run.go", Language: "go", Package: "gen", Nodes: []model.Node{fileNode("tools/gen/run.go"), fn("tools/gen/run.go", "Run", true)}},
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go#run", "tools/gen/run.go#Run", model.RelationCalls) == nil {
		t.Fatalf("a selector into a nested module's subpackage must resolve; got %+v", g.Edges)
	}
}

func TestGraphKeepsAnImportAsAStringWhenTheModulePackageIsUnindexed(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		result("cli/run.go",
			[]model.Node{fileNode("cli/run.go")},
			[]extract.RawEdge{{
				Source: "cli/run.go", Relation: model.RelationImports,
				// Inside the repository's own module, but this run's file set
				// never reached that directory -- an unscanned package, not an
				// external dependency.
				Specifier: "example.com/repo/unscanned", File: "cli/run.go",
			}},
		),
	}

	g := resolve.Graph(files, mods, stamp)
	e := edgeBetween(g, "cli/run.go", "example.com/repo/unscanned", model.RelationImports)
	if e == nil {
		t.Fatalf("an in-module import with nothing indexed there keeps its path as the target; got %+v", g.Edges)
	}
}

func TestGraphResolvesASelectorThroughAnExplicitAlias(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Alias: "b", Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "New", Receiver: "b", File: "cli/run.go",
			}},
		),
		result("blast/index.go", []model.Node{fileNode("blast/index.go"), fn("blast/index.go", "New", true)}, nil),
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go#run", "blast/index.go#New", model.RelationCalls) == nil {
		t.Fatalf("an explicit alias must bind the selector outright; got %+v", g.Edges)
	}
}

func TestGraphSkipsAnUnresolvablePlainImportWhileLookingForTheClause(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		importing("cli/run.go",
			// fmt resolves to no module of the repository, so packageDir must
			// pass over it and keep looking among the remaining plain imports.
			[]extract.Import{{Path: "fmt"}, {Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "New", Receiver: "blast", File: "cli/run.go",
			}},
		),
		result("blast/index.go", []model.Node{fileNode("blast/index.go"), fn("blast/index.go", "New", true)}, nil),
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go#run", "blast/index.go#New", model.RelationCalls) == nil {
		t.Fatalf("packageDir must skip an unresolvable plain import and find the next; got %+v", g.Edges)
	}
}

func TestGraphSortsEdgesBySourceThenRelationThenTarget(t *testing.T) {
	files := []extract.Result{
		result("a.go",
			[]model.Node{fileNode("a.go"), fn("a.go", "caller", false), fn("a.go", "helperA", false), fn("a.go", "helperB", false)},
			[]extract.RawEdge{
				{Source: "a.go", Relation: model.RelationContains, TargetID: "a.go#caller", File: "a.go"},
				// Same source as the contains edge above ("a.go" is both the
				// container and the importer): exercises the relation branch
				// of the comparator, not just the source branch.
				{Source: "a.go", Relation: model.RelationImports, Specifier: "fmt", File: "a.go"},
				{Source: "a.go#caller", Relation: model.RelationCalls, Name: "helperB", File: "a.go"},
				{Source: "a.go#caller", Relation: model.RelationCalls, Name: "helperA", File: "a.go"},
			},
		),
		result("b.go", []model.Node{fileNode("b.go")}, nil),
	}

	g := resolve.Graph(files, nil, stamp)
	var got []string
	for _, e := range g.Edges {
		got = append(got, string(e.Source)+"|"+string(e.Relation)+"|"+string(e.Target))
	}
	want := []string{
		"a.go|contains|a.go#caller",
		"a.go|imports|fmt",
		"a.go#caller|calls|a.go#helperA",
		"a.go#caller|calls|a.go#helperB",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestGraphSortsEdgesByRelationBeforeTarget(t *testing.T) {
	files := []extract.Result{
		result("a.go",
			[]model.Node{fileNode("a.go")},
			[]extract.RawEdge{
				// Same source, different relation, and the TARGET order runs
				// the opposite way from the RELATION order ("aaa" < "z.go#Sym"
				// but "contains" < "imports"): only sorting by relation before
				// target gets this right.
				{Source: "a.go", Relation: model.RelationContains, TargetID: "z.go#Sym", File: "a.go"},
				{Source: "a.go", Relation: model.RelationImports, Specifier: "aaa", File: "a.go"},
			},
		),
	}

	g := resolve.Graph(files, nil, stamp)
	if len(g.Edges) != 2 {
		t.Fatalf("got %d edges, want 2: %+v", len(g.Edges), g.Edges)
	}
	if g.Edges[0].Relation != model.RelationContains || g.Edges[1].Relation != model.RelationImports {
		t.Fatalf("edges sorted by target ahead of relation; got %+v", g.Edges)
	}
}

func TestGraphKeepsThePackageClauseFromTheNonTestFileWhateverOrderTheFilesArriveIn(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		extract.Result{
			Path: "pkg/foo.go", Language: "go", Package: "foo",
			Nodes: []model.Node{fileNode("pkg/foo.go"), fn("pkg/foo.go", "Fn", true)},
		},
		// Indexed AFTER the real file: "package foo_test" must never become
		// the clause an importer binds, whatever order the files arrive in.
		extract.Result{
			Path: "pkg/foo_test.go", Language: "go", Package: "foo_test",
			Nodes: []model.Node{fileNode("pkg/foo_test.go")},
		},
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo/pkg"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "Fn", Receiver: "foo", File: "cli/run.go",
			}},
		),
	}

	g := resolve.Graph(files, mods, stamp)
	if edgeBetween(g, "cli/run.go#run", "pkg/foo.go#Fn", model.RelationCalls) == nil {
		t.Fatalf("the package clause must come from the non-test file; got %+v", g.Edges)
	}
}

func TestGraphResolvesASameFileCallToAnEarlierFunctionInTheFile(t *testing.T) {
	files := []extract.Result{result("a.go",
		[]model.Node{fileNode("a.go"), fn("a.go", "A", false), fn("a.go", "B", false), fn("a.go", "C", false)},
		[]extract.RawEdge{{Source: "a.go#C", Relation: model.RelationCalls, Name: "A", File: "a.go"}},
	)}

	g := resolve.Graph(files, nil, stamp)
	e := edgeBetween(g, "a.go#C", "a.go#A", model.RelationCalls)
	if e == nil {
		t.Fatalf("edge missing; got %+v", g.Edges)
	}
	// A is the FIRST function indexed for this file; a per-file index that
	// forgets everything but the last function indexed would miss it and fall
	// back to the (also unique, here) global match, downgrading it to inferred.
	if e.Confidence != model.ConfidenceExtracted {
		t.Errorf("confidence = %q, want extracted: A is in the same file as the caller", e.Confidence)
	}
}

func TestGraphResolvesASelectorToTheFirstIndexedFunctionOfThePackage(t *testing.T) {
	mods := []resolve.Module{{Dir: "", Path: "example.com/repo"}}
	files := []extract.Result{
		importing("cli/run.go",
			[]extract.Import{{Path: "example.com/repo/pkg"}},
			[]model.Node{fileNode("cli/run.go"), fn("cli/run.go", "run", false)},
			[]extract.RawEdge{{
				Source: "cli/run.go#run", Relation: model.RelationCalls,
				Name: "First", Receiver: "pkg", File: "cli/run.go",
			}},
		),
		extract.Result{
			Path: "pkg/a.go", Language: "go", Package: "pkg",
			Nodes: []model.Node{fileNode("pkg/a.go"), fn("pkg/a.go", "First", true)},
		},
		extract.Result{
			Path: "pkg/b.go", Language: "go", Package: "pkg",
			Nodes: []model.Node{fileNode("pkg/b.go"), fn("pkg/b.go", "Second", true)},
		},
	}

	g := resolve.Graph(files, mods, stamp)
	// First is indexed before Second: a per-directory index that forgets
	// everything but the last function indexed would lose it.
	if edgeBetween(g, "cli/run.go#run", "pkg/a.go#First", model.RelationCalls) == nil {
		t.Fatalf("a package selector must still reach the first-indexed function of the package; got %+v", g.Edges)
	}
}

func TestGraphKeepsLanguagesApart(t *testing.T) {
	files := []extract.Result{
		result("a.go",
			[]model.Node{fileNode("a.go"), fn("a.go", "caller", false)},
			[]extract.RawEdge{{Source: "a.go#caller", Relation: model.RelationCalls, Name: "Run", File: "a.go"}},
		),
		result("b.go", []model.Node{fileNode("b.go"), fn("b.go", "Run", false)}, nil),
		// A language this package has no rules for: Go and Python both have
		// theirs, and this one stands for any language added to the build
		// before its resolver is.
		{
			Path: "tool.other", Language: "other",
			Nodes: []model.Node{fileNode("tool.other"), fn("tool.other", "Run", false), fn("tool.other", "main", false)},
			Edges: []extract.RawEdge{
				{Source: "tool.other", Relation: model.RelationContains, TargetID: "tool.other#Run", File: "tool.other"},
				{Source: "tool.other#main", Relation: model.RelationCalls, Name: "Run", File: "tool.other"},
			},
		},
	}

	g := resolve.Graph(files, nil, stamp)
	// Run exists twice, once per language. One index across both would see two
	// candidates and drop the Go call; each language has its own.
	e := edgeBetween(g, "a.go#caller", "b.go#Run", model.RelationCalls)
	if e == nil {
		t.Fatalf("the Go call must resolve against the Go files alone; got %+v", g.Edges)
	}
	if e.Confidence != model.ConfidenceInferred {
		t.Errorf("confidence = %q, want inferred: one match across Go files", e.Confidence)
	}
	// A language without resolution rules keeps its nodes and the containment
	// its extractor already resolved, and nothing it would have to guess.
	if c := edgeBetween(g, "tool.other", "tool.other#Run", model.RelationContains); c == nil || c.Confidence != model.ConfidenceExtracted {
		t.Errorf("contains of a language without rules must be carried through; got %+v", g.Edges)
	}
	for _, e := range g.Edges {
		if e.Source == "tool.other#main" {
			t.Errorf("a call of a language without rules must yield no edge; got %+v", e)
		}
	}
	if len(g.Nodes) != 7 {
		t.Errorf("got %d nodes, want the 7 of both languages", len(g.Nodes))
	}
	if !reflect.DeepEqual(g.Meta.Languages, []string{"go", "other"}) {
		t.Errorf("Meta.Languages = %v, want [go other]", g.Meta.Languages)
	}
	if err := g.Validate(); err != nil {
		t.Errorf("a graph of two languages must validate: %v", err)
	}
}

func TestEmptyGraphHasNoLanguages(t *testing.T) {
	g := resolve.Graph(nil, nil, stamp)
	// An empty list and not nil: nil would reach wiring.json as null.
	if g.Meta.Languages == nil || len(g.Meta.Languages) != 0 {
		t.Fatalf("Meta.Languages = %#v, want an empty list", g.Meta.Languages)
	}
	b, err := json.Marshal(g.Meta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"languages":[]`) {
		t.Errorf("meta = %s, want an empty languages list", b)
	}
}

func TestGraphSortsNodesAndEdges(t *testing.T) {
	files := []extract.Result{
		result("b.go", []model.Node{fileNode("b.go")}, nil),
		result("a.go", []model.Node{fileNode("a.go")}, nil),
	}

	g := resolve.Graph(files, nil, stamp)
	// Byte order, so a rebuild of an unchanged tree is byte-identical and the
	// golden files do not wander between platforms.
	if g.Nodes[0].ID != "a.go" || g.Nodes[1].ID != "b.go" {
		t.Fatalf("nodes unsorted: %+v", g.Nodes)
	}
}
