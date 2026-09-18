package ask

import (
	"regexp"
	"sort"

	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

// defaultLimit is how many hits an answer carries when the caller says
// nothing.
const defaultLimit = 8

// graphWeight is how much connectivity may lift a blended score. A
// lexically perfect node scores 1 on its axis; at 0.5 the walk can reorder
// near-ties and separate a wired-in hit from an isolated same-word collision,
// without overruling a clear lexical winner.
const graphWeight = 0.5

// rescueFloor is the share of the top node's mass a node needs to join the
// results without having matched a single word. It is what surfaces the helper
// or the configuration a task depends on and never named.
const rescueFloor = 0.15

// testRankPenalty de-ranks a test file multiplicatively.
//
// Tests mirror the tokens of the code they exercise, so a test can out-score
// the definition on a lexical tie and land on top -- observed as an answer that
// sent a reader into a Test... function. Multiplicative, so tests stay in the
// results: "where are the tests" is a fair question.
const testRankPenalty = 0.35

// testSeeking is a query that asks about tests, which lifts the penalty
// entirely. Without this a question about tests answers with everything except
// the tests.
var testSeeking = regexp.MustCompile(`(?i)\b(tests?|specs?|coverage|assert(?:ion)?s?|fixtures?|mocks?)\b`)

// isTestPath reports whether a path holds tests. Go's _test.go, and the
// directory names the reference also covers.
var isTestPath = regexp.MustCompile(`(^|/)(tests?|__tests__|spec)/|_test\.go$|(\.test|\.spec)\.[a-z]+$`)

// Options are the knobs of the RANKING. Whether the source is inlined is not
// one of them: that is Inline's argument (source.go), because the ranking does
// not read a single file and must not start.
type Options struct {
	Limit int
	In    string
}

// Hit is one answer, with both axes kept apart so a reader can see WHY it is
// here.
type Hit struct {
	ID        model.NodeID `json:"id"`
	Path      string       `json:"path"`
	Span      model.Span   `json:"span"`
	Signature string       `json:"signature,omitempty"`
	Score     float64      `json:"score"`
	Lexical   float64      `json:"lexical"`
	Graph     float64      `json:"graph"`
	Code      string       `json:"code,omitempty"`
}

// Answer is a whole reply.
type Answer struct {
	Query string `json:"query"`
	Hits  []Hit  `json:"hits"`
	Note  string `json:"note,omitempty"`
}

// Run answers one question.
//
// Four steps, in this order and for this reason:
//
//  1. lexical -- every document gets an idf-weighted word score, test files
//     de-ranked already, so a test seeds the walk with less mass too
//  2. seed    -- those scores ARE the personalized PageRank seed
//  3. rescue  -- a node the walk found central joins even with no word match
//  4. blend   -- (lexical/max + 0.5*graph) * testFactor, de-ranked AGAIN
//     because dividing by max would otherwise restore a winning test to 1.0
func Run(g *model.Graph, ix *lexicon.Index, query string, opts Options) Answer {
	// One normalization for both narrowings. lexicon owns the rule, because the
	// index and the walk have to agree about what a prefix names.
	in := lexicon.NormalizePrefix(opts.In)
	if in != "" {
		ix = ix.Filter(in)
	}
	terms := lexicon.Counts(lexicon.Tokenize(query))
	factor := testFactor(query)
	w := weights(terms, ix)
	// The question's term order, sorted once here. Every sum below runs over
	// it; sorting inside the per-document score meant three allocations and
	// three sorts of the same handful of strings per document.
	q := newQuestion(terms)

	byID := make(map[model.NodeID]model.Node, len(g.Nodes))
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}

	lex := map[model.NodeID]float64{}
	var maxLex float64
	for _, d := range ix.Docs {
		n, ok := byID[d.ID]
		if !ok {
			// A sidecar built before the graph was rewritten: a document that
			// names no node has no path and no span, so nothing to open.
			continue
		}
		score := lexical(q, d, w, ix.AvgBodyLen) * factor(n.Path)
		if score <= 0 {
			continue
		}
		lex[d.ID] = score
		if score > maxLex {
			maxLex = score
		}
	}

	pr := walk(g, byID, lex, in)

	candidates := map[model.NodeID]bool{}
	for id := range lex {
		candidates[id] = true
	}
	for _, s := range pr {
		if s.Score >= rescueFloor {
			candidates[s.ID] = true
		}
	}
	prByID := make(map[model.NodeID]float64, len(pr))
	for _, s := range pr {
		prByID[s.ID] = s.Score
	}

	var hits []Hit
	for id := range candidates {
		// Every candidate is a node: lex was keyed through byID above, and a
		// rescued id comes from the topology, which Prepare builds from
		// g.Nodes. maxLex is positive for the same reason -- a candidate exists
		// only when the lexical pass or the walk it seeds found something.
		n := byID[id]
		lexNorm := lex[id] / maxLex
		graph := prByID[id]
		hits = append(hits, Hit{
			ID: id, Path: n.Path, Span: n.Span, Signature: n.Signature,
			Score:   (lexNorm + graphWeight*graph) * factor(n.Path),
			Lexical: lexNorm, Graph: graph,
		})
	}

	// Descending by score, ascending by id on a tie -- the same rule G1's
	// ranking uses, so two layers of this system cannot disagree about order.
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].ID < hits[j].ID
	})

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}

	a := Answer{Query: query, Hits: hits}
	if len(hits) == 0 {
		// An empty list reads as "there is no such code". Say which it was.
		a.Note = "no matching nodes -- try different words, or run `loomux graph build` if the graph is empty"
	}
	return a
}

// walk runs personalized PageRank over the wiring edges, seeded by the lexical
// scores.
//
// The prefix narrows the WALK and not only the seed: without that, a neighbour
// outside the prefix could clear the rescue floor and surface, which would
// defeat the promise that a filter applies before scoring. G1's Prepare takes
// exactly this filter. The prefix arrives normalized, and the comparison is
// lexicon's own -- the index and the walk must not hold two opinions about what
// a prefix names.
//
// byID is Run's node lookup, handed over rather than rebuilt: the same scan of
// g.Nodes served the lexical pass already.
func walk(g *model.Graph, byID map[model.NodeID]model.Node, seed map[model.NodeID]float64, prefix string) []pagerank.Scored {
	if len(seed) == 0 {
		return nil
	}
	keep := func(id model.NodeID) bool { return true }
	if prefix != "" {
		keep = func(id model.NodeID) bool { return lexicon.UnderPrefix(byID[id].Path, prefix) }
	}
	return pagerank.Rank(pagerank.Prepare(g, keep), seed, pagerank.Options{})
}

// testFactor is the de-ranking multiplier for a path, or 1 throughout when the
// query itself asks about tests.
func testFactor(query string) func(path string) float64 {
	if testSeeking.MatchString(query) {
		return func(string) float64 { return 1 }
	}
	return func(path string) float64 {
		if isTestPath.MatchString(path) {
			return testRankPenalty
		}
		return 1
	}
}
