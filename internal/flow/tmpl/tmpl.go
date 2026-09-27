package tmpl

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Template is a parsed text: literal runs and {{name}} placeholders.
type Template struct {
	parts []part
}

type part struct {
	literal string
	name    string // set: this part is a placeholder
}

// Parse reads the {{name}} placeholders out of a text; \{{ writes {{ literally.
//
// A placeholder holds a name and nothing else, not even whitespace.
// Instructions carry JSON and Go code, and a lenient reading would turn their
// braces into names nobody declared.
func Parse(text string) (Template, error) {
	var t Template
	var literal strings.Builder
	flush := func() {
		if literal.Len() > 0 {
			t.parts = append(t.parts, part{literal: literal.String()})
			literal.Reset()
		}
	}
	for i := 0; i < len(text); {
		switch {
		case strings.HasPrefix(text[i:], `\{{`):
			literal.WriteString("{{")
			i += 3
		case strings.HasPrefix(text[i:], "{{"):
			end := strings.Index(text[i+2:], "}}")
			if end < 0 {
				return Template{}, fmt.Errorf("unclosed {{ at byte %d", i)
			}
			name := text[i+2 : i+2+end]
			if !isName(name) {
				return Template{}, fmt.Errorf(
					"{{%s}} is not a placeholder; a name is letters, digits and _, and \\{{ writes literal braces", name)
			}
			flush()
			t.parts = append(t.parts, part{name: name})
			i += end + 4
		default:
			literal.WriteByte(text[i])
			i++
		}
	}
	flush()
	return t, nil
}

func isName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		letter := c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		digit := c >= '0' && c <= '9'
		if !letter && !(digit && i > 0) {
			return false
		}
	}
	return true
}

// Names are the placeholders of the text, sorted and each once.
func (t Template) Names() []string {
	var names []string
	for _, p := range t.parts {
		if p.name != "" && !slices.Contains(names, p.name) {
			names = append(names, p.name)
		}
	}
	slices.Sort(names)
	return names
}

// Render fills every placeholder in one pass: a value that itself contains
// {{...}} is written as it is and never read again.
func (t Template) Render(values map[string]any) (string, error) {
	var out strings.Builder
	for _, p := range t.parts {
		if p.name == "" {
			out.WriteString(p.literal)
			continue
		}
		value, ok := values[p.name]
		if !ok {
			return "", fmt.Errorf("no value for {{%s}}", p.name)
		}
		shown, err := show(p.name, value)
		if err != nil {
			return "", err
		}
		out.WriteString(shown)
	}
	return out.String(), nil
}

// show writes one value the way a prompt reads it: a list becomes one "- "
// line per entry.
func show(name string, value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case bool:
		return strconv.FormatBool(v), nil
	case []string:
		lines := make([]string, len(v))
		for i, item := range v {
			lines[i] = "- " + item
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("{{%s}} holds a %T, which a text cannot show", name, value)
	}
}
