package guard

import (
	"fmt"
	"strconv"
	"strings"
)

// truthy is Python's `bool(value)` over what a TOML document can hold.
// The registration reads four of its flags that way
// (`bool(entry.get("readonly", False))`, src/brain/registry.py:76-79) and
// `wiki_layout` opens with `if not value` (src/brain/manifest.py:68), so a
// reader that asked for the *type* instead would answer a different
// registry than the barrier it replaces: `readonly = "yes"` is a
// read-only area there, and `wiki = 0` is an unsaid layout.
func truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	}
	// Everything else a TOML decoder can hand back is an object Python
	// calls true without asking: a datetime has no `__bool__`.
	return true
}

// pyRepr is `repr(value)` for the values a TOML document carries. The
// messages this module mirrors interpolate with `!r`, and a reader who is
// handed one of them has to be able to find the line they wrote.
//
// Its limit is named rather than hidden: a table is rendered by Go's own
// `%v`, because Python would print it in the file's order and a Go map
// has none. No message reachable from this barrier interpolates a table
// today -- the four that interpolate at all are handed a scope, a key, a
// mode or a glob element.
func pyRepr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case string:
		return pyReprString(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return pyReprFloat(typed)
	case []any:
		parts := make([]string, len(typed))
		for i, element := range typed {
			parts[i] = pyRepr(element)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprintf("%v", value)
}

// pyReprString is `repr` of a str: single quotes, unless the text carries
// a single quote and no double one.
func pyReprString(value string) string {
	quote := byte('\'')
	if strings.Contains(value, "'") && !strings.Contains(value, "\"") {
		quote = '"'
	}
	out := strings.Builder{}
	out.WriteByte(quote)
	for _, r := range value {
		switch {
		case r == rune(quote) || r == '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r == '\t':
			out.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			out.WriteString(fmt.Sprintf(`\x%02x`, r))
		default:
			out.WriteRune(r)
		}
	}
	out.WriteByte(quote)
	return out.String()
}

// pyReprFloat is `repr` of a float: the shortest text that reads back to
// the same number, and a `.0` where that text would otherwise look like
// an integer -- which is the one place Go's `'g'` and Python disagree.
func pyReprFloat(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eEni") {
		text += ".0"
	}
	return text
}

// pythonJSONString renders one string the way `json.dumps` does with its
// defaults: `ensure_ascii=True`, so every character outside ASCII becomes
// a `\uXXXX` escape and one outside the basic plane becomes the surrogate
// pair Python writes for it.
//
// Hand-rendered rather than handed to `encoding/json`, because the host
// reads these bytes and the contract is the byte shape, not an equivalent
// object: Go escapes `<`, `>` and `&` where Python leaves them, and Go
// emits real UTF-8 where Python escapes it. Measured against `json.dumps`
// over a string carrying all of them; the parity test holds the two
// renderings together.
func pythonJSONString(value string) string {
	out := strings.Builder{}
	out.WriteByte('"')
	for _, r := range value {
		switch {
		case r == '"':
			out.WriteString(`\"`)
		case r == '\\':
			out.WriteString(`\\`)
		case r == '\b':
			out.WriteString(`\b`)
		case r == '\f':
			out.WriteString(`\f`)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r == '\t':
			out.WriteString(`\t`)
		case r < 0x7f && r >= 0x20:
			out.WriteRune(r)
		case r <= 0xffff:
			// `RuneError` stands here for every byte that is not valid
			// UTF-8. Python decoded its own text before it ever got
			// here, so there is no spelling of the difference that both
			// sides could reach; the substitution is what Go's range
			// already made, not a choice taken here.
			fmt.Fprintf(&out, `\u%04x`, r)
		default:
			high, low := utf16Pair(r)
			fmt.Fprintf(&out, `\u%04x\u%04x`, high, low)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// utf16Pair is the surrogate pair `json.dumps` writes for a character
// outside the basic multilingual plane.
func utf16Pair(r rune) (rune, rune) {
	r -= 0x10000
	return 0xd800 + (r >> 10), 0xdc00 + (r & 0x3ff)
}

// pythonTypeName is `type(payload).__name__` for a decoded JSON value.
// JSON has one number type and Python has two,
// so the text of the number decides -- which is why the payload is
// decoded with `UseNumber` rather than into a float.
func pythonTypeName(value any) string {
	switch typed := value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case []any:
		return "list"
	case map[string]any:
		return "dict"
	case jsonNumber:
		if strings.ContainsAny(string(typed), ".eE") {
			return "float"
		}
		return "int"
	}
	return fmt.Sprintf("%T", value)
}
