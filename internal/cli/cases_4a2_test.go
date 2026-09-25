package cli

import (
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/cases"
)

// wantCases4a2 is pinned, not merely non-zero: a partial import must not pass
// as parity. Raise it with the corpus when a case is added.
const wantCases4a2 = 14

// expectation4a2 is how the replay of one case may differ from its recording.
// The list is exact: a listed case that starts to pass fails as loudly as one
// that grows a new difference, and a case that is not listed must pass.
type expectation4a2 struct {
	// why names the row of docs/.superpowers/parity/stufe-4a-2.md the
	// differences belong to.
	why    string
	differ []string
	// stdout, where set, is loomux's own output in full: a difference in
	// length alone would let any other output of the same length pass.
	stdout string
}

var expected4a2 = map[string]expectation4a2{
	// The staged record spells its paths with slashes; the reference holds
	// them as text against git's answer in Windows spelling and calls its own
	// installation orphaned. loomux compares them as paths, and names the
	// hook file on every line it knows one for.
	"hook/status-installed": {
		why:    "Prüfstand: Pfadvergleich als Text; status nennt die Hookdatei",
		differ: []string{"stdout mismatch: expected 41 bytes, got 83 bytes"},
	},
	// The reference prints no hook file on an orphaned or a not installed
	// line; loomux does.
	"hook/status-orphaned": {
		why:    "status nennt die Hookdatei",
		differ: []string{"stdout mismatch: expected 93 bytes, got 178 bytes"},
	},
	// The reference prints the recorded hook file through Path, in Windows
	// spelling; loomux prints the record as it stands, here the staged
	// slashes. A record either tool wrote holds the native spelling already.
	"hook/remove-installed": {
		why:    "Prüfstand: Schreibweise des Hookpfads im Eintrag",
		differ: []string{"stdout mismatch: expected 81 bytes, got 81 bytes"},
		stdout: "removed: project/a — {{WORLD}}/repo-a [{{WORLD}}/repo-a/.git/hooks/post-merge]\n",
	},
}

// hookRecords is the record file of the installations. The reference writes
// five fields -- the branch and the event path it baked into the hook -- and
// loomux three, because it bakes nothing in (E4).
const hookRecords = "maintenance/hooks.tsv"

// normalize4a2 cuts every line of the record file to the three fields both
// sides write. It changes the tree it is handed and returns it: the runner
// carries both stdouts through the same map.
func normalize4a2(_, tree map[string][]byte) map[string][]byte {
	data, ok := tree[hookRecords]
	if !ok {
		return tree
	}
	var cut strings.Builder
	for line := range strings.Lines(string(data)) {
		fields := strings.Split(strings.TrimSuffix(line, "\n"), "\t")
		cut.WriteString(strings.Join(fields[:min(3, len(fields))], "\t") + "\n")
	}
	tree[hookRecords] = []byte(cut.String())
	return tree
}

// TestCases4a2 replays the recorded cases of `brain-mcp hook` against
// `loomux merge-hook`, each in a world whose repository the replay builds as
// the recording did. stderr is not compared.
func TestCases4a2(t *testing.T) {
	corpus, err := filepath.Abs(filepath.Join("..", "..", "testdata", "cases", "4a2"))
	if err != nil {
		t.Fatal(err)
	}
	all, err := cases.DiscoverCases(corpus, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases4a2 {
		t.Fatalf("found %d cases, want %d", len(all), wantCases4a2)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			want := expected4a2[c.Verb+"/"+c.Name]
			outcome, err := cases.RunCaseWith(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", dir)
				t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
				// The git environment the recorder ran the reference under.
				for _, entry := range cases.GitEnv(dir) {
					key, value, _ := strings.Cut(entry, "=")
					t.Setenv(key, value)
				}
				return Run(args, stdin, stdout, stderr)
			}, normalize4a2)
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
			if want.stdout != "" && string(outcome.ActualStdout) != want.stdout {
				t.Fatalf("stdout %q, want %q", outcome.ActualStdout, want.stdout)
			}
		})
	}
}
