package pytext

import "unicode/utf8"

// SplitLines is `s.splitlines()`: the ten line boundaries Python knows,
// `\r\n` as one, no line ends kept. An empty text has no lines, and a
// boundary at the very end opens no empty last line.
func SplitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !isLineBoundary(r) {
			i += size
			continue
		}
		lines = append(lines, s[start:i])
		i += size
		if r == '\r' && i < len(s) && s[i] == '\n' {
			i++
		}
		start = i
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// isLineBoundary is the set `splitlines` splits on, measured over the whole
// range against Python 3.14: \n \v \f \r, the three separators \x1c-\x1e,
// NEL, LINE SEPARATOR and PARAGRAPH SEPARATOR. \x1f is space to Python but
// no boundary.
func isLineBoundary(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}
