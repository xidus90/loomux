package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/testlock"
)

const sweepSourceBlock = "sources:\n  - id: s1\n    resource: raw/s.md\n    doc_id: d\n    content_hash: h\n    revision: 1\n"

// sweepExternal is a complete source that points outside the bundle through a
// scheme, so no link rule has anything to say about it.
const sweepExternal = "sources:\n  - id: s1\n    resource: https://x.invalid/s\n    doc_id: d\n    content_hash: h\n    revision: 1\n"

func sweepWrite(t *testing.T, root, rel, text string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	writeBytes(t, path, []byte(text))
	return path
}

// sweepWorld is the world of a script in the archive release
// archive/parity-recordings, built here file for file: every edge case of the twelve rules in one
// bundle, a second wiki to be named, and a hub page.
func sweepWorld(t *testing.T) (string, SweepContext) {
	t.Helper()
	base := t.TempDir()
	wiki := filepath.Join(base, "wiki")
	other := filepath.Join(base, "other", "wiki")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	sweepWrite(t, base, "Hub/q.md", "hub\n")
	sweepWrite(t, wiki, "raw/s.md", "raw source, no frontmatter\n")
	sweepWrite(t, wiki, "index.md", "# Catalog\n\n* [a](a.md)\n* [b](topics/b.md)\n* [c](sub%20dir/c.md)\n"+
		"* [out](../other/wiki/x.md)\n* [ext](https://example.invalid/)\n* [bad](%zz.md)\n* [[Upper]]\n")
	sweepWrite(t, wiki, "_schema.md", "---\ntype: Topic\n---\n[dead](nowhere.md)\n")
	sweepWrite(t, wiki, "a.md", "---\ntitle: a\ntype: Topic\n"+sweepSourceBlock+"---\n"+
		"[b](topics/b.md) [miss](missing.md) [abs](/abs.md) [up](../up.md) [q](c.md?x=1#f)\n"+
		"[pct](%zz.md) ![img](i.md) [[Wiki Link]] [[ spaced | alias]] [dup](topics/b.md)\n"+
		"[root](/a.md) [climb](/../escape.md) [scheme](c:/x.md) [frag](#only)\n")
	sweepWrite(t, wiki, "topics/b.md", "---\ntype: Design Decision\nsources:\n  - id: s2\n    resource: ../raw/s.md\n"+
		"  - resource: r\n    doc_id: d\n---\n[a](../a.md)\n")
	sweepWrite(t, wiki, "topics/index.md", "# nested\n[o](orphaned.md)\n")
	sweepWrite(t, wiki, "topics/orphaned.md", "---\ntype: topic\n---\norphan\n")
	sweepWrite(t, wiki, "sub dir/c.md", "---\ntype: [unclosed\n---\n> [!conflict] one\n")
	sweepWrite(t, wiki, "d.md", strings.ReplaceAll("---\ntype: ''\nopen_conflicts: 2\nstale_after: 2020-01-01\nrealization: implemented\n"+
		sweepSourceBlock+"---\n```\n> [!CONFLICT] fenced\n```\n  > [!conflict] outside\n[a](a.md)\n", "\n", "\r\n"))
	e := sweepWrite(t, wiki, "e.md", "---\ntype: Decision\nrealization: planned\nimplemented_in: ''\nsources:\n"+
		"  - id: s3\n    resource: brain://project/x/topics/y\n    doc_id: d\n    content_hash: h\n    revision: 2\n"+
		"  - id: s4\n    resource: brain://engineering/x/z\n    doc_id: d\n    content_hash: h\n    revision: 2\n"+
		"  - id: s5\n    resource: brain://\n    doc_id: d\n    content_hash: h\n    revision: 2\n---\n[a](a.md)\n")
	old := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)
	if err := os.Chtimes(e, old, old); err != nil {
		t.Fatal(err)
	}
	sweepWrite(t, wiki, "f.md", "---\n- a\n- b\n---\n[a](a.md)\n")
	sweepWrite(t, wiki, "Upper.md", "---\ntype: Balancing Rule\nopen_conflicts: true\n"+sweepSourceBlock+"---\n> [!conflict] x\n")
	sweepWrite(t, wiki, "g.md", "---\ntype: Metric\nstale_after: '2020-01-01'\nsources: nope\n---\n[a](a.md)\n")
	sweepWrite(t, wiki, "h.md", "---\n---\n")
	sweepWrite(t, wiki, "i.md", "---\ntype: Topic\n---\nbad \xff byte [a](a.md)\n")
	otherAbs := AbsoluteClean(other)
	return base, SweepContext{
		Root: wiki, UntouchedDays: 30,
		Now: time.Date(2026, 9, 23, 12, 0, 0, 0, time.Local),
		ExpectedTargets: []ExpectedTarget{
			{"project/p", []string{otherAbs}},
			{"project/q", []string{filepath.Join(otherAbs, "nope"), AbsoluteClean(filepath.Join(base, "Hub", "q.md"))}},
			{"project/r", []string{AbsoluteClean(filepath.Join(base, "rwiki"))}},
		},
		IsShared: true, SharedScopes: map[string]bool{"engineering/x": true},
		DeclaredTypes: map[string]bool{"Balancing Rule": true}, IsProject: true,
	}
}

func renderSweep(findings []check.Finding) string {
	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "%s:%s:%s: %s\n", f.Relative, f.Rule, f.Severity, f.Message)
	}
	return b.String()
}

func TestTheSweepAnswersWhatLintPyAnswers(t *testing.T) {
	// testdata/sweep/python.golden is what the reference's `lint_bundle`
	// printed for this world on 2026-09-23, the base path
	// replaced by a placeholder. One difference is recorded and carried
	// here: the text of a YAML error is the parser's, PyYAML's there and
	// yaml.v3's here.
	base, ctx := sweepWorld(t)
	golden, err := os.ReadFile(filepath.Join("testdata", "sweep", "python.golden"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.ReplaceAll(string(golden), "{{BASE}}", AbsoluteClean(base))
	pyYAML := "while parsing a flow sequence\n  in \"<unicode string>\", line 1, column 7:\n    type: [unclosed\n" +
		"          ^\nexpected ',' or ']', but got '<stream end>'\n  in \"<unicode string>\", line 1, column 16:\n" +
		"    type: [unclosed\n                   ^\n"
	want = strings.Replace(want, pyYAML, "yaml: line 1: did not find expected ',' or ']'\n", 1)
	found, err := SweepBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := renderSweep(found); got != want {
		t.Fatalf("sweep:\n%s\nwant:\n%s", got, want)
	}
}

func TestTheSweepStopsAtAPageItCannotRead(t *testing.T) {
	root := t.TempDir()
	testlock.Lock(t, sweepWrite(t, root, "a.md", "---\ntype: Topic\n---\n"))
	if _, err := SweepBundle(SweepContext{Root: root}); err == nil {
		t.Fatal("an unreadable page was passed over")
	}
}

func TestAnUnreadableCatalogLinksNothing(t *testing.T) {
	// `index.md` is read for its links only; a catalog that cannot be read
	// leaves every page it would have named an orphan, rather than ending a
	// run the reference would have ended with a traceback.
	root := t.TempDir()
	sweepWrite(t, root, "a.md", "---\ntype: Topic\n"+sweepExternal+"---\n")
	sweepWrite(t, root, "raw/s.md", "---\ntype: Source\n"+sweepExternal+"---\n[a](../a.md)\n")
	testlock.Lock(t, sweepWrite(t, root, "index.md", "[s](raw/s.md)\n"))
	found, err := SweepBundle(SweepContext{Root: root, UntouchedDays: 30, Now: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if got := renderSweep(found); got != "raw/s.md:orphan:error: no page and no catalog entry links here\n" {
		t.Fatalf("findings:\n%s", got)
	}
}

func TestAYAMLKeyThatIsNoStringIsReadAsItsText(t *testing.T) {
	root := t.TempDir()
	sweepWrite(t, root, "a.md", "---\n1: one\ntype: Topic\n"+sweepExternal+"---\n")
	sweepWrite(t, root, "index.md", "[a](a.md)\n")
	found, err := SweepBundle(SweepContext{Root: root, UntouchedDays: 30, Now: time.Now()})
	if err != nil || len(found) != 0 {
		t.Fatalf("findings %v, err %v", found, err)
	}
}

func TestACatalogThatIsADirectoryLinksNothing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "index.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := indexLinks(root, newSweepPatterns()); got != nil {
		t.Fatalf("links %v", got)
	}
}

func TestAbsoluteCleanFallsBackToTheCleanPath(t *testing.T) {
	// Abs refuses a name Windows could never open; the lexical answer is
	// then all there is.
	if got := AbsoluteClean("a/../b\x00"); got != filepath.Clean("a/../b\x00") {
		t.Fatalf("AbsoluteClean = %q", got)
	}
}

func TestAWikiTargetTakesAnyLinkBelowIt(t *testing.T) {
	for _, c := range []struct {
		link, target string
		want         bool
	}{
		{`C:\v\Wiki\p.md`, `C:\v\wiki`, true},
		{`C:\v\wiki`, `C:\v\wiki`, true},
		{`C:\v\wiki-old\p.md`, `C:\v\wiki`, false},
		{`C:\v`, `C:\v\wiki`, false},
		{`C:\v\Hub\Q.md`, `C:\v\hub\q.md`, true},
		{`C:\v\hub\q.md\deeper.md`, `C:\v\hub\q.md`, false},
	} {
		if got := LinkNames(c.link, c.target); got != c.want {
			t.Errorf("LinkNames(%q, %q) = %v", c.link, c.target, got)
		}
	}
}

func TestAnUnclosedFenceHidesEveryBoxBelowIt(t *testing.T) {
	// An unclosed fence reaches to the end of the text: rather undercount a
	// box than accuse a page of bookkeeping it never got wrong.
	p := newSweepPatterns()
	if got := countConflicts("> [!conflict] a\n```\n> [!conflict] b\n", p); got != 1 {
		t.Fatalf("count %d, want 1", got)
	}
}
