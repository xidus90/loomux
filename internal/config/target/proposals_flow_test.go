package target

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var proposalTime = time.Date(2026, 9, 24, 10, 15, 30, 0, time.UTC)

func TestWordQuotesWhatIsNotOneWord(t *testing.T) {
	for in, want := range map[string]string{"de": "de", "": `""`, "a b": `"a b"`, "x\x1b": `"x\x1b"`, "ü": "ü", `a"b`: `"a\"b"`} {
		if got := Word(in); got != want {
			t.Errorf("Word(%q) = %q, want %q", in, got, want)
		}
	}
	if got := (Proposal{Op: "set", Key: "a", Input: "b c"}).Describe(); got != `set a "b c"` {
		t.Errorf("set: %q", got)
	}
	if got := (Proposal{Op: "unset", Key: "a"}).Describe(); got != "unset a" {
		t.Errorf("unset: %q", got)
	}
}

func TestStoreStampsTheProposalWithTimeAndTarget(t *testing.T) {
	root, tgt := projectTarget(t, "")
	id, err := tgt.Store(proposalTime.In(time.FixedZone("x", 3600)), Proposal{Op: "set", Key: "commit.language", Input: "de"})
	if err != nil || id != "20260924T101530Z-001" {
		t.Fatalf("%s %v", id, err)
	}
	p, err := tgt.Load(id)
	if err != nil || p.Created != "2026-09-24T10:15:30Z" || p.Target != "project" || p.ID != id {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".loomux", "state", "config", "proposals", id+".json")); err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()
	global := Global(state)
	gid, err := global.Store(proposalTime, Proposal{Op: "unset", Key: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if g, err := global.Load(gid); err != nil || g.Target != "global" {
		t.Fatalf("%+v %v", g, err)
	}
	if _, err := os.Stat(filepath.Join(state, "config", "proposals", gid+".json")); err != nil {
		t.Fatal(err)
	}
}

func TestIDsListOpenProposalsOldestFirst(t *testing.T) {
	_, tgt := projectTarget(t, "")
	if ids, err := tgt.IDs(); err != nil || len(ids) != 0 || tgt.OpenCount() != 0 {
		t.Fatalf("%v %v", ids, err)
	}
	for _, key := range []string{"a", "b"} {
		if _, err := tgt.Store(proposalTime, Proposal{Op: "unset", Key: key}); err != nil {
			t.Fatal(err)
		}
	}
	// Neither a directory nor another file type is a proposal.
	writeFile(t, filepath.Join(tgt.dir(), "note.txt"), "")
	if err := os.Mkdir(filepath.Join(tgt.dir(), "sub.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	ids, err := tgt.IDs()
	if err != nil || strings.Join(ids, " ") != "20260924T101530Z-001 20260924T101530Z-002" || tgt.OpenCount() != 2 {
		t.Fatalf("%v %v", ids, err)
	}
	// A directory that lists with an error other than absence.
	_, blocked := projectTarget(t, "")
	unlistable(t, blocked.dir())
	if _, err := blocked.IDs(); err == nil || blocked.OpenCount() != 0 {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadNamesTheFileThatDoesNotRead(t *testing.T) {
	_, tgt := projectTarget(t, "")
	if _, err := tgt.Load("missing"); err == nil || !strings.Contains(err.Error(), "proposal missing:") {
		t.Fatalf("err = %v", err)
	}
	writeFile(t, tgt.file("bad"), "{")
	if _, err := tgt.Load("bad"); err == nil || !strings.Contains(err.Error(), "proposal bad:") {
		t.Fatalf("err = %v", err)
	}
	// The file name is the id, whatever the file says.
	writeFile(t, tgt.file("a-1"), `{"id":"other","op":"unset","key":"k"}`)
	if p, err := tgt.Load("a-1"); err != nil || p.ID != "a-1" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestChooseResolvesOneIdOrAll(t *testing.T) {
	_, tgt := projectTarget(t, "")
	id, _ := tgt.Store(proposalTime, Proposal{Op: "unset", Key: "a"})
	if ids, err := tgt.Choose(id, false); err != nil || len(ids) != 1 || ids[0] != id {
		t.Fatalf("%v %v", ids, err)
	}
	if ids, err := tgt.Choose("", true); err != nil || len(ids) != 1 {
		t.Fatalf("%v %v", ids, err)
	}
	if _, err := tgt.Choose("../config", false); err == nil || !strings.Contains(err.Error(), "no open proposal") {
		t.Fatalf("err = %v", err)
	}
	_, blocked := projectTarget(t, "")
	unlistable(t, blocked.dir())
	if _, err := blocked.Choose("x", false); err == nil || strings.Contains(err.Error(), "no open proposal") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveComputesAgainstTheFileAsItIs(t *testing.T) {
	_, tgt := projectTarget(t, "")
	text := "[commit]\nlanguage = \"en\"\n"
	next, diff, err := tgt.Resolve(text, Proposal{Op: "set", Key: "commit.language", Input: "de"})
	if err != nil || next != "[commit]\nlanguage = \"de\"\n" || !strings.Contains(diff, `+ language = "de"`) {
		t.Fatalf("%q %q %v", next, diff, err)
	}
	if _, diff, err = tgt.Resolve("[commit]\nlanguage = \"de\"\n", Proposal{Op: "unset", Key: "commit.threshold"}); err != nil || diff != "" {
		t.Fatalf("already so: %q %v", diff, err)
	}
	if _, _, err = tgt.Resolve(text, Proposal{Op: "drop\x1b", Key: "k"}); err == nil || !strings.Contains(err.Error(), `unknown operation "drop\x1b"`) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err = tgt.Resolve(text, Proposal{Op: "set", Key: "nope.nothing"}); err == nil {
		t.Fatal("an unknown key was resolved")
	}
}

func TestProposalsShowEachWithItsOwnDiffOrReason(t *testing.T) {
	_, tgt := projectTarget(t, "[commit]\nlanguage = \"de\"\n")
	if views, err := tgt.Proposals(); err != nil || views == nil || len(views) != 0 {
		t.Fatalf("%v %v", views, err)
	}
	writeFile(t, tgt.file("a-1"), `{"op":"set","key":"commit.threshold","input":"4"}`)
	writeFile(t, tgt.file("a-2"), `{"op":"set","key":"commit.language","input":"de"}`)
	writeFile(t, tgt.file("a-3"), `{"op":"set","key":"nope.nothing"}`)
	writeFile(t, tgt.file("a-4"), "{")
	views, err := tgt.Proposals()
	if err != nil || len(views) != 4 {
		t.Fatalf("%v %v", views, err)
	}
	if !strings.Contains(views[0].Diff, "+ threshold = 4") || views[0].Error != "" || views[0].ID != "a-1" {
		t.Errorf("a-1: %+v", views[0])
	}
	if views[1].Diff != "" || views[1].Error != "" {
		t.Errorf("a-2 is already so: %+v", views[1])
	}
	if !strings.Contains(views[2].Error, "unknown key") {
		t.Errorf("a-3: %+v", views[2])
	}
	if views[2].Unreadable || !views[3].Unreadable {
		t.Errorf("only the file that did not load is unreadable: %+v %+v", views[2], views[3])
	}
	if views[3].ID != "a-4" || !strings.Contains(views[3].Error, "proposal a-4:") {
		t.Errorf("a-4: %+v", views[3])
	}
}

func TestProposalsReportWhatCannotBeListedOrRead(t *testing.T) {
	_, blocked := projectTarget(t, "")
	unlistable(t, blocked.dir())
	if _, err := blocked.Proposals(); err == nil {
		t.Error("an unlistable directory was listed")
	}
	_, unreadable := projectTarget(t, "")
	if err := os.Remove(unreadable.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(unreadable.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := unreadable.Proposals(); err == nil {
		t.Error("an unreadable file was read")
	}
}

func TestApplyWritesThenRemovesTheProposal(t *testing.T) {
	_, tgt := projectTarget(t, "[commit]\nlanguage = \"de\"\n")
	id, _ := tgt.Store(proposalTime, Proposal{Op: "set", Key: "commit.threshold", Input: "4"})
	if err := tgt.Apply(id); err != nil {
		t.Fatal(err)
	}
	if text, _ := tgt.Read(); text != "[commit]\nlanguage = \"de\"\nthreshold = 4\n" {
		t.Fatalf("file:\n%s", text)
	}
	if tgt.OpenCount() != 0 {
		t.Fatal("the proposal stayed")
	}
}

func TestApplyOfWhatIsAlreadySoOnlyRemovesTheProposal(t *testing.T) {
	_, tgt := projectTarget(t, "[commit]\nlanguage = \"de\"\n")
	id, _ := tgt.Store(proposalTime, Proposal{Op: "set", Key: "commit.language", Input: "de"})
	if err := tgt.Apply(id); err != nil || tgt.OpenCount() != 0 {
		t.Fatalf("%v", err)
	}
	if text, _ := tgt.Read(); text != "[commit]\nlanguage = \"de\"\n" {
		t.Fatalf("file:\n%s", text)
	}
}

func TestApplyKeepsAProposalThatDoesNotHold(t *testing.T) {
	_, tgt := projectTarget(t, "")
	if err := tgt.Apply("missing"); err == nil {
		t.Error("a proposal that is not there was applied")
	}
	writeFile(t, tgt.file("a-1"), `{"op":"set","key":"nope.nothing","input":"x"}`)
	err := tgt.Apply("a-1")
	if err == nil || !strings.Contains(err.Error(), "proposal a-1 (set nope.nothing x): unknown key") || tgt.OpenCount() != 1 {
		t.Fatalf("err = %v", err)
	}
	// A file that does not read stops it before anything is computed.
	_, unreadable := projectTarget(t, "")
	if err := os.Remove(unreadable.Path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(unreadable.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, unreadable.file("a-1"), `{"op":"unset","key":"commit.threshold"}`)
	if err := unreadable.Apply("a-1"); err == nil || unreadable.OpenCount() != 1 {
		t.Fatalf("err = %v", err)
	}
}

// A change of the file after Prepare read it is the other writer's: the
// prepared text would take it back unseen.
func TestApplyPendingRefusesAFileThatChangedSincePrepare(t *testing.T) {
	_, tgt := projectTarget(t, "[commit]\nlanguage = \"de\"\n")
	id, _ := tgt.Store(proposalTime, Proposal{Op: "set", Key: "commit.threshold", Input: "4"})
	pending, err := tgt.Prepare(id)
	if err != nil {
		t.Fatal(err)
	}
	other := "[commit]\nlanguage = \"en\"\n"
	writeFile(t, tgt.Path, other)
	if err := tgt.ApplyPending(pending); err == nil || !strings.Contains(err.Error(), "changed since it was read") {
		t.Fatalf("err = %v", err)
	}
	if text, _ := tgt.Read(); text != other || tgt.OpenCount() != 1 {
		t.Fatalf("file %q, open %d", text, tgt.OpenCount())
	}
}

func TestApplyKeepsTheProposalWhenTheWriteFails(t *testing.T) {
	restore := replaceText
	replaceText = func(string, string) error { return errors.New("disk full") }
	t.Cleanup(func() { replaceText = restore })
	_, tgt := projectTarget(t, "")
	id, _ := tgt.Store(proposalTime, Proposal{Op: "set", Key: "commit.threshold", Input: "4"})
	if err := tgt.Apply(id); err == nil || !strings.Contains(err.Error(), "disk full") || tgt.OpenCount() != 1 {
		t.Fatalf("err = %v", err)
	}
}

func TestRejectRemovesTheProposalWithoutApplying(t *testing.T) {
	_, tgt := projectTarget(t, "")
	id, _ := tgt.Store(proposalTime, Proposal{Op: "set", Key: "commit.threshold", Input: "4"})
	if err := tgt.Reject(id); err != nil || tgt.OpenCount() != 0 {
		t.Fatalf("%v", err)
	}
	if text, _ := tgt.Read(); text != "" {
		t.Fatalf("file:\n%s", text)
	}
	if err := tgt.Reject(id); err == nil {
		t.Error("a proposal that is gone was rejected")
	}
}
