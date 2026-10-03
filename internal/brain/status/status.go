// Package status answers `brain status`: one line per thing a reader should
// know before trusting an answer, rule for rule after `core.status` of the
// Python reference (src/brain/core.py:328-532). Every line but the first
// reports a defect, so a vault in order answers with its reconcile stamp
// alone.
package status

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// Lines answers `brain status` on one channel. The registry comes from
// registryDir, which is also the state directory the reconcile stamp and the
// artefacts and manifests of read-only areas come from.
//
// The steps run in the order Python evaluates them, and that order decides
// more than the order of the lines: of two defects, the one the reference
// meets first is the one that aborts. So the stamp is read before the
// registry, an area's graph before its register, every register a second
// time for the shared hashes after the loop, and the backlog is asked last.
func Lines(ch privacy.Channel, port search.SearchPort, registryDir string, now time.Time) ([]string, error) {
	first, err := lastReconcile(registryDir, now)
	if err != nil {
		return nil, err
	}
	lines := []string{first}
	areas, err := privacy.VisibleAreas(registryDir, "all", ch)
	if err != nil {
		return nil, err
	}
	for _, visible := range areas {
		area, manifest := visible.Area, visible.Manifest
		if len(manifest.IndexInclude) > 1 {
			lines = append(lines, fmt.Sprintf("%s: the search engine sees only %s; also declared: %s",
				area.Scope, manifest.IndexInclude[0], strings.Join(manifest.IndexInclude[1:], ", ")))
		}
		// Path.exists() is os.path.exists, which calls every failed stat
		// absent. Only a read-only area gets here with its path gone: a
		// writable one keeps its manifest under that path, and VisibleAreas
		// has refused it already.
		if _, err := os.Stat(area.Path); err != nil {
			lines = append(lines, fmt.Sprintf("%s: %s does not exist; skipped",
				area.Scope, pytext.PathString(runtime.GOOS, area.Path)))
			continue
		}
		if _, err := os.Stat(filepath.Join(config.ManifestDir(area, registryDir), "graph.json")); err != nil {
			lines = append(lines, fmt.Sprintf("%s: never indexed; run `brain reindex`", area.Scope))
			continue
		}
		g, err := graph.ReadGraph(area, registryDir)
		if err != nil {
			return nil, err
		}
		if links := g.Links; links.Total != 0 && links.Resolved*2 < links.Total {
			lines = append(lines, fmt.Sprintf("%s: only %d of %d links resolved (%s)",
				area.Scope, links.Resolved, links.Total, dropped(links.Dropped)))
		}
		missing, err := unfindable(visible, port, registryDir)
		if err != nil {
			return nil, err
		}
		lines = append(lines, missing...)
	}
	shared, err := sharedHashes(areas, registryDir)
	if err != nil {
		return nil, err
	}
	lines = append(lines, shared...)
	return append(lines, notYetSearchable(port)...), nil
}

// lastReconcile is `_last_reconcile`: the stamp is named whether or not it
// is old, because "when was this last checked" has no in-order value to be
// silent about. A missing, unparsable or naive stamp is one answer.
func lastReconcile(stateDir string, now time.Time) (string, error) {
	stamp, ok, err := search.ReadLastRun(stateDir)
	if err != nil {
		return "", err
	}
	if !ok {
		return "last reconcile: never; " + search.ReconcileAdvice, nil
	}
	if search.Stale(stamp, now) {
		return fmt.Sprintf("last reconcile: %s; older than 24 h, %s", pytext.IsoFormat(stamp), search.ReconcileAdvice), nil
	}
	return "last reconcile: " + pytext.IsoFormat(stamp), nil
}

// dropped renders the reasons table of L5 as Python joins
// `sorted(dropped_counts.items())`: by key, `key=value`, and nothing at all
// between the parentheses when the table is empty.
func dropped(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, key := range keys {
		parts[i] = fmt.Sprintf("%s=%d", key, counts[key])
	}
	return strings.Join(parts, ", ")
}

// registerPath is `area_artifact_dir(area, state_dir) / "_identities.tsv"`
// spelt as str(Path) spells it: the register names this path in its error
// messages, and those end up on the reader's screen.
func registerPath(area config.Area, stateDir string) string {
	return pytext.PathString(runtime.GOOS, config.ManifestDir(area, stateDir)+"/_identities.tsv")
}

// unfindable is `_unfindable`: documents the register holds and the engine
// does not list. Paths under `never` or `unsearched` are meant to be
// unfindable and stay out of both counts. The engine is asked even for an
// empty register, and an engine that does not answer is a line, not an
// abort. sort.Strings orders valid UTF-8 by code point, as Python's sorted
// orders str. A path inside the tree of an area the channel hides is neither
// counted nor named: the register of an area whose tree holds a `local_only`
// wiki lists that wiki's pages as well.
func unfindable(visible privacy.VisibleArea, port search.SearchPort, stateDir string) ([]string, error) {
	area, manifest := visible.Area, visible.Manifest
	register, err := identity.ReadIdentities(registerPath(area, stateDir))
	if err != nil {
		return nil, err
	}
	ours := make([]string, 0, len(register))
	for relative := range register {
		if privacy.IsReadable(manifest, relative) && !privacy.MatchesGlobs(manifest.IndexUnsearched, relative) &&
			!visible.Conceals(relative) {
			ours = append(ours, relative)
		}
	}
	sort.Strings(ours)
	listed, err := port.Indexed(search.CollectionName(area.Scope))
	if err != nil {
		return []string{fmt.Sprintf("%s: the search engine did not answer (%v); its index was not compared", area.Scope, err)}, nil
	}
	// Both sides in NFC: a name in NFD and in NFC is one file to the file
	// system and two strings to a comparison. The line keeps the register's
	// own spelling.
	theirs := make(map[string]bool, len(listed))
	for _, relative := range listed {
		theirs[pytext.NFC(relative)] = true
	}
	var missing []string
	for _, relative := range ours {
		if !theirs[pytext.NFC(relative)] {
			missing = append(missing, relative)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	tail := ""
	if len(missing) > 3 {
		tail = ", …" // U+2026, as Python writes it
	}
	return []string{fmt.Sprintf("%s: %d of %d indexed documents are unknown to the search engine, so `brain search` can never return them; e.g. %s%s",
		area.Scope, len(missing), len(ours), strings.Join(missing[:min(3, len(missing))], ", "), tail)}, nil
}

// sharedHashes is `_shared_hashes`: identical bytes under several paths,
// looked for across every visible area, the skipped ones included. A path
// under `never` is counted but never named, so a pair that lost one half to
// it names the survivor alone. A path inside the tree of an area the channel
// hides is not even counted: the survivor's line would tell that a hidden
// twin exists, which is what hiding the area keeps back. Lines follow the hash string, the paths inside
// a line their own order.
func sharedHashes(areas []privacy.VisibleArea, stateDir string) ([]string, error) {
	byHash := map[string][]string{}
	withheld := map[string]int{}
	for _, visible := range areas {
		register, err := identity.ReadIdentities(registerPath(visible.Area, stateDir))
		if err != nil {
			return nil, err
		}
		for _, entry := range register {
			if visible.Conceals(entry.Relative) {
				continue
			}
			if !privacy.IsReadable(visible.Manifest, entry.Relative) {
				withheld[entry.ContentHash]++
				continue
			}
			byHash[entry.ContentHash] = append(byHash[entry.ContentHash], visible.Area.Scope+"/"+entry.Relative)
		}
	}
	digests := make([]string, 0, len(byHash))
	for digest := range byHash {
		digests = append(digests, digest)
	}
	sort.Strings(digests)
	var lines []string
	for _, digest := range digests {
		paths := byHash[digest]
		if len(paths)+withheld[digest] < 2 {
			continue
		}
		if len(paths) == 1 {
			lines = append(lines, paths[0]+": same content hash as a path excluded by [privacy] never")
			continue
		}
		sort.Strings(paths)
		lines = append(lines, fmt.Sprintf("same content hash under %d paths: %s", len(paths), strings.Join(paths, ", ")))
	}
	return lines, nil
}

// notYetSearchable is `_not_yet_searchable`: the engine's backlog, asked
// once for everything it holds, and a line instead of an abort when it does
// not answer.
func notYetSearchable(port search.SearchPort) []string {
	pending, err := port.NotYetSearchable()
	if err != nil {
		return []string{fmt.Sprintf("the search engine did not answer (%v); the backlog was not counted", err)}
	}
	if pending == 0 {
		return nil
	}
	return []string{fmt.Sprintf("%d documents are indexed but not yet searchable; run `brain embed`", pending)}
}
