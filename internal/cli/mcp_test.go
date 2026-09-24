package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/bridge"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/mcptools"
)

// stubBridgeRun keeps the real bridge away from this process's stdio: it would
// take stdin and stdout for a host's MCP pipe and wait for a host that never
// speaks.
func stubBridgeRun(t *testing.T, err error) *bridge.Options {
	t.Helper()
	// The bridge reads [modules] upwards from the working directory; the
	// package directory would hand every test this repository's own file.
	t.Chdir(t.TempDir())
	seen := &bridge.Options{}
	saved := bridgeRun
	bridgeRun = func(_ context.Context, opts bridge.Options) error {
		*seen = opts
		return err
	}
	t.Cleanup(func() { bridgeRun = saved })
	return seen
}

func TestMcpBridgesTheLocalChannelByDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.StateDirEnv, dir)
	seen := stubBridgeRun(t, nil)
	released := stubServeNotify(t)

	code, out, errOut := run("mcp")
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if !*released {
		t.Error("the bridge kept the interrupt registration after it ended")
	}
	if seen.StateDir != dir {
		t.Errorf("the bridge got state directory %q, want %q", seen.StateDir, dir)
	}
	if seen.Channel != privacy.ChannelLocal {
		t.Errorf("the bridge got channel %q, want the local one", seen.Channel)
	}
	// Nothing of this command's own may be handed to the bridge: the spawner
	// it falls back to is its own, tested there, and a second one here would
	// be a second start path in production.
	if seen.Spawn != nil {
		t.Error("the command line handed the bridge a spawner of its own")
	}
	if seen.Transport != nil {
		t.Error("the command line handed the bridge a transport of its own")
	}
}

func TestMcpTakesTheChannelInBothSpellings(t *testing.T) {
	for _, args := range [][]string{{"--channel", "cloud"}, {"--channel=cloud"}} {
		t.Setenv(config.StateDirEnv, t.TempDir())
		seen := stubBridgeRun(t, nil)
		if code, _, errOut := run(append([]string{"mcp"}, args...)...); code != 0 {
			t.Fatalf("%v: code %d, err %q", args, code, errOut)
		}
		if seen.Channel != privacy.ChannelCloud {
			t.Errorf("%v: the bridge got channel %q", args, seen.Channel)
		}
	}
}

func TestMcpRefusesAnUnknownChannel(t *testing.T) {
	stubBridgeRun(t, errors.New("the bridge was run although the channel was refused"))
	code, out, errOut := run("mcp", "--channel", "public")
	if code != 2 || out != "" {
		t.Fatalf("code %d, out %q", code, out)
	}
	want := mcpUsage() + "\nloomux mcp: error: argument --channel: invalid choice: 'public' (choose from 'local', 'cloud')\n"
	if errOut != want {
		t.Fatalf("err %q, want %q", errOut, want)
	}
}

func TestMcpRefusesAChannelWithoutAValue(t *testing.T) {
	code, _, errOut := run("mcp", "--channel")
	if code != 2 || !strings.Contains(errOut, "argument --channel: expected one argument") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

func TestMcpRefusesWhatItDoesNotKnow(t *testing.T) {
	code, _, errOut := run("mcp", "serve", "--loud")
	if code != 2 || !strings.Contains(errOut, "unrecognized arguments: serve --loud") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
}

// TestMcpReportsAFailedBridgeOnStderr: stdout is the host's MCP pipe, and one
// line of ours in it is a protocol error the host reports as a broken server.
func TestMcpReportsAFailedBridgeOnStderr(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	stubBridgeRun(t, errors.New("the host hung up badly"))

	code, out, errOut := run("mcp")
	if code != 1 || out != "" || !strings.Contains(errOut, "the host hung up badly") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

// TestMcpEndsQuietlyWhenItIsAskedTo: an interrupt cancels the context, and the
// bridge hands that cancellation back. It is how this command is meant to end,
// not a failure.
func TestMcpEndsQuietlyWhenItIsAskedTo(t *testing.T) {
	t.Setenv(config.StateDirEnv, t.TempDir())
	stubBridgeRun(t, context.Canceled)

	code, out, errOut := run("mcp")
	if code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

// offers says whether the bridge was handed any tool of the module whose
// names start with prefix.
func offers(o *bridge.Options, prefix string) bool {
	for _, tool := range o.Tools {
		if strings.HasPrefix(tool.Name, prefix) {
			return true
		}
	}
	return false
}

// TestMcpReadsTheModulesOfTheProjectAbove: a host starts the bridge in the
// project, not necessarily at its root, so the lookup climbs.
func TestMcpReadsTheModulesOfTheProjectAbove(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\ngraph = false\n")
	sub := filepath.Join(root, "internal")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	seen := stubBridgeRun(t, nil)
	t.Chdir(sub)
	t.Setenv(config.StateDirEnv, t.TempDir())

	if code, out, errOut := run("mcp"); code != 0 || out != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if offers(seen, "graph_") || !offers(seen, "brain_") {
		t.Fatalf("tools %v; want graph off, brain on", seen.Tools)
	}
}

// TestMcpWithoutAProjectOffersEverything: a host may start the bridge in any
// directory, and no project there is no reason to withhold a tool.
func TestMcpWithoutAProjectOffersEverything(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv(config.StateDirEnv, t.TempDir())
	seen := stubBridgeRun(t, nil)

	if code, out, errOut := run("mcp"); code != 0 || out != "" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if len(seen.Tools) != len(mcptools.Tools()) {
		t.Fatalf("%d tools; want all %d", len(seen.Tools), len(mcptools.Tools()))
	}
}

func TestMcpTakesAnExplicitRootInBothSpellings(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\nbrain = false\n")
	for _, args := range [][]string{{"--root", root}, {"--root=" + root}} {
		t.Setenv(config.StateDirEnv, t.TempDir())
		seen := stubBridgeRun(t, nil)
		if code, out, errOut := run(append([]string{"mcp"}, args...)...); code != 0 || out != "" {
			t.Fatalf("%v: code %d, out %q, err %q", args, code, out, errOut)
		}
		if offers(seen, "brain_") || !offers(seen, "graph_") {
			t.Fatalf("%v: tools %v; want brain off, graph on", args, seen.Tools)
		}
	}
}

func TestMcpRefusesARootWithoutAValue(t *testing.T) {
	stubBridgeRun(t, errors.New("the bridge was run although --root was refused"))
	code, out, errOut := run("mcp", "--root")
	want := mcpUsage() + "\nloomux mcp: error: argument --root: expected one argument\n"
	if code != 2 || out != "" || errOut != want {
		t.Fatalf("code %d, out %q, err %q, want %q", code, out, errOut, want)
	}
}

// TestMcpRefusesABrokenModulesTable: a bridge that guessed would offer tools
// the project switched off, or hide ones it wants. And stdout stays empty:
// it is the host's MCP pipe.
func TestMcpRefusesABrokenModulesTable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".loomux", "config.toml"), "[modules]\nbrain = 3\n")
	stubBridgeRun(t, errors.New("the bridge was run although [modules] is broken"))

	code, out, errOut := run("mcp", "--root", root)
	if code != 1 || out != "" || !strings.HasPrefix(errOut, "loomux mcp: ") || !strings.Contains(errOut, "brain") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}
