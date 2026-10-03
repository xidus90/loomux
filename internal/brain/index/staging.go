package index

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/lock"
)

// graphName and identitiesName are the two files an area's stock carries
// beside its catalogs. Named here because both the writer and the copy that
// seeds a staging directory owe the reader the same spelling.
const (
	graphName      = "graph.json"
	identitiesName = "_identities.tsv"
)

// recoverStock finishes a swap a killed run left half-done, before anything
// of this area is read.
//
// Before, and not inside publish, because an aside beside a missing target is
// invisible to config.ManifestDir: the area would read as undeclared, and
// the swap at the end would never be reached to put the aside back.
//
// A writable area has no aside -- nothing is ever swapped into a repository
// -- and asking after one would be a question about a sibling of somebody's
// checkout.
func recoverStock(area config.Area, stateDir string) error {
	if !area.ReadOnly {
		return nil
	}
	return lock.Recover(config.ManifestDir(area, stateDir))
}

// publish writes one area's stock: catalogs, link graph and identity
// register.
//
// A writable area is written where it lives, because its catalogs belong
// between its notes and no rename could put a repository in place of
// another. A read-only area's stock lies under `<state>/areas/<scope>` and is
// written beside that directory first, then swapped in whole.
//
// The difference is not a matter of taste. config.ManifestDir resolves a
// read-only area by its directory: every read of that area comes from
// `areas/<scope>` -- the declaration included, which privacy.VisibleAreas reads for every registered
// area before it answers anything. A stock written file by file therefore has
// a window in which one unfinished area makes the whole vault answer nothing.
// Written beside and swapped once, that window does not exist.
func publish(
	area config.Area,
	documents []Document,
	identities map[string]identity.Identity,
	source, stateDir string,
) error {
	target := config.ManifestDir(area, stateDir)
	if !area.ReadOnly {
		return writeStock(area, documents, identities, target)
	}

	staging, err := lock.StagingDir(target)
	if err != nil {
		return err
	}
	// The staging directory is this run's alone; whatever ends the run, it
	// does not stay in the state directory.
	defer os.RemoveAll(staging)

	// Seeded with what lies there today, for two reasons. The declaration is
	// part of the stock and a swapped-in directory without it is exactly the
	// half state above -- and where the stock still comes from the old state
	// directory, this copy is what moves it. writeStock then compares against
	// the old files, so an unchanged catalog is left as it was found.
	if err := copyTree(source, staging); err != nil {
		return err
	}
	if err := writeStock(area, documents, identities, staging); err != nil {
		return err
	}
	return replaceDirFn(staging, target)
}

// writeStock puts the three kinds of artefact into one directory.
func writeStock(
	area config.Area,
	documents []Document,
	identities map[string]identity.Identity,
	dir string,
) error {
	if err := WriteCatalogs(area, documents, dir); err != nil {
		return err
	}
	// RenderGraph's error is the JSON encoder's over a value this package
	// built, which cannot fail; the write is where a run really breaks, and
	// unlike the reference this reports it rather than dropping it.
	graphBytes, _ := graph.RenderGraph(area.Scope, toGraphDocs(documents), OwnWikiPrefix(area))
	if err := writeIfChanged(filepath.Join(dir, graphName), string(graphBytes)); err != nil {
		return err
	}
	return writeIfChanged(filepath.Join(dir, identitiesName), identity.RenderIdentities(identities))
}

// copyTree copies source into target, directories and regular files alike.
//
// The relative path is cut off the walked path rather than computed, because
// a Rel that cannot fail is better than an error arm no test can reach. That
// holds only for a cleaned source -- WalkDir hands out `C:\a\b` below
// `C:\a\.\`, and the prefix would not match -- so source is cleaned here and
// no caller has to know it.
func copyTree(source, target string) error {
	source = filepath.Clean(source)
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destination := filepath.Join(target, strings.TrimPrefix(path, source))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		return copyFile(path, destination)
	})
}

// copyFile copies one file's contents. Through memory: what is copied here is
// an area's stock -- catalogs, a graph and a register -- and the largest of
// them on this machine is under a megabyte.
func copyFile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o644)
}
