package web

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/brain/guard"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/edit"
	"github.com/xidus90/loomux/internal/config/schema"
	"github.com/xidus90/loomux/internal/pathkey"
	"github.com/xidus90/loomux/internal/setup"
)

// The wizard keeps no state of its own: every step carries the path and the
// choice forward as form fields and works out the facts and the plan again,
// so what a step shows is what the files say now.

// wizardModules are the optional modules the facts page switches; the base
// module always runs.
var wizardModules = []schema.Module{schema.Hooks, schema.Brain, schema.Graph}

// languages are the commit languages the configuration knows.
func languages() []string {
	key, _ := schema.Lookup("commit.language")
	return key.Choices
}

// pathView is the data of the first page.
type pathView struct{ Path, Error string }

// moduleRow and partRow are one tick box of the choice.
type moduleRow struct {
	ID string
	On bool
}

type partRow struct {
	setup.Part
	On bool
}

// factsView is the data of the facts page, which is also where the choice is
// made, and where a failure of the next step is shown.
type factsView struct {
	Path, Error string
	Facts       setup.Facts
	Modules     []moduleRow
	Parts       []partRow
	Lang        string
	Languages   []string
	// Scope is the area's scope, which only a run that registers an area
	// uses.
	Scope string
}

// carry is the choice a page sends on to the next step.
type carry struct {
	Path, Lang, Scope string
	Modules           []string
	Parts             []string
}

// changeRow is one file of the plan with the tick box that lets it be
// written, and the hash of the text the plan was made from.
type changeRow struct {
	Path, Part, Diff, Hash string
	New                    bool
}

// planView is the data of the plan page.
type planView struct {
	Carry   carry
	Changes []changeRow
	Actions []setup.Action
	Notes   []string
	// AskLLM says the run leaves a registered area, which is where the
	// llm_config switch exists.
	AskLLM         bool
	Opens, Loosens string
	// Stale says the files changed since the plan the form was made from.
	Stale   bool
	Nothing bool
}

// doneView is the data of the page that ends the wizard.
type doneView struct {
	Report setup.Report
	// Output is what the actions printed.
	Output, Error, ID string
}

// existingDir is the directory a path names, or why it names none.
func existingDir(path string) (string, error) {
	switch {
	case strings.TrimSpace(path) == "":
		return "", errors.New("enter the path of the project folder")
	case !filepath.IsAbs(path):
		return "", fmt.Errorf("%s is not an absolute path", path)
	}
	root := filepath.Clean(path)
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s is no existing directory", path)
	}
	return root, nil
}

// wizardPath is the first page: the folder.
func (s *server) wizardPath(w http.ResponseWriter, r *http.Request) {
	s.page(w, r, "wizard_path", pathView{})
}

// failure is the status of a step that stopped on a file it cannot merge or
// read.
func failure(err error) int {
	if setup.Unmergeable(err) {
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}

// project is what every step after the first works from.
type project struct {
	root   string
	facts  setup.Facts
	choice setup.Choice
}

// gather validates the path of the form, reads the facts of the folder and
// starts the choice from what the folder already says. It answers the
// request itself when it cannot.
func (s *server) gather(w http.ResponseWriter, r *http.Request) (project, bool) {
	root, err := existingDir(r.PostFormValue("path"))
	if err != nil {
		s.render(w, r, http.StatusBadRequest, "wizard_path", pathView{Path: r.PostFormValue("path"), Error: err.Error()})
		return project{}, false
	}
	facts, err := s.options.Gather(root)
	var answers setup.Answers
	if err == nil {
		answers, err = setup.ReadAnswers(root)
	}
	if err != nil {
		s.render(w, r, failure(err), "wizard_facts", factsView{Path: root, Error: err.Error()})
		return project{}, false
	}
	return project{root: root, facts: facts, choice: setup.DefaultChoice(facts, answers)}, true
}

// chosen overrides the choice with what the form holds: each module and part
// is on when its field is there, so a tick box that is not ticked is off.
func (p *project) chosen(form url.Values) {
	for _, m := range wizardModules {
		p.choice.Modules[m] = slices.Contains(form["module"], string(m))
	}
	for _, part := range setup.Parts(p.facts) {
		p.choice.Parts[part.ID] = slices.Contains(form["part"], part.ID)
	}
	if lang := form.Get("lang"); slices.Contains(languages(), lang) {
		p.choice.CommitLanguage = lang
	}
	// The scope is only asked where area add takes it, as `init` does: an
	// area with the brain module on.
	if scope := strings.TrimSpace(form.Get("scope")); p.asksScope() && scope != "" {
		p.choice.Scope = scope
	}
}

// asksScope says whether the choice registers an area with a wiki, which is
// where a scope of the human's own is asked for.
func (p project) asksScope() bool {
	return p.choice.Parts["area"] && p.choice.Modules[schema.Brain]
}

// checkScope refuses a scope area add could not take, where one is asked.
func (p project) checkScope() error {
	if !p.asksScope() {
		return nil
	}
	return setup.CheckScope(p.choice.Scope)
}

// carry is the choice p holds, in the fields chosen reads back.
func (p project) carry() carry {
	c := carry{Path: p.root, Lang: p.choice.CommitLanguage, Scope: p.choice.Scope}
	for _, m := range wizardModules {
		if p.choice.Modules[m] {
			c.Modules = append(c.Modules, string(m))
		}
	}
	for _, part := range setup.Parts(p.facts) {
		if p.choice.Parts[part.ID] {
			c.Parts = append(c.Parts, part.ID)
		}
	}
	return c
}

// showFacts renders the facts page with the choice p holds.
func (s *server) showFacts(w http.ResponseWriter, r *http.Request, status int, p project, problem string) {
	v := factsView{Path: p.root, Error: problem, Facts: p.facts, Lang: p.choice.CommitLanguage, Languages: languages(), Scope: p.choice.Scope}
	for _, m := range wizardModules {
		v.Modules = append(v.Modules, moduleRow{ID: string(m), On: p.choice.Modules[m]})
	}
	for _, part := range setup.Parts(p.facts) {
		v.Parts = append(v.Parts, partRow{Part: part, On: p.choice.Parts[part.ID]})
	}
	s.render(w, r, status, "wizard_facts", v)
}

// wizardFacts reads the folder and offers the choice, filled in with what
// the folder already says.
func (s *server) wizardFacts(w http.ResponseWriter, r *http.Request) {
	if p, ok := s.gather(w, r); ok {
		s.showFacts(w, r, http.StatusOK, p, "")
	}
}

// plan builds what the choice of the form does to the project. It answers
// the request itself when the plan fails.
func (s *server) plan(w http.ResponseWriter, r *http.Request) (project, setup.Plan, bool) {
	p, ok := s.gather(w, r)
	if !ok {
		return p, setup.Plan{}, false
	}
	p.chosen(r.PostForm)
	if err := p.checkScope(); err != nil {
		s.showFacts(w, r, http.StatusBadRequest, p, err.Error())
		return p, setup.Plan{}, false
	}
	plan, err := setup.Build(p.facts, p.choice, setup.ProjectReader(p.root))
	if err != nil {
		s.showFacts(w, r, http.StatusUnprocessableEntity, p, err.Error())
		return p, plan, false
	}
	plan.Notes = append(plan.Notes, setup.Notes(p.facts, p.choice, plan)...)
	return p, plan, true
}

// showPlan renders every change as a diff with its tick box, ticked.
func (s *server) showPlan(w http.ResponseWriter, r *http.Request, p project, plan setup.Plan, changed bool) {
	v := planView{Carry: p.carry(), Actions: plan.Actions, Notes: plan.Notes, Stale: changed, Opens: llmOpens, Loosens: llmLoosens}
	for _, ch := range plan.Changes {
		v.Changes = append(v.Changes, changeRow{Path: ch.Path, Part: ch.Part, Diff: edit.Diff(ch.Before, ch.After), Hash: hashOf(ch.Before), New: !ch.Exists})
	}
	v.Nothing = len(plan.Changes) == 0 && len(plan.Actions) == 0
	v.AskLLM = p.facts.Registered || slices.ContainsFunc(plan.Actions, func(a setup.Action) bool {
		return a.ID == "area-add" || a.ID == "workspace-add"
	})
	s.page(w, r, "wizard_plan", v)
}

// wizardPlan shows what the choice would write and run. It writes nothing.
func (s *server) wizardPlan(w http.ResponseWriter, r *http.Request) {
	if p, plan, ok := s.plan(w, r); ok {
		s.showPlan(w, r, p, plan, false)
	}
}

// stale says whether a file of the plan is no longer the text the shown plan
// was made from: the change is new to the form, or its hash differs.
func stale(plan setup.Plan, form url.Values) bool {
	return slices.ContainsFunc(plan.Changes, func(ch setup.Change) bool {
		return form.Get("before:"+ch.Path) != hashOf(ch.Before)
	})
}

// wizardApply runs what the human ticked, if the files are still the ones
// the plan was made from. Otherwise it runs nothing and shows the plan as it
// is now.
func (s *server) wizardApply(w http.ResponseWriter, r *http.Request) {
	p, plan, ok := s.plan(w, r)
	if !ok {
		return
	}
	if stale(plan, r.PostForm) {
		s.showPlan(w, r, p, plan, true)
		return
	}
	approved := map[string]bool{}
	for _, ch := range plan.Changes {
		approved[ch.Path] = slices.Contains(r.PostForm["change"], ch.Path)
	}
	for _, a := range plan.Actions {
		approved[a.ID] = slices.Contains(r.PostForm["action"], a.ID)
	}
	var output bytes.Buffer
	deps := s.options.SetupDeps(p.root, p.facts, p.choice.Scope, &output, &output)
	report, err := setup.Execute(p.root, p.facts, p.choice, plan, approved, deps, s.version, s.now())
	if err == nil && r.PostForm.Get("llm") == "1" {
		err = s.switchLLM(p.root)
	}
	// A run that stopped half way has written files already: the project is
	// listed either way, so that it can be looked at and finished.
	err = errors.Join(err, s.remember(p.root))
	v := doneView{Report: report, Output: output.String(), ID: projectID(p.root)}
	if err != nil {
		v.Error = err.Error()
	}
	s.page(w, r, "wizard_done", v)
}

// switchLLM turns the switch on for the area the run left at root. A run
// that left none has nothing to switch.
func (s *server) switchLLM(root string) error {
	scope, _, err := guard.AreaAt(root, s.options.StateDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if scope == "" {
		return nil
	}
	return config.SetLLMConfig(s.options.StateDir, scope, true)
}

// remember adds root to the projects of ui.toml, unless it is there.
func (s *server) remember(root string) error {
	prefs, err := ReadPrefs(s.options.StateDir)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(prefs.Projects, func(known string) bool { return pathkey.Key(known) == pathkey.Key(root) }) {
		return nil
	}
	prefs.Projects = append(prefs.Projects, root)
	return writePrefs(s.options.StateDir, prefs)
}
