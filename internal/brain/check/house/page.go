// Package house holds the checks of the `house` axis: the stricter
// question of whether a bundle satisfies this repository's own specs. OKF
// conformance is the business of the package next door, which never speaks
// here.
package house

import (
	"fmt"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
)

// Context is the bundle-wide input the rules share. It is an alias and not
// a second struct, for the reason `internal/brain/check/okf/errors.go` gives for its
// own: the reader already fills `wiki.BundleContext` with exactly these
// fields, and two structs would have to be kept in step by hand until the
// first field that drifted made the two axes judge one bundle differently.
type Context = wiki.BundleContext

// Page runs the five house rules that decide at a single page. Every
// finding is an error: the design's house table (§5.2) puts all five under
// Fehler, and each of them is stricter than OKF on purpose -- OKF §11
// forbids a *consumer* to reject a bundle over a missing source, an
// unknown type or a dead link, and we are the producer of this one.
//
// Only the Context this axis actually reads is taken: `Now` for `stale` and
// `DeclaredTypes` for `unknown-type`. The other rules turn on the pages
// alone, which is why the okf package can ignore the parameter entirely.
//
// Nothing here sorts. `check.Sort` is the job of whoever merges the axes,
// and sorting twice would only make the second call decide the order.
func Page(pages []wiki.WikiPage, ctx Context) []check.Finding {
	var out []check.Finding
	out = append(out, noSources(pages)...)
	out = append(out, sourceIncomplete(pages)...)
	out = append(out, unknownType(pages, ctx)...)
	out = append(out, conflictCount(pages)...)
	out = append(out, stale(pages, ctx)...)
	return out
}

// finding fills in the two fields all five rules share, so that no rule can
// state its axis or its severity differently from its four neighbours. The
// severity is hardcoded rather than passed: the day this axis gains a
// warning, that warning needs its own door, or a rule could quietly demote
// itself out of the exit code.
func finding(p wiki.WikiPage, rule, message string) check.Finding {
	return check.Finding{
		Relative: p.Relative,
		Axis:     check.AxisHouse,
		Rule:     rule,
		Severity: check.Error,
		Message:  message,
	}
}

// judgedAsConcept says whether the page rules look at a page at all. It
// draws the same line as its namesake in `internal/brain/check/okf/soft.go`, and all
// five rules here follow it -- but the reason is this axis's own, and it is
// stronger than the OKF one.
//
// Scaffold files: Scheibe 3 §4 exempts "die Gerüstdateien `_schema.md`,
// `index.md`, `log.md`, `audit.md`, `_identities.tsv`" from these very
// rules by name, and Python honours that by never handing such a file to
// any rule at all (`src/brain/wiki/lint.py:582-585`). The design makes
// agreement with Python the thing this work has to prove (§9.1), so a
// rule judging one here would be a divergence by construction. The case is
// measured, not feared: `docs/wiki/_schema.md` documents the conflict
// syntax with a `> [!conflict]` example, which `conflictBoxRe`
// (`internal/brain/wiki/parse.go:45`) counts as a box -- without this line the file
// that defines the rule is its first violator.
//
// Unreadable frontmatter: four of the five read a field out of the block
// that did not decode, so they would report a page for stating nothing
// when it may have stated everything. `conflict-count` is the one that
// does not fall to that argument on its own -- its boxes are counted in
// the body and survive the failed decode -- and it falls to the sharper
// half of the same one: `DeclaredConflicts` does not survive, so the page
// would be blamed for bookkeeping it may well have done.
// `okf/frontmatter-unparsable` owns such a page, and one defect under two
// names sends the reader to the wrong repair.
func judgedAsConcept(p wiki.WikiPage) bool {
	return p.BrokenFrontmatter == nil && !wiki.IsScaffoldFile(p.Relative)
}

// noSources reports Scheibe 3 §4: "`sources[]` fehlt oder ein Eintrag ist
// unvollständig ... Gilt für alle vier Typen -- auch eine `Synthesis`
// belegt ihre Thesen." OKF leaves `sources` free throughout, so without
// this duty the house axis would have nothing to add on a page's origin.
//
// The message is Python's, word for word (`src/brain/wiki/lint.py:128`).
// The two axes have to agree on name, severity, path *and* message before
// Python can be retired (design §9.1), and a message invented here would
// be one more difference to reconcile later.
func noSources(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) || len(p.Sources) > 0 {
			continue
		}
		out = append(out, finding(p, "no-sources", "sources[] is empty"))
	}
	return out
}

// sourceIncomplete reports the second half of the same Scheibe 3 §4 rule:
// "Unvollständig heißt: `doc_id`, `content_hash` oder `revision` fehlt."
// The design (§5.3) splits it off under its own name, because the repair
// differs -- an empty list needs a source, a broken entry needs a field.
//
// Three fields and no more. The design calls them "die drei Felder, an
// denen die Wartung hängt", and Scheibe 3 §4 lists the same three.
//
// Two fields Python asks for in the same expression
// (`src/brain/wiki/lint.py:135`) are handed to the okf axis instead,
// because OKF binds both and reporting them here would put one defect
// on two axes:
//
//   - `resource`: OKF §5.1 makes it REQUIRED, and
//     `okf/source-resource-missing` owns it.
//   - `id`: OKF §5.1 puts it under SHOULD, and `okf/source-id-missing`
//     owns it. It was inside this rule until the first fix round, on the
//     strength of `task-7-brief.md:10` -- but a brief is a plan and the
//     design is the binding paper above it, and measured, an entry
//     without `id` drew both that warning and this error at once. `id` is
//     the footnote label of §5.1, not the maintenance trail.
//
// The message cannot mirror Python's, which folds both halves into one
// `no-sources` line. It follows the shape the okf source rules use
// instead -- the position, not just the page, because one broken entry
// among many is the normal case.
func sourceIncomplete(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) {
			continue
		}
		for i, s := range p.Sources {
			var missing []string
			if strings.TrimSpace(s.DocID) == "" {
				missing = append(missing, "doc_id")
			}
			if strings.TrimSpace(s.ContentHash) == "" {
				missing = append(missing, "content_hash")
			}
			// The pointer, not a value: `revision: 0` is a revision, and a
			// rule reading the int alone would demand a field the page has.
			if s.Revision == nil {
				missing = append(missing, "revision")
			}
			if len(missing) == 0 {
				continue
			}
			out = append(out, finding(p, "source-incomplete",
				fmt.Sprintf("sources[%d] lacks %s", i,
					strings.Join(missing, ", "))))
		}
	}
	return out
}

// unknownType reports Scheibe 3 §4: "Ein Tippfehler ist ein Befund, kein
// neuer Typ." OKF §4.1 asks a consumer to tolerate a type it does not
// know; we keep a catalogue and are the producer, so here it is an error.
//
// That severity is the point of this rule's existence in this shape. Go
// reports an unknown type as a *warning* today (`wiki.LintSingleFile`,
// `check.Warning`) while Python reports it as an error
// (`src/brain/wiki/lint.py:90`, `Severity.ERROR`) -- one of the two known
// divergences the design demands be gone (§9.3). Verified in the tree, not
// taken from the brief.
//
// A page with no usable `type` is left alone. `okf/type-missing` fires on
// both an absent and a blank one (`internal/brain/check/okf/errors.go`, its two
// messages), and `KnowsType("")` is false, so without the guard every such
// page would carry two names for one defect.
//
// The vocabulary comes from the Context and not from a manifest read here.
// `ctx.Root` is the *bundle* root -- the base of `WikiPage.Relative` --
// while `config.ReadManifest` looks for `.loomux/config.toml` at exactly
// the path it is given and never walks upward. In
// this repository the manifest sits at the repo root and declares
// `layout.wiki = "docs/wiki"`, so a read at `ctx.Root` would miss it on
// every real bundle and the rule's fallback would be its only path. Python
// settles it the same way: `missing_type` reads `context.declared_types`
// and the caller fills it (`src/brain/wiki/lint.py:579`).
//
// The map is turned back into a slice because `KnowsType` is the only way
// to reach the built-in vocabulary, which is unexported. Its order does
// not matter -- the answer is a bool over a membership test, not a
// rendering, so the determinism this package guards elsewhere is not at
// stake. An empty or nil map is the normal case, not a defect: it means
// the area declares no types of its own and the built-in set decides
// alone, which is what this repository's `.loomux/config.toml` does.
func unknownType(pages []wiki.WikiPage, ctx Context) []check.Finding {
	declared := make([]string, 0, len(ctx.DeclaredTypes))
	for t := range ctx.DeclaredTypes {
		declared = append(declared, t)
	}
	manifest := &config.Manifest{DeclaredTypes: declared}

	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) || p.PageType == nil {
			continue
		}
		// Trimmed for the blank guard only, and asked untrimmed.
		// `rank_of` normalises nothing, so a quoted `type: " Topic "`
		// is unknown to Python and has to be unknown here too. The
		// guard is there for a *blank* type, which `okf/type-missing`
		// reports under its own name.
		pageType := *p.PageType
		if strings.TrimSpace(pageType) == "" || manifest.KnowsType(pageType) {
			continue
		}
		// The message names the two ways out, as Python's does
		// (`src/brain/wiki/lint.py:107-110`). Python has a third branch
		// there for a known old name, which needs the alias table this
		// package has no reader for yet; the reconciliation of the two
		// messages belongs to whoever writes the agreement test.
		out = append(out, finding(p, "unknown-type", fmt.Sprintf(
			"%q is neither core nor catalogue; add it to `[wiki] types` "+
				"in the area's manifest, or use a catalogue name",
			pageType)))
	}
	return out
}

// conflictCount reports Architektur §9.3: `open_conflicts` in the
// frontmatter against the boxes actually in the body -- "die Buchführung
// der KI wird von Code kontrolliert". Scheibe 3 §4 adds the first arm by
// name: "Fehlendes Feld bei vorhandenen Kästen ist ebenfalls ein Befund."
//
// Two arms, two messages, and the second must not swallow the first: a
// page that states no number has to gain the key, a page that states the
// wrong one has to correct it. Both messages are Python's word for word
// (`src/brain/wiki/lint.py:447-449,458-460`). `wiki.LintSingleFile`
// phrases the same two differently; Python is the reference, because it is
// the side that stays until the agreement is proven.
//
// A page without boxes and without the key stays silent. Demanding the
// field everywhere would put a zero on every page of every bundle.
func conflictCount(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) {
			continue
		}
		switch {
		case p.DeclaredConflicts == nil:
			if p.FoundConflicts == 0 {
				continue
			}
			out = append(out, finding(p, "conflict-count", fmt.Sprintf(
				"%d conflict box(es) present, open_conflicts is missing",
				p.FoundConflicts)))
		case *p.DeclaredConflicts != p.FoundConflicts:
			out = append(out, finding(p, "conflict-count", fmt.Sprintf(
				"open_conflicts says %d, %d box(es) found",
				*p.DeclaredConflicts, p.FoundConflicts)))
		}
	}
	return out
}

// stale reports Scheibe 3 §4: "`stale_after` liegt in der Vergangenheit --
// Fehler, weil die Seite es selbst angekündigt hat."
//
// The day the page names is its last good one, not its first bad one, so
// the comparison is strictly `<` against today and a page expiring today
// stays silent. Python decides it the same way and at the same grain:
// `page.stale_after < today` with `today = context.now.date()`
// (`src/brain/wiki/lint.py:153,162`), where both sides are naive dates.
//
// Hence `calendarDay` below rather than `Before` on the two instants.
// `ctx.Now` carries a time of day while a YAML date parses to midnight, so
// comparing instants would call a page expired at every moment of the day
// it named but the very first -- the opposite of what §4 says, for all but
// an instant of that day. The measurement is against `ctx.Now` and never
// against `time.Now()`, or two runs of one bundle could disagree.
func stale(pages []wiki.WikiPage, ctx Context) []check.Finding {
	today := calendarDay(ctx.Now)
	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) || p.StaleAfter == nil {
			continue
		}
		if !calendarDay(*p.StaleAfter).Before(today) {
			continue
		}
		out = append(out, finding(p, "stale", fmt.Sprintf(
			"stale_after %s has passed",
			p.StaleAfter.Format("2006-01-02"))))
	}
	return out
}

// calendarDay drops everything below the calendar date, and the zone with
// it. The two sides arrive in different zones -- `ctx.Now` in the
// caller's, a YAML date in UTC -- and converting one into the other would
// shift the date itself. Python compares two naive dates and so ignores
// the question; reading the components as written does the same.
func calendarDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
