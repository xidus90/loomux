package query

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/model"
)

// countBuilds counts the rebuilds grep starts for one test.
func countBuilds(t *testing.T) *int {
	t.Helper()
	n := 0
	old := buildWith
	buildWith = func(root string, opts BuildOptions, say func(string)) (*model.Graph, Stats, error) {
		n++
		return old(root, opts, say)
	}
	t.Cleanup(func() { buildWith = old })
	return &n
}

// countTracked counts the git calls that list the tracked files.
func countTracked(t *testing.T) *int {
	t.Helper()
	n := 0
	old := trackedSet
	trackedSet = func(root string) ([]string, error) {
		n++
		return old(root)
	}
	t.Cleanup(func() { trackedSet = old })
	return &n
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// oracleHits is what a line scan of every search file finds, as path:line.
// The fixture's corpus is every file outside the dot directories except the
// ignored log and the binary blob.
func oracleHits(t *testing.T, root, pattern string) []string {
	t.Helper()
	re := regexp.MustCompile(pattern)
	var hits []string
	err := filepath.WalkDir(root, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if abs != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, abs)
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(abs)
		if err != nil || rel == "ignored.log" || corpus.IsBinary(data) {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			if re.MatchString(strings.TrimSuffix(line, "\r")) {
				hits = append(hits, fmt.Sprintf("%s:%d", rel, i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(hits)
	return hits
}

func resultHits(r grep.Result) []string {
	var hits []string
	for _, grp := range r.Groups {
		for _, h := range grp.Hits {
			hits = append(hits, fmt.Sprintf("%s:%d", grp.Path, h.Line))
		}
	}
	slices.Sort(hits)
	return hits
}

func TestOverlayFindsNewLines(t *testing.T) {
	root := grepFixture(t)
	writeTree(t, root, map[string]string{
		"README.md":  "# Repo\n\nzzzOverlayNeedle\n",
		"lib/lib.go": "package lib\n\n// zzzOverlayNeedle\n// Run does the thing.\nfunc Run() {}\n",
		"notes.txt":  "first\nzzzOverlayNeedle\n",
	})
	builds := countBuilds(t)
	for _, pattern := range []string{"zzzOverlayNeedle|zzzUniqueWord", `[a-z]{3}OverlayNeedle`} {
		res, notes, err := Grep(root, pattern, GrepOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Index != "overlay" {
			t.Fatalf("%s: index = %q, want overlay (notes %v)", pattern, res.Index, notes)
		}
		if got, want := resultHits(res), oracleHits(t, root, pattern); !slices.Equal(got, want) {
			t.Fatalf("%s: hits = %v, want %v", pattern, got, want)
		}
	}
	if *builds != 0 {
		t.Fatalf("builds = %d, want none", *builds)
	}
}

func TestOverlayHitsHaveNoSymbolInChangedFiles(t *testing.T) {
	root := grepFixture(t)
	// The old graph spans line 4 as Run; the line now says something else.
	writeTree(t, root, map[string]string{
		"lib/lib.go": "package lib\n\n// Run does the thing.\nvar zzzRunMoved = 1\n\nfunc Run() {}\n",
	})
	res, _, err := Grep(root, "zzzRunMoved|lib.Run", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" {
		t.Fatalf("index = %q, want overlay", res.Index)
	}
	seen := map[string]bool{}
	for _, grp := range res.Groups {
		seen[grp.Path] = true
		switch grp.Path {
		case "lib/lib.go":
			if grp.Symbol != nil || grp.InDegree != 0 {
				t.Fatalf("changed file got symbol %+v in_degree %d", grp.Symbol, grp.InDegree)
			}
		case "main.go":
			if grp.Symbol == nil {
				t.Fatal("an unchanged file lost its symbol")
			}
		}
	}
	if !seen["lib/lib.go"] || !seen["main.go"] {
		t.Fatalf("groups = %s, want lib/lib.go and main.go", symbolGroups(t, res))
	}
}

func TestOverlayIgnoresDeletedFiles(t *testing.T) {
	root := grepFixture(t)
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	builds := countBuilds(t)
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" || *builds != 0 {
		t.Fatalf("index = %q builds = %d, want overlay and none", res.Index, *builds)
	}
	if slicesHasPath(res, "README.md") {
		t.Fatal("a deleted file answered")
	}
	if res.TotalHits == 0 {
		t.Fatal("the files left must still answer")
	}
}

// committedRepo is the fixture with every file committed, so .git/index
// exists and the record tracks files.
func committedRepo(t *testing.T) string {
	t.Helper()
	root := grepGitRepo(t, grepFiles(), nil)
	commit(t, root, "init", "-A")
	if _, _, err := BuildWith(root, BuildOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	return root
}

func commit(t *testing.T, root, msg string, paths ...string) {
	t.Helper()
	grepRunGit(t, root, append([]string{"add"}, paths...)...)
	grepRunGit(t, root, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-qm", msg)
}

func statStamp(t *testing.T, path string) corpus.Stamp {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return corpus.Stamp{Size: info.Size(), MTime: info.ModTime().UnixNano()}
}

func TestOverlayAfterACommit(t *testing.T) {
	root := committedRepo(t)
	before, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no corpus record")
	}
	writeTree(t, root, map[string]string{"README.md": "# Repo\n\nzzzCommittedNeedle\n"})
	commit(t, root, "edit", "README.md")

	calls, builds := countTracked(t), countBuilds(t)
	res, notes, err := Grep(root, "zzzCommittedNeedle", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" || res.TotalHits != 1 || *builds != 0 || *calls != 1 {
		t.Fatalf("index = %q hits = %d builds = %d git calls = %d, want overlay, 1, 0, 1 (notes %v)",
			res.Index, res.TotalHits, *builds, *calls, notes)
	}
	after, ok := corpus.Read(root)
	if !ok {
		t.Fatal("no corpus record after the overlay")
	}
	if after.Digest != before.Digest {
		t.Fatal("the restamp changed the digest")
	}
	if want := statStamp(t, filepath.Join(root, ".git", "index")); after.Stamp != want {
		t.Fatalf("stamp = %+v, want the moved index %+v", after.Stamp, want)
	}

	*calls = 0
	res, _, err = Grep(root, "zzzCommittedNeedle", GrepOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" || *calls != 0 {
		t.Fatalf("index = %q git calls = %d, want overlay and no git call", res.Index, *calls)
	}
}

func TestOverlayRestampFailureIsANotice(t *testing.T) {
	root := committedRepo(t)
	writeTree(t, root, map[string]string{"README.md": "zzzCommittedNeedle\n"})
	commit(t, root, "edit", "README.md")
	old := writeRecord
	writeRecord = func(string, corpus.Record) error { return errors.New("disk full") }
	t.Cleanup(func() { writeRecord = old })
	res, notes, err := Grep(root, "zzzCommittedNeedle", GrepOptions{})
	if err != nil {
		t.Fatalf("a failed restamp must not fail the answer: %v", err)
	}
	if res.Index != "overlay" || res.TotalHits != 1 {
		t.Fatalf("index = %q hits = %d, want overlay and one hit", res.Index, res.TotalHits)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "disk full") {
		t.Fatalf("notes = %q, want the write failure", notes)
	}
}

// A git index that vanished matches the empty stamp, the way the probe reads
// it, so the next call does not list the tracked files again.
func TestOverlayRestampsAVanishedGitIndex(t *testing.T) {
	root := grepGitRepo(t, grepFiles(), nil)
	grepRunGit(t, root, "add", "README.md")
	grepRunGit(t, root, "rm", "-q", "--cached", "README.md")
	if _, _, err := BuildWith(root, BuildOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, ".git", "index")); err != nil {
		t.Fatal(err)
	}
	res, _, err := Grep(root, "zzzUniqueWord", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" {
		t.Fatalf("index = %q, want overlay", res.Index)
	}
	if rec, _ := corpus.Read(root); rec.Stamp != (corpus.Stamp{}) {
		t.Fatalf("stamp = %+v, want the empty stamp", rec.Stamp)
	}
}

// A rebuild that wrote a new record while the overlay checked the git index
// keeps its record: the restamp would bind the old file list to the new index.
func TestOverlayRestampKeepsAConcurrentRecord(t *testing.T) {
	root := committedRepo(t)
	before, _ := corpus.Read(root)
	writeTree(t, root, map[string]string{"README.md": "zzzCommittedNeedle\n"})
	commit(t, root, "edit", "README.md")
	old := trackedSet
	trackedSet = func(root string) ([]string, error) {
		rec, _ := corpus.Read(root)
		rec.Digest = "rebuilt"
		if err := corpus.Write(root, rec); err != nil {
			t.Fatal(err)
		}
		return old(root)
	}
	t.Cleanup(func() { trackedSet = old })
	res, _, err := Grep(root, "zzzCommittedNeedle", GrepOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" {
		t.Fatalf("index = %q, want overlay", res.Index)
	}
	after, _ := corpus.Read(root)
	if after.Digest != "rebuilt" || after.Stamp != before.Stamp {
		t.Fatalf("record = digest %q stamp %+v, want the concurrent record untouched", after.Digest, after.Stamp)
	}
}

func TestOverlayLimit(t *testing.T) {
	for _, c := range []struct {
		files  int
		index  string
		builds int
	}{
		{overlayLimit, "overlay", 0},
		{overlayLimit + 1, "used", 1},
	} {
		t.Run(fmt.Sprint(c.files), func(t *testing.T) {
			files := grepFiles()
			for i := range c.files {
				files[fmt.Sprintf("many/f%03d.txt", i)] = "x\n"
			}
			root := buildRepo(t, files)
			changed := map[string]string{}
			for i := range c.files {
				changed[fmt.Sprintf("many/f%03d.txt", i)] = "zzzManyNeedle\n"
			}
			writeTree(t, root, changed)
			builds := countBuilds(t)
			res, _, err := Grep(root, "zzzManyNeedle", GrepOptions{MaxHits: 1000})
			if err != nil {
				t.Fatal(err)
			}
			if res.Index != c.index || *builds != c.builds || res.TotalHits != c.files {
				t.Fatalf("index = %q builds = %d hits = %d, want %q, %d, %d",
					res.Index, *builds, res.TotalHits, c.index, c.builds, c.files)
			}
		})
	}
}

func TestOverlayUnderNoRefresh(t *testing.T) {
	root := grepFixture(t)
	writeTree(t, root, map[string]string{"README.md": "zzzFreshNeedle\n"})
	res, _, err := Grep(root, "zzzFreshNeedle", GrepOptions{NoRefresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Index != "overlay" || res.TotalHits != 1 {
		t.Fatalf("index = %q hits = %d, want overlay and one hit", res.Index, res.TotalHits)
	}
}

func TestOverlayWithGlobInAndCloud(t *testing.T) {
	root := privacyRepo(t, `"secrets/**"`)
	writeTree(t, root, map[string]string{
		"README.md":       "zzzOverlayNeedle\n",
		"docs/secret.md":  "zzzOverlayNeedle\n",
		"secrets/key.txt": "zzzOverlayNeedle\n",
		"lib/lib.go":      "package lib\n\n// zzzOverlayNeedle\nfunc Run() {}\n",
	})
	cases := []struct {
		name string
		opts GrepOptions
		want []string
	}{
		{"glob", GrepOptions{Glob: "**/*.md"}, []string{"README.md:1", "docs/secret.md:1"}},
		{"in", GrepOptions{In: "lib"}, []string{"lib/lib.go:3"}},
		{"cloud", GrepOptions{Channel: "cloud"}, []string{"README.md:1", "docs/secret.md:1", "lib/lib.go:3"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, _, err := Grep(root, "zzzOverlayNeedle", c.opts)
			if err != nil {
				t.Fatal(err)
			}
			if res.Index != "overlay" {
				t.Fatalf("index = %q, want overlay", res.Index)
			}
			if got := resultHits(res); !slices.Equal(got, c.want) {
				t.Fatalf("hits = %v, want %v", got, c.want)
			}
		})
	}
}

func TestOverlayFits(t *testing.T) {
	root := grepFixture(t)
	n := func(k int) []string { return make([]string, k) }
	base := func() grepState {
		return grepState{
			rec:         corpus.Record{Tracked: []string{"a.go"}},
			hasRecord:   true,
			corpusDrift: &corpus.Drift{Changed: n(1)},
			graphDrift:  &freshness.Drift{},
		}
	}
	listed := func(tracked []string, err error) func(string) ([]string, error) {
		return func(string) ([]string, error) { return tracked, err }
	}
	cases := []struct {
		name    string
		root    string
		modify  func(s *grepState)
		tracked func(string) ([]string, error)
		want    bool
		calls   int
	}{
		{"content", root, func(*grepState) {}, nil, true, 0},
		{"no record", root, func(s *grepState) { s.hasRecord, s.corpusDrift = false, nil }, nil, false, 0},
		{"settings", root, func(s *grepState) { s.corpusDrift.Settings = true }, nil, false, 0},
		{"config", root, func(s *grepState) { s.corpusDrift.Config = true }, nil, false, 0},
		{"sources", root, func(s *grepState) { s.corpusDrift.Sources = n(1) }, nil, false, 0},
		{"no fingerprint", root, func(s *grepState) { s.graphDrift = nil }, nil, false, 0},
		{"no ask index", t.TempDir(), func(*grepState) {}, nil, false, 0},
		{"corpus at the limit", root, func(s *grepState) {
			s.corpusDrift = &corpus.Drift{Changed: n(100), Added: n(50), Removed: n(50)}
		}, nil, true, 0},
		{"corpus over the limit", root, func(s *grepState) {
			s.corpusDrift = &corpus.Drift{Changed: n(100), Added: n(50), Removed: n(51)}
		}, nil, false, 0},
		{"graph at the limit", root, func(s *grepState) {
			s.graphDrift = &freshness.Drift{Changed: n(100), Added: n(50), Removed: n(50)}
		}, nil, true, 0},
		{"graph over the limit", root, func(s *grepState) {
			s.graphDrift = &freshness.Drift{Changed: n(100), Added: n(50), Removed: n(51)}
		}, nil, false, 0},
		// The git index flag is not a file: 200 files and a moved index fit.
		{"index, same files", root, func(s *grepState) {
			s.corpusDrift = &corpus.Drift{Changed: n(overlayLimit), Index: true}
		}, listed([]string{"a.go"}, nil), true, 1},
		{"index, other files", root, func(s *grepState) { s.corpusDrift.Index = true },
			listed([]string{"a.go", "b.go"}, nil), false, 1},
		{"index, git fails", root, func(s *grepState) { s.corpusDrift.Index = true },
			listed(nil, errors.New("git broken")), false, 1},
		{"index, too many files", root, func(s *grepState) {
			s.corpusDrift = &corpus.Drift{Changed: n(overlayLimit + 1), Index: true}
		}, listed([]string{"a.go"}, nil), false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := 0
			old := trackedSet
			trackedSet = func(r string) ([]string, error) {
				calls++
				return c.tracked(r)
			}
			t.Cleanup(func() { trackedSet = old })
			s := base()
			c.modify(&s)
			if got := overlayFits(s, c.root); got != c.want || calls != c.calls {
				t.Fatalf("fits = %v git calls = %d, want %v and %d", got, calls, c.want, c.calls)
			}
		})
	}
}

// A relative index path sits under the repository root, as the probe reads it.
func TestGitIndexStampOfARelativePath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "git", "index")
	writeTree(t, root, map[string]string{"git/index": "DIRC"})
	rec := corpus.Record{GitPaths: corpus.GitPaths{Root: root, Index: "git/index"}}
	if got, want := gitIndexStamp(rec), statStamp(t, path); got != want {
		t.Fatalf("stamp = %+v, want %+v", got, want)
	}
}
