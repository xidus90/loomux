package corpus

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestGlobSegment pins the fnmatch subset the ignore patterns use.
func TestGlobSegment(t *testing.T) {
	cases := []struct {
		pattern string
		name    string
		ci      bool
		want    bool
	}{
		{pattern: "*", name: "", want: true},
		{pattern: "*", name: "abc", want: true},
		{pattern: "*.go", name: "a.go", want: true},
		{pattern: "*.go", name: "a.py", want: false},
		{pattern: "a*c", name: "ab", want: false},
		{pattern: "a?c", name: "abc", want: true},
		{pattern: "a?c", name: "ac", want: false},
		{pattern: "?", name: "", want: false},
		{pattern: "[abc]x", name: "bx", want: true},
		{pattern: "[abc]x", name: "dx", want: false},
		{pattern: "[!abc]x", name: "dx", want: true},
		{pattern: "[!abc]x", name: "ax", want: false},
		{pattern: "[^abc]x", name: "dx", want: true},
		{pattern: "[^abc]x", name: "ax", want: false},
		{pattern: "[a-c]x", name: "bx", want: true},
		{pattern: "[a-c]x", name: "dx", want: false},
		{pattern: "[a-]x", name: "-x", want: true},
		{pattern: "[\\]]x", name: "]x", want: true},
		{pattern: "[\\a]x", name: "ax", want: true},
		{pattern: "[a-\\c]x", name: "bx", want: true},
		{pattern: "a[b", name: "a[b", want: true},
		{pattern: "a[b", name: "ab", want: false},
		{pattern: "[a]", name: "", want: false},
		{pattern: "a\\*b", name: "a*b", want: true},
		{pattern: "a\\*b", name: "axb", want: false},
		{pattern: "a\\b", name: "ax", want: false},
		{pattern: "a\\", name: "a\\", want: true},
		{pattern: "a\\", name: "a", want: false},
		{pattern: "ab", name: "abc", want: false},
		{pattern: "abc", name: "", want: false},
		{pattern: "a**b", name: "axxb", want: true},
		{pattern: "*.LOG", name: "foo.log", ci: true, want: true},
		{pattern: "*.LOG", name: "foo.log", want: false},
	}
	for _, c := range cases {
		if got := globSegment(c.pattern, c.name, c.ci); got != c.want {
			t.Errorf("globSegment(%q, %q, ci=%v) = %v, want %v", c.pattern, c.name, c.ci, got, c.want)
		}
	}
}

// TestMatchSegments pins the "**" rule: any number of whole segments,
// including none.
func TestMatchSegments(t *testing.T) {
	cases := []struct {
		pattern []string
		path    []string
		ci      bool
		want    bool
	}{
		{pattern: nil, path: nil, want: true},
		{pattern: nil, path: []string{"a"}, want: false},
		{pattern: []string{"**"}, path: nil, want: true},
		{pattern: []string{"**", "foo"}, path: []string{"foo"}, want: true},
		{pattern: []string{"**", "foo"}, path: []string{"x", "foo"}, want: true},
		{pattern: []string{"**", "foo"}, path: []string{"bar"}, want: false},
		{pattern: []string{"abc", "**"}, path: []string{"abc"}, want: true},
		{pattern: []string{"abc", "**"}, path: []string{"abc", "x"}, want: true},
		{pattern: []string{"a", "b"}, path: []string{"a"}, want: false},
		{pattern: []string{"a"}, path: []string{"a", "b"}, want: false},
		{pattern: []string{"a", "b"}, path: []string{"a", "c"}, want: false},
		{pattern: []string{"*.LOG"}, path: []string{"foo.log"}, ci: true, want: true},
	}
	for _, c := range cases {
		if got := matchSegments(c.pattern, c.path, c.ci); got != c.want {
			t.Errorf("matchSegments(%v, %v, ci=%v) = %v, want %v", c.pattern, c.path, c.ci, got, c.want)
		}
	}
}

// TestParseIgnore pins comments, blank lines, the two "nothing left" arms and
// the field mapping.
func TestParseIgnore(t *testing.T) {
	body := []byte("\n# comment\r\n   \n!neg\n!    \ndir/\n/\n//\nsub/file\n\\#hash\n")
	got := parseIgnore(body, "base")
	want := []pattern{
		{base: "base", negate: true, segs: []string{"neg"}},
		{base: "base", dirOnly: true, segs: []string{"dir"}},
		{base: "base", anchored: true, segs: []string{"sub", "file"}},
		{base: "base", segs: []string{"\\#hash"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseIgnore = %#v, want %#v", got, want)
	}
}

// TestParseStampsIgnoreCase pins the core.ignorecase stamping.
func TestParseStampsIgnoreCase(t *testing.T) {
	r := &Rules{ignoreCase: true}
	got := r.parse([]byte("*.LOG\n"), "dir")
	if len(got) != 1 || !got[0].ci {
		t.Fatalf("parse = %#v, want one pattern with ci", got)
	}
}

// TestTrimTrailingSpaces pins the escaped-trailing-space exception.
func TestTrimTrailingSpaces(t *testing.T) {
	cases := map[string]string{
		"foo   ":    "foo",
		"foo\\ ":    "foo\\ ",
		"foo\\\\ ":  "foo\\\\",
		"foo":       "foo",
		"   ":       "",
		"\\\\\\ ":   "\\\\\\ ",
		"trailing ": "trailing",
	}
	for in, want := range cases {
		if got := trimTrailingSpaces(in); got != want {
			t.Errorf("trimTrailingSpaces(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestPatternMatches pins the early arms of one pattern match.
func TestPatternMatches(t *testing.T) {
	if (pattern{dirOnly: true, segs: []string{"build"}}).matches("build", false) {
		t.Fatal("a directory-only pattern matched a file")
	}
	if (pattern{base: "sub", segs: []string{"x"}}).matches("other/x", false) {
		t.Fatal("a pattern matched outside its base")
	}
	if (pattern{base: "sub", segs: []string{"x"}}).matches("sub/", false) {
		t.Fatal("a pattern matched its base itself")
	}
	if !(pattern{base: "sub", segs: []string{"x"}}).matches("sub/x", false) {
		t.Fatal("a pattern did not match inside its base")
	}
}

// TestFullAndRelOf pins the path mapping, including the prefix and the
// outside-the-repository case.
func TestFullAndRelOf(t *testing.T) {
	r := &Rules{top: t.TempDir(), prefix: "sub"}
	cases := map[string]string{
		"":         "sub",
		".":        "sub",
		"./a/b":    "sub/a/b",
		"/a":       "sub/a",
		"sub2/a":   "sub/sub2/a",
		"../other": "sub/../other",
	}
	for in, want := range cases {
		if got := r.full(in); got != want {
			t.Errorf("full(%q) = %q, want %q", in, got, want)
		}
	}
	if got := (&Rules{top: r.top}).full("a"); got != "a" {
		t.Errorf("full without prefix = %q, want a", got)
	}
	if got := r.relOf(""); got != "" {
		t.Errorf("relOf(\"\") = %q, want empty", got)
	}
	outside := filepath.Join(os.TempDir(), "elsewhere", "file")
	if got := r.relOf(outside); got != filepath.ToSlash(outside) {
		t.Errorf("relOf(outside) = %q, want %q", got, filepath.ToSlash(outside))
	}
	inside := filepath.Join(r.top, ".git", "index")
	if got := r.relOf(inside); got != ".git/index" {
		t.Errorf("relOf(inside) = %q, want .git/index", got)
	}
}

// TestGitPathErrors pins that a path git cannot resolve is empty.
func TestGitPathErrors(t *testing.T) {
	isolateGit(t)
	r := &Rules{}
	if got := r.gitPath(t.TempDir(), "index"); got != "" {
		t.Fatalf("gitPath outside a repository = %q, want empty", got)
	}
}

// TestGlobalExcludes pins configured, XDG, home and no-home answers.
func TestGlobalExcludes(t *testing.T) {
	root := newRepo(t, map[string]string{"a.txt": "x"}, nil, nil)
	r := &Rules{}
	configured := filepath.Join(t.TempDir(), "excludes")
	gitRun(t, root, "config", "core.excludesFile", configured)
	if got := r.globalExcludes(root); got != configured {
		t.Fatalf("configured globalExcludes = %q, want %q", got, configured)
	}

	gitRun(t, root, "config", "--unset", "core.excludesFile")
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if got, want := r.globalExcludes(root), filepath.Join(xdg, "git", "ignore"); got != want {
		t.Fatalf("XDG globalExcludes = %q, want %q", got, want)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if got, want := r.globalExcludes(root), filepath.Join(home, ".config", "git", "ignore"); got != want {
		t.Fatalf("home globalExcludes = %q, want %q", got, want)
	}

	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	if got := r.globalExcludes(root); got != "" {
		t.Fatalf("globalExcludes without a home = %q, want empty", got)
	}
}

// TestLoadSourceEdges pins the empty path and the once-only rule.
func TestLoadSourceEdges(t *testing.T) {
	r := &Rules{sourceSeen: map[string]bool{}}
	if _, ok := r.loadSource("", "rel", "gitignore"); ok {
		t.Fatal("loadSource accepted an empty path")
	}
	file := filepath.Join(t.TempDir(), "ignore")
	if err := os.WriteFile(file, []byte("*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := r.loadSource(file, "rel", "gitignore"); !ok {
		t.Fatal("loadSource refused a readable file")
	}
	if _, ok := r.loadSource(file, "rel", "gitignore"); ok {
		t.Fatal("loadSource read the same file twice")
	}
}

// TestIgnoredRootEdges pins that the repository root itself is never ignored.
func TestIgnoredRootEdges(t *testing.T) {
	r := &Rules{tracked: map[string]bool{}, dirs: map[string][]pattern{}, loaded: map[string]bool{}, sourceSeen: map[string]bool{}}
	for _, q := range []struct {
		rel string
		dir bool
	}{{"", true}, {".", false}} {
		if r.Ignored(q.rel, q.dir) {
			t.Fatalf("Ignored(%q) = true, want false", q.rel)
		}
	}
}

// TestLoadFailsOnABrokenIndex pins the one git failure Load reports instead of
// silently treating every file as untracked.
func TestLoadFailsOnABrokenIndex(t *testing.T) {
	root := newRepo(t, map[string]string{"a.txt": "x"}, nil, nil)
	gitRun(t, root, "add", "a.txt")
	index := filepath.Join(root, ".git", "index")
	if err := os.Remove(index); err != nil {
		t.Fatalf("remove index: %v", err)
	}
	if err := os.Mkdir(index, 0o755); err != nil {
		t.Fatalf("replace index with a directory: %v", err)
	}
	if _, _, err := Load(root); err == nil {
		t.Fatal("Load accepted a broken index")
	}
}
