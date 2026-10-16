package rules

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/verify/linecov"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, a := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"config", "core.autocrlf", "false"}, {"config", "commit.gpgsign", "false"},
	} {
		git(t, root, a...)
	}
	return root
}

func rels(files []File) []string {
	var out []string
	for _, f := range files {
		out = append(out, f.Rel)
	}
	slices.Sort(out)
	return out
}

func TestFilesSkipsIgnoredAndWorktrees(t *testing.T) {
	root := repo(t)
	write(t, root, ".gitignore", ".godot/\n")
	write(t, root, ".godot/x.gd", "x")
	write(t, root, "core/a.gd", "a")
	nested := filepath.Join(root, ".claude", "worktrees", "w")
	write(t, nested, "core/a.gd", "w")
	git(t, nested, "init", "-q")
	got, err := Files(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{".gitignore", "core/a.gd"}; !slices.Equal(rels(got), want) {
		t.Fatalf("got %v, want %v", rels(got), want)
	}
}

func TestFilesWorkingTree(t *testing.T) {
	root := repo(t)
	write(t, root, "a.gd", "a")
	write(t, root, "c.gd", "c")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "init")
	write(t, root, "b.gd", "b")
	write(t, root, "a.gd", "a2")
	if err := os.Remove(filepath.Join(root, "c.gd")); err != nil {
		t.Fatal(err)
	}
	got, err := Files(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.gd", "b.gd"}; !slices.Equal(rels(got), want) {
		t.Fatalf("got %v, want %v", rels(got), want)
	}
	for _, f := range got {
		if f.Rel == "a.gd" && string(f.Content) != "a2" {
			t.Fatalf("content %q", f.Content)
		}
	}
}

func TestFilesWorkingTreeReadError(t *testing.T) {
	root := repo(t)
	write(t, root, "a.gd", "a")
	git(t, root, "add", ".")
	if err := os.Remove(filepath.Join(root, "a.gd")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "a.gd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Files(root, false); err == nil {
		t.Fatal("want a read error for a tracked file that is a directory")
	}
}

func TestFilesCached(t *testing.T) {
	root := repo(t)
	write(t, root, "a.gd", "x")
	git(t, root, "add", ".")
	write(t, root, "a.gd", "y")
	write(t, root, "b.gd", "b")
	got, err := Files(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Rel != "a.gd" || string(got[0].Content) != "x" {
		t.Fatalf("got %+v", got)
	}
}

func TestFilesCachedEmpty(t *testing.T) {
	got, err := Files(repo(t), true)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestFilesCachedHonoursHookIndex(t *testing.T) {
	root := repo(t)
	write(t, root, "a.gd", "old")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "init")
	// Git for Windows compares mtimes to the second, and the copied index
	// below carries a fresh mtime that hides the racy entry from git, so
	// `git add` trusts the stat data whenever size and second match. The
	// new content differs in size; pinning the old mtime keeps the second
	// equal, so the test cannot pass by luck of the clock.
	info, err := os.Stat(filepath.Join(root, "a.gd"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, root, "a.gd", "newer")
	if err := os.Chtimes(filepath.Join(root, "a.gd"), info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".git", "index"))
	if err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(root, ".git", "next-index-1.lock")
	if err := os.WriteFile(index, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", root, "add", "a.gd")
	cmd.Env = append(gitenv.Environ(), "GIT_INDEX_FILE="+index)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	t.Setenv("GIT_INDEX_FILE", index)
	got, err := Files(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || string(got[0].Content) != "newer" {
		t.Fatalf("got %+v", got)
	}
}

func TestFilesOddNames(t *testing.T) {
	root := repo(t)
	write(t, root, "a #b/ü.gd", "u")
	git(t, root, "add", ".")
	for _, cached := range []bool{false, true} {
		got, err := Files(root, cached)
		if err != nil || len(got) != 1 || got[0].Rel != "a #b/ü.gd" || string(got[0].Content) != "u" {
			t.Fatalf("cached=%v: %+v, %v", cached, got, err)
		}
	}
}

func TestFilesNotARepo(t *testing.T) {
	for _, cached := range []bool{false, true} {
		if _, err := Files(t.TempDir(), cached); err == nil {
			t.Fatalf("cached=%v: want an error outside a repository", cached)
		}
	}
}

func TestParseStage(t *testing.T) {
	out := "100644 aaa 0\ta.gd\x00160000 bbb 0\tsub\x00100644 ccc 1\tm.gd\x00100644 ddd 2\tm.gd\x00junk\x00"
	rels, ids := parseStage([]byte(out))
	if !slices.Equal(rels, []string{"a.gd", "m.gd"}) || !slices.Equal(ids, []string{"aaa", "ccc"}) {
		t.Fatalf("got %v %v", rels, ids)
	}
}

func TestCatBlobsFailure(t *testing.T) {
	if _, err := catBlobs(t.TempDir(), []string{"x"}); err == nil {
		t.Fatal("want an error outside a repository")
	}
}

func TestReadBatchErrors(t *testing.T) {
	for name, in := range map[string]string{
		"empty":     "",
		"header":    "abc missing\n",
		"size":      "abc blob x\n",
		"truncated": "abc blob 5\nab",
	} {
		if _, err := readBatch(bufio.NewReader(strings.NewReader(in)), 1); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestFilesCachedMissingObject(t *testing.T) {
	root := repo(t)
	write(t, root, "a.gd", "x")
	git(t, root, "add", ".")
	objects := filepath.Join(root, ".git", "objects")
	if err := os.RemoveAll(objects); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(objects, "pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Files(root, true); err == nil {
		t.Fatal("want an error for a blob the object store lacks")
	}
}

func TestFilesSkipsSubmodules(t *testing.T) {
	root := repo(t)
	write(t, root, "a.gd", "a")
	write(t, root, "sub/f.gd", "f")
	git(t, root, "add", "a.gd")
	git(t, root, "update-index", "--add", "--cacheinfo", "160000,1111111111111111111111111111111111111111,sub")
	for _, cached := range []bool{false, true} {
		got, err := Files(root, cached)
		if err != nil || !slices.Equal(rels(got), []string{"a.gd"}) {
			t.Fatalf("cached=%v: %v, %v", cached, rels(got), err)
		}
	}
}

// Only the files a rule parses carry their content; the others are listed by
// path, in both modes.
func TestFilesReadOnlyWhatTheRulesParse(t *testing.T) {
	root := repo(t)
	wantContent := map[string]bool{
		"a.go": true, "b.py": true, "c.gd": true, "d.tscn": true, "e.tres": true, "project.godot": true,
		"go.mod": true, "sub/plugin.cfg": true, "README.md": false, "x.png": false, "settings.toml": false,
	}
	for rel := range wantContent {
		write(t, root, rel, "body")
	}
	git(t, root, "add", ".")
	for _, cached := range []bool{false, true} {
		got, err := Files(root, cached)
		if err != nil || len(got) != len(wantContent) {
			t.Fatalf("cached=%v: %v, %v", cached, got, err)
		}
		for _, f := range got {
			if has := string(f.Content) == "body"; has != wantContent[f.Rel] || (!has && f.Content != nil) {
				t.Errorf("cached=%v: %s has content %q", cached, f.Rel, f.Content)
			}
		}
	}
}

// A tracked file that is gone from disk is left out whether or not its
// content is read.
func TestFilesWorkingTreeDropsAMissingFileWithoutContent(t *testing.T) {
	root := repo(t)
	write(t, root, "README.md", "r")
	git(t, root, "add", ".")
	if err := os.Remove(filepath.Join(root, "README.md")); err != nil {
		t.Fatal(err)
	}
	if got, err := Files(root, false); err != nil || len(got) != 0 {
		t.Fatalf("%v, %v", got, err)
	}
}

// Instrumented judges the first line of the sources of a tool with a marker,
// so every extension of such a tool must be one whose content is read.
func TestContentExtsCoverTheInstrumentedTools(t *testing.T) {
	ps, err := linecov.LoadProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		if p.Instrumented == "" {
			continue
		}
		for _, e := range p.Extensions {
			if !wantsContent("x" + e) {
				t.Errorf("%s: %s is not read", p.Name, e)
			}
		}
	}
}
