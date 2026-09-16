package privacy

import (
	"regexp"
	"slices"
	"strings"
)

// posixParts parses p the way PurePosixPath does (`PurePath._parse_path` with
// `posixpath.splitroot`): the root is "" for a relative path, "//" for exactly
// two leading slashes -- POSIX leaves that form to the implementation and
// pathlib keeps it -- and "/" for one or three and more. The parts are the
// components between slashes that are neither empty nor `.`; `..` stays a
// part, because pathlib is lexical.
func posixParts(p string) (root string, parts []string) {
	rest := p
	switch {
	case !strings.HasPrefix(p, "/"):
	case strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///"):
		root, rest = "//", p[2:]
	default:
		root, rest = "/", p[1:]
	}
	for _, part := range strings.Split(rest, "/") {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	return root, parts
}

// fullMatchForm is the string `PurePath.full_match` hands to the translator,
// for the path and for the pattern alike: `str(path)` when the path has parts,
// and "" when it has none, because "the string representation of an empty path
// is a single dot ('.')".
func fullMatchForm(p string) string {
	root, parts := posixParts(p)
	return root + strings.Join(parts, "/")
}

// translateGlob is `glob.translate(pattern, recursive=True,
// include_hidden=True, seps="/")` of Python 3.14 -- the call
// `_GlobberBase.compile` makes for `full_match` on a PurePosixPath -- written
// as a Go regular expression. The four constants are the `include_hidden` arm
// of that function spelt out for the one separator. Python matches with
// `re.match`, which anchors at the start; the leading `^` says the same.
func translateGlob(pattern string) string {
	const (
		oneLastSegment  = `[^/]+`
		oneSegment      = oneLastSegment + `/`
		anySegments     = `(?:.+/)?`
		anyLastSegments = `.*`
	)
	var results strings.Builder
	parts := strings.Split(pattern, "/")
	last := len(parts) - 1
	for idx, part := range parts {
		switch {
		case part == "*":
			if idx < last {
				results.WriteString(oneSegment)
			} else {
				results.WriteString(oneLastSegment)
			}
		case part == "**":
			if idx == last {
				results.WriteString(anyLastSegments)
			} else if parts[idx+1] != "**" {
				results.WriteString(anySegments)
			}
		default:
			if part != "" {
				results.WriteString(translateSegment(part))
			}
			if idx < last {
				results.WriteString("/")
			}
		}
	}
	return `^(?s:` + results.String() + `)\z`
}

// translateSegment is `fnmatch._translate(part, "[^/]*", "[^/]")` of Python
// 3.14, joined, for one segment. It walks code points, as Python indexes a
// str, and keeps the branches of the original in their order: a run of `*`,
// `?`, a bracket expression, an unclosed `[`, a literal.
//
// Three spellings differ because RE2 is not Python's `re`; what they match
// does not. An empty range, `(?!)` in Python, is a class of no character,
// since RE2 has no lookahead. A literal goes through regexp.QuoteMeta instead
// of `re.escape`. And classBody escapes `[` inside a class.
func translateSegment(segment string) string {
	pat := []rune(segment)
	n := len(pat)
	var res strings.Builder
	for i := 0; i < n; {
		c := pat[i]
		i++
		switch c {
		case '*':
			res.WriteString(`[^/]*`)
			for i < n && pat[i] == '*' {
				i++
			}
		case '?':
			res.WriteString(`[^/]`)
		case '[':
			j := i
			if j < n && pat[j] == '!' {
				j++
			}
			if j < n && pat[j] == ']' {
				j++
			}
			for j < n && pat[j] != ']' {
				j++
			}
			if j >= n {
				res.WriteString(`\[`)
				continue
			}
			stuff := bracketBody(pat, i, j)
			i = j + 1
			switch stuff {
			case "":
				res.WriteString(`[^\x00-\x{10FFFF}]`)
			case "!":
				res.WriteString(`.`)
			default:
				res.WriteString("[" + classBody(stuff) + "]")
			}
		default:
			res.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	return res.String()
}

// bracketBody is the part of `fnmatch._translate` that turns the characters
// between `[` (at i) and `]` (at j) into `stuff`: without a `-`, backslashes
// doubled; with one, the chunks between range hyphens, a trailing hyphen kept,
// empty ranges removed, and backslashes and the hyphens that make no range
// escaped.
func bracketBody(pat []rune, i, j int) string {
	if !slices.Contains(pat[i:j], '-') {
		return strings.ReplaceAll(string(pat[i:j]), `\`, `\\`)
	}
	var chunks [][]rune
	k := i + 1
	if pat[i] == '!' {
		k = i + 2
	}
	for {
		k = indexHyphen(pat, k, j)
		if k < 0 {
			break
		}
		chunks = append(chunks, pat[i:k])
		i = k + 1
		k += 3
	}
	if chunk := pat[i:j]; len(chunk) > 0 {
		chunks = append(chunks, chunk)
	} else {
		last := len(chunks) - 1
		chunks[last] = append(slices.Clone(chunks[last]), '-')
	}
	for k := len(chunks) - 1; k > 0; k-- {
		if prev := chunks[k-1]; prev[len(prev)-1] > chunks[k][0] {
			chunks[k-1] = append(slices.Clone(prev[:len(prev)-1]), chunks[k][1:]...)
			chunks = slices.Delete(chunks, k, k+1)
		}
	}
	escaped := make([]string, len(chunks))
	for m, chunk := range chunks {
		escaped[m] = strings.ReplaceAll(strings.ReplaceAll(string(chunk), `\`, `\\`), "-", `\-`)
	}
	return strings.Join(escaped, "-")
}

// indexHyphen is `pat.find('-', k, j)`: the first `-` at k or after and
// before j, or -1.
func indexHyphen(pat []rune, k, j int) int {
	for ; k < j; k++ {
		if pat[k] == '-' {
			return k
		}
	}
	return -1
}

// classBody finishes `stuff` the way `fnmatch._translate` does before it wraps
// it in brackets -- `&`, `~` and `|` escaped, a leading `!` made the
// negation, a leading `^` or `[` escaped -- and then escapes every other `[`:
// Python reads it as a literal, RE2 would read `[:alpha:]` as a POSIX class.
// Every backslash in stuff opens a pair with an ASCII character, so the pair
// is copied whole.
func classBody(stuff string) string {
	stuff = strings.NewReplacer("&", `\&`, "~", `\~`, "|", `\|`).Replace(stuff)
	switch stuff[0] {
	case '!':
		stuff = "^" + stuff[1:]
	case '^', '[':
		stuff = `\` + stuff
	}
	var body strings.Builder
	for i := 0; i < len(stuff); i++ {
		switch stuff[i] {
		case '\\':
			body.WriteString(stuff[i : i+2])
			i++
		case '[':
			body.WriteString(`\[`)
		default:
			body.WriteByte(stuff[i])
		}
	}
	return body.String()
}
