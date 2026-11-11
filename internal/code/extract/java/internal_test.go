package java

import (
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The Java grammar builds a tree from any input; only a missing grammar
	// makes the parse itself fail.
	_, err := file(nil, "M.java", "class K { }\n")
	if err == nil || !strings.Contains(err.Error(), "parse M.java") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}
