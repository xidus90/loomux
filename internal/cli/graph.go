package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/grep"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
	"github.com/xidus90/loomux/internal/code/repomap"
)

// graphCommand is `loomux graph`: code graph inspection and querying.
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
	case "callers":
		return graphCallers(args[1:], stdout, stderr)
	case "blast":
		return graphBlast(args[1:], stdout, stderr)
	case "skeleton":
		return graphSkeleton(args[1:], stdout, stderr)
	case "grep":
		return graphGrep(args[1:], stdout, stderr)
	case "map":
		return graphMap(args[1:], stdout, stderr)
	case "stats":
		return graphStats(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "loomux graph: unknown command %q\n", args[0])
		graphUsage(stderr)
		return 2
	}
}

func graphUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: loomux graph <command> [arguments]")
	fmt.Fprintln(w, "  build     analyse the source and write .loomux/state/graph/wiring.json")
	fmt.Fprintln(w, "  check     report whether the graph still matches the code")
	fmt.Fprintln(w, "  ask       answer a question from the graph")
	fmt.Fprintln(w, "  callers   trace callers or callees of a symbol")
	fmt.Fprintln(w, "  blast     show what a change reaches, from git's diff")
	fmt.Fprintln(w, "  skeleton  show API and signatures of a file")
	fmt.Fprintln(w, "  grep      search code with symbol grouping and ranking")
	fmt.Fprintln(w, "  map       generate repository map with hubs and hotspots")
	fmt.Fprintln(w, "  stats     print graph metrics, languages, and relations")
}

// graphBuild writes the graph and reports what it wrote.
func graphBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	noReuse := fs.Bool("no-reuse", false, "parse every file, ignoring the extract cache")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph build: %v\n", err)
		return 1
	}

	started := time.Now()
	g, stats, err := query.BuildWith(project, query.BuildOptions{NoReuse: *noReuse},
		func(s string) { fmt.Fprintf(stderr, "loomux graph build: %s\n", s) })
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
	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
		return 1
	}

	answer, notes, err := query.Ask(project, queries[0], query.AskOptions{
		Limit: *limit, In: *in, Source: *source, Full: *full, NoRefresh: *noRefresh,
	})
	// Notices go to stderr, so a piped answer stays an answer.
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux graph ask: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph ask: %v\n", err)
		return 1
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
	fmt.Fprint(stdout, query.AskReport(answer))
	return 0
}

// graphCheck re-extracts the tree and diffs it against the written graph.
//
// It does NOT read the freshness record. That sidecar answers "should a query
// bother rebuilding"; this command answers "does the graph still describe the
// code", and the only honest way to answer it is to extract again. It is also
// why a bare `touch` leaves this command at 0.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and query.Drift exports only bools, strings, string slices and an int; its one unexported field, a map of strings, json never sees -- no query.Drift this program can construct makes it fail
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

	res, err := query.Check(project)
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
		fmt.Fprint(stdout, query.CheckReport(res))
	}
	if res.OK {
		return 0
	}
	return 1
}

// report is the count lines after a build: what the extractor found, so a
// reader can see at a glance whether it found the right kind of thing, then
// one line per language with what was parsed, what the extract cache gave,
// and the files a tolerant parser had to skip parts of.
func report(g *model.Graph, stats query.Stats, took time.Duration) string {
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
	// Only Python has base classes; a graph without an extends edge reads as
	// the Go graph's report always did.
	extends := ""
	if n := byRelation[model.RelationExtends]; n > 0 {
		extends = fmt.Sprintf(", %d extends", n)
	}
	out := fmt.Sprintf(
		"%d files, %d nodes, %d edges (%d contains, %d calls, %d imports%s)\n"+
			"%d unresolved import targets, %d files without a symbol, %s\n",
		len(stats.Files), len(g.Nodes), len(g.Edges),
		byRelation[model.RelationContains], byRelation[model.RelationCalls],
		byRelation[model.RelationImports], extends,
		unresolved, stats.NoSymbol, took.Round(time.Millisecond),
	)
	for _, name := range slices.Sorted(maps.Keys(stats.PerLanguage)) {
		ls := stats.PerLanguage[name]
		out += fmt.Sprintf("  %s: %d files, %d parsed, %d reused, %d parse errors\n",
			name, ls.Files, ls.Parsed, ls.Reused, ls.ParseErrors)
		if len(ls.ErrorFiles) == 0 {
			continue
		}
		// Five names say where to look; the whole list of a repository full
		// of broken files would bury the count lines above it.
		shown, more := ls.ErrorFiles, ""
		if len(shown) > 5 {
			shown, more = shown[:5], fmt.Sprintf(" (+%d more)", len(shown)-5)
		}
		out += fmt.Sprintf("  %s parse errors in: %s%s\n", name, strings.Join(shown, ", "), more)
	}
	return out
}

// graphCallers traces callers or callees of a symbol.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and query.CallersAnswer is built entirely of strings, ints and model types -- no query.CallersAnswer this program can construct makes it fail
func graphCallers(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph callers", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	dirStr := fs.String("direction", "in", "direction to walk: 'in' (callers) or 'out' (callees)")
	depthStr := fs.String("d", "1", "depth limit: positive integer, or 'all' or 'full' for the closure")
	fs.StringVar(depthStr, "depth", "1", "alias for -d")
	in := fs.String("in", "", "narrow to nodes under this path prefix")
	noRefresh := fs.Bool("no-refresh", false, "never rebuild, answer from the graph on disk")
	asJSON := fs.Bool("json", false, "write the answer as JSON")

	var symbols []string
	for rest := args; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		symbols = append(symbols, rest[0])
	}
	if len(symbols) != 1 {
		fmt.Fprintln(stderr, `loomux graph callers: one symbol required: loomux graph callers "<symbol>" [flags]`)
		return 2
	}
	var dir blast.Direction
	switch *dirStr {
	case "in":
		dir = blast.In
	case "out":
		dir = blast.Out
	default:
		fmt.Fprintf(stderr, "loomux graph callers: --direction must be 'in' or 'out', got %q\n", *dirStr)
		return 2
	}
	var dep blast.Depth
	if strings.EqualFold(*depthStr, "all") || strings.EqualFold(*depthStr, "full") {
		dep = blast.All
	} else if n, err := strconv.Atoi(*depthStr); err == nil && n > 0 {
		dep = blast.Depth(n)
	} else {
		fmt.Fprintf(stderr, "loomux graph callers: -d/--depth must be a positive integer, 'all' or 'full', got %q\n", *depthStr)
		return 2
	}

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph callers: %v\n", err)
		return 1
	}

	answer, notes, err := query.Callers(project, symbols[0], query.CallersOptions{
		Direction: dir,
		Depth:     dep,
		In:        *in,
		NoRefresh: *noRefresh,
	})
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux graph callers: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph callers: %v\n", err)
		return 1
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph callers: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, query.CallersReport(answer))
	return 0
}

// graphSkeleton displays API signatures for a file.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and query.SkeletonAnswer is built entirely of strings, bools, and model types -- no query.SkeletonAnswer this program can construct makes it fail
func graphSkeleton(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph skeleton", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	noRefresh := fs.Bool("no-refresh", false, "never rebuild, answer from the graph on disk")
	asJSON := fs.Bool("json", false, "write the answer as JSON")

	var files []string
	for rest := args; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		files = append(files, rest[0])
	}
	if len(files) != 1 {
		fmt.Fprintln(stderr, `loomux graph skeleton: one file required: loomux graph skeleton "<file>" [flags]`)
		return 2
	}

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph skeleton: %v\n", err)
		return 1
	}

	answer, notes, err := query.Skeleton(project, files[0], query.SkeletonOptions{
		NoRefresh: *noRefresh,
	})
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux graph skeleton: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph skeleton: %v\n", err)
		return 1
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph skeleton: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, query.SkeletonReport(answer))
	return 0
}

// graphGrep searches the codebase with symbol grouping and ranking.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and query.GrepAnswer is built entirely of strings, ints and model types -- no query.GrepAnswer this program can construct makes it fail
func graphGrep(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph grep", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	ignoreCase := fs.Bool("i", false, "case-insensitive search")
	fs.BoolVar(ignoreCase, "ignore-case", false, "alias for -i")
	fixed := fs.Bool("fixed", false, "treat pattern as literal string instead of regex")
	in := fs.String("in", "", "narrow to files under this path prefix")
	maxHits := fs.Int("max-hits", grep.DefaultMaxHits, "maximum number of hits to report")
	noRefresh := fs.Bool("no-refresh", false, "never rebuild, answer from the graph on disk")
	asJSON := fs.Bool("json", false, "write the answer as JSON")

	var patterns []string
	for rest := args; ; rest = rest[1:] {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		patterns = append(patterns, rest[0])
	}
	if len(patterns) != 1 {
		fmt.Fprintln(stderr, `loomux graph grep: one pattern required: loomux graph grep "<pattern>" [flags]`)
		return 2
	}

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph grep: %v\n", err)
		return 1
	}

	answer, notes, err := query.Grep(project, patterns[0], query.GrepOptions{
		IgnoreCase: *ignoreCase,
		Fixed:      *fixed,
		In:         *in,
		MaxHits:    *maxHits,
		NoRefresh:  *noRefresh,
	})
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux graph grep: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph grep: %v\n", err)
		return 1
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph grep: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, query.GrepReport(answer))
	return 0
}

// graphMap builds a repository map with clusters, hubs, and hotspots.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and query.MapAnswer is built entirely of strings, ints and model types -- no query.MapAnswer this program can construct makes it fail
func graphMap(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph map", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	maxDirs := fs.Int("max-dirs", repomap.DefaultMaxDirs, "maximum number of directories to list")
	hubsPerDir := fs.Int("hubs-per-dir", repomap.DefaultHubsPerDir, "maximum hubs per directory")
	hotspots := fs.Int("hotspots", repomap.DefaultHotspots, "maximum hotspots to list")
	noRefresh := fs.Bool("no-refresh", false, "never rebuild, answer from the graph on disk")
	asJSON := fs.Bool("json", false, "write the answer as JSON")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "loomux graph map: takes no positional arguments, got %v\n", fs.Args())
		return 2
	}

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph map: %v\n", err)
		return 1
	}

	answer, notes, err := query.Map(project, query.MapOptions{
		MaxDirs:    *maxDirs,
		HubsPerDir: *hubsPerDir,
		Hotspots:   *hotspots,
		NoRefresh:  *noRefresh,
	})
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux graph map: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph map: %v\n", err)
		return 1
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph map: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, query.MapReport(answer))
	return 0
}

// graphStats prints graph node/edge counts, languages, and wiring size.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and query.StatsAnswer is built entirely of primitives and maps -- no query.StatsAnswer this program can construct makes it fail
func graphStats(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph stats", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	asJSON := fs.Bool("json", false, "write the answer as JSON")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "loomux graph stats: takes no positional arguments, got %v\n", fs.Args())
		return 2
	}

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph stats: %v\n", err)
		return 1
	}

	answer, err := query.GraphStats(project)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph stats: %v\n", err)
		return 1
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph stats: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, query.StatsReport(answer))
	return 0
}

// graphBlast shows what a change reaches, from git's diff.
//
//coverage:exempt the MarshalIndent arm needs a value json cannot encode, and query.BlastAnswer is built of strings, ints, slices and model types -- no query.BlastAnswer this program can construct makes it fail
func graphBlast(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph blast", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "project root; the working directory when empty")
	base := fs.String("base", "", "compare <base>...HEAD instead of the working tree")
	cached := fs.Bool("cached", false, "compare the index against HEAD")
	depthStr := fs.String("d", "1", "depth limit: positive integer, or 'all' or 'full' for the closure")
	fs.StringVar(depthStr, "depth", "1", "alias for -d")
	noRefresh := fs.Bool("no-refresh", false, "never rebuild, answer from the graph on disk")
	asJSON := fs.Bool("json", false, "write the answer as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "loomux graph blast: no arguments expected, got %q\n", fs.Arg(0))
		return 2
	}
	if *base != "" && *cached {
		fmt.Fprintf(stderr, "loomux graph blast: %v\n", query.ErrBaseAndCached)
		return 2
	}
	var dep blast.Depth
	if strings.EqualFold(*depthStr, "all") || strings.EqualFold(*depthStr, "full") {
		dep = blast.All
	} else if n, err := strconv.Atoi(*depthStr); err == nil && n > 0 {
		dep = blast.Depth(n)
	} else {
		fmt.Fprintf(stderr, "loomux graph blast: -d/--depth must be a positive integer, 'all' or 'full', got %q\n", *depthStr)
		return 2
	}

	project, err := projectRoot(*root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph blast: %v\n", err)
		return 1
	}

	answer, notes, err := query.Blast(project, query.BlastOptions{
		Base:      *base,
		Cached:    *cached,
		Depth:     dep,
		NoRefresh: *noRefresh,
	})
	for _, n := range notes {
		fmt.Fprintf(stderr, "loomux graph blast: %s\n", n)
	}
	if err != nil {
		fmt.Fprintf(stderr, "loomux graph blast: %v\n", err)
		return 1
	}

	if *asJSON {
		body, err := json.MarshalIndent(answer, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "loomux graph blast: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "%s\n", body)
		return 0
	}
	fmt.Fprint(stdout, query.BlastReport(answer))
	return 0
}
