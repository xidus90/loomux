package importcases

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/cases"
)

// ImportMCP translates every recorded MCP case below from into to.
//
// One thing is rewritten and one only: the tool's name. A recording carries
// the reference's five bare names and loomux serves the same five under the
// brain_ family, because the protocol has no nested tools and the code graph
// will put graph_* into the same server -- the prefix is the only family
// separation there is. It is a translation rule, not a deviation: nothing
// about the answer changes with the name.
//
// The worlds are folded exactly as the command-line import folds them, and the
// target is pruned first, for the reasons Import gives.
func ImportMCP(from, to string, m Mapping) error {
	found, err := cases.DiscoverMCPCases(from)
	if err != nil {
		return err
	}
	if err := pruneMCP(to, found); err != nil {
		return err
	}
	for _, c := range found {
		out := filepath.Join(to, c.Verb, c.Name)
		// The one file this import rewrites is not copied first: one writer
		// per file keeps the copy from being the one that fails.
		if err := copyTree(c.Path, out, map[string]bool{"call": true}); err != nil {
			return err
		}
		call, err := renameTool(c, m)
		if err != nil {
			return err
		}
		// Written from its parts rather than marshalled: the arguments arrive
		// as the recorder indented them, and putting them back through a
		// marshaller would reflow bytes this import has no business touching.
		data := fmt.Sprintf("{\n  \"tool\": %q,\n  \"arguments\": %s,\n  \"channel\": %q\n}\n",
			call.Tool, call.Arguments, call.Channel)
		if err := os.WriteFile(filepath.Join(out, "call"), []byte(data), 0o644); err != nil {
			return err
		}
		if err := TranslateWorld(filepath.Join(out, "world")); err != nil {
			return err
		}
	}
	return nil
}

// renameTool applies the rule that names this case's tool.
func renameTool(c *cases.MCPCase, m Mapping) (cases.MCPCall, error) {
	for _, rule := range m.Tools {
		if c.Call.Tool == rule.From {
			call := c.Call
			call.Tool = rule.To
			return call, nil
		}
	}
	return cases.MCPCall{}, fmt.Errorf("no rule for the tool of case %s/%s: %s", c.Verb, c.Name, c.Call.Tool)
}

// pruneMCP removes every MCP case in the target that no recording backs, for
// the reasons prune gives.
func pruneMCP(to string, found []*cases.MCPCase) error {
	backed := map[string]bool{}
	for _, c := range found {
		backed[filepath.Join(c.Verb, c.Name)] = true
	}
	existing, err := cases.DiscoverMCPCases(to)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading the corpus in %s: %w", to, err)
	}
	places := map[string]string{}
	for _, c := range existing {
		places[filepath.Join(c.Verb, c.Name)] = c.Path
	}
	return prunePaths(backed, places)
}
