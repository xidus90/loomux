package web

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/setup"
)

// scopesOf makes the fake of area-add remember the scope it was given.
func scopesOf(w *wizardWorld) *[]string {
	var scopes []string
	w.onRun = func(a setup.Action, scope string) error {
		if a.ID == "area-add" {
			scopes = append(scopes, scope)
		}
		return nil
	}
	return &scopes
}

func TestTheScopeIsAskedAndTheAreaIsRegisteredUnderIt(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	scopes := scopesOf(w)
	status, facts := w.facts()
	if want := `name="scope" value="` + escaped("project/mein-projekt-ü") + `"`; status != http.StatusOK || !strings.Contains(facts, want) {
		t.Fatalf("%d: the facts do not offer the default scope: %s", status, facts)
	}
	status, plan := w.plan(func(f url.Values) { f.Set("scope", " project/mine ") })
	if status != http.StatusOK || !strings.Contains(plan, `name="scope" value="project/mine"`) {
		t.Fatalf("%d: the plan does not carry the scope: %s", status, plan)
	}
	if status, done := w.apply(plan, nil); status != http.StatusOK || strings.Contains(done, `class="error"`) {
		t.Fatalf("%d: %s", status, done)
	}
	if !slices.Equal(*scopes, []string{"project/mine"}) {
		t.Fatalf("area add ran with %v, want project/mine", *scopes)
	}
}

func TestAScopeAreaAddCouldNotTakeIsRefusedOnTheFactsPage(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	for _, step := range []string{"plan", "apply"} {
		status, facts := w.facts()
		if status != http.StatusOK {
			t.Fatalf("%d: %s", status, facts)
		}
		form := formOf(facts)
		form.Set("scope", "project/a b")
		status, body := w.post("/projects/new/"+step, form)
		if status != http.StatusBadRequest || !strings.Contains(body, "no spaces") || !strings.Contains(body, `action="/projects/new/plan"`) {
			t.Errorf("%s: %d: %s", step, status, body)
		}
	}
	if len(w.actions()) != 0 || w.read(".loomux/config.toml") != "" {
		t.Fatal("a refused scope changed the project")
	}
}

func TestTheScopeIsLeftAloneWhereNoAreaIsRegistered(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	scopes := scopesOf(w)
	status, plan := w.plan(func(f url.Values) {
		f["part"] = slices.DeleteFunc(f["part"], func(p string) bool { return p == "area" })
		f.Set("scope", "project/a b")
	})
	if status != http.StatusOK || !strings.Contains(plan, `name="scope" value="`+escaped("project/mein-projekt-ü")+`"`) {
		t.Fatalf("%d: a scope nobody uses was judged or carried: %s", status, plan)
	}
	w.apply(plan, nil)
	if len(*scopes) != 0 {
		t.Fatalf("area add ran: %v", *scopes)
	}
}

func TestTheRunIsStampedWithTheInterfacesClock(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.advance.Store(int64(10 * 365 * 24 * time.Hour))
	w.setUp()
	m := regexp.MustCompile(`at = (\d{4})-`).FindStringSubmatch(w.read(".loomux/state/installed.toml"))
	if m == nil {
		t.Fatalf("no time in installed.toml: %q", w.read(".loomux/state/installed.toml"))
	}
	if year, _ := strconv.Atoi(m[1]); year < time.Now().Year()+9 {
		t.Errorf("installed.toml says %s, not the interface's clock", m[1])
	}
}

func TestAFailingWriteOfTheProjectListIsShownAfterTheRun(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	status, plan := w.plan(german)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, plan)
	}
	saved := writePrefs
	writePrefs = func(string, Prefs) error { return errors.New("disk full") }
	t.Cleanup(func() { writePrefs = saved })
	status, done := w.apply(plan, nil)
	if status != http.StatusOK || !strings.Contains(done, "disk full") || strings.Contains(done, "Go to the settings") {
		t.Fatalf("%d: %s", status, done)
	}
}

func TestAProjectWhoseRunStoppedHalfWayIsListedAnyway(t *testing.T) {
	w := newWizardWorld(t)
	w.gitRepo()
	w.onRun = func(a setup.Action, _ string) error {
		if a.ID == "hooks-path" {
			return errors.New("git refused")
		}
		return nil
	}
	status, plan := w.plan(german)
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, plan)
	}
	status, done := w.apply(plan, nil)
	if status != http.StatusOK || !strings.Contains(done, "git refused") {
		t.Fatalf("%d: the stop is not shown: %s", status, done)
	}
	if prefs, err := ReadPrefs(w.stateDir); err != nil || len(prefs.Projects) != 1 {
		t.Fatalf("ui.toml = %+v, %v; the half-done project is not listed", prefs, err)
	}
}
