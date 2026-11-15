package graphview

import (
	"path"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/brain/graph"
	"github.com/xidus90/loomux/internal/code/model"
)

// inWiki reports whether the document id lies in the wiki folder; an area
// without one has nothing but wiki.
func inWiki(id, wikiPrefix string) bool {
	return wikiPrefix == "" || strings.HasPrefix(id, wikiPrefix+"/")
}

// docGroup is the colour key of a document: its top folder inside the wiki,
// else the two top folders of its path.
func docGroup(id, wikiPrefix string) string {
	if wikiPrefix != "" && strings.HasPrefix(id, wikiPrefix+"/") {
		rest := strings.TrimPrefix(id, wikiPrefix+"/")
		if top, _, nested := strings.Cut(rest, "/"); nested {
			return top
		}
		return path.Base(wikiPrefix)
	}
	dirs := strings.Split(path.Dir(id), "/")
	if len(dirs) > 2 {
		dirs = dirs[:2]
	}
	if dirs[0] == "." {
		return ""
	}
	return strings.Join(dirs, "/")
}

// targets lists the visible elements a reference to p points at: the element
// that holds p while one of its ancestors (or itself) is closed; once all are
// open, the symbol enclosing line, else everything shown in place of p.
func (t *tree) targets(p string, line int, open map[string]bool, spans map[string][]model.SymbolSpan) []string {
	id := "dir:" + p
	if _, ok := t.elems[id]; !ok {
		id = "file:" + p
		if _, ok := t.elems[id]; !ok {
			return nil
		}
	}
	closed := ""
	for cur := id; cur != ""; cur = t.elems[cur].parent {
		if !open[cur] {
			closed = cur
		}
	}
	if closed != "" {
		return []string{closed}
	}
	if line > 0 && t.elems[id].node.Kind == "file" {
		if sym := model.Enclosing(spans[p], line); sym != nil {
			return []string{string(sym.ID)}
		}
	}
	return t.expand(id, open)
}

// Build joins the wiki documents and what they name onto the code view. A
// reference ends at the elements Code shows for its path, a link or a
// reference to another document becomes an edge between the two, and a path
// found nowhere is kept in the document's Missing. wikiPrefix is the wiki
// folder relative to the area root, "" for an area without one.
func Build(code *model.Graph, wiki *graph.Graph, wikiPrefix string, open []string, limit int) View {
	view, t, opened := shape(code, open, limit)
	if wiki == nil {
		return view
	}

	docs := map[string]*Node{}
	for _, d := range wiki.Nodes {
		label := d.Title
		if label == "" {
			label = path.Base(d.ID)
		}
		docs[d.ID] = &Node{
			ID: "doc:" + d.ID, Label: label, Kind: "doc", Path: d.ID,
			Group: docGroup(d.ID, wikiPrefix), Size: 1, Members: 1, Wiki: inWiki(d.ID, wikiPrefix),
		}
	}

	type key struct{ source, target, kind string }
	counts := map[key]int{}
	link := func(from, to string) key { return key{"doc:" + from, "doc:" + to, "link"} }
	for _, e := range wiki.Edges {
		if docs[e.From] != nil && docs[e.To] != nil && e.From != e.To {
			counts[link(e.From, e.To)]++
		}
	}
	// A mention of a document the graph already links is the same link.
	linked := map[key]bool{}
	for k := range counts {
		linked[k] = true
	}

	spans := model.FileSpans(code)
	missing := map[string]map[string]bool{}
	for _, r := range wiki.CodeRefs {
		src := docs[r.From]
		if src == nil {
			continue
		}
		if docs[r.Path] != nil {
			if k := link(r.From, r.Path); r.From != r.Path && !linked[k] {
				counts[k]++
			}
			continue
		}
		hit := t.targets(r.Path, r.Line, opened, spans)
		if len(hit) == 0 {
			if missing[r.From] == nil {
				missing[r.From] = map[string]bool{}
			}
			missing[r.From][r.Path] = true
			continue
		}
		for _, id := range hit {
			counts[key{src.ID, id, "ref"}]++
		}
	}

	for id, set := range missing {
		for p := range set {
			docs[id].Missing = append(docs[id].Missing, p)
		}
		sort.Strings(docs[id].Missing)
	}
	for _, d := range docs {
		view.Nodes = append(view.Nodes, *d)
	}
	for k, n := range counts {
		view.Edges = append(view.Edges, Edge{Source: k.source, Target: k.target, Kind: k.kind, Counts: map[string]int{k.kind: n}})
	}
	sortView(&view)
	return view
}
