package mutants

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// ifLine is an `if` whose body opens on the same line. `for` is not mutated
// by a1, a2 or a4: its condition is a loop bound, and striking it out hangs a
// run instead of failing it. Go's \s is ASCII where Python's is Unicode;
// gofmt indents with tabs, so no formatted line tells the two apart.
var ifLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^(\s*)if (.+) \{$`)
})

// comparisons flips equality against its negation and each ordering against
// its neighbour, longest first, so that `<=` is never read as `<`.
var comparisons = [][2]string{
	{"==", "!="},
	{"!=", "=="},
	{"<=", "<"},
	{">=", ">"},
	{"<", "<="},
	{">", ">="},
}

// Generate lists the mutants of one source file in the script's order: per
// line a1 true, a1 false, a4 and the a2 operands of `&&` then `||`, then a3
// per operator of comparisons and per column.
func Generate(path string, source []byte) []Mutant {
	var found []Mutant
	for index, span := range lineSpans(source) {
		number := index + 1
		line := string(source[span.start:span.end])
		if match := ifLine().FindStringSubmatch(line); match != nil {
			indent, condition := match[1], match[2]
			for _, r := range [][2]string{{"a1", "true"}, {"a1", "false"}, {"a4", "!(" + condition + ")"}} {
				found = append(found, Mutant{Family: r[0], Path: path, Line: number, Was: line, Now: indent + "if " + r[1] + " {"})
			}
			for _, operator := range []string{"&&", "||"} {
				for _, half := range splitTop(condition, operator) {
					found = append(found, Mutant{Family: "a2", Path: path, Line: number, Was: line, Now: indent + "if " + half + " {"})
				}
			}
		}
		for _, c := range comparisons {
			for _, column := range columns(line, c[0]) {
				found = append(found, Mutant{Family: "a3", Path: path, Line: number, Was: line, Now: line[:column] + c[1] + line[column+len(c[0]):]})
			}
		}
	}
	return found
}

// splitTop returns the operands of one boolean operator at the top level of a
// condition, or nil where the operator does not stand there. Depth and quoting
// are tracked because a `&&` inside a call's arguments or inside a string is
// not a half of this condition. The walk is over bytes where the script walks
// code points; every character it compares is ASCII, and no byte of a
// multi-byte UTF-8 sequence is.
func splitTop(condition, operator string) []string {
	var parts []string
	depth := 0
	var quote byte
	start := 0
	for index := 0; index < len(condition); {
		char := condition[index]
		switch {
		case quote != 0:
			if char == '\\' && quote != '`' {
				index += 2
				continue
			}
			if char == quote {
				quote = 0
			}
		case char == '"' || char == '\'' || char == '`':
			quote = char
		case char == '(' || char == '[' || char == '{':
			depth++
		case char == ')' || char == ']' || char == '}':
			depth--
		case depth == 0 && strings.HasPrefix(condition[index:], operator):
			parts = append(parts, condition[start:index])
			start = index + 2
			index += 2
			continue
		}
		index++
	}
	if parts == nil {
		return nil
	}
	parts = append(parts, condition[start:])
	for i, part := range parts {
		parts[i] = pytext.Strip(part)
	}
	return parts
}

// columns lists where an operator stands in a line as an operator: not as
// part of a longer one, not behind `//`, not inside a string by the count of
// double quotes before it. The empty-string tests are the script's: in Python
// "" is in every string, so an operator at the very start or the very end of
// the line is never counted.
func columns(line, operator string) []int {
	stripped, _, _ := strings.Cut(line, "//")
	var places []int
	for index := 0; ; index += len(operator) {
		found := strings.Index(stripped[index:], operator)
		if found < 0 {
			return places
		}
		index += found
		end := index + len(operator)
		after := stripped[end:min(end+1, len(stripped))]
		before := stripped[max(index-1, 0):index]
		neighbours := strings.Contains("=", after) || (strings.Contains("<>", operator) && strings.Contains("<>=!", before))
		if !neighbours && strings.Count(stripped[:index], `"`)%2 == 0 {
			places = append(places, index)
		}
	}
}

// span is one line of a source without its line break.
type span struct{ start, end int }

// lineSpans cuts a source where Python's str.splitlines cuts what
// Path.read_text returns: at \r\n, \n, \r, \v, \f, \x1c, \x1d, \x1e, U+0085,
// U+2028 and U+2029, with no empty piece after a final break. Line numbers
// then name the line the script names.
func lineSpans(source []byte) []span {
	var spans []span
	start := 0
	for index := 0; index < len(source); {
		width := lineBreak(source[index:])
		if width == 0 {
			index++
			continue
		}
		spans = append(spans, span{start, index})
		index += width
		start = index
	}
	if start < len(source) {
		spans = append(spans, span{start, len(source)})
	}
	return spans
}

// lineBreak is the width of the line break head opens with, or 0. Every break
// is ASCII or opens with a UTF-8 lead byte, so a walk over bytes finds the
// breaks a walk over code points finds.
func lineBreak(head []byte) int {
	switch {
	case bytes.HasPrefix(head, []byte("\r\n")), bytes.HasPrefix(head, []byte("\xc2\x85")):
		return 2
	case bytes.HasPrefix(head, []byte("\xe2\x80\xa8")), bytes.HasPrefix(head, []byte("\xe2\x80\xa9")):
		return 3
	case bytes.IndexByte([]byte("\n\r\v\f\x1c\x1d\x1e"), head[0]) >= 0:
		return 1
	}
	return 0
}

// mutate returns the source with the mutant's line replaced and every other
// byte kept, its line break included. The script writes the file back
// through read_text and so turns CRLF into LF; an overlay has no reason to.
func mutate(source []byte, m Mutant) []byte {
	line := lineSpans(source)[m.Line-1]
	return slices.Concat(source[:line.start], []byte(m.Now), source[line.end:])
}
