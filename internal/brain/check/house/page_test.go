package house

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
)

// day is the one date every test measures against. `stale` compares
// calendar days, so the time of day is picked to be the last minute of
// one: a rule that compared instants instead would call a page carrying
// `stale_after: 2026-09-02` expired, and the edge test below says so.
var day = time.Date(2026, 9, 2, 23, 59, 0, 0, time.UTC)

// read is the bundle reader the rules expect, copied in form from
// `internal/brain/check/okf/errors_test.go:18`: one walk, every `.md` file, the page
// set the rules then run over. The helpers there are unexported and stay
// so -- exporting a test helper would put it into the package's API for
// the sake of a second test file.
func read(t *testing.T, dir string) []wiki.WikiPage {
	t.Helper()
	var pages []wiki.WikiPage
	err := filepath.WalkDir(dir,
		func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}
			page, err := wiki.ReadPage(p, dir)
			if err != nil {
				return err
			}
			pages = append(pages, *page)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeManifest puts an area declaration where `config.ReadManifest`
// looks for one: `.loomux/config.toml`, the only name it reads.
func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	write(t, dir, ".loomux/config.toml", body)
}

// ctx is the glue the future `brain check` does: read the manifest, hand
// its type vocabulary to the rules. It is deliberately thin -- the
// judgement of what counts as a known type lives in `Page`, not here, or
// the tests would prove the helper rather than the rule.
//
// A missing or unreadable manifest leaves `DeclaredTypes` nil, which is
// what a bundle with no declaration of its own gets. The built-in
// vocabulary still applies; `config.KnowsType` supplies it.
func ctx(t *testing.T, dir string) Context {
	t.Helper()
	out := Context{Root: dir, Now: day}
	manifest, err := config.ReadManifest(dir)
	if err != nil {
		return out
	}
	out.UntouchedDays = manifest.UntouchedDays
	out.DeclaredTypes = map[string]bool{}
	for _, declared := range manifest.DeclaredTypes {
		out.DeclaredTypes[declared] = true
	}
	return out
}

func hasRule(findings []check.Finding, rule string) bool {
	for _, f := range findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

// sev answers with the severity of the first finding under a rule, and
// with the empty severity when the rule did not fire at all. The two
// cases are told apart by `hasRule`, which every caller asks first.
func sev(findings []check.Finding, rule string) check.Severity {
	for _, f := range findings {
		if f.Rule == rule {
			return f.Severity
		}
	}
	return ""
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

// complete is a page that breaks none of the five rules. Fixtures that
// want one rule to fire build from it, so that no assertion rests on a
// page which happens to trip a second rule as well.
const complete = "---\ntype: Topic\nsources:\n  - id: a\n" +
	"    resource: r\n    doc_id: d\n    content_hash: h\n" +
	"    revision: 1\n---\nbody\n"

// sourced is the same source block on its own, for fixtures that state a
// different set of frontmatter keys around it.
const sourced = "sources:\n  - id: a\n    resource: r\n    doc_id: d\n" +
	"    content_hash: h\n    revision: 1\n"

func TestASynthesisIsNotExemptFromSources(t *testing.T) {
	// Scheibe 3 §4: "Gilt für alle vier Typen -- auch eine Synthesis belegt
	// ihre Thesen." OKF leaves `sources` free; without the duty to cite,
	// this layer would have no purpose.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "s.md"), []byte("---\ntype: Synthesis\n---\nb\n"), 0o644)
	found := Page(read(t, dir), ctx(t, dir))
	if !hasRule(found, "no-sources") {
		t.Fatal("a synthesis without sources went unreported")
	}
	// One rule, as `complete` above promises of every fixture here. The
	// type is spelled `Synthesis` for that reason and not for tidiness:
	// in lower case it drew `unknown-type` alongside, and the assertion
	// on `no-sources` passed all the same.
	for _, f := range found {
		if f.Rule != "no-sources" {
			t.Errorf("second rule %q on a fixture that means to break one",
				f.Rule)
		}
	}
}

func TestAnUnknownTypeIsAnErrorHereEvenThoughOKFTolerates(t *testing.T) {
	// OKF §4.1 asks consumers to tolerate unknown types. We are the producer
	// and keep a type catalogue: "Ein Tippfehler ist ein Befund, kein neuer
	// Typ" (Scheibe 3 §4).
	dir := t.TempDir()
	writeManifest(t, dir, `[area]
scope = "k"
`)
	os.WriteFile(filepath.Join(dir, "p.md"),
		[]byte("---\ntype: topci\nsources:\n  - id: a\n    resource: r\n    doc_id: d\n    content_hash: h\n    revision: 1\n---\nb\n"), 0o644)
	found := Page(read(t, dir), ctx(t, dir))
	if !hasRule(found, "unknown-type") {
		t.Fatal("the typo `topci` was accepted as a new type")
	}
	if sev(found, "unknown-type") != check.Error {
		t.Fatal("unknown-type is a warning in Go and an error in Python; the two must agree")
	}
}

func TestTheFiveHouseRulesAreErrorsOnTheHouseAxis(t *testing.T) {
	// The owner's principle, pinned as a table: all five are errors, and
	// all five speak on the house axis. One run trips every one of them, so
	// no rule can carry the wrong axis or degree and go unmeasured because
	// nothing made it fire.
	want := []string{
		"no-sources", "source-incomplete", "unknown-type",
		"conflict-count", "stale",
	}
	dir := t.TempDir()
	write(t, dir, "empty.md", "---\ntype: Topic\n---\nb\n")
	write(t, dir, "partial.md",
		"---\ntype: Topic\nsources:\n  - resource: r\n---\nb\n")
	write(t, dir, "typo.md", "---\ntype: topci\n"+sourced+"---\nb\n")
	write(t, dir, "boxes.md",
		"---\ntype: Topic\n"+sourced+"---\n> [!conflict] c\n")
	write(t, dir, "old.md", "---\ntype: Topic\n"+
		"stale_after: 2026-09-01\n"+sourced+"---\nb\n")
	got := Page(read(t, dir), ctx(t, dir))
	seen := map[string]bool{}
	for _, f := range got {
		if f.Axis != check.AxisHouse {
			t.Fatalf("%s is not on the house axis", f.Name())
		}
		if f.Severity != check.Error {
			t.Fatalf("%s is %s, want error", f.Name(), f.Severity)
		}
		seen[f.Rule] = true
	}
	for _, rule := range want {
		if !seen[rule] {
			t.Fatalf("%s never fired", rule)
		}
	}
	if len(seen) != len(want) {
		t.Fatalf("unexpected rules fired: %v", seen)
	}
}

func TestAKnownTypeIsAcceptedWithoutAManifest(t *testing.T) {
	// The guard against the cheapest wrong implementation of
	// `unknown-type`: reading `ctx.DeclaredTypes` alone would make every
	// type unknown in a bundle whose area declares none, and `.loomux/config.toml`
	// of this very repository declares none. The built-in vocabulary of
	// `config.KnowsType` is what answers for `Topic` here.
	dir := t.TempDir()
	write(t, dir, "p.md", complete)
	if hasRule(Page(read(t, dir), ctx(t, dir)), "unknown-type") {
		t.Fatal("`Topic` was reported unknown although it is a core type")
	}
}

func TestAPaddedTypeIsNotTrimmedIntoAKnownOne(t *testing.T) {
	// YAML strips a plain scalar, so the padding can only arrive quoted --
	// and then it belongs to the value. `rank_of(" Topic ", ...)` is
	// unknown on the Python side, measured, so this page has to be
	// reported here as well. The rule trims for its blank guard alone and
	// asks `KnowsType` with the value as written.
	dir := t.TempDir()
	write(t, dir, "p.md", "---\ntype: \" Topic \"\n"+sourced+"---\nbody\n")
	if !hasRule(Page(read(t, dir), ctx(t, dir)), "unknown-type") {
		t.Fatal("a padded `Topic` passed as the core type it is not")
	}
}

func TestABlankTypeIsLeftToTheOtherAxis(t *testing.T) {
	// The guard the test above must not break: a type that is nothing but
	// blanks is `okf/type-missing`, and naming it here as well would give
	// one defect two names on two axes.
	dir := t.TempDir()
	write(t, dir, "p.md", "---\ntype: \"   \"\n"+sourced+"---\nbody\n")
	if hasRule(Page(read(t, dir), ctx(t, dir)), "unknown-type") {
		t.Fatal("a blank type was reported as an unknown one")
	}
}

func TestATypeTheManifestDeclaresIsAccepted(t *testing.T) {
	// The other half: an area may widen the vocabulary, and Scheibe 3 §4
	// makes the manifest the place it does so. `playbook` is in neither
	// built-in set, so only the declaration can make it known.
	dir := t.TempDir()
	writeManifest(t, dir, "[wiki]\ntypes = [\"playbook\"]\n")
	write(t, dir, "p.md", strings.Replace(complete,
		"type: Topic", "type: playbook", 1))
	if hasRule(Page(read(t, dir), ctx(t, dir)), "unknown-type") {
		t.Fatal("a type the manifest declares was reported unknown")
	}
}

func TestAnEmptyTypeIsLeftToTheOKFRule(t *testing.T) {
	// `okf/type-missing` owns both an absent and an empty `type`. A page
	// that states neither has one defect, and two names for it would send
	// the reader to the wrong repair.
	dir := t.TempDir()
	write(t, dir, "blank.md", strings.Replace(complete,
		"type: Topic", "type: \"  \"", 1))
	write(t, dir, "none.md", strings.Replace(complete,
		"type: Topic\n", "", 1))
	if hasRule(Page(read(t, dir), ctx(t, dir)), "unknown-type") {
		t.Fatal("a page with no usable type was reported as unknown-type")
	}
}

func TestAnIncompleteSourceIsNotAnEmptyOne(t *testing.T) {
	// The two rules the design splits out of the single Python
	// `no-sources` (§5.3). They divide the same page set between them, so
	// each has to stay silent where the other speaks.
	dir := t.TempDir()
	write(t, dir, "partial.md",
		"---\ntype: Topic\nsources:\n  - resource: r\n---\nb\n")
	got := Page(read(t, dir), ctx(t, dir))
	if !hasRule(got, "source-incomplete") {
		t.Fatal("an entry without doc_id went unreported")
	}
	if hasRule(got, "no-sources") {
		t.Fatal("a page that has an entry was called empty")
	}
}

func TestAnEmptySourcesListIsNotAnIncompleteEntry(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "empty.md", "---\ntype: Topic\nsources: []\n---\nb\n")
	got := Page(read(t, dir), ctx(t, dir))
	if !hasRule(got, "no-sources") {
		t.Fatal("an empty sources[] went unreported")
	}
	if hasRule(got, "source-incomplete") {
		t.Fatal("a page with no entries was reported for a broken one")
	}
}

func TestSourceIncompleteNamesEveryFieldItMisses(t *testing.T) {
	// The message names the repair. Three fields hang on this rule -- the
	// three the design names at line 156 of
	// `2026-09-01-pruefkatalog-in-go-design.md` -- and a message that only
	// said "incomplete" would make the reader diff the entry against the
	// spec by hand. The whole entry comes first and the broken one second,
	// from the mutation round: with the broken entry at position zero, a
	// rule that named position zero for every entry passed -- and one
	// broken entry among many is the normal case this rule is for.
	dir := t.TempDir()
	write(t, dir, "p.md",
		"---\ntype: Topic\nsources:\n"+
			"  - id: a\n    resource: r\n    doc_id: d\n"+
			"    content_hash: h\n    revision: 1\n"+
			"  - id: b\n    resource: r\n---\nb\n")
	got := messagesOf(Page(read(t, dir), ctx(t, dir)),
		"source-incomplete")
	want := "sources[1] lacks doc_id, content_hash, revision"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages are %v, want [%q]", got, want)
	}
}

func TestAnEntryMissingOnlyAnOKFFieldIsLeftToThatAxis(t *testing.T) {
	// The rule's own reason, applied to itself. Its comment argues that
	// naming `resource` here would report one defect on two axes, because
	// OKF §5.1 makes it REQUIRED and `okf/source-resource-missing` owns
	// it -- but nothing held the rule to that, and adding a `resource` arm
	// left the whole suite green.
	//
	// `id` is the same case and used to be inside this rule. OKF §5.1
	// puts it under SHOULD, `okf/source-id-missing` reports it, and the
	// three fields the house axis owns are the ones Scheibe 3 §4 names
	// (`2026-08-23-scheibe-3-wiki-schicht-design.md:178`) and the design
	// repeats as "die drei Felder, an denen die Wartung hängt"
	// (`2026-09-01-pruefkatalog-in-go-design.md:156`): `doc_id`,
	// `content_hash`, `revision`. Measured, an entry without `id` drew
	// both `okf/source-id-missing` and `house/source-incomplete`.
	//
	// Two fixtures, one per abandoned field, each whole in every other
	// respect -- so a rule that reached for either would have nothing else
	// to blame the entry for.
	dir := t.TempDir()
	write(t, dir, "no-resource.md", "---\ntype: Topic\nsources:\n"+
		"  - id: a\n    doc_id: d\n    content_hash: h\n"+
		"    revision: 1\n---\nb\n")
	write(t, dir, "no-id.md", "---\ntype: Topic\nsources:\n"+
		"  - resource: r\n    doc_id: d\n    content_hash: h\n"+
		"    revision: 1\n---\nb\n")
	if got := messagesOf(Page(read(t, dir), ctx(t, dir)),
		"source-incomplete"); len(got) != 0 {
		t.Fatalf("an OKF field was claimed by the house axis: %v", got)
	}
}

func TestTheTwoConflictArmsAskForDifferentRepairs(t *testing.T) {
	// Architektur §9.3 has the count checked against the boxes. A page
	// that states no number has to gain the key, a page that states the
	// wrong one has to correct it -- two repairs, so two messages, and the
	// second arm must not swallow the first.
	dir := t.TempDir()
	write(t, dir, "a-missing.md",
		"---\ntype: Topic\n"+sourced+"---\n> [!conflict] c\n")
	write(t, dir, "b-wrong.md", "---\ntype: Topic\nopen_conflicts: 0\n"+
		sourced+"---\n> [!conflict] c\n")
	write(t, dir, "c-right.md", "---\ntype: Topic\nopen_conflicts: 1\n"+
		sourced+"---\n> [!conflict] c\n")
	got := messagesOf(Page(read(t, dir), ctx(t, dir)), "conflict-count")
	want := []string{
		"1 conflict box(es) present, open_conflicts is missing",
		"open_conflicts says 0, 1 box(es) found",
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("messages are %v, want %v", got, want)
	}
}

func TestAPageWithNoBoxesNeedNotCountThem(t *testing.T) {
	// From the mutation round: dropping the zero-box arm of
	// `conflict-count` left every test green, while every page of every
	// bundle here would gain a finding -- none of them state
	// `open_conflicts`, and demanding the field everywhere would put a zero
	// on all of them. Nothing else asserts the whole run is silent, so this
	// fixture holds the five rules to saying nothing about a page that
	// breaks none of them.
	dir := t.TempDir()
	write(t, dir, "p.md", complete)
	if got := Page(read(t, dir), ctx(t, dir)); len(got) != 0 {
		t.Fatalf("a page that breaks nothing was reported: %v", got)
	}
}

func TestTwoMessagesAreQuotedFromPython(t *testing.T) {
	// Also from the mutation round: rewording either message left the
	// suite green. The design makes the agreement of the two sides cover
	// the message as well (§9.1), so the two rules that can quote Python
	// are pinned to its words -- `no-sources` at
	// src/brain/wiki/lint.py:128, `unknown-type` at :107-110. The %q
	// quoting is Go's, where Python writes !r; that difference and Python's
	// third branch for a renamed type are the reconciliation left over.
	// The length is asked before the index, the way
	// `TestSourceIncompleteNamesEveryFieldItMisses` asks it. Indexing a
	// slice that a broken rule left empty panics, and a panic takes the
	// whole test binary with it -- every other result of the run is then
	// lost, which was observed while measuring this file against the stub.
	dir := t.TempDir()
	write(t, dir, "empty.md", "---\ntype: Topic\n---\nb\n")
	write(t, dir, "typo.md", "---\ntype: topci\n"+sourced+"---\nb\n")
	found := Page(read(t, dir), ctx(t, dir))
	got := messagesOf(found, "no-sources")
	want := "sources[] is empty"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("no-sources says %v, want [%q]", got, want)
	}
	got = messagesOf(found, "unknown-type")
	want = "\"topci\" is neither core nor catalogue; add it to " +
		"`[wiki] types` in the area's manifest, or use a catalogue name"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("unknown-type says %v, want [%q]", got, want)
	}
}

func TestAZonedExpiryIsJudgedByTheDayItWrites(t *testing.T) {
	// `stale_after` may be written as a full timestamp, and then its zone
	// decides which day it names. `2026-09-02T00:00:00+02:00` is the second
	// of September where it was written and still the first in UTC, so the
	// instant falls before `ctx.Now`'s midnight while the day does not.
	// Measured with the reader: `.Date()` answers 2026-09-02 while `.UTC()`
	// answers 2026-09-01T22:00Z.
	//
	// From the fix round: dropping `calendarDay` from around the page's own
	// side of the comparison left every other fixture green, because they
	// all write a bare date, which parses to midnight UTC and cannot tell
	// the two readings apart.
	dir := t.TempDir()
	write(t, dir, "p.md", strings.Replace(complete, "type: Topic",
		"type: Topic\nstale_after: 2026-09-02T00:00:00+02:00", 1))
	if hasRule(Page(read(t, dir), ctx(t, dir)), "stale") {
		t.Fatal("a page expiring today in its own zone was called stale")
	}
}

func TestAPageThatExpiresTodayIsNotYetStale(t *testing.T) {
	// The edge, nailed down: `stale_after` names the last day the page is
	// good for, and Python compares dates -- `page.stale_after < today`
	// with `today = context.now.date()` (src/brain/wiki/lint.py:153,162).
	// A rule comparing instants would call this page expired, because
	// `ctx.Now` is the last minute of the day it names.
	dir := t.TempDir()
	write(t, dir, "p.md", strings.Replace(complete, "type: Topic",
		"type: Topic\nstale_after: 2026-09-02", 1))
	if hasRule(Page(read(t, dir), ctx(t, dir)), "stale") {
		t.Fatal("a page whose last good day is today was called stale")
	}
}

func TestAPageThatExpiredYesterdayIsStale(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "p.md", strings.Replace(complete, "type: Topic",
		"type: Topic\nstale_after: 2026-09-01", 1))
	got := messagesOf(Page(read(t, dir), ctx(t, dir)), "stale")
	want := "stale_after 2026-09-01 has passed"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages are %v, want [%q]", got, want)
	}
}

func TestScaffoldFilesAreNotJudgedByThePageRules(t *testing.T) {
	// Scheibe 3 §4 exempts the five scaffold files by name, and Python
	// never hands one to any rule (src/brain/wiki/lint.py:582-585). The
	// case is not hypothetical: `docs/wiki/_schema.md` documents the
	// conflict syntax with a `> [!conflict]` example, which the reader
	// counts as a box -- so without this line the file that defines the
	// rule would be its first violator.
	dir := t.TempDir()
	write(t, dir, "_schema.md", "---\ntype: topci\n"+
		"stale_after: 2026-09-01\nsources:\n  - resource: r\n---\n"+
		"> [!conflict] c\n")
	if got := Page(read(t, dir), ctx(t, dir)); len(got) != 0 {
		t.Fatalf("scaffold file reported: %v", got)
	}
}

func TestAnUnreadableFrontmatterIsLeftToItsOwnRule(t *testing.T) {
	// Every field these rules read comes out of the block that did not
	// decode, so all five would report a defect the page may not have.
	// `conflict-count` is the sharpest case: the boxes are counted in the
	// body and survive the failed decode, while `open_conflicts` does not
	// -- the page would be blamed for bookkeeping it may well have done.
	dir := t.TempDir()
	write(t, dir, "p.md", "---\ntype: [unclosed\n---\n> [!conflict] c\n")
	if got := Page(read(t, dir), ctx(t, dir)); len(got) != 0 {
		t.Fatalf("page with unreadable frontmatter reported: %v", got)
	}
}
