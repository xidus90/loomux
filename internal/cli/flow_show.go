package cli

import (
	"cmp"
	"fmt"
	"os"
	"strings"

	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/load"
	"github.com/xidus90/loomux/internal/flow/runs"
)

// flowShowCommand shows a run or a flow. The two cannot be mistaken for each
// other: a run number is digits, and a flow name starts with a letter.
func flowShowCommand(args []string, deps flowDeps) int {
	flags, root := flowFlags("show")
	target, code, ok := flowArguments("show", args, requiredName, flags, root, deps.Stderr)
	if !ok {
		return code
	}
	if strings.Trim(target, "0123456789") != "" {
		return flowShowFlow(*root, target, deps)
	}
	return flowShowRun(*root, target, deps)
}

// flowShowRun prints one line per journal entry, in fixed columns: node,
// kind, outcome, tokens, seconds and tool profile.
func flowShowRun(root, id string, deps flowDeps) int {
	path := runs.JournalPath(root, id)
	if _, err := os.Stat(path); err != nil {
		return flowRefuse(deps, flowNoRun(root, id))
	}
	entries, err := journal.Entries(path)
	if err != nil {
		return flowRefuse(deps, err)
	}
	for _, entry := range entries {
		tools := "-"
		if entry.Tools != nil && *entry.Tools != "" {
			tools = *entry.Tools
		}
		fmt.Fprintf(deps.Stdout, "%-24s %-6s %-7s %7d tok %7.2fs %s\n",
			entry.Node, entry.Kind, entry.Outcome, entry.Tokens, entry.Seconds, tools)
	}
	return flowExitOK
}

// flowShowFlow prints a flow as a run would load it: where it came from, its
// nodes -- an agent node with its role, the model that role resolves to and
// how -- and its edges with the condition each is taken on.
func flowShowFlow(root, name string, deps flowDeps) int {
	s, err := openFlow(root, name, nil, deps)
	if err != nil {
		return flowRefuse(deps, err)
	}
	fmt.Fprintf(deps.Stdout, "%s (%s)\nnodes:\n", s.found.Name, flowOrigin(s.found))
	rows := make([][]string, 0, len(s.graph.Nodes))
	for _, node := range s.graph.Nodes {
		row := []string{"  " + node.Name, node.Kind}
		if node.Kind == "agent" {
			resolved := s.resolve(node)
			row = append(row, cmp.Or(resolved.Role, "-"), resolved.Label(), resolved.Via())
		}
		rows = append(rows, row)
	}
	flowColumns(deps.Stdout, rows)
	fmt.Fprintln(deps.Stdout, "edges:")
	for _, edge := range s.graph.Edges {
		taken := ""
		switch {
		case edge.OnError:
			taken = " [on error]"
		case edge.Text != "":
			taken = " [" + edge.Text + "]"
		}
		fmt.Fprintf(deps.Stdout, "  %s -> %s%s\n", edge.From, edge.To, taken)
	}
	return flowExitOK
}

// flowListCommand prints every flow a project can name, with its origin and
// ok, or the findings that keep it from loading: a flow missing from the list
// reads as a flow nobody wrote. What [flow] names and is not there, and what
// of the project a bundled flow ignored, is said on stderr: a warning, but for
// a default that names no flow, which fails list as it fails run -- after the
// list, which shows what the default could name.
func flowListCommand(args []string, deps flowDeps) int {
	flags, root := flowFlags("list")
	if _, code, ok := flowArguments("list", args, noName, flags, root, deps.Stderr); !ok {
		return code
	}
	project, err := readFlowProject(*root, deps)
	if err != nil {
		return flowRefuse(deps, err)
	}
	for _, warning := range load.SettingsWarnings(deps.Bundled, project.settings) {
		flowWarn(deps, warning)
	}
	entries := load.List(*root, deps.Bundled, project.settings, project.catalog)
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		for _, warning := range entry.Warnings {
			flowWarn(deps, warning)
		}
		state := "ok"
		if entry.Problem != "" {
			state = strings.ReplaceAll(entry.Problem, "\n", "\n    ")
		}
		if entry.Default {
			state += "  (default)"
		}
		rows = append(rows, []string{entry.Name, cmp.Or(entry.Origin, "-"), state})
	}
	flowColumns(deps.Stdout, rows)
	if err := load.MissingDefault(*root, deps.Bundled, project.settings); err != nil {
		return flowRefuse(deps, err)
	}
	return flowExitOK
}
