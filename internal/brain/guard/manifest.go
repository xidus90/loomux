package guard

import (
	"os"
	"path/filepath"
)

// bundleDir and manifestName name the one manifest the barrier reads.
// ultra-brain had a second, legacy spelling; loomux cut it on 2026-09-14.
const (
	bundleDir    = ".loomux"
	manifestName = "config.toml"
)

// declarationIn is the manifest this one directory carries, or "" if it
// carries none. manifestPath answers for a registered area; this walk climbs
// a path no area is known for yet.
func declarationIn(directory string) string {
	bundled := filepath.Join(directory, bundleDir, manifestName)
	if isRegularFile(bundled) {
		return bundled
	}
	return ""
}

// isRegularFile follows links and answers no rather than failing where the
// path cannot be looked at.
func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
