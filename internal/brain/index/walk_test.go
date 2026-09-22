package index

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

func TestOwnWikiPrefix(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	wikiInside := filepath.Join(root, "docs", "wiki")
	wikiOutside := filepath.Join(tmp, "vault", "wiki")
	if err := os.MkdirAll(wikiInside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wikiOutside, 0o755); err != nil {
		t.Fatal(err)
	}

	// Case 1: No wiki path declared
	aNoWiki := config.Area{Path: root, WikiPath: ""}
	if p := OwnWikiPrefix(aNoWiki); p != nil {
		t.Errorf("expected nil prefix for empty wiki, got %v", *p)
	}

	// Case 2: Wiki outside repo root (vault)
	aOutside := config.Area{Path: root, WikiPath: wikiOutside}
	if p := OwnWikiPrefix(aOutside); p != nil {
		t.Errorf("expected nil prefix for outside wiki, got %v", *p)
	}

	// Case 3: Wiki and root coincide
	aSame := config.Area{Path: root, WikiPath: root}
	pSame := OwnWikiPrefix(aSame)
	if pSame == nil || *pSame != "" {
		t.Errorf("expected empty string prefix when wiki equals root, got %v", pSame)
	}

	// Case 4: Wiki inside repo
	aInside := config.Area{Path: root, WikiPath: wikiInside}
	pInside := OwnWikiPrefix(aInside)
	if pInside == nil || *pInside != "docs/wiki" {
		t.Errorf("expected 'docs/wiki', got %v", pInside)
	}
}

func TestInOwnWiki(t *testing.T) {
	// Prefix is nil
	if InOwnWiki("docs/wiki/note.md", nil) {
		t.Error("expected false when prefix is nil")
	}

	// Prefix is "" (coincides with root)
	empty := ""
	if !InOwnWiki("note.md", &empty) {
		t.Error("expected true when prefix is empty string")
	}

	// Prefix is "docs/wiki"
	prefix := "docs/wiki"
	if !InOwnWiki("docs/wiki", &prefix) {
		t.Error("expected true for exact prefix match")
	}
	if !InOwnWiki("docs/wiki/note.md", &prefix) {
		t.Error("expected true for child path")
	}
	if InOwnWiki("docs/other.md", &prefix) {
		t.Error("expected false for sibling directory")
	}
	if InOwnWiki("docs/wikinote.md", &prefix) {
		t.Error("expected false for prefix substring without slash boundary")
	}
}

func TestArtifactExcludes(t *testing.T) {
	// Read-only area excludes no artifacts
	roArea := config.Area{ReadOnly: true}
	if exc := ArtifactExcludes(roArea, false); len(exc) != 0 {
		t.Errorf("expected no artifact excludes for read-only area, got %v", exc)
	}
	if exc := ArtifactExcludes(roArea, true); len(exc) != 0 {
		t.Errorf("expected no artifact excludes for read-only area in wiki, got %v", exc)
	}

	// Writable area
	wrArea := config.Area{ReadOnly: false}
	outsideWiki := ArtifactExcludes(wrArea, false)
	sort.Strings(outsideWiki)
	expectedOutside := append([]string{}, OwnArtifacts...)
	sort.Strings(expectedOutside)
	if !reflect.DeepEqual(outsideWiki, expectedOutside) {
		t.Errorf("expected outside wiki to match OwnArtifacts, got %v vs %v", outsideWiki, expectedOutside)
	}

	insideWiki := ArtifactExcludes(wrArea, true)
	sort.Strings(insideWiki)
	expectedInside := append([]string{}, BundleArtifacts...)
	sort.Strings(expectedInside)
	if !reflect.DeepEqual(insideWiki, expectedInside) {
		t.Errorf("expected inside wiki to match BundleArtifacts, got %v vs %v", insideWiki, expectedInside)
	}
}

func TestFindFiles(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "project")
	wikiDir := filepath.Join(repo, "wiki")
	nestedArea := filepath.Join(repo, "subproject")

	filesToCreate := []string{
		"root.md",
		"README.md",
		"ignore.txt",                // excluded by non-.md include
		".git/config",               // ALWAYS_EXCLUDES
		".obsidian/app.json",        // ALWAYS_EXCLUDES
		".claude/settings.json",     // ALWAYS_EXCLUDES
		".tools/bin.md",             // ALWAYS_EXCLUDES
		".superpowers/sdd/run.md",   // ALWAYS_EXCLUDES
		"node_modules/mod/index.md", // ALWAYS_EXCLUDES
		"tests/fixtures/sample.md",  // ALWAYS_EXCLUDES
		".brain.toml",               // ALWAYS_EXCLUDES
		".ultra-brain/config.toml",  // ALWAYS_EXCLUDES
		"index.md",                  // OWN_ARTIFACTS (outside wiki)
		"index.intro.md",            // OWN_ARTIFACTS
		"graph.json",                // OWN_ARTIFACTS
		"_identities.tsv",           // OWN_ARTIFACTS
		"wiki/index.md",             // BUNDLE_ARTIFACTS preserves index.md in wiki!
		"wiki/index.intro.md",       // BUNDLE_ARTIFACTS excludes intro
		"wiki/topic.md",             // Valid wiki file
		"custom_exclude/note.md",    // custom manifest exclude
		"review/pkg.md",             // manifest review layout exclude
		"private/secret.md",         // privacy never glob
		"PRIVATE/capital.md",        // privacy never glob (case-insensitive)
		"subproject/nested.md",      // nested registered area
	}

	for _, f := range filesToCreate {
		full := filepath.Join(repo, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	area := config.Area{
		Path:     repo,
		WikiPath: wikiDir,
		ReadOnly: false,
	}

	manifest := &config.Manifest{
		LayoutReview: "review",
		IndexInclude: []string{"**/*.md"},
		IndexExclude: []string{"custom_exclude/**"},
		NeverGlobs:   []string{"private/**"},
	}

	nested := []string{nestedArea}

	found, err := FindFiles(area, manifest, nested)
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}

	expected := []string{
		"README.md",
		"root.md",
		"wiki/index.md",
		"wiki/topic.md",
	}

	var foundRel []string
	for _, p := range found {
		rel, err := filepath.Rel(repo, p)
		if err != nil {
			t.Fatal(err)
		}
		foundRel = append(foundRel, filepath.ToSlash(rel))
	}

	if !reflect.DeepEqual(foundRel, expected) {
		t.Fatalf("found files mismatch:\ngot:  %v\nwant: %v", foundRel, expected)
	}
}

func TestFindFilesReadOnlyAreaPreservesArtifacts(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "foreign_wiki")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "index.md"), []byte("foreign index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "index.intro.md"), []byte("foreign intro"), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{
		Path:     repo,
		ReadOnly: true,
	}
	manifest := &config.Manifest{
		IndexInclude: []string{"**/*.md"},
	}

	found, err := FindFiles(area, manifest, nil)
	if err != nil {
		t.Fatalf("FindFiles failed: %v", err)
	}
	if len(found) != 2 {
		t.Errorf("expected both index.md and index.intro.md in read-only area, got %d files", len(found))
	}
}

func TestFindFilesInvalidReviewLayout(t *testing.T) {
	tmp := t.TempDir()
	area := config.Area{Path: tmp}
	manifest := &config.Manifest{
		LayoutReview: ".",
	}
	_, err := FindFiles(area, manifest, nil)
	if err == nil {
		t.Fatal("expected error when layout.review resolves to root")
	}
}

func TestFindFilesDefaultManifest(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "note.md")
	if err := os.WriteFile(f, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	area := config.Area{Path: tmp}
	// manifest is nil, so default includes is ["**/*.md"]
	found, err := FindFiles(area, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("expected 1 file with default manifest, got %d", len(found))
	}
}

func TestFindFilesNonexistentDir(t *testing.T) {
	area := config.Area{Path: "/nonexistent/area/path"}
	_, err := FindFiles(area, nil, nil)
	if err == nil {
		t.Fatal("expected error when walking nonexistent area directory")
	}
}

func TestOwnWikiPrefixCrossDrive(t *testing.T) {
	area := config.Area{
		Path:     "C:\\repo",
		WikiPath: "Z:\\vault",
	}
	if p := OwnWikiPrefix(area); p != nil {
		t.Errorf("expected nil for cross drive, got %v", *p)
	}
}

func TestGlobMatchingPatterns(t *testing.T) {
	if !matchesAnyGlob("any/path/at/all", []string{"**"}) {
		t.Error("expected ** to match any path")
	}
	if !matchesAnyGlob("doc1.md", []string{"doc?.md"}) {
		t.Error("expected ? to match single character")
	}
	if matchesAnyGlob("doc12.md", []string{"doc?.md"}) {
		t.Error("expected ? not to match two characters")
	}
}

// relativeFound runs FindFiles over files written below a fresh area root and
// answers what it found, relative to that root.
func relativeFound(t *testing.T, manifest *config.Manifest, files ...string) []string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		full := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("content"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	found, err := FindFiles(config.Area{Path: root}, manifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	relatives := []string{}
	for _, p := range found {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			t.Fatal(err)
		}
		relatives = append(relatives, filepath.ToSlash(rel))
	}
	return relatives
}

// The review centre of `knowledge` on this machine is `95 Prüfzentrum`, and
// its packages carry source diffs. `review_excludes` keeps it out of the walk
// in the reference; a translator that took the pattern byte by byte read the
// `ü` as two characters and walked the centre.
func TestFindFilesExcludesAReviewCentreWithANonASCIIName(t *testing.T) {
	manifest := &config.Manifest{LayoutReview: "95 Prüfzentrum", IndexInclude: []string{"**/*.md"}}
	got := relativeFound(t, manifest, "note.md", "95 Prüfzentrum/knowledge/c1/package.md")
	if want := []string{"note.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("found %v, want %v", got, want)
	}
}

func TestFindFilesExcludesANonASCIILiteral(t *testing.T) {
	manifest := &config.Manifest{IndexInclude: []string{"**/*.md"}, IndexExclude: []string{"Entwürfe/**"}}
	got := relativeFound(t, manifest, "note.md", "Entwürfe/draft.md")
	if want := []string{"note.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("found %v, want %v", got, want)
	}
}

// Measured: `PurePosixPath("docs/a.md").full_match("docs/**/*.md")` is True.
// A `**` that is not the last part stands for zero segments too.
func TestFindFilesIncludesAFileDirectlyBelowATwoStarInclude(t *testing.T) {
	manifest := &config.Manifest{IndexInclude: []string{"docs/**/*.md"}}
	got := relativeFound(t, manifest, "docs/a.md", "docs/sub/b.md", "top.md")
	if want := []string{"docs/a.md", "docs/sub/b.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("found %v, want %v", got, want)
	}
}

// Measured: `PurePosixPath("a/x.md").full_match("a/**/x.md")` is True.
func TestFindFilesExcludesAFileDirectlyBelowATwoStarExclude(t *testing.T) {
	manifest := &config.Manifest{IndexInclude: []string{"**/*.md"}, IndexExclude: []string{"a/**/x.md"}}
	got := relativeFound(t, manifest, "a/x.md", "a/b/x.md", "a/y.md")
	if want := []string{"a/y.md"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("found %v, want %v", got, want)
	}
}

// junction makes a directory link the way a person would, with `mklink /J`,
// which needs no privilege. Off Windows the case it stands for does not exist.
func junction(t *testing.T, link, target string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("a junction is the Windows link this case is about")
	}
	command := exec.Command("cmd", "/c", "mklink", "/J", link, target)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("mklink /J %s %s: %v\n%s", link, target, err, out)
	}
}

// An area whose root is itself a junction: Go reports the junction as
// irregular and WalkDir does not descend, so the walk found nothing and said
// nothing -- and a reindex would have written an empty register. `rglob`
// follows the root. The paths come back under the registered root, because
// every caller measures them against area.Path.
func TestFindFilesWalksAnAreaRootThatIsAJunction(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	if err := os.MkdirAll(filepath.Join(target, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.md", "sub/b.md"} {
		if err := os.WriteFile(filepath.Join(target, filepath.FromSlash(name)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(tmp, "link")
	junction(t, link, target)

	found, err := FindFiles(config.Area{Path: link}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(link, "a.md"), filepath.Join(link, "sub", "b.md")}
	if !reflect.DeepEqual(found, want) {
		t.Fatalf("found %v, want %v", found, want)
	}
}

// A root spelt with a drive and no separator names a directory relative to
// that drive's working directory. The resolver refuses it rather than walk
// whatever that happens to be.
func TestFindFilesRefusesADriveRelativeRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("a drive-relative path is a Windows spelling")
	}
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "rel"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "rel", "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)
	root := filepath.VolumeName(tmp) + "rel"
	found, err := FindFiles(config.Area{Path: root}, nil, nil)
	if err == nil {
		t.Fatalf("expected a refusal of %s, found %v", root, found)
	}
	if !strings.Contains(err.Error(), root) {
		t.Errorf("the error should name the root, got %v", err)
	}
}
