package resolve

import (
	"path"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolveGDScript resolves the raw edges of a Godot project's files: scripts,
// scenes, resources and project.godot.
//
// project.godot is the source of what Godot itself reads from it, not a
// rule loomux rebuilds: its directory is the root res:// paths start from,
// and its [autoload] section binds global names to files. A file belongs
// to the nearest project.godot above it; a file under none has no res://
// path and no autoloads. A path without a scheme is relative to the file's
// directory, an autoload's to project.godot's. A path lands on a file of any language of the build, or gives
// no edge.
//
// Every script is a class: its class_name node, else its file node. A name
// binds, first to last: an inner class of the class the name stands in or
// of one around it; a script-level `const X = preload(...)`; a class_name of
// the project; an autoload of the project. The first stage with a candidate
// decides, and two candidates there give no edge.
//
// Calls follow GDScript: a plain name or self.name is a function of the
// class or one of its bases, and nothing else -- there are no free
// functions in other files, so a name found nowhere is the engine's or a
// builtin and gives no edge. super starts at the base. X.f() binds X.
// X.new() is X's own _init, or X.
// sig.emit(), sig.connect() and sig.disconnect() on a signal of the class or
// one of its bases reference the signal.
// X.sig.emit(), connect and disconnect do the same for a signal of X or its
// bases, where X is bound like any receiver: an autoload, a class_name or an
// inner class.
// Every edge names exactly one target, or there is none, and every edge is
// extracted.
func resolveGDScript(files []extract.Result, paths map[string]bool) []model.Edge {
	x := gdIndexOf(files, paths)
	var out []model.Edge
	var rest []extract.RawEdge
	// One edge per source, target and relation: self.f() and f() are one
	// dependency, as are a name and the alias that binds the same file.
	seen := map[model.Edge]bool{}
	keep := func(e model.Edge) {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	for _, f := range files {
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				keep(model.Edge{Source: raw.Source, Target: raw.TargetID, Relation: model.RelationContains, Confidence: model.ConfidenceExtracted})
			case model.RelationExtends:
				if base, ok := x.base(raw); ok {
					x.bases[raw.Source] = base
					keep(gdEdge(raw, base))
				}
			default:
				rest = append(rest, raw)
			}
		}
	}
	// A call in the first file may need the base of a class in the last.
	for _, raw := range rest {
		if e, ok := x.resolve(raw); ok {
			keep(e)
		}
	}
	return out
}

// gdIndex is every lookup the Godot resolver makes.
type gdIndex struct {
	paths    map[string]bool                          // every file of the build, any language
	roots    []string                                 // the directories of project.godot files, longest first
	kind     map[model.NodeID]model.Kind              // node -> its kind
	parent   map[model.NodeID]model.NodeID            // node -> the node that contains it
	members  map[model.NodeID]map[string][]model.Node // class or file -> name -> its definitions
	script   map[string]model.NodeID                  // .gd file -> its class: the class_name node, else the file
	aliases  map[string]map[string]string             // file -> alias -> path as written
	classes  map[string]map[string][]model.NodeID     // project root -> class_name -> its classes
	autoload map[string]map[string]string             // project root -> name -> path as written
	bases    map[model.NodeID]model.NodeID            // class -> its base, as resolveGDScript resolves it
}

func gdIndexOf(files []extract.Result, paths map[string]bool) *gdIndex {
	x := &gdIndex{
		paths: paths, kind: map[model.NodeID]model.Kind{}, parent: map[model.NodeID]model.NodeID{},
		members: map[model.NodeID]map[string][]model.Node{}, script: map[string]model.NodeID{},
		aliases: map[string]map[string]string{}, classes: map[string]map[string][]model.NodeID{},
		autoload: map[string]map[string]string{}, bases: map[model.NodeID]model.NodeID{},
	}
	for _, f := range files {
		if path.Base(f.Path) == "project.godot" {
			root := dirOf(f.Path)
			x.roots = append(x.roots, root)
			x.classes[root] = map[string][]model.NodeID{}
			x.autoload[root] = map[string]string{}
			for _, imp := range f.Imports {
				x.autoload[root][imp.Alias] = imp.Path
			}
		}
	}
	// The nearest project first: a/sub/project.godot owns a/sub/x.gd.
	sort.Slice(x.roots, func(i, j int) bool { return len(x.roots[i]) > len(x.roots[j]) })
	for _, f := range files {
		x.aliases[f.Path] = map[string]string{}
		for _, imp := range f.Imports {
			x.aliases[f.Path][imp.Alias] = imp.Path
		}
		if path.Ext(f.Path) == ".gd" {
			x.script[f.Path] = model.NodeID(f.Path)
			if f.Package != "" {
				// Minted first in its file, so it carries no ordinal.
				id := model.NodeID(f.Path + "#" + f.Package)
				x.script[f.Path] = id
				if root, ok := x.project(f.Path); ok {
					x.classes[root][f.Package] = append(x.classes[root][f.Package], id)
				}
			}
		}
		for _, raw := range f.Edges {
			if raw.Relation == model.RelationContains {
				x.parent[raw.TargetID] = raw.Source
			}
		}
		for _, n := range f.Nodes {
			x.kind[n.ID] = n.Kind
			if n.Kind == model.KindFile {
				continue
			}
			p := x.parent[n.ID]
			if x.members[p] == nil {
				x.members[p] = map[string][]model.Node{}
			}
			x.members[p][n.Name] = append(x.members[p][n.Name], n)
		}
	}
	return x
}

// project is the root of the project a file belongs to: the nearest
// project.godot above it.
func (x *gdIndex) project(file string) (string, bool) {
	for _, root := range x.roots {
		if root == "" || strings.HasPrefix(file, root+"/") {
			return root, true
		}
	}
	return "", false
}

// target is the file a path in file names: res:// from the project root, a
// path without a scheme from the file's directory. False when there is no
// project for res://, or no file of the build there -- an icon, a uid://
// or user:// path, a file outside the tree.
func (x *gdIndex) target(file, spec string) (string, bool) {
	var p string
	if rest, ok := strings.CutPrefix(spec, "res://"); ok {
		root, ok := x.project(file)
		if !ok {
			return "", false
		}
		p = path.Join(root, rest)
	} else {
		p = path.Join(dirOf(file), spec)
	}
	return p, x.paths[p]
}

// classOf is what a file stands for as a class: its script's class, or the
// file itself for anything that is no script.
func (x *gdIndex) classOf(file string) model.NodeID {
	if id, ok := x.script[file]; ok {
		return id
	}
	return model.NodeID(file)
}

// home is the class a raw edge's source stands in: a class is its own, a
// function or method the class or script around it, the file its script's.
func (x *gdIndex) home(raw extract.RawEdge) model.NodeID {
	switch {
	case raw.Source == model.NodeID(raw.File):
		return x.script[raw.File]
	case x.kind[raw.Source] == "class":
		return raw.Source
	}
	if p := x.parent[raw.Source]; p != model.NodeID(raw.File) {
		return p
	}
	return x.script[raw.File]
}

// bind is the class or file a name stands for in a class of file, by the
// four stages in order; the first stage with a candidate decides.
func (x *gdIndex) bind(file string, home model.NodeID, name string) (model.NodeID, bool) {
	for c := home; c != ""; c = x.parent[c] {
		if found := ofKind(x.members[c][name], "class"); len(found) > 0 {
			return only(found)
		}
	}
	if spec, ok := x.aliases[file][name]; ok {
		p, ok := x.target(file, spec)
		return x.classOf(p), ok
	}
	root, ok := x.project(file)
	if !ok {
		return "", false
	}
	if ids := x.classes[root][name]; len(ids) > 0 {
		if len(ids) != 1 {
			return "", false
		}
		return ids[0], true
	}
	if spec, ok := x.autoload[root][name]; ok {
		// A path without a scheme starts at the project, as Godot reads it.
		p, ok := x.target(path.Join(root, "project.godot"), spec)
		return x.classOf(p), ok
	}
	return "", false
}

// chain binds the first name of "A.B.C" and walks the rest as inner classes.
func (x *gdIndex) chain(file string, home model.NodeID, names string) (model.NodeID, bool) {
	parts := strings.Split(names, ".")
	c, ok := x.bind(file, home, parts[0])
	for _, p := range parts[1:] {
		if !ok {
			break
		}
		c, ok = only(ofKind(x.members[c][p], "class"))
	}
	return c, ok
}

// base resolves an extends edge: a path, or a name bound from the class
// around the one being defined. A class is never its own base.
func (x *gdIndex) base(raw extract.RawEdge) (model.NodeID, bool) {
	if raw.Specifier != "" {
		p, ok := x.target(raw.File, raw.Specifier)
		return x.classOf(p), ok && x.classOf(p) != raw.Source
	}
	name := raw.Name
	if raw.Receiver != "" {
		name = raw.Receiver + "." + raw.Name
	}
	t, ok := x.chain(raw.File, x.parent[raw.Source], name)
	return t, ok && t != raw.Source
}

// member is the definitions of name of the given kinds in the first class
// of class's line of bases that has any. Each class is visited once, so a
// cycle of bases ends.
func (x *gdIndex) member(class model.NodeID, name string, kinds ...model.Kind) []model.Node {
	seen := map[model.NodeID]bool{}
	for c := class; c != "" && !seen[c]; c = x.bases[c] {
		seen[c] = true
		if found := ofKind(x.members[c][name], kinds...); len(found) > 0 {
			return found
		}
	}
	return nil
}

// resolve resolves an imports, references or calls edge.
func (x *gdIndex) resolve(raw extract.RawEdge) (model.Edge, bool) {
	switch raw.Relation {
	case model.RelationImports:
		p, ok := x.target(raw.File, raw.Specifier)
		return gdEdge(raw, model.NodeID(p)), ok && p != raw.File
	case model.RelationReferences:
		t, ok := x.bind(raw.File, x.home(raw), raw.Name)
		return gdEdge(raw, t), ok && !x.encloses(t, raw.Source)
	default:
		return x.call(raw)
	}
}

// encloses is whether t is src or a definition around it: a name for the
// class one stands in references nothing new.
func (x *gdIndex) encloses(t, src model.NodeID) bool {
	for c := src; c != ""; c = x.parent[c] {
		if c == t {
			return true
		}
	}
	return false
}

// call resolves one call by its shape (see resolveGDScript).
func (x *gdIndex) call(raw extract.RawEdge) (model.Edge, bool) {
	home := x.home(raw)
	fns := []model.Kind{"method", "function"}
	switch {
	case raw.Receiver == "":
		return one(raw, x.member(home, raw.Name, fns...), model.ConfidenceExtracted)
	case raw.Receiver == "super":
		return one(raw, x.member(x.bases[home], raw.Name, fns...), model.ConfidenceExtracted)
	}
	if signalVerb(raw.Name) {
		sig := x.member(home, raw.Receiver, "signal")
		if i := strings.LastIndex(raw.Receiver, "."); i >= 0 {
			// X.sig: bind X like any receiver chain, then the signal in it.
			// Exactly one hit; with several the call goes on as below.
			if owner, ok := x.chain(raw.File, home, raw.Receiver[:i]); ok {
				if hits := x.member(owner, raw.Receiver[i+1:], "signal"); len(hits) == 1 {
					sig = hits
				}
			}
		}
		if len(sig) > 0 {
			e, ok := one(raw, sig, model.ConfidenceExtracted)
			e.Relation = model.RelationReferences
			return e, ok
		}
	}
	class, ok := x.chain(raw.File, home, raw.Receiver)
	if !ok {
		return model.Edge{}, false
	}
	if raw.Name == "new" {
		if inits := ofKind(x.members[class]["_init"], fns...); len(inits) == 1 {
			return gdEdge(raw, inits[0].ID), true
		}
		return gdEdge(raw, class), true
	}
	return one(raw, x.member(class, raw.Name, fns...), model.ConfidenceExtracted)
}

// signalVerb is whether a member call on a signal uses the signal itself.
func signalVerb(name string) bool {
	return name == "emit" || name == "connect" || name == "disconnect"
}

// ofKind keeps the nodes of the given kinds.
func ofKind(nodes []model.Node, kinds ...model.Kind) []model.Node {
	var out []model.Node
	for _, n := range nodes {
		for _, k := range kinds {
			if n.Kind == k {
				out = append(out, n)
			}
		}
	}
	return out
}

// only is the one node's id, or false for none or several.
func only(nodes []model.Node) (model.NodeID, bool) {
	if len(nodes) != 1 {
		return "", false
	}
	return nodes[0].ID, true
}

// gdEdge is the extracted edge of raw's relation from raw's source to t.
func gdEdge(raw extract.RawEdge, t model.NodeID) model.Edge {
	return model.Edge{Source: raw.Source, Target: t, Relation: raw.Relation, Confidence: model.ConfidenceExtracted}
}
