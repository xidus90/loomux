package privacy_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
)

// Every answer below was measured against `brain.privacy.matches_globs((pattern,), relative)`
// on 2026-09-15 under Python 3.14.7: PurePosixPath.full_match over NFC and casefold, with
// the pattern translated by glob.translate and fnmatch._translate.
func TestMatchesGlobsAnswersLikePython(t *testing.T) {
	nfd := "cafe" + string(rune(0x0301))
	nfc := "caf" + string(rune(0x00e9))
	for _, tc := range []struct {
		pattern, relative string
		want              bool
	}{
		// ** as zero or more segments, * within one segment
		{"a/**/b", "a/b", true},
		{"**/x.md", "x.md", true},
		{"a/**", "a", false},
		{"review/**", "review", false},
		{"**", "a/b/c", true},
		{"**", "", true},
		{"**", ".", true},
		{".", "", true},
		{"*", "a/b", false},
		{"*", ".hidden", true},
		{"*.md", "dir/x.md", false},
		{"*.md", ".md", true},
		{"a/*", "a/b", true},
		{"a/*/c", "a/b/c", true},
		{"a/**/**/b", "a/x/y/b", true},
		{"**/secrets/**", "secrets", false},
		{"**/secrets/**", "x/secrets/y", true},
		{"a*b*c", "axxbyyc", true},
		{"***", "abc", true},
		{"?*", "", false},
		{"a*b", "a\nb", true},
		{"test?.txt", "test1.txt", true},
		{"?", "/", false},
		{"x[!/]y", "x/y", false},
		// both sides normalised like PurePosixPath; no backslash turns into a slash
		{"a//b/./c", "a/b/c", true},
		{"a/b/c", "a//b/./c", true},
		{"a/", "a", true},
		{"//a", "//a", true},
		{"/a", "a", false},
		{"secrets/**", "secrets\\key.txt", false},
		// literals are escaped
		{"a.md", "aXmd", false},
		{"a(b)", "a(b)", true},
		{"a+", "aa", false},
		// NFC and casefold on both sides
		{"ß.md", "SS.md", true},
		{"ẞ.md", "ss.md", true},
		{"ﬁle.md", "FILE.md", true},
		{"Σ.md", "ς.md", true},
		{"**/*.PEM", "k.pem", true},
		{nfd + "/*", nfc + "/x.md", true},
		{nfc + "/*", nfd + "/x.md", true},
		// bracket expressions as fnmatch._translate builds them
		{"[a-z]*.md", "x.md", true},
		{"[!a]*.md", "a.md", false},
		{"[]a]", "]", true},
		{"[!]a]", "b", true},
		{"[!]a]", "]", false},
		{"[a-]", "-", true},
		{"[z-a]x", "x", false},
		{"[z-a]x", "zx", false},
		{"[!z-a]", "q", true},
		{"[a&&b]", "&", true},
		{"[~]", "~", true},
		{"[|]", "|", true},
		{"[^a]", "^", true},
		{"[^a]", "b", false},
		{"[[:alpha:]]", "a", false},
		{"[[:alpha:]]", "[", false},
		{"[[:alpha:]]", ":]", true},
		{"[a[:alpha:]]", "a]", true},
		{"[a[:alpha:]]", "l]", true},
		{"[\\]", "\\", true},
		{"[\\-z]", "a", true},
		{"[abc", "[abc", true},
		{"[!", "[!", true},
		{"[!]", "[!]", true},
		{"[a-c-e]", "d", false},
		{"[a-c-e]", "-", true},
		{"[b-a-z]", "c", false},
		{"[b-a-z]", "-", true},
		{"[a--z]", "-", false},
		{"[a--z]", "z", true},
		{"[a--z]", "a", false},
		{"[!b-a]", "x", true},
		{"[ä-ö]", "é", true},
	} {
		if got := privacy.MatchesGlobs([]string{tc.pattern}, tc.relative); got != tc.want {
			t.Errorf("MatchesGlobs(%q, %q) = %v, want %v", tc.pattern, tc.relative, got, tc.want)
		}
	}
}

func TestMatchesGlobsAsksEveryPattern(t *testing.T) {
	if !privacy.MatchesGlobs([]string{"*.pem", "secrets/**"}, "secrets/a.md") {
		t.Error("the second pattern must be asked when the first does not match")
	}
	if privacy.MatchesGlobs(nil, "a.md") {
		t.Error("no pattern matches nothing")
	}
}

// Measured on 2026-09-22 under Python 3.14.7 with `PurePosixPath(relative)
// .full_match(pattern)` and nothing folded -- the call `_matches_any` makes in
// src/brain/walk.py for include, exclude and the review centre.
func TestMatchesGlobsUnfoldedAnswersLikeFullMatch(t *testing.T) {
	nfd := "Pru" + string(rune(0x0308)) + "fzentrum"
	for _, tc := range []struct {
		pattern, relative string
		want              bool
	}{
		{"95 Prüfzentrum/**", "95 Prüfzentrum/knowledge/c1/package.md", true},
		{"95 Prüfzentrum", "95 Prüfzentrum", true},
		{"docs/**/*.md", "docs/a.md", true},
		{"docs/**/*.md", "docs/sub/a.md", true},
		{"a/**/x.md", "a/x.md", true},
		// Nothing is folded: neither the case nor the normal form.
		{"private/**", "Private/x.md", false},
		{"Prüfzentrum/**", nfd + "/x.md", false},
	} {
		if got := privacy.MatchesGlobsUnfolded([]string{tc.pattern}, tc.relative); got != tc.want {
			t.Errorf("MatchesGlobsUnfolded(%q, %q) = %v, want %v", tc.pattern, tc.relative, got, tc.want)
		}
	}
	if !privacy.MatchesGlobs([]string{"private/**"}, "Private/x.md") {
		t.Error("MatchesGlobs keeps folding: the never list is matched case-insensitively")
	}
	if privacy.MatchesGlobsUnfolded(nil, "a.md") {
		t.Error("no pattern matches nothing")
	}
}
