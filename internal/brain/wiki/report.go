package wiki

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/xidus90/loomux/internal/brain/check"
)

// LintReport lints one page, writes one line per finding and returns 1 when
// any finding is an error or the page cannot be linted.
func LintReport(target, wikiRoot string, w io.Writer) int {
	findings, err := LintSingleFile(target, wikiRoot)
	if err != nil {
		fmt.Fprintf(w, "lint error: %v\n", err)
		return 1
	}
	code := 0
	for _, f := range findings {
		fmt.Fprintf(w, "  • [%s:%s] %s: %s\n", f.Severity, f.Rule, f.Relative, f.Message)
		if f.Severity == check.Error {
			code = 1
		}
	}
	return code
}

// GateReport runs the wiki gate for a project and reports as `brain wiki-gate` did.
func GateReport(projectRoot string, stdout, stderr io.Writer) int {
	violations := CheckWikiGate(projectRoot)
	if len(violations) == 0 {
		fmt.Fprintln(stdout, "OK: Wiki Gate passed. All bundle links valid and no drift detected.")
		return 0
	}
	fmt.Fprintf(stderr, "\n❌ Wiki-Gate Violation(s) Detected in %s:\n", filepath.Base(projectRoot))
	for _, v := range violations {
		fmt.Fprintf(stderr, "  • [%s] %s\n", v.Name, v.Message)
	}
	fmt.Fprintf(stderr, "\nFound %d violation(s).\n", len(violations))
	return 1
}
