package hooks

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/check"
	"github.com/xidus90/loomux/internal/brain/identity"
	"github.com/xidus90/loomux/internal/brain/wiki"
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

// sourceProject is wikiProject with a register and a cited doc whose row
// is behind it: the lane must be red.
func sourceProject(t *testing.T, manifest string) string {
	t.Helper()
	old, now := "# A\n", "# A changed\n"
	pinned := fmt.Sprintf("---\ntitle: T\ntype: concept\ndescription: d\nsources:\n  - id: a\n    resource: brain://project/x/docs/de/a.md\n    doc_id: 01A\n    content_hash: \"%s\"\n    revision: 3\n---\n\n# T\n", identity.HashBytes([]byte(old)))
	root, _ := wikiProject(t, manifest, pinned)
	if err := os.MkdirAll(filepath.Join(root, "docs", "de"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		filepath.Join("docs", "de", "a.md"): now,
		"_identities.tsv": identity.RenderIdentities(map[string]identity.Identity{
			"docs/de/a.md": {DocID: "01A", Relative: "docs/de/a.md", ContentHash: identity.HashBytes([]byte(old)), Revision: 3},
		}),
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// Its own lane key, so probation holds it apart from the bundle lint.
func TestWikiSourceJobsAreALaneOfTheirOwn(t *testing.T) {
	root := sourceProject(t, "")
	eff, facts := loadedFor(t, root)
	jobs := WikiSourceJobs(eff, facts, root, []string{"lint"})
	if len(jobs) != 1 || jobs[0].Name != "lint/wiki-sources" || verify.LaneKey(jobs[0]) != "lint/wiki-sources@." {
		t.Fatalf("jobs %+v", jobs)
	}
	out, err := jobs[0].Fn()
	if err == nil || !strings.Contains(out, "  • [wiki-sources:register-stale] docs/de/a.md: changed since its register row (revision 3); cited by page.md\n") ||
		!strings.Contains(out, "fix them, then run `loomux wiki repin`") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// The same three conditions, with the lane's own switch, and the brain
// module's.
func TestWikiSourceJobsOnlyForLintAWikiAndTheirSwitch(t *testing.T) {
	root := sourceProject(t, "")
	eff, facts := loadedFor(t, root)
	if jobs := WikiSourceJobs(eff, facts, root, []string{"test"}); jobs != nil {
		t.Fatalf("jobs without lint: %+v", jobs)
	}
	plain := t.TempDir()
	plainEff, plainFacts := loadedFor(t, plain)
	if jobs := WikiSourceJobs(plainEff, plainFacts, plain, []string{"lint"}); jobs != nil {
		t.Fatalf("jobs without a wiki: %+v", jobs)
	}
	for _, manifest := range []string{"[verify.wiki-sources]\nlint = false\n", "[modules]\nbrain = false\n"} {
		off := sourceProject(t, manifest)
		offEff, offFacts := loadedFor(t, off)
		if jobs := WikiSourceJobs(offEff, offFacts, off, []string{"lint"}); jobs != nil {
			t.Fatalf("%q: jobs with the lane off: %+v", manifest, jobs)
		}
	}
	// The bundle lint's switch is not this lane's.
	on := sourceProject(t, "[verify.wiki]\nlint = false\n")
	onEff, onFacts := loadedFor(t, on)
	if jobs := WikiSourceJobs(onEff, onFacts, on, []string{"lint"}); len(jobs) != 1 {
		t.Fatalf("jobs %+v", jobs)
	}
}

// A wiki laid out elsewhere is found where the manifest says.
func TestWikiSourceJobsFollowTheLayout(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "[area]\nscope = \"project/x\"\n[layout]\nwiki = \"kb\"\n")
	if err := os.MkdirAll(filepath.Join(root, "kb"), 0o755); err != nil {
		t.Fatal(err)
	}
	pinned := "---\ntitle: T\ntype: concept\ndescription: d\nsources:\n  - id: a\n    resource: brain://project/x/a.md\n    doc_id: 01A\n    content_hash: \"sha256:old\"\n    revision: 3\n---\n"
	for name, body := range map[string]string{
		filepath.Join("kb", "t.md"): pinned,
		"a.md":                      "# A\n",
		"_identities.tsv":           "doc_id\tpfad\tcontent_hash\trevision\n01A\ta.md\tsha256:old\t3\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	eff, facts := loadedFor(t, root)
	jobs := WikiSourceJobs(eff, facts, root, []string{"lint"})
	if len(jobs) != 1 {
		t.Fatalf("jobs %+v", jobs)
	}
	if out, err := jobs[0].Fn(); err == nil || !strings.Contains(out, "[wiki-sources:register-stale] a.md:") {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// The report names, per rule present, what to do; a lane that cannot read
// fails rather than stays silent; no findings is green.
func TestSourceLaneReport(t *testing.T) {
	findings := []wiki.SourceFinding{
		{Rule: wiki.RuleRowMissing, Page: "t.md", Message: "m1"},
		{Rule: wiki.RuleCycle, Page: "t.md", Message: "m2"},
		{Rule: wiki.RulePinStale, Page: "t.md", Message: "m3"},
	}
	out, err := sourceLane(t.TempDir(), "wiki", func(string, string, string) ([]wiki.SourceFinding, error) { return findings, nil })
	if err == nil || err.Error() != "the wiki sources have 3 finding(s)" {
		t.Fatalf("err %v", err)
	}
	for _, want := range []string{
		"  • [wiki-sources:row-missing] t.md: m1\n",
		"  row-missing: repin adds no register row; a newly cited file gets one from `loomux reindex` in the registry's checkout (see `loomux wiki repin` in the CLI reference)\n",
		"  cycle: pages that pin each other can never both be current; drop one of the pins\n",
		"  register-stale, pin-stale: check the citing pages against the changed files, fix them, then run `loomux wiki repin`\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("out %q lacks %q", out, want)
		}
	}
	if strings.Contains(out, "source-missing, doc-id") {
		t.Fatalf("a hint for a rule not present: %q", out)
	}
	boom := errors.New("denied")
	if _, err := sourceLane(t.TempDir(), "wiki", func(string, string, string) ([]wiki.SourceFinding, error) { return nil, boom }); !errors.Is(err, boom) {
		t.Fatalf("err %v", err)
	}
	if out, err := sourceLane(t.TempDir(), "wiki", func(string, string, string) ([]wiki.SourceFinding, error) { return nil, nil }); out != "" || err != nil {
		t.Fatalf("out %q, err %v", out, err)
	}
}

// A hint stands once, however many of its rules are present.
func TestSourceLaneSaysAHintOnce(t *testing.T) {
	findings := []wiki.SourceFinding{
		{Rule: wiki.RuleRegisterStale, Page: "t.md", Message: "m1"},
		{Rule: wiki.RulePinStale, Page: "t.md", Message: "m2"},
	}
	out, _ := sourceLane(t.TempDir(), "wiki", func(string, string, string) ([]wiki.SourceFinding, error) { return findings, nil })
	if n := strings.Count(out, "  register-stale, pin-stale: "); n != 1 {
		t.Fatalf("%d hint lines in %q", n, out)
	}
}

// The scope comes from the manifest; without one the drift is asked with
// none, and a manifest that does not read counts as none, as the lint reads
// it.
func TestSourceLaneTakesTheScopeFromTheManifest(t *testing.T) {
	var asked []string
	spy := func(_ string, _ string, scope string) ([]wiki.SourceFinding, error) {
		asked = append(asked, scope)
		return nil, nil
	}
	declared := t.TempDir()
	writeManifest(t, declared, "[area]\nscope = \"project/x\"\n")
	broken := t.TempDir()
	writeManifest(t, broken, "[area\n")
	for _, root := range []string{declared, t.TempDir(), broken} {
		if _, err := sourceLane(root, "wiki", spy); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(asked, []string{"project/x", "", ""}) {
		t.Fatalf("asked %q", asked)
	}
}
