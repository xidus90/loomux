package schema

import (
	"fmt"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/config"
)

// SplitList reads a list the way a human types it: items apart at commas.
// A comma inside a {…} group belongs to its item -- the lists hold globs,
// and docs/**/*.{md,txt} is one of them; a } without its { closes nothing.
// An item that starts with a double quote is a TOML basic string up to its
// closing quote, which is how an item with a comma, an empty item or one
// with blanks at its ends is typed. An empty item without quotes is no item:
// "a," is the list of a alone.
func SplitList(input string) ([]string, error) {
	var items []string
	add := func(raw string) error {
		item, ok, err := listItem(raw)
		if ok {
			items = append(items, item)
		}
		return err
	}
	depth, start, quoted := 0, 0, false
	for i := 0; i < len(input); i++ {
		switch c := input[i]; {
		case quoted && c == '\\':
			// The escaped byte cannot close the string.
			i++
		case quoted:
			quoted = c != '"'
		case c == '"' && strings.TrimSpace(input[start:i]) == "":
			quoted = true
		case c == '{':
			depth++
		case c == '}' && depth > 0:
			depth--
		case c == ',' && depth == 0:
			if err := add(input[start:i]); err != nil {
				return nil, err
			}
			start = i + 1
		}
	}
	if quoted {
		return nil, fmt.Errorf("%q: a quoted item is not closed", strings.TrimSpace(input[start:]))
	}
	if err := add(input[start:]); err != nil {
		return nil, err
	}
	return items, nil
}

// listItem is one raw item between commas: trimmed, a quoted one decoded,
// and ok false for an empty one without quotes.
func listItem(raw string) (string, bool, error) {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, `"`) {
		return s, s != "", nil
	}
	// Nothing may follow the closing quote, a TOML comment included.
	var doc map[string]any
	if _, err := toml.Decode("v = "+s, &doc); err != nil || !strings.HasSuffix(s, `"`) {
		return "", false, fmt.Errorf("%s: not one quoted item", s)
	}
	// "v = " followed by a closed basic string and nothing but blanks can
	// only decode to a string.
	return doc["v"].(string), true, nil
}

// JoinList is the input form of items, the one SplitList reads back: an item
// SplitList would not return as it stands is written in quotes, and so is
// one that leaves a { open, which would take the commas after it.
func JoinList(items []string) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item
		if got, err := SplitList(item); err != nil || !slices.Equal(got, []string{item}) || leavesBraceOpen(item) {
			parts[i] = config.QuoteTOML(item)
		}
	}
	return strings.Join(parts, ", ")
}

// leavesBraceOpen says whether s has a { that no } closes, counted the way
// SplitList counts its groups.
func leavesBraceOpen(s string) bool {
	depth := 0
	for _, c := range s {
		switch {
		case c == '{':
			depth++
		case c == '}' && depth > 0:
			depth--
		}
	}
	return depth > 0
}
