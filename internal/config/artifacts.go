package config

import "path/filepath"

// ArtifactLookup names the state files of the maintenance layer: the stamp,
// the merge events, the record of qmd collections. They live in one state
// directory, the one this run was handed; a lookup never reads from anywhere
// else, so that `internal/serve`'s promise holds -- everything hangs off the
// state directory it was given.
type ArtifactLookup struct {
	Primary string
}

// NewArtifactLookup takes the state directory this machine declares.
func NewArtifactLookup() ArtifactLookup {
	return ArtifactLookup{Primary: StateDir()}
}

// Resolve is the place relative is to be read from.
func (l ArtifactLookup) Resolve(relative string) string {
	return filepath.Join(l.Primary, relative)
}

// WritePath is the place relative is to be written to.
func (l ArtifactLookup) WritePath(relative string) string {
	return filepath.Join(l.Primary, relative)
}
