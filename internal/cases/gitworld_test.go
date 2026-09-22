package cases

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/testlock"
)

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// The same environment the build ran in: a reader that took the user's
	// configuration along would answer for their machine, not for the world.
	cmd.Env = append(gitenv.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func writeWorld(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const twoCommits = `
[[commit]]
message = "base"
paths = ["a.txt"]

[[commit]]
message = "second"
[commit.files]
"a.txt" = "two\n"

[branches]
feature = 1

[remote.push]
master = 2

[worktree]
"a.txt" = "three\n"
`

func TestBuildGitWorldMakesTheDeclaredRepository(t *testing.T) {
	dir := writeWorld(t, map[string]string{
		"git.toml":              twoCommits,
		"a.txt":                 "one\n",
		".ultraloom/state.json": `{"base": "{{COMMIT:1}}", "head": "{{COMMIT:2}}"}`,
	})
	if err := BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	first := gitOut(t, dir, "rev-parse", "HEAD~1")
	second := gitOut(t, dir, "rev-parse", "HEAD")
	if got := gitOut(t, dir, "rev-parse", "feature"); got != first {
		t.Fatalf("feature at %s, want %s", got, first)
	}
	if got := gitOut(t, dir, "ls-remote", "origin", "refs/heads/master"); !strings.HasPrefix(got, second) {
		t.Fatalf("origin master: %q, want %s", got, second)
	}
	if got := gitOut(t, dir, "show", "HEAD~1:a.txt"); got != "one" {
		t.Fatalf("first commit holds %q", got)
	}
	body, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
	if string(body) != "three\n" {
		t.Fatalf("worktree a.txt = %q", body)
	}
	state, _ := os.ReadFile(filepath.Join(dir, ".ultraloom", "state.json"))
	if want := `{"base": "` + first + `", "head": "` + second + `"}`; string(state) != want {
		t.Fatalf("tokens: %s", state)
	}
	// The fixture files of the world are no change a hook should see.
	if got := gitOut(t, dir, "status", "--porcelain"); got != "M a.txt" {
		t.Fatalf("status %q", got)
	}
}

// Fixed identity and dates: the same declaration is the same SHA on every
// machine, which recordings and replays depend on.
func TestBuildGitWorldIsDeterministic(t *testing.T) {
	var heads []string
	for range 2 {
		dir := writeWorld(t, map[string]string{"git.toml": twoCommits, "a.txt": "one\n"})
		if err := BuildGitWorld(dir); err != nil {
			t.Fatal(err)
		}
		heads = append(heads, gitOut(t, dir, "rev-parse", "HEAD"))
	}
	if heads[0] != heads[1] {
		t.Fatalf("two builds, two heads: %v", heads)
	}
}

// A world whose repository is one of its directories: `dir` names it, the
// declaration's paths are the repository's own, and the tokens are replaced
// in the whole world -- a state file beside the repository names its commits.
func TestBuildGitWorldBuildsTheRepositoryInTheNamedDirectory(t *testing.T) {
	dir := writeWorld(t, map[string]string{
		"git.toml":                "dir = \"repo-a\"\n" + twoCommits,
		"repo-a/a.txt":            "one\n",
		"maintenance/events.tsv":  "{{COMMIT:1}}\t{{COMMIT:2}}\n",
		"repo-a/notes/commit.txt": "{{COMMIT:2}}",
	})
	if err := BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatalf("the world root must stay no repository: %v", err)
	}
	repo := filepath.Join(dir, "repo-a")
	first := gitOut(t, repo, "rev-parse", "HEAD~1")
	second := gitOut(t, repo, "rev-parse", "HEAD")
	if got := gitOut(t, repo, "rev-parse", "--show-toplevel"); !strings.EqualFold(filepath.Clean(got), filepath.Clean(repo)) {
		t.Fatalf("top level %s, want %s", got, repo)
	}
	if got := gitOut(t, repo, "show", "HEAD~1:a.txt"); got != "one" {
		t.Fatalf("first commit holds %q", got)
	}
	if got := gitOut(t, repo, "ls-remote", "origin", "refs/heads/master"); !strings.HasPrefix(got, second) {
		t.Fatalf("origin master: %q, want %s", got, second)
	}
	if _, err := os.Stat(filepath.Join(repo, ".origin.git")); err != nil {
		t.Fatalf("the remote lies beside the repository it serves: %v", err)
	}
	events, _ := os.ReadFile(filepath.Join(dir, "maintenance", "events.tsv"))
	if want := first + "\t" + second + "\n"; string(events) != want {
		t.Fatalf("tokens outside the repository: %q", events)
	}
	inside, _ := os.ReadFile(filepath.Join(repo, "notes", "commit.txt"))
	if string(inside) != second {
		t.Fatalf("tokens inside the repository: %q", inside)
	}
	// The worktree change is the declaration's; the untracked directory is the
	// world's own file, which no commit named.
	if got := gitOut(t, repo, "status", "--porcelain"); got != "M a.txt\n?? notes/" {
		t.Fatalf("status %q", got)
	}
}

// A dir that leaves the world, or names a place of its own, would build a
// repository outside the bench.
func TestBuildGitWorldRefusesADirOutsideTheWorld(t *testing.T) {
	outside := []string{"../out", "/rooted", "a/../../out"}
	// A volume name leaves the world only where there are volumes: on POSIX,
	// `C:/elsewhere` is a directory called `C:` below the world.
	if runtime.GOOS == "windows" {
		outside = append(outside, "C:/elsewhere")
	}
	for _, dir := range outside {
		t.Run(dir, func(t *testing.T) {
			decl := "dir = \"" + dir + "\"\n[[commit]]\nmessage = \"a\"\n"
			world := writeWorld(t, map[string]string{"git.toml": decl})
			err := BuildGitWorld(world)
			if err == nil || !strings.Contains(err.Error(), "dir") {
				t.Fatalf("want a refusal naming dir, got %v", err)
			}
		})
	}
}

// The repositories of a nested git world are the bench as much as a root one.
func TestCollectFilesLeavesANestedRepositoryOut(t *testing.T) {
	dir := writeWorld(t, map[string]string{
		"repo-a/.git/HEAD": "ref: refs/heads/master\n", "repo-a/.origin.git/HEAD": "x", "repo-a/a.txt": "x",
	})
	files, err := collectFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files["repo-a/a.txt"] == nil {
		t.Fatalf("files: %v", files)
	}
}

// The environment of GitEnv hands git no configuration file at all -- not the
// system's, not the user's -- whatever this machine carries. The variables
// are ones neither the reference's strip list nor gitenv's takes out.
func TestGitEnvLeavesGitWithoutAConfigurationFile(t *testing.T) {
	world := t.TempDir()
	env := GitEnv(world)
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if slices.Contains(gitenv.Location, key) {
			t.Errorf("%s is on gitenv's strip list; loomux's git would never see it", key)
		}
	}
	cmd := exec.Command("git", "config", "--list", "--show-origin")
	cmd.Dir = world
	cmd.Env = append(gitenv.Environ(), env...)
	out, err := cmd.CombinedOutput()
	// Exit 0 with nothing listed, or exit 1 where a git build treats an empty
	// list as a missing key; either way no file answered.
	if len(out) != 0 {
		t.Fatalf("git read configuration (%v):\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(world, noGitHome)); !os.IsNotExist(err) {
		t.Fatalf("the home is never made: %v", err)
	}
}

// The object format is part of every SHA a world_after writes out, and git
// takes its default from GIT_DEFAULT_HASH, which no strip list names -- and
// git 3.0 announces SHA-256 as the default. The build pins SHA-1 both for the
// repository and for its remote.
func TestBuildGitWorldPinsTheObjectFormat(t *testing.T) {
	t.Setenv("GIT_DEFAULT_HASH", "sha256")
	dir := writeWorld(t, map[string]string{"git.toml": twoCommits, "a.txt": "one\n"})
	if err := BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	for _, repo := range []string{".", ".origin.git"} {
		if got := gitOut(t, filepath.Join(dir, repo), "rev-parse", "--show-object-format"); got != "sha1" {
			t.Errorf("%s: object format %q", repo, got)
		}
	}
	if head := gitOut(t, dir, "rev-parse", "HEAD"); len(head) != 40 {
		t.Errorf("HEAD %q", head)
	}
}

func TestBuildGitWorldWithoutDeclarationDoesNothing(t *testing.T) {
	dir := writeWorld(t, map[string]string{"a.txt": "x"})
	if err := BuildGitWorld(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Fatalf(".git: %v", err)
	}
}

func TestBuildGitWorldRefuses(t *testing.T) {
	cases := map[string]string{
		"bad toml":            "[[commit]",
		"commit out of range": "[[commit]]\nmessage = \"a\"\n[branches]\nx = 2\n",
		"push out of range":   "[[commit]]\nmessage = \"a\"\n[remote.push]\nmaster = 0\n",
		"missing path":        "[[commit]]\nmessage = \"a\"\npaths = [\"nope.txt\"]\n",
		"token out of range":  "[[commit]]\nmessage = \"a\"\n",
	}
	for name, decl := range cases {
		t.Run(name, func(t *testing.T) {
			files := map[string]string{"git.toml": decl}
			if name == "token out of range" {
				files["x.json"] = "{{COMMIT:9}}"
			}
			if err := BuildGitWorld(writeWorld(t, files)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

func TestBuildGitWorldReportsADeclarationItCannotRead(t *testing.T) {
	dir := writeWorld(t, map[string]string{"git.toml": ""})
	testlock.Lock(t, filepath.Join(dir, GitWorldFile))
	if err := BuildGitWorld(dir); err == nil {
		t.Fatal("want an error")
	}
}

// The empty global configuration is the one file BuildGitWorld makes outside
// the world; a test can take it away only through the seam.
func TestBuildGitWorldReportsAConfigItCannotMake(t *testing.T) {
	createTemp = func(string, string) (*os.File, error) { return nil, errors.New("no temp") }
	defer func() { createTemp = defaultCreateTemp }()
	dir := writeWorld(t, map[string]string{"git.toml": twoCommits, "a.txt": "one\n"})
	if err := BuildGitWorld(dir); err == nil {
		t.Fatal("want an error")
	}
}

// A .git that is a file git cannot read stops the build at init, before the
// exclude list is written into it.
func TestBuildGitWorldReportsARepositoryItCannotInit(t *testing.T) {
	dir := writeWorld(t, map[string]string{"git.toml": twoCommits, ".git": "not a gitdir\n"})
	if err := BuildGitWorld(dir); err == nil {
		t.Fatal("want an error")
	}
}

func TestBuildGitWorldReportsAWorktreeFileItCannotWrite(t *testing.T) {
	decl := "[[commit]]\nmessage = \"a\"\n[worktree]\n\"a.txt/b.txt\" = \"x\"\n"
	dir := writeWorld(t, map[string]string{"git.toml": decl, "a.txt": "file, not a directory"})
	if err := BuildGitWorld(dir); err == nil {
		t.Fatal("want an error")
	}
}

func TestBuildGitWorldReportsAFileItCannotRewrite(t *testing.T) {
	dir := writeWorld(t, map[string]string{"git.toml": "[[commit]]\nmessage = \"a\"\n", "x.json": "{}"})
	testlock.Lock(t, filepath.Join(dir, "x.json"))
	if err := BuildGitWorld(dir); err == nil {
		t.Fatal("want an error")
	}
}

func TestBuildGitWorldReportsADirectoryItCannotWalk(t *testing.T) {
	dir := writeWorld(t, map[string]string{"git.toml": "[[commit]]\nmessage = \"a\"\n", "sub/x.json": "{}"})
	testlock.LockDir(t, filepath.Join(dir, "sub"))
	if err := BuildGitWorld(dir); err == nil {
		t.Fatal("want an error")
	}
}

func TestInfraPath(t *testing.T) {
	for rel, want := range map[string]bool{
		".git": true, ".git/index": true, ".origin.git/HEAD": true,
		"sub/.git": true, "repo-a/.git/index": true, "repo-a/.origin.git/HEAD": true,
		"a.txt": false, ".github/x": false, "sub/.gitignore": false, "sub/x.git/y": false,
	} {
		if InfraPath(rel) != want {
			t.Errorf("InfraPath(%q) = %v", rel, !want)
		}
	}
}

// RunCase builds the declared repository before loomux runs, and the tree it
// compares afterwards holds the world, not the repositories.
func TestRunCaseBuildsTheGitWorldAndLeavesItsRepositoriesOutOfTheTree(t *testing.T) {
	c := &Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x", HasWorldAfter: true, Compare: "message"}
	world := filepath.Join(c.Path, "world")
	after := filepath.Join(c.Path, "world_after")
	for dir, files := range map[string]map[string]string{
		world: {"git.toml": twoCommits, "a.txt": "one\n"},
		after: {"git.toml": twoCommits, "a.txt": "three\n"},
	} {
		for name, body := range files {
			os.MkdirAll(dir, 0o755)
			os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644)
		}
	}
	outcome, err := RunCase(c, func(_ []string, dir string, _ io.Reader, _, _ io.Writer) int {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
			t.Errorf("no repository: %v", err)
		}
		return 0
	})
	if err != nil || !outcome.Passed {
		t.Fatalf("%v %+v", err, outcome)
	}
}

// A .git that is a file -- a linked worktree's pointer -- is the bench as
// much as a .git directory is.
func TestCollectFilesLeavesAGitFileOut(t *testing.T) {
	dir := writeWorld(t, map[string]string{".git": "gitdir: elsewhere\n", "a.txt": "x"})
	files, err := collectFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[".git"]; ok || len(files) != 1 {
		t.Fatalf("files: %v", files)
	}
}

func TestRunCaseReportsAGitWorldItCannotBuild(t *testing.T) {
	c := &Case{Verb: "v", Name: "n", Path: t.TempDir(), Cmd: "loomux x"}
	os.MkdirAll(filepath.Join(c.Path, "world"), 0o755)
	os.WriteFile(filepath.Join(c.Path, "world", GitWorldFile), []byte("[[commit]"), 0o644)
	if _, err := RunCase(c, nil); err == nil {
		t.Fatal("want an error")
	}
}
