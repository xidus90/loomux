package namecheck

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
)

func TestReferencesFindsEveryNameOfTheList(t *testing.T) {
	files := map[string]string{
		"a.go":  "x := \"UltraLoom\"\n",
		"b.md":  "see brain guard and .Brain.toml\n",
		"c.txt": "nothing here\n",
	}
	got, err := References(keys(files), reader(files), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.go:1: x := \"UltraLoom\"", "b.md:1: see brain guard and .Brain.toml"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReferencesNamesEachPredecessorOnItsOwn(t *testing.T) {
	for _, name := range []string{
		"ultraloom", "ultra-brain", "ulguard", "ulinit", "ulflow", "brain-mcp",
		"brain guard", "ultraloomowned", ".brain.toml", ".ultra-brain",
		"specs-ul/", "specs-ub/", "plans-ul/", "plans-ub/", "bench-ub/",
	} {
		files := map[string]string{"a.md": "x " + strings.ToUpper(name) + " y\n"}
		got, _ := References(keys(files), reader(files), nil)
		if len(got) != 1 {
			t.Errorf("%q not found: %q", name, got)
		}
	}
	clean := map[string]string{"a.md": "brain search and ultra\nloom, brain.toml\n"}
	if got, _ := References(keys(clean), reader(clean), nil); len(got) != 0 {
		t.Errorf("near misses found: %q", got)
	}
}

func TestReferencesAreSortedAcrossFilesAndLeaveCarriageReturnsOut(t *testing.T) {
	files := map[string]string{"b.md": "ulguard\r\n", "a.md": "ok\nulguard\r\n"}
	got, _ := References([]string{"b.md", "a.md"}, reader(files), nil)
	want := []string{"a.md:2: ulguard", "b.md:1: ulguard"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAPathExceptionCoversOnlyItsFileOrFolder(t *testing.T) {
	files := map[string]string{
		"docs/en/benchmarks.md":           "ulguard 12 ms\n",
		"docs/en/benchmarks-notes.md":     "ulguard\n",
		"docs/.superpowers/specs-ul/x.md": "ultraloom\n",
		"docs/.superpowers-x/y.md":        "ultraloom\n",
	}
	ex := []Exception{
		{Path: "docs/en/benchmarks.md", Owner: "history"},
		{Path: "docs/.superpowers/specs-ul/", Owner: "flow"},
	}
	got, _ := References(keys(files), reader(files), ex)
	want := []string{"docs/.superpowers-x/y.md:1: ultraloom", "docs/en/benchmarks-notes.md:1: ulguard"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestALineExceptionCoversOnlyMatchingLines(t *testing.T) {
	files := map[string]string{"README.md": "| Flow | ulflow follow-up |\nrun ulflow now\n"}
	ex := []Exception{{Path: "README.md", Line: regexp.MustCompile(`^\| Flow \|`), Owner: "flow"}}
	got, _ := References(keys(files), reader(files), ex)
	if want := []string{"README.md:2: run ulflow now"}; !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestALineExceptionOnAFolderCoversOnlyMatchingLinesBelowIt(t *testing.T) {
	files := map[string]string{
		"docs/wiki/a.md": "sources: [docs/.superpowers/specs-ul/x.md]\nsee ultraloom now\n",
		"docs/other.md":  "docs/.superpowers/specs-ul/x.md\n",
	}
	ex := []Exception{{Path: "docs/wiki/", Line: regexp.MustCompile(`specs-ul/`), Owner: "working-papers"}}
	got, _ := References(keys(files), reader(files), ex)
	want := []string{"docs/other.md:1: docs/.superpowers/specs-ul/x.md", "docs/wiki/a.md:2: see ultraloom now"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestASectionExceptionAllowsOneHitPerNamedEntry(t *testing.T) {
	changelog := "# Changelog\n\n## [7.2.0] - 2026-10-05\n- uses ultraloom\n\n" +
		"## [7.0.1-beta] - 2026-10-04\n- reads .brain.toml\n- and .ultra-brain\n\n" +
		"## [2.3.0] - 2026-09-22\n- flags ulguard\n\n" +
		"## [4.2.2-beta.2] - 2026-09-20\n- names ulflow\n"
	files := map[string]string{"CHANGELOG.md": changelog}
	ex := []Exception{
		{Path: "CHANGELOG.md", Section: "7.0.1", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "2.3.0", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "4.2.2", Owner: "history"},
	}
	got, _ := References(keys(files), reader(files), ex)
	want := []string{"CHANGELOG.md:4: - uses ultraloom", "CHANGELOG.md:8: - and .ultra-brain"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReferencesPassesAReadErrorOn(t *testing.T) {
	broken := func(string) ([]byte, error) { return nil, errors.New("gone") }
	if _, err := References([]string{"a"}, broken, nil); err == nil || !strings.Contains(err.Error(), "a: gone") {
		t.Fatalf("err = %v", err)
	}
}

// workTree reads files below root; a file the work tree no longer has (the
// index still lists it after a plain rm) adds no reference.
func workTree(root string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) {
		data, err := os.ReadFile(filepath.Join(root, p))
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return data, err
	}
}

func TestTheWorkTreeReaderSkipsAFileThatIsGone(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "here.md"), []byte("ulguard"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	read := workTree(root)
	if data, err := read("here.md"); err != nil || string(data) != "ulguard" {
		t.Fatalf("here.md: %q, %v", data, err)
	}
	if data, err := read("gone.md"); err != nil || data != nil {
		t.Fatalf("gone.md: %q, %v", data, err)
	}
	if _, err := read("dir"); err == nil {
		t.Fatal("a directory read as a file must stay an error")
	}
}

func TestGraftIsANameExceptAfterAnUnderscore(t *testing.T) {
	files := map[string]string{
		"a.go": "// Ported from trailhq/Graft\n",
		"b.go": "\t\"GIT_GRAFT_FILE\",\n",
		"c.go": "func graftEdges() {}\n",
		"d.go": "func TestWithGraftsDefaults(t *testing.T) {}\n",
		"e.md": "Graft's crux\n",
	}
	got, _ := References(keys(files), reader(files), nil)
	want := []string{
		"a.go:1: // Ported from trailhq/Graft",
		"c.go:1: func graftEdges() {}",
		"d.go:1: func TestWithGraftsDefaults(t *testing.T) {}",
		"e.md:1: Graft's crux",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTheIdeaRuleTakesTheIdeaSentenceOnly(t *testing.T) {
	files := map[string]string{
		"README.md":               "Ranking is inspired by [trailhq/Graft](u).\n| model | Graft |\n",
		"README.de.md":            "Ranking ist angeregt von [trailhq/Graft](u).\nGrafts Crux\n",
		"docs/en/architecture.md": "principles inspired by [trailhq/Graft](u):\n",
		"docs/de/architecture.md": "angeregt von [trailhq/Graft](u):\n",
		"docs/en/guide.md":        "inspired by [trailhq/Graft](u)\n",
	}
	got, _ := References(keys(files), reader(files), exceptions())
	want := []string{
		"README.de.md:2: Grafts Crux",
		"README.md:2: | model | Graft |",
		"docs/en/guide.md:1: inspired by [trailhq/Graft](u)",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNoExceptionIsLeftForPapersWikiPlanOrRoadmap(t *testing.T) {
	files := map[string]string{
		"docs/.superpowers/specs/a.md":                   "ultraloom\n",
		"docs/.superpowers/specs-ub/a.md":                "ultraloom\n",
		"docs/wiki/log.md":                               "ultraloom\n",
		"docs/wiki/a.md":                                 "docs/.superpowers/specs-ub/x.md\n",
		"_identities.tsv":                                "x\tdocs/.superpowers/bench-ub/y.md\n",
		"internal/brain/apply/testdata/frontmatter/w.md": "specs-ub/x.md\n",
		"docs/en/migration.md":                           "ultraloom\n",
		"README.md":                                      "| **Web** | ultra-brain/web | W1 |\n",
	}
	got, _ := References(keys(files), reader(files), exceptions())
	if len(got) != len(files) {
		t.Fatalf("got %d findings, want one per file: %q", len(got), got)
	}
}

func TestTheNoticeStandsWholeAndTheGeneratorFileToo(t *testing.T) {
	files := map[string]string{
		"internal/notices/NOTICE.md":     "from trailhq/Graft\nGraft's license\n",
		"internal/dev/notices/ported.md": "trailhq/Graft <u>\n",
		"internal/dev/notices/other.md":  "trailhq/Graft\n",
	}
	got, _ := References(keys(files), reader(files), exceptions())
	want := []string{"internal/dev/notices/other.md:1: trailhq/Graft"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// The index of the real repository, not the one a commit hook hands in (see
// gitenv): a new file is scanned from the next commit on.
func TestTheRepositoryNamesNoPredecessor(t *testing.T) {
	root := filepath.Join("..", "..")
	list := exec.Command("git", "-C", root, "ls-files", "-z")
	list.Env = gitenv.Environ()
	out, err := list.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	files := strings.Split(strings.TrimRight(string(out), "\x00"), "\x00")
	found, err := References(files, workTree(root), exceptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("names of the predecessor tools or of the ported original outside the exceptions in references.go:\n%s", strings.Join(found, "\n"))
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func reader(m map[string]string) func(string) ([]byte, error) {
	return func(p string) ([]byte, error) { return []byte(m[p]), nil }
}
