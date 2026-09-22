package maintenance_test

// The second producer of cases: a landed merge.
//
// Every world here is a repository, because the trigger is a git read and
// nothing else -- the event names a range, and the evidence is what git says
// about it. `requireGit` therefore stands in front of all of them.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/gitenv"
)

// mergePathsHeading and mergeSubjectsHeading are the two headings an evidence
// block opens with, spelt out here rather than borrowed from the package: they
// are written into `package.md`, which the Python side reads back, so a test
// that took them from the constant would agree with any rewording.
const (
	mergePathsHeading    = "# geänderte Dateipfade des Merges — nur Namen, kein Dateiinhalt"
	mergeSubjectsHeading = "# Commit-Betreffzeilen des Merges — je Commit der erste Absatz (git %s)"
)

// mergeArea is the area every merge world stands on: a repository with a
// review centre, one registered source, and one wiki page that still promises
// something -- which is what makes it a candidate.
func mergeArea(scope string) areaOptions {
	return areaOptions{
		Scope:  scope,
		Review: reviewLayout,
		Wiki:   "docs/wiki",
		Git:    true,
		Files:  map[string]string{"src/a.go": "package a\n"},
		Cites:  map[string][]string{"docs/wiki/a.md": {"src/a.go"}},
		// The second page is the one that must **not** become a candidate: a
		// page whose promise is kept has nothing left for this trigger to ask
		// about, and without it here every case below would pass for an area
		// that simply has one page.
		Realizations: map[string]string{
			"docs/wiki/a.md": "planned", "docs/wiki/done.md": "implemented",
		},
	}
}

// mergeWorld is that area with a second commit on top, so a range with two
// ends exists, and with the event of that range already recorded.
func mergeWorld(t *testing.T) *world {
	t.Helper()
	requireGit(t)
	w := newWorld(t)
	w.addArea(t, mergeArea("project/one"))
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "build the promised thing")
	w.Events(t, w.Event(t, "project/one"))
	return w
}

// packageOf is the package that lies beside one raised case.
func packageOf(t *testing.T, w *world, c maintenance.Case) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(caseDirOf(w, c), "package.md"))
	if err != nil {
		t.Fatalf("ReadFile package.md: %v", err)
	}
	return string(raw)
}

// The trigger itself: a recorded merge, one case per open page, and the
// evidence in the package -- names and subjects, never a line of source.
func TestReconcileRaisesAMergeCase(t *testing.T) {
	w := mergeWorld(t)
	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("want one case, got %d: %+v", len(report.Cases), report.Cases)
	}
	landed := report.Cases[0]
	if landed.Trigger != "merge" {
		t.Fatalf("trigger is %q, not merge", landed.Trigger)
	}
	// A merge is an event and not a due date coming round, and the vocabulary
	// is closed at `change | time`: a third value would change the format
	// ahead of the measurement that would have to justify it.
	if landed.Weight != "change" {
		t.Fatalf("weight is %q, not change", landed.Weight)
	}
	if landed.State != "due" {
		t.Fatalf("state is %q, not due", landed.State)
	}
	if len(landed.Sources) != 0 {
		t.Fatalf("a merge case carries no sources, got %+v", landed.Sources)
	}
	if landed.Target != "a.md" {
		t.Fatalf("target is %q, not the open page", landed.Target)
	}
	text := packageOf(t, w, landed)
	for _, want := range []string{mergePathsHeading, "src/b.go", mergeSubjectsHeading, "build the promised thing"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the package misses %q:\n%s", want, text)
		}
	}
	// The one thing the evidence may never carry: what the changed files say.
	if strings.Contains(text, "package b") {
		t.Fatalf("source text reached the evidence:\n%s", text)
	}
	if w.Dropped(t) == "" {
		t.Fatal("the consumed event was not dropped")
	}
}

// A page already under review is not reviewed twice: the merge waits.
//
// Both halves of the rule are here, and the first is the one that decides the
// order. A changed source and a recorded merge in **one** pass: the source
// case has to land first, and the merge then finds it standing. Raised the
// other way round, the merge case would land on the free page and the source
// change would find *it* standing and step back -- one case either way, but
// the wrong one, and the source diff would wait behind an event instead of the
// other way about.
func TestMergeCaseDefersToAStandingSourceCase(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	w.addArea(t, mergeArea("project/one"))
	w.Change(t, "project/one", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "second")
	w.Events(t, w.Event(t, "project/one"))

	first := mustReconcile(t, w, w.Now())
	if len(first.Cases) != 1 {
		t.Fatalf("want one case, got %d: %+v", len(first.Cases), first.Cases)
	}
	standing := first.Cases[0]
	if standing.Trigger != "source_change" {
		t.Fatalf("the merge was raised in front of the source change: %+v", standing)
	}
	if w.Dropped(t) != "" {
		t.Fatalf("a deferred event was dropped: %q", w.Dropped(t))
	}

	// And it keeps waiting: the second pass changes nothing for either.
	second := mustReconcile(t, w, w.Now())
	if len(second.Cases) != 1 {
		t.Fatalf("want one case beside the standing one, got %d: %+v", len(second.Cases), second.Cases)
	}
	if second.Cases[0].ID != standing.ID || second.Cases[0].Trigger != "source_change" {
		t.Fatalf("the standing source case was replaced: %+v", second.Cases[0])
	}
	if w.Dropped(t) != "" {
		t.Fatalf("a deferred event was dropped on the second pass: %q", w.Dropped(t))
	}
}

// The other direction, and the one that would lose knowledge: a standing merge
// case carries evidence out of an event that is already consumed, so a source
// change may not write over it.
func TestSourceChangeDoesNotOverwriteAStandingMergeCase(t *testing.T) {
	w := mergeWorld(t)
	first := mustReconcile(t, w, w.Now())
	if len(first.Cases) != 1 || first.Cases[0].Trigger != "merge" {
		t.Fatalf("want one merge case, got %+v", first.Cases)
	}
	standing := first.Cases[0]
	evidence := packageOf(t, w, standing)

	w.Change(t, "project/one", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	second := mustReconcile(t, w, w.Now())
	if len(second.Cases) != 1 {
		t.Fatalf("want the one standing case, got %d: %+v", len(second.Cases), second.Cases)
	}
	if second.Cases[0].ID != standing.ID || second.Cases[0].Trigger != "merge" {
		t.Fatalf("the merge case was overwritten: %+v", second.Cases[0])
	}
	if got := packageOf(t, w, standing); got != evidence {
		t.Fatalf("the evidence changed:\ngot:\n%s\nwant:\n%s", got, evidence)
	}

	// The source change is not lost, only held back: once the merge case is
	// decided, the very next pass raises it.
	if err := os.RemoveAll(caseDirOf(w, standing)); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	third := mustReconcile(t, w, w.Now())
	if len(third.Cases) != 1 || third.Cases[0].Trigger != "source_change" {
		t.Fatalf("the held-back source change never landed: %+v", third.Cases)
	}
	if len(third.Cases[0].Sources) != 1 {
		t.Fatalf("the source case names no source: %+v", third.Cases[0])
	}
}

// A consumed event raises its cases once and never again.
func TestConsumedEventIsDropped(t *testing.T) {
	w := mergeWorld(t)
	mustReconcile(t, w, w.Now())
	dropped := w.Dropped(t)
	if strings.Count(dropped, "\n") != 1 {
		t.Fatalf("want one dropped line, got %q", dropped)
	}
	second := mustReconcile(t, w, w.Now())
	if len(second.Cases) != 0 {
		t.Fatalf("the spent event raised a second case: %+v", second.Cases)
	}
	if w.Dropped(t) != dropped {
		t.Fatalf("the drop log grew over a spent event: %q", w.Dropped(t))
	}
}

// The recovery path: a run that landed the cases and died before the drop.
// The event comes round again, finds its own cases standing, and is absorbed
// rather than deferred -- without that, an event would be blocked forever by
// the cases it raised itself.
func TestAnEventAbsorbsItsOwnStandingCases(t *testing.T) {
	w := mergeWorld(t)
	first := mustReconcile(t, w, w.Now())
	if len(first.Cases) != 1 {
		t.Fatalf("want one merge case, got %+v", first.Cases)
	}
	if err := os.Remove(filepath.Join(w.StateDir, "maintenance", "merge-events.done.tsv")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	second := mustReconcile(t, w, w.Now())
	if len(second.Cases) != 1 || second.Cases[0].ID != first.Cases[0].ID {
		t.Fatalf("the absorbed event did not find its own case: %+v", second.Cases)
	}
	if w.Dropped(t) == "" {
		t.Fatal("the absorbed event was not dropped again")
	}
}

// An area whose pages all keep their promise has nothing for this trigger to
// ask about -- and the event is spent all the same. Keeping it would raise the
// same nothing on every pass from now on.
func TestAMergeOverAnAreaWithoutOpenPagesIsStillDropped(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	area := mergeArea("project/one")
	area.Realizations = map[string]string{"docs/wiki/a.md": "implemented"}
	w.addArea(t, area)
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "nothing left to promise")
	w.Events(t, w.Event(t, "project/one"))

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 0 {
		t.Fatalf("a case was raised for a page nobody promised: %+v", report.Cases)
	}
	if w.Dropped(t) == "" {
		t.Fatal("an event with nothing left to produce was kept")
	}
}

// A different merge says something the standing case does not carry, so it
// defers rather than absorbs.
func TestADifferentMergeDefersToAStandingMergeCase(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	w.addArea(t, mergeArea("project/one"))
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "the first merge")
	earlier := w.Event(t, "project/one")
	w.CommitFile(t, "project/one", "src/c.go", "package c\n", "the second merge")
	later := w.Event(t, "project/one")
	w.Events(t, earlier, later)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("want one case, got %d: %+v", len(report.Cases), report.Cases)
	}
	text := packageOf(t, w, report.Cases[0])
	if !strings.Contains(text, "the first merge") || strings.Contains(text, "the second merge") {
		t.Fatalf("the standing case is not the first merge's:\n%s", text)
	}
	if strings.Count(w.Dropped(t), "\n") != 1 {
		t.Fatalf("want exactly the first event dropped, got %q", w.Dropped(t))
	}
}

// The package is the only place a merge's identity is written down, so a case
// whose package is gone cannot be shown to carry this evidence -- and the
// event waits rather than landing over it.
func TestAStandingCaseWithoutAPackageDefersTheEvent(t *testing.T) {
	w := mergeWorld(t)
	first := mustReconcile(t, w, w.Now())
	if err := os.Remove(filepath.Join(caseDirOf(w, first.Cases[0]), "package.md")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := os.Remove(filepath.Join(w.StateDir, "maintenance", "merge-events.done.tsv")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	second := mustReconcile(t, w, w.Now())
	if len(second.Cases) != 0 {
		t.Fatalf("the event landed although nothing proved its evidence: %+v", second.Cases)
	}
	if w.Dropped(t) != "" {
		t.Fatalf("a deferred event was dropped: %q", w.Dropped(t))
	}
}

// Two ways onto one id -- the same range recorded twice under two spellings of
// one repository. The log keeps both, because the key is the repository as it
// was written; the report may show the case once.
func TestReconcileDeduplicatesByID(t *testing.T) {
	w := mergeWorld(t)
	event := w.Event(t, "project/one")
	// A spelling `filepath.Join` would have folded away, which is the point:
	// the two events differ in their key and agree in every answer git gives
	// about them.
	spelt := event
	spelt.Repo = event.Repo + string(filepath.Separator) + "."
	w.Events(t, event, spelt)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("want one case for one id, got %d: %+v", len(report.Cases), report.Cases)
	}
	if strings.Count(w.Dropped(t), "\n") != 2 {
		t.Fatalf("both events are spent and both belong in the log, got %q", w.Dropped(t))
	}
}

// A range that touched nothing writes no heading over an empty list: the case
// still lands, and its package carries the page alone.
func TestAnEmptyRangeRaisesACaseWithoutEvidence(t *testing.T) {
	w := mergeWorld(t)
	head := w.Revision(t, "project/one", "HEAD")
	event := w.Event(t, "project/one")
	event.First = head
	w.Events(t, event)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("want one case, got %d: %+v", len(report.Cases), report.Cases)
	}
	text := packageOf(t, w, report.Cases[0])
	if strings.Contains(text, mergePathsHeading) || strings.Contains(text, mergeSubjectsHeading) {
		t.Fatalf("a heading was written over nothing:\n%s", text)
	}
	if !strings.Contains(text, "## W1 — Wiki,") {
		t.Fatalf("the page itself is missing from the package:\n%s", text)
	}
}

// An empty range makes the block test vacuous -- every block of nothing is
// quoted by every package -- so the trigger has to be asked on its own. Were
// it not, a merge over no commits would absorb a standing *source* case and
// land over it, and the verified source diff would be gone.
func TestAnEmptyRangeMayNotLandOnASourceCase(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	w.addArea(t, mergeArea("project/one"))
	w.Change(t, "project/one", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "second")
	event := w.Event(t, "project/one")
	event.First = event.Last
	w.Events(t, event)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 || report.Cases[0].Trigger != "source_change" {
		t.Fatalf("want the one source case, got %+v", report.Cases)
	}
	if w.Dropped(t) != "" {
		t.Fatalf("an empty range was absorbed by a source case: %q", w.Dropped(t))
	}
	// And the file on disk is the source case still, diffs and all: the report
	// would show the first version of the id either way.
	text := packageOf(t, w, report.Cases[0])
	if !strings.Contains(text, "func B() int") {
		t.Fatalf("the source diff was thrown away:\n%s", text)
	}
}

// Clause (a) of the ruling on merge events: a range that cannot be read
// keeps its event. Both ways it can fail are here -- refused before git ran, and refused
// by git -- and neither may take the pass down or spend the event.
func TestAnUnreadableRangeKeepsTheEvent(t *testing.T) {
	for name, first := range map[string]string{
		"no object name": "HEAD",
		"unknown commit": "0123456789abcdef0123456789abcdef01234567",
	} {
		t.Run(name, func(t *testing.T) {
			w := mergeWorld(t)
			event := w.Event(t, "project/one")
			event.First = first
			w.Events(t, event)

			report := mustReconcile(t, w, w.Now())
			if len(report.Cases) != 0 {
				t.Fatalf("a case was raised over an unreadable range: %+v", report.Cases)
			}
			if w.Dropped(t) != "" {
				t.Fatalf("the merge was lost for good: %q", w.Dropped(t))
			}
		})
	}
}

// A range that only one of the two reads accepts is unreadable too. A blob as
// the far end is such a range on git 2.54: `git log` walks it, `git diff`
// refuses it with a usage error. Half the evidence is no evidence, so the
// event stays and no case opens. Where a git build reads the blob in `diff`
// as well, the range is no vector and the test says so instead of passing.
func TestARangeOnlyGitLogReadsKeepsTheEvent(t *testing.T) {
	w := mergeWorld(t)
	event := w.Event(t, "project/one")
	event.Last = w.Revision(t, "project/one", "HEAD:src/b.go")
	probe := exec.Command("git", "diff", "--name-only", event.First+".."+event.Last)
	probe.Dir = event.Repo
	probe.Env = gitenv.Environ()
	if probe.Run() == nil {
		t.Skip("this git reads a blob range in diff; no range splits the two reads")
	}
	w.Events(t, event)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 0 {
		t.Fatalf("a case was raised over half the evidence: %+v", report.Cases)
	}
	if w.Dropped(t) != "" {
		t.Fatalf("the merge was lost for good: %q", w.Dropped(t))
	}
}

// Clause (b) of the same ruling: every way an event's repository cannot be
// identified is an ordinary answer, and none of the three is a reason to abort
// the pass -- a directory that is gone, one that is no repository, and one
// that is a repository nobody registered.
func TestAnUnresolvableRepositoryKeepsTheEvent(t *testing.T) {
	w := mergeWorld(t)
	gone := filepath.Join(w.Root, "gone")
	plain := filepath.Join(w.Root, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	foreign := filepath.Join(w.Root, "foreign")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	initRepo(t, foreign)
	gitRun(t, foreign, "add", "--all")
	gitCommit(t, foreign, "elsewhere")

	event := w.Event(t, "project/one")
	var events []maintenance.MergeEvent
	for _, repo := range []string{gone, plain, foreign} {
		aside := event
		aside.Repo = repo
		events = append(events, aside)
	}
	w.Events(t, events...)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 0 {
		t.Fatalf("a case was raised for an unresolvable repository: %+v", report.Cases)
	}
	if w.Dropped(t) != "" {
		t.Fatalf("an event nobody could place was spent: %q", w.Dropped(t))
	}
}

// Two areas in one checkout get one merge case between them, not two: both
// would carry the same evidence, and the same merge raised twice for two wikis
// is noise rather than a second finding.
func TestOneRepositoryRaisesTheMergeForTheFirstAreaOnly(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	w.addArea(t, mergeArea("project/one"))
	inner := mergeArea("project/one/inner")
	// The inner area lies inside the outer one's checkout and brings no `.git`
	// of its own, which is exactly how two areas come to share a repository.
	inner.Git = false
	inner.Review = ""
	w.addArea(t, inner)
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "one merge, two areas")
	w.Events(t, w.Event(t, "project/one"))

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("want one case for one repository, got %d: %+v", len(report.Cases), report.Cases)
	}
	if report.Cases[0].Area != "project/one" {
		t.Fatalf("the merge landed on %q rather than the first area", report.Cases[0].Area)
	}
}

// A page that says nothing about its realization promises nothing and is no
// candidate -- and it is read past, not tripped over.
func TestAMergePassesOverAPageWithoutARealization(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	area := mergeArea("project/one")
	area.Cites["docs/wiki/plain.md"] = []string{"src/a.go"}
	w.addArea(t, area)
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "build the promised thing")
	w.Events(t, w.Event(t, "project/one"))

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 || report.Cases[0].Target != "a.md" {
		t.Fatalf("want the one case on a.md, got %+v", report.Cases)
	}
}

// "First area wins" counts only areas that can take a merge: one without a
// declaration, and one without a wiki directory, are passed over even when
// they come first in the registry and share the repository.
func TestOneRepositorySkipsTheAreasThatCannotTakeTheMerge(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	undeclared := mergeArea("project/one/undeclared")
	undeclared.Git, undeclared.Review, undeclared.NoManifest = false, "", true
	w.addArea(t, undeclared)
	wikiless := areaOptions{Scope: "project/one/wikiless", Files: map[string]string{"notes.md": "# n\n"}}
	w.addArea(t, wikiless)
	w.addArea(t, mergeArea("project/one"))
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "one merge, three areas")
	w.Events(t, w.Event(t, "project/one"))

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 || report.Cases[0].Area != "project/one" {
		t.Fatalf("want the one case on project/one, got %+v", report.Cases)
	}
}

// The event log is the one read of this trigger that may take the pass down:
// bytes that are no text mean the log cannot be read at all, and answering
// "no merge ever happened" would be the worse lie.
func TestReconcileRefusesAnUnreadableEventLog(t *testing.T) {
	w := mergeWorld(t)
	path := filepath.Join(w.StateDir, "maintenance", "merge-events.tsv")
	if err := os.WriteFile(path, []byte{0xff, 0xfe, 0xfd}, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile read a log that is not text")
	}
}

// A drop that cannot be written is the opposite of the two clauses above: the
// case is on disk and the event is not spent, so the pass has to say so rather
// than report a success the next run will contradict.
func TestReconcileCarriesAFailedDrop(t *testing.T) {
	w := mergeWorld(t)
	blocked := filepath.Join(w.StateDir, "maintenance", "merge-events.done.tsv")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile reported success although the event stays unspent")
	}
}

// A wiki page that cannot be read takes the pass down here too.
//
// The fixture is narrow because it has to be: the derivation index walks the
// very same pages before this does, so any page that is unreadable from the
// start refuses there first. The one way to break the second walk and not the
// first is to make the offending entry *during* the pass -- a review centre
// inside the wiki, under a name ending in `.md`, is a directory the source
// case creates and the candidate walk then tries to read as a page.
func TestReconcileCarriesAnUnreadableCandidate(t *testing.T) {
	requireGit(t)
	w := newWorld(t)
	area := mergeArea("project/one")
	area.Review = "docs/wiki/95 Prüfzentrum.md"
	w.addArea(t, area)
	w.Change(t, "project/one", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	w.CommitFile(t, "project/one", "src/b.go", "package b\n", "second")
	w.Events(t, w.Event(t, "project/one"))

	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile read a directory as a wiki page and said nothing")
	}
}

// A case that cannot be written takes the pass down, the way a source case
// does. Reached through the merge trigger because nothing changed here: with
// no changed source the case directory is never made from the other side.
func TestReconcileCarriesAFailedMergeCase(t *testing.T) {
	w := mergeWorld(t)
	root, err := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if err != nil {
		t.Fatalf("ReviewRoot: %v", err)
	}
	blocked := filepath.Join(root, search.CollectionName("project/one"))
	if err := os.MkdirAll(filepath.Dir(blocked), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// A file where the scope's directory belongs: every case of this area has
	// to be made below it, and none can be.
	if err := os.WriteFile(blocked, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile reported success although no case could be written")
	}
}
