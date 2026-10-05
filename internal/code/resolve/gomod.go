// Package resolve turns the raw edges of every extracted file into edges with
// node ids and a confidence, and drops what stays ambiguous.
//
// The two-tier provenance is the original's (src/graph/resolve.ts):
//
//   - extracted -- the target is certain: a hit in the same file, an import
//     specifier, structural containment, or a package selector resolved
//     against the one package it can mean
//   - inferred  -- a bare name resolved through exactly one match across files
//
// Ambiguous means dropped, never guessed. The original's own header says why:
// name guessing "halved precision", and one same-named symbol once collected
// 1040 in-edges across 476 files, so every pull request touching it dragged a
// whole backend into its blast radius.
//
// Ported from src/graph/resolve.ts (MIT; origin under
// "Ported sources" in NOTICE.md).
package resolve

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Module is one go.mod of the repository.
type Module struct {
	Dir  string
	Path string
}

// Modules reads the module directive of every go.mod among files.
//
// Only that directive, by hand: golang.org/x/mod would parse the whole file and
// is a dependency this stage does not take. A go.mod without a module
// directive declares nothing and is skipped.
func Modules(root string, files []string) ([]Module, error) {
	var out []Module
	for _, rel := range files {
		if path.Base(rel) != "go.mod" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		mod, ok := moduleDirective(string(b))
		if !ok {
			continue
		}
		dir := path.Dir(rel)
		if dir == "." {
			dir = ""
		}
		out = append(out, Module{Dir: dir, Path: mod})
	}
	// Longest path first, so resolveImport can take the first match and have
	// the most specific module.
	sort.Slice(out, func(i, j int) bool { return len(out[i].Path) > len(out[j].Path) })
	return out, nil
}

// moduleDirective is the argument of the first `module` line.
func moduleDirective(body string) (string, bool) {
	for _, line := range strings.Split(body, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1], true
		}
	}
	return "", false
}

// importDir is the repo-relative directory an import path names.
//
// The bool is not decoration: the root package of a root module legitimately
// lives at "", and "" is also what a caller would use for "not found". Returning
// both apart is what keeps `import "example.com/repo"` from reading as the
// standard library. The original has the same distinction and spells it with a
// sentinel; in Go the pair is the honest form.
//
// The longest module path wins, so a nested module in a monorepo beats its
// parent.
func importDir(spec string, mods []Module) (string, bool) {
	for _, m := range mods {
		var sub string
		switch {
		case spec == m.Path:
			sub = ""
		case strings.HasPrefix(spec, m.Path+"/"):
			sub = spec[len(m.Path)+1:]
		default:
			continue
		}
		switch {
		case m.Dir == "":
			return sub, true
		case sub == "":
			return m.Dir, true
		default:
			return m.Dir + "/" + sub, true
		}
	}
	return "", false
}

// dirOf is the directory a node's path lies in, with path.Dir's "." for a file
// at the repository root spelled as "" -- the same form importDir returns, so
// the two can be compared at all.
func dirOf(p string) string {
	dir := path.Dir(p)
	if dir == "." {
		return ""
	}
	return dir
}
