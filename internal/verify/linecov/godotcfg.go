package linecov

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var godotString = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)

// readGodotCfg reads project.godot: [section], key=value, values that are
// strings, booleans or PackedStringArray(...), which may run over lines.
func readGodotCfg(content []byte, w Where) (Values, bool, error) {
	vals := Values{}
	section, pending := "", ""
	found := false
	for i, raw := range strings.Split(text(content), "\n") {
		line := strings.TrimSpace(raw)
		if pending != "" {
			if pending += "\n" + line; open(pending) {
				continue
			}
			line, pending = pending, ""
		} else if line == "" || line[0] == ';' {
			continue
		} else if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end < 0 {
				return nil, false, fmt.Errorf("line %d: unclosed section header", i+1)
			}
			section = line[1:end]
			found = found || section == w.Section
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, false, fmt.Errorf("line %d: no =", i+1)
		}
		if open(v) {
			pending = line
			continue
		}
		if section == w.Section {
			key := strings.TrimSpace(k)
			vals.add(key, godotValue(strings.TrimSpace(v))...)
			if w.KeepEmpty && len(vals[key]) == 0 {
				vals[key] = []string{""}
			}
		}
	}
	if pending != "" {
		return nil, false, fmt.Errorf("unclosed value: %q", pending)
	}
	return vals, found, nil
}

// open says whether a value still has a bracket ((, [ or {) or a string to
// close, so dictionaries, arrays and strings may run over lines.
func open(v string) bool {
	depth, quoted := 0, false
	for i := 0; i < len(v); i++ {
		switch c := v[i]; {
		case quoted && c == '\\':
			i++
		case c == '"':
			quoted = !quoted
		case !quoted && (c == '(' || c == '[' || c == '{'):
			depth++
		case !quoted && (c == ')' || c == ']' || c == '}'):
			depth--
		}
	}
	return depth > 0 || quoted
}

func godotValue(v string) []string {
	if !strings.Contains(v, `"`) {
		return []string{v}
	}
	var out []string
	for _, m := range godotString.FindAllStringSubmatch(v, -1) {
		s, err := strconv.Unquote(`"` + m[1] + `"`)
		if err != nil {
			s = m[1]
		}
		out = append(out, s)
	}
	return out
}
