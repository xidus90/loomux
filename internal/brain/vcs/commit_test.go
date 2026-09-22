package vcs_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/vcs"
	"github.com/xidus90/loomux/internal/gitenv"
)

// test_vcs.py:118 test_a_commit_touches_only_the_named_paths
func TestCommitPathsTouchesOnlyTheNamedPaths(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt", "fremd.md": "unberührt"})
	write(t, repo, "seite.md", "neu")
	write(t, repo, "fremd.md", "vom Nutzer geändert")

	got, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}
	if got == nil || !got.Created || got.Head != head(t, repo) {
		t.Fatalf("commit = %+v, HEAD = %s", got, head(t, repo))
	}
	if paths := changed(t, repo); !reflect.DeepEqual(paths, []string{"seite.md"}) {
		t.Fatalf("paths of the commit = %q", paths)
	}
}

// test_vcs.py:129 test_the_user_index_is_untouched
func TestCommitPathsLeavesTheUserIndexAlone(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	index := filepath.Join(repo, ".git", "index")
	before := readFile(t, index)
	write(t, repo, "seite.md", "neu")

	if _, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir()); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	if !bytes.Equal(readFile(t, index), before) {
		t.Fatal("the user's index changed")
	}
}

// test_vcs.py:137 test_a_deletion_is_part_of_the_commit
func TestCommitPathsCommitsADeletion(t *testing.T) {
	repo := newVault(t, map[string]string{"fall/case.toml": "x", "seite.md": "alt"})
	if err := os.Remove(filepath.Join(repo, "fall", "case.toml")); err != nil {
		t.Fatal(err)
	}

	if _, err := vcs.CommitPaths(repo, "Decide", nil, []string{"fall/case.toml"}, t.TempDir()); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	if status := output(t, repo, "show", "--name-status", "--format=", "HEAD"); !strings.HasPrefix(status, "D") {
		t.Fatalf("name-status = %q", status)
	}
}

// test_vcs.py:144 test_a_removal_of_an_untracked_path_is_refused
func TestCommitPathsRefusesToRemoveAnUntrackedPath(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	before := head(t, repo)
	write(t, repo, "seite.md", "neu")

	_, err := vcs.CommitPaths(repo, "Decide", []string{"seite.md"}, []string{"fall/gibtsnicht.toml"}, t.TempDir())

	if err == nil || !strings.Contains(err.Error(), "path to remove is not tracked: fall/gibtsnicht.toml") {
		t.Fatalf("err = %v", err)
	}
	if head(t, repo) != before {
		t.Fatal("a commit landed without the deletion")
	}
}

// test_vcs.py:164 test_a_path_in_both_add_and_remove_is_refused
func TestCommitPathsRefusesAPathBothAddedAndRemoved(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md", "b.md"}, []string{"b.md", "seite.md"}, t.TempDir())

	if err == nil || err.Error() != "path is both added and removed: b.md, seite.md" {
		t.Fatalf("err = %v", err)
	}
}

// test_vcs.py:173 test_the_commit_message_is_the_one_given -- byte for byte:
// the message goes to commit-tree as it is, with no line ending rewritten.
func TestCommitPathsKeepsTheMessageAsGiven(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	write(t, repo, "seite.md", "neu")

	if _, err := vcs.CommitPaths(repo, "Land a review case\n\nBody line.", []string{"seite.md"}, nil, t.TempDir()); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	object := output(t, repo, "cat-file", "commit", "HEAD")
	_, message, _ := strings.Cut(object, "\n\n")
	if message != "Land a review case\n\nBody line." {
		t.Fatalf("message = %q", message)
	}
}

// test_vcs.py:186 test_a_markdown_blob_is_stored_with_lf
func TestCommitPathsHonoursGitattributes(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt", ".gitattributes": "*.md text eol=lf\n"})
	write(t, repo, "seite.md", "eins\r\nzwei\r\n")

	if _, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir()); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	if blob := output(t, repo, "cat-file", "blob", "HEAD:seite.md"); blob != "eins\nzwei\n" {
		t.Fatalf("blob = %q", blob)
	}
}

// test_vcs.py:194 test_a_moved_ref_is_refused_not_overwritten
func TestCommitPathsRefusesAMovedRef(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	before := head(t, repo)
	write(t, repo, "seite.md", "neu")
	inWindow(t, func() { git(t, repo, "commit", "-q", "--allow-empty", "-m", "foreign") })

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	// The message names the commit the swap expected, not "nothing": that
	// spelling belongs to an unborn branch only.
	if !errors.Is(err, vcs.ErrRefMoved) || !strings.HasSuffix(err.Error(), " moved away from "+before) {
		t.Fatalf("err = %v, want ErrRefMoved from %s", err, before)
	}
	if subject := output(t, repo, "log", "-1", "--format=%s"); subject != "foreign\n" {
		t.Fatalf("subject of HEAD = %q; the foreign commit was overwritten", subject)
	}
}

// No Python counterpart: the swap on an unborn branch expects the ref to be
// absent, and a foreign first commit is a moved ref too.
func TestCommitPathsRefusesAForeignFirstCommit(t *testing.T) {
	repo := emptyVault(t)
	write(t, repo, "seite.md", "neu")
	inWindow(t, func() { git(t, repo, "commit", "-q", "--allow-empty", "-m", "foreign") })

	_, err := vcs.CommitPaths(repo, "Start", []string{"seite.md"}, nil, t.TempDir())

	if !errors.Is(err, vcs.ErrRefMoved) || !strings.HasSuffix(err.Error(), " moved away from nothing") {
		t.Fatalf("err = %v, want ErrRefMoved from nothing", err)
	}
	if subject := output(t, repo, "log", "-1", "--format=%s"); subject != "foreign\n" {
		t.Fatalf("subject of HEAD = %q", subject)
	}
}

// test_vcs.py:205 test_a_branch_switch_during_the_run_is_refused
func TestCommitPathsRefusesABranchSwitch(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	git(t, repo, "branch", "andere")
	before := head(t, repo)
	write(t, repo, "seite.md", "neu")
	inWindow(t, func() { git(t, repo, "symbolic-ref", "HEAD", "refs/heads/andere") })

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if !errors.Is(err, vcs.ErrRefMoved) || !strings.Contains(err.Error(), "HEAD left refs/heads/") {
		t.Fatalf("err = %v, want ErrRefMoved", err)
	}
	if head(t, repo) != before {
		t.Fatal("the commit landed on a branch HEAD no longer names")
	}
}

// test_vcs.py:219 test_a_directory_without_git_returns_none
func TestCommitPathsIsNothingOutsideARepository(t *testing.T) {
	requireGit(t)

	got, err := vcs.CommitPaths(t.TempDir(), "Update", nil, nil, t.TempDir())

	if got != nil || err != nil {
		t.Fatalf("CommitPaths = %+v, %v; want nil, nil", got, err)
	}
}

// test_vcs.py:223 test_a_missing_directory_is_not_a_repository
func TestCommitPathsRefusesAMissingDirectory(t *testing.T) {
	_, err := vcs.CommitPaths(filepath.Join(t.TempDir(), "weg"), "Update", nil, nil, t.TempDir())

	if !errors.Is(err, vcs.ErrNotARepository) {
		t.Fatalf("err = %v, want ErrNotARepository", err)
	}
}

// test_vcs.py:228 test_the_first_commit_of_an_empty_repository
func TestCommitPathsMakesTheFirstCommit(t *testing.T) {
	repo := emptyVault(t)
	write(t, repo, "seite.md", "neu")

	got, err := vcs.CommitPaths(repo, "Start", []string{"seite.md"}, nil, t.TempDir())

	if err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}
	if got == nil || !got.Created || got.Head != head(t, repo) {
		t.Fatalf("commit = %+v", got)
	}
	if paths := changed(t, repo); !reflect.DeepEqual(paths, []string{"seite.md"}) {
		t.Fatalf("paths of the commit = %q", paths)
	}
}

// test_vcs.py:237 test_a_stale_scratch_index_does_not_leak_into_the_first_commit
func TestCommitPathsIgnoresAStaleScratchIndex(t *testing.T) {
	repo := emptyVault(t)
	scratch := t.TempDir()
	write(t, repo, "alt.md", "aus einem früheren Lauf")
	stale := exec.Command("git", "update-index", "--add", "--", "alt.md")
	stale.Dir = repo
	stale.Env = append(gitenv.Environ(), "GIT_INDEX_FILE="+filepath.Join(scratch, "index"))
	if out, err := stale.CombinedOutput(); err != nil {
		t.Fatalf("stale index: %v\n%s", err, out)
	}
	write(t, repo, "seite.md", "neu")

	if _, err := vcs.CommitPaths(repo, "Start", []string{"seite.md"}, nil, scratch); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	if paths := changed(t, repo); !reflect.DeepEqual(paths, []string{"seite.md"}) {
		t.Fatalf("paths of the commit = %q", paths)
	}
}

// test_vcs.py:252 test_a_scratch_path_that_is_a_directory_is_a_git_error. The
// directory is not empty, because os.Remove takes an empty one without a word.
func TestCommitPathsRefusesAnUnusableScratchIndex(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	scratch := t.TempDir()
	write(t, scratch, "index/inside", "x")
	write(t, repo, "seite.md", "neu")

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, scratch)

	if err == nil || !strings.Contains(err.Error(), "is unusable") {
		t.Fatalf("err = %v", err)
	}
}

// No Python counterpart (`Path.mkdir(parents=True)` is not in question there):
// a scratch directory that does not exist yet is made, parents included, and
// the commit goes through.
func TestCommitPathsMakesAMissingScratchDirectory(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	scratch := filepath.Join(t.TempDir(), "noch", "nicht", "da")
	write(t, repo, "seite.md", "neu")

	got, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, scratch)

	if err != nil || got == nil || !got.Created {
		t.Fatalf("CommitPaths = %+v, %v", got, err)
	}
	if info, err := os.Stat(scratch); err != nil || !info.IsDir() {
		t.Fatalf("scratch directory = %v, %v; want it made", info, err)
	}
}

// No Python counterpart: a scratch path that is a file cannot hold the index,
// and the refusal says so before git is asked to write anything there.
func TestCommitPathsRefusesAScratchPathThatIsAFile(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	before := head(t, repo)
	scratch := filepath.Join(t.TempDir(), "datei")
	write(t, filepath.Dir(scratch), "datei", "keine Ablage")
	write(t, repo, "seite.md", "neu")

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, scratch)

	if err == nil || !strings.Contains(err.Error(), "is unusable") {
		t.Fatalf("err = %v, want the scratch index refused", err)
	}
	if head(t, repo) != before {
		t.Fatal("HEAD moved")
	}
}

// test_vcs.py:261 test_an_inherited_git_dir_does_not_redirect_the_commit
func TestCommitPathsIgnoresAnInheritedGitDir(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	foreign := newVault(t, map[string]string{"anderes.md": "x"})
	foreignHead := head(t, foreign)
	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))
	t.Setenv("GIT_WORK_TREE", foreign)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(foreign, ".git", "index"))
	write(t, repo, "seite.md", "neu")

	got, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err != nil || got == nil || got.Head != head(t, repo) {
		t.Fatalf("CommitPaths = %+v, %v", got, err)
	}
	if head(t, foreign) != foreignHead {
		t.Fatal("the commit landed in the repository GIT_DIR names")
	}
}

// test_vcs.py:282 test_a_detached_head_is_moved_too
func TestCommitPathsMovesADetachedHead(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	git(t, repo, "checkout", "-q", "--detach")
	write(t, repo, "seite.md", "neu")

	got, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err != nil || got == nil || got.Head != head(t, repo) {
		t.Fatalf("CommitPaths = %+v, %v", got, err)
	}
	if strings.HasPrefix(string(readFile(t, filepath.Join(repo, ".git", "HEAD"))), "ref:") {
		t.Fatal("HEAD is attached again")
	}
}

// test_vcs.py:292 test_an_unchanged_tree_does_not_grow_a_commit
func TestCommitPathsMakesNoCommitForAnUnchangedTree(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	before := head(t, repo)

	got, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err != nil || got == nil || got.Created || got.Head != before {
		t.Fatalf("CommitPaths = %+v, %v; want the old head, not created", got, err)
	}
	if head(t, repo) != before {
		t.Fatal("HEAD moved")
	}
}

// test_vcs.py:302 test_a_path_outside_the_repository_is_refused and :308
// test_an_absolute_path_is_refused -- in either list.
func TestCommitPathsRefusesAPathOutsideTheRepository(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	for _, lists := range [][2][]string{
		{{"../draussen.md"}, nil},
		{{filepath.Join(repo, "seite.md")}, nil},
		{nil, {"a/../../b"}},
	} {
		_, err := vcs.CommitPaths(repo, "Update", lists[0], lists[1], t.TempDir())

		if !errors.Is(err, vcs.ErrOutsideRepository) {
			t.Errorf("CommitPaths(%q, %q): err = %v", lists[0], lists[1], err)
		}
	}
}

// test_vcs.py:316 test_a_missing_path_fails_before_the_ref_moves
func TestCommitPathsFailsOnAMissingPathBeforeTheRefMoves(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	before := head(t, repo)

	_, err := vcs.CommitPaths(repo, "Update", []string{"gibtsnicht.md"}, nil, t.TempDir())

	if err == nil || !strings.Contains(err.Error(), "git update-index --add -- gibtsnicht.md failed: ") {
		t.Fatalf("err = %v", err)
	}
	if head(t, repo) != before {
		t.Fatal("HEAD moved")
	}
}

// test_vcs.py:324 test_a_repository_without_a_committer_identity_fails_loudly
func TestCommitPathsFailsWithoutACommitterIdentity(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	git(t, repo, "config", "--unset", "user.email")
	git(t, repo, "config", "--unset", "user.name")
	git(t, repo, "config", "user.useConfigOnly", "true")
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME"} {
		t.Setenv(name, home)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	before := head(t, repo)
	write(t, repo, "seite.md", "neu")

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err == nil || !strings.Contains(err.Error(), "git commit-tree ") {
		t.Fatalf("err = %v", err)
	}
	if head(t, repo) != before {
		t.Fatal("HEAD moved")
	}
}

// No Python counterpart: a failed commit-tree ends the call before the window
// opens. Past it, a foreign commit would turn the commit-tree failure into
// ErrRefMoved, and the caller would retry a commit that cannot succeed.
func TestCommitPathsStopsAtAFailedCommitTreeBeforeTheWindow(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	git(t, repo, "config", "--unset", "user.email")
	git(t, repo, "config", "--unset", "user.name")
	git(t, repo, "config", "user.useConfigOnly", "true")
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME"} {
		t.Setenv(name, home)
	}
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	write(t, repo, "seite.md", "neu")
	inWindow(t, func() { t.Error("the window opened after commit-tree failed") })

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err == nil || errors.Is(err, vcs.ErrRefMoved) || !strings.Contains(err.Error(), "git commit-tree ") {
		t.Fatalf("err = %v, want the commit-tree failure", err)
	}
}

// test_vcs.py:347 test_a_case_directory_is_removed_with_every_file_in_it
func TestCommitPathsRemovesACaseDirectoryWithEveryFile(t *testing.T) {
	commitsTheWholeCase(t, "95 Prüfzentrum/fall-01")
}

// test_vcs.py:369 test_a_leading_space_in_a_case_path_survives_the_expansion
func TestCommitPathsKeepsALeadingSpaceThroughTheExpansion(t *testing.T) {
	commitsTheWholeCase(t, " 95 Prüfzentrum/fall-01")
}

func commitsTheWholeCase(t *testing.T, fall string) {
	t.Helper()
	repo := newVault(t, map[string]string{fall + "/case.toml": "x", fall + "/package.md": "y", "seite.md": "alt"})
	write(t, repo, "seite.md", "neu")

	if _, err := vcs.CommitPaths(repo, "Decide", []string{"seite.md"}, []string{fall}, t.TempDir()); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	want := []string{fall + "/case.toml", fall + "/package.md", "seite.md"}
	if paths := changed(t, repo); !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths of the commit = %q, want %q", paths, want)
	}
	if tracked := output(t, repo, "ls-tree", "-r", "--name-only", "-z", "HEAD"); tracked != "seite.md\x00" {
		t.Fatalf("HEAD still tracks %q", tracked)
	}
}

// test_vcs.py:391 test_a_wildcard_in_a_removal_path_matches_nothing
func TestCommitPathsReadsAWildcardLiterally(t *testing.T) {
	repo := newVault(t, map[string]string{"fall-1/case.toml": "x", "fall-2/case.toml": "y"})
	before := head(t, repo)

	_, err := vcs.CommitPaths(repo, "Decide", nil, []string{"fall-*"}, t.TempDir())

	if err == nil || !strings.Contains(err.Error(), "not tracked: fall-*") {
		t.Fatalf("err = %v", err)
	}
	if head(t, repo) != before {
		t.Fatal("HEAD moved")
	}
}

// test_vcs.py:400 test_an_inherited_config_parameter_does_not_change_the_committer
func TestCommitPathsIgnoresAnInheritedConfigParameter(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	t.Setenv("GIT_CONFIG_PARAMETERS", "'user.email'='fremd@merge.invalid'")
	write(t, repo, "seite.md", "neu")

	if _, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir()); err != nil {
		t.Fatalf("CommitPaths: %v", err)
	}

	if who := output(t, repo, "log", "-1", "--format=%ae%n%ce"); who != "brain@example.invalid\nbrain@example.invalid\n" {
		t.Fatalf("author and committer = %q", who)
	}
}

// test_vcs.py:414 test_a_rebase_in_progress_is_refused, with the second rebase
// form beside it.
func TestCommitPathsRefusesARebaseInProgress(t *testing.T) {
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		repo := newVault(t, map[string]string{"seite.md": "alt"})
		before := head(t, repo)
		if err := os.Mkdir(filepath.Join(repo, ".git", name), 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, repo, "seite.md", "neu")

		_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

		// The suffix, not the whole path: git reports the long form of the
		// repository while t.TempDir can hand out an 8.3 short one (RUNNER~1).
		want := filepath.Join(".git", name) + " exists: finish the rebase or merge first"
		if err == nil || !strings.HasSuffix(err.Error(), want) {
			t.Fatalf("err = %v, want %q", err, want)
		}
		if head(t, repo) != before {
			t.Fatal("a commit landed during a rebase")
		}
	}
}

// test_vcs.py:427 test_an_unfinished_merge_is_refused
func TestCommitPathsRefusesAnUnfinishedMerge(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	write(t, repo, ".git/MERGE_HEAD", head(t, repo))
	write(t, repo, "seite.md", "neu")

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err == nil || !strings.HasSuffix(err.Error(), "MERGE_HEAD exists: finish the rebase or merge first") {
		t.Fatalf("err = %v", err)
	}
}

// test_vcs.py:435 test_a_failing_update_ref_that_is_no_race_is_reported_as_such
func TestCommitPathsReportsAFailingUpdateRefThatIsNoRace(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	branch := strings.TrimSpace(output(t, repo, "branch", "--show-current"))
	git(t, repo, "symbolic-ref", "HEAD", "refs/heads/"+branch+"/sub")

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())

	if err == nil || errors.Is(err, vcs.ErrRefMoved) || !strings.Contains(err.Error(), "git update-ref ") {
		t.Fatalf("err = %v, want a plain update-ref failure", err)
	}
}

// test_vcs.py:445 test_a_missing_git_executable_is_reported_as_a_git_error
func TestCommitPathsReportsAGitThatCannotBeRun(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	got, err := vcs.CommitPaths(t.TempDir(), "Update", nil, nil, t.TempDir())

	if got != nil || err == nil || !strings.Contains(err.Error(), "could not be run") {
		t.Fatalf("CommitPaths = %+v, %v; want a spawn failure", got, err)
	}
}

// No Python counterpart: git vanishing inside the window is a spawn failure,
// reported as one and not read as a moved branch.
func TestCommitPathsReportsAGitLostInTheWindow(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt"})
	before := head(t, repo)
	write(t, repo, "seite.md", "neu")
	path := os.Getenv("PATH")
	t.Setenv("PATH", path)
	empty := t.TempDir()
	inWindow(t, func() { os.Setenv("PATH", empty) })

	_, err := vcs.CommitPaths(repo, "Update", []string{"seite.md"}, nil, t.TempDir())
	os.Setenv("PATH", path)

	// The first call past the window is the second reading of HEAD, and the
	// first failure is the one reported: a session that kept calling git would
	// name update-ref instead.
	if err == nil || errors.Is(err, vcs.ErrRefMoved) || !strings.Contains(err.Error(), "git symbolic-ref --quiet HEAD in ") ||
		!strings.Contains(err.Error(), "could not be run") {
		t.Fatalf("err = %v, want the spawn failure of symbolic-ref", err)
	}
	if head(t, repo) != before {
		t.Fatal("HEAD moved")
	}
}

// No Python counterpart: a failed add stops the session, so the removal after
// it is never looked up and cannot be reported as untracked -- the add's own
// failure is the one the caller sees.
func TestCommitPathsReportsTheFailedAddAndNotTheRemovalAfterIt(t *testing.T) {
	repo := newVault(t, map[string]string{"seite.md": "alt", "fall/case.toml": "x"})

	_, err := vcs.CommitPaths(repo, "Decide", []string{"gibtsnicht.md"}, []string{"fall"}, t.TempDir())

	if err == nil || !strings.Contains(err.Error(), "git update-index --add -- gibtsnicht.md failed: ") {
		t.Fatalf("err = %v, want the failed add", err)
	}
}

// newVault is newRepo with the identity in the repository's own config.
// commit-tree takes no `-c` from a fixture: CommitPaths reads the identity from
// the repository, as the Python fixture `_repo_with` sets it, and gitenv strips
// every variable that could carry one in. commit.gpgsign is off so a machine
// that signs by default asks for no key.
func newVault(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := newRepo(t, files)
	identify(t, repo)
	return repo
}

func emptyVault(t *testing.T) string {
	t.Helper()
	repo := emptyRepo(t)
	identify(t, repo)
	return repo
}

func identify(t *testing.T, repo string) {
	t.Helper()
	git(t, repo, "config", "user.name", "Brain")
	git(t, repo, "config", "user.email", "brain@example.invalid")
	git(t, repo, "config", "commit.gpgsign", "false")
}

// inWindow runs hook once in the window between commit-tree and the swap, where
// a foreign process would strike.
func inWindow(t *testing.T, hook func()) {
	t.Helper()
	fired := false
	t.Cleanup(vcs.SetBeforeUpdateRef(func() {
		if !fired {
			fired = true
			hook()
		}
	}))
}

// changed is the paths of the head commit, sorted and never trimmed: with `-z`
// neither core.quotepath nor a strip can touch an umlaut or a leading space.
func changed(t *testing.T, repo string) []string {
	t.Helper()
	listing := output(t, repo, "diff-tree", "--root", "--no-commit-id", "-r", "-z", "--name-only", "HEAD")
	paths := []string{}
	for _, entry := range strings.Split(listing, "\x00") {
		if entry != "" {
			paths = append(paths, entry)
		}
	}
	sort.Strings(paths)
	return paths
}

func output(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = repo
	command.Env = gitenv.Environ()
	out, err := command.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
