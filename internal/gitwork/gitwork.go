// Package gitwork answers what git knows about a working tree.
//
// Its own package because more than one hook asks: session-start records the
// commit a session begins on, the stop gate measures the content tree against
// it, and the subagent hooks compare the local branches, the remote's refs and
// the log of what was added. The Python original is src/ultraloom/worktree.py.
package gitwork

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/gitenv"
)

// ErrIgnoredRoot marks the refusal of a root git ignores. Such a directory is
// inside a repository, so rev-parse answers about it readily -- with the
// surrounding repository's HEAD, which is the wrong tree to measure against.
var ErrIgnoredRoot = errors.New("git ignores this root, so it can never report a change there")

// HeadCommit is worktree.py's `head_commit`: the commit a run starts on, as
// git spells it.
//
// `rev-parse HEAD` and not `--short`: the answer travels in a run marker and
// is read back rounds later, and an abbreviated SHA is only unique for as long
// as the repository stays the size it was.
//
// Three ways of having no answer, all of them errors: no repository, a
// repository without a commit -- `git init` leaves HEAD naming a branch that
// does not exist yet -- and a root git ignores. The last one is why the ignore
// check is asked here at all: such a directory *is* inside a repository, so
// rev-parse answers readily with the surrounding repository's HEAD, and
// measuring against that is worse than not measuring, because every file of
// the parked copy then reads as somebody's change.
func HeadCommit(root string) (string, error) {
	if ignored(root) {
		return "", fmt.Errorf("%s: %w -- run loomux in a working tree of its own", root, ErrIgnoredRoot)
	}
	out, err := git(root, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ignored is worktree.py's `_refuse_if_ignored`, and it reads the outcome of
// the call itself rather than going through `git` below: `check-ignore` exits 1
// for a path it does not ignore, which is the ordinary case and no failure at
// all.
//
// So only exit 0 means "ignored", and everything else -- exit 1, git's exit
// 128 outside a repository, and a spawn that never reached an exit code
// because the directory is not there -- is read as "not ignored" on purpose.
// The call that follows refuses a directory git cannot answer about anyway,
// and turning any of those into a second way of failing here would only make
// that refusal less clear. Python arrives at the same three answers by two
// routes: `_run` raises for the failed spawn, and `_refuse_if_ignored` then
// compares the return code against 0.
func ignored(root string) bool {
	command := exec.Command("git", "check-ignore", "-q", ".")
	command.Dir = root
	command.Env = gitenv.Environ()
	return command.Run() == nil
}

func git(root string, arguments ...string) (string, error) {
	return gitWith(root, nil, arguments...)
}

// gitWith is git with extra KEY=VALUE entries appended behind the strip. The
// order is the point: a later entry wins in exec, and GIT_INDEX_FILE stands in
// gitenv.Location, so a caller that set it in its own environment would have
// it taken away again. Handing it in as a parameter is the only way to point
// git at an index of our choosing.
func gitWith(root string, extra []string, arguments ...string) (string, error) {
	command := exec.Command("git", arguments...)
	command.Dir = root
	// See gitenv: GIT_DIR and its relatives outrank command.Dir, so without
	// the strip this would answer about whatever GIT_DIR names instead of the
	// tree the caller asked about.
	command.Env = append(gitenv.Environ(), extra...)
	// The two streams kept apart, as worktree.py's `_run` keeps them: the
	// answer is stdout alone, because git writes a warning -- an ambiguous
	// refname, a safe.directory note -- to stderr on a call that succeeds, and
	// folded into the answer such a line would travel on as part of the SHA.
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		// Two kinds of failure end up here and they do not carry the same
		// information. git ran and refused: its own words are on stderr, and
		// they are the only account of why -- "not a git repository",
		// "ambiguous argument 'HEAD'". git never ran: the spawn itself failed,
		// stderr is empty, and everything there is to say arrives through
		// `err` -- the OS's chdir error for a directory that is not there,
		// which the suite pins in TestHeadCommitOfADirectoryThatIsNotThere.
		//
		// So stderr is appended only when there is stderr. Unconditionally the
		// message would end in a dangling ": " for every spawn failure, which
		// reads as a truncated error rather than as a complete one.
		detail := strings.TrimSpace(stderr.String())
		if detail != "" {
			detail = ": " + detail
		}
		return "", fmt.Errorf("cannot inspect the working tree in %s: git %s: %v%s",
			root, strings.Join(arguments, " "), err, detail)
	}
	return string(out), nil
}

// EmptyTree is the tree of a commit that holds nothing; git knows it by this
// name in every repository, with or without a commit.
const EmptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// ErrNotRepository marks a root no working tree of git covers.
var ErrNotRepository = errors.New("not a git working tree")

// Head is the commit HEAD names, and "" without an error when the
// repository has no commit yet: that is a state to measure from, the empty
// tree, and not a failure.
func Head(root string) (string, error) {
	if ignored(root) {
		return "", fmt.Errorf("%s: %w", root, ErrIgnoredRoot)
	}
	if _, err := git(root, "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", fmt.Errorf("%s: %w", root, ErrNotRepository)
	}
	// --verify -q exits 1 without a word for a HEAD that names no commit.
	out, err := git(root, "rev-parse", "-q", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// TreeOf is the tree a commit holds.
func TreeOf(root, commit string) (string, error) {
	out, err := git(root, "rev-parse", commit+"^{tree}")
	return strings.TrimSpace(out), err
}

// ContentTree is the tree the working tree would be if everything git does
// not ignore were committed now, loomux's own state left out.
//
// Written through a copy of the index, so the real one never changes: a
// copy and not an empty index, because git only re-hashes a file whose stat
// changed against the index it is given. Measured on 2026-09-19 on 7,322
// files from a shell, each step its own process: 224-245 ms (the design
// spec's cost table). BenchmarkContentTree measures the same four steps
// in-process and reads 107-115 ms over 7,341 files, the shell sequence
// repeated on 2026-09-20 reads 137-150 ms; the rest of the gap is
// unaccounted for. The same content is the same SHA whatever HEAD and the
// base are, which is what lets the stop gate remember a green tree.
func ContentTree(root, scratch string) (string, error) {
	out, err := git(root, "rev-parse", "--git-path", "index")
	if err != nil {
		return "", err
	}
	real := strings.TrimSpace(out)
	if !filepath.IsAbs(real) {
		real = filepath.Join(root, real)
	}
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		return "", err
	}
	index := filepath.Join(scratch, fmt.Sprintf("index-%d", os.Getpid()))
	defer os.Remove(index)
	data, err := os.ReadFile(real)
	switch {
	case err == nil:
		if err := os.WriteFile(index, data, 0o644); err != nil {
			return "", err
		}
		// The copy keeps the real index's mtime: git hashes a racy entry --
		// a file whose stat matches but which changed within the index's
		// own timestamp -- only when the index is no newer than the file.
		// A copy stamped now would call a same-size edit unchanged.
		keepMtime(real, index)
	case errors.Is(err, fs.ErrNotExist):
		// A repository without a commit may have no index yet; git makes one
		// where GIT_INDEX_FILE points.
	default:
		return "", err
	}
	env := []string{"GIT_INDEX_FILE=" + index}
	// Two calls, one arm: everything the working tree holds goes in, and
	// loomux's own state comes back out. --ignore-unmatch, because a
	// repository that has never carried that directory is the ordinary case.
	for _, stage := range [][]string{
		{"add", "-A"},
		{"rm", "-r", "-q", "--cached", "--ignore-unmatch", "--", ".loomux/state"},
	} {
		if _, err := gitWith(root, env, stage...); err != nil {
			return "", err
		}
	}
	tree, err := gitWith(root, env, "write-tree")
	return strings.TrimSpace(tree), err
}

// LocalBranches maps every local branch, by its full ref, to its commit, and
// names, sorted, the branches checked out in a worktree other than root's.
//
// Local branches are shared by every worktree of a repository, so a session
// working in another one moves branches root sees as well. Which worktree a
// branch is checked out in is git's word (`%(worktreepath)`), compared against
// git's word for root's worktree (`rev-parse --show-toplevel`) rather than
// against root as the caller spelled it.
func LocalBranches(root string) (heads map[string]string, elsewhere []string, err error) {
	out, err := git(root, "for-each-ref", "--format=%(objectname)%09%(refname)%09%(worktreepath)", "refs/heads")
	if err != nil {
		return nil, nil, err
	}
	// A bare repository has branches and no worktree to call its own.
	top, err := git(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, nil, err
	}
	top = strings.TrimSpace(top)
	heads = map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		// SplitN and not Fields: a branch checked out nowhere has an empty
		// third field, and a path may hold spaces.
		fields := strings.SplitN(strings.TrimRight(line, "\r"), "\t", 3)
		if len(fields) != 3 {
			continue
		}
		heads[fields[1]] = fields[0]
		if fields[2] != "" && !samePath(runtime.GOOS, fields[2], top) {
			elsewhere = append(elsewhere, fields[1])
		}
	}
	slices.Sort(elsewhere)
	return heads, elsewhere, nil
}

// samePath compares two paths git spelled, as the file system of goos does:
// Windows paths differ in slashes and in case without naming another
// directory. filepath.Clean folds the separators the running system's way;
// goos decides only the case.
func samePath(goos, a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if goos == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// LogOneline is `git log --oneline from..to`, one line per commit.
func LogOneline(root, from, to string) ([]string, error) {
	out, err := git(root, "log", "--oneline", from+".."+to)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimRight(line, "\r"); line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// parseRefs reads `<sha>\t<ref>` lines, the form of ls-remote and of the
// for-each-ref format above.
func parseRefs(out string) map[string]string {
	refs := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		sha, ref, ok := strings.Cut(strings.TrimRight(line, "\r"), "\t")
		if ok {
			refs[ref] = sha
		}
	}
	return refs
}

// keepMtime gives dst the modification time src has. A failure is not
// reported: the copy then carries the time it was written, which is what it
// had before -- blind to a racy edit, and to nothing else.
//
//coverage:exempt src was read a moment ago and dst was just written; neither the stat nor the chtimes can be made to fail from outside
func keepMtime(src, dst string) {
	if info, err := os.Stat(src); err == nil {
		_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	}
}
