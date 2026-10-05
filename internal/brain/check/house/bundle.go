package house

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
)

// Bundle runs the six house rules that a single page cannot answer. Each
// of them needs the neighbours: the link graph, the catalog, or the
// threshold the area declares for the whole bundle. Design §7 calls that
// the natural layering of the rules and gives `brain check bundle` its
// scope from it.
//
// Two grades, not one like `Page`: the design's house table (§5.2) puts
// `orphan`, `dead-link` and `implemented-without-commit` under Fehler and
// `outside-area`, `untouched` and `long-planned` under Warnung.
//
// The order is Python's (`src/brain/wiki/lint.py:539-552`), which costs
// nothing and keeps one more thing from drifting while the two sides are
// compared. It is not the output order: `check.Sort` is the job of
// whoever merges the axes, and sorting here would only make the second
// call decide.
func Bundle(pages []wiki.WikiPage, ctx Context) []check.Finding {
	var out []check.Finding
	out = append(out, orphan(pages)...)
	out = append(out, deadLink(pages, ctx)...)
	out = append(out, outsideArea(pages)...)
	out = append(out, untouched(pages, ctx)...)
	out = append(out, implementedWithoutCommit(pages, ctx)...)
	out = append(out, longPlanned(pages, ctx)...)
	return out
}

// warning is the second door `finding` asked for. Its neighbour above
// hardcodes `check.Error` and says why the severity is not a parameter:
// the day this axis gains a warning, that warning needs its own door, or
// a rule could quietly demote itself out of the exit code. This is that
// day -- three of the six are warnings -- so the two degrees keep two
// constructors rather than one that takes the degree from its caller.
func warning(p wiki.WikiPage, rule, message string) check.Finding {
	return check.Finding{
		Relative: p.Relative,
		Axis:     check.AxisHouse,
		Rule:     rule,
		Severity: check.Warning,
		Message:  message,
	}
}

// judgedInBundle says whether a page is a *subject* of these rules. Only
// the scaffold line survives from `judgedAsConcept`: Scheibe 3 §4 exempts
// the five scaffold files by name, and Python drops them before any rule
// sees one (`src/brain/wiki/lint.py:582-586`).
//
// An unreadable frontmatter is *not* exempt here, unlike on the page
// rules. Those five each read a field out of the block that did not
// decode and would blame a page for stating nothing. None of these six
// can. Two of them read no frontmatter at all -- `orphan` goes by the
// body's links, `untouched` by the file's age. `dead-link` and
// `outside-area` do read `Sources`, but a failed decode leaves that
// list empty, so they say *less* about such a page and never something
// false. And the two realization rules need no guard because
// `internal/brain/wiki/parse.go:89-95` leaves `Realization` nil on a failed
// decode, so the guard would be an exclusion nothing could make fire.
// Python judges such a page in `orphan`, `dead_link` and `untouched`
// for the same reason: it is in `pages`, and only the scaffold names
// are filtered out.
func judgedInBundle(p wiki.WikiPage) bool {
	return !wiki.IsScaffoldFile(p.Relative)
}

// targetKind is what a link or a `resource` turns out to name. Three
// answers and not two, because two rules read the same resolution and
// need different halves of it: `dead-link` acts on the paths it can look
// up, `outside-area` acts on exactly the ones it cannot -- but only on
// those that climb out, never on a deliberate reference by scheme.
type targetKind int

const (
	// targetInside is a path below the bundle root, ready to look up.
	targetInside targetKind = iota
	// targetElsewhere names nothing this bundle could hold: a scheme
	// (`brain://`, `https://`, `mailto:`) or no path at all.
	targetElsewhere
	// targetOutside climbs out of the bundle with `..`.
	targetOutside
)

// resolveTarget reads one target as a bundle-relative POSIX path.
//
// It shares three decisions with `okf/resolveLink`
// (`internal/brain/check/okf/errors.go:284-295`) and is not the same function. Shared:
// a leading slash is bundle-relative and not filesystem-absolute (OKF §6.1,
// and design §6 settles this repository's old disagreement, "die
// Bündelwurzel gewinnt"); a query is not part of the name; and the leading
// slash comes off only after the path is normalised, so `/../x` cannot
// climb. The last two were differences until the okf axis was corrected
// for them -- the query answer was a false `catalog-malformed` at the
// grade error, the climb a string ready to escape the bundle the day a
// caller put it on a filesystem.
//
// Run against seven targets from a page in `topics/`, five answers agree
// -- `b.md?x=1`, `/../escape.md`, `%zz.md`, `b%20b.md` and a plain
// `a.md` -- and three differences are left, measured rather than assumed:
//
//   - `brain://knowledge/c`: that one has no notion of a scheme and
//     answers `topics/brain:/knowledge/c`; this one answers
//     `targetElsewhere`.
//   - a target with no path left, such as a bare `?`: that one answers
//     the page's own directory, `topics`; this one answers
//     `targetElsewhere`.
//   - the third answer above: that one returns one string and its only
//     caller never has to tell "outside" from "inside".
//
// Borrowing it would also be the first time this package spoke to the okf
// one, which `page.go:1-4` says never happens. Unifying the two belongs
// to whoever gives `pkg/wiki` a resolver both axes read; until then this
// list is the only thing holding the two readings together, so it is kept
// current rather than left as it was first written.
//
// `url.Parse` does what Python's `urlsplit` plus `unquote` do in two
// steps: it names the scheme and hands back a decoded path
// (`src/brain/wiki/lint.py:265-268`). When it refuses the target
// altogether -- `%zz` is no escape -- the raw string is used as the path,
// which is where Python lands too, its `unquote` leaving a malformed
// escape standing. Such a target then matches no page and is reported
// dead, which is the direction a rule of this severity has to fail in.
func resolveTarget(relative, target string) (string, targetKind) {
	p := target
	if u, err := url.Parse(target); err == nil {
		if u.Scheme != "" {
			return "", targetElsewhere
		}
		p = u.Path
	}
	if p == "" {
		return "", targetElsewhere
	}
	if strings.HasPrefix(p, "/") {
		// Cleaned before the slash is stripped, so that `/../x` cannot
		// climb: `path.Clean` answers `/x` for it, as Python's
		// `normpath` does.
		return strings.TrimPrefix(path.Clean(p), "/"), targetInside
	}
	joined := path.Clean(path.Join(path.Dir(relative), p))
	if joined == ".." || strings.HasPrefix(joined, "../") {
		return "", targetOutside
	}
	return joined, targetInside
}

// targetsOf is everything a page points at: its Markdown links and the
// `resource` of every source entry. Scheibe 3 §4 names both -- "Geprüft
// werden wiki-interne Ziele und die `resource`-Pfade aus `sources[]`" --
// and Python builds the same pair in one tuple
// (`src/brain/wiki/lint.py:311`).
func targetsOf(p wiki.WikiPage) []string {
	out := make([]string, 0, len(p.Links)+len(p.Sources))
	out = append(out, p.Links...)
	for _, s := range p.Sources {
		out = append(out, s.Resource)
	}
	return out
}

// orphan reports Scheibe 3 §4 against schema rule 3: "keine andere Seite
// und kein `index.md` zeigt hierher", because "Jede Seite ist vernetzt"
// is a schema rule and therefore checkable.
//
// An error, and that is the point of writing it here.
// `wiki.LintBundle` reports it as a warning while
// `src/brain/wiki/lint.py:299` reports it as an error -- one of the two
// known divergences the design demands be gone (§9.3). Verified in the
// tree, not taken from the brief.
//
// Every catalog is a link source, at any depth. Scheibe 3 §4 says so
// twice and without qualification: the trigger is "keine andere Seite und
// kein `index.md` zeigt hierher" (line 179), and the scaffold paragraph
// adds "`index.md` wird aber für `orphan` gelesen -- es ist eine gültige
// Verweisquelle" (lines 187-188).
//
// Python reads `root / "index.md"` alone
// (`src/brain/wiki/lint.py:281-282`), because every scaffold file is gone
// from `pages` before a rule sees one. That narrower reading is a
// divergence here, and a deliberate one: `okf/catalog-malformed` requires
// each page to stand in the catalog of its *own* directory, so under the
// narrow reading a page that does exactly that -- listed by
// `topics/index.md` and nowhere else -- is faultless on the okf axis and
// an error on this one. Built and measured, not feared. This repository
// has four nested catalogs, and the wider reading is what keeps the two
// axes from contradicting each other on one page.
//
// The pages arrive with the scaffolds still among them, which is how this
// rule reaches those catalogs without a second trip to the disk; `Page`
// relies on the same and filters its own subjects.
//
// Only `p.Links`, never a `resource`. Python takes `page.links` alone
// (`src/brain/wiki/lint.py:290`), and a source citation is a claim about
// where a page came from, not a path a reader can walk to it.
//
// Existence is not asked. A target that resolves into the bundle counts
// as an inbound edge whether or not the file is there, because whether it
// is there is `dead-link`'s question: a rename then draws one `dead-link`
// on the page that kept the old name and one `orphan` on the page that
// took the new one -- two halves of one defect, each named where its
// repair is.
func orphan(pages []wiki.WikiPage) []check.Finding {
	linked := map[string]bool{}
	for _, p := range pages {
		if !judgedInBundle(p) && path.Base(p.Relative) != "index.md" {
			continue
		}
		for _, target := range p.Links {
			resolved, kind := resolveTarget(p.Relative, target)
			if kind == targetInside {
				linked[resolved] = true
			}
		}
	}

	var out []check.Finding
	for _, p := range pages {
		if !judgedInBundle(p) || linked[p.Relative] {
			continue
		}
		out = append(out, finding(p, "orphan",
			"no page and no catalog entry links here"))
	}
	return out
}

// deadLink reports Scheibe 3 §4: a target the bundle does not have. OKF
// §11 calls a dead cross-reference "not malformed ... not-yet-written
// knowledge" and forbids a *consumer* to reject over it; as the producer
// this is our only mechanical guard against link rot after a rename,
// because the `doc_id` self-healing covers `sources[]` and not Markdown
// links (design §5.2).
//
// The catalog is not judged, only read. Python hands `dead_link` the
// non-scaffold pages alone, and `okf/catalog-malformed` is the rule that
// asks whether a catalog lists its directory.
//
// A directory counts as existing, because Python asks
// `(context.root / resolved).exists()` (`src/brain/wiki/lint.py:316`),
// which a directory answers yes to. No stronger claim than that: measured
// over a real wiki bundle, not one of the 83 targets that reach this lookup
// is a directory -- the four directory links of that bundle all stand in its
// root `index.md`, which this rule never judges.
//
// The kind is asked before the lookup and not merely for clarity: the
// other two kinds carry the empty path, which joins to the bundle root
// itself. Measured in the mutation round -- with the guard struck out, a
// `brain://` reference makes this rule stat the root directory, which
// exists, so the defect would be invisible until a run against a root
// that does not.
func deadLink(pages []wiki.WikiPage, ctx Context) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedInBundle(p) {
			continue
		}
		for _, target := range targetsOf(p) {
			resolved, kind := resolveTarget(p.Relative, target)
			if kind != targetInside {
				continue
			}
			full := filepath.Join(ctx.Root, filepath.FromSlash(resolved))
			if _, err := os.Stat(full); err == nil {
				continue
			}
			// The raw target and the resolved path, both: Python gives
			// the reason at `src/brain/wiki/lint.py:323-326` -- the raw
			// target is what the reader can search for in the page, and
			// the resolved path alone would send them looking for a
			// string that never occurs in the source text.
			out = append(out, finding(p, "dead-link", fmt.Sprintf(
				"target does not exist: %s (resolved: %s)",
				target, resolved)))
		}
	}
	return out
}

// outsideArea reports Typkatalog §4.1: a link or source reference that
// leaves the area "wird gemeldet statt übergangen -- als Warnung, nicht
// als Fehler". The gap it closes is that `dead-link` cannot speak about a
// target it cannot look up, so without this rule such a reference would
// vanish from the check in silence -- "genau dann, wenn man sich nach der
// Migration am meisten auf sie verlassen will".
//
// A scheme goes through. §4.1 draws the line at the form: "`brain://`-
// Verweise gehen durch, nackte Pfade ins Nichts werden gemeldet."
//
// An absolute path is *not* reported, and the two sides agree on that
// now. Python once raised `absolute-link` there, because
// `graph.py` resolved such a target against the area root and `lint.py`
// did not; design §6 decides that disagreement in favour of the bundle
// root and drops the rule without replacement. Here, and there since
// then, an absolute target is an ordinary inside path that `dead-link`
// judges.
//
// Links and sources run in one loop, and Python does too again
// (`src/brain/wiki/lint.py:357-359`). It split them while
// `absolute-link` stood, solely so that the two messages of that rule
// could differ about the graph; the rule is gone and the split with it.
func outsideArea(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedInBundle(p) {
			continue
		}
		for _, target := range targetsOf(p) {
			if _, kind := resolveTarget(p.Relative, target); kind !=
				targetOutside {
				continue
			}
			out = append(out, warning(p, "outside-area",
				"target leaves the area and cannot be checked: "+target))
		}
	}
	return out
}

// cutoff is the instant a page has to be older than to count as
// untouched, and both age rules take it from here -- one threshold, one
// piece of arithmetic. Python does the same subtraction twice
// (`src/brain/wiki/lint.py:172,223`) from the same `context.now` and the
// same `context.untouched_days`.
//
// Days as a duration, not as calendar days: `timedelta(days=n)` is
// exactly n * 24h, and `AddDate` would instead land on a wall-clock date
// and drift by an hour across a daylight-saving boundary.
//
// Instants, not calendar days -- unlike `stale` next door, which compares
// two dates because a YAML date names a day and nothing finer. Here both
// sides are timestamps carrying a time of day, and Python compares them
// as such (`:181`).
//
// A zero `ctx.Now` puts the cutoff before every real modification time,
// so both rules go silent on a bundle whose pages are all far too old.
// That is the caller's to answer for and not a case to be caught here:
// the caller fills `Now`, as Python's `lint_bundle` does at
// `src/brain/wiki/lint.py:575` rather than letting a rule reach for a
// clock -- a rule reading `time.Now()` would let two runs over one
// unchanged bundle disagree.
func cutoff(ctx Context) time.Time {
	return ctx.Now.Add(-time.Duration(ctx.UntouchedDays) * 24 * time.Hour)
}

// untouched reports Scheibe 3 §4: a page nobody has touched for longer
// than the threshold the manifest declares, default 180 days. A warning,
// and Architektur §9.4 says why: "Eine gestern geschriebene Seite kann
// überholt sein, eine hundert Tage alte weiterhin stimmen; deshalb ist
// der Inhalts-Hash das starke Signal und das Datum nur ein Hinweis."
//
// A page whose modification time could not be read carries the zero time
// (`internal/brain/wiki/parse.go:66-74` decides that and says so), which is older
// than any cutoff, so it is reported. That is the reader's ruling rather
// than this rule's, and it errs towards speaking: a page whose age nobody
// could read is not one this rule can vouch for.
func untouched(pages []wiki.WikiPage, ctx Context) []check.Finding {
	limit := cutoff(ctx)
	var out []check.Finding
	for _, p := range pages {
		if !judgedInBundle(p) || !p.ModTime.Before(limit) {
			continue
		}
		out = append(out, warning(p, "untouched", fmt.Sprintf(
			"unchanged for more than %d days", ctx.UntouchedDays)))
	}
	return out
}

// implementedWithoutCommit reports Architektur §9.4: pages "die
// `implemented` melden, ohne einen Commit in `implemented_in` zu nennen".
// Without the commit the claim cannot be checked, and an unchecked claim
// is the very thing the wiki layer exists to prevent.
//
// Silent outside a project bundle. §9.4 introduces both realization rules
// with "In Projekt-Bundles kommen zwei Prüfungen dazu", and Python
// returns early on `is_project` (`src/brain/wiki/lint.py:196-197`) -- so
// a `knowledge` bundle that carries the field anyway is not this rule's
// business. The brief names no such condition; the source does.
//
// A blank `implemented_in` names no commit and counts as none. Python
// asks `not page.implemented_in`, which lets a whitespace-only value
// through; the trimming here follows this package's own precedent
// (`page.go:147`, where `sourceIncomplete` trims the three fields Python
// tests for truth) and reports one page more than Python on an input no
// bundle here holds.
func implementedWithoutCommit(
	pages []wiki.WikiPage, ctx Context,
) []check.Finding {
	if !ctx.IsProject {
		return nil
	}
	var out []check.Finding
	for _, p := range pages {
		if !judgedInBundle(p) || p.Realization == nil ||
			*p.Realization != "implemented" {
			continue
		}
		if p.ImplementedIn != nil &&
			strings.TrimSpace(*p.ImplementedIn) != "" {
			continue
		}
		out = append(out, finding(p, "implemented-without-commit",
			"realization: implemented without implemented_in"))
	}
	return out
}

// longPlanned reports the other half of Architektur §9.4: "Seiten, die
// seit langem `planned` sind". A warning, like `untouched`: a plan that
// stood still may still be the plan.
//
// The threshold is `untouched`'s, and Python states the reason at
// `src/brain/wiki/lint.py:213-214`: "The threshold is the one the area
// already declares for `untouched`: a second number would be a second
// setting to keep honest." No second setting is invented here.
//
// The overlap with `untouched` on one page is deliberate, and `:216-219`
// says why: that rule speaks to a page's age regardless of `realization`,
// this one adds the narrower claim that a specific plan was never picked
// up. Two questions about one fact, so neither suppresses the other.
//
// The message names the day the modification time carries, in its own
// zone. Converting the instant first would name a different day for every
// file written in the small hours east of UTC, and a day nobody would
// find in a file listing is worse than no day at all. Python reads the
// components as written too: `page.mtime` is
// `datetime.fromtimestamp(...).astimezone()`
// (`src/brain/wiki/page.py:151`) and `.date()` takes that zone's day
// (`src/brain/wiki/lint.py:229`).
func longPlanned(pages []wiki.WikiPage, ctx Context) []check.Finding {
	if !ctx.IsProject {
		return nil
	}
	limit := cutoff(ctx)
	var out []check.Finding
	for _, p := range pages {
		if !judgedInBundle(p) || p.Realization == nil ||
			*p.Realization != "planned" || !p.ModTime.Before(limit) {
			continue
		}
		out = append(out, warning(p, "long-planned", fmt.Sprintf(
			"planned and untouched since %s",
			p.ModTime.Format("2006-01-02"))))
	}
	return out
}
