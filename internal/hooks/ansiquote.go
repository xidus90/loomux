package hooks

import (
	"strconv"
	"strings"
	"unicode"
)

// dollarQuoted is line with each bash $'…' (ANSI-C quoting) replaced by the
// plain single-quoted word it spells, and each $"…" (locale quoting) by the
// plain "…": the shell writes the file the word names, where the guard
// would keep the $ and read an expansion. A $ inside a quote or behind a
// backslash is a $ and stays.
func dollarQuoted(line string) string {
	var b strings.Builder
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0 && c == quote:
			quote = 0
		case quote == '\'':
		case c == '\\' && i+1 < len(line):
			// An escaped byte, inside double quotes as outside them.
			b.WriteByte(c)
			i++
			c = line[i]
		case quote != 0:
		case c == '$' && i+1 < len(line) && line[i+1] == '\'':
			word, end := ansiCWord(line, i+2)
			b.WriteString(word)
			i = end
			continue
		case c == '$' && i+1 < len(line) && line[i+1] == '"':
			continue
		case c == '"' || c == '\'':
			quote = c
		}
		b.WriteByte(c)
	}
	return b.String()
}

// ansiCWord reads the body of $'…' from line[start:] up to its closing
// quote, which a backslash escapes, and answers the single-quoted word of
// the decoded text and the index of that closing quote (the end of line when
// there is none).
func ansiCWord(line string, start int) (word string, end int) {
	var text strings.Builder
	i := start
	for ; i < len(line) && line[i] != '\''; i++ {
		if line[i] != '\\' || i+1 == len(line) {
			text.WriteByte(line[i])
			continue
		}
		decoded, n := ansiCEscape(line[i+1:])
		text.WriteString(decoded)
		i += n
	}
	return "'" + strings.ReplaceAll(text.String(), "'", `'\''`) + "'", i
}

// ansiCControls are the escapes of $'…' that stand for one byte.
var ansiCControls = map[byte]string{
	'a': "\a", 'b': "\b", 'e': "\x1b", 'E': "\x1b", 'f': "\f", 'n': "\n", 'r': "\r", 't': "\t", 'v': "\v",
	'\\': `\`, '\'': "'", '"': `"`, '?': "?",
}

// ansiCEscape decodes the escape that follows a backslash in $'…' and says
// how many bytes it took: the control letters, \xHH, \NNN, \uHHHH and
// \UHHHHHHHH. An escape bash does not know stays as written.
func ansiCEscape(s string) (decoded string, n int) {
	switch c := s[0]; {
	case c == 'x':
		return numberEscape(s, 1, 2, 16)
	case c == 'u':
		return numberEscape(s, 1, 4, 16)
	case c == 'U':
		return numberEscape(s, 1, 8, 16)
	case c >= '0' && c <= '7':
		return numberEscape(s, 0, 3, 8)
	default:
		if text, ok := ansiCControls[c]; ok {
			return text, 1
		}
		return `\` + string(c), 1
	}
}

// numberEscape decodes the longest run of at most most digits of the base
// after the first skip bytes of s: one byte for \x and \NNN, one character
// for \u and \U. Without a digit the escape stays as written.
func numberEscape(s string, skip, most, base int) (string, int) {
	for k := min(most, len(s)-skip); k > 0; k-- {
		value, err := strconv.ParseUint(s[skip:skip+k], base, 32)
		switch {
		case err != nil:
		case s[0] == 'u' || s[0] == 'U':
			if value > unicode.MaxRune {
				return string(unicode.ReplacementChar), skip + k
			}
			return string(rune(value)), skip + k
		default:
			// Bash keeps the low byte of an octal escape above \377.
			return string([]byte{byte(value & 0xff)}), skip + k
		}
	}
	return `\` + s[:1], 1
}
