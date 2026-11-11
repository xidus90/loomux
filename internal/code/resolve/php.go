package resolve

import (
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolvePHP resolves the raw edges of the PHP files against an index of
// those files alone, by PHP's own name resolution (resolveName):
// containment as the extractor resolved it, an import to the file it
// loads or the file declaring what it names (importTarget), a type to its
// bases, a call to the method, function or constructor it names (call).
//
// Only a base gives an Unresolved entry; a call or an import that finds
// nothing is dropped, as in C#.
func resolvePHP(files []extract.Result) langOut {
	x := phpIndexOf(files)
	var out []model.Edge
	var un []Unresolved
	var calls []extract.RawEdge
	// The position in out of each import edge, which keeps the smallest line
	// of the imports that make it.
	imported := map[[2]model.NodeID]int{}
	for _, f := range files {
		// The extractor makes one imports edge per import, in the same
		// order: the edge's specifier alone does not say what kind it is.
		next := 0
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				out = append(out, model.Edge{
					Source: raw.Source, Target: raw.TargetID,
					Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationImports:
				imp := f.Imports[next]
				next++
				key := [2]model.NodeID{raw.Source, model.NodeID(x.importTarget(f.Path, imp))}
				if key[1] == "" || key[1] == raw.Source {
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
				name := qualify(raw.Receiver, raw.Name)
				base := x.typeNamed(raw.File, t.ns, name)
				if base == nil {
					un = append(un, Unresolved{Source: raw.Source, File: raw.File, Line: raw.Line, Name: name, Reason: ReasonNone})
					continue
				}
				// Recorded for the lookups that walk a type's bases, which
				// therefore run after this loop. A trait's method overrides
				// an inherited one, so the traits go first.
				at := len(t.bases)
				if x.trait[base] {
					at = slices.IndexFunc(t.bases, func(b *csType) bool { return !x.trait[b] })
					if at < 0 {
						at = len(t.bases)
					}
				}
				t.bases = slices.Insert(t.bases, at, base)
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

// phpIndex is every lookup the PHP resolver makes, over the PHP files
// alone. A type is a csType whose full name is `Ns\Type`, without a
// leading backslash. PHP matches the names of types, methods and
// functions without regard to case, so every key below holding one is
// lower case, and resolveName returns one so.
type phpIndex struct {
	types  map[string]*csType          // full name -> the type in the smallest path
	trait  map[*csType]bool            // the types that are traits
	typeOf map[model.NodeID]*csType    // type or method -> its type
	funcs  map[string]model.NodeID     // full name -> the function in the smallest path
	nsOf   map[model.NodeID]string     // file, type or function -> its namespace
	uses   map[string][]extract.Import // file -> its imports
	files  map[string]bool             // every PHP file
}

// phpIndexOf indexes the PHP files in path order, so a type or function
// two files declare -- a polyfill, a conditional declaration -- is the one
// in the smallest path. The bases are left empty: resolvePHP records them
// as it resolves the extends edges.
func phpIndexOf(files []extract.Result) *phpIndex {
	x := &phpIndex{
		types: map[string]*csType{}, trait: map[*csType]bool{}, typeOf: map[model.NodeID]*csType{},
		funcs: map[string]model.NodeID{}, nsOf: map[model.NodeID]string{},
		uses: map[string][]extract.Import{}, files: map[string]bool{},
	}
	sorted := slices.Clone(files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	for _, f := range sorted {
		x.files[f.Path], x.uses[f.Path] = true, f.Imports
		// Code outside every declaration runs in the namespace of the
		// file's first one.
		x.nsOf[model.NodeID(f.Path)] = f.Package
		// The file's types by name, for its methods: an owner names its
		// type without the namespace.
		local := map[string]*csType{}
		for _, n := range f.Nodes {
			ns := f.Enclosing[n.ID]
			switch n.Kind {
			case model.KindFile:
			case "class", "interface":
				x.nsOf[n.ID] = ns
				full := strings.ToLower(phpJoin(ns, n.Name))
				t := &csType{id: n.ID, full: full, ns: ns, iface: n.Kind == "interface", members: map[string]model.NodeID{}}
				if _, ok := x.types[full]; !ok {
					x.types[full] = t
				}
				// The extractor gives a trait the kind class; its header
				// still says what it is.
				x.trait[t] = strings.HasPrefix(n.Signature, "trait ")
				x.typeOf[n.ID], local[n.Name] = t, t
			case "function":
				x.nsOf[n.ID] = ns
				if full := strings.ToLower(phpJoin(ns, n.Name)); x.funcs[full] == "" {
					x.funcs[full] = n.ID
				}
			default:
				t := local[n.Owner]
				x.typeOf[n.ID], x.nsOf[n.ID] = t, t.ns
				if name := strings.ToLower(n.Name); t.members[name] == "" {
					t.members[name] = n.ID
				}
			}
		}
	}
	return x
}

// importTarget is the file one import of file points at, "" for none:
//
//   - use A\B: the file declaring the type A\B
//   - use function A\f: the file declaring the function A\f
//   - require, include: the file the path names, relative to file
//
// A use of a namespace, of a constant or of a name no file here declares
// gives none.
func (x *phpIndex) importTarget(file string, imp extract.Import) string {
	var id model.NodeID
	switch imp.Kind {
	case "":
		if t := x.types[strings.ToLower(imp.Path)]; t != nil {
			id = t.id
		}
	case "function":
		id = x.funcs[strings.ToLower(imp.Path)]
	case "require":
		if p := path.Join(path.Dir(file), imp.Path); x.files[p] {
			return p
		}
	}
	return fileOf(id)
}

// funcNamed is the function a function name written in file, in namespace
// ns, stands for, "" for none: the name resolveName resolves through the
// `use function` imports, and an unqualified one neither imported nor in
// ns in the global namespace.
func (x *phpIndex) funcNamed(file, ns, name string) model.NodeID {
	full, inNs := x.resolveName(file, ns, name, "function")
	if id := x.funcs[full]; id != "" || !inNs || strings.Contains(name, `\`) {
		return id
	}
	return x.funcs[strings.ToLower(name)]
}

// typeNamed is the type a class name written in file, in namespace ns,
// stands for, nil for none.
func (x *phpIndex) typeNamed(file, ns, name string) *csType {
	full, _ := x.resolveName(file, ns, name, "")
	return x.types[full]
}

// resolveName is the full name, lower case, a name written in file, in
// namespace ns, stands for, as PHP resolves it; kind is the kind of `use`
// an unqualified name goes through, "" for a class:
//
//   - \A\B is A\B
//   - namespace\B is B in ns
//   - B, imported by a `use` of kind, is what it imports
//   - A\B whose A a class `use` imports or aliases is B in what it imports
//   - anything else is in ns, which inNs reports: a class never leaves it,
//     an unqualified function falls back to the global namespace
func (x *phpIndex) resolveName(file, ns, name, kind string) (full string, inNs bool) {
	if rest, ok := strings.CutPrefix(name, `\`); ok {
		return strings.ToLower(rest), false
	}
	first, rest, qualified := strings.Cut(name, `\`)
	if qualified && strings.EqualFold(first, "namespace") {
		return strings.ToLower(phpJoin(ns, rest)), false
	}
	if !qualified {
		rest = ""
	} else {
		// A qualified name's first part is a namespace or class alias,
		// whatever the name is.
		kind = ""
	}
	for _, imp := range x.uses[file] {
		alias := imp.Alias
		if alias == "" {
			alias = imp.Path[strings.LastIndexByte(imp.Path, '\\')+1:]
		}
		if imp.Kind == kind && strings.EqualFold(alias, first) {
			return strings.ToLower(phpJoin(imp.Path, rest)), false
		}
	}
	return strings.ToLower(phpJoin(ns, name)), true
}

// call is the node one raw call lands on, or "" for none:
//
//   - new X(): the __construct of X or of its bases, else X itself
//   - $this->m(), self::m(), static::m(): a method of the calling type,
//     its traits or its bases
//   - parent::m(): a method of the bases but the traits
//   - X::m(): a method of the type X names or of its bases
//   - f(), X\f(): the function funcNamed finds
//
// A call through an object, $obj->m(), names no type this index can see,
// and gives none. Only a class name, never a variable, is matched without
// regard to case.
func (x *phpIndex) call(raw extract.RawEdge) model.NodeID {
	own := x.typeOf[raw.Source]
	ns := x.nsOf[raw.Source]
	scope := strings.ToLower(raw.Receiver)
	switch raw.Member {
	case ".":
		return ""
	case "new":
		t := own
		if scope != "" || !slices.Contains([]string{"self", "static"}, strings.ToLower(raw.Name)) {
			t = x.typeNamed(raw.File, ns, qualify(raw.Receiver, raw.Name))
		}
		if t == nil {
			return ""
		}
		if ctor := member([]*csType{t}, "__construct"); ctor != "" {
			return ctor
		}
		return t.id
	case "":
		if raw.Receiver != "$this" {
			return x.funcNamed(raw.File, ns, qualify(raw.Receiver, raw.Name))
		}
		scope = "self"
	}
	var start []*csType
	switch {
	case own != nil && (scope == "self" || scope == "static"):
		start = []*csType{own}
	case own != nil && scope == "parent":
		start = slices.DeleteFunc(slices.Clone(own.bases), func(b *csType) bool { return x.trait[b] })
	case scope != "self" && scope != "static" && scope != "parent":
		if t := x.typeNamed(raw.File, ns, raw.Receiver); t != nil {
			start = []*csType{t}
		}
	}
	return member(start, strings.ToLower(raw.Name))
}

// qualify is a name as written from the receiver and name the extractor
// split it into: the receiver `\` is the global namespace.
func qualify(receiver, name string) string {
	switch receiver {
	case "":
		return name
	case `\`:
		return `\` + name
	}
	return receiver + `\` + name
}

// phpJoin is name inside the namespace ns, ns "" leaving it alone.
func phpJoin(ns, name string) string {
	if ns == "" {
		return name
	}
	if name == "" {
		return ns
	}
	return ns + `\` + name
}

// fileOf is the file a node id belongs to, "" for no node.
func fileOf(id model.NodeID) string {
	file, _, _ := strings.Cut(string(id), "#")
	return file
}
