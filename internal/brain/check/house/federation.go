package house

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
)

// Federation runs the two house rules over every area at once, which is
// what the sweep needs and what design §7 calls the third width.
//
// Neither rule needs every bundle, and the claim that they do was
// measured false: `wrongDirection` reads `bundles[area.Scope]` and the
// registration, `unlistedArea` reads `bundles[signpost.Scope]`, the
// registration and the signpost's manifest. What both need is the
// *registration* -- the shared set and the list of areas that must be
// named -- and that is why they cannot run inside one area's rule set
// but can run beside it. `FederationFor` is that second door.
//
// No `Context`, unlike `Page` and `Bundle`. That struct is the *bundle*
// context -- one root, one threshold, one clock -- and this function
// spans every bundle at once; a single one of those values would be the
// wrong one for all but one area. Everything these two rules read comes
// from the registration or from the pages themselves.
//
// The map is keyed by scope, and the areas decide which keys are asked.
// Iterating the map instead would let its order pick the order of the
// findings, and the comment on `check.Finding.Name` says why two runs over an
// unchanged bundle have to render the same output. A scope in the map
// with no area in the registration is therefore not judged at all.
//
// A missing entry means the caller did not read that bundle, and neither
// rule can tell that from a bundle with no pages. They fall in opposite
// directions on it, deliberately: `wrong-direction` finds no page to
// judge and stays silent, `unlisted-area` finds no catalog and names
// every area -- which is Python's answer to an absent `index.md`
// (`src/brain/wiki/lint.py:475-476`, an empty tuple, and `any()` over
// nothing is False). The one errs towards silence because a page it
// never saw cannot be accused; the other errs towards speaking because a
// signpost that names nobody is exactly the defect it exists for.
//
// The order is Python's `RULES` (`src/brain/wiki/lint.py:543,546`):
// `unlisted_area` before `wrong_direction`. It is not the output order --
// `check.Sort` is the job of whoever merges the axes.
func Federation(
	bundles map[string][]wiki.WikiPage, areas []config.Area, lookup config.ArtifactLookup,
) []check.Finding {
	return FederationFor(areas, bundles, areas, lookup)
}

// FederationFor runs the same two rules over the areas named in
// `subjects`, and it is the door the bundle width goes through: one area
// is the subject, the whole registration still decides.
//
// The split is Python's, not an invention here. `_lint` runs
// `wrong_direction` for each linted area with the *full* registration's
// shared set, and hands `expected_targets` in only `if area.signpost`
// (`src/brain/cli.py:1524-1556`) -- so the subject of both rules is the
// area under check, while the set they judge it against is every
// registered area.
//
// Filtering by what the map happens to hold would be the tempting
// shortcut and is the wrong one. `unlistedArea` errs towards speaking on
// a bundle it was not given (see the note above), so a bundle run of a
// non-signpost area would ask a foreign signpost about a catalog nobody
// read and report the whole federation as unlisted. The subjects say who
// is asked; the map says what was read; the two are not the same
// question.
func FederationFor(
	subjects []config.Area, bundles map[string][]wiki.WikiPage,
	areas []config.Area, lookup config.ArtifactLookup,
) []check.Finding {
	var out []check.Finding
	out = append(out, unlistedArea(subjects, bundles, areas, lookup)...)
	out = append(out, wrongDirection(subjects, bundles, areas)...)
	return out
}

// federationFinding is the third door beside `finding` and `warning`,
// and it exists for the one field those two leave empty: `Scope`.
// `check.Finding.Scope` is "the area the page belongs to"
// (the comment on `check.Finding.Scope`), and `Page` and `Bundle` are handed one
// bundle without its name, so their caller has to fill it. This is the
// first function that already knows the scope of every page it judges --
// and the first that has to, because one run of it speaks about several
// areas at once, and two signposts would otherwise both report on a page
// called `index.md` with nothing to tell them apart.
//
// Both rules are errors. The design's house table puts them there
// (§5.2), and Python grades both the same way
// (`src/brain/wiki/lint.py:419,526`).
func federationFinding(
	scope, relative, rule, message string,
) check.Finding {
	return check.Finding{
		Scope:    scope,
		Relative: relative,
		Axis:     check.AxisHouse,
		Rule:     rule,
		Severity: check.Error,
		Message:  message,
	}
}

// citedScope answers with the scope a `brain://` resource names when no
// shared area owns it, which is precisely the case `wrong-direction`
// reports. A resource that names nothing, and one a shared area owns,
// come back with false.
//
// Only the *shared* scopes are consulted, because that is the set Python
// hands in (`src/brain/cli.py:1552` passes `shared_scopes`). The
// consequence is on the message, not on the verdict: a reference no
// shared scope owns comes back uncut, path and all, and
// `src/brain/wiki/lint.py:394-396` gives the reason -- "Guessing where
// the scope ends and the path begins would put a name in the message
// that nobody can look up; the whole reference is what the reader must
// see."
//
// Python takes the *longest* matching scope (`max(matches, key=len)`).
// That ranking is unobservable and is not rebuilt here: every match
// comes from `known`, which is the shared set, so any match at all --
// longest or shortest -- lands in the shared scopes and silences the
// rule. Only "some scope matched" decides anything, and that is what
// this loop asks. Rebuilding the ranking would be a branch no fixture
// could hold.
//
// The scope is cut from `netloc` plus `path` and not from the netloc
// alone, for the reason Python states at `:396-398`: scopes carry
// slashes, so `brain://project/ultra-brain/...` splits into host
// `project` and path `/ultra-brain/...`. Netloc plus path is everything
// after the scheme, and the `//` that introduces the netloc needs no
// step of its own: the trim below takes every leading slash anyway, so
// stripping it first would be a statement no fixture could hold.
//
// `url.Parse` is deliberately not used, and both reasons were measured
// against `urlsplit` rather than assumed:
//
//   - It refuses a malformed percent escape. `urlsplit` never refuses,
//     so `brain://project/space/%zz` names a scope on the Python side
//     and is reported there. Giving up on it would let a guard of this
//     severity fail silent and make the two sides disagree over one
//     page.
//   - It decodes the path. `urlsplit` does not, and `_cited_scope`
//     never unquotes -- only `_catalog_targets` does. Measured,
//     `brain://project/x/a%20b` came back as `project/x/a b`, a string
//     the reader cannot find in the page the finding names.
//
// The scheme is matched case-insensitively because `urlsplit` folds it,
// so `BRAIN://` reaches this rule on both sides. Query and fragment are
// cut the way `urlsplit` cuts them -- the fragment off the whole
// reference first, then the query -- although no resource in this
// federation carries either.
func citedScope(resource string, shared map[string]bool) (string, bool) {
	const scheme = "brain:"
	if len(resource) < len(scheme) ||
		!strings.EqualFold(resource[:len(scheme)], scheme) {
		return "", false
	}
	rest := resource[len(scheme):]
	if cut := strings.IndexByte(rest, '#'); cut >= 0 {
		rest = rest[:cut]
	}
	if cut := strings.IndexByte(rest, '?'); cut >= 0 {
		rest = rest[:cut]
	}
	// Trimmed at both ends, as Python's `.strip("/")` does: `brain://`
	// alone leaves nothing, and the empty string is no scope to accuse a
	// page of citing.
	joined := strings.Trim(rest, "/")
	if joined == "" {
		return "", false
	}
	for scope := range shared {
		if joined == scope || strings.HasPrefix(joined, scope+"/") {
			return "", false
		}
	}
	return joined, true
}

// wrongDirection reports Verbund §6.2 against architecture §5.7.5:
// "Projekte lesen geteilte Bereiche, geteilte Bereiche lesen keine
// Projekte. Sonst wandert Projektinternes durch die Hintertür in die
// Schicht, die am ehesten einem Team oder der Öffentlichkeit gezeigt
// wird." The rule was prose until Python enforced it; this is the same
// rule on the new axis.
//
// Only a `shared` area is a subject. §6.2 says so -- "in einem Bereich
// mit `shared = true` ist jeder `sources`-Eintrag ... ein Fehler" -- and
// a project citing a project is the ordinary direction.
//
// Sources only, never links. §6.2 makes that the whole reason the
// signpost is not itself a violation: "Er trägt Links, keine
// `sources`-Einträge -- Namen, keine Aussagen. Nichts Projektinternes
// wandert dadurch nach oben."
//
// Scaffold files are no subjects, the line `judgedInBundle` draws for
// the six bundle rules. Python reaches the same by dropping the scaffold
// names before any rule sees one (`src/brain/wiki/lint.py:582-586`).
//
// The subjects and the registration are two arguments and not one: the
// shared set is built from every registered area, because a page of the
// area under check may cite any of them, while only the subjects are
// judged.
func wrongDirection(
	subjects []config.Area, bundles map[string][]wiki.WikiPage,
	areas []config.Area,
) []check.Finding {
	shared := map[string]bool{}
	for _, area := range areas {
		if area.Shared {
			shared[area.Scope] = true
		}
	}

	var out []check.Finding
	for _, area := range subjects {
		if !area.Shared {
			continue
		}
		for _, p := range bundles[area.Scope] {
			if !judgedInBundle(p) {
				continue
			}
			for _, s := range p.Sources {
				cited, wrong := citedScope(s.Resource, shared)
				if !wrong {
					continue
				}
				// Python's message word for word
				// (`src/brain/wiki/lint.py:421-422`), except for the
				// quotes: `%q` writes the double quotes `page.go:224`
				// already writes where Python's `!r` writes single
				// ones.
				out = append(out, federationFinding(area.Scope,
					p.Relative, "wrong-direction", fmt.Sprintf(
						"a shared area must not cite %q; promote the "+
							"page instead (architecture 5.7.4)", cited)))
			}
		}
	}
	return out
}

// unlistedArea reports Verbund §6.1: the signpost names every area of
// the federation, or the run fails. "Damit meldet sich ein Projekt nicht
// nur an, es muss sich anmelden. Ein neues Bundle, das im Wegweiser
// fehlt, lässt den Lauf fehlschlagen; wer den Bereich anlegt, sieht den
// Befund im selben Lauf."
//
// Every signpost is asked, not the first one. §5.1 says there is exactly
// one -- "Zwei `signpost`-Bereiche sind ein Fehler in der Registrierung:
// zwei Startpunkte sind keiner, und die Frage, welcher der maßgebliche
// ist, hätte keine Antwort" -- and that invariant belongs to the reader
// of the registry. `config.ReadRegistry` enforces it, as `read_registry`
// does, but this function takes its areas from the caller and not only
// from that reader. Answering the question the spec says has no answer,
// by taking whichever entry comes first, would leave a second signpost's
// catalog silently unchecked. Asking both reports the same missing area
// twice, once per scope, which is a nuisance and not a wrong answer.
//
// No signpost at all is legal and silent, also §5.1: "Kein
// `signpost`-Bereich ist zulässig -- dann gibt es keinen Wegweiser und
// `unlisted-area` prüft nichts."
//
// A signpost without a wiki path is not asked either. Every catalog
// target is resolved against the wiki root, so without one there is
// nothing to resolve against -- and Python never reaches the rule in
// that state, because `_lint_targets` lints only areas whose `wiki_path`
// is set (`src/brain/cli.py:1489`).
//
// Only a subject is asked, and every registered area is expected of it:
// a signpost checked alone must still name the areas whose bundles this
// run never read, because what it is missing stands in the registration
// and not in their pages.
func unlistedArea(
	subjects []config.Area, bundles map[string][]wiki.WikiPage,
	areas []config.Area, lookup config.ArtifactLookup,
) []check.Finding {
	var out []check.Finding
	for _, area := range subjects {
		if !area.Signpost || area.WikiPath == "" {
			continue
		}
		hub, mayRun := hubFolder(area, lookup)
		if !mayRun {
			continue
		}
		out = append(out, missingFromSignpost(
			bundles[area.Scope], area, areas, hub)...)
	}
	return out
}

// missingFromSignpost is the rule for one signpost.
//
// The finding sits on `index.md` whatever the catalog is called and
// whether or not it is there, as Python's does
// (`src/brain/wiki/lint.py:524`): the repair is a line in that file, and
// naming the page that lacks it is what the path is for.
func missingFromSignpost(
	pages []wiki.WikiPage, signpost config.Area, areas []config.Area,
	hub string,
) []check.Finding {
	linked := catalogTargets(pages, signpost.WikiPath)

	var out []check.Finding
	for _, area := range areas {
		// The signpost is not expected: §6.1, "er ist der Ort, an dem
		// man bereits steht". An area with no wiki is not expected
		// either -- `src/brain/cli.py:1516-1517` skips it because there is
		// nothing to link to yet.
		if area.Scope == signpost.Scope || area.WikiPath == "" {
			continue
		}
		targets := expectedTargets(signpost, area, hub)
		if namesAny(linked, targets) {
			continue
		}
		out = append(out, federationFinding(signpost.Scope, "index.md",
			"unlisted-area", fmt.Sprintf(
				"the signpost does not link to area %q; expected one "+
					"of: %s", area.Scope, strings.Join(targets, ", "))))
	}
	return out
}

// hubFolder is the vault-relative folder the signpost keeps its hub
// pages in, and a second answer: whether the rule may run at all.
//
// Three cases, not two. A manifest declaring no `[layout] hub` leaves
// the folder empty and the rule runs -- `src/brain/manifest.py:110-112`
// answers None for an unsaid key the same way, and an area may keep no
// hub pages at all. A manifest that is not there is the same case,
// which is why `config.IsUndeclared` is let through. But a declaration
// this package cannot use stops the rule, because the alternative fails
// in the wrong direction: without the hub folder every area whose wiki
// lies outside the vault loses its second name and is reported, so the
// run goes red at the areas that did nothing wrong instead of at the
// one line that did. `hub_layout`'s own docstring
// (`src/brain/manifest.py:97-108`) refuses exactly that, and Python
// answers it by refusing the value outright. Measured against the real
// registration in ultra-brain before the repair: a broken `.brain.toml` at the vault
// root reported `project/ultra-brain`, an area the signpost names
// correctly.
//
// Two ways of being unusable, one answer. The file may fail to parse,
// and it may parse and state a hub folder that leaves the vault --
// `../x` builds a pointer no catalog can name, which is the same wrong
// accusation through a second door. `config.Manifest.HubLayout` owns
// the second question, because it is `hub_layout`'s counterpart and
// because the vocabulary of a legal `[layout]` value belongs to the
// reader of that file rather than to a rule of this axis.
//
// Silence is not a verdict here. A signpost whose declaration is broken
// is not judged at all, and the error itself is owed to the reader by
// whoever reads that manifest for its other values -- this rule has no
// channel to carry it.
//
// The manifest is read where `manifest_path` looks for it
// (`src/brain/registry.py:129`): at the area's path, or for a read-only
// area out of the state directory -- `config.ManifestDir`, the same
// answer `run.areaManifest` and the sweep take for every other value of
// that file.
func hubFolder(signpost config.Area, lookup config.ArtifactLookup) (string, bool) {
	manifest, err := config.ReadAreaDeclaration(
		config.ManifestDir(signpost, lookup.Primary))
	if config.IsUndeclared(err) {
		return "", true
	}
	if err != nil {
		return "", false
	}
	hub, err := manifest.HubLayout()
	if err != nil {
		return "", false
	}
	return hub, true
}

// expectedTargets is what the signpost may link for one area, and any
// one of them is enough. Two, because an area may be named in two ways:
// by its wiki, or -- when that wiki lies outside the vault, where no
// relative link from an Obsidian page reaches it -- by its hub page.
//
// The hub pointer is `<signpost.Path>/<layout.hub>/<last scope
// segment>.md`, built from the area *path* and not from the wiki path
// (`src/brain/cli.py:1520`). The two roots differ in the real
// registration -- `knowledge` is the vault, its wiki is `90 Wiki` inside
// it -- and the hub pages sit beside that wiki, not under it. The last
// segment is the name the hub page carries in the vault, which is why
// `project/ultra-brain` is looked for as `ultra-brain.md`.
//
// Both are cleaned into OS form. The registration spells its paths with
// forward slashes even on Windows (`config.Area` paths, as `internal/config/registry.go:18-22` says: the
// reader hands them on exactly as the file wrote them), while a catalog
// target arrives bundle-relative and POSIX. `filepath.Clean` converts
// the separators of both, and that is the one form the comparison
// happens in.
//
// Absolute is assumed and not enforced. A catalog target is always
// absolute -- it is joined onto the signpost's wiki root -- so a `wiki`
// entry spelt relative in the registration would stay relative here,
// match nothing, and have its area reported. Neither `registry.py` nor
// `internal/config/registry.go` forbids a relative path; all nine entries of
// the real registration are absolute, and whoever makes one relative
// owes this line.
func expectedTargets(signpost, area config.Area, hub string) []string {
	targets := []string{filepath.Clean(area.WikiPath)}
	if hub == "" {
		return targets
	}
	last := area.Scope[strings.LastIndex(area.Scope, "/")+1:]
	return append(targets, filepath.Clean(filepath.Join(
		signpost.Path, filepath.FromSlash(hub), last+".md")))
}

// catalogTargets is every link of the signpost's catalog, resolved
// against the wiki root into an absolute OS path.
//
// Only the bundle-root `index.md`, deliberately narrower than `orphan`
// in the file next door, which counts every catalog at any depth as a
// link source. Python reads `root / "index.md"`
// (`src/brain/wiki/lint.py:474`), and the reason holds on its own: a
// catalog in `topics/` is the listing of that folder, not the entrance
// to the federation, and Verbund §4 puts the federation's section into
// one named file that a person reads.
//
// A target with a scheme or with no path is skipped, as Python skips it
// (`:492-493`). `brain://` is not clickable in Obsidian, which is why
// Verbund §4 has the signpost use relative paths in the first place.
//
// The percent escapes are undone, and that is not cosmetic: Verbund §4
// requires a blank in a target to be encoded, because the link pattern
// refuses a raw one "auch nicht in Winkelklammern". A rule comparing the
// encoded text would report every area of this vault. `url.Parse` cuts
// the query before decoding, so an encoded `?` stays part of the name.
//
// `_resolve` of the bundle rules is no help here, and neither is
// `resolveTarget` beside it: both drop a target that climbs out of the
// bundle, and the signpost points nowhere else -- every one of its
// targets leaves the wiki. Python says the same at `:482-483`.
//
// Resolved lexically, never against the disk. Python calls `.resolve()`,
// which follows symlinks and consults the filesystem; a check that
// answered differently depending on what exists would make one bundle
// judge differently on two machines. One measured consequence stays
// open: a target beginning with `/` binds the wiki root away in pathlib
// and is joined onto it here. The real signpost carries no such target.
func catalogTargets(pages []wiki.WikiPage, root string) []string {
	var out []string
	for _, p := range pages {
		if p.Relative != "index.md" {
			continue
		}
		for _, link := range p.Links {
			target := link
			if u, err := url.Parse(link); err == nil {
				if u.Scheme != "" {
					continue
				}
				target = u.Path
			}
			if target == "" {
				continue
			}
			out = append(out, filepath.Clean(filepath.Join(
				root, filepath.FromSlash(target))))
		}
	}
	return out
}

// namesAny asks whether any catalog link names any of an area's targets.
//
// One link names one target by `wiki.LinkNames`, the sweep's own answer:
// a hub target by equality, a wiki by any path below it, taken by
// component so `python-old` never names `python`, and without regard to
// case, as `WindowsPath` compares. A second spelling here once compared
// exactly and reported a folder the registry spelt in another case.
func namesAny(linked, targets []string) bool {
	for _, link := range linked {
		for _, target := range targets {
			if wiki.LinkNames(link, target) {
				return true
			}
		}
	}
	return false
}
