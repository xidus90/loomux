package verify

import (
	_ "embed"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

//go:embed frameworks.toml
var frameworksText []byte

// Framework is one row of the matrix: which line of a stack's engine or
// runtime a third-party framework carries, and where that claim comes from.
type Framework struct {
	Stack, Name, Kind string
	// Found is the when_found pattern (<file>:<text>) a lane of this kind
	// detects the framework with, empty where the lane needs no detection.
	Found string
	// Carries is free text for people and docs, not something code compares.
	Carries string
	Source  string
	// Proven means this repository ran the pairing itself, not just read it.
	Proven bool
}

// LoadFrameworks reads the embedded matrix in file order.
func LoadFrameworks() ([]Framework, error) {
	return parseFrameworks(frameworksText, "frameworks.toml")
}

// parseFrameworks refuses anything that could pass for configured but is not:
// unknown keys, missing fields, unknown kinds and repeated rows.
func parseFrameworks(data []byte, path string) ([]Framework, error) {
	var doc struct {
		Framework []Framework `toml:"framework"`
	}
	md, err := toml.Decode(string(data), &doc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	lines := strings.Split(string(data), "\n")
	if un := md.Undecoded(); len(un) > 0 {
		key := un[0][len(un[0])-1]
		return nil, fmt.Errorf("%s: unknown key %q", place(path, keyLine(lines, key)), key)
	}
	headers := rowLines(lines)
	seen := map[string]bool{}
	for i, f := range doc.Framework {
		line := 0
		if i < len(headers) {
			line = headers[i]
		}
		at := place(path, line)
		// A slice, not a map: the field the error names has to be the same one
		// on every run.
		for _, field := range []struct{ name, value string }{
			{"stack", f.Stack}, {"name", f.Name}, {"carries", f.Carries}, {"source", f.Source},
		} {
			if field.value == "" {
				return nil, fmt.Errorf("%s: row has no %s", at, field.name)
			}
		}
		if !slices.Contains([]string{"lint", "test", "coverage"}, f.Kind) {
			return nil, fmt.Errorf("%s: %s/%s has kind %q; kinds are lint, test, coverage", at, f.Stack, f.Name, f.Kind)
		}
		id := f.Stack + "/" + f.Name
		if seen[id] {
			return nil, fmt.Errorf("%s: duplicate row %s", at, id)
		}
		seen[id] = true
	}
	return doc.Framework, nil
}

var rowHeader = regexp.MustCompile(`^\s*\[\[\s*framework\s*\]\]`)

// rowLines gives the 1-based line of each [[framework]] header, the nearest
// position a decoded row has, since the decoder keeps none. It can find fewer
// headers than there are rows (inline arrays), so callers must bounds-check.
func rowLines(lines []string) []int {
	var out []int
	for i, l := range lines {
		if rowHeader.MatchString(l) {
			out = append(out, i+1)
		}
	}
	return out
}

// place names the file and, when a line is known, the line.
func place(path string, line int) string {
	if line == 0 {
		return path
	}
	return fmt.Sprintf("%s:%d", path, line)
}

// keyLine is the 1-based line that assigns key or opens a table of that name,
// or 0 when none does (the key sits inside an inline table).
func keyLine(lines []string, key string) int {
	name := regexp.QuoteMeta(key)
	re := regexp.MustCompile(`^\s*(` + name + `\s*=|\[\[?\s*` + name + `\s*\]\]?)`)
	for i, l := range lines {
		if re.MatchString(l) {
			return i + 1
		}
	}
	return 0
}
