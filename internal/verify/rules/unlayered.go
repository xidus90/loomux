package rules

import (
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/verify"
)

// Unlayered reports every source file that belongs to no layer, sorted by
// path. A file in a layer with checked = false is in a layer.
func Unlayered(rels []string, m *Map) []Finding {
	var out []Finding
	for _, rel := range rels {
		if _, ok := m.Of(rel); !ok && verify.IsSource(rel) {
			out = append(out, Finding{Path: rel, Text: "is in no layer; add it to a layer in [rules.layers], or to one with checked = false"})
		}
	}
	slices.SortFunc(out, func(a, b Finding) int { return strings.Compare(a.Path, b.Path) })
	return out
}
