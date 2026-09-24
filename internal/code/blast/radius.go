package blast

import (
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/diff"
	"github.com/xidus90/loomux/internal/code/model"
)

// Signal says whether a test that reaches a changed area changed with it.
type Signal string

const (
	// SignalChanged: a test file that reaches the area is part of the diff.
	SignalChanged Signal = "changed"
	// SignalStale: test files reach the area, none of them is in the diff.
	SignalStale Signal = "stale"
	// SignalNone: no test file reaches the area.
	SignalNone Signal = "none"
	// SignalNA: the area is a test file itself, or no seed is behaviour.
	SignalNA Signal = "na"
)

// The caps on quoted diff lines, Graft's MAX_EVIDENCE and MAX_LINES.
const (
	MaxEvidence      = 4
	MaxEvidenceLines = 6
)

// Seed is a node a changed file's hunks touch, with its incoming walk edges.
type Seed struct {
	Node     *model.Node `json:"node"`
	InDegree int         `json:"in_degree"`
}

// Area is one changed file: what it seeds and the test signal for it.
type Area struct {
	Path   string      `json:"path"`
	Status diff.Status `json:"status"`
	Seeds  []Seed      `json:"seeds"`
	Signal Signal      `json:"signal"`
	Tests  []string    `json:"tests,omitempty"`
}

// RadiusHit is a node some changed file's walk reached.
type RadiusHit struct {
	Hit
	From []string `json:"from"` // changed files whose walk reached it
}

// Evidence quotes the diff lines inside one changed symbol.
type Evidence struct {
	Node  *model.Node `json:"node"`
	Lines []string    `json:"lines"`
	More  int         `json:"more,omitempty"`
}

// Report is the blast radius of a whole diff.
type Report struct {
	Areas        []Area      `json:"areas"`
	Hits         []RadiusHit `json:"hits"`
	Deleted      []string    `json:"deleted,omitempty"`
	Unindexed    []string    `json:"unindexed,omitempty"`
	Evidence     []Evidence  `json:"evidence,omitempty"`
	MoreEvidence int         `json:"more_evidence,omitempty"`
}

// Radius is the blast radius of a change: per changed file the symbols its
// hunks touch, what reaches them, and whether a test that reaches them
// changed too. withheld are paths of the same diff the caller keeps out of
// the report: they count as changed for the test signal and are neither
// areas, walked nor quoted.
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/blast/blast.ts, with the
// corrections of the G4 delta: the file node is a seed only when no symbol
// was hit, and it is walked as itself, not expanded.
func Radius(g *model.Graph, x *Index, changed []diff.File, withheld []string, depth Depth) Report {
	spans := model.FileSpans(g)
	files := map[string]*model.Node{}
	for i := range g.Nodes {
		if g.Nodes[i].Kind == model.KindFile {
			files[g.Nodes[i].Path] = &g.Nodes[i]
		}
	}
	inDiff := map[string]bool{}
	for _, f := range changed {
		inDiff[f.Path] = true
	}
	for _, p := range withheld {
		inDiff[p] = true
	}
	rep := Report{}
	hitAt := map[model.NodeID]int{}
	var evidenceSeeds []evidenceSeed
	for _, f := range changed {
		// The status decides, not the range: git reports a deleted file's
		// hunk as the line 1-1 like any other pure deletion.
		if f.Status == diff.Deleted {
			rep.Deleted = append(rep.Deleted, f.Path)
			continue
		}
		fileNode := files[f.Path]
		if fileNode == nil {
			rep.Unindexed = append(rep.Unindexed, f.Path)
			continue
		}
		seeds := seedsOf(spans[f.Path], f.Hunks)
		if len(seeds) == 0 {
			seeds = []*model.Node{fileNode}
		} else {
			for _, s := range seeds {
				evidenceSeeds = append(evidenceSeeds, evidenceSeed{s, f.Hunks})
			}
		}
		area := Area{Path: f.Path, Status: f.Status}
		ids := make([]model.NodeID, len(seeds))
		for i, s := range seeds {
			ids[i] = s.ID
			area.Seeds = append(area.Seeds, Seed{Node: s, InDegree: x.InDegree(s.ID)})
		}
		area.Signal, area.Tests = signal(x, f.Path, seeds, inDiff)
		rep.Areas = append(rep.Areas, area)
		for _, h := range x.Reach(ids, In, depth) {
			if h.Node == nil {
				continue
			}
			if i, seen := hitAt[h.ID]; seen {
				if h.Depth < rep.Hits[i].Depth {
					rep.Hits[i].Depth, rep.Hits[i].Relation = h.Depth, h.Relation
				}
				if !slices.Contains(rep.Hits[i].From, f.Path) {
					rep.Hits[i].From = append(rep.Hits[i].From, f.Path)
				}
				continue
			}
			hitAt[h.ID] = len(rep.Hits)
			rep.Hits = append(rep.Hits, RadiusHit{Hit: h, From: []string{f.Path}})
		}
	}
	sort.SliceStable(rep.Hits, func(i, j int) bool { return rep.Hits[i].Depth < rep.Hits[j].Depth })
	rep.Evidence, rep.MoreEvidence = evidence(evidenceSeeds)
	return rep
}

// seedsOf is the innermost symbols every hunk touches, each once, in the
// order the hunks meet them.
func seedsOf(spans []model.SymbolSpan, hunks []diff.Hunk) []*model.Node {
	var out []*model.Node
	seen := map[model.NodeID]bool{}
	for _, h := range hunks {
		for _, n := range model.Innermost(spans, h.From, h.To) {
			if !seen[n.ID] {
				seen[n.ID] = true
				out = append(out, n)
			}
		}
	}
	return out
}

// signal says whether a test covers the area and changed with it. The walk
// to the tests is always the closure: a test that reaches through a helper
// still reaches.
func signal(x *Index, path string, seeds []*model.Node, inDiff map[string]bool) (Signal, []string) {
	if IsTestPath(path) {
		return SignalNA, nil
	}
	var ids []model.NodeID
	for _, s := range seeds {
		// Behaviour is what the Go extractor emits for code that runs; a
		// struct, interface or named type has no test that calls it.
		if s.Kind == "function" || s.Kind == "method" {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 {
		return SignalNA, nil
	}
	set := map[string]bool{}
	for _, h := range x.Reach(ids, In, All) {
		if h.Node != nil && IsTestPath(h.Node.Path) {
			set[h.Node.Path] = true
		}
	}
	tests := slices.Sorted(maps.Keys(set))
	switch {
	case len(tests) == 0:
		return SignalNone, nil
	case slices.ContainsFunc(tests, func(t string) bool { return inDiff[t] }):
		return SignalChanged, tests
	}
	return SignalStale, tests
}

type evidenceSeed struct {
	node  *model.Node
	hunks []diff.Hunk
}

// evidence quotes the hunks inside the first MaxEvidence seeds' spans.
func evidence(seeds []evidenceSeed) ([]Evidence, int) {
	var out []Evidence
	for i, s := range seeds {
		if i == MaxEvidence {
			return out, len(seeds) - MaxEvidence
		}
		// Every seed came out of Innermost over FileSpans, which keeps only
		// nodes whose span reads, so the ok is always true here.
		from, to, _ := s.node.Span.Lines()
		var lines []string
		for _, h := range s.hunks {
			if h.To >= from && h.From <= to {
				lines = append(lines, h.Lines...)
			}
		}
		e := Evidence{Node: s.node, Lines: lines}
		if len(lines) > MaxEvidenceLines {
			e.Lines, e.More = lines[:MaxEvidenceLines], len(lines)-MaxEvidenceLines
		}
		out = append(out, e)
	}
	return out, 0
}

// IsTestPath reports whether path is a test file. Go only, as long as Go is
// the only extractor.
func IsTestPath(path string) bool { return strings.HasSuffix(path, "_test.go") }
