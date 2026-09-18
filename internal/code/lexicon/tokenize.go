// Package lexicon is the text side of the code graph: how a name, a path and a
// body become tokens, and the build-time sidecar that keeps a query from doing
// that work again.
//
// Tokenize and Counts live here and not in the asking package, for the reason
// Graft gives in the same place: the sidecar can only be a correct cache of the
// query-time arithmetic if both sides call the same function. Two tokenizers
// that agree today are two tokenizers.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/ask/index-file.ts.
package lexicon

import (
	"regexp"
	"strings"
)

// stopWords carry no query intent -- too common or too short to discriminate.
// Thirty-two words, counted against the reference rather than taken from a
// comment.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "of": true, "to": true, "in": true,
	"is": true, "are": true, "how": true, "does": true, "do": true, "what": true,
	"where": true, "which": true, "that": true, "this": true, "it": true,
	"for": true, "on": true, "and": true, "or": true, "with": true, "i": true,
	"we": true, "get": true, "set": true, "use": true, "used": true,
	"using": true, "when": true, "why": true, "can": true,
}

// StopWordCount is how many words the list holds. A test pins it, so shrinking
// or growing the list is a deliberate edit and not a slip.
func StopWordCount() int { return len(stopWords) }

// camelBoundary is where a lowercase or digit meets an uppercase letter.
//
// ASCII by design, byte for byte as in the reference. A Unicode-aware split
// would be the nicer Go version and would tokenize an identifier carrying an
// umlaut differently than every golden value in this repository.
var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// separator is every run of characters that is not a lowercase letter or digit.
var separator = regexp.MustCompile(`[^a-z0-9]+`)

// Tokenize splits prose and identifiers into lowercased subword tokens.
//
// The order matters: the camel boundary is marked BEFORE lowercasing, or the
// boundary would be gone by the time we look for it.
func Tokenize(text string) []string {
	spaced := camelBoundary.ReplaceAllString(text, "$1 $2")
	var out []string
	for _, tok := range separator.Split(strings.ToLower(spaced), -1) {
		if len(tok) > 1 && !stopWords[tok] {
			out = append(out, tok)
		}
	}
	return out
}

// Counts is a token-frequency bag.
func Counts(tokens []string) map[string]int {
	out := make(map[string]int, len(tokens))
	for _, t := range tokens {
		out[t]++
	}
	return out
}
