package resolve

import (
	"path"
	"slices"
	"strings"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/model"
)

// resolveRust resolves the raw edges of the Rust files against their module
// trees: containment as the extractor resolved it, `mod x;` to the file of
// the module and a use to the file of what it binds (imports), a path to the
// item it names and `Type::f` to a method of an impl of Type (calls), and an
// impl and a supertrait bound to the trait (implements, extends).
//
// A call bound by path, use or impl type is extracted; one bound by a name
// unique in the crate alone -- x.f(), a macro the module does not see, a call
// read from a macro's arguments -- is inferred.
//
// A godot-rust binding is offered across the bridge, under the Godot name and
// the class its impl type registers, wherever in the crate that is.
//
// Only an impl or a bound gives an Unresolved entry; anything else that finds
// nothing is dropped, as in C++.
func resolveRust(files []extract.Result, mods []Module) langOut {
	x := rustIndexOf(files, mods)
	x.settle(x.order)
	return x.resolve(files)
}

// resolve turns the raw edges of files into edges, unresolved bases and
// offers, reading the module tables.
func (x *rustIndex) resolve(files []extract.Result) langOut {
	var out []model.Edge
	var un []Unresolved
	var offers []offer
	var later []extract.RawEdge
	for _, f := range files {
		imported := map[model.NodeID]bool{}
		for _, raw := range f.Edges {
			switch raw.Relation {
			case model.RelationContains:
				out = append(out, model.Edge{
					Source: raw.Source, Target: raw.TargetID,
					Relation: model.RelationContains, Confidence: model.ConfidenceExtracted,
				})
			case model.RelationImports:
				if target, ok := x.declared[raw]; ok {
					imported[model.NodeID(target)] = true
					out = append(out, model.Edge{
						Source: raw.Source, Target: model.NodeID(target),
						Relation: model.RelationImports, Confidence: model.ConfidenceExtracted, Line: raw.Line,
					})
				}
			case model.RelationImplements, model.RelationExtends:
				e, u, ok := x.relation(raw)
				if !ok {
					un = append(un, u)
					continue
				}
				out = append(out, e)
			default:
				later = append(later, raw)
			}
		}
		out = append(out, x.uses(f.Path, imported)...)
	}
	// Calls run after every impl is read: self.f() may land on a default of
	// a trait the type implements.
	for _, raw := range later {
		if raw.Relation == model.RelationReferences {
			if o, ok := x.offer(raw); ok {
				offers = append(offers, o)
			}
			continue
		}
		if target, conf := x.call(raw); target != "" {
			out = append(out, model.Edge{Source: raw.Source, Target: target, Relation: model.RelationCalls, Confidence: conf, Line: raw.Line})
		}
	}
	return langOut{edges: out, unresolved: un, offers: offers}
}

// rsUnit is one crate as rustc compiles it: a library, a binary, a test,
// with the tree under its root. Methods, types and macros are indexed per
// unit, because an impl may stand in any module of its crate.
type rsUnit struct {
	root    *rsMod
	names   map[string]string                    // crate name in code -> its package's dir
	methods map[string]map[string][]model.NodeID // impl type -> name -> methods
	named   map[string][]model.NodeID            // name -> methods of every type
	types   map[string][]model.NodeID            // name -> types
	macros  map[string][]model.NodeID            // name -> macro_rules!
	traits  map[string][]model.NodeID            // impl type -> the traits it implements
}

// rsMod is one module: a file's own, or one written inline in it.
type rsMod struct {
	unit     *rsUnit
	file     string
	parent   *rsMod
	name     string // its segment, "" at a root
	inline   bool
	modRS    bool // a file whose modules sit beside it: a root, a mod.rs, one named by #[path]
	items    map[string][]model.NodeID
	children map[string]*rsMod
}

// rsDef is what a name stands for: a module or an item.
type rsDef struct {
	mod *rsMod
	id  model.NodeID
}

// rsWant is the namespace a name is looked up in.
type rsWant int

const (
	rsAny   rsWant = iota // a use binds whatever the name is
	rsType                // a segment before the last: a module or a type
	rsValue               // a callee: a function, or a struct as its constructor
	rsMacro
)

// rustIndex is every lookup the Rust resolver makes, over the Rust files
// alone.
type rustIndex struct {
	nodes     map[model.NodeID]model.Node
	fileMod   map[string]*rsMod             // file -> its own module
	decls     map[string][]extract.RawEdge  // file -> its mod x;
	declared  map[extract.RawEdge]string    // a mod x; -> the file of x
	imports   map[string][]extract.Import   // file -> its uses
	libs      map[string]*rsUnit            // package dir -> its library
	class     map[model.NodeID]string       // struct -> the Godot class it registers
	bind      map[*rsMod]map[string][]int   // file module -> name -> its explicit uses that bind it
	selfReads map[rsUse]bool                // an explicit use that names its own file's name back (selfReading)
	useT      map[rsUse]rsEntry             // an explicit use -> what its path names (settle)
	glob      map[rsView]map[rsName]rsEntry // a file module's glob tier (settle)
	order     []*rsMod                      // the file modules by path
	watch     rsUse                         // the use whose path is being resolved (past)
}

// rsUse is the i-th use of a file module.
type rsUse struct {
	m *rsMod
	i int
}

// rsPackage is a Cargo package: its dir, the names its code calls crates by,
// and its roots.
type rsPackage struct {
	dir     string
	names   map[string]string
	libRoot string
	roots   []string
}

// rustIndexOf builds the module tree of every crate root, the library's
// first, and then of every file no tree reached, each a root of its own; a
// file reached twice belongs to the first tree.
//
// ponytail: a file shared by two trees (lib and bin both declaring mod x;)
// resolves its paths in the first one alone.
func rustIndexOf(files []extract.Result, mods []Module) *rustIndex {
	x := &rustIndex{
		nodes: map[model.NodeID]model.Node{}, fileMod: map[string]*rsMod{},
		decls: map[string][]extract.RawEdge{}, declared: map[extract.RawEdge]string{},
		imports: map[string][]extract.Import{}, libs: map[string]*rsUnit{}, class: map[model.NodeID]string{},
		bind: map[*rsMod]map[string][]int{}, selfReads: map[rsUse]bool{}, useT: map[rsUse]rsEntry{}, glob: map[rsView]map[rsName]rsEntry{},
	}
	sorted := slices.Clone(files)
	slices.SortFunc(sorted, func(a, b extract.Result) int { return strings.Compare(a.Path, b.Path) })
	for _, f := range sorted {
		x.imports[f.Path] = f.Imports
		for _, n := range f.Nodes {
			x.nodes[n.ID] = n
		}
		for _, raw := range f.Edges {
			if raw.Relation == model.RelationImports {
				x.decls[f.Path] = append(x.decls[f.Path], raw)
			}
		}
		for _, e := range f.Exports {
			if e.Namespace == extract.NamespaceGodotClass {
				x.class[e.Target] = e.Name
			}
		}
	}
	pkgs := rsPackages(sorted, mods)
	for _, p := range pkgs {
		for _, root := range p.roots {
			if x.fileMod[root] != nil {
				continue
			}
			u := newUnit(p.names)
			x.claim(root, u, nil, "", true)
			if root == p.libRoot {
				x.libs[p.dir] = u
			}
		}
	}
	// Most files no tree reaches are modules a macro declares; their own
	// modules sit where their name puts them, as for any module file.
	for _, f := range sorted {
		if x.fileMod[f.Path] == nil {
			base := path.Base(f.Path)
			x.claim(f.Path, newUnit(nearest(pkgs, f.Path)), nil, "", base == "mod.rs" || base == "lib.rs" || base == "main.rs")
		}
	}
	for _, f := range sorted {
		m := x.fileMod[f.Path]
		x.order = append(x.order, m)
		x.bind[m] = map[string][]int{}
		for i, imp := range f.Imports {
			if imp.Name != "*" {
				x.bind[m][rsBinds(imp)] = append(x.bind[m][rsBinds(imp)], i)
			}
		}
		u := m.unit
		for _, n := range f.Nodes[1:] {
			if n.Kind == "method" {
				if u.methods[n.Owner] == nil {
					u.methods[n.Owner] = map[string][]model.NodeID{}
				}
				u.methods[n.Owner][n.Name] = append(u.methods[n.Owner][n.Name], n.ID)
				u.named[n.Name] = append(u.named[n.Name], n.ID)
				continue
			}
			holder := m.within(rsInline(n.ID, f.Path))
			holder.items[n.Name] = append(holder.items[n.Name], n.ID)
			switch {
			case rsTypeKind(n.Kind):
				u.types[n.Name] = append(u.types[n.Name], n.ID)
			case n.Kind == "macro":
				u.macros[n.Name] = append(u.macros[n.Name], n.ID)
			}
		}
	}
	// The inline modules exist only now, and a self-reading path may lead
	// through them.
	for _, m := range x.order {
		for i, imp := range x.imports[m.file] {
			if selfReading(m, imp) {
				x.selfReads[rsUse{m, i}] = true
			}
		}
	}
	return x
}

// rsPackages are the Cargo packages of mods with the roots among files: the
// ones the manifest names and the ones Cargo finds by convention, since a
// manifest read from text alone cannot list a directory. The library comes
// first.
func rsPackages(files []extract.Result, mods []Module) []rsPackage {
	var out []rsPackage
	for _, mod := range mods {
		c := mod.Cargo
		if c == nil || c.Package == "" {
			continue
		}
		p := rsPackage{dir: mod.Dir, names: map[string]string{c.Lib: mod.Dir}, libRoot: c.LibRoot}
		for name, dir := range c.Deps {
			p.names[name] = dir
		}
		var roots []string
		for _, f := range files {
			rel, ok := strings.CutPrefix(f.Path, mod.Dir+"/")
			if mod.Dir == "" {
				rel, ok = f.Path, true
			}
			if ok && (slices.Contains(c.Roots, f.Path) || slices.ContainsFunc(cargoConventionalRoots, func(p string) bool {
				matched, _ := path.Match(p, rel)
				return matched
			})) {
				roots = append(roots, f.Path)
			}
		}
		if i := slices.Index(roots, p.libRoot); i > 0 {
			roots = slices.Concat([]string{p.libRoot}, roots[:i], roots[i+1:])
		}
		p.roots = roots
		out = append(out, p)
	}
	return out
}

// nearest is the crate names of the package whose dir holds file deepest,
// for a file no tree reaches.
func nearest(pkgs []rsPackage, file string) map[string]string {
	names, depth := map[string]string{}, -1
	for _, p := range pkgs {
		if (p.dir == "" || strings.HasPrefix(file, p.dir+"/")) && len(p.dir) > depth {
			names, depth = p.names, len(p.dir)
		}
	}
	return names
}

func newUnit(names map[string]string) *rsUnit {
	return &rsUnit{
		names: names, methods: map[string]map[string][]model.NodeID{}, named: map[string][]model.NodeID{},
		types: map[string][]model.NodeID{}, macros: map[string][]model.NodeID{}, traits: map[string][]model.NodeID{},
	}
}

// claim makes file the module name under parent in unit u, and walks the
// files its `mod x;` declarations name: x.rs or x/mod.rs in the directory of
// the declaring module, or the #[path] relative to it.
func (x *rustIndex) claim(file string, u *rsUnit, parent *rsMod, name string, modRS bool) *rsMod {
	m := &rsMod{unit: u, file: file, parent: parent, name: name, modRS: modRS, items: map[string][]model.NodeID{}, children: map[string]*rsMod{}}
	x.fileMod[file] = m
	if parent == nil {
		u.root = m
	}
	for _, decl := range x.decls[file] {
		holder := m.within(decl.Receiver)
		target, childRS := path.Join(holder.pathDir(), decl.Specifier), true
		if decl.Specifier == "" {
			target, childRS = path.Join(holder.dir(), decl.Name+".rs"), false
			if x.nodes[model.NodeID(target)].ID == "" {
				target, childRS = path.Join(holder.dir(), decl.Name, "mod.rs"), true
			}
		}
		if x.nodes[model.NodeID(target)].ID == "" {
			continue
		}
		x.declared[decl] = target
		child := x.fileMod[target]
		if child == nil {
			child = x.claim(target, u, holder, decl.Name, childRS)
		}
		holder.children[decl.Name] = child
	}
	return m
}

// within is the inline module at the "::"-joined path below m, made on first
// use.
func (m *rsMod) within(inline string) *rsMod {
	if inline == "" {
		return m
	}
	for _, seg := range strings.Split(inline, "::") {
		c := m.children[seg]
		if c == nil {
			c = &rsMod{unit: m.unit, file: m.file, parent: m, name: seg, inline: true, items: map[string][]model.NodeID{}, children: map[string]*rsMod{}}
			m.children[seg] = c
		}
		m = c
	}
	return m
}

// dir is where a `mod x;` in m looks for x.rs and x/mod.rs.
func (m *rsMod) dir() string {
	switch {
	case m.inline:
		return path.Join(m.parent.dir(), m.name)
	case m.modRS:
		return dirOf(m.file)
	}
	return path.Join(dirOf(m.file), strings.TrimSuffix(path.Base(m.file), ".rs"))
}

// pathDir is what a #[path] in m is relative to: the file's directory, and
// below a file the inline modules as directories.
func (m *rsMod) pathDir() string {
	if m.inline {
		return m.dir()
	}
	return dirOf(m.file)
}

// holds says whether from is m or lies inside it: m's private items are
// visible there.
func (m *rsMod) holds(from *rsMod) bool {
	for c := from; c != nil; c = c.parent {
		if c == m {
			return true
		}
	}
	return false
}

// rsInline is the inline module a node of file is declared in: a::b for
// a::b::f and a::b::T.f, "" for the file itself.
func rsInline(id model.NodeID, file string) string {
	q, ok := strings.CutPrefix(string(id), file+"#")
	if !ok {
		return ""
	}
	q, _, _ = strings.Cut(q, "~")
	if i := strings.LastIndex(q, "."); i >= 0 {
		q = q[:i]
	}
	return parentScope(q)
}

func rsTypeKind(k model.Kind) bool {
	switch k {
	case "struct", "enum", "union", "trait", "type":
		return true
	}
	return false
}

// fits says whether d is in the namespace w.
func (x *rustIndex) fits(d rsDef, w rsWant) bool {
	if d.mod != nil {
		return w == rsAny || w == rsType
	}
	k := x.nodes[d.id].Kind
	switch w {
	case rsType:
		return rsTypeKind(k)
	case rsValue:
		return k == "function" || k == "struct"
	case rsMacro:
		return k == "macro"
	}
	return true
}

// keep is the defs of w among defs, appended to hits once each.
func (x *rustIndex) keep(hits []rsDef, w rsWant, defs ...rsDef) []rsDef {
	for _, d := range defs {
		if x.fits(d, w) && !slices.Contains(hits, d) {
			hits = append(hits, d)
		}
	}
	return hits
}

// own is what m itself defines under name in the namespace ns: its items and
// its modules, a private item only seen from inside m (pub false).
func (x *rustIndex) own(m *rsMod, name string, pub bool, ns rsWant) []rsDef {
	var hits []rsDef
	for _, id := range m.items[name] {
		if !pub || x.nodes[id].Exported {
			hits = x.keep(hits, ns, rsDef{id: id})
		}
	}
	if c := m.children[name]; c != nil && ns == rsType {
		hits = append(hits, rsDef{mod: c})
	}
	return hits
}

// rsBinds is the name a use binds in its file.
func rsBinds(imp extract.Import) string {
	switch {
	case imp.Alias != "":
		return imp.Alias
	case imp.Name != "":
		return imp.Name
	}
	return imp.Path
}

// rsSpaces are the namespaces a lookup in w reads: a use binds a name in
// each of them.
func rsSpaces(w rsWant) []rsWant {
	if w == rsAny {
		return []rsWant{rsType, rsValue, rsMacro}
	}
	return []rsWant{w}
}

// rsView is a file module as seen from inside it (pub false) or from
// elsewhere, where only its pub items and pub uses count.
type rsView struct {
	m   *rsMod
	pub bool
}

// rsName is a name in one namespace.
type rsName struct {
	ns   rsWant
	name string
}

// rsEntry is what a name stands for: every candidate the tables offer, and
// whether a path segment with more than one candidate led there. Only an
// entry of one candidate, reached through no such segment, gives an edge.
type rsEntry struct {
	defs []rsDef
	amb  bool
}

// add is e with the candidates of o in the namespace w and o's ambiguity.
func (x *rustIndex) add(e, o rsEntry, w rsWant) rsEntry {
	return rsEntry{defs: x.keep(e.defs, w, o.defs...), amb: e.amb || o.amb}
}

// table is what name means in the namespace ns of module m, inside it (pub
// false) or from elsewhere: m's own items and modules, else what the
// explicit uses of m's file bind it to, else what its globs offer. An inline
// module answers with its own items alone, except for a path written inside
// it (scope): the uses are its file's, wherever in the file they stand, as
// the extractor reads them.
func (x *rustIndex) table(m *rsMod, pub, scope bool, ns rsWant, name string) rsEntry {
	if own := x.own(m, name, pub, ns); len(own) > 0 {
		return rsEntry{defs: own}
	}
	if m.inline && !scope {
		return rsEntry{}
	}
	f := x.fileMod[m.file]
	if is := x.binders(f, pub, name); len(is) > 0 {
		var e rsEntry
		for _, i := range is {
			e = x.add(e, x.useT[rsUse{f, i}], ns)
		}
		return e
	}
	return x.glob[rsView{m: f, pub: pub}][rsName{ns: ns, name: name}]
}

// binders are the explicit uses of the file module f that bind name, seen
// from inside f or from elsewhere (pub uses alone), less those the use whose
// path is being resolved reads past (past).
func (x *rustIndex) binders(f *rsMod, pub bool, name string) []int {
	var out []int
	for _, i := range x.bind[f][name] {
		if (!pub || x.imports[f.file][i].Reexport) && !x.past(rsUse{f, i}) {
			out = append(out, i)
		}
	}
	return out
}

// past says whether the use whose path is being resolved reads past u:
// itself, and when it names its own file's name back (selfReads), every use
// that does the same. Such a use reads that one name of its file, so these
// are the uses binding it. Inline modules write them (`mod tests { use
// super::Thing; }`), hoisted to the file; read past as a group, none
// shadows what the name is without them.
func (x *rustIndex) past(u rsUse) bool {
	w := x.watch
	return w == u || w.m == u.m && x.selfReads[w] && x.selfReads[u]
}

// selfReading says whether the use imp of the file module f names f's own
// name back: a path of keywords and child modules leading to f, ending in
// the name it binds, or that name alone.
func selfReading(f *rsMod, imp extract.Import) bool {
	segs := rsSegs(imp.Path)
	if imp.Name == "*" || segs[len(segs)-1] != rsBinds(imp) {
		return false
	}
	cur := f
	for i, seg := range segs[:len(segs)-1] {
		switch {
		case seg == "crate" && i == 0:
			cur = f.unit.root
		case seg == "self" && i == 0:
		case seg == "super":
			cur = cur.parent
		default:
			cur = cur.children[seg]
		}
		if cur == nil {
			return false
		}
	}
	return cur == f
}

// find is table across the namespaces of w.
func (x *rustIndex) find(m *rsMod, pub, scope bool, w rsWant, name string) rsEntry {
	var e rsEntry
	for _, ns := range rsSpaces(w) {
		e = x.add(e, x.table(m, pub, scope, ns, name), rsAny)
	}
	return e
}

// scope is what name, the first segment of a path written in module m,
// stands for in the namespace w: m's table, and a crate the package names (a
// path dependency, or its own library) unless m's own items or its file's
// explicit uses bind the name. Whether they do is read from the source
// alone, so a glob offering the name makes it ambiguous with the crate.
func (x *rustIndex) scope(m *rsMod, name string, w rsWant) rsEntry {
	e := x.find(m, false, true, w, name)
	if w == rsValue || w == rsMacro || len(x.binders(x.fileMod[m.file], false, name)) > 0 {
		return e
	}
	for _, ns := range rsSpaces(w) {
		if len(x.own(m, name, false, ns)) > 0 {
			return e
		}
	}
	if lib := x.crate(m.unit, name); lib != nil {
		e.defs = x.keep(e.defs, rsAny, rsDef{mod: lib.root})
	}
	return e
}

// crate is the library a crate name reaches from unit u, or nil.
func (x *rustIndex) crate(u *rsUnit, name string) *rsUnit {
	dir, ok := u.names[name]
	if !ok {
		return nil
	}
	return x.libs[dir]
}

// member is what the path m::name reaches in the namespace w, written in the
// module from: m's table, its private part only from inside m.
func (x *rustIndex) member(m *rsMod, name string, from *rsMod, w rsWant) rsEntry {
	return x.find(m, !m.holds(from), false, w, name)
}

// target is what the use i of the file module f names: for a glob the
// modules it opens, ambiguous when its path names more than one candidate.
// An explicit use reads its settled entry.
func (x *rustIndex) target(f *rsMod, i int) rsEntry {
	imp := x.imports[f.file][i]
	if imp.Name != "*" {
		return x.useT[rsUse{f, i}]
	}
	e := x.resolvePath(f, rsSegs(imp.Path), rsType)
	out := rsEntry{amb: e.amb || len(e.defs) > 1}
	for _, d := range e.defs {
		if d.mod != nil {
			out.defs = append(out.defs, d)
		}
	}
	return out
}

// settle fills every explicit use's entry and every file module's glob
// tier, inside and from elsewhere, as the least fixpoint over candidate
// sets: passes from empty tables only add, until one adds nothing, so the
// result depends on neither file order nor module names, and ambiguity is
// decided on it at the end (rsItem). See docs/wiki/topics/code-graph.md.
func (x *rustIndex) settle(mods []*rsMod) {
	for changed := true; changed; {
		changed = false
		for _, m := range mods {
			for i, imp := range x.imports[m.file] {
				if imp.Name == "*" {
					continue
				}
				u := rsUse{m, i}
				x.watch = u
				old := x.useT[u]
				next := x.add(old, x.resolvePath(m, rsSegs(imp.Path), rsAny), rsAny)
				x.watch = rsUse{}
				x.useT[u] = next
				changed = rsGrew(old, next) || changed
			}
			for _, pub := range []bool{false, true} {
				changed = x.globs(rsView{m: m, pub: pub}) || changed
			}
		}
	}
}

// rsGrew says whether next, which holds old, holds more.
func rsGrew(old, next rsEntry) bool {
	return len(next.defs) != len(old.defs) || next.amb != old.amb
}

// globs adds to the glob tier of v what the modules its globs open offer,
// and says whether that added anything. A name a module offers through an
// ambiguous glob is ambiguous.
func (x *rustIndex) globs(v rsView) bool {
	if x.glob[v] == nil {
		x.glob[v] = map[rsName]rsEntry{}
	}
	t := x.glob[v]
	changed := false
	for i, imp := range x.imports[v.m.file] {
		if imp.Name != "*" || v.pub && !imp.Reexport {
			continue
		}
		opened := x.target(v.m, i)
		for _, o := range opened.defs {
			g, pub := o.mod, !o.mod.holds(v.m)
			for _, n := range x.names(g, pub) {
				add := x.table(g, pub, false, n.ns, n.name)
				if len(add.defs) == 0 && !add.amb {
					continue
				}
				add.amb = add.amb || opened.amb
				old := t[n]
				next := x.add(old, add, rsAny)
				t[n] = next
				changed = rsGrew(old, next) || changed
			}
		}
	}
	return changed
}

// names are the names m's table may hold as seen inside it or from
// elsewhere: its own items and modules, and for a file module the names its
// explicit uses bind and its globs offer.
func (x *rustIndex) names(m *rsMod, pub bool) []rsName {
	var out []rsName
	for name := range m.items {
		for _, ns := range rsSpaces(rsAny) {
			out = append(out, rsName{ns: ns, name: name})
		}
	}
	for name := range m.children {
		out = append(out, rsName{ns: rsType, name: name})
	}
	if !m.inline {
		for name := range x.bind[m] {
			for _, ns := range rsSpaces(rsAny) {
				out = append(out, rsName{ns: ns, name: name})
			}
		}
		for n := range x.glob[rsView{m: m, pub: pub}] {
			out = append(out, n)
		}
	}
	return out
}

// rsSegs splits a path at "::"; a leading "::" gives an empty first segment.
func rsSegs(p string) []string {
	return strings.Split(p, "::")
}

// resolvePath is what the path segs, written in module m, names in the
// namespace w: crate::, self::, super:: and a leading :: start where Rust
// starts them, any other first segment by scope; each later segment is a
// member of every module the one before it names, and the one after a type
// an associated item (assoc). The result is ambiguous when a segment before
// the last names more than one candidate.
func (x *rustIndex) resolvePath(m *rsMod, segs []string, w rsWant) rsEntry {
	cur, i := m, 1
	switch segs[0] {
	case "crate":
		cur = m.unit.root
	case "self":
	case "super":
		for i = 0; i < len(segs) && segs[i] == "super"; i++ {
			if cur = cur.parent; cur == nil {
				return rsEntry{}
			}
		}
	case "":
		// `use {*};` leaves the path empty, a single empty segment.
		if len(segs) < 2 {
			return rsEntry{}
		}
		lib := x.crate(m.unit, segs[1])
		if lib == nil {
			return rsEntry{}
		}
		cur, i = lib.root, 2
	default:
		if len(segs) == 1 {
			return x.scope(m, segs[0], w)
		}
		return x.walk(x.scope(m, segs[0], rsType), segs[1:], m, w)
	}
	return x.walk(rsEntry{defs: []rsDef{{mod: cur}}}, segs[i:], m, w)
}

// walk is what the segments rest name after the candidates prev of the
// segment before them, written in the module from.
func (x *rustIndex) walk(prev rsEntry, rest []string, from *rsMod, w rsWant) rsEntry {
	var out rsEntry
	for ; len(rest) > 0; rest = rest[1:] {
		ns := rsType
		if len(rest) == 1 {
			ns = w
		}
		next := rsEntry{amb: prev.amb || len(prev.defs) > 1}
		for _, d := range prev.defs {
			if d.mod == nil {
				out.defs = x.keep(out.defs, rsAny, x.assoc(d.id, rest, w)...)
				continue
			}
			next = x.add(next, x.member(d.mod, rest[0], from, ns), rsAny)
		}
		prev = next
	}
	return x.add(out, prev, rsAny)
}

// assoc is what the segments after the type id name: a method of an impl of
// it (Foo::new), else, for an enum, the enum itself, as Kind::A(1) builds
// one of its variants.
func (x *rustIndex) assoc(id model.NodeID, rest []string, w rsWant) []rsDef {
	if len(rest) != 1 {
		return nil
	}
	t := x.nodes[id]
	var hits []rsDef
	for _, m := range x.methodsOf(t, rest[0]) {
		hits = append(hits, rsDef{id: m})
	}
	if len(hits) == 0 && t.Kind == "enum" {
		hits = []rsDef{{id: id}}
	}
	return hits
}

// methodsOf is the methods called name of the type t in its crate. An impl
// names its type by the bare name, so two types of that name in the crate
// leave no telling whose the methods are.
func (x *rustIndex) methodsOf(t model.Node, name string) []model.NodeID {
	u := x.fileMod[t.Path].unit
	if len(u.types[t.Name]) > 1 {
		return nil
	}
	return u.methods[t.Name][name]
}

// self is the method name of the impl type owner, for self.f() and
// Self::f(): its own, else a default of a trait it implements.
func (x *rustIndex) self(u *rsUnit, owner, name string) model.NodeID {
	if len(u.types[owner]) > 1 {
		return ""
	}
	if own := u.methods[owner][name]; len(own) > 0 {
		return rsOne(own)
	}
	// Trait by trait outwards: the traits owner implements or extends, then
	// theirs; the first level that has the method decides.
	seen := map[model.NodeID]bool{}
	for level := u.traits[owner]; len(level) > 0; {
		var hits, next []model.NodeID
		for _, tr := range level {
			if seen[tr] {
				continue
			}
			seen[tr] = true
			t := x.nodes[tr]
			hits = append(hits, x.methodsOf(t, name)...)
			next = append(next, x.fileMod[t.Path].unit.traits[t.Name]...)
		}
		if len(hits) > 0 {
			return rsOne(hits)
		}
		level = next
	}
	return ""
}

// rsOne is the one id among ids, "" for none or several.
func rsOne(ids []model.NodeID) model.NodeID {
	if len(ids) != 1 {
		return ""
	}
	return ids[0]
}

// rsItem is the item e stands for when it is the one candidate, reached
// through no ambiguous segment.
func rsItem(e rsEntry) model.NodeID {
	if len(e.defs) != 1 || e.amb {
		return ""
	}
	return e.defs[0].id
}

// modOf is the module the source of raw is written in.
func (x *rustIndex) modOf(raw extract.RawEdge) *rsMod {
	return x.fileMod[raw.File].within(rsInline(raw.Source, raw.File))
}

// call is the node a raw call lands on and how sure that is, "" for none:
//
//   - x.f(), and a method call read from a macro's arguments: the one method
//     f in the crate, inferred
//   - self.f() and Self::f(): a method of the impl type (self)
//   - m!(): the macro the path names, else, written bare or under crate::,
//     the one macro of that name in the crate, inferred
//   - f() and a::b::f(): the item the path names (resolvePath), inferred when
//     read from a macro's arguments
func (x *rustIndex) call(raw extract.RawEdge) (model.NodeID, model.Confidence) {
	m := x.modOf(raw)
	conf := model.ConfidenceExtracted
	switch raw.Member {
	case ".", "guess.":
		return rsOne(m.unit.named[raw.Name]), model.ConfidenceInferred
	case "macro":
		if id := rsItem(x.resolvePath(m, rsSegs(ccJoin(raw.Receiver, raw.Name)), rsMacro)); id != "" {
			return id, conf
		}
		if raw.Receiver == "" || raw.Receiver == "crate" {
			return rsOne(m.unit.macros[raw.Name]), model.ConfidenceInferred
		}
		return "", conf
	case "guess":
		conf = model.ConfidenceInferred
	}
	switch {
	case raw.Owner != "":
		return x.self(m.unit, raw.Owner, raw.Name), conf
	case raw.Receiver == "Self":
		return x.self(m.unit, x.nodes[raw.Source].Owner, raw.Name), conf
	}
	return rsItem(x.resolvePath(m, rsSegs(ccJoin(raw.Receiver, raw.Name)), rsValue)), conf
}

// relation resolves an impl's trait or a supertrait bound. An impl of a type
// declared in another file starts at the file; it starts at the type when
// the crate has one type of that name, which is what its methods hang on.
func (x *rustIndex) relation(raw extract.RawEdge) (model.Edge, Unresolved, bool) {
	m := x.modOf(raw)
	name := ccJoin(raw.Receiver, raw.Name)
	source := raw.Source
	if raw.Relation == model.RelationImplements && source == model.NodeID(raw.File) {
		if t := rsOne(m.unit.types[raw.Owner]); t != "" {
			source = t
		}
	}
	var traits []rsDef
	e := x.resolvePath(m, rsSegs(name), rsType)
	for _, d := range e.defs {
		if d.mod == nil && !e.amb {
			traits = append(traits, d)
		}
	}
	if len(traits) != 1 {
		why := ReasonNone
		if len(traits) > 1 || e.amb {
			why = ReasonAmbiguous
		}
		return model.Edge{}, Unresolved{Source: source, File: raw.File, Line: raw.Line, Name: name, Reason: why}, false
	}
	// What self.f() searches: the traits of a type, and a trait's supertraits.
	owner := raw.Owner
	if raw.Relation == model.RelationExtends {
		owner = x.nodes[raw.Source].Name
	}
	m.unit.traits[owner] = append(m.unit.traits[owner], traits[0].id)
	return model.Edge{Source: source, Target: traits[0].id, Relation: raw.Relation, Confidence: model.ConfidenceExtracted, Line: raw.Line}, Unresolved{}, true
}

// uses are the imports edges of file's uses: to the file of what each binds,
// once per file and never to file itself or to one a `mod x;` already
// imports. A use carries no line.
func (x *rustIndex) uses(file string, imported map[model.NodeID]bool) []model.Edge {
	var out []model.Edge
	m, uses := x.fileMod[file], x.imports[file]
	for i := range uses {
		// A use may name a module and a function of one name, in the same
		// file (mod pending; pub use pending::pending): the file is sure.
		e := x.target(m, i)
		var files []model.NodeID
		for _, d := range e.defs {
			if d.mod != nil {
				files = append(files, model.NodeID(d.mod.file))
			} else {
				files = append(files, model.NodeID(x.nodes[d.id].Path))
			}
		}
		slices.Sort(files)
		files = slices.Compact(files)
		if len(files) != 1 || e.amb || files[0] == model.NodeID(file) || imported[files[0]] {
			continue
		}
		target := files[0]
		imported[target] = true
		out = append(out, model.Edge{Source: model.NodeID(file), Target: target, Relation: model.RelationImports, Confidence: model.ConfidenceExtracted})
	}
	return out
}

// offer is a godot-rust binding offered under the class its impl type
// registers: the crate's one type of that name, with its #[class] rename
// wherever in the crate the struct stands.
func (x *rustIndex) offer(raw extract.RawEdge) (offer, bool) {
	t := rsOne(x.fileMod[raw.File].unit.types[raw.Receiver])
	class, ok := x.class[t]
	if !ok {
		return offer{}, false
	}
	return offer{
		Export: extract.Export{Namespace: extract.Namespace(raw.Member), Name: raw.Specifier, Class: class, Target: raw.Source},
		file:   raw.File, lang: "rust",
	}, true
}
