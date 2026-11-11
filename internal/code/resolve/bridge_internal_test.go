package resolve

import (
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// The test namespaces cover every combination the built-in table may need,
// none of which it has to know.
const (
	nsOne      extract.Namespace = "t.one"      // one hit, whole realm without scope, nothing on a scoped miss
	nsOneNone  extract.Namespace = "t.one.none" // one hit, scope or nothing
	nsAll      extract.Namespace = "t.all"      // every hit, the realm as inferred on a miss
	nsGodot    extract.Namespace = "t.godot"    // one hit, Godot projects as realms
	nsUnlisted extract.Namespace = "t.unlisted" // no rule
)

var testRules = map[extract.Namespace]nsRule{
	nsOne:     {card: cardOne, realm: realmRepo, unscoped: fallRealmExtracted, scopedMiss: fallNone},
	nsOneNone: {card: cardOne, realm: realmRepo, unscoped: fallNone, scopedMiss: fallNone},
	nsAll:     {card: cardAll, realm: realmRepo, unscoped: fallRealmInferred, scopedMiss: fallRealmInferred},
	nsGodot:   {card: cardOne, realm: realmGodot, unscoped: fallRealmExtracted, scopedMiss: fallNone},
}

func off(ns extract.Namespace, name, target, file, lang string) offer {
	return offer{Export: extract.Export{Namespace: ns, Name: name, Target: model.NodeID(target)}, file: file, lang: lang}
}

func hand(ns extract.Namespace, name, file string, scope ...string) handover {
	return handover{bridge: extract.Bridge{
		Source: model.NodeID(file + "#src"), Relation: model.RelationReferences,
		Namespace: ns, Name: name, Scope: scope, File: file, Line: 7,
	}, unresolved: -1}
}

func edge(src, dst string, conf model.Confidence, line int) model.Edge {
	return model.Edge{Source: model.NodeID(src), Target: model.NodeID(dst), Relation: model.RelationReferences, Confidence: conf, Line: line}
}

func run(paths []string, offers []offer, hs []handover) []model.Edge {
	got, _ := bridgeEdges(testRules, paths, offers, hs, nil, nil)
	return got
}

func inClass(o offer, class string) offer { o.Class = class; return o }

func withClass(h handover, class string) handover { h.bridge.Class = class; return h }

func TestBridgeClassScopesTheOffers(t *testing.T) {
	offers := []offer{
		inClass(off(nsOne, "m", "a.cpp#A.m", "a.cpp", "cpp"), "A"),
		inClass(off(nsOne, "m", "b.cpp#B.m", "b.cpp", "cpp"), "B"),
	}
	got := run(nil, offers, []handover{withClass(hand(nsOne, "m", "s.tscn"), "A")})
	if want := []model.Edge{edge("s.tscn#src", "a.cpp#A.m", model.ConfidenceExtracted, 7)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBridgeClassWithoutHitFollowsScopedMiss(t *testing.T) {
	offers := []offer{
		inClass(off(nsOne, "m", "b.cpp#B.m", "b.cpp", "cpp"), "B"),
		inClass(off(nsAll, "m", "b.cpp#B.m", "b.cpp", "cpp"), "B"),
	}
	if got := run(nil, offers, []handover{withClass(hand(nsOne, "m", "s.tscn"), "A")}); got != nil {
		t.Errorf("a class miss without fallback drew %v", got)
	}
	got := run(nil, offers, []handover{withClass(hand(nsAll, "m", "s.tscn"), "A")})
	if want := []model.Edge{edge("s.tscn#src", "b.cpp#B.m", model.ConfidenceInferred, 7)}; !reflect.DeepEqual(got, want) {
		t.Errorf("a class miss with the realm as fallback: got %v, want %v", got, want)
	}
}

func TestBridgeWithoutClassIgnoresTheOffersClass(t *testing.T) {
	got := run(nil, []offer{inClass(off(nsOne, "m", "a.cpp#A.m", "a.cpp", "cpp"), "A")},
		[]handover{hand(nsOne, "m", "s.tscn")})
	if want := []model.Edge{edge("s.tscn#src", "a.cpp#A.m", model.ConfidenceExtracted, 7)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBridgeClassAndScopeBothApply(t *testing.T) {
	offers := []offer{
		inClass(off(nsOne, "m", "a.cpp#A.m", "a.cpp", "cpp"), "A"),
		inClass(off(nsOne, "m", "b.cpp#A.m", "b.cpp", "cpp"), "A"),
		inClass(off(nsOne, "m", "a.cpp#B.m", "a.cpp", "cpp"), "B"),
	}
	got := run(nil, offers, []handover{withClass(hand(nsOne, "m", "s.tscn", "a.cpp"), "A")})
	if want := []model.Edge{edge("s.tscn#src", "a.cpp#A.m", model.ConfidenceExtracted, 7)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBridgeScopedHitIsExtracted(t *testing.T) {
	got := run(nil,
		[]offer{off(nsOne, "m", "a.cs#m", "a.cs", "csharp"), off(nsOne, "m", "b.cs#m", "b.cs", "csharp")},
		[]handover{hand(nsOne, "m", "s.tscn", "a.cs")})
	if want := []model.Edge{edge("s.tscn#src", "a.cs#m", model.ConfidenceExtracted, 7)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBridgeScopedMissFallsBack(t *testing.T) {
	offers := []offer{off(nsOne, "m", "b.cs#m", "b.cs", "csharp"), off(nsAll, "m", "b.cs#m", "b.cs", "csharp")}
	if got := run(nil, offers, []handover{hand(nsOne, "m", "s.tscn", "a.cs")}); got != nil {
		t.Errorf("a scoped miss without fallback drew %v", got)
	}
	got := run(nil, offers, []handover{hand(nsAll, "m", "s.tscn", "a.cs")})
	if want := []model.Edge{edge("s.tscn#src", "b.cs#m", model.ConfidenceInferred, 7)}; !reflect.DeepEqual(got, want) {
		t.Errorf("a scoped miss with the realm as fallback: got %v, want %v", got, want)
	}
}

func TestBridgeUnscoped(t *testing.T) {
	offers := []offer{off(nsOne, "m", "b.cs#m", "b.cs", "csharp"), off(nsOneNone, "m", "b.cs#m", "b.cs", "csharp"), off(nsAll, "m", "b.cs#m", "b.cs", "csharp")}
	for _, c := range []struct {
		ns   extract.Namespace
		want []model.Edge
	}{
		{nsOne, []model.Edge{edge("s.tscn#src", "b.cs#m", model.ConfidenceExtracted, 7)}},
		{nsOneNone, nil},
		{nsAll, []model.Edge{edge("s.tscn#src", "b.cs#m", model.ConfidenceInferred, 7)}},
	} {
		if got := run(nil, offers, []handover{hand(c.ns, "m", "s.tscn")}); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.ns, got, c.want)
		}
	}
}

func TestBridgeCardOneAmbiguousGivesNoEdge(t *testing.T) {
	got := run(nil,
		[]offer{off(nsOne, "m", "a.cs#m", "a.cs", "csharp"), off(nsOne, "m", "b.cs#m", "b.cs", "csharp")},
		[]handover{hand(nsOne, "m", "s.tscn")})
	if got != nil {
		t.Fatalf("two hits drew %v", got)
	}
}

func TestBridgeCardAllLinksEveryHit(t *testing.T) {
	got := run(nil,
		[]offer{off(nsAll, "card", "a.css#card", "a.css", "css"), off(nsAll, "card", "b.css#card", "b.css", "css")},
		[]handover{hand(nsAll, "card", "p.html", "a.css", "b.css")})
	want := []model.Edge{
		edge("p.html#src", "a.css#card", model.ConfidenceExtracted, 7),
		edge("p.html#src", "b.css#card", model.ConfidenceExtracted, 7),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBridgeUnknownNamespaceGivesNoEdge(t *testing.T) {
	if got := run(nil, []offer{off(nsUnlisted, "m", "a.cs#m", "a.cs", "csharp")}, []handover{hand(nsUnlisted, "m", "s.tscn")}); got != nil {
		t.Fatalf("an unlisted namespace drew %v", got)
	}
}

func TestBridgeToItselfGivesNoEdge(t *testing.T) {
	if got := run(nil, []offer{off(nsOne, "m", "s.tscn#src", "s.tscn", "csharp")}, []handover{hand(nsOne, "m", "s.tscn")}); got != nil {
		t.Fatalf("a bridge onto its own source drew %v", got)
	}
}

func TestBridgeRealmAtRepoRoot(t *testing.T) {
	got := run([]string{"project.godot", "sub/user.gd", "native/my.cpp"},
		[]offer{off(nsGodot, "MyNode", "native/my.cpp#MyNode", "native/my.cpp", "cpp")},
		[]handover{hand(nsGodot, "MyNode", "sub/user.gd")})
	if want := []model.Edge{edge("sub/user.gd#src", "native/my.cpp#MyNode", model.ConfidenceExtracted, 7)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("a project at the repo root is no realm: got %v", got)
	}
}

func TestBridgeWithoutRealmGivesNoEdge(t *testing.T) {
	got := run([]string{"game/project.godot", "loose/x.gd", "src/my.cpp"},
		[]offer{off(nsGodot, "MyNode", "src/my.cpp#MyNode", "src/my.cpp", "cpp")},
		[]handover{hand(nsGodot, "MyNode", "loose/x.gd")})
	if got != nil {
		t.Fatalf("a bridge outside every project drew %v", got)
	}
}

func TestBridgeExportWithoutRealmCountsEverywhere(t *testing.T) {
	got := run([]string{"game/project.godot", "game/x.gd", "src/my.cpp"},
		[]offer{off(nsGodot, "MyNode", "src/my.cpp#MyNode", "src/my.cpp", "cpp")},
		[]handover{hand(nsGodot, "MyNode", "game/x.gd")})
	if want := []model.Edge{edge("game/x.gd#src", "src/my.cpp#MyNode", model.ConfidenceExtracted, 7)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBridgeOtherRealmIsInvisible(t *testing.T) {
	got := run([]string{"a/project.godot", "b/project.godot", "a/my.cpp", "b/x.gd"},
		[]offer{off(nsGodot, "MyNode", "a/my.cpp#MyNode", "a/my.cpp", "cpp")},
		[]handover{hand(nsGodot, "MyNode", "b/x.gd")})
	if got != nil {
		t.Fatalf("a name of project a answered project b: %v", got)
	}
}

func TestHandoverSkipsOwnLanguage(t *testing.T) {
	h := hand(nsOne, "Only", "b.gd")
	h.lang = "gdscript"
	got := run(nil, []offer{off(nsOne, "Only", "a.gd#Only", "a.gd", "gdscript")}, []handover{h})
	if got != nil {
		t.Fatalf("a resolver's handover was answered by its own language: %v", got)
	}
	h.lang = ""
	if got := run(nil, []offer{off(nsOne, "Only", "a.gd#Only", "a.gd", "gdscript")}, []handover{h}); len(got) != 1 {
		t.Fatalf("an extractor's bridge must see every language: %v", got)
	}
}

func TestBridgeMergesAndPrefersExtracted(t *testing.T) {
	late, early := hand(nsAll, "m", "s.tscn"), hand(nsAll, "m", "s.tscn", "a.cs")
	late.bridge.Line, early.bridge.Line = 9, 3
	got := run(nil, []offer{off(nsAll, "m", "a.cs#m", "a.cs", "csharp")}, []handover{late, early})
	if want := []model.Edge{edge("s.tscn#src", "a.cs#m", model.ConfidenceExtracted, 3)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	// The other order: the first is extracted, a later inferred one keeps it.
	got = run(nil, []offer{off(nsAll, "m", "a.cs#m", "a.cs", "csharp")}, []handover{early, late})
	if want := []model.Edge{edge("s.tscn#src", "a.cs#m", model.ConfidenceExtracted, 3)}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBridgeYieldsToLanguageEdge(t *testing.T) {
	existing := []model.Edge{edge("s.tscn#src", "a.cs#m", model.ConfidenceExtracted, 1)}
	got, _ := bridgeEdges(testRules, nil, []offer{off(nsOne, "m", "a.cs#m", "a.cs", "csharp")}, []handover{hand(nsOne, "m", "s.tscn")}, existing, nil)
	if got != nil {
		t.Fatalf("a bridge doubled a language's own edge: %v", got)
	}
}

func TestBridgeRewritesUnresolved(t *testing.T) {
	un := []Unresolved{
		{Source: "x.gd", Name: "Hit", Reason: ReasonNone},
		{Source: "y.gd", Name: "Two", Reason: ReasonNone},
		{Source: "z.gd", Name: "Gone", Reason: ReasonNone},
	}
	hit, two, gone := hand(nsOne, "Hit", "x.gd"), hand(nsOne, "Two", "y.gd"), hand(nsOne, "Gone", "z.gd")
	hit.unresolved, two.unresolved, gone.unresolved = 0, 1, 2
	offers := []offer{
		off(nsOne, "Hit", "h.cpp#Hit", "h.cpp", "cpp"),
		off(nsOne, "Two", "a.cpp#Two", "a.cpp", "cpp"), off(nsOne, "Two", "b.cpp#Two", "b.cpp", "cpp"),
	}
	_, got := bridgeEdges(testRules, nil, offers, []handover{hit, two, gone}, nil, un)
	want := []Unresolved{
		{Source: "y.gd", Name: "Two", Reason: ReasonAmbiguous},
		{Source: "z.gd", Name: "Gone", Reason: ReasonNone},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestMergeLangsShiftsUnresolvedIndex(t *testing.T) {
	first := langOut{
		edges:      []model.Edge{edge("a", "b", model.ConfidenceExtracted, 1)},
		unresolved: []Unresolved{{Name: "A"}, {Name: "B"}},
		offers:     []offer{off(nsOne, "A", "a#A", "a", "one")},
	}
	h, none := hand(nsOne, "C", "c"), hand(nsOne, "D", "d")
	h.unresolved = 0
	second := langOut{unresolved: []Unresolved{{Name: "C"}}, handovers: []handover{h, none}}
	edges, un, offers, hs := mergeLangs([]langOut{first, second})
	if len(edges) != 1 || len(un) != 3 || len(offers) != 1 || len(hs) != 2 {
		t.Fatalf("merged %d edges, %d unresolved, %d offers, %d handovers", len(edges), len(un), len(offers), len(hs))
	}
	if hs[0].unresolved != 2 || un[hs[0].unresolved].Name != "C" {
		t.Errorf("the handover points at %d, want 2 (C)", hs[0].unresolved)
	}
	if hs[1].unresolved != -1 {
		t.Errorf("a handover without an entry points at %d", hs[1].unresolved)
	}
}

func TestBuiltinRulesCoverEveryNamespace(t *testing.T) {
	if len(builtinRules) != len(extract.Namespaces) {
		t.Fatalf("%d rules for %d namespaces", len(builtinRules), len(extract.Namespaces))
	}
	for _, ns := range extract.Namespaces {
		if _, ok := builtinRules[ns]; !ok {
			t.Errorf("no rule for %s", ns)
		}
	}
	want := map[extract.Namespace]nsRule{
		extract.NamespaceGodotClass:  {card: cardOne, realm: realmGodot, unscoped: fallRealmExtracted, scopedMiss: fallNone},
		extract.NamespaceGodotMethod: {card: cardOne, realm: realmGodot, unscoped: fallNone, scopedMiss: fallNone},
		extract.NamespaceGodotSignal: {card: cardOne, realm: realmGodot, unscoped: fallNone, scopedMiss: fallNone},
	}
	if !reflect.DeepEqual(builtinRules, want) {
		t.Errorf("builtinRules = %v, want %v", builtinRules, want)
	}
}

func TestNearestRoot(t *testing.T) {
	roots := godotRoots([]string{"a/project.godot", "a/sub/project.godot", "b/x.gd", "aa/project.godot"})
	if want := []string{"a/sub", "aa", "a"}; !reflect.DeepEqual(roots, want) {
		t.Fatalf("roots = %v, want %v", roots, want)
	}
	for file, want := range map[string]string{"a/sub/x.gd": "a/sub", "a/x.gd": "a", "aa/x.gd": "aa"} {
		if got, ok := nearestRoot(roots, file); !ok || got != want {
			t.Errorf("%s: %q %v, want %q", file, got, ok, want)
		}
	}
	if _, ok := nearestRoot(roots, "b/x.gd"); ok {
		t.Error("b/x.gd lies under no project")
	}
	if got, ok := nearestRoot(godotRoots([]string{"project.godot"}), "b/x.gd"); !ok || got != "" {
		t.Errorf("the repo root project: %q %v", got, ok)
	}
}
