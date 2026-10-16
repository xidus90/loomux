package linecov

import "strings"

// Values maps a settings key to its values; a scalar is a list of one.
type Values map[string][]string

func (v Values) add(key string, vals ...string) {
	for _, s := range vals {
		if s = strings.TrimSpace(s); s != "" {
			v[key] = append(v[key], s)
		}
	}
}

// Where names the part of a settings file a profile reads: a section (ini,
// godotcfg), a dotted table (toml), or a path (xml: slash elements from the
// root element, json: dotted keys).
type Where struct {
	Section string
	Table   string
	Path    string
	// KeepEmpty makes godotcfg keep a key set to an empty string as one
	// empty value, so it differs from an absent key.
	KeepEmpty bool
}

// Syntax reads the values under w; found says whether w exists in the file,
// so a lookup order can pass over a file that does not configure the tool.
// A new syntax is one function and one entry in syntaxes.
type Syntax func(content []byte, w Where) (vals Values, found bool, err error)

func syntaxes() map[string]Syntax {
	return map[string]Syntax{
		"ini": readINI, "godotcfg": readGodotCfg,
		"toml": readTOML, "json": readJSON, "xml": readXML,
	}
}

// splitList splits a list value on commas and newlines.
// ponytail: regular expressions pass through here too (gcovr's filter and
// exclude, .lcovrc's lcov_excl_*), so one holding a comma, as in x{1,3}, is
// split; split per key when such a pattern turns up.
func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' })
}
