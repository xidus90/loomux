package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// GuardKeys are the keys [guard] knows, sorted.
func GuardKeys() []string { return []string{"graph_hint", "mode"} }

// guardSettings is what [guard] says: the strict mode, and whether the
// hint toward the code graph is switched off.
type guardSettings struct {
	strict, graphHintOff bool
}

// GuardModes are the values [guard] mode takes, the default first.
func GuardModes() []string { return []string{"default", "strict"} }

// parseGuard reads [guard] from the table ReadPolicy decoded, every finding
// at once. Only the two modes are values: a guard that fell back to default
// on a typo would look strict and not be. The table arrives undecoded
// because the TOML decoder drops a scalar `guard = 3` into a map without a
// word.
func parseGuard(path string, value any) (guardSettings, error) {
	table, ok := value.(map[string]any)
	if value != nil && !ok {
		return guardSettings{}, fmt.Errorf("%s: [guard] must be a table, found %s", path, tomlType(value))
	}
	var findings []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(GuardKeys(), key) {
			findings = append(findings, fmt.Sprintf("[guard] does not know %q; known: %s", key, strings.Join(GuardKeys(), ", ")))
		}
	}
	mode, isString := table["mode"].(string)
	if raw, present := table["mode"]; present && !slices.Contains(GuardModes(), mode) {
		// A string is named as written, any other value by its type.
		found := tomlType(raw)
		if isString {
			found = fmt.Sprintf("%q", mode)
		}
		findings = append(findings, "[guard] mode must be one of "+strings.Join(GuardModes(), ", ")+", found "+found)
	}
	hint, isBool := table["graph_hint"].(bool)
	if raw, present := table["graph_hint"]; present && !isBool {
		findings = append(findings, "[guard] graph_hint must be a boolean, found "+tomlType(raw))
	}
	if len(findings) > 0 {
		return guardSettings{}, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return guardSettings{strict: mode == "strict", graphHintOff: isBool && !hint}, nil
}
