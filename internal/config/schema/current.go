package schema

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

// Origin says where the value an entry shows comes from.
type Origin string

const (
	Set     Origin = "set"
	Default Origin = "default"
	Preset  Origin = "preset"
	Unset   Origin = "unset"
)

// Entry is one key with the value a configuration gives it.
type Entry struct {
	Key    Key
	Value  string // TOML literal as written, or the default; "" when unset; TableList: one block per line
	Input  string // the value in the form edit.Render accepts: unquoted, lists as "a, b"
	Origin Origin
	Count  int // TableList: number of [[…]] blocks
}

// Current pairs every key with its value in text and says where that value
// comes from, in Keys() order. The value shown for a set key is re-encoded
// from the decoded document, so a list written across lines is shown on one.
func Current(text string) ([]Entry, error) {
	doc := map[string]any{}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf(".loomux/config.toml: not valid TOML: %w", err)
	}
	var out []Entry
	for _, k := range Keys() {
		value, found := lookup(doc, k)
		e := Entry{Key: k, Origin: Unset}
		switch {
		case k.Kind == TableList:
			list, _ := value.([]map[string]any)
			e.Count = len(list)
			blocks := make([]string, len(list))
			for i, block := range list {
				blocks[i] = fmt.Sprint(block)
			}
			e.Value = strings.Join(blocks, "\n")
			if found {
				e.Origin = Set
			}
		case found:
			e.Origin, e.Value, e.Input = Set, literal(k, value), inputForm(value)
		// Before the default: the verify tables are filled per language by
		// the presets, and the schema default only mirrors one of them.
		case k.Section == "verify" && k.Kind == Table:
			e.Origin = Preset
		case k.Default != "":
			e.Origin, e.Value = Default, k.Default
			var decoded map[string]any
			if _, err := toml.Decode("v = "+k.Default, &decoded); err == nil {
				e.Input = inputForm(decoded["v"])
			}
		}
		out = append(out, e)
	}
	return out, nil
}

func lookup(doc map[string]any, k Key) (any, bool) {
	var node any = doc
	path := strings.Split(k.Section, ".")
	if k.Name != "" {
		path = append(path, k.Name)
	}
	for _, part := range path {
		table, ok := node.(map[string]any)
		if !ok {
			return nil, false
		}
		if node, ok = table[part]; !ok {
			return nil, false
		}
	}
	return node, true
}

// inputForm is value the way a human types it and edit.Render reads it:
// a string without quotes or escapes, a list as "a, b", anything else as Go
// prints it. Taking the quotes off the TOML literal instead would leave its
// escapes in, and Render would escape them a second time.
func inputForm(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []any:
		parts := make([]string, len(v))
		for i, item := range v {
			parts[i] = fmt.Sprint(item)
		}
		return JoinList(parts)
	default:
		return fmt.Sprint(v)
	}
}

// literal is value as one line of TOML. The encoder writes a table as a
// [v] section over several lines rather than inline, so a table is shown
// the way Go prints the map instead.
func literal(k Key, value any) string {
	if k.Kind == Table {
		return fmt.Sprint(value)
	}
	var b strings.Builder
	_ = toml.NewEncoder(&b).Encode(map[string]any{"v": value})
	return strings.TrimSpace(strings.TrimPrefix(b.String(), "v = "))
}
