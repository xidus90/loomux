package cli

import (
	"bytes"
	"io"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/fakeqmd"
)

// TestRecordedCasesOfStage1b1 replays the recordings of brain-mcp against
// loomux brain. Every world carries the fixture the recording's fake qmd
// answered from; the same fixture answers here, over HTTP for the MCP port of
// search and through the Runner seam for the command line port of status.
func TestRecordedCasesOfStage1b1(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "1b-1"), "")
	if err != nil {
		t.Fatal(err)
	}
	// The number is pinned, not merely non-zero: a partial import must not
	// pass as parity. Raise it with the corpus when a case is added.
	if len(all) != 71 {
		t.Fatalf("expected 71 recorded cases, found %d", len(all))
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Setenv("LOOMUX_STATE_DIR", dir)
				useRecordedQmd(t, dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			if !outcome.Passed {
				// The runner names only the byte counts of a stdout mismatch;
				// the text itself is what a red case is read by.
				t.Fatalf("%s\nstdout: %q\n%s", strings.Join(outcome.Mismatches, "\n"), outcome.ActualStdout, c.Notes)
			}
		})
	}
}

// useRecordedQmd points both brain seams at the fixture of the staged world
// until the case ends: search asks the fixture's MCP handler, status and the
// MCP port's own listings run the fixture's command line.
func useRecordedQmd(t *testing.T, dir string) {
	t.Helper()
	fixture, err := fakeqmd.Load(filepath.Join(dir, fakeqmd.FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(fixture.MCPHandler())
	t.Cleanup(server.Close)
	runner := func(argv []string) ([]byte, []byte, int, error) {
		var out, errOut bytes.Buffer
		code := fixture.RunCLI(argv[1:], &out, &errOut)
		return out.Bytes(), errOut.Bytes(), code, nil
	}
	mcp := search.NewQmdMcpPort(
		search.WithConnect(func(map[string]string) (search.Session, error) {
			return &search.HTTPSession{URL: server.URL}, nil
		}),
		search.WithCLI(&search.QmdPort{Executable: "qmd", Runner: runner}),
	)
	stubBrainSearchPort(t, mcp, "")
	stubBrainStatusPort(t, &search.QmdPort{Executable: "qmd", Runner: runner})
}
