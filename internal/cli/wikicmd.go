package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/brain/wiki"
	"github.com/xidus90/loomux/internal/config"
)

// wikiCommand is `loomux wiki init|types|retype`, the three wiki tools of the
// Python reference: `brain wiki init`, `brain types` and `brain retype`. The
// two that stood at the top level there live under `wiki` here, beside the
// one that did not.
//
// The exit codes are the reference's: 2 for a command line argparse would
// refuse, 1 for a refusal about the registry or a file (`main` turns a
// LookupError or an OSError into one `error:` line), 0 otherwise.
func wikiCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "error: 'loomux wiki' needs init, types or retype")
		return 2
	}
	switch args[0] {
	case "init":
		return wikiInit(args[1:], stdout, stderr)
	case "types":
		return wikiTypes(args[1:], stdout, stderr)
	case "retype":
		return wikiRetype(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "error: unknown wiki command %q\n", args[0])
	return 2
}

// wikiFlags parses the flags of one wiki command and refuses a positional
// argument or a missing required flag the way argparse does, with 2.
func wikiFlags(name string, args []string, stderr io.Writer, required ...string) (map[string]*string, bool) {
	flags := flag.NewFlagSet("wiki "+name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := map[string]*string{}
	for _, flagName := range []string{"scope", "from", "to"} {
		values[flagName] = flags.String(flagName, "", "")
	}
	free, err := parseInterspersed(flags, args)
	if err != nil {
		return nil, false
	}
	if len(free) > 0 {
		fmt.Fprintf(stderr, "error: unrecognized arguments: %s\n", free[0])
		return nil, false
	}
	for _, flagName := range required {
		if *values[flagName] == "" {
			fmt.Fprintf(stderr, "error: the following arguments are required: --%s\n", flagName)
			return nil, false
		}
	}
	return values, true
}

// wikiArea is the registered area a wiki command names, with the lookup the
// registry came from. The two refusals every such command shares are the
// reference's, word for word.
func wikiArea(scope string, stderr io.Writer) (config.Area, config.ArtifactLookup, bool) {
	lookup := config.NewArtifactLookup()
	areas, err := config.ReadRegistry(lookup.Primary)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return config.Area{}, lookup, false
	}
	for _, area := range areas {
		if area.Scope != scope {
			continue
		}
		if area.WikiPath == "" {
			fmt.Fprintf(stderr, "error: area %s declares no wiki path; add `wiki = ...` to its entry\n", pytext.Repr(scope))
			return config.Area{}, lookup, false
		}
		return area, lookup, true
	}
	fmt.Fprintf(stderr, "error: no area named %s in the registry\n", pytext.Repr(scope))
	return config.Area{}, lookup, false
}

// wikiInit is `_wiki_init`: the bundle frame of one area, printing every file
// it wrote. An existing file is never rewritten.
func wikiInit(args []string, stdout, stderr io.Writer) int {
	values, ok := wikiFlags("init", args, stderr, "scope")
	if !ok {
		return 2
	}
	area, _, ok := wikiArea(*values["scope"], stderr)
	if !ok {
		return 1
	}
	// The line retype draws as well: a frame written into a read-only area
	// would be the first forbidden write.
	if area.ReadOnly {
		fmt.Fprintf(stderr, "error: area %s is read-only; no bundle is written into it\n", pytext.Repr(area.Scope))
		return 1
	}
	written, err := wiki.InitBundle(area.WikiPath)
	if err != nil {
		// Nothing on stdout: the reference prints the paths only once the
		// whole run came back, and an error takes that answer with it.
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	for _, path := range written {
		fmt.Fprintln(stdout, path)
	}
	return 0
}

// wikiTypes is `_types`: the census over every registered area. A tally does
// not judge, so its exit is 0 whatever it counts.
func wikiTypes(args []string, stdout, stderr io.Writer) int {
	if _, ok := wikiFlags("types", args, stderr); !ok {
		return 2
	}
	lookup := config.NewArtifactLookup()
	areas, err := config.ReadRegistry(lookup.Primary)
	if err == nil {
		var counts []wiki.TypeCount
		counts, err = wiki.Census(areas, manifestDirOf(lookup))
		if err == nil {
			io.WriteString(stdout, wiki.RenderCensus(counts))
			return 0
		}
	}
	fmt.Fprintf(stderr, "error: %v\n", err)
	return 1
}

// wikiRetype is `_retype`: one type renamed in one area's bundle, printing
// every page it wrote. A read-only area takes no write. A target type the area
// does not know is a warning and not a refusal: a typo in `--to` is easy to
// make, and the run is still exactly what was asked for.
func wikiRetype(args []string, stdout, stderr io.Writer) int {
	values, ok := wikiFlags("retype", args, stderr, "scope", "from", "to")
	if !ok {
		return 2
	}
	scope, source, target := *values["scope"], *values["from"], *values["to"]
	area, lookup, ok := wikiArea(scope, stderr)
	if !ok {
		return 1
	}
	if area.ReadOnly {
		fmt.Fprintf(stderr, "error: area %s is read-only; its bundle cannot be renamed\n", pytext.Repr(scope))
		return 1
	}
	declared, err := wiki.DeclaredTypesIn(manifestDirOf(lookup)(area))
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if wiki.RankOf(&target, declared) == wiki.RankUnknown {
		fmt.Fprintf(stderr, "warning: %s is neither core, catalogue, origin nor declared in area %s's manifest; "+
			"`loomux lint` will flag it as missing-type\n", pytext.Repr(target), pytext.Repr(scope))
	}
	changed, err := wiki.Retype(area.WikiPath, source, target)
	if err != nil {
		// As in wikiInit: no paths when the run did not come back whole.
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	for _, path := range changed {
		fmt.Fprintln(stdout, path)
	}
	return 0
}

// manifestDirOf is where an area's declaration lies: in its own tree, or for a
// read-only area under the state directory.
func manifestDirOf(lookup config.ArtifactLookup) func(config.Area) string {
	return func(area config.Area) string {
		return config.ManifestDir(area, lookup.Primary)
	}
}
