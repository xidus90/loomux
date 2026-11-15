package graphview

import (
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/model"
	"github.com/xidus90/loomux/internal/code/pagerank"
)

// elem is one place in the directory tree a graph node can be folded into.
type elem struct {
	node     Node
	parent   string
	children []string
}

// tree is every directory, file and symbol of the graph, with the graph nodes
// that fold into each of them.
type tree struct {
	elems map[string]*elem
	roots []string
	// chain is the path from the top directory down to the node itself, per
	// graph node.
	chain map[model.NodeID][]string
}

func (t *tree) ensure(id, parent string, n Node) {
	if _, ok := t.elems[id]; ok {
		return
	}
	n.ID = id
	n.Group = strings.SplitN(n.Path, "/", 2)[0]
	t.elems[id] = &elem{node: n, parent: parent}
	if parent == "" {
		t.roots = append(t.roots, id)
		return
	}
	p := t.elems[parent]
	p.children = append(p.children, id)
}

func buildTree(g *model.Graph, rank map[model.NodeID]float64) *tree {
	t := &tree{elems: map[string]*elem{}, chain: map[model.NodeID][]string{}}
	for _, n := range g.Nodes {
		if _, seen := t.chain[n.ID]; seen || n.Path == "" {
			continue
		}
		segs := strings.Split(n.Path, "/")
		var chain []string
		parent := ""
		for i := 1; i < len(segs); i++ {
			dir := strings.Join(segs[:i], "/")
			id := "dir:" + dir
			t.ensure(id, parent, Node{Label: segs[i-1], Kind: "dir", Path: dir})
			chain, parent = append(chain, id), id
		}
		fileID := "file:" + n.Path
		t.ensure(fileID, parent, Node{Label: segs[len(segs)-1], Kind: "file", Path: n.Path})
		chain = append(chain, fileID)
		if n.Kind != model.KindFile {
			t.ensure(string(n.ID), fileID, Node{Label: n.Name, Kind: "symbol", Path: n.Path})
			chain = append(chain, string(n.ID))
		}
		t.chain[n.ID] = chain
		for _, id := range chain {
			e := t.elems[id]
			e.node.Members++
			e.node.Size += rank[n.ID]
		}
	}
	return t
}

// expand lists the elements shown in place of id: id itself while it is
// closed, else what its children show.
func (t *tree) expand(id string, open map[string]bool) []string {
	if !open[id] {
		return []string{id}
	}
	var out []string
	for _, c := range t.elems[id].children {
		out = append(out, t.expand(c, open)...)
	}
	return out
}

// visible lists the elements shown when the elements in open are expanded: an
// open element gives way to its children.
func (t *tree) visible(open map[string]bool) []string {
	var out []string
	for _, r := range t.roots {
		out = append(out, t.expand(r, open)...)
	}
	return out
}

// image is the visible element a graph node folds into, or "" when it has
// none: an expanded file shows its symbols instead of itself.
func (t *tree) image(id model.NodeID, open map[string]bool) string {
	chain := t.chain[id]
	for _, e := range chain {
		if !open[e] {
			return e
		}
	}
	return ""
}

// clean keeps the entries that name a directory or file of the graph and whose
// every ancestor directory is open too, and that have something to show: a
// file without symbols has nothing to expand into. The view never grows past
// MaxNodes, however much is asked for. Nothing is looked up on disk.
func (t *tree) clean(entries []string) map[string]bool {
	paths := slices.Clone(entries)
	sort.Slice(paths, func(i, j int) bool {
		di, dj := strings.Count(paths[i], "/"), strings.Count(paths[j], "/")
		if di != dj {
			return di < dj
		}
		return paths[i] < paths[j]
	})
	open := map[string]bool{}
	shown := len(t.roots)
	for _, p := range paths {
		id := "dir:" + p
		if _, ok := t.elems[id]; !ok {
			id = "file:" + p
		}
		e, ok := t.elems[id]
		if !ok || open[id] || len(e.children) == 0 || (e.parent != "" && !open[e.parent]) {
			continue
		}
		// An element is visible here, so opening it swaps itself for its
		// children; one that would burst the cap stays closed.
		if shown+len(e.children)-1 > MaxNodes {
			continue
		}
		shown += len(e.children) - 1
		open[id] = true
	}
	return open
}

// autoOpen expands, best first, the directory with the most members while the
// view stays within limit. A directory with fewer than two children would only
// swap one node for another.
func (t *tree) autoOpen(open map[string]bool, limit int) {
	for {
		shown := t.visible(open)
		var best *elem
		for _, id := range shown {
			e := t.elems[id]
			if e.node.Kind != "dir" || len(e.children) < 2 {
				continue
			}
			if best == nil || e.node.Members > best.node.Members ||
				(e.node.Members == best.node.Members && e.node.ID < best.node.ID) {
				best = e
			}
		}
		if best == nil || len(shown)-1+len(best.children) > limit {
			return
		}
		open[best.node.ID] = true
	}
}

// sortView orders nodes by ID and edges by their ends, so the output never
// depends on map iteration.
func sortView(v *View) {
	sort.Slice(v.Nodes, func(i, j int) bool { return v.Nodes[i].ID < v.Nodes[j].ID })
	sort.Slice(v.Edges, func(i, j int) bool {
		a, b := v.Edges[i], v.Edges[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Target < b.Target
	})
}

// Code folds the code graph into directories. Each graph node belongs to its
// deepest visible ancestor; a directory in open shows its children and a file
// in open its symbols. Only a nil open has directories opened automatically,
// while at most limit nodes are shown; any other open, even an empty one or
// one whose every entry is dropped, shows exactly what it names.
func Code(g *model.Graph, open []string, limit int) View {
	view, _, _ := shape(g, open, limit)
	return view
}

// shape folds g and also hands back the tree and the cleaned open set the
// view was drawn from, so documents can be joined onto the same elements.
func shape(g *model.Graph, open []string, limit int) (View, *tree, map[string]bool) {
	view := View{Nodes: []Node{}, Edges: []Edge{}, Open: []string{}}
	if g == nil {
		g = &model.Graph{}
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	ranked := map[model.NodeID]float64{}
	seed := map[model.NodeID]float64{}
	for _, n := range g.Nodes {
		seed[n.ID] = 1
	}
	for _, s := range pagerank.Rank(pagerank.Prepare(g, nil), seed, pagerank.Options{}) {
		ranked[s.ID] = s.Score
	}

	t := buildTree(g, ranked)
	opened := t.clean(open)
	if open == nil {
		t.autoOpen(opened, limit)
	}
	for id := range opened {
		view.Open = append(view.Open, t.elems[id].node.Path)
	}
	sort.Strings(view.Open)

	for _, id := range t.visible(opened) {
		view.Nodes = append(view.Nodes, t.elems[id].node)
	}

	type pair struct{ source, target string }
	counts := map[pair]map[string]int{}
	for _, e := range g.Edges {
		if !e.Relation.IsWalk() {
			continue
		}
		p := pair{t.image(e.Source, opened), t.image(e.Target, opened)}
		if p.source == "" || p.target == "" || p.source == p.target {
			continue
		}
		if counts[p] == nil {
			counts[p] = map[string]int{}
		}
		counts[p][string(e.Relation)+"/"+string(e.Confidence)]++
	}
	for p, c := range counts {
		view.Edges = append(view.Edges, Edge{Source: p.source, Target: p.target, Kind: "code", Counts: c})
	}
	sortView(&view)
	return view, t, opened
}
