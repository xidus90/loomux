package resolve

import (
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolveCPP resolves the raw edges of the C++ files against an index of
// those files alone: containment as the extractor resolved it, an include to
// the file it names (include), a type to its bases, a call to the function,
// member or constructor it names (call), and a Godot binding to the method
// it binds.
//
// A binding is also offered across the bridge, under the Godot name and the
// class it belongs to: a scene names the class of its node, not a file.
//
// Only a base gives an Unresolved entry; anything else that finds nothing is
// dropped, as in C#.
func resolveCPP(files []extract.Result) langOut {
	x := cppIndexOf(files)
	var out []model.Edge
	var un []Unresolved
	var offers []offer
	var later []extract.RawEdge
	// The position in out of each include edge, which keeps the smallest
	// line of the includes that make it.
	included := map[[2]model.NodeID]int{}
	for _, f := range files {
		quoted := map[string]bool{}
		for _, imp := range f.Imports {
			quoted[imp.Path] = quoted[imp.Path] || imp.Alias == "" && !imp.System
		}
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				out = append(out, model.Edge{
					Source: raw.Source, Target: raw.TargetID,
					Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationImports:
				target := x.include(raw.File, raw.Specifier, quoted[raw.Specifier])
				if target == "" {
					continue
				}
				key := [2]model.NodeID{raw.Source, model.NodeID(target)}
				if i, seen := included[key]; seen {
					out[i].Line = min(out[i].Line, raw.Line)
					continue
				}
				included[key] = len(out)
				out = append(out, model.Edge{
					Source: raw.Source, Target: key[1],
					Relation: model.RelationImports, Confidence: model.ConfidenceExtracted, Line: raw.Line,
				})
			case model.RelationExtends:
				t := x.typeOf[raw.Source]
				// A base clause is read outside the class it belongs to:
				// the class's own nested types are not in scope there.
				name := ccJoin(raw.Receiver, raw.Name)
				base, why := pick(x.steps(raw.File, t.outer, t.ns, name), x.localTypes[raw.File], x.types)
				if why != "" {
					un = append(un, Unresolved{Source: raw.Source, File: raw.File, Line: raw.Line, Name: name, Reason: why})
					continue
				}
				// Recorded for the lookups that walk a type's bases, which
				// therefore run after this loop.
				t.bases = append(t.bases, base)
				out = append(out, model.Edge{
					Source: raw.Source, Target: base.id,
					Relation: model.RelationExtends, Confidence: model.ConfidenceExtracted, Line: raw.Line,
				})
			default:
				later = append(later, raw)
			}
		}
	}
	for _, raw := range later {
		var target model.NodeID
		switch raw.Member {
		case model.Kind(extract.NamespaceGodotMethod):
			own := x.typeOf[raw.Source]
			t, why := pick(x.steps(raw.File, own, x.nsOf(raw.Source), raw.Receiver), x.localTypes[raw.File], x.types)
			if why != "" {
				continue
			}
			if target = member([]*csType{t}, raw.Name); target == "" {
				continue
			}
			offers = append(offers, x.offer(raw, t, target))
		case model.Kind(extract.NamespaceGodotSignal):
			// The class of _bind_methods, found as its methods are.
			if t := x.typeOf[raw.Source]; t != nil {
				offers = append(offers, x.offer(raw, t, t.id))
			}
			continue
		default:
			if target = x.call(raw); target == "" {
				continue
			}
		}
		out = append(out, model.Edge{
			Source: raw.Source, Target: target,
			Relation: raw.Relation, Confidence: model.ConfidenceExtracted, Line: raw.Line,
		})
	}
	return langOut{edges: out, unresolved: un, offers: offers}
}

// cppIndex is every lookup the C++ resolver makes, over the C++ files alone.
// Names in it are joined with "::", as C++ writes them; a type is a csType
// whose full name is "ns::Outer::Inner".
//
// A static function and a type in an anonymous namespace, with the types
// nested in it, have internal linkage: they sit in their file's own tables,
// which answer that file alone and, on each lookup step, before the shared
// ones.
type cppIndex struct {
	types      map[string]*csType                 // full name -> the type
	localTypes map[string]map[string]*csType      // file -> full name -> its own type
	typeOf     map[model.NodeID]*csType           // type or method -> its type
	funcs      map[string]model.NodeID            // full name of a free function -> the first definition
	localFuncs map[string]map[string]model.NodeID // file -> full name -> its own function
	ns         map[model.NodeID]string            // free function -> its namespace
	usings     map[string][]string                // file -> the namespaces it opens
	paths      []string                           // every C++ file, by path
	isPath     map[string]bool
}

// cppIndexOf indexes the C++ files in path order, so a type or function
// defined twice is the one in the smallest path, and a member declared
// twice -- an overload -- the first in source order.
//
// Types come first: a method defined outside its class finds the class over
// its qualifier (classOf), and the class may live in any file.
func cppIndexOf(files []extract.Result) *cppIndex {
	x := &cppIndex{
		types: map[string]*csType{}, typeOf: map[model.NodeID]*csType{},
		funcs: map[string]model.NodeID{}, ns: map[model.NodeID]string{},
		usings: map[string][]string{}, isPath: map[string]bool{},
		localTypes: map[string]map[string]*csType{}, localFuncs: map[string]map[string]model.NodeID{},
	}
	internal := map[*csType]bool{}
	sorted := slices.Clone(files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	parent := map[model.NodeID]model.NodeID{}
	// Each class node's dotted name, as an owner writes it.
	dotted := map[model.NodeID]string{}
	// owned is the type of the class node holding id, when that class is
	// the owner: a class whose body is broken has no node, and what it
	// holds belongs to no type.
	owned := func(id model.NodeID, owner string) *csType {
		if p := parent[id]; owner != "" && dotted[p] == owner {
			return x.typeOf[p]
		}
		return nil
	}
	for _, f := range sorted {
		x.paths, x.isPath[f.Path] = append(x.paths, f.Path), true
		for _, imp := range f.Imports {
			if imp.Alias == "namespace" {
				x.usings[f.Path] = append(x.usings[f.Path], imp.Path)
			}
		}
		for _, raw := range f.Edges {
			if raw.Relation == model.RelationContains {
				parent[raw.TargetID] = raw.Source
			}
		}
		x.localTypes[f.Path], x.localFuncs[f.Path] = map[string]*csType{}, map[string]model.NodeID{}
		for _, n := range f.Nodes {
			if n.Kind != "class" {
				continue
			}
			ns := f.Enclosing[n.ID]
			full := ccJoin(ns, ccJoin(strings.ReplaceAll(n.Owner, ".", "::"), n.Name))
			outer := owned(n.ID, n.Owner)
			// A nested class's Exported is its access, not its linkage.
			own := n.Owner == "" && !n.Exported || internal[outer]
			table := x.types
			if own {
				table = x.localTypes[f.Path]
			}
			t := table[full]
			if t == nil {
				t = &csType{id: n.ID, full: full, ns: ns, outer: outer, members: map[string]model.NodeID{}}
				table[full], internal[t] = t, own
			}
			x.typeOf[n.ID], dotted[n.ID] = t, join(n.Owner, n.Name)
		}
	}
	for _, f := range sorted {
		for _, n := range f.Nodes {
			var t *csType
			qualifier, outside := f.Enclosing[n.ID]
			switch {
			case n.Kind == "method" && outside:
				if t = x.classOf(f.Path, qualifier); t == nil {
					// A qualifier no class answers is a namespace, ns::f.
					x.function(f.Path, n, qualifier)
					continue
				}
			case n.Kind == "method":
				if t = owned(n.ID, n.Owner); t == nil {
					continue
				}
			case n.Kind == "function":
				x.function(f.Path, n, qualifier)
				continue
			default:
				continue
			}
			x.typeOf[n.ID] = t
			if _, ok := t.members[n.Name]; !ok {
				t.members[n.Name] = n.ID
			}
		}
	}
	return x
}

// function indexes a free function of file in namespace ns; one not
// exported has internal linkage.
func (x *cppIndex) function(file string, n model.Node, ns string) {
	x.ns[n.ID] = ns
	table := x.funcs
	if !n.Exported {
		table = x.localFuncs[file]
	}
	if full := ccJoin(ns, n.Name); table[full] == "" {
		table[full] = n.ID
	}
}

// classOf is the class a method defined outside it belongs to. qualifier
// is the namespace the definition stands in and the qualifier it writes,
// joined, and which part is which the extractor cannot tell: the longest
// tail of it that names a class decides.
//
// ponytail: a tail shorter than the written qualifier can land on a class
// of the same name elsewhere when the real one is not in the repository.
func (x *cppIndex) classOf(file, qualifier string) *csType {
	segs := strings.Split(qualifier, "::")
	for i := range segs {
		if t, why := pick(x.steps(file, nil, "", strings.Join(segs[i:], "::")), x.localTypes[file], x.types); why == "" {
			return t
		}
	}
	return nil
}

// nsOf is the namespace code in source runs in: its class's, else the
// free function's, else the global one.
func (x *cppIndex) nsOf(source model.NodeID) string {
	if t := x.typeOf[source]; t != nil {
		return t.ns
	}
	return x.ns[source]
}

// steps are the full names name may stand for in file, written inside the
// type scope (nil outside every type) in namespace ns, one step after the
// other:
//
//  1. nested in scope, then in each type around it, inside out
//  2. in ns, then in each namespace around it, inside out
//  3. in a namespace the file opens with `using namespace`
//  4. in the global namespace
func (x *cppIndex) steps(file string, scope *csType, ns, name string) [][]string {
	var steps [][]string
	for t := scope; t != nil; t = t.outer {
		steps = append(steps, []string{t.full + "::" + name})
	}
	for s := ns; s != ""; s = parentScope(s) {
		steps = append(steps, []string{s + "::" + name})
	}
	var opened []string
	for _, u := range x.usings[file] {
		opened = append(opened, u+"::"+name)
	}
	return append(steps, opened, []string{name})
}

// pick is the one entry the first step that finds any names, each step in
// the file's own table and only then in the shared one; two found in one
// table are a guess.
func pick[T comparable](steps [][]string, own, shared map[string]T) (T, Reason) {
	var none T
	for _, step := range steps {
		for _, table := range []map[string]T{own, shared} {
			var found []T
			for _, full := range step {
				if v, ok := table[full]; ok && !slices.Contains(found, v) {
					found = append(found, v)
				}
			}
			switch len(found) {
			case 0:
				continue
			case 1:
				return found[0], ""
			}
			return none, ReasonAmbiguous
		}
	}
	return none, ReasonNone
}

// call is the node one raw call lands on, or "" for none:
//
//   - new T(): the first constructor of T, else T itself
//   - f(): a member of the calling class or its bases, then of each class
//     around it, then a free function by steps; else f is a type, as in
//     T(), and that is new T()
//   - this->f(): a member of the calling class or its bases
//   - A::f(): a member of the class A names, else a free function in the
//     namespace A names
//
// obj.f() and p->f(), marked ".", name an object and give none.
func (x *cppIndex) call(raw extract.RawEdge) model.NodeID {
	own, ns := x.typeOf[raw.Source], x.nsOf(raw.Source)
	switch {
	case raw.Member == ".":
		return ""
	case raw.Member == "new":
		return x.construct(raw.File, own, ns, raw.Receiver, raw.Name)
	case raw.Receiver == "":
		for t := own; t != nil; t = t.outer {
			if m := member([]*csType{t}, raw.Name); m != "" {
				return m
			}
		}
		if f, why := pick(x.steps(raw.File, nil, ns, raw.Name), x.localFuncs[raw.File], x.funcs); why == "" {
			return f
		}
		return x.construct(raw.File, own, ns, "", raw.Name)
	case raw.Receiver == "this":
		if own == nil {
			return ""
		}
		return member([]*csType{own}, raw.Name)
	}
	if t, why := pick(x.steps(raw.File, own, ns, raw.Receiver), x.localTypes[raw.File], x.types); why == "" {
		return member([]*csType{t}, raw.Name)
	}
	f, _ := pick(x.steps(raw.File, nil, ns, ccJoin(raw.Receiver, raw.Name)), x.localFuncs[raw.File], x.funcs)
	return f
}

// construct is the first constructor of the type qualifier::name, else the
// type, else "".
func (x *cppIndex) construct(file string, own *csType, ns, qualifier, name string) model.NodeID {
	t, why := pick(x.steps(file, own, ns, ccJoin(qualifier, name)), x.localTypes[file], x.types)
	if why != "" {
		return ""
	}
	if ctor, ok := t.members[name]; ok {
		return ctor
	}
	return t.id
}

// offer is the binding raw offered as a member of class t, landing on
// target.
func (x *cppIndex) offer(raw extract.RawEdge, t *csType, target model.NodeID) offer {
	segs := strings.Split(t.full, "::")
	return offer{
		Export: extract.Export{Namespace: extract.Namespace(raw.Member), Name: raw.Specifier, Class: segs[len(segs)-1], Target: target},
		file:   raw.File, lang: "cpp",
	}
}

// include is the file an include of spec in file names, or "" for none:
// for a quoted one the file next to it first, then the one file whose path
// ends in spec over whole segments, or of several the one with the fewest
// segments, if that is one.
func (x *cppIndex) include(file, spec string, quoted bool) string {
	if p := path.Join(dirOf(file), spec); quoted && x.isPath[p] {
		return p
	}
	best, depth, tie := "", 0, false
	for _, p := range x.paths {
		if p != spec && !strings.HasSuffix(p, "/"+spec) {
			continue
		}
		switch d := strings.Count(p, "/"); {
		case best == "" || d < depth:
			best, depth, tie = p, d, false
		case d == depth:
			tie = true
		}
	}
	if tie {
		return ""
	}
	return best
}

// parentScope is the namespace around ns, "" around a top-level one.
func parentScope(ns string) string {
	i := strings.LastIndex(ns, "::")
	if i < 0 {
		return ""
	}
	return ns[:i]
}

// ccJoin is name inside the "::"-joined prefix, prefix "" leaving it alone.
func ccJoin(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "::" + name
}
