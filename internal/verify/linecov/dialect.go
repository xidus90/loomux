package linecov

import (
	"errors"
	"path"
	"regexp"
	"strings"
)

// Matcher answers for a slash path relative to the root.
type Matcher func(rel string) bool

// Dialect turns one of a tool's patterns into a Matcher, read the way that
// tool reads it. root is absolute, with slashes. A new dialect is one
// function and one entry in dialects.
type Dialect func(pattern, root string) (Matcher, error)

func dialects() map[string]Dialect {
	return map[string]Dialect{
		"coveragepy": coveragePy,
		"minimatch":  miniMatch,
		"nano":       nanoGlob,
		"lcov":       lcovGlob,
		"regex":      regexDialect,
		"dir":        dirDialect,
	}
}

// globOpts selects which parts of glob syntax a tool gives meaning to.
type globOpts struct {
	crosses bool // * and ? match a slash as well
	dirOpt  bool // **/ may match no directory at all
	classes bool // [abc] and [!abc]
	braces  bool // {a,b}
}

// globRegexp translates a glob into a regular expression body; ** always
// crosses slashes. An unclosed [ or { is an error, never a literal.
func globRegexp(p string, o globOpts) (string, error) {
	star, one := "[^/]*", "[^/]"
	if o.crosses {
		star, one = ".*", "."
	}
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		switch {
		case o.dirOpt && strings.HasPrefix(p[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString(star)
		case p[i] == '?':
			b.WriteString(one)
		case o.classes && p[i] == '[':
			end := classEnd(p, i)
			if end < 0 {
				return "", errors.New("unclosed [ in pattern")
			}
			b.WriteString(classRegexp(p[i+1 : end]))
			i = end
		case o.braces && p[i] == '{':
			end, parts := braceParts(p, i)
			if end < 0 {
				return "", errors.New("unclosed { in pattern")
			}
			alts := make([]string, len(parts))
			for k, part := range parts {
				alt, err := globRegexp(part, o)
				if err != nil {
					return "", err
				}
				alts[k] = alt
			}
			b.WriteString("(?:" + strings.Join(alts, "|") + ")")
			i = end
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	return b.String(), nil
}

// classEnd returns the index of the ] closing the class opened at p[i], or -1.
func classEnd(p string, i int) int {
	j := i + 1
	if j < len(p) && (p[j] == '!' || p[j] == '^') {
		j++
	}
	if j < len(p) && p[j] == ']' {
		j++
	}
	k := strings.IndexByte(p[j:], ']')
	if k < 0 {
		return -1
	}
	return j + k
}

// classRegexp turns the inside of a glob class into a regexp class; a
// negated class leaves out the slash. Patterns carry no backslashes by now,
// and a ] can only be the class's first character.
func classRegexp(body string) string {
	open := "["
	if body[0] == '!' || body[0] == '^' { // never empty: classEnd skips a leading ]
		open, body = "[^/", body[1:]
	}
	return open + strings.ReplaceAll(body, "]", `\]`) + "]"
}

// braceParts splits the brace set opened at p[i] on its top-level commas and
// returns the index of its closing brace, or -1.
func braceParts(p string, i int) (int, []string) {
	depth, from := 0, i+1
	var parts []string
	for j := i; j < len(p); j++ {
		switch p[j] {
		case '{':
			depth++
		case ',':
			if depth == 1 {
				parts = append(parts, p[from:j])
				from = j + 1
			}
		case '}':
			depth--
			if depth == 0 {
				return j, append(parts, p[from:j])
			}
		}
	}
	return -1, nil
}

func anchored(p string, o globOpts) (*regexp.Regexp, error) { return globCompile(p, o, "^", "$") }

func globCompile(p string, o globOpts, pre, post string) (*regexp.Regexp, error) {
	body, err := globRegexp(p, o)
	if err != nil {
		return nil, err
	}
	return regexp.Compile(pre + body + post)
}

// trimRoot drops trailing slashes, so that root + "/" + rel never doubles one.
func trimRoot(root string) string { return strings.TrimRight(root, "/") }

// slashes is the pattern with / as the only separator.
func slashes(p string) string { return strings.ReplaceAll(p, `\`, "/") }

// relPattern makes an absolute pattern relative to root; a pattern equal to
// root becomes empty. Drive letters compare without regard to case.
func relPattern(p, root string) string {
	p, root = slashes(p), trimRoot(root)
	if len(root) >= 2 && root[1] == ':' {
		if len(p) >= len(root) && strings.EqualFold(p[:len(root)], root) {
			p = root + p[len(root):]
		}
	}
	if p == root {
		return ""
	}
	return strings.TrimPrefix(strings.TrimPrefix(p, root+"/"), "./")
}

// coverage.py reads a leading */ as any prefix, none included, and a
// trailing /* as anything below; every other * stays within one directory.
var (
	pyLead  = regexp.MustCompile(`^\*+/`)
	pyTrail = regexp.MustCompile(`/\*+$`)
)

// coveragePy: a pattern without a slash matches the file name in any
// directory; any other is relative to the root.
func coveragePy(p, root string) (Matcher, error) {
	p = relPattern(p, root)
	base := !strings.Contains(p, "/")
	pre, post := "^", "$"
	if m := pyLead.FindString(p); m != "" {
		pre, p = "^(?:.*/)?", p[len(m):]
	}
	if m := pyTrail.FindString(p); m != "" {
		post, p = "/.*$", p[:len(p)-len(m)]
	}
	re, err := globCompile(p, globOpts{dirOpt: true, classes: true}, pre, post)
	if err != nil {
		return nil, err
	}
	if base {
		return func(rel string) bool { return re.MatchString(path.Base(rel)) }, nil
	}
	return func(rel string) bool { return re.MatchString(rel) }, nil
}

func miniMatch(p, root string) (Matcher, error) {
	if strings.HasPrefix(p, "!") {
		return nil, errors.New("negated patterns are not supported")
	}
	re, err := anchored(relPattern(p, root), globOpts{dirOpt: true, classes: true, braces: true})
	if err != nil {
		return nil, err
	}
	return func(rel string) bool { return re.MatchString(rel) }, nil
}

// nanoGlob matches the full res:// path, as Nano Coverage's script scanner
// does; there ** is a plain .*, so **/ needs its slash.
func nanoGlob(p, _ string) (Matcher, error) {
	re, err := anchored(slashes(p), globOpts{classes: true})
	if err != nil {
		return nil, err
	}
	return func(rel string) bool { return re.MatchString("res://" + rel) }, nil
}

// lcovGlob is lcov's transform_pattern: * crosses slashes, brackets stay
// literal, no anchors, and the pattern is matched against the absolute path.
func lcovGlob(p, root string) (Matcher, error) {
	// Without classes or braces the translation cannot fail, and it quotes
	// every metacharacter, so the expression always compiles.
	body, _ := globRegexp(slashes(p), globOpts{crosses: true})
	re := regexp.MustCompile(body)
	root = trimRoot(root)
	return func(rel string) bool { return re.MatchString(root + "/" + rel) }, nil
}

// regexDialect (gcovr) tries the path relative to the root and the absolute
// one, anchored at the start as Python's re.match is. The pattern is left
// alone: a backslash in a regular expression is an escape, not a separator.
func regexDialect(p, root string) (Matcher, error) {
	re, err := regexp.Compile("^(?:" + p + ")")
	if err != nil {
		return nil, err
	}
	root = trimRoot(root)
	return func(rel string) bool { return re.MatchString(rel) || re.MatchString(root+"/"+rel) }, nil
}

// dirDialect (PHPUnit, coverage.py's source) names a directory or a file,
// never a pattern; "." and the root itself are the whole root.
func dirDialect(p, root string) (Matcher, error) {
	d := path.Clean(relPattern(p, root))
	if d == "." {
		return func(string) bool { return true }, nil
	}
	return func(rel string) bool { return rel == d || strings.HasPrefix(rel, d+"/") }, nil
}
