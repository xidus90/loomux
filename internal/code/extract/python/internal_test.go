package python

import (
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The Python grammar builds a tree from any input; only a missing grammar
	// makes the parse itself fail.
	_, err := file(nil, "m.py", "x = 1\n")
	if err == nil || !strings.Contains(err.Error(), "parse m.py") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}
