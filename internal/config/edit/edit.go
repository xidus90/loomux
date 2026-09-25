// Package edit changes .loomux/config.toml as text, so every comment and
// every line it does not touch stays as the human wrote it. What it cannot
// place without guessing, it refuses; the human edits that by hand.
package edit

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// ErrAmbiguous marks a document whose shape would make any placement a guess.
var ErrAmbiguous = errors.New("cannot place this change without guessing; edit the file by hand")

type lineForms struct{ header, listHeader, bareKey, otherKey *regexp.Regexp }

// forms compiles on first use: this package is linked into the binary every
// hook runs, and a package-level compile would run on each of those starts.
var forms = sync.OnceValue(func() lineForms {
	return lineForms{
		header:     regexp.MustCompile(`^\s*\[([A-Za-z0-9_.-]+)\]\s*(#.*)?$`),
		listHeader: regexp.MustCompile(`^\s*\[\[([A-Za-z0-9_.-]+)\]\]\s*(#.*)?$`),
		bareKey:    regexp.MustCompile(`^\s*([A-Za-z0-9_-]+)\s*=`),
		otherKey:   regexp.MustCompile(`^\s*("[^"]*"|'[^']*'|[A-Za-z0-9_-]+\.[A-Za-z0-9_.-]+)\s*=`),
	}
})

// entry is what one line opens; a multi-line value spans several lines and
// is kept together as one entry.
type entry struct {
	first, last int    // line indexes, inclusive
	section     string // the [table] it sits in; "" before the first header
	key         string // "" for headers, comments and blank lines
	isHeader    bool
	isList      bool
}

// byteOrderMark is the UTF-8 mark some Windows editors put before the first
// line. TOML readers skip it; the line forms would read it as part of line 1.
const byteOrderMark = "\uFEFF"

func split(text string) (bom string, lines []string, eol string) {
	if strings.HasPrefix(text, byteOrderMark) {
		bom, text = byteOrderMark, text[len(byteOrderMark):]
	}
	body := strings.TrimSuffix(text, "\n")
	if body == "" {
		return bom, nil, "\n"
	}
	lines = strings.Split(body, "\n")
	// Only a file whose every line ends in \r\n is a CRLF file, and its
	// lines lose the \r here and get it back in join, new lines included.
	// A file with both endings keeps each \r on its own line, so an
	// untouched line comes back as it was and no line is glued to the next.
	// A last line without an ending of its own gives no vote.
	ended := lines
	if !strings.HasSuffix(text, "\n") {
		ended = lines[:len(lines)-1]
	}
	if len(ended) == 0 {
		return bom, lines, "\n"
	}
	for _, l := range ended {
		if !strings.HasSuffix(l, "\r") {
			return bom, lines, "\n"
		}
	}
	for i, l := range lines {
		lines[i] = strings.TrimSuffix(l, "\r")
	}
	return bom, lines, "\r\n"
}

func join(bom string, lines []string, eol string) string {
	if len(lines) == 0 {
		return bom
	}
	return bom + strings.Join(lines, eol) + eol
}

// scan walks the lines once. It refuses a document it cannot map: a dotted,
// quoted or inline-table key could hold keys of the section being changed or
// make it a value, a multi-line string hides lines that look like headers,
// and a line of no known form could be either.
func scan(lines []string, section string) ([]entry, error) {
	var out []entry
	current := ""
	f := forms()
	refuse := func(i int) ([]entry, error) {
		return nil, fmt.Errorf("line %d: %w", i+1, ErrAmbiguous)
	}
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		switch {
		case f.listHeader.MatchString(l):
			// The keys of a [[name]] entry are not keys of [name]; the
			// brackets keep them out of that section.
			name := f.listHeader.FindStringSubmatch(l)[1]
			current = "[[" + name + "]]"
			out = append(out, entry{first: i, last: i, section: name, isHeader: true, isList: true})
		case f.header.MatchString(l):
			current = f.header.FindStringSubmatch(l)[1]
			out = append(out, entry{first: i, last: i, section: current, isHeader: true})
		case f.otherKey.MatchString(l):
			m := f.otherKey.FindStringSubmatch(l)
			if related(qualify(current, strings.Trim(m[1], `"'`)), section) || tripleQuote(l[len(m[0]):]) {
				return refuse(i)
			}
			out = append(out, entry{first: i, last: i, section: current})
		case f.bareKey.MatchString(l):
			m := f.bareKey.FindStringSubmatch(l)
			key, value := m[1], l[len(m[0]):]
			full := qualify(current, key)
			// A plain key of the section is what Set edits; the same key
			// holding an inline table, or a key on the path to the section,
			// would give the section keys this scan cannot see.
			inline := strings.HasPrefix(strings.TrimSpace(value), "{")
			if related(full, section) && (inline || !strings.HasPrefix(full, section+".")) {
				return refuse(i)
			}
			if tripleQuote(value) {
				return refuse(i)
			}
			last := i
			depth := brackets(value)
			for depth > 0 && last+1 < len(lines) {
				last++
				if tripleQuote(lines[last]) {
					return refuse(last)
				}
				depth += brackets(lines[last])
			}
			out = append(out, entry{first: i, last: last, section: current, key: key})
			i = last
		default:
			if t := strings.TrimSpace(l); t != "" && !strings.HasPrefix(t, "#") {
				return refuse(i)
			}
			out = append(out, entry{first: i, last: i, section: current})
		}
	}
	return out, nil
}

// qualify is the full dotted name of key written under the table current.
func qualify(current, key string) string {
	if current == "" {
		return key
	}
	return current + "." + key
}

// related reports whether a key named full lies in, is, or is on the path to
// section.
func related(full, section string) bool {
	return full == section || strings.HasPrefix(full, section+".") || strings.HasPrefix(section, full+".")
}

// tripleQuote reports whether s opens a multi-line string (three quotes of
// either kind) outside a comment; its lines would then read as headers and
// keys that are none.
func tripleQuote(s string) bool {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			if strings.HasPrefix(s[i:], strings.Repeat(string(c), 3)) {
				return true
			}
			quote = c
		case c == '#':
			return false
		}
	}
	return false
}

// brackets is the change in [ ] depth over s, outside strings and comments.
func brackets(s string) int {
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return depth
		case c == '[':
			depth++
		case c == ']':
			depth--
		}
	}
	return depth
}

// comment is the trailing comment of a one-line value, with the spaces before it.
func comment(l string) string {
	var quote byte
	for i := strings.Index(l, "=") + 1; i < len(l); i++ {
		c := l[i]
		switch {
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			j := i
			for j > 0 && (l[j-1] == ' ' || l[j-1] == '\t') {
				j--
			}
			return l[j:]
		}
	}
	return ""
}

// locate finds the [section] headers, the entries of key in it, and the last
// line a new key of that section goes after.
func locate(entries []entry, section, key string) (headers []entry, hits []entry, lastInSection int) {
	lastInSection = -1
	for _, e := range entries {
		if e.isHeader && e.section == section && !e.isList {
			headers = append(headers, e)
			lastInSection = max(lastInSection, e.last)
		}
		if !e.isHeader && e.section == section && e.key != "" {
			lastInSection = e.last
			if e.key == key {
				hits = append(hits, e)
			}
		}
	}
	return headers, hits, lastInSection
}

// Set writes key = literal into [section]: over the key where it stands,
// after the last key of the section where it does not, and in a new section
// at the end where the section is missing.
func Set(text, section, key, literal string) (string, error) {
	bom, lines, eol := split(text)
	entries, err := scan(lines, section)
	if err != nil {
		return "", err
	}
	headers, hits, last := locate(entries, section, key)
	if len(headers) > 1 || len(hits) > 1 {
		return "", ErrAmbiguous
	}
	line := key + " = " + literal
	switch {
	case len(hits) == 1:
		hit := hits[0]
		// The indentation and the spacing around = are the human's; only
		// the value is new.
		old := lines[hit.first]
		cut := strings.IndexByte(old, '=') + 1
		for cut < len(old) && (old[cut] == ' ' || old[cut] == '\t') {
			cut++
		}
		line = old[:cut] + literal
		// A comment inside a multi-line value belongs to an item the new
		// value may not have; only a one-line value keeps its comment.
		if hit.first == hit.last {
			line += comment(old)
		}
		lines = append(lines[:hit.first], append([]string{line}, lines[hit.last+1:]...)...)
	case len(headers) == 1:
		lines = append(lines[:last+1], append([]string{line}, lines[last+1:]...)...)
	default:
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "["+section+"]", line)
	}
	return join(bom, lines, eol), nil
}

// Remove takes key out of [section]; a section left with no key goes too,
// together with the blank line before it.
func Remove(text, section, key string) (string, error) {
	bom, lines, eol := split(text)
	entries, err := scan(lines, section)
	if err != nil {
		return "", err
	}
	headers, hits, _ := locate(entries, section, key)
	if len(headers) > 1 || len(hits) > 1 {
		return "", ErrAmbiguous
	}
	if len(hits) == 0 {
		return text, nil
	}
	hit := hits[0]
	lines = append(lines[:hit.first], lines[hit.last+1:]...)
	// Dropping a line cannot make the document ambiguous; the scan that
	// passed above passes again.
	rest, _ := scan(lines, section)
	keys := 0
	for _, e := range rest {
		if !e.isHeader && e.section == section && e.key != "" {
			keys++
		}
	}
	if keys == 0 && len(headers) == 1 {
		h := headers[0].first
		start := h
		if start > 0 && strings.TrimSpace(lines[start-1]) == "" {
			start--
		}
		lines = append(lines[:start], lines[h+1:]...)
	}
	return join(bom, lines, eol), nil
}

// AppendBlock adds one [[section]] entry at the end.
func AppendBlock(text, section string, pairs [][2]string) string {
	bom, lines, eol := split(text)
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, "[["+section+"]]")
	for _, p := range pairs {
		lines = append(lines, p[0]+" = "+p[1])
	}
	return join(bom, lines, eol)
}
