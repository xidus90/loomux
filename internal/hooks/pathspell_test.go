package hooks

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// mkfile makes a file under root, and the folders above it.
func mkfile(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUnfoldBracesExpandsLikeBash(t *testing.T) {
	for word, want := range map[string][]string{
		"a/{b,c}/d":       {"a/b/d", "a/c/d"},
		"{a,b}{1,2}":      {"a1", "a2", "b1", "b2"},
		"x/{a,{b,c}}":     {"x/a", "x/b", "x/c"},
		"r{1..3}":         {"r1", "r2", "r3"},
		"r{3..1}":         {"r3", "r2", "r1"},
		"r{01..03}":       {"r01", "r02", "r03"},
		"r{1..9..4}":      {"r1", "r5", "r9"},
		"r{1..5..-2}":     {"r1", "r3", "r5"},
		"{a..c}":          {"a", "b", "c"},
		"${X}/{a,b}":      {"${X}/a", "${X}/b"},
		"{x}":             {"{x}"},
		"{}":              {"{}"},
		"open{a,b":        {"open{a,b"},
		"{a{b,c}":         {"{ab", "{ac"},
		"plain":           {"plain"},
		".loomux/{a,b}/x": {".loomux/a/x", ".loomux/b/x"},
	} {
		got, ok := unfoldBraces(word)
		if !ok || !slices.Equal(got, want) {
			t.Errorf("unfoldBraces(%q) = %q, %v; want %q", word, got, ok, want)
		}
	}
	if got, ok := unfoldBraces("{1..64}"); !ok || len(got) != 64 {
		t.Fatalf("64 variants: %d, %v", len(got), ok)
	}
	for _, word := range []string{"{1..65}", "{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}{a,b}"} {
		if _, ok := unfoldBraces(word); ok {
			t.Errorf("%q unfolds past the limit", word)
		}
	}
	for _, word := range []string{"{1..x}", "{a..bb}", "{1..3..0}", "{1..3..x}", "{1..2..3..4}"} {
		if got, ok := unfoldBraces(word); !ok || !slices.Equal(got, []string{word}) {
			t.Errorf("%q is no range and stays: %q, %v", word, got, ok)
		}
	}
}

func TestWithoutStreamCutsAnNTFSStream(t *testing.T) {
	for p, want := range map[string]string{
		".loomux/config.toml:backup":   ".loomux/config.toml",
		"C:/x/config.toml:s:$DATA":     "C:/x/config.toml",
		"C:/x/config.toml":             "C:/x/config.toml",
		"plain":                        "plain",
		"//?/C:/x/config.toml:s":       "//?/C:/x/config.toml",
		`\\?\C:\x\config.toml:s:$DATA`: `\\?\C:\x\config.toml`,
		`\\.\C:\x:y`:                   `\\.\C:\x`,
	} {
		if got := withoutStream(p); got != want {
			t.Errorf("withoutStream(%q) = %q, want %q", p, got, want)
		}
	}
}

// A glob is matched against the disk as bash matches it: a leading * or ?
// skips names that begin with a dot, and a glob that matches nothing stays.
func TestExpandGlobMatchesTheDiskLikeBash(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, ".loomux/state/runs/0001.jsonl")
	mkfile(t, root, "build/a.txt")
	mkfile(t, root, ".hidden/x")
	rel := func(ps []string) []string {
		var out []string
		for _, p := range ps {
			out = append(out, relativePath(p, root))
		}
		slices.Sort(out)
		return out
	}
	if got := rel(expandGlob(root, "*")); !slices.Equal(got, []string{"build"}) {
		t.Errorf("* = %q, want only build", got)
	}
	if got := rel(expandGlob(root, ".loomux/sta*/runs")); !slices.Equal(got, []string{".loomux/state/runs"}) {
		t.Errorf("sta* = %q", got)
	}
	if got := rel(expandGlob(root, ".*")); !slices.Contains(got, ".loomux") || !slices.Contains(got, ".hidden") {
		t.Errorf(".* = %q, want the dot folders", got)
	}
	if got := expandGlob(root, "nothing/*"); !slices.Equal(got, []string{"nothing/*"}) {
		t.Errorf("a glob without a match = %q, want it as written", got)
	}
	if got := expandGlob(root, "bad/["); !slices.Equal(got, []string{"bad/["}) {
		t.Errorf("a glob filepath.Match cannot read = %q, want it as written", got)
	}
	if got := expandGlob(root, filepath.Join(root, "build", "*")); !slices.Equal(got, []string{filepath.Join(root, "build", "a.txt")}) {
		t.Errorf("an absolute glob = %q", got)
	}
	if got := expandGlob(root, "plain/x"); !slices.Equal(got, []string{"plain/x"}) {
		t.Errorf("no glob = %q", got)
	}
}

func TestSpellingsAreRelativeToTheRoot(t *testing.T) {
	root := t.TempDir()
	got, err := spellings(root, "./.loomux/flows/{example,zz}/x")
	if err != nil || !slices.Equal(got, []string{".loomux/flows/example/x", ".loomux/flows/zz/x"}) {
		t.Fatalf("braces: %q, %v", got, err)
	}
	got, err = spellings(root, filepath.Join(root, ".loomux", "config.toml")+":backup")
	if err != nil || !slices.Equal(got, []string{".loomux/config.toml"}) {
		t.Fatalf("stream: %q, %v", got, err)
	}
	_, err = spellings(root, "x{1..65}")
	if err == nil || !strings.Contains(err.Error(), strconv.Itoa(maxBraceVariants)) {
		t.Fatalf("past the limit: %v", err)
	}
}

func TestLettersAreASCIILettersOnly(t *testing.T) {
	if !letters("sSLo") || letters("") || letters("a1") {
		t.Fatal("letters")
	}
}
