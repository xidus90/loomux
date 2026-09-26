package query

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/testlock"
)

func TestBuildAnnouncesOnlyARecordItCouldNotWrite(t *testing.T) {
	var heard []string
	listen := func(s string) { heard = append(heard, s) }

	root := repo(t, sample())
	if _, _, err := Build(root, listen); err != nil {
		t.Fatal(err)
	}
	if len(heard) != 0 {
		t.Fatalf("a build that wrote everything announced %q", heard)
	}

	// A directory in the fingerprint's place leaves the graph and the sidecar
	// alone and fails only the record: the build stands and says why.
	root = repo(t, sample())
	if err := os.MkdirAll(freshness.Path(root), 0o755); err != nil {
		t.Fatal(err)
	}
	g, _, err := Build(root, listen)
	if err != nil || g == nil {
		t.Fatalf("Build = %v, %v; want the graph and no error", g, err)
	}
	if len(heard) != 1 || !strings.Contains(heard[0], "freshness record not written") {
		t.Fatalf("notices = %q, want the one unwritten record", heard)
	}
}

// ignore is a build notice nobody reads.
func ignore(string) {}

// repo writes a small Go repository and returns its root.
func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func sample() map[string]string {
	return map[string]string{
		"go.mod":          "module example.com/repo\n",
		"main.go":         "package main\n\nimport \"example.com/repo/lib\"\n\nfunc main() { lib.Run() }\n",
		"lib/lib.go":      "package lib\n\n// Run does the thing.\nfunc Run() {}\n",
		"lib/lib_test.go": "package lib\n\nimport \"testing\"\n\nfunc TestRun(t *testing.T) { Run() }\n",
	}
}

func TestExtractFailsWhenTheRootDoesNotExist(t *testing.T) {
	if _, _, err := Extract(filepath.Join(t.TempDir(), "gone"), ExtractOptions{}); err == nil {
		t.Fatal("want an error for a missing root")
	}
}

func TestExtractFailsWhenASourceFileCannotBeRead(t *testing.T) {
	root := repo(t, sample())
	testlock.Lock(t, filepath.Join(root, "main.go"))
	if _, _, err := Extract(root, ExtractOptions{}); err == nil {
		t.Fatal("want an error for an unreadable source file")
	}
}

func TestExtractFailsWhenGoModCannotBeRead(t *testing.T) {
	root := repo(t, sample())
	testlock.Lock(t, filepath.Join(root, "go.mod"))
	if _, _, err := Extract(root, ExtractOptions{}); err == nil {
		t.Fatal("want an error for an unreadable go.mod")
	}
}

func TestGoModPathsSkipsADirectoryItCannotRead(t *testing.T) {
	root := repo(t, sample())
	blocked := filepath.Join(root, "blocked")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(blocked, "go.mod"), []byte("module blocked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	testlock.LockDir(t, blocked)
	// The walk must not fail the whole build over one unreadable directory.
	got := goModPaths(root)
	for _, p := range got {
		if strings.HasPrefix(p, "blocked/") {
			t.Errorf("goModPaths(%q) must not see into the locked directory, got %v", root, got)
		}
	}
}

func TestGoModPathsSkipsATestdataDirectory(t *testing.T) {
	// goModPaths used to keep its own, narrower skip list (dot-directories and
	// vendor only), so a fixture go.mod under testdata/ reached module
	// resolution after sourceset had already excluded its .go files from the
	// build -- two walks of the same tree disagreeing about testdata. Both now
	// share sourceset.SkipDir.
	files := sample()
	files["testdata/fixture/go.mod"] = "module fixture\n"
	root := repo(t, files)

	got := goModPaths(root)
	for _, p := range got {
		if strings.HasPrefix(p, "testdata/") {
			t.Errorf("goModPaths(%q) must not see into testdata/, got %v", root, got)
		}
	}
	if len(got) != 1 || got[0] != "go.mod" {
		t.Errorf("goModPaths(%q) = %v, want only the repository's own go.mod", root, got)
	}
}

func TestBuildReturnsTheGraphAndWhatItLearned(t *testing.T) {
	files := sample()
	files["doc.go"] = "package main\n"
	root := repo(t, files)

	g, stats, err := Build(root, ignore)
	if err != nil {
		t.Fatal(err)
	}
	if g == nil || len(g.Nodes) == 0 {
		t.Fatalf("Build must hand back the graph it wrote, got %v", g)
	}
	if len(stats.Files) != 4 || len(stats.Hashes) != 4 {
		t.Errorf("stats cover %d files and %d hashes, want 4 of each (go.mod is no source)", len(stats.Files), len(stats.Hashes))
	}
	// doc.go holds nothing but its file node; every other file has a symbol.
	if stats.NoSymbol != 1 {
		t.Errorf("NoSymbol = %d, want 1", stats.NoSymbol)
	}
}

func TestBuildFailsWhenTheRootDoesNotExist(t *testing.T) {
	if _, _, err := Build(filepath.Join(t.TempDir(), "gone"), ignore); err == nil {
		t.Fatal("want an error for a missing root")
	}
}

func TestExtractReportsTheReadFailureNotAParseError(t *testing.T) {
	root := repo(t, sample())
	testlock.Lock(t, filepath.Join(root, "main.go"))
	_, _, err := Extract(root, ExtractOptions{})
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("err = %v, want the read failure itself", err)
	}
}

func TestExtractRefusesAFileItCannotParse(t *testing.T) {
	root := repo(t, map[string]string{"go.mod": "module x\n", "broken.go": "package ???\n"})
	if _, _, err := Extract(root, ExtractOptions{}); err == nil || !strings.Contains(err.Error(), "broken.go") {
		t.Fatalf("err = %v, want a parse error naming broken.go", err)
	}
}

func TestExtractSkipsAFileNoLanguageClaims(t *testing.T) {
	// Unreachable while sourceset and extract/all agree on the extensions,
	// which all's own test holds; the seam stands in for the day they do not.
	claims := languageFor
	t.Cleanup(func() { languageFor = claims })
	languageFor = func(rel string) (extract.Language, bool) {
		if rel == "lib/lib_test.go" {
			return nil, false
		}
		return claims(rel)
	}

	g, stats, err := Extract(repo(t, sample()), ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range g.Nodes {
		if n.Path == "lib/lib_test.go" {
			t.Errorf("node %q of a file no language claims reached the graph", n.ID)
		}
	}
	if len(g.Nodes) == 0 {
		t.Fatal("the files a language claims must still be extracted")
	}
	// The freshness record still covers every file the walk listed, or the
	// next probe would call the skipped one added and rebuild for nothing.
	if _, ok := stats.Hashes["lib/lib_test.go"]; !ok || len(stats.Files) != 3 {
		t.Errorf("stats = %d files, hashes %v; want all three files hashed", len(stats.Files), stats.Hashes)
	}
}

func TestExtractStampsTheCombinedVersion(t *testing.T) {
	g, _, err := Extract(repo(t, sample()), ExtractOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if g.Meta.Extractor != all.Version() {
		t.Errorf("Meta.Extractor = %q, want %q", g.Meta.Extractor, all.Version())
	}
}

func TestGoModPathsOnAMissingRootIsEmpty(t *testing.T) {
	if got := goModPaths(filepath.Join(t.TempDir(), "gone")); len(got) != 0 {
		t.Errorf("goModPaths = %v, want none", got)
	}
}

func TestGoModPathsFindsANestedModule(t *testing.T) {
	files := sample()
	files["lib/go.mod"] = "module example.com/repo/lib\n"
	got := goModPaths(repo(t, files))
	if strings.Join(got, ",") != "go.mod,lib/go.mod" {
		t.Errorf("goModPaths = %v, want go.mod and lib/go.mod", got)
	}
}

func TestGoModPathsWalksARootThatWouldBeSkippedBelowIt(t *testing.T) {
	// The skip list is for directories under the root. A repository that
	// happens to be checked out as .../testdata is still a repository.
	root := filepath.Join(t.TempDir(), "testdata")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := goModPaths(root); strings.Join(got, ",") != "go.mod" {
		t.Errorf("goModPaths = %v, want the root's go.mod", got)
	}
}
