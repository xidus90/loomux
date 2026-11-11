package csharp

import (
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The C# grammar builds a tree from any input; only a missing grammar
	// makes the parse itself fail.
	_, err := file(nil, "m.cs", "class K { }\n")
	if err == nil || !strings.Contains(err.Error(), "parse m.cs") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}
