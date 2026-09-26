package cli

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/fakeollama"
)

// wantCases4c1 is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases4c1 = 7

// wantOllamaCalls4c1 are the requests the reference sent in each recording
// (notes.md, "ollama calls"); a case not named sent none.
var wantOllamaCalls4c1 = map[string]int{
	"reconcile/proposal-kept":     1,
	"reconcile/proposal-invented": 1,
}

// expected4c1 are the mismatches a replay has to report, exactly, per case; a
// case not named must pass. A listed case that starts to pass fails as loudly
// as one that grows a new difference. Today every case replays clean.
var expected4c1 = map[string][]string{}

// callLog is the fake's log, safe to read while its server may still be
// finishing a handler: closing an http.Server does not wait for them.
type callLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *callLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *callLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// serveFakeOllama puts the fake on the fixed port the world's config.toml
// names and hands back the lines it logged. The server closes with t, so the
// next case can take the port.
func serveFakeOllama(t *testing.T, world string) *callLog {
	t.Helper()
	calls := &callLog{}
	path := filepath.Join(world, "ollama-fixture.json")
	if _, err := os.Stat(path); err != nil {
		return calls
	}
	fixture, err := fakeollama.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:11435")
	if err != nil {
		t.Fatalf("the fixed port of the fake Ollama is taken: %v", err)
	}
	server := &http.Server{Handler: fixture.Handler(calls)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return calls
}

// TestCases4c1 replays the recordings of brain-mcp's reconcile over areas
// that ask the local model, with the same fake Ollama answering both sides.
// The cases run one after another, never in parallel: they share the port.
func TestCases4c1(t *testing.T) {
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "4c1"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases4c1 {
		t.Fatalf("found %d cases, want %d", len(all), wantCases4c1)
	}
	for _, c := range all {
		name := c.Verb + "/" + c.Name
		t.Run(name, func(t *testing.T) {
			var calls *callLog
			outcome, err := cases.RunCaseWith(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
				for _, entry := range cases.GitEnv(dir) {
					key, value, _ := strings.Cut(entry, "=")
					t.Setenv(key, value)
				}
				useRecordedEngine(t, dir)
				calls = serveFakeOllama(t, dir)
				return Run(args, stdin, stdout, stderr)
			}, cases.NormalizeState)
			if err != nil {
				t.Fatal(err)
			}
			// stderr is loomux's wording, not the reference's (as in 3b).
			got := slices.DeleteFunc(slices.Clone(outcome.Mismatches), func(m string) bool {
				return strings.HasPrefix(m, "stderr:")
			})
			wanted := slices.Clone(expected4c1[name])
			slices.Sort(got)
			slices.Sort(wanted)
			if !slices.Equal(got, wanted) {
				t.Fatalf("mismatches differ from the expected ones\ngot:\n%s\nwant:\n%s\nstdout:\n%s",
					strings.Join(got, "\n"), strings.Join(wanted, "\n"), outcome.ActualStdout)
			}
			if n := strings.Count(calls.String(), "\n"); n != wantOllamaCalls4c1[name] {
				t.Fatalf("the fake Ollama got %d requests, the reference sent %d:\n%s",
					n, wantOllamaCalls4c1[name], calls.String())
			}
		})
	}
}
