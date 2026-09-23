package wiki

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/config"
)

// noType names the pages without a `type` in the order and in the output,
// so both compare and print one value (`census.py`, `_NO_TYPE`).
const noType = "(no type)"

// TypeCount is one page type counted across every area: its worst rank, the
// pages per scope and the catalogue name if it is a known misspelling.
type TypeCount struct {
	Type     *string
	Rank     Rank
	PerScope map[string]int
	AliasOf  string
}

func (c TypeCount) name() string {
	if c.Type == nil {
		return noType
	}
	return *c.Type
}

func (c TypeCount) total() int {
	n := 0
	for _, count := range c.PerScope {
		n += count
	}
	return n
}

// Census is `census`: every page type across every area that has a bundle,
// sorted by total descending, then by name. An area without a wiki path, or
// with one that is not a directory, is passed over rather than refused: the
// tally is meant to run over the whole corpus while most areas carry no
// bundle yet. manifestDir says where an area's declaration lies, which for a
// read-only area is under the state directory.
//
// A declaration that exists and does not read stops the tally, as
// `read_manifest` raising does; one that is simply absent declares nothing.
func Census(areas []config.Area, manifestDir func(config.Area) string) ([]TypeCount, error) {
	perType := map[string]*TypeCount{}
	for _, area := range areas {
		// No wiki path is an empty name, which Stat refuses like a missing one.
		if info, err := os.Stat(area.WikiPath); err != nil || !info.IsDir() {
			continue
		}
		declared, err := DeclaredTypesIn(manifestDir(area))
		if err != nil {
			return nil, err
		}
		for _, path := range MarkdownBelow(area.WikiPath) {
			if IsScaffoldFile(path) {
				continue
			}
			page, err := ReadPage(path, area.WikiPath)
			if err != nil {
				return nil, err
			}
			key := noType
			if page.PageType != nil {
				// A leading NUL keeps a type literally spelt "(no type)"
				// apart from the missing one.
				key = "\x00" + *page.PageType
			}
			count, seen := perType[key]
			if !seen {
				count = &TypeCount{Type: page.PageType, PerScope: map[string]int{}}
				if page.PageType != nil {
					count.AliasOf, _ = AliasTarget(*page.PageType)
				}
				perType[key] = count
			}
			count.PerScope[area.Scope]++
			// The worst rank wins: core, catalogue and origin are the same
			// in every area, only declared depends on one, and a type that
			// one area declares and another uses undeclared is not settled.
			if rank := RankOf(page.PageType, declared); !seen || rank == RankUnknown {
				count.Rank = rank
			}
		}
	}
	counted := make([]TypeCount, 0, len(perType))
	for _, count := range perType {
		counted = append(counted, *count)
	}
	slices.SortFunc(counted, func(a, b TypeCount) int {
		return cmp.Or(cmp.Compare(b.total(), a.total()), strings.Compare(a.name(), b.name()))
	})
	return counted, nil
}

// RenderCensus is `render`: one line per type with its rank, its total, the
// count per area and the catalogue name of an alias. An unknown rank gets the
// visible prefix `? `, because a tally that let a gap pass unnoticed would be
// useless for a migration.
func RenderCensus(counts []TypeCount) string {
	var b strings.Builder
	for _, count := range counts {
		if count.Rank == RankUnknown {
			b.WriteString("? ")
		}
		fmt.Fprintf(&b, "%s [%s]", count.name(), count.Rank)
		if count.AliasOf != "" {
			fmt.Fprintf(&b, " -> %s", count.AliasOf)
		}
		scopes := make([]string, 0, len(count.PerScope))
		for scope := range count.PerScope {
			scopes = append(scopes, scope)
		}
		slices.Sort(scopes)
		parts := make([]string, len(scopes))
		for i, scope := range scopes {
			parts[i] = fmt.Sprintf("%s: %d", scope, count.PerScope[scope])
		}
		fmt.Fprintf(&b, ": %d (%s)\n", count.total(), strings.Join(parts, ", "))
	}
	return b.String()
}

// DeclaredTypesIn is `_declared_types`: the types the declaration in dir names
// beyond the built-in ones, or none when dir holds no declaration.
func DeclaredTypesIn(dir string) (map[string]bool, error) {
	manifest, err := config.ReadAreaManifestUntilStage4(dir)
	if errors.Is(err, config.ErrNoManifest) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	declared := map[string]bool{}
	for _, t := range manifest.DeclaredTypes {
		declared[t] = true
	}
	return declared, nil
}

// MarkdownBelow is `sorted(rglob("*.md"))` over files: every regular file
// below root whose name ends in `.md`, in the order Python sorts paths on
// Windows -- component by component, each folded to lower case. The walk
// itself goes byte by byte per directory, which puts upper case first and
// `a-c.md` before the directory `a`.
func MarkdownBelow(root string) []string {
	var paths []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	slices.SortStableFunc(paths, func(a, b string) int {
		return slices.Compare(foldedParts(a), foldedParts(b))
	})
	return paths
}

// foldedParts is a path split into its components, each in lower case.
func foldedParts(path string) []string {
	return strings.Split(strings.ToLower(filepath.ToSlash(path)), "/")
}
