package importcases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

// writtenManifest is what `brain init` writes, key for key.
const writtenManifest = "[area]\nscope = \"project/new\"\nwiki = true\n\n[maintenance]\non_merge = true\nmerge_branch = \"master\"\n"

var verbatimMapping = Mapping{Manifests: verbatim, Keys: []Rule{{From: "merge_branch", To: "branch"}}}

func putFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestTranslateWorldMovesAManifestVerbatimAndRenamesItsKeys(t *testing.T) {
	dir := t.TempDir()
	putFile(t, filepath.Join(dir, ".ultra-brain", "config.toml"), writtenManifest+"  merge_branch  = \"x\"\n# merge_branch = \"y\"\n")
	if err := translateWorld(dir, verbatimMapping); err != nil {
		t.Fatal(err)
	}
	// The bytes stay, quotes, blank lines and all; only the key's name moves,
	// and only where it opens a line as a key.
	want := strings.Replace(writtenManifest, "merge_branch", "branch", 1) + "  branch  = \"x\"\n# merge_branch = \"y\"\n"
	if got := readFile(t, filepath.Join(dir, ".loomux", "config.toml")); got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := entries(t, dir); strings.Join(got, " ") != ".loomux" {
		t.Fatalf("leftovers: %v", got)
	}
}

func TestTranslateWorldFoldsAVerbatimManifestThatHasCompany(t *testing.T) {
	dir := t.TempDir()
	buildOldWorld(t, dir, manifestWithLayout)
	if err := translateWorld(dir, verbatimMapping); err != nil {
		t.Fatal(err)
	}
	// Folded and encoded: the policy of the old ultraloom file is in it.
	if _, ok := decodeConfig(t, dir)["policy"]; !ok {
		t.Fatalf("not folded: %s", readFile(t, filepath.Join(dir, ".loomux", "config.toml")))
	}
}

func TestTranslateWorldLeavesAVerbatimWorldWithoutAManifestAlone(t *testing.T) {
	dir := t.TempDir()
	putFile(t, filepath.Join(dir, "notes.md"), "x")
	if err := translateWorld(dir, verbatimMapping); err != nil {
		t.Fatal(err)
	}
	if got := entries(t, dir); strings.Join(got, " ") != "notes.md" {
		t.Fatalf("got %v", got)
	}
}

func TestTranslateWorldReportsAVerbatimManifestItCannotMove(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, ".brain.toml")
	putFile(t, manifest, writtenManifest)
	testlock.Lock(t, manifest)
	if err := translateWorld(dir, verbatimMapping); err == nil || !strings.Contains(err.Error(), manifest) {
		t.Fatalf("an unreadable manifest: %v", err)
	}

	dir = t.TempDir()
	putFile(t, filepath.Join(dir, ".brain.toml"), writtenManifest)
	putFile(t, filepath.Join(dir, ".loomux"), "a file where the directory belongs")
	if err := translateWorld(dir, verbatimMapping); err == nil {
		t.Fatal("want error for a .loomux blocked by a file")
	}

	dir = t.TempDir()
	putFile(t, filepath.Join(dir, ".brain.toml"), writtenManifest)
	if err := os.MkdirAll(filepath.Join(dir, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := translateWorld(dir, verbatimMapping); err == nil {
		t.Fatal("want error for a config blocked by a directory")
	}
}

func TestImportRefusesManifestRulesItCannotCarryOut(t *testing.T) {
	for _, m := range []Mapping{
		{Manifests: "folded"},
		{Keys: []Rule{{From: "a", To: "b"}}},
	} {
		// The recordings are never read: a bad mapping is refused first.
		if err := Import(filepath.Join(t.TempDir(), "missing"), t.TempDir(), m); err == nil ||
			!strings.Contains(err.Error(), "manifest") {
			t.Errorf("%+v: %v", m, err)
		}
	}
}

func TestImportMovesTheManifestsOfAVerbatimStage(t *testing.T) {
	from, to := t.TempDir(), t.TempDir()
	dir := buildCase(t, from, "area-add", "new", "brain-mcp init -y", "")
	putFile(t, filepath.Join(dir, "world_after", ".ultra-brain", "config.toml"), writtenManifest)
	m := verbatimMapping
	m.Commands = []Rule{{From: "brain-mcp init ", To: "loomux area add "}}
	if err := Import(from, to, m); err != nil {
		t.Fatal(err)
	}
	got := readFile(t, filepath.Join(to, "area-add", "new", "world_after", ".loomux", "config.toml"))
	if !strings.Contains(got, "\nbranch = \"master\"\n") {
		t.Fatalf("got %q", got)
	}
}
