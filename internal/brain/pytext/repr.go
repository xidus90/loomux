// Package pytext writes and reads text the way Python 3.14 of the
// reference does: repr, splitlines, strip, read_text,
// isoformat and fromisoformat, str(Path), NFC and casefold. Five brain
// packages print or compare such text, and each must agree with the
// reference byte for byte.
package pytext

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const hexDigits = "0123456789abcdef"

// Repr is `repr(s)` of a str. The quote is `'` unless the text holds a `'`
// and no `"`; the chosen quote and the backslash are escaped, `\t` `\n` `\r`
// keep their short forms, and every other character Python does not call
// printable becomes `\xhh`, `\uhhhh` or `\Uhhhhhhhh` by size.
//
// Printable is `unicode.IsPrint` above ASCII, which is Python's rule
// (letters, marks, numbers, punctuation, symbols, and the space) over
// Go's Unicode 17 tables; Python 3.14 has Unicode 16, so the 4803 code
// points 17 assigns print here and are escaped there.
//
// A byte that is not UTF-8 has no Python counterpart -- a str never holds
// one -- and is written as `\xhh`.
func Repr(s string) string {
	quote := byte('\'')
	if strings.IndexByte(s, '\'') >= 0 && strings.IndexByte(s, '"') < 0 {
		quote = '"'
	}
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte(quote)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			writeHex(&b, `\x`, uint32(s[i]), 2)
			i++
			continue
		}
		i += size
		switch {
		case r == rune(quote) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r < ' ' || r == 0x7f:
			writeHex(&b, `\x`, uint32(r), 2)
		case r < 0x7f || unicode.IsPrint(r):
			b.WriteRune(r)
		case r <= 0xff:
			writeHex(&b, `\x`, uint32(r), 2)
		case r <= 0xffff:
			writeHex(&b, `\u`, uint32(r), 4)
		default:
			writeHex(&b, `\U`, uint32(r), 8)
		}
	}
	b.WriteByte(quote)
	return b.String()
}

// writeHex writes prefix and width lower-case hex digits of v.
func writeHex(b *strings.Builder, prefix string, v uint32, width int) {
	b.WriteString(prefix)
	for shift := (width - 1) * 4; shift >= 0; shift -= 4 {
		b.WriteByte(hexDigits[v>>uint(shift)&0xf])
	}
}
