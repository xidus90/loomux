// Package flows is the catalog of flows loomux ships: one folder per flow
// under catalog/, contributed by pull request, loaded like a project's own.
//
// The embed directive walks catalog/ and leaves out every name that starts
// with "_" or ".", so a flow's _test/ folder -- its script and golden journal
// -- stays out of the binary without a pattern per folder.
package flows

import (
	"embed"
	"io/fs"
	"slices"
)

//go:embed catalog
var catalog embed.FS

// FS is the catalog as the loader reads it: one folder per flow at the top.
func FS() fs.FS {
	// Sub fails only for a name that is not a valid path, and "catalog" is one.
	sub, _ := fs.Sub(catalog, "catalog")
	return sub
}

// Names are the bundled flows, sorted.
func Names() []string {
	// The catalog is compiled in, so reading its top never fails.
	entries, _ := fs.ReadDir(FS(), ".")
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names
}
