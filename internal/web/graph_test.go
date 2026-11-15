package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/web/graphview"
)

const wikiGraphJSON = `{"scope":"project/p","nodes":[{"id":"docs/wiki/a.md","title":"Alpha","tags":[]}],` +
	`"edges":[],"links":{"total":0,"resolved":0,"dropped":{}},` +
	`"code_refs":[{"from":"docs/wiki/a.md","path":"internal/a/x.go"}]}`

func codeGraphOf(files ...string) *model.Graph {
	g := &model.Graph{Meta: model.Meta{Version: 3, Extractor: "test/1"}}
	for _, f := range files {
		g.Nodes = append(g.Nodes, model.Node{ID: model.NodeID(f), Name: f, Kind: model.KindFile, Path: f, BodyHash: "h"})
	}
	return g
}

// graphWorld is a registered project with a wiki folder; what the test writes
// into it decides which graphs exist.
type graphWorld struct {
	u    *ui
	root string
	id   string
}

func newGraphWorld(t *testing.T, register bool) graphWorld {
	t.Helper()
	u := startUI(t)
	root := makeProject(t, "p", "[modules]\n")
	if register {
		area := config.Area{Scope: "project/p", Path: filepath.ToSlash(root), WikiPath: filepath.ToSlash(filepath.Join(root, "docs", "wiki"))}
		other := config.Area{Scope: "project/other", Path: filepath.ToSlash(t.TempDir())}
		if err := config.WriteRegistry(u.stateDir, []config.Area{other, area}); err != nil {
			t.Fatal(err)
		}
	} else {
		remember(t, u.stateDir, root)
	}
	return graphWorld{u: u, root: root, id: projectID(root)}
}

func (w graphWorld) writeCode(t *testing.T, files ...string) {
	t.Helper()
	if err := store.Write(w.root, codeGraphOf(files...)); err != nil {
		t.Fatal(err)
	}
}

func (w graphWorld) writeWiki(t *testing.T, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(w.root, "graph.json"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

type dataReply struct {
	View     graphview.View `json:"view"`
	Problems []string       `json:"problems"`
}

func (w graphWorld) data(t *testing.T, query string) dataReply {
	t.Helper()
	resp := w.u.do(req{path: "/graph/" + w.id + "/data" + query, cookie: w.u.session()})
	body := bodyOf(t, resp)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("status %d, type %q: %s", resp.StatusCode, resp.Header.Get("Content-Type"), body)
	}
	var out dataReply
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	if !strings.Contains(body, `"problems":[`) {
		t.Errorf("problems is not a list: %s", body)
	}
	return out
}

func (r dataReply) has(id string) bool {
	for _, n := range r.View.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

func (r dataReply) problem(sub string) bool {
	for _, p := range r.Problems {
		if strings.Contains(p, sub) {
			return true
		}
	}
	return false
}

func TestGraphPage_RendersWithoutGraphData(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeCode(t, "internal/a/x.go")
	status, body := w.u.get(w.u.session(), "/graph/"+w.id)
	if status != http.StatusOK {
		t.Fatalf("status %d: %s", status, body)
	}
	if !strings.Contains(body, "<h1>p</h1>") || strings.Contains(body, "nodes") || strings.Contains(body, "x.go") {
		t.Errorf("the page must carry the project's name and no graph data: %s", body)
	}
	if !strings.Contains(body, `<a aria-label="Graph of p" href="/graph/`+w.id+`">Graph</a>`) {
		t.Error("the header lacks the labelled Graph link")
	}
}

func TestGraphData_OpensAutomaticallyOnlyWithoutAnOpenParameter(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeCode(t, "internal/a/x.go", "internal/b/y.go")
	if got := w.data(t, ""); !got.has("dir:internal/a") || got.has("dir:internal") {
		t.Errorf("no open parameter: want internal opened: %+v", got.View.Nodes)
	}
	if got := w.data(t, "?open="); !got.has("dir:internal") || len(got.View.Open) != 0 {
		t.Errorf("an empty open parameter: want the root view: %+v", got.View)
	}
	if got := w.data(t, "?open=&open=internal"); len(got.View.Open) != 1 || got.View.Open[0] != "internal" {
		t.Errorf("empty entries are dropped beside real ones: %+v", got.View.Open)
	}
}

func TestGraphData_ServesTheView(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeCode(t, "internal/a/x.go")
	w.writeWiki(t, wikiGraphJSON)
	got := w.data(t, "?open=internal&open=internal/a")
	if !got.has("doc:docs/wiki/a.md") || !got.has("file:internal/a/x.go") {
		t.Fatalf("nodes of both graphs expected: %+v", got.View.Nodes)
	}
	var ref bool
	for _, e := range got.View.Edges {
		ref = ref || (e.Kind == "ref" && e.Source == "doc:docs/wiki/a.md")
	}
	if !ref || len(got.Problems) != 0 {
		t.Errorf("want a ref edge and no problems: %+v %v", got.View.Edges, got.Problems)
	}
}

func TestGraphData_MissingCodeGraphNamesTheCommand(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeWiki(t, wikiGraphJSON)
	got := w.data(t, "")
	if !got.problem("loomux graph build") || !got.has("doc:docs/wiki/a.md") {
		t.Errorf("want the command and the wiki nodes: %+v", got)
	}
}

func TestGraphData_MissingWikiGraphNamesTheCommand(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeCode(t, "internal/a/x.go")
	got := w.data(t, "")
	if !got.problem("loomux reindex") || len(got.View.Nodes) == 0 {
		t.Errorf("want the command and the code nodes: %+v", got)
	}
}

func TestGraphData_UnregisteredProjectHasNoWiki(t *testing.T) {
	w := newGraphWorld(t, false)
	w.writeCode(t, "internal/a/x.go")
	w.writeWiki(t, wikiGraphJSON)
	got := w.data(t, "")
	if !got.problem("not a registered area") || got.has("doc:docs/wiki/a.md") {
		t.Errorf("an unregistered project has no wiki: %+v", got)
	}
}

func TestGraphData_BrokenFilesAreProblemsNotErrors(t *testing.T) {
	w := newGraphWorld(t, true)
	if err := os.MkdirAll(store.Dir(w.root), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.WiringPath(w.root), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.writeWiki(t, "{")
	got := w.data(t, "")
	if len(got.Problems) != 2 || got.problem("loomux graph build") || got.problem("never indexed") {
		t.Errorf("each broken file is a problem with its own text: %v", got.Problems)
	}
}

func TestGraphData_AreaWithoutAWikiFolderStillServesItsGraph(t *testing.T) {
	w := newGraphWorld(t, true)
	if err := config.WriteRegistry(w.u.stateDir, []config.Area{{Scope: "project/p", Path: filepath.ToSlash(w.root)}}); err != nil {
		t.Fatal(err)
	}
	w.writeWiki(t, wikiGraphJSON)
	if got := w.data(t, ""); !got.has("doc:docs/wiki/a.md") || len(got.Problems) != 1 {
		t.Errorf("the wiki graph is still served, beside the missing code graph: %+v", got)
	}
}

func TestGraphData_RereadsARebuiltCodeGraph(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeCode(t, "internal/a/x.go")
	if got := w.data(t, ""); got.has("file:internal/b/y.go") {
		t.Fatal("y.go is not there yet")
	}
	w.writeCode(t, "internal/a/x.go", "internal/b/y.go")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(store.WiringPath(w.root), later, later); err != nil {
		t.Fatal(err)
	}
	if got := w.data(t, "?open=internal&open=internal/b"); !got.has("file:internal/b/y.go") {
		t.Errorf("the rebuilt graph is not served: %+v", got.View.Nodes)
	}
}

func TestCodeCache_ServesTheCachedGraphWhileTheFileIsUnchanged(t *testing.T) {
	root := t.TempDir()
	if err := store.Write(root, codeGraphOf("a.go")); err != nil {
		t.Fatal(err)
	}
	var c codeCache
	first, err := c.get(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.get(root)
	if err != nil || first != second {
		t.Errorf("an unchanged file must give the same graph: %v", err)
	}
	if _, err := c.get(t.TempDir()); err == nil {
		t.Error("a root without a graph must fail")
	}
}

func TestGraphData_UnknownProjectIs404(t *testing.T) {
	u := startUI(t)
	cookie := u.session()
	for _, path := range []string{"/graph/nope", "/graph/nope/data", "/graph/nope/search?q=x", "/graph/global", "/graph/global/data", "/graph/global/search"} {
		if status, _ := u.get(cookie, path); status != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", path, status)
		}
	}
}

func TestGraphData_NeedsASession(t *testing.T) {
	w := newGraphWorld(t, true)
	for _, path := range []string{"/graph/" + w.id, "/graph/" + w.id + "/data", "/graph/" + w.id + "/search?q=x"} {
		if resp := w.u.do(req{path: path}); resp.StatusCode != http.StatusSeeOther {
			t.Errorf("%s without a session = %d, want 303", path, resp.StatusCode)
		}
	}
}

func TestGraphSearch_ReturnsHits(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeCode(t, "internal/a/x.go")
	w.writeWiki(t, wikiGraphJSON)
	cookie := w.u.session()
	search := func(q string) []graphview.Hit {
		resp := w.u.do(req{path: "/graph/" + w.id + "/search?q=" + url.QueryEscape(q), cookie: cookie})
		var hits []graphview.Hit
		body := bodyOf(t, resp)
		if resp.Header.Get("Content-Type") != "application/json" || json.Unmarshal([]byte(body), &hits) != nil {
			t.Fatalf("not a JSON list: %s", body)
		}
		return hits
	}
	if hits := search("alpha"); len(hits) != 1 || hits[0].ID != "doc:docs/wiki/a.md" {
		t.Errorf("hits for alpha = %+v", hits)
	}
	if hits := search("x.go"); len(hits) == 0 {
		t.Error("no hit for x.go")
	}
	empty := w.u.do(req{path: "/graph/" + w.id + "/search?q=", cookie: cookie})
	if body := bodyOf(t, empty); strings.TrimSpace(body) != "[]" {
		t.Errorf("an empty query answers %q, want []", body)
	}
}

func TestGraphIsLinkedFromTheSettingsPageAndTheIndexTile(t *testing.T) {
	u := startUI(t)
	id, body, _ := settingsOf(t, u, "p", "[modules]\n", "")
	link := `href="/graph/` + id + `"`
	if !strings.Contains(body, link) {
		t.Error("the settings page does not link the graph")
	}
	_, page := u.get(u.session(), "/")
	tile := page[strings.Index(page, `id="tile-index"`):strings.Index(page, `id="tile-version"`)]
	if !strings.Contains(tile, link) {
		t.Errorf("the index tile does not link the graph: %s", tile)
	}
}

func TestGraphScriptIsServed(t *testing.T) {
	w := newGraphWorld(t, true)
	w.writeCode(t, "internal/a/x.go")
	resp := w.u.do(req{path: "/assets/graph.js", cookie: w.u.session()})
	if body := bodyOf(t, resp); resp.StatusCode != http.StatusOK ||
		resp.Header.Get("Content-Type") != "text/javascript; charset=utf-8" || !strings.Contains(body, "ForceGraph") {
		t.Errorf("graph.js = %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	status, page := w.u.get(w.u.session(), "/graph/"+w.id)
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	vendor := strings.Index(page, `<script src="/assets/vendor/force-graph.min.js"></script>`)
	island := strings.Index(page, `<script src="/assets/graph.js"></script>`)
	if vendor < 0 || island < vendor {
		t.Errorf("the page must load the library, then graph.js: %s", page)
	}
	if strings.Count(page, "<script") != 2 || strings.Contains(page, "<script>") || strings.Contains(page, " style=") {
		t.Errorf("the page carries inline script or style: %s", page)
	}
}
