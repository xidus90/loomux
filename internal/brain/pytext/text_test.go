package pytext

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"unicode"
)

func TestIsSpaceIsPythonsSetOverTheWholeRange(t *testing.T) {
	// [hex(c) for c in range(0x110000) if chr(c).isspace()] on Python 3.14.7.
	python := map[rune]bool{}
	for _, r := range []rune{0x9, 0xa, 0xb, 0xc, 0xd, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x85, 0xa0,
		0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007, 0x2008, 0x2009,
		0x200a, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000} {
		python[r] = true
	}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if IsSpace(r) != python[r] {
			t.Errorf("IsSpace(U+%04X) = %v, Python says %v", r, IsSpace(r), python[r])
		}
	}
}

func TestStripAndRStripLikePython(t *testing.T) {
	// s.strip() and s.rstrip() on Python 3.14.7.
	cases := []struct{ in, strip, rstrip string }{
		{"  a  ", "a", "  a"},
		{"\x1c a \x1f", "a", "\x1c a"},
		{"\U000000a0a\U00003000", "a", "\U000000a0a"},
		{"\U0000200ba\U0000200b", "\U0000200ba\U0000200b", "\U0000200ba\U0000200b"},
		{"\t\n", "", ""},
		{"a b", "a b", "a b"},
	}
	for _, c := range cases {
		if got := Strip(c.in); got != c.strip {
			t.Errorf("Strip(%+q) = %+q, want %+q", c.in, got, c.strip)
		}
		if got := RStrip(c.in); got != c.rstrip {
			t.Errorf("RStrip(%+q) = %+q, want %+q", c.in, got, c.rstrip)
		}
	}
}

func TestReadTextFoldsCROnlyAndKeepsTheBOM(t *testing.T) {
	// Path.read_text(encoding="utf-8") of the same bytes on Python 3.14.7.
	cases := []struct{ name, bytes, want string }{
		{"bom-crlf", "\xef\xbb\xbfa\r\nb\rc\n\xc2\x85d\r", "\U0000feffa\nb\nc\n\U00000085d\n"},
		{"lone-cr", "x\r", "x\n"},
		{"cr-crlf", "\r\r\n", "\n\n"},
	}
	for _, c := range cases {
		path := filepath.Join(t.TempDir(), c.name)
		if err := os.WriteFile(path, []byte(c.bytes), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ReadText(path)
		if err != nil || got != c.want {
			t.Errorf("ReadText(%s) = %+q, %v; want %+q", c.name, got, err, c.want)
		}
	}
}

func TestReadTextRefusesBytesPythonCannotDecode(t *testing.T) {
	// Python: UnicodeDecodeError for both, "invalid start byte" and
	// "invalid continuation byte" -- an encoded surrogate is not UTF-8.
	for name, bytes := range map[string]string{"invalid": "a\xffb", "surrogate": "a\xed\xa0\x80b"} {
		path := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(path, []byte(bytes), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := ReadText(path)
		if want := path + ": not valid UTF-8"; err == nil || err.Error() != want {
			t.Errorf("ReadText(%s) err = %v, want %q", name, err, want)
		}
	}
}

func TestReadTextPassesOtherErrorsOn(t *testing.T) {
	_, err := ReadText(filepath.Join(t.TempDir(), "missing"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want fs.ErrNotExist", err)
	}
}
