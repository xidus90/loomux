// Package namecheck holds the repository to one promise: the names of the
// tools loomux replaced, and of the project its code graph is ported from,
// stand only where an exception lets them.
package namecheck

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// The names of the two tools loomux replaced, and of the project its code
// graph is ported from: each may stand only in an exception. `graft` after
// an underscore is git's GIT_GRAFT_FILE, not the project.
var predecessorNames = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)ultraloom|ultra-brain|ulguard|ulinit|ulflow|brain-mcp|brain guard|ultraloomowned|\.brain\.toml|\.ultra-brain|specs-u[lb]/|plans-u[lb]/|bench-ub/|(?:^|[^_])graft`)
})

var changelogEntry = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^## \[(\d+\.\d+\.\d+)(?:-beta(?:\.\d+)?)?\]`)
})

// An Exception lets a name stand where it may: a file, or every
// file below a folder when Path ends in "/"; only lines Line matches, when it
// is set; only one line under the changelog entry Section, when that is set.
type Exception struct {
	Path    string
	Line    *regexp.Regexp
	Section string
	Owner   string
}

func (e Exception) covers(path string) bool {
	if strings.HasSuffix(e.Path, "/") {
		return strings.HasPrefix(path, e.Path)
	}
	return path == e.Path
}

// References names every line of files that carries a name from the list and
// no exception allows, as "<path>:<line>: <text>".
func References(files []string, read func(string) ([]byte, error), exceptions []Exception) ([]string, error) {
	var found []string
	for _, path := range files {
		data, err := read(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		found = append(found, inFile(path, string(data), exceptions)...)
	}
	slices.Sort(found)
	return found, nil
}

func inFile(path, text string, exceptions []Exception) []string {
	var own []Exception
	for _, e := range exceptions {
		if e.covers(path) {
			own = append(own, e)
		}
	}
	var found []string
	section, used := "", map[string]bool{}
	for i, line := range strings.Split(text, "\n") {
		if m := changelogEntry().FindStringSubmatch(line); m != nil {
			section = m[1]
		}
		if !predecessorNames().MatchString(line) || allowed(own, line, section, used) {
			continue
		}
		found = append(found, fmt.Sprintf("%s:%d: %s", path, i+1, strings.TrimRight(line, "\r")))
	}
	return found
}

// allowed reports whether one of the file's exceptions takes the line; a
// section exception takes the first hit under its entry and no other.
func allowed(own []Exception, line, section string, used map[string]bool) bool {
	for _, e := range own {
		switch {
		case e.Section != "":
			if e.Section == section && !used[section] {
				used[section] = true
				return true
			}
		case e.Line != nil:
			if e.Line.MatchString(line) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// ideaLine is the one sentence per README and architecture page that names
// where the code graph's idea came from.
var ideaLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)(inspired by|angeregt von) \[?trailhq/Graft`)
})

// exceptions are the places a name may stand: the measurement chronicle,
// the five changelog lines that are history, the license notice of the
// ported code and the sentence that names its idea.
var exceptions = sync.OnceValue(func() []Exception {
	return []Exception{
		{Path: "docs/en/benchmarks.md", Owner: "history"},
		{Path: "docs/de/benchmarks.md", Owner: "history"},
		{Path: "testdata/bench/1a-hooks.json", Owner: "history"},
		{Path: "testdata/bench/search/v1/baseline/", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "7.0.1", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "7.0.0", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "4.2.2", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "2.5.0", Owner: "history"},
		{Path: "CHANGELOG.md", Section: "2.3.0", Owner: "history"},
		{Path: "internal/notices/NOTICE.md", Owner: "license"},
		{Path: "internal/dev/notices/ported.md", Owner: "license"},
		{Path: "README.md", Line: ideaLine(), Owner: "idea"},
		{Path: "README.de.md", Line: ideaLine(), Owner: "idea"},
		{Path: "docs/en/architecture.md", Line: ideaLine(), Owner: "idea"},
		{Path: "docs/de/architecture.md", Line: ideaLine(), Owner: "idea"},
		{Path: "internal/namecheck/references.go", Owner: "self"},
		{Path: "internal/namecheck/references_test.go", Owner: "self"},
	}
})
