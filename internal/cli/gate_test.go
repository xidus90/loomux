package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/verify"
)

const gateUsageWant = "usage: loomux gate status [--root <dir>]\n" +
	"       loomux gate arm <lane>... [--root <dir>]\n" +
	"       loomux gate disarm <lane>...|--all [--root <dir>]\n"

func TestGateNeedsASubcommandItKnows(t *testing.T) {
	root := armedWorld(t, armedText())
	for _, args := range [][]string{
		{"gate"}, {"gate", "list"}, {"gate", "status", "extra", "--root", root},
		{"gate", "arm", "--root", root}, {"gate", "disarm", "--root", root},
		{"gate", "disarm", "--all", "lint/go@.", "--root", root}, {"gate", "arm", "--all", "--root", root},
		// --all belongs to disarm alone: beside a key, arm would take it for a
		// switch and write.
		{"gate", "arm", "--all", "lint/go@.", "--root", root},
	} {
		if code, out, _ := run(args...); code != 2 || out != "" {
			t.Errorf("%v: code %d, out %q", args, code, out)
		}
	}
	if code, _, errOut := run("gate"); code != 2 || errOut != gateUsageWant {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if armedFile(t, root) != armedText() {
		t.Fatal("a refused call wrote")
	}
}

func TestGateStatusNamesEveryLaneAndEveryOrphan(t *testing.T) {
	root := armedWorld(t, "")
	if code, out, errOut := run("gate", "status", "--root", root); code != 0 || out != "no .loomux/armed.toml: every lane is armed\n" || errOut != "" {
		t.Fatalf("no file: code %d, out %q, err %q", code, out, errOut)
	}
	root = armedWorld(t, armedText("lint/go@.", "lint/python@."))
	want := "coverage/go@.: probation\nlint/go@.: armed\nlint/python@.: orphan\ntest/go@.: probation\n"
	if code, out, _ := run("gate", "status", "--root", root); code != 0 || out != want {
		t.Fatalf("code %d, out %q", code, out)
	}
	root = armedWorld(t, "armed = 1\n")
	if code, out, errOut := run("gate", "status", "--root", root); code != 1 || out != "" || !strings.Contains(errOut, "every lane is armed") {
		t.Fatalf("unreadable: code %d, out %q, err %q", code, out, errOut)
	}
}

// The list stands, and beside it the warning: a file git ignores reaches no
// commit, so nobody else and no CI ever reads it.
func TestGateStatusWarnsAboutAnIgnoredFile(t *testing.T) {
	const warning = "loomux gate status: .loomux/armed.toml is ignored by git: it reaches no commit and holds on this machine only\n"
	root := armedWorld(t, armedText("lint/go@."))
	mustRunGit(t, root, "init", "-q")
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte(".loomux/\n"), 0o644)
	code, out, errOut := run("gate", "status", "--root", root)
	if code != 0 || !strings.Contains(out, "lint/go@.: armed\n") || errOut != warning {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	// Ignoring the state directory alone is how init sets a project up.
	os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/.loomux/state/\n"), 0o644)
	if code, _, errOut = run("gate", "status", "--root", root); code != 0 || errOut != "" {
		t.Fatalf("only the state ignored: code %d, err %q", code, errOut)
	}
}

func TestGateArmEntersALaneWhateverItsState(t *testing.T) {
	root := armedWorld(t, armedText("lint/python@."))
	code, out, errOut := run("gate", "arm", "test/go@.", "lint/go@.", "--root", root)
	if code != 0 || out != "armed: lint/go@., test/go@.\n" || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if got := armedFile(t, root); got != armedText("lint/go@.", "lint/python@.", "test/go@.") {
		t.Fatalf("%q", got)
	}
	// A key no lane answers to is a mistake, and nothing of the call is written.
	before := armedFile(t, root)
	code, out, errOut = run("gate", "arm", "coverage/go@.", "lint/go", "--root", root)
	if code != 1 || out != "" || errOut != "loomux gate arm: no lane \"lint/go\"; lanes: coverage/go@., lint/go@., test/go@.\n" || armedFile(t, root) != before {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestGateArmWithoutTheFileWritesNone(t *testing.T) {
	root := armedWorld(t, "")
	code, out, _ := run("gate", "arm", "lint/go@.", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 0 || out != "no .loomux/armed.toml: every lane is armed\n" || err == nil {
		t.Fatalf("code %d, out %q, stat %v", code, out, err)
	}
	// A key no lane answers to is an error all the same, before the file is
	// even asked about.
	code, out, errOut := run("gate", "arm", "lint/go", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 1 || out != "" || !strings.HasPrefix(errOut, "loomux gate arm: no lane \"lint/go\"; lanes: ") || err == nil {
		t.Fatalf("an unknown key without the file: code %d, out %q, err %q, stat %v", code, out, errOut, err)
	}
}

func TestGateDisarmTakesAnEntryOut(t *testing.T) {
	root := armedWorld(t, armedText("lint/go@.", "lint/python@.", "test/go@."))
	code, out, _ := run("gate", "disarm", "lint/python@.", "lint/go@.", "types/go@.", "--root", root)
	if code != 0 || out != "probation: lint/go@., lint/python@.\ntypes/go@.: was not armed\n" {
		t.Fatalf("code %d, out %q", code, out)
	}
	if got := armedFile(t, root); got != armedText("test/go@.") {
		t.Fatalf("%q", got)
	}
}

// Without the file every lane is armed; taking one out of that leaves every
// other armed, which is a file naming them.
func TestGateDisarmWithoutTheFileArmsEveryOtherLane(t *testing.T) {
	root := armedWorld(t, "")
	code, out, _ := run("gate", "disarm", "lint/go@.", "--root", root)
	if code != 0 || out != "probation: lint/go@.\n" || armedFile(t, root) != armedText("coverage/go@.", "test/go@.") {
		t.Fatalf("code %d, out %q, file %q", code, out, armedFile(t, root))
	}
	root = armedWorld(t, "")
	code, _, errOut := run("gate", "disarm", "lint/python@.", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 1 || !strings.HasPrefix(errOut, "loomux gate disarm: no lane \"lint/python@.\"") || err == nil {
		t.Fatalf("code %d, err %q, stat %v", code, errOut, err)
	}
}

func TestGateDisarmAllStartsTheProbation(t *testing.T) {
	for name, text := range map[string]string{"no file": "", "a full file": armedText("lint/go@."), "a broken file": "<<<<<<<\n"} {
		root := armedWorld(t, text)
		code, out, errOut := run("gate", "disarm", "--all", "--root", root)
		if code != 0 || out != "probation: every lane\n" || errOut != "" || armedFile(t, root) != armedText() {
			t.Errorf("%s: code %d, out %q, err %q, file %q", name, code, out, errOut, armedFile(t, root))
		}
	}
}

func TestGateReportsWhatItCannotReadOrWrite(t *testing.T) {
	root := armedWorld(t, "armed = 1\n")
	for _, args := range [][]string{{"gate", "arm", "lint/go@.", "--root", root}, {"gate", "disarm", "lint/go@.", "--root", root}} {
		if code, _, errOut := run(args...); code != 1 || !strings.Contains(errOut, "every lane is armed") || armedFile(t, root) != "armed = 1\n" {
			t.Errorf("%v: code %d, err %q", args, code, errOut)
		}
	}
	// Lanes that cannot be planned: a config that does not parse. status and
	// arm need the lanes beside a file; disarm needs them only without one.
	root = armedWorld(t, armedText())
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify\n"), 0o644)
	for _, args := range [][]string{{"gate", "status", "--root", root}, {"gate", "arm", "lint/go@.", "--root", root}} {
		if code, _, errOut := run(args...); code != 1 || !strings.HasPrefix(errOut, "loomux gate") || armedFile(t, root) != armedText() {
			t.Errorf("%v: code %d, err %q", args, code, errOut)
		}
	}
	root = armedWorld(t, "")
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify\n"), 0o644)
	if code, _, errOut := run("gate", "disarm", "lint/go@.", "--root", root); code != 1 || !strings.HasPrefix(errOut, "loomux gate disarm: ") {
		t.Errorf("disarm without a file: code %d, err %q", code, errOut)
	}
	// A file that cannot be written: a directory stands in its place.
	root = armedWorld(t, "")
	os.MkdirAll(filepath.Join(root, ".loomux", "armed.toml", "x"), 0o755)
	if code, _, errOut := run("gate", "disarm", "--all", "--root", root); code != 1 || !strings.HasPrefix(errOut, "loomux gate disarm: ") {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	// A write that fails after the file was read: the seam stands in for a
	// disk that is full.
	root = armedWorld(t, armedText("lint/go@."))
	old := gateWrite
	gateWrite = func(string, verify.ArmedSet) error { return errors.New("disk full") }
	t.Cleanup(func() { gateWrite = old })
	for sub, args := range map[string][]string{"arm": {"gate", "arm", "test/go@.", "--root", root}, "disarm": {"gate", "disarm", "lint/go@.", "--root", root}} {
		if code, out, errOut := run(args...); code != 1 || out != "" || errOut != "loomux gate "+sub+": disk full\n" {
			t.Errorf("%s: code %d, out %q, err %q", sub, code, out, errOut)
		}
	}
}

// Without --root the project is found upwards, as check finds it.
func TestGateFindsTheRootUpwards(t *testing.T) {
	root := armedWorld(t, armedText())
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644)
	sub := filepath.Join(root, "deep", "er")
	os.MkdirAll(sub, 0o755)
	t.Chdir(sub)
	if code, out, _ := run("gate", "status"); code != 0 || !strings.Contains(out, "lint/go@.: probation") {
		t.Fatalf("code %d, out %q", code, out)
	}
	// The configuration alone is enough: the file goes beside it.
	root = armedWorld(t, "")
	os.WriteFile(filepath.Join(root, ".loomux", "config.toml"), []byte("[verify]\n"), 0o644)
	sub = filepath.Join(root, "sub")
	os.MkdirAll(sub, 0o755)
	t.Chdir(sub)
	if code, out, errOut := run("gate", "disarm", "--all"); code != 0 || armedFile(t, root) != armedText() {
		t.Fatalf("beside the configuration: code %d, out %q, err %q", code, out, errOut)
	}
}

// A key typed with Windows separators names the lane the file names with
// forward ones, as LaneKey writes every area. Detection finds areas one
// level deep, so a nested one is an entry a human or an older run left:
// disarm takes it out all the same.
func TestGateTakesAKeyWithBackslashes(t *testing.T) {
	root := armedWorld(t, armedText("lint/go@sub/dir"))
	if code, out, errOut := run("gate", "arm", `lint\go@.`, "--root", root); code != 0 || out != "armed: lint/go@.\n" || armedFile(t, root) != armedText("lint/go@.", "lint/go@sub/dir") {
		t.Fatalf("arm: code %d, out %q, err %q", code, out, errOut)
	}
	if code, out, errOut := run("gate", "disarm", `lint\go@sub\dir`, "--root", root); code != 0 || out != "probation: lint/go@sub/dir\n" || armedFile(t, root) != armedText("lint/go@.") {
		t.Fatalf("disarm: code %d, out %q, err %q", code, out, errOut)
	}
}

// A project in probation needs no configuration: a Go project runs on the
// presets. From a subdirectory gate finds it by its armed lanes, and without
// them by the repository's top level -- never by the directory it runs in.
func TestGateFindsAProjectWithoutAConfiguration(t *testing.T) {
	// Outside any repository and without a file, the directory gate runs in
	// is all there is. It comes first: the repository below takes in the
	// test's own directory.
	alone := t.TempDir()
	t.Chdir(alone)
	if code, out, errOut := run("gate", "disarm", "--all"); code != 0 || armedFile(t, alone) != armedText() {
		t.Fatalf("outside a repository: code %d, out %q, err %q", code, out, errOut)
	}
	// The repository holds more than the project: the file says where the
	// project is, before the top level does.
	root := armedWorld(t, armedText())
	mustRunGit(t, filepath.Dir(root), "init", "-q")
	sub := filepath.Join(root, "sub")
	os.MkdirAll(sub, 0o755)
	t.Chdir(sub)
	if code, out, errOut := run("gate", "status"); code != 0 || !strings.Contains(out, "lint/go@.: probation") {
		t.Fatalf("status: code %d, out %q, err %q", code, out, errOut)
	}
	if code, out, errOut := run("gate", "arm", "lint/go@."); code != 0 || armedFile(t, root) != armedText("lint/go@.") {
		t.Fatalf("arm: code %d, out %q, err %q", code, out, errOut)
	}
	if code, out, errOut := run("gate", "disarm", "--all"); code != 0 || armedFile(t, root) != armedText() {
		t.Fatalf("disarm --all: code %d, out %q, err %q", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(sub, ".loomux")); err == nil {
		t.Fatal("gate wrote into the subdirectory")
	}
	// From the project's own directory, the file there is read first.
	t.Chdir(root)
	if code, out, errOut := run("gate", "status"); code != 0 || !strings.Contains(out, "lint/go@.: probation") {
		t.Fatalf("from the project itself: code %d, out %q, err %q", code, out, errOut)
	}
	// No file yet: the top level of the repository is the project.
	root = armedWorld(t, "")
	mustRunGit(t, root, "init", "-q")
	sub = filepath.Join(root, "sub")
	os.MkdirAll(sub, 0o755)
	t.Chdir(sub)
	if code, out, errOut := run("gate", "disarm", "--all"); code != 0 || armedFile(t, root) != armedText() {
		t.Fatalf("top level: code %d, out %q, err %q", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(sub, ".loomux")); err == nil {
		t.Fatal("gate wrote into the subdirectory of a repository")
	}
}

// A working directory reached through a junction into a repository's
// subdirectory is walked as git names it: up through the repository, never
// through the junction's parents.
func TestGateWalksTheRepositoryNotTheJunction(t *testing.T) {
	repo := armedWorld(t, "")
	mustRunGit(t, repo, "init", "-q")
	os.MkdirAll(filepath.Join(repo, "sub", "deeper"), 0o755)
	home := t.TempDir()
	homeText := armedText("lint/go@.")
	os.MkdirAll(filepath.Join(home, ".loomux"), 0o755)
	os.WriteFile(filepath.Join(home, ".loomux", "armed.toml"), []byte(homeText), 0o644)
	link := filepath.Join(home, "j")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, filepath.Join(repo, "sub")).CombinedOutput(); err != nil {
		t.Skipf("no junction: %v %s", err, out)
	}
	t.Chdir(filepath.Join(link, "deeper"))
	if code, out, errOut := run("gate", "disarm", "--all"); code != 0 || armedFile(t, repo) != armedText() || armedFile(t, home) != homeText {
		t.Fatalf("through a junction: code %d, out %q, err %q", code, out, errOut)
	}
}

// The search for the armed lanes ends at the repository's top: a file
// above it belongs to another project. Outside a repository there is no
// top to end at, and gate does not search at all.
func TestGateSearchesTheArmedLanesWithinTheRepository(t *testing.T) {
	outerText := armedText("lint/go@.")
	// A file above a directory that is no repository is not the project's.
	parent := armedWorld(t, outerText)
	child := filepath.Join(parent, "p", "sub")
	os.MkdirAll(child, 0o755)
	t.Chdir(child)
	if code, out, errOut := run("gate", "disarm", "--all"); code != 0 || armedFile(t, child) != armedText() || armedFile(t, parent) != outerText {
		t.Fatalf("outside a repository: code %d, out %q, err %q", code, out, errOut)
	}
	// A repository inside another project's repository: the inner top is
	// the project, and the outer file stays as it is, byte for byte.
	outer := armedWorld(t, outerText)
	mustRunGit(t, outer, "init", "-q")
	inner := filepath.Join(outer, "inner")
	os.MkdirAll(filepath.Join(inner, "sub"), 0o755)
	mustRunGit(t, inner, "init", "-q")
	t.Chdir(filepath.Join(inner, "sub"))
	if code, out, errOut := run("gate", "disarm", "--all"); code != 0 || armedFile(t, inner) != armedText() || armedFile(t, outer) != outerText {
		t.Fatalf("nested repository: code %d, out %q, err %q", code, out, errOut)
	}
	// The top's file is found past a .loomux of a subdirectory that holds
	// no file -- check run there leaves one with its state.
	top := armedWorld(t, outerText)
	mustRunGit(t, top, "init", "-q")
	sub := filepath.Join(top, "sub")
	os.MkdirAll(filepath.Join(sub, ".loomux", "state"), 0o755)
	t.Chdir(sub)
	if code, out, errOut := run("gate", "status"); code != 0 || !strings.Contains(out, "lint/go@.: armed") {
		t.Fatalf("past a stray .loomux: code %d, out %q, err %q", code, out, errOut)
	}
}
