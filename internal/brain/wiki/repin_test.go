package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/identity"
)

const repinPage = "---\n" +
	"type: Topic\n" +
	"title: T\n" +
	"description: Eine lange Beschreibung, die eine YAML-Bibliothek bei achtzig Zeichen umbrechen würde, hier aber stehen bleibt.\n" +
	"sources:\n" +
	"  - id: a\n" +
	"    resource: brain://project/x/docs/de/a.md\n" +
	"    doc_id: 01A\n" +
	"    content_hash: \"sha256:aaa\"\n" +
	"    revision: 3\n" +
	"  - id: b\n" +
	"    resource: brain://project/x/docs/de/b.md\n" +
	"    doc_id: 01B\n" +
	"    content_hash: 'sha256:bbb'\n" +
	"    revision: 5\n" +
	"  - id: c\n" +
	"    resource: brain://project/x/docs/de/c.md\n" +
	"    doc_id: \"01C\"\n" +
	"    content_hash: sha256:ccc\n" +
	"    revision: 7\n" +
	"---\n\n# T\nrevision: 3 in the body stays\n"

// Only the two values of the named entries change; quotes, order, the long
// description and the body stay byte for byte.
func TestRepinTextChangesOnlyTheTwoValues(t *testing.T) {
	got, err := repinText(repinPage, map[string]pinValue{
		"01A": {"sha256:a2", 4},
		"01C": {"sha256:c2", 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.NewReplacer(
		"content_hash: \"sha256:aaa\"\n    revision: 3", "content_hash: \"sha256:a2\"\n    revision: 4",
		"content_hash: sha256:ccc\n    revision: 7", "content_hash: sha256:c2\n    revision: 8",
	).Replace(repinPage)
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// A list written at column 0, as YAML allows under a mapping key, is read
// with its own indent.
func TestRepinTextReadsAListAtColumnZero(t *testing.T) {
	text := "---\nsources:\n- id: a\n  doc_id: 01A\n  content_hash: \"sha256:aaa\"\n  revision: 3\n---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:a2", 4}})
	if err != nil {
		t.Fatal(err)
	}
	if want := "---\nsources:\n- id: a\n  doc_id: 01A\n  content_hash: \"sha256:a2\"\n  revision: 4\n---\n"; got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// Single quotes stay single quotes.
func TestRepinTextKeepsSingleQuotes(t *testing.T) {
	got, err := repinText(repinPage, map[string]pinValue{"01B": {"sha256:b2", 6}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "    content_hash: 'sha256:b2'\n    revision: 6\n") {
		t.Fatalf("got\n%s", got)
	}
}

// A page saved with CRLF keeps CRLF on every line, the changed ones too.
func TestRepinTextKeepsCRLF(t *testing.T) {
	crlf := strings.ReplaceAll(repinPage, "\n", "\r\n")
	got, err := repinText(crlf, map[string]pinValue{"01A": {"sha256:a2", 4}})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.ReplaceAll(strings.Replace(repinPage, "content_hash: \"sha256:aaa\"\n    revision: 3", "content_hash: \"sha256:a2\"\n    revision: 4", 1), "\n", "\r\n")
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// Values equal to the update leave the page as it is.
func TestRepinTextWithoutChangeIsIdentity(t *testing.T) {
	got, err := repinText(repinPage, map[string]pinValue{"01A": {"sha256:aaa", 3}})
	if err != nil || got != repinPage {
		t.Fatalf("err %v, changed: %v", err, got != repinPage)
	}
}

// Every form it cannot edit safely is refused rather than rewritten.
func TestRepinTextRefusesWhatItCannotEditSafely(t *testing.T) {
	cases := map[string]string{
		"no frontmatter":               "# T\n",
		"unclosed":                     "---\ntitle: T\n",
		"flow list":                    "---\nsources: [{doc_id: 01A, content_hash: x, revision: 1}]\n---\n",
		"no sources":                   "---\ntitle: T\n---\n",
		"missing revision":             "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n---\n",
		"missing hash":                 "---\nsources:\n  - doc_id: 01A\n    revision: 1\n---\n",
		"comment":                      "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1 # pinned by hand\n---\n",
		"block scalar":                 "---\nsources:\n  - doc_id: 01A\n    content_hash: >\n      x\n    revision: 1\n---\n",
		"unterminated quote":           "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\n      y\"\n    revision: 1\n---\n",
		"entry not found":              "---\nsources:\n  - doc_id: 01Z\n    content_hash: \"x\"\n    revision: 1\n---\n",
		"refused before a later entry": "---\nsources:\n  - doc_id: 01A\n    revision: 1\n  - doc_id: 01Z\n    content_hash: \"x\"\n    revision: 1\n---\n",

		// The cases below each differ from one above only in what the
		// refusal hangs on, so that dropping one check is not hidden by another.
		"text before the opening fence":       "# T\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1\n---\n",
		"unclosed with entries":               "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1\n",
		"flow list then entries":              "---\nsources: [x]\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1\n---\n",
		"fields without a list item":          "---\nsources:\n doc_id: 01A\n content_hash: \"x\"\n revision: 1\n---\n",
		"field missing after a full entry":    "---\nsources:\n  - doc_id: 01Z\n    content_hash: \"y\"\n    revision: 9\n  - doc_id: 01A\n    content_hash: \"x\"\n---\n",
		"empty value":                         "---\nsources:\n  - doc_id: 01A\n    content_hash:\n    revision: 1\n---\n",
		"literal block":                       "---\nsources:\n  - doc_id: 01A\n    content_hash: |\n      x\n    revision: 1\n---\n",
		"lone quote":                          "---\nsources:\n  - doc_id: 01A\n    content_hash: \"\n    revision: 1\n---\n",
		"lone quote as doc_id":                "---\nsources:\n  - doc_id: \"\n    content_hash: \"x\"\n    revision: 1\n---\n",
		"doc_id with only an opening quote":   "---\nsources:\n  - doc_id: \"01AB\n    content_hash: \"x\"\n    revision: 1\n---\n",
		"doc_id with mismatched quotes":       "---\nsources:\n  - doc_id: '01A\"\n    content_hash: \"x\"\n    revision: 1\n---\n",
		"plain multi-line hash":               "---\nsources:\n  - doc_id: 01A\n    content_hash: x\n      y\n    revision: 1\n---\n",
		"plain multi-line revision":           "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1\n      2\n---\n",
		"plain multi-line after a blank line": "---\nsources:\n  - doc_id: 01A\n    content_hash: x\n\n      y\n    revision: 1\n---\n",
		"comment after a single-quoted value": "---\nsources:\n  - doc_id: 01A\n    content_hash: 'x' # by hand\n    revision: 1\n---\n",
	}
	for name, text := range cases {
		if _, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}}); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
}

// A doc_id that stands twice cites one file twice: both entries get the new
// values, and every other byte stays.
func TestRepinTextEditsEveryEntryOfADocID(t *testing.T) {
	text := "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1\n  - doc_id: 01B\n    content_hash: \"b\"\n    revision: 9\n  - doc_id: 01A\n    content_hash: 'y'\n    revision: 2\n---\nbody\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 5}})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nsources:\n  - doc_id: 01A\n    content_hash: \"sha256:n\"\n    revision: 5\n  - doc_id: 01B\n    content_hash: \"b\"\n    revision: 9\n  - doc_id: 01A\n    content_hash: 'sha256:n'\n    revision: 5\n---\nbody\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// A nested key of the same name deeper in an entry is not the entry's field.
func TestRepinTextIgnoresDeeperKeys(t *testing.T) {
	text := "---\nsources:\n  - doc_id: 01A\n    meta:\n      revision: 99\n    content_hash: \"x\"\n    revision: 1\n---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "      revision: 99\n") || !strings.Contains(got, "    revision: 2\n") {
		t.Fatalf("got\n%s", got)
	}
}

// Forms of a valid list the editor has to read through: a nested list and a
// deeper key that follows the entry's own fields, a blank line between
// entries, a second pin in single quotes, spacing after the colon, trailing
// blanks after an unquoted doc_id.
func TestRepinTextReadsTheShapesOfAValidList(t *testing.T) {
	text := "---\nsources:\n" +
		"  - doc_id: 01A\n    tags:\n      - x\n    content_hash:   \"sha256:aaa\"\n    revision: 3\n    meta:\n      revision: 99\n" +
		"\n" +
		"  - doc_id: '01B'\n    content_hash: \"sha256:bbb\"\n    revision: 5\n" +
		"  - doc_id: 01C  \n    content_hash: \"sha256:ccc\"\n    revision: 7\n" +
		"---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:a2", 4}, "01B": {"sha256:b2", 6}, "01C": {"sha256:c2", 8}})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nsources:\n" +
		"  - doc_id: 01A\n    tags:\n      - x\n    content_hash:   \"sha256:a2\"\n    revision: 4\n    meta:\n      revision: 99\n" +
		"\n" +
		"  - doc_id: '01B'\n    content_hash: \"sha256:b2\"\n    revision: 6\n" +
		"  - doc_id: 01C  \n    content_hash: \"sha256:c2\"\n    revision: 8\n" +
		"---\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// Two sources keys are not valid YAML; the first one is the one read.
func TestRepinTextReadsTheFirstSourcesKey(t *testing.T) {
	text := "---\nsources:\n  - doc_id: 01A\n    content_hash: \"x\"\n    revision: 1\nother: 1\nsources:\n  - doc_id: 01Z\n    content_hash: \"x\"\n    revision: 1\n---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}})
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(strings.Replace(text, "\"x\"", "\"sha256:n\"", 1), "revision: 1", "revision: 2", 1); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

const repinFlowPage = "---\n" +
	"title: T\n" +
	"generated: { by: x, at: 2026-08-23T17:30:00Z }\n" +
	"sources:\n" +
	"  - { id: a, resource: brain://project/x/docs/a.md, doc_id: 01A, content_hash: \"sha256:aaa\", revision: 1, title: Die Probe, Teil D }\n" +
	"  - {id: b, doc_id: '01B', content_hash: 'sha256:bbb', revision: 5}\n" +
	"  - { doc_id: 01C, meta: { revision: 99, doc_id: 01A }, content_hash:   sha256:ccc ,revision:  7 }  \n" +
	"---\n\n# T\n"

// A sources entry written as a flow mapping on one line changes in its two
// values only; quotes, spacing, a nested mapping and the comma in a later
// plain value stay as they were.
func TestRepinTextEditsFlowEntries(t *testing.T) {
	got, err := repinText(repinFlowPage, map[string]pinValue{
		"01A": {"sha256:a2", 2},
		"01B": {"sha256:b2", 6},
		"01C": {"sha256:c2", 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.NewReplacer(
		"content_hash: \"sha256:aaa\", revision: 1,", "content_hash: \"sha256:a2\", revision: 2,",
		"content_hash: 'sha256:bbb', revision: 5}", "content_hash: 'sha256:b2', revision: 6}",
		"content_hash:   sha256:ccc ,revision:  7 }", "content_hash:   sha256:c2 ,revision:  8 }",
	).Replace(repinFlowPage)
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// CRLF survives an edit of a flow entry.
func TestRepinTextKeepsCRLFInFlowEntries(t *testing.T) {
	crlf := strings.ReplaceAll(repinFlowPage, "\n", "\r\n")
	got, err := repinText(crlf, map[string]pinValue{"01B": {"sha256:b2", 6}})
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(crlf, "'sha256:bbb', revision: 5}", "'sha256:b2', revision: 6}", 1); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// Flow and block entries mix in one list, and a doc_id that stands twice gets
// the new values in both forms.
func TestRepinTextEditsEveryEntryOfADocIDInBothForms(t *testing.T) {
	text := "---\nsources:\n" +
		"  - { doc_id: 01A, content_hash: \"x\", revision: 1 }\n" +
		"  - doc_id: 01A\n    content_hash: 'y'\n    revision: 2\n" +
		"  - { doc_id: 01A, content_hash: z, revision: 3 }\n" +
		"---\nbody\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 5}})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nsources:\n" +
		"  - { doc_id: 01A, content_hash: \"sha256:n\", revision: 5 }\n" +
		"  - doc_id: 01A\n    content_hash: 'sha256:n'\n    revision: 5\n" +
		"  - { doc_id: 01A, content_hash: sha256:n, revision: 5 }\n" +
		"---\nbody\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// A flow entry that spans several lines is edited where its values stand,
// and the block entry after it is still read.
func TestRepinTextEditsAFlowEntryOverSeveralLines(t *testing.T) {
	text := "---\nsources:\n" +
		"  - { id: a,\n      doc_id: 01A, content_hash: \"x\",\n      revision: 1,\n      title: \"a, } b\" }\n" +
		"  - doc_id: 01B\n    content_hash: \"y\"\n    revision: 2\n" +
		"---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}, "01B": {"sha256:m", 3}})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nsources:\n" +
		"  - { id: a,\n      doc_id: 01A, content_hash: \"sha256:n\",\n      revision: 2,\n      title: \"a, } b\" }\n" +
		"  - doc_id: 01B\n    content_hash: \"sha256:m\"\n    revision: 3\n" +
		"---\n"
	if got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// Every flow form it cannot edit safely is refused rather than rewritten.
func TestRepinTextRefusesFlowEntriesItCannotEditSafely(t *testing.T) {
	cases := map[string]string{
		"not closed":              "---\nsources:\n  - { doc_id: 01A, content_hash: \"x\", revision: 1\n---\n",
		"unterminated quote":      "---\nsources:\n  - { doc_id: 01A, content_hash: \"x, revision: 1 }\n---\n",
		"escaped closing quote":   "---\nsources:\n  - { doc_id: 01A, content_hash: \"x\\\", revision: 1 }\n---\n",
		"comment":                 "---\nsources:\n  - { doc_id: 01A, # by hand\n      content_hash: \"x\", revision: 1 }\n---\n",
		"missing revision":        "---\nsources:\n  - { doc_id: 01A, content_hash: \"x\" }\n---\n",
		"missing hash":            "---\nsources:\n  - { doc_id: 01A, revision: 1 }\n---\n",
		"empty value":             "---\nsources:\n  - { doc_id: 01A, content_hash: , revision: 1 }\n---\n",
		"plain value over lines":  "---\nsources:\n  - { doc_id: 01A, content_hash: x\n      y, revision: 1 }\n---\n",
		"quoted value over lines": "---\nsources:\n  - { doc_id: 01A, content_hash: \"x\n      y\", revision: 1 }\n---\n",
		"text after a quote":      "---\nsources:\n  - { doc_id: 01A, content_hash: \"x\" y, revision: 1 }\n---\n",
		"entry not found":         "---\nsources:\n  - { doc_id: 01Z, content_hash: \"x\", revision: 1 }\n---\n",
		"empty last value":        "---\nsources:\n  - { doc_id: 01A, content_hash: \"x\", revision: }\n---\n",
	}
	for name, text := range cases {
		if _, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}}); err == nil {
			t.Errorf("%s: no refusal", name)
		}
	}
}

// The order of the two values inside the entry does not matter.
func TestRepinTextEditsFlowValuesInAnyOrder(t *testing.T) {
	text := "---\nsources:\n  - { revision: 1, doc_id: 01A, content_hash: \"x\" }\n---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 12}})
	if err != nil {
		t.Fatal(err)
	}
	if want := "---\nsources:\n  - { revision: 12, doc_id: 01A, content_hash: \"sha256:n\" }\n---\n"; got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// Doubled single quotes and backslash escapes are read through, not taken
// for the end of the value.
func TestRepinTextReadsEscapesInFlowEntries(t *testing.T) {
	text := "---\nsources:\n  - { title: 'it''s, }', note: \"a\\\"}\", doc_id: 01A, content_hash: \"x\", revision: 1 }\n---\n"
	got, err := repinText(text, map[string]pinValue{"01A": {"sha256:n", 2}})
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(strings.Replace(text, "\"x\"", "\"sha256:n\"", 1), "revision: 1", "revision: 2", 1); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func repin(t *testing.T, root string) RepinResult {
	t.Helper()
	got, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// The chain the wiki has: a doc, a topic citing it, a synthesis citing the
// topic. The doc changed; one run moves all three rows by exactly one, pins
// both pages, and leaves the check silent.
func TestRepinCarriesAChangeThroughThreeLinks(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	topic := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	synthesis := page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topic), 5})
	root := project(t, map[string]string{
		"docs/de/a.md":             now,
		"docs/wiki/topics/t.md":    topic,
		"docs/wiki/syntheses/s.md": synthesis,
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3},
		identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topic), Revision: 5},
		identity.Identity{DocID: "01S", Relative: "docs/wiki/syntheses/s.md", ContentHash: hashOf(synthesis), Revision: 9},
	)
	got := repin(t, root)
	want := RepinResult{
		Rows:  []RowAdvance{{"docs/de/a.md", 3, 4}, {"docs/wiki/topics/t.md", 5, 6}},
		Pages: []string{"syntheses/s.md", "topics/t.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result %+v\nwant %+v", got, want)
	}
	if findings := drift(t, root); len(findings) != 0 {
		t.Fatalf("after repin: %+v", findings)
	}
	register, err := identity.ReadIdentities(filepath.Join(root, "_identities.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	// The synthesis is cited by nobody: its row stays as it was although
	// its pin changed it.
	if row := register["docs/wiki/syntheses/s.md"]; row.Revision != 9 || row.ContentHash != hashOf(synthesis) {
		t.Fatalf("uncited row moved: %+v", row)
	}
	if again := repin(t, root); len(again.Rows)+len(again.Pages) != 0 {
		t.Fatalf("a second run changed %+v", again)
	}
}

// After a reindex in the registry's checkout the rows are current and only
// the pins are behind: repin writes the pins and no row. Two pages cite the
// same file, so its hash is taken once for both.
func TestRepinMovesPinsToCurrentRows(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
		"docs/wiki/topics/u.md": page("U", testPin{"docs/de/a.md", "01A", hashOf(old), 2}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(now), Revision: 4})
	before := read(t, root, "_identities.tsv")
	// A register that renders to the same bytes would hide a write; the seam
	// does not.
	saved := writeSource
	writeSource = func(path string, data []byte, perm os.FileMode) error {
		if filepath.Base(path) == "_identities.tsv" {
			t.Error("the register was written")
		}
		return os.WriteFile(path, data, perm)
	}
	defer func() { writeSource = saved }()
	got := repin(t, root)
	if len(got.Rows) != 0 || !reflect.DeepEqual(got.Pages, []string{"topics/t.md", "topics/u.md"}) {
		t.Fatalf("result %+v", got)
	}
	if read(t, root, "_identities.tsv") != before {
		t.Fatal("the register was rewritten")
	}
	if !strings.Contains(read(t, root, "docs/wiki/topics/t.md"), "content_hash: \""+hashOf(now)+"\"\n    revision: 4\n") {
		t.Fatalf("page %s", read(t, root, "docs/wiki/topics/t.md"))
	}
}

// A page that changed by hand and is cited moves its own row, and its
// citers follow; a CRLF page keeps CRLF and hashes folded.
func TestRepinFollowsAHandEditOfACitedCRLFPage(t *testing.T) {
	topicOld := strings.ReplaceAll(page("T"), "\n", "\r\n")
	topicNow := strings.ReplaceAll(page("T")+"more\n", "\n", "\r\n")
	root := project(t, map[string]string{
		"docs/wiki/topics/t.md":    topicNow,
		"docs/wiki/syntheses/s.md": strings.ReplaceAll(page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topicOld), 2}), "\n", "\r\n"),
	}, identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topicOld), Revision: 2})
	got := repin(t, root)
	if !reflect.DeepEqual(got.Rows, []RowAdvance{{"docs/wiki/topics/t.md", 2, 3}}) {
		t.Fatalf("rows %+v", got.Rows)
	}
	synthesis := read(t, root, "docs/wiki/syntheses/s.md")
	if strings.Count(synthesis, "\r\n") != strings.Count(synthesis, "\n") {
		t.Fatalf("CRLF lost: %q", synthesis)
	}
	if findings := drift(t, root); len(findings) != 0 {
		t.Fatalf("after repin: %+v", findings)
	}
}

// A page whose pins are all current is not edited, whatever form its sources
// block has: repin must not fail on a page the check never named. Flow list,
// a comment on a pin line, and a plain page that is fine all stay as they are
// beside a stale page that is written.
func TestRepinLeavesPagesWithCurrentPinsAlone(t *testing.T) {
	old, now, b := "# A\n", "# A changed\n", "# B\n"
	flow := "---\ntype: Topic\ntitle: U\ndescription: d\nsources: [{id: b, resource: brain://" + testScope + "/docs/de/b.md, doc_id: 01B, content_hash: \"" + hashOf(b) + "\", revision: 4}]\n---\n\n# U\n"
	current := page("V", testPin{"docs/de/b.md", "01B", hashOf(b), 4})
	cases := map[string]string{
		"flow list": flow,
		"comment":   strings.Replace(current, "revision: 4", "revision: 4 # pinned", 1),
		"plain":     current,
	}
	for name, other := range cases {
		stale := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
		root := project(t, map[string]string{
			"docs/de/a.md":          now,
			"docs/de/b.md":          b,
			"docs/wiki/topics/t.md": stale,
			"docs/wiki/topics/u.md": other,
		},
			identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3},
			identity.Identity{DocID: "01B", Relative: "docs/de/b.md", ContentHash: hashOf(b), Revision: 4},
		)
		got, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := RepinResult{Rows: []RowAdvance{{"docs/de/a.md", 3, 4}}, Pages: []string{"topics/t.md"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: result %+v\nwant %+v", name, got, want)
		}
		if read(t, root, "docs/wiki/topics/u.md") != other {
			t.Fatalf("%s: the current page was rewritten", name)
		}
		if findings := drift(t, root); len(findings) != 0 {
			t.Fatalf("%s: after repin: %+v", name, findings)
		}
	}
}

// A pin that is behind in only one of its two values is edited: the other
// value alone does not make it current.
func TestRepinEditsAPinThatIsBehindInOneValue(t *testing.T) {
	b := "# B\n"
	cases := map[string]testPin{
		"hash only":     {"docs/de/b.md", "01B", hashOf("# other\n"), 4},
		"revision only": {"docs/de/b.md", "01B", hashOf(b), 3},
	}
	for name, pin := range cases {
		root := project(t, map[string]string{"docs/de/b.md": b, "docs/wiki/topics/u.md": page("U", pin)},
			identity.Identity{DocID: "01B", Relative: "docs/de/b.md", ContentHash: hashOf(b), Revision: 4})
		if got := repin(t, root); len(got.Rows) != 0 || !reflect.DeepEqual(got.Pages, []string{"topics/u.md"}) {
			t.Fatalf("%s: result %+v", name, got)
		}
		if findings := drift(t, root); len(findings) != 0 {
			t.Fatalf("%s: after repin: %+v", name, findings)
		}
	}
}

// A page that cites one file in two entries is pinned in both.
func TestRepinPinsEveryEntryThatCitesTheFile(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	text := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}, testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	root := project(t, map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": text},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	repin(t, root)
	pinned := "content_hash: \"" + hashOf(now) + "\"\n    revision: 4\n"
	if got := read(t, root, "docs/wiki/topics/t.md"); strings.Count(got, pinned) != 2 {
		t.Fatalf("got\n%s", got)
	}
	if findings := drift(t, root); len(findings) != 0 {
		t.Fatalf("after repin: %+v", findings)
	}
}

// A page with a foreign pin and an incomplete entry beside the stale one:
// only the stale entry changes.
func TestRepinTouchesOnlyOwnCompleteEntries(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	text := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	text = strings.Replace(text, "---\n\n# T", "  - id: f\n    resource: brain://shared/kb/x.md\n    doc_id: 01F\n    content_hash: \"sha256:f\"\n    revision: 1\n  - id: i\n    resource: brain://project/x/docs/de/a.md\n---\n\n# T", 1)
	root := project(t, map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": text},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	repin(t, root)
	want := strings.Replace(text, "content_hash: \""+hashOf(old)+"\"\n    revision: 3", "content_hash: \""+hashOf(now)+"\"\n    revision: 4", 1)
	if got := read(t, root, "docs/wiki/topics/t.md"); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// What repin may not fix stops it before any write: row-missing,
// source-missing, doc-id and cycle findings, and a page it cannot edit safely, even when another page is fixable.
func TestRepinRefusesBeforeItWrites(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	fixable := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	// Rows for the cycle's pages and the vanished file exist, so that the one
	// rule named in each case is the only thing that blocks.
	rows := []identity.Identity{
		{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3},
		{DocID: "01G", Relative: "docs/de/gone.md", ContentHash: "sha256:g", Revision: 1},
		{DocID: "01U", Relative: "docs/wiki/topics/u.md", ContentHash: "sha256:u", Revision: 1},
		{DocID: "01V", Relative: "docs/wiki/topics/v.md", ContentHash: "sha256:v", Revision: 1},
	}
	// rule is what the refusal names; "" is a page repin cannot edit, which
	// is an error of its own and no finding.
	cases := map[string]struct {
		rule  string
		pages map[string]string
	}{
		"row missing": {RuleRowMissing, map[string]string{
			"docs/wiki/topics/u.md": page("U", testPin{"docs/de/new.md", "01N", "sha256:n", 1})}},
		"source missing": {RuleSourceMissing, map[string]string{
			"docs/wiki/topics/u.md": page("U", testPin{"docs/de/gone.md", "01G", "sha256:g", 1})}},
		"doc_id differs": {RuleDocID, map[string]string{
			"docs/wiki/topics/u.md": page("U", testPin{"docs/de/a.md", "01X", hashOf(old), 3})}},
		"cycle": {RuleCycle, map[string]string{
			"docs/wiki/topics/u.md": page("U", testPin{"docs/wiki/topics/v.md", "01V", "sha256:v", 1}),
			"docs/wiki/topics/v.md": page("V", testPin{"docs/wiki/topics/u.md", "01U", "sha256:u", 1})}},
		"not editable": {"", map[string]string{
			"docs/wiki/topics/u.md": strings.Replace(page("U", testPin{"docs/de/a.md", "01A", hashOf(old), 3}), "revision: 3", "revision: 3 # hand", 1)}},
	}
	for name, c := range cases {
		files := map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": fixable}
		for path, text := range c.pages {
			files[path] = text
		}
		root := project(t, files, rows...)
		register := read(t, root, "_identities.tsv")
		_, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
		var refusal *RepinRefusal
		switch {
		case err == nil:
			t.Fatalf("%s: no refusal", name)
		case c.rule == "" && errors.As(err, &refusal):
			t.Fatalf("%s: a page error came as a refusal: %v", name, err)
		case c.rule != "" && (!errors.As(err, &refusal) || len(refusal.Findings) != 1 || refusal.Findings[0].Rule != c.rule):
			t.Fatalf("%s: err %v", name, err)
		}
		if read(t, root, "docs/wiki/topics/t.md") != fixable || read(t, root, "_identities.tsv") != register {
			t.Fatalf("%s: something was written", name)
		}
		for path, text := range c.pages {
			if read(t, root, path) != text {
				t.Fatalf("%s: %s was written", name, path)
			}
		}
		if c.rule != "" && refusal.Error() != "1 finding(s) repin does not fix" {
			t.Fatalf("%s: message %q", name, refusal.Error())
		}
	}
}

// Nothing to repin: no register, or nothing behind.
func TestRepinWithNothingToDo(t *testing.T) {
	doc := "# A\n"
	empty := project(t, map[string]string{"docs/wiki/topics/t.md": page("T")})
	current := project(t, map[string]string{"docs/de/a.md": doc, "docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(doc), 2})},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(doc), Revision: 2})
	for _, root := range []string{empty, current} {
		if got := repin(t, root); len(got.Rows)+len(got.Pages) != 0 {
			t.Fatalf("result %+v", got)
		}
	}
}

// A write that fails stops the run and names what was written before it.
func TestRepinReportsAWriteFailure(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	root := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	boom := errors.New("disk full")
	saved := writeSource
	writeSource = func(path string, data []byte, perm os.FileMode) error {
		if filepath.Base(path) == "_identities.tsv" {
			return boom
		}
		return os.WriteFile(path, data, perm)
	}
	defer func() { writeSource = saved }()
	got, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
	if !errors.Is(err, boom) || !reflect.DeepEqual(got.Pages, []string{"topics/t.md"}) || len(got.Rows) != 0 {
		t.Fatalf("result %+v, err %v", got, err)
	}
	// A page that cannot be written stops the run before the register.
	fresh := project(t, map[string]string{
		"docs/de/a.md":          now,
		"docs/wiki/topics/t.md": page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}),
	}, identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	register := read(t, fresh, "_identities.tsv")
	writeSource = func(string, []byte, os.FileMode) error { return boom }
	got, err = Repin(fresh, filepath.Join(fresh, "docs", "wiki"), testScope)
	if !errors.Is(err, boom) || len(got.Pages)+len(got.Rows) != 0 || read(t, fresh, "_identities.tsv") != register {
		t.Fatalf("result %+v, err %v", got, err)
	}
}

// A cited file that reads for the check and fails when the plan reads it
// again stops the run before any write.
func TestRepinReportsAFailureWhilePlanning(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	text := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	root := project(t, map[string]string{"docs/de/a.md": now, "docs/wiki/topics/t.md": text},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3})
	boom := errors.New("gone between two reads")
	reads := 0
	saved := readSource
	readSource = func(path string) ([]byte, error) {
		if filepath.Base(path) == "a.md" {
			reads++
			if reads > 1 {
				return nil, boom
			}
		}
		return os.ReadFile(path)
	}
	defer func() { readSource = saved }()
	if _, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
	if read(t, root, "docs/wiki/topics/t.md") != text {
		t.Fatal("a page was written")
	}
}

// Load and drift failures reach the caller unchanged.
func TestRepinReportsAReadFailure(t *testing.T) {
	root := project(t, map[string]string{"docs/de/a.md": "# A\n"},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: "sha256:a", Revision: 1})
	if _, err := Repin(root, filepath.Join(root, "docs", "missing"), testScope); err == nil {
		t.Fatal("a wiki root that does not exist must fail")
	}
	writeAt(t, root, "docs/wiki/topics/t.md", page("T", testPin{"docs/de/a.md", "01A", "sha256:a", 1}))
	boom := errors.New("denied")
	saved := readSource
	readSource = func(path string) ([]byte, error) {
		if filepath.Base(path) == "a.md" {
			return nil, boom
		}
		return os.ReadFile(path)
	}
	defer func() { readSource = saved }()
	if _, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
}

// A cited page that cannot be edited safely stops the run when a page that
// sorts before it reaches it through its pin, and the error names the citer.
func TestRepinRefusesAnUneditablePageReachedThroughACitation(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	topic := strings.Replace(page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3}), "revision: 3", "revision: 3 # hand", 1)
	synthesis := page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topic), 5})
	root := project(t, map[string]string{
		"docs/de/a.md":             now,
		"docs/wiki/topics/t.md":    topic,
		"docs/wiki/syntheses/s.md": synthesis,
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3},
		identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topic), Revision: 5},
	)
	_, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope)
	if err == nil || !strings.HasPrefix(err.Error(), "topics/t.md: ") || !strings.Contains(err.Error(), "carries a comment") {
		t.Fatalf("err %v", err)
	}
	if read(t, root, "docs/wiki/topics/t.md") != topic || read(t, root, "docs/wiki/syntheses/s.md") != synthesis {
		t.Fatal("a page was written")
	}
}

// The result is sorted by path, not by the order the walk or a map hands
// out: five rows are too many to come out sorted by chance, and `a-b/y.md`
// sorts before `a/x.md` as a string although the walk reaches `a` first.
func TestRepinSortsRowsAndPages(t *testing.T) {
	old, now := "# old\n", "# new\n"
	ids := []string{"01A", "01B", "01C", "01D", "01E"}
	files := map[string]string{}
	var rows []identity.Identity
	var pins []testPin
	for i, id := range ids {
		path := "docs/de/d" + itoa(i) + ".md"
		files[path] = now
		rows = append(rows, identity.Identity{DocID: id, Relative: path, ContentHash: hashOf(old), Revision: 1})
		pins = append(pins, testPin{path, id, hashOf(old), 1})
	}
	// Cited in descending order: no rotation of the order the plan meets the
	// files in is sorted.
	slices.Reverse(pins)
	files["docs/wiki/topics/t.md"] = page("T", pins...)
	slices.Reverse(pins)
	files["docs/wiki/a/x.md"] = page("X", pins[0])
	files["docs/wiki/a-b/y.md"] = page("Y", pins[1])
	got := repin(t, project(t, files, rows...))
	var want []RowAdvance
	for i := range ids {
		want = append(want, RowAdvance{"docs/de/d" + itoa(i) + ".md", 1, 2})
	}
	if !reflect.DeepEqual(got.Rows, want) || !reflect.DeepEqual(got.Pages, []string{"a-b/y.md", "a/x.md", "topics/t.md"}) {
		t.Fatalf("result %+v", got)
	}
}

// A cited page that reads for the page walk and fails when the check hashes
// it as a cited file stops the run: the check's own failure is not ignored
// just because the plan would not read that file again.
func TestRepinReportsAFailureWhileChecking(t *testing.T) {
	old, now := "# A\n", "# A changed\n"
	topic := page("T", testPin{"docs/de/a.md", "01A", hashOf(old), 3})
	root := project(t, map[string]string{
		"docs/de/a.md":             now,
		"docs/wiki/topics/t.md":    topic,
		"docs/wiki/syntheses/s.md": page("S", testPin{"docs/wiki/topics/t.md", "01T", hashOf(topic), 5}),
	},
		identity.Identity{DocID: "01A", Relative: "docs/de/a.md", ContentHash: hashOf(old), Revision: 3},
		identity.Identity{DocID: "01T", Relative: "docs/wiki/topics/t.md", ContentHash: hashOf(topic), Revision: 5},
	)
	boom := errors.New("denied on the second read")
	reads := 0
	saved := readSource
	readSource = func(path string) ([]byte, error) {
		if filepath.Base(path) == "t.md" {
			reads++
			if reads > 1 {
				return nil, boom
			}
		}
		return os.ReadFile(path)
	}
	defer func() { readSource = saved }()
	if _, err := Repin(root, filepath.Join(root, "docs", "wiki"), testScope); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
}
