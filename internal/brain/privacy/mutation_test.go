package privacy_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/privacy"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for. Every expectation below was read from the Python reference or
// measured against it, never derived from the Go code that has to satisfy
// it.
//
// The measurements are `PurePosixPath(<pfad>).full_match(<muster>)` under the
// reference's own interpreter, Python 3.14.7 -- the call `MatchesGlobs`
// mirrors.

func matches(t *testing.T, pattern, relative string) bool {
	t.Helper()
	return privacy.MatchesGlobs([]string{pattern}, relative)
}

func TestATwoStarPartInTheMiddleStillNeedsASeparator(t *testing.T) {
	// Measured: `full_match('**/x')` answers False for `ax` and True for
	// `a/x`. A `**` that is not the last part stands for whole segments,
	// separator and all; only a trailing `**` swallows the rest of the
	// path.
	if matches(t, "**/x", "ax") {
		t.Fatal("**/x must not match ax: the ** is not the last part")
	}
	if !matches(t, "**/x", "a/x") {
		t.Fatal("**/x must match a/x")
	}
}

func TestAnUnclosedBracketAtTheVeryEndIsALiteral(t *testing.T) {
	// Measured: `PurePosixPath('a[').full_match('a[')` answers True. A `[`
	// that opens nothing is the character itself -- and the search for its
	// closing bracket must stop at the end of the segment rather than read
	// one rune past it.
	if !matches(t, "a[", "a[") {
		t.Fatal("a[ must match a[")
	}
}

func TestTheHyphenRightAfterANegationIsNotARangeSeparator(t *testing.T) {
	// Measured: `PurePosixPath('a').full_match('[!-a-0]')` answers True.
	// The `-` behind the `!` is a literal, so the ranges of this class are
	// `a-0` alone -- empty, because `a` stands above `0`, and dropped. What
	// is left forbids the hyphen and nothing else.
	if !matches(t, "[!-a-0]", "a") {
		t.Fatal("[!-a-0] must match a: the hyphen behind the ! opens no range")
	}
}

func TestARangeOfOneCharacterIsNotEmpty(t *testing.T) {
	// Measured: `PurePosixPath('a').full_match('[a-a]')` answers True. A
	// range is empty when its first character stands above its last, not
	// when the two are the same.
	if !matches(t, "[a-a]", "a") {
		t.Fatal("[a-a] must match a")
	}
}

func TestARangeIsFoundWhereItStandsNotAtTheFirstCharacter(t *testing.T) {
	// Measured: `PurePosixPath(p).full_match('[ab-d]')` answers True for
	// `a`, `b` and `d`. The class is the literal `a` and the range `b-d`;
	// a search that took the first character it looked at for the range
	// separator would read `a` and `-d` instead and drop the `a`.
	for _, p := range []string{"a", "b", "d"} {
		if !matches(t, "[ab-d]", p) {
			t.Fatalf("[ab-d] must match %q", p)
		}
	}
}
