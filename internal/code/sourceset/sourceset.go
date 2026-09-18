// Package sourceset is the file set a graph build looks at, and the stat
// metadata a freshness probe needs.
//
// It is a package of its own and not a part of the extractor, for the reason
// Graft gives (src/graph/source-files.ts): the probe has to enumerate exactly
// the same files as the build, and importing the builder to learn them would
// tie the hook path to the parser. Two enumerations that can drift are two
// answers to "did anything change".
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/graph/source-files.ts and
// src/ingest/fs.ts.
package sourceset

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxFileBytes is where a hand-written file stops. Above it a file is
// generated or vendored in practice; one repository indexed 4,484 files of
// vendored code against 121 written by hand, and queries slowed enough that a
// hook's budget could no longer hold them.
const maxFileBytes = 1_000_000

// skipDirs are dependency and build output, never source. The comparison is
// against a single path segment.
var skipDirs = map[string]bool{
	"node_modules": true,
	"dist":         true,
	"build":        true,
	"_build":       true,
	"out":          true,
	"target":       true,
	"vendor":       true,
	"coverage":     true,
	"__pycache__":  true,
	"venv":         true,
}

// SourceFile is one file of the set, with what a freshness probe compares.
type SourceFile struct {
	Abs   string
	Rel   string
	Size  int64
	MTime int64
}

// List returns the repo-relative paths of the Go files a build looks at,
// sorted in byte order.
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
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") {
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

// skipDir reports whether a directory of this name is walked.
//
// Every dot directory is skipped wholesale -- .git, .github, .loomux and the
// state the graph itself writes into it. That last one matters: a graph that
// indexed its own output would grow on every build.
func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || skipDirs[name]
}
