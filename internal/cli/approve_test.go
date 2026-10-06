package cli

import (
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/apply"
	"github.com/xidus90/loomux/internal/brain/maintenance"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
)

// dirtyHint is the hint for files inside the vault, without the file lines
// that follow it.
const dirtyHint = "Hinweis: der Abbruch hat diese Dateien im Vault bereits geändert und nichts " +
	"committet. Sie stehen unversioniert im Arbeitsbaum, bis der nächste Auto-Commit " +
	"eines fremden Werkzeugs (etwa obsidian-git) sie einsammelt:\n"

// technicalUpdateWarning is the reference's line for an index run that did
// not go through, with `brain` read as `loomux`.
const technicalUpdateWarning = "warning: die technische Aktualisierung ist fehlgeschlagen; " +
	"die Freigabe steht, der Index hinkt ihr hinterher -- `loomux reindex` holt das nach\n"

// qmdUpdated is what the index run of the world says on stderr, as the
// reference's `reindex` does: the one collection it wrote.
const qmdUpdated = "updated qmd collections: project-a\n"

// approveCall is what one call of the stubbed apply.Approve was handed.
type approveCall struct {
	casePath string
	areas    []config.Area
	options  apply.Options
}

// stubApprove replaces apply.Approve for one test and records every call.
// decide answers the call; nil means "must not be called".
func stubApprove(t *testing.T, decide func(approveCall) (apply.Result, error)) *[]approveCall {
	t.Helper()
	var calls []approveCall
	saved := approveCase
	t.Cleanup(func() { approveCase = saved })
	approveCase = func(casePath string, areas []config.Area, o apply.Options) (apply.Result, error) {
		call := approveCall{casePath, areas, o}
		calls = append(calls, call)
		if decide == nil {
			t.Fatal("apply.Approve was called")
		}
		return decide(call)
	}
	return &calls
}

// stubUser fixes the account running the command.
func stubUser(t *testing.T, name string, err error) {
	t.Helper()
	saved := currentUser
	t.Cleanup(func() { currentUser = saved })
	currentUser = func() (*user.User, error) { return &user.User{Username: name}, err }
}

// decided is a decision as apply reports it, for the case the world holds.
func decided(decision string, written bool, commit, warning string, dropped ...string) func(approveCall) (apply.Result, error) {
	return func(approveCall) (apply.Result, error) {
		return apply.Result{
			Case:     maintenance.Case{ID: "open-case"},
			Decision: decision, Written: written, Commit: commit, Warning: warning, Dropped: dropped,
		}, nil
	}
}

// approveWorld is the reconcile world with one case standing in its review
// centre, on a target no source of the world derives, so a catch-up opens
// nothing beside it unless the options cite.
func approveWorld(t *testing.T, opts reconcileOptions) (reconcileWorld, string) {
	t.Helper()
	w := newReconcileWorld(t, opts)
	directory := placeCase(t, w, "project-a/open-case", standingCase("open-case", "a.md", "due", ""), nil)
	stubUser(t, `MACHINE\tester`, nil)
	stubBrainStatusPort(t, search.NewFakePort())
	return w, directory
}

// The call apply gets: the case file, the registry, the reviewer, the scratch
// directory under the state directory, and the amendment -- with the flags
// standing on either side of the id.
func TestApproveHandsApplyTheReviewerTheScratchAndTheAmendment(t *testing.T) {
	w, directory := approveWorld(t, reconcileOptions{})
	calls := stubApprove(t, decided("approve", true, "abc1234", ""))
	before := time.Now().UTC()
	code, out, errOut := run("approve", "--amend", "amend.md", "open-case")
	if code != 0 || out != "Fall open-case: approve\ncommittet als abc1234\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if len(*calls) != 1 {
		t.Fatalf("apply.Approve called %d times", len(*calls))
	}
	call := (*calls)[0]
	o := call.options
	if call.casePath != filepath.Join(directory, "case.toml") || len(call.areas) != 1 ||
		call.areas[0].Scope != "project/a" {
		t.Fatalf("case = %q, areas = %+v", call.casePath, call.areas)
	}
	if o.Amend != "amend.md" || o.Decision != "approve" || o.Reviewer != "human:tester" ||
		o.Scratch != filepath.Join(w.State, "maintenance") || o.Lookup.Primary != w.State {
		t.Fatalf("options = %+v", o)
	}
	if o.Now.Location() != time.UTC || o.Now.Before(before) {
		t.Fatalf("now = %v", o.Now)
	}

	calls = stubApprove(t, decided("reject", false, "d1", ""))
	if code, _, errOut := run("approve", "open-case", "--reject"); code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
	if o := (*calls)[0].options; o.Decision != "reject" || o.Amend != "" {
		t.Fatalf("options = %+v", o)
	}
}

// A written approval runs the index once; the catch-up it starts with finds
// nothing here, and stdout carries only the decision.
func TestApproveReindexesAfterAWrittenApproval(t *testing.T) {
	w, _ := approveWorld(t, reconcileOptions{})
	stubApprove(t, decided("approve", true, "abc1234", ""))
	before, _ := readRegister(t, w)
	code, out, errOut := run("approve", "open-case")
	if code != 0 || errOut != qmdUpdated || out != "Fall open-case: approve\ncommittet als abc1234\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if after, _ := readRegister(t, w); after == before {
		t.Fatal("the register was not rewritten: no index run")
	}
}

// A rejection wrote no page, so nothing is indexed (arch 10.5).
func TestApproveDoesNotReindexAfterARejection(t *testing.T) {
	w, _ := approveWorld(t, reconcileOptions{})
	stubApprove(t, decided("reject", false, "d1", ""))
	before, stamp := readRegister(t, w)
	code, out, errOut := run("approve", "--reject", "open-case")
	if code != 0 || errOut != "" || out != "Fall open-case: reject\ncommittet als d1\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if after, again := readRegister(t, w); after != before || again != stamp {
		t.Fatal("the register was rewritten after a rejection")
	}
}

// A decision carried out scores 0 even when its commit did not land; the
// difference is a sentence on stderr, and the dropped claims are listed.
func TestApproveReportsAnUncommittedDecisionAndStillSucceeds(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	stubApprove(t, decided("approve", true, "", "kein Repository", "B2", "B3"))
	code, out, errOut := run("approve", "open-case")
	want := "Fall open-case: approve\n  verworfene Behauptung: B2\n  verworfene Behauptung: B3\n"
	if code != 0 || out != want || errOut != "geschrieben, aber nicht committet: kein Repository\n"+qmdUpdated {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}

	stubApprove(t, decided("reject", false, "", "kein Repository"))
	code, out, errOut = run("approve", "open-case", "--reject")
	if code != 0 || out != "Fall open-case: reject\n" || errOut != "entschieden, aber nicht committet: kein Repository\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
}

// The report names the case as apply read it (`result.case` is `case.id`),
// not the directory: only `--defer` prints the address.
func TestApproveReportsTheIDApplyRead(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	stubApprove(t, func(approveCall) (apply.Result, error) {
		return apply.Result{Case: maintenance.Case{ID: "old-name"}, Decision: "reject", Commit: "d1"}, nil
	})
	if code, out, _ := run("approve", "--reject", "open-case"); code != 0 || !strings.HasPrefix(out, "Fall old-name: reject\n") {
		t.Fatalf("exit = %d, stdout = %q", code, out)
	}
}

// A file outside the vault (the register of a readonly or out-of-tree area)
// is named under its own hint: the vault's auto-commit never sweeps it up.
func TestAnAbortNamesAFileOutsideTheVaultApart(t *testing.T) {
	outside := filepath.ToSlash(filepath.Join(t.TempDir(), "state", "areas", "project-ro", "_identities.tsv"))
	const outsideHint = "Hinweis: außerhalb des Vaults hat der Abbruch diese Dateien bereits geändert; " +
		"kein Commit erfasst sie:\n"
	t.Run("mixed", func(t *testing.T) {
		var out strings.Builder
		reportAbort(&out, &apply.ApplyError{Msg: "disk full",
			Dirty: []string{"90 Wiki/topics/thema.md", outside, "90 Wiki/log.md"}})
		want := "error: disk full\n" + dirtyHint + "  90 Wiki/topics/thema.md\n  90 Wiki/log.md\n" +
			outsideHint + "  " + outside + "\n"
		if out.String() != want {
			t.Fatalf("stderr = %q, want %q", out.String(), want)
		}
	})
	t.Run("all outside", func(t *testing.T) {
		var out strings.Builder
		reportAbort(&out, &apply.ApplyError{Msg: "disk full", Dirty: []string{outside}})
		want := "error: disk full\n" + outsideHint + "  " + outside + "\n"
		if out.String() != want {
			t.Fatalf("stderr = %q, want %q", out.String(), want)
		}
	})
}

// An abort names the files it already touched, whatever kind stopped it, and
// a refusal before any write says nothing about the vault.
func TestApproveNamesWhatAnAbortLeftBehind(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	dirty := []string{"95 Prüfzentrum/knowledge/x/case.toml", "90 Wiki/audit.md"}
	for _, failure := range []error{
		&apply.TargetMoved{ApplyError: apply.ApplyError{Msg: "abgebrochen", Dirty: dirty}},
		&apply.SourceMoved{ApplyError: apply.ApplyError{Msg: "abgebrochen", Dirty: dirty}},
		&apply.ProposalRefused{ApplyError: apply.ApplyError{Msg: "abgebrochen", Dirty: dirty}},
		&apply.ApplyError{Msg: "abgebrochen", Dirty: dirty},
	} {
		stubApprove(t, func(approveCall) (apply.Result, error) { return apply.Result{}, failure })
		code, out, errOut := run("approve", "open-case")
		want := "error: abgebrochen\n" + dirtyHint + "  " + dirty[0] + "\n  " + dirty[1] + "\n"
		if code != 1 || out != "" || errOut != want {
			t.Fatalf("%T: exit = %d, stdout = %q, stderr = %q", failure, code, out, errOut)
		}
	}
	for _, failure := range []error{
		&apply.ApplyError{Msg: "fremder Bereich"},
		&fs.PathError{Op: "open", Path: "x", Err: fs.ErrPermission},
	} {
		stubApprove(t, func(approveCall) (apply.Result, error) { return apply.Result{}, failure })
		code, _, errOut := run("approve", "open-case")
		if code != 1 || errOut != "error: "+failure.Error()+"\n" {
			t.Fatalf("%T: exit = %d, stderr = %q", failure, code, errOut)
		}
	}
}

// Putting a case off is the absence of a decision: apply is never called, the
// case file stays byte for byte, and the line names the directory.
func TestApproveDeferLeavesTheCaseWhereItIs(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	directory := placeCase(t, w, "project-a/open-case", standingCase("it's-old", "a.md", "due", ""), nil)
	stubApprove(t, nil)
	before, err := os.ReadFile(filepath.Join(directory, "case.toml"))
	if err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run("approve", "--defer", "open-case")
	if code != 0 || out != "Fall open-case zurückgestellt; er bleibt unverändert in der Warteschlange.\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if !strings.Contains(errOut, "calls itself \"it's-old\", but the case is addressed as 'open-case'") {
		t.Fatalf("stderr = %q", errOut)
	}
	if after, _ := os.ReadFile(filepath.Join(directory, "case.toml")); string(after) != string(before) {
		t.Fatal("the case file changed")
	}
	// `--reject=false` decides nothing, so it does not collide with `--defer`.
	if code, _, errOut := run("approve", "open-case", "--reject=false", "--defer"); code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
}

// A broken case file is named rather than deferred.
func TestApproveDeferNamesABrokenCaseFile(t *testing.T) {
	w := newReconcileWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/open-case", "kaputt = [", nil)
	stubApprove(t, nil)
	code, out, errOut := run("approve", "open-case", "--defer")
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") || !strings.Contains(errOut, "not valid TOML") {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
}

// What cannot address a case ends before apply: an unknown id, a vault
// without a review centre, a registry that does not read.
func TestApproveRefusesWhatAddressesNoCase(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	stubApprove(t, nil)
	if code, _, errOut := run("approve", "gibt-es-nicht"); code != 1 || !strings.HasPrefix(errOut, "error: no case named 'gibt-es-nicht'") {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}

	newReconcileWorld(t, reconcileOptions{NoReview: true})
	if code, _, errOut := run("approve", "open-case"); code != 1 || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
}

// The account running the command decides, without its domain -- the name
// `getpass.getuser()` gives. An account that cannot be read decides nothing.
func TestApproveNamesTheAccountAsTheReviewer(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	calls := stubApprove(t, decided("reject", false, "d1", ""))
	stubUser(t, "christoph", nil)
	run("approve", "--reject", "open-case")
	if got := (*calls)[0].options.Reviewer; got != "human:christoph" {
		t.Fatalf("reviewer = %q", got)
	}

	stubUser(t, "", errors.New("no account"))
	code, _, errOut := run("approve", "--reject", "open-case")
	if code != 1 || errOut != "error: no account\n" || len(*calls) != 1 {
		t.Fatalf("exit = %d, stderr = %q, calls = %d", code, errOut, len(*calls))
	}
}

// argparse's refusals: no id, two ids, two decisions, an unknown flag and the
// terminator as the amendment's value -- exit 2 and a line on stderr.
func TestApproveRefusesBadArguments(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	stubApprove(t, nil)
	for _, args := range [][]string{
		{"approve"},
		{"approve", "a", "b"},
		{"approve", "a", "--reject", "--defer"},
		{"approve", "--amend", "x.md", "a", "--reject"},
		{"approve", "--amend", "x.md", "--defer", "a"},
		{"approve", "--nope", "a"},
		{"approve", "open-case", "--amend", "--"},
		{"approve", "open-case", "--amend", "--reject"},
	} {
		code, out, errOut := run(args...)
		if code != 2 || out != "" || errOut == "" {
			t.Fatalf("%q: exit = %d, stdout = %q, stderr = %q", args, code, out, errOut)
		}
	}
	_, _, errOut := run("approve", "a", "--reject", "--defer")
	if errOut != "loomux approve: argument --defer: not allowed with argument --reject\n" {
		t.Fatalf("stderr = %q", errOut)
	}
}

// catchUpFailed is `_technical_update`'s warning for a catch-up that did not
// go through, around the error it names, with `brain` read as `loomux`.
func catchUpFailed(cause string) string {
	return "warning: die Aufholung vor der technischen Aktualisierung ist fehlgeschlagen (" + cause +
		"). Die Freigabe steht, aber der Suchindex des gesamten Vaults ist ab jetzt veraltet: " +
		"die Suche liefert weiter den alten Text. Ursache beheben, dann `loomux reconcile` " +
		"und danach `loomux reindex`.\n"
}

// A catch-up that fails after the decision is named in the reference's words
// and stops the update: nothing is indexed, and the approval still scores 0.
func TestApproveStopsTheUpdateWhenTheCatchUpFails(t *testing.T) {
	w, _ := approveWorld(t, reconcileOptions{})
	before, stamp := readRegister(t, w)
	stubApprove(t, func(call approveCall) (apply.Result, error) {
		writeFile(t, filepath.Join(w.Area, ".loomux", "config.toml"), "[area\n")
		return decided("approve", true, "abc1234", "")(call)
	})
	code, out, errOut := run("approve", "open-case")
	if code != 0 || out != "Fall open-case: approve\ncommittet als abc1234\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	_, _, cause := reviewCentre()
	if cause == nil || errOut != catchUpFailed(cause.Error()) {
		t.Fatalf("stderr = %q, cause = %v", errOut, cause)
	}
	if after, again := readRegister(t, w); after != before || again != stamp {
		t.Fatal("the register was rewritten past a failed catch-up")
	}
}

// Without a review centre the catch-up has failed too, on the approve path:
// the reference catches NoReviewCentreError as a ReconcileError and does not
// index. Only `reindex` treats that state as a warning and indexes on.
func TestApproveStopsTheUpdateWithoutAReviewCentre(t *testing.T) {
	w, _ := approveWorld(t, reconcileOptions{})
	before, stamp := readRegister(t, w)
	stubApprove(t, func(call approveCall) (apply.Result, error) {
		writeFile(t, filepath.Join(w.Area, ".loomux", "config.toml"),
			"[area]\nscope = \"project/a\"\n\n[layout]\nwiki = \"wiki\"\n\n[index]\ninclude = [\"**/*.md\"]\n")
		return decided("approve", true, "abc1234", "")(call)
	})
	code, _, errOut := run("approve", "open-case")
	_, _, cause := reviewCentre()
	if code != 0 || cause == nil || errOut != catchUpFailed(cause.Error()) {
		t.Fatalf("exit = %d, stderr = %q, cause = %v", code, errOut, cause)
	}
	if after, again := readRegister(t, w); after != before || again != stamp {
		t.Fatal("the register was rewritten without a review centre")
	}
}

// A catch-up that opens a case lists it on stderr and indexes on -- after
// one catch-up only: a second pass would list the same case again.
func TestApproveListsTheCasesTheCatchUpOpened(t *testing.T) {
	w, _ := approveWorld(t, reconcileOptions{Cite: true})
	stubApprove(t, decided("approve", true, "abc1234", ""))
	before, _ := readRegister(t, w)
	code, out, errOut := run("approve", "open-case")
	if code != 0 || out != "Fall open-case: approve\ncommittet als abc1234\n" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if !strings.HasPrefix(errOut, "warning: 1 neuer Fall durch die Aufholung eröffnet:\n  ") ||
		!strings.HasSuffix(errOut, "\tproject/a\tpage.md\tsource_changed\n"+qmdUpdated) ||
		strings.Count(errOut, "durch die Aufholung eröffnet") != 1 {
		t.Fatalf("stderr = %q", errOut)
	}
	if after, _ := readRegister(t, w); after == before {
		t.Fatal("the register was not rewritten")
	}
}

// An index run that fails after a green catch-up is named in the reference's
// words; the approval stands and scores 0.
func TestApproveNamesAFailedIndexRun(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	port := search.NewFakePort()
	port.Refreshes = []error{errors.New("engine is asleep")}
	stubBrainStatusPort(t, port)
	stubApprove(t, decided("approve", true, "abc1234", ""))
	code, _, errOut := run("approve", "open-case")
	if code != 0 || !strings.HasSuffix(errOut, technicalUpdateWarning) {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
}

// The wiring against the real apply: a rejection in a vault without a
// repository is decided, its case directory leaves the review centre, and the
// missing commit is a sentence rather than a failure.
func TestApproveRejectsThroughApply(t *testing.T) {
	_, directory := approveWorld(t, reconcileOptions{})
	code, out, errOut := run("approve", "--reject", "open-case")
	if code != 0 || out != "Fall open-case: reject\n" || !strings.HasPrefix(errOut, "entschieden, aber nicht committet: ") {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q", code, out, errOut)
	}
	if _, err := os.Stat(directory); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the case directory is still there: %v", err)
	}
}

// Past `--` a word is the id, never a decision: `--defer` there is a case
// name and does not collide with the `--reject` before it.
func TestApproveReadsADecisionPastTheTerminatorAsTheID(t *testing.T) {
	w, _ := approveWorld(t, reconcileOptions{})
	placeCase(t, w, "project-a/--defer", standingCase("--defer", "b.md", "due", ""), nil)
	calls := stubApprove(t, decided("reject", false, "d1", ""))
	code, _, errOut := run("approve", "--reject", "--", "--defer")
	if code != 0 || len(*calls) != 1 || filepath.Base(filepath.Dir((*calls)[0].casePath)) != "--defer" {
		t.Fatalf("exit = %d, stderr = %q, calls = %+v", code, errOut, *calls)
	}
}

// An empty `--amend=` still names a file, `.` as Python's `Path("")` does,
// and never falls back to approving the case's own proposal.
func TestApproveTakesAnEmptyAmendmentAsTheCurrentDirectory(t *testing.T) {
	approveWorld(t, reconcileOptions{})
	calls := stubApprove(t, decided("reject", false, "d1", ""))
	if code, _, errOut := run("approve", "open-case", "--amend="); code != 0 || (*calls)[0].options.Amend != "." {
		t.Fatalf("exit = %d, stderr = %q, calls = %+v", code, errOut, *calls)
	}
}
