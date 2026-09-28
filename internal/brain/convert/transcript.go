package convert

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// threshold is the one number that carries both kinds of transcript: 1289
// fragments at 37 characters and 67 at 737 (transcript.py:12-15).
const threshold = 1200

type piece struct{ mark, text string }

// ToParagraphs joins the fragments of a transcript into paragraphs, each
// carrying the mark of its first fragment, without changing a word.
func ToParagraphs(text string, f Format) string { return toParagraphs(text, f, threshold) }

func toParagraphs(text string, f Format, limit int) string {
	var paragraphs, current []string
	mark := ""
	for _, p := range split(text, f) {
		if len(current) == 0 {
			mark = p.mark
		} else if p.mark != "" && mark == "" {
			// The paragraph in progress is the lead-in without a mark; the
			// first marked fragment starts its own.
			paragraphs = append(paragraphs, joined(mark, current))
			current = nil
			mark = p.mark
		}
		current = append(current, p.text)
		if size(current) > limit {
			paragraphs = append(paragraphs, joined(mark, current))
			current = nil
		}
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, joined(mark, current))
	}
	return strings.Join(paragraphs, "\n\n")
}

// split is re.split over the marks: the lead-in, then each mark with the
// speech up to the next one, the speech's whitespace pulled together.
func split(text string, f Format) []piece {
	pattern := bracketMark()
	if f == TranscriptRange {
		pattern = rangeMark()
	}
	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	var pieces []piece
	lead := text
	if len(matches) > 0 {
		lead = text[:matches[0][0]]
	}
	if l := pytext.Strip(lead); l != "" {
		pieces = append(pieces, piece{text: l})
	}
	for i, m := range matches {
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		if body := strings.Join(strings.FieldsFunc(text[m[1]:end], pytext.IsSpace), " "); body != "" {
			pieces = append(pieces, piece{mark: minutes(text[m[2]:m[3]]), text: body})
		}
	}
	return pieces
}

// minutes writes hh:mm:ss or mm:ss as mm:ss, hours turned into minutes: an
// anchor names the place in the video, not a time of day.
func minutes(raw string) string {
	parts := strings.Split(raw, ":")
	n := make([]int, len(parts))
	for i, part := range parts {
		n[i], _ = strconv.Atoi(part)
	}
	if len(n) == 3 {
		return fmt.Sprintf("%02d:%02d", n[0]*60+n[1], n[2])
	}
	return fmt.Sprintf("%02d:%02d", n[0], n[1])
}

// size is the length the reference measures: each part plus one.
func size(parts []string) int {
	total := 0
	for _, part := range parts {
		total += utf8.RuneCountInString(part) + 1
	}
	return total
}

func joined(mark string, parts []string) string {
	body := strings.Join(parts, " ")
	if mark == "" {
		return body
	}
	return "[" + mark + "] " + body
}
