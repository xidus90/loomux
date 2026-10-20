package compat

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/gitenv"
)

func keysOf(jobs []Job) []string {
	var out []string
	for _, j := range jobs {
		out = append(out, j.Entry+"@"+j.OS)
	}
	return out
}

func TestMatrixSelects(t *testing.T) {
	list, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := Matrix(list, Selection{}); got == nil || len(got) != 0 {
		t.Fatalf("no selection = %v, want empty non-nil", got)
	}
	full := Matrix(list, Selection{Full: true})
	if len(full) != 2*len(list) || full[0] != (Job{Entry: list[0].Key(), OS: "linux"}) || full[1] != (Job{Entry: list[0].Key(), OS: "windows"}) {
		t.Fatalf("full = %v", full)
	}
	if got := keysOf(Matrix(list, Selection{Keys: []string{list[1].Key()}})); !reflect.DeepEqual(got, []string{list[1].Key() + "@linux", list[1].Key() + "@windows"}) {
		t.Errorf("entry = %v", got)
	}
	if got := Matrix(list, Selection{ChangedEntries: []string{list[2].Key()}}); len(got) != 2 || got[0].Entry != list[2].Key() {
		t.Errorf("changed entry = %v", got)
	}
	// a fixture is shared by every entry that names it
	users := 0
	for _, e := range list {
		if e.Fixture == "godot4-gdunit4" {
			users++
		}
	}
	if users < 2 {
		t.Fatal("expected a fixture shared by two entries")
	}
	if got := Matrix(list, Selection{ChangedFixtures: []string{"godot4-gdunit4"}}); len(got) != 2*users {
		t.Errorf("changed fixture = %v", got)
	}
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func put(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestChangedSince(t *testing.T) {
	list, _ := Load()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	// the old list differs in the first entry's version and names a fixture
	// that no longer exists
	old := strings.Replace(string(listText), `version = "3.6"`, `version = "3.5"`, 1) +
		"\n[[entry]]\nstack = \"go\"\nruntime = \"go\"\nversion = \"1.20\"\nfixture = \"gone\"\nkinds = [\"lint\"]\n"
	put(t, dir, listPath, old)
	put(t, dir, fixturesDir+"godot4-plain/a.gd", "1")
	put(t, dir, "docs/x.md", "1")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "one")
	put(t, dir, fixturesDir+"godot4-plain/a.gd", "2")
	put(t, dir, fixturesDir+"godot4-plain/sub/b.gd", "2")
	put(t, dir, "docs/x.md", "2")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "two")

	fx, en, err := ChangedSince(dir, "HEAD~1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fx, []string{"godot4-plain"}) {
		t.Errorf("fixtures = %v", fx)
	}
	if !reflect.DeepEqual(en, []string{list[0].Key()}) {
		t.Errorf("entries = %v, want only the first", en)
	}

	// only documentation changed: nothing to run
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "three")
	put(t, dir, "docs/x.md", "3")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "four")
	if fx, _, _ = ChangedSince(dir, "HEAD~1"); len(fx) != 0 {
		t.Errorf("a docs-only change selected fixtures %v", fx)
	}
}

func TestChangedSinceWithoutAListAtRef(t *testing.T) {
	list, _ := Load()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	put(t, dir, "a.txt", "1")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "one")
	_, en, err := ChangedSince(dir, "HEAD")
	if err != nil || len(en) != len(list) {
		t.Fatalf("entries %d, err %v; want all %d", len(en), err, len(list))
	}
}

func TestChangedSinceWithoutHEAD(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	put(t, dir, "a.txt", "1")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "one")
	rev := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	rev.Env = gitenv.Environ()
	sha, err := rev.Output()
	if err != nil {
		t.Fatal(err)
	}
	// point HEAD at a branch that does not exist: the ref resolves, the diff cannot
	put(t, dir, ".git/HEAD", "ref: refs/heads/unborn\n")
	if _, _, err := ChangedSince(dir, strings.TrimSpace(string(sha))); err == nil {
		t.Error("a repository without HEAD passed")
	}
}

func TestChangedSinceErrors(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	put(t, dir, listPath, "[[entry]\n")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-qm", "broken list")
	if _, _, err := ChangedSince(dir, "no-such-ref"); err == nil {
		t.Error("an unknown ref passed")
	}
	if _, _, err := ChangedSince(dir, "HEAD"); err == nil {
		t.Error("an unreadable old list passed")
	}
}
