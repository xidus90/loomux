package hooks

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
)

// judge holds what the targets of one tool call are judged against. The
// protected flows and the resolved root are read once per call, not once per
// target: a brace expansion alone may bring 64.
type judge struct {
	root     string
	policy   config.Policy
	flows    func() ([]string, error)
	base     func() (string, error)
	listings map[string]walked
}

// newJudge is the judge of one call at root.
func newJudge(root string, policy config.Policy) judge {
	return judge{root: root, policy: policy, flows: sync.OnceValues(func() ([]string, error) {
		return protectedFlows(root)
	}), base: sync.OnceValues(func() (string, error) {
		return guard.ResolvePath(root)
	}), listings: map[string]walked{}}
}

// reasons judges every target of a call, a writing tool's and a shell
// line's alike, and answers each reason as often as it matched; the caller
// keeps one of each. A shell target is spelled first (braces, globs, a
// stream name); a writing tool's path is the file it names.
func (j judge) reasons(targets []shellTarget) []string {
	var reasons []string
	for _, target := range targets {
		rels, err := j.rels(target)
		if err != nil {
			reasons = append(reasons, err.Error())
			continue
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

// rels are the paths one target names relative to the root: a shell
// target as the shell spells it, a writing tool's path as it is, and in
// strict mode each also as the file system resolves it. An error is a
// refusal the caller names.
func (j judge) rels(target shellTarget) ([]string, error) {
	rels := []string{relativePath(target.path, j.root)}
	if target.shell {
		spelled, err := spellings(j.root, target.path)
		if err != nil {
			return nil, err
		}
		rels = spelled
	}
	if !j.policy.Strict {
		return rels, nil
	}
	resolved, err := j.resolvedSpellings(rels)
	if err != nil {
		return nil, err
	}
	return append(rels, resolved...), nil
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

// resolvedSpellings are rels as the file system names them: through
// guard.ResolvePath, which folds 8.3 aliases, a trailing dot or blank, case
// and junctions over the longest existing part, then relative to the root
// resolved the same way. No second resolver. A device name (nul, con) is no
// place to resolve and is skipped.
func (j judge) resolvedSpellings(rels []string) ([]string, error) {
	base, err := j.base()
	if err != nil {
		return nil, fmt.Errorf("loomux cannot resolve the project root %s, so it refuses in strict mode: %v", j.root, err)
	}
	var out []string
	for _, rel := range rels {
		place := filepath.FromSlash(rel)
		if !filepath.IsAbs(place) {
			place = filepath.Join(j.root, place)
		}
		full, err := guard.ResolvePath(place)
		switch {
		case err == nil:
			out = append(out, relativePath(full, base))
		case !isDevice(rel):
			return nil, fmt.Errorf("loomux cannot resolve %s, so it refuses in strict mode: %v", rel, err)
		}
	}
	return out, nil
}

// isDevice says whether rel names a Windows device, whatever its extension.
func isDevice(rel string) bool {
	name, _, _ := strings.Cut(strings.ToLower(filepath.Base(rel)), ".")
	switch name {
	case "nul", "con", "prn", "aux", "conin$", "conout$":
		return true
	}
	return len(name) == 4 && (strings.HasPrefix(name, "com") || strings.HasPrefix(name, "lpt")) && name[3] >= '1' && name[3] <= '9'
}

// strictReasons are what strict mode adds for one line: a program the guard
// does not know that names a protected path, and a write whose path holds an
// expansion that may land on one.
func (j judge) strictReasons(found []shellTarget, unknown []unknownCall) []string {
	var reasons []string
	for _, call := range unknown {
		for _, word := range namedPaths(call.args[1:]) {
			if refused, err := j.unknownReaches(call.base, word); refused || err != nil {
				reasons = append(reasons, fmt.Sprintf("in strict mode loomux refuses `%s` on %s: it does not know whether the program writes there", verbOf(call.args[0]), word))
			}
		}
	}
	lands := func(p string) bool {
		fixed, expands := beforeExpansion(p)
		return expands && j.mayReach(fixed)
	}
	for _, target := range found {
		// The fixed part is weighed with its braces unfolded, as written and
		// as the file system names it: a short name, a trailing dot or an
		// absolute path to the root hides a protected path only in the
		// first. A word that does not resolve is weighed as written here;
		// reasons, which resolves every target in strict mode, refuses it.
		words, ok := unfoldBraces(target.path)
		if !ok {
			words = []string{target.path}
		}
		spelled := slices.Clone(words)
		for _, w := range words {
			if resolved, err := j.resolvedSpellings([]string{w}); err == nil {
				spelled = append(spelled, resolved...)
			}
		}
		if slices.ContainsFunc(spelled, lands) {
			reasons = append(reasons, fmt.Sprintf("in strict mode loomux refuses a write to %s: the expansion may land on a protected path", target.path))
		}
	}
	return reasons
}

// unknownReaches says whether a word an unknown program names, under the
// place base a cd moved to, is or holds a protected path. A form that is the
// working folder itself -- ./... of go test resolved, the : a stream cut
// leaves of cut -d: -- names no removal of the root and is skipped.
func (j judge) unknownReaches(base, word string) (bool, error) {
	target := under(base, []shellTarget{{path: word, removes: true, shell: true}})[0]
	rels, err := j.rels(target)
	for _, rel := range rels {
		if rel != "." && len(j.judgePath(rel, true)) > 0 {
			return true, nil
		}
	}
	return false, err
}

// namedPaths are the paths a program's arguments may name: each word, the
// value of a flag glued with = or :, the value glued to a one-letter flag
// (-o.loomux/config.toml), and each field of a word that holds a blank --
// the string of python -c "…" among them.
func namedPaths(words []string) []string {
	var out []string
	for _, w := range words {
		if strings.HasPrefix(w, "-") {
			if at := strings.IndexAny(w, "=:"); at >= 0 {
				out = append(out, w[at+1:])
			}
			if !strings.HasPrefix(w, "--") && len(w) > 2 {
				out = append(out, w[2:])
			}
			continue
		}
		out = append(out, w)
		if strings.ContainsAny(w, " \t") {
			for _, field := range strings.Fields(w) {
				out = append(out, strings.Trim(field, `"'`))
			}
		}
	}
	return out
}

// beforeExpansion is the part of p before its first expansion -- $X, $(…),
// ${…}, a backtick, cmd's %X% -- and whether p holds one. A brace or glob
// character before it cuts the part there as well: what follows it is not
// fixed either.
func beforeExpansion(p string) (string, bool) {
	at := strings.IndexAny(p, "$`")
	if pct := strings.IndexByte(p, '%'); pct >= 0 && strings.IndexByte(p[pct+1:], '%') >= 0 && (at < 0 || pct < at) {
		at = pct
	}
	if at < 0 {
		return "", false
	}
	if glob := strings.IndexAny(p[:at], "{*?["); glob >= 0 {
		at = glob
	}
	return p[:at], true
}

// protectedGlob is one glob the guard keeps, and whether it matches in any case.
type protectedGlob struct {
	glob string
	fold bool
}

// protectedGlobs are the globs of every rule and protected flow folder.
func (j judge) protectedGlobs() []protectedGlob {
	var out []protectedGlob
	for _, rule := range builtinPathRules {
		for _, g := range rule.Match {
			out = append(out, protectedGlob{strings.ToLower(g), true})
		}
	}
	for _, rule := range j.policy.Paths {
		for _, g := range rule.Match {
			out = append(out, protectedGlob{g, false})
		}
	}
	names, err := j.flows()
	if err != nil {
		return append(out, protectedGlob{"**/" + flowsDir + "/*", true})
	}
	for _, name := range names {
		out = append(out, protectedGlob{"**/" + flowsDir + "/" + strings.ToLower(name), true})
	}
	return out
}

// mayReach says whether a path that begins with fixed may be a protected one
// or lie above one: a glob's literal part begins with fixed, or fixed begins
// with that part and a slash -- tried from the start of fixed and, for a glob
// under any directory, from every element; a glob without a slash is tried
// against fixed's last element. An empty fixed part is the barrier's
// question: the expansion may be any absolute path.
//
// A named limit: a glob that begins with a glob character (*.pem, *.key)
// has no literal part, so no expansion reaches it -- src/$X may be
// src/x.pem and passes, as it must for any write under an expansion to go
// through.
func (j judge) mayReach(fixed string) bool {
	fixed = strings.TrimPrefix(filepath.ToSlash(fixed), "./")
	if fixed == "" {
		return false
	}
	for _, g := range j.protectedGlobs() {
		candidate := fixed
		if g.fold {
			candidate = strings.ToLower(candidate)
		}
		glob, anywhere := strings.CutPrefix(g.glob, "**/")
		if !strings.Contains(glob, "/") {
			base := candidate[strings.LastIndexByte(candidate, '/')+1:]
			literal := glob
			if meta := strings.IndexAny(glob, "*?["); meta >= 0 {
				literal = glob[:meta]
			}
			if base != "" && literal != "" && strings.HasPrefix(literal, base) {
				return true
			}
			continue
		}
		literal := literalPrefix(glob)
		for start := 0; literal != ""; {
			c := candidate[start:]
			if c != "" && (strings.HasPrefix(literal, c) || strings.HasPrefix(c, literal+"/")) {
				return true
			}
			next := strings.IndexByte(c, '/')
			if !anywhere || next < 0 {
				break
			}
			start += next + 1
		}
	}
	return false
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
