package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/lexicon"
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
	case "ask":
		return graphAsk(args[1:], stdout, stderr)
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
	fmt.Fprintln(w, "  ask     answer a question from the graph")
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
	g, stats, err := writeEverything(project)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, report(g, stats, time.Since(started)))
	return 0
}

// graphAsk answers a question from the graph.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and ask.Answer is built entirely of strings, floats and model types -- no ask.Answer this program can construct makes it fail
func graphAsk(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph ask", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	limit := fs.Int("limit", 0, "how many hits to report (default 8)")
	in := fs.String("in", "", "narrow to nodes under this path prefix, before scoring")
	source := fs.Bool("source", false, "inline the source at each hit")
	full := fs.Bool("full", false, "with --source: inline the whole span, uncapped")
	asJSON := fs.Bool("json", false, "write the answer as JSON")
	noRefresh := fs.Bool("no-refresh", false, "never rebuild, answer from the graph on disk")
	var queries []string
	for rest := args; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		queries = append(queries, rest[0])
	}
	if len(queries) != 1 {
		fmt.Fprintln(stderr, `loomux graph ask: one query required: loomux graph ask "<question>" [flags]`)
		return 2
	}
	query := queries[0]

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
		return 1
	}

	if !*noRefresh {
		// Notices go to stderr, so a piped answer stays an answer.
		ask.EnsureFresh(project, golang.Version,
			func() error { _, _, err := writeEverything(project); return err },
			func(s string) { fmt.Fprintf(stderr, "loomux graph ask: %s\n", s) })
	}

	g, err := store.Read(project)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(stderr, "loomux graph ask: no graph. Run `loomux graph build` first.")
			return 1
		}
		fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
		return 1
	}
	ix, err := lexicon.Read(project)
	if err != nil {
		// The sidecar is a cache: without it, tokenize live off the graph. The
		// body text is gone from the written graph, so a body-only word will
		// not be found -- say so rather than answer worse in silence.
		fmt.Fprintf(stderr, "loomux graph ask: no ask index, ranking on names and paths only: %v\n", err)
		ix = lexicon.Build(g)
	}

	answer := ask.Run(g, ix, query, ask.Options{Limit: *limit, In: *in})
	if *source {
		ask.Inline(project, &answer, *full)
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, askReport(answer))
	return 0
}

// writeEverything is one whole build: the graph, the ask sidecar and the
// freshness record.
//
// Both callers use it -- `graph build`, which owns the flags and the report, and
// the rebuild `graph ask` triggers, which owns neither. Two copies of this
// sequence would be two places where the sidecar can be forgotten, and a
// forgotten sidecar is a silently worse answer.
func writeEverything(root string) (*model.Graph, buildStats, error) {
	g, stats, err := buildGraph(root)
	if err != nil {
		return nil, buildStats{}, err
	}
	if err := store.Write(root, g); err != nil {
		return nil, buildStats{}, err
	}
	// From the graph in memory, which still carries the body text: the written
	// file has it stripped, and that is what makes the sidecar necessary rather
	// than redundant.
	if err := lexicon.Write(root, lexicon.Build(g)); err != nil {
		return nil, buildStats{}, err
	}
	if err := freshness.Write(root, golang.Version, stats.files, stats.hashes); err != nil {
		return nil, buildStats{}, err
	}
	return g, stats, nil
}

// askReport is the human form: one block per hit, location first.
func askReport(a ask.Answer) string {
	if a.Note != "" {
		return a.Note + "\n"
	}
	var b strings.Builder
	for i, h := range a.Hits {
		fmt.Fprintf(&b, "%d. %s  %s:%s  (%.3f lex %.3f graph %.3f)\n",
			i+1, h.ID, h.Path, h.Span, h.Score, h.Lexical, h.Graph)
		if h.Signature != "" {
			fmt.Fprintf(&b, "   %s\n", h.Signature)
		}
		if h.Code != "" {
			for _, line := range strings.Split(h.Code, "\n") {
				fmt.Fprintf(&b, "   | %s\n", line)
			}
		}
	}
	return b.String()
}

// checkResult is what `graph check` found.
type checkResult struct {
	OK       bool     `json:"ok"`
	Missing  bool     `json:"missing"`
	Foreign  string   `json:"foreign,omitempty"`
	Outdated bool     `json:"outdated,omitempty"`
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Changed  []string `json:"changed"`
}

// graphCheck re-extracts the tree and diffs it against the written graph.
//
// It does NOT read the freshness record. That sidecar answers "should a query
// bother rebuilding"; this command answers "does the graph still describe the
// code", and the only honest way to answer it is to extract again. It is also
// why a bare `touch` leaves this command at 0.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and checkResult is built entirely of bools, strings and string slices -- no checkResult this program can construct makes it fail
func graphCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	asJSON := fs.Bool("json", false, "write the drift as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph check: %v\n", err)
		return 1
	}

	res, err := check(project)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph check: %v\n", err)
		return 1
	}
	if *asJSON {
		body, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph check: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
	} else {
		fmt.Fprint(stdout, checkReport(res))
	}
	if res.OK {
		return 0
	}
	return 1
}

// check compares a fresh extraction against the graph on disk.
func check(root string) (checkResult, error) {
	written, err := store.Read(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return checkResult{Missing: true}, nil
		}
		if errors.Is(err, model.ErrSchemaVersion) {
			// A schema-1 graph has no extractor stamp to check next, so this
			// has to be caught before that comparison and given the same
			// pointer to a fix as every other unusable-graph case.
			return checkResult{Outdated: true}, nil
		}
		return checkResult{}, err
	}
	// The stamp decides before a single node is compared: a graph from another
	// extractor is not stale, it is foreign, and diffing it node by node would
	// report the whole repository.
	if written.Meta.Extractor != golang.Version {
		return checkResult{Foreign: written.Meta.Extractor}, nil
	}

	fresh, _, err := buildGraph(root)
	if err != nil {
		return checkResult{}, err
	}

	was := map[model.NodeID]string{}
	for _, n := range written.Nodes {
		was[n.ID] = n.BodyHash
	}
	res := checkResult{}
	now := map[model.NodeID]bool{}
	for _, n := range fresh.Nodes {
		now[n.ID] = true
		hash, known := was[n.ID]
		switch {
		case !known:
			res.Added = append(res.Added, string(n.ID))
		case hash != n.BodyHash:
			res.Changed = append(res.Changed, string(n.ID))
		}
	}
	for id := range was {
		if !now[id] {
			res.Removed = append(res.Removed, string(id))
		}
	}
	sort.Strings(res.Added)
	sort.Strings(res.Removed)
	sort.Strings(res.Changed)
	res.OK = len(res.Added)+len(res.Removed)+len(res.Changed) == 0
	return res, nil
}

// checkReport is the human form.
func checkReport(res checkResult) string {
	switch {
	case res.Missing:
		return "loomux graph check: NO GRAPH\n\nNothing built yet. Run `loomux graph build` first.\n"
	case res.Foreign != "":
		return fmt.Sprintf(
			"loomux graph check: FOREIGN GRAPH\n\nThe graph was written by extractor %q, this binary is %q.\nRun `loomux graph build`.\n",
			res.Foreign, golang.Version)
	case res.Outdated:
		return "loomux graph check: OUTDATED GRAPH\n\nThe graph on disk predates this binary's schema.\nRun `loomux graph build`.\n"
	case res.OK:
		return "loomux graph check: OK\n"
	}
	out := "loomux graph check: DRIFT\n\n"
	for _, group := range []struct {
		label string
		ids   []string
	}{
		{"added", res.Added}, {"removed", res.Removed}, {"changed", res.Changed},
	} {
		for _, id := range group.ids {
			out += fmt.Sprintf("  %-8s %s\n", group.label, id)
		}
	}
	return out + "\nRun `loomux graph build`.\n"
}

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
		if e.Relation == model.RelationImports && !nodes[e.Target] {
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
