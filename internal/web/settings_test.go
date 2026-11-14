package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// get fetches a page with the session and returns its status and body.
func (u *ui) get(cookie, path string) (int, string) {
	u.t.Helper()
	resp := u.do(req{path: path, cookie: cookie})
	return resp.StatusCode, bodyOf(u.t, resp)
}

// settingsOf is the page of a project made in the test's world.
func settingsOf(t *testing.T, u *ui, name, text, query string) (id, body string, status int) {
	t.Helper()
	root := makeProject(t, name, text)
	remember(t, u.stateDir, root)
	cookie := u.session()
	id = projectID(root)
	status, body = u.get(cookie, "/settings/"+id+query)
	return id, body, status
}

func writeGlobal(t *testing.T, u *ui, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(u.stateDir, "config.toml"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTheGlobalPageListsTheModelAndBothRuleLists(t *testing.T) {
	u := startUI(t)
	cookie := u.session()
	status, body := u.get(cookie, "/settings/global")
	if status != http.StatusOK {
		t.Fatalf("status %d, want 200: %s", status, body)
	}
	for _, want := range []string{
		"<code>model.enabled</code>", "<code>policy.paths.rules</code>", "<code>policy.commands.rules</code>",
		"built-in", "blocks writes, not reads", "Whether commits reach the remote",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the global page lacks %q", want)
		}
	}
	if strings.Contains(body, "<code>commit.language</code>") {
		t.Error("the global page lists a project key")
	}
}

func TestAProjectUnderAnAwkwardPathGetsItsPage(t *testing.T) {
	u := startUI(t)
	_, body, status := settingsOf(t, u, "mein projekt ü", "[commit]\nlanguage = \"de\"\n", "")
	if status != http.StatusOK || !strings.Contains(body, "<h1>mein projekt ü</h1>") {
		t.Fatalf("status %d, body %s", status, body)
	}
	if !strings.Contains(body, "<code>commit.language</code>") || !strings.Contains(body, "<code>&#34;de&#34;</code>") {
		t.Error("the page does not show the key with its value")
	}
	if !strings.Contains(body, `<a href="/settings/global">Global</a>`) || !strings.Contains(body, `href="/projects/new">Add project`) {
		t.Error("the header lacks Global or Add project")
	}
	if !strings.Contains(body, `>mein projekt ü</a>`) {
		t.Error("the header's project switcher does not link the project")
	}
}

func TestSearchFiltersByKeyAndDescriptionWithoutRegardToCase(t *testing.T) {
	u := startUI(t)
	_, body, _ := settingsOf(t, u, "p", "[modules]\n", "?q=COMMIT")
	if !strings.Contains(body, "<code>commit.language</code>") || strings.Contains(body, "<code>worktree.mirror</code>") {
		t.Error("?q=COMMIT must keep commit keys and drop the rest")
	}
	cookie := u.session()
	_, byDoc := u.get(cookie, "/settings/"+projectOnly(t, u)+"?q=how+many+SECONDS")
	if !strings.Contains(byDoc, "<code>verify.timeout</code>") || strings.Contains(byDoc, "<code>commit.language</code>") {
		t.Error("the search must read the description, not only the key")
	}
}

// projectOnly is the ID of the one project the test world holds.
func projectOnly(t *testing.T, u *ui) string {
	t.Helper()
	projects, err := Projects(u.stateDir)
	if err != nil || len(projects) != 1 {
		t.Fatalf("Projects = %v, %v, want one", projects, err)
	}
	return projects[0].ID
}

func TestModifiedShowsOnlyWhatTheFileSets(t *testing.T) {
	u := startUI(t)
	_, body, _ := settingsOf(t, u, "p", "[commit]\nlanguage = \"de\"\n", "?modified=1")
	if !strings.Contains(body, "<code>commit.language</code>") || strings.Contains(body, "<code>commit.threshold</code>") {
		t.Error("?modified=1 must keep the set key and drop the defaults")
	}
	if !strings.Contains(body, " checked>") {
		t.Error("the filter box does not show it is on")
	}
}

func TestAProjectPageShowsAMachineWideRuleAsGlobalAndReadOnly(t *testing.T) {
	u := startUI(t)
	writeGlobal(t, u, "[[policy.paths.rules]]\nmatch = \"private/**\"\nreason = \"machine-wide\"\n")
	_, body, _ := settingsOf(t, u, "p", "[[policy.paths.rules]]\nmatch = \"own/**\"\nreason = \"mine\"\n[[policy.commands.rules]]\nregex = \"rm -rf\"\nreason = \"no\"\n", "")
	for _, want := range []string{
		"<code>private/**</code> — machine-wide", "[global, read-only]",
		"<code>own/**</code> — mine", "[project]",
		"<code>rm -rf</code> — no",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the project page lacks %q", want)
		}
	}
}

func TestARuleReasonIsShownEscaped(t *testing.T) {
	u := startUI(t)
	_, body, _ := settingsOf(t, u, "p", "[[policy.paths.rules]]\nmatch = \"x\"\nreason = \"<script>alert(1)</script>\"\n", "")
	if strings.Contains(body, "<script>") || !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Error("a reason is markup in the page")
	}
}

func TestABrokenProjectConfigIsShownWithItsFileAndALinkToTheRawText(t *testing.T) {
	u := startUI(t)
	id, body, status := settingsOf(t, u, "p", "[commit\nlanguage = \n", "")
	if status != http.StatusOK {
		t.Fatalf("status %d, want 200", status)
	}
	if !strings.Contains(body, ".loomux/config.toml: not valid TOML") || !strings.Contains(body, `href="/settings/`+id+`/raw"`) {
		t.Errorf("the error does not name the file and link to the raw text: %s", body)
	}
	if strings.Contains(body, "<table>") {
		t.Error("keys are listed from a file that does not parse")
	}
}

func TestAConfigThatCannotBeReadIsShownAsAnError(t *testing.T) {
	u := startUI(t)
	root := makeProject(t, "p", "")
	if err := os.Mkdir(config.ManifestPath(root), 0o755); err != nil {
		t.Fatal(err)
	}
	remember(t, u.stateDir, root)
	cookie := u.session()
	status, body := u.get(cookie, "/settings/"+projectID(root))
	if status != http.StatusOK || !strings.Contains(body, `class="error"`) {
		t.Fatalf("status %d, body %s", status, body)
	}
}

func TestAnUnknownIDIsNotFound(t *testing.T) {
	u := startUI(t)
	cookie := u.session()
	for _, path := range []string{"/settings/0123456789ab", "/settings/GLOBAL"} {
		if status, _ := u.get(cookie, path); status != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", path, status)
		}
	}
}

func TestSettingsNeedASession(t *testing.T) {
	u := startUI(t)
	if resp := u.do(req{path: "/settings/global"}); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("without a session = %d, want 303", resp.StatusCode)
	}
}

func TestAModuleThatIsSwitchedOffSaysItsKeysDoNothing(t *testing.T) {
	u := startUI(t)
	_, body, _ := settingsOf(t, u, "p", "[modules]\nbrain = false\n", "")
	if n := strings.Count(body, "module off — these keys have no effect"); n != 1 {
		t.Fatalf("the note stands %d times, want once, over the brain group", n)
	}
	brain := body[strings.Index(body, "<h2>brain</h2>"):]
	if !strings.HasPrefix(strings.TrimPrefix(brain, "<h2>brain</h2>"), "\n<p class=\"note\">module off") || !strings.Contains(brain, "<code>area.scope</code>") {
		t.Error("the note is not over the brain group, which keeps its keys")
	}
}

func TestAModulesTableThatDoesNotReadLeavesEveryModuleOn(t *testing.T) {
	u := startUI(t)
	_, body, status := settingsOf(t, u, "p", "[modules]\ngraf = false\n", "")
	if status != http.StatusOK || strings.Contains(body, "module off") {
		t.Errorf("status %d, and a note over a file the page cannot judge", status)
	}
}

func TestTablesAndOtherListsAreShownNotEdited(t *testing.T) {
	u := startUI(t)
	_, body, _ := settingsOf(t, u, "p", "[[commit.allow]]\nregex = \"^wip\"\nreason = \"drafts\"\n", "")
	if !strings.Contains(body, "edit it in the raw text") {
		t.Error("a table does not point to the raw text")
	}
	if !strings.Contains(body, "reason = &#34;drafts&#34;") || !strings.Contains(body, "regex = &#34;^wip&#34;") {
		t.Errorf("commit.allow lists no element: %s", body)
	}
	// verify.profiles is filled by the presets and shows their default.
	if !strings.Contains(body, "precommit") {
		t.Error("a preset row shows no value")
	}
}

func TestARuleListThatDoesNotReadSaysSoInItsRow(t *testing.T) {
	u := startUI(t)
	writeGlobal(t, u, "[[policy.paths.rules]]\nmatch = \"x\"\n")
	_, body, status := settingsOf(t, u, "p", "[modules]\n", "")
	if status != http.StatusOK || !strings.Contains(body, "needs a reason") {
		t.Errorf("a broken machine-wide rule is not reported: %d", status)
	}
	// The same file is the global page's own.
	cookie := u.session()
	if _, global := u.get(cookie, "/settings/global"); !strings.Contains(global, "needs a reason") {
		t.Error("the global page does not report its own broken rule")
	}
	if err := os.Remove(filepath.Join(u.stateDir, "config.toml")); err != nil {
		t.Fatal(err)
	}
	root := makeProject(t, "q", "[[policy.commands.rules]]\nregex = \"x\"\n")
	remember(t, u.stateDir, root)
	if _, own := u.get(cookie, "/settings/"+projectID(root)); !strings.Contains(own, "needs a reason") {
		t.Error("the project page does not report its own broken rule")
	}
}

func TestAMissingProjectIsSaidToBeMissing(t *testing.T) {
	u := startUI(t)
	_, body, status := settingsOf(t, u, "p", "", "")
	if status != http.StatusOK || !strings.Contains(body, "has no .loomux/config.toml any more") {
		t.Errorf("status %d, body %s", status, body)
	}
}

func TestAnUnreadableRegistryNamesNoProject(t *testing.T) {
	u := startUI(t)
	if err := os.WriteFile(filepath.Join(u.stateDir, "registry.toml"), []byte("area = 3"), 0o600); err != nil {
		t.Fatal(err)
	}
	cookie := u.session()
	if status, _ := u.get(cookie, "/settings/0123456789ab"); status != http.StatusNotFound {
		t.Errorf("a project of an unreadable registry = %d, want 404", status)
	}
	if status, body := u.get(cookie, "/settings/global"); status != http.StatusOK || !strings.Contains(body, "Add project") {
		t.Errorf("the machine's settings must stay reachable: %d", status)
	}
}

// session signs in on first use: the code works once.
func (u *ui) session() string {
	if u.cookie == "" {
		u.cookie, _ = u.signIn()
	}
	return u.cookie
}
