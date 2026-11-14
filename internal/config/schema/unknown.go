package schema

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/config"
)

// Unknown lists the keys of text that none of keys describes, as dotted
// paths in name order. A table that holds no key the schema knows is listed
// as that table, not leaf by leaf: it is one thing the schema has no form
// for. The readers that run in operation judge what a file may say; this
// only tells a human which parts of it the key list does not show.
func Unknown(keys []Key, text string) ([]string, error) {
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf("not valid TOML: %w", err)
	}
	var out []string
	unknownIn(keys, nil, doc, &out)
	return out, nil
}

// unknownIn walks one table, whose own path is prefix.
func unknownIn(keys []Key, prefix []string, table map[string]any, out *[]string) {
	for _, name := range slices.Sorted(maps.Keys(table)) {
		path := append(slices.Clone(prefix), name)
		id := strings.Join(path, ".")
		switch sub, isTable := table[name].(map[string]any); {
		case known(keys, id):
		case isTable && leadsToKnown(keys, path):
			unknownIn(keys, path, sub, out)
		default:
			*out = append(*out, id)
		}
	}
}

// known says whether id is a key of the schema, or lies inside a table or a
// list of tables the schema shows as one entry.
func known(keys []Key, id string) bool {
	if _, ok := Match(keys, id); ok {
		return true
	}
	return slices.ContainsFunc(keys, func(k Key) bool {
		return (k.Kind == Table || k.Kind == TableList) && (k.ID() == id || strings.HasPrefix(id, k.ID()+"."))
	})
}

// leadsToKnown says whether path is the beginning of the ID of a key, a name
// standing where the key has its wildcard.
func leadsToKnown(keys []Key, path []string) bool {
	return slices.ContainsFunc(keys, func(k Key) bool {
		pattern := strings.Split(k.ID(), ".")
		if len(path) >= len(pattern) {
			return false
		}
		for i, part := range path {
			if pattern[i] != part && !(pattern[i] == Wildcard && config.IsIdentifier(part)) {
				return false
			}
		}
		return true
	})
}
