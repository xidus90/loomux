package rust

import (
	"slices"
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// scope is where a use sits: the node that owns it (the file for what no
// definition holds), the type self stands for there, and the inline module
// it is in ("" or "a::b::").
type scope struct {
	source model.NodeID
	owner  string
	mod    string
}

// refs reads everything below n that reaches outside it: calls, macro
// invocations and uses. Closures and nested functions have no node and
// belong to the definition around them.
func (x *extractor) refs(n *gts.Node, s scope) {
	treesitter.Walk(n, func(c *gts.Node) bool {
		switch x.doc.Type(c) {
		case "call_expression":
			x.call(c, s)
		case "macro_invocation":
			x.macro(c, s)
		case "use_declaration":
			x.use(c, s.mod)
		case "extern_crate_declaration":
			x.externCrate(c)
		}
		return true
	})
}

// path is a path as written with "::", generics dropped: Vec for Vec<T> and
// a::b::f for a::b::<T>::f. A leading :: stays. It is false for a path
// that names no item, <T as Tr>::f.
func (x *extractor) path(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "identifier", "type_identifier", "self", "crate", "super":
		return x.doc.Text(n), true
	case "generic_type", "generic_type_with_turbofish":
		return x.path(x.doc.Field(n, "type"))
	case "scoped_identifier", "scoped_type_identifier":
		name := x.doc.Text(x.doc.Field(n, "name"))
		head := x.doc.Field(n, "path")
		if head == nil {
			return "::" + name, true
		}
		p, ok := x.path(head)
		return p + "::" + name, ok
	}
	return "", false
}

// edge is a raw edge from s to the path as written at n: the last segment is
// the name, the ones before it the receiver.
func (x *extractor) edge(n *gts.Node, rel model.Relation, source model.NodeID, path string) extract.RawEdge {
	e := extract.RawEdge{Source: source, Relation: rel, File: x.doc.Rel, Line: int(n.StartPoint().Row) + 1, Name: path}
	if i := strings.LastIndex(path, "::"); i >= 0 {
		e.Receiver, e.Name = path[:i], path[i+2:]
	}
	return e
}

// call records a call as written: f(), a::b::f(), Foo::new(), x.f(),
// self.f(). A member call is marked "." and keeps the text of its object when
// that is a plain chain of names; one on self names the type self stands for
// instead. A callee that is none of these, (f)() or <T as Tr>::f(), names
// nothing to resolve.
func (x *extractor) call(n *gts.Node, s scope) {
	fn := x.doc.Field(n, "function")
	if x.doc.Type(fn) == "generic_function" {
		fn = x.doc.Field(fn, "function")
	}
	if x.doc.Type(fn) != "field_expression" {
		if p, ok := x.path(fn); ok {
			x.calls = append(x.calls, x.edge(n, model.RelationCalls, s.source, p))
		}
		return
	}
	e := x.edge(n, model.RelationCalls, s.source, x.doc.Text(x.doc.Field(fn, "field")))
	if value := x.doc.Field(fn, "value"); x.doc.Type(value) == "self" && s.owner != "" {
		e.Owner = s.owner
	} else {
		e.Receiver, e.Member = x.chain(value), "."
	}
	x.calls = append(x.calls, e)
}

// chain is an object as a dotted chain of names, a.b for a.b.f(); "" for
// anything else.
func (x *extractor) chain(n *gts.Node) string {
	switch x.doc.Type(n) {
	case "identifier", "self":
		return x.doc.Text(n)
	case "field_expression":
		if head := x.chain(x.doc.Field(n, "value")); head != "" {
			return head + "." + x.doc.Text(x.doc.Field(n, "field"))
		}
	}
	return ""
}

// macro records a macro invocation as a call marked "macro" and reads the calls
// among its arguments.
func (x *extractor) macro(n *gts.Node, s scope) {
	if p, ok := x.path(x.doc.Field(n, "macro")); ok {
		e := x.edge(n, model.RelationCalls, s.source, p)
		e.Member = "macro"
		x.calls = append(x.calls, e)
	}
	for _, c := range n.Children() {
		if x.doc.Type(c) == "token_tree" {
			x.guess(c, s)
		}
	}
}

// pathToken are the tokens a path in a macro's arguments is made of, besides ::.
var pathToken = map[string]bool{"identifier": true, "self": true, "crate": true, "super": true}

// guess reads the calls among the arguments of a macro, which the parser
// leaves a flat list of tokens: a path followed by a parenthesis is a call, and
// a name between "." and a parenthesis a method call. The edge is marked
// "guess": a token_tree may as well be a pattern, or the body of a macro the
// parser does not know.
func (x *extractor) guess(tt *gts.Node, s scope) {
	kids := tt.Children()
	start := -1 // the first token of the path being read, -1 outside one
	for i, c := range kids {
		t := x.doc.Type(c)
		switch {
		case t == "token_tree":
			if start >= 0 && x.doc.Type(kids[i-1]) != "::" && x.doc.Text(c.Child(0)) == "(" {
				var p strings.Builder
				for _, k := range kids[start:i] {
					p.WriteString(x.doc.Text(k))
				}
				path, member := p.String(), model.Kind("guess")
				if start > 0 && x.doc.Type(kids[start-1]) == "." {
					path, member = lastSegment(path), "guess."
				}
				e := x.edge(kids[start], model.RelationCalls, s.source, path)
				e.Member = member
				x.calls = append(x.calls, e)
			}
			x.guess(c, s)
			start = -1
		case pathToken[t]:
			if start < 0 || x.doc.Type(kids[i-1]) != "::" {
				start = i
			}
		case t != "::":
			start = -1
		}
	}
}

// use records the names a use declaration binds, one Import each.
func (x *extractor) use(n *gts.Node, mod string) {
	x.useTree(x.doc.Field(n, "argument"), "", mod, x.exported(n.Children()))
}

// fromFile rewrites a path that starts with self or super, as written in the
// inline module mod ("" or "a::b::"), to be relative to the file's own module:
// super::x in a::b is self::a::x. A super that climbs out of the file stays,
// one per level beyond it. Any other path is as written.
func fromFile(path, mod string) string {
	segs := strings.Split(path, "::")
	if mod == "" || (segs[0] != "self" && segs[0] != "super") {
		return path
	}
	in := strings.Split(strings.TrimSuffix(mod, "::"), "::")
	up := 0
	for up < len(segs) && segs[up] == "super" {
		up++
	}
	if segs[0] == "self" {
		return strings.Join(slices.Concat([]string{"self"}, in, segs[1:]), "::")
	}
	if up <= len(in) {
		return strings.Join(slices.Concat([]string{"self"}, in[:len(in)-up], segs[up:]), "::")
	}
	return strings.Join(slices.Concat(slices.Repeat([]string{"super"}, up-len(in)), segs[up:]), "::")
}

// useTree flattens one use tree below prefix: a path, a path with an alias, a
// glob, or a braced list of trees.
func (x *extractor) useTree(n *gts.Node, prefix, mod string, reexport bool) {
	join := func(p string) string {
		if prefix == "" {
			return p
		}
		return prefix + "::" + p
	}
	imp := extract.Import{Reexport: reexport}
	switch x.doc.Type(n) {
	case "use_as_clause":
		imp.Alias = x.doc.Text(x.doc.Field(n, "alias"))
		n = x.doc.Field(n, "path")
		fallthrough
	case "identifier", "self", "crate", "super", "scoped_identifier":
		p, ok := x.path(n)
		if !ok {
			return
		}
		imp.Path = join(p)
		// {self} names the module in front of the list.
		if p == "self" && prefix != "" {
			imp.Path = prefix
		}
	case "use_wildcard":
		imp.Path, imp.Name = prefix, "*"
		if n.ChildCount() > 1 {
			// The parser walls <T>:: off in an ERROR, so the path is a plain one.
			p, _ := x.path(n.Child(0))
			imp.Path = join(p)
		}
	case "scoped_use_list", "use_list":
		list := n
		if x.doc.Type(n) == "scoped_use_list" {
			list = x.doc.Field(n, "list")
			if head := x.doc.Field(n, "path"); head != nil {
				p, _ := x.path(head)
				prefix = join(p)
			}
		}
		for _, c := range list.Children() {
			x.useTree(c, prefix, mod, reexport)
		}
		return
	default:
		return
	}
	imp.Path = fromFile(imp.Path, mod)
	if imp.Name == "" {
		imp.Name = lastSegment(imp.Path)
	}
	x.imports = append(x.imports, imp)
}

// externCrate records `extern crate a as b;` as an import of the crate a.
func (x *extractor) externCrate(n *gts.Node) {
	x.imports = append(x.imports, extract.Import{
		Path:     x.doc.Text(x.doc.Field(n, "name")),
		Alias:    x.doc.Text(x.doc.Field(n, "alias")),
		Reexport: x.exported(n.Children()),
	})
}

// module records `mod x;` as an imports edge to the module x, with the
// #[path] attribute's value as its specifier and the inline module it is
// declared in, a::b, as its receiver. A module with a body is no file.
func (x *extractor) module(def *gts.Node, attrs []attr, mod string) {
	_, name, ok := x.header(def)
	if !ok {
		return
	}
	e := x.edge(def, model.RelationImports, x.fileID, name)
	e.Receiver = strings.TrimSuffix(mod, "::")
	if a, ok := find(attrs, "path"); ok {
		e.Specifier = a.eq
	}
	x.importEdges = append(x.importEdges, e)
}

// implements records `impl Trait for Type` as an implements edge from the
// type to the trait. The type may be declared in another file: its name is the
// Owner, and the edge starts at the file until hang finds the type in this one.
func (x *extractor) implements(def *gts.Node, o owner) {
	tr := x.doc.Field(def, "trait")
	if p, ok := x.path(tr); ok {
		e := x.edge(tr, model.RelationImplements, x.fileID, p)
		e.Owner = o.name
		x.impls[len(x.relations)] = o.qual
		x.relations = append(x.relations, e)
	}
}

// supertraits records `trait A: B + c::C` as an extends edge for each
// bound that is a trait; lifetimes and ?Sized name none.
func (x *extractor) supertraits(def *gts.Node, id model.NodeID) {
	bounds := x.doc.Field(def, "bounds")
	if bounds == nil {
		return
	}
	for _, c := range bounds.Children() {
		if p, ok := x.path(c); ok {
			x.relations = append(x.relations, x.edge(c, model.RelationExtends, id, p))
		}
	}
}
