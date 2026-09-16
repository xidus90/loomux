package pytext

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// NFC is `unicodedata.normalize("NFC", s)`. Measured over every code point
// against Python 3.14: x/text answers the same for all of them, both in NFC
// and NFD, though its tables are Unicode 17 and Python's 16.
func NFC(s string) string {
	return norm.NFC.String(s)
}

// CaseFold is `s.casefold()`: full case folding, so `ß` becomes `ss`.
//
// x/text's Fold folds Cherokee towards its small letters (U+13A0 to
// U+AB70, U+13F0 to U+13F8), where CaseFolding.txt and Python fold towards
// the capitals -- 86 code points, measured. The capitals are put back
// after folding. The 28 other differences are letters Unicode 17 added,
// which Python 3.14 does not know.
func CaseFold(s string) string {
	return strings.Map(cherokeeCapital, cases.Fold().String(s))
}

// cherokeeCapital maps a small Cherokee letter to its capital and leaves
// every other rune alone.
func cherokeeCapital(r rune) rune {
	switch {
	case r >= 0xab70 && r <= 0xabbf:
		return r - 0xab70 + 0x13a0
	case r >= 0x13f8 && r <= 0x13fd:
		return r - 0x13f8 + 0x13f0
	}
	return r
}
