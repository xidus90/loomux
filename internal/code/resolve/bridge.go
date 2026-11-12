package resolve

import (
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// The bridge step links a use in one language to a definition another
// language offers. It knows namespaces and their rules, never languages
// (docs/.superpowers/specs/2026-11-10-graph-bruecken-design.md).

type cardinality int

const (
	cardOne cardinality = iota // exactly one hit, else no edge
	cardAll                    // an edge to every hit
)

type realmKind int

const (
	realmRepo  realmKind = iota // the whole repository is one realm
	realmGodot                  // each Godot project is a realm: the nearest project.godot above a file
)

// fallback is where a bridge looks when the files it names hold no hit, or
// when it names none.
type fallback int

const (
	fallNone           fallback = iota // nowhere: no edge
	fallRealmExtracted                 // the whole realm, as certain as a scoped hit
	fallRealmInferred                  // the whole realm, marked a guess
)

type nsRule struct {
	card                 cardinality
	realm                realmKind
	unscoped, scopedMiss fallback
}

// builtinRules is every namespace's rule. Godot's ClassDB holds class names
// per project, so an unscoped class name is certain within one; a member is
// looked up only in the script a connection names. The web namespaces follow
// docs/.superpowers/specs/2026-11-11-web-design.md: a class or id of a style
// rule is never guessed, an element or a script function outside the scope
// is.
var builtinRules = map[extract.Namespace]nsRule{
	extract.NamespaceGodotClass:  {card: cardOne, realm: realmGodot, unscoped: fallRealmExtracted, scopedMiss: fallNone},
	extract.NamespaceGodotMethod: {card: cardOne, realm: realmGodot, unscoped: fallNone, scopedMiss: fallNone},
	extract.NamespaceGodotSignal: {card: cardOne, realm: realmGodot, unscoped: fallNone, scopedMiss: fallNone},
	extract.NamespaceCSSClass:    {card: cardAll, realm: realmRepo, unscoped: fallNone, scopedMiss: fallNone},
	extract.NamespaceCSSID:       {card: cardAll, realm: realmRepo, unscoped: fallNone, scopedMiss: fallNone},
	extract.NamespaceHTMLID:      {card: cardOne, realm: realmRepo, unscoped: fallRealmInferred, scopedMiss: fallNone},
	extract.NamespaceJSFunction:  {card: cardOne, realm: realmRepo, unscoped: fallRealmInferred, scopedMiss: fallNone},
}

// offer is an export with the file and language of the result that made
// it: an embedded part offers in its own language.
type offer struct {
	extract.Export
	file, lang string
}

// handover is a bridge waiting for the bridge step. lang is the language
// whose resolver handed it over and already searched its own index, so its
// own exports cannot answer; empty for a bridge an extractor wrote.
// unresolved indexes the Unresolved entry the bridge may settle, or is -1.
type handover struct {
	bridge     extract.Bridge
	lang       string
	unresolved int
}

// langOut is what the resolver of one language hands back.
type langOut struct {
	edges      []model.Edge
	unresolved []Unresolved
	offers     []offer
	handovers  []handover
}

// mergeLangs joins the languages' results in order; a handover's index then
// counts the entries of every language before its own.
func mergeLangs(outs []langOut) (edges []model.Edge, un []Unresolved, offers []offer, handovers []handover) {
	for _, o := range outs {
		base := len(un)
		edges = append(edges, o.edges...)
		un = append(un, o.unresolved...)
		offers = append(offers, o.offers...)
		for _, h := range o.handovers {
			if h.unresolved >= 0 {
				h.unresolved += base
			}
			handovers = append(handovers, h)
		}
	}
	return edges, un, offers, handovers
}

// godotRoots is the directory of every project.godot among paths, the
// longest first so the nearest project wins; "" is the repository root.
func godotRoots(paths []string) []string {
	var roots []string
	for _, p := range paths {
		if path.Base(p) == "project.godot" {
			roots = append(roots, dirOf(p))
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		if len(roots[i]) != len(roots[j]) {
			return len(roots[i]) > len(roots[j])
		}
		return roots[i] < roots[j]
	})
	return roots
}

// nearestRoot is the first of roots, longest first, that holds file.
func nearestRoot(roots []string, file string) (string, bool) {
	for _, root := range roots {
		if root == "" || strings.HasPrefix(file, root+"/") {
			return root, true
		}
	}
	return "", false
}

// bridgeEdges resolves every handover against the offers and returns the
// new edges and un with each settled entry dropped and each clash marked.
// existing are the languages' own edges, which a bridge never doubles.
func bridgeEdges(rules map[extract.Namespace]nsRule, paths []string, offers []offer, handovers []handover, existing []model.Edge, un []Unresolved) ([]model.Edge, []Unresolved) {
	roots := godotRoots(paths)
	realm := func(kind realmKind, file string) (string, bool) {
		if kind == realmRepo {
			return "", true
		}
		return nearestRoot(roots, file)
	}
	type key struct {
		source, target model.NodeID
		relation       model.Relation
	}
	taken := map[key]bool{}
	for _, e := range existing {
		taken[key{e.Source, e.Target, e.Relation}] = true
	}
	var out []model.Edge
	at := map[key]int{}
	settled, clashed := map[int]bool{}, map[int]bool{}

	for _, h := range handovers {
		b := h.bridge
		rule, ok := rules[b.Namespace]
		if !ok {
			continue
		}
		home, ok := realm(rule.realm, b.File)
		if !ok {
			continue
		}
		var cands []offer
		for _, o := range offers {
			if o.Namespace != b.Namespace || o.Name != b.Name || (h.lang != "" && o.lang == h.lang) {
				continue
			}
			// An offer under no project is seen from every one: GDExtension
			// sources sit beside the project they extend.
			if r, ok := realm(rule.realm, o.file); ok && r != home {
				continue
			}
			cands = append(cands, o)
		}
		hits, conf, fb := cands, model.ConfidenceExtracted, rule.unscoped
		if len(b.Scope) > 0 || b.Class != "" {
			var scoped []offer
			for _, o := range cands {
				if (len(b.Scope) == 0 || slices.Contains(b.Scope, o.file)) && (b.Class == "" || o.Class == b.Class) {
					scoped = append(scoped, o)
				}
			}
			hits, fb = scoped, rule.scopedMiss
			if len(scoped) == 0 {
				hits = cands
			} else {
				fb = fallRealmExtracted
			}
		}
		switch fb {
		case fallNone:
			hits = nil
		case fallRealmInferred:
			conf = model.ConfidenceInferred
		}
		if rule.card == cardOne && len(hits) != 1 {
			if len(hits) > 1 && h.unresolved >= 0 {
				clashed[h.unresolved] = true
			}
			continue
		}
		for _, o := range hits {
			if o.Target == b.Source {
				continue
			}
			k := key{b.Source, o.Target, b.Relation}
			if taken[k] {
				continue
			}
			if h.unresolved >= 0 {
				settled[h.unresolved] = true
			}
			if i, ok := at[k]; ok {
				out[i].Line = min(out[i].Line, b.Line)
				if conf == model.ConfidenceExtracted {
					out[i].Confidence = conf
				}
				continue
			}
			at[k] = len(out)
			out = append(out, model.Edge{Source: b.Source, Target: o.Target, Relation: b.Relation, Confidence: conf, Line: b.Line})
		}
	}

	var kept []Unresolved
	for i, u := range un {
		if settled[i] {
			continue
		}
		if clashed[i] {
			u.Reason = ReasonAmbiguous
		}
		kept = append(kept, u)
	}
	return out, kept
}
