package pytext

import (
	"testing"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for. Every expectation below was read from the Python reference or
// measured against it, never derived from the Go code that has to satisfy
// it.

func TestAZeroInsideTheFractionStaysInTheFraction(t *testing.T) {
	// Measured: `datetime.fromisoformat('2026-09-13T19:51:12.105Z')`
	// answers a stamp with `microsecond` 105000 -- the `0` is a digit of
	// the fraction like any other, not its end.
	got, ok := ParseAwareIsoFormat("2026-09-13T19:51:12.105Z")
	if !ok {
		t.Fatal("a fraction with a zero in it must parse")
	}
	if got.Nanosecond() != 105000*1000 {
		t.Fatalf("microsecond = %d, want 105000", got.Nanosecond()/1000)
	}
}

func TestAnOffsetSecondOfFiftyNineIsStillAnOffset(t *testing.T) {
	// Measured: `datetime.fromisoformat('2026-09-13T19:51:12+01:00:59')`
	// answers `utcoffset()` 3659 s -- 59 is the last accepted second, not
	// the first refused one.
	got, ok := ParseAwareIsoFormat("2026-09-13T19:51:12+01:00:59")
	if !ok {
		t.Fatal("an offset second of 59 must parse")
	}
	_, off := got.Zone()
	if off != 3659 {
		t.Fatalf("offset = %d s, want 3659", off)
	}
}

func TestTheFirstYearOfTheCalendarParses(t *testing.T) {
	// Measured: `datetime.fromisoformat('0001-01-01T00:00:00Z')` answers
	// `0001-01-01T00:00:00+00:00`. Year 1 is the smallest Python takes,
	// not the smallest it refuses.
	if _, ok := ParseAwareIsoFormat("0001-01-01T00:00:00Z"); !ok {
		t.Fatal("year 1 must parse")
	}
}

func TestDecemberParses(t *testing.T) {
	// Measured: `datetime.fromisoformat('2026-12-13T19:51:12Z')` answers
	// `2026-12-13T19:51:12+00:00`. Month 12 is the last month, not one
	// past the year.
	got, ok := ParseAwareIsoFormat("2026-12-13T19:51:12Z")
	if !ok {
		t.Fatal("December must parse")
	}
	if got.Month() != 12 {
		t.Fatalf("month = %d, want 12", got.Month())
	}
}

func TestANonDigitWhereTheShapeWantsADigitIsRefused(t *testing.T) {
	// Measured: `datetime.fromisoformat` raises
	// `ValueError: Invalid isoformat string` for both
	// '2026-09-13T19:51:1aZ' and '!026-09-13T19:51:12Z'. Both sides of
	// the digit range are refused: 'a' stands above '9', '!' below '0'.
	// The second one sits in the year, the one field with no range of its
	// own: further down a bad character in the second is caught a second
	// time by `second > 59`, and that would hide the shape.
	for _, s := range []string{"2026-09-13T19:51:1aZ", "!026-09-13T19:51:12Z"} {
		if _, ok := ParseAwareIsoFormat(s); ok {
			t.Fatalf("%q must not parse", s)
		}
	}
}

func TestTheFixedCharactersOfTheShapeMustMatch(t *testing.T) {
	// Measured: `datetime.fromisoformat('2026x09-13T19:51:12Z')` raises
	// `ValueError: Invalid isoformat string`. The `-` of the date is part
	// of the form, not decoration around the digits.
	if _, ok := ParseAwareIsoFormat("2026x09-13T19:51:12Z"); ok {
		t.Fatal("a stamp with the wrong date separator must not parse")
	}
}

func TestACarriageReturnAtTheVeryEndOpensNoEmptyLine(t *testing.T) {
	// Measured: `'a\r'.splitlines()` answers `['a']`, and
	// `'a\r\nb'.splitlines()` answers `['a', 'b']`. The lookahead for the
	// `\n` of a `\r\n` pair must stop at the end of the text.
	if got := SplitLines("a\r"); len(got) != 1 || got[0] != "a" {
		t.Fatalf("SplitLines(%q) = %q, want [a]", "a\r", got)
	}
	if got := SplitLines("a\r\nb"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("SplitLines(%q) = %q, want [a b]", "a\r\nb", got)
	}
}

func TestASixPartUNCDriveGainsARoot(t *testing.T) {
	// Measured: str(PureWindowsPath(r'\\?\UNC\server\share')) answers the
	// same path with one more separator behind it -- that separator is the
	// root a six part UNC drive gains, and ntpath.splitroot of the path
	// answers the whole path as the drive with no root of its own.
	const p = `\\?\UNC\server\share`
	if got := PathString("windows", p); got != p+`\` {
		t.Fatalf("PathString(windows, %q) = %q, want %q", p, got, p+`\`)
	}
}

func TestATailThatReadsAsADriveIsPrefixedOnlyWithoutADrive(t *testing.T) {
	// Measured: str(PureWindowsPath(r'C:\x:y')) answers the path
	// unchanged -- the . that pathlib puts before a first part reading as
	// a drive is for a path that has no drive and no root of its own.
	const p = `C:\x:y`
	if got := PathString("windows", p); got != p {
		t.Fatalf("PathString(windows, %q) = %q, want %q", p, got, p)
	}
}

func TestAPlainUNCPathIsNotReadAsAnExtendedOne(t *testing.T) {
	// Measured: ntpath.splitroot(r'\\a\b\c\d') answers drive
	// \\a\b, root \ and rest c\d, and PureWindowsPath answers the path
	// unchanged: the server and the share are the first two parts behind
	// the two slashes, not whatever stands eight characters in.
	const p = `\\a\b\c\d`
	if got := PathString("windows", p); got != p {
		t.Fatalf("PathString(windows, %q) = %q, want %q", p, got, p)
	}
	drive, root, rel := windowsSplitRoot(p)
	if drive != `\\a\b` || root != `\` || rel != `c\d` {
		t.Fatalf("windowsSplitRoot(%q) = (%q, %q, %q), want (%q, %q, %q)", p, drive, root, rel, `\\a\b`, `\`, `c\d`)
	}
}

func TestAnExtendedUNCPrefixWithNothingBehindItIsAllDrive(t *testing.T) {
	// Measured: ntpath.splitroot of the eight character prefix
	// \\?\UNC\ answers that prefix as the drive with an empty root and
	// an empty rest -- the prefix itself is the drive.
	const p = `\\?\UNC\`
	drive, root, rel := windowsSplitRoot(p)
	if drive != p || root != "" || rel != "" {
		t.Fatalf("windowsSplitRoot(%q) = (%q, %q, %q), want (%q, %q, %q)", p, drive, root, rel, p, "", "")
	}
}

func TestAThirdSlashBelongsToTheShareNotToTheSearch(t *testing.T) {
	// Measured: ntpath.splitroot of \\\share\x answers drive \\\share,
	// root \ and rest x: the separator found right at the start of the
	// search is a separator like any other.
	const p = `\\\share\x`
	drive, root, rel := windowsSplitRoot(p)
	if drive != `\\\share` || root != `\` || rel != "x" {
		t.Fatalf("windowsSplitRoot(%q) = (%q, %q, %q), want (%q, %q, %q)", p, drive, root, rel, `\\\share`, `\`, "x")
	}
}

func TestAnEmptyShareStillEndsTheDrive(t *testing.T) {
	// Measured: ntpath.splitroot of \\a\\b answers drive \\a\, root
	// \ and rest b: the second separator stands right behind the first,
	// and the drive ends there all the same.
	const p = `\\a\\b`
	drive, root, rel := windowsSplitRoot(p)
	if drive != `\\a\` || root != `\` || rel != "b" {
		t.Fatalf("windowsSplitRoot(%q) = (%q, %q, %q), want (%q, %q, %q)", p, drive, root, rel, `\\a\`, `\`, "b")
	}
}
