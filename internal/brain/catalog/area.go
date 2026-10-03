package catalog

import (
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// ReadAreaCatalog reads index.md for the given area from its artifact
// directory. config.ManifestDir names that directory: a writable area
// keeps it in its own tree, a read-only one in stateDir.
func ReadAreaCatalog(area config.Area, stateDir string) (string, error) {
	catalogPath := filepath.Join(config.ManifestDir(area, stateDir), "index.md")
	// Path.read_text(encoding="utf-8") in core.catalog: strict UTF-8 and
	// universal newlines, so a CRLF index answers with LF like the reference.
	text, err := pytext.ReadText(catalogPath)
	if err != nil {
		return "", err
	}
	return text, nil
}
