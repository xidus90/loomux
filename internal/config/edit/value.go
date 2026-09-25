package edit

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/schema"
)

// Render turns what a human typed into the TOML literal for kind. It checks
// the form only; whether the value is allowed stays with the reader of the key.
func Render(kind schema.Kind, input string) (string, error) {
	input = strings.TrimSpace(input)
	switch kind {
	case schema.String, schema.Enum:
		return config.QuoteTOML(input), nil
	case schema.Int:
		n, err := strconv.Atoi(input)
		if err != nil {
			return "", fmt.Errorf("%q is not a whole number", input)
		}
		// The canonical form lets a caller compare against a key's default
		// by text; +600 and 007 would otherwise slip past as new values.
		return strconv.Itoa(n), nil
	case schema.Bool:
		if input != "true" && input != "false" {
			return "", fmt.Errorf("%q is not true or false", input)
		}
		return input, nil
	case schema.StringList:
		// An empty item is a stray comma, not an item: "a," is the list of a.
		var parts []string
		for _, p := range splitList(input) {
			if p = strings.TrimSpace(p); p != "" {
				parts = append(parts, config.QuoteTOML(p))
			}
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	default:
		return "", fmt.Errorf("a table is edited by hand")
	}
}

// splitList cuts a typed list at its commas, except those inside a {…}
// group: the lists hold globs, and docs/**/*.{md,txt} is one of them. A }
// without its { closes nothing.
func splitList(input string) []string {
	var parts []string
	depth, start := 0, 0
	for i := 0; i < len(input); i++ {
		switch c := input[i]; {
		case c == '{':
			depth++
		case c == '}' && depth > 0:
			depth--
		case c == ',' && depth == 0:
			parts = append(parts, input[start:i])
			start = i + 1
		}
	}
	return append(parts, input[start:])
}
