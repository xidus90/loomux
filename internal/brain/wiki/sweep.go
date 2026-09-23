package wiki

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/pytext"
)

// SweepContext is `lint.BundleContext`: everything a rule of the lint over
// areas may know beyond the pages. The zero value of every field past Now is
// the reference's default -- no signpost duty, not shared, nothing declared,
// no project.
type SweepContext struct {
	Root          string
	UntouchedDays int
	Now           time.Time
	// ExpectedTargets is what a signpost must link, per area; any one target
	// of an area is enough. Empty in every bundle that is no signpost.
	ExpectedTargets []ExpectedTarget
	IsShared        bool
	SharedScopes    map[string]bool
	DeclaredTypes   map[string]bool
	IsProject       bool
}

// ExpectedTarget is one area the signpost must name, and the absolute paths
// that would name it: its wiki, and a hub page when the signpost declares one.
type ExpectedTarget struct {
	Scope   string
	Targets []string
}

// SweepBundle is `lint_bundle`: every page below the root that is no scaffold
// file, judged by the twelve rules of `src/brain/wiki/lint.py` in the order of
// its `RULES`, which is the order of the output. These rules are the
// reference's and are kept apart from LintBundle, whose five rules are the Go
// form behind `lint <file>`, `wiki-gate` and the edit lane.
//
// A page that cannot be read ends the run, as `read_page` raising does.
func SweepBundle(ctx SweepContext) ([]check.Finding, error) {
	patterns := newSweepPatterns()
	var pages []sweepPage
	for _, file := range MarkdownBelow(ctx.Root) {
		if IsScaffoldFile(file) {
			continue
		}
		page, err := readSweepPage(file, ctx.Root, patterns)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	var out []check.Finding
	for _, rule := range []func([]sweepPage, SweepContext, sweepPatterns) []check.Finding{
		missingType, noSources, orphan, unlistedArea, deadLink, outsideArea,
		wrongDirection, conflictCount, untouched, stale, implementedWithoutCommit, longPlanned,
	} {
		out = append(out, rule(pages, ctx, patterns)...)
	}
	return out, nil
}

func sweepFinding(relative, rule string, severity check.Severity, message string) check.Finding {
	return check.Finding{Relative: relative, Rule: rule, Severity: severity, Message: message}
}

// missingType names a page whose frontmatter does not parse as
// `broken-frontmatter`, and any other page whose type ranks unknown in this
// area as `missing-type`: one defect, one name.
func missingType(pages []sweepPage, ctx SweepContext, _ sweepPatterns) []check.Finding {
	var out []check.Finding
	for _, page := range pages {
		switch {
		case page.brokenFrontmatter != nil:
			out = append(out, sweepFinding(page.relative, "broken-frontmatter", check.Error,
				"frontmatter is unreadable: "+*page.brokenFrontmatter))
		case RankOf(page.pageType, ctx.DeclaredTypes) == RankUnknown:
			out = append(out, sweepFinding(page.relative, "missing-type", check.Error,
				unknownTypeMessage(page.pageType)))
		}
	}
	return out
}

// unknownTypeMessage says what to do: an old name gets its successor, any
// other type the two ways of becoming valid.
func unknownTypeMessage(pageType *string) string {
	if pageType == nil {
		return "no type; every OKF page needs one"
	}
	if target, ok := AliasTarget(*pageType); ok {
		return fmt.Sprintf("%s is an old name for %s; rename it", pytext.Repr(*pageType), pytext.Repr(target))
	}
	return pytext.Repr(*pageType) + " is neither core nor catalogue; add it to `[wiki] types` " +
		"in the area's manifest, or use a catalogue name"
}

// noSources wants at least one source on every page, synthesis included, and
// each source complete enough to be followed back.
func noSources(pages []sweepPage, _ SweepContext, _ sweepPatterns) []check.Finding {
	var out []check.Finding
	for _, page := range pages {
		if len(page.sources) == 0 {
			out = append(out, sweepFinding(page.relative, "no-sources", check.Error, "sources[] is empty"))
			continue
		}
		var incomplete []string
		for _, s := range page.sources {
			if s.id == "" || s.resource == "" || s.docID == "" || s.contentHash == "" || s.revision == nil {
				incomplete = append(incomplete, cmpOr(s.id, "<unnamed>"))
			}
		}
		if len(incomplete) > 0 {
			out = append(out, sweepFinding(page.relative, "no-sources", check.Error,
				"source entries lack id, resource, doc_id, content_hash or revision: "+strings.Join(incomplete, ", ")))
		}
	}
	return out
}

func cmpOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// resolveLink is `_resolve`: a link target as a bundle-relative POSIX path, or
// false when it points outside -- a scheme, no path, or a climb out of the
// bundle. A leading slash is bundle-relative (OKF §6.1) and normalised before
// the slash comes off, so `/../x` cannot climb out either. Percent escapes are
// undone after query and fragment are cut.
func resolveLink(relative, target string) (string, bool) {
	scheme, _, p := pytext.SplitURL(target)
	if scheme != "" || p == "" {
		return "", false
	}
	decoded := pytext.Unquote(p)
	if strings.HasPrefix(decoded, "/") {
		return strings.TrimLeft(path.Clean(decoded), "/"), true
	}
	joined := path.Clean(path.Join(path.Dir(relative), decoded))
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return "", false
	}
	return joined, true
}

// indexLinks are the links of the bundle root's catalog: no OKF page, but the
// place a page is first linked. Only the root's `index.md` counts.
func indexLinks(root string, p sweepPatterns) []string {
	// Absent, a directory or unreadable: no catalog, no links. The reference
	// asks `is_file()` first; the read answers the same question.
	page, err := readSweepPage(filepath.Join(root, "index.md"), root, p)
	if err != nil {
		return nil
	}
	return page.links
}

// orphan names a page nothing points at -- neither the catalog nor another
// page.
func orphan(pages []sweepPage, ctx SweepContext, p sweepPatterns) []check.Finding {
	linked := map[string]bool{}
	add := func(source string, links []string) {
		for _, target := range links {
			if resolved, ok := resolveLink(source, target); ok {
				linked[resolved] = true
			}
		}
	}
	add("index.md", indexLinks(ctx.Root, p))
	for _, page := range pages {
		add(page.relative, page.links)
	}
	var out []check.Finding
	for _, page := range pages {
		if !linked[page.relative] {
			out = append(out, sweepFinding(page.relative, "orphan", check.Error, "no page and no catalog entry links here"))
		}
	}
	return out
}

// targetsOf are a page's links followed by its sources' resources, the set
// `dead_link` and `outside_area` walk.
func targetsOf(page sweepPage) []string {
	targets := slices.Clone(page.links)
	for _, s := range page.sources {
		targets = append(targets, s.resource)
	}
	return targets
}

// deadLink names a link or a source path the bundle does not have.
func deadLink(pages []sweepPage, ctx SweepContext, _ sweepPatterns) []check.Finding {
	var out []check.Finding
	for _, page := range pages {
		for _, target := range targetsOf(page) {
			resolved, ok := resolveLink(page.relative, target)
			if !ok {
				continue
			}
			if _, err := os.Stat(filepath.Join(ctx.Root, filepath.FromSlash(resolved))); err == nil {
				continue
			}
			// The raw target is what the reader can search for in the page.
			out = append(out, sweepFinding(page.relative, "dead-link", check.Error,
				fmt.Sprintf("target does not exist: %s (resolved: %s)", target, resolved)))
		}
	}
	return out
}

// outsideArea names a target that climbs out of the area instead of passing
// over it; a scheme is a deliberate reference to the outside and passes.
func outsideArea(pages []sweepPage, _ SweepContext, _ sweepPatterns) []check.Finding {
	var out []check.Finding
	for _, page := range pages {
		for _, target := range targetsOf(page) {
			scheme, _, p := pytext.SplitURL(target)
			if scheme != "" || p == "" {
				continue
			}
			if _, ok := resolveLink(page.relative, target); !ok {
				out = append(out, sweepFinding(page.relative, "outside-area", check.Warning,
					"target leaves the area and cannot be checked: "+target))
			}
		}
	}
	return out
}

// citedScope is `_cited_scope`: the scope a `brain://` reference names -- a
// known scope it starts with, or the whole reference when none owns it -- and
// false when it names none. The reference takes the longest known scope; the
// one caller asks only whether the answer is known, and every match is, so
// the first one found answers the same.
func citedScope(resource string, known map[string]bool) (string, bool) {
	scheme, netloc, p := pytext.SplitURL(resource)
	if scheme != "brain" {
		return "", false
	}
	joined := strings.Trim(netloc+p, "/")
	for scope := range known {
		if joined == scope || strings.HasPrefix(joined, scope+"/") {
			return scope, true
		}
	}
	return joined, joined != ""
}

// wrongDirection: projects read shared areas, shared areas read no project.
func wrongDirection(pages []sweepPage, ctx SweepContext, _ sweepPatterns) []check.Finding {
	if !ctx.IsShared {
		return nil
	}
	var out []check.Finding
	for _, page := range pages {
		for _, s := range page.sources {
			cited, ok := citedScope(s.resource, ctx.SharedScopes)
			if !ok || ctx.SharedScopes[cited] {
				continue
			}
			out = append(out, sweepFinding(page.relative, "wrong-direction", check.Error,
				fmt.Sprintf("a shared area must not cite %s; promote the page instead (architecture 5.7.4)", pytext.Repr(cited))))
		}
	}
	return out
}

// conflictCount wants `open_conflicts` and the boxes present to agree. A page
// without boxes need not carry the field.
func conflictCount(pages []sweepPage, _ SweepContext, _ sweepPatterns) []check.Finding {
	var out []check.Finding
	for _, page := range pages {
		switch {
		case page.declaredConflicts == nil && page.foundConflicts > 0:
			out = append(out, sweepFinding(page.relative, "conflict-count", check.Error,
				fmt.Sprintf("%d conflict box(es) present, open_conflicts is missing", page.foundConflicts)))
		case page.declaredConflicts != nil && *page.declaredConflicts != page.foundConflicts:
			out = append(out, sweepFinding(page.relative, "conflict-count", check.Error,
				fmt.Sprintf("open_conflicts says %s, %d box(es) found", page.declaredSaid, page.foundConflicts)))
		}
	}
	return out
}

// catalogTargets are the catalog's links resolved against the bundle root to
// absolute paths; `resolveLink` would drop them, since the signpost points
// nowhere else but out of its bundle.
func catalogTargets(root string, p sweepPatterns) []string {
	var targets []string
	for _, link := range indexLinks(root, p) {
		scheme, _, path := pytext.SplitURL(link)
		if scheme != "" || path == "" {
			continue
		}
		targets = append(targets, AbsoluteClean(filepath.Join(root, filepath.FromSlash(pytext.Unquote(path)))))
	}
	return targets
}

// AbsoluteClean is `Path.resolve()` as far as a lexical answer goes: absolute
// and cleaned, in the operating system's spelling.
func AbsoluteClean(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return abs
}

// LinkNames is `_names`: a hub target is one page and needs the link itself,
// a wiki target is a directory and takes any link below it. Paths compare as
// Windows compares them, part by part and without regard to case.
func LinkNames(link, target string) bool {
	l, t := foldedParts(link), foldedParts(target)
	if strings.EqualFold(filepath.Ext(target), ".md") {
		return slices.Equal(l, t)
	}
	return len(l) >= len(t) && slices.Equal(l[:len(t)], t)
}

// unlistedArea: the signpost names every area of the federation, by its wiki
// or by its hub page.
func unlistedArea(_ []sweepPage, ctx SweepContext, p sweepPatterns) []check.Finding {
	if len(ctx.ExpectedTargets) == 0 {
		return nil
	}
	linked := catalogTargets(ctx.Root, p)
	var out []check.Finding
	for _, expected := range ctx.ExpectedTargets {
		named := false
		for _, link := range linked {
			for _, target := range expected.Targets {
				named = named || LinkNames(link, target)
			}
		}
		if !named {
			out = append(out, sweepFinding("index.md", "unlisted-area", check.Error,
				fmt.Sprintf("the signpost does not link to area %s; expected one of: %s",
					pytext.Repr(expected.Scope), strings.Join(expected.Targets, ", "))))
		}
	}
	return out
}

// sweepCutoff is the instant a page has to be older than to count as
// untouched: `timedelta(days=n)` is exactly n * 24 hours, where a calendar
// step would drift by an hour across a daylight-saving change.
func sweepCutoff(ctx SweepContext) time.Time {
	return ctx.Now.Add(-time.Duration(ctx.UntouchedDays) * 24 * time.Hour)
}

// untouched: nobody looked at the page for longer than the area allows. A
// warning, because age alone says nothing about correctness.
func untouched(pages []sweepPage, ctx SweepContext, _ sweepPatterns) []check.Finding {
	cutoff := sweepCutoff(ctx)
	var out []check.Finding
	for _, page := range pages {
		if page.mtime.Before(cutoff) {
			out = append(out, sweepFinding(page.relative, "untouched", check.Warning,
				fmt.Sprintf("unchanged for more than %d days", ctx.UntouchedDays)))
		}
	}
	return out
}

// stale: a page passed the expiry day it declared, measured in local days.
func stale(pages []sweepPage, ctx SweepContext, _ sweepPatterns) []check.Finding {
	today := ctx.Now.Local().Format("2006-01-02")
	var out []check.Finding
	for _, page := range pages {
		if page.staleAfter != "" && page.staleAfter < today {
			out = append(out, sweepFinding(page.relative, "stale", check.Error,
				"stale_after "+page.staleAfter+" has passed"))
		}
	}
	return out
}

// implementedWithoutCommit: a project page claiming to be built names where.
func implementedWithoutCommit(pages []sweepPage, ctx SweepContext, _ sweepPatterns) []check.Finding {
	if !ctx.IsProject {
		return nil
	}
	var out []check.Finding
	for _, page := range pages {
		if page.realization != nil && *page.realization == "implemented" &&
			(page.implementedIn == nil || *page.implementedIn == "") {
			out = append(out, sweepFinding(page.relative, "implemented-without-commit", check.Error,
				"realization: implemented without implemented_in"))
		}
	}
	return out
}

// longPlanned: a project plan that never moved within the area's threshold.
func longPlanned(pages []sweepPage, ctx SweepContext, _ sweepPatterns) []check.Finding {
	if !ctx.IsProject {
		return nil
	}
	limit := sweepCutoff(ctx)
	var out []check.Finding
	for _, page := range pages {
		if page.realization != nil && *page.realization == "planned" && page.mtime.Before(limit) {
			out = append(out, sweepFinding(page.relative, "long-planned", check.Warning,
				"planned and untouched since "+page.mtime.Local().Format("2006-01-02")))
		}
	}
	return out
}
