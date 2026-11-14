package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/extract/all"
	"github.com/xidus90/loomux/internal/code/freshness"
	"github.com/xidus90/loomux/internal/code/sourceset"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/config/target"
	"github.com/xidus90/loomux/internal/flow/journal"
	"github.com/xidus90/loomux/internal/flow/runs"
	"github.com/xidus90/loomux/internal/hosts"
	"github.com/xidus90/loomux/internal/selfupdate"
	"github.com/xidus90/loomux/internal/setup"
	"github.com/xidus90/loomux/internal/setup/hostfile"
)

// dash is a signed-in interface and its form token.
type dash struct {
	*ui
	cookie, token string
}

func newDash(t *testing.T, adjust func(*Options)) *dash {
	t.Helper()
	u := startUIWith(t, adjust)
	cookie, token := u.signIn()
	return &dash{ui: u, cookie: cookie, token: token}
}

func (d *dash) home() (int, string) { return d.get(d.cookie, "/") }

func (d *dash) post(path string, form url.Values) (int, string) {
	d.t.Helper()
	form.Set("csrf", d.token)
	resp := d.do(req{method: http.MethodPost, path: path, cookie: d.cookie, form: form})
	return resp.StatusCode, bodyOf(d.t, resp)
}

func (d *dash) arrange(tile, action string) int {
	d.t.Helper()
	status, _ := d.post("/dashboard/edit", url.Values{"tile": {tile}, "action": {action}})
	return status
}

func (d *dash) prefs() Prefs {
	d.t.Helper()
	p, err := ReadPrefs(d.stateDir)
	if err != nil {
		d.t.Fatal(err)
	}
	return p
}

// shown is the ids of the tiles on the page, in page order.
func shown(page string) []string {
	var ids []string
	for _, t := range tiles() {
		if strings.Contains(page, `<section id="tile-`+t.ID+`">`) {
			ids = append(ids, t.ID)
		}
	}
	slices.SortFunc(ids, func(a, b string) int {
		return strings.Index(page, `id="tile-`+a+`"`) - strings.Index(page, `id="tile-`+b+`"`)
	})
	return ids
}

func ids(list []Tile) []string {
	var out []string
	for _, t := range list {
		out = append(out, t.ID)
	}
	return out
}

func writeUIToml(t *testing.T, stateDir, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(stateDir, "ui.toml"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestWithoutUITomlTheDashboardShowsEveryTileInTheFixedOrder(t *testing.T) {
	d := newDash(t, nil)
	status, page := d.home()
	want := []string{"projects", "proposals", "flow-runs", "review-cases", "index", "version"}
	if status != http.StatusOK || !slices.Equal(shown(page), want) || !slices.Equal(ids(tiles()), want) {
		t.Fatalf("home = %d, tiles %v, want %v", status, shown(page), want)
	}
	if strings.Contains(page, "ui.toml could not be read") {
		t.Error("a missing ui.toml was reported as broken")
	}
}

func TestHidingATileRemovesItAndShowingPutsItAtTheEnd(t *testing.T) {
	d := newDash(t, nil)
	if status := d.arrange("proposals", "hide"); status != http.StatusSeeOther {
		t.Fatalf("hide = %d", status)
	}
	_, page := d.home()
	if got := strings.Join(shown(page), ","); got != "projects,flow-runs,review-cases,index,version" {
		t.Fatalf("after hide: %s", got)
	}
	text, err := os.ReadFile(prefsPath(d.stateDir))
	if err != nil {
		t.Fatalf("hiding a tile wrote no ui.toml: %v", err)
	}
	if !strings.Contains(string(text), "tiles") || strings.Contains(string(text), "proposals") {
		t.Errorf("ui.toml does not hold the arrangement without the hidden tile: %s", text)
	}
	d.arrange("proposals", "show")
	_, page = d.home()
	if got := strings.Join(shown(page), ","); got != "projects,flow-runs,review-cases,index,version,proposals" {
		t.Fatalf("after show: %s", got)
	}
}

func TestUpAndDownMoveATileAndDoNothingAtTheEnds(t *testing.T) {
	d := newDash(t, nil)
	// At the ends, from the defaults: nothing is written at all.
	for _, c := range [][2]string{{"projects", "up"}, {"version", "down"}, {"index", "show"}, {"proposals", "show"}} {
		if status := d.arrange(c[0], c[1]); status != http.StatusSeeOther {
			t.Fatalf("%v = %d", c, status)
		}
	}
	if _, err := os.Stat(prefsPath(d.stateDir)); !os.IsNotExist(err) {
		t.Fatalf("a change that changed nothing wrote ui.toml: %v", err)
	}
	d.arrange("version", "up")
	d.arrange("projects", "down")
	want := []string{"proposals", "projects", "flow-runs", "review-cases", "version", "index"}
	if got := d.prefs().Tiles; !slices.Equal(got, want) {
		t.Fatalf("tiles = %v, want %v", got, want)
	}
	// A hidden tile has no place to move in.
	d.arrange("index", "hide")
	before := d.prefs().Tiles
	d.arrange("index", "up")
	d.arrange("index", "down")
	d.arrange("index", "hide")
	if got := d.prefs().Tiles; !slices.Equal(got, before) {
		t.Fatalf("moving a hidden tile changed the list: %v", got)
	}
	// The list at its end and at its start, now that it is the user's.
	d.arrange("version", "down")
	d.arrange("proposals", "up")
	if got := d.prefs().Tiles; !slices.Equal(got, before) {
		t.Fatalf("the ends moved: %v", got)
	}
}

func TestAChangeKeepsTheProjectsOfUITomlAndForgetKeepsTheTiles(t *testing.T) {
	d := newDash(t, nil)
	gone := makeProject(t, "gone", "")
	remember(t, d.stateDir, gone)
	d.arrange("index", "hide")
	if got := d.prefs().Projects; len(got) != 1 || got[0] != gone {
		t.Fatalf("projects = %v", got)
	}
	if status, _ := d.post("/projects/"+projectID(gone)+"/forget", url.Values{}); status != http.StatusSeeOther {
		t.Fatalf("forget = %d", status)
	}
	p := d.prefs()
	if len(p.Projects) != 0 || !p.TilesSet || slices.Contains(p.Tiles, "index") {
		t.Fatalf("prefs after forget = %+v", p)
	}
}

func TestAnUnknownTileOrActionIsRefusedAndWritesNothing(t *testing.T) {
	d := newDash(t, nil)
	for _, c := range [][2]string{{"nope", "hide"}, {"projects", "sideways"}, {"", ""}} {
		if status := d.arrange(c[0], c[1]); status != http.StatusBadRequest {
			t.Errorf("%v = %d, want 400", c, status)
		}
	}
	if _, err := os.Stat(prefsPath(d.stateDir)); !os.IsNotExist(err) {
		t.Errorf("a refused change wrote ui.toml: %v", err)
	}
}

func TestAnUnknownOrRepeatedNameInUITomlIsPassedOver(t *testing.T) {
	d := newDash(t, nil)
	writeUIToml(t, d.stateDir, "[dashboard]\ntiles = [\"version\", \"nope\", \"projects\", \"version\"]\n")
	_, page := d.home()
	if got := strings.Join(shown(page), ","); got != "version,projects" {
		t.Fatalf("tiles = %s", got)
	}
	// An empty list is the user's choice: no tile, and a way back.
	writeUIToml(t, d.stateDir, "[dashboard]\ntiles = []\n")
	_, page = d.home()
	if len(shown(page)) != 0 || !strings.Contains(page, "No tile shows") {
		t.Fatalf("an empty list shows %v", shown(page))
	}
	// The edit page lists every tile, shown ones first.
	writeUIToml(t, d.stateDir, "[dashboard]\ntiles = [\"version\", \"projects\"]\n")
	_, edit := d.get(d.cookie, "/dashboard/edit")
	for _, want := range []string{"Version", "Projects", "Config proposals", `value="show"`, `value="hide"`} {
		if !strings.Contains(edit, want) {
			t.Errorf("the edit page lacks %q", want)
		}
	}
	if strings.Index(edit, "Version") > strings.Index(edit, "Config proposals") {
		t.Error("a hidden tile is listed before a shown one")
	}
	// The first shown tile cannot move up and the last cannot move down.
	if strings.Count(edit, `value="up"`) != 1 || strings.Count(edit, `value="down"`) != 1 {
		t.Errorf("up buttons %d, down buttons %d, want one each", strings.Count(edit, `value="up"`), strings.Count(edit, `value="down"`))
	}
}

func TestABrokenUITomlShowsEveryTileAndSaysSoAndRefusesToWrite(t *testing.T) {
	d := newDash(t, nil)
	registered := makeProject(t, "registered", "[modules]\n")
	if err := config.AddArea(d.stateDir, config.Area{Scope: "project/registered", Path: filepath.ToSlash(registered)}); err != nil {
		t.Fatal(err)
	}
	writeUIToml(t, d.stateDir, "projects = [\n")
	status, page := d.home()
	if status != http.StatusOK || len(shown(page)) != 6 || !strings.Contains(page, "ui.toml could not be read: ") {
		t.Fatalf("home = %d, tiles %v", status, shown(page))
	}
	// The file counts as missing: the registered project is still listed,
	// and no tile fails for it.
	tile := page[strings.Index(page, `id="tile-projects"`):strings.Index(page, `id="tile-proposals"`)]
	if !strings.Contains(tile, `href="/settings/`+projectID(registered)+`"`) || strings.Contains(tile, `class="error"`) {
		t.Errorf("a broken ui.toml took the project list with it: %s", tile)
	}
	_, edit := d.get(d.cookie, "/dashboard/edit")
	if !strings.Contains(edit, "ui.toml could not be read: ") {
		t.Error("the edit page does not say the file is broken")
	}
	before, err := os.ReadFile(prefsPath(d.stateDir))
	if err != nil {
		t.Fatal(err)
	}
	status, body := d.post("/dashboard/edit", url.Values{"tile": {"index"}, "action": {"hide"}})
	if status != http.StatusConflict || !strings.Contains(body, "nothing written") {
		t.Fatalf("change on a broken file = %d", status)
	}
	if status, _ := d.post("/projects/"+projectID(registered)+"/forget", url.Values{}); status != http.StatusConflict {
		t.Errorf("forget on a broken file = %d", status)
	}
	after, err := os.ReadFile(prefsPath(d.stateDir))
	if err != nil || string(after) != string(before) {
		t.Errorf("ui.toml changed: %q, %v", after, err)
	}
}

func TestAProjectWithABrokenConfigNamesTheErrorInItsRowAndTheRestStands(t *testing.T) {
	d := newDash(t, nil)
	broken := makeProject(t, "broken", "[modules\n")
	fine := makeProject(t, "fine", "[modules]\n")
	remember(t, d.stateDir, broken, fine)
	status, page := d.home()
	if status != http.StatusOK || len(shown(page)) != 6 {
		t.Fatalf("home = %d, tiles %v", status, shown(page))
	}
	row := rowOfProject(page, "broken")
	if !strings.Contains(row, `class="error"`) || !strings.Contains(row, "config.toml") {
		t.Errorf("the row of the broken project names no error: %s", row)
	}
	fineRow := rowOfProject(page, "fine")
	if strings.Contains(fineRow, `class="error"`) {
		t.Errorf("the other project's row is an error: %s", fineRow)
	}
}

// failing is a tile whose source fails.
func failing() Tile {
	return Tile{ID: "failing", Title: "Failing", Render: func(*server, string) (template.HTML, error) {
		return "", errors.New("the source is gone")
	}}
}

func TestATileWhoseRenderFailsShowsTheErrorInItsFrameAndTheOthersStand(t *testing.T) {
	s := &server{version: "1.0", options: Options{StateDir: t.TempDir()}}
	ok := Tile{ID: "ok", Title: "Fine", Render: func(_ *server, csrf string) (template.HTML, error) {
		return template.HTML("<p>fine body " + csrf + "</p>"), nil
	}}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), sessionKey{}, &session{id: "s", csrf: "tok"}))
	w := httptest.NewRecorder()
	s.showDashboard(w, r, []Tile{failing(), ok})
	body := w.Body.String()
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	frame := body[strings.Index(body, `id="tile-failing"`):strings.Index(body, `id="tile-ok"`)]
	if !strings.Contains(frame, "the source is gone") || !strings.Contains(body, "<p>fine body tok</p>") {
		t.Errorf("page = %s", body)
	}
}

// waiting puts a run at a gate into a project.
func waiting(t *testing.T, root, id, question string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(runs.Dir)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runs.WriteMarker(runs.MarkerPath(root, id), runs.Marker{Flow: "dev-cycle", Origin: "bundled"}); err != nil {
		t.Fatal(err)
	}
	entry := journal.Entry{Node: "approve", Kind: "gate", InputHash: "h", Delta: map[string]any{}, Outcome: "paused", Detail: &question}
	if err := journal.Append(runs.JournalPath(root, id), entry); err != nil {
		t.Fatal(err)
	}
}

func TestAWaitingRunAppearsWithItsNumberAndProjectAndItsQuestionEscaped(t *testing.T) {
	d := newDash(t, nil)
	root := makeProject(t, "flowy", "[modules]\n")
	remember(t, d.stateDir, root)
	waiting(t, root, "0007", "Ship <script>alert(1)</script>?\n")
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-flow-runs"`):strings.Index(page, `id="tile-review-cases"`)]
	for _, want := range []string{"0007", "dev-cycle", ">flowy<", "<code>approve</code>", "Ship &lt;script&gt;alert(1)&lt;/script&gt;?"} {
		if !strings.Contains(tile, want) {
			t.Errorf("the tile lacks %q: %s", want, tile)
		}
	}
	if strings.Contains(page, "<script>alert(1)") {
		t.Error("a question reached the page unescaped")
	}
}

func TestTheFlowRunsTileSaysWhatItCannotRead(t *testing.T) {
	d := newDash(t, nil)
	root := makeProject(t, "flowy", "[modules]\n")
	remember(t, d.stateDir, root)
	waiting(t, root, "0001", "ok?")
	// A journal that does not read, and a state folder that is a file.
	if err := os.WriteFile(runs.JournalPath(root, "0002"), []byte("{not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	other := makeProject(t, "nofolder", "[modules]\n")
	remember(t, d.stateDir, other)
	if err := os.MkdirAll(filepath.Join(other, ".loomux", "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, filepath.FromSlash(runs.Dir)), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-flow-runs"`):strings.Index(page, `id="tile-review-cases"`)]
	for _, want := range []string{"0001", "0002.jsonl", "cannot be read as a folder of runs"} {
		if !strings.Contains(tile, want) {
			t.Errorf("the tile lacks %q: %s", want, tile)
		}
	}
}

func TestAnOpenProposalAppearsWithALinkToTheSettingsPage(t *testing.T) {
	d := newDash(t, nil)
	root := makeProject(t, "prop", "[modules]\n")
	remember(t, d.stateDir, root)
	if _, err := target.Project(root).Store(time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), target.Proposal{Op: "set", Key: "commit.language", Input: "de"}); err != nil {
		t.Fatal(err)
	}
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-proposals"`):strings.Index(page, `id="tile-flow-runs"`)]
	if !strings.Contains(tile, `<a href="/settings/`+projectID(root)+`">prop</a>`) || !strings.Contains(tile, "1 open") {
		t.Errorf("tile = %s", tile)
	}
	// Without a proposal anywhere the tile says so.
	other := newDash(t, nil)
	_, page = other.home()
	if !strings.Contains(page, "No open proposals.") {
		t.Error("an empty tile says nothing")
	}
}

func TestTheProposalsTileMarksAProposalFileThatDoesNotRead(t *testing.T) {
	d := newDash(t, nil)
	root := makeProject(t, "prop", "[modules]\n")
	remember(t, d.stateDir, root)
	if _, err := target.Project(root).Store(time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), target.Proposal{Op: "set", Key: "commit.language", Input: "de"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, ".loomux", "state", "config", "proposals")
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-proposals"`):strings.Index(page, `id="tile-flow-runs"`)]
	if !strings.Contains(tile, "2 open, 1 not readable") {
		t.Errorf("tile = %s", tile)
	}
}

func TestTheProposalsTileNamesATargetItCannotList(t *testing.T) {
	d := newDash(t, nil)
	unlistable(t, filepath.Join(d.stateDir, "config", "proposals"))
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-proposals"`):strings.Index(page, `id="tile-flow-runs"`)]
	if !strings.Contains(tile, `href="/settings/global"`) || !strings.Contains(tile, `class="error"`) {
		t.Errorf("tile = %s", tile)
	}
}

func TestForgetRemovesTheMissingProjectFromUITomlAndNothingElse(t *testing.T) {
	d := newDash(t, nil)
	gone := makeProject(t, "gone", "")
	keep := makeProject(t, "keep", "[modules]\n")
	remember(t, d.stateDir, gone, keep)
	_, page := d.home()
	if !strings.Contains(page, "/projects/"+projectID(gone)+"/forget") || strings.Contains(page, "/projects/"+projectID(keep)+"/forget") {
		t.Fatalf("the Forget button is on the wrong rows: %s", page)
	}
	if status, _ := d.post("/projects/nobody/forget", url.Values{}); status != http.StatusNotFound {
		t.Errorf("forget of an unknown id = %d", status)
	}
	if status, _ := d.post("/projects/"+projectID(gone)+"/forget", url.Values{}); status != http.StatusSeeOther {
		t.Fatalf("forget = %d", status)
	}
	if got := d.prefs().Projects; len(got) != 1 || got[0] != keep {
		t.Fatalf("projects = %v", got)
	}
	// A forget needs the token like any write.
	resp := d.do(req{method: http.MethodPost, path: "/projects/" + projectID(keep) + "/forget", cookie: d.cookie, form: url.Values{}})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("forget without a token = %d", resp.StatusCode)
	}
}

func TestAnArrangementAndAForgetNeedAUITomlThatReadsAndARegistryThatDoes(t *testing.T) {
	d := newDash(t, nil)
	gone := makeProject(t, "gone", "")
	remember(t, d.stateDir, gone)
	if err := os.WriteFile(filepath.Join(d.stateDir, "registry.toml"), []byte("[[area]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if status, _ := d.post("/projects/"+projectID(gone)+"/forget", url.Values{}); status != http.StatusConflict {
		t.Errorf("forget with a broken registry = %d, want 409", status)
	}
	if got := d.prefs().Projects; len(got) != 1 {
		t.Errorf("projects = %v, want the one still there", got)
	}
	// A ui.toml that is only a folder does not read either.
	if err := os.Remove(prefsPath(d.stateDir)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(prefsPath(d.stateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if status, _ := d.post("/projects/abc/forget", url.Values{}); status != http.StatusConflict {
		t.Errorf("forget = %d, want 409", status)
	}
	if status, _ := d.post("/dashboard/edit", url.Values{"tile": {"index"}, "action": {"hide"}}); status != http.StatusConflict {
		t.Errorf("change = %d, want 409", status)
	}
}

func TestAWriteThatFailsIsSaidAndNothingIsClaimed(t *testing.T) {
	d := newDash(t, nil)
	gone := makeProject(t, "gone", "")
	remember(t, d.stateDir, gone)
	saved := writePrefs
	writePrefs = func(string, Prefs) error { return errors.New("disk full") }
	t.Cleanup(func() { writePrefs = saved })
	status, body := d.post("/dashboard/edit", url.Values{"tile": {"version"}, "action": {"hide"}})
	if status != http.StatusInternalServerError || !strings.Contains(body, "nothing written: disk full") {
		t.Errorf("change = %d, %s", status, body)
	}
	status, body = d.post("/projects/"+projectID(gone)+"/forget", url.Values{})
	if status != http.StatusInternalServerError || !strings.Contains(body, "nothing written: disk full") {
		t.Errorf("forget = %d, %s", status, body)
	}
}

// makeReviewCentre registers an area that declares the vault's review centre
// and returns the directory the cases go under.
func makeReviewCentre(t *testing.T, stateDir string) string {
	t.Helper()
	area := filepath.Join(t.TempDir(), "vault")
	declaration := "[area]\nscope = \"project/one\"\n\n[layout]\nwiki = \"wiki\"\nreview = \"review\"\n\n[index]\ninclude = [\"**/*.md\"]\n"
	if err := os.MkdirAll(filepath.Join(area, ".loomux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.ManifestPath(area), []byte(declaration), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteRegistry(stateDir, []config.Area{{Scope: "project/one", Path: filepath.ToSlash(area)}}); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(area, "review")
}

// recordGraph writes the fingerprint of the project as a graph build leaves it.
func recordGraph(t *testing.T, root string) {
	t.Helper()
	files, err := sourceset.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	hashes := map[string]string{}
	for _, f := range files {
		data, err := os.ReadFile(f.Abs)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		hashes[f.Rel] = hex.EncodeToString(sum[:])
	}
	if err := freshness.Write(root, all.Version(), files, hashes); err != nil {
		t.Fatal(err)
	}
}

// failWalks makes the walk of every source tree fail for the length of the
// test.
func failWalks(t *testing.T) {
	t.Helper()
	saved := sourceset.Stat
	sourceset.Stat = func(string) ([]sourceset.SourceFile, error) { return nil, errors.New("the walk failed") }
	t.Cleanup(func() { sourceset.Stat = saved })
}

func TestTheProjectsTileSaysWhetherTheHookEntriesStand(t *testing.T) {
	var facts func(root string) (setup.Facts, error)
	d := newDash(t, func(o *Options) {
		o.Gather = func(root string) (setup.Facts, error) { return facts(root) }
	})
	bare := makeProject(t, "bare", "[modules]\n")
	hooked := makeProject(t, "hooked", "[modules]\n")
	failed := makeProject(t, "failed", "[modules]\n")
	remember(t, d.stateDir, bare, hooked, failed)
	merged, err := hostfile.Merge(hosts.HostClaude, nil, hostfile.Entries(hosts.HostClaude, hostfile.Canonical))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(hooked, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooked, ".claude", "settings.json"), merged.Merged, 0o644); err != nil {
		t.Fatal(err)
	}
	facts = func(root string) (setup.Facts, error) {
		if root == failed {
			return setup.Facts{}, errors.New("git is not there")
		}
		return setup.Facts{Root: root, Hosts: []hosts.Host{hosts.HostClaude}, Binary: hostfile.Canonical, BinaryThere: root == hooked}, nil
	}
	_, page := d.home()
	if !strings.Contains(rowOfProject(page, "bare"), "binary missing") || strings.Contains(rowOfProject(page, "hooked"), "binary") || !strings.Contains(rowOfProject(page, "bare"), "missing") || !strings.Contains(rowOfProject(page, "hooked"), "set up") || !strings.Contains(rowOfProject(page, "failed"), "git is not there") {
		t.Errorf("rows: %s | %s | %s", rowOfProject(page, "bare"), rowOfProject(page, "hooked"), rowOfProject(page, "failed"))
	}
}

func TestHooksStandReadsEveryHostAndRefusesWhatItCannotRead(t *testing.T) {
	root := t.TempDir()
	f := setup.Facts{Root: root, Hosts: []hosts.Host{hosts.HostClaude}, Binary: hostfile.Canonical}
	if stand, err := hooksStand(f); err != nil || stand {
		t.Fatalf("an empty project = %v, %v", stand, err)
	}
	// Nothing to look at is no hook to miss.
	if stand, err := hooksStand(setup.Facts{Root: root}); err != nil || !stand {
		t.Fatalf("no host = %v, %v", stand, err)
	}
	settings := filepath.Join(root, ".claude", "settings.json")
	if err := os.MkdirAll(settings, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := hooksStand(f); err == nil {
		t.Error("a settings file that is a folder was read")
	}
	if err := os.Remove(settings); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := hooksStand(f); err == nil {
		t.Error("a settings file that is no object was merged")
	}
}

func TestTheReviewCasesTileListsTheOpenCasesAndNamesTheBrokenOnes(t *testing.T) {
	d := newDash(t, nil)
	_, page := d.home()
	if !strings.Contains(page, "No area declares a review centre.") {
		t.Error("a machine without a review centre says nothing")
	}
	review := makeReviewCentre(t, d.stateDir)
	_, page = d.home()
	if !strings.Contains(page, "No open cases.") {
		t.Errorf("an empty review centre: %s", page)
	}
	caseText := "id = \"x\"\narea = \"project/one\"\ntarget = \"a.md\"\ntarget_hash = \"aa\"\n" +
		"state = \"due\"\ntrigger = \"source_change\"\nweight = \"change\"\nmanual = true\ncreated = 2026-09-03T12:00:00Z\n"
	for name, text := range map[string]string{"alpha": caseText, "broken": "not toml ["} {
		dir := filepath.Join(review, "project-one", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "case.toml"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, page = d.home()
	tile := page[strings.Index(page, `id="tile-review-cases"`):strings.Index(page, `id="tile-index"`)]
	for _, want := range []string{"<td>alpha</td>", "a.md", "due (manual)", "Unreadable case: "} {
		if !strings.Contains(tile, want) {
			t.Errorf("the tile lacks %q: %s", want, tile)
		}
	}
}

func TestTheReviewCasesTileFailsInItsFrameOnABrokenRegistry(t *testing.T) {
	d := newDash(t, nil)
	if err := os.WriteFile(filepath.Join(d.stateDir, "registry.toml"), []byte("[[area]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, page := d.home()
	tile := page[strings.Index(page, `id="tile-review-cases"`):strings.Index(page, `id="tile-index"`)]
	if status != http.StatusOK || !strings.Contains(tile, `class="error"`) {
		t.Errorf("home = %d, tile = %s", status, tile)
	}
}

func TestTheIndexTileShowsTheGraphOfEachProjectAndTheSearchState(t *testing.T) {
	d := newDash(t, func(o *Options) {
		o.SearchStatus = func(stateDir string) (string, error) { return "last reconcile: never <b>\n", nil }
	})
	fresh := makeProject(t, "fresh", "[modules]\n")
	stale := makeProject(t, "stale", "[modules]\n")
	unbuilt := makeProject(t, "unbuilt", "[modules]\n")
	remember(t, d.stateDir, fresh, stale, unbuilt)
	for _, root := range []string{fresh, stale} {
		if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		recordGraph(t, root)
	}
	if err := os.WriteFile(filepath.Join(stale, "b.go"), []byte("package b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-index"`):strings.Index(page, `id="tile-version"`)]
	for _, want := range []string{">fresh</a>: fresh", ">stale</a>: 1 files changed since the last build", ">unbuilt</a>: not built yet", "last reconcile: never &lt;b&gt;"} {
		if !strings.Contains(tile, want) {
			t.Errorf("the tile lacks %q: %s", want, tile)
		}
	}
}

func TestTheIndexTileSaysWhatItCannotAsk(t *testing.T) {
	d := newDash(t, nil)
	root := makeProject(t, "walk", "[modules]\n")
	remember(t, d.stateDir, root)
	failWalks(t)
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-index"`):strings.Index(page, `id="tile-version"`)]
	for _, want := range []string{"the walk failed", "no way to ask the search"} {
		if !strings.Contains(tile, want) {
			t.Errorf("the tile lacks %q: %s", want, tile)
		}
	}
	failing := newDash(t, func(o *Options) {
		o.SearchStatus = func(string) (string, error) { return "", errors.New("qmd is not installed") }
	})
	_, page = failing.home()
	if !strings.Contains(page, "qmd is not installed") {
		t.Error("a search that fails is not named")
	}
}

func TestTheVersionTileShowsVersionChannelAndTheLastUpdateCheck(t *testing.T) {
	d := newDash(t, nil)
	_, page := d.home()
	tile := page[strings.Index(page, `id="tile-version"`):]
	for _, want := range []string{"loomux 9.9.9, stable channel", "no update check has run yet"} {
		if !strings.Contains(tile, want) {
			t.Errorf("the tile lacks %q: %s", want, tile)
		}
	}
	checked := time.Date(2026, 11, 13, 9, 30, 0, 0, time.UTC)
	for _, c := range []struct {
		status selfupdate.Status
		want   string
	}{
		{selfupdate.Status{Result: "updated", Version: "9.9.10"}, "the last check, 2026-11-13 09:30: updated, 9.9.10"},
		{selfupdate.Status{Result: "current"}, "the last check, 2026-11-13 09:30: current"},
		{selfupdate.Status{Result: "failed", Error: "no network"}, "failed: no network"},
	} {
		c.status.CheckedAt = checked
		if err := selfupdate.WriteStatus(d.stateDir, c.status); err != nil {
			t.Fatal(err)
		}
		if _, page := d.home(); !strings.Contains(page, c.want) {
			t.Errorf("status %+v: the tile lacks %q", c.status, c.want)
		}
	}
	if err := os.WriteFile(filepath.Join(d.stateDir, "channel"), []byte("beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, page := d.home(); !strings.Contains(page, "beta channel") {
		t.Error("the beta channel is not shown")
	}
	if err := os.WriteFile(filepath.Join(d.stateDir, "channel"), []byte("nightly\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d.stateDir, "update.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, page = d.home()
	for _, want := range []string{"stable channel", "not beta", "update.json"} {
		if !strings.Contains(page, want) {
			t.Errorf("a channel and a status that do not read are not named, %q is missing: %s", want, page)
		}
	}
}

func TestTheDashboardIsTheStartPageAndNeedsASession(t *testing.T) {
	u := startUI(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/"}, {http.MethodGet, "/dashboard/edit"}, {http.MethodPost, "/dashboard/edit"}, {http.MethodPost, "/projects/x/forget"},
	} {
		resp := u.do(req{method: c.method, path: c.path, form: url.Values{}})
		// Without a session the answer is the way to sign in, before any
		// form token is asked for.
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
			t.Errorf("%s %s without a session = %d to %q", c.method, c.path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}
}

// rowOfProject is the row of the projects tile that lists a project by name; the
// header's links name it too, so the search starts at the tile.
func rowOfProject(page, name string) string {
	tile := page[strings.Index(page, `id="tile-projects"`):]
	row := tile[strings.Index(tile, ">"+name+"<"):]
	return row[:strings.Index(row, "</tr>")]
}

func TestATileThatNamesNoBlockOfThePageFails(t *testing.T) {
	if body, err := tileHTML("tile_nowhere", nil); err == nil || body != "" {
		t.Fatalf("tileHTML = %q, %v, want an error", body, err)
	}
}

func TestTheBinaryNoteNamesAMissingOrAnOutdatedBinaryOnly(t *testing.T) {
	canonical := setup.Facts{Binary: hostfile.Canonical, BinaryThere: true, CanonicalThere: true, Installed: "3.3.0", Version: "3.3.0"}
	for name, c := range map[string]struct {
		change func(*setup.Facts)
		want   string
	}{
		"current":            {func(*setup.Facts) {}, ""},
		"newer than the run": {func(f *setup.Facts) { f.Installed = "3.4.0" }, ""},
		"missing":            {func(f *setup.Facts) { f.BinaryThere = false }, "binary missing"},
		"older":              {func(f *setup.Facts) { f.Installed = "3.2.0" }, "binary 3.2.0 is older than 3.3.0"},
		"names no version":   {func(f *setup.Facts) { f.Installed = "" }, ""},
		"development build":  {func(f *setup.Facts) { f.Version = "dev-abc123"; f.Installed = "1.0.0" }, ""},
		"checkout binary":    {func(f *setup.Facts) { f.Binary = hostfile.Checkout; f.Installed = "1.0.0" }, ""},
	} {
		f := canonical
		c.change(&f)
		if got := binaryNote(f); got != c.want {
			t.Errorf("%s: binaryNote = %q, want %q", name, got, c.want)
		}
	}
}
