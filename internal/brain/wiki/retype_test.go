package wiki

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

// retypePage carries a link and a source beside its type, so a test that only
// asked "did the type change" could not pass while the rest was rewritten.
const retypePage = "---\ntype: Design Decision\ntitle: Suchleiter\nsources:\n  - id: spec\n" +
	"    resource: /docs/spec.md\n    doc_id: 01J8F2K9XQ7M\n    content_hash: \"sha256:9f2a\"\n" +
	"    revision: 4\n---\n\nText mit einem [Link](../andere/seite.md) und einer Quelle.\n"

func writeBytes(t *testing.T, path string, body []byte) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readText(t *testing.T, path string) string {
	t.Helper()
	return string(readBytes(t, path))
}

func mustRetype(t *testing.T, root, source, target string) []string {
	t.Helper()
	changed, err := Retype(root, source, target)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func TestRetypeRenamesOnlyTheTypeLine(t *testing.T) {
	root := t.TempDir()
	page := writeBytes(t, filepath.Join(root, "topics", "suchleiter.md"), []byte(retypePage))
	if got := mustRetype(t, root, "Design Decision", "Decision"); !reflect.DeepEqual(got, []string{page}) {
		t.Fatalf("changed %v, want %v", got, page)
	}
	want := strings.Replace(retypePage, "type: Design Decision\n", "type: Decision\n", 1)
	if got := readText(t, page); got != want {
		t.Fatalf("page:\n%s\nwant:\n%s", got, want)
	}
	if got := mustRetype(t, root, "Design Decision", "Decision"); len(got) != 0 {
		t.Fatalf("a second run changed %v", got)
	}
}

func TestRetypeFoldsACRLFPageToLF(t *testing.T) {
	root := t.TempDir()
	page := writeBytes(t, filepath.Join(root, "p.md"), []byte(strings.ReplaceAll(retypePage, "\n", "\r\n")))
	mustRetype(t, root, "Design Decision", "Decision")
	got := readText(t, page)
	if strings.Contains(got, "\r") || !strings.Contains(got, "type: Decision\n") {
		t.Fatalf("page %q", got)
	}
}

func TestRetypeLeavesWhatItMustNotTouch(t *testing.T) {
	root := t.TempDir()
	untouched := map[string]string{
		"other.md":   strings.Replace(retypePage, "Design Decision", "Topic", 1),
		"prefix.md":  strings.Replace(retypePage, "Design Decision", "Design Decision Draft", 1),
		"broken.md":  "---\ntype: [unclosed\n---\n\nText\n",
		"quoted.md":  strings.Replace(retypePage, "type: Design Decision", "type: \"Design Decision\"", 1),
		"utf8.md":    "---\ntype: Design Decision\n---\nBody \xff\xfe here\n",
		"cr-only.md": "---\rtype: Design Decision\r---\rBody\r",
		"no-type.md": "# no frontmatter\n",
		"index.md":   retypePage,
		"_schema.md": retypePage,
		"log.md":     retypePage,
	}
	for name, body := range untouched {
		writeBytes(t, filepath.Join(root, name), []byte(body))
	}
	normal := writeBytes(t, filepath.Join(root, "z-normal.md"), []byte(retypePage))
	if got := mustRetype(t, root, "Design Decision", "Decision"); !reflect.DeepEqual(got, []string{normal}) {
		t.Fatalf("changed %v, want only %s", got, normal)
	}
	for name, body := range untouched {
		if got := readText(t, filepath.Join(root, name)); got != body {
			t.Errorf("%s was rewritten: %q", name, got)
		}
	}
}

func TestRetypeToItselfWritesNothing(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "p.md"), []byte(retypePage))
	if got := mustRetype(t, root, "Design Decision", "Design Decision"); len(got) != 0 {
		t.Fatalf("changed %v", got)
	}
}

func TestRetypeTakesATargetLiterally(t *testing.T) {
	// A regular expression replacement would expand `$1`.
	root := t.TempDir()
	page := writeBytes(t, filepath.Join(root, "p.md"), []byte(retypePage))
	mustRetype(t, root, "Design Decision", "Cost $1")
	if got := readText(t, page); !strings.Contains(got, "type: Cost $1\n") {
		t.Fatalf("page %q", got)
	}
}

func TestRetypeAnswersThePathsInTheOperatingSystemsSpelling(t *testing.T) {
	root := t.TempDir()
	page := writeBytes(t, filepath.Join(root, "topics", "p.md"), []byte(retypePage))
	got := mustRetype(t, filepath.ToSlash(root), "Design Decision", "Decision")
	if !reflect.DeepEqual(got, []string{page}) {
		t.Fatalf("changed %v, want %v", got, page)
	}
}

func TestRetypeStopsAtAPageItCannotRead(t *testing.T) {
	root := t.TempDir()
	first := writeBytes(t, filepath.Join(root, "a.md"), []byte(retypePage))
	testlock.Lock(t, writeBytes(t, filepath.Join(root, "b.md"), []byte(retypePage)))
	got, err := Retype(root, "Design Decision", "Decision")
	if err == nil || !reflect.DeepEqual(got, []string{first}) {
		t.Fatalf("changed %v, err %v", got, err)
	}
}

func TestRetypeStopsAtAPageItCannotWrite(t *testing.T) {
	root := t.TempDir()
	page := writeBytes(t, filepath.Join(root, "a.md"), []byte(retypePage))
	// Read-only on Windows means readable and not replaceable: the page
	// parses, and the swap over it is refused.
	if err := os.Chmod(page, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(page, 0o644) })
	if got, err := Retype(root, "Design Decision", "Decision"); err == nil || len(got) != 0 {
		t.Fatalf("changed %v, err %v", got, err)
	}
}

func TestRetypeLeavesAHeaderWithoutAClosingLineBreak(t *testing.T) {
	// The reader takes a closing fence at the end of the file without a line
	// break; the reference's header pattern needs one, so it finds no line
	// to change and leaves the page as it is.
	root := t.TempDir()
	body := "---\ntype: Design Decision\n---"
	page := writeBytes(t, filepath.Join(root, "p.md"), []byte(body))
	if got := mustRetype(t, root, "Design Decision", "Decision"); len(got) != 0 || readText(t, page) != body {
		t.Fatalf("changed %v, page %q", got, readText(t, page))
	}
}
