package corpus

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/xidus90/loomux/internal/testlock"
)

// fakeMatcher is a Matcher the tests configure, recording the paths it was
// asked about.
type fakeMatcher struct {
	ignoredDirs  map[string]bool
	ignoredFiles map[string]bool
	tracked      map[string]bool
	trackedDirs  map[string]bool
	asked        *[]string
	// askedMu guards asked: the walk reads directories concurrently, and the
	// fake does not rely on Walk serializing its calls.
	askedMu *sync.Mutex
}

func (f fakeMatcher) Ignored(rel string, isDir bool) bool {
	if f.asked != nil {
		f.askedMu.Lock()
		*f.asked = append(*f.asked, "ignored:"+rel)
		f.askedMu.Unlock()
	}
	if isDir {
		return f.ignoredDirs[rel]
	}
	if f.tracked[rel] {
		// A tracked file is never ignored, the same rule git applies.
		return false
	}
	if f.ignoredFiles[rel] {
		return true
	}
	for dir := range f.ignoredDirs {
		if strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	return false
}

func (f fakeMatcher) Tracked(rel string) bool { return f.tracked[rel] }

func (f fakeMatcher) HasTrackedUnder(dir string) bool { return f.trackedDirs[dir] }

// writeTree writes a file below root, creating its directories.
func writeTree(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// walkTree builds the test tree of the walk tests and returns its root.
func walkTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range map[string]string{
		"a.go":                           "package a\n",
		".x.go":                          "package hidden\n",
		".github/workflows/ci.yml":       "name: ci\n",
		".loomux/state/graph/graph.json": "{}\n",
		"src/main.go":                    "package main\n",
		"node_modules/dep/index.js":      "module.exports = 1;\n",
		"dist/tracked.js":                "tracked\n",
		"docs/gen/api.md":                "generated\n",
		"bin/run.sh":                     "#!/bin/sh\n",
	} {
		writeTree(t, root, rel, body)
	}
	return root
}

// walkMatcher is the fake matching the test tree: node_modules is ignored and
// has no tracked file, dist is ignored but has one, docs/gen is ignored and is
// not a pruned name.
func walkMatcher(asked *[]string) fakeMatcher {
	return fakeMatcher{
		ignoredDirs:  map[string]bool{"node_modules": true, "dist": true, "docs/gen": true},
		ignoredFiles: map[string]bool{},
		tracked:      map[string]bool{"dist/tracked.js": true},
		trackedDirs:  map[string]bool{"dist": true},
		asked:        asked,
		askedMu:      &sync.Mutex{},
	}
}

// rels is the entry paths of a listing, in order.
func rels(listing Listing) []string {
	out := make([]string, 0, len(listing.Entries))
	for _, e := range listing.Entries {
		out = append(out, e.Rel)
	}
	return out
}

// entry finds one entry by path.
func entry(t *testing.T, listing Listing, rel string) Entry {
	t.Helper()
	for _, e := range listing.Entries {
		if e.Rel == rel {
			return e
		}
	}
	t.Fatalf("entry %s missing from %v", rel, rels(listing))
	return Entry{}
}

// TestWalkPrunesDotDirs pins that no file below a dot directory appears, and
// that a dot file in a normal directory stays: the rule is about directories.
func TestWalkPrunesDotDirs(t *testing.T) {
	listing, err := Walk(walkTree(t), walkMatcher(nil))
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	got := rels(listing)
	want := []string{".x.go", "a.go", "bin/run.sh", "dist/tracked.js", "docs/gen/api.md", "src/main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	if !listing.Repo {
		t.Fatal("Repo = false with a matcher")
	}
}

// TestWalkPrunesIgnoredPrunedDirWithoutTracked pins that an ignored
// node_modules is not entered.
func TestWalkPrunesIgnoredPrunedDirWithoutTracked(t *testing.T) {
	listing, err := Walk(walkTree(t), walkMatcher(nil))
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	for _, rel := range rels(listing) {
		if strings.HasPrefix(rel, "node_modules/") {
			t.Fatalf("walk entered node_modules: %s", rel)
		}
	}
}

// TestWalkDescendsIgnoredPrunedDirWithTracked pins that a tracked file wins
// over the ignore rule, and that it is not marked ignored.
func TestWalkDescendsIgnoredPrunedDirWithTracked(t *testing.T) {
	listing, err := Walk(walkTree(t), walkMatcher(nil))
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	e := entry(t, listing, "dist/tracked.js")
	if !e.Tracked || e.Ignored {
		t.Fatalf("dist/tracked.js = %+v, want Tracked true, Ignored false", e)
	}
}

// TestWalkDescendsIgnoredNonPrunedDir pins that an ignored directory that is
// not a pruned name is walked and its files are marked ignored.
func TestWalkDescendsIgnoredNonPrunedDir(t *testing.T) {
	listing, err := Walk(walkTree(t), walkMatcher(nil))
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	e := entry(t, listing, "docs/gen/api.md")
	if !e.Ignored || e.Tracked {
		t.Fatalf("docs/gen/api.md = %+v, want Ignored true, Tracked false", e)
	}
}

// TestWalkOutsideRepo pins that without a matcher nothing is ignored and the
// pruned directories are ordinary directories.
func TestWalkOutsideRepo(t *testing.T) {
	listing, err := Walk(walkTree(t), nil)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if listing.Repo {
		t.Fatal("Repo = true without a matcher")
	}
	e := entry(t, listing, "node_modules/dep/index.js")
	if e.Ignored || e.Tracked {
		t.Fatalf("node_modules/dep/index.js = %+v, want neither flag", e)
	}
	for _, e := range listing.Entries {
		if e.Ignored {
			t.Fatalf("%s marked ignored without a matcher", e.Rel)
		}
	}
}

// TestWalkDoesNotAskDotDirs pins that a dot directory is pruned without
// consulting the matcher: no rule can make the walk enter it.
func TestWalkDoesNotAskDotDirs(t *testing.T) {
	var asked []string
	if _, err := Walk(walkTree(t), walkMatcher(&asked)); err != nil {
		t.Fatalf("Walk: %v", err)
	}
	for _, q := range asked {
		if strings.Contains(q, ".github") || strings.Contains(q, ".loomux") {
			t.Fatalf("matcher asked about a dot directory: %s", q)
		}
	}
}

// TestWalkSymlinkFlag pins that a symlink is an entry of its own, not its
// target, and never followed.
func TestWalkSymlinkFlag(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "target.txt", "body\n")
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(filepath.Join(root, "target.txt"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	listing, err := Walk(root, nil)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if e := entry(t, listing, "link.txt"); !e.Link {
		t.Fatalf("link.txt = %+v, want Link true", e)
	}
	if e := entry(t, listing, "target.txt"); e.Link {
		t.Fatalf("target.txt = %+v, want Link false", e)
	}
}

// TestWalkNeedsAnExistingRoot pins the error of a missing root, the way the
// graph's stat walk reports it.
func TestWalkNeedsAnExistingRoot(t *testing.T) {
	if _, err := Walk(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("Walk of a missing root returned no error")
	}
}

// TestWalkEntryMetadata pins size and modification time against the file the
// test wrote.
func TestWalkEntryMetadata(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "data.txt", "0123456789")
	listing, err := Walk(root, nil)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	e := entry(t, listing, "data.txt")
	if e.Size != 10 || e.MTime == 0 || e.Abs != filepath.Join(root, "data.txt") {
		t.Fatalf("data.txt = %+v, want size 10, mtime set, abs path", e)
	}
}

// TestSkipSegment pins the shared prune predicate: dot directories and the
// dependency/build output names, nothing else.
func TestSkipSegment(t *testing.T) {
	for _, name := range []string{".git", ".loomux", "node_modules", "testdata", "dist", "vendor", "obj", "htmlcov"} {
		if !SkipSegment(name) {
			t.Errorf("SkipSegment(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"src", "docs", "internal", "main.go", "bin"} {
		if SkipSegment(name) {
			t.Errorf("SkipSegment(%q) = true, want false", name)
		}
	}
}

// TestPrunedDir pins the segment filter without the dot rule: sourceset's
// filter runs on enumerated paths, where a dot segment never appears.
func TestPrunedDir(t *testing.T) {
	if !PrunedDir("node_modules") || !PrunedDir("testdata") {
		t.Fatal("PrunedDir misses a pruned name")
	}
	if PrunedDir(".git") || PrunedDir("docs") {
		t.Fatal("PrunedDir claims a name that is not one of pruneDirs")
	}
}

// TestWalkContinuesWhenADirectoryIsLocked pins that a read error of one
// directory costs that directory, not the walk.
func TestWalkContinuesWhenADirectoryIsLocked(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "accessible.go", "package main\n")
	writeTree(t, root, "locked/hidden.go", "package hidden\n")
	testlock.LockDir(t, filepath.Join(root, "locked"))

	listing, err := Walk(root, nil)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if got, want := rels(listing), []string{"accessible.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}
