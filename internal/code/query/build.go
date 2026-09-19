// Package query is the graph's question layer: build, check and ask, with the
// answers as text a terminal and a tool call both carry.
//
// It exists so the command line and serve reach the same code. Before it, the
// build sequence lived in internal/cli, where serve cannot import it; a second
// copy there would be a second place to forget the sidecar.
package query

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/lexicon"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/code/store"
)

// Stats is what a build learned on the way, for the report and for the
// freshness record.
type Stats struct {
	Files    []sourceset.SourceFile
	Hashes   map[string]string
	NoSymbol int
}

// Build is one whole build: the graph, the ask sidecar and the freshness
// record.
//
// Every caller uses it -- `graph build`, and the rebuild a query triggers, on
// the command line and over MCP. Two copies of this sequence would be two
// places where the sidecar can be forgotten, and a forgotten sidecar is a
// silently worse answer.
//
// The two writes a question reads are errors; the record is not. Once the graph
// and the sidecar are on disk, a record that could not be written costs the
// next probe its fast path and nothing more -- so it is announced through
// notice and the build stands. Turning it into a failure would let a build that
// produced everything a question needs report that it produced nothing.
func Build(root string, notice func(string)) (*model.Graph, Stats, error) {
	g, stats, err := Extract(root)
	if err != nil {
		return nil, Stats{}, err
	}
	if err := store.Write(root, g); err != nil {
		return nil, Stats{}, err
	}
	// From the graph in memory, which still carries the body text: the written
	// file has it stripped, and that is what makes the sidecar necessary rather
	// than redundant.
	if err := lexicon.Write(root, lexicon.Build(g)); err != nil {
		return nil, Stats{}, err
	}
	if err := freshness.Write(root, golang.Version, stats.Files, stats.Hashes); err != nil {
		notice(fmt.Sprintf("freshness record not written: %v", err))
	}
	return g, stats, nil
}

// Extract reads, parses and resolves the whole tree.
//
// It reads and hashes every file, every time -- never the probe's stat fast
// path. A stat may decide whether a query rebuilds; it may not decide what the
// rebuild looks at, or `check` could report drift that the `build` it
// recommends refuses to repair.
func Extract(root string) (*model.Graph, Stats, error) {
	files, err := sourceset.Stat(root)
	if err != nil {
		return nil, Stats{}, err
	}
	stats := Stats{Files: files, Hashes: map[string]string{}}

	var results []golang.Result
	for _, f := range files {
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			return nil, Stats{}, err
		}
		sum := sha256.Sum256(b)
		stats.Hashes[f.Rel] = hex.EncodeToString(sum[:])

		r, err := golang.File(f.Rel, string(b))
		if err != nil {
			return nil, Stats{}, err
		}
		if len(r.Nodes) == 1 {
			stats.NoSymbol++
		}
		results = append(results, r)
	}

	mods, err := resolve.Modules(root, goModPaths(root))
	if err != nil {
		return nil, Stats{}, err
	}
	return resolve.Graph(results, mods), stats, nil
}

// goModPaths are the repo-relative go.mod files, found by walking for them --
// sourceset lists Go sources and a go.mod is not one. This walk shares
// sourceset.SkipDir with sourceset.Stat rather than keeping its own,
// narrower list: the two used to disagree about testdata, which let a
// fixture go.mod there reach module resolution after its .go files had
// already been excluded from the file set.
func goModPaths(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && sourceset.SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}
