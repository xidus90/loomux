package pytext

import (
	"strings"
	"unicode/utf8"
)

// SplitURL is the part of `urllib.parse.urlsplit` a link rule reads: the
// scheme, folded to lower case, the network location and the path, with query
// and fragment cut off. Unlike url.Parse it never refuses: a malformed percent
// escape or a stray character stays in the path, as it does in Python.
//
// The steps are Python 3.14's in its order. Leading control characters and
// blanks are stripped and tabs and line breaks removed; the scheme is what
// precedes the first colon when it starts with an ASCII letter and holds only
// letters, digits, `+`, `-` and `.` -- so `c:/x` has the scheme `c`; a `//`
// then opens a network location that runs to the first `/`, `?` or `#`; the
// fragment is cut at the first `#`, the query at the first `?` before it.
func SplitURL(raw string) (scheme, netloc, path string) {
	raw = strings.TrimLeft(raw, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f"+
		"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	raw = strings.NewReplacer("\t", "", "\r", "", "\n", "").Replace(raw)
	if i := strings.IndexByte(raw, ':'); i > 0 && isASCIILetter(raw[0]) && isSchemeName(raw[:i]) {
		scheme, raw = strings.ToLower(raw[:i]), raw[i+1:]
	}
	if strings.HasPrefix(raw, "//") {
		end := len(raw)
		if i := strings.IndexAny(raw[2:], "/?#"); i >= 0 {
			end = i + 2
		}
		netloc, raw = raw[2:end], raw[end:]
	}
	if i := strings.IndexByte(raw, '#'); i >= 0 {
		raw = raw[:i]
	}
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		raw = raw[:i]
	}
	return scheme, netloc, raw
}

func isASCIILetter(c byte) bool {
	return 'a' <= c|0x20 && c|0x20 <= 'z'
}

func isSchemeName(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !isASCIILetter(c) && !('0' <= c && c <= '9') && c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

// Unquote is `urllib.parse.unquote`: every `%XX` with two hex digits becomes
// its byte, anything else stays as written, and the bytes are read as UTF-8
// with errors replaced (DecodeReplace).
func Unquote(s string) string {
	raw := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			raw = append(raw, unhex(s[i+1])<<4|unhex(s[i+2]))
			i += 2
			continue
		}
		raw = append(raw, s[i])
	}
	return DecodeReplace(raw)
}

// DecodeReplace is `bytes.decode("utf-8", errors="replace")`: every sequence
// that decodes is kept, every one that does not becomes one U+FFFD.
func DecodeReplace(raw []byte) string {
	var b strings.Builder
	for len(raw) > 0 {
		size, whole := sequence(raw)
		if whole {
			b.Write(raw[:size])
		} else {
			b.WriteRune(utf8.RuneError)
		}
		raw = raw[size:]
	}
	return b.String()
}

// sequence is the length of the UTF-8 sequence at the start of raw and
// whether it decodes. A sequence that does not is the lead byte and every
// continuation byte that still fits it: Python's decoder replaces such a
// maximal subpart once -- `%e2%82` is one U+FFFD there -- where a decoder
// going byte by byte would write two. The ranges of the second byte are the
// ones that keep out overlong forms, surrogates and code points past U+10FFFF.
func sequence(raw []byte) (int, bool) {
	lo, hi, need := byte(0x80), byte(0xBF), 0
	switch lead := raw[0]; {
	case lead < 0x80:
		return 1, true
	case 0xC2 <= lead && lead <= 0xDF:
		need = 1
	case lead == 0xE0:
		lo, need = 0xA0, 2
	case lead == 0xED:
		hi, need = 0x9F, 2
	case 0xE1 <= lead && lead <= 0xEF:
		need = 2
	case lead == 0xF0:
		lo, need = 0x90, 3
	case lead == 0xF4:
		hi, need = 0x8F, 3
	case 0xF1 <= lead && lead <= 0xF3:
		need = 3
	default:
		return 1, false
	}
	size := 1
	for size <= need && size < len(raw) && lo <= raw[size] && raw[size] <= hi {
		size++
		lo, hi = 0x80, 0xBF
	}
	return size, size == need+1
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c|0x20 && c|0x20 <= 'f'
}

func unhex(c byte) byte {
	if c <= '9' {
		return c - '0'
	}
	return (c | 0x20) - 'a' + 10
}
