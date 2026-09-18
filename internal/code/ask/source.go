package ask

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
)

// maxSpanLines caps an inlined excerpt. The reference's figure.
//
// There is no crux here, and that is not a gap: Graft's crux is an LLM-chosen
// excerpt stored on the node (its Tier 2), and this binary has no LLM in the
// path. The line slice below is Graft's OWN fallback for a node without one,
// and when a producer for a crux ever exists it takes precedence in exactly
// this function, with the flags unchanged.
const maxSpanLines = 80

// Inline attaches the source of each hit's span.
//
// Never fatal: a file deleted or shortened since the build is drift, and a hit
// then keeps its location and loses only its excerpt. An answer that fails
// because a file moved would be worse than one that is merely thinner.
func Inline(root string, a *Answer, full bool) {
	for i := range a.Hits {
		h := &a.Hits[i]
		from, to, ok := h.Span.Lines()
		if !ok {
			continue
		}
		code, ok := slice(filepath.Join(root, filepath.FromSlash(h.Path)), from, to, full, h.Path, h.Span)
		if !ok {
			continue
		}
		h.Code = code
	}
}

// slice reads lines [from, to] of a file, capped unless full.
func slice(abs string, from, to int, full bool, rel string, span model.Span) (string, bool) {
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", false
	}
	lines := strings.Split(string(b), "\n")
	if from < 1 {
		from = 1
	}
	// A stale graph against an edited file: clamp rather than answer nothing.
	if to > len(lines) {
		to = len(lines)
	}
	if from > to {
		return "", false
	}
	out := lines[from-1 : to]
	if !full && len(out) > maxSpanLines {
		rest := len(out) - maxSpanLines
		out = append(out[:maxSpanLines:maxSpanLines],
			fmt.Sprintf("... (+%d more lines; open %s:%s)", rest, rel, span))
	}
	return strings.Join(out, "\n"), true
}
