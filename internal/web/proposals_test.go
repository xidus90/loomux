package web

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/config/target"
)

// root is the project directory of the writer.
func (w *writer) root() string { return filepath.Dir(filepath.Dir(w.path)) }

// propose files a proposal of an agent for the writer's project.
func (w *writer) propose(p target.Proposal) string {
	w.t.Helper()
	id, err := target.Project(w.root()).Store(time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), p)
	if err != nil {
		w.t.Fatal(err)
	}
	return id
}

// open lists the ids of the proposals still open.
func (w *writer) open() []string {
	w.t.Helper()
	ids, err := target.Project(w.root()).IDs()
	if err != nil {
		w.t.Fatal(err)
	}
	return ids
}

// diffOf is the hidden diff field of the apply form of a proposal.
func diffOf(t *testing.T, page, id string) string {
	t.Helper()
	m := regexp.MustCompile(`proposals/` + regexp.QuoteMeta(id) + `/apply">\s*<input type="hidden" name="csrf" value="[^"]*">\s*<input type="hidden" name="diff" value="([^"]*)"`).FindStringSubmatch(page)
	if m == nil {
		t.Fatalf("no apply form for %s in %s", id, page)
	}
	return m[1]
}

func (w *writer) proposal(id, verb string, form url.Values) (int, string) {
	return w.post("/settings/"+w.id+"/proposals/"+url.PathEscape(id)+"/"+verb, form)
}

func TestAProposalStandsOnTopOfThePageWithItsDiffAndApplyingWritesItInOneStep(t *testing.T) {
	text := "# keep\n[commit]\nthreshold = 5\n"
	w := writerOf(t, text)
	id := w.propose(target.Proposal{Op: "set", Key: "commit.language", Input: "de"})
	_, page := w.page()
	for _, want := range []string{"<h2>Proposals</h2>", "set commit.language de", "&#43; language = &#34;de&#34;", id} {
		if !strings.Contains(page, want) {
			t.Errorf("the page lacks %q", want)
		}
	}
	if strings.Index(page, "<h2>Proposals</h2>") > strings.Index(page, "<h2>base</h2>") {
		t.Error("proposals are not above the keys")
	}
	status, _ := w.proposal(id, "apply", url.Values{"diff": {diffOf(t, page, id)}})
	if status != http.StatusSeeOther {
		t.Fatalf("apply = %d", status)
	}
	if got := w.file(); got != text+"language = \"de\"\n" {
		t.Fatalf("file = %q", got)
	}
	if len(w.open()) != 0 {
		t.Fatal("the proposal is still open")
	}
	if _, page := w.page(); strings.Contains(page, "<h2>Proposals</h2>") {
		t.Error("the applied proposal is still listed")
	}
}

func TestRejectingAProposalLeavesTheFileAlone(t *testing.T) {
	text := "[commit]\nthreshold = 5\n"
	w := writerOf(t, text)
	id := w.propose(target.Proposal{Op: "unset", Key: "commit.threshold"})
	if status, _ := w.proposal(id, "reject", url.Values{}); status != http.StatusSeeOther {
		t.Fatalf("reject = %d", status)
	}
	if w.file() != text || len(w.open()) != 0 {
		t.Fatalf("file %q, open %v", w.file(), w.open())
	}
}

func TestAProposalThatNoLongerFitsOffersOnlyRejectAndApplyWritesNothing(t *testing.T) {
	text := "[commit]\nthreshold = 5\n"
	w := writerOf(t, text)
	id := w.propose(target.Proposal{Op: "set", Key: "no.such.key", Input: "x"})
	_, page := w.page()
	if !strings.Contains(page, `class="error"`) || !strings.Contains(page, "/proposals/"+id+"/reject") || strings.Contains(page, "/proposals/"+id+"/apply") {
		t.Fatalf("want an error and only Reject: %s", page)
	}
	// A hand-made request still reaches the write path, and is refused there.
	status, body := w.proposal(id, "apply", url.Values{"diff": {hashOf("")}})
	if status != http.StatusOK || !strings.Contains(body, "unknown key") {
		t.Fatalf("apply = %d: %s", status, body)
	}
	if w.file() != text || len(w.open()) != 1 {
		t.Fatalf("file %q, open %v", w.file(), w.open())
	}
}

func TestAProposalWhoseDiffChangedSinceThePageIsNotApplied(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	id := w.propose(target.Proposal{Op: "set", Key: "commit.language", Input: "de"})
	_, page := w.page()
	shown := diffOf(t, page, id)
	other := "[commit]\nthreshold = 5\nlanguage = \"de\"\n"
	if err := os.WriteFile(w.path, []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	status, body := w.proposal(id, "apply", url.Values{"diff": {shown}})
	if status != http.StatusOK || !strings.Contains(body, "changed since the page was drawn") {
		t.Fatalf("apply = %d: %s", status, body)
	}
	if w.file() != other || len(w.open()) != 1 {
		t.Fatalf("file %q, open %v", w.file(), w.open())
	}
	// What it would change is now nothing, and the page says so.
	if _, page := w.page(); !strings.Contains(page, "nothing to change") {
		t.Error("an already applied proposal is not marked")
	}
	if status, _ := w.proposal(id, "apply", url.Values{"diff": {hashOf("")}}); status != http.StatusSeeOther || len(w.open()) != 0 {
		t.Fatalf("applying what is already so: %d, open %v", status, w.open())
	}
}

func TestAProposalThatIsNotOpenIsNamedAndNoOtherFileIsTouched(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	w.propose(target.Proposal{Op: "unset", Key: "commit.threshold"})
	for _, id := range []string{"nope", "../../../config"} {
		for _, verb := range []string{"apply", "reject"} {
			status, body := w.proposal(id, verb, url.Values{"diff": {hashOf("")}})
			if status != http.StatusOK || !strings.Contains(body, "no open proposal") {
				t.Errorf("%s %s = %d: %s", verb, id, status, body)
			}
		}
	}
	if len(w.open()) != 1 || !strings.Contains(w.file(), "threshold") {
		t.Fatal("something was touched")
	}
	if status, _ := w.post("/settings/nope/proposals/x/reject", url.Values{}); status != http.StatusNotFound {
		t.Fatalf("unknown project = %d", status)
	}
}

func TestAProposalFileThatDoesNotReadIsListedWithOnlyReject(t *testing.T) {
	w := writerOf(t, "")
	dir := filepath.Join(w.root(), ".loomux", "state", "config", "proposals")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, page := w.page()
	if !strings.Contains(page, "broken") || !strings.Contains(page, `class="error"`) || strings.Contains(page, "/proposals/broken/apply") {
		t.Fatalf("want the file named with an error and no apply: %s", page)
	}
	if status, _ := w.proposal("broken", "reject", url.Values{}); status != http.StatusSeeOther || len(w.open()) != 0 {
		t.Fatalf("reject = %d, open %v", status, w.open())
	}
}

func TestTheMachinesProposalsStandOnTheGlobalPage(t *testing.T) {
	u := startUI(t)
	cookie, token := u.signIn()
	if _, err := target.Global(u.stateDir).Store(time.Now(), target.Proposal{Op: "set", Key: "model.enabled", Input: "true"}); err != nil {
		t.Fatal(err)
	}
	if _, page := u.get(cookie, "/settings/global"); !strings.Contains(page, "set model.enabled true") {
		t.Fatalf("the global page lacks the proposal: %s", page)
	}
	ids, err := target.Global(u.stateDir).IDs()
	if err != nil || len(ids) != 1 {
		t.Fatalf("the open proposals: %v, %v", ids, err)
	}
	resp := u.do(req{method: http.MethodPost, path: "/settings/global/proposals/" + ids[0] + "/reject", cookie: cookie, form: url.Values{"csrf": {token}}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("reject = %d", resp.StatusCode)
	}
}

func TestProposalsThatCannotBeListedAreSaidOnThePage(t *testing.T) {
	w := writerOf(t, "")
	dir := filepath.Join(w.root(), ".loomux", "state", "config")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A file where the directory belongs: it exists, and cannot be listed.
	if err := os.WriteFile(filepath.Join(dir, "proposals"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	status, page := w.page()
	if status != http.StatusOK || !strings.Contains(page, `class="error"`) || !strings.Contains(page, "<h2>base</h2>") {
		t.Fatalf("%d: %s", status, page)
	}
}

func TestAnActionOnAFileThatDoesNotParseStillSaysWhyItFailed(t *testing.T) {
	w := writerOf(t, "[commit\n")
	status, body := w.proposal("x", "apply", url.Values{})
	if status != http.StatusOK || !strings.Contains(body, "Open the raw text") {
		t.Fatalf("%d: %s", status, body)
	}
}

// The proposal that is applied is the one the handler read once and compared
// with the form's diff: a proposal file that is swapped after that read is
// never written, and the file is read only once.
func TestApplyingWritesTheProposalItReadOnceNotOneSwappedInAfterwards(t *testing.T) {
	w := writerOf(t, "[commit]\nthreshold = 5\n")
	shown := w.propose(target.Proposal{Op: "set", Key: "commit.language", Input: "de"})
	other := w.propose(target.Proposal{Op: "set", Key: "commit.threshold", Input: "7"})
	_, page := w.page()
	diff := diffOf(t, page, shown)
	dir := filepath.Join(w.root(), ".loomux", "state", "config", "proposals")
	swapped, err := os.ReadFile(filepath.Join(dir, other+".json"))
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	restore := target.SetReadFileForTest(func(name string) ([]byte, error) {
		data, err := os.ReadFile(name)
		if filepath.Base(name) == shown+".json" {
			reads++
			// Whoever reads the file again meets the other proposal.
			if werr := os.WriteFile(name, swapped, 0o644); werr != nil {
				t.Error(werr)
			}
		}
		return data, err
	})
	t.Cleanup(restore)
	if status, body := w.proposal(shown, "apply", url.Values{"diff": {diff}}); status != http.StatusSeeOther {
		t.Fatalf("apply = %d: %s", status, body)
	}
	if got, want := w.file(), "[commit]\nthreshold = 5\nlanguage = \"de\"\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	if reads != 1 {
		t.Errorf("the proposal file was read %d times, want once", reads)
	}
}
