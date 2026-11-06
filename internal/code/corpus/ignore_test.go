package corpus

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
)

// isolateGit gives the test its own git configuration: no system file, no
// user file, and a HOME/XDG pair whose global excludes file the test can
// write. Without it a developer's ~/.gitconfig or ~/.config/git/ignore would
// decide what an ignore pattern means.
func isolateGit(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "gitconfig"))
	if err := os.MkdirAll(filepath.Join(home, "xdg"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

// gitRun runs git in root and fails the test on a non-zero exit.
func gitRun(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := gitRunInput(t, root, "", args...)
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

// gitRunInput runs git in root with stdin and returns its output and error.
func gitRunInput(t *testing.T, root, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(gitenv.Environ(),
		"GIT_CONFIG_NOSYSTEM="+os.Getenv("GIT_CONFIG_NOSYSTEM"),
		"GIT_CONFIG_GLOBAL="+os.Getenv("GIT_CONFIG_GLOBAL"),
	)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// newRepo writes files into a fresh repository. The map keys are
// repository-relative slash paths; the repository is initialised first, so a
// case can write .git/info/exclude. tracked names are added with -f, which is
// what the tracked exception needs.
func newRepo(t *testing.T, files map[string]string, tracked []string, ignoreCase *bool) string {
	t.Helper()
	isolateGit(t)
	root := t.TempDir()
	gitRun(t, root, "init", "-q")
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if ignoreCase != nil {
		gitRun(t, root, "config", "core.ignorecase", strconv.FormatBool(*ignoreCase))
	}
	if len(tracked) > 0 {
		gitRun(t, root, append([]string{"add", "-f", "--"}, tracked...)...)
	}
	return root
}

// writeGlobalExcludes writes git's default global excludes file.
func writeGlobalExcludes(t *testing.T, body string) {
	t.Helper()
	dir := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "git")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignore"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// boolPtr is the ignoreCase field of a case table.
func boolPtr(v bool) *bool { return &v }

type ignoreQuery struct {
	path string
	dir  bool
	want bool
}

func TestIgnoredRules(t *testing.T) {
	cases := []struct {
		name       string
		files      map[string]string
		tracked    []string
		ignoreCase *bool
		global     string
		queries    []ignoreQuery
	}{
		{
			name:  "basename matches at any depth",
			files: map[string]string{".gitignore": "*.log\n", "a.log": "x", "sub/b.log": "x", "sub/c.txt": "x"},
			queries: []ignoreQuery{
				{path: "a.log", want: true},
				{path: "sub/b.log", want: true},
				{path: "sub/c.txt", want: false},
			},
		},
		{
			name:  "negation undoes an earlier match",
			files: map[string]string{".gitignore": "*.log\n!keep.log\n", "keep.log": "x", "a.log": "x"},
			queries: []ignoreQuery{
				{path: "keep.log", want: false},
				{path: "a.log", want: true},
			},
		},
		{
			name:  "a leading slash anchors to the ignore file's directory",
			files: map[string]string{".gitignore": "/anchored.txt\n", "anchored.txt": "x", "sub/anchored.txt": "x"},
			queries: []ignoreQuery{
				{path: "anchored.txt", want: true},
				{path: "sub/anchored.txt", want: false},
			},
		},
		{
			name:  "a slash in the middle anchors too",
			files: map[string]string{".gitignore": "doc/frotz\n", "doc/frotz": "x", "x/doc/frotz": "x"},
			queries: []ignoreQuery{
				{path: "doc/frotz", want: true},
				{path: "x/doc/frotz", want: false},
			},
		},
		{
			name:  "a trailing slash names a directory and everything inside",
			files: map[string]string{".gitignore": "build/\n", "build/x.txt": "x"},
			queries: []ignoreQuery{
				{path: "build", dir: true, want: true},
				{path: "build/x.txt", want: true},
			},
		},
		{
			name:  "leading double star matches at any depth",
			files: map[string]string{".gitignore": "**/foo\n", "foo": "x", "a/foo": "x", "a/b/foo": "x"},
			queries: []ignoreQuery{
				{path: "foo", want: true},
				{path: "a/foo", want: true},
				{path: "a/b/foo", want: true},
			},
		},
		{
			name:  "trailing double star matches the directory and everything inside",
			files: map[string]string{".gitignore": "abc/**\n", "abc/x/one": "x", "abc/x/two": "x"},
			queries: []ignoreQuery{
				{path: "abc", dir: true, want: true},
				{path: "abc/x", dir: true, want: true},
				{path: "abc/x/one", want: true},
			},
		},
		{
			name:  "a negation cannot re-include below an ignored directory",
			files: map[string]string{".gitignore": "logs/\n!logs/keep.log\n", "logs/keep.log": "x"},
			queries: []ignoreQuery{
				{path: "logs", dir: true, want: true},
				{path: "logs/keep.log", want: true},
			},
		},
		{
			name: "a nested gitignore only governs its directory",
			files: map[string]string{
				".gitignore": "", "sub/.gitignore": "secret.txt\n",
				"sub/secret.txt": "x", "secret.txt": "x",
			},
			queries: []ignoreQuery{
				{path: "sub/secret.txt", want: true},
				{path: "secret.txt", want: false},
			},
		},
		{
			name:    "a tracked file is never ignored",
			files:   map[string]string{".gitignore": "*.log\n", "keep.log": "x"},
			tracked: []string{"keep.log"},
			queries: []ignoreQuery{
				{path: "keep.log", want: false},
				{path: "other.log", want: true},
			},
		},
		{
			name:  "escaped markers are literal",
			files: map[string]string{".gitignore": "\\!important.txt\n\\#hash.txt\n", "!important.txt": "x", "#hash.txt": "x"},
			queries: []ignoreQuery{
				{path: "!important.txt", want: true},
				{path: "#hash.txt", want: true},
			},
		},
		{
			name:  "unescaped trailing spaces do not count",
			files: map[string]string{".gitignore": "foo.txt   \n", "foo.txt": "x"},
			queries: []ignoreQuery{
				{path: "foo.txt", want: true},
			},
		},
		{
			name:       "core.ignorecase true folds the pattern",
			files:      map[string]string{".gitignore": "*.LOG\n", "foo.log": "x"},
			ignoreCase: boolPtr(true),
			queries: []ignoreQuery{
				{path: "foo.log", want: true},
			},
		},
		{
			name:       "core.ignorecase false keeps the case",
			files:      map[string]string{".gitignore": "*.LOG\n", "foo.log": "x"},
			ignoreCase: boolPtr(false),
			queries: []ignoreQuery{
				{path: "foo.log", want: false},
			},
		},
		{
			name:  "info/exclude is read",
			files: map[string]string{".git/info/exclude": "excluded.txt\n", "excluded.txt": "x"},
			queries: []ignoreQuery{
				{path: "excluded.txt", want: true},
			},
		},
		{
			name:   "the global excludes file is read",
			files:  map[string]string{"a.tmp": "x", "b.txt": "x"},
			global: "*.tmp\n",
			queries: []ignoreQuery{
				{path: "a.tmp", want: true},
				{path: "b.txt", want: false},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := newRepo(t, c.files, c.tracked, c.ignoreCase)
			if c.global != "" {
				writeGlobalExcludes(t, c.global)
			}
			rules, _, err := Load(root)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if rules == nil {
				t.Fatal("Load returned no rules for a repository")
			}
			for _, q := range c.queries {
				if got := rules.Ignored(q.path, q.dir); got != q.want {
					t.Errorf("Ignored(%q, dir=%v) = %v, want %v", q.path, q.dir, got, q.want)
				}
			}
		})
	}
}

// TestIgnoredMatchesGitCheckIgnore compares every queried path with git's own
// answer, the oracle the design asks for.
func TestIgnoredMatchesGitCheckIgnore(t *testing.T) {
	files := map[string]string{
		".gitignore":         "*.log\n!keep.log\nbuild/\n/anchored.txt\n**/generated/**\n\\!important.txt\nsub/*.tmp\n",
		"a.log":              "x",
		"keep.log":           "x",
		"sub/a.log":          "x",
		"anchored.txt":       "x",
		"sub/anchored.txt":   "x",
		"build/x.txt":        "x",
		"build/tracked.txt":  "x",
		"src/generated/c.go": "x",
		"!important.txt":     "x",
		"sub/secret.tmp":     "x",
		"sub/.gitignore":     "secret.txt\n",
		"sub/secret.txt":     "x",
		"plain.txt":          "x",
		"src/main.go":        "x",
		".git/info/exclude":  "info-only.txt\n",
		"info-only.txt":      "x",
	}
	root := newRepo(t, files, []string{"keep.log", "build/tracked.txt"}, boolPtr(true))
	rules, _, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var paths []string
	for rel := range files {
		if strings.HasPrefix(rel, ".git/") {
			continue
		}
		paths = append(paths, rel)
	}
	for _, dir := range []string{"build", "sub", "src", "src/generated"} {
		paths = append(paths, dir)
	}
	sort.Strings(paths)

	gitIgnored := map[string]bool{}
	var questions []string
	for _, rel := range paths {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil && info.IsDir() {
			// git treats a path as a directory only when it ends with a slash.
			questions = append(questions, rel+"/")
			continue
		}
		questions = append(questions, rel)
	}
	out, err := gitRunInput(t, root, strings.Join(questions, "\n")+"\n", "check-ignore", "--stdin")
	if err != nil {
		t.Fatalf("git check-ignore: %v\n%s", err, out)
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line != "" {
			gitIgnored[strings.TrimSuffix(line, "/")] = true
		}
	}
	for _, rel := range paths {
		dir := false
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			dir = info.IsDir()
		}
		if dir && rules.HasTrackedUnder(rel) {
			// git answers "not ignored" for a directory that still holds a
			// tracked file, even when a pattern names the directory; the walk
			// descends there for the tracked file and prunes nothing.
			continue
		}
		if got := rules.Ignored(rel, dir); got != gitIgnored[rel] {
			t.Errorf("Ignored(%q, dir=%v) = %v, git says %v", rel, dir, got, gitIgnored[rel])
		}
	}
}

// TestWalkMatchesGitFileList compares the search corpus with the list git
// itself considers searchable: cached plus other, standard exclusions.
func TestWalkMatchesGitFileList(t *testing.T) {
	files := map[string]string{
		".gitignore":       "*.log\n!keep.log\nbuild/\nsub/*.tmp\n",
		"a.log":            "x",
		"keep.log":         "x",
		"plain.txt":        "x",
		"sub/a.log":        "x",
		"sub/secret.tmp":   "x",
		"sub/keep.tmp":     "x",
		"build/x.txt":      "x",
		"build/tracked.go": "x",
		"src/main.go":      "x",
	}
	root := newRepo(t, files, []string{"keep.log", "build/tracked.go"}, boolPtr(false))
	rules, _, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	listing, err := Walk(root, rules)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	var ours []string
	for _, e := range listing.Entries {
		if e.Ignored {
			continue
		}
		ours = append(ours, e.Rel)
	}
	out := gitRun(t, root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	var want []string
	for _, rel := range strings.Split(out, "\x00") {
		if rel != "" {
			want = append(want, rel)
		}
	}
	sort.Strings(ours)
	sort.Strings(want)
	if strings.Join(ours, "\n") != strings.Join(want, "\n") {
		t.Fatalf("search corpus differs from git:\ngot  %v\nwant %v", ours, want)
	}
}

// TestLoadOutsideRepository reports no rules and no error; an unversioned
// directory is searched without ignore evaluation.
func TestLoadOutsideRepository(t *testing.T) {
	isolateGit(t)
	rules, sources, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rules != nil || sources != nil {
		t.Fatalf("Load outside a repository = (%v, %v), want (nil, nil)", rules, sources)
	}
}

// TestRulesMetadata pins the accessors the corpus record uses.
func TestRulesMetadata(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore":        "*.log\n",
		"keep.log":          "x",
		".git/info/exclude": "info.txt\n",
	}, []string{"keep.log"}, boolPtr(true))
	rules, sources, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rules.RepoTop() == "" || !rules.IgnoreCase() {
		t.Fatalf("metadata = top %q, ignorecase %v", rules.RepoTop(), rules.IgnoreCase())
	}
	if got := rules.TrackedPaths(); len(got) != 1 || got[0] != "keep.log" {
		t.Fatalf("TrackedPaths = %v", got)
	}
	if !rules.Tracked("keep.log") || rules.Tracked("other.log") {
		t.Fatal("Tracked disagrees with TrackedPaths")
	}
	if !rules.HasTrackedUnder("") || !rules.HasTrackedUnder(".") || rules.HasTrackedUnder("sub") {
		t.Fatal("HasTrackedUnder disagrees with the tracked set")
	}
	if rules.Prefix() != "" {
		t.Fatalf("Prefix = %q, want empty", rules.Prefix())
	}
	index, exclude, _ := rules.GitPaths()
	if index == "" || exclude == "" {
		t.Fatalf("GitPaths = (%q, %q)", index, exclude)
	}
	if path, size, mtime := rules.IndexStamp(); path == "" || size == 0 || mtime == 0 {
		t.Fatalf("IndexStamp = (%q, %d, %d)", path, size, mtime)
	}
	if len(sources) == 0 {
		t.Fatal("Load returned no sources")
	}
	if len(rules.Sources()) < len(sources) {
		t.Fatalf("Sources() = %d entries, want at least %d", len(rules.Sources()), len(sources))
	}
	// The .gitignore below the root is read lazily, so it joins the list only
	// after a path in its directory was evaluated.
	rules.Ignored("a.log", false)
	found := false
	for _, src := range rules.Sources() {
		if src.Rel == ".gitignore" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Sources() after a lookup = %v, want .gitignore", rules.Sources())
	}
}

// TestRulesInSubdirectory pins the prefix mapping: a walk of a subdirectory
// evaluates the same repository-relative paths.
func TestRulesInSubdirectory(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore":   "*.log\n",
		"sub/keep.txt": "x",
		"sub/drop.log": "x",
	}, nil, boolPtr(false))
	rules, _, err := Load(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if rules.Prefix() != "sub" {
		t.Fatalf("Prefix = %q, want sub", rules.Prefix())
	}
	if !rules.Ignored("drop.log", false) {
		t.Fatal("drop.log is not ignored")
	}
	if rules.Ignored("keep.txt", false) {
		t.Fatal("keep.txt is ignored")
	}
}

// TestIgnorecaseMatchesGit is the measurement the design asks for: with
// core.ignorecase true, a pattern with upper case matches a lower case name in
// git, and therefore here.
func TestIgnorecaseMatchesGit(t *testing.T) {
	root := newRepo(t, map[string]string{
		".gitignore": "*.LOG\n",
		"foo.log":    "x",
	}, nil, boolPtr(true))
	rules, _, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rel := "foo.log"
	out, err := gitRunInput(t, root, rel+"\n", "check-ignore", "--stdin")
	if err != nil {
		t.Fatalf("git check-ignore: %v\n%s", err, out)
	}
	want := strings.TrimSpace(out) == rel
	if got := rules.Ignored(rel, false); got != want {
		t.Fatalf("Ignored(%q) = %v, git says %v", rel, got, want)
	}
	if !want {
		t.Fatal("git did not ignore the file, so the oracle proves nothing")
	}
}
