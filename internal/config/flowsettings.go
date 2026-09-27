package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// FlowSettings is the [flow] table: which flow runs without a name, and
// which bundled flows a project flow of the same name may hide or overlay.
type FlowSettings struct {
	Default   string
	Overrides []string
}

// FlowKeys are the keys [flow] knows, sorted.
func FlowKeys() []string { return []string{"default", "overrides"} }

// Allows reports whether a project flow may hide or overlay the bundled flow
// name. The answer lives in this file because only a human writes it: a
// bundled flow's gates stay, since the party a gate asks may not remove it.
func (s FlowSettings) Allows(name string) bool { return slices.Contains(s.Overrides, name) }

// ReadFlowSettings reads [flow] of the project at root.
func ReadFlowSettings(root string) (FlowSettings, error) {
	path := ManifestPath(root)
	doc, err := manifestDocument(path)
	if err != nil || doc == nil {
		return FlowSettings{}, err
	}
	return ParseFlowSettings(path, doc)
}

// ParseFlowSettings reads [flow] from a decoded document, every finding at
// once. Whether default names a flow that exists is not asked here: this
// reader reads no flows.
func ParseFlowSettings(path string, doc map[string]any) (FlowSettings, error) {
	raw, present := doc["flow"]
	if !present {
		return FlowSettings{}, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return FlowSettings{}, fmt.Errorf("%s: [flow] must be a table, found %s", path, tomlType(raw))
	}
	var settings FlowSettings
	var findings []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(FlowKeys(), key) {
			findings = append(findings, fmt.Sprintf("[flow] does not know %q; known: %s", key, strings.Join(FlowKeys(), ", ")))
		}
	}
	if value, present := table["default"]; present {
		name, ok := value.(string)
		if !ok || !IsFlowName(name) {
			findings = append(findings, "[flow] default must be a flow name; a flow name is "+FlowNameRule)
		}
		settings.Default = name
	}
	if value, present := table["overrides"]; present {
		items, ok := value.([]any)
		if !ok {
			findings = append(findings, fmt.Sprintf("[flow] overrides must be a list of flow names, found %s", tomlType(value)))
		}
		for i, item := range items {
			name, ok := item.(string)
			switch {
			case !ok || !IsFlowName(name):
				findings = append(findings, fmt.Sprintf("[flow] overrides #%d must be a flow name; a flow name is %s", i+1, FlowNameRule))
			case slices.Contains(settings.Overrides, name):
				findings = append(findings, fmt.Sprintf("[flow] overrides names %q twice", name))
			default:
				settings.Overrides = append(settings.Overrides, name)
			}
		}
	}
	if len(findings) > 0 {
		return FlowSettings{}, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return settings, nil
}
