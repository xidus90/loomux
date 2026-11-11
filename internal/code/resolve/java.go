package resolve

import (
	"slices"
	"sort"
	"strings"

	"github.com/xidus90/loomux/internal/code/blast"
	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolveJava resolves the raw edges of the Java files against an index of
// those files alone: containment as the extractor resolved it, an import to
// the file of the type or package it names (importTarget), a type to its
// bases, a call to the method or constructor it names (call).
//
// A package is spread over many files, as a C# namespace is, so an
// on-demand import points at one file standing for it; an import of a type
// or package no file here declares -- java.util -- gives no edge, and
// neither does `import module`.
//
// Only a base gives an Unresolved entry; a call that finds nothing is
// dropped, as in C#.
func resolveJava(files []extract.Result) langOut {
	x := javaIndexOf(files)
	var out []model.Edge
	var un []Unresolved
	var calls []extract.RawEdge
	// The position in out of each import edge, which keeps the smallest line
	// of the imports that make it.
	imported := map[[2]model.NodeID]int{}
	for _, f := range files {
		// The extractor makes one imports edge per import, in the same
		// order: the edge's specifier alone does not say whether it is static.
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
				target := x.importTarget(imp)
				key := [2]model.NodeID{raw.Source, model.NodeID(target)}
				if target == "" || key[1] == raw.Source {
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
				// A base clause is read outside the type it belongs to: its
				// own nested types are not in scope there.
				name := join(raw.Receiver, raw.Name)
				base, why := x.typeNamed(raw.File, t.outer, name)
				if why != "" {
					un = append(un, Unresolved{Source: raw.Source, File: raw.File, Line: raw.Line, Name: name, Reason: why})
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

// javaIndex is every lookup the Java resolver makes, over the Java files
// alone. A type is a csType whose full name is "pkg.Outer.Inner", ns its
// package; the files without a package form the package "".
type javaIndex struct {
	types     map[string]*csType                 // full name -> the type in the smallest path
	fileTypes map[string]map[string]*csType      // file -> full name -> its own type
	typeOf    map[model.NodeID]*csType           // type or member -> its type
	pkg       map[string]string                  // file -> its package
	pkgFile   map[string]string                  // package -> the file standing for it
	imports   map[string][]extract.Import        // file -> its imports
	funcs     map[string]map[string]model.NodeID // file -> top-level method -> its first overload
}

// javaIndexOf indexes the Java files in path order, so a type two files
// declare is the one in the smallest path, and the first declaration of an
// overload the first in source order. The other file keeps its own type:
// two modules may each hold a class of one name, and each sees its own.
// The bases are left empty: resolveJava records them as it resolves the
// extends edges.
//
// The file standing for a package is the one with the smallest path among
// its non-test files, else among all of them, as in C#.
func javaIndexOf(files []extract.Result) *javaIndex {
	x := &javaIndex{
		types: map[string]*csType{}, fileTypes: map[string]map[string]*csType{},
		typeOf: map[model.NodeID]*csType{}, pkg: map[string]string{},
		pkgFile: map[string]string{}, imports: map[string][]extract.Import{},
		funcs: map[string]map[string]model.NodeID{},
	}
	sorted := slices.Clone(files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	for _, f := range sorted {
		x.pkg[f.Path], x.imports[f.Path] = f.Package, f.Imports
		if cur, ok := x.pkgFile[f.Package]; !ok || blast.IsTestPath(cur) && !blast.IsTestPath(f.Path) {
			x.pkgFile[f.Package] = f.Path
		}
		// The file's types by qualified name, for its members and nested
		// types: an owner names its type without the package.
		local := map[string]*csType{}
		own := map[string]*csType{}
		x.fileTypes[f.Path] = own
		funcs := map[string]model.NodeID{}
		x.funcs[f.Path] = funcs
		for _, n := range f.Nodes {
			switch n.Kind {
			case "function":
				if _, ok := funcs[n.Name]; !ok {
					funcs[n.Name] = n.ID
				}
			case "class", "interface":
				qualified := join(n.Owner, n.Name)
				t := &csType{
					id: n.ID, full: join(f.Package, qualified), ns: f.Package, iface: n.Kind == "interface",
					outer: local[n.Owner], members: map[string]model.NodeID{},
				}
				if x.types[t.full] == nil {
					x.types[t.full] = t
				}
				own[t.full], local[qualified], x.typeOf[n.ID] = t, t, t
			case "method":
				t := local[n.Owner]
				x.typeOf[n.ID] = t
				if _, ok := t.members[n.Name]; !ok {
					t.members[n.Name] = n.ID
				}
			}
		}
	}
	return x
}

// typeNamed is the type a dotted name stands for in file, written inside
// the type scope (nil outside every type). The first step that finds any
// type decides; two types found there are a guess:
//
//  1. a type nested in scope, then in each type around it, inside out
//  2. a type of the file
//  3. through a single-type import, `import a.b.N;`
//  4. a type of the file's package
//  5. through an on-demand import, `import a.b.*;`
//  6. the name as a full one, if it has a dot: no import reaches the
//     package ""
//
// A type nested in a base of scope is not looked for.
func (x *javaIndex) typeNamed(file string, scope *csType, name string) (*csType, Reason) {
	own := x.fileTypes[file]
	// scope and the types around it are the file's own, and so are the
	// types nested in them.
	for t := scope; t != nil; t = t.outer {
		if n := own[t.full+"."+name]; n != nil {
			return n, ""
		}
	}
	inPkg := join(x.pkg[file], name)
	if t := own[inPkg]; t != nil {
		return t, ""
	}
	first, _, _ := strings.Cut(name, ".")
	var single, star []string
	for _, imp := range x.imports[file] {
		switch {
		case imp.Alias != "":
		case strings.HasSuffix(imp.Path, ".*"):
			star = append(star, strings.TrimSuffix(imp.Path, "*")+name)
		case imp.Path == first || strings.HasSuffix(imp.Path, "."+first):
			single = append(single, imp.Path+name[len(first):])
		}
	}
	steps := [][]string{single, {inPkg}, star}
	if first != name {
		steps = append(steps, []string{name})
	}
	return pick(steps, nil, x.types)
}

// importTarget is the file one import points at, or "" for none:
//
//   - import a.b.C: the file of C
//   - import a.b.*: the file standing for the package a.b, else the file of
//     a type a.b, whose nested types it imports
//   - import static a.b.C.m and a.b.C.*: the file of C
//   - import module: none, as no module names a file
func (x *javaIndex) importTarget(imp extract.Import) string {
	p, last := splitLast(imp.Path)
	switch {
	case imp.Alias == "module":
		return ""
	case imp.Alias == "" && last != "*":
		p = imp.Path
	case imp.Alias == "":
		if f, ok := x.pkgFile[p]; ok {
			return f
		}
	}
	t := x.types[p]
	if t == nil {
		return ""
	}
	file, _, _ := strings.Cut(string(t.id), "#")
	return file
}

// call is the node one raw call lands on, or "" for none:
//
//   - new T(): the first constructor of T, else T itself
//   - m(): a member of the calling type or its bases, then of each type
//     around it with its bases, inside out, then a top-level method of a
//     compact source file, whose implicit class holds every type the file
//     declares, then of the one type a single static import offers it in,
//     then an on-demand one
//   - this.m(): the calling type or its bases; super.m(): the bases alone
//   - T.m(), a.b.T.m(): a member of the type T names, or of its bases
//
// obj.m(), marked ".", names an object and gives none.
func (x *javaIndex) call(raw extract.RawEdge) model.NodeID {
	own := x.typeOf[raw.Source]
	switch {
	case raw.Member == ".":
		return ""
	case raw.Member == "new":
		t, why := x.typeNamed(raw.File, own, join(raw.Receiver, raw.Name))
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
		if f, ok := x.funcs[raw.File][raw.Name]; ok {
			return f
		}
		return x.staticImport(raw.File, raw.Name)
	case own == nil && (raw.Receiver == "this" || raw.Receiver == "super"):
		return ""
	case raw.Receiver == "this":
		return member([]*csType{own}, raw.Name)
	case raw.Receiver == "super":
		return member(own.bases, raw.Name)
	}
	t, why := x.typeNamed(raw.File, own, raw.Receiver)
	if why != "" {
		return ""
	}
	return member([]*csType{t}, raw.Name)
}

// staticImport is the member name a static import of file offers: a single
// one, `import static a.C.name`, then an on-demand one, `import static
// a.C.*`. Two types offering it on one step are a guess.
func (x *javaIndex) staticImport(file, name string) model.NodeID {
	for _, last := range []string{name, "*"} {
		var found []model.NodeID
		for _, imp := range x.imports[file] {
			p, l := splitLast(imp.Path)
			t := x.types[p]
			if imp.Alias != "static" || l != last || t == nil {
				continue
			}
			if m := member([]*csType{t}, name); m != "" && !slices.Contains(found, m) {
				found = append(found, m)
			}
		}
		switch len(found) {
		case 0:
			continue
		case 1:
			return found[0]
		}
		return ""
	}
	return ""
}

// splitLast is a dotted path's head and its last name; the head of a plain
// name is "".
func splitLast(path string) (head, last string) {
	i := strings.LastIndexByte(path, '.')
	return path[:max(i, 0)], path[i+1:]
}
