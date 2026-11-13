package sourceset_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/corpus"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/testlock"
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

func TestListTakesGoAndPythonFilesAndNothingElse(t *testing.T) {
	root := tree(t, map[string]string{
		"main.go":            "package main\n",
		"pkg/helper.go":      "package pkg\n",
		"pkg/helper_test.go": "package pkg\n",
		"tool/run.py":        "def run(): pass\n",
		"tool/run.pyi":       "def run() -> None: ...\n",
		"README.md":          "# no\n",
		"web/app.rb":         "puts 1\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// _test.go stays in: the original does not exclude tests, it de-ranks them at
	// query time, and "where are the tests" is a fair question of the graph.
	// A .pyi stub stays out: it repeats every definition of its module.
	want := []string{"main.go", "pkg/helper.go", "pkg/helper_test.go", "tool/run.py"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestExtensionsIsACopy(t *testing.T) {
	want := []string{".c++", ".cc", ".cjs", ".cpp", ".cs", ".css", ".cts", ".cxx", ".gd", ".go", ".godot", ".h", ".h++", ".hh", ".hpp", ".hxx", ".inl", ".ipp", ".java", ".js", ".jsx", ".mjs", ".mts", ".php", ".py", ".scss", ".tpp", ".tres", ".ts", ".tscn", ".tsx", ".uid"}
	first := sourceset.Extensions()
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("Extensions() = %v, want %v", first, want)
	}
	// A caller that edits its answer must not change what the walk takes.
	first[0] = ".txt"
	if again := sourceset.Extensions(); !reflect.DeepEqual(again, want) {
		t.Fatalf("Extensions() after editing an earlier answer = %v, want %v", again, want)
	}
}

func TestListMatchesTheExtensionExactly(t *testing.T) {
	root := tree(t, map[string]string{
		"lower.go":  "package a\n",
		"UPPER.GO":  "package a\n",
		"notes.gox": "no\n",
		"lower.py":  "x = 1\n",
		"UPPER.PY":  "x = 1\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// UPPER.GO is no Go file to the Go toolchain, so it is none to the graph
	// either, and UPPER.PY follows the same rule. extract/all's For applies
	// it too, so no file the walk lists is one no extractor claims.
	if !reflect.DeepEqual(got, []string{"lower.go", "lower.py"}) {
		t.Fatalf("got %v, want lower.go and lower.py alone", got)
	}
}

// C++ sources and headers are listed; a .c file is not, the graph leaves C alone.
func TestListTakesCppFilesAndNotC(t *testing.T) {
	root := tree(t, map[string]string{
		"src/a.cpp": "int a;\n", "src/a.cc": "int a;\n", "include/a.hpp": "int a;\n", "include/a.h": "int a;\n",
		"src/b.c": "int b;\n", "src/B.CPP": "int b;\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"include/a.h", "include/a.hpp", "src/a.cc", "src/a.cpp"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Java sources are listed; class files, archives and the build output are not.
func TestListTakesJavaFiles(t *testing.T) {
	root := tree(t, map[string]string{
		"src/main/java/a/App.java": "class App {}\n", "src/main/java/a/App.class": "x", "lib/a.jar": "x",
		"target/classes/a/Gen.java": "class Gen {}\n", "build/gen/a/Gen.java": "class Gen {}\n", "A.JAVA": "class A {}\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"src/main/java/a/App.java"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// PHP sources are listed; Composer's vendor/ and other extensions are not.
func TestListTakesPHPFiles(t *testing.T) {
	root := tree(t, map[string]string{
		"src/App.php": "<?php\n", "vendor/a/b/Lib.php": "<?php\n", "views/a.blade.php.txt": "x",
		"tpl/a.phtml": "<?php\n", "A.PHP": "<?php\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"src/App.php"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// JavaScript and TypeScript sources are listed, declaration files among
// them; npm's node_modules/, a build's dist/ and other extensions are not.
func TestListTakesJavaScriptAndTypeScriptFiles(t *testing.T) {
	root := tree(t, map[string]string{
		"src/a.js": "x\n", "src/b.mjs": "x\n", "src/c.cjs": "x\n", "src/D.jsx": "x\n",
		"src/e.ts": "x\n", "src/f.mts": "x\n", "src/g.cts": "x\n", "src/H.tsx": "x\n", "src/i.d.ts": "x\n",
		"node_modules/p/index.js": "x\n", "dist/app.js": "x\n", "src/a.js.map": "x", "src/App.vue": "x", "A.JS": "x",
		// A minified bundle is generated; min.js itself is a name like any.
		"static/d3.min.js": "x\n", "static/x.min.mjs": "x\n", "static/y.min.cjs": "x\n", "src/min.js": "x\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"src/D.jsx", "src/H.tsx", "src/a.js", "src/b.mjs", "src/c.cjs", "src/e.ts", "src/f.mts", "src/g.cts", "src/i.d.ts", "src/min.js"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// CSS and SCSS stylesheets are listed; a minified one and the reports of
// coverage.py and istanbul are generated and are not.
func TestListTakesStylesheets(t *testing.T) {
	root := tree(t, map[string]string{
		"a.css": "x\n", "web/_b.scss": "x\n", "a.min.css": "x\n", "a.sass": "x\n", "a.less": "x\n",
		"htmlcov/x.css": "x\n", "coverage/lcov-report/base.css": "x\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.css", "web/_b.scss"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
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
		// MSBuild's intermediate output is pruned; its bin/ is not, a bin/
		// holds scripts in many repositories.
		"Game/obj/Debug/Game.AssemblyInfo.cs": "class A {}\n",
		"Game/bin/Tool.cs":                    "class T {}\n",
		"Game/Player.cs":                      "class P {}\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// A dot directory is skipped wholesale; the skip list matches a single path
	// segment, so a directory merely NAMED like an output dir deeper in the
	// tree is not exempt -- but "internal/bin" is not on the list at all.
	want := map[string]bool{"keep.go": true, "internal/bin/keep.go": true, "Game/bin/Tool.cs": true, "Game/Player.cs": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want the four keepers", got)
	}
	for _, rel := range got {
		if !want[rel] {
			t.Errorf("unexpected file %q", rel)
		}
	}
}

// A virtualenv is named whatever its owner chose (env, py311, .venv), but
// every layout keeps the installed packages in a directory site-packages:
// lib/python3.x/site-packages on POSIX, Lib/site-packages on Windows.
func TestListSkipsSitePackages(t *testing.T) {
	root := tree(t, map[string]string{
		"app.py":                                  "x = 1\n",
		"env/Lib/site-packages/django/db.py":      "x = 1\n",
		"py311/lib/python3.11/site-packages/r.py": "x = 1\n",
		"env/Scripts/activate_this.py":            "x = 1\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	// The venv's own scripts are no installed package and stay; nothing below
	// site-packages does.
	if !reflect.DeepEqual(got, []string{"app.py", "env/Scripts/activate_this.py"}) {
		t.Fatalf("got %v, want app.py and the venv script alone", got)
	}
}

func TestListSkipsTestdata(t *testing.T) {
	// testdata holds fixtures Go's own toolchain does not build. On this
	// repository none of its testdata/ fixtures are .go files, and skipping
	// the 3 testdata directories here also skips the 1,826 directories below
	// them -- of 1,910 in the whole tree -- which is what a probe that still
	// walked testdata/ paid for (docs/en/benchmarks.md, 2026-09-18).
	root := tree(t, map[string]string{
		"keep.go":                 "package a\n",
		"testdata/cases/case.go":  "package case1\n",
		"pkg/testdata/fixture.go": "package fixture\n",
		// Python fixtures lie there too, some broken on purpose.
		"testdata/cases/broken.py": "def broken(:\n",
	})

	got, err := sourceset.List(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"keep.go": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want only keep.go", got)
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

func TestStatContinuesWhenADirectoryIsLocked(t *testing.T) {
	root := tree(t, map[string]string{
		"accessible.go":    "package main\n",
		"locked/hidden.go": "package hidden\n",
	})

	lockedDir := filepath.Join(root, "locked")
	// Lock the directory so WalkDir cannot read it and passes an error to the callback.
	// The callback returns nil to continue the walk and drops the inaccessible directory.
	// This exercises the err != nil arm of the callback.
	testlock.LockDir(t, lockedDir)

	got, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	// The locked directory is inaccessible during the walk, so it and its files are skipped.
	if len(got) != 1 || got[0].Rel != "accessible.go" {
		t.Fatalf("got %v, want only accessible.go", got)
	}
}

// TestSkipDirMatchesTheWalk pins the exported predicate sourceset's callers
// use: dot directories and the pruned names, nothing else.
func TestSkipDirMatchesTheWalk(t *testing.T) {
	for _, name := range []string{".git", ".loomux", "node_modules", "testdata", "dist", "vendor"} {
		if !sourceset.SkipDir(name) {
			t.Errorf("SkipDir(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"src", "docs", "internal", "main.go"} {
		if sourceset.SkipDir(name) {
			t.Errorf("SkipDir(%q) = true, want false", name)
		}
	}
}

// TestFromListingEqualsStat pins that the filter is the one Stat applies:
// pruned directories, the extension, the size limit and a pruned segment below
// the root all fall out of the same listing.
func TestFromListingEqualsStat(t *testing.T) {
	root := tree(t, map[string]string{
		"a.go":                      "package a\n",
		"sub/b.py":                  "x = 1\n",
		"node_modules/dep/dep.go":   "package dep\n",
		"vendor/lib/lib.go":         "package lib\n",
		"a.x.go.txt":                "not source\n",
		"big.go":                    "package big\n" + strings.Repeat("// x\n", 300_000),
		"sub/vendor/inner/inner.go": "package inner\n",
	})

	listing, err := corpus.Walk(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	want, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := sourceset.FromListing(listing); !reflect.DeepEqual(got, want) {
		t.Fatalf("FromListing = %v, Stat = %v", got, want)
	}
	if len(want) != 2 {
		t.Fatalf("Stat = %v, want only a.go and sub/b.py", want)
	}
}

// TestFromListingOfAMatcherWalkEqualsStat pins that a walk with the search
// index's ignore rules still yields the graph's set: the graph does not honour
// .gitignore, so an ignored source file stays in.
func TestFromListingOfAMatcherWalkEqualsStat(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	root := tree(t, map[string]string{
		".gitignore":    "gen/\nskip.go\n",
		"a.go":          "package a\n",
		"skip.go":       "package skip\n",
		"gen/out.go":    "package gen\n",
		"gen/other.txt": "text\n",
	})
	cmd := exec.Command("git", "-C", root, "init", "-q")
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	rules, _, err := corpus.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	rec := corpus.NewRecord(rules, nil, 0)

	listing, err := corpus.Walk(root, corpus.RulesFromRecord(rec))
	if err != nil {
		t.Fatal(err)
	}
	want, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	got := sourceset.FromListing(listing)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FromListing = %v, Stat = %v", got, want)
	}
	var rels []string
	for _, f := range got {
		rels = append(rels, f.Rel)
	}
	if !reflect.DeepEqual(rels, []string{"a.go", "gen/out.go", "skip.go"}) {
		t.Fatalf("got %v, want the ignored .go files kept", rels)
	}
}
