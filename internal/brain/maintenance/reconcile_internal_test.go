package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/brain/model"
	"github.com/xidus90/loomux/internal/config"
)

// landing is the smallest world one case can be landed in: an area with a wiki
// holding one page, a review centre beside it and one changed source.
//
// Built here rather than through the world helper of reconcile_test.go,
// because these cases are about what `landCase` does when a write fails, and
// Go cannot lend an internal test the fixtures of an external one. The whole
// detection above `landCase` has nothing to say about a failed write.
type landing struct {
	area       config.Area
	manifest   *config.Manifest
	reviewRoot string
	sources    []Changed
	now        time.Time
	broken     []string
	proposer   *model.Proposer
}

func newLanding(t *testing.T) *landing {
	t.Helper()
	root := t.TempDir()
	area := config.Area{
		Scope:    "project/a",
		Path:     filepath.Join(root, "area"),
		WikiPath: filepath.Join(root, "area", "docs", "wiki"),
	}
	if err := os.MkdirAll(area.WikiPath, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	page := filepath.Join(area.WikiPath, "a.md")
	if err := os.WriteFile(page, []byte("---\ntype: note\n---\n\nWhat it says.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile page: %v", err)
	}
	baseline := "package a\n"
	return &landing{
		area:       area,
		manifest:   &config.Manifest{Scope: "project/a", PrivacyMode: "manual_cloud"},
		reviewRoot: filepath.Join(root, "review"),
		sources: []Changed{{
			DocID: "00000000000000000000000001", Relative: "src/a.go", Revision: 1,
			ContentHash: "sha256:aa", Text: "package a\n\nfunc B() {}\n", Baseline: &baseline,
		}},
		now: time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC),
	}
}

func (l *landing) land(t *testing.T) (Case, error) {
	t.Helper()
	return landCase(context.Background(), l.area, l.manifest, l.reviewRoot, "a.md", l.sources, l.now, &l.broken,
		"source_change", "source_changed", nil, l.proposer)
}

// directory is where the case of this landing will come to lie.
func (l *landing) directory() string {
	return CaseDir(l.reviewRoot, "project-a", CaseID(l.area.Scope, "a.md", l.now))
}

// blockWith puts a directory where the case wants to write a file, which is
// the one thing a reviewer can leave behind that no write can go around.
func blockWith(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
}

// A place for the case that cannot be made is a refusal, not a case written
// somewhere else.
func TestLandCaseRefusesADirectoryItCannotMake(t *testing.T) {
	l := newLanding(t)
	if err := os.MkdirAll(l.reviewRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// A regular file where the scope's folder belongs.
	if err := os.WriteFile(filepath.Join(l.reviewRoot, "project-a"), nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase wrote a case although its directory could not be made")
	}
}

// The target hash is taken off the page the case is about; a page that cannot
// be hashed leaves the case without the field that says which version was
// under review.
func TestLandCaseCarriesAFailedTargetHash(t *testing.T) {
	l := newLanding(t)
	withSeam(t, &contentHashFn, func(string) (string, error) { return "", errors.New("hash broke") })
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase wrote a case for a page it could not hash")
	}
}

// The package is what the decision may quote from; a page that cannot be read
// leaves nothing to quote, so no case is written either.
func TestLandCaseCarriesAFailedPageRead(t *testing.T) {
	l := newLanding(t)
	withSeam(t, &readFileFn, func(string) ([]byte, error) { return nil, errors.New("read broke") })
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase wrote a case for a page it could not read")
	}
}

// case.toml and package.md, each blocked in turn: a case whose file could not
// be written is no case, and the caller has to hear about it.
func TestLandCaseCarriesAFailedWrite(t *testing.T) {
	for _, name := range []string{caseName, packageName} {
		t.Run(name, func(t *testing.T) {
			l := newLanding(t)
			blockWith(t, filepath.Join(l.directory(), name))
			if _, err := l.land(t); err == nil {
				t.Fatalf("landCase reported success although %s could not be written", name)
			}
		})
	}
}

// A proposal that passed the binding and cannot be written is a refusal too:
// case.toml would carry a prompt_version for a file that is not there.
func TestLandCaseCarriesAFailedProposalWrite(t *testing.T) {
	l := newLanding(t)
	answer := "## B1 - B kam hinzu.\n\nevidence: D1\n\n```\n+func B() {}\n```\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"response": answer})
	}))
	t.Cleanup(server.Close)
	settings, err := config.ParseModelSettings("config.toml",
		"[model]\nenabled = true\nendpoint = \""+server.URL+"\"\n")
	if err != nil {
		t.Fatalf("ParseModelSettings: %v", err)
	}
	if l.proposer, err = model.ProposerFor(settings, l.manifest, "propose"); err != nil || l.proposer == nil {
		t.Fatalf("ProposerFor: %v %v", l.proposer, err)
	}
	blockWith(t, filepath.Join(l.directory(), proposalName))
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase reported success although proposal.md could not be written")
	}
}

// standing lands one case and answers where it came to lie, so a second
// landing meets it as the case on disk.
func (l *landing) standing(t *testing.T) string {
	t.Helper()
	landed, err := l.land(t)
	if err != nil {
		t.Fatalf("landCase: %v", err)
	}
	return CaseDir(l.reviewRoot, "project-a", landed.ID)
}

// moveOn rewrites the source so the next landing finds the case's sources
// changed and rebuilds it.
func (l *landing) moveOn() {
	l.sources[0].ContentHash = "sha256:bb"
	l.sources[0].Text = "package a\n\nfunc C() {}\n"
}

// A proposal that cannot be carried aside is a refusal: losing the reviewer's
// work quietly is the one outcome this step exists to prevent.
func TestSupersedeCarriesAFailedWrite(t *testing.T) {
	l := newLanding(t)
	directory := l.standing(t)
	if err := os.WriteFile(filepath.Join(directory, proposalName), []byte("a proposal\n"), 0o644); err != nil {
		t.Fatalf("WriteFile proposal: %v", err)
	}
	blockWith(t, filepath.Join(directory, supersededName))
	l.moveOn()
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase discarded a proposal it could not carry aside")
	}
}

// Twice on one day the new case lands in the old one's directory, so all that
// is left to do is drop the spent proposal -- and a proposal that will not go
// is a refusal rather than a case with a stale answer beside it.
func TestSupersedeCarriesAFailedRemoval(t *testing.T) {
	l := newLanding(t)
	directory := l.standing(t)
	// A directory under the proposal's name, with something in it: Remove
	// takes an empty one and refuses this.
	blockWith(t, filepath.Join(directory, proposalName, "notes"))
	l.moveOn()
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase went on although the spent proposal stayed")
	}
}

// A spent case directory that cannot be discarded is a refusal too: leaving it
// standing would put a second case for one page in the review centre, and the
// reviewer would decide the older one.
//
// Windows only, and that is the honest place for it: the vault is open, an
// editor holding a file of the case is the ordinary way this happens, and
// POSIX unlinks an open file without complaint. The coverage gate of this
// repository runs on Windows for the same kind of reason (ci.yml).
func TestSupersedeCarriesAFailedDiscard(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("an open handle does not hold a file back outside Windows")
	}
	l := newLanding(t)
	directory := l.standing(t)
	note := filepath.Join(directory, "note.md")
	if err := os.WriteFile(note, []byte("the reviewer's own note\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	held, err := os.Open(note)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer held.Close()

	l.moveOn()
	// A day later, so the new case wants a directory of its own and the old
	// one has to go.
	l.now = l.now.Add(24 * time.Hour)
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase opened a second case beside one it could not discard")
	}
}

// A standing case whose flag has to be brought up to date and whose file will
// not take the change is a refusal: answering the new flag while the file
// still says the old one would make the reader and the file disagree about
// where the package may go.
//
// Windows only, for the reason TestSupersedeCarriesAFailedDiscard states: a
// read-only file refuses the swap there, while POSIX judges by the directory.
func TestWithCurrentModeCarriesAFailedWrite(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a read-only file does not refuse a rename outside Windows")
	}
	l := newLanding(t)
	directory := l.standing(t)
	path := filepath.Join(directory, caseName)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	// The area closes after the case was opened, so the flag has to follow.
	l.manifest.PrivacyMode = localOnlyMode
	if _, err := l.land(t); err == nil {
		t.Fatal("landCase answered a flag it could not write")
	}
}

// A folder in the review centre that holds no case.toml is none of this
// function's business: a reviewer may keep notes of their own beside a case.
func TestStandingCaseWalksPastAFolderWithoutACase(t *testing.T) {
	l := newLanding(t)
	directory := l.standing(t)
	blockWith(t, filepath.Join(filepath.Dir(directory), "the reviewer's notes"))
	// Nothing moved, so the standing case has to be found again -- past the
	// folder, which sorts before it.
	landed, err := l.land(t)
	if err != nil {
		t.Fatalf("landCase: %v", err)
	}
	if got := CaseDir(l.reviewRoot, "project-a", landed.ID); got != directory {
		t.Fatalf("the case moved to %q, want %q", got, directory)
	}
}

// A page that carries no frontmatter, and one whose block never closes, both
// keep their whole text as the body: a broken block must still yield prose
// rather than refuse.
//
// Neither is reachable through Reconcile -- a page has to be read for its
// sources before a case is raised over it, and neither shape yields one -- so
// they are asked of the function itself.
func TestBodyOfKeepsAPageItCannotSplit(t *testing.T) {
	// The third one carries a thematic break: a page without frontmatter is
	// not split at the first `---` line of its prose.
	for _, text := range []string{"# A\n\nno frontmatter\n", "---\ntype: note\n\nnever closed\n",
		"# A\n\nabove\n---\nbelow\n"} {
		if got := bodyOf(text); got != text {
			t.Fatalf("bodyOf = %q, want the text whole", got)
		}
	}
}

// The body starts after the closing fence, with the blank lines behind it cut
// away -- and only those, so an indented first line keeps its indentation.
func TestBodyOfCutsTheFrontmatter(t *testing.T) {
	got := bodyOf("---\ntype: note\n---\n\n\n    indented\n")
	if got != "    indented\n" {
		t.Fatalf("bodyOf = %q", got)
	}
}

// A piece whose last line carries no newline ends the walk: `_hunks` never
// hands one in, because every diff line it renders is terminated, but the
// split has to answer for what it is given.
func TestSplitAtHunkHeadsEndsOnAnUnterminatedLine(t *testing.T) {
	got := splitAtHunkHeads("@@ -1 +1 @@\n-a\n@@ -9 +9 @@\n+b")
	want := []string{"@@ -1 +1 @@\n-a\n", "@@ -9 +9 @@\n+b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("splitAtHunkHeads = %q, want %q", got, want)
	}
}

// The split answers what `re.split(r"^(?=@@ )", text, flags=re.MULTILINE)`
// with `if part` answers: an empty line does not end the walk, and nothing at
// all comes out of nothing. `_hunks` hands in neither -- every body line it
// renders carries a prefix, and a group always brings its `@@` line -- but the
// function stands for the regular expression, not for its one caller.
func TestSplitAtHunkHeadsAsTheRegexpSplits(t *testing.T) {
	if got := splitAtHunkHeads(""); len(got) != 0 {
		t.Fatalf("splitAtHunkHeads(\"\") = %q, want nothing", got)
	}
	got := splitAtHunkHeads("@@ -1 +1 @@\n\n@@ -9 +9 @@\n+b\n")
	want := []string{"@@ -1 +1 @@\n\n", "@@ -9 +9 @@\n+b\n"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("splitAtHunkHeads = %q, want %q", got, want)
	}
}

// A place for the file that cannot be made is carried out, not written around.
func TestWriteIfChangedCarriesAFailedDirectory(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "a file")
	if err := os.WriteFile(blocked, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := writeIfChanged(filepath.Join(blocked, "under it", "x.md"), "text"); err == nil {
		t.Fatal("writeIfChanged wrote under a regular file")
	}
}

// The seam of landCase is asked one more question: a page read that works
// answers the package it was asked for, so the failures above are about the
// failure and not about the seam being in the wrong place.
func TestLandCaseWritesThePackageItBuilt(t *testing.T) {
	l := newLanding(t)
	directory := l.standing(t)
	text, err := os.ReadFile(filepath.Join(directory, packageName))
	if err != nil {
		t.Fatalf("ReadFile package.md: %v", err)
	}
	if !strings.Contains(string(text), "+func B() {}") {
		t.Fatalf("package.md carries no diff:\n%s", text)
	}
}

// The page reaches the package the way Python's text-mode read hands it over:
// the frontmatter cut away, one `W` segment per blank-line-separated
// paragraph, and a lone carriage return folded to a line break.
//
// That last fold is the one place a `W` segment is folded further than a `D`
// segment. A diff is drawn against a content hash and has to leave the lone
// `\r` standing; the page is only ever read as prose here, and universal
// newlines is what `open()` in text mode does.
func TestThePackageFoldsThePageTheWayPythonReadsIt(t *testing.T) {
	l := newLanding(t)
	page := filepath.Join(l.area.WikiPath, "a.md")
	if err := os.WriteFile(page, []byte("---\ntype: note\n---\n\nalpha\rbeta\n\nsecond\n"), 0o644); err != nil {
		t.Fatalf("WriteFile page: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(l.standing(t), packageName))
	if err != nil {
		t.Fatalf("ReadFile package.md: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "\r") {
		t.Fatalf("a carriage return reached the package:\n%q", text)
	}
	if !strings.Contains(text, "alpha\nbeta") {
		t.Fatalf("the lone carriage return was not folded:\n%s", text)
	}
	// The frontmatter is no paragraph of the page: it repeats what the `Q`
	// segments carry, and leaving it in would let a proposal satisfy the
	// evidence binding by quoting a doc id instead of prose.
	if strings.Contains(text, "type: note") {
		t.Fatalf("the frontmatter reached the package:\n%s", text)
	}
	if !strings.Contains(text, "## W2 — Wiki, second") {
		t.Fatalf("the second paragraph is no segment of its own:\n%s", text)
	}
}

// goldenPage is the page testdata/case-package.golden.md was rendered from,
// byte for byte: CRLF in the frontmatter, a lone carriage return inside a
// paragraph, an indented paragraph and one of nothing but whitespace. Those
// four are what bodyOf, pageText and the paragraph split decide between them.
const goldenPage = "---\r\ntype: note\r\nsources:\r\n  - doc_id: d1\r\n---\r\n\r\n" +
	"  eingerueckt, mit einem einsamen \r darin\n\n   \t  \n\nzweiter Absatz\n"

// goldenSources are the sources of the same golden, handed over **out of**
// doc-id order: segmentsOf sorts, and a port that took the slice as it came
// would number D and Q the other way round.
func goldenSources() []Changed {
	baseline := "eins\n"
	return []Changed{
		{DocID: "d2", Relative: "src/zwei.go", Revision: 2, ContentHash: "sha256:bb",
			Text: "zwei\n", Baseline: &baseline},
		{DocID: "d1", Relative: "src/ä eins.go", Revision: 1, ContentHash: "sha256:aa",
			Text: "neu\n", Baseline: nil},
	}
}

// goldenCase is the case both goldens were rendered from. Every optional field
// is set, and note carries the five characters `_quote` exists for: a line
// break, a quotation mark, a backslash, a tab and DEL.
func goldenCase() Case {
	return Case{
		ID: "a-2026-09-20-abcd", Area: "project/a",
		Target: "docs/wiki/a.md", TargetHash: "sha256:cc",
		State: "source_changed", Trigger: "source_change", Weight: "change",
		Created: time.Date(2026, 9, 20, 8, 0, 0, 0, time.FixedZone("", 2*3600)),
		Sources: []SourceState{
			{DocID: "d1", Revision: 1, ContentHash: "sha256:aa"},
			{DocID: "d2", Revision: 2, ContentHash: "sha256:bb"},
		},
		Note:               "zwei\nzeilen, ein \"Zitat\", ein \\ und ein \t sowie \u007f",
		SupersededProposal: "superseded-proposal.md",
		Manual:             true,
		LocalOnly:          true,
		PromptVersion:      "propose/7",
	}
}

// The whole package of a source case at once, against a file `_segments` and
// `render_package` of the reference wrote together.
//
// This is the golden the hunks one cannot give: it pins the parts between the
// diff and the rendering -- the order the segments are numbered in, the
// frontmatter cut away, the paragraph split, the strip on each paragraph and
// the fold of a lone carriage return that a `D` segment does not get.
func TestTheSourcePackageMatchesThePythonPackage(t *testing.T) {
	page := filepath.Join(t.TempDir(), "a.md")
	if err := os.WriteFile(page, []byte(goldenPage), 0o644); err != nil {
		t.Fatalf("WriteFile page: %v", err)
	}
	segments, err := segmentsOf(goldenSources(), page, nil)
	if err != nil {
		t.Fatalf("segmentsOf: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "case-package.golden.md"))
	if err != nil {
		t.Fatalf("ReadFile golden: %v", err)
	}
	if got := RenderPackage(goldenCase(), segments); got != string(want) {
		t.Fatalf("the package differs from the Python rendering:\ngot:\n%q\nwant:\n%q", got, string(want))
	}
}

// The case file itself, against one `write_case` wrote. The bytes are an
// interface in both directions: a case written here is read by the Python
// side and the other way round, and a file that differed in one character
// would be rewritten, and reported as changed, on every pass across the two.
func TestTheCaseFileMatchesThePythonCase(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "case.golden.toml"))
	if err != nil {
		t.Fatalf("ReadFile golden: %v", err)
	}
	if got := renderCase(goldenCase()); got != string(want) {
		t.Fatalf("the case differs from the Python rendering:\ngot:\n%q\nwant:\n%q", got, string(want))
	}
}

// A value the resolver cannot answer for joins the containment refusal, and
// this is the case that reaches it: an area registered with a drive-relative
// path. `C:rel` names the current directory of drive C, so where it lands
// depends on a process-wide state nobody here set -- guard refuses the
// spelling outright, and a review centre under it can be shown to stay inside
// nothing.
//
// Windows only: outside it, `C:rel` is an ordinary relative name with a colon
// in it and there is no drive-relative spelling to refuse.
func TestDeclaredReviewRefusesAnAreaItCannotResolve(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("no drive-relative paths on this platform")
	}
	_, err := declaredReview(
		config.Area{Scope: "project/a", Path: "C:rel"},
		&config.Manifest{LayoutReview: "95 Prüfzentrum"})
	if err == nil || !strings.Contains(err.Error(), "must stay inside the area") {
		t.Fatalf("err = %v, want the containment refusal", err)
	}
}
