package cli

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
)

func TestRecordedCasesOfStage1a(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "1a"), "")
	if err != nil {
		t.Fatal(err)
	}
	// The number is pinned, not merely non-zero: a case deleted by mistake must
	// not pass unnoticed. Raise it with the corpus when a case is added.
	if len(all) != 19 {
		t.Fatalf("expected 19 recorded cases, found %d", len(all))
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			if !outcome.Passed {
				t.Fatalf("%s\n%s", strings.Join(outcome.Mismatches, "\n"), c.Notes)
			}
		})
	}
}
