package hooks

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/detect"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/verify"
)

// loadedFor is what a caller of WikiGateJobs already holds: the facts of the
// tree and the config laid over the presets for them.
func loadedFor(t *testing.T, root string) (verify.Effective, detect.Facts) {
	t.Helper()
	facts := detect.Detect(os.DirFS(root))
	eff, err := editLoad(root, facts)
	if err != nil {
		t.Fatal(err)
	}
	return eff, facts
}

func TestWikiGateJobsRunTheGateInProcess(t *testing.T) {
	root, _ := wikiProject(t, "", cleanPage)
	eff, facts := loadedFor(t, root)
	jobs := WikiGateJobs(eff, facts, root, []string{"lint", "test"})
	if len(jobs) != 1 || jobs[0].Name != "lint/wiki" || jobs[0].Fn == nil {
		t.Fatalf("jobs %+v", jobs)
	}
	if out, err := jobs[0].Fn(); err != nil {
		t.Fatalf("a valid bundle failed: %s %v", out, err)
	}
}

// A page the bundle lint refuses is red, and the line is the one the gate
// writes for the same finding.
func TestWikiGateJobsReportAViolation(t *testing.T) {
	root, _ := wikiProject(t, "", cleanPage)
	writeWikiPage(t, root, "broken.md", "no frontmatter at all\n")
	eff, facts := loadedFor(t, root)
	out, err := WikiGateJobs(eff, facts, root, []string{"lint"})[0].Fn()
	if err == nil || !strings.Contains(out, "[wiki-lint:missing-type] broken.md: ") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// The lane checks the bundle and never the drift rule `loomux wiki-gate`
// carries beside it: a commit that changes code and no page is the ordinary
// case, and a lane that refused it would refuse every code-only commit --
// this repository's own first.
func TestWikiGateJobsPassWhenOnlyCodeChanged(t *testing.T) {
	root, _ := wikiProject(t, "", cleanPage)
	// A repository with a.txt and the bundle committed, and then one code
	// file changed and no page.
	gitInit(t, root)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "the bundle")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if status := gitStatus(t, root); !strings.Contains(status, "a.txt") || strings.Contains(status, "wiki") {
		t.Fatalf("the world must show a code-only change, got %q", status)
	}
	eff, facts := loadedFor(t, root)
	if out, err := WikiGateJobs(eff, facts, root, []string{"lint"})[0].Fn(); err != nil {
		t.Fatalf("a code-only change is green: %s %v", out, err)
	}
}

// A lint that cannot read the bundle is the lane's failure and not its
// silence.
func TestLintBundleReportsALintThatFails(t *testing.T) {
	boom := errors.New("no such bundle")
	out, err := lintBundle("wiki", func(string) ([]check.Finding, error) { return nil, boom })
	if !errors.Is(err, boom) || out != "" {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// The same three conditions the edit lane has: lint was asked for, the
// project has a wiki, and nobody switched the lane off.
func TestWikiGateJobsOnlyForLintAndAWiki(t *testing.T) {
	root, _ := wikiProject(t, "", cleanPage)
	eff, facts := loadedFor(t, root)
	if jobs := WikiGateJobs(eff, facts, root, []string{"test"}); jobs != nil {
		t.Fatalf("jobs without lint: %+v", jobs)
	}
	plain := t.TempDir()
	plainEff, plainFacts := loadedFor(t, plain)
	if jobs := WikiGateJobs(plainEff, plainFacts, plain, []string{"lint"}); jobs != nil {
		t.Fatalf("jobs without a wiki: %+v", jobs)
	}
	// The brain module owns the lane: switching it off switches the lane off.
	for _, manifest := range []string{"[verify.wiki]\nlint = false\n", "[modules]\nbrain = false\n"} {
		off, _ := wikiProject(t, manifest, cleanPage)
		offEff, offFacts := loadedFor(t, off)
		if jobs := WikiGateJobs(offEff, offFacts, off, []string{"lint"}); jobs != nil {
			t.Fatalf("%q: jobs with the lane off: %+v", manifest, jobs)
		}
	}
}

// gitStatus is what the drift rule read: `git` beside it answers nothing, and
// this world has to be seen to be believed. The environment is the isolated
// one every git call of this suite uses.
func gitStatus(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	return string(out)
}
