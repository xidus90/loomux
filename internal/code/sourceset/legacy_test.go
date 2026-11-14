package sourceset_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/sourceset"
)

// The constants and the walk of the graph list before it was derived from
// corpus.Walk (sourceset.Stat at 1816634d). They are the reference this
// package's differential test compares against, copied literally on purpose: a
// test that read them from production code would agree with any change.
const legacyMaxFileBytes = 1_000_000

var legacyExtensions = []string{".gd", ".go", ".godot", ".py", ".tres", ".tscn", ".uid"}

var legacySkipDirs = map[string]bool{
	"node_modules":  true,
	"dist":          true,
	"build":         true,
	"_build":        true,
	"out":           true,
	"target":        true,
	"vendor":        true,
	"coverage":      true,
	"__pycache__":   true,
	"venv":          true,
	"site-packages": true,
	"testdata":      true,
}

// legacyGraphList is the old Stat's enumeration, reduced to the paths.
func legacyGraphList(t *testing.T, root string) []string {
	t.Helper()
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("stat %s: %v", root, err)
	}
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") || legacySkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !slices.Contains(legacyExtensions, filepath.Ext(d.Name())) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > legacyMaxFileBytes {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}

// TestGraphListUnchanged pins the differential test of the shared walk: on a
// tree of edge cases and on this repository, the graph file list is the one the
// old walk produced.
func TestGraphListUnchanged(t *testing.T) {
	exact := "package big\n" + strings.Repeat("x", legacyMaxFileBytes-len("package big\n"))
	over := "package big\n" + strings.Repeat("x", legacyMaxFileBytes+1-len("package big\n"))
	root := tree(t, map[string]string{
		"keep.go":                 "package keep\n",
		".x.go":                   "package hidden\n",
		".hidden/secret.go":       "package secret\n",
		"testdata/case.go":        "package casex\n",
		"pkg/testdata/fixture.go": "package fixture\n",
		"node_modules/x/x.go":     "package x\n",
		"dist/nested/out.go":      "package out\n",
		"docs/gen/api.go":         "package gen\n",
		"UPPER.GO":                "package upper\n",
		"internal/bin/keep.go":    "package keep2\n",
		"exact.go":                exact,
		"over.go":                 over,
		"tool/run.py":             "x = 1\n",
		"resources/scene.tscn":    "[gd_scene]\n",
		"resources/project.godot": "[application]\n",
	})
	got, err := sourceset.List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := legacyGraphList(t, root); !slices.Equal(got, want) {
		t.Fatalf("List differs from the old walk:\ngot  %v\nwant %v", got, want)
	}

	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	gotRepo, err := sourceset.List(repo)
	if err != nil {
		t.Fatalf("List(repo): %v", err)
	}
	// The old walk knew fewer languages; the repository's files of a later one
	// (the web interface's HTML and CSS) are no difference of the walk.
	gotRepo = slices.DeleteFunc(gotRepo, func(p string) bool {
		return !slices.Contains(legacyExtensions, filepath.Ext(p))
	})
	if want := legacyGraphList(t, repo); !slices.Equal(gotRepo, want) {
		t.Fatalf("repository list differs from the old walk:\ngot  %d files\nwant %d files", len(gotRepo), len(want))
	}
}
