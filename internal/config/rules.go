package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Layer is one entry of the layer map under [rules.layers].
type Layer struct {
	Name      string
	Paths     []string // globs relative to the repository root
	MayUse    []string // layers an edge from this one may point to; meaningful when HasMayUse
	Bases     []string // unresolved base classes a file may extend; meaningful when HasBases
	HasMayUse bool
	HasBases  bool
	Tests     bool // every source needs its test module
	Checked   bool // false: known, but no rule checks its files
}

// Rules is the [rules] table: the layer map the rules read.
type Rules struct {
	Present bool    // a [rules] table exists, even an empty one
	Layers  []Layer // sorted by Name
}

// RulesKeys are the keys [rules] knows.
func RulesKeys() []string { return []string{"layers"} }

// LayerKeys are the keys one layer under [rules.layers] knows, sorted.
func LayerKeys() []string { return []string{"bases", "checked", "may_use", "paths", "tests"} }

// ReadRules reads [rules] of the project at root.
func ReadRules(root string) (Rules, error) {
	path := ManifestPath(root)
	doc, err := manifestDocument(path)
	if err != nil || doc == nil {
		return Rules{}, err
	}
	return ParseRules(path, doc)
}

// ParseRules reads [rules] from a decoded document, every finding at once.
// What only the file list can tell (a glob that matches nothing, two equally
// specific globs) is not asked here.
func ParseRules(path string, doc map[string]any) (Rules, error) {
	raw, present := doc["rules"]
	if !present {
		return Rules{}, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return Rules{}, fmt.Errorf("%s: [rules] must be a table, found %s", path, tomlType(raw))
	}
	rules := Rules{Present: true}
	var findings []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(RulesKeys(), key) {
			findings = append(findings, fmt.Sprintf("[rules] does not know %q; known: %s", key, strings.Join(RulesKeys(), ", ")))
		}
	}
	if value, present := table["layers"]; present {
		rules.Layers = parseLayers(value, &findings)
	}
	if len(findings) > 0 {
		return Rules{}, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return rules, nil
}

func parseLayers(value any, findings *[]string) []Layer {
	table, ok := value.(map[string]any)
	if !ok {
		*findings = append(*findings, fmt.Sprintf("[rules.layers] must be a table of layers, found %s", tomlType(value)))
		return nil
	}
	names := slices.Sorted(maps.Keys(table))
	var layers []Layer
	for _, name := range names {
		where := "[rules.layers." + name + "]"
		if !IsIdentifier(name) {
			*findings = append(*findings, fmt.Sprintf("%s: %q is not a layer name; a name is %s", where, name, IdentifierRule))
		}
		entry, ok := table[name].(map[string]any)
		if !ok {
			*findings = append(*findings, fmt.Sprintf("%s must be a table, found %s", where, tomlType(table[name])))
			continue
		}
		layers = append(layers, parseLayer(name, where, entry, names, findings))
	}
	return layers
}

func parseLayer(name, where string, entry map[string]any, names []string, findings *[]string) Layer {
	layer := Layer{Name: name, Checked: true}
	for _, key := range slices.Sorted(maps.Keys(entry)) {
		if !slices.Contains(LayerKeys(), key) {
			*findings = append(*findings, fmt.Sprintf("%s does not know %q; known: %s", where, key, strings.Join(LayerKeys(), ", ")))
		}
	}
	paths, present := entry["paths"]
	if !present {
		*findings = append(*findings, where+" needs paths")
	} else if globs, err := layerGlobs(paths); err != nil {
		*findings = append(*findings, where+" paths "+err.Error())
	} else {
		layer.Paths = globs
	}
	if value, present := entry["may_use"]; present {
		layer.HasMayUse = true
		layer.MayUse = layerList(where, "may_use", value, findings)
		for _, used := range layer.MayUse {
			if !slices.Contains(names, used) {
				*findings = append(*findings, fmt.Sprintf("%s may_use names %q, which is not under [rules.layers]; known: %s", where, used, strings.Join(names, ", ")))
			}
		}
	}
	if value, present := entry["bases"]; present {
		layer.HasBases = true
		layer.Bases = layerList(where, "bases", value, findings)
	}
	layer.Tests = layerFlag(where, "tests", entry, findings)
	if value, present := entry["checked"]; present {
		flag, ok := value.(bool)
		if !ok {
			*findings = append(*findings, fmt.Sprintf("%s checked must be a boolean, found %s", where, tomlType(value)))
		}
		layer.Checked = flag || !ok
	}
	if !layer.Checked {
		for _, key := range []string{"may_use", "bases", "tests"} {
			if _, set := entry[key]; set {
				*findings = append(*findings, fmt.Sprintf("%s: checked = false cannot carry %s; no rule checks the layer's files", where, key))
			}
		}
	}
	return layer
}

// layerGlobs is globList for a layer's paths, and refuses a backslash: a glob
// is written with slashes, and `godot\core\**` would silently match nothing.
func layerGlobs(value any) ([]string, error) {
	globs, err := globList(value)
	if err != nil {
		return nil, err
	}
	for i, glob := range globs {
		if strings.Contains(glob, `\`) {
			return nil, fmt.Errorf("glob #%d %q contains a backslash; write path separators as /", i+1, glob)
		}
	}
	return globs, nil
}

func layerFlag(where, key string, entry map[string]any, findings *[]string) bool {
	flag, err := optionalBool(entry, key, where, " ")
	if err != nil {
		*findings = append(*findings, err.Error())
	}
	return flag
}

// layerList reads a list of non-empty strings; an empty list is valid.
func layerList(where, key string, value any, findings *[]string) []string {
	items, ok := value.([]any)
	if !ok {
		*findings = append(*findings, fmt.Sprintf("%s %s must be a list of strings, found %s", where, key, tomlType(value)))
		return nil
	}
	list := make([]string, 0, len(items))
	for i, item := range items {
		text, ok := item.(string)
		if !ok || text == "" {
			*findings = append(*findings, fmt.Sprintf("%s %s #%d must be a non-empty string", where, key, i+1))
			continue
		}
		list = append(list, text)
	}
	return list
}
