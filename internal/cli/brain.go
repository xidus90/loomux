package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/brain/catalog"
	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/reader"
	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/brain/status"
	"github.com/xidus90/loomux/internal/config"
)

// brainSearchPort is the engine `brain search` asks: the warm qmd daemon over
// MCP. notice hears it when this call had to start the daemon first.
var brainSearchPort = func(notice func(string)) search.SearchPort {
	return search.NewQmdMcpPort(search.WithNotice(notice))
}

// brainStatusPort is the engine `brain status` asks: the qmd command line,
// because the daemon has no `ls` and counts its backlog with another model.
var brainStatusPort = func() search.SearchPort {
	return &search.QmdPort{Executable: "qmd", Runner: search.DefaultRunner}
}

// brainNow is the clock a reconcile stamp is judged against.
var brainNow = time.Now

// brainCommand is `loomux brain`: the five read commands of brain-mcp
// (cli.py:712-810) with their argument forms, exit codes and error line.
func brainCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		brainRefuse(stderr, brainTopUsage(), "loomux brain", "the following arguments are required: command")
		return 2
	}
	parser, known := brainParserFor(args[0])
	if !known {
		brainRefuse(stderr, brainTopUsage(), "loomux brain",
			"argument command: invalid choice: "+pytext.Repr(args[0])+" (choose from "+brainChoices(brainSubcommands)+")")
		return 2
	}
	parsed, refused := parser.parse(args[1:])
	if refused != nil {
		if refused.top {
			brainRefuse(stderr, brainTopUsage(), "loomux brain", refused.message)
		} else {
			brainRefuse(stderr, parser.usage(), "loomux brain "+parser.name, refused.message)
		}
		return 2
	}
	// Nothing reaches stdout before the answer stands: a failure after half an
	// answer would leave the reader holding lines that look complete.
	out, notes, err := brainRun(parser.name, parsed, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	io.WriteString(stdout, out)
	for _, note := range notes {
		fmt.Fprintf(stderr, "note: %s\n", note)
	}
	return 0
}

// brainRefuse writes argparse's usage error: the usage line, then which parser
// refused and why.
func brainRefuse(stderr io.Writer, usage, prog, message string) {
	fmt.Fprintf(stderr, "%s\n%s: error: %s\n", usage, prog, message)
}

// brainRun answers one parsed subcommand: stdout, the notes for stderr, or the
// error. The registry is loomux's; artefacts of read-only areas and the stamp
// stay in ultra-brain's state directory until stage 3.
func brainRun(name string, a brainArgs, stderr io.Writer) (string, []string, error) {
	registryDir, legacyDir := config.StateDir(), config.LegacyBrainDirUntilStage3()
	// The parser let through only the two channel names, so the conversion
	// cannot produce a third.
	channel := privacy.Channel(a.values["--channel"])
	switch name {
	case "search":
		return brainSearch(a, channel, registryDir, legacyDir, stderr)
	case "catalog":
		text, err := brainCatalog(a.values["--scope"], channel, registryDir, legacyDir)
		return text, nil, err
	case "read":
		text, err := brainRead(a.positional, a.values["--scope"], a.values["--section"], channel, registryDir, legacyDir)
		return text, nil, err
	case "neighbors":
		text, err := brainNeighbors(a.positional, a.values["--scope"], channel, registryDir, legacyDir)
		return text, nil, err
	}
	text, err := brainStatus(channel, registryDir, legacyDir)
	return text, nil, err
}

// brainSearch is core.search plus _print_search: the hits for stdout, the
// findings as notes.
func brainSearch(a brainArgs, channel privacy.Channel, registryDir, legacyDir string, stderr io.Writer) (string, []string, error) {
	port := brainSearchPort(func(message string) {
		fmt.Fprintf(stderr, "note: %s\n", message)
	})
	if closer, ok := port.(io.Closer); ok {
		// Let go of the session whatever the answer was; the answer does not
		// depend on how letting go went.
		defer closer.Close()
	}
	answer, err := search.ExecuteSearch(a.positional, a.values["--scope"], search.Profile(a.values["--profile"]),
		a.n, channel, port, registryDir, legacyDir, brainNow())
	if err != nil {
		return "", nil, err
	}
	return search.FormatSearch(answer), answer.Findings, nil
}

// brainCatalog is core.catalog: the root catalog of the visible areas, or the
// index.md of one of them.
func brainCatalog(scope string, channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", channel)
	if err != nil {
		return "", err
	}
	if scope == "all" {
		plain := make([]config.Area, len(areas))
		for i, visible := range areas {
			plain[i] = visible.Area
		}
		return catalog.RenderRootCatalog(plain), nil
	}
	area, err := privacy.Single(areas, scope)
	if err != nil {
		return "", err
	}
	return catalog.ReadAreaCatalog(area.Area, legacyDir)
}

// brainArea is the one visible area a read or a neighbour query names; every
// area of the registry is checked first, as core._visible_areas("all") does.
func brainArea(scope string, channel privacy.Channel, registryDir, legacyDir string) (privacy.VisibleArea, error) {
	areas, err := privacy.VisibleAreas(registryDir, legacyDir, "all", channel)
	if err != nil {
		return privacy.VisibleArea{}, err
	}
	return privacy.Single(areas, scope)
}

// brainRead is core.read.
func brainRead(relative, scope, section string, channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	area, err := brainArea(scope, channel, registryDir, legacyDir)
	if err != nil {
		return "", err
	}
	return reader.ReadDocument(area.Area, area.Manifest, relative, section, channel, legacyDir)
}

// brainNeighbors is core.neighbors plus render_neighbors: containment before
// the graph is read, so a path leaving the area is refused even where no graph
// exists.
func brainNeighbors(relative, scope string, channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	area, err := brainArea(scope, channel, registryDir, legacyDir)
	if err != nil {
		return "", err
	}
	inside, err := privacy.Contained(scope, relative)
	if err != nil {
		return "", err
	}
	g, err := graph.ReadGraph(area.Area, legacyDir)
	if err != nil {
		return "", err
	}
	incoming, outgoing := graph.Neighbors(g, inside)
	return graph.RenderNeighbors(incoming, outgoing), nil
}

// brainStatus is core.status plus _print_status: one line each.
func brainStatus(channel privacy.Channel, registryDir, legacyDir string) (string, error) {
	lines, err := status.Lines(channel, brainStatusPort(), registryDir, legacyDir, brainNow())
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	return b.String(), nil
}
