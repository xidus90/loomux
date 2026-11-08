package corpus

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/testlock"
)

// walkSerial is the oracle of the parallel walk: Walk as it was before it read
// directories concurrently, one filepath.WalkDir in lexical order.
func walkSerial(root string, m Matcher) (Listing, error) {
	if _, err := os.Stat(root); err != nil {
		return Listing{}, err
	}
	var entries []Entry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if m != nil && pruneDirs[name] && m.Ignored(rel, true) && !m.HasTrackedUnder(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if d.Name() == ".gitignore" {
			if loader, ok := m.(interface{ LoadIgnore(string) }); ok {
				loader.LoadIgnore(rel)
			}
		}
		entry := Entry{
			Rel:   rel,
			Abs:   path,
			Size:  info.Size(),
			MTime: info.ModTime().UnixNano(),
			Link:  d.Type()&fs.ModeSymlink != 0,
		}
		if m != nil {
			entry.Ignored = m.Ignored(rel, false)
			entry.Tracked = m.Tracked(rel)
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return Listing{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Rel < entries[j].Rel })
	return Listing{Entries: entries, Repo: m != nil}, nil
}

// parallelWalkRepo builds a repository three levels deep: .gitignore files in
// subdirectories, an ignored node_modules without a tracked file and one with,
// a symlink where the platform allows one, and a directory nobody can read.
func parallelWalkRepo(t *testing.T) string {
	t.Helper()
	files := map[string]string{
		".gitignore":                         "*.log\nnode_modules/\n",
		"a.go":                               "package a\n",
		"debug.log":                          "x",
		"pkg/.gitignore":                     "*.tmp\n!keep.tmp\n",
		"pkg/lib.go":                         "package pkg\n",
		"pkg/scratch.tmp":                    "x",
		"pkg/keep.tmp":                       "x",
		"pkg/inner/.gitignore":               "gen/\n",
		"pkg/inner/deep.go":                  "package inner\n",
		"pkg/inner/gen/out.go":               "package gen\n",
		"pkg/inner/trace.log":                "x",
		"node_modules/dep/index.js":          "module.exports = 1;\n",
		"web/node_modules/kept/index.js":     "tracked\n",
		"web/node_modules/kept/untracked.js": "x",
		"web/app.js":                         "x",
		"locked/hidden.go":                   "package hidden\n",
	}
	root := newRepo(t, files, []string{"a.go", "web/node_modules/kept/index.js"}, boolPtr(false))
	if err := os.Symlink(filepath.Join(root, "a.go"), filepath.Join(root, "pkg", "link.go")); err != nil {
		t.Logf("symlinks unavailable, the tree has none: %v", err)
	}
	testlock.LockDir(t, filepath.Join(root, "locked"))
	return root
}

// TestWalkEqualsSerialWalk pins that reading directories concurrently changes
// nothing about the listing, nor about the ignore sources the matcher read.
func TestWalkEqualsSerialWalk(t *testing.T) {
	root := parallelWalkRepo(t)
	file := filepath.Join(root, "a.go")

	for _, tc := range []struct {
		name string
		root string
		repo bool
	}{
		{"without matcher", root, false},
		{"with matcher", root, true},
		{"file as root", file, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var parallelM, serialM Matcher
			var parallelRules, serialRules *Rules
			if tc.repo {
				var err error
				if parallelRules, _, err = Load(tc.root); err != nil {
					t.Fatalf("Load: %v", err)
				}
				if serialRules, _, err = Load(tc.root); err != nil {
					t.Fatalf("Load: %v", err)
				}
				parallelM, serialM = parallelRules, serialRules
			}
			got, err := Walk(tc.root, parallelM)
			if err != nil {
				t.Fatalf("Walk: %v", err)
			}
			want, err := walkSerial(tc.root, serialM)
			if err != nil {
				t.Fatalf("walkSerial: %v", err)
			}
			if len(want.Entries) == 0 {
				t.Fatal("the oracle listed nothing; the comparison would be empty")
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Walk differs from the serial walk:\ngot  %+v\nwant %+v", got, want)
			}
			if tc.repo && !reflect.DeepEqual(parallelRules.Sources(), serialRules.Sources()) {
				t.Fatalf("Sources differ:\ngot  %v\nwant %v", parallelRules.Sources(), serialRules.Sources())
			}
		})
	}
}

// TestWalkReadsDirectoriesConcurrently pins that two sibling directories are
// read at the same time: the read of the first does not return until the read
// of the second has started, which a serial walk never reaches.
func TestWalkReadsDirectoriesConcurrently(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, "a/x.txt", "x")
	writeTree(t, root, "b/y.txt", "y")

	var mu sync.Mutex
	subdirs := 0
	second := make(chan struct{})
	overlapped := make(chan bool, 1)
	readDir = func(name string) ([]os.DirEntry, error) {
		if filepath.Clean(name) != filepath.Clean(root) {
			mu.Lock()
			subdirs++
			n := subdirs
			mu.Unlock()
			switch n {
			case 1:
				select {
				case <-second:
					overlapped <- true
				case <-time.After(5 * time.Second):
					overlapped <- false
				}
			case 2:
				close(second)
			}
		}
		return os.ReadDir(name)
	}
	t.Cleanup(func() { readDir = os.ReadDir })

	listing, err := Walk(root, nil)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	select {
	case ok := <-overlapped:
		if !ok {
			t.Fatal("the first directory read ended before the second began")
		}
	default:
		t.Fatal("no directory read went through the readDir seam")
	}
	if got, want := rels(listing), []string{"a/x.txt", "b/y.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}
