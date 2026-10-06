// Package sourceset is the file set a graph build looks at, and the stat
// metadata a freshness probe needs.
//
// It is a package of its own and not a part of the extractor, for the reason
// the original gives (src/graph/source-files.ts): the probe has to enumerate
// exactly the same files as the build, and importing the builder to learn them
// would tie the hook path to the parser. Two enumerations that can drift are
// two answers to "did anything change".
//
// The same reason keeps the list of extensions here rather than asking
// extract/all for it: all imports every language's extractor, and the probe
// runs on the hook path. The copy cannot drift unnoticed -- all's own test
// compares its Extensions with this package's.
//
// Ported from src/graph/source-files.ts and src/ingest/fs.ts (MIT; origin under
// "Ported sources" in NOTICE.md).
package sourceset

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// maxFileBytes is where a hand-written file stops. Above it a file is
// generated or vendored in practice; one repository indexed 4,484 files of
// vendored code against 121 written by hand, and queries slowed enough that a
// hook's budget could no longer hold them.
const maxFileBytes = 1_000_000

// skipDirs are dependency output, build output, and fixture input Go's own
// toolchain does not build -- not necessarily "never a symbol the graph
// should have a node for" in general, since go/parser reads a .go file under
// testdata/ as readily as any other. The comparison is against a single path
// segment.
//
// testdata is the third kind, not the first two: it holds fixtures, not
// dependency or build output, and go/build ignores it for builds by
// convention rather than by content. On THIS repository none of its fixtures
// are .go files (they are .go.txt or recorded case corpora) and its .py
// files are the worlds of recorded cases, not code of this repository, so
// skipping it costs nothing here -- but a repository whose testdata/ held
// real Go it wanted indexed would need this entry removed -- the same
// argument the design spec (§7.1) makes for keeping _test.go files in the
// graph rather than excluding them for looking like a test. A probe that
// still walked testdata/ paid for 1,826 directories below the 3 named
// testdata roots, out of 1,910 directories in this repository's tree -- the
// walk cost stayed on the probe until this entry named it. See
// docs/en/benchmarks.md, 2026-09-18.
//
// venv names one virtualenv; site-packages names the installed packages of
// every one, whatever the environment is called (env/Lib/site-packages on
// Windows, <env>/lib/python3.x/site-packages on POSIX). A dot directory such
// as .venv is skipped anyway.
var skipDirs = map[string]bool{
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

// extensions are the extensions of every extracted language, dot included,
// matched exactly: x.GO is no Go file to the Go toolchain, so it is none to
// the graph either. .godot, .tres and .tscn are the project file, resources and
// scenes of a Godot project, read with its scripts. A literal and nothing
// parsed: see the package comment for why it is not taken from extract/all.
var extensions = []string{".gd", ".go", ".godot", ".py", ".tres", ".tscn"}

// Extensions is a copy of the extensions the walk takes, so no caller can
// change what the next walk looks at.
func Extensions() []string {
	return slices.Clone(extensions)
}

// SourceFile is one file of the set, with what a freshness probe compares.
type SourceFile struct {
	Abs   string
	Rel   string
	Size  int64
	MTime int64
}

// List returns the repo-relative paths of the source files of every extracted
// language a build looks at, sorted in byte order.
func List(root string) ([]string, error) {
	files, err := Stat(root)
	if err != nil {
		return nil, err
	}
	rels := make([]string, 0, len(files))
	for _, f := range files {
		rels = append(rels, f.Rel)
	}
	return rels, nil
}

// Stat returns the file set with each file's size and modification time.
//
// A file that vanishes between the walk and the stat is dropped rather than
// reported: the next probe will see the same thing and call it removed.
//
//coverage:exempt d.Info() error arm requires file to vanish between walk and stat; filepath.Rel error arm is unreachable (WalkDir only returns paths under root)
func Stat(root string) ([]SourceFile, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	var out []SourceFile
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !slices.Contains(extensions, filepath.Ext(d.Name())) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFileBytes {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		out = append(out, SourceFile{
			Abs:   path,
			Rel:   filepath.ToSlash(rel),
			Size:  info.Size(),
			MTime: info.ModTime().UnixNano(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rel < out[j].Rel })
	return out, nil
}

// SkipDir reports whether a directory of this name is walked.
//
// Every dot directory is skipped wholesale -- .git, .github, .loomux and the
// state the graph itself writes into it. That last one matters: a graph that
// indexed its own output would grow on every build.
//
// Exported so every directory walk this package's callers run agrees with
// this one on what counts as source. internal/code/query's goModPaths used to
// keep its own, narrower list (dot-directories and vendor only), which let a
// fixture go.mod under testdata/ reach module resolution while the .go files
// beside it were already excluded from the file set -- two walks of the same
// tree, two different answers about testdata. skipDirs itself stays
// unexported: only this predicate is anyone else's business.
func SkipDir(name string) bool {
	return strings.HasPrefix(name, ".") || skipDirs[name]
}
