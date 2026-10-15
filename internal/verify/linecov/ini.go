package linecov

import (
	"fmt"
	"strings"
)

// readINI reads configparser-style files (.coveragerc, setup.cfg, tox.ini,
// gcovr.cfg, .lcovrc): keys before the first section belong to section "",
// keys are lower-cased, an indented line continues the value above, and a
// repeated key adds to its list.
func readINI(content []byte, w Where) (Values, bool, error) {
	vals := Values{}
	section, key := "", ""
	found := w.Section == ""
	for i, raw := range strings.Split(text(content), "\n") {
		line := strings.TrimRight(raw, " \t\r")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "" || trimmed[0] == '#' || trimmed[0] == ';':
		case line[0] == '[':
			end := strings.IndexByte(trimmed, ']')
			if end < 0 {
				return nil, false, fmt.Errorf("line %d: unclosed section header", i+1)
			}
			section, key = strings.TrimSpace(trimmed[1:end]), ""
			found = found || section == w.Section
		case line[0] == ' ' || line[0] == '\t':
			if key == "" {
				return nil, false, fmt.Errorf("line %d: continuation without a key", i+1)
			}
			if section == w.Section {
				vals.add(key, splitList(trimmed)...)
			}
		default:
			k, v, ok := cutAssign(trimmed)
			if !ok {
				return nil, false, fmt.Errorf("line %d: neither = nor :", i+1)
			}
			key = strings.ToLower(k)
			if section == w.Section {
				vals.add(key, splitList(v)...)
			}
		}
	}
	return vals, found, nil
}

func cutAssign(s string) (string, string, bool) {
	i := strings.IndexAny(s, "=:")
	if i <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(s[:i]), s[i+1:], true
}
