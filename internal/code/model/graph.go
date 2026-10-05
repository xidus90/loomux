// Package model is the read model of the code graph: the nodes and edges an
// extractor writes to wiring.json, and the questions about them that need no
// walk.
//
// A model of its own rather than decoding into a map: a field the extractor
// renames would otherwise reach a caller unnoticed.
//
// Ported from src/graph/types.ts (MIT; origin under
// "Ported sources" in NOTICE.md).
package model

import (
	"slices"
	"strings"
)

// NodeID identifies a node. It is path-scoped -- "src/cache.ts#Cache.get" --
// and opaque: the span is deliberately not part of it, so a definition moving
// down a file keeps its identity, and a duplicated definition can carry an
// ordinal ("Cache.get~2").
type NodeID string

// Kind is what a node represents, in LSP vocabulary.
type Kind string

// KindFile marks the node that stands for a whole file. It is the only kind
// this package tells apart.
const KindFile Kind = "file"

// Span is a line range in the form "L165-L222".
type Span string

// Lines reads "L12-L40" back into 12 and 40, reporting whether the span had
// that shape at all.
func (s Span) Lines() (int, int, bool) {
	str := string(s)
	dash := strings.Index(str, "-L")
	if !strings.HasPrefix(str, "L") || dash < 0 {
		return 0, 0, false
	}
	from, okA := atoi(str[1:dash])
	to, okB := atoi(str[dash+2:])
	// Half a span is no span: a caller that ignores ok must not get a line
	// number that happens to parse next to one that did not.
	if !okA || !okB {
		return 0, 0, false
	}
	return from, to, true
}

// atoi is strconv.Atoi reporting failure as a bool, because a malformed span is
// not an error any caller would handle differently from an unusable one.
func atoi(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// Relation is what an edge means.
type Relation string

// The six relations of the schema.
const (
	RelationContains   Relation = "contains"   // file -> symbol, class -> method
	RelationCalls      Relation = "calls"      // function -> function it invokes
	RelationImports    Relation = "imports"    // file -> module
	RelationReferences Relation = "references" // symbol -> symbol it names
	RelationImplements Relation = "implements" // class -> interface
	RelationExtends    Relation = "extends"    // class -> base class
)

// Confidence says how sure the extractor is that an edge is true, strongest
// first. G1 reads it and judges nothing by it.
type Confidence string

// The four confidences of the schema.
const (
	ConfidenceLSPResolved Confidence = "lsp_resolved"
	ConfidenceLSPDispatch Confidence = "lsp_dispatch"
	ConfidenceExtracted   Confidence = "extracted"
	ConfidenceInferred    Confidence = "inferred"
)

// IsWalk reports whether an edge of this relation carries dependency meaning
// for a walk or a rank.
//
// "contains" is excluded on purpose: a file contains every symbol defined in
// it, so walking that edge would make every same-file symbol a neighbour and
// let the file act as a hub that floods the walk.
func (r Relation) IsWalk() bool {
	switch r {
	case RelationCalls, RelationReferences, RelationImports,
		RelationImplements, RelationExtends:
		return true
	default:
		return false
	}
}

// Node is one definition: a file, or a symbol inside one.
type Node struct {
	ID    NodeID `json:"id"`
	Name  string `json:"name"`
	Kind  Kind   `json:"kind"`
	Owner string `json:"owner,omitempty"`

	// Path is repo-relative and slash-separated, on every platform -- never a
	// backslash, because an id begins with this path (a file node's id IS it,
	// a symbol's continues with "#" and the symbol) and every prefix
	// comparison runs over that shape. A backslash here cannot divide the
	// index from the walk: ask.Run judges both by this Path through one
	// admission rule, lexicon.UnderPrefix against the normalized prefix, so
	// the index drops exactly the nodes the walk drops. It empties both at
	// once instead, and it does so visibly:
	// asked with a prefix, the filtered index holds nothing, nothing is
	// seeded, and ask.Run answers "no matching nodes" for a subtree that
	// exists. The Go extractor takes it from its rel argument, which graph
	// build fills from sourceset's Rel -- and that one is filepath.ToSlash'ed.
	Path string `json:"path"`

	Span      Span   `json:"span"`
	Signature string `json:"signature"`
	Exported  bool   `json:"exported"`

	// BodyHash is sha256 (full hex) over the text of the whole declaration. It
	// is what `graph check` diffs on: a body that changed changes the hash,
	// and a doc comment that changed does not -- go/ast's Pos()..End() leaves
	// the comment out, matching tree-sitter, where the comment is a sibling.
	BodyHash string `json:"body_hash"`

	// BodyText is the searchable body: whitespace collapsed, capped. It never
	// reaches disk -- the ask sidecar holds it tokenized, and in wiring.json it
	// would be ~65% of the bytes. The tag is the enforcement, not a
	// convenience.
	BodyText string `json:"-"`
}

// Edge wires two nodes.
//
// Target is not always a node: an unresolved import names the module itself
// ("npm:lodash"). A rank drops such an edge, a walk keeps it as a hit without
// a node.
type Edge struct {
	Source     NodeID     `json:"source"`
	Target     NodeID     `json:"target"`
	Relation   Relation   `json:"relation"`
	Confidence Confidence `json:"confidence"`
}

// Meta is what the writer says about the graph it wrote.
type Meta struct {
	Version   int      `json:"version"`
	NodeCount int      `json:"nodeCount"`
	EdgeCount int      `json:"edgeCount"`
	Languages []string `json:"languages"`

	// Extractor identifies the extractors that produced this graph: the
	// version of each language, sorted and joined by "+" ("go/1" alone, or
	// "go/1+python/1@..."). `check` compares it before diffing a single node:
	// a graph from another extractor is not stale, it is foreign, and its
	// nodes say nothing about this code.
	Extractor string `json:"extractor"`
}

// BuiltBy reports whether one extractor version is a member of the graph's
// combined stamp.
//
// Membership and not equality, for a reader that knows one language only:
// the Go edit hook compares Go body hashes and needs the Go extractor to have
// taken part, whichever other languages took part with it. Membership and
// not a prefix either: "go/10" is not "go/1".
func (m Meta) BuiltBy(version string) bool {
	return m.Extractor != "" && slices.Contains(strings.Split(m.Extractor, "+"), version)
}

// Graph is a whole wiring.json.
type Graph struct {
	Meta  Meta   `json:"meta"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// SymbolsInFile returns the symbols defined in the file at path, in graph
// order.
//
// Membership is path equality, not the "contains" edge: an extractor writes
// the path on every node it emits, while "contains" is a courtesy the schema
// allows and no consumer may require.
func SymbolsInFile(g *Graph, path string) []NodeID {
	if g == nil {
		return nil
	}
	var symbols []NodeID
	for _, n := range g.Nodes {
		if n.Kind != KindFile && n.Path == path {
			symbols = append(symbols, n.ID)
		}
	}
	return symbols
}
