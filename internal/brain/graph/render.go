package graph

import (
	"encoding/json"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

// uriSchemeRegex matches RFC 3986 URI schemes: starts with letter, followed by letters, digits, +, -, or ., and ends with :.
var uriSchemeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// cleanTarget strips query parameters (?...) and fragment identifiers (#...) from link target.
func cleanTarget(target string) string {
	if idx := strings.IndexAny(target, "#?"); idx != -1 {
		return target[:idx]
	}
	return target
}

// InOwnWiki checks whether relative path lies in the area's bundle.
func InOwnWiki(relative string, prefix *string) bool {
	if prefix == nil {
		return false
	}
	p := *prefix
	return p == "" || relative == p || strings.HasPrefix(relative, p+"/")
}

// ResolveTarget resolves a link relative to the source document as a clean POSIX path.
// It trims query (?...) and fragment (#...) components (Anhang B fix).
// If the path climbs above the area root, it returns "", false.
func ResolveTarget(source, target string, wikiPrefix *string) (string, bool) {
	cleaned := cleanTarget(target)
	unescaped, err := url.PathUnescape(cleaned)
	if err != nil {
		unescaped = cleaned
	}

	if strings.HasPrefix(unescaped, "/") {
		inside := strings.TrimPrefix(path.Clean(unescaped), "/")
		if InOwnWiki(source, wikiPrefix) && wikiPrefix != nil && *wikiPrefix != "" {
			return path.Join(*wikiPrefix, inside), true
		}
		return inside, true
	}

	sourceDir := path.Dir(source)
	raw := path.Join(sourceDir, unescaped)
	if strings.HasPrefix(raw, "..") {
		return "", false
	}
	return raw, true
}

// DropReason returns why this link produces no edge in the graph, or nil if it does produce an edge.
// Classifies all RFC 3986 URI schemes as "external" (Anhang B fix).
func DropReason(source, link string, known map[string]bool, wikiPrefix *string) *string {
	if uriSchemeRegex.MatchString(link) {
		return stringPtr("external")
	}
	if strings.HasPrefix(link, "#") {
		return stringPtr("anchor")
	}

	target, ok := ResolveTarget(source, link, wikiPrefix)
	if !ok {
		return stringPtr("outside_area")
	}

	candidates := []string{target, target + ".md"}
	for _, c := range candidates {
		if known[c] {
			return nil
		}
	}

	return stringPtr("unknown_target")
}

// RenderGraph renders nodes, edges, and link statistics as formatted JSON with trailing newline.
func RenderGraph(scope string, docs []Document, wikiPrefix *string) ([]byte, error) {
	known := make(map[string]bool, len(docs))
	for _, doc := range docs {
		known[doc.Relative] = true
	}

	sortedDocs := make([]Document, len(docs))
	copy(sortedDocs, docs)
	sort.Slice(sortedDocs, func(i, j int) bool {
		return sortedDocs[i].Relative < sortedDocs[j].Relative
	})

	nodes := make([]Node, len(sortedDocs))
	for i, doc := range sortedDocs {
		tags := doc.Tags
		if tags == nil {
			tags = []string{}
		}
		nodes[i] = Node{
			ID:    doc.Relative,
			Title: doc.Title,
			Tags:  tags,
		}
	}

	edges := []Edge{}
	seen := make(map[string]bool)
	counts := make(map[string]int)
	total := 0
	resolved := 0

	for _, doc := range sortedDocs {
		for _, link := range doc.Links {
			total++
			reason := DropReason(doc.Relative, link, known, wikiPrefix)
			if reason == nil {
				resolved++
			} else {
				counts[*reason]++
			}

			target, ok := ResolveTarget(doc.Relative, link, wikiPrefix)
			if !ok {
				continue
			}

			candidates := []string{target, target + ".md"}
			for _, match := range candidates {
				if known[match] {
					key := doc.Relative + "->" + match
					if !seen[key] {
						seen[key] = true
						edges = append(edges, Edge{
							From: doc.Relative,
							To:   match,
						})
					}
					break
				}
			}
		}
	}

	g := Graph{
		Scope: scope,
		Nodes: nodes,
		Edges: edges,
		Links: LinksInfo{
			Total:    total,
			Resolved: resolved,
			Dropped:  counts,
		},
	}

	data, _ := json.MarshalIndent(g, "", "  ")
	return append(data, '\n'), nil
}

func stringPtr(s string) *string {
	return &s
}
