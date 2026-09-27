package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xidus90/loomux/flows"
	"github.com/xidus90/loomux/internal/flow"
	"github.com/xidus90/loomux/internal/flow/blocks"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/model"
	"github.com/xidus90/loomux/internal/flow/runner"
	"github.com/xidus90/loomux/internal/hosts"
)

// The exit codes of `loomux flow`. 2 is a usage error, as the flag package
// answers one; a paused run is 3 so that a script can tell "waiting at a
// gate" from "typed it wrong".
const (
	flowExitOK     = 0
	flowExitFailed = 1
	flowExitUsage  = 2
	flowExitPaused = 3
)

const flowUsage = `usage:
  loomux flow run [<flow>] [--option name=value]... [--root dir]
  loomux flow resume <run> [--answer text] [--root dir]
  loomux flow replay <run> [--root dir]
  loomux flow show <run|flow> [--root dir]
  loomux flow list [--root dir]`

// flowDeps are what a run takes from outside its arguments. Tests hand in a
// deterministic clock that gives the same sequence every run, a model that
// answers from a queue, blocks and a catalog of their own; flowCommand hands
// in the real ones.
type flowDeps struct {
	Stdout io.Writer
	Stderr io.Writer
	Clock  runner.Clock
	// Models finds the model for a provider, before this command writes anything.
	Models func(provider string) (model.Model, error)
	Blocks []flow.Block
	// Bundled is the catalog of flows this binary ships.
	Bundled fs.FS
}

// flowCommand is `loomux flow`: the edge between a command line and the
// runtime under internal/flow.
func flowCommand(args []string, _ io.Reader, stdout, stderr io.Writer) int {
	return flowCLI(args, flowProduction(stdout, stderr))
}

// flowProduction is the real outside: the wall clock, the three blocks the
// runtime ships, the bundled catalog, and no adapter for any provider yet. A
// flow that asks a model is refused before its run exists.
func flowProduction(stdout, stderr io.Writer) flowDeps {
	return flowDeps{
		Stdout: stdout,
		Stderr: stderr,
		Clock:  time.Now,
		Models: func(provider string) (model.Model, error) {
			return nil, fmt.Errorf("no adapter for provider %s yet", provider)
		},
		Blocks:  []flow.Block{blocks.Agent{}, blocks.Gate{}, blocks.Exit{}},
		Bundled: flows.FS(),
	}
}

// flowCLI is `loomux flow` without the process: a second caller hands in its
// own outside and gets the exit code back. It reads no stdin; everything a
// run is left with is in its journal and its marker.
func flowCLI(args []string, deps flowDeps) int {
	if len(args) == 0 {
		fmt.Fprintln(deps.Stderr, flowUsage)
		return flowExitUsage
	}
	switch args[0] {
	case "run":
		return flowRunCommand(args[1:], deps)
	case "resume":
		return flowResumeCommand(args[1:], deps)
	case "replay":
		return flowReplayCommand(args[1:], deps)
	case "show":
		return flowShowCommand(args[1:], deps)
	case "list":
		return flowListCommand(args[1:], deps)
	default:
		fmt.Fprintf(deps.Stderr, "loomux flow: unknown command %q\n%s\n", args[0], flowUsage)
		return flowExitUsage
	}
}

// flowName says whether a command takes a name before its flags.
type flowName int

const (
	noName       flowName = iota
	optionalName          // run: without one, [flow] default decides
	requiredName
)

// flowFlags is a command's flag set with the --root every command takes.
func flowFlags(command string) (*flag.FlagSet, *string) {
	flags := flag.NewFlagSet("loomux flow "+command, flag.ContinueOnError)
	root := flags.String("root", "", "the project directory; the nearest .loomux/config.toml above the working directory when not given")
	return flags, root
}

// flowArguments reads a command's name and its flags, and settles its root.
// flag stops at the first argument that is no flag, so the name comes first
// and the flags after it; anything left over is a mistake and gets the usage.
func flowArguments(command string, args []string, takes flowName, flags *flag.FlagSet, root *string, stderr io.Writer) (string, int, bool) {
	flags.SetOutput(stderr)
	name := ""
	hasName := len(args) > 0 && !strings.HasPrefix(args[0], "-")
	switch {
	case takes == requiredName && !hasName:
		fmt.Fprintf(stderr, "loomux flow %s: the name is missing\n%s\n", command, flowUsage)
		return "", flowExitUsage, false
	case takes != noName && hasName:
		name, args = args[0], args[1:]
	}
	if err := flags.Parse(args); err != nil {
		// Asking for help is not a failure: an exit code of 2 for -h makes
		// every wrapper script think the tool broke.
		if errors.Is(err, flag.ErrHelp) {
			return "", flowExitOK, false
		}
		return "", flowExitUsage, false
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "loomux flow %s: unexpected argument %q\n%s\n", command, flags.Arg(0), flowUsage)
		return "", flowExitUsage, false
	}
	code, ok := flowRootOrRefuse(command, root, hosts.FindRoot, stderr)
	return name, code, ok
}

// flowRootOrRefuse settles --root: as given, else the project above the
// working directory, as `loomux config` finds it, else the working directory
// itself -- a project without a .loomux/config.toml still has flows. find is
// hosts.FindRoot outside the tests; the one other failure it has is a working
// directory the process cannot name.
func flowRootOrRefuse(command string, root *string, find func(string) (string, error), stderr io.Writer) (int, bool) {
	if *root != "" {
		return flowExitOK, true
	}
	found, err := find(".")
	switch {
	case errors.Is(err, hosts.ErrNoRoot):
		*root = "."
	case err != nil:
		fmt.Fprintf(stderr, "loomux flow %s: %v\n", command, err)
		return flowExitFailed, false
	default:
		*root = found
	}
	return flowExitOK, true
}

// flowRefuse says why on stderr. A refusal is exit 1, like a failed run: the
// caller's next step is the same, reading the reason.
func flowRefuse(deps flowDeps, err error) int {
	fmt.Fprintln(deps.Stderr, err)
	return flowExitFailed
}

// flowWarn says on stderr what a command went on despite.
func flowWarn(deps flowDeps, text string) {
	fmt.Fprintf(deps.Stderr, "warning: %s\n", text)
}

// flowOrigin is where a found flow came from as run, show and list print it,
// with the files an overlay replaced.
func flowOrigin(found load.Found) string {
	if len(found.Overlays) == 0 {
		return found.Origin
	}
	return found.Origin + ": " + strings.Join(found.Overlays, ", ")
}

// flowNames joins names for a message, and says none when there are none.
func flowNames(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

// flowColumns writes rows with every cell but a row's last padded to the
// widest in its column, two spaces apart. The last cell runs to the end of
// the line, so a row may be shorter than the others, and a last cell may
// hold a line break.
func flowColumns(w io.Writer, rows [][]string) {
	var widths []int
	for _, row := range rows {
		for i, cell := range row[:len(row)-1] {
			if i == len(widths) {
				widths = append(widths, 0)
			}
			widths[i] = max(widths[i], utf8.RuneCountInString(cell))
		}
	}
	for _, row := range rows {
		var line strings.Builder
		for i, cell := range row[:len(row)-1] {
			line.WriteString(cell + strings.Repeat(" ", widths[i]-utf8.RuneCountInString(cell)+2))
		}
		line.WriteString(row[len(row)-1])
		fmt.Fprintln(w, line.String())
	}
}
