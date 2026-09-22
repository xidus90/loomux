package config

import (
	"os"
	"path/filepath"
)

// ArtifactLookup finds a state file of the maintenance layer: the new place
// first, the old one as a fallback.
//
// Until stage 3 artefacts and stamps lived in ultra-brain's directory alone,
// because loomux had no writer. Since `reindex` and `reconcile` it has one,
// and from then on: writing happens only to the new place, reading happens
// new first and, as long as nothing lies there, old. `loomux migrate`
// (stage 4) moves the rest; after that Fallback is empty and this struct is
// a shell around a path.
type ArtifactLookup struct {
	Primary  string
	Fallback string
}

// NewArtifactLookup takes the two places this machine declares.
func NewArtifactLookup() ArtifactLookup {
	return ArtifactLookup{Primary: StateDir(), Fallback: LegacyBrainDirUntilStage3()}
}

// Resolve is the place relative is to be read from.
func (l ArtifactLookup) Resolve(relative string) string {
	primary := filepath.Join(l.Primary, relative)
	if l.Fallback == "" {
		return primary
	}
	if _, err := os.Stat(primary); err == nil {
		return primary
	}
	fallback := filepath.Join(l.Fallback, relative)
	if _, err := os.Stat(fallback); err == nil {
		return fallback
	}
	// Neither here nor there: the new place is the one the caller's error
	// message should name.
	return primary
}

// WritePath is the place relative is to be written to -- always the new one.
func (l ArtifactLookup) WritePath(relative string) string {
	return filepath.Join(l.Primary, relative)
}

// ResolvedAreaDir is ManifestDir with the fallback rule of stage 3a applied:
// a writable area keeps its artefacts in its own tree, a read-only area under
// `areas/<flat scope>` of stateDir and, as long as nothing lies there, of
// fallbackDir.
//
// Both directories are arguments and neither is read from the environment
// here: `internal/serve` promises that everything hangs off the state
// directory it was handed, "never off a fixed path -- only so does a test
// isolate its serve from the real one" (serve.go), and a lookup that asked
// StateDir() itself would break that promise for every caller at once.
//
// The whole directory decides, not the single file: an area is written by one
// `reindex`, and a half-migrated area whose register came from the new place
// and whose catalog came from the old one would describe no state that ever
// existed.
func ResolvedAreaDir(area Area, stateDir, fallbackDir string) string {
	if !area.ReadOnly {
		return area.Path
	}
	lookup := ArtifactLookup{Primary: stateDir, Fallback: fallbackDir}
	return lookup.Resolve(filepath.Join("areas", flat(area.Scope)))
}
