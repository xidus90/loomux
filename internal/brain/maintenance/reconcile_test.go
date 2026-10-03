package maintenance_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// reviewLayout is the review centre every world below declares. The space and
// the umlaut are not decoration: the value is joined onto the area's path and
// then walked, and a quoting or an encoding that dropped either would be found
// nowhere else in this file.
const reviewLayout = "95 Prüfzentrum"

// sourceArea is the ordinary area of this file: one source, one wiki page
// citing it, and the review centre.
func sourceArea(scope string) areaOptions {
	return areaOptions{
		Scope:  scope,
		Review: reviewLayout,
		Wiki:   "docs/wiki",
		Files:  map[string]string{"src/a.go": "package a\n", "docs/wiki/a.md": ""},
		Cites:  map[string][]string{"docs/wiki/a.md": {"src/a.go"}},
	}
}

// changedSource is the world every case-raising test stands on: the area
// above, with its source rewritten so the next scan finds it moved.
func changedSource(t *testing.T, opts areaOptions) *world {
	t.Helper()
	w := newWorld(t)
	w.addArea(t, opts)
	w.Change(t, opts.Scope, "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	return w
}

// caseDirOf is the directory one raised case owns, so a test can look at the
// files lying beside it.
func caseDirOf(w *world, c maintenance.Case) string {
	root, err := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if err != nil {
		panic(err)
	}
	return maintenance.CaseDir(root, search.CollectionName(c.Area), c.ID)
}

func mustReconcile(t *testing.T, w *world, now time.Time) maintenance.Report {
	t.Helper()
	report, err := maintenance.Reconcile(w.Areas, w.Lookup(), now)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	return report
}

// A vault with no area at all has no review centre either, and that is the
// same refusal as a vault whose areas declare none. The task brief claims an
// empty report and no error here; the reference has no early return
// (`reconcile.py:196`), and inventing one would let `reindex` walk on in a
// vault that has nowhere to put a case.
func TestReconcileOnAnEmptyVault(t *testing.T) {
	w := newWorld(t)
	_, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now())
	if !errors.Is(err, maintenance.ErrNoReviewCentre) {
		t.Fatalf("err = %v, want ErrNoReviewCentre", err)
	}
}

// Declares no area a review centre, that is ErrNoReviewCentre -- and the
// caller has to recognise it with errors.Is, because reindex treats it
// differently from every other failure.
func TestReconcileWithoutAReviewCentre(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{
		Scope: "project/a",
		Wiki:  "docs/wiki",
		Files: map[string]string{"docs/wiki/a.md": "# A\n"},
		// No Review: that is the state under test.
	})
	_, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now())
	if !errors.Is(err, maintenance.ErrNoReviewCentre) {
		t.Fatalf("err = %v, want ErrNoReviewCentre", err)
	}
}

// Two review centres are none. The refusal names both places, and it is not
// ErrNoReviewCentre: reindex may walk around a vault that has no gate, never
// around one whose gate is ambiguous.
func TestReconcileWithTwoReviewCentres(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", Review: "95 Review", Files: map[string]string{"a.md": "# A\n"}})
	w.addArea(t, areaOptions{Scope: "project/b", Review: "96 Review", Files: map[string]string{"b.md": "# B\n"}})
	_, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now())
	if err == nil {
		t.Fatal("Reconcile accepted two review centres")
	}
	if errors.Is(err, maintenance.ErrNoReviewCentre) {
		t.Fatalf("err = %v, want a refusal that is not ErrNoReviewCentre", err)
	}
	for _, want := range []string{"95 Review", "96 Review"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("err = %v, want it to name %q", err, want)
		}
	}
}

// A review centre reaching out of its area is refused, because it is the
// ground of the write barrier's one exemption: a value pointing elsewhere
// would make every file named proposal.md on the disk writable.
func TestReconcileRefusesAReviewCentreOutsideItsArea(t *testing.T) {
	for _, declared := range []string{"../aside", "/aside", `\aside`, "C:/aside"} {
		t.Run(declared, func(t *testing.T) {
			// A drive letter is a place of its own only where paths have
			// drives. On POSIX `C:/aside` is an ordinary relative name with a
			// colon in it, and pathlib reads it the same way there -- so the
			// refusal under test does not exist on that side.
			if declared == "C:/aside" && filepath.VolumeName(declared) == "" {
				t.Skip("no drive letters on this platform")
			}
			w := newWorld(t)
			w.addArea(t, areaOptions{Scope: "project/a", Review: declared,
				Files: map[string]string{"a.md": "# A\n"}})
			_, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now())
			if err == nil || !strings.Contains(err.Error(), "must stay inside the area") {
				t.Fatalf("err = %v, want a refusal naming the containment", err)
			}
		})
	}
}

// A changed source raises one case for the page derived from it, and the two
// files a person decides on lie beside it: the case itself and the package the
// decision may quote from.
func TestReconcileRaisesACaseForAChangedSource(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want 1", len(report.Cases))
	}
	raised := report.Cases[0]
	if raised.Target != "a.md" || raised.Area != "project/a" {
		t.Fatalf("case is about %q of %q, want a.md of project/a", raised.Target, raised.Area)
	}
	if raised.State != "source_changed" || raised.Trigger != "source_change" || raised.Weight != "change" {
		t.Fatalf("state, trigger, weight = %q, %q, %q", raised.State, raised.Trigger, raised.Weight)
	}
	if len(raised.Sources) != 1 {
		t.Fatalf("case names %d sources, want 1", len(raised.Sources))
	}
	if !raised.Created.Equal(w.Now()) {
		t.Fatalf("created = %v, want %v", raised.Created, w.Now())
	}
	if report.Checked != 2 || report.Hashed != 2 {
		t.Fatalf("checked, hashed = %d, %d; want 2, 2", report.Checked, report.Hashed)
	}
	if stamp, ok, err := search.ReadLastRun(w.StateDir); err != nil || !ok || !stamp.Equal(w.Now()) {
		t.Fatalf("last run = %v, %v, %v; want %v", stamp, ok, err, w.Now())
	}
	directory := caseDirOf(w, raised)
	for _, name := range []string{"case.toml", "package.md"} {
		if _, err := os.Stat(filepath.Join(directory, name)); err != nil {
			t.Fatalf("%s is missing beside the case: %v", name, err)
		}
	}
	// The package is what a proposal may quote, so the diff has to be in it.
	text, err := os.ReadFile(filepath.Join(directory, "package.md"))
	if err != nil {
		t.Fatalf("ReadFile package.md: %v", err)
	}
	for _, want := range []string{"## D1 — Diff,", "+func B() int { return 1 }", "## W1 — Wiki,", "## Q1 — Quelle,"} {
		if !strings.Contains(string(text), want) {
			t.Fatalf("package.md does not carry %q:\n%s", want, text)
		}
	}
}

// The register a case was raised against stays unadvanced, so a second pass
// finds the same change again. That is what makes two runs over one corpus
// agree, and what a stat cache remembering the changed state would break.
func TestReconcileLeavesAStandingCaseAlone(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	first := mustReconcile(t, w, w.Now())
	if len(first.Cases) != 1 {
		t.Fatalf("first run raised %d cases, want 1", len(first.Cases))
	}
	standing, err := os.ReadFile(filepath.Join(caseDirOf(w, first.Cases[0]), "case.toml"))
	if err != nil {
		t.Fatalf("ReadFile case.toml: %v", err)
	}

	later := w.Now().Add(24 * time.Hour)
	second := mustReconcile(t, w, later)
	if len(second.Cases) != 1 {
		t.Fatalf("second run raised %d cases, want 1", len(second.Cases))
	}
	if second.Cases[0].ID != first.Cases[0].ID {
		t.Fatalf("id moved: %q -> %q", first.Cases[0].ID, second.Cases[0].ID)
	}
	if !second.Cases[0].Created.Equal(first.Cases[0].Created) {
		t.Fatalf("created moved: %v -> %v", first.Cases[0].Created, second.Cases[0].Created)
	}
	again, err := os.ReadFile(filepath.Join(caseDirOf(w, second.Cases[0]), "case.toml"))
	if err != nil {
		t.Fatalf("ReadFile case.toml: %v", err)
	}
	if string(again) != string(standing) {
		t.Fatalf("the standing case was rewritten:\n%s\nwant\n%s", again, standing)
	}
}

// A standing case is left byte for byte, even one a reviewer has touched by
// hand: the pass writes only where the privacy switch moved, and a case file
// rendered anew would drop the reviewer's comment for no change at all.
func TestReconcileLeavesAHandEditedStandingCaseAlone(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	first := mustReconcile(t, w, w.Now())
	path := filepath.Join(caseDirOf(w, first.Cases[0]), "case.toml")
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile case.toml: %v", err)
	}
	edited := "# looked at on Monday\n" + string(written)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("WriteFile case.toml: %v", err)
	}

	mustReconcile(t, w, w.Now().Add(24*time.Hour))
	again, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile case.toml: %v", err)
	}
	if string(again) != edited {
		t.Fatalf("the standing case was rewritten:\n%s\nwant\n%s", again, edited)
	}
}

// A folder of the review centre without a `case.toml` is a reviewer's own and
// no broken case: nothing is reported for it.
func TestReconcileIgnoresAReviewFolderWithoutACase(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	root, err := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if err != nil {
		t.Fatalf("ReviewRoot: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, search.CollectionName("project/a"), "notes"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	report := mustReconcile(t, w, w.Now())
	if len(report.Unreadable) != 0 {
		t.Fatalf("unreadable = %v, want none for a folder without a case", report.Unreadable)
	}
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want 1", len(report.Cases))
	}
}

// Two pages of one area each get their own case, and both stay on disk: the
// case standing for one page is not the case of the other, however early it
// sorts in the review centre.
func TestReconcileKeepsTheCasesOfTwoPages(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{
		Scope:  "project/a",
		Review: reviewLayout,
		Wiki:   "docs/wiki",
		Files: map[string]string{
			"src/a.go": "package a\n", "src/b.go": "package b\n",
			"docs/wiki/a.md": "", "docs/wiki/b.md": "",
		},
		Cites: map[string][]string{"docs/wiki/a.md": {"src/a.go"}, "docs/wiki/b.md": {"src/b.go"}},
	})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc A() int { return 1 }\n")
	w.Change(t, "project/a", "src/b.go", "package b\n\nfunc B() int { return 2 }\n")

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 2 {
		t.Fatalf("raised %d cases, want one per page", len(report.Cases))
	}
	for _, landed := range report.Cases {
		read, err := maintenance.ReadCase(filepath.Join(caseDirOf(w, landed), "case.toml"))
		if err != nil {
			t.Fatalf("the case of %s is gone from disk: %v", landed.Target, err)
		}
		if read.Target != landed.Target {
			t.Fatalf("the case of %s reads as one about %s", landed.Target, read.Target)
		}
	}
}

// A case opened for the first time supersedes nothing, wherever the pass was
// started from: a `proposal.md` in the working directory -- a reviewer who ran
// reconcile from inside a case folder -- is no earlier proposal of this one.
func TestReconcileTakesNoProposalFromTheWorkingDirectory(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	elsewhere := t.TempDir()
	if err := os.WriteFile(filepath.Join(elsewhere, "proposal.md"), []byte("not ours\n"), 0o644); err != nil {
		t.Fatalf("WriteFile proposal.md: %v", err)
	}
	t.Chdir(elsewhere)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 || report.Cases[0].SupersededProposal != "" {
		t.Fatalf("cases = %+v, want one that supersedes nothing", report.Cases)
	}
	if _, err := os.Stat(filepath.Join(caseDirOf(w, report.Cases[0]), "superseded-proposal.md")); err == nil {
		t.Fatal("a proposal from the working directory was carried into the case")
	}
}

// A proposal already written for a case is not thrown away when the sources
// move on: it is worthless as a decision, but it is work somebody did, so it
// moves aside under a name of its own and the new case records it.
func TestReconcileCarriesAProposalAside(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	first := mustReconcile(t, w, w.Now())
	proposal := filepath.Join(caseDirOf(w, first.Cases[0]), "proposal.md")
	if err := os.WriteFile(proposal, []byte("the earlier proposal\n"), 0o644); err != nil {
		t.Fatalf("WriteFile proposal.md: %v", err)
	}

	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc C() int { return 22 }\n")
	later := w.Now().Add(24 * time.Hour)
	second := mustReconcile(t, w, later)
	if len(second.Cases) != 1 {
		t.Fatalf("second run raised %d cases, want 1", len(second.Cases))
	}
	if second.Cases[0].ID == first.Cases[0].ID {
		t.Fatalf("the case kept its id %q although its sources moved", first.Cases[0].ID)
	}
	if second.Cases[0].SupersededProposal != "superseded-proposal.md" {
		t.Fatalf("superseded_proposal = %q", second.Cases[0].SupersededProposal)
	}
	carried, err := os.ReadFile(filepath.Join(caseDirOf(w, second.Cases[0]), "superseded-proposal.md"))
	if err != nil {
		t.Fatalf("ReadFile superseded-proposal.md: %v", err)
	}
	if string(carried) != "the earlier proposal\n" {
		t.Fatalf("carried %q", carried)
	}
	if _, err := os.Stat(filepath.Dir(proposal)); err == nil {
		t.Fatal("the discarded case directory is still there")
	}
}

// Twice on one day the new case lands in the very directory the old one had,
// because the id carries the day and nothing else moved. The proposal still
// has to go: it belongs to a source state that no longer exists.
func TestReconcileSupersedesWithinOneDay(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	first := mustReconcile(t, w, w.Now())
	directory := caseDirOf(w, first.Cases[0])
	if err := os.WriteFile(filepath.Join(directory, "proposal.md"), []byte("today's proposal\n"), 0o644); err != nil {
		t.Fatalf("WriteFile proposal.md: %v", err)
	}

	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc D() int { return 333 }\n")
	second := mustReconcile(t, w, w.Now())
	if second.Cases[0].ID != first.Cases[0].ID {
		t.Fatalf("id moved within one day: %q -> %q", first.Cases[0].ID, second.Cases[0].ID)
	}
	if _, err := os.Stat(filepath.Join(directory, "proposal.md")); err == nil {
		t.Fatal("the spent proposal is still lying beside the case")
	}
	carried, err := os.ReadFile(filepath.Join(directory, "superseded-proposal.md"))
	if err != nil || string(carried) != "today's proposal\n" {
		t.Fatalf("superseded-proposal.md = %q, %v", carried, err)
	}
}

// A second rebuild finds no proposal but the record of an earlier one, and
// that record stays: it is the only trace left that a person once worked here.
func TestReconcileKeepsAnEarlierSupersededProposal(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	first := mustReconcile(t, w, w.Now())
	directory := caseDirOf(w, first.Cases[0])
	if err := os.WriteFile(filepath.Join(directory, "superseded-proposal.md"), []byte("from before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile superseded-proposal.md: %v", err)
	}

	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc E() int { return 4444 }\n")
	second := mustReconcile(t, w, w.Now().Add(24*time.Hour))
	if second.Cases[0].SupersededProposal != "superseded-proposal.md" {
		t.Fatalf("superseded_proposal = %q", second.Cases[0].SupersededProposal)
	}
	carried, err := os.ReadFile(filepath.Join(caseDirOf(w, second.Cases[0]), "superseded-proposal.md"))
	if err != nil || string(carried) != "from before\n" {
		t.Fatalf("superseded-proposal.md = %q, %v", carried, err)
	}
}

// A discarded case that carried nothing leaves nothing behind either: the new
// case records no superseded proposal.
func TestReconcileRecordsNoSupersededProposalWhereThereWasNone(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	mustReconcile(t, w, w.Now())
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc F() int { return 55555 }\n")
	second := mustReconcile(t, w, w.Now().Add(24*time.Hour))
	if second.Cases[0].SupersededProposal != "" {
		t.Fatalf("superseded_proposal = %q, want none", second.Cases[0].SupersededProposal)
	}
}

// An area registered before it wrote a declaration carries nothing to compare,
// which is normal and never a reason to stop reconciling the rest.
func TestReconcileSkipsAnAreaWithoutAManifest(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, sourceArea("project/a"))
	// A Markdown source among them: without a declaration the walk would fall
	// back to `**/*.md`, so a Go file alone could not tell a skipped area from
	// one walked with the default include.
	w.addArea(t, areaOptions{Scope: "project/b", NoManifest: true,
		Files: map[string]string{"src/b.go": "package b\n", "notes.md": "# b\n"}})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want the one of the declared area", len(report.Cases))
	}
	if report.Checked != 2 {
		t.Fatalf("checked = %d, want only the two sources of the declared area", report.Checked)
	}
}

// A declaration that cannot be read is a different matter: reconciling on a
// guess about which sources an area holds would raise cases for the wrong
// files, so the pass stops and names the file.
func TestReconcileFailsOnABrokenManifest(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, sourceArea("project/a"))
	w.addArea(t, areaOptions{Scope: "project/b", BrokenManifest: true,
		Files: map[string]string{"src/b.go": "package b\n"}})
	_, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now())
	if err == nil {
		t.Fatal("Reconcile walked past a broken declaration")
	}
	if !strings.Contains(err.Error(), filepath.Join("project", "b")) {
		t.Fatalf("err = %v, want it to name the file it could not read", err)
	}
}

// An area without a declaration raises no case either, even where its wiki
// cites a source that changed elsewhere: `_cases` skips it the way `_scan`
// does, and a case landed for it would carry no privacy mode to be read by.
func TestReconcileRaisesNoCaseForAnUndeclaredAreasWiki(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", Review: reviewLayout,
		Files: map[string]string{"src/a.go": "package a\n"}})
	w.addArea(t, areaOptions{Scope: "project/b", Wiki: "docs/wiki", NoManifest: true,
		Files: map[string]string{"docs/wiki/b.md": pageCiting(w, "project/a", "src/a.go")}})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 0 {
		t.Fatalf("raised %d cases for an area without a declaration", len(report.Cases))
	}
}

// An area that names no wiki derives nothing from its sources, so a change in
// one of them has no page to be about.
func TestReconcileRaisesNoCaseWithoutAWiki(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", Review: reviewLayout,
		Files: map[string]string{"src/a.go": "package a\n"}})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 0 {
		t.Fatalf("raised %d cases for an area without a wiki", len(report.Cases))
	}
	if report.Checked != 1 {
		t.Fatalf("checked = %d, want 1", report.Checked)
	}
}

// A registered wiki directory that is not there is the same nothing: the area
// is walked for its sources and raises no case.
func TestReconcileRaisesNoCaseForAMissingWikiDirectory(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", Review: reviewLayout, Wiki: "docs/wiki",
		Files: map[string]string{"src/a.go": "package a\n"}})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")
	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 0 {
		t.Fatalf("raised %d cases for a wiki that is not there", len(report.Cases))
	}
}

// A closed area gets a case a person decides by hand: with the model off,
// nobody is asked, so no proposal lies beside it, and the case says so in as
// many words.
func TestReconcileMarksALocalOnlyCaseManual(t *testing.T) {
	area := sourceArea("project/a")
	area.PrivacyMode = "local_only"
	w := changedSource(t, area)
	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want 1", len(report.Cases))
	}
	raised := report.Cases[0]
	if !raised.Manual || !raised.LocalOnly {
		t.Fatalf("manual, local_only = %v, %v; want both true", raised.Manual, raised.LocalOnly)
	}
	// Verbatim: the note goes into case.toml, and the Python side reads the
	// same file. `_LOCAL_ONLY_NOTE`, `reconcile.py:135`.
	const want = "manual review: this area is local_only, so no skill path is offered (spec 5)"
	if raised.Note != want {
		t.Fatalf("note = %q, want %q", raised.Note, want)
	}
	if _, err := os.Stat(filepath.Join(caseDirOf(w, raised), "proposal.md")); err == nil {
		t.Fatal("a proposal lies beside a case nobody was asked for, the model being off")
	}
}

// An open area's case carries neither flag and no note: there is nothing to
// explain, and a note nobody needs would be prose in a file code reads.
func TestReconcileLeavesAnOpenAreaCaseUnmarked(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	raised := mustReconcile(t, w, w.Now()).Cases[0]
	if raised.Manual || raised.LocalOnly || raised.Note != "" {
		t.Fatalf("manual, local_only, note = %v, %v, %q", raised.Manual, raised.LocalOnly, raised.Note)
	}
}

// local_only follows today's declaration, even on a case that keeps standing:
// it is the switch that holds a source diff away from a cloud model, and it
// must not hang on the state the area was in when the case was opened.
func TestAStandingCaseFollowsTheCurrentPrivacyMode(t *testing.T) {
	area := sourceArea("project/a")
	w := changedSource(t, area)
	first := mustReconcile(t, w, w.Now())
	if first.Cases[0].LocalOnly {
		t.Fatal("the case was opened as local_only although the area was open")
	}

	// The area closes after the case was opened.
	closed := area
	closed.PrivacyMode = "local_only"
	w.Change(t, "project/a", ".loomux/config.toml", declaration("project/a",
		[]string{"**/*.go", "**/*.md", "**/*.txt"}, closed))

	second := mustReconcile(t, w, w.Now().Add(24*time.Hour))
	if second.Cases[0].ID != first.Cases[0].ID {
		t.Fatalf("the case was rebuilt: %q -> %q", first.Cases[0].ID, second.Cases[0].ID)
	}
	if !second.Cases[0].LocalOnly {
		t.Fatal("local_only stayed false after the area closed")
	}
	// note and manual are deliberately not derived anew: they record what
	// happened when the case was opened, and no second question was asked.
	if second.Cases[0].Manual || second.Cases[0].Note != "" {
		t.Fatalf("manual, note = %v, %q; want what the case was opened with",
			second.Cases[0].Manual, second.Cases[0].Note)
	}
	reread, err := maintenance.ReadCase(filepath.Join(caseDirOf(w, second.Cases[0]), "case.toml"))
	if err != nil {
		t.Fatalf("ReadCase: %v", err)
	}
	if !reread.LocalOnly {
		t.Fatal("the flag was carried in the report but never written to the file")
	}
}

// A case file that cannot be read is neither fatal nor silent: the pass goes
// on, a case opens beside the broken one, and the caller is handed its path so
// the second case is at least explicable.
func TestReconcileReportsAnUnreadableCase(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	first := mustReconcile(t, w, w.Now())
	broken := filepath.Join(caseDirOf(w, first.Cases[0]), "case.toml")
	if err := os.WriteFile(broken, []byte("state = \"nonsense\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile case.toml: %v", err)
	}

	second := mustReconcile(t, w, w.Now().Add(24*time.Hour))
	if len(second.Unreadable) != 1 || second.Unreadable[0] != broken {
		t.Fatalf("unreadable = %v, want %q", second.Unreadable, broken)
	}
	if len(second.Cases) != 1 {
		t.Fatalf("raised %d cases, want one beside the broken one", len(second.Cases))
	}
	if second.Cases[0].ID == first.Cases[0].ID {
		t.Fatal("the broken case was taken for a standing one")
	}
}

// One page, one case -- even when two of its sources moved at once. Three
// cases for one page would mean three proposals contradicting each other.
func TestReconcileRaisesOneCasePerPage(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{
		Scope:  "project/a",
		Review: reviewLayout,
		Wiki:   "docs/wiki",
		Files: map[string]string{
			"src/a.go": "package a\n", "src/b.go": "package b\n", "docs/wiki/a.md": "",
		},
		Cites: map[string][]string{"docs/wiki/a.md": {"src/a.go", "src/b.go"}},
	})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc A() int { return 1 }\n")
	w.Change(t, "project/a", "src/b.go", "package b\n\nfunc B() int { return 2 }\n")

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want 1", len(report.Cases))
	}
	if len(report.Cases[0].Sources) != 2 {
		t.Fatalf("the case names %d sources, want 2", len(report.Cases[0].Sources))
	}
	// Sorted by doc id, so two runs render case.toml byte for byte alike.
	if report.Cases[0].Sources[0].DocID > report.Cases[0].Sources[1].DocID {
		t.Fatalf("sources are out of order: %v", report.Cases[0].Sources)
	}
}

// A source of one area may well be cited by the wiki of another, so the
// changed sources are collected across the whole vault before any wiki is
// asked what it derives from them.
func TestReconcileMatchesASourceAgainstAnotherAreasWiki(t *testing.T) {
	w := newWorld(t)
	producer := areaOptions{Scope: "project/a", Review: reviewLayout,
		Files: map[string]string{"src/a.go": "package a\n"}}
	w.addArea(t, producer)
	// The second area's page cites the first area's source. addArea hands the
	// ids out per area, so the page is written by hand here with the id the
	// producer's register carries.
	w.addArea(t, areaOptions{Scope: "project/b", Wiki: "docs/wiki",
		Files: map[string]string{"docs/wiki/b.md": pageCiting(w, "project/a", "src/a.go")}})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want 1", len(report.Cases))
	}
	if report.Cases[0].Area != "project/b" || report.Cases[0].Target != "b.md" {
		t.Fatalf("case is about %q of %q", report.Cases[0].Target, report.Cases[0].Area)
	}
}

// ReviewRoot is the thin composition every caller that only wants to find a
// case uses; it answers the same place reconcile writes to.
func TestReviewRootFindsTheDeclaredCentre(t *testing.T) {
	w := newWorld(t)
	area, _ := w.addArea(t, sourceArea("project/a"))
	root, err := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if err != nil {
		t.Fatalf("ReviewRoot: %v", err)
	}
	if want := filepath.Join(area.Path, reviewLayout); root != want {
		t.Fatalf("ReviewRoot = %q, want %q", root, want)
	}
}

// ReviewRoot carries the refusal of a broken declaration through rather than
// answering a place: `brain case` asks it first, and a guess here would put a
// case file somewhere nobody looks.
func TestReviewRootCarriesABrokenManifestThrough(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, areaOptions{Scope: "project/a", BrokenManifest: true,
		Files: map[string]string{"a.md": "# A\n"}})
	// The declaration's own refusal, naming its file -- not the "no review
	// centre" that an unread declaration would otherwise amount to.
	_, err := maintenance.ReviewRoot(w.Areas, w.Lookup())
	if err == nil || errors.Is(err, maintenance.ErrNoReviewCentre) ||
		!strings.Contains(err.Error(), filepath.Join("project", "a")) {
		t.Fatalf("err = %v, want the unreadable declaration named", err)
	}
}

// Every pass moves the stamp to its own time, and a pass at the time already
// recorded leaves the file as it was -- its bytes and its timestamp both, the
// way `write_if_changed` answers an unchanged text.
func TestReconcileWritesTheLastRunOnlyWhenItMoves(t *testing.T) {
	w := newWorld(t)
	w.addArea(t, sourceArea("project/a"))
	path := filepath.Join(w.StateDir, "maintenance", "last-run.txt")

	mustReconcile(t, w, w.Now())
	later := w.Now().Add(time.Hour)
	mustReconcile(t, w, later)
	if got, ok, err := search.ReadLastRun(w.StateDir); err != nil || !ok || !got.Equal(later) {
		t.Fatalf("ReadLastRun = %v, %v, %v; want the later pass %v", got, ok, err, later)
	}

	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	mustReconcile(t, w, later)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.ModTime().Equal(past) {
		t.Fatalf("an unchanged stamp was written again at %v", info.ModTime())
	}
}

// pageCiting is one wiki page naming a source of another area by the doc id
// that area's register gave it.
func pageCiting(w *world, scope, relative string) string {
	return "---\ntype: note\ntitle: b\nsources:\n  - resource: " + relative +
		"\n    doc_id: " + docIDOf(w, scope, relative) + "\n---\n\n# b\n\nWhat it says.\n"
}

// docIDOf reads one source's key out of the register the world wrote, rather
// than counting along with it: a second counter beside the world's would agree
// with it only until someone added a file to a fixture.
func docIDOf(w *world, scope, relative string) string {
	for _, area := range w.Areas {
		if area.Scope != scope {
			continue
		}
		register := filepath.Join(config.ManifestDir(area, w.StateDir), "_identities.tsv")
		data, err := os.ReadFile(register)
		if err != nil {
			panic(err)
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) >= 2 && fields[1] == relative {
				return fields[0]
			}
		}
	}
	panic("no doc id for " + scope + " " + relative)
}

// A register that cannot be read stops the pass rather than being taken for
// empty: an area without a single source reads as "all is well".
func TestReconcileFailsOnARegisterItCannotRead(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	register := filepath.Join(w.Areas[0].Path, "_identities.tsv")
	if err := os.Remove(register); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := os.MkdirAll(register, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile walked past a register it could not read")
	}
}

// The derivation index is rebuilt from the pages on every pass, and a wiki it
// cannot walk leaves it not knowing which page a source feeds. Silence there
// would read as "no page derives from this", which is the one answer it must
// not invent.
func TestReconcileFailsOnAWikiItCannotRead(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	if err := os.MkdirAll(filepath.Join(w.Areas[0].WikiPath, "a folder.md"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile walked past a wiki it could not read")
	}
}

// A case that cannot be landed stops the pass: the alternative is a report
// that names fewer cases than the vault has, which nobody can tell from a
// vault in order.
func TestReconcileCarriesAFailedLandingOut(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	root := filepath.Join(w.Areas[0].Path, reviewLayout)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// A regular file where the scope's folder belongs.
	if err := os.WriteFile(filepath.Join(root, search.CollectionName("project/a")), nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile reported a pass in which no case could be landed")
	}
}

// A stamp that cannot be written down is a failed pass: the catch-up of task
// 14 reads it, and a pass reporting success without it would be skipped for a
// day on the strength of a file nobody wrote.
func TestReconcileFailsWhenTheStampCannotBeWritten(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	// A directory under the stamp's own name, so everything else of the pass
	// still has its place and only the last step has nowhere to go.
	if err := os.MkdirAll(filepath.Join(w.StateDir, "maintenance", "last-run.txt"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now()); err == nil {
		t.Fatal("Reconcile reported a pass it could not record")
	}
}

// A read-only area keeps every artefact in the state directory, the
// declaration among them -- and the pass has to look for it there.
//
// The area under test carries a declaration **nowhere else**, which is what
// makes the case say something: reading `area.Path` instead of
// `config.ManifestDir` would find nothing, skip the area without a word,
// and report a vault in order. That failure is silent in both directions, and
// nothing else in this suite saw it.
func TestReconcileReadsAReadOnlyAreaDeclaredInTheStateDirectory(t *testing.T) {
	w := newWorld(t)
	closed := sourceArea("project/a")
	closed.ReadOnly = true
	// The review centre belongs to a writable area: a read-only area is no
	// place to put case files.
	closed.Review = ""
	w.addArea(t, closed)
	w.addArea(t, areaOptions{Scope: "project/review", Review: reviewLayout,
		Files: map[string]string{"readme.md": "# R\n"}})
	w.Change(t, "project/a", "src/a.go", "package a\n\nfunc B() int { return 1 }\n")

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 || report.Cases[0].Area != "project/a" {
		t.Fatalf("raised %d cases, want one of project/a", len(report.Cases))
	}
	// Without this the case above would pass for the wrong reason.
	if _, err := os.Stat(filepath.Join(w.Areas[0].Path, ".loomux", "config.toml")); err == nil {
		t.Fatal("the read-only area carries a declaration in its own tree as well")
	}
}

// A standing case keeps every proposal already written for it. The task says
// so in as many words -- „er behält created, seine id und jeden Vorschlag
// daneben" -- and neither the brief's test nor the one above ever puts a
// proposal there, so the third of the three went unproven.
func TestAStandingCaseKeepsItsProposal(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	first := mustReconcile(t, w, w.Now())
	directory := caseDirOf(w, first.Cases[0])
	proposal := filepath.Join(directory, "proposal.md")
	if err := os.WriteFile(proposal, []byte("what the reviewer may accept\n"), 0o644); err != nil {
		t.Fatalf("WriteFile proposal.md: %v", err)
	}

	// Nothing moved, a day later.
	second := mustReconcile(t, w, w.Now().Add(24*time.Hour))
	if second.Cases[0].ID != first.Cases[0].ID {
		t.Fatalf("the case was rebuilt: %q -> %q", first.Cases[0].ID, second.Cases[0].ID)
	}
	standing, err := os.ReadFile(proposal)
	if err != nil || string(standing) != "what the reviewer may accept\n" {
		t.Fatalf("proposal.md = %q, %v", standing, err)
	}
	if _, err := os.Stat(filepath.Join(directory, "superseded-proposal.md")); err == nil {
		t.Fatal("the proposal was carried aside although nothing moved")
	}
}

// A review centre behind a link that leaves the area is refused, and it is
// the barrier's own resolver that says so.
//
// Windows only, because the link that reaches this is a junction: Windows
// reports it as an irregular file rather than a link, so `EvalSymlinks` hands
// it back unchanged and only guard's resolver follows it. Measured on
// 2026-09-20: before this, the barrier withdrew its exemption for the very
// same value while this side accepted it, and a write landed outside the area.
func TestReconcileRefusesAReviewCentreBehindALinkOutOfTheArea(t *testing.T) {
	requireJunctions(t)
	w := changedSource(t, sourceArea("project/a"))
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	junction(t, filepath.Join(w.Areas[0].Path, reviewLayout), outside)

	_, err := maintenance.Reconcile(w.Areas, w.Lookup(), w.Now())
	if err == nil || !strings.Contains(err.Error(), "must stay inside the area") {
		t.Fatalf("err = %v, want the containment refusal", err)
	}
}

// The other half of the same question, so the refusal above is not simply
// "every link is refused": a link that stays inside the area is an ordinary
// place and carries its cases.
func TestReconcileAcceptsAReviewCentreBehindALinkInsideTheArea(t *testing.T) {
	requireJunctions(t)
	w := changedSource(t, sourceArea("project/a"))
	inside := filepath.Join(w.Areas[0].Path, "elsewhere")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	junction(t, filepath.Join(w.Areas[0].Path, reviewLayout), inside)

	report := mustReconcile(t, w, w.Now())
	if len(report.Cases) != 1 {
		t.Fatalf("raised %d cases, want 1", len(report.Cases))
	}
	// Through the link, so the case really lies in the place it points at.
	if _, err := os.Stat(filepath.Join(inside, search.CollectionName("project/a"),
		report.Cases[0].ID, "case.toml")); err != nil {
		t.Fatalf("the case did not land behind the link: %v", err)
	}
}

// requireJunctions stands in front of the two cases above for the reason
// requireGit stands in front of the repository ones: without the link the two
// would pass for a reason that has nothing to do with what they ask.
func requireJunctions(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("a junction is the Windows link this case is about")
	}
}

// junction makes a directory link the way a person would, with `mklink /J` --
// which, unlike a symbolic link, needs no privilege.
func junction(t *testing.T, link, target string) {
	t.Helper()
	command := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("mklink /J %s %s: %v\n%s", link, target, err, out)
	}
}

// A pass whose context has ended stops before it writes anything: no case,
// no stamp. Serve cancels the pass it carries when it shuts down, and a
// half-written pass would let the next serve find a stamp nobody earned.
func TestReconcileContextStopsBeforeItWritesWhenCancelled(t *testing.T) {
	w := changedSource(t, sourceArea("project/a"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := maintenance.ReconcileContext(ctx, w.Areas, w.Lookup(), w.Now()); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if _, ok, err := search.ReadLastRun(w.StateDir); ok || err != nil {
		t.Fatalf("a cancelled pass left a stamp (ok %v, err %v)", ok, err)
	}
}
