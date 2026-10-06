package catalog

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

// linkDestination finds the destination of every inline Markdown link on a
// line: in angle brackets, as the catalog writer spells one with a blank or
// a bracket in it, or bare.
var linkDestination = regexp.MustCompile(`\]\((<[^>]*>|[^)\s]*)\)`)

// uriScheme is an RFC 3986 scheme, which marks a link that leaves the vault
// -- the rule graph.DropReason classifies external links by.
var uriScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// Withhold is an area's catalog without the lines that link into a path
// conceals hides.
//
// A catalog names what lies in its area, and the root catalog of an area
// whose tree holds the wiki of a `local_only` one names that wiki as a line
// of `## Bereiche` -- the very name an unknown-scope answer keeps from the
// cloud channel. The line goes whole, since what else it says is about the
// hidden tree; every other line stays as it was, byte for byte.
//
// Every link on a line is asked, so a hand-written catalog is held to the
// rule as well as a generated one. A destination is read as the writer
// spells it -- angle brackets off, then percent-decoded, which also undoes
// the writer's %3C and %3E -- with query and fragment cut and a leading
// slash read as the area root, as graph.ResolveTarget reads one. A link with
// a scheme leaves the vault and is not asked about.
func Withhold(text string, conceals func(relative string) bool) string {
	lines := strings.SplitAfter(text, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if !linksInto(line, conceals) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "")
}

// linksInto says whether any link on line lands in a concealed path.
func linksInto(line string, conceals func(string) bool) bool {
	for _, match := range linkDestination.FindAllStringSubmatch(line, -1) {
		destination := strings.TrimSuffix(strings.TrimPrefix(match[1], "<"), ">")
		if uriScheme.MatchString(destination) {
			continue
		}
		if unescaped, err := url.PathUnescape(destination); err == nil {
			destination = unescaped
		}
		if cut := strings.IndexAny(destination, "#?"); cut >= 0 {
			destination = destination[:cut]
		}
		if conceals(strings.TrimPrefix(path.Clean(destination), "/")) {
			return true
		}
	}
	return false
}
