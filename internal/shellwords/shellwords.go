// Package shellwords splits a command line by shell word rules, without a shell.
package shellwords

import (
	"fmt"
	"strings"
)

// Split parses a command line string into tokens respecting quotes and escapes.
func Split(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	hadQuotes := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			cur.WriteByte(c)
			escaped = false
			continue
		}
		// Inside double quotes a backslash escapes only $ ` " \ and a
		// newline (which, unlike in a shell, stays in the word); before
		// anything else (a Windows path) it is a plain character.
		if c == '\\' && inDouble && (i+1 >= len(s) || !strings.ContainsRune("$`\"\\\n", rune(s[i+1]))) {
			cur.WriteByte(c)
			continue
		}
		if c == '\\' && !inSingle {
			escaped = true
			continue
		}
		if inSingle {
			if c == '\'' {
				inSingle = false
				hadQuotes = true
			} else {
				cur.WriteByte(c)
			}
			continue
		}
		if inDouble {
			if c == '"' {
				inDouble = false
				hadQuotes = true
			} else {
				cur.WriteByte(c)
			}
			continue
		}
		switch c {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case ' ', '\t', '\n', '\r':
			if cur.Len() > 0 || hadQuotes {
				tokens = append(tokens, cur.String())
				cur.Reset()
				hadQuotes = false
			}
		default:
			cur.WriteByte(c)
		}
	}
	if inSingle || inDouble || escaped {
		return nil, fmt.Errorf("unclosed quote or escape in command: %s", s)
	}
	if cur.Len() > 0 || hadQuotes {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}
