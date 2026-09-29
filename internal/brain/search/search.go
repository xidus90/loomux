package search

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// NoMatches is the agreed string when a search returns no hits on either surface.
const NoMatches = "no matches"

var unsafeScope = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// CollectionName maps an area's scope to its qmd collection name.
//
// Slashes and unsafe characters become dashes, and leading/trailing dashes are stripped,
// matching the Python implementation in paths.collection_name.
func CollectionName(scope string) string {
	return strings.Trim(unsafeScope.ReplaceAllString(scope, "-"), "-")
}

// SearchAnswer contains the assembled search hits and any diagnostic findings.
type SearchAnswer struct {
	Hits     []SearchHit
	Findings []string
}

// ExecuteSearch runs a search against the areas this channel may see, as core.search does.
//
// The registry comes from registryDir, which is also the state directory the manifest
// and the register of read-only areas and the reconcile stamp are read from.
// fallbackDir -- ultra-brain's -- is asked only as long as nothing lies there.
// Scope can be "all" to query all visible areas, or a specific area scope; an unknown one
// -- or one invisible on this channel, which for the caller is the same thing -- is
// privacy.VisibleAreas' refusal.
//
// The channel decides which areas are asked about, and it does so *before*
// the engine is handed anything. Filtering the answer afterwards would mean
// the invisible area was queried, and the question is already the disclosure
// of the question (arch spec 7.2.1). That is the whole difference between
// this and a search that hides its results.
//
// Findings come in core.search's order: the twice-empty answer, the register findings in hit
// order, the withheld count, the aged stamp.
func ExecuteSearch(query, scope string, profile Profile, n int, channel privacy.Channel, port SearchPort, registryDir, fallbackDir string, now time.Time) (*SearchAnswer, error) {
	areas, err := privacy.VisibleAreas(registryDir, fallbackDir, scope, channel)
	if err != nil {
		return nil, err
	}
	if len(areas) == 0 {
		// Not an optimisation. An empty collection list drops qmd's
		// collection filter altogether, so the engine would search
		// everything it holds -- the local_only areas included.
		return &SearchAnswer{Hits: nil, Findings: nil}, nil
	}

	collections := make([]string, 0, len(areas))
	for _, visible := range areas {
		collections = append(collections, CollectionName(visible.Area.Scope))
	}
	sort.Strings(collections)

	hits, findings, err := askTwice(port, query, collections, profile, n)
	if err != nil {
		return nil, err
	}
	answer, err := assemble(hits, areas, registryDir, fallbackDir, n)
	if err != nil {
		return nil, err
	}
	stale, err := StaleReconcile(registryDir, fallbackDir, now)
	if err != nil {
		return nil, err
	}
	findings = append(findings, answer.Findings...)
	findings = append(findings, stale...)
	return &SearchAnswer{Hits: answer.Hits, Findings: findings}, nil
}

// askTwice is core._ask: one retry on an empty answer, then a finding instead of a silence.
func askTwice(port SearchPort, query string, collections []string, profile Profile, n int) ([]SearchHit, []string, error) {
	for range 2 {
		hits, err := port.Search(query, collections, profile, n)
		if err != nil {
			return nil, nil, err
		}
		if len(hits) > 0 {
			return hits, nil, nil
		}
	}
	return nil, []string{fmt.Sprintf(
		"the search engine answered empty twice in a row on profile %s; "+
			"an empty answer to a meaning search is practically unreachable, so this is "+
			"more likely a silent failure of the engine than an absence of matches (spec 16.14)",
		profile,
	)}, nil
}

// assemble is core._assemble. The registers of all asked areas are read before the first hit
// is looked at, so a broken one fails the search whichever area the hits come from.
//
// A hit from a collection this channel has no area for, or under `[privacy] never`, is only
// counted: a path under `never` is the one whose name may not leave the machine, so it must
// not reach a finding either -- which is why both reasons share the single count and why the
// check comes before the register is so much as looked at. A hit inside the tree of an area
// the channel hides -- a `local_only` wiki nested in the collection's area -- joins that count
// for the same reason: asking the enclosing collection cannot be avoided, naming the hit can.
// The wording stays the reference's. Every kept hit is checked against
// its register, and the list is cut to n only at the end.
//
// The first of the two reasons waits for a port that reports a collection nobody asked for.
// The MCP port, which all three profiles use, never delivers one: splitPath relabels a hit
// whose path begins with no asked collection as the first one asked (as the reference's
// _split does), so such a hit arrives under a scope this channel does have -- and is then
// held against that scope's `never` globs. The arm guards the ports that do not relabel.
func assemble(hits []SearchHit, areas []privacy.VisibleArea, stateDir, fallbackDir string, n int) (*SearchAnswer, error) {
	byCollection := make(map[string]privacy.VisibleArea, len(areas))
	for _, visible := range areas {
		byCollection[CollectionName(visible.Area.Scope)] = visible
	}
	registers := make(map[string]map[string]identity.Identity, len(areas))
	for _, visible := range areas {
		register, err := identity.ReadIdentities(filepath.Join(config.ResolvedAreaDir(visible.Area, stateDir, fallbackDir), "_identities.tsv"))
		if err != nil {
			return nil, err
		}
		registers[CollectionName(visible.Area.Scope)] = register
	}

	var findings []string
	var ordered []SearchHit
	dropped := 0
	for _, hit := range hits {
		area, known := byCollection[hit.Collection]
		if !known || !privacy.IsReadable(area.Manifest, hit.Relative) || area.Conceals(hit.Relative) {
			dropped++
			continue
		}
		if _, listed := registers[hit.Collection][hit.Relative]; !listed {
			findings = append(findings, fmt.Sprintf("%s/%s: hit is not in the register; reindex to catch up", area.Area.Scope, hit.Relative))
		}
		hit.Scope = area.Area.Scope
		ordered = append(ordered, hit)
	}

	if dropped > 0 {
		findings = append(findings, fmt.Sprintf(
			"%d of the engine's hits were withheld -- excluded by [privacy] never, "+
				"or from a collection this channel has no area for; the list is that many "+
				"places shorter than it could have been",
			dropped,
		))
	}
	if len(ordered) > n {
		ordered = ordered[:max(n, 0)]
	}
	return &SearchAnswer{Hits: ordered, Findings: findings}, nil
}

// FormatSearch renders a SearchAnswer as cli._print_search prints it: the snippet split like
// str.splitlines, so an empty snippet prints no line and a trailing newline no extra one.
func FormatSearch(answer *SearchAnswer) string {
	if len(answer.Hits) == 0 {
		return NoMatches + "\n"
	}

	var sb strings.Builder
	for _, hit := range answer.Hits {
		fmt.Fprintf(&sb, "brain://%s/%s:%d  %.0f%%  %s\n", hit.Scope, hit.Relative, hit.Line, hit.Score*100, hit.Title)
		for _, line := range pytext.SplitLines(hit.Snippet) {
			fmt.Fprintf(&sb, "    %s\n", line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
