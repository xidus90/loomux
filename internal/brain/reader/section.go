package reader

import (
	"fmt"
	"strings"

	"github.com/xidus90/loomux/internal/brain/pytext"
)

// ExtractSection extracts a heading and its content up to the next heading
// of equal or higher level.
//
// Matches headings where line starts with "#" and the stripped text matches heading.
// The lines, the strip and the quoting are Python's (core._section): splitlines,
// str.strip, str.rstrip and repr.
func ExtractSection(content, heading string) (string, error) {
	lines := pytext.SplitLines(content)

	start := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			title := pytext.Strip(strings.TrimLeft(line, "#"))
			if title == heading {
				start = i
				break
			}
		}
	}

	if start == -1 {
		return "", fmt.Errorf("no section titled %s", pytext.Repr(heading))
	}

	level := headingLevel(lines[start])
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "#") {
			l := headingLevel(lines[i])
			if l <= level {
				end = i
				break
			}
		}
	}

	sectionLines := lines[start:end]
	joined := strings.Join(sectionLines, "\n")
	return pytext.RStrip(joined) + "\n", nil
}

func headingLevel(line string) int {
	return len(line) - len(strings.TrimLeft(line, "#"))
}
