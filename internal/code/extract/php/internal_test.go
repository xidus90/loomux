package php

import (
	"strings"
	"testing"
)

func TestFileReportsAParseFailure(t *testing.T) {
	// The PHP grammar builds a tree from any input; only a missing grammar
	// makes the parse itself fail.
	_, err := file(nil, "M.php", "<?php class K { }\n")
	if err == nil || !strings.Contains(err.Error(), "parse M.php") {
		t.Errorf("file(nil grammar) = %v, want an error naming the file", err)
	}
}
