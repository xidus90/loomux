package resolve_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
)

func method(rel, owner, name, sig string) model.Node {
	return model.Node{
		ID: model.NodeID(rel + "#" + owner + "." + name), Name: name, Kind: "method", Owner: owner,
		Path: rel, BodyHash: "h", Exported: true, Signature: sig,
	}
}

func signed(n model.Node, sig string) model.Node {
	n.Signature = sig
	return n
}

var repoMod = []resolve.Module{{Dir: "", Path: "example.com/repo"}}

func TestGraphResolvesAReferenceInsideItsOwnPackageOnly(t *testing.T) {
	files := []extract.Result{
		result("a/wire.go",
			[]model.Node{fileNode("a/wire.go"), fn("a/wire.go", "wire", false)},
			[]extract.RawEdge{
				{Source: "a/wire.go#wire", Relation: model.RelationReferences, Name: "target", File: "a/wire.go"},
				{Source: "a/wire.go#wire", Relation: model.RelationReferences, Name: "elsewhere", File: "a/wire.go"},
			}),
		result("a/target.go", []model.Node{fileNode("a/target.go"), fn("a/target.go", "target", false)}, nil),
		// Unique repo-wide, but in another package: a bare name never means it.
		result("b/other.go", []model.Node{fileNode("b/other.go"), fn("b/other.go", "elsewhere", false)}, nil),
	}
	g := resolve.Graph(files, repoMod, stamp)
	e := edgeBetween(g, "a/wire.go#wire", "a/target.go#target", model.RelationReferences)
	if e == nil || e.Confidence != model.ConfidenceExtracted {
		t.Fatalf("a same-package reference must resolve as extracted; got %+v in %+v", e, g.Edges)
	}
	if edgeBetween(g, "a/wire.go#wire", "b/other.go#elsewhere", model.RelationReferences) != nil {
		t.Error("a bare reference must never resolve into another package")
	}
}

func TestGraphResolvesNoBareReferenceFromAnExternalTestPackage(t *testing.T) {
	ext := result("a/wire_test.go", []model.Node{fileNode("a/wire_test.go"), fn("a/wire_test.go", "TestWire", true)},
		[]extract.RawEdge{{Source: "a/wire_test.go#TestWire", Relation: model.RelationReferences, Name: "target", File: "a/wire_test.go"}})
	ext.Package = "a_test"
	files := []extract.Result{
		ext,
		result("a/target.go", []model.Node{fileNode("a/target.go"), fn("a/target.go", "target", false)}, nil),
	}
	g := resolve.Graph(files, repoMod, stamp)
	// `package a_test` sees package a only through its import, so a bare
	// name there can never mean a's target.
	if e := edgeBetween(g, "a/wire_test.go#TestWire", "a/target.go#target", model.RelationReferences); e != nil {
		t.Errorf("an external test package must not reach its package by a bare name; got %+v", e)
	}
}

func TestGraphResolvesASameFileReferenceFirst(t *testing.T) {
	files := []extract.Result{
		result("a/wire.go",
			[]model.Node{fileNode("a/wire.go"), fn("a/wire.go", "wire", false), fn("a/wire.go", "target", false)},
			[]extract.RawEdge{{Source: "a/wire.go#wire", Relation: model.RelationReferences, Name: "target", File: "a/wire.go"}}),
		// A same-named function in another file of the package must not make
		// the same-file one ambiguous.
		result("a/other.go", []model.Node{fileNode("a/other.go"), fn("a/other.go", "target", false)}, nil),
	}
	g := resolve.Graph(files, repoMod, stamp)
	if edgeBetween(g, "a/wire.go#wire", "a/wire.go#target", model.RelationReferences) == nil {
		t.Errorf("a same-file reference must win; got %+v", g.Edges)
	}
}

func TestGraphResolvesAChainedCallOnATwoParameterGenericResult(t *testing.T) {
	files := []extract.Result{
		result("a/use.go", []model.Node{
			fileNode("a/use.go"), fn("a/use.go", "use", false),
			signed(fn("a/use.go", "pair", false), "func pair() Pair[int, string]"),
			method("a/use.go", "Pair", "Left", ""),
		}, []extract.RawEdge{{
			Source: "a/use.go#use", Relation: model.RelationCalls, Name: "Left", File: "a/use.go",
			Via: &extract.RawEdge{Source: "a/use.go#use", Relation: model.RelationCalls, Name: "pair", File: "a/use.go"},
		}}),
	}
	g := resolve.Graph(files, repoMod, stamp)
	if edgeBetween(g, "a/use.go#use", "a/use.go#Pair.Left", model.RelationCalls) == nil {
		t.Errorf("a two-parameter generic result must type the chain; got %+v", g.Edges)
	}
}

func TestGraphResolvesAReferenceThroughASelectorAndAnOwner(t *testing.T) {
	files := []extract.Result{
		importing("cli/hook.go", []extract.Import{{Path: "example.com/repo/hooks"}},
			[]model.Node{fileNode("cli/hook.go"), fn("cli/hook.go", "wire", false), method("cli/hook.go", "S", "handle", "")},
			[]extract.RawEdge{
				{Source: "cli/hook.go", Relation: model.RelationReferences, Name: "PostToolUse", Receiver: "hooks", File: "cli/hook.go"},
				{Source: "cli/hook.go#wire", Relation: model.RelationReferences, Name: "handle", Owner: "S", File: "cli/hook.go"},
			}),
		result("hooks/post.go", []model.Node{fileNode("hooks/post.go"), fn("hooks/post.go", "PostToolUse", true)}, nil),
	}
	g := resolve.Graph(files, repoMod, stamp)
	if edgeBetween(g, "cli/hook.go", "hooks/post.go#PostToolUse", model.RelationReferences) == nil {
		t.Errorf("a package selector as a value must resolve; got %+v", g.Edges)
	}
	if edgeBetween(g, "cli/hook.go#wire", "cli/hook.go#S.handle", model.RelationReferences) == nil {
		t.Errorf("a method value must resolve through its owner; got %+v", g.Edges)
	}
}

func TestGraphResolvesAChainedCallThroughTheInnerResultType(t *testing.T) {
	files := []extract.Result{
		importing("hooks/aside.go", []extract.Import{{Path: "example.com/repo/blast"}},
			[]model.Node{fileNode("hooks/aside.go"), fn("hooks/aside.go", "aside", false)},
			[]extract.RawEdge{{
				Source: "hooks/aside.go#aside", Relation: model.RelationCalls, Name: "Reach", File: "hooks/aside.go",
				Via: &extract.RawEdge{Source: "hooks/aside.go#aside", Relation: model.RelationCalls, Name: "New", Receiver: "blast", File: "hooks/aside.go"},
			}}),
		result("blast/index.go", []model.Node{
			fileNode("blast/index.go"),
			signed(fn("blast/index.go", "New", true), "func New(g *model.Graph) *Index"),
			method("blast/index.go", "Index", "Reach", "func (x *Index) Reach(start []model.NodeID) []Hit"),
		}, nil),
		// The same owner and method in another package must not compete.
		result("other/index.go", []model.Node{fileNode("other/index.go"), method("other/index.go", "Index", "Reach", "")}, nil),
	}
	g := resolve.Graph(files, repoMod, stamp)
	e := edgeBetween(g, "hooks/aside.go#aside", "blast/index.go#Index.Reach", model.RelationCalls)
	if e == nil || e.Confidence != model.ConfidenceExtracted {
		t.Fatalf("a chained call must resolve on the inner call's result type; got %+v in %+v", e, g.Edges)
	}
}

func TestGraphTakesTheWeakestLinkOfAChainedCall(t *testing.T) {
	files := []extract.Result{
		result("a/use.go", []model.Node{fileNode("a/use.go"), fn("a/use.go", "use", false)},
			[]extract.RawEdge{{
				Source: "a/use.go#use", Relation: model.RelationCalls, Name: "Get", File: "a/use.go",
				// A bare name only the repo-wide fallback finds: inferred.
				Via: &extract.RawEdge{Source: "a/use.go#use", Relation: model.RelationCalls, Name: "open", File: "a/use.go"},
			}}),
		result("b/store.go", []model.Node{
			fileNode("b/store.go"),
			signed(fn("b/store.go", "open", false), "func open[K comparable](p string) (*Store[K], error)"),
			method("b/store.go", "Store", "Get", ""),
		}, nil),
	}
	g := resolve.Graph(files, repoMod, stamp)
	e := edgeBetween(g, "a/use.go#use", "b/store.go#Store.Get", model.RelationCalls)
	if e == nil || e.Confidence != model.ConfidenceInferred {
		t.Fatalf("a chain through an inferred inner call is inferred; got %+v in %+v", e, g.Edges)
	}
}

func TestGraphDropsAChainedCallItCannotType(t *testing.T) {
	via := func(name string) *extract.RawEdge {
		return &extract.RawEdge{Source: "a/use.go#use", Relation: model.RelationCalls, Name: name, File: "a/use.go"}
	}
	files := []extract.Result{
		result("a/use.go", []model.Node{
			fileNode("a/use.go"), fn("a/use.go", "use", false),
			signed(fn("a/use.go", "foreign", false), "func foreign() *other.T"),
			signed(fn("a/use.go", "none", false), "func none()"),
			signed(fn("a/use.go", "empty", false), "func empty() ()"),
			signed(fn("a/use.go", "broken", false), "func broken( {"),
			method("a/use.go", "T", "M", ""),
		}, []extract.RawEdge{
			{Source: "a/use.go#use", Relation: model.RelationCalls, Name: "M", File: "a/use.go", Via: via("foreign")},
			{Source: "a/use.go#use", Relation: model.RelationCalls, Name: "M", File: "a/use.go", Via: via("none")},
			{Source: "a/use.go#use", Relation: model.RelationCalls, Name: "M", File: "a/use.go", Via: via("empty")},
			{Source: "a/use.go#use", Relation: model.RelationCalls, Name: "M", File: "a/use.go", Via: via("broken")},
			{Source: "a/use.go#use", Relation: model.RelationCalls, Name: "M", File: "a/use.go", Via: via("missing")},
		}),
	}
	g := resolve.Graph(files, repoMod, stamp)
	if e := edgeBetween(g, "a/use.go#use", "a/use.go#T.M", model.RelationCalls); e != nil {
		t.Errorf("a result in another package, no result, an unparsable signature or an unresolved inner call types nothing; got %+v", e)
	}
}
