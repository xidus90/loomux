// Package vcs is git for the maintenance layer: what it reads, and the one
// commit a decided case writes.
//
// Three questions are asked here. Reconciliation needs the previous version of
// a changed source to build a real diff, and git is the only place it exists:
// the register keeps a hash, never the text, and a shadow copy in the state
// directory would be a second register. A merge event needs its evidence -- the
// names a commit range touched and the subject of each commit in it. And both
// need to know which repository a registered area belongs to, which is derived
// from its path rather than declared anywhere.
//
// `git show` is used for the baseline and nowhere on the evidence path, by
// construction: no source text may reach the evidence of a merge case, which
// is what keeps "no reconciliation against code contents" holding however the
// range is shaped. The baseline is a different matter -- it is the one file the
// case is about, and the caller already holds it.
//
// No git library, here as in the Python original: the plumbing commands below
// are a stable, documented interface, while a binding would add a dependency
// for something os/exec already does exactly. Git reports failure through its
// exit code and not through its output -- a plumbing command can print a
// plausible-looking line and still have failed -- so every call is judged by
// what os/exec returns and never by what it printed.
//
// The write side, `commit_paths` in the original, is CommitPaths in commit.go.
//
// The original is `src/brain/maintenance/vcs.py`.
package vcs

import (
	"errors"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/xidus90/loomux/internal/gitenv"
)

// ErrOutsideRepository marks a path refused before git ever saw it.
//
// Refused here rather than by git, whose message for it names no repository.
var ErrOutsideRepository = errors.New("path is not inside the repository")

// ErrNotAnObjectName marks a commit range whose ends are not object names.
var ErrNotAnObjectName = errors.New("not an object name")

// ShowBlob is the committed bytes of relative at HEAD, read from inside
// directory.
//
// nil is the answer for every way that version can be absent -- no repository
// (a vault without git keeps working), an unborn branch, a path git has never
// seen, a working directory that is gone, git not installed. None of them is a
// defect, and the caller falls back to a package without a baseline and says so
// out loud there. The one error is a path that leaves the repository, which is
// a caller's mistake rather than a state of the world.
//
// `HEAD:./<path>` resolves against the working directory, so directory may sit
// anywhere below the repository root and no prefix arithmetic is needed.
//
// Bytes, not text: the caller folds CRLF itself and hashes the result against
// the register, and decoding first would fail that comparison for any source
// git holds that is not valid UTF-8.
func ShowBlob(directory, relative string) ([]byte, error) {
	inside, err := insideRepository(relative)
	if err != nil {
		return nil, err
	}
	return baseline(run(directory, "show", "HEAD:./"+inside))
}

// baseline is what `git show` answered, read as a baseline.
//
// nil for every failure, and never nil when git handed a version over: nil is
// this package's word for "there is no baseline", while a committed empty file
// is a baseline that happens to be empty, and the caller cannot tell them apart
// any other way.
//
// The guard is not dead code dressed as caution. Measured on 2026-09-20 with Go
// 1.27 on windows/amd64, os/exec answers empty output with an empty slice that
// is not nil -- `Output` fills a bytes.Buffer, and `ReadFrom` grows it before it
// reads -- so real git never reaches the guard today. But os/exec promises
// nothing of the sort: `Cmd.Output` is documented as returning "its standard
// output", and a nil slice would satisfy that. The guard is what makes the
// distinction this package's own rather than a loan from the standard library,
// and internal_test.go calls it directly, because git cannot.
func baseline(out []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, nil
	}
	if out == nil {
		return []byte{}, nil
	}
	return out, nil
}

// ChangedPaths is the file paths a commit range touched -- names only, never
// content.
//
// An error for every way the range cannot be read: the work tree is gone (a
// throwaway worktree that recorded an event and was then removed), the commits
// were garbage-collected, git is not installed, or an end is not an object
// name. The caller keeps the event rather than passing the error on, because an
// event dropped without a case is a merge nobody hears about again.
//
// A range that touched nothing is no error: it yields no paths, and the caller
// writes no heading over an empty list.
func ChangedPaths(directory, first, last string) ([]string, error) {
	span, err := commitRange(first, last)
	if err != nil {
		return nil, err
	}
	return readLines(directory, "diff", "--name-only", span)
}

// CommitSubjects is the subject of each commit in a range.
//
// `%s` rather than `%B`, so everything after the first blank line stays out: a
// commit body carries pasted diffs, stack traces and occasionally a secret,
// while the subject answers the one question a merge case asks -- was the
// planned thing built?
//
// `%s` is not "the first line". It is git's subject: the message up to the
// first blank line, with its newlines folded into spaces. A message that opens
// with a code block and no blank line therefore arrives here as one long
// subject carrying that code. Nothing is breached by it -- commit messages are
// evidence this layer is allowed to carry -- but the assurance has to be stated
// narrowly: this call opens no file, so nothing it returns is the content of
// one. Whoever writes code into a commit subject will read it back in the
// package.
func CommitSubjects(directory, first, last string) ([]string, error) {
	span, err := commitRange(first, last)
	if err != nil {
		return nil, err
	}
	return readLines(directory, "log", "--format=%s", span)
}

// RepositoryRoot is the work tree directory belongs to, or "" when it belongs
// to none.
//
// Derived from a registered area's path rather than declared in the registry: a
// `repo = true` field would be a second place to keep the same fact, and it
// would go on claiming a repository after the user deleted the `.git` beside
// it.
func RepositoryRoot(directory string) (string, error) {
	return readPath(directory, "rev-parse", "--path-format=absolute", "--show-toplevel")
}

// CommonDirectory is the one git directory that directory and all its sibling
// worktrees share, or "" when directory belongs to no repository.
//
// This, and not the work tree, is a repository's identity. Every linked
// worktree has a top level of its own but one common directory, so a merge
// event -- recorded by a hook the worktrees share -- can only tell "my
// repository" from "a foreign repository" by this value.
func CommonDirectory(directory string) (string, error) {
	return readPath(directory, "rev-parse", "--path-format=absolute", "--git-common-dir")
}

// insideRepository is `_reject_path_outside`: the relative path as git wants to
// read it, or a refusal.
func insideRepository(relative string) (string, error) {
	// Backslashes folded on every platform, as the original folds them: a
	// caller that built the path with filepath.Join on Windows hands over
	// `sub\a.md`, and git reads only forward slashes after `HEAD:`.
	posix := strings.ReplaceAll(relative, "\\", "/")
	// The climb is read off the unclean elements. path.Clean folds `a/../b` to
	// `b`, so cleaning first would let through exactly the escape this refuses.
	for _, element := range strings.Split(posix, "/") {
		if element == ".." {
			return "", fmt.Errorf("%w: %s", ErrOutsideRepository, relative)
		}
	}
	// Three tests and not one, because neither alone is enough on Windows:
	// filepath.IsAbs says no to `/outside.md` and to `//server/share` there,
	// while the prefix test says no to `C:\outside.md` everywhere.
	if relative == "" || strings.HasPrefix(posix, "/") || filepath.IsAbs(relative) {
		return "", fmt.Errorf("%w: %s", ErrOutsideRepository, relative)
	}
	return path.Clean(posix), nil
}

// commitRange is `<first>..<last>`, or a refusal when either end is not an
// object name.
//
// The range comes off a file a shell hook appends to and a hand may edit, and
// git reads a leading `-` as an option: `--output=<file>..HEAD` is one word
// that makes `git diff` write to a file. Refusing anything but a lower-case hex
// name costs nothing -- the hook only ever writes `rev-parse` output -- and
// removes the whole class.
func commitRange(first, last string) (string, error) {
	if !objectName(first) || !objectName(last) {
		return "", fmt.Errorf("%w: %s..%s", ErrNotAnObjectName, first, last)
	}
	return first + ".." + last, nil
}

// objectName reports whether name is 4 to 64 lower-case hex digits.
//
// Spelled out rather than compiled from a pattern, because this package parses
// nothing at load time and a regexp would need a place to live.
func objectName(name string) bool {
	if len(name) < 4 || len(name) > 64 {
		return false
	}
	for index := 0; index < len(name); index++ {
		digit := name[index]
		if (digit < '0' || digit > '9') && (digit < 'a' || digit > 'f') {
			return false
		}
	}
	return true
}

// readLines is a read-only git call whose answer is a list of lines.
func readLines(directory string, arguments ...string) ([]string, error) {
	out, err := run(directory, arguments...)
	if err != nil {
		return nil, fmt.Errorf("git %s in %s failed: %w%s",
			strings.Join(arguments, " "), directory, err, said(err))
	}
	lines := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		// Only the line ending comes off. A path git prints may legitimately
		// begin or end with a space, and trimming the line itself would hand
		// back a name no file has.
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

// said is git's own account of a refusal, which the error alone does not carry.
//
// An *exec.ExitError says "exit status 128" and nothing about why; the reason --
// "fatal: Invalid revision range ..." -- is on stderr, which Output collects
// into the error for exactly this. Without it a range failure in the
// maintenance layer is not debuggable at all: the caller swallows the error by
// design, and a log line reading "exit status 128" names nothing.
//
// Empty when git never ran, because there is no stderr then and an
// unconditional separator would end every spawn failure in a dangling ": ",
// which reads as a truncated message rather than a complete one.
// internal/gitwork/gitwork.go does the same for the same reason.
func said(err error) string {
	var exit *exec.ExitError
	if !errors.As(err, &exit) || len(exit.Stderr) == 0 {
		return ""
	}
	return ": " + strings.TrimSpace(string(exit.Stderr))
}

// readPath is a read-only git call whose answer is one absolute path, with ""
// for the directory that belongs to no repository.
//
// The two kinds of failure are told apart here and they do not mean the same
// thing. git ran and refused: that is "no repository", an ordinary answer in a
// vault the user keeps without git. git never ran -- the directory is gone, git
// is not installed -- and there is no answer at all, which the caller has to
// see. The Python original arrives at the same split by two routes, raising on
// the failed spawn and comparing the return code against 0.
func readPath(directory string, arguments ...string) (string, error) {
	out, err := run(directory, arguments...)
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("git %s in %s could not be run: %w", strings.Join(arguments, " "), directory, err)
	}
	// Cleaned the way the rest of this program spells a path: git answers with
	// forward slashes on Windows too, and CommonDirectory is compared against
	// paths a caller built with filepath, where the direction would decide the
	// comparison.
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

func run(directory string, arguments ...string) ([]byte, error) {
	return runWith(directory, nil, "", arguments...)
}

// runWith is run with variables of this package's own choosing added after the
// strip -- the scratch index of CommitPaths -- and stdin fed from a string,
// which commit-tree reads its message from. An empty stdin is no input at all,
// the same as run's.
func runWith(directory string, environment []string, stdin string, arguments ...string) ([]byte, error) {
	command := exec.Command("git", arguments...)
	command.Dir = directory
	// See gitenv: GIT_DIR and its relatives outrank command.Dir, so without the
	// strip every call here would answer about whatever GIT_DIR names instead
	// of the tree the caller asked about -- and this layer runs from a hook,
	// where git exports exactly those.
	command.Env = append(gitenv.Environ(), environment...)
	command.Stdin = strings.NewReader(stdin)
	// Output and not CombinedOutput: git writes a warning -- an ambiguous
	// refname, a safe.directory note -- to stderr on a call that succeeds, and
	// folded into the answer such a line would travel on as part of a path or
	// of a blob.
	return command.Output()
}
