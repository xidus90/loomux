package verify

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/extract/html"
	"github.com/xidus90/loomux/internal/code/extract/stylesheet"
	"github.com/xidus90/loomux/internal/detect"
)

var (
	htmlValidatePinned = []string{"npx", "--yes", "html-validate@11.21.1"}
	htmlhintPinned     = []string{"npx", "--yes", "htmlhint@1.9.2"}
	stylelintPinned    = []string{"npx", "--yes", "-p", "stylelint@18.0.0", "-p", "stylelint-config-standard-scss@18.0.0", "stylelint"}
)

// webProject is a root with one page and one stylesheet, the state directory
// in a temporary folder of its own, and the real file lookups.
func webProject(t *testing.T) (string, PlanEnv) {
	t.Helper()
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	root := t.TempDir()
	put(t, root, "index.html", "")
	put(t, root, "a.css", "")
	e := env(root)
	e.HasTests = HasTests
	return root, e
}

func lintArgvs(t *testing.T, facts detect.Facts, req Request, e PlanEnv) Job {
	t.Helper()
	jobs, err := Plan(effFor(t, "", facts), req, e)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("%v %s", err, names(jobs))
	}
	return jobs[0]
}

var checkLint = Request{Kinds: []string{"lint"}}

// The project's own tool comes first, from the area's node_modules or one
// above it up to the root; without one the pinned version runs through npx.
func TestHTMLLintRunsTheProjectsToolElseThePinnedOne(t *testing.T) {
	root, e := webProject(t)
	tail := []string{"--preset", "standard", "index.html"}
	job := lintArgvs(t, factsOf("html"), checkLint, e)
	if job.Pre != "" || !slices.EqualFunc(job.Argvs, [][]string{slices.Concat(htmlValidatePinned, tail)}, slices.Equal) {
		t.Fatalf("pinned: %q %+v", job.Argvs, job)
	}
	put(t, root, "node_modules/.bin/html-validate", "")
	job = lintArgvs(t, factsOf("html"), checkLint, e)
	if !slices.EqualFunc(job.Argvs, [][]string{slices.Concat([]string{"npx", "html-validate"}, tail)}, slices.Equal) {
		t.Fatalf("installed: %q", job.Argvs)
	}
	member := detect.Facts{Stacks: []string{"html"}, Areas: map[string][]string{"html": {"site"}}}
	put(t, root, "site/index.html", "")
	job = lintArgvs(t, member, checkLint, e)
	if !slices.EqualFunc(job.Argvs, [][]string{slices.Concat([]string{"npx", "html-validate"}, tail)}, slices.Equal) {
		t.Fatalf("hoisted: %q", job.Argvs)
	}
}

// .htmlhintrc keeps htmlhint as the area's linter, by the same rule.
func TestHTMLHintVariantRunsTheProjectsToolElseThePinnedOne(t *testing.T) {
	root, e := webProject(t)
	hint := detect.Facts{Stacks: []string{"html", "htmlhint"}, Areas: map[string][]string{"html": {"."}}}
	job := lintArgvs(t, hint, checkLint, e)
	if job.Pre != "" || !slices.EqualFunc(job.Argvs, [][]string{append(slices.Clone(htmlhintPinned), "index.html")}, slices.Equal) {
		t.Fatalf("pinned: %q %+v", job.Argvs, job)
	}
	put(t, root, "node_modules/.bin/htmlhint", "")
	job = lintArgvs(t, hint, checkLint, e)
	if !slices.EqualFunc(job.Argvs, [][]string{{"npx", "htmlhint", "index.html"}}, slices.Equal) {
		t.Fatalf("installed: %q", job.Argvs)
	}
	edit := Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "index.html"}
	job = lintArgvs(t, hint, edit, e)
	if !slices.EqualFunc(job.Argvs, [][]string{{"npx", "htmlhint", "index.html"}}, slices.Equal) {
		t.Fatalf("edit: %q", job.Argvs)
	}
}

// html-validate has no switch to leave a directory out, so it is named the
// pages: the area's .html and .htm files outside every directory a walk
// prunes, and none at all leaves nothing to lint.
func TestHTMLLintNamesTheAreasPages(t *testing.T) {
	root, e := webProject(t)
	for _, f := range []string{"sub/a.htm", "node_modules/x/b.html", "coverage/lcov-report/index.html", ".cache/c.html", "sub/d.txt"} {
		put(t, root, f, "")
	}
	job := lintArgvs(t, factsOf("html"), checkLint, e)
	if got := job.Argvs[0][len(htmlValidatePinned)+2:]; !slices.Equal(got, []string{"index.html", "sub/a.htm"}) {
		t.Fatalf("pages: %q", got)
	}
	bare := t.TempDir()
	put(t, bare, "a.css", "")
	e.Root = bare
	job = lintArgvs(t, factsOf("html"), checkLint, e)
	if job.Pre != StateUnavailable || job.Note != "none found: *.html, *.htm" || Red(job.Pre, ScopeCheck) {
		t.Fatalf("no pages: %+v", job)
	}
	// An edit names its file and lists nothing.
	job = lintArgvs(t, factsOf("html"), Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "x.html"}, e)
	if job.Pre != "" || !slices.EqualFunc(job.Argvs, [][]string{slices.Concat(htmlValidatePinned, []string{"--preset", "standard", "x.html"})}, slices.Equal) {
		t.Fatalf("edit: %q %+v", job.Argvs, job)
	}
}

// Many pages are split across several commands of the lane, each under the
// 8191 characters cmd.exe takes, which runs npx.cmd on Windows, every page
// named exactly once.
func TestHTMLLintSplitsALongListOfPages(t *testing.T) {
	if maxCommandLine != 8000 {
		t.Fatalf("maxCommandLine = %d, want 8000", maxCommandLine)
	}
	root, e := webProject(t)
	os.Remove(filepath.Join(root, "index.html"))
	var want []string
	for i := range 3000 {
		f := fmt.Sprintf("pages/%028d.html", i)
		want = append(want, f)
		put(t, root, f, "")
	}
	job := lintArgvs(t, factsOf("html"), checkLint, e)
	prefix := slices.Concat(htmlValidatePinned, []string{"--preset", "standard"})
	if len(job.Argvs) < 2 {
		t.Fatalf("%d commands", len(job.Argvs))
	}
	var got []string
	for _, argv := range job.Argvs {
		if !slices.Equal(argv[:len(prefix)], prefix) || len(strings.Join(argv, " ")) > maxCommandLine {
			t.Fatalf("argv of %d characters: %q", len(strings.Join(argv, " ")), argv[:len(prefix)])
		}
		got = append(got, argv[len(prefix):]...)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("%d pages, want %d", len(got), len(want))
	}
}

func TestBatchArgvKeepsWhatFollowsTheFiles(t *testing.T) {
	got := batchArgv([]string{"lint", "--x", "--end"}, 2, []string{"aaaa", "bbbb", "c"}, 21)
	want := [][]string{{"lint", "--x", "aaaa", "--end"}, {"lint", "--x", "bbbb", "c", "--end"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("%q", got)
	}
	// A file longer than the limit still gets a command of its own.
	got = batchArgv([]string{"lint"}, 1, []string{"a", strings.Repeat("b", 30), "c"}, 10)
	want = [][]string{{"lint", "a"}, {"lint", strings.Repeat("b", 30)}, {"lint", "c"}}
	if !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("%q", got)
	}
}

// stylelint gets what PHP_CodeSniffer gets: no list, the directories a walk
// prunes and minified stylesheets left out, and a glob of its own.
func TestCSSLintNamesTheGlobAndItsExclusions(t *testing.T) {
	_, e := webProject(t)
	job := lintArgvs(t, factsOf("css"), checkLint, e)
	argv := job.Argvs[0]
	rest := argv[len(stylelintPinned)+2:]
	if job.Pre != "" || rest[0] != "--allow-empty-input" || rest[len(rest)-1] != "**/*.{css,scss}" {
		t.Fatalf("%q", argv)
	}
	for _, p := range []string{"**/coverage/**", "**/htmlcov/**", "**/dist/**", "**/node_modules/**", "**/*.min.css"} {
		i := slices.Index(rest, p)
		if i < 1 || rest[i-1] != "--ignore-pattern" {
			t.Errorf("no --ignore-pattern %s: %q", p, rest)
		}
	}
}

// Without a configuration of the project stylelint reads loomux's own from
// the state directory, never from the work tree, and an installed stylelint
// is passed by: the default configuration needs the shared one npx installs
// beside the pinned version.
func TestCSSLintTakesTheDefaultConfigurationOnlyWithoutTheProjects(t *testing.T) {
	root, e := webProject(t)
	config := []string{"--config", StylelintConfig()}
	job := lintArgvs(t, factsOf("css"), checkLint, e)
	if !slices.Equal(job.Argvs[0][:len(stylelintPinned)+2], slices.Concat(stylelintPinned, config)) {
		t.Fatalf("default: %q", job.Argvs[0])
	}
	if rel, err := filepath.Rel(root, StylelintConfig()); err == nil && filepath.IsLocal(rel) {
		t.Fatalf("%s lies under the root", StylelintConfig())
	}
	if _, err := os.Stat(StylelintConfig()); err == nil {
		t.Fatal("Plan wrote the configuration")
	}
	put(t, root, "node_modules/.bin/stylelint", "")
	job = lintArgvs(t, factsOf("css"), checkLint, e)
	if !slices.Equal(job.Argvs[0][:len(stylelintPinned)+2], slices.Concat(stylelintPinned, config)) {
		t.Fatalf("installed without configuration: %q", job.Argvs[0])
	}
	put(t, root, ".stylelintrc.json", "{}")
	job = lintArgvs(t, factsOf("css"), checkLint, e)
	if !slices.Equal(job.Argvs[0][:3], []string{"npx", "stylelint", "--allow-empty-input"}) {
		t.Fatalf("installed with configuration: %q", job.Argvs[0])
	}
	os.RemoveAll(filepath.Join(root, "node_modules"))
	job = lintArgvs(t, factsOf("css"), checkLint, e)
	if !slices.Equal(job.Argvs[0][:len(stylelintPinned)+1], append(slices.Clone(stylelintPinned), "--allow-empty-input")) {
		t.Fatalf("pinned with configuration: %q", job.Argvs[0])
	}
}

// The project's configuration is found where stylelint looks for it: from
// the area upwards, or for an edit from the file's directory upwards, and in
// a package.json only under a key of its own.
func TestCSSLintFindsTheProjectsConfiguration(t *testing.T) {
	member := detect.Facts{Stacks: []string{"css"}, Areas: map[string][]string{"css": {"web"}}}
	hasConfig := func(t *testing.T, facts detect.Facts, req Request, e PlanEnv) bool {
		t.Helper()
		return slices.Contains(lintArgvs(t, facts, req, e).Argvs[0], "--config")
	}
	for _, c := range []struct {
		name, file, body string
		configured       bool
	}{
		{"one level up", ".stylelintrc.json", "{}", true},
		{"package.json key", "package.json", `{"stylelint": {"extends": "x"}}`, true},
		{"package.json null key", "package.json", `{"stylelint": null}`, false},
		{"package.json dependency", "package.json", `{"devDependencies": {"stylelint": "17.16.0"}}`, false},
		{"broken package.json", "package.json", `{"stylelint": `, false},
		{".config directory", ".config/stylelintrc.yml", "", true},
		{"typescript", "stylelint.config.ts", "", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			root, e := webProject(t)
			put(t, root, "web/a.css", "")
			put(t, root, c.file, c.body)
			if got := hasConfig(t, member, checkLint, e); got == c.configured {
				t.Errorf("--config %v", got)
			}
		})
	}
	root, e := webProject(t)
	put(t, root, "web/a.css", "")
	put(t, root, "web/.stylelintrc", "{}")
	edit := Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "web/a.css"}
	if job := lintArgvs(t, factsOf("css"), edit, e); !slices.Equal(job.Argvs[0], append(slices.Clone(stylelintPinned), "web/a.css")) {
		t.Errorf("edit beside its configuration: %q", job.Argvs[0])
	}
	// The whole area starts at the area: a configuration only below it is
	// not found.
	if !hasConfig(t, factsOf("css"), checkLint, e) {
		t.Error("check found the configuration below the area")
	}
}

// The default configuration is SCSS: a .less or .sass file without the
// project's configuration has nothing to be linted by.
func TestCSSLintOfALessFileNeedsTheProjectsConfiguration(t *testing.T) {
	root, e := webProject(t)
	for _, ext := range []string{".less", ".sass"} {
		edit := Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "a" + ext}
		job := lintArgvs(t, factsOf("css"), edit, e)
		if job.Pre != StateUnavailable || job.Note != "no stylelint configuration for "+ext || Red(job.Pre, ScopeEdit) {
			t.Errorf("%s: %+v", ext, job)
		}
	}
	put(t, root, ".stylelintrc", "{}")
	edit := Request{Kinds: []string{"lint"}, Scope: ScopeEdit, File: "a.less"}
	if job := lintArgvs(t, factsOf("css"), edit, e); job.Pre != "" || !slices.Equal(job.Argvs[0], append(slices.Clone(stylelintPinned), "a.less")) {
		t.Errorf("configured: %+v", job)
	}
	edit.File = "a.scss"
	os.Remove(filepath.Join(root, ".stylelintrc"))
	if job := lintArgvs(t, factsOf("css"), edit, e); job.Pre != "" || !slices.Equal(job.Argvs[0], slices.Concat(stylelintPinned, []string{"--config", StylelintConfig(), "a.scss"})) {
		t.Errorf("scss: %+v", job)
	}
}

// PrepareRun writes the default configuration into the state directory for
// a run whose lanes name it, again only where it is missing or changed, and
// nothing under the root but the coverage directory.
func TestPrepareRunWritesTheDefaultStylelintConfiguration(t *testing.T) {
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	root := t.TempDir()
	if err := PrepareRun(root, []Job{job("lint/go", -1, "go vet")}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(StylelintConfig()); err == nil {
		t.Fatal("wrote the configuration for a run that does not read it")
	}
	jobs := []Job{{Argvs: [][]string{{"npx", "stylelint", "--config", StylelintConfig()}}}}
	if err := PrepareRun(root, jobs); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(StylelintConfig())
	if err != nil || string(got) != stylelintConfig {
		t.Fatalf("%v %q", err, got)
	}
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	os.Chtimes(StylelintConfig(), old, old)
	if err := PrepareRun(root, jobs); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(StylelintConfig()); !info.ModTime().Equal(old) {
		t.Errorf("rewrote an unchanged file: %v", info.ModTime())
	}
	os.WriteFile(StylelintConfig(), []byte("export default {};\n"), 0o644)
	if err := PrepareRun(root, jobs); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(StylelintConfig()); string(got) != stylelintConfig {
		t.Errorf("kept a changed file: %q", got)
	}
	var under []string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		rel, _ := filepath.Rel(root, p)
		under = append(under, filepath.ToSlash(rel))
		return nil
	})
	if !slices.Equal(under, []string{".", ".loomux", ".loomux/state", ".loomux/state/cover"}) {
		t.Errorf("under the root: %q", under)
	}
}

func TestPrepareRunReportsAStateDirectoryItCannotWrite(t *testing.T) {
	state := t.TempDir()
	t.Setenv("LOOMUX_STATE_DIR", state)
	reads := func() []Job { return []Job{{Argvs: [][]string{{"--config", StylelintConfig()}}}} }
	os.WriteFile(filepath.Join(state, "verify"), nil, 0o644)
	if err := PrepareRun(t.TempDir(), reads()); err == nil {
		t.Error("no error for a state directory that is a file")
	}
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	os.MkdirAll(StylelintConfig(), 0o755)
	if err := PrepareRun(t.TempDir(), reads()); err == nil {
		t.Error("no error for a configuration that is a directory")
	}
	// A write that fails leaves nothing half-written beside the configuration.
	if entries, _ := os.ReadDir(filepath.Dir(StylelintConfig())); len(entries) != 1 {
		t.Errorf("left behind: %v", entries)
	}
}

// The configuration is written beside its place and renamed over the old
// one, which leaves no temporary file behind.
func TestPrepareRunRenamesTheConfigurationIntoPlace(t *testing.T) {
	t.Setenv("LOOMUX_STATE_DIR", t.TempDir())
	reads := []Job{{Argvs: [][]string{{"--config", StylelintConfig()}}}}
	os.MkdirAll(filepath.Dir(StylelintConfig()), 0o755)
	os.WriteFile(StylelintConfig(), []byte("old"), 0o644)
	if err := PrepareRun(t.TempDir(), reads); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(StylelintConfig()); string(got) != stylelintConfig {
		t.Errorf("%q", got)
	}
	if entries, _ := os.ReadDir(filepath.Dir(StylelintConfig())); len(entries) != 1 {
		t.Errorf("left behind: %v", entries)
	}
}

// The graph and the rules belong to the root: a project of pages and
// stylesheets alone carries them, css ahead of html, and any ranked stack
// ahead of both.
func TestWebStacksCarryTheRootLanes(t *testing.T) {
	covered := func(kind, c string) string { return kind + " covered by " + kind + "/" + c }
	for _, kind := range []string{"graph", "rules"} {
		for _, c := range []struct {
			stacks []string
			want   [][3]string
		}{
			{[]string{"html"}, [][3]string{{kind + "/html", "", ""}}},
			{[]string{"css"}, [][3]string{{kind + "/css", "", ""}}},
			{[]string{"css", "html"}, [][3]string{{kind + "/css", "", ""}, {kind + "/html", "not-applicable", covered(kind, "css")}}},
			{[]string{"css", "typescript"}, [][3]string{{kind + "/css", "not-applicable", covered(kind, "typescript")}, {kind + "/typescript", "", ""}}},
		} {
			jobs, err := Plan(effFor(t, withRules, factsOf(c.stacks...)), Request{Kinds: []string{kind}}, graphEnv(t.TempDir(), true, "", new(int)))
			if err != nil || len(jobs) != len(c.want) {
				t.Fatalf("%v %s", err, names(jobs))
			}
			for i, w := range c.want {
				j := jobs[i]
				if j.Name != w[0] || string(j.Pre) != w[1] || j.Note != w[2] || (w[1] == "") == (j.Argvs == nil) {
					t.Errorf("job %d = %s %q %q %v, want %q", i, j.Name, j.Pre, j.Note, j.Argvs, w)
				}
			}
		}
	}
}

// The rules read what the HTML and stylesheet extractors read.
func TestIsSourceCoversTheWebExtractors(t *testing.T) {
	for _, ext := range slices.Concat(html.Language{}.Extensions(), stylesheet.CSS{}.Extensions(), stylesheet.SCSS{}.Extensions()) {
		if !IsSource("a" + ext) {
			t.Errorf("%s is no source", ext)
		}
	}
}
