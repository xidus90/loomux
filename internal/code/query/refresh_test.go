package query

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/code/ask"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/gitenv"
)

// refreshed runs RefreshGraph without waiting and returns what it said.
func refreshed(t *testing.T, root string) (ask.Status, []string, error) {
	t.Helper()
	var notices []string
	status, err := RefreshGraph(root, 0, func(s string) { notices = append(notices, s) })
	return status, notices, err
}

func TestRefreshGraphWithoutAGraphIsErrNoGraph(t *testing.T) {
	if _, _, err := refreshed(t, repo(t, sample())); !errors.Is(err, ErrNoGraph) {
		t.Fatalf("err %v, want ErrNoGraph", err)
	}
}

func TestRefreshGraphIsCleanOnAFreshGraph(t *testing.T) {
	status, notices, err := refreshed(t, built(t))
	if err != nil || status != ask.StatusClean || len(notices) != 0 {
		t.Fatalf("status %v, notices %q, err %v; want clean and silent", status, notices, err)
	}
}

// A gate must drive the graph to fresh on everything the next build repairs;
// only the reason in the notice tells the cases apart.
func TestRefreshGraphRebuildsAGraphThatCannotStand(t *testing.T) {
	cases := map[string]struct {
		wiring func(t *testing.T, root string) string
		want   string
	}{
		"outdated schema": {
			wiring: func(*testing.T, string) string { return `{"meta":{"version":1},"nodes":[],"edges":[]}` + "\n" },
			want:   "graph schema is outdated, rebuilding",
		},
		"foreign extractor": {
			wiring: func(t *testing.T, root string) string {
				b, err := os.ReadFile(store.WiringPath(root))
				if err != nil {
					t.Fatal(err)
				}
				var doc map[string]any
				if err := json.Unmarshal(b, &doc); err != nil {
					t.Fatal(err)
				}
				doc["meta"].(map[string]any)["extractor"] = "go/0"
				out, err := json.Marshal(doc)
				if err != nil {
					t.Fatal(err)
				}
				return string(out)
			},
			want: `graph was built by extractor "go/0", rebuilding`,
		},
		"unreadable": {
			wiring: func(*testing.T, string) string { return "{corrupt" },
			want:   "graph is unreadable (",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := built(t)
			writeWiring(t, root, c.wiring(t, root))
			status, notices, err := refreshed(t, root)
			if err != nil || status != ask.StatusRebuilt {
				t.Fatalf("status %v, err %v; want rebuilt", status, err)
			}
			if len(notices) == 0 || !strings.HasPrefix(notices[0], c.want) || !strings.HasSuffix(notices[0], ", rebuilding") {
				t.Fatalf("notices %q, want the first to give %q", notices, c.want)
			}
			if _, err := store.Read(root); err != nil {
				t.Fatalf("the rebuilt graph does not read: %v", err)
			}
		})
	}
}

func TestRefreshGraphReportsAFailedRebuild(t *testing.T) {
	root := built(t)
	writeWiring(t, root, "{corrupt")
	// A file the extractor cannot parse makes the forced rebuild fail.
	if err := os.WriteFile(root+"/lib/lib.go", []byte("package lib\n\nfunc {"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := refreshed(t, root)
	var failed *ask.RebuildError
	if !errors.As(err, &failed) {
		t.Fatalf("err %v, want a rebuild error", err)
	}
}

// noGit fails the test on any git call: the probe must answer from the disk.
func noGit(t *testing.T) {
	t.Helper()
	old := gitOutput
	gitOutput = func(_ string, args ...string) ([]byte, error) {
		t.Fatalf("git %v was called", args)
		return nil, nil
	}
	t.Cleanup(func() { gitOutput = old })
}

func TestGraphReadyWithoutAGraphAsksNoGit(t *testing.T) {
	root := repo(t, sample())
	noGit(t)
	if ok, note := GraphReady(root); ok || note != "no graph at .loomux/state/graph/wiring.json" {
		t.Fatalf("%v %q", ok, note)
	}
}

func TestGraphReadyWithoutAHead(t *testing.T) {
	root := built(t)
	git(t, root, "init", "-q")
	if ok, note := GraphReady(root); ok || !strings.HasPrefix(note, "no HEAD to compare with: git rev-parse: ") {
		t.Fatalf("%v %q", ok, note)
	}
}

// During a merge the index holds the other side's changes as if they were
// this commit's; the probe finds MERGE_HEAD where git says it lies.
func TestGraphReadyDuringAMerge(t *testing.T) {
	root := gitRepo(t)
	out, err := runGit(root, "rev-parse", "--git-path", "MERGE_HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, strings.TrimSpace(string(out))), []byte("0000\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, note := GraphReady(root); ok || note != "a merge is in progress" {
		t.Fatalf("%v %q", ok, note)
	}
}

// In a linked worktree git names MERGE_HEAD by an absolute path.
func TestGraphReadyTakesAnAbsoluteMergeHead(t *testing.T) {
	root := built(t)
	mergeHead := filepath.Join(t.TempDir(), "MERGE_HEAD")
	if err := os.WriteFile(mergeHead, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	fakeGit(t, answer(mergeHead+"\n0000\n", nil))
	if ok, note := GraphReady(root); ok || note != "a merge is in progress" {
		t.Fatalf("%v %q", ok, note)
	}
}

// commitAdd rewrites the body of Add in calc/calc.go from one expression to
// another and commits it.
func commitAdd(t *testing.T, root, from, to string) {
	t.Helper()
	p := filepath.Join(root, "calc", "calc.go")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	next := strings.Replace(string(data), "return "+from+"\n", "return "+to+"\n", 1)
	if next == string(data) {
		t.Fatalf("calc.go no longer holds `return %s`", from)
	}
	if err := os.WriteFile(p, []byte(next), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "commit", "-qam", to)
}

// stopped is gitRepo with a cherry-pick, a revert or a rebase stopped on a
// conflict in calc/calc.go; op is the git command and its flags.
func stopped(t *testing.T, op string) string {
	t.Helper()
	root := gitRepo(t)
	var args []string
	switch op {
	case "revert":
		commitAdd(t, root, "a + b", "b + a")
		commitAdd(t, root, "b + a", "b + a + 0")
		args = []string{"revert", "--no-edit", "HEAD~1"}
	default:
		git(t, root, "checkout", "-qb", "side")
		commitAdd(t, root, "a + b", "b + a")
		git(t, root, "checkout", "-q", "-")
		commitAdd(t, root, "a + b", "a + b + 0")
		args = append(strings.Fields(op), "side")
	}
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = gitenv.Environ()
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("git %v did not stop on a conflict:\n%s", args, out)
	}
	return root
}

// A stopped cherry-pick, revert or rebase leaves the index holding another
// commit's changes, as a merge does.
func TestGraphReadyDuringAStoppedSequence(t *testing.T) {
	for op, want := range map[string]string{
		"cherry-pick": "a cherry-pick is in progress",
		"revert":      "a revert is in progress",
		"rebase":      "a rebase is in progress",
		// The apply backend keeps rebase-apply instead of rebase-merge.
		"rebase --apply": "a rebase is in progress",
	} {
		t.Run(op, func(t *testing.T) {
			if ok, note := GraphReady(stopped(t, op)); ok || note != want {
				t.Fatalf("%v %q, want %q", ok, note, want)
			}
		})
	}
}

func TestGraphReadyWithNothingStaged(t *testing.T) {
	if ok, note := GraphReady(gitRepo(t)); ok || note != "nothing staged" {
		t.Fatalf("%v %q", ok, note)
	}
}

func TestGraphReadyWithAStagedChange(t *testing.T) {
	if ok, note := GraphReady(stagedAdd(t)); !ok || note != "" {
		t.Fatalf("%v %q", ok, note)
	}
}

// Any exit of `diff --cached --quiet` but 0 and 1 is git failing, and its
// complaint is the note.
func TestGraphReadyReportsAFailingDiff(t *testing.T) {
	root := built(t)
	fakeGit(t, answer(".git/MERGE_HEAD\n0000\n", nil), answer("", errors.New("git diff: exit status 128: fatal: bad index")))
	if ok, note := GraphReady(root); ok || note != "git diff: exit status 128: fatal: bad index" {
		t.Fatalf("%v %q", ok, note)
	}
}
