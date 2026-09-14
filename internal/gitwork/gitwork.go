// Package gitwork answers what git knows about a working tree.
//
// Its own package because more than one hook asks: session-start records the
// commit a session begins on, and the stop gate measures against it. The
// Python original is src/ultraloom/worktree.py.
package gitwork

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
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
	command := exec.Command("git", arguments...)
	command.Dir = root
	// See gitenv: GIT_DIR and its relatives outrank command.Dir, so without
	// the strip this would answer about whatever GIT_DIR names instead of the
	// tree the caller asked about.
	command.Env = gitenv.Environ()
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
