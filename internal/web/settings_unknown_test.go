package web

import (
	"net/http"
	"strings"
	"testing"
)

func TestKeysTheSchemaDoesNotKnowAreShownAndPointToTheRawText(t *testing.T) {
	u := startUI(t)
	text := "[commit]\nlanguage = \"de\"\n[dotnet]\nsdk = \"8\"\n[commit.extra]\nx = 1\n"
	id, body, status := settingsOf(t, u, "p", text, "")
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	for _, want := range []string{"Not in the schema", "<code>dotnet</code>", "<code>commit.extra</code>", `href="/settings/` + id + `/raw"`} {
		if !strings.Contains(body, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	// A key the schema knows is not listed as unknown.
	if strings.Contains(body, "<code>commit.language</code></li>") {
		t.Error("a known key is listed as unknown")
	}
	// The search narrows them like the rows.
	_, narrowed := u.get(u.session(), "/settings/"+id+"?q=dotnet")
	if !strings.Contains(narrowed, "<code>dotnet</code>") || strings.Contains(narrowed, "<code>commit.extra</code>") {
		t.Errorf("the search did not narrow the unknown keys: %s", narrowed)
	}
}

func TestAFileWithoutUnknownKeysHasNoSuchSection(t *testing.T) {
	u := startUI(t)
	_, body, _ := settingsOf(t, u, "p", "[commit]\nlanguage = \"de\"\n", "")
	if strings.Contains(body, "Not in the schema") {
		t.Error("the page lists unknown keys of a file that has none")
	}
}

func TestAModulesTableThatDoesNotReadIsSaidAndEveryModuleShownAsOn(t *testing.T) {
	u := startUI(t)
	_, body, status := settingsOf(t, u, "p", "[modules]\nhooks = \"yes\"\n", "")
	if status != http.StatusOK || !strings.Contains(body, "every module is shown as on") || !strings.Contains(body, "<h2>brain</h2>") {
		t.Errorf("%d: the page does not say that [modules] was not read: %s", status, body)
	}
}

func TestABrokenMachineWideRuleHidesNeitherTheBuiltInNorTheProjectsRules(t *testing.T) {
	u := startUI(t)
	writeGlobal(t, u, "[[policy.paths.rules]]\nmatch = \"x\"\n")
	text := "[[policy.paths.rules]]\nmatch = [\"mine/**\"]\nreason = \"project rule\"\n"
	_, body, _ := settingsOf(t, u, "p", text, "")
	for _, want := range []string{"needs a reason", "[built-in, read-only]", "<code>mine/**</code> — project rule"} {
		if !strings.Contains(body, want) {
			t.Errorf("the page lacks %q: %s", want, body)
		}
	}
}

func TestABrokenProjectRuleHidesNeitherTheBuiltInNorTheMachinesRules(t *testing.T) {
	u := startUI(t)
	writeGlobal(t, u, "[[policy.paths.rules]]\nmatch = [\"g/**\"]\nreason = \"machine rule\"\n")
	_, body, _ := settingsOf(t, u, "p", "[[policy.paths.rules]]\nmatch = \"x\"\n", "")
	for _, want := range []string{"needs a reason", "[built-in, read-only]", "<code>g/**</code> — machine rule"} {
		if !strings.Contains(body, want) {
			t.Errorf("the page lacks %q: %s", want, body)
		}
	}
}
