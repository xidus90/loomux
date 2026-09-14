package wiki

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintReportReturnsOneForAnErrorFinding(t *testing.T) {
	wikiRoot := t.TempDir()
	page := filepath.Join(wikiRoot, "page.md")
	os.WriteFile(page, []byte("no frontmatter at all\n"), 0o644)
	var out bytes.Buffer
	code := LintReport(page, wikiRoot, &out)
	if code != 1 || !strings.Contains(out.String(), "[error:") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}

func TestLintReportNamesAFileItCannotRead(t *testing.T) {
	var out bytes.Buffer
	if code := LintReport(filepath.Join(t.TempDir(), "missing.md"), t.TempDir(), &out); code != 1 || !strings.Contains(out.String(), "lint error:") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}

func TestGateReportPassesWithoutAWiki(t *testing.T) {
	var out, errb bytes.Buffer
	if code := GateReport(t.TempDir(), &out, &errb); code != 0 || !strings.HasPrefix(out.String(), "OK: Wiki Gate passed.") {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}

// The page is named sample.md, not index.md: a scaffold file lints clean
// whatever it holds, and would pass this test for the wrong reason.
func TestLintReportReturnsZeroForAValidPage(t *testing.T) {
	wikiRoot := t.TempDir()
	writePage(t, wikiRoot, "sample.md", "---\ntitle: Sample Concept\ntype: concept\ndescription: A valid OKF test document\n---\n\n# Sample Concept\nThis is a test concept.\n")
	var out bytes.Buffer
	if code := LintReport(filepath.Join(wikiRoot, "sample.md"), wikiRoot, &out); code != 0 || out.Len() != 0 {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}

// A warning is written but does not move the code.
func TestLintReportWritesAWarningAndReturnsZero(t *testing.T) {
	wikiRoot := t.TempDir()
	writePage(t, wikiRoot, "odd.md", "---\ntitle: T\ntype: poem\n---\n")
	var out bytes.Buffer
	code := LintReport(filepath.Join(wikiRoot, "odd.md"), wikiRoot, &out)
	if code != 0 || out.String() != "  • [warning:missing-type] odd.md: unknown document type \"poem\"\n" {
		t.Fatalf("code %d, out %q", code, out.String())
	}
}

func TestGateReportListsTheViolations(t *testing.T) {
	project := t.TempDir()
	writePage(t, project, "wiki/page.md", "no frontmatter at all\n")
	var out, errb bytes.Buffer
	code := GateReport(project, &out, &errb)
	want := "\n❌ Wiki-Gate Violation(s) Detected in " + filepath.Base(project) + ":\n" +
		"  • [wiki-lint:missing-type] page.md: document is missing required 'type' in frontmatter\n" +
		"\nFound 1 violation(s).\n"
	if code != 1 || out.Len() != 0 || errb.String() != want {
		t.Fatalf("code %d, out %q, err %q", code, out.String(), errb.String())
	}
}
