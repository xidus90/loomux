package config

import (
	"fmt"
	"strconv"
	"time"
)

// tomlType names a decoded value by its TOML type, the vocabulary a person
// who wrote the file can find in it. BurntSushi hands every date and time
// back as time.Time, and both []any and []map[string]any are arrays --
// nothing else comes out of a document decoded into a map.
func tomlType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case int64:
		return "integer"
	case float64:
		return "float"
	case bool:
		return "boolean"
	case time.Time:
		return "datetime"
	case map[string]any:
		return "table"
	}
	return "array"
}

// found names a value that should have been a non-empty string: a string
// is quoted -- the only one that gets here is the empty one or a
// value outside an allowed set -- and anything else is named by its type.
func found(value any) string {
	if text, ok := value.(string); ok {
		return strconv.Quote(text)
	}
	return tomlType(value)
}

// requiredString is optionalString for a key that has to be there. owner
// names the table in the refusal and sep joins it to the key, because the
// registry writes `[[area]] #1: scope` and a manifest `[area] scope`.
func requiredString(table map[string]any, key, owner, sep string) (string, error) {
	if _, present := table[key]; !present {
		return "", fmt.Errorf("%s is missing %q", owner, key)
	}
	return optionalString(table, key, owner, sep)
}

// optionalString answers "" for an absent key and refuses a present one
// that is no string or the empty string.
func optionalString(table map[string]any, key, owner, sep string) (string, error) {
	value, present := table[key]
	if !present {
		return "", nil
	}
	if text, ok := value.(string); ok && text != "" {
		return text, nil
	}
	return "", fmt.Errorf("%s%s%s must be a non-empty string, found %s", owner, sep, key, found(value))
}

// optionalBool answers false for an absent key and refuses a present one
// that is not true or false: a flag spelt "yes" is a typo, not a yes.
func optionalBool(table map[string]any, key, owner, sep string) (bool, error) {
	value, present := table[key]
	if !present {
		return false, nil
	}
	flag, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("%s%s%s must be a boolean, found %s", owner, sep, key, tomlType(value))
	}
	return flag, nil
}
