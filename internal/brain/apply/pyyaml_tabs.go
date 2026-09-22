package apply

// pyyaml_tabs.go refuses the tabs PyYAML's scanner refuses and yaml.v3's
// does not. PyYAML separates tokens with spaces only (`scan_to_next_token`,
// `scan_plain_spaces`), so a tab is "a character that cannot start any
// token" everywhere except in three places, measured on the reference:
// inside a quoted scalar, inside a comment, and in the body of a block
// scalar once a line has reached the block's indentation. yaml.v3 accepts
// more (`a:\tb`, `c: "x"\t`), and a page it reads the reference refuses.
//
// yaml.v3 keeps only where a node starts, so the spans are found again in
// the text: a quoted scalar runs from its opening quote to its closing one,
// a block scalar's body is measured the way `scan_block_scalar` measures it.

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// errTab is PyYAML's complaint, without its marks.
var errTab = errors.New(`found character '\t' that cannot start any token`)

// span is a half-open byte range of the frontmatter in which a tab is text.
type span struct{ start, end int }

// checkTabs answers errTab when text holds a tab PyYAML would scan as
// separation.
func checkTabs(text string, document *yaml.Node) error {
	if !strings.Contains(text, "\t") {
		return nil
	}
	lines := lineStarts(text)
	var spans []span
	var walk func(node *yaml.Node, parentIndent int)
	walk = func(node *yaml.Node, parentIndent int) {
		switch node.Kind {
		case yaml.ScalarNode:
			if s, ok := scalarSpan(text, lines, node, parentIndent); ok {
				spans = append(spans, s)
			}
		case yaml.MappingNode, yaml.SequenceNode:
			// A block collection's indent is the column its first key or
			// dash stands at; PyYAML's `self.indent` for everything inside.
			indent := node.Column - 1
			for _, child := range node.Content {
				walk(child, indent)
			}
		}
	}
	for _, child := range document.Content {
		walk(child, -1)
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	next := 0
	for i := 0; i < len(text); i++ {
		for next < len(spans) && spans[next].end <= i {
			next++
		}
		if next < len(spans) && spans[next].start <= i {
			i = spans[next].end - 1
			continue
		}
		switch text[i] {
		case '#':
			if i == 0 || text[i-1] == ' ' || breakEndsAt(text, i) || (next > 0 && spans[next-1].end == i) {
				for i < len(text) && breakLen(text, i) == 0 {
					i++
				}
			}
		case '\t':
			return errTab
		}
	}
	return nil
}

// The line breaks yaml.v3 counts and PyYAML's scanner reads, in UTF-8: CRLF
// as one, a lone CR, LF, NEL, LS and PS. Counting fewer puts a node's line
// on the wrong text, or past the end of it.
const (
	nel = "\xc2\x85"
	ls  = "\xe2\x80\xa8"
	ps  = "\xe2\x80\xa9"
)

// breakLen is the length in bytes of the line break at text[i], 0 if there
// is none.
func breakLen(text string, i int) int {
	rest := text[i:]
	switch {
	case strings.HasPrefix(rest, "\r\n"):
		return 2
	case rest[0] == '\r' || rest[0] == '\n':
		return 1
	case strings.HasPrefix(rest, nel):
		return len(nel)
	case strings.HasPrefix(rest, ls) || strings.HasPrefix(rest, ps):
		return len(ls)
	}
	return 0
}

// breakEndsAt says whether a line break ends just before text[i], so that i
// starts a line.
func breakEndsAt(text string, i int) bool {
	before := text[:i]
	return strings.HasSuffix(before, "\n") || strings.HasSuffix(before, "\r") ||
		strings.HasSuffix(before, nel) || strings.HasSuffix(before, ls) || strings.HasSuffix(before, ps)
}

// nextBreak is the offset of the first line break at or after i and its
// length, or len(text) and 0.
func nextBreak(text string, i int) (int, int) {
	for ; i < len(text); i++ {
		if n := breakLen(text, i); n > 0 {
			return i, n
		}
	}
	return len(text), 0
}

// lineStarts is the byte offset of each line's first byte.
func lineStarts(text string) []int {
	starts := []int{0}
	for i := 0; i < len(text); {
		if n := breakLen(text, i); n > 0 {
			i += n
			starts = append(starts, i)
			continue
		}
		i++
	}
	return starts
}

// offset turns yaml.v3's 1-based line and rune column into a byte offset.
func offset(text string, lines []int, line, column int) int {
	at := lines[line-1]
	for c := 1; c < column; c++ {
		_, size := utf8.DecodeRuneInString(text[at:])
		at += size
	}
	return at
}

// scalarSpan is the stretch of a quoted scalar, or the body of a block
// scalar, in which a tab is text. A plain scalar has none.
func scalarSpan(text string, lines []int, node *yaml.Node, parentIndent int) (span, bool) {
	start := offset(text, lines, node.Line, node.Column)
	switch {
	case node.Style&yaml.SingleQuotedStyle != 0:
		open := start + strings.IndexByte(text[start:], '\'')
		end := open + 1
		for end < len(text) {
			if text[end] == '\'' {
				if end+1 < len(text) && text[end+1] == '\'' {
					end += 2
					continue
				}
				break
			}
			end++
		}
		return span{open, end + 1}, true
	case node.Style&yaml.DoubleQuotedStyle != 0:
		open := start + strings.IndexByte(text[start:], '"')
		end := open + 1
		for end < len(text) && text[end] != '"' {
			if text[end] == '\\' {
				end++
			}
			end++
		}
		return span{open, end + 1}, true
	case node.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0:
		return blockBody(text, start, parentIndent), true
	}
	return span{}, false
}

// blockBody is `scan_block_scalar` over the lines after the header that
// starts at or after start: the body is every line up to the first one that
// has fewer spaces than the block's indentation and then something other
// than a line break. On a body line only what stands past the indentation is
// text; the spaces before it hold no tab by construction.
func blockBody(text string, start, parentIndent int) span {
	header := start + strings.IndexAny(text[start:], "|>")
	headerEnd, width := nextBreak(text, header)
	if width == 0 {
		return span{len(text), len(text)}
	}
	lineEnd := headerEnd - header
	bodyStart := headerEnd + width
	minIndent := max(parentIndent+1, 1)
	indent := 0
	if digit := strings.IndexAny(text[header+1:header+min(3, lineEnd)], "123456789"); digit >= 0 {
		indent = minIndent + int(text[header+1+digit]-'0') - 1
	} else {
		// `scan_block_scalar_indentation`: the deepest run of spaces before
		// the first character that is neither a space nor a break.
		maxIndent, column := 0, 0
		for i := bodyStart; i < len(text); {
			if n := breakLen(text, i); n > 0 {
				column = 0
				i += n
				continue
			}
			if text[i] != ' ' {
				break
			}
			column++
			maxIndent = max(maxIndent, column)
			i++
		}
		indent = max(minIndent, maxIndent)
	}
	end := bodyStart
	for end < len(text) {
		spaces := 0
		for end+spaces < len(text) && text[end+spaces] == ' ' && spaces < indent {
			spaces++
		}
		at := end + spaces
		if at < len(text) && breakLen(text, at) == 0 && spaces < indent {
			break
		}
		lineEnd, width := nextBreak(text, at)
		end = lineEnd + width
	}
	return span{bodyStart, end}
}
