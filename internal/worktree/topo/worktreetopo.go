// Package topo answers where the main checkout is and which
// directories git holds as worktrees.
//
// One git call for both answers, and that call is `git worktree list
// --porcelain`. Not a comparison of `--git-dir` against `--git-common-dir`:
// those two are not comparable as text. A directory under
// `.claude/worktrees/` that shared the main index showed it: git answered an
// absolute path (of the form `C:/work/main/.git`) for the first and a
// relative one (`../../../.git`) for the second -- one directory in two spellings, which as
// text differ, so the comparison called a shared-index directory a worktree.
package topo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/gitenv"
)

// ErrNoRepository is the answer for a directory git has nothing to say about.
// Named, because a hook that fires in every project on the machine meets it
// constantly and must treat it as "nothing to do" rather than as a fault.
var ErrNoRepository = errors.New("not a git repository")

// Where a worktree is put in this repository's practice. Both, because
// CLAUDE.md names one of each -- `.worktrees/multi-provider-llm` and
// `.claude/worktrees/project-history-planning-cf98dc` -- so the path is no
// evidence either way and the registration has to decide. Measured on
// 2026-09-07: both directories are there in the main checkout, and
// `.claude/worktrees` is empty at the moment, which is why absence and
// emptiness are both ordinary below.
var conventionDirs = []string{
	filepath.Join(".claude", "worktrees"),
	".worktrees",
}

// Topology is what one `git worktree list` call says.
type Topology struct {
	// Main is the main checkout, which is the first entry git prints and the
	// directory every junction points into. Measured on 2026-09-07: git names
	// it first whether the call is made from the main checkout or from either
	// worktree.
	Main string
	// Worktrees is every registered working tree, Main included.
	Worktrees []string
}

// Read asks git about `dir`.
func Read(dir string) (Topology, error) {
	command := exec.Command("git", "worktree", "list", "--porcelain")
	command.Dir = dir
	// See gitenv: GIT_DIR and its relatives outrank command.Dir, and git
	// exports them to every hook it runs.
	command.Env = gitenv.Environ()
	out, err := command.Output()
	if err != nil {
		// Every failure here is the same answer. Telling a broken git from an
		// unrepositoried directory would buy a second failure mode for a
		// caller that treats both as "nothing to do".
		return Topology{}, fmt.Errorf("%w: %s: %v", ErrNoRepository, dir, err)
	}
	worktrees := parse(string(out))
	if len(worktrees) == 0 {
		// No git reached this: even `git init --bare` prints a `worktree` line
		// followed by `bare`. The guard is what keeps worktrees[0] below from
		// being an index panic when something else is answering as git.
		return Topology{}, fmt.Errorf("%w: %s: git named no working tree", ErrNoRepository, dir)
	}
	return Topology{Main: worktrees[0], Worktrees: worktrees}, nil
}

// parse takes the `worktree <path>` lines and nothing else.
//
// Porcelain output is a block per working tree, and the first line of each
// block is the path. HEAD, branch, bare and detached are none of our business
// here -- a worktree without a commit still owns its directory.
//
// The paths come back cleaned, and therefore in the platform separator: git
// prints `C:/Users/...` on Windows, filepath.Clean's last step rewrites that
// to backslashes. Callers that compare the result against a literal have to
// expect the platform's spelling.
func parse(output string) []string {
	var paths []string
	for _, line := range strings.Split(output, "\n") {
		rest, found := strings.CutPrefix(strings.TrimRight(line, "\r"), "worktree ")
		if found && rest != "" {
			paths = append(paths, filepath.Clean(rest))
		}
	}
	return paths
}

// IsWorktree says whether `dir` is one of git's working trees and not the
// main checkout.
//
// Compared by what the paths open rather than by how they are spelled: git
// prints forward slashes on Windows -- measured on 2026-09-07, every path in
// the porcelain output came back as `C:/Users/...` -- and a caller hands over
// whatever the hook gave it. Two spellings of one directory compared as text
// is the mistake this package exists to avoid.
func (t Topology) IsWorktree(dir string) bool {
	if sameDir(dir, t.Main) {
		return false
	}
	for _, worktree := range t.Worktrees {
		if sameDir(dir, worktree) {
			return true
		}
	}
	return false
}

// Orphans are the directories under the two conventions that git no longer
// holds as working trees.
//
// They have to be scanned for, because being unregistered is what makes them
// orphans -- `git worktree list` is the one answer that cannot name them. One
// level deep, which is this repository's convention and not a rule of the
// tool: `git worktree add <path>` puts a worktree wherever <path> says, at any
// depth. The two entries in `.worktrees` are what the convention looks like.
func (t Topology) Orphans() ([]string, error) {
	var orphans []string
	for _, convention := range conventionDirs {
		parent := filepath.Join(t.Main, convention)
		entries, err := os.ReadDir(parent)
		if err != nil {
			if os.IsNotExist(err) {
				// Neither convention has to be in use. Absent is not a fault.
				continue
			}
			return nil, fmt.Errorf("scanning %s: %w", parent, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			candidate := filepath.Join(parent, entry.Name())
			if !t.registered(candidate) {
				orphans = append(orphans, candidate)
			}
		}
	}
	return orphans, nil
}

func (t Topology) registered(dir string) bool {
	for _, worktree := range t.Worktrees {
		if sameDir(dir, worktree) {
			return true
		}
	}
	return false
}

// sameDir compares two paths by what they open, not by how they are spelled.
func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	left, err := os.Stat(a)
	if err != nil {
		return false
	}
	right, err := os.Stat(b)
	if err != nil {
		return false
	}
	return os.SameFile(left, right)
}
