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
// it.

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
	// A project that keeps no wiki bundle has nothing the gate can hold it
	// to. The chdir is what makes the difference visible: an empty wiki
	// path asked about its changes would otherwise ask the directory the
	// session happens to stand in, and report drift against a stranger.
	t.Chdir(t.TempDir())
	root := decoyRepo(t)
	if got := CheckWikiGate(root); len(got) != 0 {
		t.Fatalf("CheckWikiGate = %v, want no violation: the project has no wiki", got)
	}
}

func TestAPageThatDeclaresTheWrongNumberOfConflictsIsReported(t *testing.T) {
	// OKF §11: `open_conflicts` counts the conflict boxes of the page. A
	// page that says two and shows none is as wrong as one that shows two
	// and says nothing, and the rule that reads the declared number is
	// the only one that can see it.
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
	// A link that starts with `/` names the bundle root; one that does not
	// names the page's own directory. A page in a subdirectory is what
	// tells the two apart, because only there do the two roots differ.
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
	// The orphan rule is the reason the first pass counts inbound links at
	// all: a knowledge page no other page names is unreachable, and a
	// bundle that never said so would let it rot unseen.
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
	// A page's `Links` are the targets inside the bundle. The three
	// schemes the reader names lead out of it, and an empty target leads
	// nowhere: none of the four is a page this bundle can be held to.
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
	// A manifest without a `[layout] wiki` key says nothing about where
	// the bundle is, and the fallbacks answer. Reading the unsaid value as
	// a path would join nothing onto the project root and hand back the
	// project itself as its own wiki.
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
	// `filepath.WalkDir` hands its callback a nil entry together with the
	// error for a root it cannot read. The callback has to see the error
	// before it asks the entry anything, or a wiki path that names no
	// directory takes the linter down instead of coming back empty.
	root := filepath.Join(t.TempDir(), "absent")
	findings, err := LintBundle(root)
	if err != nil {
		t.Fatalf("LintBundle: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
}
