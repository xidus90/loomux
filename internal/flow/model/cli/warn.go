package cli

import "fmt"

// Warnings are the lines to print, before a run, for a node whose provider
// does not hold the tool list. The write barrier judges file-writing tools
// only, so no line claims it covers shell writes.
func Warnings(provider, node, profile string, tools []string) []string {
	var out []string
	switch provider {
	case "agy":
		out = append(out, fmt.Sprintf("warning: node %q runs on agy, which cannot enforce the tool profile %s; "+
			"only loomux's guard policy and the write barrier on file-writing tools limit it", node, profile))
	case "qoder":
		_, dropped := qoderTools(tools)
		for _, tool := range dropped {
			out = append(out, fmt.Sprintf("warning: node %q: qoder has no tool for %s; the node runs without it", node, tool))
		}
	}
	return out
}
