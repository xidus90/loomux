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
