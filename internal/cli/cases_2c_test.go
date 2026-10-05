package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/cases"
	"github.com/xidus90/loomux/internal/dev/faketool"
	"github.com/xidus90/loomux/internal/hooks"
)

// wantCases2c is pinned, not merely non-zero: a case deleted by mistake must not pass
// unnoticed. Raise it with the corpus when a case is added.
const wantCases2c = 15

// approved2c names every case whose replay differs from its recording, with
// the number of the deviation in docs/.superpowers/parity/stufe-2c.md that
// explains it. A case missing here must pass; a case listed here must fail.
var approved2c = map[string]string{
	"hook-stop/gave-up":              "2: the counter counts blocks in a row and giving up resets it to 0; Python kept 3",
	"hook-subagent-stop/no-snapshot": "9: silent without a snapshot; Python printed a line nobody read",
}

// TestCases2c replays the recordings of the reference's session hooks against
// loomux hook. The tools of the stop gate answer from the world's fixture,
// at the seam the gate starts its processes through.
func TestCases2c(t *testing.T) {
	all, err := cases.DiscoverCases(filepath.Join("..", "..", "testdata", "cases", "2c"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != wantCases2c {
		t.Fatalf("found %d cases, want %d", len(all), wantCases2c)
	}
	for _, c := range all {
		t.Run(c.Verb+"/"+c.Name, func(t *testing.T) {
			outcome, err := cases.RunCase(c, func(args []string, dir string, stdin io.Reader, stdout, stderr io.Writer) int {
				t.Chdir(dir)
				t.Setenv("LOOMUX_STATE_DIR", dir)
				// The git environment the recorder now runs under, so the
				// hooks' git calls answer the same on every machine. Unlike
				// 3a, no world of 2c names an XDG_CONFIG_HOME; an empty one
				// sends git to the home GitEnv names, where no file lies.
				for _, entry := range cases.GitEnv(dir) {
					key, value, _ := strings.Cut(entry, "=")
					t.Setenv(key, value)
				}
				t.Setenv("XDG_CONFIG_HOME", "")
				useFakeStopTools(t, dir)
				return Run(args, stdin, stdout, stderr)
			})
			if err != nil {
				t.Fatal(err)
			}
			reason, approved := approved2c[c.Verb+"/"+c.Name]
			switch {
			case approved && outcome.Passed:
				t.Fatalf("listed as a deviation (%s) but passes; remove it from approved2c and parity/stufe-2c.md", reason)
			case approved:
				// The arm above only says that something differs. What
				// differs belongs in the record, so a deviation that starts
				// to differ for another reason is visible in a -v run
				// instead of hiding behind its entry.
				t.Logf("deviation %s\n%s", reason, strings.Join(outcome.Mismatches, "\n"))
			case !approved && !outcome.Passed:
				// RunCase already puts stderr into the mismatches where there
				// is one, so the refusal is read without printing it twice.
				t.Fatal(strings.Join(outcome.Mismatches, "\n"))
			}
		})
	}
}

// useFakeStopTools points the stop gate at the world's fixture, the way
// useFakeTools points check at it, for as long as the case runs. A world
// without a fixture is a world of the subagent hooks: they start no tool, so
// there is nothing to point anywhere, and useFakeTools would fail on the
// missing file.
func useFakeStopTools(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, faketool.FixtureName)); err != nil {
		return
	}
	useFakeTools(t, dir) // sets checkStart, checkLook, checkExecutable
	old := stopHook
	t.Cleanup(func() { stopHook = old })
	stopHook = func(stdin io.Reader, stderr io.Writer, root, host string, budget time.Duration) int {
		return hooks.RunStop(stdin, stderr, root, host, hooks.StopEnv{
			Start: checkStart, Look: checkLook, Loomux: fakeSelf, Budget: budget, Now: time.Now,
		})
	}
}
