package query

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/gitenv"
)

var gitOutput = runGit // seam: tests replace it to reach git's failure arms

// runGit runs git in dir with paths left unquoted and without the
// repository pointers a surrounding git hook exports, which would outrank
// dir, save the index a commit hook is handed for dir itself. It keeps git's
// own complaint in the error: an exit status alone does not say what was
// wrong.
func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-c", "core.quotePath=false"}, args...)...)
	cmd.Dir = dir
	env := gitenv.Environ()
	if index, ok := indexFileFor(dir); ok {
		env = append(env, "GIT_INDEX_FILE="+index)
	}
	cmd.Env = env
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, fmt.Errorf("git %s: %w: %s", args[0], err, bytes.TrimSpace(exit.Stderr))
	}
	return out, err
}

// GraphEnv is the environment a graph lane's child processes need at root:
// their environment loses the git pointers a surrounding hook exports, and
// with them the index a commit hook is handed. It carries that index when
// indexFileFor accepts it, and nothing otherwise.
func GraphEnv(root string) []string {
	if index, ok := indexFileFor(root); ok {
		return []string{"GIT_INDEX_FILE=" + index}
	}
	return nil
}

// indexFileFor returns the inherited GIT_INDEX_FILE when it is the index of
// the repository at dir. Under `git commit -a` or `git commit <path>` git
// hands its hooks a temporary index inside the git directory, and only that
// one holds what is being committed; without it a hook's `diff --cached`
// sees nothing staged. Only a file inside dir's own git directory counts:
// not that directory itself, and not a path under the main git directory's
// worktrees/, which holds the git directories of linked worktrees (in a
// linked worktree --absolute-git-dir is already that inner directory). Any
// other index, one of another repository or worktree or a relative path
// nobody can place, stays stripped as gitenv strips it.
func indexFileFor(dir string) (string, bool) {
	index := os.Getenv("GIT_INDEX_FILE")
	if !filepath.IsAbs(index) {
		return "", false
	}
	cmd := exec.Command("git", "rev-parse", "--absolute-git-dir")
	cmd.Dir = dir
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	// Both sides go through the barrier's resolver before they are compared:
	// git answers the long name of the git directory, while the hook's index
	// may carry an 8.3 alias of the same directory (a CI runner's %TEMP% is
	// C:\Users\RUNNER~1\...) or another case, and a plain filepath.Rel would
	// place the index outside the directory it lies in.
	rel, err := filepath.Rel(resolved(strings.TrimSpace(string(out))), resolved(index))
	if err != nil || !filepath.IsLocal(rel) || rel == "." {
		return "", false
	}
	// Case-folded: on Windows the hook's path may differ from git's in case.
	if first, _, _ := strings.Cut(filepath.ToSlash(rel), "/"); strings.EqualFold(first, "worktrees") {
		return "", false
	}
	return index, true
}

// resolved is path with every existing component spelt as the file system
// names it, the missing rest joined on as given. A path the resolver refuses
// (a drive-relative one) stays as it was, cleaned: the comparison then
// answers as it did before, which refuses rather than accepts.
func resolved(path string) string {
	if full, err := guard.ResolvePath(path); err == nil {
		return full
	}
	return filepath.Clean(path)
}
