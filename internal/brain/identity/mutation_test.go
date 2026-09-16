package identity

import (
	"testing"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for. Every expectation below was read from the Python reference or
// measured against it, never derived from the Go code that has to satisfy
// it.

func TestTheOuterDigitsAreRevisionsLikeAnyOther(t *testing.T) {
	// Measured: `'0'.isdigit(), int('0')` answers `(True, 0)` and
	// `'9'.isdigit(), int('9')` answers `(True, 9)`. Both ends of the
	// digit range belong to it; `parseRevision` mirrors that test.
	for _, c := range []struct {
		in   string
		want int
	}{{"0", 0}, {"9", 9}} {
		got, ok := parseRevision(c.in)
		if !ok || got != c.want {
			t.Fatalf("parseRevision(%q) = (%d, %v), want (%d, true)", c.in, got, ok, c.want)
		}
	}
}

func TestAPathThatStayedIsNotAlsoAPathThatWent(t *testing.T) {
	// `match_renames` (src/brain/identity.py:93) reads
	// `gone = {path: identity for path, identity in previous.items() if
	// path not in current}`: a path that is in both maps has not
	// vanished, so it lends its hash to nobody. The second file of the
	// same content is a copy, not a rename.
	previous := map[string]Identity{
		"kept.md": {DocID: "d1", Relative: "kept.md", ContentHash: "h", Revision: 1},
	}
	current := map[string]string{"kept.md": "h", "copy.md": "h"}
	if got := MatchRenames(previous, current); len(got) != 0 {
		t.Fatalf("MatchRenames = %v, want no rename: kept.md never vanished", got)
	}
}

func TestAPathThatWasAlreadyThereIsNotFresh(t *testing.T) {
	// The other half of the same rule, `match_renames`
	// (src/brain/identity.py:94): `fresh = {path: digest for path, digest
	// in current.items() if path not in previous}`. A path the previous
	// run already knew is not an arrival, so a vanished file of the same
	// content may not hand its doc id to it.
	previous := map[string]Identity{
		"gone.md": {DocID: "d1", Relative: "gone.md", ContentHash: "h", Revision: 1},
		"old.md":  {DocID: "d2", Relative: "old.md", ContentHash: "h", Revision: 1},
	}
	current := map[string]string{"old.md": "h"}
	if got := MatchRenames(previous, current); len(got) != 0 {
		t.Fatalf("MatchRenames = %v, want no rename: old.md was there before", got)
	}
}
