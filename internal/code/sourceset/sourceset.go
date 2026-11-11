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
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/code/corpus"
)

// maxFileBytes is where a hand-written file stops. Above it a file is
// generated or vendored in practice; one repository indexed 4,484 files of
// vendored code against 121 written by hand, and queries slowed enough that a
// hook's budget could no longer hold them.
const maxFileBytes = 1_000_000

// extensions are the extensions of every extracted language, dot included,
// matched exactly: x.GO is no Go file to the Go toolchain, so it is none to
// the graph either. .cs is C#. .godot, .tres and .tscn are the project file, resources and
// scenes of a Godot project, read with its scripts, and .uid the uid Godot 4.4
// keeps beside a script. A literal and nothing
// parsed: see the package comment for why it is not taken from extract/all.
var extensions = []string{".cs", ".gd", ".go", ".godot", ".py", ".tres", ".tscn", ".uid"}

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
// The enumeration is corpus.Walk's, the same walk the search index uses; the
// filter (FromListing) is the graph's: eight extensions, one megabyte, and no
// file below a pruned directory segment. A dot directory never reaches the
// filter, because the walk does not enter one.
//
// A file that vanishes between the walk and the stat is dropped rather than
// reported: the next probe will see the same thing and call it removed.
//
// It is a variable so a test of another package can count the graph's walks
// of one call.
var Stat = stat

func stat(root string) ([]SourceFile, error) {
	listing, err := corpus.Walk(root, nil)
	if err != nil {
		return nil, err
	}
	return FromListing(listing), nil
}

// FromListing is Stat's filter over a walk the caller already ran, for a
// caller whose walk serves another probe too. The listing may come from a walk
// with ignore rules: the graph does not honour them, so an Entry's Ignored
// flag plays no part here.
func FromListing(l corpus.Listing) []SourceFile {
	out := make([]SourceFile, 0, len(l.Entries))
	for _, e := range l.Entries {
		if !slices.Contains(extensions, filepath.Ext(e.Abs)) {
			continue
		}
		if e.Size > maxFileBytes {
			continue
		}
		if hasPrunedSegment(e.Rel) {
			continue
		}
		out = append(out, SourceFile{Abs: e.Abs, Rel: e.Rel, Size: e.Size, MTime: e.MTime})
	}
	return out
}

// hasPrunedSegment reports whether a repo-relative path has a directory
// segment the graph never descends into. Only pruneDirs count: a dot directory
// is already absent from the walk, and a file named like one is a file.
func hasPrunedSegment(rel string) bool {
	for _, segment := range strings.Split(rel, "/") {
		if corpus.PrunedDir(segment) {
			return true
		}
	}
	return false
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
// tree, two different answers about testdata. The list itself lives in
// corpus.pruneDirs and is shared with the search index's walk; only this
// predicate is anyone else's business.
func SkipDir(name string) bool {
	return corpus.SkipSegment(name)
}
