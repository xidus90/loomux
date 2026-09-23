package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/check"
)

// full is a page that breaks none of the seven soft rules. Fixtures that
// want one rule to fire build from it, so that no assertion rests on a page
// which happens to trip a second rule as well.
const full = "---\ntype: Topic\ntitle: T\ndescription: d\n" +
	"generated:\n  by: human:x\n---\nbody\n"

func rulesOn(findings []check.Finding, relative string) []string {
	var out []string
	for _, f := range findings {
		if f.Relative == relative {
			out = append(out, f.Rule)
		}
	}
	return out
}

func messagesOf(findings []check.Finding, rule string) []string {
	var out []string
	for _, f := range findings {
		if f.Rule == rule {
			out = append(out, f.Message)
		}
	}
	return out
}

func TestTheFourShouldsWarnAndTheThreeMaysNote(t *testing.T) {
	// The owner's principle, pinned as a table: what OKF calls Recommended
	// or SHOULD is a warning, what it only permits is a note. One fixture
	// trips all seven, so no rule can carry the wrong degree and go
	// unmeasured because nothing made it fire.
	want := map[string]check.Severity{
		"title-missing":                   check.Warning,
		"description-missing":             check.Warning,
		"source-id-missing":               check.Warning,
		"index-entry-without-description": check.Warning,
		"no-index":                        check.Note,
		"no-log":                          check.Note,
		"no-trust-family":                 check.Note,
	}
	dir := t.TempDir()
	write(t, dir, "index.md", "# c\n\n* [a](a.md)\n* [sub](sub/) - s\n")
	write(t, dir, "a.md",
		"---\ntype: Topic\nsources:\n  - resource: r\n---\nbody\n")
	write(t, dir, "sub/x.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	seen := map[string]bool{}
	for _, f := range got {
		degree, known := want[f.Rule]
		if !known {
			t.Fatalf("unexpected rule %s on %s", f.Name(), f.Relative)
		}
		if f.Severity != degree {
			t.Fatalf("%s is %s, want %s", f.Name(), f.Severity, degree)
		}
		if f.Axis != check.AxisOKF {
			t.Fatalf("%s is not on the okf axis", f.Name())
		}
		seen[f.Rule] = true
	}
	if len(seen) != len(want) {
		t.Fatalf("only %d of %d rules fired: %v", len(seen), len(want), seen)
	}
}

func TestSoftNeverReportsAnError(t *testing.T) {
	// The twin of TestAMissingCatalogIsNotAnError on this side: a bundle
	// that merely misses what OKF recommends stays consumable, and §11
	// forbids rejecting one over a missing index.md by name.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: Topic\n---\nbody\n")
	got := Soft(read(t, dir), Context{Root: dir})
	if len(got) == 0 {
		t.Fatal("the fixture tripped no soft rule at all")
	}
	for _, f := range got {
		if f.Severity == check.Error {
			t.Fatalf("%s is an error", f.Name())
		}
	}
}

func TestNoRuleFiresForAMissingResource(t *testing.T) {
	// Deliberate: OKF calls `resource` recommended and in the same breath
	// says it is absent for abstract concepts. Three of the schema's four
	// page types are abstract.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"),
		[]byte("---\ntype: Topic\ntitle: T\ndescription: d\n---\nb\n"), 0o644)
	for _, f := range Soft(read(t, dir), Context{Root: dir}) {
		if strings.Contains(f.Rule, "resource") || strings.Contains(f.Rule, "tags") {
			t.Fatalf("unexpected finding %s", f.Name())
		}
	}
}

func TestTitleAndDescriptionAreJudgedSeparately(t *testing.T) {
	// The two warnings fire together on nearly every page that states
	// neither key, so only pages breaking exactly one of them can tell the
	// rules apart. A fixture that tripped both would survive having the
	// two names swapped, or one rule reading the other's field.
	dir := t.TempDir()
	write(t, dir, "titled.md",
		"---\ntype: Topic\ntitle: T\ngenerated:\n  by: human:x\n---\nb\n")
	write(t, dir, "described.md",
		"---\ntype: Topic\ndescription: d\ngenerated:\n  by: h:x\n---\nb\n")
	got := Soft(read(t, dir), Context{Root: dir})
	if hasRuleOn(got, "titled.md", "title-missing") {
		t.Fatal("a page carrying a title was reported as missing one")
	}
	if !hasRuleOn(got, "titled.md", "description-missing") {
		t.Fatal("a page carrying no description was not reported")
	}
	if hasRuleOn(got, "described.md", "description-missing") {
		t.Fatal("a page carrying a description was reported as missing it")
	}
	if !hasRuleOn(got, "described.md", "title-missing") {
		t.Fatal("a page carrying no title was not reported")
	}
}

func TestTheTwoRecommendedKeysAskForDifferentRepairs(t *testing.T) {
	// Two keys, two repairs. One shared message would let the two rules
	// collapse into one without any assertion noticing.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: Topic\n---\nbody\n")
	got := Soft(read(t, dir), Context{Root: dir})
	title := messageOf(t, got, "a.md", "title-missing")
	description := messageOf(t, got, "a.md", "description-missing")
	if title == description {
		t.Fatalf("one message for two repairs: %q", title)
	}
}

func TestBrokenFrontmatterGetsNoSoftFinding(t *testing.T) {
	// Title, description and the trust keys all read empty here only
	// because the decode failed; every one of them may well stand in the
	// block. `okf/frontmatter-unparsable` owns this page, and a warning
	// beside it would name a defect that is not there.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: [unclosed\n---\nbody\n")
	got := Soft(read(t, dir), Context{Root: dir})
	for _, rule := range rulesOn(got, "a.md") {
		t.Fatalf("%s fired on an unreadable frontmatter", rule)
	}
}

func TestScaffoldFilesAreNotJudgedAsConcepts(t *testing.T) {
	// §3.1 reserves index.md and log.md, and this repository adds three
	// more names that carry structure rather than knowledge. §4.1 and §5
	// speak about concept documents, so the page rules skip all five.
	dir := t.TempDir()
	write(t, dir, "index.md", "# c\n\n* [a](a.md) - d\n")
	write(t, dir, "log.md", "# l\n\n## 2026-01-01\n* x\n")
	write(t, dir, "_schema.md", "# s\n")
	write(t, dir, "audit.md", "# a\n")
	write(t, dir, "a.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	if len(got) != 0 {
		t.Fatalf("a scaffolded bundle produced %v", got)
	}
}

func TestSourceIdMissingJudgesTheIdAndNotTheResource(t *testing.T) {
	// The two source rules sit on the same entry of the same axis. An
	// entry lacking both keys would let this rule read `Resource`
	// unnoticed, so one entry breaks each key and neither breaks both.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: Topic\ntitle: T\ndescription: d\n"+
		"generated:\n  by: human:x\nsources:\n"+
		"  - resource: https://example.invalid/one\n"+
		"  - id: two\n    resource: \"\"\n---\nbody\n")
	got := Soft(read(t, dir), Context{Root: dir})
	ids := messagesOf(got, "source-id-missing")
	if len(ids) != 1 || !strings.Contains(ids[0], "sources[0]") {
		t.Fatalf("expected the first entry alone, got %v", ids)
	}
}

func TestIndexEntryWithoutDescriptionNamesTheEntry(t *testing.T) {
	// §8: "Entries SHOULD include the description from the linked
	// concept's frontmatter." Naming the entry turns the finding into a
	// repair, and a described entry beside it keeps the rule from simply
	// counting links.
	//
	// b.md is listed twice, bare and then described. The scan has to walk a
	// cursor through the body for that: restarting each search at the top
	// would answer both entries with the first occurrence and report the
	// described one as well.
	dir := t.TempDir()
	write(t, dir, "index.md",
		"# c\n\n* [A](a.md) - what a is\n* [B](b.md)\n"+
			"* [C](c.md) -\n* [B](b.md) - said properly\n")
	write(t, dir, "a.md", full)
	write(t, dir, "b.md", full)
	write(t, dir, "c.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	entries := messagesOf(got, "index-entry-without-description")
	if len(entries) != 2 {
		t.Fatalf("expected b.md and c.md, got %v", entries)
	}
	if !strings.Contains(entries[0], "b.md") ||
		!strings.Contains(entries[1], "c.md") {
		t.Fatalf("the messages do not name the entries: %v", entries)
	}
}

func TestOnlyACatalogHasEntriesToDescribe(t *testing.T) {
	// The rule reads `p.Links`, and a concept page carries those too, so
	// the index.md guard needs a page that would trip the rule without it.
	// Dropping the guard turns every undescribed markdown link in the
	// bundle into a warning: over docs/wiki that is 23 findings instead of
	// seven. No earlier fixture gave a concept page a link at all.
	dir := t.TempDir()
	write(t, dir, "index.md", "# c\n\n* [a](a.md) - d\n* [b](b.md) - d\n")
	write(t, dir, "a.md", "---\ntype: Topic\ntitle: T\ndescription: d\n"+
		"generated:\n  by: human:x\n---\nsee [b](b.md)\n")
	write(t, dir, "b.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	if hasRuleOn(got, "a.md", "index-entry-without-description") {
		t.Fatalf("a concept page's link was judged as an entry: %v", got)
	}
}

func TestACatalogWithBrokenFrontmatterKeepsItsEntriesRead(t *testing.T) {
	// `judgedAsConcept` drops a page whose frontmatter did not parse,
	// because the fields it guards -- title, description, the key list --
	// read empty for that reason alone. This rule reads `Links` and
	// `RawBody`, which the failed decode leaves untouched, so the same
	// exclusion here would drop a finding that is true. The line runs
	// along what a rule reads, not along the page.
	dir := t.TempDir()
	write(t, dir, "index.md",
		"---\nokf_version: [unclosed\n---\n# c\n\n* [a](a.md)\n")
	write(t, dir, "a.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "index.md", "index-entry-without-description") {
		t.Fatalf("an unreadable block hid a readable entry: %v", got)
	}
}

func TestATitledLinkDoesNotAnswerForALaterEntry(t *testing.T) {
	// The reader's link pattern refuses a title, so `[X](a.md "T")` never
	// reaches `Links` (`internal/brain/wiki/parse.go:44`). Its `](a.md` stands in the
	// body all the same, and a scan taking the first occurrence would let
	// that line describe the listed entry below -- a warning on a catalog
	// that does describe what it lists.
	dir := t.TempDir()
	write(t, dir, "index.md",
		"# c\n\n* [X](a.md \"T\")\n* [A](a.md) - what a is\n")
	write(t, dir, "a.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	if hasRule(got, "index-entry-without-description") {
		t.Fatalf("a titled link answered for the listed entry: %v", got)
	}
}

func TestAFragmentedTargetIsReadAsOneEntry(t *testing.T) {
	// §6.1 links may carry a `#fragment`, which the reader drops from
	// `Links`. The scan has to take the fragment as closing the target,
	// or a catalog written that way loses every entry to the miss branch.
	got, _ := entryDescription("* [A](a.md#sec) - d\n", "a.md", 0)
	if got != "d" {
		t.Fatalf("a fragmented target was not described: %q", got)
	}
}

func TestEntryDescriptionIsTotalOnAForeignBody(t *testing.T) {
	// Through the reader both misses are unreachable -- the target was cut
	// out of this very body. The guards are measured here directly, since
	// a caller handing in another body has to get an empty answer rather
	// than a panic.
	if got, _ := entryDescription("no link at all", "a.md", 0); got != "" {
		t.Fatalf("a body without the target described something: %q", got)
	}
	if got, _ := entryDescription("[A](a.md", "a.md", 0); got != "" {
		t.Fatalf("an unclosed link described something: %q", got)
	}
	if got, _ := entryDescription("[A](a.md#s", "a.md", 0); got != "" {
		t.Fatalf("an unclosed fragment described something: %q", got)
	}
}

func TestNoIndexNotesEachDirectoryWithoutACatalogOnce(t *testing.T) {
	// §8 lets an index.md stand in any directory and §11 forbids rejecting
	// a bundle for its absence, so this is a note. The root carries one
	// here, which pins that the rule names a root catalog "index.md" and
	// not "./index.md".
	dir := t.TempDir()
	write(t, dir, "index.md",
		"# c\n\n* [a](a.md) - d\n* [sub](sub/) - d\n")
	write(t, dir, "a.md", full)
	write(t, dir, "sub/x.md", full)
	write(t, dir, "sub/y.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	if hasRuleOn(got, "index.md", "no-index") {
		t.Fatal("a directory carrying a catalog was noted")
	}
	if n := len(messagesOf(got, "no-index")); n != 1 {
		t.Fatalf("expected one note for sub, got %d", n)
	}
	if !hasRuleOn(got, "sub/index.md", "no-index") {
		t.Fatal("the missing catalog was not named where it belongs")
	}
}

func TestNoIndexNamesTheRootCatalogWithoutADotSegment(t *testing.T) {
	// `path.Dir` answers "." for a page at the bundle root, so the note has
	// to join the two the way `path` does and not by hand: "./index.md"
	// names no file any reader or catalog rule would recognise.
	dir := t.TempDir()
	write(t, dir, "a.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "index.md", "no-index") {
		t.Fatalf("the root catalog was not named index.md: %v", got)
	}
}

func TestNoIndexCountsAScaffoldFileAsContent(t *testing.T) {
	// §8 has a catalog enumerate "the directory's contents", and this
	// repository's own root catalog duly lists `_schema.md`, `audit.md`
	// and `log.md` (`ultra-brain/docs/wiki/index.md:12-14`). A directory holding
	// nothing but a scaffold file is therefore still a directory nothing
	// lists. Reading the skip as `IsScaffoldFile` reverses this case and
	// this case alone, so it needs a fixture of its own.
	dir := t.TempDir()
	write(t, dir, "index.md", "# c\n\n* [sub](sub/) - d\n")
	write(t, dir, "sub/log.md", "# l\n\n## 2026-01-01\n* x\n")
	got := Soft(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "sub/index.md", "no-index") {
		t.Fatalf("a directory holding only a log was not noted: %v", got)
	}
}

func TestNoLogAsksAfterTheBundleRootOnly(t *testing.T) {
	// §9 lets a log stand "at any level of the hierarchy". Noting every
	// level without one would put a note on every directory of every
	// bundle, so only the root is asked after.
	dir := t.TempDir()
	write(t, dir, "sub/x.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	if !hasRuleOn(got, "log.md", "no-log") {
		t.Fatal("a bundle root without a log was not noted")
	}
	if hasRuleOn(got, "sub/log.md", "no-log") {
		t.Fatal("a subdirectory without a log was noted")
	}
}

func TestABundleWithALogIsNotNoted(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "log.md", "# l\n\n## 2026-01-01\n* x\n")
	write(t, dir, "a.md", full)
	if hasRule(Soft(read(t, dir), Context{Root: dir}), "no-log") {
		t.Fatal("a bundle carrying a log was noted")
	}
}

func TestAnEmptyBundleIsJudgedByNothing(t *testing.T) {
	// A tree without a single page is not a bundle whose log went missing.
	dir := t.TempDir()
	if got := Soft(read(t, dir), Context{Root: dir}); len(got) != 0 {
		t.Fatalf("an empty tree produced %v", got)
	}
}

func TestVerifiedAloneIsATrustFamily(t *testing.T) {
	// §5.2 keeps `generated` and `verified` apart, and §5.3 reads the
	// trust tier off `verified` alone. The reader has no typed `verified`
	// field, so the rule asks `FrontmatterKeys`; reading `p.Generated`
	// instead would note a page that states its trust the other way.
	dir := t.TempDir()
	write(t, dir, "v.md", "---\ntype: Topic\ntitle: T\ndescription: d\n"+
		"verified:\n  - by: human:x\n---\nbody\n")
	write(t, dir, "n.md",
		"---\ntype: Topic\ntitle: T\ndescription: d\n---\nbody\n")
	got := Soft(read(t, dir), Context{Root: dir})
	if hasRuleOn(got, "v.md", "no-trust-family") {
		t.Fatal("a verified page was noted as stating no trust family")
	}
	if !hasRuleOn(got, "n.md", "no-trust-family") {
		t.Fatal("a page stating neither family was not noted")
	}
}

func TestAnEmptyGeneratedBlockIsStillATrustFamily(t *testing.T) {
	// `generated:` with no value leaves `p.Generated` nil while the key
	// stands in the block. §5.3 makes the key the criterion, and the empty
	// block already answers to `okf/generated-by-missing`.
	dir := t.TempDir()
	write(t, dir, "a.md", "---\ntype: Topic\ntitle: T\ndescription: d\n"+
		"generated:\n---\nbody\n")
	if hasRule(Soft(read(t, dir), Context{Root: dir}), "no-trust-family") {
		t.Fatal("a page declaring generated was noted as stating none")
	}
}

func TestAnEntryWithNoFrontmatterToQuoteIsNotWarnedAbout(t *testing.T) {
	// §8 asks for "the description from the linked concept's
	// frontmatter". A subdirectory has none, and neither has a scaffold
	// file -- so on those two the recommendation has no object, and a
	// warning about them asks for something no page could supply.
	//
	// The generator agrees and cannot do otherwise: `render_catalog`
	// writes a subdirectory line as `* [name](name/)` with no room for a
	// description at all, and only a file line carries one and only when
	// the document has one (`src/brain/catalog.py:51,57-58`).
	//
	// The two arms are separate entries here, because either alone would
	// leave the other's line unmeasured.
	dir := t.TempDir()
	write(t, dir, "index.md",
		"# c\n\n* [Bereich](topics/)\n* [Schema](_schema.md)\n"+
			"* [Protokoll](log.md)\n* [Prüfung](audit.md)\n"+
			"* [Unten](topics/index.md)\n* [A](a.md)\n")
	write(t, dir, "a.md", full)
	got := Soft(read(t, dir), Context{Root: dir})
	entries := messagesOf(got, "index-entry-without-description")
	if len(entries) != 1 || !strings.Contains(entries[0], "a.md") {
		t.Fatalf("expected the concept page alone, got %v", entries)
	}
}

func TestAnEntryRightAtTheStartOfTheScanIsDescribed(t *testing.T) {
	// The scan resumes where the last entry ended; an entry whose marker
	// stands right there is found at offset zero of what is left. Found by
	// the mutation round of 2026-09-23.
	if got, _ := entryDescription("* [A](a.md) - d\n", "a.md", 4); got != "d" {
		t.Fatalf("an entry at the scan's start was not described: %q", got)
	}
}
