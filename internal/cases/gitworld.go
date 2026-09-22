package cases

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/xidus90/loomux/internal/gitenv"
)

// GitWorldFile declares the repository a staged world becomes.
const GitWorldFile = "git.toml"

// GitWorld is git.toml: the directory the repository is made in, commits in
// order, local branches and what the remote holds by commit number (from 1),
// and files changed after the last commit.
//
// Dir is relative to the world and empty for the world itself; every path of
// the declaration is relative to the repository.
type GitWorld struct {
	Dir      string            `toml:"dir"`
	Commits  []GitCommit       `toml:"commit"`
	Remote   *GitRemote        `toml:"remote"`
	Branches map[string]int    `toml:"branches"`
	Worktree map[string]string `toml:"worktree"`
}

// GitCommit commits the staged files it names and the files it spells out.
type GitCommit struct {
	Message string            `toml:"message"`
	Paths   []string          `toml:"paths"`
	Files   map[string]string `toml:"files"`
}

// GitRemote is a bare repository at .origin.git, and the commit each of its
// branches is pushed to.
type GitRemote struct {
	Push map[string]int `toml:"push"`
}

// The empty global configuration is the one file a build makes outside the
// world, and the one step a test cannot fail from outside, so the call is a
// seam.
var defaultCreateTemp = os.CreateTemp

var createTemp = defaultCreateTemp

// excluded are the files of the test bench itself: a hook that measures the
// working tree must not see them as somebody's change.
var excluded = []string{"/git.toml", "/faketool.json", "/.origin.git/", "/.ultraloom/", "/.loomux/", "/.claude/"}

// noGitHome is the home GitEnv names inside a world. Nothing makes it, so git
// finds no .gitconfig there and nothing lands in the tree a case compares.
const noGitHome = ".no-git-home"

// GitEnv is the git environment a recording and its replay both run the
// command under: no system configuration, and a home without a .gitconfig.
// XDG_CONFIG_HOME, git's other place for a user file, is the caller's to set:
// a 3a world names its own, and the 2c replay empties it.
//
// The obvious GIT_CONFIG_GLOBAL and GIT_CONFIG_SYSTEM are useless here: the
// reference's vcs.py and loomux's gitenv both strip them before git starts, so
// the recorded command would have read the user's file all the same. Neither
// list names HOME or GIT_CONFIG_NOSYSTEM. BuildGitWorld is a third party and
// keeps its own isolation; it runs git directly, without either strip.
func GitEnv(world string) []string {
	return []string{"GIT_CONFIG_NOSYSTEM=1", "HOME=" + filepath.ToSlash(world) + "/" + noGitHome}
}

// InfraPath reports whether a slash-separated path inside a world belongs to
// the repositories BuildGitWorld made rather than to the world.
//
// Any segment and not only the first: the repository may be one of the
// world's directories, and a caller holds the path without the declaration
// that says which.
func InfraPath(rel string) bool {
	for segment := range strings.SplitSeq(rel, "/") {
		if segment == ".git" || segment == ".origin.git" {
			return true
		}
	}
	return false
}

// BuildGitWorld makes the repository its git.toml declares -- in the world
// dir itself, or in the directory the declaration's `dir` names below it --
// then puts each commit's SHA in place of {{COMMIT:<n>}} in every other file
// of the world.
//
// Git runs here without gitenv's strip and with an identity of its own: the
// variables gitenv takes out are exactly the ones that make the same
// declaration the same SHA on every machine. The user's global and system
// configuration are kept out for the same reason, and the object format is
// named rather than left to GIT_DEFAULT_HASH, because a world_after holds the
// SHAs written out.
func BuildGitWorld(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, GitWorldFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var w GitWorld
	if _, err := toml.Decode(string(raw), &w); err != nil {
		return fmt.Errorf("%s: %w", GitWorldFile, err)
	}
	if w.Dir != "" && !filepath.IsLocal(w.Dir) {
		return fmt.Errorf("%s: dir %q leaves the world", GitWorldFile, w.Dir)
	}
	empty, err := createTemp("", "gitworld-config-*")
	if err != nil {
		return err
	}
	empty.Close()
	defer os.Remove(empty.Name())
	repo := filepath.Join(dir, filepath.FromSlash(w.Dir))
	g := &worldGit{dir: repo, config: empty.Name()}
	g.fail(os.MkdirAll(repo, 0o755))
	g.run("init", "-q", "--object-format=sha1", "-b", "master")
	if g.err == nil {
		// Written, not appended: with the user's configuration kept out, no
		// template put anything there worth keeping.
		g.fail(writeFile(repo, ".git/info/exclude", strings.Join(excluded, "\n")+"\n"))
	}
	var shas []string
	for i, c := range w.Commits {
		shas = append(shas, g.commit(i+1, c))
	}
	commit := func(n int) (string, error) {
		if n < 1 || n > len(shas) {
			return "", fmt.Errorf("%s names commit %d of %d", GitWorldFile, n, len(shas))
		}
		return shas[n-1], nil
	}
	for _, name := range slices.Sorted(maps.Keys(w.Branches)) {
		sha, err := commit(w.Branches[name])
		g.fail(err)
		g.run("branch", name, sha)
	}
	if w.Remote != nil {
		g.run("init", "-q", "--bare", "--object-format=sha1", "-b", "master", ".origin.git")
		g.run("remote", "add", "origin", "./.origin.git")
		for _, branch := range slices.Sorted(maps.Keys(w.Remote.Push)) {
			sha, err := commit(w.Remote.Push[branch])
			g.fail(err)
			g.run("push", "-q", "origin", sha+":refs/heads/"+branch)
		}
	}
	for _, path := range slices.Sorted(maps.Keys(w.Worktree)) {
		g.fail(writeFile(repo, path, w.Worktree[path]))
	}
	if g.err != nil {
		return g.err
	}
	return replaceCommitTokens(dir, shas)
}

// worldGit runs git in one world and keeps the first failure.
type worldGit struct {
	dir, config string
	err         error
}

func (g *worldGit) fail(err error) {
	if g.err == nil {
		g.err = err
	}
}

func (g *worldGit) run(args ...string) string {
	return g.runAt(0, args...)
}

// runAt runs git with commit n's date; a date per commit keeps the SHAs
// apart when two commits carry the same tree and message.
func (g *worldGit) runAt(n int, args ...string) string {
	if g.err != nil {
		return ""
	}
	date := fmt.Sprintf("@%d +0000", 1767225600+60*n)
	cmd := exec.Command("git", append([]string{"-c", "core.autocrlf=false", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=" + os.DevNull}, args...)...)
	cmd.Dir = g.dir
	cmd.Env = append(gitenv.Environ(),
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+g.config,
		"GIT_AUTHOR_NAME=loomux cases", "GIT_AUTHOR_EMAIL=cases@loomux.invalid", "GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_NAME=loomux cases", "GIT_COMMITTER_EMAIL=cases@loomux.invalid", "GIT_COMMITTER_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		g.err = fmt.Errorf("git %s in %s: %v: %s", strings.Join(args, " "), g.dir, err, strings.TrimSpace(string(out)))
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (g *worldGit) commit(n int, c GitCommit) string {
	paths := slices.Clone(c.Paths)
	for _, path := range slices.Sorted(maps.Keys(c.Files)) {
		g.fail(writeFile(g.dir, path, c.Files[path]))
		paths = append(paths, path)
	}
	if len(paths) > 0 {
		g.run(append([]string{"add", "--"}, paths...)...)
	}
	g.runAt(n, "commit", "-q", "--allow-empty", "-m", c.Message)
	return g.run("rev-parse", "HEAD")
}

func writeFile(dir, rel, body string) error {
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

// replaceCommitTokens puts the SHAs into every file of the world outside the
// repositories and the declaration.
func replaceCommitTokens(dir string, shas []string) error {
	token := regexp.MustCompile(`\{\{COMMIT:(\d+)\}\}`)
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if InfraPath(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == GitWorldFile {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var bad error
		out := token.ReplaceAllFunc(data, func(m []byte) []byte {
			n, _ := strconv.Atoi(string(token.FindSubmatch(m)[1]))
			if n < 1 || n > len(shas) {
				bad = fmt.Errorf("%s names commit %d of %d", rel, n, len(shas))
				return m
			}
			return []byte(shas[n-1])
		})
		if bad != nil {
			return bad
		}
		return os.WriteFile(path, out, 0o644)
	})
}
