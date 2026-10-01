package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/child"
	"github.com/xidus90/loomux/internal/verify"
)

// armedWorld is goWorld with the file holding text; "" leaves the file out.
// It pins what the process inherits: this suite runs inside loomux's own
// pre-commit gate, whose GIT_INDEX_FILE names loomux's index and would make
// every arming run here a commit of paths. Nothing is staged for real either:
// the world is no repository, and staging is gitwork's to test.
func armedWorld(t *testing.T, text string) string {
	t.Helper()
	t.Setenv("GIT_INDEX_FILE", "")
	oldStage := checkStage
	checkStage = func(string, string, string) error { return nil }
	t.Cleanup(func() { checkStage = oldStage })
	root := goWorld(t)
	if err := os.MkdirAll(filepath.Join(root, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if text != "" {
		if err := os.WriteFile(filepath.Join(root, ".loomux", "armed.toml"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func armedText(keys ...string) string { return verify.ArmedSet{}.With(keys...).Text() }

func armedFile(t *testing.T, root string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".loomux", "armed.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// failing answers a command that begins with words with 1, every other green.
func failing(words ...string) func(child.Spec) child.Result {
	return func(s child.Spec) child.Result {
		if len(s.Argv) >= len(words) && slices.Equal(s.Argv[:len(words)], words) {
			return child.Result{Code: 1, Stdout: strings.Join(words, " ") + ": bad\n"}
		}
		return green(s)
	}
}

func TestWithoutTheFileARedLaneFailsAndNothingSpeaksOfProbation(t *testing.T) {
	root := armedWorld(t, "")
	stubCheck(t, failing("go", "vet"))
	code, out, errOut := run("check", "precommit", "--root", root)
	if code != 1 || strings.Contains(out, "probation") || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestALaneInProbationWarnsAndAnArmedOneFails(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, failing("go", "vet"))
	code, out, errOut := run("check", "precommit", "--root", root)
	if code != 0 || errOut != "" {
		t.Fatalf("probation: code %d, out %q, err %q", code, out, errOut)
	}
	for _, want := range []string{
		"lint/go: failed (probation) [preset]",
		"go vet: bad",
		"probation: coverage/go@., lint/go@., test/go@. (warn only until a green commit arms them)\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %q", want, out)
		}
	}
	if !strings.HasSuffix(out, "arms them)\n") {
		t.Errorf("the probation line ends the report: %q", out)
	}

	root = armedWorld(t, armedText("lint/go@."))
	code, out, _ = run("check", "precommit", "--root", root)
	if code != 1 || strings.Contains(out, "lint/go: failed (probation)") || !strings.Contains(out, "probation: coverage/go@., test/go@. (") {
		t.Fatalf("armed: code %d, out %q", code, out)
	}
}

func TestBlockedBehindALaneInProbationDoesNotFail(t *testing.T) {
	stubCheck(t, failing("go", "test"))
	root := armedWorld(t, armedText("coverage/go@.", "lint/go@."))
	code, out, _ := run("check", "precommit", "--root", root)
	if code != 0 || !strings.Contains(out, "coverage/go: blocked (probation) [preset] by test/go") {
		t.Fatalf("behind probation: code %d, out %q", code, out)
	}
	root = armedWorld(t, armedText("coverage/go@.", "lint/go@.", "test/go@."))
	code, out, _ = run("check", "precommit", "--root", root)
	if code != 1 || !strings.Contains(out, "coverage/go: blocked [preset] by test/go") {
		t.Fatalf("behind an armed lane: code %d, out %q", code, out)
	}
}

// staging records what --arm stages: root, index and path of each call.
func staging(t *testing.T, answer error) *[]string {
	t.Helper()
	calls := []string{}
	old := checkStage
	checkStage = func(root, index, rel string) error {
		calls = append(calls, root+"|"+index+"|"+rel)
		return answer
	}
	t.Cleanup(func() { checkStage = old })
	return &calls
}

// handed stands in for the index git hands the hook; it records what the
// command passed on from its environment.
func handed(t *testing.T, index string, whole bool) *[]string {
	t.Helper()
	asked := []string{}
	old := checkCommitIndex
	checkCommitIndex = func(root, inherited string) (string, bool) {
		asked = append(asked, root+"|"+inherited)
		return index, whole
	}
	t.Cleanup(func() { checkCommitIndex = old })
	return &asked
}

func TestArmWritesTheGreenLanesOfAGreenRun(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staged := staging(t, nil)
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "" {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	want := armedText("coverage/go@.", "lint/go@.", "test/go@.")
	if got := armedFile(t, root); got != want {
		t.Fatalf("file %q, want %q", got, want)
	}
	// Written, then staged once into git's own index: nothing was handed in.
	if !slices.Equal(*staged, []string{root + "||.loomux/armed.toml"}) {
		t.Fatalf("staged %q", *staged)
	}
	if !strings.Contains(out, "armed: coverage/go@., lint/go@., test/go@.\n") || strings.Contains(out, "probation:") {
		t.Fatalf("out %q", out)
	}
	// A second run has nothing to add and says nothing about arming.
	info, _ := os.Stat(filepath.Join(root, ".loomux", "armed.toml"))
	code, out, _ = run("check", "precommit", "--arm", "--root", root)
	after, _ := os.Stat(filepath.Join(root, ".loomux", "armed.toml"))
	if code != 0 || strings.Contains(out, "armed:") || !after.ModTime().Equal(info.ModTime()) || armedFile(t, root) != want || len(*staged) != 1 {
		t.Fatalf("second run: code %d, out %q, staged %q", code, out, *staged)
	}
	// Without --arm nothing is written, green or not.
	root = armedWorld(t, armedText())
	if code, _, _ = run("check", "precommit", "--root", root); code != 0 || armedFile(t, root) != armedText() {
		t.Fatalf("a run without --arm wrote: %q", armedFile(t, root))
	}
}

// A commit of paths (`git commit <path>`, --only) hands the hook an index
// that does not become the real one: a file staged there would be committed
// and stand in the real index as a staged revert. Such a run arms nothing,
// writes nothing and stages nothing; the next whole commit does.
func TestArmLeavesAPartialCommitAlone(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staged := staging(t, nil)
	t.Setenv("GIT_INDEX_FILE", "next-index-4711.lock")
	asked := handed(t, "", false)
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "" || armedFile(t, root) != armedText() || len(*staged) != 0 {
		t.Fatalf("code %d, err %q, file %q, staged %q", code, errOut, armedFile(t, root), *staged)
	}
	if !slices.Equal(*asked, []string{root + "|next-index-4711.lock"}) {
		t.Fatalf("the command did not pass its GIT_INDEX_FILE on: %q", *asked)
	}
	want := "not armed: this commit takes only some paths; the next whole commit arms the lanes\n" +
		"probation: coverage/go@., lint/go@., test/go@. (warn only until a green commit arms them)\n"
	if !strings.HasSuffix(out, want) || strings.Contains(out, "\narmed: ") {
		t.Fatalf("out %q", out)
	}
	// With nothing to arm the index is not even asked about.
	root = armedWorld(t, armedText("coverage/go@.", "lint/go@.", "test/go@."))
	*asked = nil
	if code, out, _ = run("check", "precommit", "--arm", "--root", root); code != 0 || len(*asked) != 0 || strings.Contains(out, "not armed") {
		t.Fatalf("nothing to arm: code %d, asked %q, out %q", code, *asked, out)
	}
}

// Under `git commit -a` the index is index.lock, which becomes the real one:
// the file is staged into exactly the index the hook was handed.
func TestArmStagesIntoTheIndexTheHookWasHanded(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staged := staging(t, nil)
	handed(t, "/repo/.git/index.lock", true)
	if code, _, errOut := run("check", "precommit", "--arm", "--root", root); code != 0 || errOut != "" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if !slices.Equal(*staged, []string{root + "|/repo/.git/index.lock|.loomux/armed.toml"}) {
		t.Fatalf("staged %q", *staged)
	}
}

// A file that cannot be staged is said; the lanes are armed on disk, and the
// gate's code stands.
func TestArmSaysAFileItCannotStage(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	staging(t, errors.New("the path is ignored"))
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "loomux check: .loomux/armed.toml not staged, commit it by hand: the path is ignored\n" {
		t.Fatalf("code %d, err %q", code, errOut)
	}
	if armedFile(t, root) != armedText("coverage/go@.", "lint/go@.", "test/go@.") || !strings.Contains(out, "armed: coverage/go@., lint/go@., test/go@.\n") {
		t.Fatalf("file %q, out %q", armedFile(t, root), out)
	}
}

// What stands in the repository comes from a commit that went through: a
// red run arms no lane, not even one that was green in it.
func TestArmWritesNothingWhenTheRunIsRed(t *testing.T) {
	before := armedText("lint/go@.")
	root := armedWorld(t, before)
	stubCheck(t, failing("go", "vet"))
	staged := staging(t, nil)
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if code != 1 || armedFile(t, root) != before || strings.Contains(out, "armed:") || len(*staged) != 0 {
		t.Fatalf("code %d, file %q, out %q, staged %q", code, armedFile(t, root), out, *staged)
	}
	if !strings.Contains(out, "test/go: ok") {
		t.Fatalf("the world must have a green lane the run did not arm: %q", out)
	}
}

func TestArmLeavesARedLaneInProbation(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, failing("go", "vet"))
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || armedFile(t, root) != armedText("coverage/go@.", "test/go@.") {
		t.Fatalf("code %d, file %q", code, armedFile(t, root))
	}
	if !strings.Contains(out, "armed: coverage/go@., test/go@.\n") || !strings.Contains(out, "probation: lint/go@. (") {
		t.Fatalf("out %q", out)
	}
}

// The line names what this run armed, not what the file holds afterwards: a
// lane armed before is no news.
func TestArmNamesOnlyTheLanesItAdded(t *testing.T) {
	root := armedWorld(t, armedText("lint/go@."))
	stubCheck(t, green)
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || armedFile(t, root) != armedText("coverage/go@.", "lint/go@.", "test/go@.") {
		t.Fatalf("code %d, file %q", code, armedFile(t, root))
	}
	if !strings.Contains(out, "\narmed: coverage/go@., test/go@.\n") || strings.Contains(out, "lint/go@.,") {
		t.Fatalf("out %q", out)
	}
}

func TestArmWithoutTheFileWritesNone(t *testing.T) {
	root := armedWorld(t, "")
	stubCheck(t, green)
	code, out, _ := run("check", "precommit", "--arm", "--root", root)
	if _, err := os.Stat(filepath.Join(root, ".loomux", "armed.toml")); code != 0 || err == nil || strings.Contains(out, "armed:") {
		t.Fatalf("code %d, stat %v, out %q", code, err, out)
	}
}

func TestArmBelongsToThePrecommitProfile(t *testing.T) {
	root := armedWorld(t, armedText())
	seen := stubCheck(t, green)
	for _, request := range []string{"all", "lint", "stop"} {
		code, _, errOut := run("check", request, "--arm", "--root", root)
		if code != 2 || errOut != "loomux check: --arm belongs to the precommit profile\n" {
			t.Errorf("%s: code %d, err %q", request, code, errOut)
		}
	}
	if len(*seen) != 0 || armedFile(t, root) != armedText() {
		t.Fatalf("a refused call ran or wrote: %v", *seen)
	}
}

func TestAnUnreadableFileFailsARedLaneAndIsNotWrittenOver(t *testing.T) {
	const broken = "<<<<<<< HEAD\narmed = []\n=======\n"
	root := armedWorld(t, broken)
	stubCheck(t, failing("go", "vet"))
	code, out, errOut := run("check", "precommit", "--root", root)
	if code != 1 || strings.Contains(out, "probation") ||
		!strings.HasPrefix(errOut, "loomux check: .loomux/armed.toml is no TOML") || !strings.HasSuffix(errOut, "; every lane is armed\n") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	stubCheck(t, green)
	if code, _, _ = run("check", "precommit", "--arm", "--root", root); code != 0 || armedFile(t, root) != broken {
		t.Fatalf("a green --arm wrote over a broken file: code %d, %q", code, armedFile(t, root))
	}
}

// The commit does not hang on the file: a write that fails is said, and the
// gate's code stands.
func TestArmThatCannotWriteKeepsTheGatesCode(t *testing.T) {
	root := armedWorld(t, armedText())
	stubCheck(t, green)
	old := checkWriteArmed
	checkWriteArmed = func(string, verify.ArmedSet) error { return errors.New("disk full") }
	t.Cleanup(func() { checkWriteArmed = old })
	code, out, errOut := run("check", "precommit", "--arm", "--root", root)
	if code != 0 || errOut != "loomux check: .loomux/armed.toml not written: disk full\n" || strings.Contains(out, "armed:") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
	if !strings.Contains(out, "probation: coverage/go@., lint/go@., test/go@. (") {
		t.Fatalf("the lanes stay in probation: %q", out)
	}
}
