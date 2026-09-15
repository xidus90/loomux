package guard

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
)

// errLinkCycle says that the links on a path lead in a circle, so there
// is no place for the path to name. Every caller reads it as a refusal:
// a barrier whose whole answer is *where* a write lands cannot answer a
// path that lands nowhere.
var errLinkCycle = errors.New("its links lead in a circle")

// resolvePath is `Path.resolve()` as the barrier uses it: absolute, with
// `..` collapsed, with every link on the existing part of the path
// followed, aliases folded to the names they stand for and the spelling
// of each existing component closed. Non-strict like Python's -- a target
// that does not exist yet is exactly the call a write makes, so a
// resolver that failed there would refuse every new page.
//
// It is a port of `ntpath.realpath` rather than an imitation of it, and
// that is the lesson of the round that produced it: three hand-written
// approximations in a row were each weaker than the side they replace.
// `realpath` is two steps, and leaving either out opens a spelling.
//
//   - The anchor. `ntpath.isabs` is False for a rooted path that names
//     no drive, so `realpath` reaches for the working directory and
//     `ntpath.join` keeps that directory's drive alone. `anchor` below.
//   - `GetFinalPathNameByHandle`, through `finalName`, over the longest
//     prefix of the path that can be opened; whatever is missing is
//     joined back on as it was spelt. That one call folds an 8.3 alias
//     to the name it stands for, closes the case and the trailing dot or
//     blank of every component that exists, and follows junctions --
//     none of which `filepath.EvalSymlinks` does on Windows, where it
//     answers a junction's own path and `os.Lstat` calls the junction
//     irregular rather than a link.
//
// One difference from `realpath` is deliberate. Python keeps the `\\?\`
// device prefix where the plain path would not resolve back to the same
// file, which is how a component spelt with a trailing blank comes back
// as a device path and then matches no registered tree at all; this side
// strips the prefix unconditionally, so a write into the bundle's own
// `sub /x.md` lands in the bundle. That is the one call where this side
// is the more permissive, and it is the right one: the runtime creates
// `sub` and the file lands inside the tree the registration opens.
// `tests/test_guard_parity.py` states both answers.
//
// One spelling is refused before any of that happens: a volume without a
// root. `errDriveRelative` says why.
func resolvePath(target string) (string, error) {
	if err := rejectDriveRelative(target); err != nil {
		return "", err
	}
	full := target
	if !filepath.IsAbs(full) {
		if cwd, err := os.Getwd(); err == nil {
			full = anchor(cwd, full)
		}
	}
	return finalPath(filepath.Clean(full))
}

// finalPath is `_getfinalpathname_nonstrict` (ntpath.py:607-660): find as
// much of the path as the file system will answer for and join the rest
// back on.
//
// Three arms per turn, in Python's order. The path opens, and the answer
// is the whole of it. It does not, but its links lead somewhere the
// opening could not reach -- a junction pointing at a directory that is
// itself absent -- and then the link target carries the rest. Or neither,
// and one component moves from the path onto the tail and the shortened
// path is asked again.
func finalPath(path string) (string, error) {
	tail := ""
	for path != "" {
		resolved, err := finalName(path)
		if err == nil {
			return withTail(resolved, tail), nil
		}
		// An error the file system raises about something other than the
		// path's absence is no answer at all, and guessing past it would
		// be the barrier deciding on a path it never resolved.
		if !stopsResolving(err) {
			return "", err
		}
		linked, err := readlinkDeep(path)
		if err != nil {
			return "", err
		}
		if linked != path {
			return withTail(linked, tail), nil
		}
		head, name := splitPath(path)
		if name == "" {
			// The anchor, which has no parent to move to. Python returns
			// `path + tail` here for the same reason and guards it with
			// `if path and not name`. The left half of that guard has
			// nothing to do on this side: the loop has already said the
			// path is not empty, and a non-empty path whose last name is
			// empty ends in a separator, so its parent carries at least
			// that separator and cannot be empty either. Left standing,
			// it survived (a2) -- a guard nothing can make false.
			return withTail(head, tail), nil
		}
		path, tail = head, withTail(name, tail)
	}
	return tail, nil
}

// withTail is `join(head, tail) if tail else head`, which is how ntpath
// puts the two halves back together. `filepath.Join` cleans, and that is
// harmless here: the whole path went through `Clean` before the walk
// began, so no `.` or `..` is left for a second cleaning to find, and
// `Clean` leaves a trailing dot or blank on a component alone.
//
// Which is also why the empty-tail arm survives (a1): `Join(head, "")`
// is `Clean(head)`, and every head that reaches here is already clean,
// so the two arms answer alike. It stands because it is ntpath's shape
// and because a join for nothing is an allocation on the call the host
// makes most often -- not because it decides anything.
func withTail(head, tail string) string {
	if tail == "" {
		return head
	}
	return filepath.Join(head, tail)
}

// readlinkDeep is `_readlink_deep` (ntpath.py:565-605): follow the links
// at this one path until something that is not a link answers, and give
// back what was reached -- the path itself where it is no link at all.
//
// The set of visited paths is what makes the answer independent of any
// bound. A hop *count* cannot decide a cycle of even length: each turn
// swaps one link for the next, so the path alternates and the parity of
// wherever the count stops picks the verdict. Python's own bound is its
// `seen` set, and it lands on one side of such a cycle only because it
// holds the first entry unprefixed and every later one as a `\\?\` path
// -- an accident of the same standing as a hop count. Refusing is the one
// answer that rests on neither: measured, Python allows the two-junction
// cycle, and this side refuses it. Nothing honest is lost, because the
// file system refuses to open such a path anyway.
func readlinkDeep(path string) (string, error) {
	seen := map[string]bool{}
	current := path
	for {
		key := strings.ToLower(current)
		if seen[key] {
			return "", errLinkCycle
		}
		seen[key] = true
		via, err := os.Readlink(current)
		if err != nil {
			// Not a link, or not readable as one. Python stops on the
			// same set of errors and answers what it holds.
			return current, nil
		}
		current = under(filepath.Dir(current), via)
	}
}

// splitPath is `ntpath.split`: the parent and the last name, with the
// anchor's separator left on the parent so that `C:\` splits into itself
// and nothing. `filepath.Dir` and `filepath.Base` cannot serve, because
// `Base("C:\\")` answers the separator rather than the empty name that
// ends the walk.
func splitPath(path string) (head, name string) {
	volume := filepath.VolumeName(path)
	rest := path[len(volume):]
	cut := len(rest)
	for cut > 0 && !os.IsPathSeparator(rest[cut-1]) {
		cut--
	}
	name = rest[cut:]
	head = rest[:cut]
	for len(head) > 1 && os.IsPathSeparator(head[len(head)-1]) {
		head = head[:len(head)-1]
	}
	return volume + head, name
}

// errDriveRelative says that a target names a volume but no root, so the
// place it stands for is that volume's own current directory -- a directory
// this process does not hold and cannot ask for.
var errDriveRelative = errors.New(
	"it names a volume without a root, so it points at that volume's own " +
		"current directory")

// rejectDriveRelative closes the spelling `D:evil.txt`, inherited verbatim
// from ultra-brain along with `anchor`.
//
// `filepath.IsAbs` is false for it, so `anchor` joins it onto the working
// directory -- and `filepath.Join` concatenates, answering
// `D:\work\C:secrets.txt` for `Join("D:\\work", "C:secrets.txt")`. Windows
// resolves the spelling against the current directory *of that drive*, so the
// write lands somewhere this answer never looked, while `spelled` folds the
// bogus component to what stands before its colon and lets the fake path pass
// for a component prefix of a registered tree. Measured with the hook standing
// inside a workspace: `Decide` answered ALLOW for `D:evil.txt` and the write
// went to `D:\evil.txt`.
//
// Refusing is the fix rather than a reimplementation of ntpath's per-drive
// current directories: nothing this process can read says where drive D
// stands, and no legitimate host sends the spelling. `Decide` turns the error
// into a refusal, which is the whole answer.
//
// A bare volume (`C:`, and a UNC share with nothing after it) takes the same
// arm: it has no byte after the volume at all, and a directory is not a place
// a write lands either.
func rejectDriveRelative(target string) error {
	volume := filepath.VolumeName(target)
	if volume == "" {
		return nil
	}
	rest := target[len(volume):]
	if rest == "" || !os.IsPathSeparator(rest[0]) {
		return errDriveRelative
	}
	return nil
}

// anchor is `join(cwd, path)` as `ntpath` performs it, which is not what
// `filepath.Join` does for one spelling: a path that starts with a
// separator and names no volume.
//
// Measured on this machine, Python 3.14: `ntpath.isabs("\\Users\\x")` is
// False, so `realpath` reaches for the working directory -- and
// `ntpath.join` then keeps that directory's *drive* alone, answering
// `C:\Users\x`. `filepath.Join` would answer `<cwd>\Users\x` instead, and
// the difference is not academic: a hook stands in the tree the session
// edits, so joining folds every such path into a tree that is open.
// Node's `path.resolve` anchors at the drive as well, so the runtime that
// performs the write puts the file where Python looked, not where a join
// would.
//
// The volume is prepended rather than joined, because joining would clean
// the result and `Clean` is the caller's own next step -- and because a
// trailing dot or blank on the last component has to survive both.
func anchor(cwd, path string) string {
	if filepath.VolumeName(path) == "" && path != "" &&
		os.IsPathSeparator(path[0]) {
		return filepath.VolumeName(cwd) + path
	}
	return filepath.Join(cwd, path)
}

// under joins a link target onto the directory the link sat in, the way
// `Path(base) / via` does it: an absolute right-hand side replaces the
// left outright. That is how a target naming a place of its own carries
// the walk out of the tree the link stood in -- and on Windows every
// junction target is absolute, so this is the arm the ordinary case
// takes there while a posix symlink usually takes the other.
func under(base, via string) string {
	if filepath.IsAbs(via) {
		return filepath.Clean(via)
	}
	return filepath.Join(base, via)
}

// spelling is `_spelling`: one path
// component reduced to what the file system actually distinguishes.
// Everything from the first colon is a stream name and never part of a
// file name, the case is folded, and a trailing dot or blank comes off --
// Windows strips both when it creates the file, and `resolve` closes the
// spelling only where the path already exists.
func spelling(name string) string {
	if cut := strings.IndexByte(name, ':'); cut >= 0 {
		name = name[:cut]
	}
	return strings.TrimRight(strings.ToLower(name), ". ")
}

// components splits a path the way `PurePath.parts` does: the anchor
// first, then every name. The anchor is the volume plus the separator
// that follows it, so `C:\a` yields `C:\` and `a` -- and an anchor of no
// separator (`C:a`, drive-relative) yields just `C:`.
func components(path string) []string {
	volume := filepath.VolumeName(path)
	rest := path[len(volume):]
	anchor := volume
	if rest != "" && os.IsPathSeparator(rest[0]) {
		anchor += rest[:1]
		rest = rest[1:]
	}
	parts := make([]string, 0, strings.Count(rest, "/")+2)
	if anchor != "" {
		parts = append(parts, anchor)
	}
	// The ASCII test comes first because `byte(r)` truncates: U+012F
	// carries `/` in its low byte, and without it that one rune would
	// cut a name in half. Where the bound itself sits decides nothing --
	// `<` against `<=` survives every round, since 0x80 is no separator
	// either -- so only the *presence* of the test is a rule.
	return append(parts, strings.FieldsFunc(rest, func(r rune) bool {
		return r < 0x80 && os.IsPathSeparator(byte(r))
	})...)
}

// spelled is `_spelled`: one resolved path
// as the components the file system keeps apart. The anchor goes through
// `spelling` like every other component, which reduces `C:\` to `c` -- a
// lossy answer and a harmless one, because both sides of every comparison
// are reduced the same way.
func spelled(path string) []string {
	parts := components(path)
	for i, part := range parts {
		parts[i] = spelling(part)
	}
	return parts
}

// inside is `_inside`: whether this one path
// lies in one of these trees, writable or forbidden.
//
// A prefix of *components*, never of characters, which is what keeps
// `.../space-neu` out of `.../space`. And spelled rather than raw,
// because `resolve` closes a spelling only where the path exists: with
// the directory absent, `vault/91/space./x.md` resolves unchanged, goes on
// lying in the writable tree above it and stops lying in the read-only one
// below -- and the runtime that performs the write then creates the
// missing directory under the name Windows folds it to.
func inside(resolved string, roots []string) bool {
	target := spelled(resolved)
	for _, root := range roots {
		parts := spelled(root)
		if len(parts) <= len(target) &&
			slices.Equal(target[:len(parts)], parts) {
			return true
		}
	}
	return false
}

// pathsFold says whether this platform's file system equates paths that
// differ only in case. `PurePath.__eq__` and `is_relative_to` both go
// through `normcase`, which lowers on Windows and does nothing on posix:
// measured here, `Path("C:/A") == Path("c:/a")` is True and
// `Path("C:/a/b").is_relative_to(Path("C:/A"))` is True.
//
// A variable and not a call, so that a test can drive the branch this
// build does not take; nothing outside a test writes it.
var pathsFold = runtime.GOOS == "windows"

// normalised is one path as `normcase` leaves it, split into components.
// Unlike `spelled` it folds nothing but the case -- a trailing dot is a
// different name to pathlib, and the four comparisons that use this are
// pathlib's own.
func normalised(path string) []string {
	parts := components(path)
	if !pathsFold {
		return parts
	}
	for i, part := range parts {
		parts[i] = strings.ToLower(part)
	}
	return parts
}

// pathsEqual is `Path.__eq__`.
func pathsEqual(left, right string) bool {
	return slices.Equal(normalised(left), normalised(right))
}

// isRelativeTo is `Path.is_relative_to`: a prefix of components under
// `normcase`. `PROPOSAL` is compared with `==` on the bare name instead,
// and that difference is deliberate on the Python side -- so a capital P
// names a file this exemption does not cover, even where the disk would
// open the same one.
func isRelativeTo(path, base string) bool {
	parts := normalised(base)
	target := normalised(path)
	return len(parts) <= len(target) &&
		slices.Equal(target[:len(parts)], parts)
}

// gitCommonDirTimeout is the wait `_git_common_dir` allows. A hook that
// waited forever on a hung git would block the session instead of
// deciding, which is its own kind of failure.
const gitCommonDirTimeout = 10 * time.Second

// askGit runs `git rev-parse --git-common-dir` in one directory. A
// variable so that the arms below it -- a failure, an empty answer, a
// relative answer -- are reachable without a repository per case; nothing
// outside a test writes it.
var askGit = func(directory string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(),
		gitCommonDirTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", "-C", directory,
		"rev-parse", "--git-common-dir")
	// `GIT_DIR` and its relatives outrank `-C`, and this hook runs wherever
	// the session runs -- inside a git hook among other places. Inherited,
	// they make every directory answer with the same common directory, so
	// two unrelated trees compare equal and a foreign one passes for a
	// worktree of the registered repository. Observed, not feared.
	command.Env = gitenv.Environ()
	out, err := command.Output()
	return string(out), err
}

// gitCommonDir is `_git_common_dir`: the
// directory every worktree of one repository shares, or "" on any
// failure -- and the caller reads that as "not the same repo". Refusing
// is the only safe direction: a barrier that opens when git is missing,
// slow or confused is not a barrier.
//
// Both of this function's error arms survive (a1), and for one reason:
// each failure it catches also leaves the value it would have used
// empty, so the guard below catches what the guard above would have.
// A failing `askGit` prints nothing, and the emptiness test answers "";
// a failing `resolvePath` answers "" beside its error. The arms stand
// because a caller reading a value the callee said nothing about is a
// habit that stops being harmless the moment either callee changes.
func gitCommonDir(path string) string {
	out, err := askGit(path)
	if err != nil {
		return ""
	}
	printed := strings.TrimSpace(out)
	// git answers relative to the directory it was pointed at
	// (`../../.git` from a subdirectory) and absolutely from a linked
	// worktree; joining covers both, because an absolute right-hand side
	// replaces the left. An empty answer is a failure rather than a join:
	// it would name the directory itself, and two unrelated directories
	// would then compare equal.
	if printed == "" {
		return ""
	}
	// A path this side cannot resolve is one more way for the question
	// "is this the same repository" to have no answer, and "" is what
	// this function already says in that case.
	if !filepath.IsAbs(printed) {
		printed = filepath.Join(path, printed)
	}
	common, err := resolvePath(printed)
	if err != nil {
		return ""
	}
	return common
}

// sameRepository is `_same_repository`:
// whether `candidate` is the registered tree itself or a worktree of it.
//
// Two demands, and the second is the one that is easy to miss. Sharing
// the common directory only says "somewhere in this repository", which
// every subdirectory does too -- so a manifest planted in one of them
// would open its own subtree. A worktree root is the one directory that
// carries a `.git` of its own (a directory in the main checkout, a file
// in a linked one), and asking for that costs no further process.
func sameRepository(candidate, registered string) bool {
	here, hereErr := resolvePath(candidate)
	there, thereErr := resolvePath(registered)
	// Unresolvable is not the same repository. The caller reads a false
	// here as "this manifest declares nothing", which closes the tree
	// rather than opening it.
	if hereErr != nil || thereErr != nil {
		return false
	}
	if pathsEqual(here, there) {
		return true
	}
	if _, err := os.Stat(filepath.Join(candidate, ".git")); err != nil {
		return false
	}
	common := gitCommonDir(candidate)
	return common != "" && common == gitCommonDir(registered)
}
