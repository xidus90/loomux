package detect

import (
	"io/fs"
	"path"
	"strings"
)

// A Runner runs one command in one directory and returns what it printed.
//
// Injected rather than called directly, so the two facts below stay testable
// without git and without a filesystem -- the same reason Detect takes an
// fs.FS. An unset setting is not a failure: git exits 1 and prints nothing,
// and a Runner reports that as an empty string with no error.
type Runner func(dir string, argv ...string) (string, error)

// HooksPath is the one fact a directory tree cannot tell about itself.
//
// It lives in git's configuration, so reading it costs the single subprocess
// the design allows. An empty answer means no hooks path is set, which is the
// common case and not an error.
func HooksPath(run Runner, dir string) (string, error) {
	out, err := run(dir, "git", "config", "--get", "core.hooksPath")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// NeighbourWiki looks beside the project rather than inside it.
//
// A family of repositories can keep one wiki as a sibling repository named
// `<family>_wiki`: iam_backend, iam_frontend and iam_workers all keep theirs
// in iam_wiki. That wiki belongs to `<family>` and to every `<family>_*`
// sibling, and to nobody else. Until 2026-09-11 the suffix alone decided, and
// ultraloom, which merely stands in the same parent directory, had iam_wiki
// recorded as its own.
//
// The wiki lies outside the project root and is therefore invisible to
// Detect. The parent directory is taken as an fs.FS for the same testability,
// and projectDir names the project within it so a repository called `x_wiki`
// does not find itself.
func NeighbourWiki(parent fs.FS, projectDir string) (mode string, wikiPath string) {
	entries, err := fs.ReadDir(parent, ".")
	if err != nil {
		return "", ""
	}
	for _, entry := range entries {
		name := entry.Name()
		family, isWiki := strings.CutSuffix(name, "_wiki")
		if !entry.IsDir() || name == projectDir || !isWiki || !ofFamily(projectDir, family) {
			continue
		}
		// A directory of notes is not a wiki repository; the commit duty
		// this mode brings needs something to commit into.
		if _, err := fs.Stat(parent, path.Join(name, ".git")); err != nil {
			continue
		}
		return "neighbour_repo", name + "/"
	}
	return "", ""
}

// ofFamily says whether projectDir belongs to the family a wiki is named for:
// the family itself, or a sibling joined to it by an underscore. The
// underscore is the whole test -- iamx shares three letters with iam and is
// not a member.
func ofFamily(projectDir, family string) bool {
	return projectDir == family || strings.HasPrefix(projectDir, family+"_")
}
