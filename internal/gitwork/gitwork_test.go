package gitwork

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/gitenv"
)

func TestHeadCommitOfARepository(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")

	got, err := HeadCommit(root)
	if err != nil {
		t.Fatal(err)
	}

	// The full SHA and not --short: the answer travels in a run marker and is
	// read back rounds later, and an abbreviation is only unique for as long
	// as the repository stays the size it was.
	if len(got) != 40 {
		t.Fatalf("expected a full 40-character sha, got %q", got)
	}
	if got != revParse(t, root) {
		t.Fatalf("HeadCommit = %q, git says %q", got, revParse(t, root))
	}
}

// A detached HEAD is no special case: what a run records is a commit, not a
// branch, and rev-parse answers the same either way.
func TestHeadCommitReadsADetachedHead(t *testing.T) {
	root := repo(t)
	commit(t, root, "first")
	sha, err := HeadCommit(root)
	if err != nil {
		t.Fatal(err)
	}
	run(t, root, "checkout", "-q", "--detach", sha)

	got, err := HeadCommit(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != sha {
		t.Fatalf("HeadCommit = %q after detaching, want %q", got, sha)
	}
}

// git ran and refused, so its own words on stderr are the only account of
// why -- and they arrive behind a separator. Written straight onto the exit
// status they would read as one word with it, which is the other half of the
// rule the dangling separator below pins.
var runsIntoTheExitStatus = regexp.MustCompile(`exit status \d+[^\d:]`)

func TestHeadCommitOutsideARepository(t *testing.T) {
	_, err := HeadCommit(t.TempDir())

	if err == nil {
		t.Fatal("a directory that is not a repository has no head")
	}
	if runsIntoTheExitStatus.MatchString(err.Error()) {
		t.Fatalf("git's own words run into the exit status: %q", err.Error())
	}
}

// From a subdirectory the top level is the repository's, without the line
// end git prints; outside any repository there is none.
func TestTopLevelOfASubdirectory(t *testing.T) {
	root := repo(t)
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := TopLevel(sub)
	if err != nil {
		t.Fatal(err)
	}
	gotInfo, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	rootInfo, _ := os.Stat(root)
	if !os.SameFile(gotInfo, rootInfo) {
		t.Fatalf("TopLevel = %q, want %q", got, root)
	}
	if top, err := TopLevel(t.TempDir()); err == nil || top != "" {
		t.Fatalf("outside a repository: %q, %v", top, err)
	}
	if prefix, err := Prefix(sub); err != nil || prefix != "a/b/" {
		t.Fatalf("Prefix = %q, %v", prefix, err)
	}
	if prefix, err := Prefix(root); err != nil || prefix != "" {
		t.Fatalf("Prefix at the top = %q, %v", prefix, err)
	}
	if prefix, err := Prefix(t.TempDir()); err == nil || prefix != "" {
		t.Fatalf("Prefix outside a repository: %q, %v", prefix, err)
	}
}

// A root that is not there never reaches an exit code: the spawn itself fails.
// Same answer as a non-zero one -- an error, never an empty string a caller
// could mistake for a commit.
//
// And the message ends where it stops having something to say. git never ran,
// so there is nothing on stderr, and the account of the failure is the one the
// OS gave through `err`. Appending an empty stderr behind a separator would
// end every such message in a dangling ": ", which reads as an error that was
// cut off rather than one that is complete.
func TestHeadCommitOfADirectoryThatIsNotThere(t *testing.T) {
	_, err := HeadCommit(filepath.Join(t.TempDir(), "nowhere"))

	if err == nil {
		t.Fatal("a directory that is not there has no head")
	}
	if strings.HasSuffix(err.Error(), ": ") {
		t.Fatalf("the message ends in a dangling separator: %q", err.Error())
	}
}

// `git init` leaves HEAD naming a branch that does not exist yet. That is a
// repository without an answer, not an answer.
func TestHeadCommitOfARepositoryWithoutACommit(t *testing.T) {
	if _, err := HeadCommit(repo(t)); err == nil {
		t.Fatal("a repository with no commit has no head")
	}
}

// The one refusal that is not git's own. An ignored directory *is* inside a
// repository, so rev-parse would answer readily -- with the surrounding
// repository's HEAD -- and this is the case where a successful call is the
// wrong one: measuring against that outer HEAD is worse than not measuring,
// because every file of the parked copy then reads as somebody's change. So
// what the refusal buys is an error where there would otherwise be an answer,
// and that, not its position among two calls, is what this pins.
func TestHeadCommitRefusesAnIgnoredRoot(t *testing.T) {
	outer := repo(t)
	commit(t, outer, "first")
	parked := filepath.Join(outer, "parked")
	if err := os.MkdirAll(parked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outer, ".gitignore"), []byte("parked/\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := HeadCommit(parked)

	if !errors.Is(err, ErrIgnoredRoot) {
		t.Fatalf("expected ErrIgnoredRoot, got %v", err)
	}
}

func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	run(t, root, "init")
	run(t, root, "config", "user.email", "t@example.invalid")
	run(t, root, "config", "user.name", "Test")
	// Pinned locally: under a global status.renames=false, git status reports a
	// rename as an addition and a deletion, and the rename tests would pass
	// without ever reaching parseStatus's rename branch.
	run(t, root, "config", "status.renames", "true")
	return root
}

func commit(t *testing.T, root, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(message), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, root, "add", "a.txt")
	run(t, root, "commit", "-m", message)
}

func revParse(t *testing.T, root string) string {
	t.Helper()
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = root
	command.Env = gitenv.Environ()
	out, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	// Trimmed the way HeadCommit trims, so the comparison is about the SHA and
	// not about how many bytes of line ending git happened to write.
	return strings.TrimSpace(string(out))
}

func run(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	// The same strip the package under test uses, and for a measured reason
	// rather than a hypothetical one: this project's `.githooks/pre-commit`
	// runs `go test ./...` itself, and git exports GIT_DIR and GIT_INDEX_FILE
	// to that hook. Both outrank command.Dir, so
	// unstripped these fixture calls would build their repository in, add to
	// and commit into the repository being committed -- see the comment on
	// gitenv.Location, where exactly that was measured on 2026-09-07.
	command.Env = gitenv.Environ()
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// A repository without a commit is a state to measure from -- the empty tree
// -- and not a failure, which is the whole difference between Head and
// HeadCommit above.
func TestHeadOfAnUnbornRepository(t *testing.T) {
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	if head, err := Head(root); err != nil || head != "" {
		t.Fatalf("head %q, %v", head, err)
	}
}

// A HEAD that is broken is not the unborn state: the empty tree is a base to
// measure from only where there is no commit, and a HEAD naming a missing
// object or a blob would hide every change behind it.
func TestHeadOfABrokenHeadIsAnError(t *testing.T) {
	for _, tc := range []struct {
		name string
		blob bool // else an id no object has
		// detached: HEAD itself holds the id; else the branch ref does, and
		// HEAD stays a name.
		detached bool
	}{
		{"a detached head on a missing object", false, true},
		{"a detached head on a blob", true, true},
		{"a branch on a missing object", false, false},
		{"a branch on a blob", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := repoWithCommit(t)
			id := "1111111111111111111111111111111111111111"
			if tc.blob {
				command := exec.Command("git", "hash-object", "-w", "--stdin")
				command.Dir = root
				command.Env = gitenv.Environ()
				command.Stdin = strings.NewReader("")
				out, err := command.Output()
				if err != nil {
					t.Fatal(err)
				}
				id = strings.TrimSpace(string(out))
			}
			name := "HEAD"
			if !tc.detached {
				name = mustGit(t, root, "symbolic-ref", "HEAD")
			}
			file := mustGit(t, root, "rev-parse", "--path-format=absolute", "--git-path", name)
			if err := os.WriteFile(file, []byte(id+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			head, err := Head(root)
			if err == nil {
				t.Fatalf("head %q, no error", head)
			}
		})
	}
}

// A branch ref git cannot read is a failure too, not a repository without a
// commit: only a HEAD that names a branch which does not exist yet is unborn.
func TestHeadOfAnUnreadableBranchIsAnError(t *testing.T) {
	root := repoWithCommit(t)
	branch := mustGit(t, root, "symbolic-ref", "HEAD")
	ref := mustGit(t, root, "rev-parse", "--path-format=absolute", "--git-path", branch)
	if err := os.WriteFile(ref, []byte("garbage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Loose ref files are the files backend's; under another one the write
	// above changed nothing and the test could not tell.
	probe := exec.Command("git", "rev-parse", "-q", "--verify", "HEAD^{commit}")
	probe.Dir = root
	probe.Env = gitenv.Environ()
	if probe.Run() == nil {
		t.Skip("the ref backend does not read loose ref files")
	}
	head, err := Head(root)
	if err == nil {
		t.Fatalf("head %q, no error", head)
	}
}

func TestHeadOfARepositoryWithACommit(t *testing.T) {
	root := repoWithCommit(t)
	head, err := Head(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := mustGit(t, root, "rev-parse", "HEAD"); head != want {
		t.Fatalf("head %q, want %q", head, want)
	}
}

func TestHeadOutsideARepository(t *testing.T) {
	if _, err := Head(t.TempDir()); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("err %v", err)
	}
}

// The same refusal HeadCommit makes, and for the same reason: rev-parse would
// answer with the surrounding repository's HEAD.
func TestHeadRefusesAnIgnoredRoot(t *testing.T) {
	outer := repo(t)
	commit(t, outer, "first")
	parked := filepath.Join(outer, "parked")
	if err := os.MkdirAll(parked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outer, ".gitignore"), []byte("parked/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Head(parked); !errors.Is(err, ErrIgnoredRoot) {
		t.Fatalf("err %v", err)
	}
}

func TestTreeOfRefusesACommitThatIsNotThere(t *testing.T) {
	if _, err := TreeOf(repoWithCommit(t), "nosuchref"); err == nil {
		t.Fatal("want an error")
	}
}

func TestContentTreeMatchesHeadWhenClean(t *testing.T) {
	root := repoWithCommit(t)
	tree, err := ContentTree(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	head, _ := TreeOf(root, "HEAD")
	if tree != head {
		t.Fatalf("tree %s, head tree %s", tree, head)
	}
}

func TestContentTreeSeesChangesAndNewFilesButNotState(t *testing.T) {
	root := repoWithCommit(t)
	clean, _ := ContentTree(root, t.TempDir())
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "state", "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if tree, _ := ContentTree(root, t.TempDir()); tree != clean {
		t.Fatal("machine state moved the tree")
	}
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if tree, _ := ContentTree(root, t.TempDir()); tree == clean {
		t.Fatal("a new file did not move the tree")
	}
	// The real index is untouched.
	if out := mustGit(t, root, "diff", "--cached", "--name-only"); out != "" {
		t.Fatalf("staged: %q", out)
	}
}

func TestContentTreeOfAnUnbornRepository(t *testing.T) {
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	if tree, err := ContentTree(root, t.TempDir()); err != nil || tree != EmptyTree {
		t.Fatalf("tree %s, %v", tree, err)
	}
}

// The first call is the one that refuses, and the message says so. Carried
// on instead, the empty answer would read as a relative path and send the
// copy looking for an index under the root, where the failure it ran into
// would be the wrong one to report.
func TestContentTreeOutsideARepository(t *testing.T) {
	_, err := ContentTree(t.TempDir(), t.TempDir())

	if err == nil {
		t.Fatal("want an error")
	}
	if !strings.Contains(err.Error(), "--git-path") {
		t.Fatalf("the message does not name the call that refused: %q", err.Error())
	}
}

// A linked worktree is where `rev-parse --git-path` answers with an absolute
// path, because the index lives under the main repository's
// `.git/worktrees/<name>/`. Joined to the root all the same, that path is no
// path at all and the copy fails.
func TestContentTreeInALinkedWorktree(t *testing.T) {
	main := repoWithCommit(t)
	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, main, "worktree", "add", "-q", linked)

	tree, err := ContentTree(linked, t.TempDir())

	if err != nil {
		t.Fatalf("a linked worktree has a content tree: %v", err)
	}
	if want := mustGit(t, linked, "rev-parse", "HEAD^{tree}"); tree != want {
		t.Fatalf("ContentTree = %q, want %q", tree, want)
	}
}

// The three ways the copy of the index can fail, each made by putting the
// wrong kind of thing where the call expects its own: a file where the
// scratch directory goes, a directory where the index is read, and an index
// of bytes git cannot read.
func TestContentTreeReportsWhatItCannotCopy(t *testing.T) {
	t.Run("scratch is a file", func(t *testing.T) {
		root := repoWithCommit(t)
		scratch := filepath.Join(t.TempDir(), "scratch")
		if err := os.WriteFile(scratch, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := ContentTree(root, scratch)
		if err == nil {
			t.Fatal("want an error")
		}
		// The scratch directory is reported where it fails, not two steps
		// later through the copy that could not be written into it.
		if strings.Contains(err.Error(), "index-") {
			t.Fatalf("the message blames the copy, not the directory: %q", err.Error())
		}
	})
	t.Run("the index is a directory", func(t *testing.T) {
		root := t.TempDir()
		mustGit(t, root, "init", "-q")
		if err := os.MkdirAll(filepath.Join(root, ".git", "index"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := ContentTree(root, t.TempDir()); err == nil {
			t.Fatal("want an error")
		}
	})
	t.Run("the index is not an index", func(t *testing.T) {
		root := repoWithCommit(t)
		if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("not an index"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ContentTree(root, t.TempDir()); err == nil {
			t.Fatal("want an error")
		}
	})
}

// A name of the copy that belongs to one call: a lock git left on the old
// per-process name, or a copy another call is still using, is not in its way.
func TestContentTreeKeepsClearOfAnotherCallsCopy(t *testing.T) {
	root := repoWithCommit(t)
	scratch := t.TempDir()
	stale := filepath.Join(scratch, fmt.Sprintf("index-%d.lock", os.Getpid()))
	if err := os.WriteFile(stale, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	tree, err := ContentTree(root, scratch)

	if err != nil {
		t.Fatalf("ContentTree: %v", err)
	}
	if want := mustGit(t, root, "rev-parse", "HEAD^{tree}"); tree != want {
		t.Fatalf("ContentTree = %q, want %q", tree, want)
	}
	left, _ := os.ReadDir(scratch)
	if len(left) != 1 {
		t.Fatalf("scratch holds %d entries after the call, want only the stale file", len(left))
	}
}

func TestContentTreeReportsAPrivateDirectoryItCannotMake(t *testing.T) {
	root := repoWithCommit(t)
	want := errors.New("no room")
	old := mkdirTemp
	mkdirTemp = func(string, string) (string, error) { return "", want }
	defer func() { mkdirTemp = old }()

	if _, err := ContentTree(root, t.TempDir()); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestKeptContentTreeReportsACopyItCannotWrite(t *testing.T) {
	root := repoWithCommit(t)
	copied := filepath.Join(root, ".git", fmt.Sprintf("%s%d", KeptIndexPrefix, os.Getpid()))
	if err := os.MkdirAll(copied, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := KeptContentTree(root); err == nil {
		t.Fatal("want an error")
	}
}

func TestKeptContentTreeLiesInTheGitDir(t *testing.T) {
	root := repoWithCommit(t)
	if err := os.WriteFile(filepath.Join(root, "new.go"), []byte("package x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".loomux", "state", "x"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tree, index, err := KeptContentTree(root)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(index)

	sameDir(t, filepath.Dir(index), mustGit(t, root, "rev-parse", "--absolute-git-dir"))
	// The name the leak checks look for: a check that finds nothing proves
	// nothing unless it would find the copy while it is there.
	if left := keptCopies(t, root); len(left) != 1 || !strings.HasPrefix(filepath.Base(index), KeptIndexPrefix) {
		t.Fatalf("copy %s, found %v", index, left)
	}
	files := strings.Fields(mustGitWith(t, root, []string{"GIT_INDEX_FILE=" + index}, "ls-files"))
	if !slices.Contains(files, "new.go") || slices.Contains(files, ".loomux/state/x") {
		t.Fatalf("the kept index holds %v", files)
	}
	want, err := ContentTree(root, t.TempDir())
	if err != nil || tree != want {
		t.Fatalf("tree %s, ContentTree %s, %v", tree, want, err)
	}
}

func TestKeptContentTreeInALinkedWorktree(t *testing.T) {
	main := repoWithCommit(t)
	linked := filepath.Join(t.TempDir(), "wt")
	mustGit(t, main, "worktree", "add", "-q", linked)

	tree, index, err := KeptContentTree(linked)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(index)

	sameDir(t, filepath.Dir(index), filepath.Join(main, ".git", "worktrees", "wt"))
	if want := mustGit(t, linked, "rev-parse", "HEAD^{tree}"); tree != want {
		t.Fatalf("tree %s, want %s", tree, want)
	}
}

func TestKeptContentTreeLeavesNothingOnFailure(t *testing.T) {
	if _, index, err := KeptContentTree(t.TempDir()); err == nil || index != "" {
		t.Fatalf("index %q, %v", index, err)
	}
	// Failing after the copy was made: an index git cannot read.
	root := repoWithCommit(t)
	if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("not an index"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, index, err := KeptContentTree(root)
	if err == nil || index != "" {
		t.Fatalf("index %q, %v", index, err)
	}
	if left := keptCopies(t, root); len(left) != 0 {
		t.Fatalf("left behind: %v", left)
	}
}

// A process killed during the chain never removes its copy. The next one
// drops every copy whose process is gone, and leaves a running one's alone:
// two sessions in one repository share its git directory.
func TestKeptContentTreeDropsTheCopiesOfGoneProcesses(t *testing.T) {
	root := repoWithCommit(t)
	dir := mustGit(t, root, "rev-parse", "--absolute-git-dir")
	gone := filepath.Join(dir, KeptIndexPrefix+goneProcess(t))
	live := filepath.Join(dir, KeptIndexPrefix+strconv.Itoa(os.Getppid()))
	odd := filepath.Join(dir, KeptIndexPrefix+"x")
	for _, p := range []string{gone, live, odd} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, index, err := KeptContentTree(root)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(index)
	if _, err := os.Stat(gone); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the copy of a gone process stayed: %v", err)
	}
	for _, p := range []string{live, odd} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s was dropped: %v", p, err)
		}
	}
}

// keptCopies lists the kept copies of the index in the git directory of root.
func keptCopies(t *testing.T, root string) []string {
	t.Helper()
	entries, err := os.ReadDir(mustGit(t, root, "rev-parse", "--absolute-git-dir"))
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), KeptIndexPrefix) {
			found = append(found, e.Name())
		}
	}
	return found
}

// goneProcess is the number of a process that ran and was reaped: the test
// binary itself, told to run no test.
func goneProcess(t *testing.T) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return strconv.Itoa(cmd.Process.Pid)
}

// sameDir compares by identity: git and t.TempDir may spell one directory
// differently on Windows (8.3 names, case).
func sameDir(t *testing.T, got, want string) {
	t.Helper()
	a, err := os.Stat(got)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.Stat(want)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(a, b) {
		t.Fatalf("%s is not %s", got, want)
	}
}

func TestLocalHeadsAndLog(t *testing.T) {
	root := repoWithCommit(t)
	first := mustGit(t, root, "rev-parse", "HEAD")
	mustGit(t, root, "branch", "feature")
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", "b.txt")
	mustGit(t, root, "commit", "-q", "-m", "second")
	heads, elsewhere, err := LocalBranches(root)
	if err != nil || heads["refs/heads/feature"] != first || len(heads) != 2 || elsewhere != nil {
		t.Fatalf("heads %v, elsewhere %v, %v", heads, elsewhere, err)
	}
	log, err := LogOneline(root, first, "HEAD")
	if err != nil || len(log) != 1 || !strings.HasSuffix(log[0], " second") {
		t.Fatalf("log %v, %v", log, err)
	}
}

// A branch checked out in another worktree is named, whichever worktree
// asks; root's own branch and a branch checked out nowhere are not. Asked
// from the other worktree -- a path git spells with forward slashes, the
// test with its own -- the roles swap.
func TestLocalBranchesNamesTheBranchesOfOtherWorktrees(t *testing.T) {
	root := repoWithCommit(t)
	mustGit(t, root, "branch", "free")
	other := filepath.Join(t.TempDir(), "other")
	mustGit(t, root, "worktree", "add", "-q", "-b", "wt", other)

	_, elsewhere, err := LocalBranches(root)
	if err != nil || !slices.Equal(elsewhere, []string{"refs/heads/wt"}) {
		t.Fatalf("from root: %v, %v", elsewhere, err)
	}
	own := mustGit(t, root, "symbolic-ref", "HEAD")
	_, elsewhere, err = LocalBranches(other)
	if err != nil || !slices.Equal(elsewhere, []string{own}) {
		t.Fatalf("from the other worktree: %v, %v", elsewhere, err)
	}
}

// A bare repository has branches but no worktree of its own to compare them
// against.
func TestLocalBranchesInABareRepository(t *testing.T) {
	root := t.TempDir()
	mustGit(t, root, "init", "-q", "--bare")
	if _, _, err := LocalBranches(root); err == nil {
		t.Fatal("want an error")
	}
}

func TestSamePathFoldsCaseOnlyOnWindows(t *testing.T) {
	// One separator style on both sides: filepath.Clean folds separators the
	// running system's way, and goos decides only the case.
	if !samePath("windows", "C:/Repo/#GIT/x", "c:/repo/#git/x/") {
		t.Fatal("windows: the same directory spelled twice")
	}
	if samePath("linux", "/repo/X", "/repo/x") {
		t.Fatal("linux: two directories")
	}
	if !samePath("linux", "/repo/x/", "/repo/x") {
		t.Fatal("linux: the same directory spelled twice")
	}
}

func TestLocalHeadsAndLogOutsideARepository(t *testing.T) {
	if _, _, err := LocalBranches(t.TempDir()); err == nil {
		t.Fatal("want an error")
	}
	if _, err := LogOneline(t.TempDir(), "a", "b"); err == nil {
		t.Fatal("want an error")
	}
}

// mustGit runs git in root and answers with its stdout, trimmed. The identity
// travels on the command line so that every call can commit, whatever the
// repository's own configuration says, and the two config variables keep the
// machine's settings out: a fixture that read them would build a different
// repository on every developer's box.
func mustGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	return mustGitWith(t, root, nil, args...)
}

// mustGitWith is mustGit with extra KEY=VALUE entries in git's environment.
func mustGitWith(t *testing.T, root string, env []string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	command.Dir = root
	command.Env = append(append(gitenv.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull), env...)
	// stdout alone: git's warnings go to stderr, and folded in they would
	// travel on as part of a SHA or of an empty answer.
	out, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// repoWithCommit is a repository with one commit over one file. The content
// carries no line ending on purpose: ContentTree measures under the user's
// own git configuration, and a text file would be rewritten by an autocrlf
// this fixture does not set.
func repoWithCommit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, root, "add", "a.txt")
	mustGit(t, root, "commit", "-q", "-m", "first")
	return root
}

// BenchmarkContentTree measures the fingerprint the stop gate takes at every
// turn end, on this repository rather than on a fixture: the cost is the
// working tree's size, and only the real one has it. A repository is the one
// thing this benchmark needs, so a checkout without one skips instead of
// reporting a number that is a git failure.
func BenchmarkContentTree(b *testing.B) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		b.Fatal(err)
	}
	if _, err := ContentTree(root, b.TempDir()); err != nil {
		b.Skipf("no repository to measure: %v", err)
	}
	scratch := b.TempDir()
	for b.Loop() {
		if _, err := ContentTree(root, scratch); err != nil {
			b.Fatal(err)
		}
	}
}

// A file rewritten to the same size within the index's own timestamp is one
// git calls racy: its stat matches the entry, and only the index's mtime
// tells git to hash it again. The copy must keep that mtime, or the change
// reads as no change -- the stop gate then takes a new edit for the old tree.
func TestContentTreeSeesARacyChange(t *testing.T) {
	root := repoWithCommit(t)
	head := mustGit(t, root, "rev-parse", "HEAD^{tree}")
	stamp := time.Now().Add(-time.Hour).Truncate(time.Second)
	file := filepath.Join(root, "a.txt")
	if err := os.Chtimes(file, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	// Refresh the entry to that stamp, then give the index the same one.
	mustGit(t, root, "update-index", "--refresh")
	index := filepath.Join(root, ".git", "index")
	if err := os.Chtimes(index, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(file, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	tree, err := ContentTree(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if tree == head {
		t.Fatal("a same-size change within the index's second reads as no change")
	}
}

func TestIgnoredPathAsksGitAboutOnePath(t *testing.T) {
	root := repo(t)
	os.MkdirAll(filepath.Join(root, ".loomux", "state"), 0o755)
	os.WriteFile(filepath.Join(root, ".loomux", "armed.toml"), []byte("armed = []\n"), 0o644)
	if IgnoredPath(root, ".loomux/armed.toml") {
		t.Fatal("nothing is ignored yet")
	}
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/.loomux/state/\n"), 0o644)
	if IgnoredPath(root, ".loomux/armed.toml") || !IgnoredPath(root, ".loomux/state/x") {
		t.Fatal("only the state directory is ignored")
	}
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".loomux/\n"), 0o644)
	if !IgnoredPath(root, ".loomux/armed.toml") {
		t.Fatal("an ignored folder takes the file with it")
	}
	// A file git already holds is not ignored, whatever .gitignore says: it
	// reaches every commit.
	run(t, root, "add", "-f", ".loomux/armed.toml")
	run(t, root, "commit", "-m", "hold the file")
	if IgnoredPath(root, ".loomux/armed.toml") {
		t.Fatal("a tracked file counts as ignored")
	}
	if IgnoredPath(t.TempDir(), ".loomux/armed.toml") {
		t.Fatal("outside a repository a path counts as ignored")
	}
}

// The hook directory is git's answer, never a join of root and ".git/hooks":
// core.hooksPath moves it, a relative one counts from the worktree's top, and
// a linked worktree runs the hooks of the repository it belongs to.
func TestHooksDirIsWhereGitRunsHooksFrom(t *testing.T) {
	hooksDir := func(root string) string {
		t.Helper()
		dir, err := HooksDir(root)
		if err != nil || !filepath.IsAbs(dir) {
			t.Fatalf("HooksDir(%s) = %q, %v; want an absolute path", root, dir, err)
		}
		// Stat needs the directory; git names it whether it is there or not.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	main := repoWithCommit(t)
	sameDir(t, hooksDir(main), filepath.Join(main, ".git", "hooks"))

	linked := filepath.Join(t.TempDir(), "linked")
	mustGit(t, main, "worktree", "add", "-q", linked)
	sameDir(t, hooksDir(linked), filepath.Join(main, ".git", "hooks"))

	run(t, main, "config", "core.hooksPath", ".githooks")
	sameDir(t, hooksDir(main), filepath.Join(main, ".githooks"))
	// Relative to the top of the worktree git runs the hook in, not to the
	// repository that holds the setting.
	sameDir(t, hooksDir(linked), filepath.Join(linked, ".githooks"))

	elsewhere := t.TempDir()
	run(t, main, "config", "core.hooksPath", filepath.ToSlash(elsewhere))
	sameDir(t, hooksDir(main), elsewhere)

	if _, err := HooksDir(t.TempDir()); err == nil {
		t.Fatal("a directory that is no repository went through")
	}
}
