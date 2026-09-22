package maintenance

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// CaseFiles is every `case.toml` beneath the review centre, at whatever depth
// it sits, in the order a reader is shown the queue (`_case_files`).
//
// A walk of the whole tree and not a fixed two levels: the review centre is a
// directory in the vault, synchronised and hand-arranged, and a case a reader
// moved into a folder must not drop out of the queue in silence.
//
// A review centre that does not exist yet answers nothing, and so does a
// directory that merely carries the name.
func CaseFiles(root string) []string {
	var paths []string
	// No Stat of root first: for a missing root WalkDir calls back once with
	// the Lstat error, and answering nil there is the empty queue.
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && entry.Name() == caseName {
			paths = append(paths, path)
		}
		return nil
	})
	// The whole path is split, root and all: every path shares the root, so
	// those parts always tie and the order is decided below them. Stable, so
	// two names that differ only in case keep the walk's order, as Python's
	// sorted() keeps paths it considers equal.
	separator := string(filepath.Separator)
	slices.SortStableFunc(paths, func(a, b string) int {
		return compareCaseParts(strings.Split(a, separator), strings.Split(b, separator))
	})
	return paths
}

// compareCaseParts orders two case files the way `sorted()` orders Windows
// paths: part by part, each part in lower case, a shorter path before a longer
// one it is the start of.
//
// That is Python's order on Windows, where the reference ran; loomux carries
// it to every system rather than sort case-sensitively on a POSIX one.
// Neither obvious spelling gets it right: a byte-order walk puts `Zeta-case`
// before `alpha-case`, and a sort over the joined strings puts `alpha-later`
// before `alpha/...`, because the dash sorts before either separator.
func compareCaseParts(a, b []string) int {
	return slices.CompareFunc(a, b, func(x, y string) int {
		return strings.Compare(strings.ToLower(x), strings.ToLower(y))
	})
}

// FindCase is the one case directory beneath the review centre named
// identifier (`_find_case`).
//
// Names are compared, never pasted into a glob: `case` and `approve` take the
// id straight from the command line, and a `*` or a `..` in it would otherwise
// reach past the review centre.
//
// Two directories carrying one id are refused rather than resolved to the
// first. `CaseID` cannot mint such a pair -- its digest runs over area and
// target together -- so a duplicate got there by hand, and the two carry
// different targets: approving the wrong one writes onto a page nobody looked
// at.
func FindCase(reviewRoot, identifier string) (string, error) {
	missing := "no case named " + pytext.Repr(identifier)
	if info, err := os.Stat(reviewRoot); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s; there is no review centre at %s yet", missing, reviewRoot)
	}
	var found []string
	for _, path := range CaseFiles(reviewRoot) {
		if directory := filepath.Dir(path); filepath.Base(directory) == identifier {
			found = append(found, directory)
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("%s in the review centre at %s", missing, reviewRoot)
	case 1:
		return found[0], nil
	}
	return "", fmt.Errorf("%s unambiguously; %s and %s both carry it", missing, found[0], found[1])
}

// AreaManifest is the current declaration of one area, or nil when nothing
// declares it (`area_manifest`). nil covers an area the registry does not
// name and one registered without a declaration, and the caller reads either
// as closed.
//
// The case's own mark is a snapshot: a write-back can drop it, and an area
// closed after its cases were formed is only caught up by the next pass. This
// is how `case` asks the area as it stands now.
//
// A declaration that does not read answers nil as well, where the reference
// raises. `case` asks ReviewRoot first, which reads the same declarations and
// refuses a broken one, so only a file broken between the two reads gets
// here -- and nil closes the case rather than opening it.
func AreaManifest(areas []config.Area, lookup config.ArtifactLookup, scope string) *config.Manifest {
	manifests, _ := manifestsOf(areas, lookup)
	return manifests[scope]
}
