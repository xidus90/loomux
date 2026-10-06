// Package all is the fixed list of languages a graph build extracts.
//
// A list and not a registry: registration at run time would need init()
// (AGENTS.md forbids it) or an order nobody can read off the source.
//
// The hook path does not import this package. It extracts one Go file per
// edit, and every language added here would reach that path's binary along
// with its parser.
package all

import (
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/gdscript"
	"github.com/xidus90/loomux/internal/code/extract/golang"
	"github.com/xidus90/loomux/internal/code/extract/python"
)

// Languages is every extractor a build runs, Go first.
func Languages() []extract.Language {
	return []extract.Language{golang.Language{}, python.Language{}, gdscript.Language{}}
}

// Version is the build's combined extractor identity: the versions of every
// language, sorted and joined by "+".
//
// It is what the graph's meta and the freshness record carry and what `check`
// compares. Adding a language or bumping one changes it, and every graph
// built without that language then reads as foreign and is rebuilt.
func Version() string {
	var vs []string
	for _, l := range Languages() {
		vs = append(vs, l.Version())
	}
	sort.Strings(vs)
	return strings.Join(vs, "+")
}

// For is the language that claims a file, by its extension matched exactly:
// the Go toolchain ignores x.GO, and a graph that took it would hold a file
// no build compiles. sourceset's walk applies the same rule.
func For(rel string) (extract.Language, bool) {
	ext := path.Ext(rel)
	for _, l := range Languages() {
		if slices.Contains(l.Extensions(), ext) {
			return l, true
		}
	}
	return nil, false
}

// Extensions is every extension some language claims, sorted. sourceset keeps
// its own copy for the hook path, and this package's test holds the two equal.
func Extensions() []string {
	var out []string
	for _, l := range Languages() {
		out = append(out, l.Extensions()...)
	}
	sort.Strings(out)
	return out
}
