package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/resolve"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/code/store"
)

// graphCommand is `loomux graph`: the two commands of stage G2a.
func graphCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		graphUsage(stderr)
		return 2
	}
	switch args[0] {
	case "build":
		return graphBuild(args[1:], stdout, stderr)
	case "check":
		return graphCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "loomux graph: unknown command %q\n", args[0])
		graphUsage(stderr)
		return 2
	}
}

func graphUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: loomux graph <command> [arguments]")
	fmt.Fprintln(w, "  build   analyse the source and write .loomux/state/graph/wiring.json")
	fmt.Fprintln(w, "  check   report whether the graph still matches the code")
}

// graphBuild writes the graph and reports what it wrote.
func graphBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}

	started := time.Now()
	g, stats, err := buildGraph(project)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}
	if err := store.Write(project, g); err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}
	if err := freshness.Write(project, golang.Version, stats.files, stats.hashes); err != nil {
		// The graph is on disk already, so a failed record costs the next probe
		// its fast path and nothing more.
		fmt.Fprintf(stderr, "loomux graph build: freshness record not written: %v\n", err)
	}
	fmt.Fprint(stdout, report(g, stats, time.Since(started)))
	return 0
}

// graphCheck is Task 9.
func graphCheck(args []string, stdout, stderr io.Writer) int { return 2 }

// buildStats is what a build learned on the way, for the report and for the
// freshness record.
type buildStats struct {
	files    []sourceset.SourceFile
	hashes   map[string]string
	noSymbol int
}

// buildGraph reads, parses and resolves the whole tree.
//
// It reads and hashes every file, every time -- never the probe's stat fast
// path. A stat may decide whether a query rebuilds; it may not decide what the
// rebuild looks at, or `check` could report drift that the `build` it
// recommends refuses to repair.
func buildGraph(root string) (*model.Graph, buildStats, error) {
	files, err := sourceset.Stat(root)
	if err != nil {
		return nil, buildStats{}, err
	}
	stats := buildStats{files: files, hashes: map[string]string{}}

	var results []golang.Result
	for _, f := range files {
		b, err := os.ReadFile(f.Abs)
		if err != nil {
			return nil, buildStats{}, err
		}
		sum := sha256.Sum256(b)
		stats.hashes[f.Rel] = hex.EncodeToString(sum[:])

		r, err := golang.File(f.Rel, string(b))
		if err != nil {
			return nil, buildStats{}, err
		}
		if len(r.Nodes) == 1 {
			stats.noSymbol++
		}
		results = append(results, r)
	}

	mods, err := resolve.Modules(root, goModPaths(root))
	if err != nil {
		return nil, buildStats{}, err
	}
	return resolve.Graph(results, mods), stats, nil
}

// goModPaths are the repo-relative go.mod files, found by walking for them --
// sourceset lists Go sources and a go.mod is not one.
func goModPaths(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor") {
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

// report is the count line after a build: what the extractor found, so a
// reader can see at a glance whether it found the right kind of thing.
func report(g *model.Graph, stats buildStats, took time.Duration) string {
	byRelation := map[model.Relation]int{}
	unresolved := 0
	nodes := map[model.NodeID]bool{}
	for _, n := range g.Nodes {
		nodes[n.ID] = true
	}
	for _, e := range g.Edges {
		byRelation[e.Relation]++
		if !nodes[e.Target] {
			unresolved++
		}
	}
	return fmt.Sprintf(
		"%d files, %d nodes, %d edges (%d contains, %d calls, %d imports)\n"+
			"%d unresolved import targets, %d files without a symbol, %s\n",
		len(stats.files), len(g.Nodes), len(g.Edges),
		byRelation[model.RelationContains], byRelation[model.RelationCalls],
		byRelation[model.RelationImports],
		unresolved, stats.noSymbol, took.Round(time.Millisecond),
	)
}
