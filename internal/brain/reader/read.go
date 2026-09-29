package reader

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/privacy"
	"github.com/xidus90/loomux/internal/brain/pytext"
	"github.com/xidus90/loomux/internal/config"
)

// ReadVisible is ReadDocument for an area privacy.VisibleAreas answered,
// with the one refusal that needs the whole registry: a path inside the tree
// of an area the channel hides -- a `local_only` wiki nested in the one read
// through -- is not there.
//
// Not there, rather than refused: the error is the one os.ReadFile gives for
// a missing file beside it, the same operation, path and cause, so a cloud
// caller cannot tell an existing hidden page from an absent one. A refusal of
// its own would confirm what it withholds. Containment is asked first, as
// ReadDocument asks it, so a path leaving the area keeps its own refusal.
func ReadVisible(visible privacy.VisibleArea, relative, section string, ch privacy.Channel) (string, error) {
	inside, err := privacy.Contained(visible.Area.Scope, relative)
	if err != nil {
		return "", err
	}
	if visible.Conceals(inside) {
		return "", &fs.PathError{Op: "open", Path: filepath.Join(visible.Area.Path, inside), Err: errNotFound}
	}
	return ReadDocument(visible.Area, visible.Manifest, relative, section, ch)
}

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
