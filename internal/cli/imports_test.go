package cli_test

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

// hooksPackage is the per-edit path: every file edit of every session runs it,
// and nothing else in loomux is asked as often.
const hooksPackage = "github.com/xidus90/loomux/internal/hooks"

// forbiddenForHooks is what the per-edit path may never link. The MCP SDK
// brings a JSON-RPC stack, reflection over schemas and a HTTP client with it;
// internal/serve and internal/bridge each pull the whole SDK in behind them.
//
// The rule is structural, and the whole start-time argument of this stage
// rests on it. A sentence in a spec does not hold it -- an import added three
// layers down would break it without anyone reading that sentence again.
func forbiddenForHooks() []string {
	return []string{
		"github.com/xidus90/loomux/internal/serve",
		"github.com/xidus90/loomux/internal/bridge",
		"github.com/modelcontextprotocol/go-sdk/mcp",
	}
}

// forbiddenMaintenanceForHooks is the part of brain that decides review cases
// -- the PyYAML port, the evidence binding, the write barrier and the commit
// plumbing -- and the three axes behind `brain check`, which read whole
// bundles and every registered area. None of it belongs on the per-edit path;
// the edit lint there is `wiki`'s own. It is a list of its own
// because TestTheCommandLineDoesReachServeAndTheBridge asserts that the
// command line reaches every entry of forbiddenForHooks, and that test is
// about the MCP stack, not about these packages.
func forbiddenMaintenanceForHooks() []string {
	return []string{
		"github.com/xidus90/loomux/internal/brain/apply",
		"github.com/xidus90/loomux/internal/brain/evidence",
		"github.com/xidus90/loomux/internal/brain/maintenance",
		"github.com/xidus90/loomux/internal/brain/vcs",
		"github.com/xidus90/loomux/internal/brain/check/okf",
		"github.com/xidus90/loomux/internal/brain/check/house",
		"github.com/xidus90/loomux/internal/brain/check/run",
	}
}

// forbiddenConfigUIForHooks is the configuration command's weight: the
// schema pulls every reader, the editor its text surgery, the interface the
// terminal. [modules] on the per-edit path is read by config.ReadModules
// alone.
func forbiddenConfigUIForHooks() []string {
	return []string{
		"github.com/xidus90/loomux/internal/config/schema",
		"github.com/xidus90/loomux/internal/config/edit",
		"github.com/xidus90/loomux/internal/tui",
		"golang.org/x/term",
	}
}

// dependencies is the import graph of one package as a set of whole lines.
// Both directions ask through it: a check by substring would read
// `.../internal/serve/brain` as `.../internal/serve`, and a go list that
// answered nothing would let every question pass while measuring none.
func dependencies(pkg string) (map[string]bool, error) {
	out, err := exec.Command("go", "list", "-deps", pkg).Output()
	if err != nil {
		return nil, fmt.Errorf("go list -deps %s: %w", pkg, err)
	}
	seen := map[string]bool{}
	for _, dep := range strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n") {
		if dep != "" {
			seen[dep] = true
		}
	}
	if len(seen) < 2 {
		return nil, fmt.Errorf("go list -deps %s listed nothing:\n%s", pkg, out)
	}
	return seen, nil
}

func TestHooksNeverImportServeOrBridge(t *testing.T) {
	if testing.Short() {
		t.Skip("asks the go tool for the import graph")
	}
	seen, err := dependencies(hooksPackage)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range forbiddenForHooks() {
		if seen[forbidden] {
			t.Errorf("%s depends on %s, which puts the MCP stack on the per-edit path", hooksPackage, forbidden)
		}
	}
}

// TestHooksNeverImportTheMaintenanceLayer holds the same boundary for the
// packages behind approve, reject and defer. The command line reaches each of
// them, so a hit here is a real edge and not a list gone stale.
func TestHooksNeverImportTheMaintenanceLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("asks the go tool for the import graph")
	}
	hooks, err := dependencies(hooksPackage)
	if err != nil {
		t.Fatal(err)
	}
	cli, err := dependencies("github.com/xidus90/loomux/internal/cli")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range forbiddenMaintenanceForHooks() {
		if hooks[forbidden] {
			t.Errorf("%s depends on %s, which puts the maintenance layer on the per-edit path", hooksPackage, forbidden)
		}
		if !cli[forbidden] {
			t.Errorf("the command line does not reach %s, so the boundary test proves nothing", forbidden)
		}
	}
}

func TestHooksNeverImportTheConfigurationCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("asks the go tool for the import graph")
	}
	hooks, err := dependencies(hooksPackage)
	if err != nil {
		t.Fatal(err)
	}
	cli, err := dependencies("github.com/xidus90/loomux/internal/cli")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range forbiddenConfigUIForHooks() {
		if hooks[forbidden] {
			t.Errorf("%s depends on %s", hooksPackage, forbidden)
		}
		if !cli[forbidden] {
			t.Errorf("the command line no longer reaches %s; the list is stale", forbidden)
		}
	}
}

// setupTree is the installer: templates, host files, git files and the
// writer that applies a plan. It runs once per project, by a human; the
// per-edit path reaches none of it, and a package added below it later falls
// under the same rule without being listed.
const setupTree = "github.com/xidus90/loomux/internal/setup"

// inSetupTree says whether a package is the installer or lies below it; a
// sibling like `.../internal/setupx` is neither.
func inSetupTree(pkg string) bool {
	return pkg == setupTree || strings.HasPrefix(pkg, setupTree+"/")
}

func TestHooksNeverImportTheInstaller(t *testing.T) {
	if testing.Short() {
		t.Skip("asks the go tool for the import graph")
	}
	hooks, err := dependencies(hooksPackage)
	if err != nil {
		t.Fatal(err)
	}
	cli, err := dependencies("github.com/xidus90/loomux/internal/cli")
	if err != nil {
		t.Fatal(err)
	}
	for dep := range hooks {
		if inSetupTree(dep) {
			t.Errorf("%s depends on %s, which puts the installer on the per-edit path", hooksPackage, dep)
		}
	}
	if !cli[setupTree] {
		t.Errorf("the command line no longer reaches %s; the boundary test proves nothing", setupTree)
	}
}

func TestInSetupTreeTakesTheTreeAndNoSibling(t *testing.T) {
	for pkg, want := range map[string]bool{
		setupTree:            true,
		setupTree + "/write": true,
		setupTree + "x":      false,
		"github.com/xidus90/loomux/internal/hooks": false,
	} {
		if got := inSetupTree(pkg); got != want {
			t.Errorf("inSetupTree(%q) = %v, want %v", pkg, got, want)
		}
	}
}

// TestTheCommandLineDoesReachServeAndTheBridge is the other half: the test
// above passes just as happily when nothing in loomux links the SDK at all,
// and then it measures a boundary that costs nothing to keep.
func TestTheCommandLineDoesReachServeAndTheBridge(t *testing.T) {
	if testing.Short() {
		t.Skip("asks the go tool for the import graph")
	}
	seen, err := dependencies("github.com/xidus90/loomux/internal/cli")
	if err != nil {
		t.Fatal(err)
	}
	for _, wanted := range forbiddenForHooks() {
		if !seen[wanted] {
			t.Errorf("the command line does not reach %s, so the boundary test proves nothing", wanted)
		}
	}
}

// selfupdate sits below its callers serve, hooks and cli, and away from the
// MCP bridge; an import of any of them would be a cycle or would pull the MCP
// stack onto the session-start path.
func TestSelfupdateStaysBelowItsCallers(t *testing.T) {
	if testing.Short() {
		t.Skip("asks the go tool for the import graph")
	}
	deps, err := dependencies("github.com/xidus90/loomux/internal/selfupdate")
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{
		"github.com/xidus90/loomux/internal/serve",
		"github.com/xidus90/loomux/internal/hooks",
		"github.com/xidus90/loomux/internal/cli",
		"github.com/xidus90/loomux/internal/bridge",
	} {
		if deps[pkg] {
			t.Errorf("internal/selfupdate depends on %s", pkg)
		}
	}
}
