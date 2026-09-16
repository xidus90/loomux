package guard

import (
	"fmt"
	"strings"
)

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
