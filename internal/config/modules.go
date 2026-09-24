package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"

	"github.com/BurntSushi/toml"
)

// Modules is the [modules] table: which parts of loomux run in a project.
// The guard is not among them. The write barrier is global -- it protects
// the read-only areas of every repository -- so no project may switch it off.
type Modules struct {
	Hooks bool
	Brain bool
	Graph bool
}

// AllModules is what a project that says nothing gets: everything on, so a
// repository without [modules] behaves as it did before the table existed.
func AllModules() Modules { return Modules{Hooks: true, Brain: true, Graph: true} }

// ModuleKeys are the keys [modules] knows, sorted.
func ModuleKeys() []string { return []string{"brain", "graph", "hooks"} }

// ReadModules reads [modules] of the project at root. It is the one reader
// on the per-edit path, so it decodes nothing but the document and judges
// nothing but this table.
func ReadModules(root string) (Modules, error) {
	path := ManifestPath(root)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return AllModules(), nil
	}
	if err != nil {
		return Modules{}, fmt.Errorf("%s: %w", path, err)
	}
	doc := map[string]any{}
	if err := toml.Unmarshal(data, &doc); err != nil {
		return Modules{}, fmt.Errorf("%s: not valid TOML: %w", path, err)
	}
	return ParseModules(path, doc)
}

// ParseModules reads [modules] from a decoded document. An unknown key is
// refused: `graf = false` that silently left the graph on would look
// configured and not be.
func ParseModules(path string, doc map[string]any) (Modules, error) {
	modules := AllModules()
	raw, present := doc["modules"]
	if !present {
		return modules, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return Modules{}, fmt.Errorf("%s: [modules] must be a table, found %s", path, tomlType(raw))
	}
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !slices.Contains(ModuleKeys(), key) {
			return Modules{}, fmt.Errorf("%s: [modules] does not know %q; known: %v", path, key, ModuleKeys())
		}
		on, ok := table[key].(bool)
		if !ok {
			return Modules{}, fmt.Errorf("%s: [modules] %s must be true or false, found %s", path, key, tomlType(table[key]))
		}
		switch key {
		case "hooks":
			modules.Hooks = on
		case "brain":
			modules.Brain = on
		default:
			modules.Graph = on
		}
	}
	return modules, nil
}
