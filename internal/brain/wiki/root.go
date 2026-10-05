package wiki

import (
	"os"
	"path/filepath"

	"github.com/xidus90/loomux/internal/config"
)

// Root answers where a project's wiki bundle is: the manifest's [layout] wiki
// when it names an existing directory, then docs/wiki, then wiki, then a
// neighbour wiki of the same family. "" means the project has none.
//
// loomux has one key for this, [layout] wiki. The value goes through
// WikiLayout, so a layout that leaves the repository is ignored like an
// unreadable manifest: the fallbacks answer instead.
func Root(projectRoot string) string {
	if manifest, err := config.ReadManifest(projectRoot); err == nil {
		if layout, err := manifest.WikiLayout(); err == nil && layout != "" {
			if dir := filepath.Join(projectRoot, filepath.FromSlash(layout)); isDir(dir) {
				return dir
			}
		}
	}
	for _, candidate := range []string{filepath.Join(projectRoot, "docs", "wiki"), filepath.Join(projectRoot, "wiki")} {
		if isDir(candidate) {
			return candidate
		}
	}
	return neighbour(projectRoot)
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
