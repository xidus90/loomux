package reader

import (
	"fmt"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// ReadDocument reads a document or section within an area with full privacy,
// containment, and channel enforcement.
func ReadDocument(
	area config.Area,
	manifest *config.Manifest,
	relative, section string,
	ch privacy.Channel,
) (string, error) {
	inside, err := privacy.Contained(area.Scope, relative)
	if err != nil {
		return "", err
	}

	if !privacy.IsReadable(manifest, inside) {
		return "", fmt.Errorf("%s/%s is excluded by [privacy] never", area.Scope, inside)
	}

	if ch == privacy.ChannelCloud {
		excludes, err := privacy.ReviewExcludes(manifest)
		if err != nil {
			return "", err
		}
		if privacy.MatchesGlobs(excludes, inside) {
			return "", fmt.Errorf("%s/%s is the review centre; refused on the cloud channel", area.Scope, inside)
		}
	}

	fullPath := filepath.Join(area.Path, inside)
	// Path.read_text(encoding="utf-8") in core.read: strict UTF-8 and
	// universal newlines, so a CRLF file answers with LF like the reference.
	text, err := pytext.ReadText(fullPath)
	if err != nil {
		return "", err
	}

	if section != "" {
		return ExtractSection(text, section)
	}
	return text, nil
}
