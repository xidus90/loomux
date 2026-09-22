package blast

import (
	"fmt"
	"path"
	"strings"

	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
)

// Resolve finds nodes in g that match query, optionally scoped to in prefix.
//
// Matches in three stages:
//  1. Exact name or ID suffix (#query or .query), case-insensitively, with
//     ordinals (~2) stripped at segment boundaries.
//     If query contains a dot (<pkg>.<symbol>), checks whether <pkg> matches the
//     package directory of a node named <symbol> before falling back to bare name.
//  2. Bare name: last dot-separated segment as symbol name across the repo.
//  3. File node: by basename, path, or path suffix. File path queries are
//     prioritized over bare name fallback to protect files like runner.go from
//     being captured by a symbol named go.
//
// An empty or unindexed in prefix produces an error (assertPrefixIndexed).
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/traverse.ts (resolveSymbol).
func Resolve(g *model.Graph, query, in string) ([]*model.Node, error) {
	if g == nil || query == "" {
		return nil, nil
	}

	normIn := lexicon.NormalizePrefix(in)
	if normIn != "" {
		prefixFound := false
		for i := range g.Nodes {
			if lexicon.UnderPrefix(g.Nodes[i].Path, normIn) {
				prefixFound = true
				break
			}
		}
		if !prefixFound {
			return nil, fmt.Errorf("prefix not indexed: %s", in)
		}
	}

	var candidates []*model.Node
	for i := range g.Nodes {
		n := &g.Nodes[i]
		if normIn == "" || lexicon.UnderPrefix(n.Path, normIn) {
			candidates = append(candidates, n)
		}
	}

	cleanQ := stripOrdinals(query)
	cleanQ = strings.ReplaceAll(cleanQ, `\`, "/")
	cleanQ = strings.TrimPrefix(strings.TrimPrefix(cleanQ, "#"), ".")
	qLower := strings.ToLower(cleanQ)

	// Stage 1: Exact name or ID suffix (#query or .query)
	var stage1Hits []*model.Node
	for _, n := range candidates {
		if n.Kind == model.KindFile {
			continue
		}
		cleanName := strings.ToLower(stripOrdinals(n.Name))
		cleanID := strings.ToLower(stripOrdinals(string(n.ID)))
		if cleanName == qLower || cleanID == qLower ||
			strings.HasSuffix(cleanID, "#"+qLower) ||
			strings.HasSuffix(cleanID, "."+qLower) {
			stage1Hits = append(stage1Hits, n)
		}
	}
	if len(stage1Hits) > 0 {
		return stage1Hits, nil
	}

	// Go package filter (Spec E3): if query has <pkg>.<symbol>, match package dir segment
	if strings.Contains(cleanQ, ".") {
		lastDot := strings.LastIndex(cleanQ, ".")
		pkgPart := strings.ToLower(cleanQ[:lastDot])
		symPart := strings.ToLower(cleanQ[lastDot+1:])

		var pkgHits []*model.Node
		for _, n := range candidates {
			if n.Kind == model.KindFile {
				continue
			}
			if strings.ToLower(stripOrdinals(n.Name)) != symPart {
				continue
			}
			dir := path.Dir(n.Path)
			dirBase := strings.ToLower(path.Base(dir))
			if dirBase == pkgPart || strings.ToLower(dir) == pkgPart || strings.HasSuffix(strings.ToLower(dir), "/"+pkgPart) {
				pkgHits = append(pkgHits, n)
			}
		}
		if len(pkgHits) > 0 {
			return pkgHits, nil
		}
	}

	// If the query is a file query, match file nodes directly without bare name fallback
	if isFileQuery(query) {
		return matchFiles(candidates, qLower), nil
	}

	// Stage 2: Bare name (last dot segment)
	if strings.Contains(cleanQ, ".") {
		lastDot := strings.LastIndex(cleanQ, ".")
		bareName := strings.ToLower(cleanQ[lastDot+1:])
		var bareHits []*model.Node
		for _, n := range candidates {
			if strings.ToLower(stripOrdinals(n.Name)) == bareName {
				bareHits = append(bareHits, n)
			}
		}
		if len(bareHits) > 0 {
			return bareHits, nil
		}
	}

	// Stage 3: File node fallback
	return matchFiles(candidates, qLower), nil
}

func matchFiles(candidates []*model.Node, qLower string) []*model.Node {
	var fileHits []*model.Node
	for _, n := range candidates {
		if n.Kind != model.KindFile {
			continue
		}
		cleanName := strings.ToLower(stripOrdinals(n.Name))
		cleanPath := strings.ToLower(n.Path)
		if cleanName == qLower || cleanPath == qLower || strings.HasSuffix(cleanPath, "/"+qLower) || strings.ToLower(string(n.ID)) == qLower {
			fileHits = append(fileHits, n)
		}
	}
	return fileHits
}

func stripOrdinals(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '~' {
			j := i + 1
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				j++
			}
			if j > i+1 {
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func isFileQuery(q string) bool {
	if strings.Contains(q, "/") || strings.Contains(q, `\`) {
		return true
	}
	ext := strings.ToLower(path.Ext(q))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".c", ".cc", ".cpp", ".cxx", ".h", ".hpp", ".java", ".json", ".toml", ".yaml", ".yml", ".md":
		return true
	default:
		return false
	}
}
