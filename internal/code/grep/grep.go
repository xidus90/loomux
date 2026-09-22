package grep

import (
	"fmt"
	"regexp"
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
}

// Result summarizes grep matches and groupings.
type Result struct {
	Pattern       string  `json:"pattern"`
	FilesSearched int     `json:"files_searched"`
	TotalHits     int     `json:"total_hits"`
	Groups        []Group `json:"groups"`
	TruncatedHits int     `json:"truncated_hits"`
	Unreadable    int     `json:"unreadable"`
}

// FileReader abstracts reading file content.
type FileReader func(path string) ([]byte, error)

// Search executes a symbol-coupled regex search across indexed files.
// Matches are capped at 160 runes per line, grouped by enclosing symbol
// (via model.Enclosing), and ranked stably by inDegree descending then path ascending.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/grep.ts.
func Search(g *model.Graph, x *blast.Index, spans map[string][]model.SymbolSpan, pattern string, opts Options, read FileReader) (Result, error) {
	if pattern == "" {
		return Result{}, fmt.Errorf("empty search pattern")
	}
	if g == nil || read == nil {
		return Result{Pattern: pattern}, nil
	}

	maxHits := opts.MaxHits
	if maxHits <= 0 {
		maxHits = DefaultMaxHits
	}

	var re *regexp.Regexp
	if opts.Fixed {
		p := regexp.QuoteMeta(pattern)
		if opts.IgnoreCase {
			p = "(?i)" + p
		}
		re = regexp.MustCompile(p)
	} else {
		if strings.Contains(pattern, "(?=") || strings.Contains(pattern, "(?!") ||
			strings.Contains(pattern, "(?<=") || strings.Contains(pattern, "(?<!") {
			return Result{}, fmt.Errorf("lookaround not supported in RE2 regex: %s", pattern)
		}
		for i := 1; i <= 9; i++ {
			if strings.Contains(pattern, fmt.Sprintf(`\%d`, i)) {
				return Result{}, fmt.Errorf("backreference not supported in RE2 regex: %s", pattern)
			}
		}
		p := pattern
		if opts.IgnoreCase {
			p = "(?i)" + p
		}
		var err error
		re, err = regexp.Compile(p)
		if err != nil {
			return Result{}, fmt.Errorf("invalid regex: %w", err)
		}
	}

	filesSet := make(map[string]bool)
	for i := range g.Nodes {
		filesSet[g.Nodes[i].Path] = true
	}

	normIn := lexicon.NormalizePrefix(opts.In)
	if normIn != "" {
		prefixFound := false
		for p := range filesSet {
			if lexicon.UnderPrefix(p, normIn) {
				prefixFound = true
				break
			}
		}
		if !prefixFound {
			return Result{}, fmt.Errorf("prefix not indexed: %s", opts.In)
		}
	}

	var files []string
	for p := range filesSet {
		if normIn == "" || lexicon.UnderPrefix(p, normIn) {
			files = append(files, p)
		}
	}
	sort.Strings(files)

	if spans == nil {
		spans = model.FileSpans(g)
	}

	res := Result{Pattern: pattern}
	type groupKey struct {
		path     string
		symbolID model.NodeID
	}
	groupsMap := make(map[groupKey]*Group)
	var groupsOrder []groupKey

	for _, filePath := range files {
		data, err := read(filePath)
		if err != nil {
			res.Unreadable++
			continue
		}
		res.FilesSearched++

		lines := strings.Split(string(data), "\n")
		fileSpans := spans[filePath]

		for lineIdx, rawLine := range lines {
			lineNum := lineIdx + 1
			line := strings.TrimSuffix(rawLine, "\r")
			if !re.MatchString(line) {
				continue
			}

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

			encNode := model.Enclosing(fileSpans, lineNum)
			var symID model.NodeID
			var inDeg int
			if encNode != nil {
				symID = encNode.ID
				if x != nil {
					inDeg = x.InDegree(encNode.ID)
				}
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
