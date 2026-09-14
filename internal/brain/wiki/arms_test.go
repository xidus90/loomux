package wiki

import (
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

func TestLintSingleFileSkipsAScaffoldFile(t *testing.T) {
	findings, err := LintSingleFile(filepath.Join(t.TempDir(), "index.md"), t.TempDir())
	if err != nil || findings != nil {
		t.Fatalf("findings %v, err %v", findings, err)
	}
}

func TestLintSingleFileWantsOpenConflictsWhenABoxIsPresent(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "page.md", conceptPage+"> [!CONFLICT]\n")
	findings, err := LintSingleFile(filepath.Join(root, "page.md"), root)
	if err != nil || len(findingsOf(findings, "conflict-count")) != 1 {
		t.Fatalf("findings %v, err %v", findings, err)
	}
}

func TestLintSingleFileComparesOpenConflictsWithTheBoxes(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "page.md", "---\ntitle: T\ntype: concept\nopen_conflicts: 2\n---\n\n> [!CONFLICT]\n")
	findings, err := LintSingleFile(filepath.Join(root, "page.md"), root)
	if err != nil || len(findingsOf(findings, "conflict-count")) != 1 {
		t.Fatalf("findings %v, err %v", findings, err)
	}
}

// A page that cannot be read is left out of the bundle, not reported.
func TestLintBundleSkipsAPageItCannotRead(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "locked.md", "no frontmatter at all\n")
	testlock.Lock(t, filepath.Join(root, "locked.md"))
	findings, err := LintBundle(root)
	if err != nil || len(findings) != 0 {
		t.Fatalf("findings %v, err %v", findings, err)
	}
}

// A link with a scheme and a link with no path name no page of the bundle.
func TestLintBundleIgnoresLinksWithoutABundlePath(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "index.md", "[p](page.md)\n")
	writePage(t, root, "page.md", conceptPage+"[ftp](ftp://host/x.md) [query](?x=1)\n")
	findings, err := LintBundle(root)
	if err != nil || len(findings) != 0 {
		t.Fatalf("findings %v, err %v", findings, err)
	}
}

// filepath.Rel cannot relate an absolute path to a relative root; the page
// keeps its path as given.
func TestReadPageKeepsThePathWhenItCannotBeMadeRelative(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "page.md", conceptPage)
	path := filepath.Join(root, "page.md")
	page, err := ReadPage(path, "relative-root")
	if err != nil || page.Relative != filepath.ToSlash(path) {
		t.Fatalf("page %+v, err %v", page, err)
	}
}

// A null document decodes into the struct and is no mapping, so it has no keys.
func TestReadPageCollectsNoKeysFromANullBlock(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "page.md", "---\n~\n---\n")
	page, err := ReadPage(filepath.Join(root, "page.md"), root)
	if err != nil || page.BrokenFrontmatter != nil || page.FrontmatterKeys != nil {
		t.Fatalf("page %+v, err %v", page, err)
	}
}
