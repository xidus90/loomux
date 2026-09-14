package wiki

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(parts...)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRootPrefersTheManifestLayout(t *testing.T) {
	root := t.TempDir()
	want := mkdir(t, root, "knowledge", "pages")
	mkdir(t, root, "docs", "wiki")
	mkdir(t, root, ".loomux")
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[area]\nscope = \"project/x\"\n[layout]\nwiki = \"knowledge/pages\"\n"), 0o644)
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootFallsBackToDocsWikiThenWiki(t *testing.T) {
	root := t.TempDir()
	want := mkdir(t, root, "wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q", got)
	}
	want = mkdir(t, root, "docs", "wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q", got)
	}
}

func TestRootIsEmptyWithoutAnyWiki(t *testing.T) {
	if got := Root(t.TempDir()); got != "" {
		t.Fatalf("got %q", got)
	}
}

// writeManifest declares one [layout] wiki value for the project at root.
func writeManifest(t *testing.T, root, layout string) {
	t.Helper()
	dir := mkdir(t, root, ".loomux")
	body := "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"" + layout + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRootFallsBackWhenTheManifestLayoutDoesNotExist(t *testing.T) {
	root := t.TempDir()
	want := mkdir(t, root, "docs", "wiki")
	writeManifest(t, root, "knowledge/pages")
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A layout that leaves the repository is refused by WikiLayout even where the
// directory it names exists; the raw value would have been taken.
func TestRootIgnoresALayoutOutsideTheRepository(t *testing.T) {
	parent := t.TempDir()
	mkdir(t, parent, "outside")
	root := mkdir(t, parent, "project")
	want := mkdir(t, root, "docs", "wiki")
	writeManifest(t, root, "../outside")
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootFindsTheFamilyWikiOfASuffixedProject(t *testing.T) {
	parent := t.TempDir()
	root := mkdir(t, parent, "iam_backend")
	want := mkdir(t, parent, "iam_wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootSkipsASuffixWithoutAFamilyWiki(t *testing.T) {
	parent := t.TempDir()
	root := mkdir(t, parent, "iam_backend")
	want := mkdir(t, parent, "iam_backend-wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootFindsTheUnderscoreNeighbourWiki(t *testing.T) {
	parent := t.TempDir()
	root := mkdir(t, parent, "proj")
	want := mkdir(t, parent, "proj_wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRootFindsTheHyphenNeighbourWiki(t *testing.T) {
	parent := t.TempDir()
	root := mkdir(t, parent, "proj")
	want := mkdir(t, parent, "proj-wiki")
	if got := Root(root); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
