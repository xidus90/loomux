package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// GuardKeys are the keys [guard] knows, sorted.
func GuardKeys() []string { return []string{"mode"} }

// GuardModes are the values [guard] mode takes, the default first.
func GuardModes() []string { return []string{"default", "strict"} }

// parseGuard reads [guard] from the table ReadPolicy decoded and answers
// whether it asks for the strict mode, every finding at once. Only the two
// modes are values: a guard that fell back to default on a typo would look
// strict and not be. The table arrives undecoded because the TOML decoder
// drops a scalar `guard = 3` into a map without a word.
func parseGuard(path string, value any) (bool, error) {
	table, ok := value.(map[string]any)
	if value != nil && !ok {
		return false, fmt.Errorf("%s: [guard] must be a table, found %s", path, tomlType(value))
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
	if len(findings) > 0 {
		return false, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return mode == "strict", nil
}
