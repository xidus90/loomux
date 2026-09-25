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
		items, err := schema.SplitList(input)
		if err != nil {
			return "", err
		}
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = config.QuoteTOML(item)
		}
		return "[" + strings.Join(parts, ", ") + "]", nil
	default:
		return "", fmt.Errorf("a table is edited by hand")
	}
}
