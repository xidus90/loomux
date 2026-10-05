package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/brain/search"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
)

// areaWorld is one machine for `area add`: a state directory and qmd's
// configuration all under one temporary directory, a
// fake search engine, and an empty repository beside them. The working
// directory is a temporary one too, twice over -- the process's own and the
// getwd seam -- so a run that forgot `--path` onboards a scratch directory
// and never the checkout the tests run in.
func areaWorld(t *testing.T) (state, repo string, port *search.FakePort) {
	t.Helper()
	tmp := t.TempDir()
	state = filepath.Join(tmp, "state")
	repo = filepath.Join(tmp, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOOMUX_STATE_DIR", state)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	scratch := filepath.Join(tmp, "cwd")
	if err := os.MkdirAll(scratch, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(scratch)
	stubGetwd(t, scratch, nil)
	port = search.NewFakePort()
	stubBrainStatusPort(t, port)
	return state, repo, port
}

func readText(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// wantRoutingRule is the paragraph of init.py:22-30, spelled out here a second
// time on purpose: a test that compared the code with its own constant would
// pass on any wording.
const wantRoutingRule = "## Wohin welches Wissen gehört\n" +
	"\n" +
	"Wissen, das nur für dieses Projekt gilt — Architektur, Entscheidungen,\n" +
	"Messungen, Betriebswissen dieses Repos — kommt nach `docs/wiki/`. Wissen, das\n" +
	"ein zweites Projekt genauso brauchen könnte — Werkzeuge, Sprachen, Verfahren,\n" +
	"Fremdprodukte — kommt in einen geteilten Bereich (`engineering/*`,\n" +
	"`knowledge`). Im Zweifel: geteilt, und aus dem Projekt per Verweis darauf\n" +
	"zeigen.\n"

func TestAreaAddRegistersANewArea(t *testing.T) {
	state, repo, port := areaWorld(t)
	if err := os.MkdirAll(filepath.Join(repo, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "docs", "note.md"), "---\ntitle: Note\n---\n# Note\n")

	code, out, errOut := run("area", "add", "--path", repo, "-y")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}

	areas, err := config.ReadRegistry(state)
	if err != nil {
		t.Fatal(err)
	}
	want := config.Area{
		Scope:     "project/repo",
		Path:      filepath.ToSlash(repo),
		WikiPath:  filepath.ToSlash(filepath.Join(repo, "docs", "wiki")),
		Workspace: true,
	}
	if len(areas) != 1 || areas[0] != want {
		t.Fatalf("registry = %+v, want %+v", areas, want)
	}

	manifest := filepath.Join(repo, ".loomux", "config.toml")
	wantManifest := "[area]\n" +
		"scope = \"project/repo\"\n" +
		"wiki = true\n" +
		"\n" +
		"[layout]\n" +
		"sources = \"docs\"\n" +
		"wiki = \"docs/wiki\"\n" +
		"\n" +
		"[index]\n" +
		"include = [\"docs/**/*.md\", \"README*.md\"]\n" +
		"\n" +
		"[privacy]\n" +
		"mode = \"manual_cloud\"\n" +
		"\n" +
		"[maintenance]\n" +
		"on_merge = true\n" +
		"branch = \"master\"\n"
	if got := readText(t, manifest); got != wantManifest {
		t.Fatalf("manifest =\n%s\nwant\n%s", got, wantManifest)
	}
	declared, err := config.ReadDeclaration(manifest)
	if err != nil {
		t.Fatalf("the written manifest does not read: %v", err)
	}
	if declared.Scope != "project/repo" || declared.LayoutWiki != "docs/wiki" {
		t.Fatalf("declaration = %+v", declared)
	}

	if !exists(filepath.Join(repo, "docs", "wiki", "_schema.md")) {
		t.Fatal("the wiki bundle was not scaffolded")
	}
	if got := readText(t, filepath.Join(repo, "AGENTS.md")); got != wantRoutingRule {
		t.Fatalf("AGENTS.md = %q", got)
	}

	// The index run went through: the register of a writable area lies in
	// the area itself, and the engine was told about its collection.
	if !exists(filepath.Join(repo, "_identities.tsv")) {
		t.Fatal("the index run wrote no register")
	}
	if len(port.Refreshed) != 1 {
		t.Fatalf("refreshed = %v", port.Refreshed)
	}
	for _, line := range []string{"registry: area \"project/repo\" added", "manifest: ", "routing rule: ", "wiki: scaffolded at ", "indexed"} {
		if !strings.Contains(out, line) {
			t.Fatalf("stdout lacks %q: %q", line, out)
		}
	}
	// The catch-up ran ahead of the index run: no area declares a review
	// centre in this world, and that is the warning it gives.
	if !strings.Contains(errOut, "Prüfzentrum") {
		t.Fatalf("stderr shows no catch-up: %q", errOut)
	}
}

// A duplicate is refused before the repository is touched: the registry
// decides first, so a refused scope leaves no configuration and no rule
// behind.
func TestAreaAddRefusesADuplicateScope(t *testing.T) {
	state, repo, port := areaWorld(t)
	if err := config.AddArea(state, config.Area{Scope: "project/repo", Path: "C:/elsewhere"}); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := run("area", "add", "--path", repo, "--yes")
	if code == 0 {
		t.Fatalf("a duplicate scope was accepted: %s", out)
	}
	if !strings.Contains(errOut, `"project/repo"`) {
		t.Fatalf("stderr does not name the scope: %q", errOut)
	}
	areas, err := config.ReadRegistry(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 || areas[0].Path != "C:/elsewhere" {
		t.Fatalf("registry = %+v", areas)
	}
	for _, name := range []string{".loomux", "AGENTS.md", "wiki"} {
		if exists(filepath.Join(repo, name)) {
			t.Fatalf("%s was written for a refused scope", name)
		}
	}
	if len(port.Refreshed) != 0 {
		t.Fatalf("a refused run indexed: %v", port.Refreshed)
	}
}

func TestAreaAddSkipsReindexOnRequest(t *testing.T) {
	state, repo, port := areaWorld(t)
	code, out, errOut := run("area", "add", "--path", repo, "-y", "--no-reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if exists(filepath.Join(state, "areas")) {
		t.Fatal("a register was written under <state>/areas")
	}
	if exists(filepath.Join(repo, "_identities.tsv")) {
		t.Fatal("a register was written into the area")
	}
	if len(port.Refreshed) != 0 {
		t.Fatalf("the engine was asked: %v", port.Refreshed)
	}
	if strings.Contains(out, "indexed") || strings.Contains(errOut, "Prüfzentrum") {
		t.Fatalf("an index run happened: %q / %q", out, errOut)
	}
}

// The brief asked for a refusal without --yes and without a terminal. The
// reference has none: `-y` is parsed and never read (cli.py:814-855), and
// nothing in `run_init` reads stdin. What the refusal was meant to prevent --
// a hook hanging on an answer that never comes -- cannot happen when nothing
// is ever asked, and that is what this test holds: no --yes, a closed stdin,
// and the command runs to its end.
func TestAreaAddNeverWaitsForAnAnswer(t *testing.T) {
	state, repo, _ := areaWorld(t)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	writer.Close()
	defer reader.Close()
	var out, errb bytes.Buffer
	code := Run([]string{"area", "add", "--path", repo, "--no-reindex"}, reader, &out, &errb)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errb.String())
	}
	if areas, err := config.ReadRegistry(state); err != nil || len(areas) != 1 {
		t.Fatalf("registry = %+v, %v", areas, err)
	}
}

func TestAreaAddWritesTheRoutingRuleInGerman(t *testing.T) {
	_, repo, _ := areaWorld(t)
	agents := writeFile(t, filepath.Join(repo, "AGENTS.md"), "# Project\n\nOur rules.")
	code, _, errOut := run("area", "add", "--path", repo, "-y", "--no-reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	want := "# Project\n\nOur rules.\n\n" + wantRoutingRule
	if got := readText(t, agents); got != want {
		t.Fatalf("AGENTS.md =\n%q\nwant\n%q", got, want)
	}

	// A second run finds the heading and leaves the file as it is: a second
	// copy would read as a second, competing rule.
	code, out, errOut := run("area", "add", "--path", repo, "--scope", "project/again", "-y", "--no-reindex")
	if code != 0 {
		t.Fatalf("second run: exit = %d, stderr = %s", code, errOut)
	}
	if got := readText(t, agents); got != want {
		t.Fatalf("the second run changed AGENTS.md:\n%q", got)
	}
	if strings.Contains(out, "routing rule:") {
		t.Fatalf("stdout reports a rule that was not written: %q", out)
	}
}

// The blank line before the heading is one, however many line feeds the
// file ended with; a CRLF file keeps the reference's bytes, its last carriage
// return included.
func TestAreaAddSeparatesTheRoutingRuleByOneBlankLine(t *testing.T) {
	cases := map[string]struct{ standing, want string }{
		"one line feed":   {"# Project\n", "# Project\n\n"},
		"two line feeds":  {"# Project\n\n", "# Project\n\n"},
		"many line feeds": {"# Project\n\n\n\n", "# Project\n\n"},
		"CRLF":            {"# Project\r\n", "# Project\r\n\n"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, repo, _ := areaWorld(t)
			agents := writeFile(t, filepath.Join(repo, "AGENTS.md"), c.standing)
			if code, _, errOut := run("area", "add", "--path", repo, "-y", "--no-reindex"); code != 0 {
				t.Fatalf("exit = %d, stderr = %s", code, errOut)
			}
			if got := readText(t, agents); got != c.want+wantRoutingRule {
				t.Fatalf("AGENTS.md = %q", got)
			}
		})
	}
}

// An existing configuration is the repository's policy and check chain as
// well as its declaration, and `area add` leaves it byte for byte -- the
// reference's `write_manifest` returns as soon as the file is there. The
// cost is said out loud: a file without [area] declares no area.
func TestAreaAddKeepsAnExistingConfiguration(t *testing.T) {
	state, repo, _ := areaWorld(t)
	standing := "# our policy\n[[policy.paths.rules]]\nmatch = [\"bin/*\"]\nreason = \"Build output.\"\n\n" +
		"[verify.go.lint]\ncommands = [\"go vet ./...\"]\n"
	manifest := writeFile(t, filepath.Join(repo, ".loomux", "config.toml"), standing)
	// With the index run: a policy-only file is no declaration, and the
	// passes read it as "no area here" rather than stopping at it.
	code, out, errOut := run("area", "add", "--path", repo, "-y")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if got := readText(t, manifest); got != standing {
		t.Fatalf("the configuration was changed:\n%s", got)
	}
	if !strings.Contains(out, "kept as it stood") {
		t.Fatalf("stdout does not say the configuration was kept: %q", out)
	}
	if !strings.Contains(errOut, "declares no [area]") {
		t.Fatalf("stderr does not warn about the missing [area]: %q", errOut)
	}
	if areas, err := config.ReadRegistry(state); err != nil || len(areas) != 1 {
		t.Fatalf("registry = %+v, %v", areas, err)
	}
}

func TestAreaAddWarnsWhenTheKeptConfigurationNamesAnotherScope(t *testing.T) {
	_, repo, _ := areaWorld(t)
	writeFile(t, filepath.Join(repo, ".loomux", "config.toml"), "[area]\nscope = \"project/old\"\n")
	code, _, errOut := run("area", "add", "--path", repo, "-y", "--no-reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, `"project/old"`) || !strings.Contains(errOut, `"project/repo"`) {
		t.Fatalf("stderr does not name both scopes: %q", errOut)
	}
}

// A kept configuration the declaration reader refuses is refused before the
// first write. Registered, it would stop the reconcile pass ahead of every
// index run on the machine -- for every area, not just this one -- and a
// second `area add` to repair it would be refused as a duplicate. The run is
// made *with* its index run, because that is where the damage showed.
func TestAreaAddRefusesAKeptConfigurationThatDoesNotRead(t *testing.T) {
	cases := map[string]string{
		"broken TOML":               "[area\n",
		"an [area] no reader takes": "[area]\nscope = 7\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			state, repo, port := areaWorld(t)
			manifest := writeFile(t, filepath.Join(repo, ".loomux", "config.toml"), body)
			code, out, errOut := run("area", "add", "--path", repo, "-y")
			if code != 1 {
				t.Fatalf("exit = %d, stderr = %s", code, errOut)
			}
			if !strings.Contains(errOut, "config.toml") || !strings.Contains(errOut, "not registered") {
				t.Fatalf("stderr does not name the file and the consequence: %q", errOut)
			}
			if out != "" {
				t.Fatalf("stdout reports work: %q", out)
			}
			if exists(filepath.Join(state, "registry.toml")) {
				t.Fatal("the area was registered")
			}
			for _, name := range []string{"AGENTS.md", "wiki"} {
				if exists(filepath.Join(repo, name)) {
					t.Fatalf("%s was written", name)
				}
			}
			if got := readText(t, manifest); got != body {
				t.Fatalf("the configuration was changed: %q", got)
			}
			if len(port.Refreshed) != 0 {
				t.Fatalf("a refused run indexed: %v", port.Refreshed)
			}
		})
	}
}

// A kept configuration that declares this very area is the quiet case.
func TestAreaAddSaysNothingWhenTheKeptConfigurationDeclaresTheArea(t *testing.T) {
	_, repo, _ := areaWorld(t)
	writeFile(t, filepath.Join(repo, ".loomux", "config.toml"), "[area]\nscope = \"project/repo\"\n")
	code, _, errOut := run("area", "add", "--path", repo, "-y")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if strings.Contains(errOut, "config.toml") {
		t.Fatalf("stderr warns about a configuration that fits: %q", errOut)
	}
}

// Every flag lands where the reference puts it. `--wiki` moves the bundle
// and the registry entry, but not [layout] wiki: the manifest records the
// place detection proposes (cli.py:835-838).
func TestAreaAddTakesItsFlags(t *testing.T) {
	state, repo, _ := areaWorld(t)
	wiki := filepath.Join(filepath.Dir(repo), "bundle")
	code, _, errOut := run("area", "add", "--path", repo, "--scope", "engineering/tools",
		"--wiki", wiki, "--sources", "src", "--merge-branch", "trunk", "--privacy", "local_only",
		"-y", "--no-reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	wantManifest := "[area]\n" +
		"scope = \"engineering/tools\"\n" +
		"wiki = true\n" +
		"\n" +
		"[layout]\n" +
		"sources = \"src\"\n" +
		"wiki = \"wiki\"\n" +
		"\n" +
		"[index]\n" +
		"include = [\"**/*.md\"]\n" +
		"\n" +
		"[privacy]\n" +
		"mode = \"local_only\"\n" +
		"\n" +
		"[maintenance]\n" +
		"on_merge = true\n" +
		"branch = \"trunk\"\n"
	if got := readText(t, filepath.Join(repo, ".loomux", "config.toml")); got != wantManifest {
		t.Fatalf("manifest =\n%s", got)
	}
	areas, err := config.ReadRegistry(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 || areas[0].Scope != "engineering/tools" || areas[0].WikiPath != filepath.ToSlash(wiki) {
		t.Fatalf("registry = %+v", areas)
	}
	if !exists(filepath.Join(wiki, "index.md")) {
		t.Fatal("the bundle is not where --wiki put it")
	}
}

// A file named docs is no source directory: detection asks for a directory,
// as `is_dir()` does, and the bundle then sits at wiki/.
func TestAreaAddTakesAFileNamedDocsForNoSourceDirectory(t *testing.T) {
	state, repo, _ := areaWorld(t)
	writeFile(t, filepath.Join(repo, "docs"), "not a directory")
	if code, _, errOut := run("area", "add", "--path", repo, "-y", "--no-reindex"); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	areas, err := config.ReadRegistry(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 1 || areas[0].WikiPath != filepath.ToSlash(filepath.Join(repo, "wiki")) {
		t.Fatalf("registry = %+v", areas)
	}
}

// A value from the command line that would reach TOML is quoted, not pasted.
func TestAreaAddQuotesWhatItWritesIntoTOML(t *testing.T) {
	_, repo, _ := areaWorld(t)
	code, _, errOut := run("area", "add", "--path", repo, "--scope", `project/"quoted"`,
		"--merge-branch", "a\\b", "-y", "--no-reindex")
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	declared, err := config.ReadDeclaration(filepath.Join(repo, ".loomux", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if declared.Scope != `project/"quoted"` {
		t.Fatalf("scope = %q", declared.Scope)
	}
}

// Without --path the working directory is the repository, as in the
// reference; a relative --path is read against it.
func TestAreaAddDefaultsToTheWorkingDirectory(t *testing.T) {
	state, repo, _ := areaWorld(t)
	stubGetwd(t, repo, nil)
	if code, _, errOut := run("area", "add", "-y", "--no-reindex"); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	if err := os.MkdirAll(filepath.Join(repo, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run("area", "add", "--path", "sub", "-y", "--no-reindex"); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut)
	}
	areas, err := config.ReadRegistry(state)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != 2 || areas[0].Path != filepath.ToSlash(repo) || areas[1].Path != filepath.ToSlash(filepath.Join(repo, "sub")) {
		t.Fatalf("registry = %+v", areas)
	}
}

func TestAreaAddIsRedWhenTheWorkingDirectoryIsGone(t *testing.T) {
	areaWorld(t)
	stubGetwd(t, "", errors.New("the directory is gone"))
	code, _, errOut := run("area", "add", "-y")
	if code != 1 || !strings.Contains(errOut, "the directory is gone") {
		t.Fatalf("exit = %d, stderr = %q", code, errOut)
	}
}

// git answers the branch; no repository, an unborn branch and a detached
// HEAD all fall back to master, as `_detect_branch` does.
func TestAreaAddReadsTheMergeBranchFromGit(t *testing.T) {
	cases := map[string]struct {
		setup [][]string
		want  string
	}{
		"a branch":        {[][]string{{"checkout", "-q", "-b", "trunk"}}, "trunk"},
		"a detached HEAD": {[][]string{{"checkout", "-q", "--detach"}}, "master"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, repo, _ := areaWorld(t)
			gitInit(t, repo)
			for _, args := range c.setup {
				gitIn(t, repo, args...)
			}
			if code, _, errOut := run("area", "add", "--path", repo, "-y", "--no-reindex"); code != 0 {
				t.Fatalf("exit = %d, stderr = %s", code, errOut)
			}
			manifest := readText(t, filepath.Join(repo, ".loomux", "config.toml"))
			if !strings.Contains(manifest, "branch = \""+c.want+"\"\n") {
				t.Fatalf("manifest =\n%s", manifest)
			}
		})
	}
	t.Run("an unborn branch", func(t *testing.T) {
		_, repo, _ := areaWorld(t)
		gitIn(t, repo, "init", "-q", "-b", "trunk")
		if code, _, errOut := run("area", "add", "--path", repo, "-y", "--no-reindex"); code != 0 {
			t.Fatalf("exit = %d, stderr = %s", code, errOut)
		}
		manifest := readText(t, filepath.Join(repo, ".loomux", "config.toml"))
		if !strings.Contains(manifest, "branch = \"master\"\n") {
			t.Fatalf("manifest =\n%s", manifest)
		}
	})
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = gitenv.Environ()
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// Each refusal comes before the first write: the registry stays empty and the
// repository untouched.
func TestAreaAddRefusesBeforeWriting(t *testing.T) {
	cases := map[string]struct {
		prepare func(t *testing.T, repo string) []string
		code    int
		says    string
	}{
		"an unknown privacy mode": {
			func(t *testing.T, repo string) []string {
				return []string{"--path", repo, "--privacy", "cloudy"}
			}, 2, "--privacy must be one of automatic_cloud, local_only, manual_cloud",
		},
		"an empty scope segment": {
			func(t *testing.T, repo string) []string {
				return []string{"--path", repo, "--scope", "project/ "}
			}, 1, "scope must name every segment",
		},
		"a relative wiki": {
			func(t *testing.T, repo string) []string {
				return []string{"--path", repo, "--wiki", "docs/wiki"}
			}, 1, "wiki must be absolute",
		},
		"a path that is no directory": {
			func(t *testing.T, repo string) []string {
				return []string{"--path", writeFile(t, filepath.Join(repo, "file.txt"), "x")}
			}, 1, "is not a directory",
		},
		// Not created: a mistyped --path would otherwise become a new
		// directory and a registered area nobody meant.
		"a path that does not exist": {
			func(t *testing.T, repo string) []string {
				return []string{"--path", filepath.Join(repo, "missing")}
			}, 1, "is not a directory",
		},
		"a configuration that cannot be read": {
			func(t *testing.T, repo string) []string {
				if err := os.MkdirAll(filepath.Join(repo, ".loomux", "config.toml"), 0o755); err != nil {
					t.Fatal(err)
				}
				return []string{"--path", repo}
			}, 1, "config.toml",
		},
		"a project instruction that cannot be read": {
			func(t *testing.T, repo string) []string {
				if err := os.MkdirAll(filepath.Join(repo, "AGENTS.md"), 0o755); err != nil {
					t.Fatal(err)
				}
				return []string{"--path", repo}
			}, 1, "AGENTS.md",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			state, repo, _ := areaWorld(t)
			args := append([]string{"area", "add", "-y"}, c.prepare(t, repo)...)
			code, out, errOut := run(args...)
			if code != c.code {
				t.Fatalf("exit = %d, want %d; stderr = %s", code, c.code, errOut)
			}
			if !strings.Contains(errOut, c.says) {
				t.Fatalf("stderr does not say %q: %q", c.says, errOut)
			}
			if out != "" {
				t.Fatalf("stdout reports work: %q", out)
			}
			if exists(filepath.Join(state, "registry.toml")) {
				t.Fatal("the registry was written")
			}
			if exists(filepath.Join(repo, "wiki")) {
				t.Fatal("the bundle was scaffolded")
			}
		})
	}
}

// A write that fails after registration is red and says which file; the
// area stays registered, because taking it back would be a second write that
// can fail the same way.
func TestAreaAddIsRedWhenARepositoryWriteFails(t *testing.T) {
	cases := map[string]struct {
		blocked string
		says    string
	}{
		"the configuration directory is a file": {".loomux", ".loomux"},
		"the bundle is a file":                  {"wiki", "wiki"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			state, repo, port := areaWorld(t)
			writeFile(t, filepath.Join(repo, c.blocked), "in the way")
			code, _, errOut := run("area", "add", "--path", repo, "-y")
			if code != 1 {
				t.Fatalf("exit = %d, stderr = %s", code, errOut)
			}
			if !strings.Contains(errOut, c.says) {
				t.Fatalf("stderr does not name %s: %q", c.says, errOut)
			}
			if areas, err := config.ReadRegistry(state); err != nil || len(areas) != 1 {
				t.Fatalf("registry = %+v, %v", areas, err)
			}
			if len(port.Refreshed) != 0 {
				t.Fatalf("a failed run indexed: %v", port.Refreshed)
			}
		})
	}
}

// The index run's exit is the command's exit: an engine that refused leaves
// `area add` red, although everything before it landed.
func TestAreaAddHandsOnTheExitOfTheIndexRun(t *testing.T) {
	_, repo, port := areaWorld(t)
	port.Refreshes = []error{errors.New("engine is asleep")}
	if code, _, _ := run("area", "add", "--path", repo, "-y"); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestAreaAddRefusesAPositionalArgument(t *testing.T) {
	state, _, _ := areaWorld(t)
	code, out, errOut := run("area", "add", "somewhere")
	if code != 2 {
		t.Fatalf("exit = %d, want 2; stderr = %s", code, errOut)
	}
	if !strings.Contains(errOut, "loomux area add: unrecognized arguments: somewhere") {
		t.Fatalf("stderr does not name the argument: %q", errOut)
	}
	if out != "" || exists(filepath.Join(state, "registry.toml")) {
		t.Fatalf("a refused run did work: %q", out)
	}
}

func TestAreaAddRefusesTheStateDirFlag(t *testing.T) {
	areaWorld(t)
	if code, _, _ := run("area", "add", "--state-dir", "somewhere"); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestAreaWithoutAKnownSubcommandIsAUsageError(t *testing.T) {
	areaWorld(t)
	for _, args := range [][]string{{"area"}, {"area", "remove"}} {
		code, _, errOut := run(args...)
		if code != 2 {
			t.Fatalf("%v: exit = %d, want 2", args, code)
		}
		if !strings.Contains(errOut, "loomux area add") {
			t.Fatalf("%v: stderr does not show the usage: %q", args, errOut)
		}
	}
}

func TestAreaCheckIsGone(t *testing.T) {
	code, out, errOut := run("area", "check", t.TempDir())
	if code != 2 || out != "" || !strings.HasPrefix(errOut, "usage: loomux area add") || strings.Contains(errOut, "check") {
		t.Fatalf("code %d, out %q, err %q", code, out, errOut)
	}
}
