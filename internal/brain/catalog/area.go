package catalog

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

var unsafeScope = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func flatScope(scope string) string {
	return strings.Trim(unsafeScope.ReplaceAllString(scope, "-"), "-")
}

// AreaArtifactDir returns the directory where this area's generated artifacts (such as index.md) live.
// Read-only areas store artifacts under stateDir/areas/<flat-scope>, while writable areas store them in area.Path.
func AreaArtifactDir(area config.Area, stateDir string) string {
	if area.ReadOnly {
		return filepath.Join(stateDir, "areas", flatScope(area.Scope))
	}
	return area.Path
}

// ReadAreaCatalog reads index.md for the given area from its artifact directory.
func ReadAreaCatalog(area config.Area, stateDir string) (string, error) {
	catalogPath := filepath.Join(AreaArtifactDir(area, stateDir), "index.md")
	// Path.read_text(encoding="utf-8") in core.catalog: strict UTF-8 and
	// universal newlines, so a CRLF index answers with LF like the reference.
	text, err := pytext.ReadText(catalogPath)
	if err != nil {
		return "", err
	}
	return text, nil
}
