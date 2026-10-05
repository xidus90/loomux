package gitenv

import (
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestCleanDropsEveryInheritedGitVariable(t *testing.T) {
	parent := []string{
		"PATH=/bin",
		"GIT_DIR=/x", "GIT_CONFIG_PARAMETERS='a=b'", "GIT_AUTHOR_NAME=Brain",
		"GIT_CONFIG_KEY_0=user.name", "GIT_CONFIG_VALUE_0=Brain",
		"GIT_TRACE=1",
	}
	got := Clean(parent)
	want := []string{"PATH=/bin", "GIT_TRACE=1"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCleanDropsGitsRepositoryPointers(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin",
		"GIT_DIR=/repo/.git/worktrees/feature",
		"GIT_WORK_TREE=/repo",
		"GIT_COMMON_DIR=/repo/.git",
		"GIT_INDEX_FILE=/repo/.git/index",
		"GIT_PREFIX=sub/dir/",
		"GIT_OBJECT_DIRECTORY=/repo/.git/objects",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES=/other/.git/objects",
	}
	if got := Clean(parent); !slices.Equal(got, []string{"PATH=/usr/bin"}) {
		t.Fatalf("Clean = %q, want only PATH", got)
	}
}

// What goes is what redirects git at a repository, how it is configured and
// whom it writes as -- the test above pins GIT_AUTHOR_NAME and the
// GIT_CONFIG pairs among them. What stays is the user's own settings.
func TestCleanKeepsGitsOtherVariables(t *testing.T) {
	parent := []string{"GIT_EDITOR=vi", "GIT_TERMINAL_PROMPT=0"}
	if got := Clean(parent); !slices.Equal(got, parent) {
		t.Fatalf("Clean = %q, want the input unchanged", got)
	}
}

// An entry without a separator is not a name we can match, so it stays.
func TestCleanPassesThroughAnEntryWithoutAValue(t *testing.T) {
	parent := []string{"GIT_DIR", "PATH=/usr/bin"}
	if got := Clean(parent); !slices.Equal(got, parent) {
		t.Fatalf("Clean = %q, want the input unchanged", got)
	}
}

// On Windows a name is one variable however it is spelled, and git.exe reads
// `git_dir` as GIT_DIR; on POSIX `git_dir` is a variable git never reads.
func TestCleanFoldsCaseOnlyWhereTheEnvironmentDoes(t *testing.T) {
	parent := []string{"git_dir=/repo/.git", "Git_Work_Tree=/repo", "git_config_key_0=user.name", "PATH=/bin"}
	if got := clean(parent, true); !slices.Equal(got, []string{"PATH=/bin"}) {
		t.Fatalf("folding: Clean = %q, want only PATH", got)
	}
	if got := clean(parent, false); !slices.Equal(got, parent) {
		t.Fatalf("not folding: Clean = %q, want the input unchanged", got)
	}
}

func TestCleanFoldsCaseOnWindows(t *testing.T) {
	folded := len(Clean([]string{"git_dir=/repo/.git"})) == 0
	if folded != (runtime.GOOS == "windows") {
		t.Fatalf("Clean folded case: %v on %s", folded, runtime.GOOS)
	}
}

func TestEnvironReadsThisProcess(t *testing.T) {
	t.Setenv("GIT_DIR", "/repo/.git")
	t.Setenv("LOOMUX_GITENV_PROBE", "1")

	environ := Environ()
	if slices.Contains(environ, "GIT_DIR=/repo/.git") {
		t.Fatal("Environ kept GIT_DIR")
	}
	if !slices.Contains(environ, "LOOMUX_GITENV_PROBE=1") {
		t.Fatal("Environ dropped a variable that is not git's")
	}
	if len(environ) >= len(os.Environ()) {
		t.Fatalf("Environ = %d entries, os.Environ = %d; want fewer", len(environ), len(os.Environ()))
	}
}
