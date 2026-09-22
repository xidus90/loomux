package maintenance

// The derivation edge: source -> wiki pages, rebuilt from the pages.
//
// The original is `src/brain/maintenance/derive.py`.

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/wiki"
)

// Dependents is the backward index source -> pages, rebuilt from the pages
// themselves (`dependents`).
//
// Nothing is stored: a derived index that can go stale would need maintenance
// of its own, which is the problem this layer exists to solve. Sorted
// throughout, so two runs over an unchanged bundle agree.
//
// A page citing the same doc id more than once is listed for it once -- its
// presence is a fact about the page, not a count of citations.
//
// A page with broken frontmatter raises no edge: wiki.ReadPage carries the
// defect on the page rather than answering an error, so one unreadable block
// can never abort the whole rebuild. That is deliberately not the same as
// "known to have no sources" -- whoever reconciles on this index has to find
// the broken frontmatter itself and open its own case, rather than reading
// silence here as a clean bill of health.
//
// A map and not a sorted slice of pairs: the one caller looks a doc id up and
// never walks the keys, and the ordering Python promises there lives in the
// sorted value slices.
func Dependents(wikiPath string) (map[string][]string, error) {
	edges := map[string][]string{}
	err := walkPages(wikiPath, func(page *wiki.WikiPage) error {
		for _, source := range page.Sources {
			if !slices.Contains(edges[source.DocID], page.Relative) {
				edges[source.DocID] = append(edges[source.DocID], page.Relative)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Sorted here and not left to the walk: the walk reaches a directory `a`
	// before the file `a-x.md`, while the relative paths sort the other way
	// round -- `-` is below `/`.
	for _, pages := range edges {
		slices.Sort(pages)
	}
	return edges, nil
}

// walkPages reads every wiki page of wikiPath and hands it to visit.
//
// One walk for the two readers of a wiki here -- the derivation index above
// and the open-page candidates a merge case is raised over. Both need the same
// two exclusions, and a second copy of them would be the second place a
// scaffold name has to reach.
func walkPages(wikiPath string, visit func(*wiki.WikiPage) error) error {
	return filepath.WalkDir(wikiPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// `rglob` walks what is inside the wiki and never yields the wiki
		// itself. Without this, a bundle whose wiki directory happens to be
		// called `wiki.md` would be read as its own first page.
		if path == wikiPath {
			return nil
		}
		// The name and not the mode: `rglob("*.md")` hands a directory called
		// `a.md` to the reader as well, and the read then fails. Skipping it
		// by its mode would hide a wiki that has grown one.
		if !strings.HasSuffix(entry.Name(), ".md") || wiki.IsScaffoldFile(entry.Name()) {
			return nil
		}
		page, err := wiki.ReadPage(path, wikiPath)
		if err != nil {
			return err
		}
		return visit(page)
	})
}
