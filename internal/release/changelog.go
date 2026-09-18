package release

import (
	"errors"
	"fmt"
	"strings"
)

// ChangelogHeader opens a CHANGELOG.md that the first release creates.
const ChangelogHeader = "# Changelog\n\nAll notable changes to loomux are listed here, newest first. The format\nfollows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions\nfollow [Semantic Versioning](https://semver.org/).\n"

// ErrDuplicate refuses a version the file already names, so a repeated
// release run cannot write an entry twice.
var ErrDuplicate = errors.New("version already in changelog")

// InsertChangelog puts the entry of one release above all older entries,
// directly below the header. nil or an empty file stands for a missing one.
func InsertChangelog(existing []byte, version, date, link, notes string) ([]byte, error) {
	text := string(existing)
	if len(existing) == 0 {
		text = ChangelogHeader
	}
	if strings.Contains(text, "\n## ["+version+"]") {
		return nil, fmt.Errorf("%w: %s", ErrDuplicate, version)
	}
	entry := fmt.Sprintf("\n## [%s] - %s\n\n<%s>\n\n%s", version, date, link, strings.TrimRight(notes, "\n")+"\n")
	if i := strings.Index(text, "\n## ["); i >= 0 {
		return []byte(text[:i+1] + entry[1:] + "\n" + text[i+1:]), nil
	}
	return []byte(strings.TrimRight(text, "\n") + "\n" + entry), nil
}
