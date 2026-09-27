package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// profiles are the tool profiles an agent node can name, each sorted and
// without duplicates: the list becomes part of a prompt and of a journal
// entry, so its order must not depend on anything. The table is built per
// call, so no hook that never runs a flow pays for it.
func profiles() map[string][]string {
	return map[string][]string{
		"read_only": {"Glob", "Grep", "Read"},
		"edit":      {"Edit", "Glob", "Grep", "Read", "Write"},
		"shell":     {"Bash", "Glob", "Grep", "Read"},
		"mcp":       {"Glob", "Grep", "Read"},
	}
}

// Tools is the tools a node with this profile may use. Only the mcp profile
// takes servers, one mcp__<server> each.
//
// The result is always a fresh slice. The profiles are the only ceiling between
// an agent node and Bash or Write, and a caller that edited a shared slice would
// widen it for every node after.
func Tools(profile string, servers []string) ([]string, error) {
	table := profiles()
	base, ok := table[profile]
	if !ok {
		return nil, fmt.Errorf("unknown tool profile %q; known profiles: %s", profile, joinKeys(table))
	}
	tools := slices.Clone(base)
	if profile == "mcp" {
		for _, server := range servers {
			tools = append(tools, "mcp__"+server)
		}
		slices.Sort(tools)
		tools = slices.Compact(tools)
	}
	return tools, nil
}

func joinKeys[T any](table map[string]T) string {
	return strings.Join(slices.Sorted(maps.Keys(table)), ", ")
}
