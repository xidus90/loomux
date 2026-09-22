package apply

// pyyaml_emit.go is PyYAML's Emitter (emitter.py, 6.0.3) for what SafeDumper
// hands it from a safe_load result with `default_flow_style=False` and
// `allow_unicode=True`: width 80, indent 2, LF breaks, no document markers.
// Two simplifications follow from that input and are the only ones:
//
//   - A collection is written in flow style only when it is empty, so the
//     flow level at any scalar is 0 and a flow collection is `[]` or `{}`.
//   - The representer never asks for a scalar style, so no literal or folded
//     block is written; and every non-str scalar the representer makes
//     (`null`, `true`, `12`, `-.inf`, `2026-09-22 08:16:27+00:00`, ...) passes
//     the plain analysis, so it is always plain.
//
// Columns count code points, as Python's `len` of a str does.

import "strings"

const (
	bestWidth  = 80
	bestIndent = 2
)

type emitter struct {
	out        strings.Builder
	indent     int // -1 is Python's None
	indents    []int
	column     int
	whitespace bool
	indention  bool
}

// dumpYAML is `yaml.safe_dump(value, sort_keys=False, allow_unicode=True,
// default_flow_style=False)` of a mapping.
func dumpYAML(value *pyValue) string {
	e := &emitter{indent: -1, whitespace: true, indention: true}
	e.node(value, false, false)
	e.writeIndent() // expect_document_end
	return e.out.String()
}

// increaseIndent is `increase_indent`. Its arm for a flow node at the root,
// which starts at indent 2, is left out: the root here is always a
// non-empty block mapping.
func (e *emitter) increaseIndent(indentless bool) {
	e.indents = append(e.indents, e.indent)
	switch {
	case e.indent < 0:
		e.indent = 0
	case !indentless:
		e.indent += bestIndent
	}
}

func (e *emitter) popIndent() {
	e.indent = e.indents[len(e.indents)-1]
	e.indents = e.indents[:len(e.indents)-1]
}

// node is `expect_node`; mapping is its mapping_context and simpleKey its
// simple_key_context.
func (e *emitter) node(value *pyValue, mapping, simpleKey bool) {
	switch value.kind {
	case pyList:
		if len(value.items) == 0 {
			e.writeIndicator("[", true, true, false)
			e.writeIndicator("]", false, false, false)
			return
		}
		e.increaseIndent(mapping && !e.indention)
		for _, item := range value.items {
			e.writeIndent()
			e.writeIndicator("-", true, false, true)
			e.node(item, false, false)
		}
		e.popIndent()
	case pyDict:
		if len(value.keys) == 0 {
			e.writeIndicator("{", true, true, false)
			e.writeIndicator("}", false, false, false)
			return
		}
		e.increaseIndent(false)
		for i, key := range value.keys {
			e.writeIndent()
			if checkSimpleKey(key) {
				e.node(key, true, true)
				e.writeIndicator(":", false, false, false)
			} else {
				e.writeIndicator("?", true, false, true)
				e.node(key, true, false)
				e.writeIndent()
				e.writeIndicator(":", true, false, true)
			}
			e.node(value.values[i], true, false)
		}
		e.popIndent()
	default:
		e.increaseIndent(false)
		e.scalar(value, simpleKey)
		e.popIndent()
	}
}

// shortTags are `prepare_tag` of each scalar's tag; check_simple_key counts
// them into the length of a key even though the tag is never written.
var shortTags = map[pyKind]int{
	pyNone:     len("!!null"),
	pyBool:     len("!!bool"),
	pyInt:      len("!!int"),
	pyFloat:    len("!!float"),
	pyStr:      len("!!str"),
	pyDate:     len("!!timestamp"),
	pyDateTime: len("!!timestamp"),
}

// checkSimpleKey is `check_simple_key` for a scalar key; a key is never a
// collection, since safe_load refuses an unhashable one.
func checkSimpleKey(key *pyValue) bool {
	a := analyzeScalar(key.text)
	return len([]rune(key.text))+shortTags[key.kind] < 128 && !a.empty && !a.multiline
}

// scalar is `choose_scalar_style` and `process_scalar`.
func (e *emitter) scalar(value *pyValue, simpleKey bool) {
	text := []rune(value.text)
	split := !simpleKey
	if value.kind != pyStr {
		e.writePlain(text, split)
		return
	}
	a := analyzeScalar(value.text)
	switch {
	case implicitTag(value.text) == "str" && !(simpleKey && (a.empty || a.multiline)) && a.allowBlockPlain:
		e.writePlain(text, split)
	case a.allowSingleQuoted && !(simpleKey && a.multiline):
		e.writeSingleQuoted(text, split)
	default:
		e.writeDoubleQuoted(text, split)
	}
}

type analysis struct {
	empty, multiline                   bool
	allowBlockPlain, allowSingleQuoted bool
}

func isBreak(ch rune) bool { return ch == 0x0a || ch == 0x85 || ch == 0x2028 || ch == 0x2029 }

func isBlank(ch rune) bool { return ch == 0 || ch == ' ' || ch == '\t' || ch == '\r' || isBreak(ch) }

// analyzeScalar is `analyze_scalar` with `allow_unicode=True`, down to the
// answers a block-context dump asks.
func analyzeScalar(scalar string) analysis {
	if scalar == "" {
		return analysis{empty: true, allowBlockPlain: true, allowSingleQuoted: true}
	}
	text := []rune(scalar)
	blockIndicators := strings.HasPrefix(scalar, "---") || strings.HasPrefix(scalar, "...")
	var lineBreaks, special bool
	var leadingSpace, leadingBreak, trailingSpace, trailingBreak, breakSpace, spaceBreak bool
	precededByWhitespace := true
	followedByWhitespace := len(text) == 1 || isBlank(text[1])
	previousSpace, previousBreak := false, false
	for index, ch := range text {
		if index == 0 {
			if strings.ContainsRune("#,[]{}&*!|>'\"%@`", ch) {
				blockIndicators = true
			}
			if (ch == '?' || ch == ':' || ch == '-') && followedByWhitespace {
				blockIndicators = true
			}
		} else if (ch == ':' && followedByWhitespace) || (ch == '#' && precededByWhitespace) {
			blockIndicators = true
		}
		if isBreak(ch) {
			lineBreaks = true
		}
		if !(ch == '\n' || (ch >= 0x20 && ch <= 0x7e)) {
			printable := ch == 0x85 || (ch >= 0xa0 && ch <= 0xd7ff) || (ch >= 0xe000 && ch <= 0xfffd) || (ch >= 0x10000 && ch < 0x10ffff)
			if !printable || ch == 0xfeff {
				special = true
			}
		}
		switch {
		case ch == ' ':
			leadingSpace = leadingSpace || index == 0
			trailingSpace = trailingSpace || index == len(text)-1
			breakSpace = breakSpace || previousBreak
			previousSpace, previousBreak = true, false
		case isBreak(ch):
			leadingBreak = leadingBreak || index == 0
			trailingBreak = trailingBreak || index == len(text)-1
			spaceBreak = spaceBreak || previousSpace
			previousSpace, previousBreak = false, true
		default:
			previousSpace, previousBreak = false, false
		}
		precededByWhitespace = isBlank(ch)
		followedByWhitespace = index+2 >= len(text) || isBlank(text[index+2])
	}
	a := analysis{multiline: lineBreaks, allowBlockPlain: true, allowSingleQuoted: true}
	if leadingSpace || leadingBreak || trailingSpace || trailingBreak || lineBreaks || blockIndicators {
		a.allowBlockPlain = false
	}
	if breakSpace || spaceBreak || special {
		a.allowBlockPlain, a.allowSingleQuoted = false, false
	}
	return a
}

// writeIndicator is `write_indicator`.
func (e *emitter) writeIndicator(indicator string, needWhitespace, whitespace, indention bool) {
	data := indicator
	if !e.whitespace && needWhitespace {
		data = " " + indicator
	}
	e.whitespace = whitespace
	e.indention = e.indention && indention
	e.column += len([]rune(data))
	e.out.WriteString(data)
}

// writeIndent is `write_indent`.
func (e *emitter) writeIndent() {
	indent := max(e.indent, 0)
	if !e.indention || e.column > indent || (e.column == indent && !e.whitespace) {
		e.writeLineBreak("\n")
	}
	if e.column < indent {
		e.whitespace = true
		e.out.WriteString(strings.Repeat(" ", indent-e.column))
		e.column = indent
	}
}

// writeLineBreak is `write_line_break`.
func (e *emitter) writeLineBreak(data string) {
	e.whitespace = true
	e.indention = true
	e.column = 0
	e.out.WriteString(data)
}

// writeText writes a run of text and counts its code points.
func (e *emitter) writeText(text []rune) {
	e.column += len(text)
	e.out.WriteString(string(text))
}

// writeBreaks writes the breaks of text as `write_single_quoted` and
// `write_plain` do: a leading LF once more, since a single break folds away
// when read back.
func (e *emitter) writeBreaks(text []rune) {
	if text[0] == '\n' {
		e.writeLineBreak("\n")
	}
	for _, br := range text {
		e.writeLineBreak(string(br))
	}
	e.writeIndent()
}

// writePlain is `write_plain` for a non-empty scalar with no break, which is
// all the analysis lets through.
func (e *emitter) writePlain(text []rune, split bool) {
	if !e.whitespace {
		e.writeText([]rune(" "))
	}
	e.whitespace = false
	e.indention = false
	spaces := false
	start := 0
	for end := 0; end <= len(text); end++ {
		var ch rune = -1
		if end < len(text) {
			ch = text[end]
		}
		if spaces {
			if ch != ' ' {
				if start+1 == end && e.column > bestWidth && split {
					e.writeIndent()
					e.whitespace = false
					e.indention = false
				} else {
					e.writeText(text[start:end])
				}
				start = end
			}
		} else if ch == -1 || ch == ' ' {
			e.writeText(text[start:end])
			start = end
		}
		spaces = ch == ' '
	}
}

// writeSingleQuoted is `write_single_quoted`.
func (e *emitter) writeSingleQuoted(text []rune, split bool) {
	e.writeIndicator("'", true, false, false)
	spaces, breaks := false, false
	start := 0
	for end := 0; end <= len(text); end++ {
		var ch rune = -1
		if end < len(text) {
			ch = text[end]
		}
		switch {
		case spaces:
			if ch != ' ' {
				if start+1 == end && e.column > bestWidth && split && start != 0 && end != len(text) {
					e.writeIndent()
				} else {
					e.writeText(text[start:end])
				}
				start = end
			}
		case breaks:
			if ch == -1 || !isBreak(ch) {
				e.writeBreaks(text[start:end])
				start = end
			}
		default:
			if (ch == -1 || ch == ' ' || isBreak(ch) || ch == '\'') && start < end {
				e.writeText(text[start:end])
				start = end
			}
		}
		if ch == '\'' {
			e.writeText([]rune("''"))
			start = end + 1
		}
		if ch != -1 {
			spaces = ch == ' '
			breaks = isBreak(ch)
		}
	}
	e.writeIndicator("'", false, false, false)
}

// escapes is `ESCAPE_REPLACEMENTS`.
var escapes = map[rune]string{
	0: "0", 0x07: "a", 0x08: "b", 0x09: "t", 0x0a: "n", 0x0b: "v", 0x0c: "f",
	0x0d: "r", 0x1b: "e", '"': "\"", '\\': "\\", 0x85: "N", 0xa0: "_",
	0x2028: "L", 0x2029: "P",
}

// writeDoubleQuoted is `write_double_quoted` with `allow_unicode=True`.
func (e *emitter) writeDoubleQuoted(text []rune, split bool) {
	e.writeIndicator("\"", true, false, false)
	start := 0
	for end := 0; end <= len(text); end++ {
		var ch rune = -1
		if end < len(text) {
			ch = text[end]
		}
		plain := (ch >= 0x20 && ch <= 0x7e) || (ch >= 0xa0 && ch <= 0xd7ff) || (ch >= 0xe000 && ch <= 0xfffd)
		if ch == -1 || ch == '"' || ch == '\\' || ch == 0x85 || ch == 0x2028 || ch == 0x2029 || ch == 0xfeff || !plain {
			if start < end {
				e.writeText(text[start:end])
				start = end
			}
			if ch != -1 {
				e.writeText([]rune(escape(ch)))
				start = end + 1
			}
		}
		if 0 < end && end < len(text)-1 && (ch == ' ' || start >= end) && e.column+(end-start) > bestWidth && split {
			// Python's text[start:end] is empty when an escape has moved
			// start one past end; a Go slice would panic.
			data := []rune{'\\'}
			if start < end {
				data = append(append([]rune{}, text[start:end]...), '\\')
				start = end
			}
			e.writeText(data)
			e.writeIndent()
			e.whitespace = false
			e.indention = false
			if text[start] == ' ' {
				e.writeText([]rune("\\"))
			}
		}
	}
	e.writeIndicator("\"", false, false, false)
}

// escape is the escape sequence `write_double_quoted` writes for ch.
func escape(ch rune) string {
	if short, ok := escapes[ch]; ok {
		return "\\" + short
	}
	switch {
	case ch <= 0xff:
		return "\\x" + hex(ch, 2)
	case ch <= 0xffff:
		return "\\u" + hex(ch, 4)
	default:
		return "\\U" + hex(ch, 8)
	}
}

func hex(ch rune, width int) string {
	const digits = "0123456789ABCDEF"
	out := make([]byte, width)
	for i := width - 1; i >= 0; i-- {
		out[i] = digits[ch&0xf]
		ch >>= 4
	}
	return string(out)
}
