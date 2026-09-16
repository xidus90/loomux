package pytext

import (
	"slices"
	"testing"
	"unicode"
)

func TestSplitLinesLikePython(t *testing.T) {
	// print(ascii(s), ascii(s.splitlines())) on Python 3.14.7.
	cases := []struct {
		in   string
		want []string
	}{
		{"a\n", []string{"a"}},
		{"\n", []string{""}},
		{"", nil},
		{"a\r\nb", []string{"a", "b"}},
		{"\r\r\n", []string{"", ""}},
		{"a\n\rb", []string{"a", "", "b"}},
		{"\n\n", []string{"", ""}},
		{"a", []string{"a"}},
		{"a\U00000085b\U00002028c\U00002029d\x0be\x0cf\x1cg\x1dh\x1ei\rj\r\nk\nl",
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}},
	}
	for _, c := range cases {
		if got := SplitLines(c.in); !slices.Equal(got, c.want) {
			t.Errorf("SplitLines(%+q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitLinesSplitsOnTheTenBoundariesAndNothingElse(t *testing.T) {
	// [hex(c) for c in range(0x110000) if not 0xd800<=c<=0xdfff
	//  and len(('a'+chr(c)+'b').splitlines())==2] on Python 3.14.7.
	boundaries := map[rune]bool{0x0a: true, 0x0b: true, 0x0c: true, 0x0d: true, 0x1c: true,
		0x1d: true, 0x1e: true, 0x85: true, 0x2028: true, 0x2029: true}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		if split := len(SplitLines("a"+string(r)+"b")) == 2; split != boundaries[r] {
			t.Errorf("U+%04X splits = %v, Python says %v", r, split, boundaries[r])
		}
	}
}
