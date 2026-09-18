package ask_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
)

// symbol is a node with the fields the ranking reads.
func symbol(id, name, path, body string) model.Node {
	return model.Node{
		ID: model.NodeID(id), Name: name, Kind: "function", Path: path,
		Span: "L1-L4", Signature: "func " + name + "()", BodyHash: "h", BodyText: body,
	}
}

func edge(from, to string) model.Edge {
	return model.Edge{
		Source: model.NodeID(from), Target: model.NodeID(to),
		Relation: model.RelationCalls, Confidence: model.ConfidenceExtracted,
	}
}

// world builds a graph and its sidecar together, the way a build does.
func world(nodes []model.Node, edges []model.Edge) (*model.Graph, *lexicon.Index) {
	g := &model.Graph{
		Meta:  model.Meta{Version: 2, Extractor: "go/1"},
		Nodes: nodes, Edges: edges,
	}
	return g, lexicon.Build(g)
}

func idsOf(a ask.Answer) []model.NodeID {
	var out []model.NodeID
	for _, h := range a.Hits {
		out = append(out, h.ID)
	}
	return out
}

func TestRunRanksAWordMatchByName(t *testing.T) {
	g, ix := world([]model.Node{
		symbol("a.go#ParseConfig", "ParseConfig", "a.go", "read the file"),
		symbol("b.go#Unrelated", "Unrelated", "b.go", "nothing here"),
	}, nil)

	got := ask.Run(g, ix, "parse config", ask.Options{})
	if len(got.Hits) == 0 || got.Hits[0].ID != "a.go#ParseConfig" {
		t.Fatalf("got %v, want ParseConfig first", idsOf(got))
	}
	// The top hit is normalized to 1 on the lexical axis before the graph
	// weight is added, so a score is comparable between queries.
	if got.Hits[0].Lexical <= 0 {
		t.Errorf("lexical = %v, want a positive score", got.Hits[0].Lexical)
	}
	if got.Note != "" {
		t.Errorf("Note = %q, want empty when hits are present", got.Note)
	}
}

func TestRunLetsConnectivityBreakALexicalNearTie(t *testing.T) {
	// Two nodes share the query word. One sits in the cluster the question is
	// about, the other is isolated -- the window "overlay" against the scroll
	// "overlay", which is the collision graph rank exists to fix.
	nodes := []model.Node{
		symbol("core/overlay.go#Overlay", "Overlay", "core/overlay.go", "the real one"),
		symbol("misc/overlay.go#Overlay", "Overlay", "misc/overlay.go", "the isolated one"),
		symbol("core/a.go#A", "A", "core/a.go", "x"),
		symbol("core/b.go#B", "B", "core/b.go", "y"),
	}
	edges := []model.Edge{
		edge("core/a.go#A", "core/overlay.go#Overlay"),
		edge("core/b.go#B", "core/overlay.go#Overlay"),
	}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "overlay", ask.Options{})
	if len(got.Hits) < 2 {
		t.Fatalf("got %v, want both", idsOf(got))
	}
	if got.Hits[0].ID != "core/overlay.go#Overlay" {
		t.Fatalf("got %v, want the wired-in one first", idsOf(got))
	}
	if got.Hits[0].Graph <= got.Hits[1].Graph {
		t.Errorf("graph scores %v and %v: the connected node must carry more mass",
			got.Hits[0].Graph, got.Hits[1].Graph)
	}
}

func TestRunDoesNotLetConnectivityOverruleAClearLexicalWinner(t *testing.T) {
	// GRAPH_WEIGHT is 0.5: connectivity reorders near-ties, it does not
	// overturn a node the query plainly names.
	//
	// That sentence is the intent, not what this test pins: it answers with
	// exactly ONE hit, because the hub cluster matches no query word and is
	// never seeded, so it holds at any weight. The weight itself is pinned by
	// TestRunWeighsTheGraphAxisAtHalfOfTheLexicalOne, which derives a literal
	// blended score. This test stays as documentation of the intent.
	nodes := []model.Node{
		symbol("a.go#ParseConfigFile", "ParseConfigFile", "a.go", "parse config file"),
		symbol("hub/hub.go#Hub", "Hub", "hub/hub.go", "unrelated"),
		symbol("hub/x.go#X", "X", "hub/x.go", "unrelated"),
		symbol("hub/y.go#Y", "Y", "hub/y.go", "unrelated"),
	}
	edges := []model.Edge{
		edge("hub/x.go#X", "hub/hub.go#Hub"),
		edge("hub/y.go#Y", "hub/hub.go#Hub"),
	}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "parse config file", ask.Options{})
	if got.Hits[0].ID != "a.go#ParseConfigFile" {
		t.Fatalf("got %v, want the lexical winner first", idsOf(got))
	}
}

func TestRunRescuesACentralNodeTheQueryNeverNamed(t *testing.T) {
	// RESCUE_FLOOR is 0.15: a node the query never word-matched joins the
	// results when the walk gives it at least 15% of the top node's mass. This
	// is what surfaces the helper a task depends on but did not mention.
	nodes := []model.Node{
		symbol("a.go#Seeded", "Seeded", "a.go", "seeded"),
		symbol("b.go#Helper", "Helper", "b.go", "nothing matching"),
	}
	edges := []model.Edge{edge("a.go#Seeded", "b.go#Helper")}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "seeded", ask.Options{})
	ids := idsOf(got)
	if len(ids) != 2 {
		t.Fatalf("got %v, want the seeded node and its rescued neighbour", ids)
	}
	var helper ask.Hit
	for _, h := range got.Hits {
		if h.ID == "b.go#Helper" {
			helper = h
		}
	}
	if helper.Lexical != 0 {
		t.Errorf("the rescued node matched no word; lexical = %v", helper.Lexical)
	}
	if helper.Graph <= 0 {
		t.Errorf("the rescued node is there on graph mass alone; graph = %v", helper.Graph)
	}
}

func TestRunDeRanksATestFile(t *testing.T) {
	nodes := []model.Node{
		symbol("lib/cache.go#Cache", "Cache", "lib/cache.go", "the implementation"),
		symbol("lib/cache_test.go#TestCache", "TestCache", "lib/cache_test.go", "the cache test"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "cache", ask.Options{})
	if got.Hits[0].ID != "lib/cache.go#Cache" {
		t.Fatalf("got %v, want the implementation first", idsOf(got))
	}
	// Still present -- "where are the tests" is a fair question -- just below.
	if len(got.Hits) != 2 {
		t.Errorf("got %v, want the test in the results as well", idsOf(got))
	}
}

func TestRunDeRanksATestEvenWhenItIsTheStrongestRawMatch(t *testing.T) {
	// The penalty applies to the raw score AND again after normalization. With
	// only the first, dividing by max would restore a winning test to 1.0 and
	// erase the de-rank; with only the second, the test would still seed the
	// walk as if it were the answer.
	nodes := []model.Node{
		symbol("lib/cache_test.go#TestCacheCacheCache", "TestCacheCacheCache", "lib/cache_test.go", "cache cache cache"),
		symbol("lib/cache.go#Cache", "Cache", "lib/cache.go", "implementation"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "cache", ask.Options{})
	if got.Hits[0].ID != "lib/cache.go#Cache" {
		t.Fatalf("got %v, want the source above the test; scores %+v", idsOf(got), got.Hits)
	}
}

func TestRunLiftsThePenaltyWhenTheQueryAsksForTests(t *testing.T) {
	nodes := []model.Node{
		symbol("lib/cache.go#Cache", "Cache", "lib/cache.go", "implementation"),
		symbol("lib/cache_test.go#TestCache", "TestCache", "lib/cache_test.go", "cache test"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "cache tests", ask.Options{})
	// A question about tests must answer with tests. Without this the ported
	// vector test/ask.test.ts:40 cannot pass.
	if got.Hits[0].ID != "lib/cache_test.go#TestCache" {
		t.Fatalf("got %v, want the test first for a test-seeking query", idsOf(got))
	}
}

func TestRunHonoursTheLimitAndDefaultsToEight(t *testing.T) {
	var nodes []model.Node
	for _, name := range []string{"CacheA", "CacheB", "CacheC", "CacheD", "CacheE", "CacheF", "CacheG", "CacheH", "CacheI", "CacheJ"} {
		nodes = append(nodes, symbol("p/"+name+".go#"+name, name, "p/"+name+".go", "cache"))
	}
	g, ix := world(nodes, nil)

	if got := ask.Run(g, ix, "cache", ask.Options{}); len(got.Hits) != 8 {
		t.Fatalf("got %d hits, want the default 8", len(got.Hits))
	}
	if got := ask.Run(g, ix, "cache", ask.Options{Limit: 3}); len(got.Hits) != 3 {
		t.Fatalf("got %d hits, want 3", len(got.Hits))
	}
}

func TestRunFiltersBeforeScoringAndSegmentAware(t *testing.T) {
	nodes := []model.Node{
		symbol("widgets/a.go#Render", "Render", "widgets/a.go", "render"),
		symbol("widgets-extra/b.go#Render", "Render", "widgets-extra/b.go", "render"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "render", ask.Options{In: "widgets"})
	// A prefix is a path prefix: "widgets" never matches "widgets-extra".
	if len(got.Hits) != 1 || got.Hits[0].ID != "widgets/a.go#Render" {
		t.Fatalf("got %v, want only the node under widgets/", idsOf(got))
	}
}

func TestRunNarrowsTheWalkAndNotOnlyTheSeed(t *testing.T) {
	// Without a filter on the walk itself, a neighbour outside the prefix could
	// clear the rescue floor and surface -- defeating "filters before scoring".
	nodes := []model.Node{
		symbol("in/a.go#Seeded", "Seeded", "in/a.go", "seeded"),
		symbol("out/b.go#Neighbour", "Neighbour", "out/b.go", "nothing matching"),
	}
	edges := []model.Edge{edge("in/a.go#Seeded", "out/b.go#Neighbour")}
	g, ix := world(nodes, edges)

	got := ask.Run(g, ix, "seeded", ask.Options{In: "in"})
	for _, h := range got.Hits {
		if h.ID == "out/b.go#Neighbour" {
			t.Fatalf("a neighbour outside the prefix must never surface; got %v", idsOf(got))
		}
	}
}

func TestRunOnAMissedQuerySaysSoInsteadOfAnEmptyList(t *testing.T) {
	g, ix := world([]model.Node{symbol("a.go#F", "F", "a.go", "x")}, nil)

	got := ask.Run(g, ix, "nothing here matches this", ask.Options{})
	if len(got.Hits) != 0 {
		t.Fatalf("got %v, want nothing", idsOf(got))
	}
	// A silent empty list reads as "there is no such code". A note says which
	// it was.
	if got.Note == "" {
		t.Error("a query that matched nothing must say so")
	}
}

func TestRunRanksAFileNodeByItsResidualText(t *testing.T) {
	// A file node is a document of the lexical pass -- that is how a word in an
	// import header finds its file -- and it is not a hit a reader can open at
	// a signature. It ranks, and it ranks as a file.
	g, ix := world([]model.Node{
		{
			ID: "lib/only.go", Name: "only.go", Kind: model.KindFile, Path: "lib/only.go",
			Span: "L1-L9", BodyHash: "h", BodyText: "package lib needle",
		},
	}, nil)

	got := ask.Run(g, ix, "needle", ask.Options{})
	if len(got.Hits) != 1 || got.Hits[0].ID != "lib/only.go" {
		t.Fatalf("got %v, want the file found by a word in its residual", idsOf(got))
	}
}

func TestRunIsDeterministicOnATie(t *testing.T) {
	nodes := []model.Node{
		symbol("b.go#Same", "Same", "b.go", "same"),
		symbol("a.go#Same", "Same", "a.go", "same"),
	}
	g, ix := world(nodes, nil)

	// Fifty runs, with the id assertion inside the loop. The candidate set is a
	// map, so a comparator that stops telling a tie apart leaves the order to
	// one range over it -- a coin flip per run, which five runs and a single
	// assertion outside them pass about three times in a hundred. Measured:
	// striking out the score comparison in the sort left the old shape of this
	// test green.
	first := ask.Run(g, ix, "same", ask.Options{})
	for i := 0; i < 50; i++ {
		again := ask.Run(g, ix, "same", ask.Options{})
		for j := range first.Hits {
			if first.Hits[j].ID != again.Hits[j].ID {
				t.Fatalf("run %d differs: %v against %v", i, idsOf(first), idsOf(again))
			}
		}
		// Equal score: the id decides, ascending, as in G1's ranking.
		if again.Hits[0].ID != "a.go#Same" {
			t.Fatalf("run %d: got %v, want the lower id first on a tie", i, idsOf(again))
		}
	}
}

func TestRunSkipsADocWhoseNodeIsGone(t *testing.T) {
	// The sidecar is derived data on disk beside the graph. A graph rewritten
	// without a rebuilt sidecar leaves documents behind that name no node, and
	// such a document must not become a hit: it has no path and no span, so
	// nothing a reader could open.
	g, ix := world([]model.Node{
		symbol("a.go#Kept", "Kept", "a.go", "needle"),
		symbol("b.go#Gone", "Gone", "b.go", "needle"),
	}, nil)
	g.Nodes = g.Nodes[:1]

	got := ask.Run(g, ix, "needle", ask.Options{})
	if len(got.Hits) != 1 || got.Hits[0].ID != "a.go#Kept" {
		t.Fatalf("got %v, want only the node the graph still has", idsOf(got))
	}
}

func TestRunNormalizesThePrefixTheWayLexiconDoes(t *testing.T) {
	// "lib/" is the form every shell's tab completion produces, and "lib\" the
	// form Windows produces. Unnormalized they matched nothing and the answer
	// reported zero documents without an error.
	nodes := []model.Node{
		symbol("lib/a.go#Render", "Render", "lib/a.go", "render"),
		symbol("other/b.go#Render", "Render", "other/b.go", "render"),
	}
	g, ix := world(nodes, nil)

	for _, prefix := range []string{"lib", "lib/", "/lib/", `lib\`} {
		got := ask.Run(g, ix, "render", ask.Options{In: prefix})
		if len(got.Hits) != 1 || got.Hits[0].ID != "lib/a.go#Render" {
			t.Errorf("In %q: got %v, want the one node under lib/", prefix, idsOf(got))
		}
	}
	// A prefix that normalizes away is no prefix: the whole repository answers.
	if got := ask.Run(g, ix, "render", ask.Options{In: "/"}); len(got.Hits) != 2 {
		t.Errorf("In %q: got %v, want the whole repository", "/", idsOf(got))
	}
}

// overlayWorld is the blend's discriminating world: one strong lexical match
// wired into nothing, one weaker match at the centre of a cycle.
//
// helpers many neighbours hang off the weaker node, each of them a word match
// of its own, so the walk's mass pools there. No node dangles, which matters
// because pagerank.Rank returns a dangling node's mass to the seeds weighted
// by their seed mass and would hand the heaviest seed -- the strong node --
// most of it back. The reason is pagerank.Prepare, not the edge list: it
// appends to -> from beside from -> to, so one weak -> h edge already gives h
// its neighbour. The explicit h -> weak edge only doubles that pair's
// multiplicity, since Prepare does not dedupe, and the assertions below hold
// without it.
func overlayWorld(helpers int) (*model.Graph, *lexicon.Index) {
	nodes := []model.Node{
		symbol("overlay/strong.go#Overlay", "Overlay", "overlay/strong.go", "the definition"),
		symbol("core/weak.go#Overlay", "Overlay", "core/weak.go", "nothing matching"),
	}
	edges := []model.Edge{edge("overlay/strong.go#Overlay", "core/weak.go#Overlay")}
	for i := 0; i < helpers; i++ {
		id := fmt.Sprintf("core/h%d.go#OverlayHelper%d", i, i)
		nodes = append(nodes, symbol(id, fmt.Sprintf("OverlayHelper%d", i), fmt.Sprintf("core/h%d.go", i), "nothing matching"))
		edges = append(edges, edge("core/weak.go#Overlay", id), edge(id, "core/weak.go#Overlay"))
	}
	return world(nodes, edges)
}

func TestRunWeighsTheGraphAxisAtHalfOfTheLexicalOne(t *testing.T) {
	// What the brief's two constant-bearing tests do NOT pin, measured:
	//
	//   TestRunDoesNotLetConnectivityOverruleAClearLexicalWinner answers with
	//   exactly ONE hit. "Hub" tokenizes to "hub", "X" and "Y" fall to the
	//   single-character gate, so no node of the hub cluster matches "parse
	//   config file", the cluster is never seeded, and Rank leaves unreached
	//   nodes out. That test holds at any graphWeight, 50 included.
	//
	//   TestRunLetsConnectivityBreakALexicalNearTie pins graphWeight >= 0.
	//   Hit.Graph is the raw walk score and does not depend on the weight; at
	//   weight 0 both blended scores are exactly 1.0 and the ascending-id
	//   tiebreak still puts core/... first.
	//
	// This world makes the blend decide. The query is one term, so its idf is
	// the same factor in every field and cancels out of the ratio:
	//
	//   overlay/strong.go#Overlay -- name {overlay}, path {overlay,strong,go}
	//     lexical = 3*w (name) + 2*w (path) = 5w, body holds no query term
	//   core/weak.go#Overlay      -- name {overlay}, path {core,weak,go}
	//     lexical = 3*w
	//   maxLex = 5w, so lexNorm is 1 for the strong node and 3/5 for the weak
	//
	// The weak node is the walk's maximum and normalizes to exactly 1; the
	// strong node keeps only its own restart share, measured 0.132388114 at
	// eight helpers. The two blended scores are therefore
	//
	//   weak   = 3/5 + 0.5*1           = 1.1
	//   strong = 1   + 0.5*0.132388114 = 1.066194057
	//
	// and the weaker match wins by 0.034. At graphWeight 0.4 the weak node
	// scores 1.0 against 1.052955 and the order flips; at 0.6 it scores 1.2 and
	// the literal below fails. Both directions discriminate.
	g, ix := overlayWorld(8)

	got := ask.Run(g, ix, "overlay", ask.Options{})
	if len(got.Hits) < 2 {
		t.Fatalf("got %v, want the whole cluster", idsOf(got))
	}
	if got.Hits[0].ID != "core/weak.go#Overlay" || got.Hits[1].ID != "overlay/strong.go#Overlay" {
		t.Fatalf("got %v, want the graph-heavy weaker match above the lexical winner", idsOf(got))
	}
	if math.Abs(got.Hits[1].Lexical-1.0) > 1e-9 {
		t.Errorf("strong lexical = %v, want exactly 1.0 (normalized)", got.Hits[1].Lexical)
	}
	if math.Abs(got.Hits[0].Lexical-0.6) > 1e-9 {
		t.Errorf("weak lexical = %v, want exactly 0.6 (3/5)", got.Hits[0].Lexical)
	}
	// Graph is v/max with v == max here, which is exactly 1.0 -- that is what
	// makes the literal above hand-derivable rather than read off a run.
	if got.Hits[0].Graph != 1 {
		t.Fatalf("graph = %v, want the walk maximum at exactly 1", got.Hits[0].Graph)
	}
	if math.Abs(got.Hits[0].Score-1.1) > 1e-12 {
		t.Errorf("score = %v, want 3/5 + 0.5*1 = 1.1 -- graphWeight is 0.5", got.Hits[0].Score)
	}
}

func TestRunNormalizesTheLexicalAxisByTheMaximumAndNotTheLastDocument(t *testing.T) {
	// The lexical axis is divided by the maximum of the corpus, and the maximum
	// is tracked with a comparison. Striking that comparison out leaves the
	// LAST document's score in the sidecar's id order in its place, which
	// rescales every lexical value while the graph axis stays fixed -- so the
	// blend changes without any score changing.
	//
	// Every other test of the blend happens to put its strongest match last in
	// id order ("overlay/..." sorts after "core/..."), where the two are the
	// same number. This world puts the strongest match FIRST: a.go matches the
	// query by name, worth 3, and z.go only in its body, worth about 1.3 of the
	// same idf.
	nodes := []model.Node{
		symbol("a.go#Cache", "Cache", "a.go", "nothing matching"),
		symbol("z.go#Store", "Store", "z.go", "cache cache lookup"),
	}
	g, ix := world(nodes, nil)

	got := ask.Run(g, ix, "cache", ask.Options{})

	if len(got.Hits) != 2 {
		t.Fatalf("got %v, want both documents", idsOf(got))
	}
	if got.Hits[0].ID != "a.go#Cache" {
		t.Fatalf("got %v, want the name match first", idsOf(got))
	}
	// The maximum normalizes to exactly 1 by definition, whatever the corpus.
	if got.Hits[0].Lexical != 1 {
		t.Errorf("lexical = %v, want exactly 1 -- the maximum is the divisor", got.Hits[0].Lexical)
	}
	if !(got.Hits[1].Lexical > 0 && got.Hits[1].Lexical < 1) {
		t.Errorf("lexical = %v, want a weaker match strictly between 0 and 1", got.Hits[1].Lexical)
	}
}

// rescueWorld is one seeded node fanning out to n neighbours the query never
// names.
//
// No neighbour is a sink: the walk meets the edges undirected, so each of them
// has exactly one neighbour -- the seed -- and pagerank's dangling pool is
// never touched here. What divides the mass is the fan-out itself.
func rescueWorld(neighbours int) (*model.Graph, *lexicon.Index) {
	nodes := []model.Node{symbol("a.go#Seeded", "Seeded", "a.go", "seeded")}
	var edges []model.Edge
	for i := 0; i < neighbours; i++ {
		id := fmt.Sprintf("h%d.go#Helper%d", i, i)
		nodes = append(nodes, symbol(id, fmt.Sprintf("Helper%d", i), fmt.Sprintf("h%d.go", i), "nothing matching"))
		edges = append(edges, edge("a.go#Seeded", id))
	}
	return world(nodes, edges)
}

func TestRunRescuesJustAboveTheFloorAndNotJustBelowIt(t *testing.T) {
	// What TestRunRescuesACentralNodeTheQueryNeverNamed does NOT pin, measured:
	// its single rescued helper carries 0.750988271 of the top node's mass,
	// five times the floor, so every rescueFloor up to 0.75 keeps that test
	// green -- the plan's own example mutant 0.15 -> 0.16 survives it.
	//
	// Fanning the same seed out to n neighbours divides that mass. The seed is
	// the only restart target and, the topology being undirected, every
	// neighbour's whole mass flows back to it, so the seed's own dynamics do
	// not depend on n and each neighbour lands at the one-neighbour mass over
	// n, measured:
	//
	//   n = 4   0.187747068
	//   n = 5   0.150197654  >  0.15   rescued
	//   n = 6   0.125164712  <  0.15   dropped
	//
	// The pair brackets the constant as 0.125164712 < rescueFloor <=
	// 0.150197654. Two honest limits of that bracket: a mutant downwards to
	// 0.14 is NOT caught, and at full convergence n = 5 would sit at exactly
	// 0.15 -- the 2.0e-4 of air above the floor exists only because
	// pagerank.defaultIterations is 25, so this pin holds the iteration count
	// as much as the floor.
	above, aboveIx := rescueWorld(5)
	got := ask.Run(above, aboveIx, "seeded", ask.Options{Limit: 20})
	if len(got.Hits) != 6 {
		t.Fatalf("got %v, want the seed and all five rescued neighbours at 0.150197654", idsOf(got))
	}
	for _, h := range got.Hits[1:] {
		if math.Abs(h.Graph-0.150197654) > 1e-9 {
			t.Errorf("%s: graph = %v, want 0.750988271/5 just above the floor", h.ID, h.Graph)
		}
	}

	below, belowIx := rescueWorld(6)
	got = ask.Run(below, belowIx, "seeded", ask.Options{Limit: 20})
	if len(got.Hits) != 1 || got.Hits[0].ID != "a.go#Seeded" {
		t.Fatalf("got %v, want the seed alone -- 0.125164712 is under the floor", idsOf(got))
	}
}
