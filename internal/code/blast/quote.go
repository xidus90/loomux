package blast

import (
	"regexp"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
)

// QuoteLine locates the first line inside span in path that references targetName
// as a distinct word.
//
// Returns the 1-based line number, trimmed line content, and true if found.
// If the reader fails, target is empty, span is invalid or not found, returns (0, "", false).
//
// Ported from trailhq/Graft @ 1e352a3 (MIT), src/blast/evidence.ts (referenceLine).
func QuoteLine(read func(string) ([]byte, error), path string, span model.Span, targetName string) (int, string, bool) {
	if read == nil || targetName == "" {
		return 0, "", false
	}
	from, to, ok := span.Lines()
	if !ok || from < 1 || to < from {
		return 0, "", false
	}
	data, err := read(path)
	if err != nil {
		return 0, "", false
	}

	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(targetName) + `\b`)
	lines := strings.Split(string(data), "\n")
	for lineNum := from; lineNum <= to && lineNum <= len(lines); lineNum++ {
		raw := strings.TrimSuffix(lines[lineNum-1], "\r")
		if re.MatchString(raw) {
			return lineNum, strings.TrimSpace(raw), true
		}
	}
	return 0, "", false
}
