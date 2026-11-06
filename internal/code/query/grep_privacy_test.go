package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/testlock"
)

// privacyRepo is the fixture plus the two files a declaration hides, written
// before the build so the index knows them too.
func privacyRepo(t *testing.T, never string) string {
	t.Helper()
	files := grepFiles()
	files["secrets/key.txt"] = "zzzUniqueWord\n"
	files["docs/secret.md"] = "zzzUniqueWord\n"
	root := buildRepo(t, files)
	writeManifest(t, root, never)
	return root
}

func writeManifest(t *testing.T, root, never string) {
	t.Helper()
	dir := filepath.Join(root, ".loomux")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[area]\nscope = \"fixture\"\n\n[privacy]\nnever = [" + never + "]\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCloudHidesNeverPath(t *testing.T) {
	root := privacyRepo(t, `"secrets/**"`)

	local, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slicesHasPath(local, "secrets/key.txt") {
		t.Fatal("the local channel must see the file")
	}
	cloud, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "cloud"})
	if err != nil {
		t.Fatalf("cloud: %v", err)
	}
	if slicesHasPath(cloud, "secrets/key.txt") {
		t.Fatal("the cloud channel returned a hit from a never file")
	}
}

func TestCloudHidesNeverMarkdown(t *testing.T) {
	root := privacyRepo(t, `"docs/**"`)
	cloud, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "cloud"})
	if err != nil {
		t.Fatal(err)
	}
	if slicesHasPath(cloud, "docs/secret.md") {
		t.Fatal("a markdown file under never reached the cloud answer")
	}
	if cloud.TotalHits == 0 {
		t.Fatal("the open files must still answer")
	}
}

func TestLocalUnchanged(t *testing.T) {
	root := grepFixture(t)
	writeManifest(t, root, `"docs/**"`)
	plain, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	local, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "local"})
	if err != nil {
		t.Fatal(err)
	}
	if comparable(plain).TotalHits != comparable(local).TotalHits ||
		len(plain.Groups) != len(local.Groups) {
		t.Fatalf("local changed the answer: %+v vs %+v", plain, local)
	}
}

func TestCloudUnreadableManifestExitsOne(t *testing.T) {
	root := grepFixture(t)
	writeManifest(t, root, `"secrets/**"`)
	testlock.Lock(t, filepath.Join(root, ".loomux", "config.toml"))
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "cloud"}); err == nil {
		t.Fatal("an unreadable manifest must fail the cloud channel")
	}
}

func TestCloudPrefixUnderNeverEmpty(t *testing.T) {
	root := privacyRepo(t, `"secrets/**"`)
	res, notes, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "cloud", In: "secrets"})
	if err != nil {
		t.Fatalf("a hidden prefix must not be an error: %v", err)
	}
	if res.TotalHits != 0 {
		t.Fatalf("hits = %d, want none", res.TotalHits)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "readable") {
		t.Fatalf("notes = %q, want the hidden prefix note", notes)
	}
}

func TestGlobFilters(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Glob: "**/*.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Groups) == 0 {
		t.Fatal("the glob matched nothing")
	}
	for _, grp := range res.Groups {
		if filepath.Ext(grp.Path) != ".md" {
			t.Fatalf("glob let %s through", grp.Path)
		}
	}
}

// slicesHasPath reports whether any group is the path.
func slicesHasPath(r grep.Result, path string) bool {
	for _, grp := range r.Groups {
		if grp.Path == path {
			return true
		}
	}
	return false
}

func TestCloudWithoutManifestReadsEverything(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "cloud"})
	if err != nil {
		t.Fatalf("an absent declaration hides nothing: %v", err)
	}
	if res.TotalHits == 0 {
		t.Fatal("the cloud channel must answer without a declaration")
	}
}

func TestCloudLocalOnlyAreaFails(t *testing.T) {
	root := grepFixture(t)
	dir := filepath.Join(root, ".loomux")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[area]\nscope = \"fixture\"\n\n[privacy]\nmode = \"local_only\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Channel: "cloud"}); err == nil ||
		!strings.Contains(err.Error(), "local_only") {
		t.Fatalf("err = %v, want the local_only refusal", err)
	}
}

func TestGlobWithoutSlashMatchesEveryDepth(t *testing.T) {
	files := grepFiles()
	files["docs/page.md"] = "zzzUniqueWord\n"
	root := buildRepo(t, files)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Glob: "*.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !slicesHasPath(res, "README.md") || !slicesHasPath(res, "docs/page.md") {
		t.Fatalf("a bare glob must match at every depth, got %+v", res.Groups)
	}
}

func TestGlobDoesNotMakeAPrefixLookMissing(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{Glob: "*.md", In: "lib"})
	if err != nil {
		t.Fatalf("an indexed prefix narrowed by a glob must not fail: %v", err)
	}
	if res.TotalHits != 0 {
		t.Fatalf("no markdown file lives under lib, got %d hits", res.TotalHits)
	}
}
