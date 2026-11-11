package resolve

import (
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolveCSharp resolves the raw edges of the C# files against an index of
// those files alone: containment as the extractor resolved it, a using to a
// file of the namespace it opens, a type to its base types, a call to the
// member or constructor it names (call).
//
// A namespace is one package spread over many files, as in Go, so a using
// points at one file standing for it (csIndexOf); an edge to every file of
// the namespace would make each user depend on each of them. A using of a
// namespace no file here declares -- System, Godot -- gives no edge, as in
// Python. Neither does `using static X.T` or an alias of a type: they name a
// type, and no namespace of that name exists.
//
// Only a base gives an Unresolved entry; a call that finds nothing is
// dropped, as in Python.
func resolveCSharp(files []extract.Result) langOut {
	x := csIndexOf(files)
	var out []model.Edge
	var un []Unresolved
	var calls []extract.RawEdge
	// The position in out of each import edge, which keeps the smallest line
	// of the usings that make it.
	imported := map[[2]model.NodeID]int{}
	for _, f := range files {
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				out = append(out, model.Edge{
					Source: raw.Source, Target: raw.TargetID,
					Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationImports:
				target, ok := x.nsFile[raw.Specifier]
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
			case model.RelationExtends:
				t := x.typeOf[raw.Source]
				// A base list is read outside the type it belongs to: its own
				// nested types are not in scope there.
				base, why := x.typeNamed(raw.File, t.outer, t.ns, join(raw.Receiver, raw.Name))
				if why != "" {
					un = append(un, Unresolved{
						Source: raw.Source, File: raw.File, Line: raw.Line,
						Name: join(raw.Receiver, raw.Name), Reason: why,
					})
					continue
				}
				// Recorded for the lookups that walk a type's bases, which
				// therefore run after this loop.
				t.bases = append(t.bases, base)
				rel := model.RelationExtends
				if base.iface {
					rel = model.RelationImplements
				}
				out = append(out, model.Edge{
					Source: raw.Source, Target: base.id,
					Relation: rel, Confidence: model.ConfidenceExtracted, Line: raw.Line,
				})
			case model.RelationCalls:
				calls = append(calls, raw)
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
	return langOut{edges: out, unresolved: un}
}

// csIndex is every lookup the C# resolver makes, over the C# files alone.
type csIndex struct {
	types   map[string]*csType          // full name, "Ns.Outer.Inner" -> the type
	typeOf  map[model.NodeID]*csType    // type part or member -> its type
	nsFile  map[string]string           // namespace -> the file standing for it
	usings  map[string][]extract.Import // file -> its usings but the global ones
	globals []extract.Import            // the global usings of every file
}

// csType is one logical type: every partial part of it, which share a
// namespace and a qualified name, merged.
type csType struct {
	id      model.NodeID            // the part in the file with the smallest path
	full    string                  // its namespace and qualified name, dotted
	ns      string                  // its namespace, "" for the global one
	iface   bool                    // an interface: a base of it is implemented
	outer   *csType                 // the type it is nested in, nil at namespace level
	members map[string]model.NodeID // name -> the first declaration
	bases   []*csType               // its base types in this repository, as written
}

// csIndexOf indexes the C# files. Files are read in path order, so the first
// part of a type and the first declaration of an overload are the ones in
// the smallest path, and in source order within a file. The bases are left
// empty: resolveCSharp records them as it resolves the extends edges.
//
// The file standing for a namespace is the one with the smallest path among
// the non-test files declaring a type in it, else among all of them: no
// importer sees a test project, but a namespace only tests declare is still
// what its own tests import.
func csIndexOf(files []extract.Result) *csIndex {
	x := &csIndex{
		types: map[string]*csType{}, typeOf: map[model.NodeID]*csType{},
		nsFile: map[string]string{}, usings: map[string][]extract.Import{},
	}
	sorted := slices.Clone(files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	for _, f := range sorted {
		for _, imp := range f.Imports {
			if imp.Global {
				x.globals = append(x.globals, imp)
			} else {
				x.usings[f.Path] = append(x.usings[f.Path], imp)
			}
		}
		// The file's types by qualified name, for its members: a member's
		// owner names its type without the namespace.
		local := map[string]*csType{}
		for _, n := range f.Nodes {
			switch n.Kind {
			case model.KindFile:
			case "class", "interface":
				ns := f.Enclosing[n.ID]
				if cur, ok := x.nsFile[ns]; !ok || blast.IsTestPath(cur) && !blast.IsTestPath(f.Path) {
					x.nsFile[ns] = f.Path
				}
				qualified := join(n.Owner, n.Name)
				full := join(ns, qualified)
				t := x.types[full]
				if t == nil {
					t = &csType{id: n.ID, full: full, ns: ns, outer: local[n.Owner], members: map[string]model.NodeID{}}
					x.types[full] = t
				}
				t.iface = n.Kind == "interface"
				x.typeOf[n.ID], local[qualified] = t, t
			default:
				t := local[n.Owner]
				if t == nil {
					// No type of the file owns it: a result some other
					// producer than the C# extractor built.
					continue
				}
				x.typeOf[n.ID] = t
				if _, ok := t.members[n.Name]; !ok {
					t.members[n.Name] = n.ID
				}
			}
		}
	}
	return x
}

// typeNamed is the type a dotted name stands for in file, written inside the
// type scope (nil outside every type) in namespace ns. The first step that
// finds any type decides; two types found there are a guess:
//
//  1. a type nested in scope, then in each type around it, inside out
//  2. a type in ns, then in each namespace around it, inside out
//  3. through a using alias of the file, of a type or a namespace
//  4. a type in a namespace the file opens with a using, or any file with a
//     global using
//  5. a type in the global namespace
func (x *csIndex) typeNamed(file string, scope *csType, ns, name string) (*csType, Reason) {
	var steps [][]string
	for t := scope; t != nil; t = t.outer {
		steps = append(steps, []string{t.full + "." + name})
	}
	for s := ns; s != ""; s = parentNs(s) {
		steps = append(steps, []string{s + "." + name})
	}
	first, _, _ := strings.Cut(name, ".")
	var aliased, opened []string
	for _, imp := range x.usingsOf(file) {
		switch imp.Alias {
		case "":
			opened = append(opened, imp.Path+"."+name)
		case first:
			aliased = append(aliased, imp.Path+name[len(first):])
		}
	}
	steps = append(steps, aliased, opened, []string{name})
	for _, step := range steps {
		var found []*csType
		for _, full := range step {
			if t := x.types[full]; t != nil && !slices.Contains(found, t) {
				found = append(found, t)
			}
		}
		switch len(found) {
		case 0:
			continue
		case 1:
			return found[0], ""
		}
		return nil, ReasonAmbiguous
	}
	return nil, ReasonNone
}

// call is the node one raw call lands on, or "" for none:
//
//   - new T(): the first constructor of T, else T itself
//   - M(): a member of the calling type or its bases, then of each type
//     around it with its bases, inside out, then of the one type a
//     `using static` of the file offers it in
//   - this.M(): the same without the usings; base.M(): the bases alone
//   - T.M(): a member of the type T names, or of its bases
//
// Anything else -- a receiver that is a local, a field, an expression --
// names no type this index can see, and gives none.
func (x *csIndex) call(raw extract.RawEdge) model.NodeID {
	own := x.typeOf[raw.Source]
	var self []*csType
	ns := ""
	if own != nil {
		self, ns = []*csType{own}, own.ns
	}
	switch {
	case raw.Member == "new":
		t, why := x.typeNamed(raw.File, own, ns, join(raw.Receiver, raw.Name))
		if why != "" {
			return ""
		}
		if ctor, ok := t.members[raw.Name]; ok {
			return ctor
		}
		return t.id
	case raw.Receiver == "":
		for t := own; t != nil; t = t.outer {
			if m := member([]*csType{t}, raw.Name); m != "" {
				return m
			}
		}
		var found []model.NodeID
		for _, imp := range x.usingsOf(raw.File) {
			if t := x.types[imp.Path]; imp.Alias == "static" && t != nil {
				// The same using in force twice, local and global, offers
				// the same member twice: that is no choice.
				if m := member([]*csType{t}, raw.Name); m != "" && !slices.Contains(found, m) {
					found = append(found, m)
				}
			}
		}
		if len(found) == 1 {
			return found[0]
		}
		return ""
	case raw.Receiver == "this":
		return member(self, raw.Name)
	case raw.Receiver == "base":
		if own == nil {
			return ""
		}
		return member(own.bases, raw.Name)
	}
	t, why := x.typeNamed(raw.File, own, ns, raw.Receiver)
	if why != "" {
		return ""
	}
	return member([]*csType{t}, raw.Name)
}

// member is the first declaration of name in the first type that has one:
// the types of start, then their bases breadth first, in the order written.
// Each type is visited once, so a cycle of bases ends the walk.
func member(start []*csType, name string) model.NodeID {
	queue := slices.Clone(start)
	visited := map[*csType]bool{}
	for len(queue) > 0 {
		t := queue[0]
		queue = queue[1:]
		if visited[t] {
			continue
		}
		visited[t] = true
		if m, ok := t.members[name]; ok {
			return m
		}
		queue = append(queue, t.bases...)
	}
	return ""
}

// usingsOf is every using in force in file: its own, and the global ones of
// every file, each once.
func (x *csIndex) usingsOf(file string) []extract.Import {
	return slices.Concat(x.usings[file], x.globals)
}

// parentNs is the namespace around ns, "" around a top-level one.
func parentNs(ns string) string {
	i := strings.LastIndexByte(ns, '.')
	if i < 0 {
		return ""
	}
	return ns[:i]
}

// join is name inside the dotted prefix, prefix "" leaving it alone.
func join(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
