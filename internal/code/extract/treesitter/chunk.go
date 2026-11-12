package treesitter

import "bytes"

// A tree can lose the rest of a file: gotreesitter can end it early, and a
// grammar can wrap whole definitions in one ERROR. A language package then
// reads that rest again in chunks, each parsed alone at its own offsets:
// Chunks says where they start, ReparseChunk parses one. Which line can
// open a definition is the language's to say.

// Chunks are where the chunks of the text after the line holding end
// start: every later line whose first byte opens says can open a
// definition, and a chunk runs to the next start. More than max starts are
// thinned to every k-th. Text before the first start is not read.
func Chunks(src []byte, end uint32, max int, opens func(pos int) bool) []int {
	var starts []int
	for pos := int(end); ; {
		nl := bytes.IndexByte(src[pos:], '\n')
		if nl < 0 {
			break
		}
		pos += nl + 1
		if pos < len(src) && opens(pos) {
			starts = append(starts, pos)
		}
	}
	if len(starts) <= max {
		return starts
	}
	k := (len(starts) + max - 1) / max
	thinned := make([]int, 0, max)
	for i := 0; i < len(starts); i += k {
		thinned = append(thinned, starts[i])
	}
	return thinned
}

// ReparseChunk parses chunk i of src into the document, everything before
// it blanked and everything after it left off; Reparse says what it keeps.
func (d *Doc) ReparseChunk(src []byte, starts []int, i int) error {
	end := len(src)
	if i+1 < len(starts) {
		end = starts[i+1]
	}
	chunk := bytes.Clone(src[:end])
	Blank(chunk[:starts[i]])
	return d.Reparse(chunk)
}

// Blank overwrites b with spaces, keeping its line breaks: what a parse
// should not see, at offsets that do not move.
func Blank(b []byte) {
	for i, c := range b {
		if c != '\n' && c != '\r' {
			b[i] = ' '
		}
	}
}

// LineEnd is the offset of the line break after from, or the end of src.
func LineEnd(src []byte, from int) int {
	if nl := bytes.IndexByte(src[from:], '\n'); nl >= 0 {
		return from + nl
	}
	return len(src)
}

// Closing is the offset after the next end at or after from, or the end of
// src.
func Closing(src []byte, from int, end string) int {
	if k := bytes.Index(src[from:], []byte(end)); k >= 0 {
		return from + k + len(end)
	}
	return len(src)
}

// QuotedEnd is the offset after the quote that closes the literal opening
// at src[open], honouring backslash escapes, or its line's end when it has
// none.
func QuotedEnd(src []byte, open int) int {
	for i := open + 1; i < len(src); i++ {
		switch src[i] {
		case '\\':
			i++
		case src[open]:
			return i + 1
		case '\n':
			return i
		}
	}
	return len(src)
}
