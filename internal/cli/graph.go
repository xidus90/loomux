package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/query"
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
	g, stats, err := query.Build(project,
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

// report is the count line after a build: what the extractor found, so a
// reader can see at a glance whether it found the right kind of thing.
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
	return fmt.Sprintf(
		"%d files, %d nodes, %d edges (%d contains, %d calls, %d imports)\n"+
			"%d unresolved import targets, %d files without a symbol, %s\n",
		len(stats.Files), len(g.Nodes), len(g.Edges),
		byRelation[model.RelationContains], byRelation[model.RelationCalls],
		byRelation[model.RelationImports],
		unresolved, stats.NoSymbol, took.Round(time.Millisecond),
	)
}
