package grep

import (
	"fmt"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
)

// Hit is one matching line inside a file.
type Hit struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

// Group clusters matching lines belonging to the same enclosing symbol (or file level).
type Group struct {
	Symbol   *model.Node `json:"symbol,omitempty"`
	Path     string      `json:"path"`
	InDegree int         `json:"in_degree"`
	Hits     []Hit       `json:"hits"`
}

// DefaultMaxHits is the cap on reported lines when Options.MaxHits is not positive.
const DefaultMaxHits = 300

// Options configure the grep search.
type Options struct {
	IgnoreCase bool
	Fixed      bool
	In         string
	MaxHits    int // DefaultMaxHits when not positive

	// Files is the search corpus, repo-relative and sorted. nil keeps the old
	// behaviour of searching the paths the graph's nodes sit in.
	Files []string
	// All is the corpus before --glob narrowed it, used only to tell a prefix
	// that has no file at all from one whose files the glob filtered out.
	// nil means Files is the whole corpus.
	All []string
	// Candidates limits the scan to these paths; nil scans every file in
	// Files. The index supplies it, and the scan path is the same list with
	// nil, which is what makes the two comparable.
	Candidates map[string]bool
	// Index names what the answer rests on: used, scan, stale, none, overlay
	// or trusted. The caller decides it; Search only carries it into the
	// result.
	Index string
	// Spans and InDegree, when set, supply the symbols and their callers in
	// place of the spans map and the blast index; g may then be nil. Search
	// calls them from one goroutine only.
	Spans    func(path string) []model.SymbolSpan
	InDegree func(model.NodeID) int
}

// Result summarizes grep matches and groupings.
type Result struct {
	Pattern       string  `json:"pattern"`
	FilesSearched int     `json:"files_searched"`
	TotalHits     int     `json:"total_hits"`
	Groups        []Group `json:"groups"`
	TruncatedHits int     `json:"truncated_hits"`
	Unreadable    int     `json:"unreadable"`

	// Index says what the answer rested on: used, scan, stale, none, overlay
	// or trusted.
	Index string `json:"index"`
	// IndexBuilt is when a trusted index was written (RFC 3339); empty
	// otherwise. The caller sets it.
	IndexBuilt string `json:"index_built,omitempty"`
	// FilesInScope is how many files the In filter left of the corpus.
	FilesInScope int `json:"files_in_scope"`
	// Candidates is how many files of the scope the scan reads: the index's
	// narrowing, or the whole scope when no index narrows the pattern.
	Candidates int `json:"candidates"`
	// SkippedBinary and SkippedSize count what the corpus left out; Ignored
	// counts the files git's ignore rules kept out of it.
	SkippedBinary int `json:"skipped_binary"`
	SkippedSize   int `json:"skipped_size"`
	Ignored       int `json:"ignored"`
}

// FileReader abstracts reading file content.
type FileReader func(path string) ([]byte, error)

// Search executes a symbol-coupled regex search across indexed files.
// Matches are capped at 160 runes per line, grouped by enclosing symbol
// (via model.Enclosing), and ranked stably by inDegree descending then path ascending.
//
// Ported from src/graph/grep.ts (MIT; origin under
// "Ported sources" in NOTICE.md).
func Search(g *model.Graph, x *blast.Index, spans map[string][]model.SymbolSpan, pattern string, opts Options, read FileReader) (Result, error) {
	if pattern == "" {
		return Result{}, fmt.Errorf("empty search pattern")
	}
	if read == nil || g == nil && opts.Files == nil {
		return Result{Pattern: pattern}, nil
	}

	maxHits := opts.MaxHits
	if maxHits <= 0 {
		maxHits = DefaultMaxHits
	}

	sc, err := compile(pattern, opts)
	if err != nil {
		return Result{}, err
	}

	all := opts.Files
	if all == nil {
		filesSet := make(map[string]bool)
		for i := range g.Nodes {
			filesSet[g.Nodes[i].Path] = true
		}
		all = make([]string, 0, len(filesSet))
		for p := range filesSet {
			all = append(all, p)
		}
		sort.Strings(all)
	}

	normIn := lexicon.NormalizePrefix(opts.In)
	if normIn != "" {
		prefixFound := false
		scopeAll := all
		if opts.All != nil {
			scopeAll = opts.All
		}
		for _, p := range scopeAll {
			if lexicon.UnderPrefix(p, normIn) {
				prefixFound = true
				break
			}
		}
		if !prefixFound {
			return Result{Pattern: pattern, Index: opts.Index}, fmt.Errorf("prefix not indexed: %s", opts.In)
		}
	}

	files := make([]string, 0, len(all))
	scope := 0
	for _, p := range all {
		if normIn != "" && !lexicon.UnderPrefix(p, normIn) {
			continue
		}
		scope++
		if opts.Candidates != nil && !opts.Candidates[p] {
			continue
		}
		files = append(files, p)
	}

	spansOf, inDegree := opts.Spans, opts.InDegree
	if spansOf == nil {
		if spans == nil {
			spans = model.FileSpans(g)
		}
		spansOf = func(p string) []model.SymbolSpan { return spans[p] }
	}
	if inDegree == nil {
		inDegree = x.InDegree
	}

	res := Result{Pattern: pattern, Index: opts.Index, FilesInScope: scope, Candidates: len(files)}
	type groupKey struct {
		path     string
		symbolID model.NodeID
	}
	groupsMap := make(map[groupKey]*Group)
	var groupsOrder []groupKey

	// The merge runs serially in file order, so the cap falls on the same
	// hits as a serial scan and Spans and InDegree never run concurrently.
	for i, sf := range sc.all(files, read) {
		filePath := files[i]
		if sf.unreadable {
			res.Unreadable++
			continue
		}
		if sf.binary {
			continue
		}
		res.FilesSearched++

		var fileSpans []model.SymbolSpan
		spansLoaded := false
		for j, lineNum := range sf.lines {
			line := sf.texts[j]
			if res.TotalHits >= maxHits {
				res.TruncatedHits++
				continue
			}
			res.TotalHits++

			runes := []rune(line)
			hitText := line
			if len(runes) > 160 {
				hitText = string(runes[:160])
			}
			hit := Hit{Line: lineNum, Text: hitText}

			if !spansLoaded {
				// Only a counted hit needs its symbol; a file whose hits all
				// fall past the cap never decodes its spans.
				fileSpans, spansLoaded = spansOf(filePath), true
			}
			encNode := model.Enclosing(fileSpans, lineNum)
			var symID model.NodeID
			var inDeg int
			if encNode != nil {
				symID = encNode.ID
				inDeg = inDegree(encNode.ID)
			}

			key := groupKey{path: filePath, symbolID: symID}
			grp, exists := groupsMap[key]
			if !exists {
				grp = &Group{
					Symbol:   encNode,
					Path:     filePath,
					InDegree: inDeg,
				}
				groupsMap[key] = grp
				groupsOrder = append(groupsOrder, key)
			}
			grp.Hits = append(grp.Hits, hit)
		}
	}

	for _, k := range groupsOrder {
		res.Groups = append(res.Groups, *groupsMap[k])
	}

	sort.SliceStable(res.Groups, func(i, j int) bool {
		if res.Groups[i].InDegree != res.Groups[j].InDegree {
			return res.Groups[i].InDegree > res.Groups[j].InDegree
		}
		if res.Groups[i].Path != res.Groups[j].Path {
			return res.Groups[i].Path < res.Groups[j].Path
		}
		if res.Groups[i].Symbol != nil && res.Groups[j].Symbol != nil {
			return res.Groups[i].Symbol.Name < res.Groups[j].Symbol.Name
		}
		return res.Groups[i].Symbol != nil
	})

	return res, nil
}

// unsupported names the first form RE2 has no answer for -- "lookaround" or
// "backreference" -- or "" when there is none. It walks the pattern instead of
// searching it, because a backslash escapes the character after it: `\\1` is
// a literal backslash and a 1, `\(?=` an optional literal parenthesis.
func unsupported(pattern string) string {
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
