package cases_test

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/testlock"
)

const (
	stamp     = "2026-09-22T07:32:30.995024+00:00"
	caseDir   = "repo-a/review/project-a/a-2026-09-22-5bd8/"
	knownID   = "0A00000000000000000000000A"
	mintedID  = "01M340FYACT4WSDTGY9T6VPBQD"
	mintedID2 = "01M340FYACT4WSDTGY9T6VPBQE"
	header    = "doc_id\tpfad\tcontent_hash\trevision\n"
)

// writeTree lays files down under root, keyed by slash-separated path.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func tree(pairs ...string) map[string][]byte {
	files := map[string][]byte{}
	for i := 0; i < len(pairs); i += 2 {
		files[pairs[i]] = []byte(pairs[i+1])
	}
	return files
}

func TestNormalizeStateTokenizesTheRunStampAndTheDayOfEveryCaseID(t *testing.T) {
	got := cases.NormalizeState(nil, tree(
		"maintenance/last-run.txt", stamp+"\n",
		caseDir+"case.toml", "id = \"a-2026-09-22-5bd8\"\ncreated = "+stamp+"\n",
		caseDir+"package.md", "case: a-2026-09-22-5bd8\ngenerated.at: "+stamp+"\n",
		// Another day and another shape stay as they are.
		"repo-a/review/project-a/a-2000-01-01-5bd8/case.toml", "id = \"a-2000-01-01-5bd8\"\n",
		"notes.md", "on 2026-09-22-later and 2026-09-22-5bd8x\n",
	))
	want := tree(
		"maintenance/last-run.txt", "{{NOW}}\n",
		"repo-a/review/project-a/a-{{TODAY}}-5bd8/case.toml", "id = \"a-{{TODAY}}-5bd8\"\ncreated = {{NOW}}\n",
		"repo-a/review/project-a/a-{{TODAY}}-5bd8/package.md", "case: a-{{TODAY}}-5bd8\ngenerated.at: {{NOW}}\n",
		"repo-a/review/project-a/a-2000-01-01-5bd8/case.toml", "id = \"a-2000-01-01-5bd8\"\n",
		"notes.md", "on 2026-09-22-later and 2026-09-22-5bd8x\n",
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// Python's isoformat drops the fraction when it is zero; that is the one other
// spelling of the stamp a correct writer can produce.
func TestNormalizeStateTakesAStampWithoutAFraction(t *testing.T) {
	got := cases.NormalizeState(nil, tree("maintenance/last-run.txt", "2026-09-22T07:32:30+00:00\n"))
	if string(got["maintenance/last-run.txt"]) != "{{NOW}}\n" {
		t.Fatalf("got %q", got)
	}
}

// A stamp in any other shape is left to fail the comparison: a local zone,
// a Z, a millisecond fraction or a date alone is a writer that no longer
// writes what the reference writes, and the normalization must not hide it.
func TestNormalizeStateLeavesAStampOfAnotherShapeAlone(t *testing.T) {
	for _, text := range []string{
		"2026-09-22T09:32:30.995024+02:00\n",
		"2026-09-22T07:32:30.995024Z\n",
		"2026-09-22T07:32:30.995+00:00\n",
		"2026-09-22\n",
	} {
		files := tree("maintenance/last-run.txt", text, caseDir+"case.toml", "id = \"a-2026-09-22-5bd8\"\n")
		got := cases.NormalizeState(nil, files)
		if !reflect.DeepEqual(got, tree("maintenance/last-run.txt", text, caseDir+"case.toml", "id = \"a-2026-09-22-5bd8\"\n")) {
			t.Errorf("%q: got %q", text, got)
		}
	}
}

func TestNormalizeStateTokenizesTheModificationTimeOfTheStatCache(t *testing.T) {
	cache := "pfad\tmtime_ns\tsize\nnotes/source.md\t1790062348871195500\t41\n" +
		"broken\tx\t1\nempty\t\t1\nshort\n" +
		// Seconds and milliseconds are no nanosecond count, and stay.
		"seconds\t1790062348\t41\nmillis\t1790062348871\t41\n"
	got := cases.NormalizeState(nil, tree(
		"maintenance/project-a/stats.tsv", cache,
		// Only the cache of a pass has the column; the same shape elsewhere is data.
		"repo-a/stats.tsv", cache,
	))
	want := tree(
		"maintenance/project-a/stats.tsv", "pfad\tmtime_ns\tsize\nnotes/source.md\t{{MTIME}}\t41\n"+
			"broken\tx\t1\nempty\t\t1\nshort\n"+
			"seconds\t1790062348\t41\nmillis\t1790062348871\t41\n",
		"repo-a/stats.tsv", cache,
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestNormalizeStateTokenizesTheDocIDsARunMintedAndSortsTheirRows(t *testing.T) {
	world := tree(
		"repo-a/_identities.tsv", header+knownID+"\tnotes/source.md\tsha256:1\t1\n",
		// A file that is no register names no id, whatever it holds.
		"repo-a/notes/source.md", mintedID+"\tnotes/source.md\n",
	)
	got := cases.NormalizeState(world, tree(
		"repo-a/_identities.tsv", header+
			mintedID2+"\twiki/z.md\tsha256:3\t1\n"+
			mintedID+"\twiki/page.md\tsha256:2\t1\n"+
			knownID+"\tnotes/source.md\tsha256:1\t2\n",
		// Without its last line feed, and it stays without it.
		"repo-b/_identities.tsv", header+"01M340FYACT4WSDTGY9T6VPBQF\tx.md\tsha256:4\t1",
		// A register without a new id keeps its order, broken rows and all.
		"areas/notes/_identities.tsv", header+knownID+"\tb.md\tsha256:1\t1\n"+"short row\n",
	))
	want := tree(
		"repo-a/_identities.tsv", header+
			knownID+"\tnotes/source.md\tsha256:1\t2\n"+
			"{{DOCID}}\twiki/page.md\tsha256:2\t1\n"+
			"{{DOCID}}\twiki/z.md\tsha256:3\t1\n",
		"repo-b/_identities.tsv", header+"{{DOCID}}\tx.md\tsha256:4\t1",
		"areas/notes/_identities.tsv", header+knownID+"\tb.md\tsha256:1\t1\n"+"short row\n",
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// A doc id is unique across a vault. An id minted onto two rows -- in one
// register or in two -- is the duplicate the fold would hide, so it stays.
func TestNormalizeStateLeavesAnIDMintedTwiceAlone(t *testing.T) {
	twice := header + mintedID + "\tx.md\tsha256:1\t1\n" + mintedID + "\ty.md\tsha256:2\t1\n"
	other := header + mintedID2 + "\tz.md\tsha256:3\t1\n"
	elsewhere := header + mintedID2 + "\tw.md\tsha256:4\t1\n"
	got := cases.NormalizeState(nil, tree(
		"repo-a/_identities.tsv", twice,
		"repo-b/_identities.tsv", other,
		"areas/notes/_identities.tsv", elsewhere,
	))
	want := tree(
		"repo-a/_identities.tsv", twice,
		"repo-b/_identities.tsv", other,
		"areas/notes/_identities.tsv", elsewhere,
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}

// Only a well-formed id is a minted one; anything else in the column is a
// register loomux wrote wrongly, and it must stay visible.
func TestNormalizeStateLeavesAnIDOfAnotherShapeAlone(t *testing.T) {
	text := header + "not-an-id\twiki/page.md\tsha256:2\t1\n" + "01M340FYACT4WSDTGY9T6VPBQU\tx.md\tsha256:2\t1\n"
	got := cases.NormalizeState(nil, tree("repo-a/_identities.tsv", text))
	if string(got["repo-a/_identities.tsv"]) != text {
		t.Fatalf("got %q", got)
	}
}

func TestRunCaseWithNormalizesBothSidesOfTheWorldAfter(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reconcile", "stamp")
	writeTree(t, dir, map[string]string{
		"cmd":                                  "loomux reconcile\n",
		"exit":                                 "0\n",
		"stdout":                               "",
		"compare":                              "message\n",
		"world/keep.txt":                       "k",
		"world_after/keep.txt":                 "k",
		"world_after/maintenance/last-run.txt": "2000-01-01T00:00:00+00:00\n",
	})
	c, err := cases.LoadCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	run := func(_ []string, world string, _ io.Reader, _, _ io.Writer) int {
		writeTree(t, world, map[string]string{"maintenance/last-run.txt": "2026-09-22T07:32:30+00:00\n"})
		return 0
	}
	outcome, err := cases.RunCaseWith(c, run, cases.NormalizeState)
	if err != nil || !outcome.Passed {
		t.Fatalf("normalized: %+v, %v", outcome, err)
	}
	outcome, err = cases.RunCase(c, run)
	if err != nil || outcome.Passed {
		t.Fatalf("plain: %+v, %v", outcome, err)
	}
}

// A data case prints the case id, day and all; each side's day comes from its
// own tree, so a replay on another day than the recording's still matches --
// and a stdout that differs in anything else still does not.
func TestRunCaseWithFoldsTheDayOnStdout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reconcile", "day")
	writeTree(t, dir, map[string]string{
		"cmd":                                  "loomux reconcile\n",
		"exit":                                 "0\n",
		"stdout":                               "  a-2000-01-01-5bd8\tproject/a\n1 Fall\n",
		"world/keep.txt":                       "k",
		"world_after/keep.txt":                 "k",
		"world_after/maintenance/last-run.txt": "2000-01-01T00:00:00+00:00\n",
	})
	c, err := cases.LoadCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	printing := func(line string) cases.RunFunc {
		return func(_ []string, world string, _ io.Reader, stdout, _ io.Writer) int {
			writeTree(t, world, map[string]string{"maintenance/last-run.txt": "2026-09-22T07:32:30+00:00\n"})
			io.WriteString(stdout, line)
			return 0
		}
	}
	today := printing("  a-2026-09-22-5bd8\tproject/a\n1 Fall\n")
	if outcome, err := cases.RunCaseWith(c, today, cases.NormalizeState); err != nil || !outcome.Passed {
		t.Fatalf("normalized: %+v, %v", outcome, err)
	}
	if outcome, err := cases.RunCase(c, today); err != nil || outcome.Passed {
		t.Fatalf("plain: %+v, %v", outcome, err)
	}
	counted := printing("  a-2026-09-22-5bd8\tproject/a\n2 Fälle\n")
	outcome, err := cases.RunCaseWith(c, counted, cases.NormalizeState)
	if err != nil || outcome.Passed || !strings.HasPrefix(outcome.Mismatches[0], "stdout mismatch") {
		t.Fatalf("another count: %+v, %v", outcome, err)
	}
}

// A recording without a world_after says the run left the world as it found
// it; a normalized replay holds loomux to that, and a plain one does not look.
func TestRunCaseWithHoldsACaseWithoutAWorldAfterToItsWorld(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reconcile", "refused")
	writeTree(t, dir, map[string]string{
		"cmd": "loomux reconcile\n", "exit": "1\n", "stdout": "", "world/registry.toml": "at {{WORLD}}\n",
	})
	c, err := cases.LoadCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	quiet := func([]string, string, io.Reader, io.Writer, io.Writer) int { return 1 }
	outcome, err := cases.RunCaseWith(c, quiet, cases.NormalizeState)
	if err != nil || !outcome.Passed {
		t.Fatalf("a run that writes nothing: %+v, %v", outcome, err)
	}
	writes := func(_ []string, world string, _ io.Reader, _, _ io.Writer) int {
		writeTree(t, world, map[string]string{"registry.lock": ""})
		return 1
	}
	outcome, err = cases.RunCaseWith(c, writes, cases.NormalizeState)
	if err != nil || outcome.Passed || outcome.Mismatches[0] != "unexpected extra file in actual: registry.lock" {
		t.Fatalf("a run that writes: %+v, %v", outcome, err)
	}
	if outcome, err = cases.RunCase(c, writes); err != nil || !outcome.Passed {
		t.Fatalf("plain: %+v, %v", outcome, err)
	}
}

func TestRunCaseWithReportsAWorldItCannotRead(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reconcile", "stamp")
	writeTree(t, dir, map[string]string{
		"cmd": "loomux reconcile\n", "exit": "0\n", "stdout": "", "world/x": "x", "world_after/x": "x",
	})
	c, err := cases.LoadCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	// The world is read after the run, for the ids it already held: a world
	// the OS stops handing out by then is a comparison that cannot be made.
	run := func([]string, string, io.Reader, io.Writer, io.Writer) int {
		testlock.Lock(t, filepath.Join(dir, "world", "x"))
		return 0
	}
	if _, err := cases.RunCaseWith(c, run, cases.NormalizeState); err == nil {
		t.Fatal("want error")
	}
}

const olderLog = "# Log\n\n- 2026-01-01 — `a.md`: 1 Behauptung(en) eingearbeitet (Fall `w`)\n"

// approvalTree is what an approve of case x on a.md leaves behind: a new audit
// block under stamp below older, a new log line of stamp's day, and the page's
// frontmatter advanced to stamp with reviewer's `verified` entry, both spelled
// the way PyYAML's safe_dump spells them.
func approvalTree(older, stamp, reviewer string) map[string][]byte {
	return tree(
		"vault/audit.md", older+"\n## "+stamp+" — a.md (Fall `x`)\n\n"+
			"- vorgeschlagen: 1 Behauptung(en), 0 ohne Beleg\n"+
			"- entschieden: freigegeben durch "+reviewer+"\n"+
			"- tatsächlich geändert: a.md, 1 Behauptung(en) eingearbeitet\n",
		"vault/log.md", olderLog+"\n- "+stamp[:10]+" — `a.md`: 1 Behauptung(en) eingearbeitet (Fall `x`)\n",
		"vault/wiki/a.md", "---\ngenerated:\n  at: "+stamp+"\nverified:\n- by: human:carol\n  at: 2026-01-01T00:00:00+00:00\n"+
			"- by: "+reviewer+"\n  at: '"+stamp+"'\n---\nwritten by human:dave\n",
	)
}

func TestNormalizeStateFoldsTheStampOfAnApproval(t *testing.T) {
	const older = "## 2026-01-01T00:00:00+00:00 — a.md (Fall `x`)\n"
	world := map[string][]byte{"vault/audit.md": []byte(older)}
	left := cases.NormalizeState(world, approvalTree(older, "2026-09-22T08:16:27.936837+00:00", "human:alice"))
	right := cases.NormalizeState(world, approvalTree(older, "2026-09-23T10:00:00+00:00", "human:bob"))
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("trees differ after normalizing:\n%s\n%s", left["vault/audit.md"], right["vault/audit.md"])
	}
	if !strings.HasPrefix(string(left["vault/audit.md"]), older) {
		t.Fatal("an older audit block was folded")
	}
}

// The places an approval spells its stamp, its day and its reviewer, and
// nothing beside them: an older `verified` entry keeps its time, the body of a
// page its reviewer, another file its day and a stamp of another shape, an
// older log line its date, and a page whose frontmatter never closes its
// reviewer.
func TestNormalizeStateFoldsAnApprovalExactlyWhereItWrites(t *testing.T) {
	// An older line of the same day stays: only the new line is this run's.
	const sameDay = "- 2026-09-22 — `b.md`: 1 Behauptung(en) eingearbeitet (Fall `v`)\n"
	world := tree("vault/audit.md", "# Audit\n", "vault/log.md", olderLog+sameDay)
	files := approvalTree("# Audit\n", "2026-09-22T08:16:27.936837+00:00", "human:alice")
	files["vault/log.md"] = []byte(olderLog + sameDay + "\n- 2026-09-22 — `a.md`: 1 Behauptung(en) eingearbeitet (Fall `x`)\n")
	files["vault/notes.md"] = []byte("on 2026-09-22 at 2026-09-22T08:16:27.936837Z\n")
	files["vault/broken.md"] = []byte("---\nby: human:erin\n")
	got := cases.NormalizeState(world, files)
	want := tree(
		"vault/notes.md", "on 2026-09-22 at 2026-09-22T08:16:27.936837Z\n",
		"vault/broken.md", "---\nby: human:erin\n",
		"vault/audit.md", "# Audit\n\n## {{NOW}} — a.md (Fall `x`)\n\n"+
			"- vorgeschlagen: 1 Behauptung(en), 0 ohne Beleg\n"+
			"- entschieden: freigegeben durch human:{{USER}}\n"+
			"- tatsächlich geändert: a.md, 1 Behauptung(en) eingearbeitet\n",
		"vault/log.md", olderLog+sameDay+"\n- {{TODAY}} — `a.md`: 1 Behauptung(en) eingearbeitet (Fall `x`)\n",
		"vault/wiki/a.md", "---\ngenerated:\n  at: {{NOW}}\nverified:\n- by: human:{{USER}}\n  at: 2026-01-01T00:00:00+00:00\n"+
			"- by: human:{{USER}}\n  at: '{{NOW}}'\n---\nwritten by human:dave\n",
	)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// A reviewer that is not `human:` is not the account running the command, and
// a stamp with a Z is not the isoformat() the reference writes: both stay to
// fail the comparison.
func TestNormalizeStateLeavesAnApprovalOfAnotherShapeAlone(t *testing.T) {
	world := tree("vault/audit.md", "# Audit\n", "vault/log.md", olderLog)
	for _, stamp := range []string{"2026-09-22T08:16:27Z", "2026-09-22T10:16:27.936837+02:00"} {
		got := cases.NormalizeState(world, approvalTree("# Audit\n", stamp, "human:alice"))
		for name, data := range got {
			if strings.Contains(string(data), "{{NOW}}") || strings.Contains(string(data), "{{TODAY}}") {
				t.Errorf("%s: %s folded: %s", stamp, name, data)
			}
		}
	}
	got := cases.NormalizeState(world, approvalTree("# Audit\n", "2026-09-22T08:16:27.936837+00:00", "model:x"))
	if !strings.Contains(string(got["vault/audit.md"]), "durch model:x\n") {
		t.Errorf("model:x folded in the audit: %s", got["vault/audit.md"])
	}
	if !strings.Contains(string(got["vault/wiki/a.md"]), "- by: model:x\n") {
		t.Errorf("model:x folded in the page: %s", got["vault/wiki/a.md"])
	}
}

// An audit block the world already held is no stamp of this run: a tree that
// only carries it, or whose new block reuses a stamp the world holds, is not
// folded -- the older blocks stay byte-equal.
func TestNormalizeStateLeavesAStampOfTheWorldAlone(t *testing.T) {
	const block = "## 2026-09-22T08:16:27+00:00 — a.md (Fall `w`)\n"
	world := tree("vault/audit.md", block)
	for _, files := range []map[string][]byte{
		tree("vault/audit.md", block),
		tree("vault/audit.md", block+"\n## 2026-09-22T08:16:27+00:00 — a.md (Fall `x`)\n"),
		tree("vault/audit.md", block, "vault/b/audit.md", "## 2026-09-22T08:16:27+00:00 — b.md (Fall `y`)\n"),
	} {
		got := cases.NormalizeState(world, files)
		if !reflect.DeepEqual(got, files) {
			t.Errorf("got %q", got)
		}
	}
	// Two runs' stamps in one tree name no single run: neither is folded.
	two := tree("vault/audit.md", "## 2026-09-22T08:16:27+00:00 — a.md (Fall `x`)\n## 2026-09-23T08:16:27+00:00 — a.md (Fall `y`)\n")
	if got := cases.NormalizeState(nil, two); !reflect.DeepEqual(got, two) {
		t.Errorf("two stamps: got %q", got)
	}
}

// The SHA a commit got differs on every run; its line on stdout is folded,
// and a SHA of any other length is not one git printed.
func TestRunCaseWithFoldsTheCommitOnStdout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "case", "approve")
	writeTree(t, dir, map[string]string{
		"cmd":                  "loomux case x --approve\n",
		"exit":                 "0\n",
		"stdout":               "Fall x: freigegeben\ncommittet als " + strings.Repeat("a", 40) + "\n",
		"world/keep.txt":       "k",
		"world_after/keep.txt": "k",
	})
	c, err := cases.LoadCase(dir)
	if err != nil {
		t.Fatal(err)
	}
	printing := func(line string) cases.RunFunc {
		return func(_ []string, _ string, _ io.Reader, stdout, _ io.Writer) int {
			io.WriteString(stdout, line)
			return 0
		}
	}
	another := printing("Fall x: freigegeben\ncommittet als " + strings.Repeat("b", 40) + "\n")
	if outcome, err := cases.RunCaseWith(c, another, cases.NormalizeState); err != nil || !outcome.Passed {
		t.Fatalf("normalized: %+v, %v", outcome, err)
	}
	if outcome, err := cases.RunCase(c, another); err != nil || outcome.Passed {
		t.Fatalf("plain: %+v, %v", outcome, err)
	}
	for _, sha := range []string{strings.Repeat("b", 39), strings.Repeat("b", 41), strings.Repeat("B", 40)} {
		short := printing("Fall x: freigegeben\ncommittet als " + sha + "\n")
		if outcome, err := cases.RunCaseWith(c, short, cases.NormalizeState); err != nil || outcome.Passed {
			t.Fatalf("%s: %+v, %v", sha, outcome, err)
		}
	}
}
