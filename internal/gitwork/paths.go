package gitwork

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StagedPaths names the paths a commit hook's index changes against HEAD:
// the paths of the commit under way. inherited is the hook's GIT_INDEX_FILE,
// relative to the directory git starts the hook in. Only an index in root's
// own git directory answers -- index, index.lock or a commit of paths'
// next-index-<pid>.lock (see CommitIndex) --, so an absolute index outside
// that directory, or one that is not there, is an error, not a list. A
// relative one is resolved against the working directory, as git does.
//
// --no-renames lists both ends of a rename, -z hands paths over unquoted, and
// the last two keep the repository's config (diff.ignoreSubmodules,
// diff.relative) from hiding or renaming a staged path.
func StagedPaths(root, inherited string) ([]string, error) {
	index := inherited
	if !filepath.IsAbs(index) {
		wd, _ := os.Getwd()
		index = filepath.Join(wd, index)
	}
	out, err := git(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return nil, err
	}
	// A failed Stat leaves a nil FileInfo, which SameFile answers false for.
	own, _ := os.Stat(strings.TrimSpace(out))
	given, err := os.Stat(filepath.Dir(index))
	if err != nil || !os.SameFile(own, given) {
		return nil, fmt.Errorf("%s is no index of %s", inherited, root)
	}
	if _, err := os.Stat(index); err != nil {
		return nil, err
	}
	out, err = gitWith(root, []string{"GIT_INDEX_FILE=" + index}, "diff", "--cached", "--name-only", "--no-renames", "--ignore-submodules=none", "--no-relative", "-z")
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// ChangedBetween names the paths in which two trees differ, both ends of a
// rename among them: diff-tree is plumbing and detects no renames unless
// asked to.
func ChangedBetween(root, from, to string) ([]string, error) {
	out, err := git(root, "diff-tree", "-r", "--name-only", "-z", from, to)
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

func splitNUL(out string) []string {
	return strings.FieldsFunc(out, func(r rune) bool { return r == 0 })
}
