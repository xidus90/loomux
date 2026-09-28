package hooks

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// maxBraceVariants is how many words one brace expansion may unfold to
// before the guard refuses rather than judge them one by one.
const maxBraceVariants = 64

// spellings are the paths a shell makes of one target word, each relative to
// root the way a rule spells a path: braces unfolded, a stream name cut off,
// globs matched against the disk. PowerShell unfolds no braces, and bash none
// inside quotes; reading them anyway only ever refuses more. An error is a
// refusal the caller names.
func spellings(root, word string) ([]string, error) {
	words, ok := unfoldBraces(word)
	if !ok {
		return nil, fmt.Errorf("loomux does not unfold more than %d brace variants of %q, so it refuses", maxBraceVariants, word)
	}
	var out []string
	for _, w := range words {
		for _, p := range expandGlob(root, withoutStream(w)) {
			out = append(out, relativePath(p, root))
		}
	}
	return out, nil
}

// unfoldBraces is bash's brace expansion of word: {a,b}, {1..3}, {01..03},
// {1..9..2} and {a..c}, nested and in sequence, never a ${…}. A brace group
// without a comma or a range, or one left open, stays as written. ok is false
// past maxBraceVariants.
func unfoldBraces(word string) ([]string, bool) {
	out := []string{word}
	for i := 0; i < len(out); {
		next, unfolded := unfoldFirst(out[i])
		if !unfolded {
			i++
			continue
		}
		out = append(out[:i], append(next, out[i+1:]...)...)
		if len(out) > maxBraceVariants {
			return nil, false
		}
	}
	return out, true
}

// unfoldFirst unfolds the first brace group of word that expands.
func unfoldFirst(word string) ([]string, bool) {
	for open := 0; open < len(word); open++ {
		if word[open] != '{' || open > 0 && word[open-1] == '$' {
			continue
		}
		end, commas := braceGroup(word, open)
		if end < 0 {
			continue
		}
		var alternatives []string
		if len(commas) > 0 {
			last := open + 1
			for _, c := range commas {
				alternatives = append(alternatives, word[last:c])
				last = c + 1
			}
			alternatives = append(alternatives, word[last:end])
		} else if sequence, ok := braceRange(word[open+1 : end]); ok {
			alternatives = sequence
		} else {
			continue
		}
		out := make([]string, len(alternatives))
		for i, alternative := range alternatives {
			out[i] = word[:open] + alternative + word[end+1:]
		}
		return out, true
	}
	return nil, false
}

// braceGroup is the index of the } that closes the { at open, or -1, and the
// commas at the group's own depth.
func braceGroup(word string, open int) (int, []int) {
	depth := 0
	var commas []int
	for i := open; i < len(word); i++ {
		switch word[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, commas
			}
		case ',':
			if depth == 1 {
				commas = append(commas, i)
			}
		}
	}
	return -1, nil
}

// braceRange is the sequence a range stands for: numbers, zero-padded when
// either end is, or single letters, with an optional step. A range longer
// than maxBraceVariants stops one past it, which the caller refuses.
func braceRange(inner string) ([]string, bool) {
	parts := strings.Split(inner, "..")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, false
	}
	step := 1
	if len(parts) == 3 {
		s, err := strconv.Atoi(parts[2])
		if err != nil || s == 0 {
			return nil, false
		}
		if s < 0 {
			s = -s
		}
		step = s
	}
	from, errFrom := strconv.Atoi(parts[0])
	to, errTo := strconv.Atoi(parts[1])
	isLetters := errFrom != nil || errTo != nil
	width := 0
	if isLetters {
		if len(parts[0]) != 1 || len(parts[1]) != 1 || !isLetter(parts[0][0]) || !isLetter(parts[1][0]) {
			return nil, false
		}
		from, to = int(parts[0][0]), int(parts[1][0])
	} else if padded(parts[0]) || padded(parts[1]) {
		width = max(len(parts[0]), len(parts[1]))
	}
	direction := 1
	if to < from {
		direction = -1
	}
	var out []string
	for v := from; (v-to)*direction <= 0 && len(out) <= maxBraceVariants; v += direction * step {
		if isLetters {
			out = append(out, string(rune(v)))
		} else {
			out = append(out, fmt.Sprintf("%0*d", width, v))
		}
	}
	return out, true
}

// padded says whether a range end is written with leading zeros.
func padded(end string) bool {
	return len(end) > 1 && end[0] == '0'
}

// withoutStream cuts an NTFS stream name off a path -- x.toml:backup writes
// x.toml -- after the volume: a drive letter, behind an extended-length or
// device prefix too, read the same on every system.
func withoutStream(p string) string {
	start := 0
	if len(p) >= 4 && isSlash(p[0]) && isSlash(p[1]) && (p[2] == '?' || p[2] == '.') && isSlash(p[3]) {
		start = 4
	}
	if len(p) >= start+2 && p[start+1] == ':' && isLetter(p[start]) {
		start += 2
	}
	start = max(start, len(filepath.VolumeName(p)))
	if colon := strings.IndexByte(p[start:], ':'); colon >= 0 {
		return p[:start+colon]
	}
	return p
}

// expandGlob matches word against the disk under root as bash does: a *, ?
// or [ in a name, and a name that begins with a dot only for a pattern
// element that does too. A pattern that matches nothing, or one filepath.Glob
// cannot read, stays as written, as bash leaves it.
func expandGlob(root, word string) []string {
	if !strings.ContainsAny(word, "*?[") {
		return []string{word}
	}
	pattern := word
	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(root, pattern)
	}
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return []string{word}
	}
	var out []string
	for _, m := range matches {
		if !hidesDotNames(pattern, m) {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return []string{word}
	}
	return out
}

// hidesDotNames says whether bash would skip match for pattern: one of its
// names begins with a dot where the pattern element does not. filepath.Glob
// gives a match as many names as the pattern has.
func hidesDotNames(pattern, match string) bool {
	want := strings.Split(filepath.ToSlash(filepath.Clean(pattern)), "/")
	got := strings.Split(filepath.ToSlash(match), "/")
	for i, name := range got {
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(want[i], ".") {
			return true
		}
	}
	return false
}

// isSlash says whether c separates path names on either system.
func isSlash(c byte) bool {
	return c == '/' || c == '\\'
}

// isLetter says whether c is an ASCII letter.
func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// letters says whether s is one or more ASCII letters.
func letters(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isLetter(s[i]) {
			return false
		}
	}
	return s != ""
}
