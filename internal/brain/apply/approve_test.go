package apply

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/evidence"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/vcs"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// An approval, ported from `approve` and `_apply` (apply.py:307-358,
// :730-811) with `_guard`, `_guard_sources`, `_refuse` and `_same_file`
// (:913-1062). Where a test carries over one of test_apply.py, its comment
// names it and its line.

const (
	appSources = "10 Rohquellen"
	appReview  = "95 Prüfzentrum"
	appTarget  = "topics/thema.md"
	appOld     = "Die Suchkette braucht 120 ms.\n"
	appNew     = "Die Suchkette braucht 80 ms.\n"
	appDiff    = "@@ -12,1 +12,1 @@\n-Die Suchkette braucht 120 ms.\n+Die Suchkette braucht 80 ms."
	appDocID   = "01DOC0"
	appHuman   = "human:cw"
	appClaim   = "B1"
)

var appNow = time.Date(2026, 8, 27, 9, 14, 0, 0, time.UTC)

// appVault is `_make_vault` (test_apply.py:179-204): a vault with one
// source, one page derived from it and two protocols, the source changed
// under it and a case raised by a real reconciliation, with a proposal
// beside the case that quotes the package's diff. Commits are faked unless
// a test says otherwise.
type appVault struct {
	base, root, scratch string
	areas               []config.Area
	lookup              config.ArtifactLookup
	dir                 string // the case directory
	calls               *[]commitCall
}

func appManifest(mode string) string {
	return "[area]\nscope = \"knowledge\"\n\n" +
		"[layout]\nwiki = \"" + testWiki + "\"\nreview = \"" + appReview + "\"\n\n" +
		"[index]\ninclude = [\"" + appSources + "/**/*.md\"]\n\n" +
		"[privacy]\nmode = \"" + mode + "\"\n"
}

func newAppVault(t *testing.T) appVault {
	t.Helper()
	return newAppVaultIn(t, "manual_cloud")
}

func newAppVaultIn(t *testing.T, mode string) appVault {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "vault")
	state := filepath.Join(base, "state")
	v := appVault{
		base: base, root: root, scratch: filepath.Join(state, "maintenance"),
		areas:  []config.Area{{Scope: "knowledge", Path: root, WikiPath: filepath.Join(root, testWiki)}},
		lookup: config.ArtifactLookup{Primary: state},
	}
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), appManifest(mode))
	relative := appSources + "/quelle.md"
	writeFile(t, v.source(), appOld)
	hash := hashOf(t, v.source())
	writeFile(t, v.register(), identity.RenderIdentities(map[string]identity.Identity{
		relative: {DocID: appDocID, Relative: relative, ContentHash: hash, Revision: 1},
	}))
	writeFile(t, v.page(), "---\ntype: Topic\ntitle: Thema\nsources:\n  - id: quelle\n"+
		"    resource: "+relative+"\n    doc_id: "+appDocID+"\n    content_hash: \""+hash+"\"\n"+
		"    revision: 1\n---\n\n"+appOld)
	writeFile(t, v.wiki("log.md"), "# Protokoll\n")
	writeFile(t, v.wiki("audit.md"), "# Wartungsprotokoll\n")
	v.reconcile(t)
	writeFile(t, v.source(), appNew)
	if report := v.reconcile(t); len(report.Cases) != 1 {
		t.Fatalf("reconcile raised %d cases, want 1", len(report.Cases))
	}
	matches, err := filepath.Glob(filepath.Join(root, appReview, "knowledge", "*", "case.toml"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("cases = %v, %v; want one", matches, err)
	}
	v.dir = filepath.Dir(matches[0])
	v.propose(t, v.quote(t), appDiff)
	v.calls = fakeCommits(t, landed("abc"))
	return v
}

func (v appVault) reconcile(t *testing.T) maintenance.Report {
	t.Helper()
	report, err := maintenance.Reconcile(v.areas, v.lookup, appNow)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	return report
}

func (v appVault) wiki(name string) string { return filepath.Join(v.root, testWiki, name) }
func (v appVault) page() string            { return v.wiki(filepath.FromSlash(appTarget)) }
func (v appVault) source() string          { return filepath.Join(v.root, appSources, "quelle.md") }
func (v appVault) register() string        { return filepath.Join(v.root, registerName) }
func (v appVault) casePath() string        { return filepath.Join(v.dir, "case.toml") }
func (v appVault) proposal() string        { return filepath.Join(v.dir, "proposal.md") }
func (v appVault) caseID() string          { return filepath.Base(v.dir) }
func (v appVault) caseRel() string         { return appReview + "/knowledge/" + v.caseID() }

func hashOf(t *testing.T, path string) string {
	t.Helper()
	hash, err := identity.ContentHash(path)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

// quote is `_quote`: the added line of the package's first diff segment.
func (v appVault) quote(t *testing.T) string {
	t.Helper()
	segments, err := evidence.ReadPackage(readFile(t, filepath.Join(v.dir, "package.md")))
	if err != nil {
		t.Fatal(err)
	}
	for _, segment := range segments {
		for _, line := range strings.Split(segment.Body, "\n") {
			if segment.Number == "D1" && strings.HasPrefix(line, "+Die Suchkette") {
				return line
			}
		}
	}
	t.Fatal("no quotable line in D1")
	return ""
}

// propose is `_proposal`: a one-claim proposal, a verbatim quote, then the
// diff it justifies.
func (v appVault) propose(t *testing.T, quote, diff string) {
	t.Helper()
	writeFile(t, v.proposal(), proposalText(v.caseID(), quote, diff))
}

func proposalText(caseID, quote, diff string) string {
	return "---\ncase: " + caseID + "\ncategory: fix\nconfidence: high\nuncertainty: none\n---\n\n" +
		"## " + appClaim + " — Die Suchkette braucht jetzt 80 ms\n\nevidence: D1\n\n" +
		"```\n" + quote + "\n```\n\n```diff\n" + diff + "\n```\n"
}

func (v appVault) readCase(t *testing.T) maintenance.Case {
	t.Helper()
	c, err := maintenance.ReadCase(v.casePath())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (v appVault) editCase(t *testing.T, edit func(*maintenance.Case)) {
	t.Helper()
	c := v.readCase(t)
	edit(&c)
	if _, err := maintenance.WriteCase(v.casePath(), c); err != nil {
		t.Fatal(err)
	}
}

// retarget is `_retarget`: another page under the standing case, and the
// proposal aimed at it.
func (v appVault) retarget(t *testing.T, text, diff string) {
	t.Helper()
	writeFile(t, v.page(), text)
	hash := hashOf(t, v.page())
	v.editCase(t, func(c *maintenance.Case) { c.TargetHash = hash })
	v.propose(t, v.quote(t), diff)
}

func (v appVault) options(change ...func(*Options)) Options {
	o := Options{Decision: "approve", Reviewer: appHuman, Now: appNow, Scratch: v.scratch, Lookup: v.lookup}
	for _, c := range change {
		c(&o)
	}
	return o
}

func (v appVault) run(change ...func(*Options)) (Result, error) {
	return Approve(v.casePath(), v.areas, v.options(change...))
}

func (v appVault) mustApprove(t *testing.T, change ...func(*Options)) Result {
	t.Helper()
	result, err := v.run(change...)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	return result
}

func amend(path string) func(*Options) { return func(o *Options) { o.Amend = path } }

// stopped asserts an error of kind T whose message carries phrase.
func stopped[T error](t *testing.T, err error, phrase string) T {
	t.Helper()
	var kind T
	if !errors.As(err, &kind) {
		t.Fatalf("err = %T %v, want %T", err, err, kind)
	}
	if !strings.Contains(err.Error(), phrase) {
		t.Fatalf("want %q in %q", phrase, err.Error())
	}
	return kind
}

// snapshot is every file below root with its bytes.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		files[path] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func sameSnapshot(t *testing.T, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("files: %d before, %d after", len(before), len(after))
	}
	for path, text := range before {
		if after[path] != text {
			t.Fatalf("%s changed:\n%s\nwas\n%s", path, after[path], text)
		}
	}
}

func count(t *testing.T, path, phrase string) int {
	t.Helper()
	return strings.Count(readFile(t, path), phrase)
}

func failOn(name string, broken error, next func(string, string) error) func(string, string) error {
	return func(path, text string) error {
		if filepath.Base(path) == name {
			return broken
		}
		return next(path, text)
	}
}

// --- Rule 1: the entrance -------------------------------------------------------

// test_an_unknown_decision_is_refused_before_anything_is_read
// (test_apply.py:341). `defer` is no decision here: the CLI handles it.
func TestApproveRefusesAnUnknownDecision(t *testing.T) {
	for _, decision := range []string{"vielleicht", "defer", ""} {
		_, err := Approve(filepath.Join(t.TempDir(), "gibt-es-nicht", "case.toml"), nil,
			Options{Decision: decision, Reviewer: appHuman, Scratch: "s"})
		e := stopped[*ApplyError](t, err, "decision must be one of approve, reject, found ")
		if e.Dirty != nil {
			t.Fatalf("dirty = %q", e.Dirty)
		}
	}
}

// test_a_reviewer_without_the_human_prefix_is_refused (test_apply.py:346)
// and test_an_empty_human_id_is_refused (:351).
func TestApproveRefusesAReviewerWhoIsNoHuman(t *testing.T) {
	for _, reviewer := range []string{"model:opus", "human:", "human: \t", "Human:cw"} {
		_, err := Approve("case.toml", nil, Options{Decision: "approve", Reviewer: reviewer, Scratch: "s"})
		stopped[*ApplyError](t, err, "reviewer must be human:<id> with a non-empty id, found ")
	}
}

func TestApproveRefusesAnEmptyScratchDirectory(t *testing.T) {
	_, err := Approve("case.toml", nil, Options{Decision: "approve", Reviewer: appHuman})
	stopped[*ApplyError](t, err, "no scratch directory")
}

// test_an_unreadable_case_file_is_refused (test_apply.py:690).
func TestApproveRefusesAnUnreadableCase(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, v.casePath(), "kein TOML = = =\n")
	_, err := v.run()
	if e := stopped[*ApplyError](t, err, "case.toml"); e.Dirty != nil || errors.Unwrap(err) == nil {
		t.Fatalf("err = %+v", e)
	}
}

// test_a_target_with_a_newline_is_refused (test_apply.py:945) and
// test_a_target_with_a_newline_is_refused_on_a_rejection_too (:954).
func TestApproveChecksTheTargetBeforeEitherDecision(t *testing.T) {
	for _, decision := range Decisions {
		v := newAppVault(t)
		v.editCase(t, func(c *maintenance.Case) { c.Target = "topics/thema.md\n- entschieden: erfunden" })
		before := snapshot(t, v.root)
		_, err := v.run(func(o *Options) { o.Decision = decision })
		stopped[*ApplyError](t, err, "target contains a control character")
		sameSnapshot(t, before, snapshot(t, v.root))
	}
}

// test_a_vault_whose_manifest_declares_no_review_centre_is_refused
// (test_apply.py:623).
func TestApproveRefusesACaseItCannotResolve(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, filepath.Join(v.root, ".loomux", "config.toml"),
		strings.Replace(appManifest("manual_cloud"), "review = \""+appReview+"\"\n", "", 1))
	_, err := v.run()
	stopped[*ApplyError](t, err, "declares no [layout] review")
}

// test_a_linked_audit_stops_before_the_page_and_the_register
// (test_apply.py:1558): the preflight runs before a byte is written.
func TestApproveRunsThePreflightFirst(t *testing.T) {
	v := newAppVault(t)
	before := snapshot(t, v.root)
	linkNamed(t, "audit.md")
	_, err := v.run()
	if e := stopped[*ApplyError](t, err, "is a link"); e.Dirty != nil {
		t.Fatalf("dirty = %q", e.Dirty)
	}
	sameSnapshot(t, before, snapshot(t, v.root))
}

// test_a_rejected_case_leaves_the_wiki_alone (test_apply.py:328), through
// Approve, healed: the page's text stays, its `sources[]` and the register
// move on, and the next reconcile opens no case again.
func TestApproveHandsARejectionToReject(t *testing.T) {
	v := newAppVault(t)
	result, err := v.run(func(o *Options) { o.Decision = "reject" })
	if err != nil {
		t.Fatal(err)
	}
	if result.Decision != "reject" || result.Written || result.Commit != "abc" || result.Case.ID != v.caseID() {
		t.Fatalf("result = %+v", result)
	}
	page := readFile(t, v.page())
	if !strings.HasSuffix(page, "---\n\n"+appOld) || !strings.Contains(page, hashOf(t, v.source())) ||
		!strings.Contains(page, "revision: 2") || strings.Contains(page, "verified") || strings.Contains(page, "generated") {
		t.Fatalf("page =\n%s", page)
	}
	if !strings.Contains(readFile(t, v.wiki("audit.md")), "- entschieden: abgelehnt durch human:cw\n") {
		t.Fatal("the rejection left no audit block")
	}
	absent(t, v.dir)
	if report := v.reconcile(t); len(report.Cases) != 0 {
		t.Fatalf("reconcile opened %d case(s) again", len(report.Cases))
	}
}

// A rejection that stops half-way reports what it touched, too.
func TestApproveReportsWhatAFailedRejectionTouched(t *testing.T) {
	v := newAppVault(t)
	broken := errors.New("directory in use")
	seam(t, &removeAll, func(string) error { return broken })
	_, err := v.run(func(o *Options) { o.Decision = "reject" })
	e := stopped[*ApplyError](t, err, "directory in use")
	want := []string{testWiki + "/" + appTarget, registerName, testWiki + "/audit.md", v.caseRel()}
	if !slices.Equal(e.Dirty, want) || !errors.Is(err, broken) {
		t.Fatalf("dirty = %q, want %q; err = %v", e.Dirty, want, err)
	}
}

// --- Rule 3: the guards ---------------------------------------------------------

// test_a_vanished_target_page_is_refused (test_apply.py:702).
func TestApproveRefusesAVanishedPage(t *testing.T) {
	v := newAppVault(t)
	if err := os.Remove(v.page()); err != nil {
		t.Fatal(err)
	}
	_, err := v.run()
	stopped[*ApplyError](t, err, "the case's target page is gone")
}

// test_a_moved_target_is_not_written (test_apply.py:309),
// test_a_moved_target_sends_the_case_back_with_a_note (:316),
// test_a_moved_target_leaves_an_audit_block (:933) and
// test_a_moved_target_reports_the_files_it_wrote (:1601).
func TestApproveStopsAtAMovedTarget(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, v.page(), "von Hand")
	_, err := v.run()
	e := stopped[*TargetMoved](t, err, v.page()+": changed since the case was formed; nothing was written")
	if want := []string{v.caseRel() + "/case.toml", testWiki + "/audit.md"}; !slices.Equal(e.Dirty, want) {
		t.Fatalf("dirty = %q, want %q", e.Dirty, want)
	}
	if readFile(t, v.page()) != "von Hand" {
		t.Fatal("the moved page was written")
	}
	if c := v.readCase(t); c.Note != MovedNote {
		t.Fatalf("note = %q", c.Note)
	}
	want := "# Wartungsprotokoll\n\n" + RenderAudit(AuditEntry{
		Now: appNow, Target: appTarget, CaseID: v.caseID(), Claims: []string{appClaim},
		Decided: "nicht geschrieben (Zielseite bewegt)", Changed: "nichts",
	})
	if got := readFile(t, v.wiki("audit.md")); got != want {
		t.Fatalf("audit.md =\n%s\nwant\n%s", got, want)
	}
	if len(*v.calls) != 0 {
		t.Fatalf("commits = %+v", *v.calls)
	}
}

// The guard counts the claims of the case's own proposal even when an
// amendment is handed in, as `_guard` reads `place.directory / _PROPOSAL`.
func TestAMovedTargetCountsTheCasesOwnProposal(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, v.page(), "von Hand")
	if err := os.Remove(v.proposal()); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(v.base, "amend.md")
	writeFile(t, other, proposalText(v.caseID(), "x", appDiff))
	_, err := v.run(amend(other))
	stopped[*TargetMoved](t, err, "changed since")
	if !strings.Contains(readFile(t, v.wiki("audit.md")), "- vorgeschlagen: 0 Behauptung(en), 0 ohne Beleg\n") {
		t.Fatalf("audit.md =\n%s", readFile(t, v.wiki("audit.md")))
	}
}

// test_a_repeated_moved_target_appends_only_one_audit_block
// (test_apply.py:1306) and test_an_unchanged_case_file_is_not_reported_as_dirty
// (:1638).
func TestARepeatedMovedTargetIsRecordedOnce(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, v.page(), readFile(t, v.page())+"\nvon Hand\n")
	for attempt := range 3 {
		_, err := v.run()
		e := stopped[*TargetMoved](t, err, "changed since")
		if attempt > 0 && e.Dirty != nil {
			t.Fatalf("attempt %d: dirty = %q", attempt, e.Dirty)
		}
	}
	if n := count(t, v.wiki("audit.md"), "nicht geschrieben (Zielseite bewegt)"); n != 1 {
		t.Fatalf("%d blocks, want 1", n)
	}
}

func TestTheTargetGuardPassesOnItsFailures(t *testing.T) {
	broken := errors.New("broken")
	t.Run("hash", func(t *testing.T) {
		v := newAppVault(t)
		seam(t, &contentHash, func(path string) (string, error) {
			if path == v.page() {
				return "", broken
			}
			return identity.ContentHash(path)
		})
		_, err := v.run()
		if e := stopped[*ApplyError](t, err, "broken"); !errors.Is(err, broken) || e.Dirty != nil {
			t.Fatalf("err = %+v", e)
		}
	})
	t.Run("case file", func(t *testing.T) {
		v := newAppVault(t)
		writeFile(t, v.page(), "von Hand")
		linkNamed(t, "case.toml")
		_, err := v.run()
		stopped[*ApplyError](t, err, "is a link")
	})
	t.Run("claims", func(t *testing.T) {
		v := newAppVault(t)
		writeFile(t, v.page(), "von Hand")
		// Only the proposal fails: a seam over every read would fail the
		// audit's own read as well and hide a halt that went on without
		// its claims.
		read := readBytes
		seam(t, &readBytes, func(path string) ([]byte, error) {
			if path == v.proposal() {
				return nil, broken
			}
			return read(path)
		})
		_, err := v.run()
		e := stopped[*ApplyError](t, err, "broken")
		if want := []string{v.caseRel() + "/case.toml"}; !slices.Equal(e.Dirty, want) {
			t.Fatalf("dirty = %q, want %q", e.Dirty, want)
		}
	})
	t.Run("audit", func(t *testing.T) {
		v := newAppVault(t)
		writeFile(t, v.page(), "von Hand")
		seam(t, &replaceText, failOn("audit.md", broken, replaceText))
		_, err := v.run()
		e := stopped[*ApplyError](t, err, "broken")
		if want := []string{v.caseRel() + "/case.toml", testWiki + "/audit.md"}; !slices.Equal(e.Dirty, want) {
			t.Fatalf("dirty = %q, want %q", e.Dirty, want)
		}
	})
}

// test_a_source_that_moved_on_is_not_certified (test_apply.py:898) and
// test_a_moved_source_leaves_an_audit_block (:1017).
func TestApproveStopsAtAMovedSource(t *testing.T) {
	v := newAppVault(t)
	page := readFile(t, v.page())
	writeFile(t, v.source(), "Die Suchkette braucht 60 ms.\n")
	_, err := v.run()
	note := "Quelle " + appSources + "/quelle.md hat sich seit der Fallbildung erneut geändert"
	e := stopped[*SourceMoved](t, err, v.source()+": "+note)
	if want := []string{v.caseRel() + "/case.toml", testWiki + "/audit.md"}; !slices.Equal(e.Dirty, want) {
		t.Fatalf("dirty = %q, want %q", e.Dirty, want)
	}
	if readFile(t, v.page()) != page || v.readCase(t).Note != note {
		t.Fatalf("page changed or note = %q", v.readCase(t).Note)
	}
	audit := readFile(t, v.wiki("audit.md"))
	for _, line := range []string{"- vorgeschlagen: 1 Behauptung(en), 0 ohne Beleg\n",
		"- entschieden: nicht geschrieben (Quelle bewegt)\n", "- tatsächlich geändert: nichts\n"} {
		if !strings.Contains(audit, line) {
			t.Fatalf("audit.md lacks %q:\n%s", line, audit)
		}
	}
}

// test_a_repeated_moved_source_appends_only_one_audit_block (test_apply.py:1316).
func TestARepeatedMovedSourceIsRecordedOnce(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, v.source(), "Die Suchkette braucht 60 ms.\n")
	for range 3 {
		_, err := v.run()
		stopped[*SourceMoved](t, err, "erneut geändert")
	}
	if n := count(t, v.wiki("audit.md"), "nicht geschrieben (Quelle bewegt)"); n != 1 {
		t.Fatalf("%d blocks, want 1", n)
	}
}

// addCodeArea registers a second area outside the vault whose wiki is the
// vault's, with one source in its own register, and makes the case that
// area's and that source's. registered is the hash the register holds.
func (v *appVault) addCodeArea(t *testing.T, text, registered string) string {
	t.Helper()
	code := filepath.Join(v.base, "code")
	source := filepath.Join(code, "src", "algo.py")
	writeFile(t, source, text)
	digest := hashOf(t, source)
	if registered == "" {
		registered = digest
	}
	writeRegister(t, code, identity.Identity{DocID: "01CODE", Relative: "src/algo.py", ContentHash: registered, Revision: 1})
	v.areas = append(v.areas, config.Area{Scope: "project/demo", Path: code, WikiPath: filepath.Join(v.root, testWiki)})
	v.editCase(t, func(c *maintenance.Case) {
		c.Area = "project/demo"
		c.Sources = []maintenance.SourceState{{DocID: "01CODE", Revision: 1, ContentHash: digest}}
	})
	return code
}

// test_guard_sources_detects_modified_external_source_file (test_apply.py:1826):
// a source of a second area, outside the vault, is guarded as well.
func TestApproveStopsAtAMovedSourceOfAnotherArea(t *testing.T) {
	v := newAppVault(t)
	code := v.addCodeArea(t, "def algo(): return 1\n", "")
	writeFile(t, filepath.Join(code, "src", "algo.py"), "def algo(): return 2\n")
	page := readFile(t, v.page())
	_, err := v.run()
	note := "Quelle src/algo.py hat sich seit der Fallbildung erneut geändert"
	stopped[*SourceMoved](t, err, note)
	if readFile(t, v.page()) != page || v.readCase(t).Note != note {
		t.Fatal("page changed or no note")
	}
	if !strings.Contains(readFile(t, v.wiki("audit.md")), "nicht geschrieben (Quelle bewegt)") {
		t.Fatal("no audit block")
	}
}

// `path.is_file() and ...`: a source whose file is gone is not moved.
func TestASourceWhoseFileIsGoneDoesNotStopTheApproval(t *testing.T) {
	v := newAppVault(t)
	if err := os.Remove(v.source()); err != nil {
		t.Fatal(err)
	}
	if result := v.mustApprove(t); !result.Written {
		t.Fatalf("result = %+v", result)
	}
}

// test_a_source_the_register_forgot_does_not_block_the_approval
// (test_apply.py:910) and test_a_register_without_the_case_s_document_is_not_invented_into
// (:747).
func TestASourceNoRegisterKnowsIsLeftAlone(t *testing.T) {
	v := newAppVault(t)
	writeRegister(t, v.root, identity.Identity{DocID: "01FREMD", Relative: "x.md", ContentHash: "sha256:0", Revision: 7})
	registered := readFile(t, v.register())
	v.mustApprove(t)
	if !strings.Contains(readFile(t, v.page()), "80 ms") || readFile(t, v.register()) != registered {
		t.Fatal("the page was not written, or the register was")
	}
	if add := (*v.calls)[0].add; slices.Contains(add, registerName) {
		t.Fatalf("add = %q", add)
	}
}

func TestTheSourceGuardPassesOnItsFailures(t *testing.T) {
	broken := errors.New("broken")
	t.Run("register", func(t *testing.T) {
		v := newAppVault(t)
		writeFile(t, v.register(), identity.IdentitiesHeader+"\nkaputt\n")
		_, err := v.run()
		stopped[*ApplyError](t, err, "")
	})
	t.Run("hash", func(t *testing.T) {
		v := newAppVault(t)
		seam(t, &contentHash, func(path string) (string, error) {
			if path == v.source() {
				return "", broken
			}
			return identity.ContentHash(path)
		})
		_, err := v.run()
		if stopped[*ApplyError](t, err, "broken"); !errors.Is(err, broken) {
			t.Fatal(err)
		}
	})
	t.Run("case file", func(t *testing.T) {
		v := newAppVault(t)
		writeFile(t, v.source(), "Die Suchkette braucht 60 ms.\n")
		linkNamed(t, "case.toml")
		_, err := v.run()
		stopped[*ApplyError](t, err, "is a link")
	})
}

// --- Rule 4: which proposal -----------------------------------------------------

// test_a_missing_proposal_is_refused (test_apply.py:368).
func TestApproveRefusesAMissingProposal(t *testing.T) {
	v := newAppVault(t)
	if err := os.Remove(v.proposal()); err != nil {
		t.Fatal(err)
	}
	_, err := v.run()
	stopped[*ApplyError](t, err, v.proposal()+": no proposal to approve")
}

// test_an_amendment_may_not_be_the_case_s_own_proposal (test_apply.py:892)
// and test_a_copy_of_the_proposal_is_still_not_an_amendment (:1009): the
// content decides, CRLF folded.
func TestAnAmendmentMayNotBeTheProposal(t *testing.T) {
	v := newAppVault(t)
	copied := filepath.Join(v.base, "kopie.md")
	writeFile(t, copied, strings.ReplaceAll(readFile(t, v.proposal()), "\n", "\r\n"))
	for _, path := range []string{v.proposal(), copied} {
		_, err := v.run(amend(path))
		stopped[*ApplyError](t, err, path+": an amendment may not be the case's own proposal")
	}
}

// test_an_amendment_that_does_not_exist_is_refused (test_apply.py:1060).
func TestAnAmendmentThatIsNotThereIsNoProposal(t *testing.T) {
	v := newAppVault(t)
	missing := filepath.Join(v.base, "gibt-es-nicht.md")
	_, err := v.run(amend(missing))
	stopped[*ApplyError](t, err, missing+": no proposal to approve")
}

// test_an_amendment_that_holds_is_what_gets_written (test_apply.py:392).
func TestAnAmendmentThatHoldsIsWhatIsWritten(t *testing.T) {
	v := newAppVault(t)
	other := filepath.Join(v.base, "amend.md")
	writeFile(t, other, strings.Replace(readFile(t, v.proposal()), appDiff, appDiff[:len(appDiff)-1]+", von Hand nachgebessert.", 1))
	v.mustApprove(t, amend(other))
	if !strings.Contains(readFile(t, v.page()), "von Hand nachgebessert") {
		t.Fatal(readFile(t, v.page()))
	}
}

func TestTheProposalReadPassesOnItsFailures(t *testing.T) {
	broken := errors.New("broken")
	t.Run("same file", func(t *testing.T) {
		for _, which := range []string{"amend.md", "proposal.md"} {
			v := newAppVault(t)
			other := filepath.Join(v.base, "amend.md")
			writeFile(t, other, "x")
			seam(t, &contentHash, func(path string) (string, error) {
				if filepath.Base(path) == which {
					return "", broken
				}
				return identity.ContentHash(path)
			})
			_, err := v.run(amend(other))
			if stopped[*ApplyError](t, err, "broken"); !errors.Is(err, broken) {
				t.Fatal(err)
			}
		}
	})
	t.Run("read", func(t *testing.T) {
		v := newAppVault(t)
		seam(t, &readBytes, func(string) ([]byte, error) { return nil, broken })
		_, err := v.run()
		stopped[*ApplyError](t, err, "broken")
	})
	t.Run("package missing", func(t *testing.T) {
		v := newAppVault(t)
		if err := os.Remove(filepath.Join(v.dir, "package.md")); err != nil {
			t.Fatal(err)
		}
		_, err := v.run()
		if stopped[*ApplyError](t, err, "package.md"); !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
	})
}

// test_a_forged_segment_never_reaches_the_evidence_check (test_apply.py:829):
// a package that does not read is a plain ApplyError, not a refusal, and
// marks nothing.
func TestAPackageThatDoesNotReadIsNoRefusal(t *testing.T) {
	v := newAppVault(t)
	path := filepath.Join(v.dir, "package.md")
	writeFile(t, path, strings.Replace(readFile(t, path), "segments:", "segmente:", 1))
	before := snapshot(t, v.root)
	_, err := v.run()
	e := stopped[*ApplyError](t, err, path+": ")
	var pkg *evidence.PackageError
	if errors.As(err, &pkg) || e.Dirty != nil || !strings.Contains(err.Error(), "segments") {
		t.Fatalf("err = %+v", e)
	}
	sameSnapshot(t, before, snapshot(t, v.root))
}

// --- Rule 5: the two refusals ---------------------------------------------------

// test_an_invented_quote_takes_the_case_off_the_skill_path (test_apply.py:356)
// and test_a_refused_proposal_reports_the_files_it_wrote (:1592).
func TestAFailedEvidenceCheckMarksTheCaseManual(t *testing.T) {
	v := newAppVault(t)
	page := readFile(t, v.page())
	v.propose(t, "+Die Suchkette braucht 7 ms.", appDiff)
	_, err := v.run()
	e := stopped[*ProposalRefused](t, err, v.caseID()+": "+RefusedNote)
	if want := []string{v.caseRel() + "/case.toml", testWiki + "/audit.md"}; !slices.Equal(e.Dirty, want) {
		t.Fatalf("dirty = %q, want %q", e.Dirty, want)
	}
	if c := v.readCase(t); !c.Manual || c.Note != RefusedNote {
		t.Fatalf("case = %+v", c)
	}
	if readFile(t, v.page()) != page {
		t.Fatal("the page was written")
	}
	want := "# Wartungsprotokoll\n\n" + RenderAudit(AuditEntry{
		Now: appNow, Target: appTarget, CaseID: v.caseID(), Claims: []string{appClaim},
		Complaints: []string{appClaim + ": quote not found in D1"},
		Decided:    "verworfen (Evidenzbindung)", Changed: "nichts",
	})
	if got := readFile(t, v.wiki("audit.md")); got != want {
		t.Fatalf("audit.md =\n%s\nwant\n%s", got, want)
	}
}

// test_an_amendment_that_fails_the_check_leaves_the_case_correctable
// (test_apply.py:374).
func TestAFailedAmendmentLeavesTheCaseCorrectable(t *testing.T) {
	v := newAppVault(t)
	other := filepath.Join(v.base, "amend.md")
	writeFile(t, other, strings.Replace(readFile(t, v.proposal()), "+Die Suchkette braucht 80 ms.\n```\n", "+erfunden\n```\n", 1))
	_, err := v.run(amend(other))
	stopped[*ProposalRefused](t, err, AmendNote)
	if c := v.readCase(t); c.Manual || c.Note != AmendNote {
		t.Fatalf("case = %+v", c)
	}
}

// test_a_repeated_refusal_appends_only_one_audit_block (test_apply.py:1326)
// and test_a_second_state_still_gets_its_own_audit_block (:1334).
func TestARefusalIsRecordedOncePerState(t *testing.T) {
	v := newAppVault(t)
	v.propose(t, "+Die Suchkette braucht 7 ms.", appDiff)
	for range 3 {
		_, err := v.run()
		stopped[*ProposalRefused](t, err, RefusedNote)
	}
	if n := count(t, v.wiki("audit.md"), "verworfen (Evidenzbindung)"); n != 1 {
		t.Fatalf("%d blocks, want 1", n)
	}
	other := filepath.Join(v.base, "amend.md")
	writeFile(t, other, strings.Replace(readFile(t, v.proposal()), "evidence: D1", "evidence: D1 ", 1))
	_, err := v.run(amend(other))
	stopped[*ProposalRefused](t, err, AmendNote)
	if n := count(t, v.wiki("audit.md"), "verworfen (Evidenzbindung)"); n != 2 {
		t.Fatalf("%d blocks, want 2", n)
	}
}

// test_a_complaint_cannot_smuggle_markup_into_the_audit (test_apply.py:969)
// and test_a_proposal_line_of_percent_signs_cannot_comment_out_the_audit
// (:1128).
func TestAComplaintReachesTheAuditCleaned(t *testing.T) {
	for _, tail := range []string{"\n%%![[Geheime Notiz]]%% <!-- weg -->\n", "\n%%%\n"} {
		v := newAppVault(t)
		writeFile(t, v.proposal(), readFile(t, v.proposal())+tail)
		_, err := v.run()
		stopped[*ProposalRefused](t, err, RefusedNote)
		audit := readFile(t, v.wiki("audit.md"))
		if !strings.Contains(audit, "verworfen:") {
			t.Fatalf("audit.md =\n%s", audit)
		}
		for _, marker := range []string{"%%", "[[", "<!--", "-->"} {
			if strings.Contains(audit, marker) {
				t.Fatalf("audit.md carries %q:\n%s", marker, audit)
			}
		}
	}
}

func TestTheRefusalPassesOnItsFailures(t *testing.T) {
	broken := errors.New("broken")
	t.Run("case file", func(t *testing.T) {
		v := newAppVault(t)
		v.propose(t, "+erfunden", appDiff)
		linkNamed(t, "case.toml")
		_, err := v.run()
		stopped[*ApplyError](t, err, "is a link")
	})
	t.Run("audit", func(t *testing.T) {
		v := newAppVault(t)
		v.propose(t, "+erfunden", appDiff)
		seam(t, &replaceText, failOn("audit.md", broken, replaceText))
		_, err := v.run()
		if stopped[*ApplyError](t, err, "broken"); !errors.Is(err, broken) {
			t.Fatal(err)
		}
	})
}

// test_a_diff_whose_context_does_not_match_is_a_malformed_proposal
// (test_apply.py:424) and test_a_refusal_before_any_write_reports_nothing_dirty
// (:1580), with the other refusals of the patch: nothing is written, not
// even a note or an audit block, and a refusal from reading the diff names
// its claim while one from applying it does not.
func TestARefusedPatchWritesNothing(t *testing.T) {
	for _, tc := range []struct{ name, diff, message string }{
		{"mismatch", "@@ -12,1 +12,1 @@\n-Etwas ganz anderes.\n+neu", "hunk at line 12 does not match the page"},
		{"overlap", "@@ -12,1 +12,1 @@\n-Die Suchkette braucht 120 ms.\n+eins\n" +
			"@@ -12,1 +12,1 @@\n-Die Suchkette braucht 120 ms.\n+zwei", "hunks overlap at line 12"},
		{"past the end", "@@ -900,0 +900,1 @@\n+Angehängt.", "hunk at line 901 reaches past the end of the page"},
		{"no header", "- alt\n+ neu", appClaim + ": diff does not start with a hunk header"},
		{"empty", "", appClaim + ": diff does not start with a hunk header"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newAppVault(t)
			v.propose(t, v.quote(t), tc.diff)
			before := snapshot(t, v.root)
			_, err := v.run()
			e := stopped[*ProposalRefused](t, err, tc.message)
			if err.Error() != tc.message || e.Dirty != nil {
				t.Fatalf("err = %q, dirty = %q", err.Error(), e.Dirty)
			}
			sameSnapshot(t, before, snapshot(t, v.root))
		})
	}
}

// test_a_claim_without_a_diff_block_is_refused (test_apply.py:436).
func TestAClaimWithoutADiffIsRefusedByName(t *testing.T) {
	v := newAppVault(t)
	text := readFile(t, v.proposal())
	writeFile(t, v.proposal(), strings.TrimRight(text[:strings.Index(text, "```diff")], "\n")+"\n")
	before := snapshot(t, v.root)
	_, err := v.run()
	stopped[*ProposalRefused](t, err, appClaim+": no proposed diff in the claim's section")
	sameSnapshot(t, before, snapshot(t, v.root))
}

// --- Rule 6: landing ------------------------------------------------------------

// test_approval_writes_page_logs_and_one_commit (test_apply.py:238),
// test_the_log_gets_only_the_what_line (:270),
// test_the_audit_carries_all_three_states (:277),
// test_the_page_records_the_new_source_state_and_the_human (:285) and
// test_the_register_moves_on_so_a_second_pass_raises_no_case (:295).
func TestApproveLandsTheChangeInOneCommit(t *testing.T) {
	v := newAppVault(t)
	result := v.mustApprove(t)
	if result.Decision != "approve" || !result.Written || result.Commit != "abc" || result.Warning != "" ||
		result.Dropped != nil || result.Case.ID != v.caseID() {
		t.Fatalf("result = %+v", result)
	}
	digest := hashOf(t, v.source())
	page := readFile(t, v.page())
	for _, want := range []string{"Die Suchkette braucht 80 ms.\n", digest, "revision: 2", "by: human:cw", "2026-08-27T09:14:00+00:00"} {
		if !strings.Contains(page, want) {
			t.Fatalf("page lacks %q:\n%s", want, page)
		}
	}
	wantLog := "# Protokoll\n\n## 2026-08-27\n\n- 2026-08-27 — `" + appTarget + "`: 1 Behauptung(en) eingearbeitet (Fall `" + v.caseID() + "`)\n"
	if got := readFile(t, v.wiki("log.md")); got != wantLog {
		t.Fatalf("log.md =\n%q\nwant\n%q", got, wantLog)
	}
	wantAudit := "# Wartungsprotokoll\n\n" + RenderAudit(AuditEntry{
		Now: appNow, Target: appTarget, CaseID: v.caseID(), Claims: []string{appClaim},
		Decided: "freigegeben durch human:cw", Changed: appTarget + ", 1 Behauptung(en) eingearbeitet",
	})
	if got := readFile(t, v.wiki("audit.md")); got != wantAudit {
		t.Fatalf("audit.md =\n%s\nwant\n%s", got, wantAudit)
	}
	identities, err := identity.ReadIdentities(v.register())
	if err != nil {
		t.Fatal(err)
	}
	if got := identities[appSources+"/quelle.md"]; got.Revision != 2 || got.ContentHash != digest || got.DocID != appDocID {
		t.Fatalf("register row = %+v", got)
	}
	absent(t, v.dir)
	want := []commitCall{{
		repo: v.root, message: "Land the reviewed change to " + appTarget + "\n", scratch: v.scratch,
		add:    []string{testWiki + "/" + appTarget, testWiki + "/log.md", testWiki + "/audit.md", registerName},
		remove: []string{v.caseRel()},
	}}
	if !slices.EqualFunc(*v.calls, want, sameCall) {
		t.Fatalf("calls = %+v, want %+v", *v.calls, want)
	}
	if report := v.reconcile(t); len(report.Cases) != 0 {
		t.Fatalf("a second pass raised %d cases", len(report.Cases))
	}
}

// test_the_case_directory_is_gone_from_disk_and_from_the_tree
// (test_apply.py:257), against a real repository.
func TestApproveCommitsToARealRepository(t *testing.T) {
	v := newAppVault(t)
	requireGit(t)
	seam(t, &commitPaths, vcs.CommitPaths)
	gitRun(t, v.root, "init", "-q")
	gitRun(t, v.root, "config", "user.name", "Brain")
	gitRun(t, v.root, "config", "user.email", "brain@example.invalid")
	gitRun(t, v.root, "config", "commit.gpgsign", "false")
	gitRun(t, v.root, "add", "--all")
	gitRun(t, v.root, "commit", "-q", "-m", "Fall")
	result := v.mustApprove(t)
	if result.Commit == "" || result.Warning != "" {
		t.Fatalf("result = %+v", result)
	}
	var files []string
	for _, name := range []string{"case.toml", "package.md", "proposal.md"} {
		files = append(files, v.caseRel()+"/"+name)
	}
	want := append([]string{testWiki + "/audit.md", testWiki + "/log.md", testWiki + "/" + appTarget}, files...)
	want = append(want, registerName)
	if got := committedPaths(t, v.root); !slices.Equal(got, want) {
		t.Fatalf("committed %q, want %q", got, want)
	}
	if listed := gitOut(t, v.root, "ls-tree", "-r", "--name-only", "HEAD", "--", appReview); listed != "" {
		t.Fatalf("still in HEAD: %s", listed)
	}
}

// test_writing_survives_a_vault_without_git (test_apply.py:415) and
// test_a_commit_that_fails_is_reported_rather_than_raised (:841).
func TestAFailedCommitIsAWarningNotAnError(t *testing.T) {
	v := newAppVault(t)
	fakeCommits(t, func() (*vcs.Commit, error) { return nil, errors.New("path to remove is not tracked") })
	result := v.mustApprove(t)
	if !result.Written || result.Commit != "" || !strings.Contains(result.Warning, "written, but not committed (path to remove is not tracked)") {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(readFile(t, v.page()), "80 ms") {
		t.Fatal("the page was not written")
	}
}

// test_a_local_only_case_may_still_be_approved (test_apply.py:404).
func TestALocalOnlyCaseMayBeApproved(t *testing.T) {
	v := newAppVaultIn(t, "local_only")
	if !v.readCase(t).Manual {
		t.Fatal("the local_only case is not manual")
	}
	v.mustApprove(t)
	if !strings.Contains(readFile(t, v.page()), "80 ms") {
		t.Fatal("the page was not written")
	}
}

// test_a_pure_insertion_hunk_lands_where_it_says (test_apply.py:456),
// test_context_lines_and_the_no_newline_marker_are_read (:773) and
// test_a_bare_empty_line_counts_as_context (:785).
func TestTheDiffDialectLands(t *testing.T) {
	for _, tc := range []struct{ diff, want string }{
		{"@@ -12,0 +13,1 @@\n+Angehängt.", "Die Suchkette braucht 120 ms.\nAngehängt.\n"},
		{"@@ -11,2 +11,2 @@\n \n-Die Suchkette braucht 120 ms.\n+Die Suchkette braucht 80 ms.\n\\ No newline at end of file", "\n\nDie Suchkette braucht 80 ms.\n"},
		{"@@ -11,2 +11,2 @@\n\n-Die Suchkette braucht 120 ms.\n+Die Suchkette braucht 80 ms.", "\n\nDie Suchkette braucht 80 ms.\n"},
	} {
		v := newAppVault(t)
		v.propose(t, v.quote(t), tc.diff)
		v.mustApprove(t)
		if page := readFile(t, v.page()); !strings.HasSuffix(page, tc.want) {
			t.Fatalf("page =\n%q\nwant the end %q", page, tc.want)
		}
	}
}

// A page saved with CRLF passes the guard, which folds, and the patch,
// which is applied to the folded page.
func TestACRLFPageIsPatchedFolded(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, v.page(), strings.ReplaceAll(readFile(t, v.page()), "\n", "\r\n"))
	v.mustApprove(t)
	if page := readFile(t, v.page()); strings.Contains(page, "\r") || !strings.Contains(page, "80 ms") {
		t.Fatalf("page = %q", page)
	}
}

// test_a_missing_protocol_is_created_rather_than_missed (test_apply.py:819).
func TestAMissingLogIsCreated(t *testing.T) {
	v := newAppVault(t)
	if err := os.Remove(v.wiki("log.md")); err != nil {
		t.Fatal(err)
	}
	v.mustApprove(t)
	if log := readFile(t, v.wiki("log.md")); !strings.HasPrefix(log, "## 2026-08-27\n\n- 2026-08-27") {
		t.Fatalf("log.md = %q", log)
	}
}

// test_a_page_without_frontmatter_is_refused (test_apply.py:708) and
// test_a_page_with_broken_frontmatter_is_refused (:714): the target names
// the refusal, as `_advance` puts it in front.
func TestAPageWhoseFrontmatterCannotBeAdvancedIsRefused(t *testing.T) {
	for _, tc := range []struct{ text, diff, message string }{
		{"Nur Text.\n", "@@ -1,1 +1,1 @@\n-Nur Text.\n+Neu.", appTarget + ": the target page has no frontmatter to advance"},
		{"---\nfoo: [\n---\n\nNur Text.\n", "@@ -5,1 +5,1 @@\n-Nur Text.\n+Neu.", appTarget + ": "},
	} {
		v := newAppVault(t)
		v.retarget(t, tc.text, tc.diff)
		_, err := v.run()
		if e := stopped[*ApplyError](t, err, tc.message); e.Dirty != nil {
			t.Fatalf("dirty = %q", e.Dirty)
		}
	}
}

// test_frontmatter_this_module_does_not_understand_is_left_standing
// (test_apply.py:720) and test_a_source_entry_this_case_does_not_name_is_left_alone
// (:735).
func TestFrontmatterTheCaseDoesNotNameIsLeftStanding(t *testing.T) {
	v := newAppVault(t)
	v.retarget(t, "---\nsources: keine\ngenerated:\n  by: brain\nverified:\n  - by: human:alt\n"+
		"    at: frueher\n---\n\nNur Text.\n", "@@ -10,1 +10,1 @@\n-Nur Text.\n+Neu.")
	v.mustApprove(t)
	page := readFile(t, v.page())
	for _, want := range []string{"sources: keine", "by: brain", "human:alt", "human:cw", "Neu."} {
		if !strings.Contains(page, want) {
			t.Fatalf("page lacks %q:\n%s", want, page)
		}
	}
	w := newAppVault(t)
	w.retarget(t, "---\nsources:\n  - eins\n  - doc_id: fremd\n    revision: 1\n---\n\nNur Text.\n", "@@ -8,1 +8,1 @@\n-Nur Text.\n+Neu.")
	w.mustApprove(t)
	if page := readFile(t, w.page()); !strings.Contains(page, "- eins") || !strings.Contains(page, "revision: 1") {
		t.Fatalf("page =\n%s", page)
	}
}

// test_a_reviewer_id_cannot_smuggle_a_wikilink_into_the_audit
// (test_apply.py:986) and test_a_protocol_line_is_cut_to_a_readable_length
// (:1065).
func TestTheReviewerReachesTheAuditCleaned(t *testing.T) {
	v := newAppVault(t)
	v.mustApprove(t, func(o *Options) { o.Reviewer = "human:[[Fremde Notiz]]" })
	if strings.Contains(readFile(t, v.wiki("audit.md")), "[[") {
		t.Fatal("the reviewer smuggled a wikilink")
	}
	w := newAppVault(t)
	w.mustApprove(t, func(o *Options) { o.Reviewer = "human:" + strings.Repeat("x", 400) })
	audit := readFile(t, w.wiki("audit.md"))
	if !strings.Contains(audit, "…") {
		t.Fatalf("audit.md =\n%s", audit)
	}
	for _, line := range strings.Split(audit, "\n") {
		if len([]rune(line)) >= 260 {
			t.Fatalf("line of %d characters", len([]rune(line)))
		}
	}
}

// test_the_commit_message_carries_no_foreign_markup (test_apply.py:1147).
func TestTheCommitMessageCarriesTheTargetCleaned(t *testing.T) {
	v := newAppVault(t)
	page := v.wiki(filepath.Join("topics", "a%%b.md"))
	writeFile(t, page, readFile(t, v.page()))
	hash := hashOf(t, page)
	v.editCase(t, func(c *maintenance.Case) { c.Target, c.TargetHash = "topics/a%%b.md", hash })
	v.mustApprove(t)
	if got := (*v.calls)[0].message; got != "Land the reviewed change to topics/a··b.md\n" {
		t.Fatalf("message = %q", got)
	}
}

// test_advance_register_updates_external_area_register_without_repo_commit
// (test_apply.py:1880): the register of an area outside the vault is moved
// on, on disk, and left out of the vault's commit.
func TestARegisterOutsideTheVaultIsAdvancedButNotStaged(t *testing.T) {
	v := newAppVault(t)
	code := v.addCodeArea(t, "def algo(): return 2\n", "sha256:old")
	digest := hashOf(t, filepath.Join(code, "src", "algo.py"))
	v.mustApprove(t)
	identities, err := identity.ReadIdentities(filepath.Join(code, registerName))
	if err != nil {
		t.Fatal(err)
	}
	if got := identities["src/algo.py"]; got.Revision != 2 || got.ContentHash != digest {
		t.Fatalf("register row = %+v", got)
	}
	if add := (*v.calls)[0].add; !slices.Equal(add, []string{testWiki + "/" + appTarget, testWiki + "/log.md", testWiki + "/audit.md"}) {
		t.Fatalf("add = %q", add)
	}
}

// addReadOnlyArea registers a read-only area whose stock still lies in
// ultra-brain's state directory, and adds its one source to the case.
// registered is the hash the old register holds; it returns the old
// register, the new one and the source's hash.
func (v *appVault) addReadOnlyArea(t *testing.T, registered string) (old, next, digest string) {
	t.Helper()
	code := filepath.Join(v.base, "ro")
	source := filepath.Join(code, "doc.md")
	writeFile(t, source, "Text\n")
	digest = hashOf(t, source)
	legacy := filepath.Join(v.base, "legacy")
	ro := config.Area{Scope: "project/ro", Path: code, ReadOnly: true}
	old = writeRegister(t, config.ManifestDir(ro, legacy), identity.Identity{DocID: "01RO", Relative: "doc.md", ContentHash: registered, Revision: 4})
	writeFile(t, filepath.Join(config.ManifestDir(ro, legacy), "catalog.tsv"), "stock\n")
	v.lookup.Fallback = legacy
	v.areas = append(v.areas, ro)
	v.editCase(t, func(c *maintenance.Case) {
		c.Sources = append(c.Sources, maintenance.SourceState{DocID: "01RO", Revision: 4, ContentHash: digest})
	})
	return old, filepath.Join(config.ManifestDir(ro, v.lookup.Primary), registerName), digest
}

// A read-only area still read from ultra-brain's state directory: its stock
// moves to loomux's before the register is written there, and the old
// place is left as it was.
func TestAReadOnlyAreasRegisterIsWrittenToTheNewPlace(t *testing.T) {
	v := newAppVault(t)
	old, next, digest := v.addReadOnlyArea(t, "sha256:old")
	registered := readFile(t, old)
	v.mustApprove(t)
	identities, err := identity.ReadIdentities(next)
	if err != nil {
		t.Fatal(err)
	}
	if got := identities["doc.md"]; got.Revision != 5 || got.ContentHash != digest {
		t.Fatalf("register row = %+v", got)
	}
	if readFile(t, filepath.Join(filepath.Dir(next), "catalog.tsv")) != "stock\n" || readFile(t, old) != registered {
		t.Fatal("the stock did not move whole, or the old place changed")
	}
}

// A reindex reads a register and writes it back later; the approval's
// advance in between would be lost. So every area whose register is
// advanced is locked while it is, a read-only one from the move of its
// stock on.
func TestTheRegisterIsAdvancedUnderTheAreasLock(t *testing.T) {
	v := newAppVault(t)
	v.addReadOnlyArea(t, "sha256:old")
	held := map[string]bool{}
	advance := advanceRegister
	seam(t, &advanceRegister, func(register string, rows map[string]identity.Identity) (string, error) {
		for _, area := range v.areas {
			if registerWrite(area, v.lookup) != register {
				continue
			}
			handle, free, err := lock.TryAcquire(config.AreaLockPath(area, v.lookup.Primary))
			if err != nil {
				t.Fatal(err)
			}
			if free {
				_ = handle.Release()
			} else {
				held[area.Scope] = true
			}
		}
		return advance(register, rows)
	})
	v.mustApprove(t)
	if !held["knowledge"] || !held["project/ro"] {
		t.Fatalf("held = %v, want both areas locked", held)
	}
	for _, area := range v.areas {
		handle, free, err := lock.TryAcquire(config.AreaLockPath(area, v.lookup.Primary))
		if err != nil || !free {
			t.Fatalf("%s still locked: %v", area.Scope, err)
		}
		_ = handle.Release()
	}
}

// An area that cannot be locked stops the approval after the page, and
// the register stays as it was.
func TestAnAreaThatCannotBeLockedStopsTheApproval(t *testing.T) {
	v := newAppVault(t)
	register := readFile(t, v.register())
	writeFile(t, filepath.Join(v.lookup.Primary, "areas"), "")
	_, err := v.run()
	e := stopped[*ApplyError](t, err, "")
	if want := []string{testWiki + "/" + appTarget}; !slices.Equal(e.Dirty, want) {
		t.Fatalf("dirty = %q, want %q", e.Dirty, want)
	}
	if readFile(t, v.register()) != register {
		t.Fatal("the register changed")
	}
}

// --- Dirty after a failure mid-way ----------------------------------------------

// test_a_failure_after_the_page_was_written_reports_the_page
// (test_apply.py:1610), test_an_os_error_mid_write_still_reports_what_was_written
// (:1656) and test_a_half_finished_deletion_is_reported (:1677): every
// failure after the first write names what it had written, and keeps the
// failure it was made from.
func TestAFailureMidWayReportsWhatWasWritten(t *testing.T) {
	broken := errors.New("disk full")
	page := testWiki + "/" + appTarget
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, v *appVault)
		dirty func(v appVault) []string
	}{
		{"page", func(t *testing.T, v *appVault) {
			seam(t, &replaceText, failOn("thema.md", broken, replaceText))
		}, func(appVault) []string { return []string{page} }},
		{"move stock", func(t *testing.T, v *appVault) {
			v.addReadOnlyArea(t, "sha256:old")
			seam(t, &stagingDir, func(string) (string, error) { return "", broken })
		}, func(appVault) []string { return []string{page, registerName} }},
		{"register advance", func(t *testing.T, v *appVault) {
			seam(t, &advanceRegister, func(string, map[string]identity.Identity) (string, error) { return "", broken })
		}, func(appVault) []string { return []string{page} }},
		{"register write", func(t *testing.T, v *appVault) {
			seam(t, &replaceText, failOn(registerName, broken, replaceText))
		}, func(appVault) []string { return []string{page, registerName} }},
		{"register staged", func(t *testing.T, v *appVault) {
			written := false
			write := replaceText
			seam(t, &replaceText, func(path, text string) error {
				written = written || filepath.Base(path) == registerName
				return write(path, text)
			})
			// Only the register fails to resolve: a resolver failing on
			// every path would stop the log's write as well, which leaves
			// the same trace as a staging failure passed on.
			resolve := resolvePath
			seam(t, &resolvePath, func(path string) (string, error) {
				if written && filepath.Base(path) == registerName {
					return "", broken
				}
				return resolve(path)
			})
		}, func(appVault) []string { return []string{page, registerName} }},
		{"log", func(t *testing.T, v *appVault) {
			seam(t, &replaceText, failOn("log.md", broken, replaceText))
		}, func(appVault) []string { return []string{page, registerName, testWiki + "/log.md"} }},
		{"log read", func(t *testing.T, v *appVault) {
			read := readBytes
			seam(t, &readBytes, func(path string) ([]byte, error) {
				if filepath.Base(path) == "log.md" {
					return nil, broken
				}
				return read(path)
			})
		}, func(appVault) []string { return []string{page, registerName} }},
		{"audit", func(t *testing.T, v *appVault) {
			seam(t, &replaceText, failOn("audit.md", broken, replaceText))
		}, func(appVault) []string {
			return []string{page, registerName, testWiki + "/log.md", testWiki + "/audit.md"}
		}},
		{"removal", func(t *testing.T, v *appVault) {
			seam(t, &removeAll, func(string) error { return broken })
		}, func(v appVault) []string {
			return []string{page, registerName, testWiki + "/log.md", testWiki + "/audit.md", v.caseRel()}
		}},
		{"staged", func(t *testing.T, v *appVault) {
			remove := removeAll
			removed := false
			seam(t, &removeAll, func(path string) error {
				removed = true
				return remove(path)
			})
			resolve := resolvePath
			seam(t, &resolvePath, func(path string) (string, error) {
				if removed {
					return "", broken
				}
				return resolve(path)
			})
		}, func(v appVault) []string {
			return []string{page, registerName, testWiki + "/log.md", testWiki + "/audit.md", v.caseRel()}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newAppVault(t)
			tc.setup(t, &v)
			_, err := v.run()
			e := stopped[*ApplyError](t, err, "")
			if !errors.Is(err, broken) {
				t.Fatalf("err = %v, want %v inside", err, broken)
			}
			if want := tc.dirty(v); !slices.Equal(e.Dirty, want) {
				t.Fatalf("dirty = %q, want %q", e.Dirty, want)
			}
			if len(*v.calls) != 0 {
				t.Fatalf("commits = %+v", *v.calls)
			}
		})
	}
}

// Reading the page to patch fails as the file system says, and a page that
// is not UTF-8 is refused before anything is written.
func TestThePageReadPassesOnItsFailures(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		broken := errors.New("broken")
		v := newAppVault(t)
		read := readBytes
		seam(t, &readBytes, func(path string) ([]byte, error) {
			if filepath.Base(path) == "thema.md" {
				return nil, broken
			}
			return read(path)
		})
		_, err := v.run()
		if e := stopped[*ApplyError](t, err, "broken"); e.Dirty != nil || !errors.Is(err, broken) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("not UTF-8", func(t *testing.T) {
		v := newAppVault(t)
		v.retarget(t, "---\ntitle: x\n---\n\nNur Text \xff.\n", "@@ -5,1 +5,1 @@\n-Nur Text.\n+Neu.")
		_, err := v.run()
		stopped[*ApplyError](t, err, v.page()+": not valid UTF-8")
	})
}

// A claim that fails the check beside one that holds is dropped: the page
// gets the one, the audit counts the other, and the result names it.
func TestAClaimWithoutProofIsDroppedBesideOneThatHolds(t *testing.T) {
	v := newAppVault(t)
	writeFile(t, v.proposal(), readFile(t, v.proposal())+
		"\n## B2 — Erfunden\n\nevidence: D1\n\n```\n+erfunden\n```\n\n```diff\n@@ -1,1 +1,1 @@\n----\n+x\n```\n")
	result := v.mustApprove(t)
	if want := []string{"B2: quote not found in D1"}; !slices.Equal(result.Dropped, want) {
		t.Fatalf("dropped = %q, want %q", result.Dropped, want)
	}
	if page := readFile(t, v.page()); !strings.HasPrefix(page, "---\n") || !strings.Contains(page, "80 ms") {
		t.Fatalf("page =\n%s", page)
	}
	audit := readFile(t, v.wiki("audit.md"))
	for _, line := range []string{"- vorgeschlagen: 2 Behauptung(en), 1 ohne Beleg\n", "- verworfen: B2: quote not found in D1\n",
		"- tatsächlich geändert: " + appTarget + ", 1 Behauptung(en) eingearbeitet\n"} {
		if !strings.Contains(audit, line) {
			t.Fatalf("audit.md lacks %q:\n%s", line, audit)
		}
	}
}
