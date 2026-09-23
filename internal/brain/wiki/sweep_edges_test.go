package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The edges the mutation round of 2026-09-23 found untested. Each fixture is
// one bundle; the expected lines are what `lint_bundle` answered for the same
// fixture, run by docs/.superpowers/parity/stufe-3c-orakel/edges.py.

var edgeNow = time.Date(2026, 9, 23, 12, 0, 0, 0, time.Local)

func edgeSweep(t *testing.T, ctx SweepContext, pages map[string]string) string {
	t.Helper()
	root := t.TempDir()
	links := "# c\n"
	for name, body := range pages {
		sweepWrite(t, root, name, body)
		links += "* [" + name + "](" + name + ")\n"
	}
	sweepWrite(t, root, "index.md", links)
	ctx.Root = root
	if ctx.UntouchedDays == 0 {
		ctx.UntouchedDays = 30
	}
	if ctx.Now.IsZero() {
		ctx.Now = edgeNow
	}
	found, err := SweepBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return renderSweep(found)
}

func source(fields string) string {
	return "---\ntype: Topic\nsources:\n  - " + fields + "\n---\n"
}

func TestEveryMissingFieldMakesASourceIncomplete(t *testing.T) {
	full := map[string]string{
		"id": "id: s", "resource": "resource: https://x.invalid/s", "doc_id": "doc_id: d",
		"content_hash": "content_hash: h", "revision": "revision: 1",
	}
	for missing := range full {
		var fields []string
		for name, field := range full {
			if name != missing {
				fields = append(fields, field)
			}
		}
		got := edgeSweep(t, SweepContext{}, map[string]string{"a.md": source(strings.Join(fields, "\n    "))})
		if !strings.Contains(got, "a.md:no-sources:error: source entries lack") {
			t.Errorf("a source without %s passed:\n%s", missing, got)
		}
	}
}

func TestALinkToTheParentLeavesTheArea(t *testing.T) {
	got := edgeSweep(t, SweepContext{}, map[string]string{"a.md": source("id: s\n    resource: https://x.invalid/s\n    doc_id: d\n    content_hash: h\n    revision: 1") + "[up](..)\n"})
	if !strings.Contains(got, "a.md:outside-area:warning: target leaves the area and cannot be checked: ..\n") {
		t.Fatalf("findings:\n%s", got)
	}
}

func TestOnlyASharedAreaIsAskedAboutItsDirection(t *testing.T) {
	page := source("id: s\n    resource: brain://project/a/y\n    doc_id: d\n    content_hash: h\n    revision: 1")
	if got := edgeSweep(t, SweepContext{SharedScopes: map[string]bool{"engineering/x": true}}, map[string]string{"a.md": page}); got != "" {
		t.Fatalf("a project area was judged:\n%s", got)
	}
}

func TestAStaleAfterOfTodayHasNotPassed(t *testing.T) {
	page := "---\ntype: Topic\nstale_after: " + edgeNow.Format("2006-01-02") + "\n" +
		strings.TrimPrefix(source("id: s\n    resource: https://x.invalid/s\n    doc_id: d\n    content_hash: h\n    revision: 1"), "---\ntype: Topic\n")
	if got := edgeSweep(t, SweepContext{}, map[string]string{"a.md": page}); got != "" {
		t.Fatalf("findings:\n%s", got)
	}
}

func TestTheRealizationRulesAskOnlyAProject(t *testing.T) {
	complete := "sources:\n  - id: s\n    resource: https://x.invalid/s\n    doc_id: d\n    content_hash: h\n    revision: 1\n"
	pages := map[string]string{
		"built.md":   "---\ntype: Topic\nrealization: implemented\n" + complete + "---\n",
		"planned.md": "---\ntype: Topic\nrealization: planned\n" + complete + "---\n",
		"done.md":    "---\ntype: Topic\nrealization: implemented\nimplemented_in: abc123\n" + complete + "---\n",
		"old.md":     "---\ntype: Topic\n" + complete + "---\n",
	}
	root := t.TempDir()
	links := "# c\n"
	for name, body := range pages {
		path := sweepWrite(t, root, name, body)
		old := edgeNow.AddDate(0, -3, 0)
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
		links += "* [" + name + "](" + name + ")\n"
	}
	sweepWrite(t, root, "index.md", links)
	lint := func(project bool) string {
		found, err := SweepBundle(SweepContext{Root: root, UntouchedDays: 30, Now: edgeNow, IsProject: project})
		if err != nil {
			t.Fatal(err)
		}
		var kept []string
		for _, line := range strings.Split(renderSweep(found), "\n") {
			if line != "" && !strings.Contains(line, ":untouched:") {
				kept = append(kept, line)
			}
		}
		return strings.Join(kept, "\n")
	}
	if got := lint(false); got != "" {
		t.Errorf("outside a project:\n%s", got)
	}
	want := "built.md:implemented-without-commit:error: realization: implemented without implemented_in\n" +
		"planned.md:long-planned:warning: planned and untouched since " + edgeNow.AddDate(0, -3, 0).Format("2006-01-02")
	if got := lint(true); got != want {
		t.Errorf("in a project:\n%s\nwant:\n%s", got, want)
	}
}

func TestAFalseOpenConflictsCountsAsZero(t *testing.T) {
	page := "---\ntype: Topic\nopen_conflicts: false\nsources:\n  - id: s\n    resource: https://x.invalid/s\n" +
		"    doc_id: d\n    content_hash: h\n    revision: 1\n---\n> [!conflict] one\n"
	if got := edgeSweep(t, SweepContext{}, map[string]string{"a.md": page}); got != "a.md:conflict-count:error: open_conflicts says False, 1 box(es) found\n" {
		t.Fatalf("findings:\n%s", got)
	}
}

func TestFencesOpenAndCloseInTurn(t *testing.T) {
	p := newSweepPatterns()
	for text, want := range map[string]int{
		// Two fenced examples with a real box between them: only that one.
		"```\n> [!conflict] a\n```\n> [!conflict] b\n```\n> [!conflict] c\n```\n": 1,
		// A fence at the very start of the text opens like any other.
		"```\n> [!conflict] a\n```\n> [!conflict] b\n": 1,
		// And an unclosed one there hides everything below it.
		"~~~\n> [!conflict] a\n": 0,
	} {
		if got := countConflicts(text, p); got != want {
			t.Errorf("countConflicts(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestTheWalkTakesOnlyMarkdownFiles(t *testing.T) {
	root := t.TempDir()
	sweepWrite(t, root, "a.md", "x")
	sweepWrite(t, root, "graph.json", "{}")
	if err := os.MkdirAll(filepath.Join(root, "folder.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := MarkdownBelow(root)
	if len(got) != 1 || filepath.Base(got[0]) != "a.md" {
		t.Fatalf("walk %v", got)
	}
	if got := MarkdownBelow(filepath.Join(root, "missing")); got != nil {
		t.Fatalf("a missing root walked %v", got)
	}
}

func TestATrueOpenConflictsCountsAsOneAndReadsTrue(t *testing.T) {
	page := "---\ntype: Topic\nopen_conflicts: true\nsources:\n  - id: s\n    resource: https://x.invalid/s\n" +
		"    doc_id: d\n    content_hash: h\n    revision: 1\n---\n> [!conflict] one\n\n> [!conflict] two\n"
	if got := edgeSweep(t, SweepContext{}, map[string]string{"a.md": page}); got != "a.md:conflict-count:error: open_conflicts says True, 2 box(es) found\n" {
		t.Fatalf("findings:\n%s", got)
	}
}
