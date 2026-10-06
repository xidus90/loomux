package vcs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ErrRefMoved marks a branch a foreign process moved while the commit was being
// built.
//
// Nothing was written, and the caller may start over from HEAD -- once. A
// second ErrRefMoved on the retry goes to the user instead: the window is
// milliseconds wide, so losing it twice means something commits continuously,
// and a loop would never end.
var ErrRefMoved = errors.New("the branch moved while committing")

// ErrNotARepository marks a vault path that is gone, a state no caller recovers
// from. A directory that merely has no repository is a legitimate setup and no
// error at all.
var ErrNotARepository = errors.New("not a directory")

// Commit is what CommitPaths did to the ref.
//
// Created is what the caller cannot see from Head alone: an unchanged tree
// yields the head that was already there, and without the flag that answer is
// indistinguishable from a commit that was written.
type Commit struct {
	Head    string // the ref's commit after the call
	Created bool   // false: the tree was unchanged, nothing was committed
}

// inProgress is what a rebase or an unfinished merge leaves in the git
// directory. Both park HEAD somewhere it will not stay. A cherry-pick is not
// among them, as in the original.
var inProgress = []string{"rebase-merge", "rebase-apply", "MERGE_HEAD"}

// beforeUpdateRef runs, when set, between commit-tree and the second reading of
// HEAD -- the window a foreign process would use. Tests only; nil otherwise.
var beforeUpdateRef func()

// CommitPaths commits exactly add (written) and remove (deleted) onto the
// current ref of repo through a scratch index. It returns nil, nil when
// repo is no git repository.
//
// The user's index is never touched: it is shared with Obsidian's git plugin
// and every other process in the tree, and a foreign `index.lock` would fail
// this call after the wiki is already written. `git commit -- <paths>` is no
// way out either, because it commits the working-tree state of those paths
// while the caller's hash guard protects only the target page. Every call
// builds its tree in an index of its own, in a fresh directory below scratch
// that is removed on return: scratch is shared by every vault and every process
// of the machine, and a shared index file would let one call empty the other's
// tree -- a commit that deletes every file but its own. A fresh index also never
// carries an earlier run's staged paths into the first commit of an unborn
// branch, which has no tree to read over it.
//
// `update-ref` gets the old value it expects, so a foreign commit made in
// between is refused rather than silently lost. Telling a lost swap from any
// other `update-ref` failure means re-reading the ref, because git's stderr is
// localised and unfit to branch on. That reading is not exact in either
// direction -- a permanent failure that coincides with a foreign commit reads
// as ErrRefMoved, and a lost swap reads as a plain error when the foreign
// process put the ref back -- but both misreadings are safe: nothing is
// overwritten either way.
//
// HEAD is re-read just before the swap, because the swap protects the ref's
// value and not its role: after a branch switch the old ref still holds the
// expected commit, and the commit would land where `git show HEAD` does not
// find it. One process spawn still separates that reading from the swap; git
// offers no operation that takes "this ref, only while HEAD names it".
//
// A failing rev-parse is read as "no repository", which a safe.directory
// refusal also produces. Removal paths are read from the index, not from disk,
// and the working tree is never touched: the caller deletes the case directory
// there itself. commit-tree bypasses the repository's pre-commit hooks by
// construction -- the price of not sharing the index. The identity is the
// repository's own configuration; gitenv strips every variable that could
// carry another one in.
//
// The original is `commit_paths` in `src/brain/maintenance/vcs.py`.
func CommitPaths(repo, message string, add, remove []string, scratch string) (*Commit, error) {
	if info, err := os.Stat(repo); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotARepository, repo)
	}
	add, remove, err := named(add, remove)
	if err != nil {
		return nil, err
	}

	g := &session{repo: repo}
	gitDir, ok := g.answer("rev-parse", "--absolute-git-dir")
	if g.err != nil {
		return nil, g.err
	}
	if !ok {
		return nil, nil
	}
	if err := refuseOperationInProgress(strings.TrimSpace(gitDir)); err != nil {
		return nil, err
	}

	ref := g.currentRef()
	old := g.resolve(ref)
	index, dir, err := scratchIndex(scratch)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	g.index = index
	if old != "" {
		g.must("", "read-tree", old)
	}
	for _, relative := range add {
		// `--add <path>` rather than `--cacheinfo`: it reads the working tree
		// through the clean filter, so `.gitattributes` still decides the line
		// endings, and the file mode stays git's business.
		g.must("", "update-index", "--add", "--", relative)
	}
	for _, relative := range remove {
		for _, tracked := range g.expand(relative) {
			g.must("", "update-index", "--force-remove", "--", tracked)
		}
	}
	tree := strings.TrimSpace(g.must("", "write-tree"))
	// An empty commit would record a decision that left no trace in the vault.
	if old != "" && g.err == nil && tree == strings.TrimSpace(g.must("", "rev-parse", old+"^{tree}")) {
		return &Commit{Head: old, Created: false}, nil
	}
	arguments := []string{"commit-tree", tree}
	if old != "" {
		arguments = append(arguments, "-p", old)
	}
	created := strings.TrimSpace(g.must(message, arguments...))
	if g.err != nil {
		return nil, g.err
	}

	if beforeUpdateRef != nil {
		beforeUpdateRef()
	}
	if g.currentRef() != ref && g.err == nil {
		return nil, fmt.Errorf("%w: HEAD left %s", ErrRefMoved, ref)
	}
	g.must("", "update-ref", ref, created, old)
	if g.err == nil {
		return &Commit{Head: created, Created: true}, nil
	}
	// The ref is read again on a session of its own: the failure above must not
	// silence the one reading that tells a lost swap from a refusal.
	failed := g.err
	g.err = nil
	if g.resolve(ref) != old && g.err == nil {
		return nil, fmt.Errorf("%w: %s moved away from %s", ErrRefMoved, ref, nothing(old))
	}
	return nil, failed
}

// named is add and remove as git wants to read them, or the refusal of a path
// that leaves the repository or stands in both lists.
//
// The removes run last, so a path in both would leave as a deletion -- a
// contradictory request answered silently, in the one direction that loses
// knowledge. The comparison is on the cleaned spelling, so `a\b` and `a/b` are
// the same path here as they are to git.
func named(add, remove []string) ([]string, []string, error) {
	cleanAdd, err := insideAll(add)
	if err != nil {
		return nil, nil, err
	}
	cleanRemove, err := insideAll(remove)
	if err != nil {
		return nil, nil, err
	}
	added := map[string]bool{}
	for _, relative := range cleanAdd {
		added[relative] = true
	}
	both := []string{}
	for _, relative := range cleanRemove {
		if added[relative] {
			both = append(both, relative)
		}
	}
	if len(both) > 0 {
		sort.Strings(both)
		return nil, nil, fmt.Errorf("path is both added and removed: %s", strings.Join(both, ", "))
	}
	return cleanAdd, cleanRemove, nil
}

func insideAll(paths []string) ([]string, error) {
	inside := make([]string, 0, len(paths))
	for _, relative := range paths {
		clean, err := insideRepository(relative)
		if err != nil {
			return nil, err
		}
		inside = append(inside, clean)
	}
	return inside, nil
}

// refuseOperationInProgress refuses a commit that the end of a rebase would
// orphan.
//
// Nothing downstream could notice: `symbolic-ref` fails during a rebase just as
// on a legitimately detached HEAD, so the commit goes onto HEAD, and the second
// reading still says HEAD. obsidian-git runs `pull --rebase`, so this is a
// window the user opens routinely.
func refuseOperationInProgress(gitDir string) error {
	for _, name := range inProgress {
		marker := filepath.Join(gitDir, name)
		if _, err := os.Stat(marker); err == nil {
			return fmt.Errorf("%s exists: finish the rebase or merge first", marker)
		}
	}
	return nil
}

// scratchIndex is a fresh directory below scratch and the absolute path of the
// index inside it. The index file does not exist yet, which git reads as an
// empty index; a zero-byte file would be refused. Absolute, because git
// resolves GIT_INDEX_FILE against the repository it runs in, not against this
// process.
func scratchIndex(scratch string) (file, dir string, err error) {
	dir, err = filepath.Abs(scratch)
	if err == nil {
		err = os.MkdirAll(dir, 0o755)
	}
	if err == nil {
		dir, err = os.MkdirTemp(dir, "index-")
	}
	if err != nil {
		return "", "", fmt.Errorf("scratch index in %s is unusable: %w", scratch, err)
	}
	return filepath.Join(dir, "index"), dir, nil
}

// nothing is how an unborn ref's value reads in a message.
func nothing(old string) string {
	if old == "" {
		return "nothing"
	}
	return old
}

// session is the git calls of one CommitPaths, stopping at the first failure.
//
// Once err is set every later call is skipped and answers empty, so the steps
// of a commit read as a plain sequence and are judged where the sequence
// branches, not after each call. The original raises instead; the effect is
// the same -- nothing after a failed call runs, and the first failure is the
// one reported.
type session struct {
	repo  string
	index string // GIT_INDEX_FILE, once the scratch index exists
	err   error
}

// answer is a call whose refusal is an answer rather than a failure: ok is
// false when git ran and said no. Only a git that cannot be started fails the
// session.
func (g *session) answer(arguments ...string) (string, bool) {
	out, err := g.call("", arguments...)
	return out, err == nil
}

// must is a call whose refusal fails the session, with git's own reason.
func (g *session) must(stdin string, arguments ...string) string {
	out, err := g.call(stdin, arguments...)
	if err != nil && g.err == nil {
		g.err = fmt.Errorf("git %s failed%s", strings.Join(arguments, " "), said(err))
	}
	return out
}

// call runs one git command unless an earlier one failed, and fails the session
// when git cannot be started at all. The output is raw: `ls-files -z` carries
// path bytes that trimming would falsify, so every caller that wants a bare
// token trims for itself.
func (g *session) call(stdin string, arguments ...string) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	var environment []string
	if g.index != "" {
		environment = []string{"GIT_INDEX_FILE=" + g.index}
	}
	out, err := runWith(g.repo, environment, stdin, arguments...)
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		g.err = fmt.Errorf("git %s in %s could not be run: %w", strings.Join(arguments, " "), g.repo, err)
	}
	return string(out), err
}

// currentRef is the branch HEAD names, or HEAD itself when it is detached -- a
// detached HEAD has no branch, but takes the same swap.
func (g *session) currentRef() string {
	out, ok := g.answer("symbolic-ref", "--quiet", "HEAD")
	if !ok {
		return "HEAD"
	}
	return strings.TrimSpace(out)
}

// resolve is the commit ref points at, or "" for an unborn branch -- which is
// how `update-ref` spells "must not exist yet".
func (g *session) resolve(ref string) string {
	out, _ := g.answer("rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return strings.TrimSpace(out)
}

// expand turns one removal path into the index entries it covers.
//
// `update-index --force-remove` reads its argument literally, while `ls-files`
// reads a pathspec: a directory -- the shape a case takes -- passes any check
// phrased with `ls-files` and is then removed by nothing, exit 0. So the
// expansion happens here and only literal entries are staged. `:(literal)`
// keeps a `*` from reaching every case directory beside the one meant. The
// listing stays untrimmed: git sorts bytewise, an entry beginning with a space
// comes first, and trimming would cost it that space -- `--force-remove` on
// the shortened name exits 0 and removes nothing. An empty expansion is a path
// git does not track, and that is an error rather than a silent no-op.
func (g *session) expand(relative string) []string {
	listing := g.must("", "ls-files", "-z", "--", ":(literal)"+relative)
	tracked := []string{}
	for _, entry := range strings.Split(listing, "\x00") {
		if entry != "" {
			tracked = append(tracked, entry)
		}
	}
	if len(tracked) == 0 && g.err == nil {
		g.err = fmt.Errorf("path to remove is not tracked: %s", relative)
	}
	return tracked
}
