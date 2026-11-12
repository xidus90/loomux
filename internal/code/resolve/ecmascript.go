package resolve

import (
	"cmp"
	"maps"
	"net/url"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolveECMAScript resolves the raw edges of the JavaScript and TypeScript
// files, one family, against an index of those files alone
// (docs/.superpowers/specs/2026-11-11-ecmascript-design.md): an import to
// the file it loads (importFile), a base and a call through the file's
// declarations and import bindings (target), this and super through the
// bases of the calling class. pages maps a script to the HTML pages that
// load it (scriptPages): they scope its element ids, and they make a
// classic script offer its functions.
//
// Only a base gives an Unresolved entry, and only one whose name an import
// binds to a file of the repository: a global or a package says nothing
// about the project.
func resolveECMAScript(files []extract.Result, mods []Module, pages map[string][]string) langOut {
	x := esIndexOf(files, mods)
	var out []model.Edge
	var un []Unresolved
	var calls []extract.RawEdge
	var hs []handover
	// The position in out of each import edge, which keeps the smallest
	// line of the imports that make it.
	imported := map[[2]model.NodeID]int{}
	for _, f := range files {
		// An embedded part shares its page's path, and its page is its scope.
		scope := pages[f.Path]
		if _, ok := esExts[path.Ext(f.Path)]; !ok {
			scope = []string{f.Path}
		}
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				out = append(out, model.Edge{
					Source: raw.Source, Target: raw.TargetID,
					Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationImports:
				target, ok := x.importFile(f.Path, raw.Specifier)
				key := [2]model.NodeID{raw.Source, model.NodeID(target)}
				if !ok || key[1] == raw.Source {
					continue
				}
				if i, seen := imported[key]; seen {
					out[i].Line = min(out[i].Line, raw.Line)
					continue
				}
				imported[key] = len(out)
				out = append(out, model.Edge{
					Source: raw.Source, Target: key[1],
					Relation: model.RelationImports, Confidence: model.ConfidenceExtracted, Line: raw.Line,
				})
			case model.RelationExtends, model.RelationImplements:
				base := x.target(raw.File, raw.Receiver, raw.Name)
				if base == "" || x.isFile(base) {
					if x.repoBinding(raw.File, cmp.Or(raw.Receiver, raw.Name)) {
						un = append(un, Unresolved{Source: raw.Source, File: raw.File, Line: raw.Line, Name: join(raw.Receiver, raw.Name), Reason: ReasonNone})
					}
					continue
				}
				// Recorded for this and super, whose calls therefore
				// resolve after this loop.
				if own, t := x.class[raw.Source], x.class[base]; own != nil && t != nil {
					own.bases = append(own.bases, t)
				}
				out = append(out, model.Edge{
					Source: raw.Source, Target: base,
					Relation: raw.Relation, Confidence: model.ConfidenceExtracted, Line: raw.Line,
				})
			case model.RelationCalls:
				calls = append(calls, raw)
			case model.RelationReferences:
				// An element id: only the pages loading the script know it.
				hs = append(hs, handover{bridge: extract.Bridge{
					Source: raw.Source, Relation: raw.Relation, Namespace: extract.Namespace(raw.Member),
					Name: raw.Name, Scope: scope, File: raw.File, Line: raw.Line,
				}, lang: f.Language, unresolved: -1})
			}
		}
	}
	for _, raw := range calls {
		if target := x.call(raw); target != "" {
			out = append(out, model.Edge{
				Source: raw.Source, Target: target,
				Relation: model.RelationCalls, Confidence: model.ConfidenceExtracted, Line: raw.Line,
			})
		}
	}
	return langOut{edges: out, unresolved: un, offers: esOffers(files, pages), handovers: hs}
}

// esExts are the extensions of the family's files.
var esExts = map[string]bool{
	".js": true, ".mjs": true, ".cjs": true, ".jsx": true,
	".ts": true, ".mts": true, ".cts": true, ".tsx": true,
}

// probeExts are the extensions an import without one is tried with, in
// order; then the same after /index.
var probeExts = []string{".ts", ".tsx", ".d.ts", ".js", ".jsx", ".mjs", ".cjs"}

// compiledFrom are the sources a TypeScript file may name by what they
// compile to, as tsc's node16 resolution maps them.
var compiledFrom = map[string][]string{
	".js":  {".ts", ".tsx", ".d.ts"},
	".jsx": {".tsx"},
	".mjs": {".mts", ".d.mts"},
	".cjs": {".cts", ".d.cts"},
}

// esIndex is every lookup the ECMAScript resolver makes, over the family's
// files alone.
type esIndex struct {
	files   map[string]bool                  // every file an import may load
	nodes   map[model.NodeID]model.Node      // every node
	top     map[string]map[string]model.Node // file -> name -> its first top-level declaration
	imports map[string][]extract.Import      // file -> its imports
	class   map[model.NodeID]*csType         // class or interface -> its type
	typeOf  map[model.NodeID]*csType         // method -> its class
	config  map[string]*Module               // file -> the tsconfig nearest it, nil for none
	ts      map[string]bool                  // file -> TypeScript: its own, or a page's TypeScript part
}

// esIndexOf indexes the family's files. A type's bases are left empty:
// resolveECMAScript records them as it resolves the extends edges.
func esIndexOf(files []extract.Result, mods []Module) *esIndex {
	x := &esIndex{
		files: map[string]bool{}, nodes: map[model.NodeID]model.Node{}, top: map[string]map[string]model.Node{},
		imports: map[string][]extract.Import{}, class: map[model.NodeID]*csType{}, typeOf: map[model.NodeID]*csType{},
		config: map[string]*Module{}, ts: map[string]bool{},
	}
	for _, f := range files {
		x.ts[f.Path] = x.ts[f.Path] || f.Language == "typescript"
		if _, ok := esExts[path.Ext(f.Path)]; ok {
			x.files[f.Path] = true
		}
		x.imports[f.Path] = append(x.imports[f.Path], f.Imports...)
		x.config[f.Path] = nearestTSConfig(mods, f.Path)
		if x.top[f.Path] == nil {
			x.top[f.Path] = map[string]model.Node{}
		}
		// The file's types by qualified name, for its methods.
		local := map[string]*csType{}
		for _, n := range f.Nodes {
			x.nodes[n.ID] = n
			switch n.Kind {
			case model.KindFile:
				continue
			case "method":
				// The extractor makes a method only after its class; a result
				// another producer built may not.
				if t := local[n.Owner]; t != nil {
					x.typeOf[n.ID] = t
					if _, ok := t.members[n.Name]; !ok {
						t.members[n.Name] = n.ID
					}
				}
				continue
			case "class", "interface":
				t := &csType{id: n.ID, members: map[string]model.NodeID{}}
				x.class[n.ID], local[join(n.Owner, n.Name)] = t, t
			}
			// A namespace's member is reached through the namespace, not by
			// its name alone.
			if _, ok := x.top[f.Path][n.Name]; !ok && n.Owner == "" {
				x.top[f.Path][n.Name] = n
			}
		}
	}
	return x
}

// importFile is the file an import of file loads, false for none:
//
//   - ./x, ../x, . and .. relative to file's directory
//   - any other name through the paths, then the baseUrl, of the tsconfig
//     nearest file; a package without either
//
// For a TypeScript importer a hit on a declaration file stops the search
// and gives none; a JavaScript importer passes over it.
func (x *esIndex) importFile(file, spec string) (string, bool) {
	// A leading # is a subpath import or an alias, not a fragment.
	rest, hash := strings.CutPrefix(spec, "#")
	spec, ok := localPath(rest)
	if !ok {
		return "", false
	}
	if hash {
		spec = "#" + spec
	}
	var bases []string
	if spec == "." || spec == ".." || strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") {
		bases = []string{path.Join(path.Dir(file), spec)}
	} else {
		bases = aliased(x.config[file], spec)
	}
	ts := x.ts[file]
	for _, base := range bases {
		if p, hit := x.probe(base, ts); hit {
			return p, p != ""
		}
	}
	return "", false
}

// probe is the file base names, as written, compiled from in TypeScript,
// with an extension, or as a directory's index; "" when a TypeScript
// importer hits a declaration file. hit is whether any file answered.
func (x *esIndex) probe(base string, ts bool) (p string, hit bool) {
	cands := []string{base}
	if ext := path.Ext(base); ts {
		for _, src := range compiledFrom[ext] {
			cands = append(cands, strings.TrimSuffix(base, ext)+src)
		}
	}
	for _, ext := range probeExts {
		cands = append(cands, base+ext)
	}
	for _, ext := range probeExts {
		cands = append(cands, path.Join(base, "index"+ext))
	}
	for _, c := range cands {
		if !x.files[c] {
			continue
		}
		if !declaration(c) {
			return c, true
		}
		// tsc stops at the declaration; Node never loads one and goes on.
		if ts {
			return "", true
		}
	}
	return "", false
}

// declaration says whether p is a TypeScript declaration file.
func declaration(p string) bool {
	return strings.HasSuffix(p, ".d.ts") || strings.HasSuffix(p, ".d.mts") || strings.HasSuffix(p, ".d.cts")
}

// aliased are the paths a bare import names under the tsconfig m, in the
// order tsc tries them: the targets of the paths pattern matching it with
// the longest prefix before its *, an exact pattern first, then the name
// under baseUrl. paths resolve from baseUrl, without one from the config's
// directory.
func aliased(m *Module, spec string) []string {
	if m == nil {
		return nil
	}
	root := path.Join(m.Dir, m.TS.BaseURL)
	targets, capture := m.TS.Paths[spec], ""
	if targets == nil {
		best := -1
		// In sorted order, so a tie keeps the same pattern on every run.
		for _, pattern := range slices.Sorted(maps.Keys(m.TS.Paths)) {
			prefix, suffix, wild := strings.Cut(pattern, "*")
			if !wild || len(prefix) <= best || len(spec) < len(prefix)+len(suffix) ||
				!strings.HasPrefix(spec, prefix) || !strings.HasSuffix(spec, suffix) {
				continue
			}
			best, targets, capture = len(prefix), m.TS.Paths[pattern], spec[len(prefix):len(spec)-len(suffix)]
		}
	}
	var out []string
	for _, t := range targets {
		out = append(out, path.Join(root, strings.Replace(t, "*", capture, 1)))
	}
	if m.TS.BaseURL != "" {
		out = append(out, path.Join(root, spec))
	}
	return out
}

// target is the node a name written in file stands for, "" for none: name
// itself, or the export name of the module receiver binds.
func (x *esIndex) target(file, receiver, name string) model.NodeID {
	if receiver == "" {
		return x.local(file, name, map[[2]string]bool{})
	}
	if ns := x.local(file, receiver, map[[2]string]bool{}); x.isFile(ns) {
		return x.exported(string(ns), name, map[[2]string]bool{})
	}
	return ""
}

// local is what name stands for at the top level of file: its declaration,
// else what an import binds it to -- a whole module as its file node.
// seen holds the (file, name) pairs already followed, so a cycle of
// re-exports ends.
func (x *esIndex) local(file, name string, seen map[[2]string]bool) model.NodeID {
	if n, ok := x.top[file][name]; ok {
		return n.ID
	}
	imp, ok := x.binding(file, name)
	if !ok {
		return ""
	}
	target, ok := x.importFile(file, imp.Path)
	switch {
	case !ok:
		return ""
	case imp.Name == "*":
		return model.NodeID(target)
	}
	return x.exported(target, imp.Name, seen)
}

// binding is the import that binds name in file: a static import or a
// require. A re-export or an export list binds no name of its own.
func (x *esIndex) binding(file, name string) (extract.Import, bool) {
	for _, imp := range x.imports[file] {
		if (imp.Kind == "" || imp.Kind == "require") && cmp.Or(imp.Alias, imp.Name) == name {
			return imp, true
		}
	}
	return extract.Import{}, false
}

// repoBinding says whether an import binds name in file to a file of the
// repository: a relative one, or one that loads a file here.
func (x *esIndex) repoBinding(file, name string) bool {
	imp, ok := x.binding(file, name)
	if !ok {
		return false
	}
	_, found := x.importFile(file, imp.Path)
	return found || strings.HasPrefix(imp.Path, ".")
}

// exported is the node file exports as name, "" for none: an exported
// declaration, an export list naming a local name, a re-export, else the
// one module among its export * sources that has the name. export * never
// passes on a default.
func (x *esIndex) exported(file, name string, seen map[[2]string]bool) model.NodeID {
	key := [2]string{file, name}
	if seen[key] {
		return ""
	}
	seen[key] = true
	var stars []string
	// Exported also marks a name an export list renames, which the file
	// exports under its new name alone.
	listed := false
	for _, imp := range x.imports[file] {
		listed = listed || imp.Kind == "export" && imp.Name == name
		switch {
		case imp.Kind == "reexport" && imp.Name == "*" && imp.Alias == "":
			stars = append(stars, imp.Path)
		case imp.Kind == "export" && cmp.Or(imp.Alias, imp.Name) == name:
			return x.local(file, imp.Name, seen)
		case imp.Kind == "reexport" && cmp.Or(imp.Alias, imp.Name) == name:
			target, ok := x.importFile(file, imp.Path)
			switch {
			case !ok:
				return ""
			case imp.Name == "*":
				return model.NodeID(target)
			}
			return x.exported(target, imp.Name, seen)
		}
	}
	if n, ok := x.top[file][name]; ok && n.Exported && !listed {
		return n.ID
	}
	var hit model.NodeID
	for _, spec := range stars {
		target, ok := x.importFile(file, spec)
		if !ok || name == "default" {
			continue
		}
		if id := x.exported(target, name, seen); id != "" {
			if hit != "" {
				return "" // two modules export it: tsc reports the clash
			}
			hit = id
		}
	}
	return hit
}

// call is the node one raw call lands on, "" for none:
//
//   - this.m(), super.m(): a method of the calling class or its bases, of
//     its bases alone for super
//   - f(), ns.f(): what target finds; a module called itself, a require
//     called as m(), runs its default export
//   - new X(): the constructor of X, else what target finds
func (x *esIndex) call(raw extract.RawEdge) model.NodeID {
	switch raw.Receiver {
	case "this", "super":
		own := x.typeOf[raw.Source]
		if own == nil {
			return ""
		}
		start := []*csType{own}
		if raw.Receiver == "super" {
			start = own.bases
		}
		return member(start, raw.Name)
	}
	id := x.target(raw.File, raw.Receiver, raw.Name)
	if x.isFile(id) {
		id = x.exported(string(id), "default", map[[2]string]bool{})
	}
	if t := x.class[id]; t != nil && raw.Member == "new" {
		if ctor, ok := t.members["constructor"]; ok {
			return ctor
		}
	}
	return id
}

// isFile says whether id is a file's node: a module, as a binding of it
// stands for it.
func (x *esIndex) isFile(id model.NodeID) bool {
	return x.nodes[id].Kind == model.KindFile
}

// esOffers are the top-level functions of every classic script a page
// loads or embeds, under js.function: a script with no import, require or
// export, whose names share the page's global scope.
func esOffers(files []extract.Result, pages map[string][]string) []offer {
	var out []offer
	for _, f := range files {
		_, page := pages[f.Path]
		_, own := esExts[path.Ext(f.Path)]
		if own && !page || len(f.Imports) > 0 || slices.ContainsFunc(f.Nodes, func(n model.Node) bool {
			return n.Exported && n.Kind != model.KindFile
		}) {
			continue
		}
		for _, n := range f.Nodes {
			if n.Kind == "function" && n.Owner == "" {
				out = append(out, offer{
					Export: extract.Export{Namespace: extract.NamespaceJSFunction, Name: n.Name, Target: n.ID},
					file:   f.Path, lang: f.Language,
				})
			}
		}
	}
	return out
}

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

// scriptPages maps each script to the HTML pages that load it with
// <script src>, sorted and each once, the src read by localPath; /x.js is
// x.js at the root, any other src is relative to the page's directory.
func scriptPages(files []extract.Result) map[string][]string {
	pages := map[string][]string{}
	for _, f := range files {
		if f.Language != "html" {
			continue
		}
		for _, imp := range f.Imports {
			src, ok := localPath(imp.Path)
			if imp.Kind != "script" || !ok {
				continue
			}
			p := path.Join(path.Dir(f.Path), src)
			if rest, ok := strings.CutPrefix(src, "/"); ok {
				p = path.Clean(rest)
			}
			pages[p] = append(pages[p], f.Path)
		}
	}
	for s, ps := range pages {
		sort.Strings(ps)
		pages[s] = slices.Compact(ps)
	}
	return pages
}
