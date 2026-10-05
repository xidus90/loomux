package detect

import "strings"

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
// common case and not an error. It is read as a path, so git expands a
// leading ~ the way it does when it runs a hook.
func HooksPath(run Runner, dir string) (string, error) {
	out, err := run(dir, "git", "config", "--type=path", "--get", "core.hooksPath")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
