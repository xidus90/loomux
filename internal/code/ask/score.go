// Package ask answers a plain-text question from the code graph.
//
// The pipeline is the reference's, and its slogan is worth keeping because it
// is the design: lexical proposes, graph disposes. A word match nominates
// candidates; the walk over the wiring edges decides which of them the question
// was actually about, and rescues the helper the question never named.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/ask/ask.ts and
// src/ask/graphrank.ts.
package ask

import (
	"math"
	"sort"

	"github.com/xidus90/loomux/internal/code/lexicon"
)

// The field weights. A name hit is worth three, a path hit two, and the body
// goes through BM25 at one. They decide whether a name beats a path, so they
// are copied and not rounded.
const (
	nameWeight = 3.0
	pathWeight = 2.0
)

// BM25 parameters: k1 saturates a repeated term, b normalizes by body length.
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// idf is the inverse document frequency of a term: log(1 + n/(1+df)).
//
// A word that appears across the whole corpus -- "handler" in a repository of
// handlers -- weighs far less than a rare, discriminating identifier. The 1+ in
// the denominator is what makes a term no document holds finite rather than a
// division by zero.
func idf(df, docCount int) float64 {
	return math.Log(1 + float64(docCount)/float64(1+df))
}

// weights is the idf of every term of a query, against one corpus.
func weights(query map[string]int, ix *lexicon.Index) map[string]float64 {
	out := make(map[string]float64, len(query))
	for term := range query {
		out[term] = idf(ix.DF[term], ix.DocCount)
	}
	return out
}

// question is one query in the two forms the sums need: its terms in a fixed
// order, and their counts.
//
// Every sum below runs over that order and never over the counts map: Go
// randomizes map iteration order per range, and the same floats added in
// another order differ in the last bits. That difference decides a tie between
// two nodes with identical bags, and an Answer promises descending score and
// ascending id -- not descending score and whatever the last range happened to
// yield. pagerank.Rank guards the same arithmetic the same way.
//
// It is a type and not a sorted slice beside a map because newQuestion is then
// the only way to obtain one, and there is no way to obtain one unsorted.
type question struct {
	order  []string
	counts map[string]int
}

// newQuestion sorts a query's terms once.
//
// Run calls this ONCE per query and hands the result down, because the order
// belongs to the question and not to the document being scored. Sorted inside
// the per-document score it was three allocations and three sorts of the same
// strings per document -- 3D of each for a corpus of D documents.
func newQuestion(counts map[string]int) question {
	order := make([]string, 0, len(counts))
	for term := range counts {
		order = append(order, term)
	}
	sort.Strings(order)
	return question{order: order, counts: counts}
}

// overlap is the idf-weighted term overlap of a query and a short field -- a
// name or a path, where length carries no information worth normalizing.
func overlap(q question, doc map[string]int, idf map[string]float64) float64 {
	var s float64
	for _, term := range q.order {
		if dn, ok := doc[term]; ok {
			s += float64(q.counts[term]) * float64(dn) * idf[term]
		}
	}
	return s
}

// bm25 scores a query against a body, saturating repeats and normalizing by
// length.
//
// Only the question's order is read and never its counts: the query's own term
// frequency does not enter here, and the reference does not use it either. BM25
// weighs how often the DOCUMENT says a word, not how often the question did.
func bm25(q question, body map[string]int, idf map[string]float64, bodyLen, avgLen float64) float64 {
	norm := bm25K1 * (1 - bm25B + bm25B*lengthRatio(bodyLen, avgLen))
	var s float64
	for _, term := range q.order {
		tf, ok := body[term]
		if !ok {
			continue
		}
		s += idf[term] * (float64(tf) * (bm25K1 + 1) / (float64(tf) + norm))
	}
	return s
}

// lengthRatio is a body's length against the corpus average, guarded: an empty
// corpus would otherwise divide by zero and poison every score with NaN.
func lengthRatio(bodyLen, avgLen float64) float64 {
	if avgLen <= 0 {
		return 1
	}
	return bodyLen / avgLen
}

// lexical is one document's whole lexical score.
func lexical(q question, d lexicon.Doc, idf map[string]float64, avgLen float64) float64 {
	return nameWeight*overlap(q, d.Name, idf) +
		pathWeight*overlap(q, d.Path, idf) +
		bm25(q, d.Body, idf, bagLen(d.Body), avgLen)
}

// bagLen is a field's length in tokens.
func bagLen(bag map[string]int) float64 {
	var n int
	for _, c := range bag {
		n += c
	}
	return float64(n)
}
