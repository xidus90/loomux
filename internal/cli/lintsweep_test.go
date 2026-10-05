package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/testlock"
)

func TestLintSweepsEveryAreaWithAWiki(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	_, b := checkedArea(t, base, "engineering/b", "")
	checkWorld(t, a, b, wikiEntry("project/nowiki", base, "", ""))
	want := "project/a\n  no findings\nengineering/b\n  no findings\nno findings\n"
	for _, args := range [][]string{{"lint"}, {"lint", "--scope", "all"}, {"lint", filepath.Join(base, "project-a")}} {
		code, out, errOut := run(args...)
		if code != 0 || out != want || errOut != "" {
			t.Errorf("%v: code %d, out %q, err %q", args, code, out, errOut)
		}
	}
	code, out, _ := run("lint", "--scope", "engineering/b")
	if code != 0 || out != "engineering/b\n  no findings\nno findings\n" {
		t.Fatalf("one scope: code %d, out %q", code, out)
	}
}

// A workspace that declares no [area] is no brain area: the sweep over all
// passes it over without a word, and naming it is refused. A workspace that
// declares itself is swept like any area.
func TestLintPassesOverAWorkspaceThatDeclaresNoArea(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	codeWiki, code := checkedArea(t, base, "project/code", "workspace = true\n")
	writeFile(t, filepath.Join(codeWiki, "..", ".loomux", "config.toml"), "[verify]\n")
	declaredWiki, declared := checkedArea(t, base, "project/declared", "workspace = true\n")
	writeFile(t, filepath.Join(declaredWiki, "..", ".loomux", "config.toml"), "[area]\nscope = \"project/declared\"\n")
	// Without a wiki the refusal is still this one, not the advice to add a
	// wiki path to its entry.
	bare := wikiEntry("project/bare", t.TempDir(), "", "workspace = true\n")
	checkWorld(t, a, code, declared, bare)
	want := "project/a\n  no findings\nproject/declared\n  no findings\nno findings\n"
	if code, out, errOut := run("lint", "--scope", "all"); code != 0 || out != want || errOut != "" {
		t.Errorf("all: code %d, out %q, err %q", code, out, errOut)
	}
	for _, scope := range []string{"project/code", "project/bare"} {
		if code, out, errOut := run("lint", "--scope", scope); code != 1 || out != "" ||
			errOut != "error: area '"+scope+"' is a workspace that declares no [area]; lint sweeps only a brain area\n" {
			t.Errorf("%s: code %d, out %q, err %q", scope, code, out, errOut)
		}
	}
}

// A workspace that keeps only an old manifest is not taken as one without
// [area]: the sweep stops at it with the hint.
func TestLintStopsAtAWorkspaceWithOnlyAnOldManifest(t *testing.T) {
	base := t.TempDir()
	oldWiki, old := checkedArea(t, base, "project/old", "workspace = true\n")
	writeFile(t, filepath.Join(oldWiki, "..", ".brain.toml"), "[area]\nscope = \"project/old\"\n")
	checkWorld(t, old)
	if code, out, errOut := run("lint", "--scope", "all"); code != 1 || out != "" || !strings.Contains(errOut, "an old manifest lies there") {
		t.Errorf("code %d, out %q, err %q", code, out, errOut)
	}
}

// The signpost is not held to link the wiki of a workspace that declares no
// [area]; an undeclared area that is no workspace it still is.
func TestLintSparesTheSignpostAWorkspaceThatDeclaresNoArea(t *testing.T) {
	base := t.TempDir()
	_, post := checkedArea(t, base, "knowledge", "signpost = true\n")
	codeWiki, code := checkedArea(t, base, "project/code", "workspace = true\n")
	writeFile(t, filepath.Join(codeWiki, "..", ".loomux", "config.toml"), "[verify]\n")
	checkWorld(t, post, code)
	if code, out, errOut := run("lint", "--scope", "knowledge"); code != 0 || out != "knowledge\n  no findings\nno findings\n" || errOut != "" {
		t.Errorf("workspace: code %d, out %q, err %q", code, out, errOut)
	}
	checkWorld(t, post, strings.Replace(code, "workspace = true\n", "", 1))
	if code, out, _ := run("lint", "--scope", "knowledge"); code != 1 || !strings.Contains(out, "unlisted-area: the signpost does not link to area 'project/code'") {
		t.Errorf("no workspace: code %d, out %q", code, out)
	}
}

func TestLintCountsErrorsAndWarnings(t *testing.T) {
	base := t.TempDir()
	wiki, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	writeFile(t, filepath.Join(wiki, "a.md"), checkedPage("a")+"\n[b](b.md) [out](../out.md)\n")
	code, out, _ := run("lint")
	if code != 0 || out != "project/a\n  a.md:outside-area: target leaves the area and cannot be checked: ../out.md\n"+
		"1 findings (0 errors, 1 warnings)\n" {
		t.Fatalf("a warning alone: code %d, out %q", code, out)
	}
	writeFile(t, filepath.Join(wiki, "c.md"), "---\ntitle: c\n---\n")
	code, out, _ = run("lint")
	if code != 1 || !strings.HasSuffix(out, "4 findings (3 errors, 1 warnings)\n") ||
		!strings.Contains(out, "  c.md:missing-type: no type; every OKF page needs one\n") {
		t.Fatalf("with errors: code %d, out %q", code, out)
	}
}

func TestLintRefusesWhatItCannotSweep(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	missing := filepath.Join(base, "gone", "wiki")
	checkWorld(t, a, wikiEntry("project/nowiki", base, "", ""), wikiEntry("project/gone", base, missing, ""))
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"lint", "--scope", "project/x"}, "error: no area named 'project/x' in the registry\n"},
		{[]string{"lint", "--scope", "project/nowiki"}, "error: area 'project/nowiki' declares no wiki path; add `wiki = ...` to its entry\n"},
		{[]string{"lint", "--scope", "project/gone"}, "error: area 'project/gone' has no wiki at " + missing +
			"; run `loomux wiki init --scope project/gone` first\n"},
		// A missing wiki stops the sweep over all too, before anything is
		// printed.
		{[]string{"lint"}, "error: area 'project/gone' has no wiki at " + missing +
			"; run `loomux wiki init --scope project/gone` first\n"},
	} {
		code, out, errOut := run(c.args...)
		if code != 1 || out != "" || errOut != c.want {
			t.Errorf("%v: code %d, out %q, err %q", c.args, code, out, errOut)
		}
	}
}

func TestLintRefusesAnUnreadableRegistry(t *testing.T) {
	checkWorld(t, "this is not TOML {{{\n")
	if code, out, errOut := run("lint"); code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestLintStopsAtADeclarationItCannotRead(t *testing.T) {
	base := t.TempDir()
	_, a := checkedArea(t, base, "project/a", "")
	_, b := checkedArea(t, base, "project/b", "")
	checkWorld(t, a, b)
	writeFile(t, filepath.Join(base, "project-b", ".loomux", "config.toml"), "[area\n")
	code, out, errOut := run("lint")
	if code != 1 || out != "project/a\n  no findings\n" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestLintStopsAtAPageItCannotRead(t *testing.T) {
	base := t.TempDir()
	wiki, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	testlock.Lock(t, filepath.Join(wiki, "b.md"))
	if code, out, errOut := run("lint"); code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestLintTakesTheAreasDeclaration(t *testing.T) {
	// The declared vocabulary and threshold, and the project family from the
	// scope: a planned page older than the threshold is long planned in a
	// project, and in no other family.
	base := t.TempDir()
	wiki, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	writeFile(t, filepath.Join(base, "project-a", ".loomux", "config.toml"),
		"[area]\nscope = \"project/a\"\n\n[wiki]\ntypes = [\"Balancing Rule\"]\nuntouched_days = 1\n")
	page := writeFile(t, filepath.Join(wiki, "a.md"), strings.Replace(checkedPage("a"), "type: Topic",
		"type: Balancing Rule\nrealization: planned", 1)+"\n[b](b.md)\n")
	old := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)
	if err := os.Chtimes(page, old, old); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run("lint")
	if code != 0 || !strings.Contains(out, "  a.md:untouched: unchanged for more than 1 days\n") ||
		!strings.Contains(out, "  a.md:long-planned: planned and untouched since 2026-01-01\n") ||
		strings.Contains(out, "missing-type") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestLintAsksTheSignpostToNameEveryArea(t *testing.T) {
	base := t.TempDir()
	_, post := checkedArea(t, base, "knowledge", "signpost = true\n")
	_, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, post, a)
	writeFile(t, filepath.Join(base, "knowledge", ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[layout]\nhub = \"Hub\"\n")
	code, out, _ := run("lint", "--scope", "knowledge")
	want := "  index.md:unlisted-area: the signpost does not link to area 'project/a'; expected one of: " +
		filepath.Join(base, "project-a", "wiki") + ", " + filepath.Join(base, "knowledge", "Hub", "a.md") + "\n"
	if code != 1 || !strings.Contains(out, want) {
		t.Fatalf("code %d, out %q", code, out)
	}
	// The hub page names the area as well as its wiki would.
	writeFile(t, filepath.Join(base, "knowledge", "wiki", "index.md"),
		"# c\n\n* [a](a.md)\n* [b](b.md)\n* [hub](../Hub/a.md)\n")
	if code, out, _ := run("lint", "--scope", "knowledge"); code != 0 || strings.Contains(out, "unlisted-area") {
		t.Fatalf("named by its hub page: code %d, out %q", code, out)
	}
}

func TestLintRefusesAHubItCannotUse(t *testing.T) {
	base := t.TempDir()
	_, post := checkedArea(t, base, "knowledge", "signpost = true\n")
	checkWorld(t, post)
	writeFile(t, filepath.Join(base, "knowledge", ".loomux", "config.toml"), "[area]\nscope = \"knowledge\"\n\n[layout]\nhub = \"../out\"\n")
	if code, out, errOut := run("lint"); code != 1 || out != "" || !strings.HasPrefix(errOut, "error: ") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}

func TestLintKeepsASharedAreaFromCitingAProject(t *testing.T) {
	base := t.TempDir()
	wiki, shared := checkedArea(t, base, "engineering/x", "shared = true\n")
	_, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, shared, a)
	writeFile(t, filepath.Join(wiki, "a.md"), strings.Replace(checkedPage("a"), "brain://x/y", "brain://project/a/topics/y", 1)+"\n[b](b.md)\n")
	code, out, _ := run("lint", "--scope", "engineering/x")
	if code != 1 || !strings.Contains(out, "  a.md:wrong-direction: a shared area must not cite 'project/a/topics/y'; "+
		"promote the page instead (architecture 5.7.4)\n") {
		t.Fatalf("code %d, out %q", code, out)
	}
}

func TestLintTellsASinglePageFromTheSweep(t *testing.T) {
	base := t.TempDir()
	wiki, a := checkedArea(t, base, "project/a", "")
	checkWorld(t, a)
	// A name ending in `.md` is a page even when it does not exist, and
	// `--file` names one too; either way no sweep line is printed.
	for _, args := range [][]string{
		{"lint", "--root", base, filepath.Join(wiki, "a.md")},
		{"lint", "--root", base, "--file", filepath.Join(wiki, "a.md")},
	} {
		if code, out, errOut := run(args...); code != 0 || out != "" || errOut != "" {
			t.Errorf("%v: code %d, out %q, err %q", args, code, out, errOut)
		}
	}
	if code, _, errOut := run("lint", "a.md", "b.md"); code != 2 || !strings.Contains(errOut, "unrecognized arguments: b.md") {
		t.Fatalf("two pages: code %d, err %q", code, errOut)
	}
}
