package wiki

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldSetMatchesThePythonSide(t *testing.T) {
	// Scheibe 3 §4 names exactly these five. `log.md` and `audit.md` were
	// missing here, so both were linted as concept pages and reported
	// missing-type; README.md and SPEC.md were listed although a bundle
	// holds neither.
	want := []string{"_schema.md", "index.md", "log.md", "audit.md", "_identities.tsv"}
	if len(ScaffoldFiles) != len(want) {
		t.Fatalf("scaffold set has %d entries, want %d", len(ScaffoldFiles), len(want))
	}
	for _, name := range want {
		if !ScaffoldFiles[name] {
			t.Errorf("%s is not treated as scaffold", name)
		}
	}
	for _, name := range []string{"README.md", "SPEC.md", "changelog.md", "_template.md"} {
		if ScaffoldFiles[name] {
			t.Errorf("%s is treated as scaffold but a bundle holds none", name)
		}
	}
}

func TestUnreadableFrontmatterIsCarried(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.md")
	os.WriteFile(path, []byte("---\ntype: [unclosed\n---\nbody\n"), 0o644)
	page, err := ReadPage(path, dir)
	if err != nil {
		t.Fatalf("ReadPage returned %v; a broken page is a finding, not an error", err)
	}
	if page.BrokenFrontmatter == nil {
		t.Fatal("the YAML error was swallowed")
	}
}

func TestSourceCarriesTheFieldsMaintenanceNeeds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	os.WriteFile(path, []byte(
		"---\ntype: Topic\nsources:\n  - id: a\n    resource: r\n"+
			"    doc_id: d\n    content_hash: sha256:x\n    revision: 2\n---\nbody\n"), 0o644)
	page, _ := ReadPage(path, dir)
	s := page.Sources[0]
	if s.ID != "a" || s.DocID != "d" || s.ContentHash != "sha256:x" || s.Revision == nil || *s.Revision != 2 {
		t.Fatalf("source parsed as %+v; doc_id, content_hash and revision are what the maintenance layer hangs on", s)
	}
}

func TestRealizationUsesThePythonNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	os.WriteFile(path, []byte("---\ntype: Topic\nrealization: implemented\nimplemented_in: abc123\n---\nb\n"), 0o644)
	page, _ := ReadPage(path, dir)
	if page.Realization == nil || *page.Realization != "implemented" {
		t.Error("realization not read; Go called it RealizedCommit")
	}
	if page.ImplementedIn == nil || *page.ImplementedIn != "abc123" {
		t.Error("implemented_in not read")
	}
}

// The brief's four cases leave the lifecycle fields, `generated` and `mtime`
// unmeasured, although the model gains all three here. This case measures
// them, so that no field enters the model on an assumption -- in particular
// that yaml.v3 turns the date-only scalar of `stale_after` into a time.Time
// rather than refusing it.
func TestLifecycleAndProvenanceAreRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	os.WriteFile(path, []byte(
		"---\ntitle: A Page\ntype: Topic\ndescription: what it is\n"+
			"status: current\nstale_after: 2026-03-01\ngenerated:\n  by: brain\n---\nb\n"), 0o644)
	page, _ := ReadPage(path, dir)
	// Verbatim, in the case the file wrote. The reader normalises
	// nothing, which is what lets `config.KnowsType` compare the
	// letters the way `rank_of` does.
	if page.PageType == nil || *page.PageType != "Topic" {
		t.Errorf("page_type parsed as %v", page.PageType)
	}
	if page.Title != "A Page" || page.Description != "what it is" {
		t.Errorf("title %q, description %q", page.Title, page.Description)
	}
	if page.Status == nil || *page.Status != "current" {
		t.Errorf("status parsed as %v", page.Status)
	}
	if page.StaleAfter == nil {
		t.Fatal("stale_after not read")
	}
	if got := page.StaleAfter.Format("2006-01-02"); got != "2026-03-01" {
		t.Errorf("stale_after parsed as %s", got)
	}
	// Fatal, not Errorf: the `At` check below dereferences this, and a run
	// that ends in a nil panic names the test's line instead of the defect.
	if page.Generated == nil {
		t.Fatal("generated not read")
	}
	if page.Generated.By != "brain" {
		t.Errorf("generated.by parsed as %q", page.Generated.By)
	}
	// OKF demands only `by`; a page without `at` is no finding, so the zero
	// value must stay distinguishable from a stated timestamp.
	if page.Generated.At != nil {
		t.Errorf("generated.at invented as %v", page.Generated.At)
	}
	if page.ModTime.IsZero() {
		t.Error("mtime not read; the untouched-days rules hang on it")
	}
}

func TestTheKeySetAndTheRuntimeAreRead(t *testing.T) {
	// The typed Frontmatter drops every key it does not name, and OKF §12
	// judges a bundle-root index.md by exactly the keys it carries. The
	// written order is kept, because two runs of the check have to render the
	// same output and a map would not give that.
	dir := t.TempDir()
	path := filepath.Join(dir, "c.md")
	os.WriteFile(path, []byte("---\nzulu: 1\ntype: Attested Computation\nruntime: bigquery\n---\nbody\n"), 0o644)
	page, err := ReadPage(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasFrontmatter {
		t.Error("a page opening with a block is not marked as carrying one")
	}
	if page.Runtime != "bigquery" {
		t.Errorf("runtime is %q; OKF §10.2 requires it for this type", page.Runtime)
	}
	want := []string{"zulu", "type", "runtime"}
	if len(page.FrontmatterKeys) != len(want) {
		t.Fatalf("keys are %v, want %v", page.FrontmatterKeys, want)
	}
	for i, key := range want {
		if page.FrontmatterKeys[i] != key {
			t.Errorf("key %d is %q, want %q -- the written order is lost", i, page.FrontmatterKeys[i], key)
		}
	}
}

func TestAFileWithoutAFrontmatterBlockSaysSo(t *testing.T) {
	// An empty block and no block leave every value zero alike, and OKF §8
	// permits a block in exactly one index.md of a bundle -- so the two
	// states have to stay apart.
	dir := t.TempDir()
	path := filepath.Join(dir, "index.md")
	os.WriteFile(path, []byte("# c\n\n* [a](a.md)\n"), 0o644)
	page, err := ReadPage(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if page.HasFrontmatter {
		t.Error("a file without a block is marked as carrying one")
	}
	if len(page.FrontmatterKeys) != 0 {
		t.Errorf("keys %v were read from a file that has none", page.FrontmatterKeys)
	}
}

func TestTwoRuleLinesAreNotAFrontmatterBlock(t *testing.T) {
	// The opening fence has to stand at the start of the file, not merely at
	// the start of some line. A document with no frontmatter that uses `---`
	// as a horizontal rule twice would otherwise be read as a block reaching
	// from the first rule to the second, and everything above the second one
	// would drop out of the body.
	dir := t.TempDir()
	body := "# Title\n\nintro\n\n---\n\nmiddle\n\n---\n\nend\n"
	path := filepath.Join(dir, "doc.md")
	os.WriteFile(path, []byte(body), 0o644)
	page, err := ReadPage(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if page.HasFrontmatter {
		t.Error("two rule lines were read as a frontmatter block")
	}
	if page.RawBody != body {
		t.Errorf("the body lost its opening; got %q", page.RawBody)
	}
	if page.PageType != nil || page.BrokenFrontmatter != nil {
		t.Errorf("a type or a parse error was invented: type=%v broken=%v",
			page.PageType, page.BrokenFrontmatter)
	}
}

func TestAYAMLExampleInAFenceIsNotFrontmatter(t *testing.T) {
	// The same anchor from the other side: a page that documents the format
	// shows a frontmatter block inside a code fence. That is the page most
	// likely to carry `---` lines, and reading its example as the page's own
	// frontmatter would give it a type it never claimed.
	dir := t.TempDir()
	body := "# Doc\n\n```yaml\n---\ntype: example\n---\n```\n\ntail\n"
	path := filepath.Join(dir, "guide.md")
	os.WriteFile(path, []byte(body), 0o644)
	page, err := ReadPage(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	if page.HasFrontmatter {
		t.Error("a fenced example was read as the page's frontmatter")
	}
	if page.PageType != nil {
		t.Errorf("the page took a type from the example: %q", *page.PageType)
	}
	if page.RawBody != body {
		t.Errorf("the body lost its opening; got %q", page.RawBody)
	}
}

// linksOf reads one body through ReadPage and answers the targets it
// recorded, so that a case is one line and the pattern is asked through
// the reader rather than directly.
func linksOf(t *testing.T, body string) []string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "p.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	page, err := ReadPage(path, dir)
	if err != nil {
		t.Fatal(err)
	}
	return page.Links
}

func TestAnEmbeddedImageIsNoLink(t *testing.T) {
	// Python's `_MARKDOWN_LINK` refuses one with a negative lookbehind
	// (`src/brain/document.py:11`, `(?<!!)`), and the consequence is not
	// cosmetic: `house/dead-link` is an error, so a page holding a
	// picture that is not a wiki page would fail a run that Python lets
	// through -- and a picture is the most ordinary thing in a wiki.
	//
	// Two inputs, because the rule is "the byte before is `!`" and not
	// "this is a Markdown image". The second is a link, on both sides.
	if got := linksOf(t, "# t\n\n![Diagramm](bilder/d.png)\n"); len(got) != 0 {
		t.Fatalf("links = %v for an embedded image, want none", got)
	}
	if got := linksOf(t, "# t\n\nWow![x](y.md)\n"); len(got) != 0 {
		t.Fatalf("links = %v; Python's lookbehind reads the byte, "+
			"not the syntax", got)
	}
	// The plain link next to it, so the case is not passed by a reader
	// that records nothing at all.
	if got := linksOf(t, "# t\n\n[x](y.md)\n"); len(got) != 1 ||
		got[0] != "y.md" {
		t.Fatalf("links = %v, want [y.md]", got)
	}
	// A link at the very first byte of the body: there is no byte
	// before it, and a reader that looked at one anyway would run off
	// the front. An image cannot reach index 0 -- its `[` stands
	// behind the `!` -- so a picture is not the case that finds this.
	if got := linksOf(t, "[first](y.md)\n"); len(got) != 1 ||
		got[0] != "y.md" {
		t.Fatalf("links = %v for a link at the body's first byte", got)
	}
	// An image at the start of the body all the same, because the two
	// are one byte apart and the guard has to hold for both.
	if got := linksOf(t, "![d](d.png)\n"); len(got) != 0 {
		t.Fatalf("links = %v for an image at the body's start", got)
	}
	// A linked image. Measured against Python rather than assumed:
	// both sides record `i.png` here, not `y.md` -- the outer `[` is
	// unpreceded, and the text class then runs to the *first* `]`,
	// which is the image's. A blind spot the two share, and the fix
	// does not close it: a picture inside a link still reaches
	// `dead-link`.
	if got := linksOf(t, "# t\n\n[![alt](i.png)](y.md)\n"); len(got) != 1 ||
		got[0] != "i.png" {
		t.Fatalf("links = %v, want [i.png] for a linked image", got)
	}
	// Three targets on one line, the middle one a picture: the scan
	// has to go on past what it refuses, not stop at it.
	got := linksOf(t, "# t\n\n[a](x.md) ![i](p.png) [b](z.md)\n")
	if len(got) != 2 || got[0] != "x.md" || got[1] != "z.md" {
		t.Fatalf("links = %v, want [x.md z.md]", got)
	}
	// The one input where a lookbehind and a filter on the finished
	// match part company: the refused `[` is not where the pattern
	// would start again. Measured, Python answers `c.md` here --
	// it retries one byte on rather than dropping the line.
	if got := linksOf(t, "# t\n\n![a[b](c.md)\n"); len(got) != 1 ||
		got[0] != "c.md" {
		t.Fatalf("links = %v, want [c.md]", got)
	}
}

func TestALinkWithNoTextIsStillALink(t *testing.T) {
	// `mdLinkRe` asked for `[^\]]+` where Python asks for `[^\]]*`
	// (`src/brain/document.py:11`), so `[](x.md)` reached `Links` on one
	// side and not the other. It is the harmless direction -- Go stayed
	// silent where Python reports a dead link -- but it is silence about
	// a real edge: `house/orphan` counts one incoming link fewer, and a
	// page reachable only through such a line is called orphaned here
	// and not there.
	//
	// Measured before it was changed: no tracked `.md` in this
	// repository carries the form, so nothing in the stock moves.
	if got := linksOf(t, "# t\n\n[](x.md)\n"); len(got) != 1 ||
		got[0] != "x.md" {
		t.Fatalf("links = %v, want [x.md]", got)
	}
	// The image rule still wins over it: an empty text does not make
	// one a link.
	if got := linksOf(t, "# t\n\n![](b.png)\n"); len(got) != 0 {
		t.Fatalf("links = %v for an image with no alt text, want none",
			got)
	}
}
