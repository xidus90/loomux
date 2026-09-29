package maintenance_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// writePage lays one page down under root, creating whatever directories its
// relative path names.
func writePage(t *testing.T, root, relative, text string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// page is a wiki page citing the given doc ids.
func page(docIDs ...string) string {
	text := "---\ntype: knowledge\nsources:\n"
	for _, docID := range docIDs {
		text += "  - doc_id: " + docID + "\n    resource: src/x.go\n"
	}
	return text + "---\n\nbody\n"
}

func dependents(t *testing.T, root string) map[string][]string {
	t.Helper()
	index, err := maintenance.Dependents(root, nil)
	if err != nil {
		t.Fatalf("Dependents: %v", err)
	}
	return index
}

// The backward edge itself: a source is answered with the page that cites it.
func TestDependentsNamesThePageThatCitesASource(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "a.md", page("doc-1"))
	writePage(t, root, "b.md", page("doc-2", "doc-1"))
	want := map[string][]string{"doc-1": {"a.md", "b.md"}, "doc-2": {"b.md"}}
	if got := dependents(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("index is %v, want %v", got, want)
	}
}

// Sorted, and sorted over the relative paths rather than over the order the
// tree was walked in. The two differ here: a walk reaches the directory `a`
// before the file `a-x.md`, while `-` sorts before `/`.
func TestDependentsSortsThePagesOfASource(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "a-x.md", page("doc-1"))
	writePage(t, root, "a/b.md", page("doc-1"))
	want := map[string][]string{"doc-1": {"a-x.md", "a/b.md"}}
	if got := dependents(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("index is %v, want %v", got, want)
	}
}

// A page citing one source twice is listed for it once: its presence is a
// fact about the page, not a count of citations.
func TestDependentsListsAPageOncePerSource(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "a.md", page("doc-1", "doc-1"))
	want := map[string][]string{"doc-1": {"a.md"}}
	if got := dependents(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("index is %v, want %v", got, want)
	}
}

// Scaffold files carry structure rather than knowledge, so they raise no edge
// even when they cite one.
func TestDependentsSkipsAScaffoldFile(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "index.md", page("doc-1"))
	if got := dependents(t, root); len(got) != 0 {
		t.Fatalf("index is %v, want nothing", got)
	}
}

// A page whose frontmatter is unreadable contributes no edge and aborts
// nothing: one broken page may never cost the whole rebuild. That silence is
// not a clean bill of health -- whoever reconciles has to find
// `broken_frontmatter` itself.
func TestDependentsCarriesOnPastABrokenPage(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "broken.md", "---\nsources: [\n---\n\nbody\n")
	writePage(t, root, "whole.md", page("doc-1"))
	want := map[string][]string{"doc-1": {"whole.md"}}
	if got := dependents(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("index is %v, want %v", got, want)
	}
}

// Only `*.md` is read. A citation in a file of another name is no page.
func TestDependentsReadsOnlyMarkdown(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "notes.txt", page("doc-1"))
	writePage(t, root, "sub/deep.md", page("doc-2"))
	want := map[string][]string{"doc-2": {"sub/deep.md"}}
	if got := dependents(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("index is %v, want %v", got, want)
	}
}

// A directory named like a page is what `rglob("*.md")` hands the reader as
// well, and reading it fails. Answered rather than skipped: a wiki that has
// grown a directory called `a.md` is broken in a way a silent walk would hide.
func TestDependentsAnswersAPageItCannotRead(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "a.md"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if _, err := maintenance.Dependents(root, nil); err == nil {
		t.Fatal("Dependents read a directory as a page")
	}
}

// The wiki directory itself is never a page of the wiki, whatever it is
// called. `rglob` walks what is inside; a walk starts at the root and would
// otherwise read the directory as its own first page and fail.
func TestDependentsIsNoPageOfItself(t *testing.T) {
	root := filepath.Join(t.TempDir(), "wiki.md")
	writePage(t, root, "a.md", page("doc-1"))
	want := map[string][]string{"doc-1": {"a.md"}}
	if got := dependents(t, root); !reflect.DeepEqual(got, want) {
		t.Fatalf("index is %v, want %v", got, want)
	}
}

// A wiki path that is not there is an error and not an empty index: an empty
// one reads as "no page cites anything", which would leave every source
// looking unused.
func TestDependentsAnswersAMissingWiki(t *testing.T) {
	if _, err := maintenance.Dependents(filepath.Join(t.TempDir(), "absent"), nil); err == nil {
		t.Fatal("Dependents walked a wiki that is not there")
	}
}

// A wiki nested inside this one belongs to its own area and is not walked;
// the nested wiki is recognised by identity, so a junction naming it counts,
// and a registered wiki that is not there carves nothing.
func TestDependentsLeavesANestedWikiToItsArea(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "a.md", page("doc-1"))
	writePage(t, root, "inner/b.md", page("doc-1"))
	writePage(t, root, "inner-neu/c.md", page("doc-1"))
	nested := filepath.Join(root, "inner")
	if runtime.GOOS == "windows" {
		nested = filepath.Join(t.TempDir(), "link")
		junction(t, nested, filepath.Join(root, "inner"))
	}
	index, err := maintenance.Dependents(root, []string{filepath.Join(t.TempDir(), "absent"), nested})
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string][]string{"doc-1": {"a.md", "inner-neu/c.md"}}; !reflect.DeepEqual(index, want) {
		t.Fatalf("index is %v, want %v", index, want)
	}
}
