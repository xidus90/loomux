package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// GrepKeys are the keys [grep] knows, sorted.
func GrepKeys() []string { return []string{"max_filesize"} }

// ReadGrepMaxFilesize reads [grep] max_filesize of the project at root. nil
// means the project declares no limit, so the default applies; 0 means
// unlimited.
func ReadGrepMaxFilesize(root string) (*int64, error) {
	path := ManifestPath(root)
	doc, err := manifestDocument(path)
	if err != nil || doc == nil {
		return nil, err
	}
	return ParseGrepMaxFilesize(path, doc)
}

// ParseGrepMaxFilesize reads [grep] from a decoded document and reports every
// finding at once. The limit is what the search corpus reads for a call that
// gives no flag.
func ParseGrepMaxFilesize(path string, doc map[string]any) (*int64, error) {
	raw, present := doc["grep"]
	if !present {
		return nil, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: [grep] must be a table, found %s", path, tomlType(raw))
	}
	var findings []string
	for _, key := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(GrepKeys(), key) {
			findings = append(findings, fmt.Sprintf("[grep] does not know %q; known: %s", key, strings.Join(GrepKeys(), ", ")))
		}
	}
	var limit *int64
	if value, present := table["max_filesize"]; present {
		n, ok := value.(int64)
		switch {
		case !ok:
			findings = append(findings, fmt.Sprintf("[grep] max_filesize must be an integer >= 0, found %s", tomlType(value)))
		case n < 0:
			findings = append(findings, fmt.Sprintf("[grep] max_filesize must be an integer >= 0, found %d", n))
		default:
			limit = &n
		}
	}
	if len(findings) > 0 {
		return nil, fmt.Errorf("%s: %s", path, strings.Join(findings, "; "))
	}
	return limit, nil
}
