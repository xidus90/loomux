package house

import (
	"testing"
	"time"
	// The embedded zone database, for the one test that needs a zone with
	// a daylight-saving boundary. This host answers `Europe/Berlin`
	// without it -- measured -- because the Go installation carries
	// `lib/time/zoneinfo.zip`; a stripped toolchain or a container image
	// carries neither that nor a system database, and the test would then
	// fail for the environment rather than for the rule.
	_ "time/tzdata"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
)

// page is one whole concept page of the given type. Its single source
// names a `brain://` resource on purpose: `dead-link` follows a
// `resource` like a link (Scheibe 3 §4, line 180 of
// `2026-08-23-scheibe-3-wiki-schicht-design.md`), so a fixture citing a
// bare path would put an edge into every graph test that means to build
// none.
func page(pageType string) string {
	return "---\ntype: " + pageType + "\nsources:\n  - id: a\n" +
		"    resource: brain://knowledge/a\n    doc_id: d\n" +
		"    content_hash: h\n    revision: 1\n---\nbody\n"
}

// writeBundle lays a whole bundle down in one call. The map is written in
// whatever order it iterates, which no rule can see: the pages reach the
// rules through `read`, which walks the tree in the order the filesystem
// hands it out.
func writeBundle(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		write(t, dir, name, body)
	}
}

// bundleCtx is the context the graph tests share, and it is deliberately
// not `ctx`: that helper leaves `UntouchedDays` at zero whenever the
// bundle carries no manifest, and a zero threshold puts the cutoff at
// `Now` itself -- every file a test had just written would then be
// untouched, and the age rules would fire in tests that never mentioned
// them. 180 days is what a manifest saying nothing gets
// (`config.DefaultUntouchedDays`), and `day` lies before every file this
// suite writes, so a fresh fixture stays silent no matter when it runs.
func bundleCtx(dir string) Context {
	return Context{Root: dir, Now: day, UntouchedDays: 180}
}

// ref hands out a pointer to a literal, for the frontmatter fields the
// model keeps as pointers to tell an absent value from an empty one.
func ref(s string) *string {
	return &s
}

// rulesOf is `messagesOf`'s sibling for the tests that ask which pages a
// rule named rather than what it said about them.
// catalog is the bundle's root index.md as the rules see it: a link
// source for `orphan`, and the subject of no rule at all. Fixtures built
// from `wiki.WikiPage` literals put their pages into the graph with it
// rather than with a self-link -- a self-link keeps a page out of
// `orphan` just as well, but `dead-link` would then look for that page on
// disk and report the fixture, while a scaffold file's own targets are
// never followed.
func catalog(targets ...string) wiki.WikiPage {
	return wiki.WikiPage{Relative: "index.md", Links: targets}
}

func rulesOf(findings []check.Finding, rule string) []string {
	var out []string
	for _, f := range findings {
		if f.Rule == rule {
			out = append(out, f.Relative)
		}
	}
	return out
}

func TestOrphanIsAnErrorNotAWarning(t *testing.T) {
	// The second measured divergence: Go had this as a warning, Python as an
	// error. The same finding must not let one run pass and the other fail.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md":  "# c\n\n* [a](a.md)\n",
		"a.md":      page("Topic"),
		"lonely.md": page("Topic"),
	})
	found := Bundle(read(t, dir), ctx(t, dir))
	if sev(found, "orphan") != check.Error {
		t.Fatalf("orphan severity = %v, want error", sev(found, "orphan"))
	}
}

func TestTheSixBundleRulesCarryTheirDegreeOnTheHouseAxis(t *testing.T) {
	// The owner's table, pinned as a run: three errors, three warnings, all
	// six on the house axis. One input trips every one of them, so no rule
	// can carry the wrong axis or degree and go unmeasured because nothing
	// made it fire.
	//
	// The pages are built by hand rather than written to disk. Two of the
	// six turn on `ModTime`, which a file written by the test carries as the
	// moment the test ran -- and a fixture whose age depends on the day the
	// suite runs decides nothing.
	root := t.TempDir()
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	old := time.Date(2026, 1, 2, 0, 30, 0, 0, time.FixedZone("+02", 7200))
	pages := []wiki.WikiPage{
		{
			Relative:    "a.md",
			ModTime:     old,
			Links:       []string{"gone.md", "../out.md"},
			Realization: ref("planned"),
		},
		{
			Relative:    "b.md",
			ModTime:     now,
			Links:       []string{"a.md"},
			Realization: ref("implemented"),
		},
	}
	want := map[string]check.Severity{
		"orphan":                     check.Error,
		"dead-link":                  check.Error,
		"implemented-without-commit": check.Error,
		"outside-area":               check.Warning,
		"untouched":                  check.Warning,
		"long-planned":               check.Warning,
	}
	got := Bundle(pages, Context{
		Root: root, Now: now, UntouchedDays: 180, IsProject: true,
	})
	seen := map[string]bool{}
	for _, f := range got {
		if f.Axis != check.AxisHouse {
			t.Fatalf("%s is not on the house axis", f.Name())
		}
		if want[f.Rule] != f.Severity {
			t.Fatalf("%s is %s, want %s", f.Name(), f.Severity,
				want[f.Rule])
		}
		seen[f.Rule] = true
	}
	for rule := range want {
		if !seen[rule] {
			t.Fatalf("%s never fired", rule)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("unexpected rules fired: %v", seen)
	}
}

func TestANestedCatalogIsAValidReferenceSource(t *testing.T) {
	// Scheibe 3 §4 puts no depth on it: the trigger is "keine andere Seite
	// und kein `index.md` zeigt hierher" (line 179), and the scaffold
	// paragraph adds "`index.md` wird aber für `orphan` gelesen -- es ist
	// eine gültige Verweisquelle" (lines 187-188).
	//
	// This exact fixture is the contradiction the narrow reading built.
	// `okf/catalog-malformed` demands that a page stand in the catalog of
	// its own directory, and `topics/only-here.md` does; reading only the
	// root catalog made that page faultless on one axis and an error on the
	// other. Python is narrower here (`src/brain/wiki/lint.py:281-282` reads
	// `root / "index.md"` alone, every other scaffold being gone from
	// `pages`), so this is a knowing divergence -- reported, and argued from
	// the source rather than from the brief.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md":            "# c\n\n* [t](topics/)\n",
		"topics/index.md":     "# t\n\n* [only](only-here.md)\n",
		"topics/only-here.md": page("Topic"),
	})
	if got := Bundle(read(t, dir), bundleCtx(dir)); len(got) != 0 {
		t.Fatalf("a page its own catalog lists was reported: %v", got)
	}
}

func TestAnOrdinaryPageRescuesAnotherFromOrphanhood(t *testing.T) {
	// Half of the trigger, and it was unguarded: a mutant that counted only
	// catalogs as link sources survived the whole suite and reported all 24
	// pages of a real wiki bundle. "Keine andere Seite ... zeigt hierher" is the
	// first half of Scheibe 3 §4 line 179, and no catalog names `b.md` here.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"a.md":     page("Topic") + "\n[to b](b.md)\n",
		"b.md":     page("Topic"),
	})
	if got := Bundle(read(t, dir), bundleCtx(dir)); len(got) != 0 {
		t.Fatalf("a page another page links to was reported: %v", got)
	}
}

func TestASourceCitationIsNoInboundEdge(t *testing.T) {
	// `dead-link` follows a `resource` like a link, `orphan` does not:
	// Python walks `page.links` alone at `src/brain/wiki/lint.py:290` while
	// `dead_link` builds the wider tuple at `:294`. A citation says where a
	// page came from; it is not a path a reader can walk to it, and a page
	// whose only mention is somebody else's `sources[]` is unreachable.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"a.md": "---\ntype: Topic\nsources:\n  - id: a\n" +
			"    resource: b.md\n    doc_id: d\n    content_hash: h\n" +
			"    revision: 1\n---\nbody\n",
		"b.md": page("Topic"),
	})
	got := rulesOf(Bundle(read(t, dir), bundleCtx(dir)), "orphan")
	if len(got) != 1 || got[0] != "b.md" {
		t.Fatalf("orphans are %v, want [\"b.md\"]", got)
	}
}

func TestTheCatalogIsALinkSourceAndNotASubject(t *testing.T) {
	// Three exclusions in one fixture, each of them Python's. The catalog
	// links a page, so that page is not orphaned; the catalog is no page, so
	// it is never orphaned itself; and its own targets are not followed --
	// `lint_bundle` hands `dead_link` and `outside_area` only the
	// non-scaffold pages (`src/brain/wiki/lint.py:582-586`), while
	// `okf/catalog-malformed` is the rule that judges a catalog.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n* [gone](gone.md)\n" +
			"* [out](../out.md)\n",
		"a.md": page("Topic"),
	})
	if got := Bundle(read(t, dir), bundleCtx(dir)); len(got) != 0 {
		t.Fatalf("the catalog was judged as a page: %v", got)
	}
}

func TestAnUnreadableFrontmatterIsStillAPageInTheGraph(t *testing.T) {
	// The page rules exempt such a page because each of the five reads a
	// field out of the block that did not decode. None of the six here does
	// so blindly: two read no frontmatter at all -- `orphan` goes by the
	// body's links, `untouched` by the file's age -- and the two that read
	// `sources[]` see an empty list, which makes them say less rather than
	// something false. Python keeps such a page in `pages` for that reason.
	// The two realization rules need no exemption either:
	// `internal/brain/wiki/parse.go:89-95` leaves `Realization` nil on a failed decode,
	// so a guard would be an exclusion nothing could ever make fire.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n",
		"p.md":     "---\ntype: [unclosed\n---\nbody\n",
	})
	got := rulesOf(Bundle(read(t, dir), bundleCtx(dir)), "orphan")
	if len(got) != 1 || got[0] != "p.md" {
		t.Fatalf("orphans are %v, want [\"p.md\"]", got)
	}
}

func TestAgedScaffoldFilesStaySilent(t *testing.T) {
	// The other half of the scaffold line, for the three rules a written
	// fixture cannot age. Scheibe 3 §4 exempts the scaffold files by name,
	// and Python never hands one to any rule -- so a catalog nobody has
	// touched in two years, carrying whatever frontmatter, is not this
	// axis's business.
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	pages := []wiki.WikiPage{{
		Relative:    "log.md",
		ModTime:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Realization: ref("planned"),
	}, {
		Relative:    "audit.md",
		ModTime:     now,
		Realization: ref("implemented"),
	}}
	got := Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180, IsProject: true,
	})
	if len(got) != 0 {
		t.Fatalf("a scaffold file was judged: %v", got)
	}
}

func TestADeadTargetIsNamedRawAndResolved(t *testing.T) {
	// The message carries both forms, and Python says why
	// (`src/brain/wiki/lint.py:323-326`): the raw target is what the reader
	// can search for in the page, and the resolved path alone would send
	// them looking for a string that never occurs in the source text.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md":    "# c\n\n* [a](topics/a.md)\n",
		"topics/a.md": page("Topic") + "\n[gone](gone.md)\n",
	})
	got := messagesOf(Bundle(read(t, dir), bundleCtx(dir)), "dead-link")
	want := "target does not exist: gone.md (resolved: topics/gone.md)"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages are %v, want [%q]", got, want)
	}
}

func TestAnAbsoluteTargetIsBundleRelative(t *testing.T) {
	// Design §6: "Entschieden wird: die Bündelwurzel gewinnt. Ein absoluter
	// Pfad in einer Bundle-Seite ist bündelrelativ, wie OKF ihn definiert."
	// The old `absolute-link` warning goes with it, so neither half of this
	// fixture may reach `outside-area`: the target that hits a page is
	// simply well, and the one that hits nothing is dead like any other.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](topics/a.md)\n",
		"topics/a.md": page("Topic") +
			"\n[hit](/index.md)\n[miss](/nope.md)\n",
	})
	got := Bundle(read(t, dir), bundleCtx(dir))
	if hasRule(got, "outside-area") {
		t.Fatal("an absolute target was reported as leaving the area")
	}
	msgs := messagesOf(got, "dead-link")
	want := "target does not exist: /nope.md (resolved: nope.md)"
	if len(msgs) != 1 || msgs[0] != want {
		t.Fatalf("messages are %v, want [%q]", msgs, want)
	}
}

func TestAClimbingTargetIsOutsideTheAreaAndNotDead(t *testing.T) {
	// Typkatalog §4.1: a reference leaving the area "wird gemeldet statt
	// übergangen -- als Warnung, nicht als Fehler". It cannot also be dead:
	// nothing below the bundle root answers for it, so the check that would
	// call it missing has nowhere to look.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md":    "# c\n\n* [a](topics/a.md)\n",
		"topics/a.md": page("Topic") + "\n[out](../../out.md)\n",
	})
	got := Bundle(read(t, dir), bundleCtx(dir))
	if hasRule(got, "dead-link") {
		t.Fatal("a target outside the bundle was checked for existence")
	}
	msgs := messagesOf(got, "outside-area")
	want := "target leaves the area and cannot be checked: ../../out.md"
	if len(msgs) != 1 || msgs[0] != want {
		t.Fatalf("messages are %v, want [%q]", msgs, want)
	}
}

func TestAClimbThatStaysInsideIsChecked(t *testing.T) {
	// `..` alone does not leave the area, and the rule reporting the climb
	// must not read it as one: a target that lands back inside the bundle is
	// an ordinary link, and `dead-link` owns it.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md":    "# c\n\n* [a](topics/a.md)\n* [r](root.md)\n",
		"root.md":     page("Topic"),
		"topics/a.md": page("Topic") + "\n[up](../root.md)\n",
	})
	if got := Bundle(read(t, dir), bundleCtx(dir)); len(got) != 0 {
		t.Fatalf("a target that stays inside was reported: %v", got)
	}
}

func TestASchemeTargetIsNeitherDeadNorOutside(t *testing.T) {
	// Typkatalog §4.1 draws the line at the form: "`brain://`-Verweise gehen
	// durch, nackte Pfade ins Nichts werden gemeldet." Scheibe 3 §4 answers
	// the same for the other schemes -- external URLs are not checked,
	// because "das wäre Netzverkehr in einem deterministischen Lauf".
	//
	// A scheme also names no page of this bundle, so it cannot rescue one
	// from `orphan`: `b.md` stays orphaned although `a.md` writes a link
	// that ends in its name.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"a.md":     page("Topic") + "\n[k](brain://knowledge/b.md)\n",
		"b.md":     page("Topic"),
	})
	got := Bundle(read(t, dir), bundleCtx(dir))
	if hasRule(got, "dead-link") || hasRule(got, "outside-area") {
		t.Fatalf("a brain:// reference was judged as a path: %v", got)
	}
	if orphans := rulesOf(got, "orphan"); len(orphans) != 1 ||
		orphans[0] != "b.md" {
		t.Fatalf("orphans are %v, want [\"b.md\"]", orphans)
	}
}

func TestAnEmptyResourceIsLeftToTheOKFRule(t *testing.T) {
	// `okf/source-resource-missing` owns an entry without `resource` (OKF
	// §5.1, REQUIRED). Following the empty string as a path would put one
	// defect on two axes -- and it names no file, so there is nothing to
	// look for either.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"a.md": "---\ntype: Topic\nsources:\n  - id: a\n" +
			"    resource: \"\"\n---\nbody\n",
	})
	if got := Bundle(read(t, dir), bundleCtx(dir)); len(got) != 0 {
		t.Fatalf("an empty resource was followed: %v", got)
	}
}

func TestASourcePathIsCheckedLikeALink(t *testing.T) {
	// Scheibe 3 §4 names both: "Geprüft werden wiki-interne Ziele und die
	// `resource`-Pfade aus `sources[]`."
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"a.md": "---\ntype: Topic\nsources:\n  - id: a\n" +
			"    resource: raw/spec.md\n---\nbody\n",
	})
	got := messagesOf(Bundle(read(t, dir), bundleCtx(dir)), "dead-link")
	want := "target does not exist: raw/spec.md (resolved: raw/spec.md)"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages are %v, want [%q]", got, want)
	}
}

func TestAPercentEscapeIsUndoneBeforeTheLookup(t *testing.T) {
	// Python undoes the escapes before joining and gives the reason at
	// `_resolve` (`src/brain/wiki/lint.py:258-261`): the link pattern
	// refuses a raw space, so a page whose name carries one can only be
	// linked encoded -- and without undoing it here that page never matches
	// on disk.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a%20b.md)\n",
		"a b.md":   page("Topic"),
	})
	if got := Bundle(read(t, dir), bundleCtx(dir)); len(got) != 0 {
		t.Fatalf("an encoded target missed the page it names: %v", got)
	}
}

func TestAMalformedEscapeIsReportedRatherThanDropped(t *testing.T) {
	// `%zz` is no escape. Python's `unquote` leaves it standing, so the
	// target is an ordinary relative path that matches no page and the
	// finding is `dead-link`. Dropping it as unreadable would be the one
	// direction a rule of this severity must not fail in -- the argument
	// `internal/brain/check/okf/errors.go:250-252` gives for its own resolver.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"a.md":     page("Topic") + "\n[bad](%zz)\n",
	})
	got := messagesOf(Bundle(read(t, dir), bundleCtx(dir)), "dead-link")
	want := "target does not exist: %zz (resolved: %zz)"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages are %v, want [%q]", got, want)
	}
}

func TestADirectoryTargetCountsAsSomethingThatExists(t *testing.T) {
	// A catalog may link a subdirectory rather than the pages inside it, and
	// a directory is something that exists. Asking for a regular file would
	// report the link to `topics/` as dead.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md":    "# c\n\n* [t](topics/)\n* [a](topics/a.md)\n",
		"topics/a.md": page("Topic"),
	})
	if got := Bundle(read(t, dir), bundleCtx(dir)); len(got) != 0 {
		t.Fatalf("a directory target was reported: %v", got)
	}
}

func TestUntouchedAndLongPlannedBothSpeakAboutOnePage(t *testing.T) {
	// The overlap is deliberate, and `src/brain/wiki/lint.py:216-219` says
	// why: `untouched` speaks to every page's age regardless of
	// `realization`, `long-planned` adds the narrower claim that a specific
	// plan was never picked up. Two questions about one fact, both worth
	// stating -- so neither rule may suppress the other.
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	pages := []wiki.WikiPage{catalog("p.md"), {
		Relative:    "p.md",
		ModTime:     time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Realization: ref("planned"),
	}}
	got := Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180, IsProject: true,
	})
	if !hasRule(got, "untouched") || !hasRule(got, "long-planned") {
		t.Fatalf("one rule swallowed the other: %v", got)
	}
	msgs := messagesOf(got, "untouched")
	want := "unchanged for more than 180 days"
	if len(msgs) != 1 || msgs[0] != want {
		t.Fatalf("untouched says %v, want [%q]", msgs, want)
	}
}

func TestTheRealizationRulesAreSilentOutsideAProjectBundle(t *testing.T) {
	// `realization` is a project-only field (Architektur §9.4: "In
	// Projekt-Bundles kommen zwei Prüfungen dazu"), and Python returns early
	// from both rules when the bundle is not one
	// (`src/brain/wiki/lint.py:196-197,221-222`). A `knowledge` bundle
	// carrying the field anyway is not their business -- while `untouched`,
	// which asks about the file and not about the field, still speaks.
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	pages := []wiki.WikiPage{catalog("p.md", "q.md"), {
		Relative:    "p.md",
		ModTime:     time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Realization: ref("planned"),
	}, {
		Relative:    "q.md",
		ModTime:     now,
		Realization: ref("implemented"),
	}}
	got := Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180,
	})
	if hasRule(got, "long-planned") ||
		hasRule(got, "implemented-without-commit") {
		t.Fatalf("a realization rule spoke outside a project: %v", got)
	}
	if !hasRule(got, "untouched") {
		t.Fatal("untouched went silent with the realization rules")
	}
}

func TestLongPlannedSharesTheUntouchedThreshold(t *testing.T) {
	// One number, not two. `src/brain/wiki/lint.py:213-214`: "The threshold
	// is the one the area already declares for `untouched`: a second number
	// would be a second setting to keep honest." A page younger than that
	// threshold is a plan, not a forgotten one -- and `untouched` has to
	// stay silent on it as well, or the shared cutoff would be two.
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	pages := []wiki.WikiPage{catalog("p.md"), {
		Relative:    "p.md",
		ModTime:     now.AddDate(0, 0, -179),
		Realization: ref("planned"),
	}}
	got := Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180, IsProject: true,
	})
	if len(got) != 0 {
		t.Fatalf("a plan younger than the threshold was reported: %v", got)
	}
}

func TestBothAgeRulesMeasureAgainstTheContextsNow(t *testing.T) {
	// Never `time.Now()`: two runs over one unchanged bundle would then be
	// free to disagree, and no run could be replayed. The zero `Now` is the
	// visible consequence -- the cutoff falls before every real modification
	// time, so both rules go silent on a bundle whose pages are decades too
	// old. That is the caller's to answer for; Python fills the field rather
	// than defaulting it inside the rules (`src/brain/wiki/lint.py:575`).
	pages := []wiki.WikiPage{catalog("p.md"), {
		Relative:    "p.md",
		ModTime:     time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Realization: ref("planned"),
	}}
	got := Bundle(pages, Context{
		Root: t.TempDir(), UntouchedDays: 180, IsProject: true,
	})
	if len(got) != 0 {
		t.Fatalf("a zero Now was answered from the clock: %v", got)
	}
}

func TestLongPlannedNamesTheDayInTheZoneItCarries(t *testing.T) {
	// The message names a day, and a modification time carries the zone that
	// decides which. `2026-01-02T00:30+02:00` is the second of January where
	// the file lives and still the first in UTC, so a message built from the
	// converted instant would name a day nobody would find in a file
	// listing. Python reads the components as written too: `page.mtime` is
	// `datetime.fromtimestamp(...).astimezone()`
	// (`src/brain/wiki/page.py:151`) and `.date()` takes that zone's day
	// (`src/brain/wiki/lint.py:229`).
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	zone := time.FixedZone("+02", 7200)
	pages := []wiki.WikiPage{catalog("p.md"), {
		Relative:    "p.md",
		ModTime:     time.Date(2026, 1, 2, 0, 30, 0, 0, zone),
		Realization: ref("planned"),
	}}
	got := messagesOf(Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180, IsProject: true,
	}), "long-planned")
	want := "planned and untouched since 2026-01-02"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages are %v, want [%q]", got, want)
	}
}

func TestImplementedWithoutCommitAsksForTheCommit(t *testing.T) {
	// Architektur §9.4 names the case: pages "die `implemented` melden, ohne
	// einen Commit in `implemented_in` zu nennen". Three states and two
	// answers: no field and a blank one both name nothing, a filled one
	// names the commit and ends the matter.
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	pages := []wiki.WikiPage{
		catalog("a-none.md", "b-blank.md", "c-named.md"), {
			Relative:    "a-none.md",
			ModTime:     now,
			Realization: ref("implemented"),
		}, {
			Relative:      "b-blank.md",
			ModTime:       now,
			Realization:   ref("implemented"),
			ImplementedIn: ref("  "),
		}, {
			Relative:      "c-named.md",
			ModTime:       now,
			Realization:   ref("implemented"),
			ImplementedIn: ref("a3953ff"),
		}}
	found := Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180, IsProject: true,
	})
	got := rulesOf(found, "implemented-without-commit")
	if len(got) != 2 || got[0] != "a-none.md" || got[1] != "b-blank.md" {
		t.Fatalf("reported %v, want the first two pages", got)
	}
	msgs := messagesOf(found, "implemented-without-commit")
	want := "realization: implemented without implemented_in"
	if msgs[0] != want {
		t.Fatalf("message is %q, want %q", msgs[0], want)
	}
}

func TestOnlyAPlannedPageIsLongPlanned(t *testing.T) {
	// From the mutation round: dropping the `planned` comparison left every
	// test green, and the rule then named every aged page that states any
	// `realization` at all -- `implemented` included, where the plan was
	// picked up and finished. `untouched` is the rule that speaks about age
	// alone, and it still does here.
	now := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)
	pages := []wiki.WikiPage{catalog("p.md"), {
		Relative:      "p.md",
		ModTime:       time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Realization:   ref("implemented"),
		ImplementedIn: ref("a3953ff"),
	}}
	got := Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180, IsProject: true,
	})
	if hasRule(got, "long-planned") {
		t.Fatalf("a finished page was called a standing plan: %v", got)
	}
	if !hasRule(got, "untouched") {
		t.Fatal("the age rule went silent with the realization rule")
	}
}

func TestAnAbsolutePathCannotClimbOutOfTheBundle(t *testing.T) {
	// Also from the mutation round: stripping the leading slash before
	// normalising left `/../escape.md` standing as `../escape.md`, and the
	// existence check then looked one directory above the bundle root --
	// the one thing an absolute target must not be able to do now that it
	// is read bundle-relative. Python normalises first as well: its
	// `posixpath.normpath("/../x")` answers `/x`.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md":    "# c\n\n* [a](topics/a.md)\n",
		"topics/a.md": page("Topic") + "\n[up](/../escape.md)\n",
	})
	got := messagesOf(Bundle(read(t, dir), bundleCtx(dir)), "dead-link")
	want := "target does not exist: /../escape.md (resolved: escape.md)"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages are %v, want [%q]", got, want)
	}
}

func TestTheThresholdIsCountedInDaysNotInWallClockDates(t *testing.T) {
	// `timedelta(days=n)` is exactly n * 24 h, and Python subtracts it from
	// an aware datetime (`src/brain/wiki/lint.py:172`), so the cutoff is an
	// instant rather than a date on a wall clock. The difference is one hour
	// across a daylight-saving boundary, and only a real zone shows it: 180
	// days before 2026-09-02T12:00+02:00 is 11:00 CET, while the same date
	// arithmetic lands on 12:00. This page sits between the two.
	//
	// The zone database comes from the blank import above rather than from
	// the host, so the answer does not depend on what is installed.
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, berlin)
	pages := []wiki.WikiPage{catalog("p.md"), {
		Relative:    "p.md",
		ModTime:     time.Date(2026, 3, 6, 11, 30, 0, 0, berlin),
		Realization: ref("planned"),
	}}
	got := Bundle(pages, Context{
		Root: t.TempDir(), Now: now, UntouchedDays: 180, IsProject: true,
	})
	if len(got) != 0 {
		t.Fatalf("a page younger than the cutoff was reported: %v", got)
	}
}

func TestALinkInTheLogDoesNotRescueAPage(t *testing.T) {
	// The scaffold line cuts both ways, and only one file crosses it.
	// Scheibe 3 §4 exempts five files by name and then makes a single
	// exception -- "`index.md` wird aber für `orphan` gelesen -- es ist
	// eine gültige Verweisquelle" (lines 187-188). It names no other, so a
	// mention in `log.md`, `audit.md` or `_schema.md` is not a way to reach
	// a page. From the fix round: widening the catalog exception to every
	// index.md left the whole-guard mutation with nothing to catch it.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"log.md":   "# log\n\n## 2026-09-02\n\nTouched [b](b.md).\n",
		"a.md":     page("Topic"),
		"b.md":     page("Topic"),
	})
	got := rulesOf(Bundle(read(t, dir), bundleCtx(dir)), "orphan")
	if len(got) != 1 || got[0] != "b.md" {
		t.Fatalf("orphans are %v, want [\"b.md\"]", got)
	}
}

func TestTheParentOfTheRootIsOutsideTheArea(t *testing.T) {
	// `..` from a page at the bundle root cleans to `..` itself, not to
	// something below `../`, and is outside all the same. Found by the
	// mutation round of 2026-09-23: nothing held the bare form.
	dir := t.TempDir()
	writeBundle(t, dir, map[string]string{
		"index.md": "# c\n\n* [a](a.md)\n",
		"a.md":     page("Topic") + "\n[up](..)\n",
	})
	msgs := messagesOf(Bundle(read(t, dir), bundleCtx(dir)), "outside-area")
	if want := "target leaves the area and cannot be checked: .."; len(msgs) != 1 || msgs[0] != want {
		t.Fatalf("messages are %v, want [%q]", msgs, want)
	}
}
