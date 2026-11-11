package csharp

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"

	"github.com/xidus90/loomux/internal/code/extract"
	"github.com/xidus90/loomux/internal/code/extract/treesitter"
	"github.com/xidus90/loomux/internal/code/model"
)

// using records one using directive as an import and an imports edge.
// `global using` is marked Global: which files it reaches is the resolver's
// question. Import has no field for `using static`, which opens
// the members of a type, so its alias marks it; no alias can be the keyword.
// A path that is no chain of names -- `global::X` -- names nothing to
// resolve and is left out.
func (x *extractor) using(n *gts.Node) {
	var imp extract.Import
	for i, c := range n.Children() {
		switch {
		case n.FieldNameForChild(i, x.doc.Lang) == "name":
			imp.Alias = x.doc.Text(c)
		case x.doc.Type(c) == "static":
			imp.Alias = "static"
		case x.doc.Type(c) == "global":
			imp.Global = true
		default:
			if p, ok := x.chain(c); ok {
				imp.Path = p
			}
		}
	}
	if imp.Path == "" {
		return
	}
	x.imports = append(x.imports, imp)
	x.importEdges = append(x.importEdges, extract.RawEdge{
		Source: x.fileID, Relation: model.RelationImports, Specifier: imp.Path, File: x.doc.Rel, Line: line(n),
	})
}

// bases records every entry of a base list that names a type as an extends
// edge; whether a base is a class or an interface only the resolver can
// tell. Type arguments are dropped: B<T> names B, and so are the arguments
// a primary constructor passes on: B(a) names B.
func (x *extractor) bases(id model.NodeID, list *gts.Node) {
	for _, c := range list.Children() {
		if x.doc.Type(c) == "primary_constructor_base_type" {
			c = c.Child(0)
		}
		if path, ok := x.chain(c); ok {
			x.extends = append(x.extends, x.edge(id, model.RelationExtends, path, c))
		}
	}
}

// collect walks a subtree that holds no declaration with a node and records
// its calls for source. An ERROR node is skipped whole: a call found there
// sits in text the grammar rejected.
func (x *extractor) collect(n *gts.Node, source model.NodeID) {
	treesitter.Walk(n, func(c *gts.Node) bool {
		switch x.doc.Type(c) {
		case "ERROR":
			return false
		case "invocation_expression":
			// nameof(X) is evaluated by the compiler and calls nothing.
			if path, ok := x.chain(x.doc.Field(c, "function")); ok && path != "nameof" {
				x.calls = append(x.calls, x.edge(source, model.RelationCalls, path, c))
			}
		case "object_creation_expression":
			if path, ok := x.chain(x.doc.Field(c, "type")); ok {
				e := x.edge(source, model.RelationCalls, path, c)
				e.Member = "new"
				x.calls = append(x.calls, e)
			}
		}
		return true
	})
}

// edge is a raw edge from source to the name path as written at n, split
// into the receiver chain and the last name: "A.B.T.M" calls M on "A.B.T".
func (x *extractor) edge(source model.NodeID, rel model.Relation, path string, n *gts.Node) extract.RawEdge {
	e := extract.RawEdge{Source: source, Relation: rel, Name: path, File: x.doc.Rel, Line: line(n)}
	if i := strings.LastIndexByte(path, '.'); i >= 0 {
		e.Receiver, e.Name = path[:i], path[i+1:]
	}
	return e
}

// chain is a dotted chain of plain names as written, "this.a.M", with type
// arguments dropped; false when anything else stands in it -- a call, an
// index, a literal, a predefined type. this and base count as names: they
// are a receiver the resolver knows.
func (x *extractor) chain(n *gts.Node) (string, bool) {
	switch x.doc.Type(n) {
	case "identifier", "this", "base":
		return x.doc.Text(n), true
	case "generic_name":
		return x.doc.Text(n.Child(0)), true
	case "qualified_name":
		return x.pair(x.doc.Field(n, "qualifier"), x.doc.Field(n, "name"))
	case "member_access_expression":
		return x.pair(x.doc.Field(n, "expression"), x.doc.Field(n, "name"))
	}
	return "", false
}

// pair is the chain of head, a dot and the chain of tail.
func (x *extractor) pair(head, tail *gts.Node) (string, bool) {
	h, okH := x.chain(head)
	t, okT := x.chain(tail)
	return h + "." + t, okH && okT
}

// dotted is a namespace's name as one string, "A.B".
func (x *extractor) dotted(n *gts.Node) string {
	name, _ := x.chain(n)
	return name
}

// attributed is whether a header carries the attribute name, in any form C#
// accepts for it: [Signal], [SignalAttribute], [Godot.Signal], or
// [GlobalClass] among others in one list. The namespace is not checked.
func (x *extractor) attributed(head []*gts.Node, name string) bool {
	for _, list := range head {
		if x.doc.Type(list) != "attribute_list" {
			continue
		}
		for _, a := range list.Children() {
			got, _ := x.chain(x.doc.Field(a, "name"))
			got = got[strings.LastIndexByte(got, '.')+1:]
			if got == name || got == name+"Attribute" {
				return true
			}
		}
	}
	return false
}

// offer exports target under name in ns, unless the file has offered that
// name there already: an overload offers its first declaration alone.
func (x *extractor) offer(ns extract.Namespace, name string, target model.NodeID) {
	key := extract.Export{Namespace: ns, Name: name}
	if x.offered[key] {
		return
	}
	x.offered[key] = true
	x.exports = append(x.exports, extract.Export{Namespace: ns, Name: name, Target: target})
}

// line is the 1-based line a node starts on.
func line(n *gts.Node) int { return int(n.StartPoint().Row) + 1 }
