package reader_test

import (
	"testing"

	"github.com/xidus90/loomux/internal/brain/reader"
)

// The tests the mutation round of stage 1b-1 asked for. They are gathered
// here rather than scattered because what they have in common is how they
// were found: each one is the case a surviving mutant proved nobody had
// asked for. Every expectation below was read from the Python reference or
// measured against it, never derived from the Go code that has to satisfy
// it.

func TestOnlyAHeadingLineCanOpenASection(t *testing.T) {
	// `_section` (src/brain/core.py:549) reads
	// `if line.startswith("#") and line.lstrip("#").strip() == heading`:
	// the `#` is half of the test. A body line that reads like the title
	// is body, and the heading below it is the section.
	const doc = "Summary\n# Summary\nbody\n"
	got, err := reader.ExtractSection(doc, "Summary")
	if err != nil {
		t.Fatalf("ExtractSection: %v", err)
	}
	if want := "# Summary\nbody\n"; got != want {
		t.Fatalf("ExtractSection = %q, want %q", got, want)
	}
}
