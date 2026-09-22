package apply

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/vcs"
	"github.com/xidus90/loomux/internal/config"
)

// A rejection, ported from `_reject` (apply.py:698-727). Where a test
// carries over one of test_apply.py, its comment names it and its line.

const (
	rejectPage     = "---\ntitle: x\n---\n"
	rejectReviewer = "human:anna"
	rejectCaseDir  = testReview + "/knowledge/c1"
	twoClaims      = "## B1 — Die Suchkette ist schneller\n\nevidence: D1\n\n" +
		"## B2 — Der Index ist kleiner\n\nevidence: D2\n"
)

var rejectNow = time.Date(2026, 8, 27, 9, 14, 0, 0, time.UTC)

// rejection is a resolved vault with a case to reject, the place built
// from it as `_resolve` builds `_Place`, preflight included.
type rejection struct {
	vault string
	r     resolved
	p     *place
	c     maintenance.Case
	areas []config.Area
}

func newRejection(t *testing.T) rejection {
	t.Helper()
	_, casePath, areas := newVault(t)
	return rejectionAt(t, casePath, areas)
}

func rejectionAt(t *testing.T, casePath string, areas []config.Area) rejection {
	t.Helper()
	c := caseOf("knowledge")
	r, err := resolve(casePath, c, areas)
	if err != nil {
		t.Fatal(err)
	}
	p := &place{anchor: r.vault, wiki: r.wiki, registers: registersOf(areas, config.ArtifactLookup{})}
	if err := p.preflight(r.directory); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(r.directory, "proposal.md"), twoClaims)
	return rejection{vault: r.vault, r: r, p: p, c: c, areas: areas}
}

func (j rejection) run(t *testing.T) (Result, error) {
	t.Helper()
	return reject(j.r, j.p, j.c, rejectReviewer, rejectNow, t.TempDir())
}

func (j rejection) audit(t *testing.T) string {
	t.Helper()
	return readFile(t, filepath.Join(j.r.wiki, "audit.md"))
}

func mustReject(t *testing.T, j rejection) Result {
	t.Helper()
	result, err := j.run(t)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	return result
}

func noRepository(t *testing.T) {
	t.Helper()
	fakeCommits(t, func() (*vcs.Commit, error) { return nil, nil })
}

// test_a_rejected_case_leaves_the_wiki_alone (test_apply.py:328).
func TestRejectWritesTheAuditAndRemovesTheCase(t *testing.T) {
	j := newRejection(t)
	fakeCommits(t, landed("abc"))
	result := mustReject(t, j)

	want := RenderAudit(AuditEntry{
		Now: rejectNow, Target: "topics/thema.md", CaseID: "c1",
		Claims:  []string{"B1", "B2"},
		Decided: "abgelehnt durch " + rejectReviewer, Changed: "nichts",
	})
	if got := j.audit(t); got != want {
		t.Fatalf("audit.md =\n%s\nwant\n%s", got, want)
	}
	for _, line := range []string{"- vorgeschlagen: 2 Behauptung(en), 0 ohne Beleg",
		"- entschieden: abgelehnt durch human:anna", "- tatsächlich geändert: nichts"} {
		if !strings.Contains(want, line+"\n") {
			t.Fatalf("audit.md lacks %q:\n%s", line, want)
		}
	}
	absent(t, j.r.directory)
	if result.Case.ID != "c1" || result.Decision != "reject" || result.Written ||
		result.Commit != "abc" || result.Warning != "" || result.Dropped != nil {
		t.Fatalf("result = %+v", result)
	}
	wantTouched := []string{testWiki + "/audit.md", rejectCaseDir}
	if !slices.Equal(j.p.touched, wantTouched) {
		t.Fatalf("touched = %q, want %q", j.p.touched, wantTouched)
	}
}

// The commit carries audit.md alone and the case directory as a removal:
// no page, no log, no register.
func TestRejectCommitsOnlyTheAudit(t *testing.T) {
	j := newRejection(t)
	calls := fakeCommits(t, landed("abc"))
	mustReject(t, j)
	want := []commitCall{{
		repo: j.vault, message: "Reject the proposed change to topics/thema.md\n",
		add: []string{testWiki + "/audit.md"}, remove: []string{rejectCaseDir},
	}}
	got := *calls
	if len(got) != 1 {
		t.Fatalf("calls = %+v", got)
	}
	got[0].scratch = ""
	if !slices.EqualFunc(got, want, sameCall) {
		t.Fatalf("calls = %+v, want %+v", got, want)
	}
}

// The target reaches the commit message through Safe.
func TestRejectCleansTheTargetInTheCommitMessage(t *testing.T) {
	j := newRejection(t)
	j.c.Target = "topics/a%%b.md"
	calls := fakeCommits(t, landed("abc"))
	mustReject(t, j)
	if got := (*calls)[0].message; got != "Reject the proposed change to topics/a··b.md\n" {
		t.Fatalf("message = %q", got)
	}
}

// test_a_rejection_without_a_proposal_still_leaves_an_audit_block
// (test_apply.py:764).
func TestRejectWithoutAProposalCountsNoClaims(t *testing.T) {
	j := newRejection(t)
	if err := os.Remove(filepath.Join(j.r.directory, "proposal.md")); err != nil {
		t.Fatal(err)
	}
	noRepository(t)
	mustReject(t, j)
	if !strings.Contains(j.audit(t), "- vorgeschlagen: 0 Behauptung(en), 0 ohne Beleg\n") {
		t.Fatalf("audit.md =\n%s", j.audit(t))
	}
}

// `_claims` asks `is_file`: a proposal.md that is a directory is no
// proposal, not a failure.
func TestRejectReadsAProposalThatIsNoFileAsNone(t *testing.T) {
	j := newRejection(t)
	proposal := filepath.Join(j.r.directory, "proposal.md")
	if err := os.Remove(proposal); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(proposal, 0o755); err != nil {
		t.Fatal(err)
	}
	noRepository(t)
	mustReject(t, j)
	if !strings.Contains(j.audit(t), "- vorgeschlagen: 0 Behauptung(en)") {
		t.Fatalf("audit.md =\n%s", j.audit(t))
	}
}

// `read_text(errors="replace")`: a proposal that is not UTF-8 is still
// counted, not refused.
func TestRejectCountsTheClaimsOfAProposalThatIsNotUTF8(t *testing.T) {
	j := newRejection(t)
	writeFile(t, filepath.Join(j.r.directory, "proposal.md"), "## B1 — Satz \xff\n\nevidence: D1\n")
	noRepository(t)
	mustReject(t, j)
	if !strings.Contains(j.audit(t), "- vorgeschlagen: 1 Behauptung(en)") {
		t.Fatalf("audit.md =\n%s", j.audit(t))
	}
}

func TestRejectPassesOnAProposalThatCannotBeRead(t *testing.T) {
	j := newRejection(t)
	broken := errors.New("unreadable")
	seam(t, &readBytes, func(string) ([]byte, error) { return nil, broken })
	if _, err := j.run(t); !errors.Is(err, broken) {
		t.Fatalf("err = %v, want %v", err, broken)
	}
	if len(j.p.touched) != 0 {
		t.Fatalf("touched = %q, want nothing", j.p.touched)
	}
}

// `_append`: the block goes below what audit.md holds, one blank line
// between, however many newlines the file ended in.
func TestRejectAppendsToAnExistingAudit(t *testing.T) {
	j := newRejection(t)
	writeFile(t, filepath.Join(j.r.wiki, "audit.md"), "# Audit\r\n\n\n")
	noRepository(t)
	mustReject(t, j)
	if got := j.audit(t); !strings.HasPrefix(got, "# Audit\r\n\n## 2026-08-27T09:14:00+00:00 — topics/thema.md (Fall `c1`)\n") {
		t.Fatalf("audit.md =\n%q", got)
	}
}

// `_append` reads audit.md strictly as UTF-8: a file that is not stops the
// rejection before anything is written.
func TestRejectRefusesAnAuditThatIsNotUTF8(t *testing.T) {
	j := newRejection(t)
	writeFile(t, filepath.Join(j.r.wiki, "audit.md"), "\xff\n")
	if _, err := j.run(t); err == nil || !strings.Contains(err.Error(), "not valid UTF-8") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(j.r.directory); err != nil {
		t.Fatalf("the case directory is gone: %v", err)
	}
}

func TestRejectPassesOnAnAuditThatCannotBeRead(t *testing.T) {
	j := newRejection(t)
	writeFile(t, filepath.Join(j.r.wiki, "audit.md"), "# Audit\n")
	broken := errors.New("unreadable")
	seam(t, &readBytes, func(path string) ([]byte, error) {
		if filepath.Base(path) == "audit.md" {
			return nil, broken
		}
		return os.ReadFile(path)
	})
	if _, err := j.run(t); !errors.Is(err, broken) {
		t.Fatalf("err = %v, want %v", err, broken)
	}
}

// test_a_case_id_cannot_comment_out_the_audit (test_apply.py:1142).
func TestRejectKeepsTheCaseIDFromCommentingOutTheAudit(t *testing.T) {
	j := newRejection(t)
	j.c.ID = "f-%%%"
	noRepository(t)
	mustReject(t, j)
	if strings.Contains(j.audit(t), "%%") {
		t.Fatalf("audit.md =\n%s", j.audit(t))
	}
}

// test_a_rejection_also_refuses_a_linked_write_target
// (test_apply.py:1291): audit.md goes through the barrier like every write.
func TestRejectWritesTheAuditThroughTheBarrier(t *testing.T) {
	j := newRejection(t)
	linkNamed(t, "audit.md")
	_, err := j.run(t)
	refused(t, err, "is a link")
	if _, err := os.Stat(j.r.directory); err != nil {
		t.Fatalf("the case directory is gone: %v", err)
	}
}

// A deletion that fails is passed on, and the audit block stands recorded.
func TestRejectPassesOnAFailedRemoval(t *testing.T) {
	j := newRejection(t)
	broken := errors.New("in use")
	seam(t, &removeAll, func(string) error { return broken })
	if _, err := j.run(t); !errors.Is(err, broken) {
		t.Fatalf("err = %v, want %v", err, broken)
	}
	wantTouched := []string{testWiki + "/audit.md", rejectCaseDir}
	if !slices.Equal(j.p.touched, wantTouched) {
		t.Fatalf("touched = %q, want %q", j.p.touched, wantTouched)
	}
}

// `_staged` resolves audit.md once more after the removal; a failure there
// is passed on rather than committing a path it could not place.
func TestRejectPassesOnAResolverFailureWhenStaging(t *testing.T) {
	j := newRejection(t)
	removed := false
	seam(t, &removeAll, func(path string) error {
		removed = true
		return os.RemoveAll(path)
	})
	broken := errors.New("unresolvable")
	real := resolvePath
	// The vault still resolves: were it to fail too, its own failure would
	// stand in for the one this test is about.
	seam(t, &resolvePath, func(path string) (string, error) {
		if removed && path != j.vault {
			return "", broken
		}
		return real(path)
	})
	if _, err := j.run(t); !errors.Is(err, broken) {
		t.Fatalf("err = %v, want %v", err, broken)
	}
}

func TestRejectPassesOnAResolverFailureForTheVaultWhenStaging(t *testing.T) {
	j := newRejection(t)
	removed := false
	seam(t, &removeAll, func(path string) error {
		removed = true
		return os.RemoveAll(path)
	})
	broken := errors.New("unresolvable")
	real := resolvePath
	seam(t, &resolvePath, func(path string) (string, error) {
		if removed && path == j.vault {
			return "", broken
		}
		return real(path)
	})
	if _, err := j.run(t); !errors.Is(err, broken) {
		t.Fatalf("err = %v, want %v", err, broken)
	}
}

// `_staged` (apply.py:683-695): an audit.md under a wiki outside the vault
// is written there but cannot be staged into the vault's repository, so the
// commit carries the removal alone.
func TestRejectLeavesAnAuditOutsideTheVaultOutOfTheCommit(t *testing.T) {
	vault, casePath, areas := newVault(t)
	bundle := t.TempDir()
	writeFile(t, filepath.Join(bundle, "_schema.md"), "")
	writeFile(t, filepath.Join(bundle, "topics", "thema.md"), rejectPage)
	areas[0].WikiPath = bundle
	j := rejectionAt(t, casePath, areas)
	calls := fakeCommits(t, landed("abc"))
	mustReject(t, j)
	got := (*calls)[0]
	if len(got.add) != 0 || !slices.Equal(got.remove, []string{rejectCaseDir}) || got.repo != vault {
		t.Fatalf("call = %+v", got)
	}
	if !strings.Contains(readFile(t, filepath.Join(bundle, "audit.md")), "abgelehnt durch") {
		t.Fatal("audit.md of the bundle holds no block")
	}
}

// A commit that fails is a warning, and a rejection says it recorded a
// decision, not that it wrote.
func TestRejectReportsAFailedCommitAsARecordedDecision(t *testing.T) {
	j := newRejection(t)
	fakeCommits(t, func() (*vcs.Commit, error) { return nil, errors.New("kaputt") })
	result := mustReject(t, j)
	if result.Commit != "" || result.Warning != "c1: decision recorded, but not committed (kaputt)" || result.Written {
		t.Fatalf("result = %+v", result)
	}
}

// Inherited, not healed: a rejection advances neither the page's
// `sources[]` revision and hash nor the register, so the next reconcile
// opens the same case again. Spec stufe-3-design.md:330 ("Geerbter
// Fehler"), `OFFENE_AUFGABEN.md:213` in ultra-brain, and the parity record
// docs/.superpowers/parity/stufe-3b.md, section "Geerbt".
func TestRejectAdvancesNeitherThePageNorTheRegister(t *testing.T) {
	j := newRejection(t)
	page := filepath.Join(j.r.wiki, "topics", "thema.md")
	pageText := "---\nsources:\n  - doc_id: 01DOC0\n    content_hash: \"sha256:aa\"\n    revision: 1\n---\n\nText\n"
	writeFile(t, page, pageText)
	register := filepath.Join(j.vault, registerName)
	registerText := "doc_id\trelative\tcontent_hash\trevision\n01DOC0\tq.md\tsha256:aa\t1\n"
	writeFile(t, register, registerText)
	noRepository(t)
	mustReject(t, j)
	if readFile(t, page) != pageText || readFile(t, register) != registerText {
		t.Fatal("the rejection changed the page or the register")
	}
}

// End to end against a real repository: audit.md added, every file of the
// case removed, nothing else in the commit.
func TestRejectCommitsInARealRepository(t *testing.T) {
	vault := gitVault(t, map[string]string{
		".loomux/config.toml":          vaultManifest(testReview),
		testWiki + "/topics/thema.md":  rejectPage,
		rejectCaseDir + "/case.toml":   "id = \"c1\"\n",
		rejectCaseDir + "/proposal.md": twoClaims,
		testWiki + "/fremd.md":         "alt\n",
	})
	writeFile(t, filepath.Join(vault, testWiki, "fremd.md"), "vom Nutzer geändert\n")
	areas := []config.Area{{Scope: "knowledge", Path: vault, WikiPath: filepath.Join(vault, testWiki)}}
	j := rejectionAt(t, filepath.Join(vault, filepath.FromSlash(rejectCaseDir), "case.toml"), areas)
	result := mustReject(t, j)
	if result.Warning != "" || result.Commit != gitOut(t, vault, "rev-parse", "HEAD") {
		t.Fatalf("result = %+v", result)
	}
	want := []string{rejectCaseDir + "/case.toml", rejectCaseDir + "/proposal.md", testWiki + "/audit.md"}
	if got := committedPaths(t, vault); !slices.Equal(got, want) {
		t.Fatalf("paths of the commit = %q, want %q", got, want)
	}
}
