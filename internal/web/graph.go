package web

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/brain/index"
	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/store"
	"github.com/xidus90/loomux/internal/config"
	"github.com/xidus90/loomux/internal/web/graphview"
)

// codeCache keeps the decoded code graph of each project, so that a page's
// requests do not decode a large wiring.json each time. An entry is valid for
// as long as the file's modification time stays what it was.
type codeCache struct {
	mu      sync.Mutex
	entries map[string]cachedCode
}

type cachedCode struct {
	modified time.Time
	graph    *model.Graph
}

// get returns the code graph of root, read again when wiring.json changed. A
// missing file is an error that wraps fs.ErrNotExist, as store.Read has it.
//
// ponytail: one lock over every project's read; per-root locks if a slow
// decode of one project must not hold up the others.
func (c *codeCache) get(root string) (*model.Graph, error) {
	info, err := os.Stat(store.WiringPath(root))
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[root]; ok && e.modified.Equal(info.ModTime()) {
		return e.graph, nil
	}
	g, err := store.Read(root)
	if err != nil {
		return nil, err
	}
	if c.entries == nil {
		c.entries = map[string]cachedCode{}
	}
	c.entries[root] = cachedCode{modified: info.ModTime(), graph: g}
	return g, nil
}

// graphSources is what the endpoints build their answer from: both graphs as far
// as they could be read, and one sentence for each that could not.
type graphSources struct {
	code       *model.Graph
	wiki       *graph.Graph
	wikiPrefix string
	problems   []string
}

// loadGraphs reads the code graph and the wiki graph of a project. A graph
// that is not there or does not read is a problem for the page to show, not a
// failure of the request: the other graph is still served.
func (s *server) loadGraphs(p Project) graphSources {
	d := graphSources{problems: []string{}}
	var err error
	if d.code, err = s.codes.get(p.Root); err != nil {
		d.problems = append(d.problems, codeProblem(err))
	}
	// A project that is not registered has no scope, which no area has. An
	// error from the registry is one that broke since Projects read it, and
	// leaves no area either.
	areas, _ := config.ReadRegistry(s.options.StateDir)
	for _, area := range areas {
		if area.Scope != p.Scope {
			continue
		}
		if d.wiki, err = graph.ReadGraph(area, s.options.StateDir); err != nil {
			d.problems = append(d.problems, wikiProblem(err))
		}
		if prefix := index.OwnWikiPrefix(area); prefix != nil {
			d.wikiPrefix = strings.TrimSuffix(*prefix, "/")
		}
		return d
	}
	d.problems = append(d.problems, "not a registered area: no wiki graph")
	return d
}

func codeProblem(err error) string {
	if errors.Is(err, fs.ErrNotExist) {
		return "no code graph yet: run `loomux graph build`"
	}
	return err.Error()
}

func wikiProblem(err error) string {
	if errors.Is(err, graph.ErrNotIndexed) {
		return "no wiki graph yet: run `loomux reindex`"
	}
	return err.Error()
}

// graphProject finds the project an address names. The machine-wide settings
// have no graph, so `global` is as unknown as any other ID.
func (s *server) graphProject(w http.ResponseWriter, r *http.Request) (Project, bool) {
	id := r.PathValue("id")
	if _, p, ok := s.target(id); ok && id != globalID {
		return p, true
	}
	http.NotFound(w, r)
	return Project{}, false
}

// graphPageView is the data of the graph page: which project, never any graph.
type graphPageView struct {
	ID   string
	Name string
}

// graphPage is the shell the browser draws the graph into; the graph itself
// comes from the data endpoint.
func (s *server) graphPage(w http.ResponseWriter, r *http.Request) {
	if p, ok := s.graphProject(w, r); ok {
		s.page(w, r, "graph", graphPageView{ID: p.ID, Name: p.Name})
	}
}

// graphJSON answers with a JSON body. The value is built from this package's
// own types, so encoding it cannot fail.
func graphJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// graphView answers the view of both graphs with the directories and files
// of the repeated `open` parameter expanded. Without any `open` the view opens
// itself; with one, even an empty `open=`, it shows what is named.
func (s *server) graphView(w http.ResponseWriter, r *http.Request) {
	p, ok := s.graphProject(w, r)
	if !ok {
		return
	}
	d := s.loadGraphs(p)
	// DeleteFunc keeps an absent list nil and a present one non-nil.
	open := slices.DeleteFunc(r.URL.Query()["open"], func(o string) bool { return o == "" })
	graphJSON(w, struct {
		View     graphview.View `json:"view"`
		Problems []string       `json:"problems"`
	}{graphview.Build(d.code, d.wiki, d.wikiPrefix, open, graphview.DefaultLimit), d.problems})
}

// graphSearch answers the search box: the hits for q across both graphs. What
// could not be read is the data endpoint's to report.
func (s *server) graphSearch(w http.ResponseWriter, r *http.Request) {
	p, ok := s.graphProject(w, r)
	if !ok {
		return
	}
	d := s.loadGraphs(p)
	graphJSON(w, graphview.Search(d.code, d.wiki, r.URL.Query().Get("q")))
}
