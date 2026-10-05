package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/dev/faketool"
	"github.com/xidus90/loomux/internal/verify/commit"
)

// wantCases2b is pinned, not merely non-zero: a case deleted by mistake must not pass
// unnoticed. Raise it with the corpus when a case is added.
const wantCases2b = 19

// Every case here must pass. A refusal carries exit 1 and not the 2 its
// recording under 2b-source holds (deviation 5 in
// `docs/.superpowers/parity/stufe-2b.md` in the working papers of the archive
// release `archive/parity-recordings`): a case that merely "must fail" would
// accept any wrong code, a crash included.

// TestCases2b replays the recordings of the reference's `commit-msg` against loomux
// check commit-msg.
func TestCases2b(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "2b"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases2b {
		t.Fatalf("found %d cases, want %d", len(all), wantCases2b)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				useFakeTools2b(t, dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			if !outcome.Passed {
				t.Fatalf("%s\nstdout:\n%s\nstderr:\n%s", strings.Join(outcome.Mismatches, "\n"), outcome.ActualStdout, outcome.ActualStderr)
			}
		})
	}
}

func useFakeTools2b(t *testing.T, dir string) {
	t.Helper()
	fixturePath := filepath.Join(dir, faketool.FixtureName)
	fixture, err := faketool.Load(fixturePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}

	oldGitRunner := commit.GitRunner
	t.Cleanup(func() { commit.GitRunner = oldGitRunner })
	commit.GitRunner = func(dir string, timeout time.Duration, argv ...string) (child.Result, error) {
		answer, ok := fixture.Match(argv)
		if !ok {
			return child.Result{Code: 127, Stderr: "faketool: no answer\n"}, nil
		}
		return child.Result{Code: answer.Exit, Stdout: answer.Stdout}, nil
	}
}
