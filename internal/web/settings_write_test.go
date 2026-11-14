package web

import (
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xidus90/loomux/internal/config"
)

// writer is a signed-in interface with one project in its world.
type writer struct {
	*ui
	cookie, token, id, path string
}

// writerOf makes a project holding text (not a file when text is empty) and
// signs in.
func writerOf(t *testing.T, text string) *writer {
	t.Helper()
	u := startUI(t)
	root := makeProject(t, "mein projekt ü", text)
	remember(t, u.stateDir, root)
	cookie, token := u.signIn()
	return &writer{ui: u, cookie: cookie, token: token, id: projectID(root), path: config.ManifestPath(root)}
}

// post sends a form with the session's token and returns status and body.
func (w *writer) post(path string, form url.Values) (int, string) {
	w.t.Helper()
	form.Set("csrf", w.token)
	resp := w.do(req{method: http.MethodPost, path: path, cookie: w.cookie, form: form})
	return resp.StatusCode, bodyOf(w.t, resp)
}

func (w *writer) file() string {
	w.t.Helper()
	data, err := os.ReadFile(w.path)
	if err != nil {
		w.t.Fatal(err)
	}
	return string(data)
}

// hidden is the value of a hidden form field, as a browser would send it.
func hidden(t *testing.T, body, name string) string {
	t.Helper()
	m := regexp.MustCompile(`name="` + name + `" value="([^"]*)"`).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no field %q in %s", name, body)
	}
	return html.UnescapeString(m[1])
}

// change previews one key and returns the page.
func (w *writer) change(key, value string) string {
	w.t.Helper()
	status, body := w.post("/settings/"+w.id+"/preview", url.Values{"key": {key}, "value": {value}})
	if status != http.StatusOK {
		w.t.Fatalf("preview = %d: %s", status, body)
	}
	return body
}

// applyOf sends the apply form a preview page carries.
func (w *writer) applyOf(body string) (int, string) {
	w.t.Helper()
	return w.post("/settings/"+w.id+"/apply", url.Values{"before": {hidden(w.t, body, "before")}, "next": {hidden(w.t, body, "next")}})
}

func TestAPreviewShowsTheDiffAndOnlyApplyWritesIt(t *testing.T) {
	text := "# keep this\n[commit]\nthreshold = 5 # and this\n"
	w := writerOf(t, text)
	body := w.change("commit.language", "de")
	if !strings.Contains(body, "&#43; language = &#34;de&#34;") || !strings.Contains(body, ">Apply</button>") {
		t.Fatalf("the preview lacks the diff or the button: %s", body)
	}
	if w.file() != text {
		t.Fatal("a preview wrote the file")
	}
	status, _ := w.applyOf(body)
	if status != http.StatusSeeOther {
		t.Fatalf("apply = %d, want 303", status)
	}
	if want := "# keep this\n[commit]\nthreshold = 5 # and this\nlanguage = \"de\"\n"; w.file() != want {
		t.Fatalf("file = %q, want %q", w.file(), want)
	}
}

func TestTheDefaultIsNeverWrittenAndResetRemovesTheLine(t *testing.T) {
	for name, form := range map[string]url.Values{
		"the default as value": {"key": {"commit.language"}, "value": {"en"}},
		"reset":                {"key": {"commit.language"}, "reset": {"1"}},
	} {
		w := writerOf(t, "[commit]\nlanguage = \"de\"\n")
		status, body := w.post("/settings/"+w.id+"/preview", form)
		if status != http.StatusOK || !strings.Contains(body, "- language = &#34;de&#34;") {
			t.Errorf("%s: %d, no removal in %s", name, status, body)
		}
		if status, _ := w.applyOf(body); status != http.StatusSeeOther {
			t.Errorf("%s: apply = %d", name, status)
		}
		if strings.Contains(w.file(), "language") {
			t.Errorf("%s: the line is still there: %q", name, w.file())
		}
	}
}

func TestAnInvalidValueShowsTheErrorAndNoApplyButton(t *testing.T) {
	for key, value := range map[string]string{"commit.threshold": "abc", "commit.language": "fr", "nonsense.key": "x"} {
		w := writerOf(t, "[commit]\nthreshold = 5\n")
		page := w.change(key, value)
		if !strings.Contains(page, `class="error"`) || strings.Contains(page, `name="next"`) || strings.Contains(page, ">Apply<") {
			t.Errorf("%s=%s: want an error without an apply button: %s", key, value, page)
		}
		if w.file() != "[commit]\nthreshold = 5\n" {
			t.Errorf("%s=%s wrote the file", key, value)
		}
	}
	w := writerOf(t, "")
	if page := w.change("commit.threshold", "abc"); !strings.Contains(page, "&#34;abc&#34; is not a whole number") {
		t.Errorf("the error is not the CLI's: %s", page)
	}
}

func TestABrainKeyWithoutAreaGivesTheErrorOfChange(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	page := w.change("privacy.mode", "local_only")
	if !strings.Contains(page, "belongs to the brain, which reads nothing without [area]") {
		t.Fatalf("no brain error: %s", page)
	}
}

func TestAFileChangedAfterThePreviewIsNotWrittenOver(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	body := w.change("commit.language", "de")
	other := "[commit]\nthreshold = 7\n"
	if err := os.WriteFile(w.path, []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	status, page := w.applyOf(body)
	if status != http.StatusOK || !strings.Contains(page, "the file changed since the preview") {
		t.Fatalf("status %d, page %s", status, page)
	}
	if w.file() != other {
		t.Fatalf("the stale apply wrote: %q", w.file())
	}
	if hidden(t, page, "before") != hashOf(other) {
		t.Error("the new preview is not made against the file as it is")
	}
	// It shows what applying would now do to the other change.
	if !strings.Contains(page, "- threshold = 7") {
		t.Errorf("the new preview does not show the change it would take back: %s", page)
	}
}

func TestAnApplyOfATextThatDoesNotCheckWritesNothing(t *testing.T) {
	text := "[commit]\nthreshold = 5\n"
	w := writerOf(t, text)
	status, page := w.post("/settings/"+w.id+"/apply", url.Values{"before": {hashOf(text)}, "next": {"[commit]\nlanguage = \"fr\"\n"}})
	if status != http.StatusOK || !strings.Contains(page, `class="error"`) || strings.Contains(page, ">Apply<") {
		t.Fatalf("status %d, page %s", status, page)
	}
	if w.file() != text {
		t.Fatal("an unchecked text was written")
	}
}

func TestARawTextFromABrowserKeepsTheLineEndingsOfTheFile(t *testing.T) {
	text := "# c\n[commit]\nthreshold = 5\n"
	w := writerOf(t, text)
	raw := strings.ReplaceAll("# c\n[commit]\nthreshold = 6\n", "\n", "\r\n")
	_, body := w.post("/settings/"+w.id+"/preview", url.Values{"raw": {raw}})
	if strings.Count(body, "\n- ") != 1 || strings.Count(body, "\n&#43; ") != 1 || !strings.Contains(body, "&#43; threshold = 6") {
		t.Fatalf("the diff is not one line: %s", body)
	}
	// A browser sends the hidden text back with CRLF as well.
	next := strings.ReplaceAll(hidden(t, body, "next"), "\n", "\r\n")
	w.post("/settings/"+w.id+"/apply", url.Values{"before": {hidden(t, body, "before")}, "next": {next}})
	if want := "# c\n[commit]\nthreshold = 6\n"; w.file() != want {
		t.Fatalf("file = %q, want %q", w.file(), want)
	}
}

func TestARawTextKeepsABomAndCRLFOfTheFile(t *testing.T) {
	text := "\xef\xbb\xbf# c\r\n[commit]\r\nthreshold = 5\r\n"
	for name, raw := range map[string]string{
		"no bom, lf":   "# c\n[commit]\nthreshold = 6\n",
		"bom and crlf": "\xef\xbb\xbf# c\r\n[commit]\r\nthreshold = 6\r\n",
	} {
		w := writerOf(t, "")
		if err := os.WriteFile(w.path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, page := w.get(w.cookie, "/settings/"+w.id+"/raw"); strings.Contains(page, "\xef\xbb\xbf# c") {
			t.Errorf("%s: the field shows the BOM", name)
		}
		_, body := w.post("/settings/"+w.id+"/preview", url.Values{"raw": {raw}})
		if strings.Count(body, "\n- ") != 1 || strings.Count(body, "\n&#43; ") != 1 {
			t.Fatalf("%s: the diff is not one line: %s", name, body)
		}
		w.applyOf(body)
		if want := "\xef\xbb\xbf# c\r\n[commit]\r\nthreshold = 6\r\n"; w.file() != want {
			t.Errorf("%s: file = %q, want %q", name, w.file(), want)
		}
	}
}

func TestAKeyEditKeepsTheBomAndLineEndings(t *testing.T) {
	text := "\xef\xbb\xbf[commit]\r\nthreshold = 5\r\n"
	w := writerOf(t, "")
	if err := os.WriteFile(w.path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	body := w.change("commit.threshold", "6")
	w.applyOf(body)
	if want := "\xef\xbb\xbf[commit]\r\nthreshold = 6\r\n"; w.file() != want {
		t.Fatalf("file = %q, want %q", w.file(), want)
	}
}

func TestARawTextRepairsAFileThatDoesNotParse(t *testing.T) {
	w := writerOf(t, "[commit\nthreshold = 5\n")
	status, page := w.get(w.cookie, "/settings/"+w.id)
	if status != http.StatusOK || !strings.Contains(page, `class="error"`) || !strings.Contains(page, "/raw\">Open the raw text") {
		t.Fatalf("the settings page of a broken file: %d %s", status, page)
	}
	status, page = w.get(w.cookie, "/settings/"+w.id+"/raw")
	if status != http.StatusOK || !strings.Contains(page, "[commit\nthreshold = 5") {
		t.Fatalf("the raw page of a broken file: %d %s", status, page)
	}
	_, body := w.post("/settings/"+w.id+"/preview", url.Values{"raw": {"[commit]\nthreshold = 5\n"}})
	if !strings.Contains(body, "&#43; [commit]") {
		t.Fatalf("no repair in the preview: %s", body)
	}
	w.applyOf(body)
	if want := "[commit]\nthreshold = 5\n"; w.file() != want {
		t.Fatalf("file = %q, want %q", w.file(), want)
	}
	// A raw text that does not check is refused.
	_, bad := w.post("/settings/"+w.id+"/preview", url.Values{"raw": {"[commit\n"}})
	if !strings.Contains(bad, `class="error"`) || strings.Contains(bad, ">Apply<") {
		t.Errorf("a broken raw text was offered: %s", bad)
	}
}

func TestAnUnchangedRawTextHasNothingToChange(t *testing.T) {
	w := writerOf(t, "\n[commit]\nthreshold = 5\n")
	_, page := w.get(w.cookie, "/settings/"+w.id+"/raw")
	// The first newline after the tag is the parser's, the second the file's.
	if !strings.Contains(page, ">\n\n[commit]") {
		t.Fatalf("a leading blank line is lost: %s", page)
	}
	_, body := w.post("/settings/"+w.id+"/preview", url.Values{"raw": {"\n[commit]\nthreshold = 5\n"}})
	if !strings.Contains(body, "nothing to change") || strings.Contains(body, ">Apply<") {
		t.Fatalf("an empty diff: %s", body)
	}
}

func TestAnEmptyRawTextIsARequestToEmptyTheFile(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	_, body := w.post("/settings/"+w.id+"/preview", url.Values{"raw": {""}})
	if !strings.Contains(body, "- threshold = 5") {
		t.Fatalf("emptying is not shown: %s", body)
	}
	w.applyOf(body)
	if w.file() != "" {
		t.Fatalf("file = %q, want empty", w.file())
	}
}

func TestTheGlobalFileIsCreatedWithLFAndKeepsTheLoopbackGuard(t *testing.T) {
	u := startUI(t)
	cookie, token := u.signIn()
	w := &writer{ui: u, cookie: cookie, token: token, id: globalID, path: filepath.Join(u.stateDir, "config.toml")}

	bad := w.change("model.endpoint", "http://example.com:11434")
	if !strings.Contains(bad, `class="error"`) || !strings.Contains(bad, "loopback") || strings.Contains(bad, ">Apply<") {
		t.Fatalf("a non-loopback endpoint was not refused: %s", bad)
	}
	if _, err := os.Stat(w.path); err == nil {
		t.Fatal("the refused endpoint created the file")
	}

	body := w.change("model.name", "tiny")
	w.applyOf(body)
	if want := "[model]\nname = \"tiny\"\n"; w.file() != want {
		t.Fatalf("file = %q, want %q", w.file(), want)
	}
}

func TestTheWritingRoutesAnswerWhatTheyCannotReach(t *testing.T) {
	w := writerOf(t, "")
	for _, path := range []string{"/preview", "/apply"} {
		if status, _ := w.post("/settings/nonesuch"+path, url.Values{}); status != http.StatusNotFound {
			t.Errorf("POST %s on an unknown id = %d, want 404", path, status)
		}
	}
	if status, _ := w.get(w.cookie, "/settings/nonesuch/raw"); status != http.StatusNotFound {
		t.Errorf("raw on an unknown id = %d, want 404", status)
	}
	// A directory where the file belongs cannot be read.
	if err := os.MkdirAll(w.path, 0o755); err != nil {
		t.Fatal(err)
	}
	if status, _ := w.post("/settings/"+w.id+"/preview", url.Values{"key": {"commit.language"}, "value": {"de"}}); status != http.StatusInternalServerError {
		t.Errorf("an unreadable file = %d, want 500", status)
	}
}

func TestTheSettingsPageHasAFormForEveryKeyThatTakesOne(t *testing.T) {
	w := writerOf(t, "[commit]\nlanguage = \"de\"\nthreshold = 5\n[agent.roles]\nreviewer = \"r\"\n")
	_, page := w.get(w.cookie, "/settings/"+w.id)
	for _, want := range []string{
		`action="/settings/` + w.id + `/preview"`, `name="csrf" value="`,
		`name="key" value="commit.language"`, `<option selected>de</option>`,
		`name="key" value="commit.conventional"`, `<option selected>true</option>`,
		`name="value" value="5"`, `name="reset" value="1"`,
		`name="key" value="agent.roles.reviewer"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the settings page lacks %q", want)
		}
	}
	for _, not := range []string{`name="key" value="policy.paths.rules"`, `name="key" value="verify.profiles"`, `value="agent.roles.*"`} {
		if strings.Contains(page, not) {
			t.Errorf("the settings page has a form for %q", not)
		}
	}
}
