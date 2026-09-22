package skeleton

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
)

// Entry is one symbol definition inside a file skeleton.
type Entry struct {
	Name      string     `json:"name"`
	Kind      model.Kind `json:"kind"`
	Span      model.Span `json:"span"`
	Signature string     `json:"signature,omitempty"`
	Doc       string     `json:"doc,omitempty"`
}

// Extract extracts symbol signatures, kinds, and line spans for the file matching
// fileQuery. Resolves by exact relative path or unique basename.
//
// Only paths keep accepts take part (all when keep is nil): a rejected file
// answers like an unknown one, so it neither makes a basename ambiguous nor
// appears in an error.
//
// Returns the resolved repo-relative path and a slice of entries sorted by line span.
func Extract(g *model.Graph, fileQuery string, keep func(path string) bool) (string, []Entry, error) {
	if g == nil {
		return "", nil, fmt.Errorf("nil graph")
	}

	normQuery := strings.Trim(strings.ReplaceAll(fileQuery, `\`, "/"), "/")
	if normQuery == "" {
		return "", nil, fmt.Errorf("empty file query")
	}

	pathsSet := make(map[string]bool)
	for i := range g.Nodes {
		if p := g.Nodes[i].Path; keep == nil || keep(p) {
			pathsSet[p] = true
		}
	}

	var resolvedPath string
	if pathsSet[normQuery] {
		resolvedPath = normQuery
	} else {
		var matches []string
		for p := range pathsSet {
			if path.Base(p) == normQuery || strings.HasSuffix(p, "/"+normQuery) {
				matches = append(matches, p)
			}
		}
		sort.Strings(matches)
		if len(matches) == 0 {
			return "", nil, fmt.Errorf("file not found: %s", fileQuery)
		}
		if len(matches) > 1 {
			return "", nil, fmt.Errorf("ambiguous file query %q: matches %s", fileQuery, strings.Join(matches, ", "))
		}
		resolvedPath = matches[0]
	}

	var entries []Entry
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if n.Path != resolvedPath || n.Kind == model.KindFile {
			continue
		}
		entries = append(entries, Entry{
			Name:      n.Name,
			Kind:      n.Kind,
			Span:      n.Span,
			Signature: n.Signature,
		})
	}

	sort.SliceStable(entries, func(i, j int) bool {
		fromI, _, okI := entries[i].Span.Lines()
		fromJ, _, okJ := entries[j].Span.Lines()
		if okI && okJ {
			if fromI != fromJ {
				return fromI < fromJ
			}
			return entries[i].Name < entries[j].Name
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return entries[i].Name < entries[j].Name
	})

	return resolvedPath, entries, nil
}
