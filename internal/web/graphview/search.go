package graphview

import (
	"path"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/code/model"
)

// MaxHits is the most hits a search answers with.
const MaxHits = 20

// Hit is one search result.
type Hit struct {
	ID    string   `json:"id"` // node ID as Build emits it once Open is applied
	Label string   `json:"label"`
	Kind  string   `json:"kind"`
	Open  []string `json:"open"` // the open list under which ID is visible
}

// ranked is a hit with what orders it.
type ranked struct {
	hit   Hit
	path  string
	exact bool
}

// Search finds the directories, files, symbols and documents whose name,
// path or title contains q, ignoring case. Exact name or title matches come
// first, then the rest by path. An empty q finds nothing.
func Search(code *model.Graph, wiki *graph.Graph, q string) []Hit {
	q = strings.ToLower(strings.TrimSpace(q))
	hits := []Hit{}
	if q == "" {
		return hits
	}
	match := func(label string, paths ...string) (ok, exact bool) {
		l := strings.ToLower(label)
		ok = strings.Contains(l, q)
		for _, p := range paths {
			ok = ok || strings.Contains(strings.ToLower(p), q)
		}
		return ok, l == q
	}

	var found []ranked
	if code != nil {
		t := buildTree(code, nil)
		for id, e := range t.elems {
			ok, exact := match(e.node.Label, e.node.Path)
			if !ok {
				continue
			}
			// Everything above the hit, outermost first; a symbol needs its
			// file open too, which is among its ancestors.
			open := []string{}
			for p := e.parent; p != ""; p = t.elems[p].parent {
				open = append([]string{t.elems[p].node.Path}, open...)
			}
			found = append(found, ranked{Hit{id, e.node.Label, e.node.Kind, open}, e.node.Path, exact})
		}
	}
	if wiki != nil {
		for _, d := range wiki.Nodes {
			label := d.Title
			if label == "" {
				label = path.Base(d.ID)
			}
			ok, exact := match(d.Title, d.ID)
			if ok {
				found = append(found, ranked{Hit{"doc:" + d.ID, label, "doc", []string{}}, d.ID, exact})
			}
		}
	}

	sort.Slice(found, func(i, j int) bool {
		a, b := found[i], found[j]
		if a.exact != b.exact {
			return a.exact
		}
		if a.path != b.path {
			return a.path < b.path
		}
		return a.hit.ID < b.hit.ID
	})
	for _, f := range found[:min(len(found), MaxHits)] {
		hits = append(hits, f.hit)
	}
	return hits
}
