package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// sweepPage is one page as `src/brain/wiki/page.py` reads it for the lint
// over areas. It is not WikiPage, whose reader is the Go form of 1a and
// answers differently where the sweep's output depends on it: the links are a
// sorted set of Markdown and `[[wiki]]` links from the whole text, conflict
// boxes inside fenced code are not counted, and a value of the wrong kind
// reads as absent rather than breaking the page.
type sweepPage struct {
	relative          string
	pageType          *string
	sources           []sweepSource
	links             []string
	declaredConflicts *int
	// declaredSaid is the declared count as Python prints it: a bool is an
	// int there and compares as one, but reads `True` or `False`.
	declaredSaid      string
	foundConflicts    int
	staleAfter        string // YYYY-MM-DD, or "" when absent
	mtime             time.Time
	brokenFrontmatter *string
	realization       *string
	implementedIn     *string
}

// sweepSource is `SourceRef`: every field present or empty, the revision an
// integer or absent.
type sweepSource struct {
	id, resource, docID, contentHash string
	revision                         *int
}

// sweepPatterns are the reader's regular expressions, built on first use and
// not at package load: this package sits on the edit path.
type sweepPatterns struct {
	frontmatter, markdownLink, wikiLink, conflict, fence *regexp.Regexp
}

func newSweepPatterns() sweepPatterns {
	return sweepPatterns{
		// `document._FRONTMATTER`: DOTALL, no multiline, and a closing fence
		// that must end in a line break.
		frontmatter: regexp.MustCompile(`\A---\r?\n((?s:.*?))\r?\n---\r?\n`),
		// `_MARKDOWN_LINK` and `_WIKI_LINK` without their `(?<!!)`, which
		// RE2 cannot express; findRefused applies it.
		markdownLink: regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`),
		wikiLink:     regexp.MustCompile(`\[\[([^\]|#]+)`),
		conflict:     regexp.MustCompile(`(?im)^[ \t]*>\s*\[!conflict\]`),
		fence:        regexp.MustCompile("(?m)^[ \\t]*(```|~~~)"),
	}
}

// readSweepPage is `read_page`. The text is read as Python reads it -- UTF-8
// with each undecodable byte replaced, line endings folded to LF -- and a
// frontmatter that does not parse is carried as a finding, not an error.
func readSweepPage(path, root string, p sweepPatterns) (sweepPage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sweepPage{}, err
	}
	text := strings.ReplaceAll(strings.ReplaceAll(pytext.DecodeReplace(data), "\r\n", "\n"), "\r", "\n")
	rel, _ := filepath.Rel(root, path)
	page := sweepPage{
		relative:       filepath.ToSlash(rel),
		links:          sweepLinks(text, p),
		foundConflicts: countConflicts(text, p),
	}
	// After a successful read, a failing stat means the file went away in
	// between. The zero time is older than every threshold, so the age rules
	// speak rather than stay silent, as they do for ReadPage.
	if info, err := os.Stat(path); err == nil {
		page.mtime = info.ModTime()
	}
	meta, broken := parseSweepFrontmatter(text, p)
	if broken != "" {
		page.brokenFrontmatter = &broken
		return page, nil
	}
	page.pageType = asString(meta["type"])
	page.sources = asSources(meta["sources"])
	page.declaredConflicts = asInt(meta["open_conflicts"])
	page.declaredSaid = saidInt(meta["open_conflicts"])
	page.staleAfter = asDate(meta["stale_after"])
	page.realization = asString(meta["realization"])
	page.implementedIn = asString(meta["implemented_in"])
	return page, nil
}

// parseSweepFrontmatter is `parse_frontmatter(text, strict=True)`: the block's
// keys, or the reason it is broken. No block at all is an empty mapping.
func parseSweepFrontmatter(text string, p sweepPatterns) (map[string]any, string) {
	match := p.frontmatter.FindStringSubmatch(text)
	if match == nil {
		return map[string]any{}, ""
	}
	var loaded any
	if err := yaml.Unmarshal([]byte(match[1]), &loaded); err != nil {
		return nil, err.Error()
	}
	switch m := loaded.(type) {
	case map[string]any:
		return m, ""
	case map[any]any:
		// Keys that are no strings are rebuilt as their text, as `str(key)`.
		out := map[string]any{}
		for key, value := range m {
			out[fmt.Sprint(key)] = value
		}
		return out, ""
	}
	return nil, "frontmatter is not a mapping"
}

// sweepLinks is `_links`: every Markdown and wiki link target of the text,
// deduplicated and sorted.
func sweepLinks(text string, p sweepPatterns) []string {
	seen := map[string]bool{}
	for _, target := range findRefused(text, p.markdownLink) {
		seen[target] = true
	}
	for _, target := range findRefused(text, p.wikiLink) {
		seen[strings.TrimSpace(target)] = true
	}
	links := make([]string, 0, len(seen))
	for target := range seen {
		links = append(links, target)
	}
	slices.Sort(links)
	return links
}

// findRefused is re's `finditer` with the `(?<!!)` of the Python pattern: a
// match whose `[` follows a `!` is refused, and the search goes on one byte
// later, where Python's engine tries the next position.
func findRefused(text string, re *regexp.Regexp) []string {
	var out []string
	for from := 0; from < len(text); {
		at := re.FindStringSubmatchIndex(text[from:])
		if at == nil {
			break
		}
		start := from + at[0]
		if start > 0 && text[start-1] == '!' {
			from = start + 1
			continue
		}
		out = append(out, text[from+at[2]:from+at[3]])
		from += at[1]
	}
	return out
}

// countConflicts is `count_conflicts`: conflict boxes outside fenced code. An
// unclosed fence reaches to the end of the text.
func countConflicts(text string, p sweepPatterns) int {
	var spans [][2]int
	open := -1
	for _, at := range p.fence.FindAllStringIndex(text, -1) {
		if open < 0 {
			open = at[0]
			continue
		}
		spans = append(spans, [2]int{open, at[1]})
		open = -1
	}
	if open >= 0 {
		spans = append(spans, [2]int{open, len(text)})
	}
	count := 0
	for _, at := range p.conflict.FindAllStringIndex(text, -1) {
		inside := false
		for _, span := range spans {
			if span[0] <= at[0] && at[0] < span[1] {
				inside = true
			}
		}
		if !inside {
			count++
		}
	}
	return count
}

func asString(value any) *string {
	if s, ok := value.(string); ok {
		return &s
	}
	return nil
}

func asStringOrEmpty(value any) string {
	s, _ := value.(string)
	return s
}

// asInt is `_as_int_or_none`. Python's bool is an int, so `true` reads as 1.
func asInt(value any) *int {
	switch v := value.(type) {
	case int:
		return &v
	case bool:
		n := 0
		if v {
			n = 1
		}
		return &n
	}
	return nil
}

// saidInt is how Python's f-string prints what asInt read.
func saidInt(value any) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "True"
		}
		return "False"
	default:
		return fmt.Sprint(v)
	}
}

// asDate is `_as_date_or_none` as the day it names: YAML gives an ISO day as a
// timestamp, anything else is absent.
func asDate(value any) string {
	if t, ok := value.(time.Time); ok {
		return t.Format("2006-01-02")
	}
	return ""
}

func asSources(value any) []sweepSource {
	entries, ok := value.([]any)
	if !ok {
		return nil
	}
	sources := make([]sweepSource, 0, len(entries))
	for _, entry := range entries {
		fields, _ := entry.(map[string]any)
		sources = append(sources, sweepSource{
			id:          asStringOrEmpty(fields["id"]),
			resource:    asStringOrEmpty(fields["resource"]),
			docID:       asStringOrEmpty(fields["doc_id"]),
			contentHash: asStringOrEmpty(fields["content_hash"]),
			revision:    asInt(fields["revision"]),
		})
	}
	return sources
}
