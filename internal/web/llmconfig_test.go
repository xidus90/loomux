package web

import (
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// register puts the writer's project into the registry as an area.
func (w *writer) register(llm bool) {
	w.t.Helper()
	err := config.WriteRegistry(w.stateDir, []config.Area{{Scope: "project/mein-projekt", Path: filepath.ToSlash(w.root()), LLMConfig: llm}})
	if err != nil {
		w.t.Fatal(err)
	}
}

func (w *writer) llm() bool {
	w.t.Helper()
	areas, err := config.ReadRegistry(w.stateDir)
	if err != nil || len(areas) != 1 {
		w.t.Fatalf("%v %v", areas, err)
	}
	return areas[0].LLMConfig
}

func TestTheSwitchIsTurnedOnAfterThePageSaidWhatItOpens(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	w.register(false)
	_, settings := w.page()
	if !strings.Contains(settings, "The switch is off") || !strings.Contains(settings, "/settings/"+w.id+"/llm") {
		t.Fatalf("the settings page does not offer the switch: %s", settings)
	}
	status, page := w.ui.get(w.cookie, "/settings/"+w.id+"/llm")
	if status != http.StatusOK {
		t.Fatalf("%d: %s", status, page)
	}
	for _, want := range []string{template.HTMLEscapeString(llmOpens), template.HTMLEscapeString(llmLoosens), `name="on" value="true"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the confirmation lacks %q", want)
		}
	}
	if w.llm() {
		t.Fatal("reading the page turned the switch on")
	}
	if status, _ := w.post("/settings/"+w.id+"/llm", url.Values{"on": {"true"}}); status != http.StatusSeeOther || !w.llm() {
		t.Fatalf("on: %d, %v", status, w.llm())
	}
	if _, page := w.ui.get(w.cookie, "/settings/"+w.id+"/llm"); !strings.Contains(page, `name="on" value="false"`) {
		t.Errorf("an open switch offers no way to close it: %s", page)
	}
	if status, _ := w.post("/settings/"+w.id+"/llm", url.Values{"on": {"false"}}); status != http.StatusSeeOther || w.llm() {
		t.Fatalf("off: %d, %v", status, w.llm())
	}
}

func TestTheSwitchTakesOnlyTrueOrFalse(t *testing.T) {
	w := writerOf(t, "")
	w.register(false)
	for _, on := range []string{"", "yes", "1"} {
		if status, _ := w.post("/settings/"+w.id+"/llm", url.Values{"on": {on}}); status != http.StatusBadRequest || w.llm() {
			t.Errorf("on=%q: %d, %v", on, status, w.llm())
		}
	}
}

func TestOnlyARegisteredAreaHasTheSwitch(t *testing.T) {
	w := writerOf(t, "")
	_, page := w.page()
	if !strings.Contains(page, "only a registered area has this switch") || strings.Contains(page, "/llm\"") {
		t.Errorf("the page offers a switch to a project that is not registered: %s", page)
	}
	for _, id := range []string{w.id, "global"} {
		if status, body := w.ui.get(w.cookie, "/settings/"+id+"/llm"); status != http.StatusConflict || !strings.Contains(body, "only a registered area") {
			t.Errorf("GET %s: %d %s", id, status, body)
		}
		if status, _ := w.post("/settings/"+id+"/llm", url.Values{"on": {"true"}}); status != http.StatusConflict {
			t.Errorf("POST %s: %d", id, status)
		}
	}
	if status, _ := w.post("/settings/nope/llm", url.Values{"on": {"true"}}); status != http.StatusNotFound {
		t.Errorf("unknown project: %d", status)
	}
	if _, global := w.ui.get(w.cookie, "/settings/global"); strings.Contains(global, "llm_config") {
		t.Error("the machine's page shows the switch")
	}
}

func TestAnUnwritableRegistryIsAnErrorAndLeavesTheSwitch(t *testing.T) {
	w := writerOf(t, "")
	w.register(false)
	// A directory where the lock file belongs.
	if err := os.Remove(filepath.Join(w.stateDir, "registry.lock")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(w.stateDir, "registry.lock"), 0o755); err != nil {
		t.Fatal(err)
	}
	if status, _ := w.post("/settings/"+w.id+"/llm", url.Values{"on": {"true"}}); status != http.StatusInternalServerError || w.llm() {
		t.Fatalf("%d, %v", status, w.llm())
	}
}

func TestTheConfirmationQuotesTheDocumentation(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "en", "configuration.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, sentence := range []string{llmOpens, llmLoosens} {
		if !strings.Contains(string(data), sentence) {
			t.Errorf("docs/en/configuration.md does not say %q", sentence)
		}
	}
}
