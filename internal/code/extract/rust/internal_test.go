package rust

import (
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The Rust grammar builds a tree from any input; only a missing grammar
	// makes the parse itself fail.
	_, err := file(nil, "m.rs", "fn f() {}\n")
	if err == nil || !strings.Contains(err.Error(), "parse m.rs") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}
