package wiki

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/check"
)

func TestLintSingleFile(t *testing.T) {
	tempDir := t.TempDir()

	validDoc := `---
title: Sample Concept
type: concept
description: A valid OKF test document
---

# Sample Concept
This is a test concept.
`
	validPath := filepath.Join(tempDir, "sample.md")
	if err := os.WriteFile(validPath, []byte(validDoc), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err := LintSingleFile(validPath, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d: %v", len(findings), findings)
	}

	invalidDoc := `---
title: Missing Type Doc
---

# Missing Type
No type declared here.
`
	invalidPath := filepath.Join(tempDir, "invalid.md")
	if err := os.WriteFile(invalidPath, []byte(invalidDoc), 0644); err != nil {
		t.Fatal(err)
	}

	findings, err = LintSingleFile(invalidPath, tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Errorf("expected missing-type finding, got 0")
	}
}

func TestScaffoldFilesExempt(t *testing.T) {
	if !IsScaffoldFile("index.md") {
		t.Errorf("expected index.md to be scaffold")
	}
	if !IsScaffoldFile("_schema.md") {
		t.Errorf("expected _schema.md to be scaffold")
	}
	if IsScaffoldFile("concept.md") {
		t.Errorf("concept.md should not be scaffold")
	}
}

// writePage puts one file into the bundle and fails the test if it cannot.
func writePage(t *testing.T, root, relative, body string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// findingsOf keeps the findings of one rule, because every fixture below
// also draws findings of the other rules -- an unlinked page is an orphan
// whatever the case under test is about.
func findingsOf(findings []check.Finding, rule string) []check.Finding {
	var out []check.Finding
	for _, f := range findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

const conceptPage = "---\ntitle: T\ntype: concept\n---\n\n"

// TestLintBundleExemptsScaffoldFromLinkRules holds the bundle rules to the
// same scaffold line the single-file rules already keep. `_schema.md` is the
// example document of a bundle, so its `/pfad/a.md` is anschauliches
// Beispielmaterial rather than a target -- `pkg/check/house/bundle.go:58-77`
// takes the whole of `judgedInBundle` from that and reports neither rule on
// a scaffold page.
func TestLintBundleExemptsScaffoldFromLinkRules(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "_schema.md", "# Schema\n\n[a](/pfad/a.md) [out](../out.md)\n")

	findings, err := LintBundle(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := findingsOf(findings, "dead-link"); len(got) != 0 {
		t.Errorf("scaffold page judged by dead-link: %v", got)
	}
	if got := findingsOf(findings, "outside-area"); len(got) != 0 {
		t.Errorf("scaffold page judged by outside-area: %v", got)
	}
}

// TestLintBundleStillJudgesOrdinaryPages is the counter-probe to the one
// above: the exemption has to cost the two rules nothing on a page that is
// not scaffold, or the fix would have switched them off altogether.
func TestLintBundleStillJudgesOrdinaryPages(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "topics/page.md", conceptPage+"[a](/pfad/a.md) [out](../../out.md)\n")

	findings, err := LintBundle(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := findingsOf(findings, "dead-link"); len(got) != 1 {
		t.Errorf("expected 1 dead-link on an ordinary page, got %v", got)
	}
	if got := findingsOf(findings, "outside-area"); len(got) != 1 {
		t.Errorf("expected 1 outside-area on an ordinary page, got %v", got)
	}
}

// TestLintBundleReadsScaffoldAsLinkSource guards the half of the exemption
// that must not move. A catalog is a valid source of links for the orphan
// rule (`pkg/wiki/parse.go:195-196`), so exempting scaffold pages as
// *subjects* may not stop them being read as *sources*: a page listed only
// by `index.md` would otherwise turn into an orphan.
func TestLintBundleReadsScaffoldAsLinkSource(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "index.md", "# Catalog\n\n[page](topics/page.md)\n")
	writePage(t, root, "topics/page.md", conceptPage+"# Page\n")

	findings, err := LintBundle(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := findingsOf(findings, "orphan"); len(got) != 0 {
		t.Errorf("a page listed by index.md is no orphan: %v", got)
	}
}
