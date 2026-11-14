package web

import (
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/gitenv"
	"github.com/xidus90/loomux/internal/setup"
)

// wizardWorld is a signed-in interface whose setup runs against a fake that
// records the actions, in a project folder under a name that stresses every
// path handling (a space, a hash, an umlaut).
type wizardWorld struct {
	*ui
	cookie, token, root string
	mu                  sync.Mutex
	ran                 []string
	// onRun lets a test make an action do more than be recorded.
	onRun func(a setup.Action, scope string) error
}

// realGit runs git as setup.Gather asks: an exit 1 without output is an
// unset setting.
func realGit(dir string, argv ...string) (string, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = gitenv.Environ()
	out, err := cmd.Output()
	if exit, ok := err.(*exec.ExitError); ok && len(out) == 0 && exit.ExitCode() == 1 {
		return "", nil
	}
	return string(out), err
}

func newWizardWorld(t *testing.T) *wizardWorld {
	t.Helper()
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	// The installed binary the written entries call.
	if err := os.MkdirAll(filepath.Join(local, "loomux", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(local, "loomux", "bin", "loomux.exe"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	w := &wizardWorld{}
	home := t.TempDir()
	w.ui = startUIWith(t, func(o *Options) {
		o.Gather = func(root string) (setup.Facts, error) {
			return setup.Gather(root, home, setup.Running{Version: "9.9.9", VersionOf: func(string) string { return "" }}, realGit)
		}
		o.SetupDeps = func(root string, _ setup.Facts, scope string, stdout, _ io.Writer) setup.Deps {
			return setup.Deps{
				BinaryThere: func() bool { return true },
				Run: func(a setup.Action) error {
					w.mu.Lock()
					defer w.mu.Unlock()
					w.ran = append(w.ran, a.ID)
					io.WriteString(stdout, "ran "+a.ID+"\n")
					if w.onRun != nil {
						return w.onRun(a, scope)
					}
					return nil
				},
			}
		}
	})
	w.root = filepath.Join(t.TempDir(), "#GIT", "mein projekt ü")
	if err := os.MkdirAll(w.root, 0o755); err != nil {
		t.Fatal(err)
	}
	w.cookie, w.token = w.ui.signIn()
	return w
}

// gitRepo makes the project folder a repository with an agent folder.
func (w *wizardWorld) gitRepo() {
	w.t.Helper()
	initGit := exec.Command("git", "init", "-q", w.root)
	initGit.Env = gitenv.Environ()
	if out, err := initGit.CombinedOutput(); err != nil {
		w.t.Fatalf("git init: %v: %s", err, out)
	}
	w.put(".claude/keep", "")
}

// german asks for a commit language other than the default, the one thing
// that gives a fresh project's configuration something to say.
func german(f url.Values) { f.Set("lang", "de") }

// put writes a file of the project.
func (w *wizardWorld) put(rel, text string) {
	w.t.Helper()
	path := filepath.Join(w.root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		w.t.Fatal(err)
	}
}

// read is a file of the project, "" when it is not there.
func (w *wizardWorld) read(rel string) string {
	data, _ := os.ReadFile(filepath.Join(w.root, filepath.FromSlash(rel)))
	return string(data)
}

func (w *wizardWorld) post(path string, form url.Values) (int, string) {
	w.t.Helper()
	form.Set("csrf", w.token)
	resp := w.do(req{method: http.MethodPost, path: path, cookie: w.cookie, form: form})
	return resp.StatusCode, bodyOf(w.t, resp)
}

func (w *wizardWorld) actions() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.ran)
}

var (
	hiddenField = regexp.MustCompile(`<input type="hidden" name="([^"]*)" value="([^"]*)">`)
	tickedField = regexp.MustCompile(`<input type="checkbox" name="([^"]*)" value="([^"]*)" checked>`)
	textField   = regexp.MustCompile(`<input type="text" name="([^"]*)" value="([^"]*)"`)
	chosenLang  = regexp.MustCompile(`<select name="lang">.*?<option value="([^"]*)" selected>`)
)

// formOf is what a browser would send for the form of the page: the hidden
// fields, the ticked boxes and the chosen language.
func formOf(body string) url.Values {
	_, body, _ = strings.Cut(body, "<main>")
	form := url.Values{}
	for _, m := range hiddenField.FindAllStringSubmatch(body, -1) {
		form.Add(m[1], html.UnescapeString(m[2]))
	}
	for _, m := range tickedField.FindAllStringSubmatch(body, -1) {
		form.Add(m[1], html.UnescapeString(m[2]))
	}
	for _, m := range textField.FindAllStringSubmatch(body, -1) {
		form.Set(m[1], html.UnescapeString(m[2]))
	}
	if m := chosenLang.FindStringSubmatch(strings.ReplaceAll(body, "\n", " ")); m != nil {
		form.Set("lang", m[1])
	}
	return form
}

// facts runs step one and two and returns the answer of the facts step.
func (w *wizardWorld) facts() (int, string) {
	return w.post("/projects/new/facts", url.Values{"path": {w.root}})
}

// plan is the page the choice of the facts page leads to.
func (w *wizardWorld) plan(mutate func(url.Values)) (int, string) {
	w.t.Helper()
	status, body := w.facts()
	if status != http.StatusOK {
		w.t.Fatalf("facts = %d: %s", status, body)
	}
	form := formOf(body)
	if mutate != nil {
		mutate(form)
	}
	return w.post("/projects/new/plan", form)
}

// apply sends the form of a plan page, changed by mutate.
func (w *wizardWorld) apply(planPage string, mutate func(url.Values)) (int, string) {
	w.t.Helper()
	form := formOf(planPage)
	if mutate != nil {
		mutate(form)
	}
	return w.post("/projects/new/apply", form)
}

// setUp walks the wizard to the end with everything ticked.
func (w *wizardWorld) setUp() string {
	w.t.Helper()
	status, plan := w.plan(german)
	if status != http.StatusOK {
		w.t.Fatalf("plan = %d: %s", status, plan)
	}
	status, done := w.apply(plan, nil)
	if status != http.StatusOK || strings.Contains(done, `class="error"`) {
		w.t.Fatalf("apply = %d: %s", status, done)
	}
	return done
}

func TestTheWizardNeedsASession(t *testing.T) {
	u := startUI(t)
	resp := u.do(req{path: "/projects/new"})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("GET /projects/new = %d %s, want the login", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestTheFirstPageAsksForThePath(t *testing.T) {
	w := newWizardWorld(t)
	status, body := w.get(w.cookie, "/projects/new")
	if status != http.StatusOK || !strings.Contains(body, `action="/projects/new/facts"`) || !strings.Contains(body, `name="path"`) {
		t.Fatalf("%d: %s", status, body)
	}
}

func TestAnEmptyRelativeOrMissingPathIsAnErrorOnTheFirstPage(t *testing.T) {
	w := newWizardWorld(t)
	w.put("a-file", "x")
	for path, want := range map[string]string{
		"":                               "enter the path",
		"   ":                            "enter the path",
		"relative/dir":                   "is not an absolute path",
		filepath.Join(w.root, "missing"): "is no existing directory",
		filepath.Join(w.root, "a-file"):  "is no existing directory",
	} {
		for _, step := range []string{"facts", "plan", "apply"} {
			status, body := w.post("/projects/new/"+step, url.Values{"path": {path}})
			if status != http.StatusBadRequest || !strings.Contains(body, want) || !strings.Contains(body, `action="/projects/new/facts"`) {
				t.Errorf("%s with %q = %d, want 400 with %q on the first page: %s", step, path, status, want, body)
			}
		}
	}
	if got := w.read("a-file"); got != "x" {
		t.Errorf("a refusal changed the folder: %q", got)
	}
}

func TestTheFactsNameTheHostsAndTheChoiceStartsFromTheDefaults(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.put("go.mod", "module example.com/demo\n")
	status, body := w.facts()
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, body)
	}
	for _, want := range []string{
		"Agents: claude", "a repository", "go", "not registered", "none yet",
		`name="module" value="hooks" checked`, `name="part" value="config" checked`, `name="part" value="git-hooks" checked`,
		`<option value="en" selected>`, escaped(w.root),
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the facts lack %q", want)
		}
	}
	if len(w.actions()) != 0 || w.read(".loomux/config.toml") != "" {
		t.Fatal("looking at the folder changed it")
	}
}

// escaped is text as the page spells it.
func escaped(text string) string { return template.HTMLEscapeString(text) }

func TestAFreshRepositoryPlansItsConfigurationAsNew(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	status, body := w.plan(german)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, body)
	}
	for _, want := range []string{
		`name="change" value=".loomux/config.toml" checked`, "new file", "&#43;", `name="before:.loomux/config.toml"`,
		`name="action" value="`, "Set up the project",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the plan lacks %q: %s", want, body)
		}
	}
	if w.read(".loomux/config.toml") != "" || len(w.actions()) != 0 {
		t.Fatal("the plan wrote or ran something")
	}
}

func TestApplyingEveryTickWritesTheFilesRegistersTheProjectAndReports(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	status, plan := w.plan(german)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, plan)
	}
	status, done := w.apply(plan, nil)
	if status != http.StatusOK || strings.Contains(done, `class="error"`) {
		t.Fatalf("%d: %s", status, done)
	}
	if !strings.Contains(w.read(".loomux/config.toml"), `language = "de"`) {
		t.Errorf("the configuration lacks the language: %q", w.read(".loomux/config.toml"))
	}
	if w.read(".gitignore") == "" {
		t.Error(".gitignore was not written")
	}
	prefs, err := ReadPrefs(w.stateDir)
	if err != nil || len(prefs.Projects) != 1 || prefs.Projects[0] != w.root {
		t.Fatalf("ui.toml = %+v, %v; want the project", prefs, err)
	}
	for _, want := range []string{"written: .loomux/config.toml", "written: .gitignore", "ran ", `href="/settings/` + projectID(w.root) + `"`} {
		if !strings.Contains(done, want) {
			t.Errorf("the report lacks %q: %s", want, done)
		}
	}
	if len(w.actions()) == 0 {
		t.Error("no action ran")
	}
	if status, _ := w.get(w.cookie, "/settings/"+projectID(w.root)); status != http.StatusOK {
		t.Errorf("the settings of the new project = %d", status)
	}
}

func TestAChangeThatIsNotTickedStaysAsItIs(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.put(".gitignore", "mine\n")
	status, plan := w.plan(german)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, plan)
	}
	status, done := w.apply(plan, func(f url.Values) {
		f["change"] = slices.DeleteFunc(f["change"], func(p string) bool { return p == ".gitignore" })
		f["action"] = nil
	})
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, done)
	}
	if w.read(".gitignore") != "mine\n" || w.read(".loomux/config.toml") == "" {
		t.Errorf("gitignore %q, config %q: want the first kept and the second written", w.read(".gitignore"), w.read(".loomux/config.toml"))
	}
	if !strings.Contains(done, "refused: .gitignore") || !slices.Equal(w.actions(), []string{"hooks-path"}) {
		t.Errorf("the report lacks the refusal, or an unticked action ran (%v): %s", w.actions(), done)
	}
}

func TestAFileChangedBetweenPlanAndApplyIsNeitherOverwrittenNorTheRestRun(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	status, plan := w.plan(german)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, plan)
	}
	w.put(".loomux/config.toml", "# written by hand\n")
	status, again := w.apply(plan, nil)
	if status != http.StatusOK || !strings.Contains(again, "Nothing was run") || !strings.Contains(again, "Set up the project") {
		t.Fatalf("%d: want the plan again: %s", status, again)
	}
	if w.read(".loomux/config.toml") != "# written by hand\n" || w.read(".gitignore") != "" || len(w.actions()) != 0 {
		t.Fatal("a stale apply wrote or ran something")
	}
	if prefs, _ := ReadPrefs(w.stateDir); len(prefs.Projects) != 0 {
		t.Fatalf("a stale apply registered the project: %+v", prefs)
	}
	// The new plan carries the new hashes, and applying that goes through.
	status, done := w.apply(again, nil)
	if status != http.StatusOK || strings.Contains(done, "Nothing was run") {
		t.Fatalf("%d: %s", status, done)
	}
}

func TestAPathWithASpaceAHashAndAnUmlautRunsThroughEveryStep(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	if !strings.Contains(w.root, "#GIT") || !strings.Contains(w.root, "mein projekt ü") {
		t.Fatalf("the world's path is too tame: %s", w.root)
	}
	done := w.setUp()
	if !strings.Contains(done, "Go to the settings") {
		t.Fatalf("no way on: %s", done)
	}
	projects, err := Projects(w.stateDir)
	if err != nil || len(projects) != 1 || projects[0].Root != w.root || projects[0].Name != "mein projekt ü" || projects[0].Missing {
		t.Fatalf("Projects = %+v, %v", projects, err)
	}
}

func TestAnMCPFileThatIsNoObjectIsAnErrorNamingTheFile(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.put(".mcp.json", "[]")
	status, body := w.plan(nil)
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, ".mcp.json") || !strings.Contains(body, `class="error"`) {
		t.Fatalf("%d: %s", status, body)
	}
	if !strings.Contains(body, `action="/projects/new/plan"`) {
		t.Errorf("the choice is not offered again: %s", body)
	}
	if w.read(".mcp.json") != "[]" || w.read(".loomux/config.toml") != "" {
		t.Fatal("a refused plan changed the folder")
	}
}

func TestAFileGatherCannotReadIsAnErrorNamingTheFile(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	// A directory where the configuration should be: reading it fails.
	if err := os.MkdirAll(filepath.Join(w.root, ".loomux", "config.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	status, body := w.facts()
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, ".loomux/config.toml") || strings.Contains(body, `action="/projects/new/plan"`) {
		t.Fatalf("%d: %s", status, body)
	}
}

func TestAnAnswersFileThatDoesNotParseIsAnErrorNamingIt(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.put(".loomux/state/answers.toml", "hosts = [")
	status, body := w.facts()
	if status != http.StatusInternalServerError || !strings.Contains(body, "answers.toml") {
		t.Fatalf("%d: %s", status, body)
	}
}

func TestAProjectThatIsSetUpShowsOnlyWhatIsMissing(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.setUp()
	before := w.read(".loomux/config.toml")
	status, body := w.plan(nil)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, body)
	}
	if strings.Contains(body, `name="change" value=".loomux/config.toml"`) || strings.Contains(body, "new file") {
		t.Errorf("the plan offers the configuration again: %s", body)
	}
	// With nothing chosen there is nothing to do, and nothing to apply.
	status, body = w.plan(func(f url.Values) {
		for _, name := range []string{"module", "part", "lang"} {
			f.Del(name)
		}
	})
	if status != http.StatusOK || !strings.Contains(body, "nothing to do") || strings.Contains(body, `action="/projects/new/apply"`) {
		t.Fatalf("%d: %s", status, body)
	}
	if w.read(".loomux/config.toml") != before {
		t.Fatal("a plan changed the configuration")
	}
}

func TestRunningTheWizardTwiceListsTheProjectOnce(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.setUp()
	w.setUp()
	if prefs, err := ReadPrefs(w.stateDir); err != nil || len(prefs.Projects) != 1 {
		t.Fatalf("ui.toml = %+v, %v; want the project once", prefs, err)
	}
}

func TestAUIFileThatDoesNotParseIsShownAfterTheRun(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	status, plan := w.plan(german)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, plan)
	}
	if err := os.WriteFile(prefsPath(w.stateDir), []byte("projects = ["), 0o600); err != nil {
		t.Fatal(err)
	}
	status, done := w.apply(plan, nil)
	if status != http.StatusOK || !strings.Contains(done, `class="error"`) || !strings.Contains(done, "ui.toml") || strings.Contains(done, "Go to the settings") {
		t.Fatalf("%d: %s", status, done)
	}
	if w.read(".loomux/config.toml") == "" {
		t.Error("the files of the run are gone with the error")
	}
}

// registerArea is the fake of area-add.
func registerArea(w *wizardWorld) func(a setup.Action, scope string) error {
	return func(a setup.Action, scope string) error {
		if a.ID != "area-add" {
			return nil
		}
		return config.AddArea(w.stateDir, config.Area{Scope: scope, Path: filepath.ToSlash(w.root)})
	}
}

func llmOf(t *testing.T, w *wizardWorld) bool {
	t.Helper()
	areas, err := config.ReadRegistry(w.stateDir)
	if err != nil || len(areas) != 1 {
		t.Fatalf("%v %v", areas, err)
	}
	return areas[0].LLMConfig
}

func TestTheLLMSwitchIsAskedLastAndOffUnlessTicked(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.onRun = registerArea(w)
	status, plan := w.plan(nil)
	if status != http.StatusOK || !strings.Contains(plan, `name="llm" value="1">`) || strings.Contains(plan, `name="llm" value="1" checked`) {
		t.Fatalf("%d: the switch is not offered, or on by default: %s", status, plan)
	}
	for _, want := range []string{escaped(llmOpens), escaped(llmLoosens)} {
		if !strings.Contains(plan, want) {
			t.Errorf("the plan does not say what the switch opens: lacks %q", want)
		}
	}
	if status, done := w.apply(plan, nil); status != http.StatusOK || strings.Contains(done, `class="error"`) {
		t.Fatalf("%d: %s", status, done)
	}
	if llmOf(t, w) {
		t.Fatal("the switch is on without a tick")
	}
	status, done := w.apply(plan, func(f url.Values) { f.Set("llm", "1") })
	if status != http.StatusOK || strings.Contains(done, `class="error"`) || !llmOf(t, w) {
		t.Fatalf("%d: the switch is not on after the tick: %s", status, done)
	}
}

func TestNoAreaNoSwitchAndNoSwitchPageToAsk(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	// Without the registering parts the plan does not ask.
	status, plan := w.plan(func(f url.Values) {
		f["part"] = slices.DeleteFunc(f["part"], func(p string) bool { return p == "area" || p == "workspace" })
	})
	if status != http.StatusOK || strings.Contains(plan, `name="llm"`) {
		t.Fatalf("%d: the plan asks for a switch it cannot set: %s", status, plan)
	}
	// A forged tick changes nothing where no registry exists, or where the
	// registry knows other areas only.
	for _, other := range []bool{false, true} {
		if other {
			err := config.WriteRegistry(w.stateDir, []config.Area{{Scope: "project/other", Path: filepath.ToSlash(filepath.Join(t.TempDir(), "other"))}})
			if err != nil {
				t.Fatal(err)
			}
		}
		status, done := w.apply(plan, func(f url.Values) { f.Set("llm", "1") })
		if status != http.StatusOK || strings.Contains(done, `class="error"`) {
			t.Fatalf("other=%v: %d: %s", other, status, done)
		}
	}
	if areas, _ := config.ReadRegistry(w.stateDir); len(areas) != 1 || areas[0].LLMConfig {
		t.Fatalf("registry = %+v, want the other area alone", areas)
	}
}

func TestARegistryThatCannotBeReadAfterTheRunIsShown(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.onRun = func(a setup.Action, _ string) error {
		if a.ID != "area-add" {
			return nil
		}
		return os.WriteFile(filepath.Join(w.stateDir, "registry.toml"), []byte("[[area"), 0o600)
	}
	status, plan := w.plan(nil)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, plan)
	}
	status, done := w.apply(plan, func(f url.Values) { f.Set("llm", "1") })
	if status != http.StatusOK || !strings.Contains(done, `class="error"`) {
		t.Fatalf("%d: %s", status, done)
	}
}
