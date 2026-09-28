package convert

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// The converters' own counts, raised by hand when the output changes
// (header.py:11-17). The PDF count is 2: pdftotext writes another text than
// the reference's pypdf.
const (
	TranscriptConverter = "brain-transcript/1"
	PDFConverter        = "brain-pdf/2"
)

// The head lines are read with Python's \s and \S (header.py:25-26): a
// no-break space around a value is not part of it.
var youtubeID = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`\(([0-9A-Za-z_-]{11})\)`) })
var converterLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^converter:` + pytext.SpaceClass + `*(` + pytext.NonSpaceClass + `+)` + pytext.SpaceClass + `*$`)
})
var descriptionLine = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^description:` + pytext.SpaceClass + `*(` + pytext.NonSpaceClass + `.*?)` + pytext.SpaceClass + `*$`)
})

// SourceURLFrom is the source's URL as far as a file name yields one: an
// eleven-character parenthesised run is read as a YouTube id, even where it
// is none -- the length is all a name offers.
func SourceURLFrom(name string) string {
	if m := youtubeID().FindStringSubmatch(name); m != nil {
		return "https://www.youtube.com/watch?v=" + m[1]
	}
	return ""
}

// Head is the provenance head: four lines, five with a sentence from the
// local model. `asr` says whether speech recognition made the text, so that
// it never counts as a verbatim quote later.
type Head struct {
	SourceURL   string
	Retrieved   time.Time
	Converter   string
	ASR         bool
	Description string
}

// String is the head as YAML frontmatter with a blank line after. The
// description line is left out rather than written empty: an empty line
// would rewrite every file converted before the model existed.
func (h Head) String() string {
	var b strings.Builder
	b.WriteString("---\n")
	if h.SourceURL == "" {
		b.WriteString("source_url:\n")
	} else {
		fmt.Fprintf(&b, "source_url: %s\n", h.SourceURL)
	}
	fmt.Fprintf(&b, "retrieved: %s\nconverter: %s\nasr: %t\n", h.Retrieved.Format("2006-01-02"), h.Converter, h.ASR)
	if h.Description != "" {
		fmt.Fprintf(&b, "description: %s\n", h.Description)
	}
	b.WriteString("---\n\n")
	return b.String()
}

// frontmatter is the head block, or false where there is none to read.
func frontmatter(text string) (string, bool) {
	if !strings.HasPrefix(text, "---\n") {
		return "", false
	}
	end := strings.Index(text[3:], "\n---")
	if end == -1 {
		return "", false
	}
	return text[:3+end], true
}

// ConvertedBy names the converter that wrote a file. What names none was
// written by a person and is never overwritten.
func ConvertedBy(text string) (string, bool) { return headLine(text, converterLine()) }

// DescriptionOf is the sentence a head already carries; a second run reads
// it back rather than ask for another.
func DescriptionOf(text string) (string, bool) { return headLine(text, descriptionLine()) }

func headLine(text string, line *regexp.Regexp) (string, bool) {
	head, ok := frontmatter(text)
	if !ok {
		return "", false
	}
	m := line.FindStringSubmatch(head)
	if m == nil {
		return "", false
	}
	return m[1], true
}
