package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/bridge"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/mcptools"
)

// bridgeRun is the seam of this file. A test that let the real bridge run
// would hand it this process's stdin and stdout and wait for a host that never
// speaks.
var bridgeRun = bridge.Run

func mcpUsage() string { return "usage: loomux mcp [--channel {local,cloud}] [--root DIR]" }

// mcpChannels are the two channels, in the order `loomux brain` names them.
func mcpChannels() []string {
	return []string{string(privacy.ChannelLocal), string(privacy.ChannelCloud)}
}

// mcpCommand is `loomux mcp`: the stdio bridge an MCP host starts.
//
// Nothing here ever writes to stdout. Stdout is the host's MCP pipe, and one
// line of ours in it is a protocol error the host reports as a broken server.
// Refusals and failures go to stderr, which the host either logs or drops.
//
// bridge.Options.Spawn stays unset on purpose. Starting a service is something
// the bridge does for itself, in the background and unasked, and it is the only
// caller that knows when: it owns that default (bridge.defaultSpawn over
// serve.Spawn), and bridge_internal_test.go covers it there. A spawner handed
// down from here would be a second construction site for the same thing, and it
// would leave the bridge's own default reachable from the bridge's tests alone.
// The command line's spawner belongs to `loomux serve`, where a human asked for
// a start.
func mcpCommand(args []string, _ io.Reader, _, stderr io.Writer) int {
	channel, root, refused := mcpArgs(args)
	if refused != nil {
		brainRefuse(stderr, mcpUsage(), "loomux mcp", refused.message)
		return 2
	}
	root = mcpRoot(root)
	modules, err := projectModules(root)
	if err != nil {
		fmt.Fprintf(stderr, "loomux mcp: %v\n", err)
		return 1
	}
	// Ctrl+C ends the bridge the way a host hanging up does, through the
	// context: Run closes its session downwards on the way out, and a killed
	// bridge would leave that session to the service's own timeout.
	ctx, stop := serveNotify(context.Background(), os.Interrupt)
	defer stop()
	// A cancelled context is how this command ends when it is asked to; it is
	// not a failure and must not be reported as one.
	stateDir := config.StateDir()
	opts := bridge.Options{
		StateDir: stateDir, Channel: channel, Tools: mcptools.For(modules),
		DefaultScope: defaultScope(stateDir, root, channel),
	}
	if err := bridgeRun(ctx, opts); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintf(stderr, "loomux mcp: %v\n", err)
		return 1
	}
	return 0
}

// mcpArgs reads the two options this command knows. The shape is small enough
// to read by hand -- two options of one value each and no positional -- and
// the refusals are argparse's, as `loomux brain`'s are.
//
// Without --channel the answer is local, exactly as `loomux brain` falls back
// (brainParserFor): the channel decides what a model may see, and the narrow
// one is the only safe default.
func mcpArgs(args []string) (privacy.Channel, string, *brainUsageError) {
	channel := string(privacy.ChannelLocal)
	root := ""
	var extra []string
	for i := 0; i < len(args); i++ {
		flag, value, glued := strings.Cut(args[i], "=")
		if flag != "--channel" && flag != "--root" {
			extra = append(extra, args[i])
			continue
		}
		if !glued {
			if i+1 == len(args) {
				return "", "", &brainUsageError{message: "argument " + flag + ": expected one argument"}
			}
			i++
			value = args[i]
		}
		if flag == "--root" {
			root = value
			continue
		}
		if value != string(privacy.ChannelLocal) && value != string(privacy.ChannelCloud) {
			return "", "", &brainUsageError{message: "argument --channel: invalid choice: " +
				pytext.Repr(value) + " (choose from " + brainChoices(mcpChannels()) + ")"}
		}
		channel = value
	}
	if len(extra) > 0 {
		return "", "", &brainUsageError{message: "unrecognized arguments: " + strings.Join(extra, " ")}
	}
	return privacy.Channel(channel), root, nil
}

// mcpRoot is the project this bridge serves: the one named by --root, else
// the first .loomux/config.toml above the working directory the host started
// it in, else "" -- a host may start the bridge anywhere.
func mcpRoot(root string) string {
	if root != "" {
		return root
	}
	found, err := hosts.FindRoot(".")
	if err != nil {
		return ""
	}
	return found
}

// projectModules is the [modules] of the project at root. No project offers
// every tool.
func projectModules(root string) (config.Modules, error) {
	if root == "" {
		return config.AllModules(), nil
	}
	return config.ReadModules(root)
}

// defaultScope is the scope a graph call without one asks: the innermost area
// that holds root among those the channel sees, read once at start.
//
// Only visible areas are candidates, and a root inside a tree the channel
// hides gets none: a cloud caller must not be handed a local_only scope, nor a
// visible hub whose answers would come from inside one. Every failure --
// no project, no registry, a broken one -- is "", and the caller names the
// scope itself as before.
func defaultScope(stateDir, root string, channel privacy.Channel) string {
	if root == "" {
		return ""
	}
	if abs, err := filepath.Abs(root); err == nil {
		root = abs
	}
	areas, err := privacy.VisibleAreas(stateDir, "all", channel)
	if err != nil || len(areas) == 0 {
		return ""
	}
	for _, hidden := range areas[0].Hidden {
		if privacy.Within(hidden, root) {
			return ""
		}
	}
	scope, depth := "", -1
	for _, area := range areas {
		if len(area.Area.Path) > depth && privacy.Within(area.Area.Path, root) {
			scope, depth = area.Area.Scope, len(area.Area.Path)
		}
	}
	return scope
}
