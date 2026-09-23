// Package okf holds the checks of the `okf` axis: the question whether a
// foreign reader of the Open Knowledge Format may consume this bundle at all.
// The house axis, which asks the stricter question of this repository, lives
// next door and never speaks here.
package okf

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
)

// Context is the bundle-wide input the rules share. It is an alias, not a
// second struct: the plan writes `Context`, while the reader already fills
// `wiki.BundleContext` with exactly these fields. A separate type would have
// to be kept in step with that one by hand, and the first field that drifted
// would make two rules judge the same bundle differently.
type Context = wiki.BundleContext

// isoDay is the form OKF §9 requires of a date heading: "Date headings MUST
// use ISO 8601 `YYYY-MM-DD` form." It judges the form and not the day --
// whether 2026-13-45 exists is a second question, and answering it here would
// report a heading under a rule whose name promises the other one.
//
// Compiled on first use, not at package load: the start path compiles nothing.
var isoDay = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`) })

// Errors runs the nine musts of OKF over a page set. Every finding is an
// error on the okf axis: each of the nine reports a MUST or a REQUIRED, and
// what OKF only recommends is the business of the warnings next to this file.
//
// The Context goes unread. Everything the nine turn on stands on the pages
// themselves -- the bundle root that §8 and §12 hinge on reached them as the
// base of `WikiPage.Relative`, and asking `ctx.Root` again would put the same
// question twice, with two chances of a different answer. The parameter stays
// because it is the shape the warnings and the house rules share.
func Errors(pages []wiki.WikiPage, _ Context) []check.Finding {
	var out []check.Finding
	out = append(out, frontmatterUnparsable(pages)...)
	out = append(out, typeMissing(pages)...)
	out = append(out, sourceResourceMissing(pages)...)
	out = append(out, generatedByMissing(pages)...)
	out = append(out, reservedNameAsConcept(pages)...)
	out = append(out, catalogMalformed(pages)...)
	out = append(out, logDateForm(pages)...)
	out = append(out, computationRuntimeMissing(pages)...)
	out = append(out, indexFrontmatterMisplaced(pages)...)
	return out
}

// finding fills in the two fields all nine rules share, so that no rule can
// state its axis or its severity differently from its eight neighbours. All
// nine build through it, including the one the plan spelled out by hand: the
// plan fixes that rule's name and message, not the way its Finding is put
// together.
func finding(p wiki.WikiPage, rule, message string) check.Finding {
	return check.Finding{
		Relative: p.Relative,
		Axis:     check.AxisOKF,
		Rule:     rule,
		Severity: check.Error,
		Message:  message,
	}
}

// frontmatterUnparsable reports OKF §11.1: every non-reserved .md file
// contains a parseable YAML frontmatter block.
//
// It judges the reserved files too, although neither §11.1 nor the
// design binds them -- the design's table writes "YAML-Block einer
// Nicht-Gerüstdatei nicht lesbar" at line 109. The wider reach is
// deliberate, and this is the one place the axis departs from that
// table.
//
// A bundle-root `index.md` whose block did not parse carries
// frontmatter (§8 lets it) but yields no keys, so
// `indexFrontmatterMisplaced` finds nothing to name and stays silent.
// Skipping the reserved names here would leave that file with no rule
// at all.
func frontmatterUnparsable(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if p.BrokenFrontmatter != nil {
			out = append(out, finding(p, "frontmatter-unparsable",
				"frontmatter is unreadable: "+*p.BrokenFrontmatter))
		}
	}
	return out
}

// typeMissing reports OKF §11.2: every frontmatter block contains a non-empty
// `type` field -- "the only always-required key" of §4.1.
//
// Scaffold files are exempt, all five that `wiki.IsScaffoldFile` knows. §11.1
// and §11.2 bind the non-reserved files, which by §3.1 are only `index.md` and
// `log.md`; the design's table names no exemption here at all (line 110). The
// wider set is the owner's ruling, and it follows this repository's own
// practice rather than the letter: `wiki.LintSingleFile` and
// `src/brain/wiki/lint.py:585` both drop the same five before a rule sees
// them. `_schema.md` and `audit.md` are house
// scaffolding with no OKF type to give, so reporting them would be a false
// finding. Measured, the letter costs exactly those two pages in docs/wiki.
//
// A page whose frontmatter did not parse stays with the rule above. It has no
// type either, and two names for one defect send the reader to the wrong
// repair.
func typeMissing(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if p.BrokenFrontmatter != nil || wiki.IsScaffoldFile(p.Relative) {
			continue
		}
		// Two states, two repairs: a page that declares no `type` has to gain
		// the key, a page whose `type` is empty has to fill the one it has.
		// `wiki/lint.go` folds both into one message and says in its own
		// comment that this check owes the second one.
		var message string
		switch {
		case p.PageType == nil:
			message = "no type; every OKF page needs one"
		case strings.TrimSpace(*p.PageType) == "":
			message = "type is empty; every OKF page needs a non-empty one"
		default:
			continue
		}
		out = append(out, finding(p, "type-missing", message))
	}
	return out
}

// sourceResourceMissing reports OKF §5.1: `resource` is REQUIRED within a
// `sources` entry. It is what a consumer follows a claim back with, and an
// entry without it names nothing at all.
func sourceResourceMissing(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		for i, s := range p.Sources {
			// The position, not just the page: a page may carry many entries
			// and only one of them be broken.
			if strings.TrimSpace(s.Resource) == "" {
				out = append(out, finding(p, "source-resource-missing",
					fmt.Sprintf("sources[%d] has no resource", i)))
			}
		}
	}
	return out
}

// generatedByMissing reports OKF §5.2: `generated.by` is REQUIRED within
// `generated`. §5.2 keeps `generated` apart from `verified` precisely so that
// who wrote a page stays readable, and a block naming no actor records
// nothing.
func generatedByMissing(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if p.Generated != nil && strings.TrimSpace(p.Generated.By) == "" {
			out = append(out, finding(p, "generated-by-missing",
				"generated is present but names no `by`"))
		}
	}
	return out
}

// reservedNameAsConcept reports OKF §3.1: `index.md` and `log.md` "MUST NOT
// be used for concept documents", and "all other .md files are concept
// documents" -- so carrying a `type` is what makes a file one.
//
// Only these two names, not the wider scaffold set: §3.1 reserves exactly
// them. The three names this repository adds are its own, and a foreign
// reader is free to read those as concepts.
func reservedNameAsConcept(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		base := path.Base(p.Relative)
		if base != "index.md" && base != "log.md" {
			continue
		}
		if p.PageType == nil {
			continue
		}
		out = append(out, finding(p, "reserved-name-as-concept",
			fmt.Sprintf("%s is a reserved filename and must not carry a concept type",
				base)))
	}
	return out
}

// catalogMalformed reports OKF §11.3 against §8: a reserved filename has to
// follow its structure when present, and §8 makes an `index.md` the listing
// of its directory's contents. This is the blind spot that let a generated
// catalog overwrite a bundle catalog in three areas for weeks.
//
// It fires on incomplete-when-present and never on absent: §8 makes the file
// optional, and §11 forbids a consumer to reject a bundle over "missing
// index.md files".
//
// A catalog answers for its own directory only. The tree below it has
// catalogs of its own, and the example in §8 lists a subdirectory as a single
// entry rather than unrolling it.
func catalogMalformed(pages []wiki.WikiPage) []check.Finding {
	catalogs := map[string]wiki.WikiPage{}
	listed := map[string]map[string]bool{}
	for _, p := range pages {
		if path.Base(p.Relative) != "index.md" {
			continue
		}
		dir := path.Dir(p.Relative)
		catalogs[dir] = p
		targets := map[string]bool{}
		for _, link := range p.Links {
			targets[resolveLink(dir, link)] = true
		}
		listed[dir] = targets
	}

	var out []check.Finding
	// Over the pages in read order, not over the maps: two runs of an
	// unchanged bundle have to render the same output, and a map hands its
	// keys out in a different order every time.
	for _, p := range pages {
		if wiki.IsScaffoldFile(p.Relative) {
			continue
		}
		dir := path.Dir(p.Relative)
		catalog, hasCatalog := catalogs[dir]
		if !hasCatalog || listed[dir][p.Relative] {
			continue
		}
		out = append(out, finding(catalog, "catalog-malformed",
			fmt.Sprintf("catalog does not list %s", path.Base(p.Relative))))
	}
	return out
}

// resolveLink turns one catalog entry into a bundle-relative path.
//
// A leading slash is bundle-relative, not filesystem-absolute. OKF §6.1 calls
// it "the recommended form because it is stable when documents are moved
// within their subdirectory" -- the closing clause matters, since it is a move
// inside the bundle the form survives, not any move at all. The design settles
// this repository's old disagreement about such targets in favour of the
// bundle root.
//
// A query is cut off, because the reader does not carry one into the edge:
// `mdLinkRe` (`internal/brain/wiki/parse.go:44`) hands `b.md?x=1` on as that whole
// string, and a resolution keeping the `?` answers `topics/b.md?x=1` for an
// entry the reader means as `b.md`. Measured before it was repaired: such an
// entry drew `catalog-malformed` at the grade error against a catalog that
// does list the page. The fragment needs no such step -- the same pattern
// consumes it in a group outside its capture, so `a.md#top` never arrives
// here as anything but `a.md`.
//
// Cut before the escapes are undone, in Python's order
// (`src/brain/wiki/lint.py:265-268` splits and only then unquotes). The other
// way round, an encoded `?` would be read as the separator and `b%3F.md`
// would lose its name after the first character.
//
// Percent escapes are undone for the reason `wiki/lint.py` gives at
// `_resolve` for the same step: the link pattern refuses a raw space, so a
// page whose name carries one can only be linked encoded, and a catalog that
// does list such a page would otherwise be reported for omitting it. A
// malformed escape is left standing rather than dropped -- it then matches no
// page, which is the direction a rule of this severity should fail in.
//
// The absolute form is normalised *before* the slash comes off, as
// `house/resolveTarget` (`internal/brain/check/house/bundle.go:147-151`) already does
// it: `path.Clean` answers `/x` for `/../x`, so the entry cannot climb out
// of the bundle. Stripping first left the string `../escape.md`, which no
// page matches and no rule here acts on -- this file imports no `os` -- but
// it is an escape ready-made for the first caller that puts the answer on a
// filesystem.
//
// That no page matches is a fact about the *callers*, not about the model.
// `WikiPage.Relative` is whatever the reader put there, and `ReadPage` falls
// back to the raw path when `filepath.Rel` fails (`internal/brain/wiki/parse.go:54-57`),
// so a hand-built page may carry `../../escape.md` and be matched by exactly
// such a string. What rules it out today is that every reader walks the
// bundle root and hands out paths below it (the walk in `wiki.LintBundle`, and the
// walk each test package keeps). Whoever feeds this axis from somewhere else
// owes that guarantee.
func resolveLink(dir, target string) string {
	if query := strings.IndexByte(target, '?'); query >= 0 {
		target = target[:query]
	}
	if decoded, err := url.PathUnescape(target); err == nil {
		target = decoded
	}
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(path.Clean(target), "/")
	}
	return path.Clean(path.Join(dir, target))
}

// logDateForm reports OKF §9: "Date headings MUST use ISO 8601 `YYYY-MM-DD`
// form." Ordering the entries is the one thing a consumer can do with a log
// mechanically, and a heading in any other form takes that away.
//
// §9 puts the date at the second level, below one title heading, so `## ` is
// what a date heading looks like and every `## ` heading is judged as one.
func logDateForm(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if path.Base(p.Relative) != "log.md" {
			continue
		}
		for _, line := range strings.Split(p.RawBody, "\n") {
			heading, isHeading := strings.CutPrefix(line, "## ")
			if !isHeading {
				continue
			}
			// TrimSpace and not TrimRight(line, "\r") before the prefix: a log
			// checked out with CRLF leaves a `\r` on every heading, and trimming
			// it here covers that as well as trailing blanks. Doing both
			// would be one guard the other already makes unobservable.
			heading = strings.TrimSpace(heading)
			if isoDay().MatchString(heading) {
				continue
			}
			out = append(out, finding(p, "log-date-form",
				fmt.Sprintf("log heading %q is not an ISO YYYY-MM-DD date", heading)))
		}
	}
	return out
}

// computationRuntimeMissing reports OKF §10.2: `runtime` is "REQUIRED for
// this type", "the single field that says how to run the computation, and so
// how the executor and attester interpret it".
//
// The type is matched case-insensitively. §4.1 registers no type values
// centrally and leaves their spelling to the producer, so `Attested
// Computation` and `attested computation` are one type named twice.
func computationRuntimeMissing(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if p.PageType == nil ||
			!strings.EqualFold(strings.TrimSpace(*p.PageType), "attested computation") {
			continue
		}
		if strings.TrimSpace(p.Runtime) != "" {
			continue
		}
		out = append(out, finding(p, "computation-runtime-missing",
			"an attested computation needs a runtime; without it nothing can run it"))
	}
	return out
}

// indexFrontmatterMisplaced reports OKF §12: `okf_version` in a bundle-root
// `index.md` is "the only place frontmatter is permitted in an `index.md`",
// and §8 says the same from the other side.
//
// This rule and `reservedNameAsConcept` both judge an `index.md`, and neither
// stands in for the other: a non-root catalog carrying only `okf_version` is
// misplaced without being a concept document, and a typed `log.md` is a
// concept document this rule never sees. A page that breaks both is reported
// twice, because it has to be repaired twice.
func indexFrontmatterMisplaced(pages []wiki.WikiPage) []check.Finding {
	var out []check.Finding
	for _, p := range pages {
		if path.Base(p.Relative) != "index.md" || !p.HasFrontmatter {
			continue
		}
		// `Relative` is the path from the bundle root, so the root catalog is
		// the one whose relative path is nothing but the name. The reader has
		// already answered that question against the root it was handed.
		if p.Relative != "index.md" {
			out = append(out, finding(p, "index-frontmatter-misplaced",
				"only the bundle-root index.md may carry frontmatter"))
			continue
		}
		for _, key := range p.FrontmatterKeys {
			if key == "okf_version" {
				continue
			}
			out = append(out, finding(p, "index-frontmatter-misplaced",
				fmt.Sprintf("a bundle-root index.md may carry only okf_version, not %q",
					key)))
		}
	}
	return out
}
