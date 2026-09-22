package vcs_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/vcs"
	"github.com/xidus90/loomux/internal/gitenv"
)

// The simplest real case: a repository with one commit hands back the bytes of
// the committed file.
func TestShowBlobReturnsTheCommittedBytes(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})

	got, err := vcs.ShowBlob(repo, "a.md")

	if err != nil {
		t.Fatalf("ShowBlob: %v", err)
	}
	if string(got) != "first\n" {
		t.Fatalf("blob = %q", got)
	}
}

// `HEAD:./<path>` resolves against the working directory, which is what lets a
// caller sit anywhere below the repository root without prefix arithmetic.
func TestShowBlobReadsAPathRelativeToTheDirectory(t *testing.T) {
	repo := newRepo(t, map[string]string{"sub/a.md": "below\n"})

	got, err := vcs.ShowBlob(filepath.Join(repo, "sub"), "a.md")

	if err != nil {
		t.Fatalf("ShowBlob: %v", err)
	}
	if string(got) != "below\n" {
		t.Fatalf("blob = %q", got)
	}
}

// A path HEAD does not know is no error: there is no baseline, and the caller
// says so out loud in the package it builds.
func TestShowBlobReturnsNilForAnUnknownPath(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})

	got, err := vcs.ShowBlob(repo, "never.md")

	if err != nil {
		t.Fatalf("ShowBlob: %v", err)
	}
	if got != nil {
		t.Fatalf("blob = %q, want nil", got)
	}
}

// A vault without git is a legitimate setup, not a defect.
func TestShowBlobReturnsNilWithoutARepository(t *testing.T) {
	requireGit(t)

	got, err := vcs.ShowBlob(t.TempDir(), "a.md")

	if err != nil || got != nil {
		t.Fatalf("ShowBlob = %q, %v; want nil, nil", got, err)
	}
}

// An unborn branch has no HEAD to read, which is again "no baseline".
func TestShowBlobReturnsNilOnAnUnbornBranch(t *testing.T) {
	got, err := vcs.ShowBlob(emptyRepo(t), "a.md")

	if err != nil || got != nil {
		t.Fatalf("ShowBlob = %q, %v; want nil, nil", got, err)
	}
}

// A working directory that is gone -- a throwaway worktree that was removed --
// must not take a whole reconciliation down over one unreadable source.
func TestShowBlobReturnsNilWhenTheDirectoryIsGone(t *testing.T) {
	requireGit(t)

	got, err := vcs.ShowBlob(filepath.Join(t.TempDir(), "gone"), "a.md")

	if err != nil || got != nil {
		t.Fatalf("ShowBlob = %q, %v; want nil, nil", got, err)
	}
}

// nil is this call's word for "no baseline", so a committed empty file has to
// arrive as something else -- an empty slice that is not nil.
func TestShowBlobDistinguishesAnEmptyFileFromNoBaseline(t *testing.T) {
	repo := newRepo(t, map[string]string{"empty.md": ""})

	got, err := vcs.ShowBlob(repo, "empty.md")

	if err != nil {
		t.Fatalf("ShowBlob: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("blob = %v, want an empty non-nil slice", got)
	}
}

func TestShowBlobRefusesAPathLeavingTheRepository(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})

	_, err := vcs.ShowBlob(repo, "../outside.md")

	if !errors.Is(err, vcs.ErrOutsideRepository) {
		t.Fatalf("err = %v, want ErrOutsideRepository", err)
	}
}

// Checked on the raw elements: path.Clean would fold `a/../b` to `b` and let
// exactly the escape through that this refuses.
func TestShowBlobRefusesAPathThatClimbsInTheMiddle(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})

	_, err := vcs.ShowBlob(repo, "sub/../../outside.md")

	if !errors.Is(err, vcs.ErrOutsideRepository) {
		t.Fatalf("err = %v, want ErrOutsideRepository", err)
	}
}

func TestShowBlobRefusesAnEmptyPath(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})

	_, err := vcs.ShowBlob(repo, "")

	if !errors.Is(err, vcs.ErrOutsideRepository) {
		t.Fatalf("err = %v, want ErrOutsideRepository", err)
	}
}

// Refused here rather than by git, whose message names no repository.
func TestShowBlobRefusesAnAbsolutePath(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})

	_, err := vcs.ShowBlob(repo, "/outside.md")

	if !errors.Is(err, vcs.ErrOutsideRepository) {
		t.Fatalf("err = %v, want ErrOutsideRepository", err)
	}
}

// The whole evidence of a merge event: the names a range touched and the
// subject of each commit in it. Never `git show`, so no source text can reach
// the evidence.
func TestARangeYieldsItsPathsAndSubjects(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})
	first := head(t, repo)
	write(t, repo, "b.md", "second\n")
	add(t, repo, "b.md")
	commit(t, repo, "Build the thing")
	last := head(t, repo)

	paths, err := vcs.ChangedPaths(repo, first, last)
	if err != nil {
		t.Fatalf("ChangedPaths: %v", err)
	}
	subjects, subjectsErr := vcs.CommitSubjects(repo, first, last)
	if subjectsErr != nil {
		t.Fatalf("CommitSubjects: %v", subjectsErr)
	}

	if !reflect.DeepEqual(paths, []string{"b.md"}) {
		t.Fatalf("paths = %q", paths)
	}
	if !reflect.DeepEqual(subjects, []string{"Build the thing"}) {
		t.Fatalf("subjects = %q", subjects)
	}
}

// `%s` is git's subject, not "the first line": the message up to the first
// blank line, with its newlines folded into spaces. A caller that promises
// "only the first line" would be promising something this does not give.
func TestASubjectIsTheFirstParagraphFoldedNotTheFirstLine(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})
	first := head(t, repo)
	write(t, repo, "b.md", "second\n")
	add(t, repo, "b.md")
	commit(t, repo, "Build the thing\nand the other thing\n\nA body that stays out.")
	last := head(t, repo)

	subjects, err := vcs.CommitSubjects(repo, first, last)

	if err != nil {
		t.Fatalf("CommitSubjects: %v", err)
	}
	want := []string{"Build the thing and the other thing"}
	if !reflect.DeepEqual(subjects, want) {
		t.Fatalf("subjects = %q, want %q", subjects, want)
	}
}

// An empty range is an answer, not a failure: nothing happened between the two
// commits, and the caller writes no heading over nothing.
func TestAnEmptyRangeIsEmptyAndNotAnError(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})
	only := head(t, repo)

	paths, err := vcs.ChangedPaths(repo, only, only)

	if err != nil {
		t.Fatalf("ChangedPaths: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("paths = %q, want none", paths)
	}
}

func TestARangeFailsWhenTheDirectoryIsGone(t *testing.T) {
	requireGit(t)

	_, err := vcs.ChangedPaths(filepath.Join(t.TempDir(), "gone"), strings.Repeat("a", 40), strings.Repeat("b", 40))

	if err == nil {
		t.Fatal("ChangedPaths of a directory that is not there: want an error")
	}
}

// Commits that were garbage-collected, or never existed: git refuses the range
// and the caller keeps the event rather than inventing evidence.
//
// The message has to carry git's own words. The caller swallows this error by
// design -- an event dropped without a case is a merge nobody hears about
// again -- so a log line is all anyone will ever see of it, and "exit status
// 128" names nothing at all.
func TestARangeFailsWhenGitDoesNotKnowItAndSaysWhatGitSaid(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})

	_, err := vcs.CommitSubjects(repo, strings.Repeat("0", 40), strings.Repeat("1", 40))

	if err == nil {
		t.Fatal("CommitSubjects of an unknown range: want an error")
	}
	if !strings.Contains(err.Error(), "fatal:") {
		t.Fatalf("err = %q, want git's own reason in it", err)
	}
}

// The range comes off a file a hook appends to and a hand may edit, and git
// reads a leading `-` as an option: `--output=<file>..HEAD` is one word that
// makes `git diff` write to a file. Only a hex object name gets through.
func TestARangeMustBeAPairOfObjectNames(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})
	good := head(t, repo)

	cases := []struct {
		name        string
		first, last string
	}{
		{"an option in the first end", "--output=" + filepath.Join(t.TempDir(), "x"), good},
		{"an option in the last end", good, "-n1"},
		{"a symbolic name", "HEAD", good},
		{"too short", "abc", good},
		{"too long", strings.Repeat("a", 65), good},
		{"upper case", strings.ToUpper(good), good},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := vcs.ChangedPaths(repo, testCase.first, testCase.last); !errors.Is(err, vcs.ErrNotAnObjectName) {
				t.Fatalf("ChangedPaths err = %v, want ErrNotAnObjectName", err)
			}
			if _, err := vcs.CommitSubjects(repo, testCase.first, testCase.last); !errors.Is(err, vcs.ErrNotAnObjectName) {
				t.Fatalf("CommitSubjects err = %v, want ErrNotAnObjectName", err)
			}
		})
	}
}

// A directory that belongs to no repository is a legitimate setup, not a
// failure: the empty string is the answer, and the caller skips the area.
func TestRepositoryRootOutsideARepo(t *testing.T) {
	requireGit(t)

	root, err := vcs.RepositoryRoot(t.TempDir())

	if err != nil {
		t.Fatalf("RepositoryRoot: %v", err)
	}
	if root != "" {
		t.Fatalf("root = %q, want \"\"", root)
	}
}

// Derived from the path rather than declared: a subdirectory answers with the
// work tree above it, so nothing has to keep the fact a second time.
func TestRepositoryRootOfASubdirectory(t *testing.T) {
	repo := newRepo(t, map[string]string{"sub/a.md": "below\n"})

	below, err := vcs.RepositoryRoot(filepath.Join(repo, "sub"))
	if err != nil {
		t.Fatalf("RepositoryRoot: %v", err)
	}
	// Against git's own answer for the root, not against the temporary
	// directory's name: on Windows that name can come back shortened or in
	// another case, and the comparison would be about the spelling.
	above, aboveErr := vcs.RepositoryRoot(repo)
	if aboveErr != nil {
		t.Fatalf("RepositoryRoot: %v", aboveErr)
	}

	if below != above || below == "" {
		t.Fatalf("root below = %q, above = %q", below, above)
	}
}

func TestRepositoryRootFailsWhenTheDirectoryIsGone(t *testing.T) {
	requireGit(t)

	_, err := vcs.RepositoryRoot(filepath.Join(t.TempDir(), "gone"))

	if err == nil {
		t.Fatal("RepositoryRoot of a directory that is not there: want an error")
	}
}

func TestCommonDirectoryOutsideARepo(t *testing.T) {
	requireGit(t)

	common, err := vcs.CommonDirectory(t.TempDir())

	if err != nil {
		t.Fatalf("CommonDirectory: %v", err)
	}
	if common != "" {
		t.Fatalf("common = %q, want \"\"", common)
	}
}

// This, and not the work tree, is a repository's identity: every linked
// worktree has a top level of its own but one common directory, so only this
// value tells "my repository" from "a foreign repository".
func TestCommonDirectoryIsSharedByALinkedWorktree(t *testing.T) {
	repo := newRepo(t, map[string]string{"a.md": "first\n"})
	linked := filepath.Join(t.TempDir(), "linked")
	git(t, repo, "worktree", "add", "-q", "-b", "other", linked)

	common, err := vcs.CommonDirectory(linked)
	if err != nil {
		t.Fatalf("CommonDirectory: %v", err)
	}
	shared, sharedErr := vcs.CommonDirectory(repo)
	if sharedErr != nil {
		t.Fatalf("CommonDirectory: %v", sharedErr)
	}
	root, rootErr := vcs.RepositoryRoot(linked)
	if rootErr != nil {
		t.Fatalf("RepositoryRoot: %v", rootErr)
	}
	above, aboveErr := vcs.RepositoryRoot(repo)
	if aboveErr != nil {
		t.Fatalf("RepositoryRoot: %v", aboveErr)
	}

	if common == "" || common != shared {
		t.Fatalf("common of the linked worktree = %q, of the repository = %q", common, shared)
	}
	if root == above {
		t.Fatalf("work tree of the linked worktree = %q, same as the repository's", root)
	}
}

func TestCommonDirectoryFailsWhenTheDirectoryIsGone(t *testing.T) {
	requireGit(t)

	_, err := vcs.CommonDirectory(filepath.Join(t.TempDir(), "gone"))

	if err == nil {
		t.Fatal("CommonDirectory of a directory that is not there: want an error")
	}
}

// newRepo lays down a repository with one commit. Real git, no stub: the parity
// cases of this stage run against real git too, and a stub would hide exactly
// the differences the parity proof looks for.
//
// Written out here rather than borrowed: internal/gitwork's helper is package
// private, and Go cannot import another package's test code. The maintenance
// package gets one of its own for the same reason.
func newRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := emptyRepo(t)
	for name, content := range files {
		write(t, root, name, content)
	}
	add(t, root, "--all")
	commit(t, root, "first")
	return root
}

func emptyRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	git(t, root, "init", "-q")
	return root
}

// requireGit stands in front of every case, not only the ones that build a
// fixture. Without git on the PATH the spawn fails, and a failed spawn looks
// exactly like the condition several cases check: "no baseline", "the directory
// is gone", "the range cannot be read". Those would pass for the wrong reason
// on a machine without git, and the two that expect "" for a directory outside
// a repository would fail loudly -- a suite that is half wrong in each
// direction. The skip is the only honest answer.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func add(t *testing.T, root string, paths ...string) {
	t.Helper()
	git(t, root, append([]string{"add"}, paths...)...)
}

// The identity travels as `-c` on the call rather than through `git config`, so
// nothing about it is left behind in the fixture, and `commit.gpgsign=false`
// keeps a machine whose user signs by default from asking for a key.
func commit(t *testing.T, root, message string) {
	t.Helper()
	git(t, root,
		"-c", "user.name=Test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"commit", "-q", "-m", message)
}

func head(t *testing.T, root string) string {
	t.Helper()
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = root
	command.Env = gitenv.Environ()
	out, err := command.Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	// The same strip the package under test uses, and for a measured reason:
	// this project's `.githooks/pre-commit` runs `go test ./...` itself, and
	// git exports GIT_DIR and GIT_INDEX_FILE to that hook. Both outrank
	// command.Dir, so unstripped these fixture calls would build their
	// repository in, add to and commit into the repository being committed --
	// see the comment on gitenv.Location.
	command.Env = gitenv.Environ()
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
