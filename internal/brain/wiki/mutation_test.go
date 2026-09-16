package wiki

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/brain/check"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for. Every expectation below was read from the Python reference or
// measured against it, never derived from the Go code that has to satisfy
// it -- except where the rule has no counterpart there, and then the comment
// says so instead of inventing a citation.

// bundle writes the named files below a fresh wiki root and answers the root.
func bundle(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// lintedFor answers the findings of one rule over a bundle; findingsOf,
// beside the tests of the linter itself, does the filtering.
func lintedFor(t *testing.T, root, rule string) []check.Finding {
	t.Helper()
	all, err := LintBundle(root)
	if err != nil {
		t.Fatalf("LintBundle: %v", err)
	}
	return findingsOf(all, rule)
}

func TestAProjectWithoutAWikiIsNotGated(t *testing.T) {
	// `check_wiki_gate` (src/brain/wiki/gate.py:114-116) reads
	// `wiki_path = find_wiki_path(project_root)` and then
	// `if wiki_path is None: return []` -- a project that keeps no bundle
	// has nothing the gate can hold it to, and the gate stops before it
	// asks anything about changes. The chdir is what makes the difference
	// visible: an empty wiki path asked about its changes would otherwise
	// ask the directory the session happens to stand in, and report drift
	// against a stranger.
	t.Chdir(t.TempDir())
	root := decoyRepo(t)
	if got := CheckWikiGate(root); len(got) != 0 {
		t.Fatalf("CheckWikiGate = %v, want no violation: the project has no wiki", got)
	}
}

func TestAPageThatDeclaresTheWrongNumberOfConflictsIsReported(t *testing.T) {
	// `conflict_count` (src/brain/wiki/lint.py:429) splits the rule in
	// two: `if page.declared_conflicts is None` (:439) reports the missing
	// field, and `elif page.declared_conflicts != page.found_conflicts`
	// (:452-463) reports the number that disagrees. A page that says two
	// and shows none is the second arm, and the arm that reads the
	// declared number is the only one that can see it.
	root := bundle(t, map[string]string{
		"index.md": "---\ntype: bogus\n---\n\n# Index\n\n[a](a.md)\n",
		"a.md":     "---\ntype: concept\nopen_conflicts: 2\n---\n\n# A\n",
	})
	got := lintedFor(t, root, "conflict-count")
	if len(got) != 1 {
		t.Fatalf("conflict-count findings = %v, want exactly one", got)
	}
}

func TestAnAbsoluteLinkResolvesFromTheWikiRootNotFromThePage(t *testing.T) {
	// `_resolve` (src/brain/wiki/lint.py:269-271) reads
	// `if decoded.startswith("/")` and answers
	// `posixpath.normpath(decoded).lstrip("/")`; otherwise it joins the
	// target onto `posixpath.dirname(relative)`. A link that starts with
	// `/` names the bundle root, one that does not names the page's own
	// directory. A page in a subdirectory is what tells the two apart,
	// because only there do the two roots differ.
	root := bundle(t, map[string]string{
		"index.md":     "# Index\n\n[s](sub/s.md)\n[t](target.md)\n",
		"target.md":    "# Target\n",
		"sub/s.md":     "---\ntype: concept\n---\n\n# S\n\n[up](/target.md)\n[i](/index.md)\n[o](other.md)\n",
		"sub/other.md": "---\ntype: concept\n---\n\n# Other\n",
	})
	if got := lintedFor(t, root, "dead-link"); len(got) != 0 {
		t.Fatalf("dead-link findings = %v, want none: every link resolves", got)
	}
}

func TestAPageNobodyLinksIsAnOrphan(t *testing.T) {
	// `orphan` (src/brain/wiki/lint.py:285-300) gathers every resolved
	// link of the catalog and of every page and reports the pages that are
	// in none of them. That is the reason the first pass counts inbound
	// links at all: a knowledge page no other page names is unreachable,
	// and a bundle that never said so would let it rot unseen. loomux
	// files the finding as a warning where the reference files an error;
	// this test asks only which page the rule names.
	root := bundle(t, map[string]string{
		"index.md": "# Index\n",
		"lost.md":  "---\ntype: concept\n---\n\n# Lost\n",
	})
	got := lintedFor(t, root, "orphan")
	if len(got) != 1 || got[0].Relative != "lost.md" {
		t.Fatalf("orphan findings = %v, want exactly lost.md", got)
	}
}

func TestLinksWithASchemeAreNotWikiLinks(t *testing.T) {
	// `_resolve` (src/brain/wiki/lint.py:265-267) drops a target with
	// `if split.scheme or not split.path: return None`: a target that
	// names a scheme leads out of the bundle, and one with no path leads
	// nowhere. loomux asks the question earlier and narrower -- it names
	// the three schemes its pages carry instead of asking `urlsplit` for
	// any -- but all four targets below are refused by both.
	root := bundle(t, map[string]string{
		"p.md": "# P\n\n[h](http://a)\n[s](https://b)\n[m](mailto:c)\n[e]()\n[ok](ok.md)\n",
	})
	page, err := ReadPage(filepath.Join(root, "p.md"), root)
	if err != nil {
		t.Fatalf("ReadPage: %v", err)
	}
	if len(page.Links) != 1 || page.Links[0] != "ok.md" {
		t.Fatalf("Links = %q, want [ok.md]", page.Links)
	}
}

func TestAnUnsaidLayoutIsNoLayout(t *testing.T) {
	// No reference for the key itself: `find_wiki_path`
	// (src/brain/wiki/gate.py:21-55) knows no manifest key at all and
	// starts at `docs/wiki`. The `[layout] wiki` key is loomux's own
	// (parity list, stage 1a). What the reference does fix is what has to
	// answer when nothing names a layout -- `docs/wiki` first, `wiki/`
	// second (gate.py:24-35). Reading the unsaid value as a path would
	// join nothing onto the project root and hand back the project itself
	// as its own wiki, which is neither.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[area]\nscope = \"project/x\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "wiki"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := Root(root), filepath.Join(root, "docs", "wiki"); got != want {
		t.Fatalf("Root = %q, want %q", got, want)
	}
}

func TestABundleThatIsNotThereIsNoBundle(t *testing.T) {
	// Measured: `list(Path(<missing>).rglob("*.md"))` answers `[]`, so
	// `lint_bundle` (src/brain/wiki/lint.py:582-587) reads no pages from a
	// path that is not there and raises nothing. The decision the mutant
	// changes has no counterpart in the reference: `filepath.WalkDir`
	// hands its callback a nil entry together with the error for a root it
	// cannot read, and the callback has to see the error before it asks
	// the entry anything, or the linter goes down where Python comes back
	// empty.
	root := filepath.Join(t.TempDir(), "absent")
	findings, err := LintBundle(root)
	if err != nil {
		t.Fatalf("LintBundle: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
}
