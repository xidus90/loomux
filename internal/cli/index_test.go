package cli

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
)

// indexWorld is one machine for an index run: a state directory with a
// registry, one registered area with a declaration and a note, and qmd's
// configuration redirected into the same temporary place, so that a test
// never writes into the qmd installation of whoever runs it.
func indexWorld(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	state := filepath.Join(tmp, "state")
	area := filepath.Join(tmp, "area")
	t.Setenv("LOOMUX_STATE_DIR", state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", filepath.Join(tmp, "legacy"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	writeFile(t, filepath.Join(state, "registry.toml"),
		"[[area]]\nscope = \"project/a\"\npath = "+strconv.Quote(filepath.ToSlash(area))+"\n")
	writeFile(t, filepath.Join(area, ".loomux", "config.toml"), "[area]\nscope = \"project/a\"\n")
	writeFile(t, filepath.Join(area, "note.md"), "---\ntitle: Note\n---\n# Note\n")
	return state
}

// emptyState is a machine on which nothing has been registered yet: the state
// directory exists, the registry does not.
func emptyState(t *testing.T) string {
	t.Helper()
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	t.Setenv("LOOMUX_LEGACY_BRAIN_DIR", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	return state
}

// stubIndexLook replaces the PATH lookup of the index commands.
func stubIndexLook(t *testing.T, path string, err error) {
	t.Helper()
	saved := indexLook
	indexLook = func(string) (string, error) { return path, err }
	t.Cleanup(func() { indexLook = saved })
}

// An empty state is no failure: a machine without a registered area has
// nothing to index, and that is a statement, not a collapse.
func TestReindexOnAnEmptyStateSaysSoAndSucceeds(t *testing.T) {
	emptyState(t)
	code, out, errOut := run("reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if strings.TrimSpace(out) == "" && strings.TrimSpace(errOut) == "" {
		t.Fatal("reindex said nothing at all")
	}
	if !strings.Contains(out, "no areas registered") {
		t.Fatalf("stdout does not say the state is empty: %q", out)
	}
}

// A registry named by hand and missing is a typo, not an empty machine: the
// caller said where to look, and nothing lies there.
func TestReindexIsRedWhenTheNamedRegistryIsMissing(t *testing.T) {
	state := emptyState(t)
	code, _, errOut := run("reindex", "--registry", filepath.Join(state, "elsewhere", "registry.toml"))
	if code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "registry.toml") {
		t.Fatalf("stderr does not name the registry: %q", errOut)
	}
}

func TestReindexRefusesAnUnknownFlag(t *testing.T) {
	emptyState(t)
	if code, _, _ := run("reindex", "--state-dir", "somewhere"); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

// A registry that exists and cannot be read is red: only its absence is a
// statement.
func TestReindexIsRedOnABrokenRegistry(t *testing.T) {
	state := emptyState(t)
	writeFile(t, filepath.Join(state, "registry.toml"), "this is not TOML {{{\n")
	code, _, errOut := run("reindex")
	if code != 1 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "error:") {
		t.Fatalf("stderr does not carry the error: %q", errOut)
	}
}

func TestReindexIndexesTheRegisteredAreaAndRefreshesTheEngine(t *testing.T) {
	indexWorld(t)
	port := search.NewFakePort()
	stubBrainStatusPort(t, port)
	code, out, errOut := run("reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if len(port.Refreshed) != 1 || len(port.Refreshed[0]) != 1 {
		t.Fatalf("refreshed = %v", port.Refreshed)
	}
	if !strings.Contains(out, "indexed") {
		t.Fatalf("stdout does not report the run: %q", out)
	}
}

// --registry takes the directory the registry lies in as well as the file
// itself, because the run it feeds accepts both spellings too.
func TestReindexTakesTheDirectoryOfTheRegistry(t *testing.T) {
	state := indexWorld(t)
	stubBrainStatusPort(t, search.NewFakePort())
	code, out, errOut := run("reindex", "--registry", state)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(out, "registry.toml") {
		t.Fatalf("stdout does not name the registry: %q", out)
	}
}

// An engine that refuses to refresh leaves the run red, and the command hands
// that exit on instead of reporting a run it did not finish.
func TestReindexHandsOnTheExitOfTheRun(t *testing.T) {
	indexWorld(t)
	port := search.NewFakePort()
	port.Refreshes = []error{errors.New("engine is asleep")}
	stubBrainStatusPort(t, port)
	if code, _, _ := run("reindex"); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

// Missing qmd, embed names it and the install command and is red -- never an
// empty success message.
func TestEmbedWithoutQmdIsRed(t *testing.T) {
	emptyState(t)
	t.Setenv("PATH", t.TempDir())
	code, _, errOut := run("embed")
	if code == 0 {
		t.Fatalf("exit = 0 without qmd, stderr = %s", errOut)
	}
	if !strings.Contains(errOut, "qmd") {
		t.Fatalf("stderr does not name qmd: %s", errOut)
	}
	if !strings.Contains(errOut, "npm install -g @tobilu/qmd") {
		t.Fatalf("stderr does not name the install command: %s", errOut)
	}
}

func TestEmbedOnAnEmptyStateSaysSoAndSucceeds(t *testing.T) {
	emptyState(t)
	stubIndexLook(t, "qmd", nil)
	code, out, errOut := run("embed")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(out, "no areas registered") {
		t.Fatalf("stdout does not say the state is empty: %q", out)
	}
}

func TestEmbedRefusesAnUnknownFlag(t *testing.T) {
	emptyState(t)
	if code, _, _ := run("embed", "--state-dir", "somewhere"); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestEmbedAsksTheEngineForEveryRegisteredArea(t *testing.T) {
	indexWorld(t)
	stubIndexLook(t, "qmd", nil)
	port := search.NewFakePort()
	stubBrainStatusPort(t, port)
	code, _, errOut := run("embed")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if len(port.Embedded) != 1 || len(port.Embedded[0]) != 1 || port.Embedded[0][0] != "project/a" {
		t.Fatalf("embedded = %v", port.Embedded)
	}
	if !strings.Contains(errOut, "embedded 1 area(s)") {
		t.Fatalf("stderr does not report the run: %q", errOut)
	}
}

func TestEmbedIsRedOnABrokenRegistry(t *testing.T) {
	state := emptyState(t)
	stubIndexLook(t, "qmd", nil)
	writeFile(t, filepath.Join(state, "registry.toml"), "this is not TOML {{{\n")
	if code, _, _ := run("embed"); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

// refusesAPositional runs a command that takes no positional argument with one
// and holds it to argparse's answer: exit 2, the argument named, and nothing
// done. The empty stdout is the part that matters -- a command that accepted
// the word and went on would run against whatever state the environment names,
// and outside a test that sets its own, that is the real one of the machine.
func refusesAPositional(t *testing.T, command string) {
	t.Helper()
	emptyState(t)
	code, out, errOut := run(command, "somewhere")
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "loomux "+command+": unrecognized arguments: somewhere") {
		t.Fatalf("stderr does not name the argument: %q", errOut)
	}
	if out != "" {
		t.Fatalf("stdout reports a run that must not have started: %q", out)
	}
}

func TestReindexRefusesAPositionalArgument(t *testing.T) {
	refusesAPositional(t, "reindex")
}

// Asked before the engine: a refusal of the command line must not depend on
// whether qmd happens to lie on this machine's PATH. The lookup answers "not
// found", so a refusal asked after it would end with exit 1, not 2.
func TestEmbedRefusesAPositionalArgument(t *testing.T) {
	stubIndexLook(t, "", exec.ErrNotFound)
	refusesAPositional(t, "embed")
}

func TestReconcileRefusesAPositionalArgument(t *testing.T) {
	refusesAPositional(t, "reconcile")
}
