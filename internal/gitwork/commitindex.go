package gitwork

import (
	"os"
	"path/filepath"
	"strings"
)

// CommitIndex judges the index git handed a pre-commit hook at root.
// inherited is the hook's GIT_INDEX_FILE, "" when it has none. whole says
// that the commit takes the whole index: the index handed in is the real one
// or becomes it once the commit is made. index is then where a hook stages
// into, absolute, and "" for git's own. For a commit of paths both are zero:
// what a hook stages there is committed, while the real index keeps the old
// entry as a staged revert.
//
// This is git's inner working, so it is measured and not derived. With git
// 2.54.0.vfs.0.4 (measured 2026-10-01) a pre-commit hook sees:
//
//	git commit                   .git/index (relative to the hook's directory)
//	git commit --amend           .git/index
//	git commit -a                <git dir>/index.lock
//	git commit --amend -a        <git dir>/index.lock
//	git commit --include <path>  <git dir>/index.lock
//	git commit <path>            <git dir>/next-index-<pid>.lock
//	git commit --only <path>     <git dir>/next-index-<pid>.lock
//	git commit --amend <path>    <git dir>/next-index-<pid>.lock
//
// From a subdirectory the three are the same, and the hook runs at the top of
// the working tree. In a linked worktree <git dir> is that worktree's own,
// <main git dir>/worktrees/<name>, and its plain commit names the index by
// an absolute path. A merge concluded by git commit hands in .git/index.
// Not measured: git commit -p, which needs a console; it is judged by the
// index it hands in, as every form is.
//
// Only index and index.lock of root's own git directory count, and they are
// compared as files, not as spellings: git answers the long name of its
// directory, the variable may carry a short name, another case or a link. An
// index that is not there, one of another repository, and a root no
// repository covers are all a commit of paths to the caller: nothing is
// written that could land in the wrong place.
func CommitIndex(root, inherited string) (index string, whole bool) {
	if inherited == "" {
		return "", true
	}
	index = inherited
	if !filepath.IsAbs(index) {
		// git names it relative to the directory it runs the hook in.
		wd, _ := os.Getwd()
		index = filepath.Join(wd, index)
	}
	given, err := os.Stat(index)
	if err != nil {
		return "", false
	}
	out, err := git(root, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", false
	}
	dir := strings.TrimSpace(out)
	for _, name := range []string{"index", "index.lock"} {
		if own, err := os.Stat(filepath.Join(dir, name)); err == nil && os.SameFile(given, own) {
			return index, true
		}
	}
	return "", false
}

// Stage puts rel of root into the index a commit hook was handed: index as
// CommitIndex answered it, "" for git's own. git strips GIT_INDEX_FILE from
// every call this package makes, so the index is handed in, not inherited.
func Stage(root, index, rel string) error {
	var extra []string
	if index != "" {
		extra = []string{"GIT_INDEX_FILE=" + index}
	}
	_, err := gitWith(root, extra, "add", "--", rel)
	return err
}
