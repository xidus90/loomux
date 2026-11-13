package resolve

import (
	"cmp"
	"net/url"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// The resolvers of the web families, html and stylesheet, and the links of
// the pages that scope the web bridges
// (docs/.superpowers/specs/2026-11-11-web-design.md).

// localPath is the file part of a reference as a browser or a bundler reads
// it, a <script src> or an import specifier: the query (vite's ?worker) and
// the fragment dropped, escapes decoded. False for a scheme or a host, which
// name no file of the repository, and for an invalid escape.
func localPath(ref string) (string, bool) {
	u, err := url.Parse(ref)
	if err != nil || u.Scheme != "" || u.Host != "" {
		return "", false
	}
	return u.Path, true
}

// pageRef is the repository path a reference in the file from names, read
// by localPath: /x is x at the root, any other path is relative to from's
// directory. It is the one reading of <script src>, <link href>, @import
// and @use.
func pageRef(from, ref string) (string, bool) {
	p, ok := localPath(ref)
	if !ok {
		return "", false
	}
	if rest, ok := strings.CutPrefix(p, "/"); ok {
		return path.Clean(rest), true
	}
	return path.Join(path.Dir(from), p), true
}

// webLinks are what the pages load, read from the HTML files by pageRef,
// each list without duplicates.
type webLinks struct {
	pages   map[string][]string // script -> the pages loading it, sorted
	scripts map[string][]string // page -> the scripts it loads, in order
	sheets  map[string][]string // page -> the stylesheets it links, in order
}

// webOf reads the links of every HTML page among files.
func webOf(files []extract.Result) webLinks {
	w := webLinks{pages: map[string][]string{}, scripts: map[string][]string{}, sheets: map[string][]string{}}
	for _, f := range files {
		if f.Language != "html" {
			continue
		}
		for _, imp := range f.Imports {
			p, ok := pageRef(f.Path, imp.Path)
			switch {
			case !ok:
			case imp.Kind == "script":
				w.pages[p] = append(w.pages[p], f.Path)
				w.scripts[f.Path] = appendNew(w.scripts[f.Path], p)
			case imp.Kind == "stylesheet":
				w.sheets[f.Path] = appendNew(w.sheets[f.Path], p)
			}
		}
	}
	for s, ps := range w.pages {
		sort.Strings(ps)
		w.pages[s] = slices.Compact(ps)
	}
	return w
}

// appendNew is list with s at its end, unless list holds it already.
func appendNew(list []string, s string) []string {
	if slices.Contains(list, s) {
		return list
	}
	return append(list, s)
}

// edgeList gathers a web family's edges, all extracted. An import is kept
// once per file and target, at its smallest line, as ECMAScript keeps it;
// every other edge once per use.
type edgeList struct {
	out []model.Edge
	at  map[[2]model.NodeID]int // importer, imported -> its edge in out
}

func (l *edgeList) add(source, target model.NodeID, rel model.Relation, line int) {
	if rel == model.RelationImports {
		k := [2]model.NodeID{source, target}
		if i, ok := l.at[k]; ok {
			l.out[i].Line = min(l.out[i].Line, line)
			return
		}
		if l.at == nil {
			l.at = map[[2]model.NodeID]int{}
		}
		l.at[k] = len(l.out)
	}
	l.out = append(l.out, model.Edge{Source: source, Target: target, Relation: rel, Confidence: model.ConfidenceExtracted, Line: line})
}

// resolveHTML resolves the pages: an import to the file it names, when the
// repository holds it, and a class or an on* handler to a handover whose
// scope is what the page links or loads plus the page itself, which stands
// for its own <style> and inline scripts.
func resolveHTML(files []extract.Result, paths map[string]bool, w webLinks) langOut {
	var es edgeList
	var hs []handover
	for _, f := range files {
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				es.add(raw.Source, raw.TargetID, raw.Relation, 0)
			case model.RelationImports:
				if p, ok := pageRef(f.Path, raw.Specifier); ok && paths[p] {
					es.add(raw.Source, model.NodeID(p), raw.Relation, raw.Line)
				}
			default:
				links := w.sheets
				if extract.Namespace(raw.Member) == extract.NamespaceJSFunction {
					links = w.scripts
				}
				hs = append(hs, handover{bridge: extract.Bridge{
					Source: raw.Source, Relation: raw.Relation, Namespace: extract.Namespace(raw.Member), Name: raw.Name,
					Scope: appendNew(slices.Clone(links[f.Path]), f.Path), File: raw.File, Line: raw.Line,
				}, lang: f.Language, unresolved: -1})
			}
		}
	}
	return langOut{edges: es.out, handovers: hs}
}

// sheetLink is an import of a stylesheet resolved to a file: its kind
// (use, forward, import, or "" in CSS) and the namespace an @use binds.
type sheetLink struct {
	file, kind, ns string
}

// resolveStylesheet resolves the CSS and SCSS files: an import to the file
// it loads, an @include or a call to the one mixin or function nearest it,
// and an @extend to every placeholder or rule of the class it names.
//
// Nearest is breadth first over the resolved imports, the file itself
// first: an ns.m through the @use that binds ns and that file's @forward
// chain alone, anything else through every import. The first depth that
// holds the name decides, and only when it holds it once. An @extend sees
// every depth.
func resolveStylesheet(files []extract.Result, paths map[string]bool) langOut {
	links := map[string][]sheetLink{}
	callable := map[[3]string][]model.NodeID{}   // file, kind, name -> nodes
	extendable := map[[2]string][]model.NodeID{} // file, .class or %placeholder -> rules
	for _, f := range files {
		for _, imp := range f.Imports {
			if p, ok := sheetFile(f, imp.Path, paths); ok {
				base := path.Base(imp.Path)
				ns := cmp.Or(imp.Alias, strings.TrimPrefix(strings.TrimSuffix(base, path.Ext(base)), "_"))
				links[f.Path] = append(links[f.Path], sheetLink{file: p, kind: imp.Kind, ns: ns})
			}
		}
		for _, n := range f.Nodes {
			k := [3]string{f.Path, string(n.Kind), n.Name}
			callable[k] = append(callable[k], n.ID)
			if strings.HasPrefix(n.Name, "%") {
				k := [2]string{f.Path, n.Name}
				extendable[k] = append(extendable[k], n.ID)
			}
		}
		for _, e := range f.Exports {
			if e.Namespace == extract.NamespaceCSSClass {
				k := [2]string{f.Path, "." + e.Name}
				extendable[k] = append(extendable[k], e.Target)
			}
		}
	}
	var es edgeList
	for _, f := range files {
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				es.add(raw.Source, raw.TargetID, raw.Relation, 0)
			case model.RelationImports:
				if p, ok := sheetFile(f, raw.Specifier, paths); ok {
					es.add(raw.Source, model.NodeID(p), raw.Relation, raw.Line)
				}
			case model.RelationCalls:
				start, forward := f.Path, false
				if raw.Receiver != "" {
					// Nowhere, unless an @use binds the namespace.
					start, forward = "", true
					if i := slices.IndexFunc(links[f.Path], func(l sheetLink) bool { return l.kind == "use" && l.ns == raw.Receiver }); i >= 0 {
						start = links[f.Path][i].file
					}
				}
				for _, layer := range reach(start, links, forward) {
					var hits []model.NodeID
					for _, p := range layer {
						hits = append(hits, callable[[3]string{p, string(raw.Member), raw.Name}]...)
					}
					if len(hits) == 1 {
						es.add(raw.Source, hits[0], raw.Relation, raw.Line)
					}
					if len(hits) > 0 {
						break
					}
				}
			case model.RelationExtends:
				for _, layer := range reach(f.Path, links, false) {
					for _, p := range layer {
						for _, id := range extendable[[2]string{p, raw.Name}] {
							if id != raw.Source {
								es.add(raw.Source, id, raw.Relation, raw.Line)
							}
						}
					}
				}
			}
		}
	}
	return langOut{edges: es.out}
}

// reach is the files breadth first from start over links, every link or
// @forward alone, one layer per depth, each file once.
func reach(start string, links map[string][]sheetLink, forward bool) [][]string {
	seen := map[string]bool{start: true}
	var layers [][]string
	for layer := []string{start}; len(layer) > 0; {
		layers = append(layers, layer)
		var next []string
		for _, p := range layer {
			for _, l := range links[p] {
				if (!forward || l.kind == "forward") && !seen[l.file] {
					seen[l.file] = true
					next = append(next, l.file)
				}
			}
		}
		layer = next
	}
	return layers
}

// sheetFile is the file an import of f loads, false for none in paths:
// SCSS's by sassFile, CSS's by pageRef.
func sheetFile(f extract.Result, ref string, paths map[string]bool) (string, bool) {
	if f.Language == "scss" {
		return sassFile(f.Path, ref, paths)
	}
	p, ok := pageRef(f.Path, ref)
	return p, ok && paths[p]
}

// sassFile is the file an SCSS import in from loads, false for none in
// paths: ref read by pageRef, then with an extension the path and its
// partial, without one the partial, the file and the directory's index.
func sassFile(from, ref string, paths map[string]bool) (string, bool) {
	p, ok := pageRef(from, ref)
	if !ok {
		return "", false
	}
	dir, base := path.Split(p)
	cands := []string{p, dir + "_" + base}
	if ext := path.Ext(p); ext != ".scss" && ext != ".css" {
		cands = []string{dir + "_" + base + ".scss", p + ".scss", p + "/_index.scss"}
	}
	for _, c := range cands {
		if paths[c] {
			return c, true
		}
	}
	return "", false
}
