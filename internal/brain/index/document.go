package index

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatterRegex matches YAML frontmatter delineated by leading and trailing triple-dashes.
var frontmatterRegex = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n`)

// Go's standard regexp engine does not support negative lookbehinds (?<!!).
// We capture an optional leading exclamation mark to distinguish links from image embeddings.
var (
	markdownLinkRegex = regexp.MustCompile(`(!)?\[[^\]]*\]\(([^)\s]+)\)`)
	wikiLinkRegex     = regexp.MustCompile(`(!)?\[\[([^\]|#]+)`)
)

// Document represents a reduced markdown document holding only the metadata
// and link graph edges needed by cataloging, search, and indexing.
type Document struct {
	Relative    string
	Title       string
	Description string
	DocType     string
	Tags        []string
	Links       []string
}

// ParseFrontmatter parses the YAML block at the beginning of a document.
// In non-strict mode (used by the indexer), broken YAML or non-mapping values
// are gracefully discarded so unparsable notes remain indexed by filename.
// In strict mode (used by wiki linting), syntax defects and non-mappings return an error.
func ParseFrontmatter(text string, strict bool) (map[string]any, error) {
	match := frontmatterRegex.FindStringSubmatch(text)
	if match == nil {
		return map[string]any{}, nil
	}

	var raw any
	if err := yaml.Unmarshal([]byte(match[1]), &raw); err != nil {
		if strict {
			return nil, fmt.Errorf("broken frontmatter: %w", err)
		}
		return map[string]any{}, nil
	}

	mapping, ok := raw.(map[string]any)
	if !ok {
		if strict {
			return nil, errors.New("frontmatter is not a mapping")
		}
		return map[string]any{}, nil
	}

	result := make(map[string]any, len(mapping))
	for k, v := range mapping {
		result[k] = v
	}
	return result, nil
}

// ReadDocument reads a markdown file from disk, extracting its relative path,
// frontmatter metadata (or filename stem fallback), and outbound links.
func ReadDocument(filePath, rootDir string) (*Document, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	text := string(data)

	meta, _ := ParseFrontmatter(text, false)

	rel, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		return nil, err
	}

	title := ""
	if t, ok := meta["title"].(string); ok && t != "" {
		title = t
	} else {
		base := filepath.Base(filePath)
		title = strings.TrimSuffix(base, filepath.Ext(base))
	}

	desc := ""
	if d, ok := meta["description"].(string); ok {
		desc = d
	}

	docType := ""
	if dt, ok := meta["type"].(string); ok {
		docType = dt
	}

	var tags []string
	if rawTags, ok := meta["tags"].([]any); ok {
		for _, tag := range rawTags {
			tags = append(tags, fmt.Sprint(tag))
		}
	}

	linkSet := make(map[string]struct{})
	for _, m := range markdownLinkRegex.FindAllStringSubmatch(text, -1) {
		if m[1] == "" && len(m) > 2 {
			linkSet[m[2]] = struct{}{}
		}
	}
	for _, m := range wikiLinkRegex.FindAllStringSubmatch(text, -1) {
		if m[1] == "" && len(m) > 2 {
			target := strings.TrimSpace(m[2])
			if target != "" {
				linkSet[target] = struct{}{}
			}
		}
	}

	links := make([]string, 0, len(linkSet))
	for link := range linkSet {
		links = append(links, link)
	}
	sort.Strings(links)

	return &Document{
		Relative:    filepath.ToSlash(rel),
		Title:       title,
		Description: desc,
		DocType:     docType,
		Tags:        tags,
		Links:       links,
	}, nil
}
