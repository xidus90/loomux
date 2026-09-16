package pytext

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// newInUnicode17 are the code points Go's `unicode.IsPrint` calls printable
// and Python 3.14.7 does not: all 4803 are category Cn in Python's Unicode
// 16 tables and assigned in Unicode 17. Measured by listing
// `not chr(c).isprintable()` from U+007F up in Python, comparing in Go and
// compressing the differences to ranges.
var newInUnicode17 = [][2]rune{
	{0x88F, 0x88F}, {0xC5C, 0xC5C}, {0xCDC, 0xCDC}, {0x1ACF, 0x1ADD}, {0x1AE0, 0x1AEB},
	{0x20C1, 0x20C1}, {0x2B96, 0x2B96}, {0xA7CE, 0xA7CF}, {0xA7D2, 0xA7D2}, {0xA7D4, 0xA7D4},
	{0xA7F1, 0xA7F1}, {0xFBC3, 0xFBD2}, {0xFD90, 0xFD91}, {0xFDC8, 0xFDCE}, {0x10940, 0x10959},
	{0x10EC5, 0x10EC7}, {0x10ED0, 0x10ED8}, {0x10EFA, 0x10EFB}, {0x11B60, 0x11B67}, {0x11DB0, 0x11DDB},
	{0x11DE0, 0x11DE9}, {0x16EA0, 0x16EB8}, {0x16EBB, 0x16ED3}, {0x16FF2, 0x16FF6}, {0x187F8, 0x187FF},
	{0x18D09, 0x18D1E}, {0x18D80, 0x18DF2}, {0x1CCFA, 0x1CCFC}, {0x1CEBA, 0x1CED0}, {0x1CEE0, 0x1CEF0},
	{0x1E6C0, 0x1E6DE}, {0x1E6E0, 0x1E6F5}, {0x1E6FE, 0x1E6FF}, {0x1F6D8, 0x1F6D8}, {0x1F777, 0x1F77A},
	{0x1F8D0, 0x1F8D8}, {0x1FA54, 0x1FA57}, {0x1FA8A, 0x1FA8A}, {0x1FA8E, 0x1FA8E}, {0x1FAC8, 0x1FAC8},
	{0x1FACD, 0x1FACD}, {0x1FAEA, 0x1FAEA}, {0x1FAEF, 0x1FAEF}, {0x1FBFA, 0x1FBFA}, {0x2B73A, 0x2B73F},
	{0x2CEA2, 0x2CEAD}, {0x323B0, 0x33479},
}

func isNewInUnicode17(r rune) bool {
	for _, span := range newInUnicode17 {
		if r >= span[0] && r <= span[1] {
			return true
		}
	}
	return false
}

// rangeDigest is sha256 over f(string(r)) followed by a zero byte for every
// code point but the surrogates, and without the Unicode 17 additions when
// skipNew is set -- the loop the Python measurement runs.
func rangeDigest(f func(string) string, skipNew bool) string {
	h := sha256.New()
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff || skipNew && isNewInUnicode17(r) {
			continue
		}
		io.WriteString(h, f(string(r)))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestTheTablesAreTheOnesMeasured(t *testing.T) {
	// The digests in this package were compared against Unicode 17 tables,
	// which Go 1.27 brings; go.mod's `go 1.25.0` lets an older toolchain
	// build the gate, and this stops it with the reason instead of a wrong
	// digest. A Go or x/text with other tables must be measured against the
	// reference again, not have its digests copied in.
	if unicode.Version != "17.0.0" || norm.Version != "17.0.0" || cases.UnicodeVersion != "17.0.0" {
		t.Fatalf("pytext tables are measured against Unicode 17.0.0 (Go 1.27); re-measure for unicode %s, norm %s, cases %s against Python 3.14 of the reference",
			unicode.Version, norm.Version, cases.UnicodeVersion)
	}
	count := 0
	for _, span := range newInUnicode17 {
		count += int(span[1]-span[0]) + 1
	}
	if count != 4803 {
		t.Errorf("newInUnicode17 holds %d code points, measured 4803", count)
	}
}
