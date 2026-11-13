package stylesheet

import (
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The grammars build a tree from any input; only a missing grammar makes
	// the parse itself fail.
	_, err := file(nil, cssName, "m.css", ".a { }\n", false)
	if err == nil || !strings.Contains(err.Error(), "parse m.css") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}
