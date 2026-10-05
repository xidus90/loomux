package okf

import (
	"fmt"
	"path"
	"strings"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
)

// Soft runs the four warnings and three notes of the okf axis, the second
// and third degree of one principle: what OKF makes a MUST is an error and
// lives next door, what it calls Recommended or SHOULD is a warning, and
// what it merely permits is a note that may never fail a bundle. §11 spells
// the last part out by name -- a consumer "MUST NOT reject a bundle because
// of ... Missing `index.md` files".
//
// `resource` and `tags` get no rule at all, although §4.1 lists both under
// Recommended. OKF supplies the exception in the same breath: "`resource`:
// ... Absent for concepts that describe abstract ideas rather than physical
// resources." Three of this schema's four page types -- Topic, Entity,
// Synthesis -- describe exactly that, so a warning on them would be the
// message one learns to skip, and a skipped message protects nothing. The
// design (line 139) still had the two as notes; the brief drops them, and
// its own test holds the axis to it.
//
// The Context goes unread, for the reason `Errors` gives: everything these
// seven turn on stands on the pages, and the bundle root reached them as
// the base of `WikiPage.Relative`. The parameter stays because it is the
// shape the house rules share.
//
// Nothing here sorts. `check.Sort` is the job of whoever merges the axes,
// and sorting twice would only make the second call decide the order.
func Soft(pages []wiki.WikiPage, _ Context) []check.Finding {
	var out []check.Finding
	out = append(out, titleMissing(pages)...)
	out = append(out, descriptionMissing(pages)...)
	out = append(out, sourceIDMissing(pages)...)
	out = append(out, indexEntryWithoutDescription(pages)...)
	out = append(out, noIndex(pages)...)
	out = append(out, noLog(pages)...)
	out = append(out, noTrustFamily(pages)...)
	return out
}

// soft fills the two fields every rule in this file shares. `finding` next
// door hardcodes `check.Error` on purpose, so a second degree needs a second
// door rather than a severity parameter added to that one: the nine musts
// must stay unable to report anything but an error.
//
// It takes a path and not a page, unlike `finding`, because two of the seven
// report a file that is not there. `no-index` and `no-log` name the missing
// catalog and the missing log, and there is no `WikiPage` for either.
func soft(relative string, degree check.Severity,
	rule, message string) check.Finding {
	return check.Finding{
		Relative: relative,
		Axis:     check.AxisOKF,
		Rule:     rule,
		Severity: degree,
		Message:  message,
	}
}

// warning and note are the two doors. The severity stays out of the call
// sites the same way `finding` keeps it out of the nine musts, so no rule
// can quietly change its own degree.
func warning(p wiki.WikiPage, rule, message string) check.Finding {
	return soft(p.Relative, check.Warning, rule, message)
}

func note(relative, rule, message string) check.Finding {
	return soft(relative, check.Note, rule, message)
}

// judgedAsConcept says whether the page rules look at a page at all.
//
// Two exclusions, each with its own reason. A page whose frontmatter did not
// parse reads empty in every field, so `title`, `description` and the trust
// keys may all stand in the block and still be reported missing -- a false
// finding, and `okf/frontmatter-unparsable` already owns the page. A
// scaffold file is no concept document: §3.1 reserves `index.md` and
// `log.md`, this repository adds three more names that carry structure
// rather than knowledge, and §4.1 and §5 speak about concepts throughout.
// The wider set follows `typeMissing`, which drops the same five.
func judgedAsConcept(p wiki.WikiPage) bool {
	return p.BrokenFrontmatter == nil && !wiki.IsScaffoldFile(p.Relative)
}

// titleMissing reports OKF §4.1, which lists `title` under Recommended:
// "Human-readable display name. If omitted, consumers MAY derive a title
// from the filename." A derived title is what the message names, because
// that is what the reader gets instead.
func titleMissing(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) || strings.TrimSpace(p.Title) != "" {
			continue
		}
		out = append(out, warning(p, "title-missing",
			"no title; a consumer can only derive one from the filename"))
	}
	return out
}

// descriptionMissing reports OKF §4.1, which lists `description` under
// Recommended: "A single sentence summarizing the concept. Used by
// `index.md` generators, search snippets, and previews."
//
// Two keys, two repairs, so two messages. This rule and the one above fire
// together on nearly every page that states neither, and a shared message
// would let them collapse into one.
func descriptionMissing(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) || strings.TrimSpace(p.Description) != "" {
			continue
		}
		out = append(out, warning(p, "description-missing",
			"no description; catalogs and search snippets show nothing"))
	}
	return out
}

// sourceIDMissing reports OKF §5.1: "`id`: Optional. A stable key used to
// attribute individual claims. SHOULD be present when the body cites the
// source."
//
// The condition is dropped and the superset reported, as the design settles
// it at line 125. Binding the warning to §5.1's own citation form was
// weighed and measured instead of argued: §5.1 attributes a claim through a
// markdown footnote whose label is the `id`, so `[^label]` in the body is
// what "cites the source" looks like mechanically. Counted over every
// tracked `.md` whose text holds a line `sources:` at column zero, the OKF
// spec itself excluded: 39 pages carry such a block -- 24 of them in one
// wiki bundle -- and 0 carry a single footnote marker. A rule
// bound to that form would be silent on every page of every bundle here,
// which is worse than a broad warning -- it would be a rule that cannot
// fire. Prose names a source without marking it, and that is the case no
// rule can catch. The measurement stands here so the next reader can redo
// it rather than re-argue it.
//
// The house axis will make `id` mandatory, which is why this warning is not
// expected to land alone in this repository's own bundles. Not yet, though:
// `house/source-incomplete` does not exist in the tree, and the design
// (line 156) lists only `doc_id`, `content_hash` and `revision` for it. It
// is `task-7-brief.md:10` that first names `ID` among that rule's fields.
//
// No guard against an unreadable frontmatter here: such a page carries no
// `Sources` at all, so the loop never runs and a guard would be a branch
// no input can reach.
func sourceIDMissing(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		for i, s := range p.Sources {
			// The position, not just the page, exactly as
			// `sourceResourceMissing` names it: one broken entry among many
			// is the normal case.
			if strings.TrimSpace(s.ID) == "" {
				out = append(out, warning(p, "source-id-missing",
					fmt.Sprintf("sources[%d] has no id", i)))
			}
		}
	}
	return out
}

// indexEntryWithoutDescription reports OKF §8: "Entries SHOULD include the
// description from the linked concept's frontmatter." Progressive
// disclosure is what §8 asks a catalog for, and an entry that is a bare
// link makes the reader open the file to learn what it holds -- the one
// thing the catalog exists to spare them.
//
// The entries are `p.Links` on an `index.md`, which is the very list
// `catalogMalformed` judges. What is shared is exactly the *set* of
// entries, and no more: reading the text behind one is this rule's own
// business, because the error rule never needed it. That much sharing is
// the point -- a line-based parser would newly decide what a listing line
// looks like, and the two rules would then disagree about what a catalog
// contains. The error rule's blind spot comes along with its definition:
// an external link never reaches `p.Links`, so neither rule judges one.
//
// No guard on an unreadable frontmatter, unlike `judgedAsConcept`. That
// exclusion exists because the fields it guards read empty for no reason
// but the failed decode; `Links` and `RawBody` are untouched by it, so a
// catalog whose block did not parse still lists what it lists. The line
// runs along what a rule reads, not along the page.
func indexEntryWithoutDescription(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if path.Base(p.Relative) != "index.md" {
			continue
		}
		// A cursor rather than a fresh search per entry: `p.Links` is in
		// body order, and a catalog that lists the same target twice would
		// otherwise have both entries answered by the first occurrence.
		cursor := 0
		for _, target := range p.Links {
			description, next := entryDescription(p.RawBody, target, cursor)
			cursor = next
			if description != "" || !describable(target) {
				continue
			}
			out = append(out, warning(p, "index-entry-without-description",
				fmt.Sprintf("catalog entry %s carries no description", target)))
		}
	}
	return out
}

// describable answers whether §8's recommendation has an object behind
// one entry, and the wording is where the line is drawn: "Entries SHOULD
// include the description **from the linked concept's frontmatter**".
// What carries no frontmatter cannot supply one, and a warning about it
// asks the reader for something no page could give.
//
// Two kinds, and both were counted before the line was drawn. All 52
// lines of the standard output of `brain check all` over the registered
// stock were findings of this rule, and 51 of them were of one of these
// two kinds -- 43 about a scaffold file, 8 about a subdirectory. The
// 52nd is the rule at work and stays: a catalog line that does not
// quote a description the linked page's frontmatter carries. So the
// rule was the whole output while all but one line of it said nothing a
// reader could act on. Design §4 names exactly that danger for the
// third grade: "Ein Hinweis, den man überspringen lernt, schützt
// nichts."
//
// A subdirectory is told by the trailing slash and not by asking the
// disk. The generator writes it that way and only that way
// (`src/brain/catalog.py:51`, `_destination(f'{name}/')`), and a rule
// that stat'ed its target would answer differently on two machines --
// the same reason `catalogTargets` next door resolves lexically.
//
// A scaffold file is told by its base name, which is what
// `wiki.IsScaffoldFile` asks and what makes a `../92 Engineering/python/
// index.md` in the signpost's catalog count as one. Those seven entries
// are the case that a check on the bundle's own scaffold names would
// have missed.
func describable(target string) bool {
	return !strings.HasSuffix(target, "/") && !wiki.IsScaffoldFile(target)
}

// entryDescription reads the text a catalog line puts behind one entry and
// returns it with the offset to continue the scan from.
//
// Total by construction. Through the reader neither miss is reachable,
// because the target was cut out of this very body by the link pattern; the
// guards stand so that a caller handing in a foreign body gets an empty
// answer rather than a panic, and the tests reach them through this
// function directly.
func entryDescription(body, target string, from int) (string, int) {
	marker := "](" + target
	next := from
	for {
		at := strings.Index(body[next:], marker)
		if at < 0 {
			return "", len(body)
		}
		next += at + len(marker)
		// Not every `](target` in the body is the entry `Links` recorded.
		// The reader's pattern refuses a link title, so `[X](a.md "T")`
		// never reaches `Links` while its marker still stands in the text;
		// taking it would let that line describe an entry listed further
		// down. Only a target closed right here counts -- by the paren, or
		// by a `#fragment`, which the reader drops from what it records
		// (`internal/brain/wiki/parse.go:44`, the `(?:#[^\)]*)?` group). §6.1 names no
		// fragment itself; that part is the reader's doing, not OKF's.
		// A link inside a fenced example does reach `Links` and is warned
		// about here -- the same blind spot `catalogMalformed` has, and
		// deliberately the same, so both rules read one set of entries.
		if next >= len(body) {
			return "", len(body)
		}
		if c := body[next]; c != ')' && c != '#' {
			continue
		}
		return descriptionAfter(body, next), next
	}
}

// descriptionAfter reads from the closing paren of a link to the end of its
// line. Split off so that the scan above stays a loop over candidates and
// this stays the reading of one line.
func descriptionAfter(body string, next int) string {
	rest := body[next:]
	end := strings.IndexByte(rest, ')')
	if end < 0 {
		return ""
	}
	line := rest[end+1:]
	if nl := strings.IndexByte(line, '\n'); nl >= 0 {
		line = line[:nl]
	}
	// The separator is convention, not syntax: §8's example writes " - ",
	// and a producer reaching for an en or em dash means the same thing. A
	// line that is nothing but the separator describes nothing.
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line),
		"-–—"))
}

// noIndex reports OKF §8 from the permissive side: an `index.md` "MAY
// appear in any directory", so its absence is a note and never an error --
// §11 forbids rejecting a bundle over "missing `index.md` files" in those
// words, and `catalogMalformed` next door fires only on a catalog that is
// present.
//
// A directory counts once its walk yields any page other than its own
// catalog, scaffold files included: §8 has the catalog enumerate "the
// directory's contents", and this repository's own root catalog duly lists
// `log.md` and `_schema.md`.
func noIndex(pages []wiki.WikiPage) []check.Finding {
	catalogued := map[string]bool{}
	for _, p := range pages {
		if path.Base(p.Relative) == "index.md" {
			catalogued[path.Dir(p.Relative)] = true
		}
	}

	var out []check.Finding
	// Over the pages in read order and deduplicated by hand, not over the
	// map: two runs of an unchanged bundle have to render the same output,
	// and a map hands its keys out in a different order every time.
	noted := map[string]bool{}
	for _, p := range pages {
		dir := path.Dir(p.Relative)
		if path.Base(p.Relative) == "index.md" || catalogued[dir] ||
			noted[dir] {
			continue
		}
		noted[dir] = true
		// `path.Dir` answers "." for a page at the bundle root, and
		// `path.Join` drops it again -- so the root catalog is named
		// "index.md", the way `indexFrontmatterMisplaced` identifies it.
		out = append(out, note(path.Join(dir, "index.md"), "no-index",
			"no index.md; nothing lists this directory's contents"))
	}
	return out
}

// noLog reports OKF §9 from the permissive side: a `log.md` "MAY appear at
// any level of the hierarchy", so its absence is a note.
//
// Only the bundle root is asked after, although §9 allows one at any level.
// Noting every level without a log would put a note on every directory of
// every bundle here, which is the message one learns to skip; the root is
// the one place a reader looks for the bundle's own history.
//
// A tree without a single page is not a bundle whose log went missing, so
// an empty page set yields nothing. The alternative would report a bundle
// that is not there.
func noLog(pages []wiki.WikiPage) []check.Finding {
	if len(pages) == 0 {
		return nil
	}
	for _, p := range pages {
		if p.Relative == "log.md" {
			return nil
		}
	}
	return []check.Finding{note("log.md", "no-log",
		"no log.md at the bundle root; its history is unrecorded")}
}

// noTrustFamily reports OKF §5.2 and §5.3: a page states neither
// `generated` nor `verified`. §5.3 settles the degree in its own words --
// "A concept with no trust frontmatter is still consumable; consumers MUST
// NOT reject it" -- so this is a note.
//
// It asks `FrontmatterKeys` and not the typed fields. Two reasons, and the
// first is a gap in the reader: `wiki.Frontmatter` has no `verified` field
// at all (`internal/brain/wiki/model.go:39-59`), so `p.Generated` is the only trust
// state a typed check could see, and a page verified by a human but never
// marked generated would be noted as stating no trust. The keys are already
// collected for §12 (`internal/brain/wiki/parse.go:116-127`), so the rule needs no
// change to the reader. The second reason is that §5.3 makes the key itself
// the criterion -- "No `verified` key ⇒ unverified" -- which also settles
// `generated:` written with no value: the pointer stays nil while the key
// stands, and that block answers to `okf/generated-by-missing` already.
func noTrustFamily(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if !judgedAsConcept(p) {
			continue
		}
		stated := false
		for _, key := range p.FrontmatterKeys {
			if key == "generated" || key == "verified" {
				stated = true
			}
		}
		if stated {
			continue
		}
		out = append(out, note(p.Relative, "no-trust-family",
			"neither generated nor verified; the page states no trust"))
	}
	return out
}
