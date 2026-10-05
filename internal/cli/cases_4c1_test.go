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

// wantCases4c1 is pinned, not merely non-zero: a case deleted by mistake must not pass
// unnoticed. Raise it with the corpus when a case is added.
const wantCases4c1 = 7

// wantOllamaCalls4c1 are the requests the reference sent in each recording
// (notes.md, "ollama calls"); a case not named sent none. loomux sends the
// same questions and, before a client's first one, a warm-up the reference
// does not know; checkOllamaCalls holds both.
var wantOllamaCalls4c1 = map[string]int{
	"reconcile/proposal-kept":     1,
	"reconcile/proposal-invented": 1,
}

// checkOllamaCalls holds the fake's log to the reference's count of
// questions, and to one warm-up in front of them when there are any: every
// recorded case asks through a single client, and a client warms the model
// once, before its first question and only if it asks one.
func checkOllamaCalls(t *testing.T, log string, questions int) {
	t.Helper()
	var lines []string
	if log != "" {
		lines = strings.Split(strings.TrimSuffix(log, "\n"), "\n")
	}
	warm := 0
	for _, line := range lines {
		if strings.HasSuffix(line, " num_predict=1") {
			warm++
		}
	}
	wantWarm := min(questions, 1)
	if len(lines)-warm != questions || warm != wantWarm || (warm > 0 && !strings.HasSuffix(lines[0], " num_predict=1")) {
		t.Fatalf("the fake Ollama got %d questions and %d warm-ups; the reference sent %d questions, and loomux warms %d times before them:\n%s",
			len(lines)-warm, warm, questions, wantWarm, log)
	}
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

// recordedOllama is the address the recordings' config.toml names, the one
// the reference's fake listened on.
const recordedOllama = "127.0.0.1:11435"

// serveFakeOllama puts the fake on a free port, points the staged world's
// config.toml at it and hands back the lines it logged and the address. A
// fixed port would be shared by every test process on the machine, and a
// second gate running at the same time would find it taken. The server closes
// with t.
func serveFakeOllama(t *testing.T, world string) (*callLog, string) {
	t.Helper()
	calls := &callLog{}
	path := filepath.Join(world, "ollama-fixture.json")
	if _, err := os.Stat(path); err != nil {
		return calls, ""
	}
	fixture, err := fakeollama.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	config := filepath.Join(world, "config.toml")
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(recordedOllama)) {
		t.Fatalf("%s names no %s for the fake to stand in for", config, recordedOllama)
	}
	if err := os.WriteFile(config, bytes.ReplaceAll(data, []byte(recordedOllama), []byte(addr)), 0o644); err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: fixture.Handler(calls)}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return calls, addr
}

// foldOllama is NormalizeState that also writes the fake's address back as
// the recorded one, on either side, so the world_after comparison does not see
// the port the run happened to get.
func foldOllama(addr *string) cases.Normalizer {
	return func(world, tree map[string][]byte) map[string][]byte {
		if *addr != "" {
			folded := make(map[string][]byte, len(tree))
			for name, data := range tree {
				folded[name] = bytes.ReplaceAll(data, []byte(*addr), []byte(recordedOllama))
			}
			tree = folded
		}
		return cases.NormalizeState(world, tree)
	}
}

// TestCases4c1 replays the recordings of the reference's reconcile over areas
// that ask the local model, with the same fake Ollama answering both sides.
// The cases run one after another, never in parallel: they share the
// environment the run sets.
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
			var addr string
			outcome, err := cases.RunCaseWith(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
				for _, entry := range cases.GitEnv(dir) {
					key, value, _ := strings.Cut(entry, "=")
					t.Setenv(key, value)
				}
				useRecordedEngine(t, dir)
				calls, addr = serveFakeOllama(t, dir)
				return Run(args, stdin, stdout, stderr)
			}, foldOllama(&addr))
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
			checkOllamaCalls(t, calls.String(), wantOllamaCalls4c1[name])
		})
	}
}
