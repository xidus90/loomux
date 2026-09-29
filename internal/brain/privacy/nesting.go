package privacy

import (
	"os"
	"path/filepath"
	"strings"
)

// Conceals reports whether relative, a path inside this area as Contained
// spells it, lies in a tree the channel hides -- the wiki or the source tree
// of an area it may not see.
//
// Visibility is decided per area, and areas nest: on this machine the wiki of
// "hub" holds the wiki of the `local_only` "project/obsidian-ai". Asked only
// by scope, the hub would hand the cloud channel every page of the inner
// area, and it did (found 2026-09-29). Nesting does not lift `local_only`, so
// a hidden tree stays hidden whichever visible area it is reached through,
// and a visible area registered inside a hidden tree is hidden there too.
// On the local channel Hidden is empty and nothing is concealed.
func (v VisibleArea) Conceals(relative string) bool {
	if len(v.Hidden) == 0 {
		return false
	}
	full := filepath.Join(v.Area.Path, filepath.FromSlash(relative))
	for _, root := range v.Hidden {
		if Within(root, full) {
			return true
		}
	}
	return false
}

// Within reports whether path lies inside root or is root itself. It is
// `within` of internal/setup, which privacy cannot import (setup depends on
// it), reduced to the question.
//
// Directories are compared by identity, not by spelling: a root given as an
// 8.3 short name, in another case or through a junction reads differently
// from the path it holds. filepath.EvalSymlinks is not enough -- since Go
// 1.23 it leaves a junction as it is -- while os.Stat follows all three. So
// path's directories are walked upwards until one is root itself. A root
// that does not exist is compared by its cleaned spelling: nothing under it
// exists either, and identity has nothing to go on.
//
// The walk goes up path's own spelling. A link inside the visible tree that
// points at the root is found, because its directory stats as the root; a
// link that points at a directory below the root is not, since no directory
// of the walk is the root itself.
func Within(root, path string) bool {
	rootInfo, err := os.Stat(root)
	if err != nil {
		rel, err := filepath.Rel(root, path)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	for dir := filepath.Clean(path); ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(dir); err == nil && os.SameFile(info, rootInfo) {
			return true
		}
		if filepath.Dir(dir) == dir {
			return false
		}
	}
}
