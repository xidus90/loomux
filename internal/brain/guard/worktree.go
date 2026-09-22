package guard

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// readGitFile reads one of git's pointer files. A variable so that a test
// can hold that a write the registry already opens reads none of them;
// nothing outside a test writes it.
var readGitFile = os.ReadFile

// linkedWorktreeRoots is the root of every linked git worktree that holds
// one of `targets` and belongs to the repository of a workspace area.
//
// Not in the Python barrier, where a worktree needed a registry entry of its
// own (spec 2026-09-15-loomux-schranke-worktrees). A linked worktree is the
// registered repository checked out a second time, so what `workspace` opens
// in one it opens in the other. Git is read from its files instead of being
// asked: `git rev-parse` took a 42 ms median on this machine, against a whole
// hook of 24.5 ms.
func linkedWorktreeRoots(targets []string, areas []area) []string {
	commons := []string{}
	for _, registered := range areas {
		if !registered.workspace {
			continue
		}
		if common := repositoryCommon(registered.path); common != "" {
			commons = append(commons, common)
		}
	}
	found := []string{}
	for _, target := range targets {
		if root := linkedWorktreeRoot(target, commons); root != "" {
			found = append(found, root)
		}
	}
	return found
}

// linkedWorktreeRoot is the nearest directory above `target` that is a
// linked worktree sharing one of `commons`, or "".
//
// It climbs past a directory that does not match instead of stopping there:
// a repository nested in a worktree lies below the worktree's root, just as
// the same path in the main checkout lies below the registered tree.
func linkedWorktreeRoot(target string, commons []string) string {
	for _, directory := range parents(target) {
		common := linkedCommon(directory)
		if common == "" {
			continue
		}
		if slices.ContainsFunc(commons, func(registered string) bool {
			return pathsEqual(registered, common)
		}) {
			return directory
		}
	}
	return ""
}

// repositoryCommon is the common git directory of the repository checked out
// at `root`: its `.git` directory, or the one its `.git` file leads to where
// the registration names a linked worktree itself. "" where `root` has
// neither.
func repositoryCommon(root string) string {
	dotGit := filepath.Join(root, ".git")
	if info, err := os.Stat(dotGit); err == nil && info.IsDir() {
		return resolvedOrEmpty(dotGit)
	}
	return linkedCommon(root)
}

// registeredCommon is the common git directory of the repository a
// registered area lies in, or "" where there is none this reading
// understands.
//
// It climbs where the candidate side does not, because the question it
// replaces -- `git rev-parse --git-common-dir` -- answers from any directory
// inside a checkout, and a registered area need not be a checkout root. Two
// limits keep the climb from answering more than git did. It starts only
// from a directory that is there: git refuses a missing path and a file
// alike, and a climb would pass over either to whatever ancestor still
// stands. And it stops at the first `.git` of any kind, understood or not:
// past a submodule's `.git` file lies the superproject, whose worktrees git
// keeps apart from the submodule.
func registeredCommon(registered string) string {
	if info, err := os.Stat(registered); err != nil || !info.IsDir() {
		return ""
	}
	for _, directory := range append([]string{registered},
		parents(registered)...) {
		if _, err := os.Lstat(filepath.Join(directory, ".git")); err == nil {
			return repositoryCommon(directory)
		}
	}
	return ""
}

// linkedCommon is the common git directory of the linked worktree rooted at
// `directory`, or "" where it is none.
//
// Three guards, each against a different forgery. The `.git` has to be a
// regular file: a symlink to a real worktree's `.git` would resolve to it and
// borrow its back pointer. The back pointer makes a copied `.git` file
// worthless: the administration directory it names leads back to the real
// worktree, not to the copy. And the administration directory has to sit
// directly below `<common>/worktrees`, where git always puts it: anywhere
// else -- a nested repository's inside the workspace, whose `commondir` a
// writing tool may rewrite -- it could name the workspace's repository and
// open a tree outside every registered one. A submodule and a
// `--separate-git-dir` checkout carry a `.git` file as well, but their git
// directory holds neither `gitdir` nor `commondir` (Git 2.54), so they fall
// out at the back pointer already.
func linkedCommon(directory string) string {
	dotGit := filepath.Join(directory, ".git")
	if info, err := os.Lstat(dotGit); err != nil || !info.Mode().IsRegular() {
		return ""
	}
	admin := pointer(dotGit, "gitdir: ", directory)
	if admin == "" {
		return ""
	}
	back := pointer(filepath.Join(admin, "gitdir"), "", admin)
	if back == "" || !pathsEqual(back, resolvedOrEmpty(dotGit)) {
		return ""
	}
	common := pointer(filepath.Join(admin, "commondir"), "", admin)
	if common == "" ||
		!pathsEqual(filepath.Dir(admin), filepath.Join(common, "worktrees")) {
		return ""
	}
	return common
}

// pointer is the resolved path a git pointer file names, counted from `base`
// where it is relative. "" where the file cannot be read, lacks `prefix`, or
// names a path that does not resolve: each one means "not a worktree", and
// that answer keeps the tree shut.
func pointer(file, prefix, base string) string {
	data, err := readGitFile(file)
	if err != nil {
		return ""
	}
	named, found := strings.CutPrefix(
		strings.TrimRight(string(data), " \t\r\n"), prefix)
	if !found {
		return ""
	}
	if !filepath.IsAbs(named) {
		named = filepath.Join(base, named)
	}
	return resolvedOrEmpty(named)
}

// resolvedOrEmpty is `ResolvePath` where an error and no answer mean the
// same thing: not this repository.
func resolvedOrEmpty(path string) string {
	resolved, err := ResolvePath(path)
	if err != nil {
		return ""
	}
	return resolved
}
