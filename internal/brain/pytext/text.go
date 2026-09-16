package pytext

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// IsSpace is `str.isspace()` of one character: the 29 code points Python
// 3.14 calls space, listed over the whole range. It is not
// `unicode.IsSpace`, which leaves out \x1c-\x1f.
func IsSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, ' ',
		0x85, 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

// Strip is `s.strip()` without arguments.
func Strip(s string) string {
	return strings.TrimFunc(s, IsSpace)
}

// RStrip is `s.rstrip()` without arguments.
func RStrip(s string) string {
	return strings.TrimRightFunc(s, IsSpace)
}

// ReadText is `Path(path).read_text(encoding="utf-8")`: strict UTF-8, and
// universal newlines, which turn `\r\n` and a lone `\r` into `\n` and leave
// every other boundary alone. A byte order mark stays, because "utf-8" is
// not "utf-8-sig".
//
// Python ends with a UnicodeDecodeError traceback on bytes that are not
// UTF-8; this answers an error naming the file. Every other failure is
// os.ReadFile's, unwrapped.
func ReadText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s: not valid UTF-8", path)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n"), nil
}
