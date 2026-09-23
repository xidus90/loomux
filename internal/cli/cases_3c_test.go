package cli

import (
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
)

// wantCases3c is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases3c = 35

// expectation3c is how the replay of one case may differ from its recording.
// The list is exact: a listed case that starts to pass fails as loudly as one
// that grows a new difference, and a case that is not listed must pass.
type expectation3c struct {
	// why names the row of docs/.superpowers/parity/stufe-3c.md the
	// differences belong to.
	why    string
	differ []string
}

var expected3c = map[string]expectation3c{
	// The reference answers the fourth width with the code lanes it finds in
	// no manifest: exit 0 and nothing. loomux refuses the width with 2.
	"check/code": {why: "brain check code entfällt (E2)", differ: []string{"exit code: expected 0, got 2"}},
}

// TestCases3c replays the recorded cases of stage 3c against loomux: the check
// widths against the Go binary of the tag, lint and the wiki tools against the
// Python reference. stderr is not compared.
func TestCases3c(t *testing.T) {
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "3c"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases3c {
		t.Fatalf("found %d cases, want %d", len(all), wantCases3c)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			want := expected3c[c.Verb+"/"+c.Name]
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			got := slices.DeleteFunc(slices.Clone(outcome.Mismatches), func(m string) bool {
				return strings.HasPrefix(m, "stderr:")
			})
			wanted := slices.Clone(want.differ)
			slices.Sort(got)
			slices.Sort(wanted)
			if !slices.Equal(got, wanted) {
				why := want.why
				if why == "" {
					why = "none expected"
				}
				t.Fatalf("differences %q, want %q (%s)\nstdout:\n%s\nstderr:\n%s",
					got, wanted, why, outcome.ActualStdout, outcome.ActualStderr)
			}
		})
	}
}
