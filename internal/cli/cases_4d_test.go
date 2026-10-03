package cli

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/xidus90/loomux/internal/brain/convert"
	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/dev/faketool"
)

// wantCases4d is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases4d = 29

// wantOllamaCalls4d are the requests the reference sent in each recording
// (notes.md, "ollama calls"); a case not named sent none. The warm-up loomux
// sends in front of them is held by checkOllamaCalls.
var wantOllamaCalls4d = map[string]int{
	"convert/describe-kept":       1,
	"convert/describe-refused":    1,
	"convert/place-suggested":     1,
	"convert/place-unknown-scope": 1,
}

// expected4d are the mismatches a replay has to report, exactly, per case; a
// case not named must pass. A listed case that starts to pass fails as loudly
// as one that grows a new difference. Today every case replays clean.
var expected4d = map[string][]string{}

// retrievedDay is compiled on first use, as every regexp in loomux is.
var retrievedDay = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`(?m)^retrieved: \d{4}-\d{2}-\d{2}$`)
})

// normalize4d holds both sides to what two runs on two days cannot share:
// the day a staged world's files were written, and the PDF converter's
// count, which pdftotext raised from 1 to 2 (a released deviation).
func normalize4d(_, tree map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(tree))
	for name, data := range tree {
		data = retrievedDay().ReplaceAll(data, []byte("retrieved: {{DAY}}"))
		out[name] = bytes.ReplaceAll(data, []byte("converter: brain-pdf/1\n"), []byte("converter: brain-pdf/2\n"))
	}
	return out
}

// useFakePdftotext answers pdftotext from the world's faketool.json, which
// carries what the reference's pypdf read; a world without one has none.
func useFakePdftotext(t *testing.T, world string) {
	t.Helper()
	fixture, err := faketool.Load(filepath.Join(world, faketool.FixtureName))
	if err != nil {
		t.Fatal(err)
	}
	saved := convertTools
	t.Cleanup(func() { convertTools = saved })
	convertTools = convert.Tools{
		Look: func(name string) (string, error) {
			if len(fixture.Answers) == 0 {
				return "", exec.ErrNotFound
			}
			return name, nil
		},
		Run: func(spec child.Spec) child.Result {
			answer, ok := fixture.Match(spec.Argv)
			if !ok {
				return child.Result{Code: 127, Stderr: "faketool: no answer\n"}
			}
			return child.Result{Code: answer.Exit, Stdout: answer.Stdout}
		},
	}
}

// TestCases4d replays the recordings of brain-mcp's convert. The reference
// read each PDF with pypdf; loomux reads it through pdftotext, whose answers
// here are what pypdf read, page by page. The same fake Ollama answers both
// sides, and the cases run one after another because they share its port.
func TestCases4d(t *testing.T) {
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "4d"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases4d {
		t.Fatalf("found %d cases, want %d", len(all), wantCases4d)
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
				useFakePdftotext(t, dir)
				calls, addr = serveFakeOllama(t, dir)
				return Run(args, stdin, stdout, stderr)
			}, func(world, tree map[string][]byte) map[string][]byte {
				tree = normalize4d(world, tree)
				if addr == "" {
					return tree
				}
				// The fake's free port goes back to the recorded address, as
				// in foldOllama, without widening 4d's normalizing to
				// NormalizeState.
				for name, data := range tree {
					tree[name] = bytes.ReplaceAll(data, []byte(addr), []byte(recordedOllama))
				}
				return tree
			})
			if err != nil {
				t.Fatal(err)
			}
			// stderr is loomux's wording, not the reference's (as in 3b).
			got := slices.DeleteFunc(slices.Clone(outcome.Mismatches), func(m string) bool { return strings.HasPrefix(m, "stderr:") })
			wanted := slices.Clone(expected4d[name])
			slices.Sort(got)
			slices.Sort(wanted)
			if !slices.Equal(got, wanted) {
				t.Fatalf("mismatches differ from the expected ones\ngot:\n%s\nwant:\n%s\nstdout:\n%s\nstderr:\n%s",
					strings.Join(got, "\n"), strings.Join(wanted, "\n"), outcome.ActualStdout, outcome.ActualStderr)
			}
			checkOllamaCalls(t, calls.String(), wantOllamaCalls4d[name])
		})
	}
}
