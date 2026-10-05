// Package run holds the three widths of a check run: one file, one
// bundle, every registered area. Design 7 calls that the natural layering
// of the rules -- some decide at a page, some at a bundle, two only at the
// registration -- and this package is the layer that decides which of them
// a question needs, supplies them, and brings their answers into one
// comparable order.
//
// It sits beside `internal/brain/check` rather than inside it, and the reason is
// mechanical, not a matter of taste: `internal/brain/check/okf` and
// `internal/brain/check/house` return `check.Finding` and therefore import
// `internal/brain/check`, so a file in that package importing them back would close
// an import cycle. Measured, not argued -- `go build` names it:
// "imports .../check/okf ... imports .../check from errors.go:
// import cycle not allowed".
//
// This package is the first real filler of `wiki.BundleContext`. Until
// now that struct had only ever been handed to a rule by a test fixture,
// and two of its fields were quietly dangerous while nobody filled them:
// a zero `Now` lies before every deadline, so `stale` and `untouched`
// would stay silent on a stock that is anything but clean.
package run

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/check/house"
	"github.com/xidus90/loomux/internal/brain/check/okf"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
)

// needsNeighbours names the rules a single file cannot answer even though
// they live in a width `CheckFile` runs. Both were read rather than
// guessed, and both do fire on a one-page slice:
//
//   - `okf/no-index` reports every directory whose pages carry no
//     catalog (`internal/brain/check/okf/soft.go`, `noIndex`), and one page is such
//     a directory.
//   - `okf/no-log` reports any non-empty page set without a `log.md` at
//     the bundle root (`noLog`), and a set of one is non-empty.
//
// Three further rules need neighbours and are absent for a stronger
// reason than a filter: `house/orphan`, `house/dead-link` and
// `house/unlisted-area` live in `house.Bundle` and `house.Federation`,
// which this width never calls at all.
//
// `okf/catalog-malformed` is deliberately not in this map. It needs the
// page set, but on one page it has nothing to say by construction: the
// only catalog it could judge is the page itself, and a page that is a
// catalog is scaffold and is skipped before the question is asked. A
// line here would be a statement no mutation could kill; a test holds it
// instead.
var needsNeighbours = map[string]bool{
	"no-index": true,
	"no-log":   true,
}

// CheckFile is the fast path, the one a Stop hook runs after every write.
// It asks everything that is decidable without the neighbours: the OKF
// musts and recommendations that stand on the page itself, and the five
// house rules of `house.Page`.
//
// It reads one file and at most two small declarations, never the bundle
// -- that is the whole point of the width, and design 7 puts the Go
// core's budget for it under five milliseconds.
//
// Unlike `wiki.LintSingleFile` it does not drop a scaffold file. The okf
// rules judge the reserved names on purpose (`internal/brain/check/okf/errors.go`
// says so of 11.1), and the house rules draw that line themselves in
// `judgedAsConcept`; a second line here would take the decision away from
// the rule that owns it.
//
// The `Scope` of every finding stays empty. A run over one file knows a
// path but no area (the comment on `check.Finding.Scope`), and the root-finding
// below deliberately does not turn its registry hit into a name: the two
// other paths to a root produce no scope at all, and a field that were
// filled on one path in three would sort the same run two ways.
func CheckFile(path string, lookup config.ArtifactLookup) []check.Finding {
	root, manifest := fileRoot(path, lookup)
	page, err := wiki.ReadPage(path, root)
	if err != nil {
		// No finding, because there is nothing to report it on: a file
		// that cannot be read has no relative path to name and no
		// frontmatter to judge. Python answers the same way -- an
		// argument that is not a file leaves `lint_single_file` as an
		// empty tuple (`src/brain/wiki/lint.py:597-598`).
		return nil
	}
	ctx := wiki.BundleContext{
		Root:          root,
		UntouchedDays: untouchedDays(manifest),
		Now:           time.Now(),
		DeclaredTypes: declaredTypes(manifest),
	}
	pages := []wiki.WikiPage{*page}
	var out []check.Finding
	for _, f := range collect(pages, ctx) {
		if needsNeighbours[f.Rule] {
			continue
		}
		out = append(out, f)
	}
	return sorted(out)
}

// CheckBundle adds the eight rules that need more than one page: the six
// that stop at the edge of one bundle, and the two federation rules.
//
// It takes an `config.Area` and not a path because three of the values
// the rules turn on are properties of the *area* and not of the
// directory: the threshold and the type vocabulary come from its
// manifest, and `is_project` from the first segment of its scope
// (`src/brain/cli.py:1557`).
//
// The second argument is the whole registration, and it is what the two
// federation rules judge the one bundle against: the shared set a page
// may cite, and the areas a signpost must name. Neither rule reads a
// foreign bundle's pages, so one read bundle is enough -- and Python
// runs both in every single-area lint for that reason
// (`src/brain/cli.py:1524-1556`). `areas` is the registration as read,
// not the walkable subset: `Targets` refuses a run over an area whose
// wiki is missing, and a bundle run must not fail over an area it was
// not asked about.
func CheckBundle(area config.Area, areas []config.Area, lookup config.ArtifactLookup) []check.Finding {
	pages, findings := readArea(area, lookup, time.Now())
	findings = append(findings, house.FederationFor(
		[]config.Area{area},
		map[string][]wiki.WikiPage{area.Scope: pages}, areas, lookup)...)
	return sorted(findings)
}

// One goroutine per area: eight areas, cleanly separated stocks, no shared
// write target. Inside a bundle nothing is parallelised -- the rules are
// I/O-bound, and concurrency there costs reliability in a chain whose worth
// is its reliability. The measurement of design 9.5 decides whether that holds.
// The second argument is the whole registration, and it is there for
// the same reason as in CheckBundle: `Targets` refuses to walk an area
// whose wiki is missing, and handing that shortened list to the
// federation rules as the registration too would take such an area out
// of the shared set. A citation into it then turned from allowed into
// `house/wrong-direction`, an error the bundle width does not report --
// two widths contradicting each other about one page. Python builds
// `shared_scopes` from `read_registry`, the whole file
// (`src/brain/cli.py:1532-1533`), and never had the question.
func CheckAll(subjects, areas []config.Area, lookup config.ArtifactLookup) []check.Finding {
	return sorted(runAreas(subjects, areas, lookup, true))
}

// runAreas is both entry points' engine and the shape the measurement of
// design 9.5 needs: the same work, scheduled two ways, so that the
// decision of 7 can be confirmed or overturned by a number rather than by
// an argument. A run that answered differently depending on how it was
// scheduled would make that measurement meaningless, which is why a test
// compares the two renderings byte for byte.
//
// One clock for the whole run, taken here. Eight goroutines each calling
// `time.Now()` would put eight instants into one verdict, and a page
// whose `stale_after` fell between two of them would be reported or not
// depending on which goroutine got there first.
//
// The results are collected into a slice indexed by the area's position
// and concatenated in that order, not appended as they finish. The sort
// below is stable, so findings equal on all four keys keep the order they
// arrived in -- and "the order they arrived in" must not mean "the order
// the scheduler happened to produce".
func runAreas(subjects, areas []config.Area, lookup config.ArtifactLookup, parallel bool) []check.Finding {
	now := time.Now()
	perArea := make([][]check.Finding, len(subjects))
	bundles := map[string][]wiki.WikiPage{}

	if parallel {
		var mu sync.Mutex
		var wg sync.WaitGroup
		for i, area := range subjects {
			wg.Add(1)
			go func(i int, area config.Area) {
				defer wg.Done()
				pages, findings := readArea(area, lookup, now)
				perArea[i] = findings
				// The map is the one shared write target, and it is
				// guarded rather than sharded: every goroutine writes
				// exactly once, under its own key, so the lock is held
				// for a pointer assignment and contention is not the
				// question. `perArea` needs no lock -- each goroutine
				// owns one element of a slice whose length never
				// changes.
				mu.Lock()
				bundles[area.Scope] = pages
				mu.Unlock()
			}(i, area)
		}
		wg.Wait()
	} else {
		for i, area := range subjects {
			pages, findings := readArea(area, lookup, now)
			perArea[i] = findings
			bundles[area.Scope] = pages
		}
	}

	var out []check.Finding
	for _, findings := range perArea {
		out = append(out, findings...)
	}
	// After every bundle, never inside one area's goroutine -- but the
	// reason is the map, not the rules. Each rule reads only the bundle
	// of the area it judges (`internal/brain/check/house/federation.go:227`,
	// `:296`); what it needs of the *others* comes from the
	// registration. Run inside a goroutine it would judge every subject
	// against a map holding one bundle, and `unlisted-area` errs towards
	// speaking on a bundle it was not given, so it would report the
	// federation as unlisted once per area. Here the map is complete
	// before any subject is asked.
	return append(out, house.Federation(bundles, areas, lookup)...)
}

// readArea reads one bundle and runs every rule that stops at its edge.
//
// The bundle root is the area's registered wiki path, which is where
// `_lint_targets` takes it from as well (`src/brain/cli.py:1489-1497`).
// An area that declares none has no bundle, and Python does not lint it
// either.
func readArea(
	area config.Area, lookup config.ArtifactLookup, now time.Time,
) ([]wiki.WikiPage, []check.Finding) {
	if area.WikiPath == "" {
		return nil, nil
	}
	manifest, findings := areaManifest(area, lookup)
	pages := readBundle(area.WikiPath)
	ctx := wiki.BundleContext{
		Root:          area.WikiPath,
		UntouchedDays: untouchedDays(manifest),
		Now:           now,
		DeclaredTypes: declaredTypes(manifest),
		IsProject:     strings.SplitN(area.Scope, "/", 2)[0] == "project",
	}
	found := append(findings, collect(pages, ctx)...)
	found = append(found, house.Bundle(pages, ctx)...)
	for i := range found {
		// The rules of these two widths are handed one bundle without
		// its name, so filling `Scope` is their caller's job
		// (`internal/brain/check/house/federation.go:56-64` says the same of the
		// one function that already knows it).
		found[i].Scope = area.Scope
	}
	return pages, found
}

// collect runs the rules that stop at a page: the nine OKF musts, the
// four warnings and three notes beside them, and the five house rules of
// `house.Page`. It is shared by `CheckFile` and `readArea` so that the
// narrow width can never ask a different set than the wide one minus what
// it filters.
func collect(
	pages []wiki.WikiPage, ctx wiki.BundleContext,
) []check.Finding {
	var out []check.Finding
	out = append(out, okf.Errors(pages, ctx)...)
	out = append(out, okf.Soft(pages, ctx)...)
	return append(out, house.Page(pages, ctx)...)
}

// sorted is the one place `check.Sort` is called, and the rules
// deliberately do not call it: each of the four rule sets says so in its
// own comment -- "sorting twice would only make the second call decide
// the order". Comparability is the purpose (the comment on `check.Sort`),
// and a purpose served in five places is served by whichever place runs
// last.
func sorted(findings []check.Finding) []check.Finding {
	check.Sort(findings)
	return findings
}

// readBundle is the single read pass over a bundle, and everything below
// runs off its result -- design 7: "ein einziger Lesedurchlauf befuellt
// den Seitenbestand, ueber den alle Regeln dann sequenziell laufen".
//
// The scaffold files stay in the set, unlike in Python's `lint_bundle`,
// which drops them before any rule sees one
// (`src/brain/wiki/lint.py:582-586`). The Go rules need them: `index.md`
// is the catalog `okf/catalog-malformed`, `okf/no-index` and
// `house/unlisted-area` judge, and `log.md` is what `okf/log-date-form`
// reads. Every rule that must not judge a scaffold page as knowledge
// draws that line itself, in `judgedAsConcept` and `judgedInBundle`.
//
// The order is `wiki.MarkdownBelow`'s, Python's `sorted(rglob(...))` on
// Windows. WalkDir's own byte order is not it: it puts `B.md` before
// `a.md`, and the stable sort after the rules keeps whatever order equal
// findings arrived in.
//
// A page that cannot be read is skipped rather than reported. That is the
// behaviour `internal/brain/wiki/lint.go` already has, and this width has no rule
// for an unreadable file; whoever writes `okf/file-unreadable` owes the
// change here.
func readBundle(root string) []wiki.WikiPage {
	var pages []wiki.WikiPage
	for _, path := range wiki.MarkdownBelow(root) {
		if page, err := wiki.ReadPage(path, root); err == nil {
			pages = append(pages, *page)
		}
	}
	return pages
}

// areaManifest reads an area's declaration and turns the two ways of it
// being unusable into findings.
//
// This is the channel the rules do not have. `house.Federation` stops
// `unlisted-area` when the signpost's manifest cannot be used, and it
// stops *silently* -- `hubFolder`
// (`internal/brain/check/house/federation.go:320-323`) says so itself: "Silence is
// not a verdict here ... the error itself is owed to the reader by
// whoever reads that manifest for its other values -- this rule has no
// channel to carry it." This function is that reader.
//
// Two rule names and not one, because a file that parses and states an
// unusable value is not unreadable and the two send the reader to
// different repairs. Both are new: design 5.2 lists neither, and that
// gap is named in the report rather than papered over.
//
// A manifest that exists and cannot be opened is `manifest-unreadable`,
// not "nothing declared": `config.ReadAreaDeclaration` answers the read
// error of a `.loomux/config.toml` that is a regular file, so a locked
// declaration never reads as absent. The case is pinned in `run_test.go`.
//
// The reader is `config.ReadAreaDeclaration`, stricter than
// `config.ReadManifest`: it refuses `[area]` without `scope` and any
// declaration whose known keys carry the wrong type, and both arrive here
// as `manifest-unreadable`.
//
// The layout values are asked even where this run does not use them: the
// `hub` value only matters to a signpost, and the `wiki` value only to
// `CheckFile` and the write barrier. A defect in a declaration is a
// defect whether or not the current width trips over it, and the run
// that reads the file is the only one in a position to say so.
func areaManifest(
	area config.Area, lookup config.ArtifactLookup,
) (*config.Manifest, []check.Finding) {
	manifest, err := config.ReadAreaDeclaration(
		config.ManifestDir(area, lookup.Primary))
	if config.IsUndeclared(err) {
		// Not a defect: an area may declare nothing, and the caller then
		// falls back to the defaults (`config.DefaultUntouchedDays`).
		return nil, nil
	}
	if err != nil {
		return nil, []check.Finding{declarationFinding(
			"manifest-unreadable", err.Error())}
	}
	var out []check.Finding
	if _, err := manifest.HubLayout(); err != nil {
		out = append(out, declarationFinding(
			"manifest-layout-invalid", err.Error()))
	}
	if _, err := manifest.WikiLayout(); err != nil {
		out = append(out, declarationFinding(
			"manifest-layout-invalid", err.Error()))
	}
	return manifest, out
}

// declarationFinding is the fourth door beside the three the rule
// packages keep, and it exists for a subject none of them has: this
// finding is not about a page.
//
// `Relative` therefore names no file. The parentheses are what say so --
// no page is called that -- and the message carries the manifest's own
// path, which `config.ReadAreaDeclaration` puts into every error it answers
// with. The alternative was to put the finding on the bundle's
// `index.md`, the way `house/unlisted-area` does, and it is wrong here
// for the reason that rule gives for its own choice: the repair is a
// line in the file the finding names, and the repair for this one is not
// in `index.md`.
//
// Error, both of them. A declaration this reader cannot use stops rules
// from running at all, and a check that reported a silent rule failure
// as a warning would let the run go green over the one defect that hides
// every other.
func declarationFinding(rule, message string) check.Finding {
	return check.Finding{
		Relative: "(declaration)",
		Axis:     check.AxisHouse,
		Rule:     rule,
		Severity: check.Error,
		Message:  message,
	}
}

// untouchedDays is the threshold the area declares, or the default an
// area with no declaration gets. Python decides it in the caller too, and
// in one line: `manifest.untouched_days if manifest else
// DEFAULT_UNTOUCHED_DAYS` (`src/brain/cli.py:1538`). A rule that fell
// back on its own would have to know about manifests.
func untouchedDays(manifest *config.Manifest) int {
	if manifest == nil {
		return config.DefaultUntouchedDays
	}
	return manifest.UntouchedDays
}

// declaredTypes is the vocabulary beyond the built-in one, as a set
// because `wiki.BundleContext` holds it that way. `frozenset(...) if
// manifest else frozenset()` is the same line on the Python side
// (`src/brain/cli.py:1539`).
//
// The values are not folded here. `house/unknown-type` builds a
// `config.Manifest` out of this set and asks `KnowsType`
// (`internal/brain/check/house/page.go:203-207`), which lowercases and trims both
// sides, so folding here would be the same answer computed twice.
func declaredTypes(manifest *config.Manifest) map[string]bool {
	types := map[string]bool{}
	if manifest == nil {
		return types
	}
	for _, t := range manifest.DeclaredTypes {
		types[t] = true
	}
	return types
}

// fileRoot is how the fast path finds the bundle a lone file belongs to,
// and it is a chain of three because no single answer covers the stock.
//
//  1. **The declaration above it.** Walking the parents for a manifest is
//     what the reference's write barrier does, and `[layout] wiki` is the one statement of the root that survives a
//     linked worktree -- the manifest of this repository says so in as many
//     words, and that is why the wiki moved into the repo at all. The
//     registration cannot answer here: it holds one absolute path per area,
//     and a worktree has another.
//  2. **The registration, longest match wins.** A vault may register a
//     bundle inside another bundle: the registry on this machine notes that
//     `hub` encloses the wikis of `project/space`, `project/iam-wiki`,
//     `project/obsidian-ai` and `project/ecoflow`. Taking the first or the
//     outermost match would judge a page of the inner area against the
//     enclosing bundle and give every finding of it the wrong path.
//  3. **The file's own directory.** Python's answer when no root is handed
//     in (`src/brain/wiki/lint.py:599`), and the honest one: a page outside
//     every declared bundle still has frontmatter to judge.
//
// The manifest travels back with the root because the caller needs it for
// the type vocabulary and the threshold, and finding it twice would be
// two reads with two chances of a different answer -- the reason
// the reference's barrier gives for reading it once.
//
// A manifest whose `[layout] wiki` this reader refuses falls through to
// the next link rather than stopping the run. `CheckFile` has no output
// but findings on the page it was given, and refusing to check a page
// over a key that names a different directory would be the wrong
// direction of failure; `CheckBundle` and `CheckAll` do report that
// defect, on the area it belongs to.
func fileRoot(path string, lookup config.ArtifactLookup) (string, *config.Manifest) {
	dir := filepath.Dir(path)
	for at := dir; ; {
		if manifest, err := config.ReadAreaDeclaration(at); err == nil {
			if place, err := manifest.WikiLayout(); err == nil &&
				place != "" {
				// No FromSlash: `filepath.Join` cleans what it
				// joins, and cleaning converts the separators.
				// Measured on this machine, joining one repository
				// root with "docs/wiki" directly and through
				// FromSlash gave one and the same Windows path. The
				// mutation round struck the call and no test noticed
				// it was gone.
				root := filepath.Join(at, place)
				if under(path, root) {
					return root, manifest
				}
			}
			// Found, parsable and unusable: no second declaration is
			// looked for above it. A manifest is the statement of the
			// area one is standing in, and climbing past it would
			// answer with the area that encloses it -- the containment
			// the reference's barrier insists on.
			//
			// Parsable is the word that matters. A manifest that does
			// not decode never enters this arm at all -- `err == nil`
			// above is false for it -- so the walk climbs past that
			// one, reaches no declaration, and lets the registration
			// answer. That is the better direction: a broken file
			// costs the vocabulary, and losing the root as well would
			// give every finding on the page a path measured from
			// somewhere else.
			break
		}
		parent := filepath.Dir(at)
		if parent == at {
			break
		}
		at = parent
	}
	if root, area, ok := registeredRoot(path, lookup); ok {
		manifest, err := config.ReadAreaDeclaration(
			config.ManifestDir(area, lookup.Primary))
		if err != nil {
			// The bundle root is still the right one; only the
			// vocabulary is missing, and `unknown-type` then judges by
			// the built-in set alone. The defect itself is reported by
			// the two wider widths, which read the same file.
			return root, nil
		}
		return root, manifest
	}
	return dir, nil
}

// registeredRoot is the longest registered wiki path that contains this
// file, with the area it belongs to.
//
// A registration that cannot be read is no defect of the page: the fast
// path then falls through to the file's own directory, which is exactly
// the state Python's `lint_single_file` is called in from `brain lint`
// (`src/brain/cli.py:733` passes no root at all).
func registeredRoot(path string, lookup config.ArtifactLookup) (string, config.Area, bool) {
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil {
		return "", config.Area{}, false
	}
	var best string
	var owner config.Area
	for _, area := range areas {
		if area.WikiPath == "" || !under(path, area.WikiPath) {
			continue
		}
		if len(area.WikiPath) > len(best) {
			best, owner = area.WikiPath, area
		}
	}
	return best, owner, best != ""
}

// under says whether a file lies inside a directory. It compares by path
// component and not by string, or `.../python-old/x.md` would count as
// lying in `.../python` -- the same reason
// `internal/brain/check/house/federation.go` gives for its own prefix test.
//
// Nothing is cleaned here, and an earlier version of this line did clean
// both sides. `filepath.Rel` cleans what it is given, so the two calls
// decided nothing and the mutation round found them: struck, every test
// still passed. Measured on this machine rather than read off the
// documentation -- with a trailing separator, with a `.` component, and
// with the two flavours of separator mixed, `Rel` answered `c\d.md` and
// `c.md` in every spelling. That matters because the two sides really do
// arrive spelt differently: the registration writes forward slashes even
// on Windows (`internal/config/registry.go:18-22`), and the path a hook hands
// in is the operating system's.
func under(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+
		string(filepath.Separator))
}
