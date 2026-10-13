package pathkey

import (
	"path"
	"strings"
	"testing"
	"time"
)

// A leading **/ stands for any directory, the root included, and is anchored
// at an element: my.loomux is no .loomux.
func TestGlobReadsALeadingDoubleStarAsAnyDirectory(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"**/.loomux/config.toml", ".loomux/config.toml", true},
		{"**/.loomux/config.toml", "../sibling/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "C:/repo/.loomux/config.toml", true},
		{"**/.loomux/config.toml", "my.loomux/config.toml", false},
		{"**/.loomux/config.toml", ".loomux/config.toml.bak", false},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs", true},
		{"**/.loomux/state/runs/**", "x/.loomux/state/runs/0001.jsonl", true},
		{"**/.loomux/state/runs/**", ".loomux/state/runsx/0001.jsonl", false},
	} {
		got, err := Glob(row.pattern, row.name)
		if err != nil || got != row.want {
			t.Errorf("Glob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
	if _, err := Glob("**/foo/*[x", "a/foo/b"); err == nil {
		t.Fatal("a bad class behind **/ must still be an error")
	}
}

// A pattern with a slash ending in /** takes the directory and all below it;
// one without a slash looks at the file name in any directory.
func TestGlobOfDirectoriesAndNames(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"docs/**", "docs", true},
		{"docs/**", "docs/de/a.md", true},
		{"docs/**", "docsx/a.md", false},
		{"*.md", "docs/de/ä.md", true},
		{"*.md", "a.mdx", false},
		{"README.md", "sub/README.md", true},
	} {
		if got, err := Glob(row.pattern, row.name); err != nil || got != row.want {
			t.Errorf("Glob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
}

// An ending /** takes everything below the directory its prefix names, and a
// wildcard in the prefix does not cut that short; the prefix itself stays as
// narrow as it reads, a `*` within one element.
func TestGlobOfAnEndingDoubleStarAfterAWildcard(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"services/*/secrets/**", "services/api/secrets", true},
		{"services/*/secrets/**", "services/api/secrets/key.txt", true},
		{"services/*/secrets/**", "services/api/secrets/prod/key.txt", true},
		{"services/*/secrets/**", "services/api/other/key.txt", false},
		{"services/*/secrets/**", "services/secrets/key.txt", false},
		{"a/*/b/**", "a/x/b/c/d/e", true},
		{"a/*/b/**", "a/x/y/b/c", false},
		{"a*/**", "ab/c/d", true},
		{"a*/**", "ba/c/d", false},
		{"**/build/*/keys/**", "x/build/a/keys/k", true},
		{"**/build/*/keys/**", "x/build/a/keys/d/k", true},
		{"**/build/*/keys/**", "x/build/a/b/keys/k", false},
		{"a[b]/**", "a[b]/c", true},
	} {
		if got, err := Glob(row.pattern, row.name); err != nil || got != row.want {
			t.Errorf("Glob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
	if _, err := Glob("foo/*[x/**", "foo/a/b"); err == nil {
		t.Fatal("a bad class in front of an ending /** must still be an error")
	}
}

// A character class is the one part of path.Match that can take a `/`; the
// prefix of an ending /** keeps that reach, or such a rule would protect less
// than it did.
func TestGlobOfAnEndingDoubleStarKeepsAClassThatTakesASlash(t *testing.T) {
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"a[^x]b/**", "a/b/c", true},
		{"a[^x]b/**", "a/x/b/c", false},
		{"a[^x]b[^x]c/**", "a/b/c/d", true},
		{"a[^x]b[^x]c/**", "a/b/c", true},
		{"a[^x]b[^x]c/**", "a/b", false},
		// A `/` written inside the class is no element of the prefix.
		{"vault[^/]/**", "vaultA/sub/key.txt", true},
		{"vault[^/]/**", "vault/sub/key.txt", false},
		{"**/vault[^/]/**", "x/vaultA/sub/key.txt", true},
		{"services/*[^/]/secrets/**", "services/api/secrets/prod/key.txt", true},
		{"services/*[^/]/secrets/**", "services/secrets/prod/key.txt", false},
		{"[a/]/**", "a/b/c", true},
	} {
		if got, err := Glob(row.pattern, row.name); err != nil || got != row.want {
			t.Errorf("Glob(%q, %q) = %v, %v; want %v", row.pattern, row.name, got, err, row.want)
		}
	}
}

// everyDirectory is the reading of an ending /** in its plain form, the prefix
// tried against the name and each directory above it, and says whether the
// prefix is one path.Match can read.
func everyDirectory(prefix, name string) (matched, readable bool) {
	for dir := name; ; {
		if dir == prefix {
			return true, true
		}
		ok, err := path.Match(prefix, dir)
		if err != nil {
			return false, false
		}
		if ok {
			return true, true
		}
		slash := strings.LastIndexByte(dir, '/')
		if slash < 0 {
			return false, true
		}
		dir = dir[:slash]
	}
}

// wordsUpTo is every string of at most longest letters out of alphabet.
func wordsUpTo(alphabet string, longest int) []string {
	words, last := []string{""}, []string{""}
	for range longest {
		var next []string
		for _, word := range last {
			for _, letter := range alphabet {
				next = append(next, word+string(letter))
			}
		}
		words, last = append(words, next...), next
	}
	return words
}

// Looking only at the directories that can match is a speed-up, not a new
// reading: for every short prefix over the characters that shape a pattern
// (classes, escapes, stars, slashes) and every short name, Glob answers as the
// plain reading does.
func TestGlobOfAnEndingDoubleStarAgreesWithTryingEveryDirectory(t *testing.T) {
	names := wordsUpTo("a/", 6)
	for _, prefix := range wordsUpTo(`a/*[]^\`, 5) {
		if strings.HasPrefix(prefix, "**/") {
			continue
		}
		for _, name := range names {
			want, readable := everyDirectory(prefix, name)
			if !readable {
				continue
			}
			if got, err := Glob(prefix+"/**", name); err != nil || got != want {
				t.Fatalf("Glob(%q, %q) = %v, %v; want %v", prefix+"/**", name, got, err, want)
			}
		}
	}
}

// The guard that reads a path with this has a host timeout to meet, and a call
// that outlasts it goes through: matching an ending /** is linear in the depth
// of the name, even behind a leading **/ that tries every tail of it.
func TestGlobOfAnEndingDoubleStarIsLinearInDepth(t *testing.T) {
	deep := strings.Repeat("a/", 20000)
	for _, row := range []struct {
		pattern, name string
		want          bool
	}{
		{"**/.loomux/state/hooks/**", deep + "x.txt", false},
		{"**/.loomux/state/hooks/**", deep + ".loomux/state/hooks/x", true},
		{"**/.loomux/state/runs/**", deep, false},
		{"a/*/b/**", deep, false},
	} {
		start := time.Now()
		got, err := Glob(row.pattern, row.name)
		if err != nil || got != row.want {
			t.Errorf("Glob(%q, %d-deep name) = %v, %v; want %v", row.pattern, strings.Count(row.name, "/"), got, err, row.want)
		}
		if took := time.Since(start); took > 500*time.Millisecond {
			t.Errorf("Glob(%q, %d-deep name) took %v; want linear in depth", row.pattern, strings.Count(row.name, "/"), took)
		}
	}
}
