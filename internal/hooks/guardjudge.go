package hooks

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/config"
)

// judge holds what the targets of one tool call are judged against. The
// protected flows are read once per call, not once per target: a brace
// expansion alone may bring 64.
type judge struct {
	root     string
	policy   config.Policy
	flows    func() ([]string, error)
	listings map[string]walked
}

// newJudge is the judge of one call at root.
func newJudge(root string, policy config.Policy) judge {
	return judge{root: root, policy: policy, flows: sync.OnceValues(func() ([]string, error) {
		return protectedFlows(root)
	}), listings: map[string]walked{}}
}

// reasons judges every target of a call, a writing tool's and a shell
// line's alike, and answers each reason as often as it matched; the caller
// keeps one of each. A shell target is spelled first (braces, globs, a
// stream name); a writing tool's path is the file it names.
func (j judge) reasons(targets []shellTarget) []string {
	var reasons []string
	for _, target := range targets {
		rels := []string{relativePath(target.path, j.root)}
		if target.shell {
			spelled, err := spellings(j.root, target.path)
			if err != nil {
				reasons = append(reasons, err.Error())
				continue
			}
			rels = spelled
		}
		for _, rel := range rels {
			if len(target.filters) > 0 {
				reasons = append(reasons, j.filteredReasons(rel, target, maxListed)...)
				continue
			}
			reasons = append(reasons, j.judgePath(rel, target.removes)...)
		}
	}
	return reasons
}

// maxListed is how many entries a filtered removal reads before the guard
// takes it for the removal of its start path whole.
const maxListed = 50000

// filteredReasons judges a removal of rel that the target's name filters
// narrow: each path on disk it takes, or rel whole past limit entries.
func (j judge) filteredReasons(rel string, target shellTarget, limit int) []string {
	hits, ok := j.listing(rel, target, limit)
	if !ok {
		return j.judgePath(rel, true)
	}
	var reasons []string
	for _, hit := range hits {
		reasons = append(reasons, j.judgePath(hit, true)...)
	}
	return reasons
}

// walked is one walk of a filtered removal, kept for the rest of the call.
type walked struct {
	hits []string
	ok   bool
}

// listing is what a filtered removal of rel takes: the paths at and below
// rel on disk, relative to the root, that one of the target's filters
// matches, each as find would print it -- rel with and without ./, or the
// start as the line wrote it, then the path below; false past limit
// entries. The many readings of a line ask the same walk, so each rel and
// target is walked once per call.
func (j judge) listing(rel string, target shellTarget, limit int) ([]string, bool) {
	key := fmt.Sprintf("%q %q %v", rel, target.start, target.filters)
	if l, ok := j.listings[key]; ok {
		return l.hits, l.ok
	}
	top := filepath.FromSlash(rel)
	if !filepath.IsAbs(top) {
		top = filepath.Join(j.root, top)
	}
	var hits []string
	seen := 0
	_ = filepath.WalkDir(top, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if seen++; seen > limit {
			return fs.SkipAll
		}
		sub, _ := filepath.Rel(top, p)
		at := path.Join(rel, filepath.ToSlash(sub))
		prints := []string{at, "./" + at, path.Join(filepath.ToSlash(target.start), filepath.ToSlash(sub))}
		if slices.ContainsFunc(target.filters, func(f nameFilter) bool { return slices.ContainsFunc(prints, f.matches) }) {
			hits = append(hits, at)
		}
		return nil
	})
	j.listings[key] = walked{hits, seen <= limit}
	return hits, seen <= limit
}

// judgePath judges one path relative to the root: loomux's own rules in any
// case -- Windows and macOS keep .LOOMUX/State/hooks and .loomux/state/hooks
// as one folder -- the project's as the project spelled them, the protected
// flow folders, and for a removal what it takes with it.
func (j judge) judgePath(rel string, removes bool) []string {
	reasons := pathReasons(builtinPathRules, rel, true)
	reasons = append(reasons, pathReasons(j.policy.Paths, rel, false)...)
	reasons = append(reasons, flowFolderReasons(rel, j.flows)...)
	if removes {
		reasons = append(reasons, j.ancestorReasons(rel)...)
	}
	return reasons
}

// ancestorReasons judges a removal of rel by what lies below it: .loomux
// takes the manifest, the run files and the flows with it, and the root (git
// clean without a path) takes everything. Copying into such a folder stays
// open; only a removal or the source of a move asks here.
func (j judge) ancestorReasons(rel string) []string {
	var reasons []string
	for _, book := range []struct {
		rules []config.PathRule
		fold  bool
	}{{builtinPathRules, true}, {j.policy.Paths, false}} {
		for _, rule := range book.rules {
			if slices.ContainsFunc(rule.Match, func(glob string) bool { return below(rel, glob, book.fold) }) {
				reasons = append(reasons, rule.Reason)
			}
		}
	}
	protected, err := j.flows()
	if err != nil {
		if below(rel, "**/"+flowsDir+"/*", true) {
			reasons = append(reasons, unreadableFlowsReason(err))
		}
		return reasons
	}
	for _, name := range protected {
		if below(rel, "**/"+flowsDir+"/"+name, true) {
			reasons = append(reasons, bundledFlowReason(name))
		}
	}
	return reasons
}

// below says whether rel is a folder above what glob keeps. A glob's fixed
// part is its path up to the element with the first glob character; rel is
// above it when that part lies below rel, or is rel itself while the glob
// goes on below it (for a glob under any directory, rel may end in that part),
// and the root is above every glob with a slash. A glob
// under any directory (**/) is also below a folder that ends in a leading
// part of it -- x/.loomux for **/.loomux/config.toml -- and nowhere else, or
// every folder would be above it. A glob without a slash (*.pem) names no
// folder and has none above it.
func below(rel, glob string, fold bool) bool {
	if fold {
		rel, glob = strings.ToLower(rel), strings.ToLower(glob)
	}
	glob, anywhere := strings.CutPrefix(glob, "**/")
	if !strings.Contains(glob, "/") {
		return false
	}
	if rel == "." {
		return true
	}
	fixed := literalPrefix(glob)
	if fixed == "" {
		return false
	}
	if strings.HasPrefix(fixed, rel+"/") {
		return true
	}
	if (rel == fixed || (anywhere && strings.HasSuffix(rel, "/"+fixed))) && fixed != glob {
		return true
	}
	for part := fixed; anywhere; {
		cut := strings.LastIndexByte(part, '/')
		if cut < 0 {
			return false
		}
		part = part[:cut]
		if strings.HasSuffix(rel, "/"+part) {
			return true
		}
	}
	return false
}

// literalPrefix is the part of glob before the element that holds its first
// glob character, without the slash; the whole glob when it holds none.
func literalPrefix(glob string) string {
	meta := strings.IndexAny(glob, "*?[")
	if meta < 0 {
		return glob
	}
	cut := strings.LastIndexByte(glob[:meta], '/')
	if cut < 0 {
		return ""
	}
	return glob[:cut]
}

// uniqueReasons keeps the first of each reason, in order: a refusal names a
// reason once per call, however many targets or rules brought it.
func uniqueReasons(reasons []string) []string {
	var out []string
	for _, r := range reasons {
		if !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	return out
}
