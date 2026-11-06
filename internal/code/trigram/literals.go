// Package trigram is the trigram index of the graph's search corpus: it turns
// file bytes into case-folded trigrams, stores file-level postings, and uses a
// grep pattern's literals to narrow the files a scan has to read.
//
// The index never answers a search by itself. It only says which files cannot
// match; the scan of grep.Search remains the verdict, so a wrong or stale
// index costs time, never a hit.
package trigram

import (
	"fmt"
	"regexp/syntax"
	"sort"
	"strings"
)

// maxSets and maxLiterals bound the disjunction a pattern may produce. A
// pattern that needs more is not worth narrowing: the scan over the
// candidates it would leave is cheaper than the bookkeeping.
const (
	maxSets     = 16
	maxLiterals = 8
)

// Literals derives the literal conditions a pattern imposes on a matching
// file.
//
// The result is a disjunction of conjunctions: outer index = one alternative,
// inner slice = literals that all have to appear. ok == false means the
// pattern imposes no condition the index can use, and the caller scans
// everything. That is not an error; a pattern like "[a-z]+" simply has no
// literal.
//
// Only ASCII literals are conditions: the index is folded to ASCII lowercase,
// so a case-insensitive non-ASCII letter ("Ä" against "ä") is outside its
// contract. Such a pattern falls back to the scan rather than risking a
// candidate set that misses matches. The same holds for a case-sensitive
// non-ASCII literal, whose bytes the folded index does not carry as a
// trigram; the conservative answer costs a scan, never a hit.
//
// fixed takes pattern as one literal, the way grep --fixed does. Errors are
// the two the scan itself refuses: a pattern RE2 cannot compile, and a form
// RE2 does not support at all (lookaround, backreference).
func Literals(pattern string, fixed, ignoreCase bool) ([][]string, bool, error) {
	if pattern == "" {
		return nil, false, fmt.Errorf("empty search pattern")
	}
	if fixed {
		lit := pattern
		if ignoreCase {
			if !isASCII(lit) {
				return nil, false, nil
			}
			lit = strings.ToLower(lit)
		}
		if !isASCII(lit) || len(lit) < 3 {
			return nil, false, nil
		}
		return [][]string{{lit}}, true, nil
	}
	if form := unsupportedForm(pattern); form != "" {
		return nil, false, fmt.Errorf("%s not supported in RE2 regex: %s", form, pattern)
	}
	p := pattern
	if ignoreCase {
		p = "(?i)" + p
	}
	re, err := syntax.Parse(p, syntax.Perl)
	if err != nil {
		return nil, false, fmt.Errorf("invalid regex: %w", err)
	}
	sets := walk(re)
	if sets == nil {
		return nil, false, nil
	}
	return normalize(sets), true, nil
}

// walk returns the disjunction of conjunctions re imposes, or nil when re
// imposes no usable condition.
func walk(re *syntax.Regexp) [][]string {
	switch re.Op {
	case syntax.OpLiteral:
		lit, ok := literalOf(re)
		if !ok {
			return nil
		}
		return [][]string{{lit}}
	case syntax.OpCapture:
		return walk(re.Sub[0])
	case syntax.OpConcat:
		out := [][]string{{}}
		for _, sub := range re.Sub {
			subs := walk(sub)
			if subs == nil {
				continue
			}
			out = product(out, subs)
			if out == nil {
				return nil
			}
		}
		if len(out) == 1 && len(out[0]) == 0 {
			return nil
		}
		return out
	case syntax.OpAlternate:
		var out [][]string
		for _, sub := range re.Sub {
			subs := walk(sub)
			if subs == nil {
				// One branch without a condition makes the whole disjunction
				// unconstrained: the other branches cannot narrow it.
				return nil
			}
			out = append(out, subs...)
			if len(out) > maxSets {
				return nil
			}
		}
		return out
	default:
		return nil
	}
}

// literalOf turns one OpLiteral into its folded byte string, when it is long
// enough to carry a trigram.
func literalOf(re *syntax.Regexp) (string, bool) {
	s := string(re.Rune)
	if !isASCII(s) {
		return "", false
	}
	if re.Flags&syntax.FoldCase != 0 {
		s = strings.ToLower(s)
	}
	if len(s) < 3 {
		return "", false
	}
	return s, true
}

// product joins two disjunctions into their cross product.
func product(left, right [][]string) [][]string {
	out := make([][]string, 0, len(left)*len(right))
	for _, l := range left {
		for _, r := range right {
			joined := make([]string, 0, len(l)+len(r))
			joined = append(joined, l...)
			joined = append(joined, r...)
			if len(joined) > maxLiterals || len(out)+1 > maxSets {
				return nil
			}
			out = append(out, joined)
		}
	}
	return out
}

// normalize sorts each conjunction and the disjunction itself, so the same
// pattern yields the same slices in the same order on every run. A repeated
// branch needs no removal here: syntax.Parse already drops duplicates.
func normalize(sets [][]string) [][]string {
	for _, set := range sets {
		sort.Strings(set)
	}
	sort.Slice(sets, func(i, j int) bool {
		return strings.Join(sets[i], "\x00") < strings.Join(sets[j], "\x00")
	})
	return sets
}

// isASCII reports whether s is a byte string of ASCII only.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// unsupportedForm names the first form RE2 has no answer for -- "lookaround"
// or "backreference" -- or "" when there is none. It walks the pattern instead
// of searching it, because a backslash escapes the character after it:
// `\\1` is a literal backslash and a 1, `\(?=` an optional literal
// parenthesis. grep.unsupported does the same walk; the copy keeps the error
// wording identical without widening grep's API for one caller.
func unsupportedForm(pattern string) string {
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			if i+1 < len(pattern) && pattern[i+1] >= '1' && pattern[i+1] <= '9' {
				return "backreference"
			}
			i++
		case '(':
			rest := pattern[i+1:]
			for _, open := range []string{"?=", "?!", "?<=", "?<!"} {
				if strings.HasPrefix(rest, open) {
					return "lookaround"
				}
			}
		}
	}
	return ""
}
