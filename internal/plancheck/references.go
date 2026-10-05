package plancheck

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
)

// The names the fusion spec's addendum #24 searched for when it took stock of
// what loomux still owes its two predecessors; every one that remains has to
// sit in a named exception.
var predecessorNames = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?i)ultraloom|ultra-brain|ulguard|ulinit|ulflow|brain-mcp|brain guard|ultraloomowned|\.brain\.toml|\.ultra-brain|specs-u[lb]/|plans-u[lb]/|bench-ub/`)
})

var changelogEntry = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^## \[(\d+\.\d+\.\d+)(?:-beta(?:\.\d+)?)?\]`)
})

// An Exception lets a name stand where the spec says it may: a file, or every
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

// References names every line of files that carries a predecessor's name and
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

// inTheArchives matches the lines that point into the archives of the two
// predecessors; they stay until the last follow-up project takes them away.
var inTheArchives = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`specs-ul/|specs-ub/|plans-ul/|plans-ub/|bench-ub/`)
})

// followUps matches the roadmap rows of the follow-up projects still open;
// prose that names them is not a row.
var followUps = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^\|.*(ulflow|ultra-brain/web)`)
})

// exceptions are the places the fusion spec lets a predecessor's name stand
// ("Die benannten Ausnahmen", addenda #24, #34, #35). A working-paper folder
// is a row of its own, so a new folder under docs/.superpowers/ is reported.
// The archives are rows of Owner "archives": the spec lets both follow-up
// projects, Flow and Web, draw on them and has the last one to finish take
// them away, so whoever finishes last removes these rows and the Line rules
// that point into them.
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
		{Path: "docs/en/migration.md", Owner: "plan-end"},
		{Path: "docs/de/migration.md", Owner: "plan-end"},
		{Path: "docs/.superpowers/specs/", Owner: "working-papers"},
		{Path: "docs/.superpowers/plans/", Owner: "working-papers"},
		{Path: "docs/.superpowers/parity/", Owner: "working-papers"},
		{Path: "docs/.superpowers/specs-ul/", Owner: "archives"},
		{Path: "docs/.superpowers/specs-ub/", Owner: "archives"},
		{Path: "docs/.superpowers/plans-ul/", Owner: "archives"},
		{Path: "docs/.superpowers/plans-ub/", Owner: "archives"},
		{Path: "docs/.superpowers/bench-ub/", Owner: "archives"},
		{Path: "docs/wiki/log.md", Owner: "working-papers"},
		{Path: "docs/wiki/", Line: inTheArchives(), Owner: "archives"},
		{Path: "internal/brain/apply/testdata/frontmatter/", Line: inTheArchives(), Owner: "archives"},
		{Path: "_identities.tsv", Line: inTheArchives(), Owner: "archives"},
		{Path: "README.md", Line: followUps(), Owner: "flow"},
		{Path: "README.de.md", Line: followUps(), Owner: "flow"},
		{Path: "internal/plancheck/references.go", Owner: "self"},
		{Path: "internal/plancheck/references_test.go", Owner: "self"},
	}
})
