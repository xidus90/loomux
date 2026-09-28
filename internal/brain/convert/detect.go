// Package convert turns what lands in an area's inbox into Markdown with a
// provenance head: transcripts joined into paragraphs, PDFs through
// Poppler's pdftotext, and a video's subtitles fetched through yt-dlp. It
// follows src/brain/convert/ of the reference.
package convert

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// Format is what a file in an inbox carries.
type Format int

const (
	Unsupported Format = iota
	TranscriptBracket
	TranscriptRange
	PDF
)

// headChars is how far detection reads: a timestamp that only shows up
// after that does not make the file a transcript (detect.py:17-20).
const headChars = 8192

// Both marks stand at the start of a line (detect.py:11-15); the range line
// ends in Python's \s. Digits are ASCII, where Python's \d takes every
// script's; a transcript in other digits is a row of the parity list.
var bracketMark = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^\[(\d{2}:\d{2}(?::\d{2})?)\]`)
})

var rangeMark = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^(\d{2}:\d{2}:\d{2}) - \d{2}:\d{2}:\d{2}` + pytext.SpaceClass + `*$`)
})

// Detect reads the ending first and the head of a .txt second: the
// transcripts on hand all end in .txt and still carry two kinds of mark.
// Unsupported is an answer, not an error.
func Detect(path string) Format {
	switch strings.ToLower(pySuffix(filepath.Base(path))) {
	case ".pdf":
		return PDF
	case ".txt":
	default:
		return Unsupported
	}
	head, ok := readHead(path)
	switch {
	case !ok:
		return Unsupported
	case rangeMark().MatchString(head):
		return TranscriptRange
	case bracketMark().MatchString(head):
		return TranscriptBracket
	}
	return Unsupported
}

// pySuffix is Path.suffix of Python 3.14 (pathlib/__init__.py:461): the
// leading dots stripped, then from the last dot on. `..txt` has none.
func pySuffix(name string) string {
	name = strings.TrimLeft(name, ".")
	if i := strings.LastIndex(name, "."); i != -1 {
		return name[i:]
	}
	return ""
}

// readHead is the first headChars characters in Python's text mode:
// universal newlines, and a byte that is not UTF-8 among them fails the
// read. Python decodes whole chunks and so also fails on a broken byte a
// little past the head; this reads no further than the head, a row of the
// parity list. Four bytes a character hold headChars characters in full, so
// no character the head needs is ever cut off by the buffer.
func readHead(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	buf := make([]byte, 4*headChars)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", false
	}
	data := buf[:n]
	var out strings.Builder
	for count := 0; len(data) > 0 && count < headChars; count++ {
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size <= 1 {
			return "", false
		}
		if r == '\r' {
			if len(data) > 1 && data[1] == '\n' {
				size = 2
			}
			r = '\n'
		}
		out.WriteRune(r)
		data = data[size:]
	}
	return out.String(), true
}
