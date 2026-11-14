package web

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const twoPathRules = "[[policy.paths.rules]]\nmatch = [\"a/**\"]\nreason = \"first\"\n\n[[policy.paths.rules]]\nmatch = [\"b/**\"]\nreason = \"second\"\n"

// rule previews one entry for a list and returns the page.
func (w *writer) rule(form url.Values) string {
	w.t.Helper()
	status, body := w.post("/settings/"+w.id+"/rules/preview", form)
	if status != http.StatusOK {
		w.t.Fatalf("rules/preview = %d: %s", status, body)
	}
	return body
}

func TestAPathRuleIsPreviewedThenWrittenAndListedAsAProjectRule(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	body := w.rule(url.Values{"section": {"policy.paths.rules"}, "match": {"docs/**\n secrets/* \n"}, "reason": {"keep out"}})
	for _, want := range []string{"[[policy.paths.rules]]", "match = [&#34;docs/**&#34;, &#34;secrets/*&#34;]", "reason = &#34;keep out&#34;", ">Apply</button>"} {
		if !strings.Contains(body, want) {
			t.Errorf("the preview lacks %q: %s", want, body)
		}
	}
	if w.file() != "[commit]\nthreshold = 5\n" {
		t.Fatal("a preview wrote the file")
	}
	if status, _ := w.applyOf(body); status != http.StatusSeeOther {
		t.Fatalf("apply = %d", status)
	}
	if got := w.file(); !strings.Contains(got, "match = [\"docs/**\", \"secrets/*\"]\nreason = \"keep out\"\n") {
		t.Fatalf("file = %q", got)
	}
	_, page := w.page()
	if !strings.Contains(page, "docs/**, secrets/*") || !strings.Contains(page, "[project]") {
		t.Errorf("the page does not list the new rule as the project's: %s", page)
	}
}

// page is the settings page of the writer's project.
func (w *writer) page() (int, string) { return w.ui.get(w.cookie, "/settings/"+w.id) }

func TestARuleLandsInAnInlineListAndNotAsABlock(t *testing.T) {
	text := "[policy.paths]\nrules = [\n  { match = [\"a/**\"], reason = \"first\" },\n]\n"
	w := writerOf(t, text)
	body := w.rule(url.Values{"section": {"policy.paths.rules"}, "match": {"b/**"}, "reason": {"second"}})
	w.applyOf(body)
	got := w.file()
	if strings.Contains(got, "[[policy.paths.rules]]") || !strings.Contains(got, `match = ["b/**"]`) {
		t.Fatalf("file = %q", got)
	}
	if _, page := w.page(); !strings.Contains(page, "a/**") || !strings.Contains(page, "b/**") {
		t.Errorf("the page lacks a rule: %s", page)
	}
}

func TestACommandRuleAndAnAllowedLineTakeARegexAndAReason(t *testing.T) {
	w := writerOf(t, "")
	for section, want := range map[string]string{
		"policy.commands.rules": "[[policy.commands.rules]]\nregex = \"^rm -rf\"\nreason = \"no\"\n",
		"commit.allow":          "[[commit.allow]]\nregex = \"(Müller|Zürich)\"\nreason = \"names\"\n",
	} {
		regex := "^rm -rf"
		reason := "no"
		if section == "commit.allow" {
			regex, reason = "(Müller|Zürich)", "names"
		}
		body := w.rule(url.Values{"section": {section}, "regex": {regex}, "reason": {reason}})
		w.applyOf(body)
		if !strings.Contains(w.file(), want) {
			t.Errorf("%s: file = %q, want %q", section, w.file(), want)
		}
	}
	_, page := w.page()
	if !strings.Contains(page, "(Müller|Zürich)") || !strings.Contains(page, `name="remove" value="0"`) {
		t.Errorf("the allowed line has no removal on the page: %s", page)
	}
}

func TestARuleIsRemovedByItsIndexInTheFile(t *testing.T) {
	w := writerOf(t, twoPathRules)
	body := w.rule(url.Values{"section": {"policy.paths.rules"}, "remove": {"1"}})
	if !strings.Contains(body, "- reason = &#34;second&#34;") {
		t.Fatalf("the preview shows no removal: %s", body)
	}
	w.applyOf(body)
	if got := w.file(); !strings.Contains(got, "first") || strings.Contains(got, "second") {
		t.Fatalf("file = %q", got)
	}
}

func TestARuleWithABrokenGlobOrNoReasonIsAnErrorOnThePreview(t *testing.T) {
	for name, form := range map[string]url.Values{
		"broken glob": {"section": {"policy.paths.rules"}, "match": {"a/["}, "reason": {"r"}},
		"no glob":     {"section": {"policy.paths.rules"}, "match": {"  \n"}, "reason": {"r"}},
		"no reason":   {"section": {"policy.paths.rules"}, "match": {"a/**"}, "reason": {" "}},
		"bad regex":   {"section": {"policy.commands.rules"}, "regex": {"("}, "reason": {"r"}},
		"allow regex": {"section": {"commit.allow"}, "regex": {"("}, "reason": {"r"}},
		"no section":  {"section": {"verify"}, "regex": {"x"}, "reason": {"r"}},
	} {
		w := writerOf(t, "[commit]\nthreshold = 5\n")
		body := w.rule(form)
		if !strings.Contains(body, `class="error"`) || strings.Contains(body, ">Apply<") {
			t.Errorf("%s: want an error and no apply button: %s", name, body)
		}
		if w.file() != "[commit]\nthreshold = 5\n" {
			t.Errorf("%s wrote the file", name)
		}
	}
}

func TestARemovalNamesOnlyRulesOfTheProjectFile(t *testing.T) {
	w := writerOf(t, "[[policy.paths.rules]]\nmatch = [\"a/**\"]\nreason = \"mine\"\n")
	writeGlobal(t, w.ui, "[[policy.paths.rules]]\nmatch = [\"g1/**\"]\nreason = \"global one\"\n\n[[policy.paths.rules]]\nmatch = [\"g2/**\"]\nreason = \"global two\"\n")
	_, page := w.page()
	for _, want := range []string{"[built-in, read-only]", "[global, read-only]", "[project]"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	// Only the project's one rule has a button, and its index counts in the
	// file, not in the list the page shows.
	if n := strings.Count(page, `name="remove"`); n != 1 || !strings.Contains(page, `name="remove" value="0"`) {
		t.Errorf("want one removal, number 0: %d", n)
	}
	for _, index := range []string{"1", "2", "-1", "x"} {
		body := w.rule(url.Values{"section": {"policy.paths.rules"}, "remove": {index}})
		if !strings.Contains(body, `class="error"`) || strings.Contains(body, ">Apply<") {
			t.Errorf("remove=%s: want an error: %s", index, body)
		}
	}
	if !strings.Contains(w.file(), "mine") {
		t.Fatal("the project's rule is gone")
	}
}

func TestTheMachinesFileTakesRulesOfItsTwoListsOnly(t *testing.T) {
	u := startUI(t)
	cookie, token := u.signIn()
	post := func(form url.Values) string {
		form.Set("csrf", token)
		resp := u.do(req{method: http.MethodPost, path: "/settings/global/rules/preview", cookie: cookie, form: form})
		return bodyOf(t, resp)
	}
	if body := post(url.Values{"section": {"commit.allow"}, "regex": {"x"}, "reason": {"r"}}); !strings.Contains(body, `class="error"`) {
		t.Errorf("the global file took a project list: %s", body)
	}
	body := post(url.Values{"section": {"policy.commands.rules"}, "regex": {"^curl"}, "reason": {"no network"}})
	if !strings.Contains(body, "[[policy.commands.rules]]") {
		t.Fatalf("no new block: %s", body)
	}
	form := url.Values{"before": {hidden(t, body, "before")}, "next": {hidden(t, body, "next")}, "csrf": {token}}
	resp := u.do(req{method: http.MethodPost, path: "/settings/global/apply", cookie: cookie, form: form})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("apply = %d", resp.StatusCode)
	}
	data, err := os.ReadFile(filepath.Join(u.stateDir, "config.toml"))
	if err != nil || !strings.Contains(string(data), `regex = "^curl"`) {
		t.Fatalf("%q %v", data, err)
	}
}

func TestARuleFormOfAnUnknownProjectIs404(t *testing.T) {
	w := writerOf(t, "")
	if status, _ := w.post("/settings/nope/rules/preview", url.Values{"section": {"commit.allow"}}); status != http.StatusNotFound {
		t.Fatalf("status %d", status)
	}
}
