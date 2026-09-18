package sourceset_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xidus90/loomux/internal/code/sourceset"
)

// tree writes files into a fresh directory. The map's keys are slash paths
// relative to the root.
func tree(t *testing.T, files map[string]string) string {
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

func TestListTakesGoFilesAndNothingElse(t *testing.T) {
	root := tree(t, map[string]string{
		"main.go":            "package main\n",
		"pkg/helper.go":      "package pkg\n",
		"pkg/helper_test.go": "package pkg\n",
		"README.md":          "# no\n",
		"web/app.ts":         "export {}\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// _test.go stays in: Graft does not exclude tests, it de-ranks them at
	// query time, and "where are the tests" is a fair question of the graph.
	want := []string{"main.go", "pkg/helper.go", "pkg/helper_test.go"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestListSkipsDependencyAndBuildDirectories(t *testing.T) {
	root := tree(t, map[string]string{
		"keep.go":              "package a\n",
		"vendor/dep/dep.go":    "package dep\n",
		"node_modules/x/x.go":  "package x\n",
		"dist/out.go":          "package out\n",
		"_build/gen.go":        "package gen\n",
		".hidden/secret.go":    "package secret\n",
		"internal/bin/keep.go": "package keep\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// A dot directory is skipped wholesale; the skip list matches a single path
	// segment, so a directory merely NAMED like an output dir deeper in the
	// tree is not exempt -- but "internal/bin" is not on the list at all.
	want := map[string]bool{"keep.go": true, "internal/bin/keep.go": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want the two keepers", got)
	}
	for _, rel := range got {
		if !want[rel] {
			t.Errorf("unexpected file %q", rel)
		}
	}
}

func TestListDropsAFileOverTheSizeLimit(t *testing.T) {
	big := "package big\n" + string(make([]byte, 1_000_001))
	root := tree(t, map[string]string{"big.go": big, "small.go": "package small\n"})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// Above a megabyte a file is generated or vendored in practice, not written
	// by hand.
	if len(got) != 1 || got[0] != "small.go" {
		t.Fatalf("got %v, want only small.go", got)
	}
}

func TestStatCarriesSizeAndMTimeAndSlashPaths(t *testing.T) {
	root := tree(t, map[string]string{"pkg/a.go": "package pkg\n"})

	got, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d files, want 1", len(got))
	}
	f := got[0]
	if f.Rel != "pkg/a.go" {
		t.Errorf("Rel = %q, want forward slashes and no leading dot", f.Rel)
	}
	if f.Size != int64(len("package pkg\n")) {
		t.Errorf("Size = %d, want %d", f.Size, len("package pkg\n"))
	}
	// mtime is nanoseconds as an int64 -- never a float, because equality of
	// this field decides whether a rebuild happens.
	if f.MTime == 0 {
		t.Error("MTime = 0, want the file's modification time")
	}
	if !filepath.IsAbs(f.Abs) {
		t.Errorf("Abs = %q, want an absolute path", f.Abs)
	}
}

func TestListSkipsTheOutputDirectory(t *testing.T) {
	root := tree(t, map[string]string{
		"a.go":                            "package a\n",
		".loomux/state/graph/leftover.go": "package leftover\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "a.go" {
		t.Fatalf("got %v, want only a.go -- the graph must not index itself", got)
	}
}

func TestListRefusesAMissingRoot(t *testing.T) {
	_, err := sourceset.List(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("got nil, want an error for a root that is not there")
	}
}

func TestStatRefusesAMissingRoot(t *testing.T) {
	_, err := sourceset.Stat(filepath.Join(t.TempDir(), "nope"))
	if err == nil {
		t.Fatal("got nil, want an error for a root that is not there")
	}
}
