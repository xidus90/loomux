package model

import "sort"

// SymbolSpan binds a non-file node to its parsed start and end lines.
type SymbolSpan struct {
	Node     *Node
	From, To int
}

// FileSpans indexes the non-file nodes of g by path, sorted by From asc, then
// To desc, then ID asc, so the order never depends on the graph's.
func FileSpans(g *Graph) map[string][]SymbolSpan {
	m := make(map[string][]SymbolSpan)
	if g == nil {
		return m
	}
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Kind == KindFile {
			continue
		}
		from, to, ok := n.Span.Lines()
		if !ok {
			continue
		}
		m[n.Path] = append(m[n.Path], SymbolSpan{Node: n, From: from, To: to})
	}
	for _, s := range m {
		sort.SliceStable(s, func(i, j int) bool {
			if s[i].From != s[j].From {
				return s[i].From < s[j].From
			}
			if s[i].To != s[j].To {
				return s[i].To > s[j].To
			}
			return s[i].Node.ID < s[j].Node.ID
		})
	}
	return m
}

// Enclosing returns the symbol whose span contains line. When spans nest, the
// one with the largest start line wins (the original's grep.ts rule: innermost
// in enclosing sense).
func Enclosing(spans []SymbolSpan, line int) *Node {
	var best *Node
	maxStart := -1
	for _, s := range spans {
		if line >= s.From && line <= s.To {
			if s.From > maxStart {
				maxStart = s.From
				best = s.Node
			}
		}
	}
	return best
}

// Innermost returns symbols overlapping [from, to] that contain no other symbol
// overlapping that range (the original's blast.ts rule).
func Innermost(spans []SymbolSpan, from, to int) []*Node {
	var hit []SymbolSpan
	for _, s := range spans {
		if s.From <= to && s.To >= from {
			hit = append(hit, s)
		}
	}
	var out []*Node
	for i, s1 := range hit {
		containsOther := false
		for j, s2 := range hit {
			if i != j && s2.From >= s1.From && s2.To <= s1.To {
				containsOther = true
				break
			}
		}
		if !containsOther {
			out = append(out, s1.Node)
		}
	}
	return out
}
