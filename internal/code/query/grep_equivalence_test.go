package query

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/gitenv"
)

// gitRepo writes files into a fresh repository, so the corpus has ignore
// rules to apply. tracked names are added with -f, which is the exception the
// tracked set exists for.
func grepGitRepo(t *testing.T, files map[string]string, tracked []string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	if err := os.MkdirAll(filepath.Join(home, "xdg"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	grepRunGit(t, root, "init", "-q")
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if len(tracked) > 0 {
		args := append([]string{"add", "-f"}, tracked...)
		grepRunGit(t, root, args...)
	}
	return root
}

func grepRunGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(gitenv.Environ(),
		"GIT_CONFIG_NOSYSTEM="+os.Getenv("GIT_CONFIG_NOSYSTEM"),
		"GIT_CONFIG_GLOBAL="+os.Getenv("GIT_CONFIG_GLOBAL"),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// grepFiles is a search corpus wider than the graph: prose, an ignored log, a
// binary blob and a file too large for a small limit.
func grepFiles() map[string]string {
	files := sample()
	files["README.md"] = "# Repo\n\nThe needle zzzUniqueWord lives here.\n"
	files["ignored.log"] = "zzzUniqueWord\n"
	files[".gitignore"] = "*.log\n"
	files["blob.bin"] = "text\x00zzzUniqueWord\n"
	files["big.txt"] = strings.Repeat("x", 64) + "\nzzzUniqueWord\n"
	return files
}

// buildRepo writes files into a fresh repository and builds its graph, corpus
// and index.
func buildRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := grepGitRepo(t, files, nil)
	if _, _, err := BuildWith(root, BuildOptions{}, ignore); err != nil {
		t.Fatal(err)
	}
	return root
}

func grepFixture(t *testing.T) string {
	t.Helper()
	return buildRepo(t, grepFiles())
}

func indexPath(root string) string { return store.CachePath(root, "trigram.idx") }

// comparable blanks the fields that are allowed to differ between the index
// path and the scan path.
func comparable(r grep.Result) grep.Result {
	r.FilesSearched = 0
	r.Index = ""
	r.FilesInScope = 0
	r.Candidates = 0
	return r
}

func TestEquivalenceWithScan(t *testing.T) {
	root := grepFixture(t)
	saved, err := os.ReadFile(indexPath(root))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		pattern string
		opts    GrepOptions
	}{
		{"literal", "zzzUniqueWord", GrepOptions{}},
		{"literal case folded", "ZZZUNIQUEWORD", GrepOptions{IgnoreCase: true}},
		{"alternation", "zzzUniqueWord|package lib", GrepOptions{}},
		{"no literal", `[A-Z]{3}\w+`, GrepOptions{}},
		{"fixed", "zzzUniqueWord", GrepOptions{Fixed: true}},
		{"in prefix", "package", GrepOptions{In: "lib"}},
		{"max hits", "e", GrepOptions{MaxHits: 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			indexed, _, err := Grep(root, c.pattern, c.opts)
			if err != nil {
				t.Fatalf("indexed: %v", err)
			}
			if indexed.Index == "none" || indexed.Index == "stale" {
				t.Fatalf("index state = %q, want a current index", indexed.Index)
			}
			if err := os.Remove(indexPath(root)); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.WriteFile(indexPath(root), saved, 0o644); err != nil {
					t.Fatal(err)
				}
			}()
			scan, _, err := Grep(root, c.pattern, c.opts)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			if scan.Index != "none" {
				t.Fatalf("scan index state = %q, want none", scan.Index)
			}
			if a, b := comparable(indexed), comparable(scan); a.TotalHits != b.TotalHits ||
				a.TruncatedHits != b.TruncatedHits || a.Unreadable != b.Unreadable ||
				a.SkippedBinary != b.SkippedBinary || a.SkippedSize != b.SkippedSize ||
				a.Ignored != b.Ignored || len(a.Groups) != len(b.Groups) {
				t.Fatalf("indexed = %+v\nscan = %+v", a, b)
			}
			if len(indexed.Groups) > 0 {
				for i := range indexed.Groups {
					if indexed.Groups[i].Path != scan.Groups[i].Path ||
						indexed.Groups[i].InDegree != scan.Groups[i].InDegree ||
						!slices.Equal(indexed.Groups[i].Hits, scan.Groups[i].Hits) {
						t.Fatalf("group %d differs:\nindexed %+v\nscan %+v", i, indexed.Groups[i], scan.Groups[i])
					}
				}
			}
		})
	}
}

func TestScanCoversNonGraphFiles(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, grp := range res.Groups {
		if grp.Path == "README.md" {
			found = true
			if grp.Symbol != nil {
				t.Fatalf("a README hit must be file level, got %+v", grp.Symbol)
			}
		}
	}
	if !found {
		t.Fatalf("README.md did not reach the result: %+v", res.Groups)
	}
}

// An invalid pattern is rejected by the search, not by the corpus walk, so the
// error travels back through Grep with whatever the search found until then.
func TestGrepReportsTheSearchError(t *testing.T) {
	root := grepFixture(t)
	if _, _, err := Grep(root, "(", GrepOptions{}); err == nil {
		t.Fatal("want an error for an invalid pattern")
	}
}

func TestIgnoredFileNotSearched(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Ignored == 0 {
		t.Fatal("the ignored log was not counted")
	}
	if slices.ContainsFunc(res.Groups, func(g grep.Group) bool { return g.Path == "ignored.log" }) {
		t.Fatal("an ignored file was searched")
	}
}

func TestBinaryAndSizeCounted(t *testing.T) {
	root := grepFixture(t)
	limit := int64(16)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{MaxFilesize: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if res.SkippedBinary != 1 {
		t.Fatalf("SkippedBinary = %d, want 1", res.SkippedBinary)
	}
	if res.SkippedSize == 0 {
		t.Fatal("the size limit skipped nothing")
	}
	for _, grp := range res.Groups {
		if grp.Path == "blob.bin" || grp.Path == "big.txt" {
			t.Fatalf("%s must not be searched", grp.Path)
		}
	}
}

func TestIndexUsedWhenLiteral(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "used" || res.Candidates == 0 {
		t.Fatalf("index = %q candidates = %d, want used with candidates", res.Index, res.Candidates)
	}
	if res.Candidates >= res.FilesInScope {
		t.Fatalf("candidates = %d of %d files; the literal narrowed nothing", res.Candidates, res.FilesInScope)
	}
}

func TestIndexScanWhenNoLiteral(t *testing.T) {
	root := grepFixture(t)
	res, _, err := Grep(root, `[A-Z]{3}\w+`, GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "scan" || res.Candidates != res.FilesInScope {
		t.Fatalf("index = %q candidates = %d of %d files, want a plain scan of the whole scope", res.Index, res.Candidates, res.FilesInScope)
	}
}

func TestStaleIndexScans(t *testing.T) {
	root := grepFixture(t)
	raw, err := os.ReadFile(indexPath(root))
	if err != nil {
		t.Fatal(err)
	}
	raw[0] = 'X'
	if err := os.WriteFile(indexPath(root), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale", res.Index)
	}
	if res.TotalHits == 0 {
		t.Fatal("a stale index must still answer")
	}
}

func TestSettingsMismatchIsStale(t *testing.T) {
	root := grepFixture(t)
	limit := int64(1024)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{MaxFilesize: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "stale" {
		t.Fatalf("index = %q, want stale for another limit", res.Index)
	}
}

func TestNoIndexScans(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(indexPath(root)); err != nil {
		t.Fatal(err)
	}
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "none" || res.TotalHits == 0 {
		t.Fatalf("index = %q hits = %d, want none and an answer", res.Index, res.TotalHits)
	}
}

func TestMaxHitsSameOnBothPaths(t *testing.T) {
	root := grepFixture(t)
	opts := GrepOptions{MaxHits: 1}
	indexed, _, err := Grep(root, "e", opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(indexPath(root)); err != nil {
		t.Fatal(err)
	}
	scan, _, err := Grep(root, "e", opts)
	if err != nil {
		t.Fatal(err)
	}
	if indexed.TotalHits != 1 || scan.TotalHits != 1 || indexed.TruncatedHits != scan.TruncatedHits {
		t.Fatalf("indexed hits=%d truncated=%d, scan hits=%d truncated=%d",
			indexed.TotalHits, indexed.TruncatedHits, scan.TotalHits, scan.TruncatedHits)
	}
}

func TestGrepCorpusSelectionErrorIsReported(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(corpus.RecordPath(root)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(corpus.RecordPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Grep(root, "zzzUniqueWord", GrepOptions{NoRefresh: true}); err != nil {
		t.Fatalf("a directory in the record's place must not fail a read-only grep: %v", err)
	}
}
