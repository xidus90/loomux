package model

import "sort"

// SymbolSpan binds a non-file node to its parsed start and end lines.
type SymbolSpan struct {
	Node     *Node
	From, To int
}

// FileSpans indexes the non-file nodes of g by path, sorted by From asc (ties by To desc).
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
	for path := range m {
		sort.Slice(m[path], func(i, j int) bool {
			if m[path][i].From != m[path][j].From {
				return m[path][i].From < m[path][j].From
			}
			return m[path][i].To > m[path][j].To
		})
	}
	return m
}

// Enclosing returns the symbol whose span contains line. When spans nest, the
// one with the largest start line wins (Graft grep.ts rule: innermost in
// enclosing sense).
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
// overlapping that range (Graft blast.ts rule).
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
