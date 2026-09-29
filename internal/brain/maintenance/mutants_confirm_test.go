package maintenance_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xidus90/loomux/internal/brain/maintenance"
)

// Only a directory is carved. The registry takes any string as an area's
// wiki, so another area may name a page of this wiki as its own; that page is
// still this wiki's, and so are the pages beside it that the walk reaches
// after it.
func TestDependentsCarvesNoPageThatAnotherAreaNamesAsItsWiki(t *testing.T) {
	root := t.TempDir()
	writePage(t, root, "a.md", page("doc-1"))
	writePage(t, root, "b.md", page("doc-1"))
	index, err := maintenance.Dependents(root, []string{filepath.Join(root, "a.md")})
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string][]string{"doc-1": {"a.md", "b.md"}}; !reflect.DeepEqual(index, want) {
		t.Fatalf("index is %v, want %v", index, want)
	}
}
